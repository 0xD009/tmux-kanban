package main

import (
	"fmt"

	"tmux-kanban/internal/config"
	"tmux-kanban/internal/hermeslog"
	"tmux-kanban/internal/mesh"
	"tmux-kanban/internal/tmuxscan"
)

func (m *model) appendHermesWorkLog(entry hermeslog.Entry) {
	appendHermesWorkLogForConfig(m.cfg, entry)
}

func appendHermesWorkLogForConfig(cfg config.Config, entry hermeslog.Entry) {
	if cfg.Hermes.WorkLog == "" {
		return
	}
	if entry.Conditions == nil {
		entry.Conditions = map[string]string{}
	}
	entry.Conditions["hermes_enabled"] = onOff(cfg.Hermes.Enabled)
	entry.Conditions["auto_review"] = onOff(cfg.Hermes.AutoReview)
	entry.Conditions["done_advice"] = onOff(cfg.Hermes.DoneAdvice)
	entry.Conditions["auto_done"] = onOff(cfg.Hermes.AutoDone)
	entry.Conditions["idle_advice"] = onOff(cfg.Hermes.IdleAdvice)
	entry.Conditions["auto_idle"] = onOff(cfg.Hermes.AutoIdle)
	_, _ = hermeslog.Append(cfg.Hermes.WorkLog, entry)
}

func addEffectiveHermesConditions(entry *hermeslog.Entry, hermesCfg config.HermesConfig) {
	if entry.Conditions == nil {
		entry.Conditions = map[string]string{}
	}
	entry.Conditions["effective_hermes_enabled"] = onOff(hermesCfg.Enabled)
	entry.Conditions["effective_auto_review"] = onOff(hermesCfg.AutoReview)
	entry.Conditions["effective_done_advice"] = onOff(hermesCfg.DoneAdvice)
	entry.Conditions["effective_auto_done"] = onOff(hermesCfg.AutoDone)
	entry.Conditions["effective_idle_advice"] = onOff(hermesCfg.IdleAdvice)
	entry.Conditions["effective_auto_idle"] = onOff(hermesCfg.AutoIdle)
}

func reviewHermesWorkLogEntry(item reviewItem, hostLabel string, mode string, event string) hermeslog.Entry {
	host := item.HostName
	if host == "" {
		host = hostLabel
	}
	return hermeslog.Entry{
		Flow:    "review",
		Event:   event,
		Mode:    mode,
		Trigger: "need-review",
		Status:  string(sessionNeedReview),
		Host:    host,
		Session: item.SessionName,
		Target:  item.Row.attachTarget,
		Agent:   string(item.Agent),
	}
}

func nextStepHermesWorkLogEntry(msg hermesNextStepResult, event string) hermeslog.Entry {
	return hermeslog.Entry{
		Flow:    "next-step",
		Event:   event,
		Mode:    autoModeLabel(msg.auto),
		Trigger: "status-transition",
		Status:  string(normalizeSessionStatus(msg.status)),
		Host:    msg.hostName,
		Session: msg.sessionName,
		Target:  msg.target.target,
		Agent:   string(msg.target.agent),
		Conditions: map[string]string{
			"auto_requested": fmt.Sprintf("%v", msg.auto),
		},
	}
}

func memoryHermesWorkLogEntry(scope mesh.Scope, event string) hermeslog.Entry {
	return hermeslog.Entry{
		Flow:    "memory",
		Event:   event,
		Mode:    "manual",
		Trigger: "memory-update-command",
		Scope:   scope.Key(),
	}
}

func autoModeLabel(auto bool) string {
	if auto {
		return "auto"
	}
	return "manual"
}

func agentKindLabel(agent tmuxscan.AgentKind) string {
	if agent == tmuxscan.AgentNone {
		return ""
	}
	return string(agent)
}
