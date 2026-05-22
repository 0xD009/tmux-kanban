package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"tmux-kanban/internal/config"
	"tmux-kanban/internal/mesh"
	"tmux-kanban/internal/tmuxscan"
)

func TestChoiceKeysUsesRemoteMovementWhenSelectionKnown(t *testing.T) {
	screen := tmuxscan.AgentScreen{
		SelectedChoice: 0,
		Choices: []tmuxscan.AgentChoice{
			{Number: "1", Label: "Allow", Selected: true},
			{Number: "2", Label: "Deny"},
			{Number: "3", Label: "Always allow"},
		},
	}

	got := choiceKeys(screen, "3")
	want := []string{"Down", "Down", "C-m"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("choiceKeys() = %#v, want %#v", got, want)
	}
}

func TestChoiceKeysFallsBackToMovementFromFirstChoice(t *testing.T) {
	screen := tmuxscan.AgentScreen{
		SelectedChoice: -1,
		Choices: []tmuxscan.AgentChoice{
			{Number: "1", Label: "Allow"},
			{Number: "2", Label: "Deny"},
			{Number: "3", Label: "Always allow"},
		},
	}

	got := choiceKeys(screen, "3")
	want := []string{"Down", "Down", "C-m"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("choiceKeys() = %#v, want %#v", got, want)
	}
}

func TestChoiceKeysCanMoveUnnumberedChoiceWhenSelectionUnknown(t *testing.T) {
	screen := tmuxscan.AgentScreen{
		SelectedChoice: -1,
		Choices: []tmuxscan.AgentChoice{
			{Label: "Allow"},
			{Label: "Deny"},
		},
	}

	got := choiceKeys(screen, "2")
	want := []string{"Down", "C-m"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("choiceKeys() = %#v, want %#v", got, want)
	}
}

func TestSelectedAgentTargetFallsThroughFromSession(t *testing.T) {
	m := model{
		hosts: []hostState{{
			host: config.Host{Name: "local", SSH: "local"},
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{{
				ID:   "$1",
				Name: "agents",
				Windows: []tmuxscan.Window{
					{
						ID:    "@1",
						Index: "0",
						Panes: []tmuxscan.Pane{
							{ID: "%1", Index: "0", Agent: tmuxscan.AgentCodex},
						},
					},
					{
						ID:     "@2",
						Index:  "1",
						Active: true,
						Panes: []tmuxscan.Pane{
							{ID: "%2", Index: "0", Active: true, Agent: tmuxscan.AgentClaude},
						},
					},
				},
			}}},
			loaded: true,
		}},
		expanded: map[string]bool{"host:0": true},
		cursor:   0,
	}

	target, ok := m.selectedAgentTarget()
	if !ok {
		t.Fatalf("selectedAgentTarget() ok = false, want true")
	}
	if target.target != "%2" {
		t.Fatalf("target = %q, want %%2", target.target)
	}
	if target.agent != tmuxscan.AgentClaude {
		t.Fatalf("agent = %q, want %q", target.agent, tmuxscan.AgentClaude)
	}
}

func TestAgentTargetDisplayLabelUsesHumanReadablePaneLocation(t *testing.T) {
	m := model{
		hosts: []hostState{{
			host: config.Host{Name: "local", SSH: "local"},
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{{
				ID:   "$1",
				Name: "agents",
				Windows: []tmuxscan.Window{{
					ID:    "@1",
					Index: "2",
					Panes: []tmuxscan.Pane{{
						ID:    "%1",
						Index: "3",
						Agent: tmuxscan.AgentCodex,
					}},
				}},
			}}},
			loaded: true,
		}},
	}
	target := selectedAgentTarget{
		hostIndex: 0,
		target:    "%1",
		agent:     tmuxscan.AgentCodex,
	}

	label := m.agentTargetDisplayLabel(target)
	if label != "local/agents:2.3 (codex)" {
		t.Fatalf("label = %q, want local/agents:2.3 (codex)", label)
	}
}

func TestSelectedPaneAgentTargetCarriesDisplayLabel(t *testing.T) {
	m := model{
		hosts: []hostState{{
			host: config.Host{Name: "local", SSH: "local"},
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{{
				ID:   "$1",
				Name: "agents",
				Windows: []tmuxscan.Window{{
					ID:    "@1",
					Index: "2",
					Panes: []tmuxscan.Pane{{
						ID:    "%1",
						Index: "3",
						Agent: tmuxscan.AgentCodex,
					}},
				}},
			}}},
			loaded: true,
		}},
		expanded: map[string]bool{
			"host:0":                      true,
			"host:0:session:$1":           true,
			"host:0:session:$1:window:@1": true,
		},
		cursor: 2,
	}

	target, ok := m.selectedAgentTarget()
	if !ok {
		t.Fatalf("selectedAgentTarget() ok = false, want true")
	}
	if target.label != "agents:2.3 (codex)" {
		t.Fatalf("target label = %q, want agents:2.3 (codex)", target.label)
	}
	if label := m.agentTargetDisplayLabel(target); label != "local/agents:2.3 (codex)" {
		t.Fatalf("display label = %q, want local/agents:2.3 (codex)", label)
	}
}

func TestComposeInputPrefixDoesNotRepeatTarget(t *testing.T) {
	label := composeInputPrefix(composeState{target: "%1", label: "local/agents:2.3 (codex)"})
	if strings.Contains(label, "%1") {
		t.Fatalf("label = %q, want no tmux pane id", label)
	}
	if label != "" {
		t.Fatalf("label = %q, want empty message input prefix", label)
	}
}

func TestComposeInputPrefixDoesNotExposeRawPaneIDWhenLabelMissing(t *testing.T) {
	label := composeInputPrefix(composeState{target: "%1"})
	if strings.Contains(label, "%1") {
		t.Fatalf("label = %q, want no raw tmux pane id", label)
	}
}

func TestSelectedAgentTargetFallsThroughFromWindow(t *testing.T) {
	m := model{
		hosts: []hostState{{
			host: config.Host{Name: "local", SSH: "local"},
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{{
				ID:   "$1",
				Name: "agents",
				Windows: []tmuxscan.Window{{
					ID:    "@1",
					Index: "0",
					Panes: []tmuxscan.Pane{
						{ID: "%1", Index: "0", Agent: tmuxscan.AgentCodex},
					},
				}},
			}}},
			loaded: true,
		}},
		expanded: map[string]bool{
			"host:0":            true,
			"host:0:session:$1": true,
		},
		cursor: 1,
	}

	target, ok := m.selectedAgentTarget()
	if !ok {
		t.Fatalf("selectedAgentTarget() ok = false, want true")
	}
	if target.target != "%1" {
		t.Fatalf("target = %q, want %%1", target.target)
	}
	if target.agent != tmuxscan.AgentCodex {
		t.Fatalf("agent = %q, want %q", target.agent, tmuxscan.AgentCodex)
	}
}

