package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tmux-kanban/internal/config"
	"tmux-kanban/internal/hermeslog"
	"tmux-kanban/internal/mesh"
	"tmux-kanban/internal/tmuxscan"
)

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

func TestSelectedMemoryUpdateTargetBuildsRequestedScope(t *testing.T) {
	host := config.Host{Name: "local", Local: true}
	window := tmuxscan.Window{
		ID:     "@1",
		Index:  "0",
		Active: true,
		Panes:  []tmuxscan.Pane{{ID: "%1", Index: "0", Agent: tmuxscan.AgentCodex, Active: true}},
	}
	session := tmuxscan.Session{ID: "$1", Name: "agents", Windows: []tmuxscan.Window{window}}
	m := model{
		hosts: []hostState{{
			host:     host,
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{session}},
			loaded:   true,
		}},
		expanded: map[string]bool{"host:0": true, "host:0:session:$1": true, "host:0:session:$1:window:@1": true},
		cursor:   1,
	}

	_, _, scope, ok := m.selectedMemoryUpdateTarget("pane")
	if !ok {
		t.Fatalf("selectedMemoryUpdateTarget() ok = false, want true")
	}
	if scope.Host != "local" || scope.Session != "agents" || scope.Window != "@1" || scope.Pane != "%1" {
		t.Fatalf("pane scope = %#v", scope)
	}
	_, _, scope, ok = m.selectedMemoryUpdateTarget("session")
	if !ok {
		t.Fatalf("selectedMemoryUpdateTarget(session) ok = false, want true")
	}
	if scope.Host != "local" || scope.Session != "agents" || scope.Window != "" || scope.Pane != "" {
		t.Fatalf("session scope = %#v", scope)
	}
}

func TestHermesMemoryUpdatePromptIncludesSkillAndScope(t *testing.T) {
	ref := selectedSessionRef{
		Host:    config.Host{Name: "local", Local: true},
		Session: tmuxscan.Session{Name: "agents"},
	}
	target := selectedAgentTarget{target: "%1", agent: tmuxscan.AgentCodex}
	scope := mesh.Scope{Host: "local", Session: "agents"}
	prompt := hermesMemoryUpdatePrompt(ref, target, scope, []string{"task completed"}, "memory skill text", []mesh.MemoryNode{{
		Scope:   mesh.Scope{Host: "local"},
		Summary: "host memory",
	}}, "")
	for _, want := range []string{
		"Memory skill:",
		"memory skill text",
		"Target memory scope:",
		"session/local/agents",
		"host/local: host memory",
		"task completed",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
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
		cfg:           config.Config{Hermes: config.HermesConfig{Enabled: true, AutoReview: true}},
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

func TestApplyHermesAutoReviewReturnsAuditCommandForUnactionableAdvice(t *testing.T) {
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
		cfg: config.Config{
			Hermes:       config.HermesConfig{Enabled: true, AutoReview: true},
			Notification: config.NotificationConfig{AutoReviewAuditQQ: config.AutoReviewAuditQQUncertain},
		},
		hosts:         []hostState{{host: host, snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{session}}, loaded: true}},
		statuses:      map[string]sessionStatus{key: sessionNeedReview},
		reviewTargets: map[string]selectedAgentTarget{key: {hostIndex: 0, target: "%1", agent: tmuxscan.AgentCodex}},
	}

	cmd := m.applyHermesAutoReview(item, "local", []string{"Do you want to continue?"}, "I cannot decide safely.")
	if cmd == nil {
		t.Fatalf("applyHermesAutoReview() cmd = nil, want QQ audit command")
	}
	if got := m.statuses[key]; got != sessionNeedReview {
		t.Fatalf("status = %q, want need review left for human", got)
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

func TestAutoHermesReviewCmdUsesSessionScope(t *testing.T) {
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
			Enabled:        false,
			AutoReview:     false,
			Command:        "hermes",
			Args:           []string{"--oneshot"},
			TimeoutSeconds: 120,
			Scopes: []config.HermesScopeConfig{{
				Host:       "local",
				Session:    "agents",
				Enabled:    boolSettingPtr(true),
				AutoReview: boolSettingPtr(true),
			}},
		}},
		hosts:         []hostState{{host: host, snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{session}}, loaded: true}},
		statuses:      map[string]sessionStatus{key: sessionNeedReview},
		reviewTargets: map[string]selectedAgentTarget{key: {hostIndex: 0, target: "%1", agent: tmuxscan.AgentCodex}},
		hermes:        map[string]hermesAdvice{},
	}

	cmd := m.autoHermesReviewCmd(true, sessionWorking, sessionNeedReview, key)
	if cmd == nil {
		t.Fatalf("autoHermesReviewCmd() = nil, want scoped Hermes command")
	}
}

