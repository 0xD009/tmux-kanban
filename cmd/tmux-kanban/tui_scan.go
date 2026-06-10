package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"tmux-kanban/internal/agent"
	"tmux-kanban/internal/config"
	"tmux-kanban/internal/core"
	"tmux-kanban/internal/mesh"
	tmuxclient "tmux-kanban/internal/tmux"
	"tmux-kanban/internal/tmuxscan"
)

func (m model) startScan() (tea.Model, tea.Cmd) {
	return m.startScanModel()
}

func (m model) startScanModel() (model, tea.Cmd) {
	return m.startScanModelWithAnnounce(true)
}

func (m model) startBackgroundScanModel() (model, tea.Cmd) {
	return m.startScanModelWithAnnounce(false)
}

func (m model) startScanModelWithAnnounce(announce bool) (model, tea.Cmd) {
	if len(m.hosts) == 0 {
		m.status = "no hosts configured"
		return m, nil
	}

	cmds := make([]tea.Cmd, 0, len(m.hosts))
	for i := range m.hosts {
		m.hosts[i].loading = true
		cmds = append(cmds, scanCmd(i, m.hosts[i].host))
	}

	m.scanAnnounce = announce
	if announce {
		m.status = "updating tmux sessions..."
	}
	return m, tea.Batch(cmds...)
}

func (m *model) carrySessionStatuses(hostIndex int, next tmuxscan.Snapshot) {
	if hostIndex < 0 || hostIndex >= len(m.hosts) || len(m.statuses) == 0 {
		return
	}

	currentHost := m.hosts[hostIndex].host
	statusByName := map[string]sessionStatus{}
	reviewTargetByName := map[string]selectedAgentTarget{}
	for _, session := range m.hosts[hostIndex].snapshot.Sessions {
		key := sessionStatusKey(currentHost, session)
		if status, ok := m.statuses[key]; ok {
			statusByName[session.Name] = normalizeSessionStatus(status)
		}
		if target, ok := m.reviewTargets[key]; ok && normalizeSessionStatus(m.statuses[key]) == sessionNeedReview {
			reviewTargetByName[session.Name] = target
		}
	}
	if len(statusByName) == 0 {
		return
	}

	nextHost := next.Host
	if nextHost.Name == "" && nextHost.SSH == "" && !nextHost.Local {
		nextHost = currentHost
	}
	if len(reviewTargetByName) > 0 && m.reviewTargets == nil {
		m.reviewTargets = map[string]selectedAgentTarget{}
	}
	for _, session := range next.Sessions {
		key := sessionStatusKey(nextHost, session)
		if _, ok := m.statuses[key]; !ok {
			if status, ok := statusByName[session.Name]; ok {
				m.statuses[key] = status
			}
		}
		if target, ok := reviewTargetByName[session.Name]; ok {
			m.reviewTargets[key] = target
		}
	}
}

func (m *model) applyAgentStatusResult(result agentStatusResult) {
	if m.statuses == nil {
		m.statuses = map[string]sessionStatus{}
	}
	if m.statusStreaks == nil {
		m.statusStreaks = map[string]statusStreak{}
	}
	if m.reviewTargets == nil {
		m.reviewTargets = map[string]selectedAgentTarget{}
	}

	current, hasCurrent := m.statuses[result.key]
	current = normalizeSessionStatus(current)
	oldTarget, hadOldTarget := m.reviewTargets[result.key]

	next := normalizeSessionStatus(result.status)
	streak := m.statusStreaks[result.key]
	if streak.status == next {
		streak.count++
	} else {
		streak = statusStreak{status: next, count: 1}
	}
	m.statusStreaks[result.key] = streak

	next = core.ApplyPolledStatus(current, hasCurrent, next)

	m.statuses[result.key] = next
	if next == sessionNeedReview && result.target.target != "" {
		m.reviewTargets[result.key] = result.target
	} else {
		delete(m.reviewTargets, result.key)
	}

	targetChanged := hadOldTarget && result.target.target != "" && oldTarget.target != result.target.target
	if next != sessionNeedReview || !hasCurrent || current != sessionNeedReview || targetChanged {
		m.clearHermesAdvice(result.key)
	}
}

func (m model) anyHostLoading() bool {
	for _, host := range m.hosts {
		if host.loading {
			return true
		}
	}
	return false
}

