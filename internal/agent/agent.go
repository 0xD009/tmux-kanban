package agent

import (
	"context"

	"tmux-kanban/internal/tmuxscan"
)

type Kind = tmuxscan.AgentKind

const (
	None   = tmuxscan.AgentNone
	Codex  = tmuxscan.AgentCodex
	Claude = tmuxscan.AgentClaude
)

type Screen = tmuxscan.AgentScreen
type Choice = tmuxscan.AgentChoice

type Process struct {
	Command string
	Args    string
}

type Target struct {
	Key       string
	HostIndex int
	Target    string
	Label     string
	Agent     Kind
}

type TargetResolver interface {
	Targets() []Target
}

type ReviewRequest struct {
	Host        string
	Target      string
	SessionName string
	Agent       Kind
	Screen      Screen
	Capture     []string
	Intent      string
}

type ReviewAdvice struct {
	Text   string
	Choice string
}

type Reviewer interface {
	Review(ctx context.Context, request ReviewRequest) (ReviewAdvice, error)
}

func AnalyzeScreen(lines []string) Screen {
	return tmuxscan.AnalyzeAgentScreen(lines)
}

func Detect(command string, processes []Process) Kind {
	pane := tmuxscan.Pane{Command: command}
	for _, process := range processes {
		pane.Processes = append(pane.Processes, tmuxscan.Process{
			Command: process.Command,
			Args:    process.Args,
		})
	}
	return tmuxscan.DetectAgent(pane)
}
