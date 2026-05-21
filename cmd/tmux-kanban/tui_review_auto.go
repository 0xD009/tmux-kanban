package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"tmux-kanban/internal/agent"
)

type hermesAutoReviewAction struct {
	kind   string
	choice string
	reason string
}

func (m *model) autoHermesReviewCmd(hadOldStatus bool, oldStatus sessionStatus, nextStatus sessionStatus, key string) tea.Cmd {
	if !m.cfg.Hermes.Enabled || !m.cfg.Hermes.AutoReview || strings.TrimSpace(m.cfg.Hermes.Command) == "" {
		return nil
	}
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
	return hermesQueryCmd(m.cfg, item, m.hosts[item.Row.hostIndex].host, true)
}

func (m *model) applyHermesAutoReview(item reviewItem, hostLabel string, lines []string, advice string) tea.Cmd {
	if !m.cfg.Hermes.AutoReview || m.sessionStatusForKey(item.SessionKey) != sessionNeedReview {
		return nil
	}
	action, ok := parseHermesAutoReviewAction(advice)
	if !ok {
		m.status = "Hermes auto review needs human review"
		m.addAgentActivity(agentActivity{
			Source:  agentActivityReview,
			Agent:   "Hermes",
			Target:  reviewItemDisplayLabel(item, hostLabel),
			State:   "needs human",
			Message: "auto advice was not actionable",
		})
		return nil
	}

	switch action.kind {
	case "choose":
		cmd := m.sendChoiceForReviewItem(item, hostLabel, lines, action.choice)
		if cmd == nil {
			m.status = "Hermes auto choice " + action.choice + " is not visible"
			return nil
		}
		m.status = "Hermes auto chose " + action.choice + " for " + item.HostName + "/" + item.SessionName
		return cmd
	case "skip":
		m.skipReviewItemByKey(item.SessionKey, "Hermes")
		m.status = "Hermes auto skipped " + item.HostName + "/" + item.SessionName
		return nil
	default:
		m.status = "Hermes auto review needs human review"
		return nil
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
	return sendKeySequenceCmd("choice "+digit, m.hosts[item.Row.hostIndex].host, item.Row.attachTarget, keys...)
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
	m.advanceReviewCursorAfter(item.SessionKey)
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
	m.advanceReviewCursorAfter(key)
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
