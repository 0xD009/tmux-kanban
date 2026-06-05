package main

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"tmux-kanban/internal/config"
	"tmux-kanban/internal/tmuxscan"
)

func TestSplitWorkspaceHeightsAddsUpToRequestedHeight(t *testing.T) {
	for _, height := range []int{2, 8, 14, 18, 19, 24, 32, 48} {
		hostHeight, previewHeight := splitWorkspaceHeights(height)
		if hostHeight+previewHeight != height {
			t.Fatalf("height %d split to %d+%d", height, hostHeight, previewHeight)
		}
		if height >= 14 && hostHeight < 8 {
			t.Fatalf("height %d hostHeight = %d, want at least 8", height, hostHeight)
		}
		if height >= 14 && previewHeight < 6 {
			t.Fatalf("height %d previewHeight = %d, want at least 6", height, previewHeight)
		}
	}

	hostHeight, previewHeight := splitWorkspaceHeights(36)
	if previewHeight <= hostHeight {
		t.Fatalf("36 split to host=%d preview=%d, want preview larger", hostHeight, previewHeight)
	}
	if hostHeight != 8 {
		t.Fatalf("36 split to host=%d preview=%d, want terminal preview raised to host minimum", hostHeight, previewHeight)
	}
}

func TestWorkspacePanelsRenderToRequestedHeight(t *testing.T) {
	m := model{
		hosts: []hostState{{
			host: config.Host{Name: "local", Local: true},
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{{
				ID:   "$1",
				Name: "agents",
			}}},
			loaded: true,
		}},
		expanded: map[string]bool{"host:0": true},
		statuses: map[string]sessionStatus{},
		cursor:   0,
	}

	for _, height := range []int{18, 24, 36, 48} {
		if got := lipgloss.Height(m.renderWorkspace(80, height, 5, 40)); got != height {
			t.Fatalf("workspace height = %d, want %d", got, height)
		}
		if got := lipgloss.Height(m.renderKanban(36, height)); got != height {
			t.Fatalf("kanban height = %d, want %d", got, height)
		}
	}
}

func TestMainSessionKanbanLinesStayWithinPanelWidth(t *testing.T) {
	cfg := config.Default()
	host := config.Host{Name: "local", Local: true}
	mainSession := tmuxscan.Session{ID: "$main", Name: "tmux-kanban-main"}
	m := model{
		cfg: cfg,
		hosts: []hostState{{
			host:     host,
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{mainSession}},
			loaded:   true,
		}},
		statuses: map[string]sessionStatus{sessionStatusKey(host, mainSession): sessionWorking},
	}

	view := m.renderKanban(38, 36)
	if got := lipgloss.Height(view); got != 36 {
		t.Fatalf("kanban height = %d, want 36", got)
	}
	lines := strings.Split(view, "\n")
	frameWidth := lipgloss.Width(lines[0])
	for i, line := range lines {
		if width := lipgloss.Width(line); width > frameWidth {
			t.Fatalf("line %d width = %d, want <= frame width %d: %q", i+1, width, frameWidth, line)
		}
	}
}

func TestSidePanelWidthsLeavePreviewRoom(t *testing.T) {
	for _, totalWidth := range []int{140, 148, 160, 200, 240} {
		left := threeColumnSideWidth(totalWidth)
		right := threeColumnActivityWidth(totalWidth, left)
		preview := totalWidth - left - right - 4
		if left < 38 || right < 38 {
			t.Fatalf("three-column sides at %d = %d/%d, want at least 38", totalWidth, left, right)
		}
		if preview < 60 {
			t.Fatalf("three-column preview at %d = %d, want at least 60", totalWidth, preview)
		}
		if right*3 < preview {
			t.Fatalf("three-column activity at %d = %d, want at least one third of preview %d", totalWidth, right, preview)
		}
	}

	for _, totalWidth := range []int{100, 104, 120, 160} {
		side := twoColumnSideWidth(totalWidth)
		preview := totalWidth - side - 2
		if side < 38 {
			t.Fatalf("two-column side at %d = %d, want at least 38", totalWidth, side)
		}
		if preview < 60 {
			t.Fatalf("two-column preview at %d = %d, want at least 60", totalWidth, preview)
		}
	}
}

func TestAddAgentActivityCapsHistory(t *testing.T) {
	m := model{}
	for i := 0; i < maxAgentActivities+5; i++ {
		m.addAgentActivity(agentActivity{
			At:      time.Unix(int64(i), 0),
			Source:  agentActivitySession,
			Agent:   "codex",
			Target:  "local/agents",
			State:   "working",
			Message: "status changed",
		})
	}

	if len(m.activities) != maxAgentActivities {
		t.Fatalf("activity count = %d, want %d", len(m.activities), maxAgentActivities)
	}
	if got := m.activities[0].At.Unix(); got != 5 {
		t.Fatalf("oldest retained activity = %d, want 5", got)
	}
}

func TestRenderAgentActivityShowsSessionAndReviewEvents(t *testing.T) {
	m := model{}
	m.addAgentActivity(agentActivity{
		At:      time.Date(2026, 5, 20, 10, 1, 0, 0, time.Local),
		Source:  agentActivitySession,
		Agent:   "codex",
		Target:  "local/agents:2.3",
		State:   "need review",
		Message: "status changed",
	})
	m.addAgentActivity(agentActivity{
		At:      time.Date(2026, 5, 20, 10, 2, 0, 0, time.Local),
		Source:  agentActivityReview,
		Agent:   "Hermes",
		Target:  "local/agents",
		State:   "replied",
		Message: "CHOOSE 1\nThe visible prompt is a normal approval request.",
	})

	view := m.renderAgentActivity(34, 18)
	plain := ansi.Strip(view)
	for _, want := range []string{"Agent Activity", "SESSION", "REVIEW", "Hermes", "Q: local/agents", "A: CHOOSE 1", "visible prompt"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("activity view missing %q:\n%s", want, plain)
		}
	}
	if got := lipgloss.Height(view); got != 18 {
		t.Fatalf("activity view height = %d, want 18", got)
	}
}

