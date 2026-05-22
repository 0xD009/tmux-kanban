package debug

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tmux-kanban/internal/config"
)

type Snapshot struct {
	Version     int             `json:"version"`
	CreatedAt   time.Time       `json:"created_at"`
	Description string          `json:"description,omitempty"`
	Config      ConfigSummary   `json:"config"`
	Runtime     RuntimeState    `json:"runtime"`
	Hosts       []HostSnapshot  `json:"hosts"`
	ReviewQueue []ReviewItem    `json:"review_queue"`
	Activities  []AgentActivity `json:"agent_activities,omitempty"`
	Preview     PreviewState    `json:"preview"`
	Errors      []string        `json:"errors,omitempty"`
}

type ConfigSummary struct {
	Hosts        []config.Host `json:"hosts"`
	MainAgent    MainSummary   `json:"main_agent"`
	AgentMesh    MeshSummary   `json:"agent_mesh"`
	Hermes       HermesSummary `json:"hermes"`
	Notification struct {
		QQEnabled bool `json:"qq_enabled"`
	} `json:"notification"`
	Debug struct {
		SnapshotDir string `json:"snapshot_dir,omitempty"`
	} `json:"debug"`
}

type MainSummary struct {
	Enabled bool     `json:"enabled"`
	Host    string   `json:"host"`
	Session string   `json:"session"`
	Agent   string   `json:"agent"`
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
}

type MeshSummary struct {
	Enabled          bool            `json:"enabled"`
	SharedShortAgent bool            `json:"shared_short_agent"`
	DefaultAgent     string          `json:"default_agent"`
	SkillRoot        string          `json:"skill_root,omitempty"`
	MemoryRoot       string          `json:"memory_root,omitempty"`
	Policies         []PolicySummary `json:"policies,omitempty"`
	Mail             struct {
		Enabled bool   `json:"enabled"`
		Dir     string `json:"dir,omitempty"`
	} `json:"mail"`
}

type PolicySummary struct {
	Name    string   `json:"name"`
	Role    string   `json:"role"`
	Scope   string   `json:"scope"`
	Backend string   `json:"backend,omitempty"`
	Skill   string   `json:"skill,omitempty"`
	Agent   string   `json:"agent"`
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	Enabled bool     `json:"enabled"`
}

type HermesSummary struct {
	Enabled        bool                 `json:"enabled"`
	AutoReview     bool                 `json:"auto_review"`
	DoneAdvice     bool                 `json:"done_advice"`
	AutoDone       bool                 `json:"auto_done"`
	IdleAdvice     bool                 `json:"idle_advice"`
	AutoIdle       bool                 `json:"auto_idle"`
	Command        string               `json:"command"`
	Args           []string             `json:"args,omitempty"`
	TimeoutSeconds int                  `json:"timeout_seconds"`
	WorkLog        string               `json:"work_log,omitempty"`
	Scopes         []HermesScopeSummary `json:"scopes,omitempty"`
}

type HermesScopeSummary struct {
	Host       string `json:"host,omitempty"`
	Session    string `json:"session,omitempty"`
	Enabled    *bool  `json:"enabled,omitempty"`
	AutoReview *bool  `json:"auto_review,omitempty"`
	DoneAdvice *bool  `json:"done_advice,omitempty"`
	AutoDone   *bool  `json:"auto_done,omitempty"`
	IdleAdvice *bool  `json:"idle_advice,omitempty"`
	AutoIdle   *bool  `json:"auto_idle,omitempty"`
}

type RuntimeState struct {
	ViewMode        string            `json:"view_mode"`
	Status          string            `json:"status"`
	SessionStatuses map[string]string `json:"session_statuses,omitempty"`
	ReviewTargets   map[string]string `json:"review_targets,omitempty"`
	SkippedReview   []string          `json:"skipped_review,omitempty"`
	ReviewCursor    int               `json:"review_cursor"`
	ReviewCursorKey string            `json:"review_cursor_key,omitempty"`
}

type HostSnapshot struct {
	Name     string        `json:"name"`
	SSH      string        `json:"ssh,omitempty"`
	Local    bool          `json:"local"`
	Loading  bool          `json:"loading"`
	Loaded   bool          `json:"loaded"`
	Error    string        `json:"error,omitempty"`
	Sessions []interface{} `json:"sessions"`
}

type ReviewItem struct {
	SessionKey   string   `json:"session_key"`
	Host         string   `json:"host"`
	SessionName  string   `json:"session_name"`
	Agent        string   `json:"agent"`
	Target       string   `json:"target"`
	ScreenStatus string   `json:"screen_status,omitempty"`
	NeedsReview  bool     `json:"needs_review,omitempty"`
	Capture      []string `json:"capture,omitempty"`
}

type AgentActivity struct {
	At      time.Time `json:"at"`
	Source  string    `json:"source"`
	Agent   string    `json:"agent,omitempty"`
	Target  string    `json:"target,omitempty"`
	State   string    `json:"state,omitempty"`
	Message string    `json:"message,omitempty"`
}