func TestCycleSelectedSessionStatusFromPane(t *testing.T) {
	m := model{
		hosts: []hostState{{
			host: config.Host{Name: "local", SSH: "local"},
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{{
				ID:   "$1",
				Name: "agents",
				Windows: []tmuxscan.Window{{
					ID:    "@1",
					Index: "0",
					Panes: []tmuxscan.Pane{
						{ID: "%1", Index: "0", Agent: tmuxscan.AgentCodex},
					},
				}},
			}}},
			loaded: true,
		}},
		expanded: map[string]bool{
			"host:0":                      true,
			"host:0:session:$1":           true,
			"host:0:session:$1:window:@1": true,
		},
		statuses: map[string]sessionStatus{},
		cursor:   2,
	}

	m.cycleSelectedSessionStatus()
	ref, ok := m.selectedSessionRef()
	if !ok {
		t.Fatalf("selectedSessionRef() ok = false, want true")
	}
	if got := m.sessionStatusForKey(ref.Key); got != sessionWorking {
		t.Fatalf("status = %q, want %q", got, sessionWorking)
	}

	m.cycleSelectedSessionStatus()
	if got := m.sessionStatusForKey(ref.Key); got != sessionNeedReview {
		t.Fatalf("status = %q, want %q", got, sessionNeedReview)
	}

	m.cycleSelectedSessionStatus()
	if got := m.sessionStatusForKey(ref.Key); got != sessionDone {
		t.Fatalf("status = %q, want %q", got, sessionDone)
	}

	m.cycleSelectedSessionStatus()
	if got := m.sessionStatusForKey(ref.Key); got != sessionIdle {
		t.Fatalf("status = %q, want %q", got, sessionIdle)
	}
}

func TestSessionCardsByStatusGroupsSessions(t *testing.T) {
	session := tmuxscan.Session{ID: "$1", Name: "agents"}
	host := config.Host{Name: "local", SSH: "local"}
	key := sessionStatusKey(host, session)
	m := model{
		hosts: []hostState{{
			host:     host,
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{session}},
			loaded:   true,
		}},
		expanded: map[string]bool{"host:0": true},
		statuses: map[string]sessionStatus{key: sessionNeedReview},
		cursor:   0,
	}

	cards := m.sessionCardsByStatus()
	if len(cards[sessionNeedReview]) != 1 {
		t.Fatalf("need review cards = %d, want 1", len(cards[sessionNeedReview]))
	}
	if cards[sessionNeedReview][0].Name != "agents" {
		t.Fatalf("card name = %q, want agents", cards[sessionNeedReview][0].Name)
	}
	if len(cards[sessionIdle]) != 0 {
		t.Fatalf("idle cards = %d, want 0", len(cards[sessionIdle]))
	}
}

func TestSessionCardsByStatusIncludesDoneColumn(t *testing.T) {
	session := tmuxscan.Session{ID: "$1", Name: "agents"}
	host := config.Host{Name: "local", SSH: "local"}
	key := sessionStatusKey(host, session)
	m := model{
		hosts: []hostState{{
			host:     host,
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{session}},
			loaded:   true,
		}},
		statuses: map[string]sessionStatus{key: sessionDone},
	}

	columns := sessionStatusColumns()
	if columns[len(columns)-1] != sessionDone {
		t.Fatalf("last column = %q, want done", columns[len(columns)-1])
	}
	cards := m.sessionCardsByStatus()
	if len(cards[sessionDone]) != 1 {
		t.Fatalf("done cards = %d, want 1", len(cards[sessionDone]))
	}
}

func TestSessionCardsByStatusHidesMainSession(t *testing.T) {
	cfg := config.Default()
	host := config.Host{Name: "local", Local: true}
	mainSession := tmuxscan.Session{ID: "$main", Name: "tmux-kanban-main"}
	workerSession := tmuxscan.Session{ID: "$worker", Name: "agents"}
	mainKey := sessionStatusKey(host, mainSession)
	workerKey := sessionStatusKey(host, workerSession)
	m := model{
		cfg: cfg,
		hosts: []hostState{{
			host: host,
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{
				mainSession,
				workerSession,
			}},
			loaded: true,
		}},
		statuses: map[string]sessionStatus{
			mainKey:   sessionWorking,
			workerKey: sessionNeedReview,
		},
	}

	cards := m.sessionCardsByStatus()
	if len(cards[sessionWorking]) != 0 {
		t.Fatalf("working cards = %d, want main session excluded", len(cards[sessionWorking]))
	}
	if len(cards[sessionNeedReview]) != 1 || cards[sessionNeedReview][0].Name != "agents" {
		t.Fatalf("need review cards = %#v, want only worker session", cards[sessionNeedReview])
	}
}

func TestRowsHideMainSessionFromExplorer(t *testing.T) {
	cfg := config.Default()
	host := config.Host{Name: "local", Local: true}
	m := model{
		cfg: cfg,
		hosts: []hostState{{
			host: host,
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{
				{ID: "$main", Name: "tmux-kanban-main"},
				{ID: "$worker", Name: "agents"},
			}},
			loaded: true,
		}},
		expanded: map[string]bool{"host:0": true},
	}

	rows := m.rows()
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want worker session only: %#v", len(rows), rows)
	}
	if rows[0].kind != rowSession || rows[0].attachTarget != "agents" {
		t.Fatalf("row[0] = %#v, want worker session row", rows[0])
	}
	label := m.hostLabel(0, m.hosts[0])
	if strings.Contains(label, "2 sessions") {
		t.Fatalf("host label = %q, want main session excluded from local count", label)
	}
}

