package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"tmux-kanban/internal/config"
)

func (m *model) executeSetCommand(args []string) tea.Cmd {
	if len(args) < 2 {
		m.status = "usage: set qq|auto_review_audit_qq|terminal_bell|terminal_review|hermes|hermes.auto_review|hermes.done_advice|hermes.auto_done|hermes.idle_advice|hermes.auto_idle|status <on|off|value>"
		return nil
	}

	key := strings.ToLower(args[0])
	valueArgs := args[1:]
	switch key {
	case "qq", "qq_enabled", "notification.qq", "notification.qq_enabled":
		m.executeBoolSettingCommand("qq", valueArgs, func(value bool) {
			m.cfg.Notification.QQEnabled = value
			m.status = "QQ notification " + onOff(value)
		})
	case "auto_review_audit_qq", "audit_qq", "notification.auto_review_audit_qq":
		m.executeAutoReviewAuditQQSettingCommand(valueArgs)
	case "terminal_bell", "bell", "notification.terminal_bell":
		m.executeBoolSettingCommand("terminal_bell", valueArgs, func(value bool) {
			m.cfg.Notification.TerminalBell = value
			m.status = "terminal bell notification " + onOff(value)
		})
	case "terminal_review", "review_terminal", "notification.terminal_review":
		m.executeBoolSettingCommand("terminal_review", valueArgs, func(value bool) {
			m.cfg.Notification.TerminalReview = value
			m.status = "terminal review notification " + onOff(value)
		})
		return m.syncReviewTerminalTitleCmd()
	case "hermes", "hermes.enabled":
		m.setHermesScopedBool(valueArgs, func(value bool) {
			m.cfg.Hermes.Enabled = value
			m.status = "Hermes " + onOff(value)
		}, func(scope *config.HermesScopeConfig, value bool) {
			scope.Enabled = boolSettingPtr(value)
		}, "hermes")
	case "hermes.auto", "hermes.auto_review", "hermes.review_auto", "auto_review":
		m.setHermesScopedBool(valueArgs, func(value bool) {
			m.cfg.Hermes.AutoReview = value
			m.status = "Hermes auto review " + onOff(value)
		}, func(scope *config.HermesScopeConfig, value bool) {
			scope.AutoReview = boolSettingPtr(value)
		}, "hermes.auto_review")
	case "hermes.done_advice", "hermes.done", "hermes.next_done", "done_advice":
		m.setHermesScopedBool(valueArgs, func(value bool) {
			m.cfg.Hermes.DoneAdvice = value
			m.status = "Hermes done advice " + onOff(value)
		}, func(scope *config.HermesScopeConfig, value bool) {
			scope.DoneAdvice = boolSettingPtr(value)
		}, "hermes.done_advice")
	case "hermes.auto_done", "hermes.done_auto", "auto_done":
		m.setHermesScopedBool(valueArgs, func(value bool) {
			m.cfg.Hermes.AutoDone = value
			m.status = "Hermes auto done " + onOff(value)
		}, func(scope *config.HermesScopeConfig, value bool) {
			scope.AutoDone = boolSettingPtr(value)
		}, "hermes.auto_done")
	case "hermes.idle_advice", "hermes.idle", "hermes.next_idle", "idle_advice":
		m.setHermesScopedBool(valueArgs, func(value bool) {
			m.cfg.Hermes.IdleAdvice = value
			m.status = "Hermes idle advice " + onOff(value)
		}, func(scope *config.HermesScopeConfig, value bool) {
			scope.IdleAdvice = boolSettingPtr(value)
		}, "hermes.idle_advice")
	case "hermes.auto_idle", "hermes.idle_auto", "auto_idle":
		m.setHermesScopedBool(valueArgs, func(value bool) {
			m.cfg.Hermes.AutoIdle = value
			m.status = "Hermes auto idle " + onOff(value)
		}, func(scope *config.HermesScopeConfig, value bool) {
			scope.AutoIdle = boolSettingPtr(value)
		}, "hermes.auto_idle")
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
			return nil
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
	case "status":
		return m.executeStatusCommand(valueArgs)
	case "view":
		next, cmd := m.executeViewCommand(valueArgs)
		*m = next
		return cmd
	default:
		m.status = "unknown setting: " + args[0]
	}
	return nil
}

func (m *model) executeAutoReviewAuditQQSettingCommand(args []string) {
	if len(args) != 1 {
		m.status = "usage: auto_review_audit_qq off|uncertain|all"
		return
	}
	value, ok := config.NormalizeAutoReviewAuditQQMode(args[0])
	if !ok {
		m.status = "usage: auto_review_audit_qq off|uncertain|all"
		return
	}
	m.cfg.Notification.AutoReviewAuditQQ = value
	m.status = "auto review audit QQ " + value.String()
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

func (m *model) executeStatusCommand(args []string) tea.Cmd {
	if len(args) == 0 {
		m.status = "usage: status idle|working|need-review|done"
		return nil
	}
	status, ok := parseSessionStatusValue(strings.Join(args, " "))
	if !ok {
		m.status = "usage: status idle|working|need-review|done"
		return nil
	}
	return m.setActiveSessionStatus(status)
}

func (m *model) setViewMode(mode viewMode) {
	switch mode {
	case viewReview:
		if m.compose.active {
			m.compose = composeState{}
		}
		m.viewMode = viewReview
		m.focusedPanel = panelReviewQueue
		m.clampReviewCursor()
		m.status = "review queue"
	case viewTree:
		if m.compose.active {
			m.compose = composeState{}
		}
		m.viewMode = viewTree
		m.focusedPanel = panelExplorer
		m.status = "tree view"
	default:
		return
	}
	m.preview = previewState{}
	m.resetPreviewScroll()
}

func (m *model) setActiveSessionStatus(status sessionStatus) tea.Cmd {
	if m.statuses == nil {
		m.statuses = map[string]sessionStatus{}
	}

	if m.viewMode == viewReview {
		item, ok := m.currentReviewItem()
		if !ok {
			m.status = "review queue is empty"
			return nil
		}
		oldStatus, hadOldStatus := m.statuses[item.SessionKey]
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
		return tea.Batch(m.autoHermesNextStepCmd(hadOldStatus, oldStatus, status, item.SessionKey), m.syncReviewTerminalTitleCmd())
	}

	ref, ok := m.selectedSessionRef()
	if !ok {
		m.status = "select a session, window, or pane to set status"
		return nil
	}
	oldStatus, hadOldStatus := m.statuses[ref.Key]
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
	return tea.Batch(m.autoHermesNextStepCmd(hadOldStatus, oldStatus, status, ref.Key), m.syncReviewTerminalTitleCmd())
}
