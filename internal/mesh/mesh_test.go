package mesh

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"tmux-kanban/internal/config"
)

func TestScopePathBuildsMemoryHierarchy(t *testing.T) {
	scope := Scope{Host: "local", Session: "agents", Window: "2", Pane: "%3"}
	path := scope.Path()
	want := []string{"global", "host/local", "session/local/agents", "window/local/agents/2", "pane/local/agents/2/%3"}
	if len(path) != len(want) {
		t.Fatalf("path length = %d, want %d: %#v", len(path), len(want), path)
	}
	for i, scope := range path {
		if got := scope.Key(); got != want[i] {
			t.Fatalf("path[%d] = %q, want %q", i, got, want[i])
		}
	}
}

func TestMemoryTreeReturnsRootToLeafContext(t *testing.T) {
	tree := NewMemoryTree()
	at := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	tree.Upsert(Scope{}, "global", "global summary", at)
	tree.Upsert(Scope{Host: "local"}, "host", "host summary", at)
	tree.Upsert(Scope{Host: "local", Session: "agents"}, "session", "session summary", at)

	context := tree.Context(Scope{Host: "local", Session: "agents", Window: "1"})
	if len(context) != 3 {
		t.Fatalf("context length = %d, want 3: %#v", len(context), context)
	}
	if context[0].Summary != "global summary" || context[2].Summary != "session summary" {
		t.Fatalf("context = %#v, want global to session summaries", context)
	}
}

func TestLocalMemoryContextReadsRootToLeafFiles(t *testing.T) {
	root := t.TempDir()
	scope := Scope{Host: "local", Session: "agents", Window: "0", Pane: "%1"}
	files := map[Scope]string{
		{}:                                 "global memory",
		{Host: "local"}:                    "host memory",
		{Host: "local", Session: "agents"}: "session memory",
		{Host: "local", Session: "other"}:  "wrong session",
		{Host: "local", Session: "agents", Window: "0", Pane: "%1"}: "pane memory",
	}
	for fileScope, text := range files {
		path := LocalMemoryPath(root, fileScope)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
	}

	context, err := LocalMemoryContext(root, scope, 0)
	if err != nil {
		t.Fatalf("LocalMemoryContext() error = %v", err)
	}
	want := []string{"global memory", "host memory", "session memory", "pane memory"}
	if len(context) != len(want) {
		t.Fatalf("context length = %d, want %d: %#v", len(context), len(want), context)
	}
	for i, node := range context {
		if node.Summary != want[i] {
			t.Fatalf("context[%d] = %q, want %q", i, node.Summary, want[i])
		}
	}
}

func TestSpecsForScopeBuildsSharedSessionAgents(t *testing.T) {
	cfg := config.Default().AgentMesh
	cfg.Enabled = true
	scope := Scope{Host: "local", Session: "agents"}

	specs := SpecsForScope(cfg, scope)
	if len(specs) != 3 {
		t.Fatalf("spec count = %d, want 3 session-scoped specs: %#v", len(specs), specs)
	}
	for _, spec := range specs {
		if !spec.Shared {
			t.Fatalf("spec.Shared = false, want true: %#v", spec)
		}
		if spec.Scope.Key() != scope.Key() {
			t.Fatalf("spec scope = %q, want %q", spec.Scope.Key(), scope.Key())
		}
		if spec.Command != "codex" {
			t.Fatalf("spec command = %q, want codex", spec.Command)
		}
		if spec.Backend != "codex" {
			t.Fatalf("spec backend = %q, want codex", spec.Backend)
		}
		if spec.Skill == "" || spec.SkillPath != "mesh-skills/"+spec.Skill+"/SKILL.md" {
			t.Fatalf("spec skill = %q path = %q, want mesh skill path", spec.Skill, spec.SkillPath)
		}
	}
}

func TestSpecsForScopeCarriesClaudeBackendAndSkill(t *testing.T) {
	cfg := config.Default().AgentMesh
	cfg.Enabled = true
	cfg.SkillRoot = "custom-skills"
	cfg.Policies = []config.AgentPolicyConfig{{
		Name:    "session-advice",
		Role:    "review-advice",
		Scope:   "session",
		Backend: "claude-code",
		Skill:   "review-advice",
		Agent:   "claude-code",
		Enabled: true,
	}}

	specs := SpecsForScope(cfg, Scope{Host: "local", Session: "agents"})
	if len(specs) != 1 {
		t.Fatalf("spec count = %d, want 1: %#v", len(specs), specs)
	}
	spec := specs[0]
	if spec.Backend != "claude-code" || spec.Command != "claude" {
		t.Fatalf("spec = %#v, want claude backend and command", spec)
	}
	if spec.SkillPath != "custom-skills/review-advice/SKILL.md" {
		t.Fatalf("skill path = %q, want custom skill path", spec.SkillPath)
	}
}

func TestMailboxSendsAndFiltersInbox(t *testing.T) {
	var box Mailbox
	at := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	scope := Scope{Host: "local", Session: "agents"}
	box.Send("review-advice", "dispatcher", "needs follow-up", "ask the session agent to continue", scope, at)
	box.Send("dispatcher", "review-advice", "done", "sent", scope, at.Add(time.Second))

	inbox := box.Inbox("dispatcher")
	if len(inbox) != 1 {
		t.Fatalf("dispatcher inbox = %d, want 1: %#v", len(inbox), inbox)
	}
	if inbox[0].Status != MailQueued || inbox[0].Scope.Key() != scope.Key() {
		t.Fatalf("mail = %#v, want queued scoped mail", inbox[0])
	}
}
