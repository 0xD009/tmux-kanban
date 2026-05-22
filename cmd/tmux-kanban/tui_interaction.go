package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"tmux-kanban/internal/config"
	tmuxclient "tmux-kanban/internal/tmux"
)

func (m model) selectedAttachTarget() (config.Host, string, bool) {
	selected, ok := m.activePreviewRow()
	if !ok {
		return config.Host{}, "", false
	}
	if selected.attachTarget == "" {
		return config.Host{}, "", false
	}

	host := m.hosts[selected.hostIndex].host
	target := selected.attachTarget
	if m.viewMode == viewReview {
		if item, ok := m.currentReviewItem(); ok && item.SessionName != "" {
			target = item.SessionName
		}
	} else if ref, ok := m.selectedSessionRef(); ok {
		target = ref.Session.Name
	}
	return host, target, true
}

func (m model) attachSelected() tea.Cmd {
	host, target, ok := m.selectedAttachTarget()
	if !ok {
		return nil
	}
	client := tmuxclient.DefaultClient{}
	cmd := client.AttachCommand(host, target)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return attachFinished{err: err}
	})
}

func (m *model) beginAgentControl() {
	target, ok := m.activeAgentTarget()
	if !ok {
		m.status = "select a codex/claude-code session, window, or pane first"
		return
	}

	m.control = agentControlState{
		active:    true,
		key:       target.key,
		hostIndex: target.hostIndex,
		target:    target.target,
		agent:     target.agent,
	}
	m.status = "relay mode for " + target.target + ": j/k/arrows move, enter chooses, esc stops"
}

func (m model) updateAgentControl(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.control = agentControlState{}
		m.status = "relay mode stopped"
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		return m, m.sendControlKeys("selection key", "Up")
	case "down", "j":
		return m, m.sendControlKeys("selection key", "Down")
	case "left", "h":
		return m, m.sendControlKeys("selection key", "Left")
	case "right", "l":
		return m, m.sendControlKeys("selection key", "Right")
	case "enter":
		return m, m.sendControlKeys("selection key", "C-m")
	case "tab":
		return m, m.sendControlKeys("selection key", "Tab")
	case " ":
		return m, m.sendControlKeys("selection key", "Space")
	default:
		return m, nil
	}
}

func (m model) sendControlKeys(action string, keys ...string) tea.Cmd {
	host := m.hosts[m.control.hostIndex].host
	target := m.control.target
	return sendKeysCmd(action, host, target, keys...)
}

func (m *model) beginCompose() {
	target, ok := m.activeAgentTarget()
	if !ok {
		m.status = "select a codex/claude-code session, window, or pane first"
		return
	}

	m.compose = composeState{
		active:    true,
		key:       target.key,
		hostIndex: target.hostIndex,
		target:    target.target,
		label:     m.agentTargetDisplayLabel(target),
		agent:     target.agent,
	}

	label := m.compose.label
	if label == "" {
		label = fallbackAgentTargetLabel(target)
	}
	screen := m.currentAgentScreen()
	if screen.Busy && !screen.Idle {
		m.status = "message mode for " + label + " (agent looks busy)"
	} else {
		m.status = "message mode for " + label
	}
}