func TestRowsGroupAgentSessionsBeforeOtherSessions(t *testing.T) {
	host := config.Host{Name: "local", Local: true}
	plainSession := tmuxscan.Session{ID: "$plain", Name: "plain"}
	agentSession := tmuxscan.Session{
		ID:   "$agent",
		Name: "agent",
		Windows: []tmuxscan.Window{{
			ID:    "@1",
			Index: "0",
			Panes: []tmuxscan.Pane{{ID: "%1", Index: "0", Agent: tmuxscan.AgentCodex}},
		}},
	}
	m := model{
		hosts: []hostState{{
			host:     host,
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{plainSession, agentSession}},
			loaded:   true,
		}},
		expanded: map[string]bool{"host:0": true},
	}

	rows := m.rows()
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want two sessions: %#v", len(rows), rows)
	}
	if rows[0].kind != rowSession || rows[0].attachTarget != "agent" {
		t.Fatalf("row[0] = %#v, want agent session first", rows[0])
	}
	if rows[1].kind != rowSession || rows[1].attachTarget != "plain" {
		t.Fatalf("row[1] = %#v, want non-agent session second", rows[1])
	}
}

func TestRenderHostsShowsAgentAndOtherSessionGroups(t *testing.T) {
	host := config.Host{Name: "local", Local: true}
	plainSession := tmuxscan.Session{ID: "$plain", Name: "plain"}
	agentSession := tmuxscan.Session{
		ID:   "$agent",
		Name: "agent",
		Windows: []tmuxscan.Window{{
			ID:    "@1",
			Index: "0",
			Panes: []tmuxscan.Pane{{ID: "%1", Index: "0", Agent: tmuxscan.AgentCodex}},
		}},
	}
	m := model{
		hosts: []hostState{{
			host:     host,
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{plainSession, agentSession}},
			loaded:   true,
		}},
		expanded: map[string]bool{"host:0": true},
	}

	view := ansi.Strip(m.renderHosts(80, 18))
	agentIndex := strings.Index(view, "AGENT SESSIONS")
	otherIndex := strings.Index(view, "OTHER SESSIONS")
	if agentIndex < 0 || otherIndex < 0 {
		t.Fatalf("view missing session group headers:\n%s", view)
	}
	if agentIndex > otherIndex {
		t.Fatalf("group order wrong, want agent column before other column:\n%s", view)
	}
	if !strings.Contains(view, "local/agent") || !strings.Contains(view, "local/plain") {
		t.Fatalf("view missing host-qualified session labels:\n%s", view)
	}
}

func TestRowsGroupSessionsGloballyAcrossHosts(t *testing.T) {
	plainHost := config.Host{Name: "plain-host", SSH: "plain-host"}
	agentHost := config.Host{Name: "agent-host", SSH: "agent-host"}
	plainSession := tmuxscan.Session{ID: "$plain", Name: "plain"}
	agentSession := tmuxscan.Session{
		ID:   "$agent",
		Name: "agent",
		Windows: []tmuxscan.Window{{
			ID:    "@1",
			Index: "0",
			Panes: []tmuxscan.Pane{{ID: "%1", Index: "0", Agent: tmuxscan.AgentClaude}},
		}},
	}
	m := model{
		hosts: []hostState{
			{host: plainHost, snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{plainSession}}, loaded: true},
			{host: agentHost, snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{agentSession}}, loaded: true},
		},
		expanded: map[string]bool{"host:0": true, "host:1": true},
	}

	rows := m.rows()
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want two sessions: %#v", len(rows), rows)
	}
	if rows[0].hostIndex != 1 || rows[0].attachTarget != "agent" {
		t.Fatalf("row[0] = %#v, want agent session from second host first", rows[0])
	}
	if rows[1].hostIndex != 0 || rows[1].attachTarget != "plain" {
		t.Fatalf("row[1] = %#v, want plain session from first host second", rows[1])
	}

	view := ansi.Strip(m.renderHosts(92, 18))
	if strings.Count(view, "AGENT SESSIONS") != 1 || strings.Count(view, "OTHER SESSIONS") != 1 {
		t.Fatalf("view should show one global pair of session columns:\n%s", view)
	}
}

func TestCollectCapabilitiesDescribesMainSessionSkillAndCLI(t *testing.T) {
	cfg := config.Default()
	response := collectCapabilities(cfg)

	if !response.OK {
		t.Fatalf("capabilities OK = false, want true")
	}
	if response.MainAgent.Session != "tmux-kanban-main" {
		t.Fatalf("main agent = %#v, want default main session", response.MainAgent)
	}
	if len(response.Skills) != 1 || response.Skills[0] != mainSessionSkillName {
		t.Fatalf("skills = %#v, want main session skill", response.Skills)
	}
	if !containsString(response.CLICommands, "review-list") || !containsString(response.CLICommands, "choose") {
		t.Fatalf("cli commands = %#v, want agent control commands", response.CLICommands)
	}
	if !strings.Contains(response.Summary, mainSessionSkillName) {
		t.Fatalf("summary = %q, want skill name", response.Summary)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

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

func TestMainSessionKeyStartsMainRoom(t *testing.T) {
	cfg := config.Default()
	host := config.Host{Name: "local", Local: true}
	m := model{
		cfg:   cfg,
		hosts: []hostState{{host: host, loaded: true}},
	}

	nextModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	if cmd != nil {
		t.Fatalf("main room command = %#v, want no main session agent command", cmd)
	}
	next := nextModel.(model)
	if !next.mainActive {
		t.Fatalf("mainActive = false, want true")
	}
	if next.viewMode != viewMain {
		t.Fatalf("view mode = %q, want main", next.viewMode)
	}
	if !next.compose.active || next.compose.label != "Main Room" {
		t.Fatalf("compose = %#v, want active local main room composer", next.compose)
	}
	if row, ok := next.activePreviewRow(); ok {
		t.Fatalf("activePreviewRow() = %#v, true; want no main preview row", row)
	}
}

func TestMainRoomRendersChatAndParticipants(t *testing.T) {
	host := config.Host{Name: "local", Local: true}
	session := tmuxscan.Session{
		ID:   "$1",
		Name: "agents",
		Windows: []tmuxscan.Window{{
			ID:    "@1",
			Index: "0",
			Panes: []tmuxscan.Pane{{ID: "%1", Index: "0", Agent: tmuxscan.AgentCodex}},
		}},
	}
	key := sessionStatusKey(host, session)
	m := model{
		cfg: config.Default(),
		hosts: []hostState{{
			host:     host,
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{session}},
			loaded:   true,
		}},
		statuses:      map[string]sessionStatus{key: sessionNeedReview},
		reviewTargets: map[string]selectedAgentTarget{key: {hostIndex: 0, target: "%1", agent: tmuxscan.AgentCodex}},
		viewMode:      viewMain,
		mainActive:    true,
		mainMessages: []mainMessage{{
			At:     time.Date(2026, 5, 21, 10, 0, 0, 0, time.Local),
			Author: "You",
			Role:   "user",
			Text:   "帮我看一下哪个 review 最重要",
		}},
		compose: composeState{active: true, label: "Main Room"},
	}

	view := m.renderMainRoom(120, 30, 5, 1)
	plain := ansi.Strip(view)
	for _, want := range []string{"Main Room", "Participants", "You", "No agent harness", "agents", "帮我看一下"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("main room missing %q:\n%s", want, plain)
		}
	}
}

