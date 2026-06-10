package main

import (
	"strings"
	"testing"
	"time"

	"tmux-kanban/internal/config"
	"tmux-kanban/internal/tmuxscan"
)

func TestSessionStatusFromAgentScreen(t *testing.T) {
	tests := []struct {
		name   string
		screen tmuxscan.AgentScreen
		want   sessionStatus
		ok     bool
	}{
		{
			name: "choices need review",
			screen: tmuxscan.AgentScreen{
				Choices:     []tmuxscan.AgentChoice{{Number: "1", Label: "Allow"}},
				Idle:        true,
				NeedsReview: true,
			},
			want: sessionNeedReview,
			ok:   true,
		},
		{
			name: "idle menu choices stay idle",
			screen: tmuxscan.AgentScreen{
				Choices: []tmuxscan.AgentChoice{{Label: "New task", Selected: true}},
				Idle:    true,
			},
			want: sessionIdle,
			ok:   true,
		},
		{
			name:   "busy working",
			screen: tmuxscan.AgentScreen{Busy: true},
			want:   sessionWorking,
			ok:     true,
		},
		{
			name: "busy wins over visible prompt",
			screen: tmuxscan.AgentScreen{
				Busy:    true,
				Idle:    true,
				Choices: []tmuxscan.AgentChoice{{Label: "Run /review on my current changes", Selected: true}},
			},
			want: sessionWorking,
			ok:   true,
		},
		{
			name:   "idle",
			screen: tmuxscan.AgentScreen{Idle: true},
			want:   sessionIdle,
			ok:     true,
		},
		{
			name:   "unknown",
			screen: tmuxscan.AgentScreen{},
			want:   "",
			ok:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := sessionStatusFromAgentScreen(tt.screen)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("status = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReviewQueueIncludesNeedReviewAgentSessions(t *testing.T) {
	first := tmuxscan.Session{
		ID:   "$1",
		Name: "needs-review",
		Windows: []tmuxscan.Window{{
			ID:    "@1",
			Index: "0",
			Panes: []tmuxscan.Pane{{ID: "%1", Index: "0", Agent: tmuxscan.AgentCodex}},
		}},
	}
	second := tmuxscan.Session{
		ID:   "$2",
		Name: "idle",
		Windows: []tmuxscan.Window{{
			ID:    "@2",
			Index: "0",
			Panes: []tmuxscan.Pane{{ID: "%2", Index: "0", Agent: tmuxscan.AgentClaude}},
		}},
	}
	host := config.Host{Name: "local", Local: true}
	m := model{
		hosts: []hostState{{
			host:     host,
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{first, second}},
			loaded:   true,
		}},
		statuses: map[string]sessionStatus{
			sessionStatusKey(host, first):  sessionNeedReview,
			sessionStatusKey(host, second): sessionIdle,
		},
	}

	queue := m.reviewQueue()
	if len(queue) != 1 {
		t.Fatalf("review queue = %d, want 1", len(queue))
	}
	if queue[0].SessionName != "needs-review" {
		t.Fatalf("session = %q, want needs-review", queue[0].SessionName)
	}
	if queue[0].Row.attachTarget != "%1" {
		t.Fatalf("target = %q, want %%1", queue[0].Row.attachTarget)
	}
}

func TestReviewQueueUsesDetectedNeedReviewPaneInMultiAgentSession(t *testing.T) {
	session := tmuxscan.Session{
		ID:   "$1",
		Name: "nemotron",
		Windows: []tmuxscan.Window{
			{
				ID:    "@1",
				Index: "0",
				Panes: []tmuxscan.Pane{{ID: "%1", Index: "0", Agent: tmuxscan.AgentClaude}},
			},
			{
				ID:    "@2",
				Index: "1",
				Panes: []tmuxscan.Pane{{ID: "%2", Index: "0", Agent: tmuxscan.AgentCodex}},
			},
		},
	}
	host := config.Host{Name: "wmg22008", SSH: "wmg22008"}
	key := sessionStatusKey(host, session)
	m := model{
		hosts: []hostState{{
			host:     host,
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{session}},
			loaded:   true,
		}},
		statuses: map[string]sessionStatus{key: sessionNeedReview},
		reviewTargets: map[string]selectedAgentTarget{key: {
			key:       "host:0:session:$1:window:@2:pane:%2",
			hostIndex: 0,
			target:    "%2",
			agent:     tmuxscan.AgentCodex,
		}},
	}

	queue := m.reviewQueue()
	if len(queue) != 1 {
		t.Fatalf("review queue = %d, want 1", len(queue))
	}
	if queue[0].Row.attachTarget != "%2" {
		t.Fatalf("target = %q, want %%2", queue[0].Row.attachTarget)
	}
	if queue[0].Agent != tmuxscan.AgentCodex {
		t.Fatalf("agent = %q, want codex", queue[0].Agent)
	}
}

func TestAgentTargetsInSessionReturnsAllAgentPanes(t *testing.T) {
	session := tmuxscan.Session{
		ID:   "$1",
		Name: "agents",
		Windows: []tmuxscan.Window{{
			ID:    "@1",
			Index: "0",
			Panes: []tmuxscan.Pane{
				{ID: "%1", Index: "0", Agent: tmuxscan.AgentClaude},
				{ID: "%2", Index: "1", Agent: tmuxscan.AgentCodex},
			},
		}},
	}

	targets := agentTargetsInSession(0, "host:0:session:$1", session)
	if len(targets) != 2 {
		t.Fatalf("targets = %d, want 2", len(targets))
	}
	if targets[0].target != "%1" || targets[1].target != "%2" {
		t.Fatalf("targets = %#v, want both panes", targets)
	}
}

func TestAgentStatusResultStoresAndClearsReviewTarget(t *testing.T) {
	target := selectedAgentTarget{hostIndex: 0, target: "nemotron:1.0", agent: tmuxscan.AgentCodex}
	m := model{
		statuses:      map[string]sessionStatus{},
		reviewTargets: map[string]selectedAgentTarget{},
	}

	nextModel, _ := m.Update(agentStatusResult{key: "session", status: sessionNeedReview, target: target, ok: true})
	next := nextModel.(model)
	if next.statuses["session"] != sessionNeedReview {
		t.Fatalf("status = %q, want need review", next.statuses["session"])
	}
	if next.reviewTargets["session"].target != "nemotron:1.0" {
		t.Fatalf("review target = %#v, want nemotron:1.0", next.reviewTargets["session"])
	}

	nextModel, _ = next.Update(agentStatusResult{key: "session", status: sessionIdle, target: target, ok: true})
	next = nextModel.(model)
	if _, ok := next.reviewTargets["session"]; ok {
		t.Fatalf("review target still present after idle status")
	}
}

func TestStartScanPreservesPreviousSnapshotWhileRefreshing(t *testing.T) {
	m := model{
		hosts: []hostState{{
			host: config.Host{Name: "local", Local: true},
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{{
				ID:   "$1",
				Name: "agents",
			}}},
			loaded: true,
		}},
	}

	next, cmd := m.startScanModel()
	if cmd == nil {
		t.Fatalf("startScanModel() cmd = nil, want scan command")
	}
	if !next.hosts[0].loading {
		t.Fatalf("loading = false, want true")
	}
	if !next.hosts[0].loaded {
		t.Fatalf("loaded = false, want previous snapshot preserved")
	}
	if len(next.hosts[0].snapshot.Sessions) != 1 {
		t.Fatalf("sessions = %d, want previous snapshot preserved", len(next.hosts[0].snapshot.Sessions))
	}
}

