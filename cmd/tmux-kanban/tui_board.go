package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"tmux-kanban/internal/config"
	"tmux-kanban/internal/core"
	"tmux-kanban/internal/tmuxscan"
)

func (m model) renderKanban(width int, height int) string {
	columns := sessionStatusColumns()
	cards := m.sessionCardsByStatus()
	innerHeight := panelInnerHeight(height)
	innerWidth := maxInt(18, width-8)

	lines := []string{
		panelTitleStyle.Render("Session Board"),
		headerMetaStyle.Render("s cycle status | done is manual"),
		ruleStyle.Render(strings.Repeat("-", innerWidth)),
	}

	if innerHeight >= len(columns)*4+len(lines) {
		remaining := maxInt(len(columns)*4, innerHeight-len(lines))
		baseHeight := remaining / len(columns)
		extra := remaining % len(columns)

		for i, status := range columns {
			blockHeight := baseHeight
			if i < extra {
				blockHeight++
			}
			lines = append(lines, renderKanbanColumn(status, cards[status], i, blockHeight, innerWidth)...)
		}
	} else {
		lines = append(lines, "")
		for i, status := range columns {
			lines = append(lines, renderCompactKanbanColumn(status, len(cards[status]), i, innerWidth))
		}
	}

	for len(lines) < innerHeight {
		lines = append(lines, "")
	}
	if len(lines) > innerHeight {
		lines = lines[:innerHeight]
	}

	return panelStyle.Width(width).Height(innerHeight).Render(strings.Join(lines, "\n"))
}

func sessionStatusColumns() []sessionStatus {
	return []sessionStatus{sessionIdle, sessionWorking, sessionNeedReview, sessionDone}
}

func renderKanbanColumn(status sessionStatus, cards []sessionCard, index int, height int, width int) []string {
	height = maxInt(4, height)
	lines := []string{
		"",
		renderCompactKanbanColumn(status, len(cards), index, width),
		ruleStyle.Render(strings.Repeat("-", width)),
	}
	if len(cards) == 0 {
		lines = append(lines, mutedStyle.Render("  no sessions"))
	} else {
		limit := maxInt(0, height-len(lines))
		for i, card := range cards {
			if i >= limit {
				lines = append(lines, mutedStyle.Render(fmt.Sprintf("  +%d more", len(cards)-i)))
				break
			}
			lines = append(lines, renderSessionCard(card, width))
		}
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return lines
}

func renderCompactKanbanColumn(status sessionStatus, count int, index int, width int) string {
	label := strings.ToUpper(statusLabel(status))
	countText := fmt.Sprintf("%d", count)
	gapWidth := maxInt(1, width-lipgloss.Width(label)-lipgloss.Width(countText))
	return statusTextStyle(status).Render(label) + strings.Repeat(" ", gapWidth) + statusCountStyle(status).Render(" "+countText+" ")
}

func renderSessionCard(card sessionCard, width int) string {
	prefix := "  "
	if card.Selected {
		prefix = "> "
	}
	label := card.Host + "/" + card.Name
	if card.Agent != "" {
		label += " " + card.Agent
	}
	if card.Meta != "" {
		label += " | " + card.Meta
	}
	return renderListLine(prefix+label, width, card.Selected, false)
}

func statusLabel(status sessionStatus) string {
	return core.StatusLabel(status)
}

func statusBadge(status sessionStatus) string {
	return " " + statusChipStyle(status).Render("["+statusLabel(status)+"]")
}

func nextSessionStatus(status sessionStatus) sessionStatus {
	return core.NextManualStatus(status)
}

func (m model) sessionCardsByStatus() map[sessionStatus][]sessionCard {
	cards := map[sessionStatus][]sessionCard{
		sessionIdle:       {},
		sessionWorking:    {},
		sessionNeedReview: {},
		sessionDone:       {},
	}

	selected, _ := m.selectedSessionRef()
	for hostIndex, state := range m.hosts {
		if !state.loaded || state.snapshot.Err != "" {
			continue
		}
		for _, session := range state.snapshot.Sessions {
			if m.isMainSession(hostIndex, session) {
				continue
			}
			key := sessionStatusKey(state.host, session)
			status := m.sessionStatusForKey(key)
			cards[status] = append(cards[status], sessionCard{
				Key:      key,
				Host:     displayHostName(state.host),
				Name:     session.Name,
				Agent:    sessionAgentBadge(session),
				Selected: selected.Key == key && selected.HostIndex == hostIndex,
			})
		}
	}

	return cards
}

func sessionStatusKey(host config.Host, session tmuxscan.Session) string {
	hostID := host.Name
	if hostID == "" {
		hostID = host.SSH
	}
	return hostID + ":" + session.ID
}

func (m model) sessionStatusForKey(key string) sessionStatus {
	if status, ok := m.statuses[key]; ok {
		return normalizeSessionStatus(status)
	}
	return sessionIdle
}

func (m model) sessionStatusForSession(host config.Host, session tmuxscan.Session) (sessionStatus, bool) {
	if status, ok := m.statuses[sessionStatusKey(host, session)]; ok {
		return normalizeSessionStatus(status), true
	}
	return "", false
}

func normalizeSessionStatus(status sessionStatus) sessionStatus {
	normalized := core.NormalizeStatus(status)
	if normalized == "" {
		return sessionIdle
	}
	return normalized
}

func kanbanColumnStyle(index int) lipgloss.Style {
	colors := []lipgloss.Color{
		lipgloss.Color("105"),
		lipgloss.Color("81"),
		lipgloss.Color("214"),
		lipgloss.Color("212"),
		lipgloss.Color("120"),
	}
	return lipgloss.NewStyle().Bold(true).Foreground(colors[index%len(colors)])
}

func chipStyle(background lipgloss.Color, foreground lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Background(background).
		Foreground(foreground).
		Bold(true)
}

func statusTextStyle(status sessionStatus) lipgloss.Style {
	switch status {
	case sessionIdle:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("153"))
	case sessionWorking:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
	case sessionNeedReview:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	case sessionDone:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("120"))
	default:
		return mutedStyle
	}
}

