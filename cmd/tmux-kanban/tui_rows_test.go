package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"tmux-kanban/internal/config"
	"tmux-kanban/internal/tmuxscan"
)

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

func TestRenderHostsScrollsColumnsToSelectedSession(t *testing.T) {
	host := config.Host{Name: "local", Local: true}
	sessions := make([]tmuxscan.Session, 0, 8)
	for i := 0; i < 8; i++ {
		sessions = append(sessions, tmuxscan.Session{ID: fmt.Sprintf("$plain%d", i), Name: fmt.Sprintf("plain-%d", i)})
	}
	m := model{
		hosts: []hostState{{
			host:     host,
			snapshot: tmuxscan.Snapshot{Sessions: sessions},
			loaded:   true,
		}},
		expanded: map[string]bool{"host:0": true},
		cursor:   7,
	}

	view := ansi.Strip(m.renderHosts(80, 10))
	if !strings.Contains(view, "local/plain-7") {
		t.Fatalf("view missing selected row after scroll:\n%s", view)
	}
	if strings.Contains(view, "local/plain-0") {
		t.Fatalf("view still shows top row instead of scrolling:\n%s", view)
	}
}

func TestRenderHostsScrollsStackToSelectedOtherSession(t *testing.T) {
	host := config.Host{Name: "local", Local: true}
	sessions := make([]tmuxscan.Session, 0, 10)
	for i := 0; i < 4; i++ {
		sessions = append(sessions, tmuxscan.Session{
			ID:   fmt.Sprintf("$agent%d", i),
			Name: fmt.Sprintf("agent-%d", i),
			Windows: []tmuxscan.Window{{
				ID:    fmt.Sprintf("@agent%d", i),
				Index: "0",
				Panes: []tmuxscan.Pane{{ID: fmt.Sprintf("%%agent%d", i), Index: "0", Agent: tmuxscan.AgentCodex}},
			}},
		})
	}
	for i := 0; i < 6; i++ {
		sessions = append(sessions, tmuxscan.Session{ID: fmt.Sprintf("$plain%d", i), Name: fmt.Sprintf("plain-%d", i)})
	}
	m := model{
		hosts: []hostState{{
			host:     host,
			snapshot: tmuxscan.Snapshot{Sessions: sessions},
			loaded:   true,
		}},
		expanded: map[string]bool{"host:0": true},
		cursor:   9,
	}

	view := ansi.Strip(m.renderHosts(60, 12))
	if !strings.Contains(view, "local/plain-5") {
		t.Fatalf("view missing selected other row after stack scroll:\n%s", view)
	}
	if strings.Contains(view, "local/agent-0") {
		t.Fatalf("stack view still shows top agent row instead of scrolling:\n%s", view)
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
	for _, command := range []string{"review-list", "choose", "session-open", "session-close"} {
		if !containsString(response.CLICommands, command) {
			t.Fatalf("cli commands = %#v, want %s", response.CLICommands, command)
		}
	}
	if !strings.Contains(response.Summary, mainSessionSkillName) {
		t.Fatalf("summary = %q, want skill name", response.Summary)
	}
}

func TestSessionCloseConfirmationTokenIncludesHostAndSession(t *testing.T) {
	host := config.Host{Name: "nebula", SSH: "nebula.example"}
	if got := sessionCloseConfirmationToken(host, "work"); got != "nebula/work" {
		t.Fatalf("confirmation token = %q, want nebula/work", got)
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
