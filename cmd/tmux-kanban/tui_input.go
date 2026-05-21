package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"tmux-kanban/internal/ui"
)

func (m model) inputLine(width int) (string, int, bool) {
	switch {
	case m.snapshotInput.active:
		line, cursorCol := renderSnapshotDescriptionInput(m.snapshotInput, width)
		return line, cursorCol, true
	case m.compose.active:
		line, cursorCol := renderComposeInput(composeInputPrefix(m.compose), composeRunes(m.compose), m.compose.cursor, width)
		return line, cursorCol, true
	case m.command.active:
		line, cursorCol := renderCommandInput(m.command, width)
		return line, cursorCol, true
	default:
		return "", 0, false
	}
}

func (m model) inputBoxTitle() string {
	switch {
	case m.snapshotInput.active:
		return "Snapshot Description"
	case m.compose.active:
		if m.viewMode == viewMain {
			return "Main Room"
		}
		target := m.compose.label
		if target == "" {
			target = fallbackAgentTargetLabel(selectedAgentTarget{
				target: m.compose.target,
				agent:  m.compose.agent,
			})
		}
		return "Message -> " + target
	case m.command.active:
		return "Command"
	default:
		return ""
	}
}

func composeInputPrefix(compose composeState) string {
	return ""
}

func composeRunes(compose composeState) []rune {
	if compose.textRunes != nil {
		return compose.textRunes
	}
	return []rune(compose.text)
}

func renderComposeInput(prefix string, text []rune, cursor int, width int) (string, int) {
	return renderInputRunes(prefix, text, cursor, width)
}

func renderCommandInput(command commandState, width int) (string, int) {
	return renderInputLine(ui.InputBar{Mode: ui.InputCommand}.Prompt(), command.text, len([]rune(command.text)), width)
}

func renderSnapshotDescriptionInput(input snapshotDescriptionState, width int) (string, int) {
	return renderInputRunes("description: ", snapshotDescriptionRunes(input), input.cursor, width)
}

func inputBoxContentWidth(width int) int {
	return maxInt(1, width-4)
}

func renderInputBox(title string, inputLine string, width int) []string {
	width = maxInt(6, width)
	contentWidth := inputBoxContentWidth(width)
	title = ansi.Truncate(strings.TrimSpace(title), maxInt(1, width-4), "...")
	if title == "" {
		title = "Input"
	}
	label := " " + title + " "
	styledLabel := inputBoxTitleStyle.Render(label)
	topFill := maxInt(0, width-2-lipgloss.Width(label))
	top := inputBoxBorderStyle.Render("╭") + styledLabel + inputBoxBorderStyle.Render(strings.Repeat("─", topFill)+"╮")
	middle := inputBoxBorderStyle.Render("│ ") + padRightCells(inputLine, contentWidth) + inputBoxBorderStyle.Render(" │")
	bottom := inputBoxBorderStyle.Render("╰" + strings.Repeat("─", maxInt(0, width-2)) + "╯")
	return []string{top, middle, bottom}
}

func padRightCells(value string, width int) string {
	if width <= 0 {
		return ""
	}
	value = ansi.Truncate(value, width, "")
	if missing := width - lipgloss.Width(value); missing > 0 {
		value += strings.Repeat(" ", missing)
	}
	return value
}

func renderInputLine(prefix string, text string, cursor int, width int) (string, int) {
	return renderInputRunes(prefix, []rune(text), cursor, width)
}

func renderInputRunes(prefix string, text []rune, cursor int, width int) (string, int) {
	if width <= 0 {
		return "", 0
	}

	prefixWidth := lipgloss.Width(prefix)
	if prefixWidth >= width {
		visible := ansi.Truncate(prefix, width, "...")
		return inputStyle.Render(visible), minInt(width, lipgloss.Width(visible)+1)
	}

	textWidth := width - prefixWidth
	visibleText, cursorOffset := inputWindowAroundCursorRunes(text, cursor, textWidth)
	visible := prefix + visibleText
	cursorCol := minInt(width, prefixWidth+cursorOffset+1)
	return inputStyle.Render(visible), cursorCol
}

func inputWindowAroundCursorRunes(runes []rune, cursor int, width int) (string, int) {
	if width <= 0 || len(runes) == 0 {
		return "", 0
	}

	cursor = clampInt(cursor, 0, len(runes))
	start := cursor
	end := cursor
	for {
		expanded := false
		if end < len(runes) && inputWindowWidth(runes, start, end+1) <= width {
			end++
			expanded = true
		}
		if start > 0 && inputWindowWidth(runes, start-1, end) <= width {
			start--
			expanded = true
		}
		if !expanded {
			break
		}
	}

	return buildInputWindow(runes, start, end), inputWindowCursorOffset(runes, start, cursor)
}

func inputWindowWidth(runes []rune, start int, end int) int {
	width := 0
	if start > 0 {
		width += lipgloss.Width("...")
	}
	for _, r := range runes[start:end] {
		if display, ok := displayInputRune(r); ok {
			width += lipgloss.Width(string(display))
		}
	}
	if end < len(runes) {
		width += lipgloss.Width("...")
	}
	return width
}

func buildInputWindow(runes []rune, start int, end int) string {
	var out strings.Builder
	if start > 0 {
		out.WriteString("...")
	}
	for _, r := range runes[start:end] {
		if display, ok := displayInputRune(r); ok {
			out.WriteRune(display)
		}
	}
	if end < len(runes) {
		out.WriteString("...")
	}
	return out.String()
}

func inputWindowCursorOffset(runes []rune, start int, cursor int) int {
	offset := 0
	if start > 0 {
		offset += lipgloss.Width("...")
	}
	for _, r := range runes[start:cursor] {
		if display, ok := displayInputRune(r); ok {
			offset += lipgloss.Width(string(display))
		}
	}
	return offset
}

func displayInputRune(r rune) (rune, bool) {
	switch r {
	case '\n', '\r', '\t':
		return ' ', true
	default:
		if r < 0x20 || r == 0x7f {
			return 0, false
		}
		return r, true
	}
}

func tailCells(value string, width int) string {
	if width <= 0 || value == "" {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return value
	}

	marker := "..."
	markerWidth := lipgloss.Width(marker)
	if width <= markerWidth {
		return rightCells(value, width)
	}
	return marker + rightCells(value, width-markerWidth)
}

func rightCells(value string, width int) string {
	if width <= 0 || value == "" {
		return ""
	}

	runes := []rune(value)
	out := make([]rune, 0, len(runes))
	used := 0
	for i := len(runes) - 1; i >= 0; i-- {
		runeText := string(runes[i])
		runeWidth := lipgloss.Width(runeText)
		if runeWidth > width {
			continue
		}
		if used+runeWidth > width {
			break
		}
		out = append(out, runes[i])
		used += runeWidth
	}

	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}
