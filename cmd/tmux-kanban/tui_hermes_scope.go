package main

import (
	"strings"

	"tmux-kanban/internal/config"
)

type hermesScopeBoolSetter func(*config.HermesScopeConfig, bool)

func (m model) scopedHermesConfig(host config.Host, session string) config.HermesConfig {
	return m.cfg.Hermes.Resolve(host, session)
}

func configWithHermes(cfg config.Config, hermes config.HermesConfig) config.Config {
	cfg.Hermes = hermes
	return cfg
}

func (m model) hermesConfigForReviewItem(item reviewItem) (config.HermesConfig, bool) {
	if item.Row.hostIndex < 0 || item.Row.hostIndex >= len(m.hosts) {
		return config.HermesConfig{}, false
	}
	return m.scopedHermesConfig(m.hosts[item.Row.hostIndex].host, item.SessionName), true
}

func (m *model) setHermesScopedBool(args []string, global func(bool), scoped hermesScopeBoolSetter, settingName string) bool {
	if len(args) == 1 {
		value, ok := parseOnOff(args[0])
		if !ok {
			m.status = hermesScopeUsage(settingName)
			return false
		}
		global(value)
		return true
	}
	if len(args) == 2 && isAllScopeArg(args[0]) {
		value, ok := parseOnOff(args[1])
		if !ok {
			m.status = "usage: " + settingName + " all on|off"
			return false
		}
		global(value)
		m.status = "Hermes " + settingName + " for all " + onOff(value)
		return true
	}

	scopeKind := strings.ToLower(strings.TrimSpace(args[0]))
	switch scopeKind {
	case "host":
		if len(args) != 3 {
			m.status = "usage: " + settingName + " host <host> on|off"
			return false
		}
		value, ok := parseOnOff(args[2])
		if !ok {
			m.status = "usage: " + settingName + " host <host> on|off"
			return false
		}
		m.setHermesScopeBool(args[1], "", value, scoped)
		m.status = "Hermes " + settingName + " for host " + args[1] + " " + onOff(value)
		return true
	case "session":
		if len(args) != 3 {
			m.status = "usage: " + settingName + " session [host/]session on|off"
			return false
		}
		value, ok := parseOnOff(args[2])
		if !ok {
			m.status = "usage: " + settingName + " session [host/]session on|off"
			return false
		}
		host, session := splitScopedSessionTarget(args[1])
		m.setHermesScopeBool(host, session, value, scoped)
		m.status = "Hermes " + settingName + " for session " + args[1] + " " + onOff(value)
		return true
	case "here", "selected":
		if len(args) != 2 {
			m.status = "usage: " + settingName + " here on|off"
			return false
		}
		value, ok := parseOnOff(args[1])
		if !ok {
			m.status = "usage: " + settingName + " here on|off"
			return false
		}
		ref, ok := m.selectedSessionRef()
		if !ok {
			m.status = "select a session, window, or pane first"
			return false
		}
		host := displayHostName(ref.Host)
		m.setHermesScopeBool(host, ref.Session.Name, value, scoped)
		m.status = "Hermes " + settingName + " for session " + host + "/" + ref.Session.Name + " " + onOff(value)
		return true
	default:
		m.status = hermesScopeUsage(settingName)
		return false
	}
}

func hermesScopeUsage(settingName string) string {
	return "usage: " + settingName + " on|off [or all on|off, host <host|all> on|off, session [host/]session|all on|off, here on|off]"
}

func isAllScopeArg(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "all")
}

func splitScopedSessionTarget(value string) (string, string) {
	value = strings.TrimSpace(value)
	if before, after, ok := strings.Cut(value, "/"); ok {
		return strings.TrimSpace(before), strings.TrimSpace(after)
	}
	return "", value
}

func (m *model) setHermesScopeBool(host string, session string, value bool, set hermesScopeBoolSetter) {
	host = strings.TrimSpace(host)
	session = strings.TrimSpace(session)
	for i := range m.cfg.Hermes.Scopes {
		if strings.TrimSpace(m.cfg.Hermes.Scopes[i].Host) == host && strings.TrimSpace(m.cfg.Hermes.Scopes[i].Session) == session {
			set(&m.cfg.Hermes.Scopes[i], value)
			return
		}
	}
	scope := config.HermesScopeConfig{Host: host, Session: session}
	set(&scope, value)
	m.cfg.Hermes.Scopes = append(m.cfg.Hermes.Scopes, scope)
}

func boolSettingPtr(value bool) *bool {
	return &value
}
