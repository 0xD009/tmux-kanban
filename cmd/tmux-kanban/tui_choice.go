package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"tmux-kanban/internal/agent"
)

func (m *model) sendChoice(digit string) tea.Cmd {
	target, ok := m.activeAgentTarget()
	if !ok {
		return nil
	}

	screen := m.currentAgentScreen()
	if len(screen.Choices) == 0 {
		m.status = "no detected choice in current preview"
		return nil
	}

	keys := choiceKeys(screen, digit)
	if len(keys) == 0 {
		m.status = "choice " + digit + " is not visible"
		return nil
	}

	host := m.hosts[target.hostIndex].host
	m.markChoiceSent(target)
	return sendKeySequenceCmd("choice "+digit, host, target.target, keys...)
}

func (m *model) markChoiceSent(target selectedAgentTarget) {
	if m.statuses == nil {
		m.statuses = map[string]sessionStatus{}
	}

	if m.viewMode == viewReview {
		if item, ok := m.currentReviewItem(); ok {
			m.statuses[item.SessionKey] = sessionWorking
			delete(m.reviewSkipped, item.SessionKey)
			delete(m.reviewTargets, item.SessionKey)
			m.clearHermesAdvice(item.SessionKey)
			m.preview = previewState{}
			m.advanceReviewCursorAfter(item.SessionKey)
		}
		return
	}

	ref, ok := m.sessionRefForAgentTarget(target)
	if !ok || m.sessionStatusForKey(ref.Key) != sessionNeedReview {
		return
	}
	m.statuses[ref.Key] = sessionWorking
	delete(m.reviewSkipped, ref.Key)
	delete(m.reviewTargets, ref.Key)
	m.clearHermesAdvice(ref.Key)
	m.preview = previewState{}
}

func choiceKeys(screen agent.Screen, digit string) []string {
	targetIndex := -1
	for i, choice := range screen.Choices {
		if choice.Number == digit || (choice.Number == "" && fmt.Sprintf("%d", i+1) == digit) {
			targetIndex = i
			break
		}
	}
	if targetIndex == -1 {
		return nil
	}

	keys := make([]string, 0, 8)
	selectedIndex := screen.SelectedChoice
	if selectedIndex < 0 || selectedIndex >= len(screen.Choices) {
		selectedIndex = 0
	}
	diff := targetIndex - selectedIndex
	key := "Down"
	if diff < 0 {
		key = "Up"
		diff = -diff
	}
	for i := 0; i < diff; i++ {
		keys = append(keys, key)
	}
	keys = append(keys, "C-m")
	return keys
}
