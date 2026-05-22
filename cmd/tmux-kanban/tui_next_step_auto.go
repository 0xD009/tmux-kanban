package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"tmux-kanban/internal/config"
	"tmux-kanban/internal/mesh"
	tmuxclient "tmux-kanban/internal/tmux"
)

type hermesAutoNextStepAction struct {
	kind    string
	message string
	reason  string
}

type hermesNextStepContext struct {
	Skill         string
	Memory        []mesh.MemoryNode
	MemoryWarning string
}

func (m *model) autoHermesNextStepCmd(hadOldStatus bool, oldStatus sessionStatus, nextStatus sessionStatus, key string) tea.Cmd {
	status := normalizeSessionStatus(nextStatus)
	if status != sessionDone && status != sessionIdle {
		return nil
	}
	if hadOldStatus && normalizeSessionStatus(oldStatus) == status {
		return nil
	}
	if advice, ok := m.hermes[key]; ok && advice.loading {
		return nil
	}

	ref, target, ok := m.sessionAdviceTargetByKey(key)
	if !ok {
		return nil
	}
	hermesCfg := m.scopedHermesConfig(ref.Host, ref.Session.Name)
	if !hermesNextStepAdviceEnabled(hermesCfg, status) {
		return nil
	}
	if !hermesCfg.Enabled || strings.TrimSpace(hermesCfg.Command) == "" {
		return nil
	}
	if m.hermes == nil {
		m.hermes = map[string]hermesAdvice{}
	}
	m.hermes[key] = hermesAdvice{loading: true}
	label := displayHostName(ref.Host) + "/" + ref.Session.Name
	m.status = "auto asking Hermes next step for " + label
	m.addAgentActivity(agentActivity{
		Source:  agentActivitySession,
		Agent:   "Hermes",
		Target:  label,
		State:   "asking next",
		Message: "auto next-step advice requested",
	})
	return hermesNextStepCmd(configWithHermes(m.cfg, hermesCfg), ref, target, status, hermesNextStepAutoEnabled(hermesCfg, status))
}

func hermesNextStepAdviceEnabled(hermesCfg config.HermesConfig, status sessionStatus) bool {
	switch normalizeSessionStatus(status) {
	case sessionDone:
		return hermesCfg.DoneAdvice || hermesCfg.AutoDone
	case sessionIdle:
		return hermesCfg.IdleAdvice || hermesCfg.AutoIdle
	default:
		return false
	}
}

func hermesNextStepAutoEnabled(hermesCfg config.HermesConfig, status sessionStatus) bool {
	switch normalizeSessionStatus(status) {
	case sessionDone:
		return hermesCfg.AutoDone
	case sessionIdle:
		return hermesCfg.AutoIdle
	default:
		return false
	}
}

func (m model) sessionAdviceTargetByKey(key string) (selectedSessionRef, selectedAgentTarget, bool) {
	for hostIndex, state := range m.hosts {
		if !state.loaded || state.snapshot.Err != "" {
			continue
		}
		for sessionIndex, session := range state.snapshot.Sessions {
			if sessionStatusKey(state.host, session) != key {
				continue
			}
			target, ok := activeAgentTargetInSession(hostIndex, sessionKey(hostIndex, session.ID), session)
			if !ok {
				return selectedSessionRef{}, selectedAgentTarget{}, false
			}
			return selectedSessionRef{
				Key:          key,
				HostIndex:    hostIndex,
				SessionIndex: sessionIndex,
				Host:         state.host,
				Session:      session,
			}, target, true
		}
	}
	return selectedSessionRef{}, selectedAgentTarget{}, false
}

