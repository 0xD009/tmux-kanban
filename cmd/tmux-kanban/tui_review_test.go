package main

import (
	"strings"
	"testing"

	"tmux-kanban/internal/config"
	"tmux-kanban/internal/tmuxscan"
)

func TestSkipReviewItemTemporarilyHidesIt(t *testing.T) {
	session := tmuxscan.Session{
		ID:   "$1",
		Name: "needs-review",
		Windows: []tmuxscan.Window{{
			ID:    "@1",
			Index: "0",
			Panes: []tmuxscan.Pane{{ID: "%1", Index: "0", Agent: tmuxscan.AgentCodex}},
		}},
	}
	host := config.Host{Name: "local", Local: true}
	m := model{
		hosts: []hostState{{
			host:     host,
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{session}},
			loaded:   true,
		}},
		statuses: map[string]sessionStatus{sessionStatusKey(host, session): sessionNeedReview},
		viewMode: viewReview,
	}

	m.skipReviewItem()
	if len(m.reviewQueue()) != 0 {
		t.Fatalf("queue length after skip = %d, want 0", len(m.reviewQueue()))
	}
	if m.skippedReviewCount() != 1 {
		t.Fatalf("skipped count = %d, want 1", m.skippedReviewCount())
	}

	m.unskipReviewItems()
	if len(m.reviewQueue()) != 1 {
		t.Fatalf("queue length after unskip = %d, want 1", len(m.reviewQueue()))
	}
}

func TestReviewCursorTracksStableSessionKey(t *testing.T) {
	first := tmuxscan.Session{
		ID:   "$1",
		Name: "first",
		Windows: []tmuxscan.Window{{
			ID:    "@1",
			Index: "0",
			Panes: []tmuxscan.Pane{{ID: "%1", Index: "0", Agent: tmuxscan.AgentCodex}},
		}},
	}
	second := tmuxscan.Session{
		ID:   "$2",
		Name: "second",
		Windows: []tmuxscan.Window{{
			ID:    "@2",
			Index: "0",
			Panes: []tmuxscan.Pane{{ID: "%2", Index: "0", Agent: tmuxscan.AgentClaude}},
		}},
	}
	third := tmuxscan.Session{
		ID:   "$3",
		Name: "third",
		Windows: []tmuxscan.Window{{
			ID:    "@3",
			Index: "0",
			Panes: []tmuxscan.Pane{{ID: "%3", Index: "0", Agent: tmuxscan.AgentCodex}},
		}},
	}
	host := config.Host{Name: "local", Local: true}
	firstKey := sessionStatusKey(host, first)
	secondKey := sessionStatusKey(host, second)
	thirdKey := sessionStatusKey(host, third)
	m := model{
		hosts: []hostState{{
			host:     host,
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{first, second, third}},
			loaded:   true,
		}},
		statuses: map[string]sessionStatus{
			firstKey:  sessionNeedReview,
			secondKey: sessionNeedReview,
			thirdKey:  sessionNeedReview,
		},
		viewMode: viewReview,
	}

	m.moveReviewCursor(1)
	if item, ok := m.currentReviewItem(); !ok || item.SessionKey != secondKey {
		t.Fatalf("current after move = %#v, %v; want second", item, ok)
	}

	m.statuses[firstKey] = sessionIdle
	m.clampReviewCursor()
	if item, ok := m.currentReviewItem(); !ok || item.SessionKey != secondKey {
		t.Fatalf("current after queue shrink = %#v, %v; want second", item, ok)
	}

	m.skipReviewItem()
	if item, ok := m.currentReviewItem(); !ok || item.SessionKey != thirdKey {
		t.Fatalf("current after skip = %#v, %v; want third", item, ok)
	}
}

func TestSendChoiceInReviewModeAdvancesQueue(t *testing.T) {
	first := tmuxscan.Session{
		ID:   "$1",
		Name: "first",
		Windows: []tmuxscan.Window{{
			ID:    "@1",
			Index: "0",
			Panes: []tmuxscan.Pane{{ID: "%1", Index: "0", Agent: tmuxscan.AgentCodex}},
		}},
	}
	second := tmuxscan.Session{
		ID:   "$2",
		Name: "second",
		Windows: []tmuxscan.Window{{
			ID:    "@2",
			Index: "0",
			Panes: []tmuxscan.Pane{{ID: "%2", Index: "0", Agent: tmuxscan.AgentClaude}},
		}},
	}
	host := config.Host{Name: "local", Local: true}
	firstKey := sessionStatusKey(host, first)
	secondKey := sessionStatusKey(host, second)
	m := model{
		hosts: []hostState{{
			host:     host,
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{first, second}},
			loaded:   true,
		}},
		statuses: map[string]sessionStatus{
			firstKey:  sessionNeedReview,
			secondKey: sessionNeedReview,
		},
		viewMode: viewReview,
		preview: previewState{
			lines: []string{
				"Do you want to allow this command?",
				"❯ 1. Allow",
				"  2. Deny",
			},
		},
	}

	cmd := m.sendChoice("1")
	if cmd == nil {
		t.Fatalf("sendChoice() cmd = nil, want command")
	}
	if got := m.statuses[firstKey]; got != sessionWorking {
		t.Fatalf("first status = %q, want %q", got, sessionWorking)
	}
	item, ok := m.currentReviewItem()
	if !ok {
		t.Fatalf("currentReviewItem() ok = false, want second item")
	}
	if item.SessionKey != secondKey {
		t.Fatalf("current session key = %q, want %q", item.SessionKey, secondKey)
	}
}

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
		viewMode:      viewTree,
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

func TestSendChoiceInTreeModeMarksNeedReviewSessionWorking(t *testing.T) {
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
		viewMode:      viewTree,
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
