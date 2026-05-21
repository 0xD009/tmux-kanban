package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadHermesConfigDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("hosts:\n  - name: local\n    local: true\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Hermes.Enabled {
		t.Fatalf("hermes enabled = true, want false")
	}
	if cfg.Hermes.AutoReview {
		t.Fatalf("hermes auto_review = true, want false")
	}
	if cfg.Hermes.Command != "hermes" {
		t.Fatalf("hermes command = %q, want hermes", cfg.Hermes.Command)
	}
	if len(cfg.Hermes.Args) != 1 || cfg.Hermes.Args[0] != "--oneshot" {
		t.Fatalf("hermes args = %#v, want --oneshot", cfg.Hermes.Args)
	}
	if cfg.Hermes.TimeoutSeconds != 120 {
		t.Fatalf("hermes timeout = %d, want 120", cfg.Hermes.TimeoutSeconds)
	}
	if cfg.Notification.QQEnabled {
		t.Fatalf("notification qq_enabled = true, want false")
	}
	if !cfg.MainAgent.Enabled {
		t.Fatalf("main agent enabled = false, want true")
	}
	if cfg.MainAgent.Host != "local" {
		t.Fatalf("main agent host = %q, want local", cfg.MainAgent.Host)
	}
	if cfg.MainAgent.Session != "tmux-kanban-main" {
		t.Fatalf("main agent session = %q, want tmux-kanban-main", cfg.MainAgent.Session)
	}
	if cfg.MainAgent.Agent != "codex" || cfg.MainAgent.Command != "codex" {
		t.Fatalf("main agent = %#v, want codex defaults", cfg.MainAgent)
	}
}

func TestLoadHermesAutoReviewConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("hermes:\n  enabled: true\n  auto_review: true\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.Hermes.Enabled || !cfg.Hermes.AutoReview {
		t.Fatalf("hermes config = %#v, want enabled auto review", cfg.Hermes)
	}
}

func TestLoadNotificationConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("notification:\n  qq_enabled: true\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.Notification.QQEnabled {
		t.Fatalf("notification qq_enabled = false, want true")
	}
}

func TestLoadDebugSnapshotDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("debug:\n  snapshot_dir: /tmp/tmux-kanban-snapshots\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Debug.SnapshotDir != "/tmp/tmux-kanban-snapshots" {
		t.Fatalf("snapshot_dir = %q, want configured dir", cfg.Debug.SnapshotDir)
	}
}

func TestLoadMainAgentConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	config := []byte("main_agent:\n  enabled: true\n  host: nebula\n  session: conductor\n  agent: claude-code\n  args:\n    - --dangerously-skip-permissions\n")
	if err := os.WriteFile(path, config, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.MainAgent.Enabled {
		t.Fatalf("main agent enabled = false, want true")
	}
	if cfg.MainAgent.Host != "nebula" || cfg.MainAgent.Session != "conductor" {
		t.Fatalf("main agent target = %#v, want nebula/conductor", cfg.MainAgent)
	}
	if cfg.MainAgent.Command != "claude" {
		t.Fatalf("main agent command = %q, want claude", cfg.MainAgent.Command)
	}
	if len(cfg.MainAgent.Args) != 1 || cfg.MainAgent.Args[0] != "--dangerously-skip-permissions" {
		t.Fatalf("main agent args = %#v", cfg.MainAgent.Args)
	}
}

func TestLoadAgentMeshDefaults(t *testing.T) {
	cfg := Default()
	if cfg.AgentMesh.Enabled {
		t.Fatalf("agent mesh enabled = true, want default off")
	}
	if !cfg.AgentMesh.SharedShortAgent {
		t.Fatalf("shared_short_agent = false, want true")
	}
	if cfg.AgentMesh.DefaultAgent != "codex" {
		t.Fatalf("default agent = %q, want codex", cfg.AgentMesh.DefaultAgent)
	}
	if cfg.AgentMesh.SkillRoot != "mesh-skills" {
		t.Fatalf("skill root = %q, want mesh-skills", cfg.AgentMesh.SkillRoot)
	}
	if len(cfg.AgentMesh.Policies) != 4 {
		t.Fatalf("policies = %d, want 4", len(cfg.AgentMesh.Policies))
	}
	if cfg.AgentMesh.Policies[0].Backend != "codex" || cfg.AgentMesh.Policies[0].Skill == "" {
		t.Fatalf("policy backend/skill = %#v, want codex with skill", cfg.AgentMesh.Policies[0])
	}
}

func TestLoadAgentMeshConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	config := []byte("agent_mesh:\n  enabled: true\n  shared_short_agent: false\n  default_agent: claude-code\n  skill_root: ./mesh-skills\n  memory_root: /tmp/tmux-kanban-memory\n  policies:\n    - name: session-advice\n      role: review-advice\n      scope: session\n      backend: claude-code\n      skill: review-advice\n      agent: claude-code\n      enabled: true\n  mail:\n    enabled: true\n    dir: /tmp/tmux-kanban-mail\n")
	if err := os.WriteFile(path, config, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.AgentMesh.Enabled {
		t.Fatalf("agent mesh enabled = false, want true")
	}
	if cfg.AgentMesh.SharedShortAgent {
		t.Fatalf("shared_short_agent = true, want false")
	}
	if cfg.AgentMesh.Policies[0].Command != "claude" {
		t.Fatalf("policy command = %q, want claude", cfg.AgentMesh.Policies[0].Command)
	}
	if cfg.AgentMesh.Policies[0].Backend != "claude-code" || cfg.AgentMesh.Policies[0].Skill != "review-advice" {
		t.Fatalf("policy backend/skill = %#v", cfg.AgentMesh.Policies[0])
	}
	if cfg.AgentMesh.SkillRoot != "./mesh-skills" {
		t.Fatalf("skill root = %q, want ./mesh-skills", cfg.AgentMesh.SkillRoot)
	}
	if cfg.AgentMesh.Mail.Dir != "/tmp/tmux-kanban-mail" {
		t.Fatalf("mail dir = %q, want configured dir", cfg.AgentMesh.Mail.Dir)
	}
}
