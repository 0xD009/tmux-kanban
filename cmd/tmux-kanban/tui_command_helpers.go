package main

import (
	"fmt"
	"strings"

	"tmux-kanban/internal/config"
	"tmux-kanban/internal/core"
)

func parseOnOff(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "true", "1", "yes", "enabled", "enable":
		return true, true
	case "off", "false", "0", "no", "disabled", "disable":
		return false, true
	default:
		return false, false
	}
}

func parseSessionStatusValue(value string) (sessionStatus, bool) {
	return core.ParseStatus(value)
}

func commandRemainder(input string, commandName string) string {
	if len(input) <= len(commandName) {
		return ""
	}
	return strings.TrimSpace(input[len(commandName):])
}

func onOff(value bool) string {
	if value {
		return "on"
	}
	return "off"
}

func normalizeConfigAgent(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "claude", "claude-code":
		return "claude-code"
	case "codex", "":
		return "codex"
	default:
		return strings.TrimSpace(value)
	}
}

func normalizeMeshBackend(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "claude", "claude-code":
		return "claude-code"
	case "codex", "":
		return "codex"
	case "builtin", "internal":
		return "builtin"
	case "hermes":
		return "hermes"
	case "command", "exec":
		return "command"
	default:
		return strings.TrimSpace(value)
	}
}

func configValueOrEmpty(args []string) string {
	value := strings.TrimSpace(strings.Join(args, " "))
	switch strings.ToLower(value) {
	case "clear", "none", "default", "\"\"", "''":
		return ""
	default:
		return value
	}
}

func emptyValueLabel(value string) string {
	if strings.TrimSpace(value) == "" {
		return "<empty>"
	}
	return value
}

func (m *model) ensureMeshPolicyDefaults() {
	for i := range m.cfg.AgentMesh.Policies {
		if strings.TrimSpace(m.cfg.AgentMesh.Policies[i].Agent) == "" {
			m.cfg.AgentMesh.Policies[i].Agent = m.cfg.AgentMesh.DefaultAgent
		}
		if strings.TrimSpace(m.cfg.AgentMesh.Policies[i].Backend) == "" {
			m.cfg.AgentMesh.Policies[i].Backend = m.cfg.AgentMesh.Policies[i].Agent
		}
		if strings.TrimSpace(m.cfg.AgentMesh.Policies[i].Skill) == "" {
			m.cfg.AgentMesh.Policies[i].Skill = m.cfg.AgentMesh.Policies[i].Role
		}
		if strings.TrimSpace(m.cfg.AgentMesh.Policies[i].Command) == "" {
			m.cfg.AgentMesh.Policies[i].Command = config.DefaultMainAgentCommand(m.cfg.AgentMesh.Policies[i].Agent)
		}
		if m.cfg.AgentMesh.Policies[i].Args == nil {
			m.cfg.AgentMesh.Policies[i].Args = []string{}
		}
	}
}

func (m model) meshPolicyIndex(name string) int {
	needle := strings.ToLower(strings.TrimSpace(name))
	for i, policy := range m.cfg.AgentMesh.Policies {
		if strings.ToLower(policy.Name) == needle || strings.ToLower(policy.Role) == needle {
			return i
		}
	}
	return -1
}

func (m model) meshPolicySummary() string {
	if len(m.cfg.AgentMesh.Policies) == 0 {
		return "<none>"
	}
	parts := make([]string, 0, len(m.cfg.AgentMesh.Policies))
	for _, policy := range m.cfg.AgentMesh.Policies {
		name := policy.Name
		if name == "" {
			name = policy.Role
		}
		backend := policy.Backend
		if backend == "" {
			backend = policy.Agent
		}
		parts = append(parts, fmt.Sprintf("%s:%s/%s/%s/%s", name, onOff(policy.Enabled), policy.Scope, backend, policy.Skill))
	}
	return strings.Join(parts, ", ")
}