func (m model) updateCompose(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.compose.ensureRunes()
	m.compose.cursor = clampInt(m.compose.cursor, 0, len(m.compose.textRunes))
	switch msg.String() {
	case "esc":
		m.compose = composeState{}
		m.status = "message canceled"
		return m, tea.HideCursor
	case "ctrl+c":
		m.compose = composeState{}
		m.status = "message canceled"
		return m, tea.HideCursor
	case "ctrl+u":
		m.compose.setRunes(nil)
		m.compose.cursor = 0
		return m, nil
	case "left":
		if m.compose.cursor > 0 {
			m.compose.cursor--
		}
		m.tryFastInputCursorMove()
		return m, nil
	case "right":
		if m.compose.cursor < len(m.compose.textRunes) {
			m.compose.cursor++
		}
		m.tryFastInputCursorMove()
		return m, nil
	case "home", "ctrl+a":
		m.compose.cursor = 0
		m.tryFastInputCursorMove()
		return m, nil
	case "end", "ctrl+e":
		m.compose.cursor = len(m.compose.textRunes)
		m.tryFastInputCursorMove()
		return m, nil
	case "backspace", "ctrl+h":
		if m.compose.cursor > 0 {
			m.compose.setRunes(removeRunes(m.compose.textRunes, m.compose.cursor-1, m.compose.cursor))
			m.compose.cursor--
		}
		return m, nil
	case "delete", "ctrl+d":
		if m.compose.cursor < len(m.compose.textRunes) {
			m.compose.setRunes(removeRunes(m.compose.textRunes, m.compose.cursor, m.compose.cursor+1))
		}
		return m, nil
	case "enter":
		text := strings.TrimSpace(m.compose.text)
		if text == "" {
			m.compose = composeState{}
			m.status = "message canceled"
			return m, tea.HideCursor
		}

		host := m.hosts[m.compose.hostIndex].host
		target := m.compose.target
		m.compose = composeState{}
		return m, tea.Batch(tea.HideCursor, sendTextCmd("message", host, target, text, true))
	default:
		if len(msg.Runes) > 0 {
			insert := inputRunesFromKey(msg, true)
			textRunes, cursor := insertRunesAtCursor(m.compose.textRunes, m.compose.cursor, insert)
			m.compose.setRunes(textRunes)
			m.compose.cursor = cursor
		}
		return m, nil
	}
}

func (m *model) beginSnapshotDescription() {
	m.snapshotInput = snapshotDescriptionState{active: true}
	m.status = "snapshot description"
}

func (m model) updateSnapshotDescription(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.snapshotInput.ensureRunes()
	m.snapshotInput.cursor = clampInt(m.snapshotInput.cursor, 0, len(m.snapshotInput.textRunes))
	switch msg.String() {
	case "esc":
		m.snapshotInput = snapshotDescriptionState{}
		m.status = "snapshot canceled"
		return m, tea.HideCursor
	case "ctrl+c":
		m.snapshotInput = snapshotDescriptionState{}
		m.status = "snapshot canceled"
		return m, tea.HideCursor
	case "ctrl+u":
		m.snapshotInput.setRunes(nil)
		m.snapshotInput.cursor = 0
		return m, nil
	case "left":
		if m.snapshotInput.cursor > 0 {
			m.snapshotInput.cursor--
		}
		m.tryFastInputCursorMove()
		return m, nil
	case "right":
		if m.snapshotInput.cursor < len(m.snapshotInput.textRunes) {
			m.snapshotInput.cursor++
		}
		m.tryFastInputCursorMove()
		return m, nil
	case "home", "ctrl+a":
		m.snapshotInput.cursor = 0
		m.tryFastInputCursorMove()
		return m, nil
	case "end", "ctrl+e":
		m.snapshotInput.cursor = len(m.snapshotInput.textRunes)
		m.tryFastInputCursorMove()
		return m, nil
	case "backspace", "ctrl+h":
		if m.snapshotInput.cursor > 0 {
			m.snapshotInput.setRunes(removeRunes(m.snapshotInput.textRunes, m.snapshotInput.cursor-1, m.snapshotInput.cursor))
			m.snapshotInput.cursor--
		}
		return m, nil
	case "delete", "ctrl+d":
		if m.snapshotInput.cursor < len(m.snapshotInput.textRunes) {
			m.snapshotInput.setRunes(removeRunes(m.snapshotInput.textRunes, m.snapshotInput.cursor, m.snapshotInput.cursor+1))
		}
		return m, nil
	case "enter":
		description := strings.TrimSpace(m.snapshotInput.text)
		m.snapshotInput = snapshotDescriptionState{}
		return m, tea.Batch(tea.HideCursor, m.saveSnapshotCmd(description))
	default:
		if len(msg.Runes) > 0 {
			insert := inputRunesFromKey(msg, false)
			textRunes, cursor := insertRunesAtCursor(m.snapshotInput.textRunes, m.snapshotInput.cursor, insert)
			m.snapshotInput.setRunes(textRunes)
			m.snapshotInput.cursor = cursor
		}
		return m, nil
	}
}

