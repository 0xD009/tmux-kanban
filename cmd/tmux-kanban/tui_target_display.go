package main

import (
	"strings"
	"time"

	"tmux-kanban/internal/tmuxscan"
)

func (m model) agentTargetDisplayLabel(target selectedAgentTarget) string {
	if target.hostIndex < 0 || target.hostIndex >= len(m.hosts) {
		return fallbackAgentTargetLabel(target)
	}

	hostName := m.hosts[target.hostIndex].host.Name
	sessions := m.hosts[target.hostIndex].snapshot.Sessions
	for _, session := range sessions {
		for _, window := range session.Windows {
			for _, pane := range window.Panes {
				if tmuxPaneTarget(session.Name, window, pane) != target.target {
					continue
				}
				label := session.Name + ":" + window.Index + "." + pane.Index
				if hostName != "" {
					label = hostName + "/" + label
				}
				if target.agent != tmuxscan.AgentNone {
					label += " (" + string(target.agent) + ")"
				}
				return label
			}
		}
	}

	if target.label != "" {
		if hostName != "" {
			return hostName + "/" + target.label
		}
		return target.label
	}
	return fallbackAgentTargetLabel(target)
}

func (m *model) addAgentActivity(activity agentActivity) {
	if activity.At.IsZero() {
		activity.At = time.Now()
	}
	if strings.TrimSpace(activity.Target) == "" {
		activity.Target = "unknown target"
	}
	m.activities = append(m.activities, activity)
	if len(m.activities) > maxAgentActivities {
		m.activities = append([]agentActivity(nil), m.activities[len(m.activities)-maxAgentActivities:]...)
	}
}

func (m model) shouldLogPolledStatusChange(hadOld bool, oldStatus sessionStatus, nextStatus sessionStatus) bool {
	next := normalizeSessionStatus(nextStatus)
	if !hadOld {
		return next != sessionIdle
	}
	return normalizeSessionStatus(oldStatus) != next
}

func (m model) reviewItemLabelByKey(key string) string {
	for _, item := range m.reviewItems() {
		if item.SessionKey != key {
			continue
		}
		label := item.HostName + "/" + item.SessionName
		if item.Agent != tmuxscan.AgentNone {
			label += " (" + string(item.Agent) + ")"
		}
		return label
	}
	if strings.TrimSpace(key) != "" {
		return key
	}
	return "review item"
}

func sendResultTargetLabel(result tmuxscan.SendResult) string {
	host := displayHostName(result.Host)
	target := strings.TrimSpace(result.Target)
	if target == "" {
		return host
	}
	return host + "/" + target
}

func (m model) sendResultDisplayLabel(result tmuxscan.SendResult) string {
	target := strings.TrimSpace(result.Target)
	if target == "" {
		return displayHostName(result.Host)
	}

	for hostIndex, hostState := range m.hosts {
		if !sameHostIdentity(hostState.host, result.Host) {
			continue
		}
		hostName := displayHostName(hostState.host)
		for _, session := range hostState.snapshot.Sessions {
			if target == session.Name {
				return hostName + "/" + session.Name
			}
			for _, window := range session.Windows {
				if target == session.Name+":"+window.Index {
					return hostName + "/" + session.Name + ":" + window.Index
				}
				for _, pane := range window.Panes {
					if tmuxPaneTarget(session.Name, window, pane) != target {
						continue
					}
					return m.agentTargetDisplayLabel(selectedAgentTarget{
						hostIndex: hostIndex,
						target:    target,
						label:     agentPaneDisplayLabel(session.Name, window, pane),
						agent:     pane.Agent,
					})
				}
			}
		}
	}

	return sendResultTargetLabel(result)
}
