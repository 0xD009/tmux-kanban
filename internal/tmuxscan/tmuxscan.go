package tmuxscan

import (
	"context"
	"strings"
	"time"

	"tmux-kanban/internal/config"
)

type Snapshot struct {
	Host       config.Host
	Sessions   []Session
	ScannedAt  time.Time
	Err        string
	RawSummary string
}

type Session struct {
	ID       string
	Name     string
	Windows  []Window
	Attached int
}

type Window struct {
	ID     string
	Index  string
	Name   string
	Panes  []Pane
	Active bool
}

type AgentKind string

const (
	AgentNone   AgentKind = ""
	AgentCodex  AgentKind = "codex"
	AgentClaude AgentKind = "claude-code"
)

type Pane struct {
	ID          string
	Index       string
	PID         string
	Command     string
	CurrentPath string
	Active      bool
	Processes   []Process
	Agent       AgentKind
}

type Process struct {
	PID     string
	Command string
	Args    string
}

type Capture struct {
	Host       config.Host
	Target     string
	Lines      []string
	CapturedAt time.Time
	Err        string
}

func ScanHost(ctx context.Context, host config.Host) Snapshot {
	snapshot := Snapshot{Host: host, ScannedAt: time.Now()}

	if !host.Local && strings.TrimSpace(host.SSH) == "" {
		snapshot.Err = "missing ssh target"
		return snapshot
	}

	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	cmd := listCommand(ctx, host)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		snapshot.Err = "scan timed out"
		return snapshot
	}
	if err != nil {
		snapshot.Err = compactError(err, output)
		return snapshot
	}

	sessions, parseErr := Parse(string(output))
	if parseErr != nil {
		snapshot.Err = parseErr.Error()
		snapshot.RawSummary = strings.TrimSpace(string(output))
		return snapshot
	}

	snapshot.Sessions = sessions
	return snapshot
}
