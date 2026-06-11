package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"tmux-kanban/internal/config"
	"tmux-kanban/internal/tmuxscan"
)

func TestMouseWheelIsThrottledPerDirection(t *testing.T) {
	m := model{
		hosts: []hostState{{
			host: config.Host{Name: "local", SSH: "local"},
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{
				{ID: "$1", Name: "one"},
				{ID: "$2", Name: "two"},
				{ID: "$3", Name: "three"},
			}},
			loaded: true,
		}},
		expanded: map[string]bool{"host:0": true},
	}
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	wheelDown := tea.MouseMsg{Type: tea.MouseWheelDown}

	m.handleMouse(wheelDown, now)
	if m.cursor != 1 {
		t.Fatalf("cursor after first wheel = %d, want 1", m.cursor)
	}

	m.handleMouse(wheelDown, now.Add(wheelThrottleInterval/2))
	if m.cursor != 1 {
		t.Fatalf("cursor after throttled wheel = %d, want 1", m.cursor)
	}

	m.handleMouse(wheelDown, now.Add(wheelThrottleInterval+time.Millisecond))
	if m.cursor != 2 {
		t.Fatalf("cursor after later wheel = %d, want 2", m.cursor)
	}
}

func TestMouseFocusRoutesWheelToPanelUnderPointer(t *testing.T) {
	m := model{
		width:  120,
		height: 40,
		hosts: []hostState{{
			host: config.Host{Name: "local", SSH: "local"},
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{
				{ID: "$1", Name: "one"},
				{ID: "$2", Name: "two"},
				{ID: "$3", Name: "three"},
			}},
			loaded: true,
		}},
		expanded: map[string]bool{"host:0": true},
	}
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	preview := testPanelBounds(t, m, panelPreview)
	explorer := testPanelBounds(t, m, panelExplorer)

	m.handleMouse(tea.MouseMsg{Type: tea.MouseWheelUp, X: preview.x, Y: preview.y}, now)
	if m.previewScroll != 5 {
		t.Fatalf("previewScroll = %d, want 5", m.previewScroll)
	}
	if m.cursor != 0 {
		t.Fatalf("cursor = %d, want unchanged", m.cursor)
	}

	m.handleMouse(tea.MouseMsg{Type: tea.MouseWheelDown, X: explorer.x, Y: explorer.y}, now.Add(wheelThrottleInterval+time.Millisecond))
	if m.cursor != 1 {
		t.Fatalf("cursor after explorer wheel = %d, want 1", m.cursor)
	}
	if m.previewScroll != 5 {
		t.Fatalf("previewScroll after cursor move = %d, want preserved (reset deferred to preview load)", m.previewScroll)
	}
}

func TestKeyboardNavigationIgnoresPreviewFocus(t *testing.T) {
	m := model{
		width:  120,
		height: 40,
		hosts: []hostState{{
			host: config.Host{Name: "local", SSH: "local"},
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{
				{ID: "$1", Name: "one"},
				{ID: "$2", Name: "two"},
			}},
			loaded: true,
		}},
		expanded: map[string]bool{"host:0": true},
	}
	preview := testPanelBounds(t, m, panelPreview)

	m.handleMouse(tea.MouseMsg{Type: tea.MouseLeft, X: preview.x, Y: preview.y}, time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC))
	if m.focusedPanel != panelPreview {
		t.Fatalf("focusedPanel = %q, want preview", m.focusedPanel)
	}

	nextModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	next := nextModel.(model)
	if next.cursor != 1 {
		t.Fatalf("cursor after j with preview focus = %d, want 1", next.cursor)
	}
	if next.previewScroll != 0 {
		t.Fatalf("previewScroll after j = %d, want unchanged", next.previewScroll)
	}
}

func TestPageKeysScrollPreviewWithoutMovingCursor(t *testing.T) {
	m := model{
		width:  120,
		height: 40,
		hosts: []hostState{{
			host: config.Host{Name: "local", SSH: "local"},
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{
				{ID: "$1", Name: "one"},
				{ID: "$2", Name: "two"},
			}},
			loaded: true,
		}},
		expanded: map[string]bool{"host:0": true},
	}

	nextModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	next := nextModel.(model)
	if next.previewScroll != 5 {
		t.Fatalf("previewScroll after page up = %d, want 5", next.previewScroll)
	}
	if next.cursor != 0 {
		t.Fatalf("cursor after page up = %d, want unchanged", next.cursor)
	}

	nextModel, _ = next.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	next = nextModel.(model)
	if next.previewScroll != 0 {
		t.Fatalf("previewScroll after page down = %d, want 0", next.previewScroll)
	}
}

func TestMouseWheelInPreviewDoesNotMoveExplorerCursor(t *testing.T) {
	m := model{
		width:  120,
		height: 40,
		hosts: []hostState{{
			host: config.Host{Name: "local", SSH: "local"},
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{
				{ID: "$1", Name: "one"},
				{ID: "$2", Name: "two"},
			}},
			loaded: true,
		}},
		expanded: map[string]bool{"host:0": true},
	}
	preview := testPanelBounds(t, m, panelPreview)
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)

	m.handleMouse(tea.MouseMsg{Type: tea.MouseWheelUp, X: preview.x, Y: preview.y}, now)
	if m.cursor != 0 {
		t.Fatalf("cursor after preview wheel = %d, want unchanged", m.cursor)
	}
	if m.previewScroll != 5 {
		t.Fatalf("previewScroll = %d, want 5", m.previewScroll)
	}
}

func TestPreviewScrollOffsetChangesVisibleLines(t *testing.T) {
	lines := []string{"one", "two", "three", "four", "five"}

	tail := strings.Join(scrolledPreviewLines(lines, 20, 3, 0), "\n")
	if !strings.Contains(tail, "three") || !strings.Contains(tail, "five") || strings.Contains(tail, "two") {
		t.Fatalf("tail view = %q, want last three lines", tail)
	}

	scrolled := strings.Join(scrolledPreviewLines(lines, 20, 3, 2), "\n")
	if !strings.Contains(scrolled, "one") || !strings.Contains(scrolled, "three") || strings.Contains(scrolled, "five") {
		t.Fatalf("scrolled view = %q, want older lines", scrolled)
	}
}

func TestActivityPanelCanScrollIndependently(t *testing.T) {
	m := model{width: 180, height: 40}
	for i := 0; i < 8; i++ {
		m.activities = append(m.activities, agentActivity{
			At:      time.Date(2026, 5, 27, 12, i, 0, 0, time.UTC),
			Source:  agentActivitySession,
			Agent:   "session",
			Target:  fmt.Sprintf("target-%d", i),
			State:   "working",
			Message: fmt.Sprintf("message-%d", i),
		})
	}
	activity := testPanelBounds(t, m, panelActivity)

	m.handleMouse(tea.MouseMsg{Type: tea.MouseWheelDown, X: activity.x, Y: activity.y}, time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC))
	if m.activityScroll != 3 {
		t.Fatalf("activityScroll = %d, want 5", m.activityScroll)
	}
}

func testPanelBounds(t *testing.T, m model, panel focusedPanel) panelBounds {
	t.Helper()
	for _, bounds := range m.panelBounds() {
		if bounds.panel == panel {
			return bounds
		}
	}
	t.Fatalf("panel %q not found in bounds %#v", panel, m.panelBounds())
	return panelBounds{}
}