func scanCmd(index int, host config.Host) tea.Cmd {
	return func() tea.Msg {
		client := tmuxclient.DefaultClient{}
		return scanResult{
			index:    index,
			snapshot: client.ScanHost(context.Background(), host),
		}
	}
}

func (m model) agentStatusCmds(hostIndex int) []tea.Cmd {
	if hostIndex < 0 || hostIndex >= len(m.hosts) {
		return nil
	}

	state := m.hosts[hostIndex]
	if state.snapshot.Err != "" {
		return nil
	}

	cmds := make([]tea.Cmd, 0, len(state.snapshot.Sessions))
	for _, session := range state.snapshot.Sessions {
		targets := agentTargetsInSession(hostIndex, sessionKey(hostIndex, session.ID), session)
		if len(targets) == 0 {
			continue
		}
		key := sessionStatusKey(state.host, session)
		cmds = append(cmds, agentStatusCmd(key, state.host, targets))
	}
	return cmds
}

func agentStatusCmd(key string, host config.Host, targets []selectedAgentTarget) tea.Cmd {
	return func() tea.Msg {
		client := tmuxclient.DefaultClient{}
		result := agentStatusResult{key: key}
		for _, target := range targets {
			capture := client.CapturePane(context.Background(), host, target.target, 48)
			if capture.Err != "" {
				continue
			}
			status, ok := sessionStatusFromAgentScreen(agent.AnalyzeScreen(capture.Lines))
			if !ok {
				continue
			}
			if !result.ok || sessionStatusPriority(status) > sessionStatusPriority(result.status) {
				result.status = status
				result.target = target
				result.ok = true
			}
			if status == sessionNeedReview {
				break
			}
		}
		return result
	}
}

func sessionStatusPriority(status sessionStatus) int {
	return core.StatusPriority(status)
}

func (m *model) applyPreviewAgentStatus(key string, lines []string) tea.Cmd {
	selected, ok := m.activePreviewRow()
	if !ok || previewKey(selected) != key {
		return nil
	}

	ref, ok := m.sessionRefForRow(selected)
	if !ok {
		return nil
	}

	status, ok := sessionStatusFromAgentScreen(agent.AnalyzeScreen(lines))
	if !ok {
		return nil
	}

	target, _ := m.activeAgentTarget()
	oldStatus, hadOldStatus := m.statuses[ref.Key]
	m.applyAgentStatusResult(agentStatusResult{
		key:    ref.Key,
		status: status,
		target: target,
		ok:     true,
	})
	nextStatus := m.sessionStatusForKey(ref.Key)
	autoCmd := m.autoHermesReviewCmd(hadOldStatus, oldStatus, nextStatus, ref.Key)
	nextStepCmd := m.autoHermesNextStepCmd(hadOldStatus, oldStatus, nextStatus, ref.Key)
	if m.shouldLogPolledStatusChange(hadOldStatus, oldStatus, nextStatus) {
		m.addAgentActivity(agentActivity{
			Source:  agentActivitySession,
			Agent:   string(target.agent),
			Target:  m.agentTargetDisplayLabel(target),
			State:   statusLabel(nextStatus),
			Message: "status changed",
		})
	}
	return tea.Batch(needReviewBellCmd(m.cfg.Notification.TerminalBell, hadOldStatus, oldStatus, nextStatus, autoCmd != nil), autoCmd, nextStepCmd, m.syncReviewTerminalTitleCmd())
}

func hermesQueryCmd(cfg config.Config, item reviewItem, host config.Host, auto bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := hermesTimeoutContext(context.Background(), cfg.Hermes)
		defer cancel()

		client := tmuxclient.DefaultClient{}
		capture := client.CapturePane(ctx, host, item.Row.attachTarget, 40)
		if capture.Err != "" {
			return hermesQueryResult{key: item.SessionKey, err: capture.Err, auto: auto, item: item, host: host, hermes: cfg.Hermes}
		}

		prompt := hermesReviewPromptWithContext(item, capture.Lines, reviewHermesPromptContext(cfg, item))
		text, err := runHermesOneshot(ctx, cfg.Hermes, prompt)
		if err != nil {
			return hermesQueryResult{key: item.SessionKey, err: err.Error(), auto: auto, item: item, host: host, lines: capture.Lines, hermes: cfg.Hermes}
		}
		return hermesQueryResult{key: item.SessionKey, text: text, auto: auto, item: item, host: host, lines: capture.Lines, hermes: cfg.Hermes}
	}
}

