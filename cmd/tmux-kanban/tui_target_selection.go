package main

import (
	"fmt"

	"tmux-kanban/internal/tmuxscan"
)

type selectedAgentTarget struct {
	key       string
	hostIndex int
	target    string
	label     string
	agent     tmuxscan.AgentKind
}

func (m model) selectedAgentTarget() (selectedAgentTarget, bool) {
	selected, ok := m.selectedRow()
	if !ok || selected.hostIndex < 0 || selected.hostIndex >= len(m.hosts) {
		return selectedAgentTarget{}, false
	}
	sessions := m.hosts[selected.hostIndex].snapshot.Sessions
	switch selected.kind {
	case rowPane:
		if selected.agent == tmuxscan.AgentNone || selected.attachTarget == "" {
			return selectedAgentTarget{}, false
		}
		session, ok := sessionAt(sessions, selected.sessionIndex)
		if !ok {
			return selectedAgentTarget{}, false
		}
		window, ok := windowAt(session.Windows, selected.windowIndex)
		if !ok {
			return selectedAgentTarget{}, false
		}
		pane, ok := paneAt(window.Panes, selected.paneIndex)
		if !ok {
			return selectedAgentTarget{}, false
		}
		return selectedAgentTargetForPane(selected.hostIndex, windowKey(selected.hostIndex, session.ID, window.ID), session.Name, window, pane), true
	case rowWindow:
		session, ok := sessionAt(sessions, selected.sessionIndex)
		if !ok {
			return selectedAgentTarget{}, false
		}
		window, ok := windowAt(session.Windows, selected.windowIndex)
		if !ok {
			return selectedAgentTarget{}, false
		}
		return activeAgentTargetInWindow(selected.hostIndex, selected.key, session.Name, window)
	case rowSession:
		session, ok := sessionAt(sessions, selected.sessionIndex)
		if !ok {
			return selectedAgentTarget{}, false
		}
		return activeAgentTargetInSession(selected.hostIndex, selected.key, session)
	case rowHost:
		for _, session := range sessions {
			if target, ok := firstAgentTargetInSession(selected.hostIndex, selected.key+":session:"+session.ID, session); ok {
				return target, true
			}
		}
	}

	return selectedAgentTarget{}, false
}

func (m model) selectedReviewAgentTarget() (selectedAgentTarget, selectedSessionRef, bool) {
	selected, ok := m.selectedRow()
	if !ok || selected.hostIndex < 0 || selected.hostIndex >= len(m.hosts) {
		return selectedAgentTarget{}, selectedSessionRef{}, false
	}
	switch selected.kind {
	case rowSession, rowWindow:
	default:
		return selectedAgentTarget{}, selectedSessionRef{}, false
	}

	sessions := m.hosts[selected.hostIndex].snapshot.Sessions
	session, ok := sessionAt(sessions, selected.sessionIndex)
	if !ok {
		return selectedAgentTarget{}, selectedSessionRef{}, false
	}
	host := m.hosts[selected.hostIndex].host
	key := sessionStatusKey(host, session)
	if m.sessionStatusForKey(key) != sessionNeedReview {
		return selectedAgentTarget{}, selectedSessionRef{}, false
	}

	target, ok := m.reviewTargetForSession(key, selected.hostIndex, session)
	if !ok {
		return selectedAgentTarget{}, selectedSessionRef{}, false
	}
	if selected.kind == rowWindow {
		window, ok := windowAt(session.Windows, selected.windowIndex)
		if !ok || !targetExistsInWindow(target.target, session.Name, window) {
			return selectedAgentTarget{}, selectedSessionRef{}, false
		}
	}

	ref := selectedSessionRef{
		Key:          key,
		HostIndex:    selected.hostIndex,
		SessionIndex: selected.sessionIndex,
		Host:         host,
		Session:      session,
	}
	return target, ref, true
}

func (m model) rowForAgentTarget(target selectedAgentTarget) (row, bool) {
	if target.hostIndex < 0 || target.hostIndex >= len(m.hosts) {
		return row{}, false
	}
	sessions := m.hosts[target.hostIndex].snapshot.Sessions
	for sessionIndex, session := range sessions {
		for windowIndex, window := range session.Windows {
			for paneIndex, pane := range window.Panes {
				if tmuxPaneTarget(session.Name, window, pane) != target.target {
					continue
				}
				key := target.key
				if key == "" {
					key = fmt.Sprintf("%s:pane:%s", windowKey(target.hostIndex, session.ID, window.ID), pane.ID)
				}
				return row{
					key:          key,
					kind:         rowPane,
					hostIndex:    target.hostIndex,
					sessionIndex: sessionIndex,
					windowIndex:  windowIndex,
					paneIndex:    paneIndex,
					label:        paneLabel(pane),
					attachTarget: target.target,
					agent:        pane.Agent,
				}, true
			}
		}
	}
	return row{}, false
}

func (m model) sessionRefForAgentTarget(target selectedAgentTarget) (selectedSessionRef, bool) {
	if target.hostIndex < 0 || target.hostIndex >= len(m.hosts) {
		return selectedSessionRef{}, false
	}
	host := m.hosts[target.hostIndex].host
	sessions := m.hosts[target.hostIndex].snapshot.Sessions
	for sessionIndex, session := range sessions {
		for _, window := range session.Windows {
			for _, pane := range window.Panes {
				if tmuxPaneTarget(session.Name, window, pane) != target.target {
					continue
				}
				return selectedSessionRef{
					Key:          sessionStatusKey(host, session),
					HostIndex:    target.hostIndex,
					SessionIndex: sessionIndex,
					Host:         host,
					Session:      session,
				}, true
			}
		}
	}
	return selectedSessionRef{}, false
}
