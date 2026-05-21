package main

import (
	"fmt"
	"strings"
)

func (m *model) executeSetCommand(args []string) {
	if len(args) < 2 {
		m.status = "usage: set qq|hermes|hermes.auto_review|status <on|off|value>"
		return
	}

	key := strings.ToLower(args[0])
	valueArgs := args[1:]
	switch key {
	case "qq", "qq_enabled", "notification.qq", "notification.qq_enabled":
		m.executeBoolSettingCommand("qq", valueArgs, func(value bool) {
			m.cfg.Notification.QQEnabled = value
			m.status = "QQ notification " + onOff(value)
		})
	case "hermes", "hermes.enabled":
		m.executeBoolSettingCommand("hermes", valueArgs, func(value bool) {
			m.cfg.Hermes.Enabled = value
			m.status = "Hermes " + onOff(value)
		})
	case "hermes.auto", "hermes.auto_review", "hermes.review_auto", "auto_review":
		m.executeBoolSettingCommand("hermes.auto_review", valueArgs, func(value bool) {
			m.cfg.Hermes.AutoReview = value
			m.status = "Hermes auto review " + onOff(value)
		})
	case "mesh", "agent_mesh", "agent_mesh.enabled", "mesh.enabled":
		m.executeBoolSettingCommand("mesh", valueArgs, func(value bool) {
			m.cfg.AgentMesh.Enabled = value
			m.status = "agent mesh " + onOff(value)
		})
	case "mesh.shared", "agent_mesh.shared", "agent_mesh.shared_short_agent", "mesh.shared_short_agent":
		m.executeBoolSettingCommand("mesh.shared", valueArgs, func(value bool) {
			m.cfg.AgentMesh.SharedShortAgent = value
			m.status = "agent mesh shared_short_agent " + onOff(value)
		})
	case "mesh.default", "mesh.default_agent", "agent_mesh.default_agent":
		if len(valueArgs) != 1 {
			m.status = "usage: set mesh.default_agent codex|claude-code"
			return
		}
		m.cfg.AgentMesh.DefaultAgent = normalizeConfigAgent(valueArgs[0])
		m.ensureMeshPolicyDefaults()
		m.status = "agent mesh default_agent " + m.cfg.AgentMesh.DefaultAgent
	case "mesh.memory", "mesh.memory_root", "agent_mesh.memory_root":
		m.cfg.AgentMesh.MemoryRoot = configValueOrEmpty(valueArgs)
		m.status = "agent mesh memory_root " + emptyValueLabel(m.cfg.AgentMesh.MemoryRoot)
	case "mesh.skill_root", "agent_mesh.skill_root", "mesh.skills":
		m.cfg.AgentMesh.SkillRoot = configValueOrEmpty(valueArgs)
		m.status = "agent mesh skill_root " + emptyValueLabel(m.cfg.AgentMesh.SkillRoot)
	case "mesh.mail", "agent_mesh.mail", "agent_mesh.mail.enabled", "mesh.mail.enabled":
		m.executeBoolSettingCommand("mesh.mail", valueArgs, func(value bool) {
			m.cfg.AgentMesh.Mail.Enabled = value
			m.status = "agent mesh mail " + onOff(value)
		})
	case "mesh.mail_dir", "agent_mesh.mail.dir", "mesh.mail.dir":
		m.cfg.AgentMesh.Mail.Dir = configValueOrEmpty(valueArgs)
		m.status = "agent mesh mail dir " + emptyValueLabel(m.cfg.AgentMesh.Mail.Dir)
	case "main", "main_agent", "main_agent.enabled", "main.enabled":
		m.executeBoolSettingCommand("main", valueArgs, func(value bool) {
			m.cfg.MainAgent.Enabled = value
			if !value {
				m.mainActive = false
				m.preview = previewState{}
			}
			m.status = "main agent " + onOff(value)
		})
	case "main.agent", "main_agent.agent":
		if len(valueArgs) != 1 {
			m.status = "usage: set main.agent codex|claude-code"
			return
		}
		m.setMainAgent(valueArgs[0])
		m.status = "main agent set to " + m.cfg.MainAgent.Agent
	case "main.host", "main_agent.host":
		m.cfg.MainAgent.Host = configValueOrEmpty(valueArgs)
		m.preview = previewState{}
		m.status = "main host set to " + emptyValueLabel(m.cfg.MainAgent.Host)
	case "main.session", "main_agent.session":
		m.cfg.MainAgent.Session = configValueOrEmpty(valueArgs)
		m.preview = previewState{}
		m.status = "main session set to " + emptyValueLabel(m.cfg.MainAgent.Session)
	case "main.command", "main_agent.command":
		if len(valueArgs) == 0 {
			m.status = "usage: set main.command <command> [args...]"
			return
		}
		m.cfg.MainAgent.Command = valueArgs[0]
		m.cfg.MainAgent.Args = append([]string(nil), valueArgs[1:]...)
		m.status = "main command set to " + strings.Join(valueArgs, " ")
	case "status":
		m.executeStatusCommand(valueArgs)
	case "view":
		m.executeViewCommand(valueArgs)
	default:
		m.status = "unknown setting: " + args[0]
	}
}