func TestMainRoomEnterAppendsUserMessageAndKeepsComposer(t *testing.T) {
	host := config.Host{Name: "local", Local: true}
	m := model{
		cfg:      config.Default(),
		hosts:    []hostState{{host: host, loaded: true}},
		viewMode: viewMain,
		compose: composeState{
			active: true,
			label:  "Main Room",
			text:   "帮我看 review",
			cursor: len([]rune("帮我看 review")),
		},
	}

	nextModel, cmd := m.updateCompose(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("main room command = %#v, want no agent command without Hermes", cmd)
	}
	next := nextModel.(model)
	if !next.compose.active {
		t.Fatalf("compose.active = false, want main room composer to stay focused")
	}
	if next.compose.text != "" || next.compose.cursor != 0 {
		t.Fatalf("compose after send = %#v, want cleared focused composer", next.compose)
	}
	if len(next.mainMessages) != 2 || next.mainMessages[0].Author != "You" {
		t.Fatalf("main messages = %#v, want user message plus empty-harness system message", next.mainMessages)
	}
	if next.mainMessages[0].Text != "帮我看 review" {
		t.Fatalf("main message text = %q, want sent text", next.mainMessages[0].Text)
	}
	if next.mainMessages[1].Author != "system" || !strings.Contains(next.mainMessages[1].Text, "No agent harness") {
		t.Fatalf("system message = %#v, want no harness note", next.mainMessages[1])
	}
}

func TestMainRoomUsesHermesWhenEnabled(t *testing.T) {
	host := config.Host{Name: "local", Local: true}
	m := model{
		cfg: config.Config{
			Hermes: config.HermesConfig{
				Enabled:        true,
				Command:        "hermes",
				Args:           []string{"--oneshot"},
				TimeoutSeconds: 1,
			},
		},
		hosts:    []hostState{{host: host, loaded: true}},
		viewMode: viewMain,
		compose: composeState{
			active: true,
			label:  "Main Room",
			text:   "现在该处理什么",
			cursor: len([]rune("现在该处理什么")),
		},
	}

	nextModel, cmd := m.updateCompose(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("main room Hermes command = nil, want command")
	}
	next := nextModel.(model)
	if len(next.mainMessages) != 2 {
		t.Fatalf("main messages = %#v, want user message and Hermes thinking", next.mainMessages)
	}
	if next.mainMessages[0].Author != "You" || next.mainMessages[1].Author != "Hermes" {
		t.Fatalf("main messages = %#v, want You then Hermes", next.mainMessages)
	}
	if next.mainMessages[1].Text != "thinking..." {
		t.Fatalf("Hermes placeholder = %q, want thinking", next.mainMessages[1].Text)
	}
}

func TestMainHermesResultReplacesThinkingMessage(t *testing.T) {
	m := model{
		mainMessages: []mainMessage{
			{Author: "You", Role: "user", Text: "现在该处理什么"},
			{Author: "Hermes", Role: "conductor", Text: "thinking..."},
		},
	}

	nextModel, _ := m.Update(mainHermesResult{text: "先处理 need review 队列第一项。"})
	next := nextModel.(model)
	if len(next.mainMessages) != 2 {
		t.Fatalf("main messages = %#v, want thinking replaced in place", next.mainMessages)
	}
	if next.mainMessages[1].Author != "Hermes" || next.mainMessages[1].Role != "conductor" {
		t.Fatalf("Hermes message = %#v, want conductor reply", next.mainMessages[1])
	}
	if next.mainMessages[1].Text != "先处理 need review 队列第一项。" {
		t.Fatalf("Hermes text = %q, want reply", next.mainMessages[1].Text)
	}
}

func TestExecuteMainClaudeCommandSwitchesConfiguredAgent(t *testing.T) {
	host := config.Host{Name: "local", Local: true}
	m := model{
		cfg:   config.Default(),
		hosts: []hostState{{host: host, loaded: true}},
	}

	next, cmd := m.executeCommand("main claude")
	if cmd != nil {
		t.Fatalf("main claude command = %#v, want no main session agent command", cmd)
	}
	if next.cfg.MainAgent.Agent != "claude-code" {
		t.Fatalf("main agent = %q, want claude-code", next.cfg.MainAgent.Agent)
	}
	if next.cfg.MainAgent.Command != "claude" {
		t.Fatalf("main command = %q, want claude", next.cfg.MainAgent.Command)
	}
	if !next.mainActive {
		t.Fatalf("mainActive = false, want true")
	}
}