func hermesNextStepCmd(cfg config.Config, ref selectedSessionRef, target selectedAgentTarget, status sessionStatus, auto bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := hermesTimeoutContext(context.Background(), cfg.Hermes)
		defer cancel()

		client := tmuxclient.DefaultClient{}
		capture := client.CapturePane(ctx, ref.Host, target.target, 40)
		if capture.Err != "" {
			return hermesNextStepResult{
				key:         ref.Key,
				status:      status,
				err:         capture.Err,
				auto:        auto,
				host:        ref.Host,
				hostName:    displayHostName(ref.Host),
				sessionName: ref.Session.Name,
				target:      target,
				hermes:      cfg.Hermes,
			}
		}

		prompt := hermesNextStepPromptWithContext(ref, target, status, capture.Lines, auto, nextStepHermesPromptContext(cfg, ref, target))
		text, err := runHermesOneshot(ctx, cfg.Hermes, prompt)
		if err != nil {
			return hermesNextStepResult{
				key:         ref.Key,
				status:      status,
				err:         err.Error(),
				auto:        auto,
				host:        ref.Host,
				hostName:    displayHostName(ref.Host),
				sessionName: ref.Session.Name,
				target:      target,
				lines:       capture.Lines,
				hermes:      cfg.Hermes,
			}
		}
		return hermesNextStepResult{
			key:         ref.Key,
			status:      status,
			text:        text,
			auto:        auto,
			host:        ref.Host,
			hostName:    displayHostName(ref.Host),
			sessionName: ref.Session.Name,
			target:      target,
			lines:       capture.Lines,
			hermes:      cfg.Hermes,
		}
	}
}

