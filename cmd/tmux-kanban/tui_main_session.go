package main

import (
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