func hermesReviewPrompt(item reviewItem, lines []string) string {
	return hermesReviewPromptWithContext(item, lines, hermesReviewContext{})
}

type hermesReviewContext struct {
	Skill         string
	Memory        []mesh.MemoryNode
	MemoryWarning string
}

func hermesReviewPromptWithContext(item reviewItem, lines []string, context hermesReviewContext) string {
	screen := agent.AnalyzeScreen(lines)
	var body strings.Builder
	body.WriteString("You are advising a tmux-kanban review workflow.\n")
	body.WriteString("A Codex/Claude session is waiting for human review. Decide what action looks safest.\n\n")
	if strings.TrimSpace(context.Skill) != "" {
		body.WriteString("Review skill:\n")
		body.WriteString("```markdown\n")
		body.WriteString(strings.TrimSpace(context.Skill))
		body.WriteString("\n```\n\n")
	}
	if len(context.Memory) > 0 || strings.TrimSpace(context.MemoryWarning) != "" {
		body.WriteString("Scoped memory:\n")
		for _, node := range context.Memory {
			body.WriteString(fmt.Sprintf("- %s: %s\n", node.Scope.Key(), compactPromptLine(node.Summary, 360)))
		}
		if strings.TrimSpace(context.MemoryWarning) != "" {
			body.WriteString("- memory warning: " + compactPromptLine(context.MemoryWarning, 240) + "\n")
		}
		body.WriteString("\n")
	}
	body.WriteString("Context:\n")
	body.WriteString("- Host: " + item.HostName + "\n")
	body.WriteString("- Session: " + item.SessionName + "\n")
	body.WriteString("- Target: " + item.Row.attachTarget + "\n")
	body.WriteString("- Agent: " + string(item.Agent) + "\n\n")
	if len(screen.Choices) > 0 {
		body.WriteString("Detected choices:\n")
		for i, choice := range screen.Choices {
			number := choice.Number
			if number == "" {
				number = fmt.Sprintf("%d", i+1)
			}
			marker := ""
			if choice.Selected {
				marker = " (currently selected)"
			}
			body.WriteString(fmt.Sprintf("- %s: %s%s\n", number, choice.Label, marker))
		}
		body.WriteString("\n")
	}
	body.WriteString("Visible terminal tail:\n")
	body.WriteString("```text\n")
	body.WriteString(strings.Join(tailPreviewLines(lines, 120, 30), "\n"))
	body.WriteString("\n```\n\n")
	body.WriteString("Reply concisely in Chinese. Start with one of:\n")
	body.WriteString("- CHOOSE <number>: <short reason>\n")
	body.WriteString("- SKIP: <short reason>\n")
	body.WriteString("- ASK: <what extra info is needed>\n")
	return body.String()
}

func reviewHermesPromptContext(cfg config.Config, item reviewItem) hermesReviewContext {
	context := hermesReviewContext{
		Skill: reviewAdviceSkillSnippet(cfg.AgentMesh),
	}
	memory, err := mesh.LocalMemoryContext(cfg.AgentMesh.MemoryRoot, reviewItemScope(item), 2400)
	if err != nil {
		context.MemoryWarning = err.Error()
	}
	context.Memory = memory
	return context
}

func reviewAdviceSkillSnippet(cfg config.AgentMeshConfig) string {
	skill := "review-advice"
	for _, policy := range cfg.Policies {
		if mesh.NormalizeRole(policy.Role) == mesh.RoleReviewAdvice {
			if strings.TrimSpace(policy.Skill) != "" {
				skill = strings.TrimSpace(policy.Skill)
			}
			break
		}
	}
	path := mesh.MeshSkillPath(cfg.SkillRoot, skill)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return clipString(strings.TrimSpace(string(data)), 3000)
}