func (m *model) tryFastInputCursorMove() bool {
	cache := tuiCachedView
	if !cache.valid || !cache.inputActive {
		return false
	}

	inputLine, cursorCol, ok := m.inputLine(cache.inputWidth)
	if !ok || inputLine != cache.inputLine {
		return false
	}

	m.skipRender = true
	tuiCachedView.inputCursorCol = cursorCol
	writeTUIInputCursor(cache.inputRow, cache.inputBaseCol+cursorCol)
	return true
}

func (c *composeState) ensureRunes() {
	if c.textRunes == nil && c.text != "" {
		c.textRunes = []rune(c.text)
	}
	if c.textRunes == nil {
		c.textRunes = []rune{}
	}
	c.text = string(c.textRunes)
}

func (c *composeState) setRunes(runes []rune) {
	if len(runes) == 0 {
		c.textRunes = []rune{}
		c.text = ""
		return
	}
	c.textRunes = runes
	c.text = string(runes)
}

func snapshotDescriptionRunes(input snapshotDescriptionState) []rune {
	if input.textRunes != nil {
		return input.textRunes
	}
	return []rune(input.text)
}

func (s *snapshotDescriptionState) ensureRunes() {
	if s.textRunes == nil && s.text != "" {
		s.textRunes = []rune(s.text)
	}
	if s.textRunes == nil {
		s.textRunes = []rune{}
	}
	s.text = string(s.textRunes)
}

func (s *snapshotDescriptionState) setRunes(runes []rune) {
	if len(runes) == 0 {
		s.textRunes = []rune{}
		s.text = ""
		return
	}
	s.textRunes = runes
	s.text = string(runes)
}

func inputRunesFromKey(msg tea.KeyMsg, allowNewline bool) []rune {
	if len(msg.Runes) == 0 {
		return nil
	}
	if !msg.Paste {
		return msg.Runes
	}

	out := make([]rune, 0, len(msg.Runes))
	for i := 0; i < len(msg.Runes); i++ {
		r := msg.Runes[i]
		switch r {
		case '\r':
			if allowNewline {
				out = append(out, '\n')
			} else {
				out = append(out, ' ')
			}
			if i+1 < len(msg.Runes) && msg.Runes[i+1] == '\n' {
				i++
			}
		case '\n':
			if allowNewline {
				out = append(out, '\n')
			} else {
				out = append(out, ' ')
			}
		case '\t':
			out = append(out, r)
		default:
			if r >= 0x20 && r != 0x7f {
				out = append(out, r)
			}
		}
	}
	return out
}

func insertRunesAtCursor(runes []rune, cursor int, insert []rune) ([]rune, int) {
	if len(insert) == 0 {
		return append([]rune(nil), runes...), clampInt(cursor, 0, len(runes))
	}
	cursor = clampInt(cursor, 0, len(runes))
	next := make([]rune, 0, len(runes)+len(insert))
	next = append(next, runes[:cursor]...)
	next = append(next, insert...)
	next = append(next, runes[cursor:]...)
	return next, cursor + len(insert)
}

func removeRunes(runes []rune, start int, end int) []rune {
	start = clampInt(start, 0, len(runes))
	end = clampInt(end, start, len(runes))
	next := make([]rune, 0, len(runes)-(end-start))
	next = append(next, runes[:start]...)
	next = append(next, runes[end:]...)
	return next
}
