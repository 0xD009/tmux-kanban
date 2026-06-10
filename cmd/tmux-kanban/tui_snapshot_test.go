package main

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"tmux-kanban/internal/config"
	"tmux-kanban/internal/tmuxscan"
)

func TestDebugSnapshotIncludesRuntimeAndReviewQueue(t *testing.T) {
	session := tmuxscan.Session{
		ID:   "$1",
		Name: "agents",
		Windows: []tmuxscan.Window{{
			ID:    "@1",
			Index: "0",
			Panes: []tmuxscan.Pane{{ID: "%1", Index: "0", Agent: tmuxscan.AgentCodex}},
		}},
	}
	host := config.Host{Name: "local", Local: true}
	key := sessionStatusKey(host, session)
	m := model{
		cfg: config.Config{Hosts: []config.Host{host}},
		hosts: []hostState{{
			host:     host,
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{session}},
			loaded:   true,
		}},
		statuses:      map[string]sessionStatus{key: sessionNeedReview},
		reviewTargets: map[string]selectedAgentTarget{key: {hostIndex: 0, target: "%1", agent: tmuxscan.AgentCodex}},
		activities: []agentActivity{{
			At:      time.Date(2026, 5, 20, 10, 3, 0, 0, time.Local),
			Source:  agentActivityReview,
			Agent:   "Hermes",
			Target:  "local/agents",
			State:   "replied",
			Message: "review advice ready",
		}},
		preview: previewState{target: "%1", lines: []string{"Do you want to allow this command?"}},
	}

	snapshot := m.debugSnapshot("FARI marked done while working")
	if snapshot.Description != "FARI marked done while working" {
		t.Fatalf("description = %q, want debug note", snapshot.Description)
	}
	if len(snapshot.ReviewQueue) != 1 {
		t.Fatalf("review queue = %d, want 1", len(snapshot.ReviewQueue))
	}
	if snapshot.Runtime.SessionStatuses[key] != "need review" {
		t.Fatalf("runtime statuses = %#v, want need review", snapshot.Runtime.SessionStatuses)
	}
	if len(snapshot.Hosts) != 1 || len(snapshot.Hosts[0].Sessions) != 1 {
		t.Fatalf("hosts = %#v, want one host with one session", snapshot.Hosts)
	}
	if len(snapshot.Activities) != 1 || snapshot.Activities[0].Agent != "Hermes" {
		t.Fatalf("activities = %#v, want Hermes activity", snapshot.Activities)
	}
}

func TestLowercaseDPromptsForSnapshotDescription(t *testing.T) {
	m := model{}
	nextModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if cmd == nil {
		t.Fatalf("snapshot key command = nil, want show cursor command")
	}
	next := nextModel.(model)
	if !next.snapshotInput.active {
		t.Fatalf("snapshotInput.active = false, want true")
	}
	if next.status != "snapshot description" {
		t.Fatalf("status = %q, want snapshot description", next.status)
	}
}

func TestSnapshotDescriptionEnterStartsSave(t *testing.T) {
	m := model{snapshotInput: snapshotDescriptionState{active: true}}
	nextModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("nebula FARI should be working")})
	next := nextModel.(model)
	nextModel, cmd := next.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("snapshot description enter command = nil, want save command")
	}
	next = nextModel.(model)
	if next.snapshotInput.active {
		t.Fatalf("snapshotInput.active = true, want false")
	}
	if next.status != "saving snapshot..." {
		t.Fatalf("status = %q, want saving snapshot", next.status)
	}
}