func TestRowsKeepPreviousSessionsWhileHostRefreshes(t *testing.T) {
	m := model{
		hosts: []hostState{{
			host: config.Host{Name: "local", Local: true},
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{{
				ID:   "$1",
				Name: "agents",
			}}},
			loading: true,
			loaded:  true,
		}},
		expanded: map[string]bool{"host:0": true},
	}

	rows := m.rows()
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want previous session", len(rows))
	}
	if rows[0].kind != rowSession || rows[0].attachTarget != "agents" {
		t.Fatalf("row[0] = %#v, want previous session row", rows[0])
	}
}

func TestSessionLabelDoesNotShowDefaultIdleBeforeStatusKnown(t *testing.T) {
	host := config.Host{Name: "local", Local: true}
	session := tmuxscan.Session{ID: "$1", Name: "agents"}
	m := model{hosts: []hostState{{host: host}}}

	label := m.sessionLabel(0, session)
	if strings.Contains(label, "[idle]") {
		t.Fatalf("label = %q, want no default idle badge", label)
	}

	m.statuses = map[string]sessionStatus{sessionStatusKey(host, session): sessionWorking}
	label = m.sessionLabel(0, session)
	if !strings.Contains(label, "[working]") {
		t.Fatalf("label = %q, want working badge", label)
	}
}

