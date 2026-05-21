package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m model) renderMainRoom(width int, height int, topRow int, leftCol int) string {
	if width >= 120 {
		sideWidth := twoColumnSideWidth(width)
		roomWidth := maxInt(70, width-sideWidth-2)
		room := m.renderMainChat(roomWidth, height, topRow, leftCol)
		side := m.renderMainParticipants(sideWidth, height)
		return lipgloss.JoinHorizontal(lipgloss.Top, room, "  ", side)
	}
	return m.renderMainChat(width, height, topRow, leftCol)
}

func (m model) renderMainChat(width int, height int, topRow int, leftCol int) string {
	innerHeight := panelInnerHeight(height)
	lineWidth := maxInt(18, width-8)
	lines := []string{
		panelTitleStyle.Render("Main Room"),
		headerMetaStyle.Render("conductor channel | enter send | esc blur | : command"),
		ruleStyle.Render(strings.Repeat("-", lineWidth)),
	}

	inputWidth := lineWidth
	if m.compose.active || m.command.active {
		inputWidth = inputBoxContentWidth(lineWidth)
	}
	inputLine, inputCursorCol, hasInput := m.inputLine(inputWidth)
	inputReserve := 0
	if hasInput {
		inputReserve = 3
	}

	contentHeight := maxInt(0, innerHeight-len(lines)-inputReserve)
	messages := m.mainRoomMessages()
	messageLines := make([]string, 0, contentHeight)
	for i := len(messages) - 1; i >= 0 && len(messageLines) < contentHeight; i-- {
		rendered := renderMainMessage(messages[i], lineWidth)
		for j := len(rendered) - 1; j >= 0 && len(messageLines) < contentHeight; j-- {
			messageLines = append(messageLines, rendered[j])
		}
	}
	for i, j := 0, len(messageLines)-1; i < j; i, j = i+1, j-1 {
		messageLines[i], messageLines[j] = messageLines[j], messageLines[i]
	}
	if len(messageLines) == 0 {
		messageLines = append(messageLines, mutedStyle.Render("main room is quiet"))
		messageLines = append(messageLines, mutedStyle.Render("agent events and your conductor messages will appear here"))
	}
	lines = append(lines, messageLines...)

	return m.renderPreviewFrame(width, innerHeight, lines, inputLine, lineWidth, inputWidth, inputCursorCol, hasInput, topRow, leftCol)
}

func (m model) renderMainParticipants(width int, height int) string {
	innerHeight := panelInnerHeight(height)
	lineWidth := maxInt(18, width-8)
	lines := []string{
		panelTitleStyle.Render("Participants"),
		headerMetaStyle.Render(fmt.Sprintf("review %d | agents %d", len(m.reviewQueue()), m.agentPaneCount())),
		ruleStyle.Render(strings.Repeat("-", lineWidth)),
		"",
		renderListLine("You", lineWidth, false, false),
	}
	if m.cfg.Hermes.Enabled {
		lines = append(lines, renderListLine("Hermes", lineWidth, m.viewMode == viewMain, false))
	} else {
		lines = append(lines, mutedStyle.Render("No agent harness"))
	}

	if len(m.reviewQueue()) > 0 {
		lines = append(lines, "", successStyle.Render("Need Review"))
		for _, item := range m.reviewQueue() {
			label := item.HostName + "/" + item.SessionName
			if item.Agent != "" {
				label += " " + string(item.Agent)
			}
			lines = append(lines, renderListLine(label, lineWidth, false, false))
			if len(lines) >= innerHeight {
				break
			}
		}
	}

	for len(lines) < innerHeight {
		lines = append(lines, "")
	}
	if len(lines) > innerHeight {
		lines = lines[:innerHeight]
	}
	return renderFixedPanel(width, innerHeight, lines)
}

func (m model) mainRoomMessages() []mainMessage {
	if len(m.mainMessages) > 0 {
		return m.mainMessages
	}
	return []mainMessage{{
		At:     time.Now(),
		Author: "system",
		Role:   "system",
		Target: "Main Room",
		Text:   "Main Room ready. Agent backend is empty; enable Hermes or wire a harness to get replies.",
	}}
}

func renderMainMessage(message mainMessage, width int) []string {
	author := strings.TrimSpace(message.Author)
	if author == "" {
		author = "system"
	}
	role := strings.TrimSpace(message.Role)
	if role == "" {
		role = "event"
	}
	timeText := "--:--"
	if !message.At.IsZero() {
		timeText = message.At.Local().Format("15:04")
	}
	prefix := fmt.Sprintf("%s  %-10s %s", timeText, author, role)
	header := mainMessageHeaderStyle(role).Render(ansi.Truncate(prefix, width, "..."))
	text := strings.TrimSpace(message.Text)
	if text == "" {
		text = strings.TrimSpace(message.Target)
	}
	if text == "" {
		return []string{header}
	}
	lines := []string{header}
	for _, line := range compactTextLines(text, width, 4) {
		lines = append(lines, previewBorderStyle.Render("  "+line))
	}
	return lines
}

func mainMessageHeaderStyle(role string) lipgloss.Style {
	switch strings.ToLower(role) {
	case "user":
		return successStyle
	case "conductor":
		return titleStyle
	case "review":
		return warnStyle
	case "error":
		return errorStyle
	default:
		return mutedStyle
	}
}

func (m *model) addMainMessage(message mainMessage) {
	if message.At.IsZero() {
		message.At = time.Now()
	}
	if strings.TrimSpace(message.Author) == "" {
		message.Author = "system"
	}
	if strings.TrimSpace(message.Role) == "" {
		message.Role = "event"
	}
	m.mainMessages = append(m.mainMessages, message)
	if len(m.mainMessages) > maxMainMessages {
		m.mainMessages = append([]mainMessage(nil), m.mainMessages[len(m.mainMessages)-maxMainMessages:]...)
	}
}

func (m *model) replaceLastMainHermesThinking(message mainMessage) {
	for i := len(m.mainMessages) - 1; i >= 0; i-- {
		if m.mainMessages[i].Author == "Hermes" && m.mainMessages[i].Role == "conductor" && strings.TrimSpace(m.mainMessages[i].Text) == "thinking..." {
			if message.At.IsZero() {
				message.At = time.Now()
			}
			m.mainMessages[i] = message
			return
		}
	}
	m.addMainMessage(message)
}

func (m *model) addMainMessageForActivity(activity agentActivity) {
	role := "event"
	if activity.Source == agentActivityReview || normalizeSessionStatus(sessionStatus(activity.State)) == sessionNeedReview {
		role = "review"
	}
	if strings.EqualFold(activity.State, "error") {
		role = "error"
	}
	text := strings.TrimSpace(activity.Message)
	if text == "" {
		text = strings.TrimSpace(activity.State)
	}
	if strings.TrimSpace(activity.Target) != "" {
		text = strings.TrimSpace(activity.Target) + " - " + text
	}
	m.addMainMessage(mainMessage{
		At:     activity.At,
		Author: activity.Agent,
		Role:   role,
		Target: activity.Target,
		Text:   text,
	})
}

func (m model) agentPaneCount() int {
	count := 0
	for _, state := range m.hosts {
		if !state.loaded || state.snapshot.Err != "" {
			continue
		}
		for _, session := range state.snapshot.Sessions {
			for _, window := range session.Windows {
				for _, pane := range window.Panes {
					if pane.Agent != "" {
						count++
					}
				}
			}
		}
	}
	return count
}
