package config

import (
	"fmt"
	"os"
	"strings"

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
	Enabled        bool                `yaml:"enabled"`
	AutoReview     bool                `yaml:"auto_review"`
	DoneAdvice     bool                `yaml:"done_advice"`
	AutoDone       bool                `yaml:"auto_done"`
	IdleAdvice     bool                `yaml:"idle_advice"`
	AutoIdle       bool                `yaml:"auto_idle"`
	Command        string              `yaml:"command"`
	Args           []string            `yaml:"args"`
	TimeoutSeconds int                 `yaml:"timeout_seconds"`
	WorkLog        string              `yaml:"work_log"`
	Scopes         []HermesScopeConfig `yaml:"scopes"`
}

type HermesScopeConfig struct {
	Host       string `yaml:"host"`
	Session    string `yaml:"session"`
	Enabled    *bool  `yaml:"enabled"`
	AutoReview *bool  `yaml:"auto_review"`
	DoneAdvice *bool  `yaml:"done_advice"`
	AutoDone   *bool  `yaml:"auto_done"`
	IdleAdvice *bool  `yaml:"idle_advice"`
	AutoIdle   *bool  `yaml:"auto_idle"`
}

type NotificationConfig struct {
	QQEnabled         bool                  `yaml:"qq_enabled"`
	TerminalReview    bool                  `yaml:"terminal_review"`
	AutoReviewAuditQQ AutoReviewAuditQQMode `yaml:"auto_review_audit_qq"`
}

type AutoReviewAuditQQMode string

const (
	AutoReviewAuditQQOff       AutoReviewAuditQQMode = "off"
	AutoReviewAuditQQUncertain AutoReviewAuditQQMode = "uncertain"
	AutoReviewAuditQQAll       AutoReviewAuditQQMode = "all"
)

func NormalizeAutoReviewAuditQQMode(value string) (AutoReviewAuditQQMode, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "off", "false", "no", "disabled", "none":
		return AutoReviewAuditQQOff, true
	case "uncertain", "unclear", "manual", "review", "human", "fail", "failed", "failure", "failures":
		return AutoReviewAuditQQUncertain, true
	case "all", "on", "true", "yes", "enabled", "always":
		return AutoReviewAuditQQAll, true
	default:
		return AutoReviewAuditQQOff, false
	}
}

func (m AutoReviewAuditQQMode) String() string {
	normalized, ok := NormalizeAutoReviewAuditQQMode(string(m))
	if !ok {
		return string(AutoReviewAuditQQOff)
	}
	return string(normalized)
}

func AutoReviewAuditQQEnabled(mode AutoReviewAuditQQMode) bool {
	normalized, ok := NormalizeAutoReviewAuditQQMode(string(mode))
	return ok && normalized != AutoReviewAuditQQOff
}

func ShouldSendAutoReviewAuditQQ(mode AutoReviewAuditQQMode, uncertain bool) bool {
	normalized, ok := NormalizeAutoReviewAuditQQMode(string(mode))
	if !ok {
		return false
	}
	switch normalized {
	case AutoReviewAuditQQAll:
		return true
	case AutoReviewAuditQQUncertain:
		return uncertain
	default:
		return false
	}
}

func (m *AutoReviewAuditQQMode) UnmarshalYAML(value *yaml.Node) error {
	if value == nil {
		*m = AutoReviewAuditQQOff
		return nil
	}
	if value.Tag == "!!bool" {
		if strings.EqualFold(value.Value, "true") {
			*m = AutoReviewAuditQQAll
		} else {
			*m = AutoReviewAuditQQOff
		}
		return nil
	}
	normalized, ok := NormalizeAutoReviewAuditQQMode(value.Value)
	if !ok {
		return fmt.Errorf("invalid notification.auto_review_audit_qq %q (want off, uncertain, or all)", value.Value)
	}
	*m = normalized
	return nil
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
			WorkLog:        "~/.local/state/tmux-kanban/hermes-worklog.jsonl",
		},
		Notification: NotificationConfig{
			AutoReviewAuditQQ: AutoReviewAuditQQOff,
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
	if cfg.Hermes.WorkLog == "" {
		cfg.Hermes.WorkLog = Default().Hermes.WorkLog
	}
	if normalized, ok := NormalizeAutoReviewAuditQQMode(string(cfg.Notification.AutoReviewAuditQQ)); ok {
		cfg.Notification.AutoReviewAuditQQ = normalized
	} else {
		cfg.Notification.AutoReviewAuditQQ = AutoReviewAuditQQOff
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

func HostDisplayName(host Host) string {
	if host.Name != "" {
		return host.Name
	}
	if host.SSH != "" {
		return host.SSH
	}
	return "local"
}

func (cfg HermesConfig) Resolve(host Host, session string) HermesConfig {
	resolved := cfg
	resolved.Scopes = nil
	for _, scope := range cfg.Scopes {
		if !scope.Matches(host, session) {
			continue
		}
		scope.Apply(&resolved)
	}
	return resolved
}

func (scope HermesScopeConfig) Matches(host Host, session string) bool {
	scopeHost := strings.TrimSpace(scope.Host)
	scopeSession := strings.TrimSpace(scope.Session)
	if !hermesScopeHostMatches(scopeHost, host) {
		return false
	}
	if scopeSession == "" || isHermesAllScope(scopeSession) {
		return true
	}
	session = strings.TrimSpace(session)
	if scopedHost, scopedSession, ok := strings.Cut(scopeSession, "/"); ok {
		if !hermesScopeHostMatches(scopedHost, host) {
			return false
		}
		return isHermesAllScope(scopedSession) || strings.TrimSpace(scopedSession) == session
	}
	return scopeSession == session
}

func hermesScopeHostMatches(scopeHost string, host Host) bool {
	scopeHost = strings.TrimSpace(scopeHost)
	return scopeHost == "" || isHermesAllScope(scopeHost) || scopeHost == host.Name || scopeHost == host.SSH || scopeHost == HostDisplayName(host)
}

func isHermesAllScope(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "all")
}

func (scope HermesScopeConfig) Apply(cfg *HermesConfig) {
	if scope.Enabled != nil {
		cfg.Enabled = *scope.Enabled
	}
	if scope.AutoReview != nil {
		cfg.AutoReview = *scope.AutoReview
	}
	if scope.DoneAdvice != nil {
		cfg.DoneAdvice = *scope.DoneAdvice
	}
	if scope.AutoDone != nil {
		cfg.AutoDone = *scope.AutoDone
	}
	if scope.IdleAdvice != nil {
		cfg.IdleAdvice = *scope.IdleAdvice
	}
	if scope.AutoIdle != nil {
		cfg.AutoIdle = *scope.AutoIdle
	}
}
