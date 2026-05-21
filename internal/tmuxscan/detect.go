package tmuxscan

import "strings"

func DetectAgent(pane Pane) AgentKind {
	if isClaudeCommand(pane.Command, "") {
		return AgentClaude
	}
	if isCodexCommand(pane.Command, "") {
		return AgentCodex
	}

	for _, process := range pane.Processes {
		if isClaudeCommand(process.Command, process.Args) {
			return AgentClaude
		}
	}
	for _, process := range pane.Processes {
		if isCodexCommand(process.Command, process.Args) {
			return AgentCodex
		}
	}

	return AgentNone
}

func isClaudeCommand(command, args string) bool {
	name := normalizeCommandName(command)
	if name == "claude" || name == "claude-code" {
		return true
	}

	args = strings.ToLower(args)
	return strings.Contains(args, "@anthropic-ai/claude-code") ||
		strings.Contains(args, "claude-code") ||
		hasExecutableName(args, "claude") ||
		hasExecutableName(args, "claude-code")
}

func isCodexCommand(command, args string) bool {
	name := normalizeCommandName(command)
	if name == "codex" || strings.HasPrefix(name, "codex-") {
		return true
	}

	args = strings.ToLower(args)
	return strings.Contains(args, "@openai/codex") ||
		strings.Contains(args, "codex app-server") ||
		strings.Contains(args, "codex exec") ||
		hasExecutableName(args, "codex")
}

func normalizeCommandName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	value = strings.Trim(value, "()[]{}")
	if value == "" {
		return ""
	}

	value = strings.TrimRight(value, ":,;")
	parts := strings.Split(value, "/")
	return strings.ToLower(parts[len(parts)-1])
}

func hasExecutableName(args, wanted string) bool {
	if args == "" {
		return false
	}

	wanted = strings.ToLower(wanted)
	afterWrapper := false
	for _, field := range strings.Fields(args) {
		token := strings.Trim(field, `"'`)
		if token == "" {
			continue
		}
		if strings.Contains(token, "=") && !strings.Contains(token, "/") {
			continue
		}

		name := normalizeCommandName(token)
		if name == wanted {
			return true
		}
		if isCommandWrapper(name) {
			afterWrapper = true
			continue
		}
		if afterWrapper && strings.HasPrefix(token, "-") {
			continue
		}

		return false
	}

	return false
}

func isCommandWrapper(name string) bool {
	switch name {
	case "bash", "bun", "corepack", "deno", "env", "fish", "node", "nodejs", "npm", "npx", "pnpm", "sh", "sudo", "yarn", "zsh":
		return true
	default:
		return false
	}
}
