package main

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"tmux-kanban/internal/config"
	"tmux-kanban/internal/mesh"
	tmuxclient "tmux-kanban/internal/tmux"
)

func (m model) executeMemoryCommand(args []string) (model, tea.Cmd) {
	if len(args) == 0 {
		m.status = "usage: memory update [global|host|session|window|pane]"
		return m, nil
	}
	switch strings.ToLower(args[0]) {
	case "update", "summarize", "write":
		scopeKind := "pane"
		if len(args) > 1 {
			scopeKind = args[1]
		}
		return m, m.memoryUpdateCmd(scopeKind)
	case "global", "host", "session", "window", "pane":
		return m, m.memoryUpdateCmd(args[0])
	default:
		m.status = "usage: memory update [global|host|session|window|pane]"
		return m, nil
	}
}

func (m *model) memoryUpdateCmd(scopeKind string) tea.Cmd {
	if strings.TrimSpace(m.cfg.AgentMesh.MemoryRoot) == "" {
		m.status = "agent mesh memory_root is not configured"
		return nil
	}
	if !m.cfg.Hermes.Enabled {
		m.status = "Hermes is disabled in config"
		return nil
	}
	if strings.TrimSpace(m.cfg.Hermes.Command) == "" {
		m.status = "Hermes command is not configured"
		return nil
	}
	ref, target, scope, ok := m.selectedMemoryUpdateTarget(scopeKind)
	if !ok {
		m.status = "select a session, window, or pane with an agent first"
		return nil
	}
	label := memoryScopeLabel(scope)
	m.status = "asking Hermes to update memory for " + label
	m.addAgentActivity(agentActivity{
		Source:  agentActivitySession,
		Agent:   "Hermes",
		Target:  label,
		State:   "memory",
		Message: "memory update requested",
	})
	return hermesMemoryUpdateCmd(m.cfg, ref, target, scope)
}

func (m model) selectedMemoryUpdateTarget(scopeKind string) (selectedSessionRef, selectedAgentTarget, mesh.Scope, bool) {
	ref, ok := m.selectedSessionRef()
	if !ok {
		return selectedSessionRef{}, selectedAgentTarget{}, mesh.Scope{}, false
	}
	target, ok := m.activeAgentTarget()
	if !ok {
		return selectedSessionRef{}, selectedAgentTarget{}, mesh.Scope{}, false
	}
	scope := sessionAdviceScope(ref, target)
	switch strings.ToLower(strings.TrimSpace(scopeKind)) {
	case "", "pane":
	case "window":
		scope.Pane = ""
	case "session":
		scope.Window = ""
		scope.Pane = ""
	case "host":
		scope.Session = ""
		scope.Window = ""
		scope.Pane = ""
	case "global", "all":
		scope = mesh.Scope{}
	default:
		return selectedSessionRef{}, selectedAgentTarget{}, mesh.Scope{}, false
	}
	return ref, target, scope, true
}

func hermesMemoryUpdateCmd(cfg config.Config, ref selectedSessionRef, target selectedAgentTarget, scope mesh.Scope) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := hermesTimeoutContext(context.Background(), cfg.Hermes)
		defer cancel()

		client := tmuxclient.DefaultClient{}
		capture := client.CapturePane(ctx, ref.Host, target.target, 80)
		if capture.Err != "" {
			return memoryUpdateResult{scope: scope, err: capture.Err}
		}
		contextNodes, err := mesh.LocalMemoryContext(cfg.AgentMesh.MemoryRoot, scope, 2400)
		warning := ""
		if err != nil {
			warning = err.Error()
		}
		prompt := hermesMemoryUpdatePrompt(ref, target, scope, capture.Lines, memorySummarizerSkillSnippet(cfg.AgentMesh), contextNodes, warning)
		text, err := runHermesOneshot(ctx, cfg.Hermes, prompt)
		if err != nil {
			return memoryUpdateResult{scope: scope, err: err.Error()}
		}
		text = cleanHermesMemoryText(text)
		path, err := mesh.WriteLocalMemory(cfg.AgentMesh.MemoryRoot, scope, text)
		if err != nil {
			return memoryUpdateResult{scope: scope, text: text, err: err.Error()}
		}
		return memoryUpdateResult{scope: scope, path: path, text: text}
	}
}

func hermesMemoryUpdatePrompt(ref selectedSessionRef, target selectedAgentTarget, scope mesh.Scope, lines []string, skill string, memory []mesh.MemoryNode, warning string) string {
	var body strings.Builder
	body.WriteString("You are updating tmux-kanban scoped memory.\n")
	body.WriteString("Produce the complete replacement content for the target memory file using only the supplied context.\n\n")
	if strings.TrimSpace(skill) != "" {
		body.WriteString("Memory skill:\n")
		body.WriteString("```markdown\n")
		body.WriteString(strings.TrimSpace(skill))
		body.WriteString("\n```\n\n")
	}
	body.WriteString("Target memory scope:\n")
	body.WriteString("- " + scope.Key() + "\n\n")
	body.WriteString("Related pane:\n")
	body.WriteString("- Host: " + displayHostName(ref.Host) + "\n")
	body.WriteString("- Session: " + ref.Session.Name + "\n")
	body.WriteString("- Target: " + target.target + "\n")
	body.WriteString("- Agent: " + string(target.agent) + "\n\n")
	if len(memory) > 0 || strings.TrimSpace(warning) != "" {
		body.WriteString("Existing scoped memory context:\n")
		for _, node := range memory {
			body.WriteString(fmt.Sprintf("- %s: %s\n", node.Scope.Key(), compactPromptLine(node.Summary, 520)))
		}
		if strings.TrimSpace(warning) != "" {
			body.WriteString("- memory warning: " + compactPromptLine(warning, 240) + "\n")
		}
		body.WriteString("\n")
	}
	body.WriteString("Visible terminal tail:\n")
	body.WriteString("```text\n")
	body.WriteString(strings.Join(tailPreviewLines(lines, 160, 50), "\n"))
	body.WriteString("\n```\n\n")
	body.WriteString("Reply with memory file content only. Use concise Chinese. Do not wrap the output in code fences.\n")
	body.WriteString("Preferred format:\n")
	body.WriteString("TITLE: <short title>\n")
	body.WriteString("SUMMARY: <facts, decisions, current state, and stable guidance>\n")
	return body.String()
}

func memorySummarizerSkillSnippet(cfg config.AgentMeshConfig) string {
	return meshRoleSkillSnippet(cfg, mesh.RoleMemorySummarizer, "memory-summarizer")
}

func cleanHermesMemoryText(text string) string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		lines := strings.Split(text, "\n")
		if len(lines) >= 2 {
			lines = lines[1:]
			if strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "```") {
				lines = lines[:len(lines)-1]
			}
			text = strings.TrimSpace(strings.Join(lines, "\n"))
		}
	}
	return text
}

func memoryScopeLabel(scope mesh.Scope) string {
	return scope.Key()
}
