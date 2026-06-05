package main

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestRenderInputLineKeepsWideTextWithinWidth(t *testing.T) {
	line, cursorCol := renderInputLine("message to nemotron: ", "中文输入测试abcdefghijklmnopqrstuvwxyz", 6, 36)
	if width := lipgloss.Width(line); width > 36 {
		t.Fatalf("input width = %d, want <= 36: %q", width, line)
	}
	if cursorCol <= 0 || cursorCol > 36 {
		t.Fatalf("cursor col = %d, want within input width", cursorCol)
	}
	if strings.Contains(line, "|") {
		t.Fatalf("input line %q contains fake cursor", line)
	}
}

func TestRenderInputLineCursorCanSitInMiddle(t *testing.T) {
	line, cursorCol := renderInputLine(":", "abcdef", 3, 20)
	if line != ":abcdef" {
		t.Fatalf("line = %q, want :abcdef", line)
	}
	if cursorCol != 5 {
		t.Fatalf("cursor col = %d, want 5", cursorCol)
	}
}

func TestRenderInputLineKeepsPastedNewlinesSingleLine(t *testing.T) {
	line, cursorCol := renderInputLine("message: ", "hello\nworld", 6, 30)
	if strings.Contains(line, "\n") {
		t.Fatalf("line = %q, want no literal newline", line)
	}
	if cursorCol <= 0 || cursorCol > 30 {
		t.Fatalf("cursor col = %d, want within input width", cursorCol)
	}
}

func TestRenderInputBoxCreatesSeparateFrame(t *testing.T) {
	line, _ := renderInputLine("message: ", "hello", 5, 24)
	box := renderInputBox("Message -> local/agents", line, 32)
	if len(box) != 3 {
		t.Fatalf("box lines = %d, want 3", len(box))
	}
	for _, line := range box {
		if width := lipgloss.Width(line); width > 32 {
			t.Fatalf("box line width = %d, want <= 32: %q", width, line)
		}
	}
	if !strings.Contains(box[0], "Message") {
		t.Fatalf("box title = %q, want Message title", box[0])
	}
}

func TestUpdateComposeMovesCursorAndEditsAtCursor(t *testing.T) {
	m := model{compose: composeState{active: true, text: "你好世界", cursor: 4}}

	nextModel, cmd := m.updateCompose(tea.KeyMsg{Type: tea.KeyLeft})
	if cmd != nil {
		t.Fatalf("left returned cmd, want nil")
	}
	next := nextModel.(model)
	if next.compose.cursor != 3 {
		t.Fatalf("cursor after left = %d, want 3", next.compose.cursor)
	}

	nextModel, cmd = next.updateCompose(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("啊")})
	if cmd != nil {
		t.Fatalf("insert returned cmd, want nil")
	}
	next = nextModel.(model)
	if next.compose.text != "你好世啊界" {
		t.Fatalf("text after insert = %q, want 你好世啊界", next.compose.text)
	}
	if next.compose.cursor != 4 {
		t.Fatalf("cursor after insert = %d, want 4", next.compose.cursor)
	}

	nextModel, cmd = next.updateCompose(tea.KeyMsg{Type: tea.KeyBackspace})
	if cmd != nil {
		t.Fatalf("backspace returned cmd, want nil")
	}
	next = nextModel.(model)
	if next.compose.text != "你好世界" || next.compose.cursor != 3 {
		t.Fatalf("after backspace text=%q cursor=%d, want 你好世界 cursor 3", next.compose.text, next.compose.cursor)
	}
}

func TestUpdateComposeCtrlCCancelsMessageMode(t *testing.T) {
	m := model{compose: composeState{active: true, text: "draft", cursor: 5}}

	nextModel, cmd := m.updateCompose(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatalf("ctrl-c returned nil cmd, want hide cursor command")
	}
	next := nextModel.(model)
	if next.compose.active {
		t.Fatalf("compose active = true, want false")
	}
	if next.status != "message canceled" {
		t.Fatalf("status = %q, want message canceled", next.status)
	}
}

func TestUpdateComposeUsesFastCursorMoveWhenInputLineUnchanged(t *testing.T) {
	oldWriter := writeTUIInputCursor
	oldCache := tuiCachedView
	defer func() {
		writeTUIInputCursor = oldWriter
		tuiCachedView = oldCache
	}()

	prefix := composeInputPrefix(composeState{target: "%1", label: "local/agents:2.3 (codex)"})
	inputWidth := 40
	cachedLine, _ := renderComposeInput(prefix, []rune("abcdef"), 4, inputWidth)
	tuiCachedView = tuiViewCacheState{
		valid:        true,
		view:         "cached view",
		inputActive:  true,
		inputLine:    cachedLine,
		inputWidth:   inputWidth,
		inputRow:     12,
		inputBaseCol: 5,
	}

	var gotRow, gotCol int
	writeTUIInputCursor = func(row int, col int) {
		gotRow = row
		gotCol = col
	}

	m := model{compose: composeState{active: true, target: "%1", label: "local/agents:2.3 (codex)", text: "abcdef", cursor: 4}}
	nextModel, cmd := m.updateCompose(tea.KeyMsg{Type: tea.KeyLeft})
	if cmd != nil {
		t.Fatalf("left returned cmd, want nil")
	}
	next := nextModel.(model)
	if !next.skipRender {
		t.Fatalf("skipRender = false, want true for cursor-only move")
	}

	_, expectedCursorCol := renderComposeInput(prefix, []rune("abcdef"), 3, inputWidth)
	if gotRow != 12 || gotCol != 5+expectedCursorCol {
		t.Fatalf("cursor moved to row=%d col=%d, want row=12 col=%d", gotRow, gotCol, 5+expectedCursorCol)
	}
}

func TestUpdateComposePastesAtCursor(t *testing.T) {
	m := model{compose: composeState{active: true, text: "你好世界", cursor: 2}}

	nextModel, cmd := m.updateCompose(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("一行\r\n二行\x03"),
		Paste: true,
	})
	if cmd != nil {
		t.Fatalf("paste returned cmd, want nil")
	}
	next := nextModel.(model)
	if next.compose.text != "你好一行\n二行世界" {
		t.Fatalf("text after paste = %q, want 你好一行\\n二行世界", next.compose.text)
	}
	if next.compose.cursor != len([]rune("你好一行\n二行")) {
		t.Fatalf("cursor after paste = %d, want %d", next.compose.cursor, len([]rune("你好一行\n二行")))
	}
}

func TestCursorAwareOutputPreservesTTYFileDescriptor(t *testing.T) {
	output := cursorAwareOutput{file: os.Stdout}
	if output.Fd() != os.Stdout.Fd() {
		t.Fatalf("fd = %d, want stdout fd %d", output.Fd(), os.Stdout.Fd())
	}
	if err := output.Close(); err != nil {
		t.Fatalf("close = %v, want nil", err)
	}
}