type PreviewState struct {
	Key        string    `json:"key,omitempty"`
	HostIndex  int       `json:"host_index"`
	Target     string    `json:"target,omitempty"`
	Loading    bool      `json:"loading"`
	Refreshing bool      `json:"refreshing"`
	Error      string    `json:"error,omitempty"`
	CapturedAt time.Time `json:"captured_at,omitempty"`
	Lines      []string  `json:"lines,omitempty"`
}

func NewConfigSummary(cfg config.Config) ConfigSummary {
	summary := ConfigSummary{
		Hosts: cfg.Hosts,
		MainAgent: MainSummary{
			Enabled: cfg.MainAgent.Enabled,
			Host:    cfg.MainAgent.Host,
			Session: cfg.MainAgent.Session,
			Agent:   cfg.MainAgent.Agent,
			Command: cfg.MainAgent.Command,
			Args:    append([]string(nil), cfg.MainAgent.Args...),
		},
		AgentMesh: MeshSummary{
			Enabled:          cfg.AgentMesh.Enabled,
			SharedShortAgent: cfg.AgentMesh.SharedShortAgent,
			DefaultAgent:     cfg.AgentMesh.DefaultAgent,
			SkillRoot:        cfg.AgentMesh.SkillRoot,
			MemoryRoot:       cfg.AgentMesh.MemoryRoot,
			Policies:         make([]PolicySummary, 0, len(cfg.AgentMesh.Policies)),
		},
		Hermes: HermesSummary{
			Enabled:        cfg.Hermes.Enabled,
			AutoReview:     cfg.Hermes.AutoReview,
			DoneAdvice:     cfg.Hermes.DoneAdvice,
			AutoDone:       cfg.Hermes.AutoDone,
			IdleAdvice:     cfg.Hermes.IdleAdvice,
			AutoIdle:       cfg.Hermes.AutoIdle,
			Command:        cfg.Hermes.Command,
			Args:           append([]string(nil), cfg.Hermes.Args...),
			TimeoutSeconds: cfg.Hermes.TimeoutSeconds,
			WorkLog:        cfg.Hermes.WorkLog,
			Scopes:         make([]HermesScopeSummary, 0, len(cfg.Hermes.Scopes)),
		},
	}
	for _, scope := range cfg.Hermes.Scopes {
		summary.Hermes.Scopes = append(summary.Hermes.Scopes, HermesScopeSummary{
			Host:       scope.Host,
			Session:    scope.Session,
			Enabled:    copyBoolPtr(scope.Enabled),
			AutoReview: copyBoolPtr(scope.AutoReview),
			DoneAdvice: copyBoolPtr(scope.DoneAdvice),
			AutoDone:   copyBoolPtr(scope.AutoDone),
			IdleAdvice: copyBoolPtr(scope.IdleAdvice),
			AutoIdle:   copyBoolPtr(scope.AutoIdle),
		})
	}
	for _, policy := range cfg.AgentMesh.Policies {
		summary.AgentMesh.Policies = append(summary.AgentMesh.Policies, PolicySummary{
			Name:    policy.Name,
			Role:    policy.Role,
			Scope:   policy.Scope,
			Backend: policy.Backend,
			Skill:   policy.Skill,
			Agent:   policy.Agent,
			Command: policy.Command,
			Args:    append([]string(nil), policy.Args...),
			Enabled: policy.Enabled,
		})
	}
	summary.AgentMesh.Mail.Enabled = cfg.AgentMesh.Mail.Enabled
	summary.AgentMesh.Mail.Dir = cfg.AgentMesh.Mail.Dir
	summary.Notification.QQEnabled = cfg.Notification.QQEnabled
	summary.Debug.SnapshotDir = cfg.Debug.SnapshotDir
	return summary
}

func copyBoolPtr(value *bool) *bool {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func DefaultSnapshotDir() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return filepath.Join(os.TempDir(), "tmux-kanban", "snapshots")
	}
	return filepath.Join(home, ".local", "state", "tmux-kanban", "snapshots")
}

func ResolveSnapshotDir(cfg config.Config) string {
	if strings.TrimSpace(cfg.Debug.SnapshotDir) != "" {
		return cfg.Debug.SnapshotDir
	}
	return DefaultSnapshotDir()
}

func WriteSnapshot(dir string, snapshot Snapshot) (string, error) {
	if strings.TrimSpace(dir) == "" {
		dir = DefaultSnapshotDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create snapshot dir: %w", err)
	}
	if snapshot.CreatedAt.IsZero() {
		snapshot.CreatedAt = time.Now()
	}
	if snapshot.Version == 0 {
		snapshot.Version = 1
	}

	name := "snapshot-" + snapshot.CreatedAt.Format("20060102-150405.000") + ".json"
	path := filepath.Join(dir, name)
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode snapshot: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write snapshot: %w", err)
	}
	return path, nil
}
