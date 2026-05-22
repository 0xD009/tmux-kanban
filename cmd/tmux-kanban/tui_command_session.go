package main

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"tmux-kanban/internal/config"
	tmuxclient "tmux-kanban/internal/tmux"
)

func (m model) executeSessionCommand(args []string) (model, tea.Cmd) {
	if len(args) == 0 {
		m.status = "usage: session open <name|host/name> | session close [here|host/name] | session close confirm <host/name>"
		return m, nil
	}

	switch strings.ToLower(args[0]) {
	case "open", "start":
		if len(args) < 2 {
			m.status = "usage: session open <name|host/name>"
			return m, nil
		}
		host, session, ok := m.resolveSessionCommandTarget(args[1])
		if !ok {
			return m, nil
		}
		m.status = "opening session " + sessionCloseConfirmationToken(host, session) + "..."
		return m, startSessionCmd(host, session)
	case "close", "kill":
		return m.executeSessionCloseCommand(args[1:])
	default:
		m.status = "usage: session open <name|host/name> | session close [here|host/name]"
		return m, nil
	}
}

func (m model) executeSessionCloseCommand(args []string) (model, tea.Cmd) {
	if len(args) > 0 && strings.EqualFold(args[0], "confirm") {
		if len(args) != 2 {
			m.status = "usage: session close confirm <host/session>"
			return m, nil
		}
		token := strings.TrimSpace(args[1])
		if m.sessionClose.token == "" || token != m.sessionClose.token {
			m.status = "close confirmation mismatch"
			return m, nil
		}
		pending := m.sessionClose
		m.sessionClose = pendingSessionClose{}
		m.status = "closing session " + pending.token + "..."
		return m, closeSessionCmd(pending.host, pending.session)
	}

	host, session, ok := m.sessionCloseTarget(args)
	if !ok {
		return m, nil
	}
	token := sessionCloseConfirmationToken(host, session)
	m.sessionClose = pendingSessionClose{host: host, session: session, token: token}
	m.status = "confirm close " + token + ": run :session close confirm " + token
	return m, nil
}

func (m model) sessionCloseTarget(args []string) (config.Host, string, bool) {
	if len(args) == 0 || strings.EqualFold(args[0], "here") {
		ref, ok := m.selectedSessionRef()
		if !ok {
			m.status = "select a session, window, or pane to close"
			return config.Host{}, "", false
		}
		return ref.Host, ref.Session.Name, true
	}
	if len(args) != 1 {
		m.status = "usage: session close [here|host/session]"
		return config.Host{}, "", false
	}
	return m.resolveSessionCommandTarget(args[0])
}

func (m model) resolveSessionCommandTarget(value string) (config.Host, string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		m.status = "missing session name"
		return config.Host{}, "", false
	}

	if hostName, session, ok := strings.Cut(value, "/"); ok {
		host, found := findHost(m.cfg, strings.TrimSpace(hostName))
		if !found {
			m.status = "host not found: " + strings.TrimSpace(hostName)
			return config.Host{}, "", false
		}
		session = strings.TrimSpace(session)
		if session == "" {
			m.status = "missing session name"
			return config.Host{}, "", false
		}
		return host, session, true
	}

	host, ok := m.defaultSessionCommandHost()
	if !ok {
		m.status = "no hosts configured"
		return config.Host{}, "", false
	}
	return host, value, true
}

func (m model) defaultSessionCommandHost() (config.Host, bool) {
	if selected, ok := m.selectedRow(); ok && selected.hostIndex >= 0 && selected.hostIndex < len(m.hosts) {
		return m.hosts[selected.hostIndex].host, true
	}
	if len(m.hosts) > 0 {
		return m.hosts[0].host, true
	}
	return config.Host{}, false
}

func startSessionCmd(host config.Host, session string) tea.Cmd {
	return func() tea.Msg {
		client := tmuxclient.DefaultClient{}
		result := client.StartSession(context.Background(), host, session, "")
		return sessionOperationResult{
			action:  "session-open",
			host:    host,
			session: session,
			created: result.Created,
			err:     result.Err,
		}
	}
}

func closeSessionCmd(host config.Host, session string) tea.Cmd {
	return func() tea.Msg {
		client := tmuxclient.DefaultClient{}
		result := client.CloseSession(context.Background(), host, session)
		return sessionOperationResult{
			action:  "session-close",
			host:    host,
			session: session,
			closed:  result.Closed,
			err:     result.Err,
		}
	}
}