func TestParseHermesAutoNextStepAction(t *testing.T) {
	tests := []struct {
		text        string
		wantOK      bool
		wantKind    string
		wantMessage string
	}{
		{text: "SEND: 继续执行下一项任务", wantOK: true, wantKind: "send", wantMessage: "继续执行下一项任务"},
		{text: "WAIT: 没有明确下一步", wantOK: false},
		{text: "ASK: 需要产品确认", wantOK: false},
		{text: "继续下一步", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got, ok := parseHermesAutoNextStepAction(tt.text)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v; action = %#v", ok, tt.wantOK, got)
			}
			if !ok {
				return
			}
			if got.kind != tt.wantKind || got.message != tt.wantMessage {
				t.Fatalf("action = %#v, want kind %q message %q", got, tt.wantKind, tt.wantMessage)
			}
		})
	}
}

func TestAutoHermesNextStepCmdStartsWhenEnteringDone(t *testing.T) {
	host := config.Host{Name: "local", Local: true}
	session := tmuxscan.Session{
		ID:   "$1",
		Name: "agents",
		Windows: []tmuxscan.Window{{
			ID:    "@1",
			Index: "0",
			Panes: []tmuxscan.Pane{{ID: "%1", Index: "0", Agent: tmuxscan.AgentCodex, Active: true}},
		}},
	}
	key := sessionStatusKey(host, session)
	m := model{
		cfg: config.Config{Hermes: config.HermesConfig{
			Enabled:        true,
			DoneAdvice:     true,
			Command:        "hermes",
			Args:           []string{"--oneshot"},
			TimeoutSeconds: 120,
		}},
		hosts:  []hostState{{host: host, snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{session}}, loaded: true}},
		hermes: map[string]hermesAdvice{},
	}

	cmd := m.autoHermesNextStepCmd(true, sessionWorking, sessionDone, key)
	if cmd == nil {
		t.Fatalf("autoHermesNextStepCmd() = nil, want Hermes command")
	}
	if !m.hermes[key].loading {
		t.Fatalf("hermes[%q].loading = false, want true", key)
	}

	again := m.autoHermesNextStepCmd(true, sessionDone, sessionDone, key)
	if again != nil {
		t.Fatalf("autoHermesNextStepCmd() while already done = %#v, want nil", again)
	}
}

