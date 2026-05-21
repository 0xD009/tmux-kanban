package main

import (
	"fmt"
	"strings"

	"tmux-kanban/internal/config"
	"tmux-kanban/internal/tmuxscan"
)

func (m model) rows() []row {
	agentRows := make([]row, 0)
	otherRows := make([]row, 0)

	for hostIndex, state := range m.hosts {
		if state.loading && !state.loaded {
			continue
		}
		if !state.loaded {
			otherRows = append(otherRows, row{
				key:       hostKey(hostIndex) + ":empty",
				kind:      rowPane,
				hostIndex: hostIndex,
				label:     fmt.Sprintf("  %s  not scanned yet", displayHostName(state.host)),
			})
			continue
		}
		if state.snapshot.Err != "" {
			otherRows = append(otherRows, row{
				key:       hostKey(hostIndex) + ":error",
				kind:      rowPane,
				hostIndex: hostIndex,
				label:     fmt.Sprintf("  %s  error: %s", displayHostName(state.host), state.snapshot.Err),
			})
			continue
		}
		for sessionIndex, session := range state.snapshot.Sessions {
			if m.isMainSession(hostIndex, session) {
				continue
			}

			sessionRows := m.sessionRows(hostIndex, sessionIndex, session)
			if sessionHasAgent(session) {
				agentRows = append(agentRows, sessionRows...)
			} else {
				otherRows = append(otherRows, sessionRows...)
			}
		}
	}

	rows := make([]row, 0, len(agentRows)+len(otherRows))
	rows = append(rows, agentRows...)
	rows = append(rows, otherRows...)
	return rows
}

func (m model) sessionRows(hostIndex int, sessionIndex int, session tmuxscan.Session) []row {
	sessionKey := sessionKey(hostIndex, session.ID)
	rows := []row{{
		key:          sessionKey,
		kind:         rowSession,
		hostIndex:    hostIndex,
		sessionIndex: sessionIndex,
		label:        m.sessionLabel(hostIndex, session),
		attachTarget: session.Name,
	}}

	if !m.expanded[sessionKey] {
		return rows
	}

	for windowIndex, window := range session.Windows {
		windowKey := windowKey(hostIndex, session.ID, window.ID)
		rows = append(rows, row{
			key:          windowKey,
			kind:         rowWindow,
			hostIndex:    hostIndex,
			sessionIndex: sessionIndex,
			windowIndex:  windowIndex,
			label:        m.windowLabel(hostIndex, session.ID, window),
			attachTarget: session.Name + ":" + window.Index,
		})

		if !m.expanded[windowKey] {
			continue
		}

		for paneIndex, pane := range window.Panes {
			rows = append(rows, row{
				key:          fmt.Sprintf("%s:pane:%s", windowKey, pane.ID),
				kind:         rowPane,
				hostIndex:    hostIndex,
				sessionIndex: sessionIndex,
				windowIndex:  windowIndex,
				paneIndex:    paneIndex,
				label:        paneLabel(pane),
				attachTarget: tmuxPaneTarget(session.Name, window, pane),
				agent:        pane.Agent,
			})
		}
	}

	return rows
}

type visibleSession struct {
	index   int
	session tmuxscan.Session
}

func (m model) visibleHostSessions(hostIndex int, sessions []tmuxscan.Session) []visibleSession {
	visible := make([]visibleSession, 0, len(sessions))
	for sessionIndex, session := range sessions {
		if m.isMainSession(hostIndex, session) {
			continue
		}
		visible = append(visible, visibleSession{index: sessionIndex, session: session})
	}
	return visible
}

func sessionHasAgent(session tmuxscan.Session) bool {
	for _, window := range session.Windows {
		for _, pane := range window.Panes {
			if pane.Agent != tmuxscan.AgentNone {
				return true
			}
		}
	}
	return false
}

func (m model) visibleHostSessionCounts(hostIndex int, sessions []tmuxscan.Session) (int, int) {
	agentSessions := 0
	otherSessions := 0
	for _, session := range sessions {
		if m.isMainSession(hostIndex, session) {
			continue
		}
		if sessionHasAgent(session) {
			agentSessions++
		} else {
			otherSessions++
		}
	}
	return agentSessions, otherSessions
}

