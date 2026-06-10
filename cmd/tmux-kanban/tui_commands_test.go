package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"tmux-kanban/internal/config"
	"tmux-kanban/internal/tmuxscan"
)

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
	next, cmd = next.executeCommand("set terminal_review on")
	if cmd != nil {
		t.Fatalf("set terminal_review command returned cmd with empty review queue, want nil")
	}
	if !next.cfg.Notification.TerminalReview {
		t.Fatalf("terminal_review = false, want true")
	}
	if next.status != "terminal review notification on" {
		t.Fatalf("status = %q, want terminal review notification on", next.status)
	}
	next, cmd = next.executeCommand("set auto_review_audit_qq uncertain")
	if cmd != nil {
		t.Fatalf("set auto_review_audit_qq command returned cmd, want nil")
	}
	if next.cfg.Notification.AutoReviewAuditQQ != config.AutoReviewAuditQQUncertain {
		t.Fatalf("auto_review_audit_qq = %q, want uncertain", next.cfg.Notification.AutoReviewAuditQQ)
	}
	if next.status != "auto review audit QQ uncertain" {
		t.Fatalf("status = %q, want auto review audit QQ uncertain", next.status)
	}

	next, _ = next.executeCommand("hermes off")
	if next.cfg.Hermes.Enabled {
		t.Fatalf("hermes enabled = true, want false")
	}
	next, _ = next.executeCommand("set hermes.auto_review on")
	if !next.cfg.Hermes.AutoReview {
		t.Fatalf("hermes auto_review = false, want true")
	}
	next, _ = next.executeCommand("set hermes.auto_review all off")
	if next.cfg.Hermes.AutoReview {
		t.Fatalf("hermes auto_review = true, want false after all off")
	}
	next, _ = next.executeCommand("set hermes.auto_done session local/agents off")
	if len(next.cfg.Hermes.Scopes) != 1 {
		t.Fatalf("hermes scopes = %d, want 1", len(next.cfg.Hermes.Scopes))
	}
	scope := next.cfg.Hermes.Scopes[0]
	if scope.Host != "local" || scope.Session != "agents" || scope.AutoDone == nil || *scope.AutoDone {
		t.Fatalf("hermes scope = %#v, want local/agents auto_done=false", scope)
	}
	next, _ = next.executeCommand("set hermes.auto_idle host all off")
	if len(next.cfg.Hermes.Scopes) != 2 {
		t.Fatalf("hermes scopes = %d, want 2", len(next.cfg.Hermes.Scopes))
	}
	scope = next.cfg.Hermes.Scopes[1]
	if scope.Host != "all" || scope.Session != "" || scope.AutoIdle == nil || *scope.AutoIdle {
		t.Fatalf("hermes all-host scope = %#v, want host all auto_idle=false", scope)
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

func TestExecuteCommandPreparesSelectedSessionClose(t *testing.T) {
	host := config.Host{Name: "local", Local: true}
	session := tmuxscan.Session{ID: "$1", Name: "agents"}
	m := model{
		cfg: config.Config{Hosts: []config.Host{host}},
		hosts: []hostState{{
			host:     host,
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{session}},
			loaded:   true,
		}},
		expanded: map[string]bool{"host:0": true},
		cursor:   0,
	}

	next, cmd := m.executeCommand("session close here")
	if cmd != nil {
		t.Fatalf("session close preparation returned cmd, want nil")
	}
	if next.sessionClose.token != "local/agents" {
		t.Fatalf("pending close = %#v, want local/agents", next.sessionClose)
	}
	if !strings.Contains(next.status, "session close confirm local/agents") {
		t.Fatalf("status = %q, want confirmation command", next.status)
	}
}

func TestExecuteCommandRequiresMatchingSessionCloseConfirmation(t *testing.T) {
	host := config.Host{Name: "local", Local: true}
	m := model{
		cfg:          config.Config{Hosts: []config.Host{host}},
		hosts:        []hostState{{host: host}},
		sessionClose: pendingSessionClose{host: host, session: "agents", token: "local/agents"},
	}

	next, cmd := m.executeCommand("session close confirm local/other")
	if cmd != nil {
		t.Fatalf("mismatched confirmation returned cmd, want nil")
	}
	if next.sessionClose.token != "local/agents" {
		t.Fatalf("pending close changed to %#v, want original", next.sessionClose)
	}
	if next.status != "close confirmation mismatch" {
		t.Fatalf("status = %q, want mismatch", next.status)
	}

	next, cmd = next.executeCommand("session close confirm local/agents")
	if cmd == nil {
		t.Fatalf("matching confirmation returned nil cmd, want close command")
	}
	if next.sessionClose.token != "" {
		t.Fatalf("pending close = %#v, want cleared", next.sessionClose)
	}
}

func TestResolveSessionCommandTargetUsesSelectedHostByDefault(t *testing.T) {
	local := config.Host{Name: "local", Local: true}
	remote := config.Host{Name: "nebula", SSH: "nebula"}
	m := model{
		cfg: config.Config{Hosts: []config.Host{local, remote}},
		hosts: []hostState{
			{host: local, loaded: true},
			{host: remote, snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{{ID: "$1", Name: "remote-agent"}}}, loaded: true},
		},
		expanded: map[string]bool{},
		cursor:   0,
	}

	host, session, ok := m.resolveSessionCommandTarget("work")
	if !ok {
		t.Fatalf("resolve target ok = false, status=%q", m.status)
	}
	if host.Name != "nebula" || session != "work" {
		t.Fatalf("target = %s/%s, want nebula/work", host.Name, session)
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

	nextModel, _ := m.updateCommand(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("refresh")})
	next := nextModel.(model)
	nextModel, cmd := next.updateCommand(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("enter command returned nil cmd, want scan command")
	}
	next = nextModel.(model)
	if next.command.active {
		t.Fatalf("command active = true, want closed after execute")
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
	m := model{command: commandState{active: true, text: "status ", selected: 1}}
	lines := m.renderCommandSuggestionLines(40)
	if len(lines) == 0 {
		t.Fatalf("suggestion lines empty, want candidates")
	}
	if !strings.Contains(lines[0], ":status idle") {
		t.Fatalf("first suggestion = %q, want status idle", lines[0])
	}
	if len(lines) < 2 || !strings.Contains(lines[1], "> :status working") {
		t.Fatalf("selected suggestion = %q, want status working selected", lines)
	}
	for _, line := range lines {
		if width := lipgloss.Width(line); width > 40 {
			t.Fatalf("suggestion width = %d, want <= 40: %q", width, line)
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