func TestHermesReviewPromptIncludesChoices(t *testing.T) {
	item := reviewItem{
		HostName:    "local",
		SessionName: "agents",
		Agent:       tmuxscan.AgentCodex,
		Row: row{
			attachTarget: "agents:0.0",
		},
	}
	prompt := hermesReviewPrompt(item, []string{
		"Do you want to allow this command?",
		"❯ 1. Allow",
		"  2. Deny",
	})

	for _, want := range []string{
		"Host: local",
		"Session: agents",
		"1: Allow",
		"2: Deny",
		"CHOOSE <number>",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestHermesReviewPromptIncludesSkillAndMemory(t *testing.T) {
	item := reviewItem{
		HostName:    "local",
		SessionName: "agents",
		Agent:       tmuxscan.AgentCodex,
		Row: row{
			attachTarget: "agents:0.0",
		},
	}
	prompt := hermesReviewPromptWithContext(item, []string{"ready"}, hermesReviewContext{
		Skill: "Prefer ASK for destructive commands.",
		Memory: []mesh.MemoryNode{
			{Scope: mesh.Scope{}, Summary: "User prefers conservative review decisions."},
			{Scope: mesh.Scope{Host: "local", Session: "agents"}, Summary: "Run Go tests before approving completion."},
		},
	})

	for _, want := range []string{
		"Review skill:",
		"Prefer ASK for destructive commands.",
		"Scoped memory:",
		"global: User prefers conservative review decisions.",
		"session/local/agents: Run Go tests before approving completion.",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestReviewHermesPromptContextLoadsSkillAndMemory(t *testing.T) {
	root := t.TempDir()
	skillRoot := filepath.Join(root, "skills")
	memoryRoot := filepath.Join(root, "memory")
	skillPath := filepath.Join(skillRoot, "review-advice", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(skillPath, []byte("review skill text"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	memoryScope := mesh.Scope{Host: "local", Session: "agents", Window: "0", Pane: "0"}
	memoryPath := mesh.LocalMemoryPath(memoryRoot, memoryScope)
	if err := os.MkdirAll(filepath.Dir(memoryPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(memoryPath, []byte("pane memory text"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg := config.Default()
	cfg.AgentMesh.SkillRoot = skillRoot
	cfg.AgentMesh.MemoryRoot = memoryRoot
	context := reviewHermesPromptContext(cfg, reviewItem{
		HostName:    "local",
		SessionName: "agents",
		Target:      selectedAgentTarget{key: "host:0:session:$1:window:0:pane:0"},
		Row:         row{key: "host:0:session:$1:window:0:pane:0", attachTarget: "%1"},
	})

	if !strings.Contains(context.Skill, "review skill text") {
		t.Fatalf("skill = %q, want skill text", context.Skill)
	}
	if len(context.Memory) != 1 || context.Memory[0].Summary != "pane memory text" {
		t.Fatalf("memory = %#v, want pane memory", context.Memory)
	}
}

func TestReviewItemScopePrefersTargetKeyWindowAndPane(t *testing.T) {
	scope := reviewItemScope(reviewItem{
		HostName:    "local",
		SessionName: "agents",
		Target:      selectedAgentTarget{key: "host:0:session:$1:window:@1:pane:%2"},
		Row:         row{attachTarget: "%2"},
	})

	if scope.Window != "@1" || scope.Pane != "%2" {
		t.Fatalf("scope = %#v, want window @1 and pane %%2", scope)
	}
}

func TestParseHermesAutoReviewAction(t *testing.T) {
	tests := []struct {
		text       string
		wantOK     bool
		wantKind   string
		wantChoice string
	}{
		{text: "CHOOSE 2: visible approval is safe", wantOK: true, wantKind: "choose", wantChoice: "2"},
		{text: "\nSKIP: ambiguous", wantOK: true, wantKind: "skip"},
		{text: "ASK: need logs", wantOK: false},
		{text: "looks fine", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got, ok := parseHermesAutoReviewAction(tt.text)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v; action = %#v", ok, tt.wantOK, got)
			}
			if !ok {
				return
			}
			if got.kind != tt.wantKind || got.choice != tt.wantChoice {
				t.Fatalf("action = %#v, want kind %q choice %q", got, tt.wantKind, tt.wantChoice)
			}
		})
	}
}

func TestApplyHermesAutoReviewChoosesVisibleChoice(t *testing.T) {
	host := config.Host{Name: "local", Local: true}
	session := tmuxscan.Session{
		ID:   "$1",
		Name: "agents",
		Windows: []tmuxscan.Window{{
			ID:    "@1",
			Index: "0",
			Panes: []tmuxscan.Pane{{ID: "%1", Index: "0", Agent: tmuxscan.AgentCodex}},
		}},
	}
	key := sessionStatusKey(host, session)
	item := reviewItem{
		SessionKey:  key,
		HostName:    "local",
		SessionName: "agents",
		Agent:       tmuxscan.AgentCodex,
		Row:         row{hostIndex: 0, attachTarget: "%1"},
	}
	m := model{
		cfg:           config.Config{Hermes: config.HermesConfig{AutoReview: true}},
		hosts:         []hostState{{host: host, snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{session}}, loaded: true}},
		statuses:      map[string]sessionStatus{key: sessionNeedReview},
		reviewTargets: map[string]selectedAgentTarget{key: {hostIndex: 0, target: "%1", agent: tmuxscan.AgentCodex}},
		reviewSkipped: map[string]bool{},
	}

	cmd := m.applyHermesAutoReview(item, "local", []string{
		"Do you want to allow this command?",
		"  1. Allow",
		"❯ 2. Deny",
	}, "CHOOSE 1: safer option")
	if cmd == nil {
		t.Fatalf("applyHermesAutoReview() cmd = nil, want send choice command")
	}
	if got := m.statuses[key]; got != sessionWorking {
		t.Fatalf("status = %q, want working", got)
	}
	if _, ok := m.reviewTargets[key]; ok {
		t.Fatalf("review target still present after auto choice")
	}
	if len(m.activities) != 1 || m.activities[0].State != "auto chose" {
		t.Fatalf("activities = %#v, want auto chose activity", m.activities)
	}
}

func TestAutoHermesReviewCmdStartsWhenEnteringNeedReview(t *testing.T) {
	host := config.Host{Name: "local", Local: true}
	session := tmuxscan.Session{
		ID:   "$1",
		Name: "agents",
		Windows: []tmuxscan.Window{{
			ID:    "@1",
			Index: "0",
			Panes: []tmuxscan.Pane{{ID: "%1", Index: "0", Agent: tmuxscan.AgentCodex}},
		}},
	}
	key := sessionStatusKey(host, session)
	m := model{
		cfg: config.Config{Hermes: config.HermesConfig{
			Enabled:        true,
			AutoReview:     true,
			Command:        "hermes",
			Args:           []string{"--oneshot"},
			TimeoutSeconds: 120,
		}},
		hosts: []hostState{{host: host, snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{session}}, loaded: true}},
		statuses: map[string]sessionStatus{
			key: sessionNeedReview,
		},
		reviewTargets: map[string]selectedAgentTarget{key: {hostIndex: 0, target: "%1", agent: tmuxscan.AgentCodex}},
		hermes:        map[string]hermesAdvice{},
	}

	cmd := m.autoHermesReviewCmd(true, sessionWorking, sessionNeedReview, key)
	if cmd == nil {
		t.Fatalf("autoHermesReviewCmd() = nil, want Hermes command")
	}
	if !m.hermes[key].loading {
		t.Fatalf("hermes[%q].loading = false, want true", key)
	}

	again := m.autoHermesReviewCmd(true, sessionNeedReview, sessionNeedReview, key)
	if again != nil {
		t.Fatalf("autoHermesReviewCmd() while already need-review = %#v, want nil", again)
	}
}

func TestHermesAdviceClearedWhenSessionReentersNeedReview(t *testing.T) {
	key := "local:$1"
	target := selectedAgentTarget{hostIndex: 0, target: "%1", agent: tmuxscan.AgentCodex}
	m := model{
		statuses:      map[string]sessionStatus{key: sessionDone},
		statusStreaks: map[string]statusStreak{},
		reviewTargets: map[string]selectedAgentTarget{},
		hermes:        map[string]hermesAdvice{key: {text: "old advice"}},
	}

	m.applyAgentStatusResult(agentStatusResult{key: key, status: sessionNeedReview, target: target, ok: true})
	if _, ok := m.hermes[key]; ok {
		t.Fatalf("hermes advice for %q was kept after re-entering need review", key)
	}
}

func TestHermesAdviceKeptForSameNeedReviewPoll(t *testing.T) {
	key := "local:$1"
	target := selectedAgentTarget{hostIndex: 0, target: "%1", agent: tmuxscan.AgentCodex}
	m := model{
		statuses:      map[string]sessionStatus{key: sessionNeedReview},
		statusStreaks: map[string]statusStreak{},
		reviewTargets: map[string]selectedAgentTarget{key: target},
		hermes:        map[string]hermesAdvice{key: {text: "current advice"}},
	}

	m.applyAgentStatusResult(agentStatusResult{key: key, status: sessionNeedReview, target: target, ok: true})
	if got := m.hermes[key].text; got != "current advice" {
		t.Fatalf("hermes advice = %q, want current advice", got)
	}
}

func TestStaleHermesResultDoesNotReplaceCurrentReviewAdvice(t *testing.T) {
	key := "local:$1"
	m := model{
		statuses:      map[string]sessionStatus{key: sessionNeedReview},
		reviewTargets: map[string]selectedAgentTarget{key: {hostIndex: 0, target: "%2", agent: tmuxscan.AgentCodex}},
		hermes:        map[string]hermesAdvice{key: {text: "current advice"}},
	}

	nextModel, _ := m.Update(hermesQueryResult{
		key:  key,
		text: "stale advice",
		item: reviewItem{
			SessionKey: key,
			Row:        row{attachTarget: "%1"},
		},
	})
	next := nextModel.(model)
	if got := next.hermes[key].text; got != "current advice" {
		t.Fatalf("hermes advice = %q, want current advice", got)
	}
}

func TestHermesQueryResultRecordsAnswerInActivity(t *testing.T) {
	host := config.Host{Name: "local", Local: true}
	session := tmuxscan.Session{
		ID:   "$1",
		Name: "agents",
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

	nextModel, _ := m.Update(hermesQueryResult{
		key:  key,
		text: "CHOOSE 1\nThe command matches the visible approval prompt.",
		item: reviewItem{
			SessionKey: key,
			Row:        row{attachTarget: "%1"},
		},
	})
	next := nextModel.(model)
	if len(next.activities) != 1 {
		t.Fatalf("activities = %#v, want Hermes reply activity", next.activities)
	}
	activity := next.activities[0]
	if activity.Agent != "Hermes" || activity.Target != "local/agents (codex)" {
		t.Fatalf("activity = %#v, want Hermes answer for local/agents", activity)
	}
	if !strings.Contains(activity.Message, "CHOOSE 1") || !strings.Contains(activity.Message, "approval prompt") {
		t.Fatalf("activity message = %q, want Hermes answer text", activity.Message)
	}
}

func TestApplyHermesAutoReviewSkipsWhenSuggested(t *testing.T) {
	host := config.Host{Name: "local", Local: true}
	session := tmuxscan.Session{
		ID:   "$1",
		Name: "agents",
		Windows: []tmuxscan.Window{{
			ID:    "@1",
			Index: "0",
			Panes: []tmuxscan.Pane{{ID: "%1", Index: "0", Agent: tmuxscan.AgentCodex}},
		}},
	}
	key := sessionStatusKey(host, session)
	item := reviewItem{
		SessionKey:  key,
		HostName:    "local",
		SessionName: "agents",
		Agent:       tmuxscan.AgentCodex,
		Row:         row{hostIndex: 0, attachTarget: "%1"},
	}
	m := model{
		cfg:           config.Config{Hermes: config.HermesConfig{AutoReview: true}},
		hosts:         []hostState{{host: host, snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{session}}, loaded: true}},
		statuses:      map[string]sessionStatus{key: sessionNeedReview},
		reviewTargets: map[string]selectedAgentTarget{key: {hostIndex: 0, target: "%1", agent: tmuxscan.AgentCodex}},
		reviewSkipped: map[string]bool{},
	}

	cmd := m.applyHermesAutoReview(item, "local", nil, "SKIP: not enough visible context")
	if cmd != nil {
		t.Fatalf("applyHermesAutoReview() cmd = %#v, want nil for skip", cmd)
	}
	if !m.reviewSkipped[key] {
		t.Fatalf("reviewSkipped[%q] = false, want true", key)
	}
}

func TestHermesQQNotificationPromptIncludesContext(t *testing.T) {
	item := cliReviewItem{
		Host:        "nebula",
		Target:      "agents:0.1",
		Agent:       string(tmuxscan.AgentClaude),
		SessionName: "agents",
		WindowIndex: "0",
		WindowName:  "work",
		PaneIndex:   "1",
		Screen: cliScreen{
			NeedsReview: true,
			Choices: []cliChoice{
				{Number: "1", Label: "Allow command", Selected: true},
				{Number: "2", Label: "Deny"},
			},
		},
		Capture: []string{
			"Run shell command?",
			"> 1. Allow command",
			"  2. Deny",
		},
	}

	prompt := hermesQQNotificationPrompt([]cliReviewItem{item}, "wake me when review is needed")
	for _, want := range []string{
		`send_message(target="qqbot", message=...)`,
		"User intent:",
		"wake me when review is needed",
		"Host: nebula",
		"Target: agents:0.1",
		"Agent: claude-code",
		"1: Allow command",
		"Run shell command?",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestNotifyQQForReviewItemsSkipsWhenDisabled(t *testing.T) {
	result := notifyQQForReviewItems(config.Config{}, []cliReviewItem{{
		Screen: cliScreen{NeedsReview: true},
	}}, "notify me")

	if result.Attempted {
		t.Fatalf("attempted = true, want false")
	}
	if result.Reason != "notification.qq_enabled is false" {
		t.Fatalf("reason = %q, want disabled reason", result.Reason)
	}
	if result.NeedsReviewCount != 1 {
		t.Fatalf("needs review count = %d, want 1", result.NeedsReviewCount)
	}
}

func TestNotifyQQForReviewItemsSkipsWhenNoNeedsReview(t *testing.T) {
	result := notifyQQForReviewItems(config.Config{
		Notification: config.NotificationConfig{QQEnabled: true},
	}, []cliReviewItem{{
		Screen: cliScreen{NeedsReview: false},
	}}, "notify me")

	if result.Attempted {
		t.Fatalf("attempted = true, want false")
	}
	if result.Reason != "no needs_review items" {
		t.Fatalf("reason = %q, want no items reason", result.Reason)
	}
}

func TestExecuteCommandTogglesRuntimeSettings(t *testing.T) {
	m := initialModel(config.Config{})

	next, cmd := m.executeCommand("set qq on")
	if cmd != nil {
		t.Fatalf("set qq command returned cmd, want nil")
	}
	if !next.cfg.Notification.QQEnabled {
		t.Fatalf("qq_enabled = false, want true")
	}
	if next.status != "QQ notification on" {
		t.Fatalf("status = %q, want QQ notification on", next.status)
	}

	next, _ = next.executeCommand("hermes off")
	if next.cfg.Hermes.Enabled {
		t.Fatalf("hermes enabled = true, want false")
	}
	next, _ = next.executeCommand("set hermes.auto_review on")
	if !next.cfg.Hermes.AutoReview {
		t.Fatalf("hermes auto_review = false, want true")
	}
}

func TestExecuteCommandConfiguresMainAgent(t *testing.T) {
	m := initialModel(config.Default())

	next, cmd := m.executeCommand("main host nebula")
	if cmd != nil {
		t.Fatalf("main host command returned cmd, want nil")
	}
	if next.cfg.MainAgent.Host != "nebula" {
		t.Fatalf("main host = %q, want nebula", next.cfg.MainAgent.Host)
	}

	next, cmd = next.executeCommand("main session conductor")
	if cmd != nil {
		t.Fatalf("main session command returned cmd, want nil")
	}
	if next.cfg.MainAgent.Session != "conductor" {
		t.Fatalf("main session = %q, want conductor", next.cfg.MainAgent.Session)
	}

	next, cmd = next.executeCommand("main command codex --profile kanban")
	if cmd != nil {
		t.Fatalf("main command returned cmd, want nil")
	}
	if next.cfg.MainAgent.Command != "codex" || len(next.cfg.MainAgent.Args) != 2 {
		t.Fatalf("main command config = %#v, want codex with two args", next.cfg.MainAgent)
	}

	next, cmd = next.executeCommand("set main.agent claude")
	if cmd != nil {
		t.Fatalf("set main.agent returned cmd, want nil")
	}
	if next.cfg.MainAgent.Agent != "claude-code" || next.cfg.MainAgent.Command != "claude" {
		t.Fatalf("main agent config = %#v, want claude-code/claude", next.cfg.MainAgent)
	}
}

func TestExecuteCommandConfiguresAgentMesh(t *testing.T) {
	m := initialModel(config.Default())

	next, cmd := m.executeCommand("mesh on")
	if cmd != nil {
		t.Fatalf("mesh on returned cmd, want nil")
	}
	if !next.cfg.AgentMesh.Enabled {
		t.Fatalf("agent mesh enabled = false, want true")
	}

	next, _ = next.executeCommand("mesh shared off")
	if next.cfg.AgentMesh.SharedShortAgent {
		t.Fatalf("shared_short_agent = true, want false")
	}

	next, _ = next.executeCommand("mesh default claude")
	if next.cfg.AgentMesh.DefaultAgent != "claude-code" {
		t.Fatalf("default agent = %q, want claude-code", next.cfg.AgentMesh.DefaultAgent)
	}

	next, _ = next.executeCommand("mesh skill-root ./mesh-skills")
	if next.cfg.AgentMesh.SkillRoot != "./mesh-skills" {
		t.Fatalf("skill root = %q, want ./mesh-skills", next.cfg.AgentMesh.SkillRoot)
	}

	next, _ = next.executeCommand("mesh policy review-advice backend claude")
	index := next.meshPolicyIndex("review-advice")
	if index < 0 {
		t.Fatalf("review-advice policy not found")
	}
	policy := next.cfg.AgentMesh.Policies[index]
	if policy.Backend != "claude-code" || policy.Agent != "claude-code" || policy.Command != "claude" {
		t.Fatalf("policy = %#v, want claude-code backend/agent/command", policy)
	}

	next, _ = next.executeCommand("mesh policy review-advice skill review-advice")
	if next.cfg.AgentMesh.Policies[index].Skill != "review-advice" {
		t.Fatalf("policy skill = %q, want review-advice", next.cfg.AgentMesh.Policies[index].Skill)
	}

	next, _ = next.executeCommand("set mesh.mail_dir /tmp/tmux-kanban-mail")
	if next.cfg.AgentMesh.Mail.Dir != "/tmp/tmux-kanban-mail" {
		t.Fatalf("mail dir = %q, want configured dir", next.cfg.AgentMesh.Mail.Dir)
	}

	next, _ = next.executeCommand("mesh policy review-advice off")
	if next.cfg.AgentMesh.Policies[index].Enabled {
		t.Fatalf("review-advice enabled = true, want false")
	}
}

func TestExecuteCommandSetsViewMode(t *testing.T) {
	m := initialModel(config.Config{})

	next, cmd := m.executeCommand("view review")
	if cmd != nil {
		t.Fatalf("view command returned cmd, want nil")
	}
	if next.viewMode != viewReview {
		t.Fatalf("view mode = %q, want review", next.viewMode)
	}

	next, _ = next.executeCommand("tree")
	if next.viewMode != viewTree {
		t.Fatalf("view mode = %q, want tree", next.viewMode)
	}
}

func TestExecuteCommandSetsSelectedSessionStatus(t *testing.T) {
	host := config.Host{Name: "local", Local: true}
	session := tmuxscan.Session{ID: "$1", Name: "agents"}
	m := model{
		hosts: []hostState{{
			host:     host,
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{session}},
			loaded:   true,
		}},
		expanded: map[string]bool{"host:0": true},
		statuses: map[string]sessionStatus{},
		cursor:   0,
	}

	next, cmd := m.executeCommand("status need-review")
	if cmd != nil {
		t.Fatalf("status command returned cmd, want nil")
	}
	key := sessionStatusKey(host, session)
	if got := next.statuses[key]; got != sessionNeedReview {
		t.Fatalf("status = %q, want need review", got)
	}

	next, cmd = next.executeCommand("status done")
	if cmd != nil {
		t.Fatalf("done status command returned cmd, want nil")
	}
	if got := next.statuses[key]; got != sessionDone {
		t.Fatalf("status = %q, want done", got)
	}
}

func TestUpdateCommandEditsAndExecutes(t *testing.T) {
	m := initialModel(config.Config{})
	m.beginCommand()

	nextModel, cmd := m.updateCommand(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("settings")})
	if cmd != nil {
		t.Fatalf("typing command returned cmd, want nil")
	}
	next := nextModel.(model)
	if next.command.text != "settings" {
		t.Fatalf("command text = %q, want settings", next.command.text)
	}

	nextModel, cmd = next.updateCommand(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("enter command returned nil cmd, want cursor hide command")
	}
	next = nextModel.(model)
	if next.command.active {
		t.Fatalf("command active = true, want false")
	}
	if !strings.Contains(next.status, "settings:") {
		t.Fatalf("status = %q, want settings output", next.status)
	}
}

func TestUpdateCommandCanSelectCandidate(t *testing.T) {
	m := initialModel(config.Config{})
	m.beginCommand()

	nextModel, cmd := m.updateCommand(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("status n")})
	if cmd != nil {
		t.Fatalf("typing command returned cmd, want nil")
	}
	next := nextModel.(model)
	nextModel, cmd = next.updateCommand(tea.KeyMsg{Type: tea.KeyTab})
	if cmd != nil {
		t.Fatalf("tab command returned cmd, want nil")
	}
	next = nextModel.(model)
	if next.command.text != "status need-review" {
		t.Fatalf("command text = %q, want status need-review", next.command.text)
	}
}

func TestUpdateCommandEnterRunsSelectedCandidatePrefix(t *testing.T) {
	m := initialModel(config.Config{})
	m.beginCommand()

	nextModel, _ := m.updateCommand(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("view r")})
	next := nextModel.(model)
	nextModel, cmd := next.updateCommand(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("enter command returned nil cmd, want cursor hide command")
	}
	next = nextModel.(model)
	if next.viewMode != viewReview {
		t.Fatalf("view mode = %q, want review", next.viewMode)
	}
}

func TestUpdateCommandEnterCompletesMergedToggleCandidate(t *testing.T) {
	m := initialModel(config.Config{})
	m.beginCommand()

	nextModel, _ := m.updateCommand(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("set q")})
	next := nextModel.(model)
	nextModel, cmd := next.updateCommand(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("enter command returned cmd, want nil while completing candidate")
	}
	next = nextModel.(model)
	if !next.command.active {
		t.Fatalf("command active = false, want command mode to stay active")
	}
	if next.command.text != "set qq " {
		t.Fatalf("command text = %q, want set qq with trailing space", next.command.text)
	}
}

func TestUpdateCommandEnterExecutesMergedToggleValue(t *testing.T) {
	m := initialModel(config.Config{})
	m.beginCommand()

	nextModel, _ := m.updateCommand(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("set qq on")})
	next := nextModel.(model)
	nextModel, cmd := next.updateCommand(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("enter command returned nil cmd, want cursor hide command")
	}
	next = nextModel.(model)
	if next.command.active {
		t.Fatalf("command active = true, want command mode closed")
	}
	if !next.cfg.Notification.QQEnabled {
		t.Fatalf("QQEnabled = false, want true")
	}
	if next.status != "QQ notification on" {
		t.Fatalf("status = %q, want QQ notification on", next.status)
	}
}

func TestRenderCommandSuggestionLinesShowsSelectableOptions(t *testing.T) {
	m := model{command: commandState{active: true, text: "view", selected: 1}}
	lines := m.renderCommandSuggestionLines(36)
	if len(lines) == 0 {
		t.Fatalf("suggestion lines empty, want candidates")
	}
	if !strings.Contains(lines[0], ":view tree") {
		t.Fatalf("first suggestion = %q, want view tree", lines[0])
	}
	if !strings.Contains(lines[1], "> :view review") {
		t.Fatalf("selected suggestion = %q, want view review marker", lines[1])
	}
	for _, line := range lines {
		if width := lipgloss.Width(line); width > 36 {
			t.Fatalf("suggestion width = %d, want <= 36: %q", width, line)
		}
	}
}

func TestRenderCommandSuggestionLinesMergesToggleOptions(t *testing.T) {
	m := model{command: commandState{active: true, text: "set q"}}
	lines := m.renderCommandSuggestionLines(40)
	if len(lines) != 1 {
		t.Fatalf("suggestion lines = %d, want one merged toggle candidate: %#v", len(lines), lines)
	}
	if !strings.Contains(lines[0], ":set qq on/off") {
		t.Fatalf("suggestion = %q, want merged on/off option", lines[0])
	}
}

func TestUpdateCommandCtrlCCancelsCommandMode(t *testing.T) {
	m := model{command: commandState{active: true, text: "status done"}}

	nextModel, cmd := m.updateCommand(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatalf("ctrl-c returned nil cmd, want hide cursor command")
	}
	next := nextModel.(model)
	if next.command.active {
		t.Fatalf("command active = true, want false")
	}
	if next.status != "command canceled" {
		t.Fatalf("status = %q, want command canceled", next.status)
	}
}

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

			cmd := needReviewBellCmd(tt.hadOld, tt.oldStatus, tt.nextStatus, false)
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
	cmd := needReviewBellCmd(true, sessionWorking, sessionNeedReview, true)
	if cmd != nil {
		t.Fatalf("needReviewBellCmd() with Hermes handling = %#v, want nil", cmd)
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
		statusStreaks: map[string]statusStreak{},
		reviewTargets: map[string]selectedAgentTarget{},
		cache:         map[string]previewCacheEntry{},
		cursor:        0,
		viewMode:      viewTree,
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
		viewMode: viewReview,
		preview:  previewState{target: "%1", lines: []string{"Do you want to allow this command?"}},
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