func TestApplyHermesAutoNextStepSendsWhenSuggested(t *testing.T) {
	host := config.Host{Name: "local", Local: true}
	session := tmuxscan.Session{
		ID:   "$1",
		Name: "agents",
		Windows: []tmuxscan.Window{{
			ID:    "@1",
			Index: "0",
			Panes: []tmuxscan.Pane{{ID: "%1", Index: "0", Agent: tmuxscan.AgentCodex, Active: true}},
		}},
	}
	key := sessionStatusKey(host, session)
	m := model{
		cfg:      config.Config{Hermes: config.HermesConfig{Enabled: true, AutoDone: true}},
		hosts:    []hostState{{host: host, snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{session}}, loaded: true}},
		statuses: map[string]sessionStatus{key: sessionDone},
	}

	cmd := m.applyHermesAutoNextStep(hermesNextStepResult{
		key:         key,
		status:      sessionDone,
		text:        "SEND: 继续处理 README 里的下一项",
		auto:        true,
		host:        host,
		hostName:    "local",
		sessionName: "agents",
		target:      selectedAgentTarget{hostIndex: 0, target: "%1", agent: tmuxscan.AgentCodex},
	})
	if cmd == nil {
		t.Fatalf("applyHermesAutoNextStep() cmd = nil, want send command")
	}
	if got := m.statuses[key]; got != sessionWorking {
		t.Fatalf("status = %q, want working", got)
	}
	if len(m.activities) != 1 || m.activities[0].State != "auto sent" {
		t.Fatalf("activities = %#v, want auto sent activity", m.activities)
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
	workLog := filepath.Join(t.TempDir(), "hermes.jsonl")
	m := model{
		cfg: config.Config{Hermes: config.HermesConfig{WorkLog: workLog}},
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
			SessionKey:  key,
			HostName:    "local",
			SessionName: "agents",
			Agent:       tmuxscan.AgentCodex,
			Row:         row{attachTarget: "%1"},
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
	data, err := os.ReadFile(workLog)
	if err != nil {
		t.Fatalf("ReadFile(workLog) error = %v", err)
	}
	var entry hermeslog.Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("Unmarshal(workLog) error = %v; data = %q", err, string(data))
	}
	if entry.Flow != "review" || entry.Event != "reply" || entry.Advice == "" || entry.Host != "local" || entry.Session != "agents" {
		t.Fatalf("work log entry = %#v", entry)
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
		cfg:           config.Config{Hermes: config.HermesConfig{Enabled: true, AutoReview: true}},
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

func TestHermesAutoReviewAuditQQPromptIncludesDecisionAndAdvice(t *testing.T) {
	item := reviewItem{
		HostName:    "nebula",
		SessionName: "agents",
		Agent:       tmuxscan.AgentClaude,
		Row:         row{attachTarget: "agents:0.1"},
	}
	prompt := hermesAutoReviewAuditQQPrompt(item, "nebula", []string{
		"Run shell command?",
		"> 1. Allow command",
		"  2. Deny",
	}, "CHOOSE 1: visible approval prompt", "choose 1: visible approval prompt")

	for _, want := range []string{
		`send_message(target="qqbot", message=...)`,
		"auto review decision",
		"Host: nebula",
		"Session: agents",
		"Target: agents:0.1",
		"Hermes decision: choose 1",
		"CHOOSE 1",
		"1: Allow command",
		"Run shell command?",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestHermesAutoReviewAuditQQCmdOnlySendsUncertainInUncertainMode(t *testing.T) {
	m := model{cfg: config.Config{
		Notification: config.NotificationConfig{AutoReviewAuditQQ: config.AutoReviewAuditQQUncertain},
	}}
	item := reviewItem{HostName: "local", SessionName: "agents"}

	if cmd := m.hermesAutoReviewAuditQQCmd(item, "local", nil, "CHOOSE 1", "choose 1", false); cmd != nil {
		t.Fatalf("clear decision audit cmd = non-nil, want nil in uncertain mode")
	}
	if cmd := m.hermesAutoReviewAuditQQCmd(item, "local", nil, "I cannot decide", "needs human review", true); cmd == nil {
		t.Fatalf("uncertain decision audit cmd = nil, want QQ audit command")
	}
}

func TestNotifyQQForHermesAutoReviewSkipsWhenAuditDisabled(t *testing.T) {
	result := notifyQQForHermesAutoReview(config.Config{}, reviewItem{}, "local", nil, "CHOOSE 1", "choose 1")
	if result.Attempted {
		t.Fatalf("attempted = true, want false")
	}
	if result.Reason != "notification.auto_review_audit_qq is off" {
		t.Fatalf("reason = %q, want audit disabled reason", result.Reason)
	}
	if result.NeedsReviewCount != 1 {
		t.Fatalf("needs review count = %d, want 1", result.NeedsReviewCount)
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