func TestHostLabelDoesNotFlashScanningForLoadedBackgroundRefresh(t *testing.T) {
	host := config.Host{Name: "local", Local: true}
	m := model{expanded: map[string]bool{"host:0": true}}
	state := hostState{
		host:    host,
		loading: true,
		loaded:  true,
		snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{
			{ID: "$1", Name: "agents"},
		}},
	}

	label := m.hostLabel(0, state)
	if strings.Contains(label, "scanning") {
		t.Fatalf("label = %q, want stable loaded host label without scanning", label)
	}
	if !strings.Contains(label, "1 sessions") {
		t.Fatalf("label = %q, want session count retained", label)
	}
}

func TestBackgroundScanDoesNotOverwriteStatusLine(t *testing.T) {
	m := model{
		status: "message sent",
		hosts:  []hostState{{host: config.Host{Name: "local", Local: true}, loaded: true}},
	}

	next, cmd := m.startBackgroundScanModel()
	if cmd == nil {
		t.Fatalf("startBackgroundScanModel() cmd = nil, want scan command")
	}
	if next.status != "message sent" {
		t.Fatalf("status = %q, want previous status retained", next.status)
	}
	if next.scanAnnounce {
		t.Fatalf("scanAnnounce = true, want false for background scan")
	}
}

func TestWorkingStatusMovesToDoneOnIdlePoll(t *testing.T) {
	key := "local:$1"
	m := model{
		statuses:      map[string]sessionStatus{key: sessionWorking},
		statusStreaks: map[string]statusStreak{},
		reviewTargets: map[string]selectedAgentTarget{},
	}

	m.applyAgentStatusResult(agentStatusResult{key: key, status: sessionIdle, ok: true})
	if got := m.statuses[key]; got != sessionDone {
		t.Fatalf("status after idle = %q, want done", got)
	}
}

func TestWorkingToDoneCanBeOverwrittenByLaterWorkingPoll(t *testing.T) {
	key := "local:$1"
	m := model{
		statuses:      map[string]sessionStatus{key: sessionWorking},
		statusStreaks: map[string]statusStreak{},
		reviewTargets: map[string]selectedAgentTarget{},
	}

	m.applyAgentStatusResult(agentStatusResult{key: key, status: sessionIdle, ok: true})
	m.applyAgentStatusResult(agentStatusResult{key: key, status: sessionWorking, ok: true})
	if got := m.statuses[key]; got != sessionWorking {
		t.Fatalf("status after later working poll = %q, want working", got)
	}
}

func TestDoneStatusIsOnlyStickyOverIdlePolling(t *testing.T) {
	key := "local:$1"
	m := model{
		statuses:      map[string]sessionStatus{key: sessionDone},
		statusStreaks: map[string]statusStreak{},
		reviewTargets: map[string]selectedAgentTarget{},
	}

	m.applyAgentStatusResult(agentStatusResult{key: key, status: sessionIdle, ok: true})
	if got := m.statuses[key]; got != sessionDone {
		t.Fatalf("status after idle poll = %q, want done", got)
	}

	m.applyAgentStatusResult(agentStatusResult{key: key, status: sessionWorking, ok: true})
	if got := m.statuses[key]; got != sessionWorking {
		t.Fatalf("status after working poll = %q, want working", got)
	}
}

func TestNeedReviewStatusCanReviveDoneStatus(t *testing.T) {
	key := "local:$1"
	target := selectedAgentTarget{target: "%1"}
	m := model{
		statuses:      map[string]sessionStatus{key: sessionDone},
		statusStreaks: map[string]statusStreak{},
		reviewTargets: map[string]selectedAgentTarget{},
	}

	m.applyAgentStatusResult(agentStatusResult{key: key, status: sessionNeedReview, ok: true, target: target})
	if got := m.statuses[key]; got != sessionNeedReview {
		t.Fatalf("status after need-review poll = %q, want need review", got)
	}
	if got := m.reviewTargets[key].target; got != "%1" {
		t.Fatalf("review target = %q, want %%1", got)
	}
}

