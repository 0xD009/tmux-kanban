package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"tmux-kanban/internal/agent"
	"tmux-kanban/internal/config"
)

type hermesAutoReviewAction struct {
	kind   string
	choice string
	reason string
}

func (m *model) autoHermesReviewCmd(hadOldStatus bool, oldStatus sessionStatus, nextStatus sessionStatus, key string) tea.Cmd {
	if normalizeSessionStatus(nextStatus) != sessionNeedReview {
		return nil
	}
	if hadOldStatus && normalizeSessionStatus(oldStatus) == sessionNeedReview {
		return nil
	}
	if advice, ok := m.hermes[key]; ok && advice.loading {
		return nil
	}
	item, ok := m.reviewItemByKey(key)
	if !ok || item.Row.hostIndex < 0 || item.Row.hostIndex >= len(m.hosts) {
		return nil
	}
	hermesCfg, ok := m.hermesConfigForReviewItem(item)
	if !ok || !hermesCfg.Enabled || !hermesCfg.AutoReview || strings.TrimSpace(hermesCfg.Command) == "" {
		return nil
	}
	if m.hermes == nil {
		m.hermes = map[string]hermesAdvice{}
	}
	m.hermes[key] = hermesAdvice{loading: true}
	m.status = "auto asking Hermes about " + item.HostName + "/" + item.SessionName
	m.addAgentActivity(agentActivity{
		Source:  agentActivityReview,
		Agent:   "Hermes",
		Target:  item.HostName + "/" + item.SessionName,
		State:   "auto asking",
		Message: "auto review requested",
	})
	return hermesQueryCmd(configWithHermes(m.cfg, hermesCfg), item, m.hosts[item.Row.hostIndex].host, true)
}

func (m *model) applyHermesAutoReview(item reviewItem, hostLabel string, lines []string, advice string) tea.Cmd {
	hermesCfg, ok := m.hermesConfigForReviewItem(item)
	if !ok || !hermesCfg.Enabled || !hermesCfg.AutoReview || m.sessionStatusForKey(item.SessionKey) != sessionNeedReview {
		return nil
	}
	action, ok := parseHermesAutoReviewAction(advice)
	if !ok {
		auditCmd := m.hermesAutoReviewAuditQQCmd(item, hostLabel, lines, advice, "needs human review", true)
		entry := reviewHermesWorkLogEntry(item, hostLabel, "auto", "auto_action")
		entry.Advice = advice
		entry.ParsedAction = "unactionable"
		entry.Accepted = false
		addEffectiveHermesConditions(&entry, hermesCfg)
		m.appendHermesWorkLog(entry)
		m.status = "Hermes auto review needs human review"
		m.addAgentActivity(agentActivity{
			Source:  agentActivityReview,
			Agent:   "Hermes",
			Target:  reviewItemDisplayLabel(item, hostLabel),
			State:   "needs human",
			Message: "auto advice was not actionable",
		})
		return auditCmd
	}

	switch action.kind {
	case "choose":
		decision := "choose " + action.choice
		if action.reason != "" {
			decision += ": " + action.reason
		}
		auditCmd := m.hermesAutoReviewAuditQQCmd(item, hostLabel, lines, advice, decision, false)
		cmd := m.sendChoiceForReviewItem(item, hostLabel, lines, action.choice)
		if cmd == nil {
			auditCmd = m.hermesAutoReviewAuditQQCmd(item, hostLabel, lines, advice, decision, true)
			entry := reviewHermesWorkLogEntry(item, hostLabel, "auto", "auto_action")
			entry.Advice = advice
			entry.ParsedAction = "choose"
			entry.Choice = action.choice
			entry.Accepted = false
			entry.Error = "choice is not visible"
			addEffectiveHermesConditions(&entry, hermesCfg)
			m.appendHermesWorkLog(entry)
			m.status = "Hermes auto choice " + action.choice + " is not visible"
			return auditCmd
		}
		entry := reviewHermesWorkLogEntry(item, hostLabel, "auto", "auto_action")
		entry.Advice = advice
		entry.ParsedAction = "choose"
		entry.Choice = action.choice
		entry.Accepted = true
		entry.Modified = true
		addEffectiveHermesConditions(&entry, hermesCfg)
		m.appendHermesWorkLog(entry)
		m.status = "Hermes auto chose " + action.choice + " for " + item.HostName + "/" + item.SessionName
		return tea.Batch(cmd, auditCmd)
	case "skip":
		decision := "skip"
		if action.reason != "" {
			decision += ": " + action.reason
		}
		auditCmd := m.hermesAutoReviewAuditQQCmd(item, hostLabel, lines, advice, decision, false)
		m.skipReviewItemByKey(item.SessionKey, "Hermes")
		entry := reviewHermesWorkLogEntry(item, hostLabel, "auto", "auto_action")
		entry.Advice = advice
		entry.ParsedAction = "skip"
		entry.Accepted = true
		entry.Modified = true
		addEffectiveHermesConditions(&entry, hermesCfg)
		m.appendHermesWorkLog(entry)
		m.status = "Hermes auto skipped " + item.HostName + "/" + item.SessionName
		return tea.Batch(m.syncReviewTerminalTitleCmd(), auditCmd)
	default:
		auditCmd := m.hermesAutoReviewAuditQQCmd(item, hostLabel, lines, advice, action.kind, true)
		entry := reviewHermesWorkLogEntry(item, hostLabel, "auto", "auto_action")
		entry.Advice = advice
		entry.ParsedAction = action.kind
		entry.Accepted = false
		addEffectiveHermesConditions(&entry, hermesCfg)
		m.appendHermesWorkLog(entry)
		m.status = "Hermes auto review needs human review"
		return auditCmd
	}
}