func reviewItemScope(item reviewItem) mesh.Scope {
	scope := mesh.Scope{
		Host:    item.HostName,
		Session: item.SessionName,
	}
	key := item.Target.key
	if key == "" {
		key = item.Row.key
	}
	if window, ok := scopedTargetKeyValue(key, "window"); ok {
		scope.Window = window
	}
	if pane, ok := scopedTargetKeyValue(key, "pane"); ok {
		scope.Pane = pane
	}
	if scope.Window != "" || scope.Pane != "" {
		return scope
	}

	target := strings.TrimSpace(item.Row.attachTarget)
	if strings.HasPrefix(target, "%") {
		scope.Pane = target
		return scope
	}
	if colon := strings.LastIndex(target, ":"); colon >= 0 && colon+1 < len(target) {
		windowPane := target[colon+1:]
		parts := strings.SplitN(windowPane, ".", 2)
		scope.Window = strings.TrimSpace(parts[0])
		if len(parts) == 2 {
			scope.Pane = strings.TrimSpace(parts[1])
		}
	}
	return scope
}

func scopedTargetKeyValue(key string, name string) (string, bool) {
	parts := strings.Split(key, ":")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == name && strings.TrimSpace(parts[i+1]) != "" {
			return strings.TrimSpace(parts[i+1]), true
		}
	}
	return "", false
}

func compactPromptLine(text string, width int) string {
	text = strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	return clipString(text, width)
}

func compactCommandError(err error, output []byte) string {
	text := strings.TrimSpace(string(output))
	if text == "" {
		return err.Error()
	}
	return err.Error() + ": " + clipString(text, 240)
}

func sessionStatusFromAgentScreen(screen agent.Screen) (sessionStatus, bool) {
	return core.StatusFromAgentScreen(screen.NeedsReview, screen.Busy, screen.Idle, len(screen.Choices))
}

func captureCmd(key string, host config.Host, target string) tea.Cmd {
	return func() tea.Msg {
		client := tmuxclient.DefaultClient{}
		return captureResult{
			key:     key,
			capture: client.CapturePane(context.Background(), host, target, previewCaptureHeight),
		}
	}
}

func (m *model) cachePreview(key string, capture tmuxscan.Capture) previewCacheEntry {
	if m.cache == nil {
		m.cache = map[string]previewCacheEntry{}
	}
	entry := previewCacheEntry{
		lines:      append([]string(nil), capture.Lines...),
		err:        capture.Err,
		capturedAt: capture.CapturedAt,
	}
	if capture.Err != "" && len(capture.Lines) == 0 {
		if previous, ok := m.cache[key]; ok && len(previous.lines) > 0 {
			entry.lines = append([]string(nil), previous.lines...)
		}
	}

	m.cache[key] = entry
	return entry
}

func (m *model) ensurePreview() tea.Cmd {
	selected, ok := m.activePreviewRow()
	if !ok || selected.attachTarget == "" {
		m.preview = previewState{}
		return nil
	}

	key := previewKey(selected)
	if m.preview.key == key && (m.preview.loading || m.preview.refreshing || m.preview.err != "" || len(m.preview.lines) > 0) {
		return nil
	}

	host := m.hosts[selected.hostIndex].host
	if cached, ok := m.cache[key]; ok {
		m.preview = previewState{
			key:        key,
			hostIndex:  selected.hostIndex,
			target:     selected.attachTarget,
			refreshing: true,
			lines:      append([]string(nil), cached.lines...),
			err:        cached.err,
			capturedAt: cached.capturedAt,
		}
		return captureCmd(key, host, selected.attachTarget)
	}

	m.preview = previewState{
		key:       key,
		hostIndex: selected.hostIndex,
		target:    selected.attachTarget,
		loading:   true,
	}
	return captureCmd(key, host, selected.attachTarget)
}

func (m model) selectedRow() (row, bool) {
	rows := m.rows()
	if len(rows) == 0 || m.cursor < 0 || m.cursor >= len(rows) {
		return row{}, false
	}
	return rows[m.cursor], true
}

func (m model) scanStatus() string {
	loading := 0
	errors := 0
	sessions := 0
	for _, host := range m.hosts {
		if host.loading {
			loading++
		}
		if host.loaded && host.snapshot.Err != "" {
			errors++
		}
		if host.loaded {
			sessions += len(host.snapshot.Sessions)
		}
	}

	if loading > 0 {
		return fmt.Sprintf("scanning... %d hosts remaining", loading)
	}
	return fmt.Sprintf("scan complete: %d sessions, %d errors", sessions, errors)
}
