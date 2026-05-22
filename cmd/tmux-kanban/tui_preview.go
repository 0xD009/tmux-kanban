package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"tmux-kanban/internal/agent"
	"tmux-kanban/internal/tmuxscan"
)

func (m model) renderHosts(width int, height int) string {
	innerHeight := panelInnerHeight(height)
	lineWidth := maxInt(18, width-8)
	lines := []string{
		panelTitleStyle.Render("Tmux Explorer"),
		headerMetaStyle.Render("enter expand | a attach | x relay | m message"),
		ruleStyle.Render(strings.Repeat("-", lineWidth)),
	}

	rows := m.rows()
	if len(rows) == 0 {
		lines = append(lines, mutedStyle.Render("no hosts configured"))
		return renderFixedPanel(width, innerHeight, lines)
	}

	if m.cursor >= len(rows) {
		m.cursor = len(rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}

	contentHeight := maxInt(0, innerHeight-len(lines))
	selectedLine := m.explorerSelectedContentLine(rows, lineWidth)
	content := []string{}
	if lineWidth >= 58 {
		content = m.renderSessionColumns(rows, lineWidth)
	} else {
		content = m.renderSessionStack(rows, lineWidth)
	}
	lines = append(lines, scrollExplorerContent(content, contentHeight, selectedLine)...)

	return renderFixedPanel(width, innerHeight, lines)
}

type indexedRow struct {
	index int
	row   row
}

func (m model) renderSessionColumns(rows []row, width int) []string {
	agentRows, otherRows := m.partitionSessionRows(rows)
	gap := "  "
	columnWidth := maxInt(18, (width-len(gap))/2)
	lines := []string{
		padRightCells(renderSessionGroupHeader("Agent Sessions", columnWidth), columnWidth) + gap + renderSessionGroupHeader("Other Sessions", columnWidth),
	}
	left := m.renderIndexedRows(agentRows, columnWidth)
	right := m.renderIndexedRows(otherRows, columnWidth)
	height := maxInt(len(left), len(right))
	for i := 0; i < height; i++ {
		leftLine := ""
		if i < len(left) {
			leftLine = left[i]
		}
		rightLine := ""
		if i < len(right) {
			rightLine = right[i]
		}
		lines = append(lines, padRightCells(leftLine, columnWidth)+gap+padRightCells(rightLine, columnWidth))
	}
	return lines
}

func (m model) renderSessionStack(rows []row, width int) []string {
	agentRows, otherRows := m.partitionSessionRows(rows)
	lines := []string{renderSessionGroupHeader("Agent Sessions", width)}
	lines = append(lines, m.renderIndexedRows(agentRows, width)...)
	lines = append(lines, renderSessionGroupHeader("Other Sessions", width))
	lines = append(lines, m.renderIndexedRows(otherRows, width)...)
	return lines
}

func (m model) partitionSessionRows(rows []row) ([]indexedRow, []indexedRow) {
	agentRows := make([]indexedRow, 0, len(rows))
	otherRows := make([]indexedRow, 0, len(rows))
	for i, row := range rows {
		if m.rowHasAgentSession(row) {
			agentRows = append(agentRows, indexedRow{index: i, row: row})
		} else {
			otherRows = append(otherRows, indexedRow{index: i, row: row})
		}
	}
	return agentRows, otherRows
}

func (m model) explorerSelectedContentLine(rows []row, width int) int {
	if len(rows) == 0 || m.cursor < 0 || m.cursor >= len(rows) {
		return 0
	}

	agentRows, otherRows := m.partitionSessionRows(rows)
	for i, item := range agentRows {
		if item.index == m.cursor {
			return 1 + i
		}
	}
	for i, item := range otherRows {
		if item.index != m.cursor {
			continue
		}
		if width >= 58 {
			return 1 + i
		}
		return 1 + renderedIndexedRowsHeight(agentRows) + 1 + i
	}
	return 0
}

func renderedIndexedRowsHeight(rows []indexedRow) int {
	if len(rows) == 0 {
		return 1
	}
	return len(rows)
}

func scrollExplorerContent(lines []string, height int, selectedLine int) []string {
	if height <= 0 || len(lines) <= height {
		return lines
	}

	selectedLine = clampInt(selectedLine, 0, len(lines)-1)
	start := 0
	if selectedLine >= height {
		start = selectedLine - height + 1
	}
	start = clampInt(start, 0, len(lines)-height)
	return lines[start : start+height]
}

func (m model) renderIndexedRows(rows []indexedRow, width int) []string {
	if len(rows) == 0 {
		return []string{mutedStyle.Render(padRightCells("  none", width))}
	}
	lines := make([]string, 0, len(rows))
	for _, item := range rows {
		prefix := "  "
		if item.index == m.cursor {
			prefix = "> "
		}
		lines = append(lines, m.renderHostRow(prefix+item.row.label, item.row.kind, item.index == m.cursor, width))
	}
	return lines
}

func (m model) rowHasAgentSession(row row) bool {
	if row.hostIndex < 0 || row.hostIndex >= len(m.hosts) {
		return false
	}
	session, ok := sessionAt(m.hosts[row.hostIndex].snapshot.Sessions, row.sessionIndex)
	if !ok {
		return false
	}
	return sessionHasAgent(session)
}

func renderSessionGroupHeader(label string, width int) string {
	line := strings.ToUpper(label)
	return mutedStyle.Bold(true).Render(ansi.Truncate(line, width, "..."))
}

func (m model) renderHostRow(label string, kind rowKind, selected bool, width int) string {
	line := ansi.Truncate(label, width, "...")
	if selected {
		return selectedRowStyle.Width(width).Render(line)
	}
	switch kind {
	case rowHost:
		return hostRowStyle.Render(line)
	case rowSession:
		return sessionRowStyle.Render(line)
	case rowWindow:
		return windowRowStyle.Render(line)
	case rowPane:
		return paneRowStyle.Render(line)
	default:
		return line
	}
}

func (m model) renderPreviewPanel(width int, height int, topRow int, leftCol int) string {
	innerHeight := panelInnerHeight(height)
	lineWidth := maxInt(18, width-8)
	title := "Terminal Preview"
	subtitle := "live capture | a attach | m message"
	lines := []string{
		panelTitleStyle.Render(title),
		headerMetaStyle.Render(subtitle),
		ruleStyle.Render(strings.Repeat("-", lineWidth)),
	}
	inputWidth := lineWidth
	if m.compose.active || m.command.active {
		inputWidth = inputBoxContentWidth(lineWidth)
	}
	inputLine, inputCursorCol, hasInput := m.inputLine(inputWidth)

	selected, ok := m.activePreviewRow()
	if !ok {
		message := "no tmux target selected"
		lines = append(lines, "", mutedStyle.Render(message))
		return m.renderPreviewFrame(width, innerHeight, lines, inputLine, lineWidth, inputWidth, inputCursorCol, hasInput, topRow, leftCol)
	}
	if selected.attachTarget == "" {
		lines = append(lines, "", mutedStyle.Render("select a session, window, or pane"))
		return m.renderPreviewFrame(width, innerHeight, lines, inputLine, lineWidth, inputWidth, inputCursorCol, hasInput, topRow, leftCol)
	}

	preview := m.previewForSelected(selected)
	lines = append(lines, renderPreviewMeta(m.previewMeta(selected, preview), lineWidth), "")
	inputReserve := 0
	if hasInput {
		inputReserve = 3
	}
	remaining := maxInt(0, innerHeight-len(lines)-inputReserve)
	hermesLines := m.renderHermesAdviceLines(lineWidth)
	contentHeight := remaining
	if len(hermesLines) > 0 && remaining > len(hermesLines)+2 {
		contentHeight = remaining - len(hermesLines) - 1
	}
	lines = append(lines, renderPreviewContent(selected, preview, lineWidth, contentHeight)...)
	if len(hermesLines) > 0 && remaining > len(hermesLines)+2 {
		lines = append(lines, "")
		lines = append(lines, hermesLines...)
	}

	return m.renderPreviewFrame(width, innerHeight, lines, inputLine, lineWidth, inputWidth, inputCursorCol, hasInput, topRow, leftCol)
}

func (m model) renderPreviewFrame(width int, innerHeight int, lines []string, inputLine string, boxWidth int, inputWidth int, inputCursorCol int, hasInput bool, topRow int, leftCol int) string {
	if hasInput {
		suggestionLines := m.renderCommandSuggestionLines(boxWidth)
		inputStart := maxInt(0, innerHeight-3-len(suggestionLines))
		if len(lines) > inputStart {
			lines = lines[:inputStart]
		}
		for len(lines) < inputStart {
			lines = append(lines, "")
		}
		lines = append(lines, suggestionLines...)
		box := renderInputBox(m.inputBoxTitle(), inputLine, boxWidth)
		lines = append(lines, box...)
		inputIndex := len(lines) - 2
		inputRow := topRow + 2 + inputIndex
		inputBaseCol := leftCol + 4
		recordTUIViewInput(inputLine, inputWidth, inputRow, inputBaseCol, inputCursorCol)
		setTUIInputCursor(inputRow, inputBaseCol+inputCursorCol)
	}
	return renderFixedPanel(width, innerHeight, lines)
}

func (m model) renderCommandSuggestionLines(width int) []string {
	if !m.command.active {
		return nil
	}
	candidates := commandCandidates(m.command.text)
	if len(candidates) == 0 {
		return nil
	}

	visible := minInt(4, len(candidates))
	selected := clampInt(m.command.selected, 0, len(candidates)-1)
	start := clampInt(selected-visible/2, 0, maxInt(0, len(candidates)-visible))
	lines := make([]string, 0, visible)
	for i := start; i < start+visible; i++ {
		candidate := candidates[i]
		marker := "  "
		if i == selected {
			marker = "> "
		}
		line := marker + ":" + candidate.Label()
		if candidate.Description != "" {
			line += "  " + candidate.Description
		}
		line = padRightCells(line, width)
		if i == selected {
			lines = append(lines, selectedRowStyle.Width(width).Render(line))
		} else {
			lines = append(lines, mutedStyle.Render(line))
		}
	}
	return lines
}

func (m model) renderHermesAdviceLines(width int) []string {
	key := ""
	prompt := "Hermes | h ask for a recommendation"
	switch m.viewMode {
	case viewReview:
		item, ok := m.currentReviewItem()
		if !ok {
			return nil
		}
		key = item.SessionKey
	case viewTree:
		ref, ok := m.selectedSessionRef()
		if !ok {
			return nil
		}
		key = ref.Key
		prompt = "Hermes | waiting for auto next-step advice"
	default:
		return nil
	}
	advice, ok := m.hermes[key]
	if !ok {
		if m.viewMode == viewReview {
			return []string{previewBorderStyle.Render(prompt)}
		}
		return nil
	}
	if advice.loading {
		return []string{warnStyle.Render("Hermes | thinking...")}
	}
	if advice.err != "" {
		return []string{errorStyle.Render("Hermes error | " + clipString(advice.err, width-15))}
	}
	if strings.TrimSpace(advice.text) == "" {
		return nil
	}

	lines := []string{successStyle.Render("Hermes")}
	for _, line := range compactTextLines(advice.text, width, 5) {
		lines = append(lines, previewBorderStyle.Render(line))
	}
	return lines
}

func (m model) previewForSelected(selected row) previewState {
	key := previewKey(selected)
	if m.preview.key == key {
		return m.preview
	}
	if cached, ok := m.cache[key]; ok {
		return previewState{
			key:        key,
			hostIndex:  selected.hostIndex,
			target:     selected.attachTarget,
			lines:      append([]string(nil), cached.lines...),
			err:        cached.err,
			capturedAt: cached.capturedAt,
		}
	}
	return previewState{
		key:       key,
		hostIndex: selected.hostIndex,
		target:    selected.attachTarget,
		loading:   true,
	}
}

func (m model) previewMeta(selected row, preview previewState) string {
	target := selected.attachTarget
	if selected.hostIndex >= 0 && selected.hostIndex < len(m.hosts) {
		target = m.hosts[selected.hostIndex].host.Name + " / " + target
	}
	if selected.agent != tmuxscan.AgentNone {
		target += " [" + string(selected.agent) + "]"
	}

	switch {
	case preview.loading && len(preview.lines) == 0 && preview.err == "":
		return target + " - loading"
	case !preview.capturedAt.IsZero():
		return target + " - " + preview.capturedAt.Local().Format("15:04:05")
	case preview.refreshing:
		return target + " - refreshing"
	default:
		return target
	}
}

func renderPreviewMeta(meta string, width int) string {
	return previewBorderStyle.Render(ansi.Truncate(meta, width, "..."))
}

func renderPreviewContent(selected row, preview previewState, width int, height int) []string {
	if height <= 0 {
		return nil
	}

	switch {
	case preview.loading && len(preview.lines) == 0 && preview.err == "":
		return []string{previewBorderStyle.Render("loading...")}
	case preview.err != "" && len(preview.lines) == 0:
		return []string{previewBorderStyle.Render("error: " + clipString(preview.err, width-7))}
	}

	screenLines := preview.lines
	if len(screenLines) == 0 {
		screenLines = []string{"<empty pane>"}
	}

	footer := []string{}
	if preview.err != "" && height >= 3 {
		footer = append(footer, "", previewBorderStyle.Render("refresh error: "+clipString(preview.err, width-15)))
	}

	bridge := []string{}
	if selected.agent != tmuxscan.AgentNone && height-len(footer) >= 5 {
		bridgeBudget := minInt(6, maxInt(3, (height-len(footer))/2))
		bridge = renderAgentBridgeLines(width, agent.AnalyzeScreen(screenLines), bridgeBudget)
		if len(bridge) > 0 {
			bridge = append([]string{""}, bridge...)
		}
	}

	frameHeight := height - len(footer) - len(bridge)
	if frameHeight < 1 {
		bridge = nil
		frameHeight = height - len(footer)
	}
	if frameHeight < 1 {
		footer = nil
		frameHeight = height
	}

	lines := tailPreviewLines(screenLines, width, frameHeight)
	lines = append(lines, bridge...)
	lines = append(lines, footer...)
	return lines
}

func renderAgentBridgeLines(width int, screen agent.Screen, maxLines int) []string {
	if maxLines < 2 {
		return nil
	}

	status := "agent bridge: active"
	switch {
	case screen.NeedsReview && len(screen.Choices) > 0:
		status = "Agent | needs review"
	case screen.Busy:
		status = "Agent | working"
	case screen.Idle:
		status = "Agent | idle"
	}

	lines := []string{previewBorderStyle.Render(ansi.Truncate(status, width, "..."))}
	choiceLimit := 0
	if screen.NeedsReview {
		choiceLimit = minInt(len(screen.Choices), maxInt(0, maxLines-2))
	}
	for i := 0; i < choiceLimit; i++ {
		choice := screen.Choices[i]
		marker := " "
		if choice.Selected {
			marker = ">"
		}
		number := choice.Number
		if number == "" {
			number = fmt.Sprintf("%d", i+1)
		}
		label := fmt.Sprintf("%s %s %s", number, marker, choice.Label)
		line := "  " + ansi.Truncate(label, maxInt(8, width-2), "...")
		if choice.Selected {
			lines = append(lines, selectedRowStyle.Width(width).Render(line))
		} else {
			lines = append(lines, previewBorderStyle.Render(line))
		}
	}
	lines = append(lines, mutedStyle.Render(ansi.Truncate("1-9 choose | x relay | m message", width, "...")))
	return lines
}

func tailPreviewLines(lines []string, width int, height int) []string {
	if height <= 0 {
		return nil
	}
	if len(lines) > height {
		lines = lines[len(lines)-height:]
	}

	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, ansi.Truncate(line, width, "..."))
	}
	return out
}

func compactTextLines(text string, width int, maxLines int) []string {
	if maxLines <= 0 {
		return nil
	}

	out := make([]string, 0, maxLines)
	for _, line := range strings.Split(strings.TrimSpace(text), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, ansi.Truncate(line, width, "..."))
		if len(out) == maxLines {
			break
		}
	}
	return out
}

func renderFixedPanel(width int, innerHeight int, lines []string) string {
	for len(lines) < innerHeight {
		lines = append(lines, "")
	}
	if len(lines) > innerHeight {
		lines = lines[:innerHeight]
	}
	return panelStyle.Width(width).Height(innerHeight).Render(strings.Join(lines, "\n"))
}
