package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Hosts        []Host             `yaml:"hosts"`
	Kanban       KanbanConfig       `yaml:"kanban"`
	MainAgent    MainAgentConfig    `yaml:"main_agent"`
	AgentMesh    AgentMeshConfig    `yaml:"agent_mesh"`
	Hermes       HermesConfig       `yaml:"hermes"`
	Notification NotificationConfig `yaml:"notification"`
	Debug        DebugConfig        `yaml:"debug"`
}

type Host struct {
	Name  string `yaml:"name"`
	SSH   string `yaml:"ssh"`
	Local bool   `yaml:"local"`
}

type KanbanConfig struct {
	Columns []string `yaml:"columns"`
}

type MainAgentConfig struct {
	Enabled bool     `yaml:"enabled"`
	Host    string   `yaml:"host"`
	Session string   `yaml:"session"`
	Agent   string   `yaml:"agent"`
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
}

type AgentMeshConfig struct {
	Enabled          bool                `yaml:"enabled"`
	SharedShortAgent bool                `yaml:"shared_short_agent"`
	DefaultAgent     string              `yaml:"default_agent"`
	SkillRoot        string              `yaml:"skill_root"`
	MemoryRoot       string              `yaml:"memory_root"`
	Policies         []AgentPolicyConfig `yaml:"policies"`
	Mail             AgentMailConfig     `yaml:"mail"`
}

type AgentPolicyConfig struct {
	Name    string   `yaml:"name"`
	Role    string   `yaml:"role"`
	Scope   string   `yaml:"scope"`
	Backend string   `yaml:"backend"`
	Skill   string   `yaml:"skill"`
	Agent   string   `yaml:"agent"`
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
	Enabled bool     `yaml:"enabled"`
}

type AgentMailConfig struct {
	Enabled bool   `yaml:"enabled"`
	Dir     string `yaml:"dir"`
}

type HermesConfig struct {
	Enabled        bool     `yaml:"enabled"`
	AutoReview     bool     `yaml:"auto_review"`
	Command        string   `yaml:"command"`
	Args           []string `yaml:"args"`
	TimeoutSeconds int      `yaml:"timeout_seconds"`
}

type NotificationConfig struct {
	QQEnabled bool `yaml:"qq_enabled"`
}

type DebugConfig struct {
	SnapshotDir string `yaml:"snapshot_dir"`
}

func Default() Config {
	return Config{
		Hosts: []Host{
			{Name: "local", Local: true},
		},
		Kanban: KanbanConfig{
			Columns: []string{"Idle", "Working", "Need Review"},
		},
		MainAgent: MainAgentConfig{
			Enabled: true,
			Host:    "local",
			Session: "tmux-kanban-main",
			Agent:   "codex",
			Command: "codex",
		},
		AgentMesh: AgentMeshConfig{
			SharedShortAgent: true,
			DefaultAgent:     "codex",
			SkillRoot:        "mesh-skills",
			Policies: []AgentPolicyConfig{
				{Name: "review-permission", Role: "review-permission", Scope: "session", Backend: "codex", Skill: "review-permission", Agent: "codex", Enabled: true},
				{Name: "review-advice", Role: "review-advice", Scope: "session", Backend: "codex", Skill: "review-advice", Agent: "codex", Enabled: true},
				{Name: "dispatcher", Role: "dispatcher", Scope: "session", Backend: "codex", Skill: "dispatcher", Agent: "codex", Enabled: true},
				{Name: "session-link", Role: "session-link", Scope: "host", Backend: "codex", Skill: "session-link", Agent: "codex", Enabled: true},
			},
			Mail: AgentMailConfig{Enabled: true},
		},
		Hermes: HermesConfig{
			Command:        "hermes",
			Args:           []string{"--oneshot"},
			TimeoutSeconds: 120,
		},
	}
}

func Load(path string) (Config, error) {
	if path == "" {
		return Default(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	if len(cfg.Kanban.Columns) == 0 {
		cfg.Kanban.Columns = Default().Kanban.Columns
	}
	if cfg.MainAgent.Host == "" {
		cfg.MainAgent.Host = Default().MainAgent.Host
	}
	if cfg.MainAgent.Session == "" {
		cfg.MainAgent.Session = Default().MainAgent.Session
	}
	if cfg.MainAgent.Agent == "" {
		cfg.MainAgent.Agent = Default().MainAgent.Agent
	}
	if cfg.MainAgent.Command == "" || (cfg.MainAgent.Command == Default().MainAgent.Command && cfg.MainAgent.Agent != Default().MainAgent.Agent) {
		cfg.MainAgent.Command = DefaultMainAgentCommand(cfg.MainAgent.Agent)
	}
	if cfg.MainAgent.Args == nil {
		cfg.MainAgent.Args = Default().MainAgent.Args
	}
	if cfg.AgentMesh.DefaultAgent == "" {
		cfg.AgentMesh.DefaultAgent = Default().AgentMesh.DefaultAgent
	}
	if cfg.AgentMesh.SkillRoot == "" {
		cfg.AgentMesh.SkillRoot = Default().AgentMesh.SkillRoot
	}
	if cfg.AgentMesh.Policies == nil {
		cfg.AgentMesh.Policies = Default().AgentMesh.Policies
	}
	for i := range cfg.AgentMesh.Policies {
		if cfg.AgentMesh.Policies[i].Agent == "" {
			cfg.AgentMesh.Policies[i].Agent = cfg.AgentMesh.DefaultAgent
		}
		if cfg.AgentMesh.Policies[i].Backend == "" {
			cfg.AgentMesh.Policies[i].Backend = cfg.AgentMesh.Policies[i].Agent
		}
		if cfg.AgentMesh.Policies[i].Skill == "" {
			cfg.AgentMesh.Policies[i].Skill = cfg.AgentMesh.Policies[i].Role
		}
		if cfg.AgentMesh.Policies[i].Command == "" {
			cfg.AgentMesh.Policies[i].Command = DefaultMainAgentCommand(cfg.AgentMesh.Policies[i].Agent)
		}
		if cfg.AgentMesh.Policies[i].Args == nil {
			cfg.AgentMesh.Policies[i].Args = []string{}
		}
	}
	if cfg.Hermes.Command == "" {
		cfg.Hermes.Command = Default().Hermes.Command
	}
	if cfg.Hermes.Args == nil {
		cfg.Hermes.Args = Default().Hermes.Args
	}
	if cfg.Hermes.TimeoutSeconds <= 0 {
		cfg.Hermes.TimeoutSeconds = Default().Hermes.TimeoutSeconds
	}

	return cfg, nil
}

func DefaultMainAgentCommand(agent string) string {
	switch agent {
	case "claude", "claude-code":
		return "claude"
	default:
		return "codex"
	}
}