func statusChipStyle(status sessionStatus) lipgloss.Style {
	switch status {
	case sessionIdle:
		return chipStyle(lipgloss.Color("236"), lipgloss.Color("153"))
	case sessionWorking:
		return chipStyle(lipgloss.Color("236"), lipgloss.Color("81"))
	case sessionNeedReview:
		return chipStyle(lipgloss.Color("236"), lipgloss.Color("214"))
	case sessionDone:
		return chipStyle(lipgloss.Color("236"), lipgloss.Color("120"))
	default:
		return chipStyle(lipgloss.Color("236"), lipgloss.Color("245"))
	}
}

func statusCountStyle(status sessionStatus) lipgloss.Style {
	return statusChipStyle(status)
}

func agentBadgeStyle(agent tmuxscan.AgentKind) string {
	if agent == tmuxscan.AgentNone {
		return ""
	}
	color := lipgloss.Color("183")
	if agent == tmuxscan.AgentCodex {
		color = lipgloss.Color("81")
	}
	if agent == tmuxscan.AgentClaude {
		color = lipgloss.Color("214")
	}
	return chipStyle(lipgloss.Color("236"), color).Render("[" + string(agent) + "]")
}

func renderAgentActivityHeader(activity agentActivity, width int) string {
	when := activity.At.Format("15:04:05")
	if activity.At.IsZero() {
		when = "--:--:--"
	}
	agentName := strings.TrimSpace(activity.Agent)
	if agentName == "" {
		agentName = "agent"
	}
	state := strings.TrimSpace(activity.State)
	if state != "" {
		state = " " + activityStateStyle(state).Render(state)
	}
	line := fmt.Sprintf("%s %s %s%s", when, activitySourceBadge(activity.Source), agentName, state)
	return ansi.Truncate(line, width, "...")
}

func activitySourceBadge(source agentActivitySource) string {
	switch source {
	case agentActivityReview:
		return chipStyle(lipgloss.Color("236"), lipgloss.Color("183")).Render("REVIEW")
	case agentActivitySession:
		return chipStyle(lipgloss.Color("236"), lipgloss.Color("81")).Render("SESSION")
	default:
		return chipStyle(lipgloss.Color("236"), lipgloss.Color("245")).Render("AGENT")
	}
}

func activityStateStyle(state string) lipgloss.Style {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case string(sessionNeedReview), "thinking":
		return warnStyle.Bold(true)
	case "error":
		return errorStyle.Bold(true)
	case string(sessionWorking):
		return titleStyle
	case string(sessionDone), "sent", "replied":
		return successStyle.Bold(true)
	case string(sessionIdle):
		return mutedStyle.Bold(true)
	default:
		return headerMetaStyle.Bold(true)
	}
}

func renderListLine(label string, width int, selected bool, dim bool) string {
	line := ansi.Truncate(label, width, "...")
	if selected {
		return selectedRowStyle.Width(width).Render(line)
	}
	if dim {
		return mutedStyle.Render(line)
	}
	return line
}
