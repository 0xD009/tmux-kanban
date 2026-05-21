package main

import (
	"fmt"
	"strings"

	"tmux-kanban/internal/config"
	"tmux-kanban/internal/tmuxscan"
)

func (m model) mainHostIndex() (int, bool) {
	configured := strings.TrimSpace(m.cfg.MainAgent.Host)
	for i, state := range m.hosts {
		host := state.host
		if configured == "" {
			if host.Local {
				return i, true
			}
			continue
		}
		if host.Name == configured || host.SSH == configured || displayHostName(host) == configured {
			return i, true
		}
	}
	if configured == "" && len(m.hosts) > 0 {
		return 0, true
	}
	return 0, false
}

func (m model) mainSessionName() string {
	session := strings.TrimSpace(m.cfg.MainAgent.Session)
	if session == "" {
		return config.Default().MainAgent.Session
	}
	return session
}

func (m model) mainAgentKind() tmuxscan.AgentKind {
	switch strings.ToLower(strings.TrimSpace(m.cfg.MainAgent.Agent)) {
	case "claude", "claude-code":
		return tmuxscan.AgentClaude
	case "codex", "":
		return tmuxscan.AgentCodex
	default:
		return tmuxscan.AgentKind(m.cfg.MainAgent.Agent)
	}
}

func (m model) mainAgentName() string {
	agent := m.mainAgentKind()
	if agent == tmuxscan.AgentNone {
		return "main"
	}
	return string(agent)
}

func (m model) mainSessionLabel() string {
	hostIndex, ok := m.mainHostIndex()
	hostName := strings.TrimSpace(m.cfg.MainAgent.Host)
	if ok {
		hostName = displayHostName(m.hosts[hostIndex].host)
	}
	if hostName == "" {
		hostName = "main"
	}
	label := hostName + "/" + m.mainSessionName()
	if agent := m.mainAgentName(); agent != "" {
		label += " (" + agent + ")"
	}
	return label
}

func (m model) isMainSession(hostIndex int, session tmuxscan.Session) bool {
	if !m.cfg.MainAgent.Enabled || strings.TrimSpace(session.Name) == "" {
		return false
	}
	mainHostIndex, ok := m.mainHostIndex()
	if !ok || mainHostIndex != hostIndex {
		return false
	}
	return session.Name == m.mainSessionName()
}

func (m model) mainAgentTarget() (selectedAgentTarget, bool) {
	hostIndex, ok := m.mainHostIndex()
	if !ok {
		return selectedAgentTarget{}, false
	}
	session := m.mainSessionName()
	if strings.TrimSpace(session) == "" {
		return selectedAgentTarget{}, false
	}
	return selectedAgentTarget{
		key:       fmt.Sprintf("main-session:%d:%s", hostIndex, session),
		hostIndex: hostIndex,
		target:    session,
		label:     session + " (" + m.mainAgentName() + ")",
		agent:     m.mainAgentKind(),
	}, true
}