func TestCarrySessionStatusesAcrossRefreshByName(t *testing.T) {
	host := config.Host{Name: "local", Local: true}
	oldSession := tmuxscan.Session{ID: "$1", Name: "agents"}
	newSession := tmuxscan.Session{ID: "$2", Name: "agents"}
	m := model{
		hosts: []hostState{{
			host:     host,
			snapshot: tmuxscan.Snapshot{Host: host, Sessions: []tmuxscan.Session{oldSession}},
			loaded:   true,
		}},
		statuses: map[string]sessionStatus{sessionStatusKey(host, oldSession): sessionNeedReview},
	}

	m.carrySessionStatuses(0, tmuxscan.Snapshot{Host: host, Sessions: []tmuxscan.Session{newSession}})
	if got := m.statuses[sessionStatusKey(host, newSession)]; got != sessionNeedReview {
		t.Fatalf("carried status = %q, want need review", got)
	}
}

func TestEnsurePreviewUsesCachedLinesWhileRefreshing(t *testing.T) {
	m := model{
		hosts: []hostState{{
			host: config.Host{Name: "local", SSH: "local"},
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{{
				ID:   "$1",
				Name: "agents",
			}}},
			loaded: true,
		}},
		expanded: map[string]bool{"host:0": true},
		cache:    map[string]previewCacheEntry{},
		cursor:   0,
	}
	selected, ok := m.selectedRow()
	if !ok {
		t.Fatalf("selectedRow() ok = false, want true")
	}
	key := previewKey(selected)
	m.cache[key] = previewCacheEntry{
		lines:      []string{"cached preview"},
		capturedAt: time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC),
	}

	cmd := m.ensurePreview()
	if cmd == nil {
		t.Fatalf("ensurePreview() cmd = nil, want refresh command")
	}
	if m.preview.loading {
		t.Fatalf("preview.loading = true, want false")
	}
	if !m.preview.refreshing {
		t.Fatalf("preview.refreshing = false, want true")
	}
	if len(m.preview.lines) != 1 || m.preview.lines[0] != "cached preview" {
		t.Fatalf("preview lines = %#v, want cached line", m.preview.lines)
	}
}

func TestCaptureResultUpdatesPreviewedAgentStatus(t *testing.T) {
	session := tmuxscan.Session{
		ID:   "$27",
		Name: "FARI",
		Windows: []tmuxscan.Window{{
			ID:     "@31",
			Index:  "0",
			Active: true,
			Panes: []tmuxscan.Pane{{
				ID:     "%31",
				Index:  "0",
				Active: true,
				Agent:  tmuxscan.AgentCodex,
			}},
		}},
	}
	host := config.Host{Name: "nebula", SSH: "nebula"}
	m := model{
		hosts: []hostState{{
			host:     host,
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{session}},
			loaded:   true,
		}},
		expanded:      map[string]bool{"host:0": true},
		statuses:      map[string]sessionStatus{sessionStatusKey(host, session): sessionIdle},
		statusStreaks:  map[string]statusStreak{},
		reviewTargets: map[string]selectedAgentTarget{},
		cache:         map[string]previewCacheEntry{},
		cursor:        0,
	}
	selected, ok := m.selectedRow()
	if !ok {
		t.Fatalf("selectedRow() ok = false, want true")
	}
	key := previewKey(selected)
	m.preview = previewState{key: key, hostIndex: 0, target: selected.attachTarget, refreshing: true}

	nextModel, _ := m.Update(captureResult{
		key: key,
		capture: tmuxscan.Capture{
			Host:   host,
			Target: selected.attachTarget,
			Lines: []string{
				"• Working (44s • esc to interrupt)",
			},
			CapturedAt: time.Date(2026, 5, 21, 17, 38, 20, 0, time.Local),
		},
	})

	next := nextModel.(model)
	statusKey := sessionStatusKey(host, session)
	if got := next.statuses[statusKey]; got != sessionWorking {
		t.Fatalf("status = %q, want working", got)
	}
	if next.preview.refreshing {
		t.Fatalf("preview.refreshing = true, want false")
	}
}

func TestCachePreviewPreservesLinesOnRefreshError(t *testing.T) {
	m := model{cache: map[string]previewCacheEntry{
		"preview": {lines: []string{"old frame"}},
	}}

	entry := m.cachePreview("preview", tmuxscan.Capture{
		Err:        "ssh failed",
		CapturedAt: time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC),
	})

	if len(entry.lines) != 1 || entry.lines[0] != "old frame" {
		t.Fatalf("entry lines = %#v, want old frame", entry.lines)
	}
	if entry.err != "ssh failed" {
		t.Fatalf("entry err = %q, want ssh failed", entry.err)
	}
}
