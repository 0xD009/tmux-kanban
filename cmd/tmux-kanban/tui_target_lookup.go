package main

import (
	"fmt"
	"strings"

	"tmux-kanban/internal/config"
	"tmux-kanban/internal/tmuxscan"
)

func sameHostIdentity(left config.Host, right config.Host) bool {
	return displayHostName(left) == displayHostName(right) && left.SSH == right.SSH && left.Local == right.Local
}

func sessionAt(sessions []tmuxscan.Session, index int) (tmuxscan.Session, bool) {
	if index < 0 || index >= len(sessions) {
		return tmuxscan.Session{}, false
	}
	return sessions[index], true
}

func windowAt(windows []tmuxscan.Window, index int) (tmuxscan.Window, bool) {
	if index < 0 || index >= len(windows) {
		return tmuxscan.Window{}, false
	}
	return windows[index], true
}

func paneAt(panes []tmuxscan.Pane, index int) (tmuxscan.Pane, bool) {
	if index < 0 || index >= len(panes) {
		return tmuxscan.Pane{}, false
	}
	return panes[index], true
}

func firstAgentTargetInSession(hostIndex int, baseKey string, session tmuxscan.Session) (selectedAgentTarget, bool) {
	targets := agentTargetsInSession(hostIndex, baseKey, session)
	if len(targets) == 0 {
		return selectedAgentTarget{}, false
	}
	return targets[0], true
}

func activeAgentTargetInSession(hostIndex int, baseKey string, session tmuxscan.Session) (selectedAgentTarget, bool) {
	for _, window := range session.Windows {
		if !window.Active {
			continue
		}
		windowKey := baseKey + ":window:" + window.ID
		if target, ok := activeAgentTargetInWindow(hostIndex, windowKey, session.Name, window); ok {
			return target, true
		}
		break
	}
	return firstAgentTargetInSession(hostIndex, baseKey, session)
}

func agentTargetsInSession(hostIndex int, baseKey string, session tmuxscan.Session) []selectedAgentTarget {
	targets := make([]selectedAgentTarget, 0)
	for _, window := range session.Windows {
		windowKey := baseKey + ":window:" + window.ID
		targets = append(targets, agentTargetsInWindow(hostIndex, windowKey, session.Name, window)...)
	}
	return targets
}

func firstAgentTargetInWindow(hostIndex int, baseKey string, sessionName string, window tmuxscan.Window) (selectedAgentTarget, bool) {
	targets := agentTargetsInWindow(hostIndex, baseKey, sessionName, window)
	if len(targets) == 0 {
		return selectedAgentTarget{}, false
	}
	return targets[0], true
}

func activeAgentTargetInWindow(hostIndex int, baseKey string, sessionName string, window tmuxscan.Window) (selectedAgentTarget, bool) {
	for _, pane := range window.Panes {
		if !pane.Active || pane.Agent == tmuxscan.AgentNone {
			continue
		}
		return selectedAgentTargetForPane(hostIndex, baseKey, sessionName, window, pane), true
	}
	return firstAgentTargetInWindow(hostIndex, baseKey, sessionName, window)
}

func agentTargetsInWindow(hostIndex int, baseKey string, sessionName string, window tmuxscan.Window) []selectedAgentTarget {
	targets := make([]selectedAgentTarget, 0)
	for _, pane := range window.Panes {
		if pane.Agent == tmuxscan.AgentNone {
			continue
		}

		targets = append(targets, selectedAgentTargetForPane(hostIndex, baseKey, sessionName, window, pane))
	}
	return targets
}

func selectedAgentTargetForPane(hostIndex int, baseKey string, sessionName string, window tmuxscan.Window, pane tmuxscan.Pane) selectedAgentTarget {
	return selectedAgentTarget{
		key:       fmt.Sprintf("%s:pane:%s", baseKey, pane.ID),
		hostIndex: hostIndex,
		target:    tmuxPaneTarget(sessionName, window, pane),
		label:     agentPaneDisplayLabel(sessionName, window, pane),
		agent:     pane.Agent,
	}
}

func agentPaneDisplayLabel(sessionName string, window tmuxscan.Window, pane tmuxscan.Pane) string {
	label := sessionName + ":" + window.Index + "." + pane.Index
	if pane.Agent != tmuxscan.AgentNone {
		label += " (" + string(pane.Agent) + ")"
	}
	return label
}

func fallbackAgentTargetLabel(target selectedAgentTarget) string {
	if target.label != "" {
		return target.label
	}
	if target.agent != tmuxscan.AgentNone {
		return "selected " + string(target.agent) + " pane"
	}
	return "selected pane"
}

func tmuxPaneTarget(sessionName string, window tmuxscan.Window, pane tmuxscan.Pane) string {
	if strings.TrimSpace(pane.ID) != "" {
		return pane.ID
	}
	return sessionName + ":" + window.Index + "." + pane.Index
}

func targetExistsInSession(target string, session tmuxscan.Session) bool {
	for _, window := range session.Windows {
		if targetExistsInWindow(target, session.Name, window) {
			return true
		}
	}
	return false
}

func targetExistsInWindow(target string, sessionName string, window tmuxscan.Window) bool {
	for _, pane := range window.Panes {
		if tmuxPaneTarget(sessionName, window, pane) == target {
			return true
		}
	}
	return false
}