func hermesNextStepPromptWithContext(ref selectedSessionRef, target selectedAgentTarget, status sessionStatus, lines []string, auto bool, context hermesNextStepContext) string {
	var body strings.Builder
	body.WriteString("You are advising a tmux-kanban next-step workflow.\n")
	body.WriteString("A Codex/Claude session has just entered " + statusLabel(status) + ". Decide whether there is a clear next execution step.\n\n")
	if strings.TrimSpace(context.Skill) != "" {
		body.WriteString("Dispatcher skill:\n")
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
	body.WriteString("- Host: " + displayHostName(ref.Host) + "\n")
	body.WriteString("- Session: " + ref.Session.Name + "\n")
	body.WriteString("- Target: " + target.target + "\n")
	body.WriteString("- Agent: " + string(target.agent) + "\n")
	body.WriteString("- Status: " + statusLabel(status) + "\n")
	if auto {
		body.WriteString("- Auto adoption is enabled; only use SEND when the instruction is safe, concrete, and self-contained.\n")
	} else {
		body.WriteString("- Auto adoption is disabled; provide advice for the human operator.\n")
	}
	body.WriteString("\nVisible terminal tail:\n")
	body.WriteString("```text\n")
	body.WriteString(strings.Join(tailPreviewLines(lines, 120, 30), "\n"))
	body.WriteString("\n```\n\n")
	body.WriteString("Reply concisely in Chinese. Start with one of:\n")
	body.WriteString("- SEND: <message to send to the session agent>\n")
	body.WriteString("- WAIT: <why no next action should be sent now>\n")
	body.WriteString("- ASK: <what extra info is needed before continuing>\n")
	return body.String()
}

func nextStepHermesPromptContext(cfg config.Config, ref selectedSessionRef, target selectedAgentTarget) hermesNextStepContext {
	context := hermesNextStepContext{
		Skill: meshRoleSkillSnippet(cfg.AgentMesh, mesh.RoleDispatcher, "dispatcher"),
	}
	memory, err := mesh.LocalMemoryContext(cfg.AgentMesh.MemoryRoot, sessionAdviceScope(ref, target), 2400)
	if err != nil {
		context.MemoryWarning = err.Error()
	}
	context.Memory = memory
	return context
}

func meshRoleSkillSnippet(cfg config.AgentMeshConfig, role mesh.Role, fallbackSkill string) string {
	skill := fallbackSkill
	for _, policy := range cfg.Policies {
		if mesh.NormalizeRole(policy.Role) == role {
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

func sessionAdviceScope(ref selectedSessionRef, target selectedAgentTarget) mesh.Scope {
	scope := mesh.Scope{
		Host:    displayHostName(ref.Host),
		Session: ref.Session.Name,
	}
	if window, ok := scopedTargetKeyValue(target.key, "window"); ok {
		scope.Window = window
	}
	if pane, ok := scopedTargetKeyValue(target.key, "pane"); ok {
		scope.Pane = pane
	}
	return scope
}

func (m model) hermesNextStepStillCurrent(msg hermesNextStepResult) bool {
	if m.sessionStatusForKey(msg.key) != normalizeSessionStatus(msg.status) {
		return false
	}
	ref, target, ok := m.sessionAdviceTargetByKey(msg.key)
	if !ok {
		return false
	}
	return ref.Session.Name == msg.sessionName && target.target == msg.target.target
}

func (m *model) applyHermesAutoNextStep(msg hermesNextStepResult) tea.Cmd {
	status := normalizeSessionStatus(msg.status)
	hermesCfg := msg.hermes
	if hermesCfg.Command == "" {
		hermesCfg = m.scopedHermesConfig(msg.host, msg.sessionName)
	}
	if !hermesCfg.Enabled || !hermesNextStepAutoEnabled(hermesCfg, status) || m.sessionStatusForKey(msg.key) != status {
		return nil
	}
	action, ok := parseHermesAutoNextStepAction(msg.text)
	if !ok || action.kind != "send" {
		entry := nextStepHermesWorkLogEntry(msg, "auto_action")
		entry.Advice = msg.text
		if ok {
			entry.ParsedAction = action.kind
			entry.Message = action.reason
		} else {
			entry.ParsedAction = "unactionable"
		}
		entry.Accepted = false
		addEffectiveHermesConditions(&entry, hermesCfg)
		m.appendHermesWorkLog(entry)
		m.status = "Hermes next-step advice needs human review"
		m.addAgentActivity(agentActivity{
			Source:  agentActivitySession,
			Agent:   "Hermes",
			Target:  hermesNextStepTargetLabel(msg),
			State:   "needs human",
			Message: "next-step advice was not auto-sendable",
		})
		return nil
	}
	if strings.TrimSpace(action.message) == "" {
		entry := nextStepHermesWorkLogEntry(msg, "auto_action")
		entry.Advice = msg.text
		entry.ParsedAction = "send"
		entry.Accepted = false
		entry.Error = "empty send message"
		addEffectiveHermesConditions(&entry, hermesCfg)
		m.appendHermesWorkLog(entry)
		return nil
	}
	if msg.target.hostIndex < 0 || msg.target.hostIndex >= len(m.hosts) {
		entry := nextStepHermesWorkLogEntry(msg, "auto_action")
		entry.Advice = msg.text
		entry.ParsedAction = "send"
		entry.Message = action.message
		entry.Accepted = false
		entry.Error = "target host index is invalid"
		addEffectiveHermesConditions(&entry, hermesCfg)
		m.appendHermesWorkLog(entry)
		return nil
	}
	if m.statuses == nil {
		m.statuses = map[string]sessionStatus{}
	}
	m.statuses[msg.key] = sessionWorking
	m.status = "Hermes auto sent next step to " + hermesNextStepTargetLabel(msg)
	m.addAgentActivity(agentActivity{
		Source:  agentActivitySession,
		Agent:   "Hermes",
		Target:  hermesNextStepTargetLabel(msg),
		State:   "auto sent",
		Message: "accepted Hermes next-step advice",
	})
	entry := nextStepHermesWorkLogEntry(msg, "auto_action")
	entry.Advice = msg.text
	entry.ParsedAction = "send"
	entry.Message = action.message
	entry.Accepted = true
	entry.Modified = true
	addEffectiveHermesConditions(&entry, hermesCfg)
	m.appendHermesWorkLog(entry)
	return sendTextCmd("Hermes next step", m.hosts[msg.target.hostIndex].host, msg.target.target, action.message, true)
}

func parseHermesAutoNextStepAction(text string) (hermesAutoNextStepAction, bool) {
	for _, line := range strings.Split(strings.TrimSpace(text), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		upper := strings.ToUpper(line)
		if strings.HasPrefix(upper, "SEND:") {
			return hermesAutoNextStepAction{kind: "send", message: strings.TrimSpace(line[len("SEND:"):])}, true
		}
		if strings.HasPrefix(upper, "WAIT:") {
			return hermesAutoNextStepAction{kind: "wait", reason: strings.TrimSpace(line[len("WAIT:"):])}, false
		}
		if strings.HasPrefix(upper, "ASK:") {
			return hermesAutoNextStepAction{kind: "ask", reason: strings.TrimSpace(line[len("ASK:"):])}, false
		}
		return hermesAutoNextStepAction{}, false
	}
	return hermesAutoNextStepAction{}, false
}

func hermesNextStepTargetLabel(msg hermesNextStepResult) string {
	host := strings.TrimSpace(msg.hostName)
	if host == "" {
		host = displayHostName(msg.host)
	}
	if host == "" {
		host = "unknown"
	}
	session := strings.TrimSpace(msg.sessionName)
	if session == "" {
		session = "session"
	}
	return host + "/" + session
}