func (m *model) hermesAutoReviewAuditQQCmd(item reviewItem, hostLabel string, lines []string, advice string, decision string, uncertain bool) tea.Cmd {
	if !config.ShouldSendAutoReviewAuditQQ(m.cfg.Notification.AutoReviewAuditQQ, uncertain) {
		return nil
	}
	cfg := m.cfg
	if hermesCfg, ok := m.hermesConfigForReviewItem(item); ok {
		cfg = configWithHermes(cfg, hermesCfg)
	}
	target := reviewItemDisplayLabel(item, hostLabel)
	return func() tea.Msg {
		return hermesAutoReviewAuditResult{
			target: target,
			result: notifyQQForHermesAutoReview(cfg, item, hostLabel, lines, advice, decision),
		}
	}
}

func (m *model) sendChoiceForReviewItem(item reviewItem, hostLabel string, lines []string, digit string) tea.Cmd {
	if item.Row.hostIndex < 0 || item.Row.hostIndex >= len(m.hosts) {
		return nil
	}
	screen := agent.AnalyzeScreen(lines)
	if len(screen.Choices) == 0 {
		return nil
	}
	keys := choiceKeys(screen, digit)
	if len(keys) == 0 {
		return nil
	}
	m.markReviewChoiceSent(item, hostLabel)
	return tea.Batch(sendKeySequenceCmd("choice "+digit, m.hosts[item.Row.hostIndex].host, item.Row.attachTarget, keys...), m.syncReviewTerminalTitleCmd())
}

func (m *model) markReviewChoiceSent(item reviewItem, hostLabel string) {
	if m.statuses == nil {
		m.statuses = map[string]sessionStatus{}
	}
	m.statuses[item.SessionKey] = sessionWorking
	delete(m.reviewSkipped, item.SessionKey)
	delete(m.reviewTargets, item.SessionKey)
	m.clearHermesAdvice(item.SessionKey)
	m.preview = previewState{}
	m.addAgentActivity(agentActivity{
		Source:  agentActivityReview,
		Agent:   "Hermes",
		Target:  reviewItemDisplayLabel(item, hostLabel),
		State:   "auto chose",
		Message: "accepted Hermes review advice",
	})
}

func (m *model) skipReviewItemByKey(key string, agentName string) {
	item, ok := m.reviewItemByKey(key)
	if !ok {
		return
	}
	if m.reviewSkipped == nil {
		m.reviewSkipped = map[string]bool{}
	}
	m.reviewSkipped[key] = true
	m.preview = previewState{}
	m.addAgentActivity(agentActivity{
		Source:  agentActivityReview,
		Agent:   agentName,
		Target:  item.HostName + "/" + item.SessionName,
		State:   "auto skipped",
		Message: "accepted Hermes review advice",
	})
}

func parseHermesAutoReviewAction(text string) (hermesAutoReviewAction, bool) {
	for _, line := range strings.Split(strings.TrimSpace(text), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		upper := strings.ToUpper(line)
		if strings.HasPrefix(upper, "CHOOSE ") {
			rest := strings.TrimSpace(line[len("CHOOSE "):])
			choice := leadingDigits(rest)
			if choice == "" {
				return hermesAutoReviewAction{}, false
			}
			return hermesAutoReviewAction{kind: "choose", choice: choice, reason: strings.TrimSpace(rest[len(choice):])}, true
		}
		if strings.HasPrefix(upper, "SKIP") {
			return hermesAutoReviewAction{kind: "skip", reason: strings.TrimSpace(line[len("SKIP"):])}, true
		}
		if strings.HasPrefix(upper, "ASK") {
			return hermesAutoReviewAction{kind: "ask", reason: strings.TrimSpace(line[len("ASK"):])}, false
		}
		return hermesAutoReviewAction{}, false
	}
	return hermesAutoReviewAction{}, false
}

func leadingDigits(value string) string {
	end := 0
	for _, r := range value {
		if r < '0' || r > '9' {
			break
		}
		end++
	}
	return value[:end]
}

func reviewItemDisplayLabel(item reviewItem, fallbackHost string) string {
	host := strings.TrimSpace(item.HostName)
	if host == "" {
		host = strings.TrimSpace(fallbackHost)
	}
	if host == "" {
		host = "unknown"
	}
	return fmt.Sprintf("%s/%s", host, item.SessionName)
}
