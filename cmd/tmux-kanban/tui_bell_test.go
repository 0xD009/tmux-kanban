package main

import (
	"reflect"
	"strings"
	"testing"

	"tmux-kanban/internal/config"
	"tmux-kanban/internal/tmuxscan"
)

func TestNeedReviewBellOnlyWhenEnteringNeedReview(t *testing.T) {
	original := writeTerminalBell
	defer func() { writeTerminalBell = original }()

	tests := []struct {
		name       string
		hadOld     bool
		oldStatus  sessionStatus
		nextStatus sessionStatus
		wantBell   bool
	}{
		{name: "first need review", nextStatus: sessionNeedReview, wantBell: true},
		{name: "working to need review", hadOld: true, oldStatus: sessionWorking, nextStatus: sessionNeedReview, wantBell: true},
		{name: "done to need review", hadOld: true, oldStatus: sessionDone, nextStatus: sessionNeedReview, wantBell: true},
		{name: "already need review", hadOld: true, oldStatus: sessionNeedReview, nextStatus: sessionNeedReview},
		{name: "not need review", hadOld: true, oldStatus: sessionWorking, nextStatus: sessionWorking},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bells := 0
			writeTerminalBell = func() { bells++ }

			cmd := needReviewBellCmd(true, tt.hadOld, tt.oldStatus, tt.nextStatus, false)
			if cmd == nil {
				if tt.wantBell {
					t.Fatalf("needReviewBellCmd() = nil, want bell command")
				}
				return
			}
			if !tt.wantBell {
				t.Fatalf("needReviewBellCmd() returned command, want nil")
			}

			cmd()
			if bells != 1 {
				t.Fatalf("bells = %d, want 1", bells)
			}
		})
	}
}

func TestNeedReviewBellSkipsWhenHermesHandlesReview(t *testing.T) {
	cmd := needReviewBellCmd(true, true, sessionWorking, sessionNeedReview, true)
	if cmd != nil {
		t.Fatalf("needReviewBellCmd() with Hermes handling = %#v, want nil", cmd)
	}
}

func TestNeedReviewBellDisabledByDefaultSetting(t *testing.T) {
	cmd := needReviewBellCmd(false, true, sessionWorking, sessionNeedReview, false)
	if cmd != nil {
		t.Fatalf("needReviewBellCmd() with terminal review disabled = %#v, want nil", cmd)
	}
}

func TestNeedReviewTerminalAlertSequenceSetsTabTitle(t *testing.T) {
	sequence := string(needReviewTerminalAlertSequence())
	for _, want := range []string{"\a", "\x1b]1;tmux-kanban: NEED REVIEW\x1b\\", "\x1b]2;tmux-kanban: NEED REVIEW\x1b\\"} {
		if !strings.Contains(sequence, want) {
			t.Fatalf("terminal alert sequence missing %q: %#v", want, sequence)
		}
	}
}

func TestReviewTerminalTitleSequenceCanClearNeedReview(t *testing.T) {
	sequence := string(reviewTerminalTitleSequence(false))
	if strings.Contains(sequence, "NEED REVIEW") {
		t.Fatalf("clear sequence should not contain NEED REVIEW: %#v", sequence)
	}
	for _, want := range []string{"\x1b]1;tmux-kanban\x1b\\", "\x1b]2;tmux-kanban\x1b\\"} {
		if !strings.Contains(sequence, want) {
			t.Fatalf("clear sequence missing %q: %#v", want, sequence)
		}
	}
}

func TestReviewTerminalTitleSyncTracksReviewQueue(t *testing.T) {
	original := writeReviewTerminalTitle
	defer func() { writeReviewTerminalTitle = original }()

	calls := []bool{}
	writeReviewTerminalTitle = func(active bool) {
		calls = append(calls, active)
	}

	host := config.Host{Name: "local", Local: true}
	session := tmuxscan.Session{
		ID:   "$1",
		Name: "agent",
		Windows: []tmuxscan.Window{{
			ID:    "@1",
			Index: "0",
			Panes: []tmuxscan.Pane{{ID: "%1", Index: "0", Agent: tmuxscan.AgentCodex}},
		}},
	}
	key := sessionStatusKey(host, session)
	m := model{
		cfg: config.Config{Notification: config.NotificationConfig{TerminalReview: true}},
		hosts: []hostState{{
			host:     host,
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{session}},
			loaded:   true,
		}},
		statuses:      map[string]sessionStatus{key: sessionNeedReview},
		reviewTargets: map[string]selectedAgentTarget{key: {hostIndex: 0, target: "%1", agent: tmuxscan.AgentCodex}},
	}

	cmd := m.syncReviewTerminalTitleCmd()
	if cmd == nil {
		t.Fatalf("syncReviewTerminalTitleCmd() = nil, want command to set title")
	}
	cmd()
	if !m.reviewTitleActive {
		t.Fatalf("reviewTitleActive = false, want true")
	}
	if !reflect.DeepEqual(calls, []bool{true}) {
		t.Fatalf("title calls = %#v, want [true]", calls)
	}

	m.statuses[key] = sessionWorking
	delete(m.reviewTargets, key)
	cmd = m.syncReviewTerminalTitleCmd()
	if cmd == nil {
		t.Fatalf("syncReviewTerminalTitleCmd() = nil, want command to clear title")
	}
	cmd()
	if m.reviewTitleActive {
		t.Fatalf("reviewTitleActive = true, want false")
	}
	if !reflect.DeepEqual(calls, []bool{true, false}) {
		t.Fatalf("title calls = %#v, want [true false]", calls)
	}
}

func TestReviewTerminalTitleSyncDisabledDoesNotSetNeedReview(t *testing.T) {
	original := writeReviewTerminalTitle
	defer func() { writeReviewTerminalTitle = original }()

	calls := []bool{}
	writeReviewTerminalTitle = func(active bool) {
		calls = append(calls, active)
	}

	host := config.Host{Name: "local", Local: true}
	session := tmuxscan.Session{
		ID:   "$1",
		Name: "agent",
		Windows: []tmuxscan.Window{{
			ID:    "@1",
			Index: "0",
			Panes: []tmuxscan.Pane{{ID: "%1", Index: "0", Agent: tmuxscan.AgentCodex}},
		}},
	}
	key := sessionStatusKey(host, session)
	m := model{
		hosts: []hostState{{
			host:     host,
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{session}},
			loaded:   true,
		}},
		statuses:      map[string]sessionStatus{key: sessionNeedReview},
		reviewTargets: map[string]selectedAgentTarget{key: {hostIndex: 0, target: "%1", agent: tmuxscan.AgentCodex}},
	}

	if cmd := m.syncReviewTerminalTitleCmd(); cmd != nil {
		t.Fatalf("syncReviewTerminalTitleCmd() with terminal review disabled = %#v, want nil", cmd)
	}
	if len(calls) != 0 {
		t.Fatalf("title calls = %#v, want none", calls)
	}

	m.reviewTitleActive = true
	cmd := m.syncReviewTerminalTitleCmd()
	if cmd == nil {
		t.Fatalf("syncReviewTerminalTitleCmd() disabled with active title = nil, want clear command")
	}
	cmd()
	if !reflect.DeepEqual(calls, []bool{false}) {
		t.Fatalf("title calls = %#v, want [false]", calls)
	}
}

