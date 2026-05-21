package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"tmux-kanban/internal/config"
)

func (m model) executeViewCommand(args []string) (model, tea.Cmd) {
	if len(args) != 1 {
		m.status = "usage: view tree|review|main"
		return m, nil
	}
	switch strings.ToLower(args[0]) {
	case "tree":
		m.setViewMode(viewTree)
		return m, nil
	case "review", "queue":
		m.setViewMode(viewReview)
		return m, nil
	case "main", "cockpit":
		return m.startMainSession()
	default:
		m.status = "usage: view tree|review|main"
		return m, nil
	}
}

func (m model) executeMainCommand(args []string) (model, tea.Cmd) {
	if len(args) == 0 {
		return m.startMainSession()
	}

	switch strings.ToLower(args[0]) {
	case "start", "show", "on", "room":
		m.cfg.MainAgent.Enabled = true
		return m.startMainSession()
	case "hide", "off":
		m.mainActive = false
		m.setViewMode(viewTree)
		m.preview = previewState{}
		m.compose = composeState{}
		m.status = "main room hidden"
		return m, m.ensurePreview()
	case "codex":
		m.cfg.MainAgent.Enabled = true
		m.setMainAgent("codex")
		return m.startMainSession()
	case "claude", "claude-code":
		m.cfg.MainAgent.Enabled = true
		m.setMainAgent("claude-code")
		return m.startMainSession()
	case "host":
		if len(args) != 2 {
			m.status = "usage: main host <host-name>"
			return m, nil
		}
		m.cfg.MainAgent.Host = args[1]
		m.preview = previewState{}
		m.status = "main host set to " + args[1]
		return m, nil
	case "session":
		if len(args) != 2 {
			m.status = "usage: main session <tmux-session>"
			return m, nil
		}
		m.cfg.MainAgent.Session = args[1]
		m.preview = previewState{}
		m.status = "main session set to " + args[1]
		return m, nil
	case "command":
		if len(args) < 2 {
			m.status = "usage: main command <command> [args...]"
			return m, nil
		}
		m.cfg.MainAgent.Command = args[1]
		m.cfg.MainAgent.Args = append([]string(nil), args[2:]...)
		m.status = "main command set to " + strings.Join(args[1:], " ")
		return m, nil
	case "args":
		m.cfg.MainAgent.Args = append([]string(nil), args[1:]...)
		m.status = "main args set"
		return m, nil
	case "status", "settings":
		m.status = fmt.Sprintf("main: enabled=%s host=%s session=%s agent=%s command=%s args=%s", onOff(m.cfg.MainAgent.Enabled), m.cfg.MainAgent.Host, m.mainSessionName(), m.cfg.MainAgent.Agent, m.cfg.MainAgent.Command, strings.Join(m.cfg.MainAgent.Args, " "))
		return m, nil
	default:
		m.status = "usage: main start|room|codex|claude|host|session|command|args|hide|status"
		return m, nil
	}
}

func (m *model) setMainAgent(agentName string) {
	normalized := normalizeConfigAgent(agentName)
	m.cfg.MainAgent.Agent = normalized
	m.cfg.MainAgent.Command = config.DefaultMainAgentCommand(normalized)
}