func (m model) hostLabel(index int, state hostState) string {
	prefix := "[+]"
	if m.expanded[hostKey(index)] {
		prefix = "[-]"
	}

	name := displayHostName(state.host)
	suffix := hostTargetLabel(state.host)
	switch {
	case state.loading && !state.loaded:
		suffix += " | scanning"
	case state.loaded && state.snapshot.Err != "":
		suffix += " | error"
	case state.loaded:
		agentSessions, otherSessions := m.visibleHostSessionCounts(index, state.snapshot.Sessions)
		suffix += fmt.Sprintf(" | %d sessions", agentSessions+otherSessions)
		if agentSessions > 0 || otherSessions > 0 {
			suffix += fmt.Sprintf(" | %d agent / %d other", agentSessions, otherSessions)
		}
	default:
		suffix += " | idle"
	}

	return fmt.Sprintf("%s %s  %s", prefix, name, suffix)
}

func hostTargetLabel(host config.Host) string {
	if host.Local {
		return "local"
	}
	if strings.TrimSpace(host.SSH) == "" {
		return "missing ssh"
	}
	return host.SSH
}

func (m model) sessionLabel(hostIndex int, session tmuxscan.Session) string {
	prefix := "  [+]"
	if m.expanded[sessionKey(hostIndex, session.ID)] {
		prefix = "  [-]"
	}

	attached := ""
	if session.Attached > 0 {
		attached = fmt.Sprintf(", attached %d", session.Attached)
	}
	badge := ""
	if status, ok := m.sessionStatusForSession(m.hosts[hostIndex].host, session); ok {
		badge = statusBadge(status)
	}
	hostName := displayHostName(m.hosts[hostIndex].host)
	return fmt.Sprintf("%s %s/%s%s%s  %d windows%s", prefix, hostName, session.Name, badge, sessionAgentBadge(session), len(session.Windows), attached)
}

func (m model) windowLabel(hostIndex int, sessionID string, window tmuxscan.Window) string {
	prefix := "    [+]"
	if m.expanded[windowKey(hostIndex, sessionID, window.ID)] {
		prefix = "    [-]"
	}

	active := ""
	if window.Active {
		active = " active"
	}
	return fmt.Sprintf("%s %s:%s%s%s  %d panes", prefix, window.Index, window.Name, active, windowAgentBadge(window), len(window.Panes))
}

func paneLabel(pane tmuxscan.Pane) string {
	active := ""
	if pane.Active {
		active = " active"
	}

	path := pane.CurrentPath
	if path == "" {
		path = "unknown path"
	}
	return fmt.Sprintf("      %s %s%s%s  %s", pane.Index, pane.Command, active, agentBadge(pane.Agent), path)
}

func sessionAgentBadge(session tmuxscan.Session) string {
	agents := map[tmuxscan.AgentKind]bool{}
	for _, window := range session.Windows {
		collectWindowAgents(window, agents)
	}
	return agentSetBadge(agents)
}

func windowAgentBadge(window tmuxscan.Window) string {
	agents := map[tmuxscan.AgentKind]bool{}
	collectWindowAgents(window, agents)
	return agentSetBadge(agents)
}

func collectWindowAgents(window tmuxscan.Window, agents map[tmuxscan.AgentKind]bool) {
	for _, pane := range window.Panes {
		if pane.Agent != tmuxscan.AgentNone {
			agents[pane.Agent] = true
		}
	}
}

func agentSetBadge(agents map[tmuxscan.AgentKind]bool) string {
	labels := make([]string, 0, 2)
	if agents[tmuxscan.AgentCodex] {
		labels = append(labels, string(tmuxscan.AgentCodex))
	}
	if agents[tmuxscan.AgentClaude] {
		labels = append(labels, string(tmuxscan.AgentClaude))
	}
	if len(labels) == 0 {
		return ""
	}
	return " [" + strings.Join(labels, ", ") + "]"
}

func agentBadge(agent tmuxscan.AgentKind) string {
	if agent == tmuxscan.AgentNone {
		return ""
	}
	return " [" + string(agent) + "]"
}
