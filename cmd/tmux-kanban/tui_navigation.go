package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"tmux-kanban/internal/config"
	"tmux-kanban/internal/tmuxscan"
)

func (m *model) handleMouse(msg tea.MouseMsg, now time.Time) {
	hitPanel := m.panelAt(msg.X, msg.Y)
	if msg.Type == tea.MouseLeft {
		if hitPanel != panelNone {
			m.focusedPanel = hitPanel
			if label := focusPanelLabel(hitPanel); label != "" {
				m.status = "focus: " + label
			}
		}
		return
	}

	direction := wheelDirection(msg)
	if direction == 0 {
		return
	}

	panel := hitPanel
	if panel == panelNone {
		panel = m.focusedPanel
	}
	if direction == m.lastWheelDirection && panel == m.lastWheelPanel && now.Sub(m.lastWheelAt) < wheelThrottleInterval {
		return
	}

	m.lastWheelAt = now
	m.lastWheelDirection = direction
	m.lastWheelPanel = panel
	m.moveFocusedPanel(panel, direction)
}

func (m *model) moveFocusedPanel(panel focusedPanel, direction int) {
	if panel != panelNone {
		m.focusedPanel = panel
	}
	switch panel {
	case panelPreview:
		m.scrollPreview(direction)
	case panelActivity:
		m.scrollActivity(direction)
	case panelReviewQueue:
		m.moveReviewCursor(direction)
	case panelExplorer, panelKanban:
		m.moveCursor(direction)
	default:
		m.movePrimaryCursor(direction)
	}
}

func (m *model) movePrimaryCursor(direction int) {
	if m.viewMode == viewReview {
		m.moveReviewCursor(direction)
	} else {
		m.moveCursor(direction)
	}
}

func wheelDirection(msg tea.MouseMsg) int {
	switch msg.Type {
	case tea.MouseWheelUp:
		return -1
	case tea.MouseWheelDown:
		return 1
	default:
		return 0
	}
}

func (m *model) moveCursor(delta int) {
	rows := m.rows()
	if len(rows) == 0 {
		m.cursor = 0
		m.resetPreviewScroll()
		return
	}

	next := m.cursor + delta
	if next < 0 {
		next = 0
	}
	if next >= len(rows) {
		next = len(rows) - 1
	}
	if next != m.cursor {
		m.cursor = next
		m.resetPreviewScroll()
		return
	}
	m.cursor = next
}

func (m *model) scrollPreview(direction int) {
	step := 3
	if direction < 0 {
		m.previewScroll += step
	} else {
		m.previewScroll -= step
	}
	if m.previewScroll < 0 {
		m.previewScroll = 0
	}
}

func (m *model) resetPreviewScroll() {
	m.previewScroll = 0
}

func (m *model) scrollActivity(direction int) {
	step := 3
	if direction > 0 {
		m.activityScroll += step
	} else {
		m.activityScroll -= step
	}
	if m.activityScroll < 0 {
		m.activityScroll = 0
	}
}

func (m *model) toggleSelected() {
	rows := m.rows()
	if len(rows) == 0 || m.cursor >= len(rows) {
		return
	}

	selected := rows[m.cursor]
	switch selected.kind {
	case rowHost, rowSession, rowWindow:
		m.expanded[selected.key] = !m.expanded[selected.key]
	}
}

type selectedSessionRef struct {
	Key          string
	HostIndex    int
	SessionIndex int
	Host         config.Host
	Session      tmuxscan.Session
}

func (m model) selectedSessionRef() (selectedSessionRef, bool) {
	selected, ok := m.selectedRow()
	if !ok {
		return selectedSessionRef{}, false
	}
	return m.sessionRefForRow(selected)
}

func (m model) sessionRefForRow(selected row) (selectedSessionRef, bool) {
	if selected.hostIndex < 0 || selected.hostIndex >= len(m.hosts) {
		return selectedSessionRef{}, false
	}
	switch selected.kind {
	case rowSession, rowWindow, rowPane:
		sessions := m.hosts[selected.hostIndex].snapshot.Sessions
		session, ok := sessionAt(sessions, selected.sessionIndex)
		if !ok {
			return selectedSessionRef{}, false
		}
		host := m.hosts[selected.hostIndex].host
		return selectedSessionRef{
			Key:          sessionStatusKey(host, session),
			HostIndex:    selected.hostIndex,
			SessionIndex: selected.sessionIndex,
			Host:         host,
			Session:      session,
		}, true
	default:
		return selectedSessionRef{}, false
	}
}

func (m *model) cycleSelectedSessionStatus() tea.Cmd {
	ref, ok := m.selectedSessionRef()
	if !ok {
		m.status = "select a session, window, or pane to cycle status"
		return nil
	}

	if m.statuses == nil {
		m.statuses = map[string]sessionStatus{}
	}

	oldStatus, hadOldStatus := m.statuses[ref.Key]
	next := nextSessionStatus(m.sessionStatusForKey(ref.Key))
	m.statuses[ref.Key] = next
	delete(m.statusStreaks, ref.Key)
	if next != sessionNeedReview {
		delete(m.reviewSkipped, ref.Key)
		delete(m.reviewTargets, ref.Key)
	}
	m.clearHermesAdvice(ref.Key)
	m.status = fmt.Sprintf("%s/%s -> %s", ref.Host.Name, ref.Session.Name, statusLabel(next))
	m.addAgentActivity(agentActivity{
		Source:  agentActivitySession,
		Agent:   "session",
		Target:  displayHostName(ref.Host) + "/" + ref.Session.Name,
		State:   statusLabel(next),
		Message: "manual status cycle",
	})
	return m.autoHermesNextStepCmd(hadOldStatus, oldStatus, next, ref.Key)
}