func (m *model) executeBoolSettingCommand(name string, args []string, apply func(bool)) {
	if len(args) != 1 {
		m.status = "usage: " + name + " on|off"
		return
	}
	value, ok := parseOnOff(args[0])
	if !ok {
		m.status = "usage: " + name + " on|off"
		return
	}
	apply(value)
}

func (m *model) executeStatusCommand(args []string) {
	if len(args) == 0 {
		m.status = "usage: status idle|working|need-review|done"
		return
	}
	status, ok := parseSessionStatusValue(strings.Join(args, " "))
	if !ok {
		m.status = "usage: status idle|working|need-review|done"
		return
	}
	m.setActiveSessionStatus(status)
}

func (m *model) setViewMode(mode viewMode) {
	switch mode {
	case viewReview:
		m.mainActive = false
		if m.compose.active {
			m.compose = composeState{}
		}
		m.viewMode = viewReview
		m.clampReviewCursor()
		m.status = "review queue"
	case viewTree:
		m.mainActive = false
		if m.compose.active {
			m.compose = composeState{}
		}
		m.viewMode = viewTree
		m.status = "tree view"
	case viewMain:
		m.mainActive = true
		m.viewMode = viewMain
		m.status = "main room"
	default:
		return
	}
	m.preview = previewState{}
}

func (m *model) setActiveSessionStatus(status sessionStatus) {
	if m.statuses == nil {
		m.statuses = map[string]sessionStatus{}
	}

	if m.viewMode == viewReview {
		item, ok := m.currentReviewItem()
		if !ok {
			m.status = "review queue is empty"
			return
		}
		m.statuses[item.SessionKey] = status
		delete(m.statusStreaks, item.SessionKey)
		m.clearHermesAdvice(item.SessionKey)
		if status != sessionNeedReview {
			delete(m.reviewSkipped, item.SessionKey)
			delete(m.reviewTargets, item.SessionKey)
			m.advanceReviewCursorAfter(item.SessionKey)
		} else {
			m.clampReviewCursor()
		}
		m.preview = previewState{}
		m.status = fmt.Sprintf("%s/%s -> %s", item.HostName, item.SessionName, statusLabel(status))
		m.addAgentActivity(agentActivity{
			Source:  agentActivitySession,
			Agent:   "session",
			Target:  item.HostName + "/" + item.SessionName,
			State:   statusLabel(status),
			Message: "manual status set",
		})
		return
	}

	ref, ok := m.selectedSessionRef()
	if !ok {
		m.status = "select a session, window, or pane to set status"
		return
	}
	m.statuses[ref.Key] = status
	delete(m.statusStreaks, ref.Key)
	m.clearHermesAdvice(ref.Key)
	if status != sessionNeedReview {
		delete(m.reviewTargets, ref.Key)
	}
	m.status = fmt.Sprintf("%s/%s -> %s", ref.Host.Name, ref.Session.Name, statusLabel(status))
	m.addAgentActivity(agentActivity{
		Source:  agentActivitySession,
		Agent:   "session",
		Target:  displayHostName(ref.Host) + "/" + ref.Session.Name,
		State:   statusLabel(status),
		Message: "manual status set",
	})
}
