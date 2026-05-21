package main

import (
	"fmt"
	"strings"

	"tmux-kanban/internal/config"
)

func (m *model) executeMeshCommand(args []string) {
	if len(args) == 0 || strings.EqualFold(args[0], "status") || strings.EqualFold(args[0], "settings") {
		m.status = fmt.Sprintf("mesh: enabled=%s shared=%s default=%s skill_root=%s policies=%d mail=%s memory=%s", onOff(m.cfg.AgentMesh.Enabled), onOff(m.cfg.AgentMesh.SharedShortAgent), m.cfg.AgentMesh.DefaultAgent, emptyValueLabel(m.cfg.AgentMesh.SkillRoot), len(m.cfg.AgentMesh.Policies), onOff(m.cfg.AgentMesh.Mail.Enabled), emptyValueLabel(m.cfg.AgentMesh.MemoryRoot))
		return
	}

	switch strings.ToLower(args[0]) {
	case "on", "enable":
		m.cfg.AgentMesh.Enabled = true
		m.status = "agent mesh on"
	case "off", "disable":
		m.cfg.AgentMesh.Enabled = false
		m.status = "agent mesh off"
	case "shared":
		if len(args) != 2 {
			m.status = "usage: mesh shared on|off"
			return
		}
		value, ok := parseOnOff(args[1])
		if !ok {
			m.status = "usage: mesh shared on|off"
			return
		}
		m.cfg.AgentMesh.SharedShortAgent = value
		m.status = "agent mesh shared_short_agent " + onOff(value)
	case "default", "default-agent", "default_agent":
		if len(args) != 2 {
			m.status = "usage: mesh default codex|claude-code"
			return
		}
		m.cfg.AgentMesh.DefaultAgent = normalizeConfigAgent(args[1])
		m.ensureMeshPolicyDefaults()
		m.status = "agent mesh default_agent " + m.cfg.AgentMesh.DefaultAgent
	case "memory", "memory-root", "memory_root":
		m.cfg.AgentMesh.MemoryRoot = configValueOrEmpty(args[1:])
		m.status = "agent mesh memory_root " + emptyValueLabel(m.cfg.AgentMesh.MemoryRoot)
	case "skill-root", "skill_root", "skills":
		m.cfg.AgentMesh.SkillRoot = configValueOrEmpty(args[1:])
		m.status = "agent mesh skill_root " + emptyValueLabel(m.cfg.AgentMesh.SkillRoot)
	case "mail":
		m.executeMeshMailCommand(args[1:])
	case "policy":
		m.executeMeshPolicyCommand(args[1:])
	default:
		m.status = "usage: mesh on|off|status|shared|default|memory|skill-root|mail|policy"
	}
}

func (m *model) executeMeshMailCommand(args []string) {
	if len(args) == 0 {
		m.status = fmt.Sprintf("mesh mail: enabled=%s dir=%s", onOff(m.cfg.AgentMesh.Mail.Enabled), emptyValueLabel(m.cfg.AgentMesh.Mail.Dir))
		return
	}
	switch strings.ToLower(args[0]) {
	case "on", "enable":
		m.cfg.AgentMesh.Mail.Enabled = true
		m.status = "agent mesh mail on"
	case "off", "disable":
		m.cfg.AgentMesh.Mail.Enabled = false
		m.status = "agent mesh mail off"
	case "dir":
		m.cfg.AgentMesh.Mail.Dir = configValueOrEmpty(args[1:])
		m.status = "agent mesh mail dir " + emptyValueLabel(m.cfg.AgentMesh.Mail.Dir)
	default:
		m.status = "usage: mesh mail on|off|dir <path>"
	}
}

func (m *model) executeMeshPolicyCommand(args []string) {
	if len(args) == 0 || strings.EqualFold(args[0], "list") {
		m.status = "mesh policies: " + m.meshPolicySummary()
		return
	}
	if len(args) < 2 {
		m.status = "usage: mesh policy <name> on|off|backend|skill|agent|command|scope|role"
		return
	}

	name := args[0]
	index := m.meshPolicyIndex(name)
	if index < 0 {
		m.status = "unknown mesh policy: " + name
		return
	}
	policy := &m.cfg.AgentMesh.Policies[index]
	switch strings.ToLower(args[1]) {
	case "on", "enable":
		policy.Enabled = true
		m.status = "mesh policy " + policy.Name + " on"
	case "off", "disable":
		policy.Enabled = false
		m.status = "mesh policy " + policy.Name + " off"
	case "agent":
		if len(args) != 3 {
			m.status = "usage: mesh policy " + name + " agent codex|claude-code"
			return
		}
		policy.Agent = normalizeConfigAgent(args[2])
		policy.Command = config.DefaultMainAgentCommand(policy.Agent)
		if strings.TrimSpace(policy.Backend) == "" || policy.Backend == "codex" || policy.Backend == "claude-code" {
			policy.Backend = policy.Agent
		}
		m.status = "mesh policy " + policy.Name + " agent " + policy.Agent
	case "backend":
		if len(args) != 3 {
			m.status = "usage: mesh policy " + name + " backend builtin|codex|claude-code|hermes|command"
			return
		}
		policy.Backend = normalizeMeshBackend(args[2])
		if policy.Backend == "codex" || policy.Backend == "claude-code" {
			policy.Agent = policy.Backend
			policy.Command = config.DefaultMainAgentCommand(policy.Agent)
		}
		m.status = "mesh policy " + policy.Name + " backend " + policy.Backend
	case "skill":
		if len(args) != 3 {
			m.status = "usage: mesh policy " + name + " skill <skill-folder>"
			return
		}
		policy.Skill = configValueOrEmpty(args[2:])
		m.status = "mesh policy " + policy.Name + " skill " + emptyValueLabel(policy.Skill)
	case "command":
		if len(args) < 3 {
			m.status = "usage: mesh policy " + name + " command <command> [args...]"
			return
		}
		policy.Command = args[2]
		policy.Args = append([]string(nil), args[3:]...)
		m.status = "mesh policy " + policy.Name + " command " + strings.Join(args[2:], " ")
	case "scope":
		if len(args) != 3 {
			m.status = "usage: mesh policy " + name + " scope global|host|session|window|pane"
			return
		}
		policy.Scope = args[2]
		m.status = "mesh policy " + policy.Name + " scope " + policy.Scope
	case "role":
		if len(args) != 3 {
			m.status = "usage: mesh policy " + name + " role <role>"
			return
		}
		policy.Role = args[2]
		m.status = "mesh policy " + policy.Name + " role " + policy.Role
	default:
		m.status = "usage: mesh policy <name> on|off|backend|skill|agent|command|scope|role"
	}
}
