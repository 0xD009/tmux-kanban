package main

import (
	"strings"
	"testing"

	"tmux-kanban/internal/config"
	"tmux-kanban/internal/tmuxscan"
)

func TestNeedReviewSessionRowUsesReviewTargetForPreview(t *testing.T) {
	window := tmuxscan.Window{
		ID:     "@1",
		Index:  "0",
		Active: true,
		Panes: []tmuxscan.Pane{
			{ID: "%1", Index: "0", Active: true},
			{ID: "%2", Index: "1", Agent: tmuxscan.AgentCodex},
		},
	}
	session := tmuxscan.Session{ID: "$1", Name: "agents", Windows: []tmuxscan.Window{window}}
	host := config.Host{Name: "local", Local: true}
	key := sessionStatusKey(host, session)
	target := selectedAgentTargetForPane(0, windowKey(0, session.ID, window.ID), session.Name, window, window.Panes[1])
	m := model{
		hosts: []hostState{{
			host:     host,
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{session}},
			loaded:   true,
		}},
		expanded:      map[string]bool{"host:0": true},
		statuses:      map[string]sessionStatus{key: sessionNeedReview},
		reviewTargets: map[string]selectedAgentTarget{key: target},
		cursor:        0,
	}

	row, ok := m.activePreviewRow()
	if !ok {
		t.Fatalf("activePreviewRow() ok = false, want true")
	}
	if row.attachTarget != "%2" || row.agent != tmuxscan.AgentCodex {
		t.Fatalf("preview row = %#v, want codex review pane %%2", row)
	}

	activeTarget, ok := m.activeAgentTarget()
	if !ok {
		t.Fatalf("activeAgentTarget() ok = false, want true")
	}
	if activeTarget.target != "%2" {
		t.Fatalf("active target = %q, want %%2", activeTarget.target)
	}
}

func TestSendChoiceMarksNeedReviewSessionWorking(t *testing.T) {
	window := tmuxscan.Window{
		ID:    "@1",
		Index: "0",
		Panes: []tmuxscan.Pane{{ID: "%1", Index: "0", Agent: tmuxscan.AgentClaude}},
	}
	session := tmuxscan.Session{ID: "$1", Name: "agents", Windows: []tmuxscan.Window{window}}
	host := config.Host{Name: "local", Local: true}
	key := sessionStatusKey(host, session)
	target := selectedAgentTargetForPane(0, windowKey(0, session.ID, window.ID), session.Name, window, window.Panes[0])
	m := model{
		hosts: []hostState{{
			host:     host,
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{session}},
			loaded:   true,
		}},
		expanded:      map[string]bool{"host:0": true},
		statuses:      map[string]sessionStatus{key: sessionNeedReview},
		reviewTargets: map[string]selectedAgentTarget{key: target},
		cursor:        0,
		preview: previewState{lines: []string{
			"Do you want to allow this command?",
			"❯ 1. Allow",
			"  2. Deny",
		}},
	}

	cmd := m.sendChoice("1")
	if cmd == nil {
		t.Fatalf("sendChoice() cmd = nil, want command")
	}
	if got := m.statuses[key]; got != sessionWorking {
		t.Fatalf("session status = %q, want %q", got, sessionWorking)
	}
	if _, ok := m.reviewTargets[key]; ok {
		t.Fatalf("review target still present after sending choice")
	}
}

func TestSendResultStatusUsesFriendlyPaneLabel(t *testing.T) {
	host := config.Host{Name: "local", Local: true}
	m := model{
		hosts: []hostState{{
			host: host,
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{{
				ID:   "$1",
				Name: "agents",
				Windows: []tmuxscan.Window{{
					ID:    "@1",
					Index: "2",
					Panes: []tmuxscan.Pane{{ID: "%1", Index: "3", Agent: tmuxscan.AgentCodex}},
				}},
			}}},
			loaded: true,
		}},
	}

	nextModel, _ := m.Update(sendResult{
		action: "message",
		result: tmuxscan.SendResult{Host: host, Target: "%1"},
	})
	next := nextModel.(model)

	if strings.Contains(next.status, "%1") {
		t.Fatalf("status = %q, want no raw tmux pane id", next.status)
	}
	if want := "message sent to local/agents:2.3 (codex)"; next.status != want {
		t.Fatalf("status = %q, want %q", next.status, want)
	}
	if len(next.activities) != 1 || strings.Contains(next.activities[0].Target, "%1") {
		t.Fatalf("activity target = %#v, want friendly target label", next.activities)
	}
}
