package main

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"tmux-kanban/internal/config"
)

func mainHermesCmd(cfg config.HermesConfig, prompt string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := hermesTimeoutContext(context.Background(), cfg)
		defer cancel()

		text, err := runHermesOneshot(ctx, cfg, prompt)
		if err != nil {
			return mainHermesResult{err: err.Error()}
		}
		return mainHermesResult{text: text}
	}
}

func (m model) mainHermesPrompt(userText string) string {
	var body strings.Builder
	body.WriteString("You are Hermes inside tmux-kanban Main Room, a chat-style conductor channel.\n")
	body.WriteString("Reply in concise Chinese. Help the user coordinate tmux agent sessions, review queue items, and next actions.\n")
	body.WriteString("Do not claim you executed tmux actions unless the context explicitly says so. Suggest concrete commands or choices when useful.\n\n")

	body.WriteString("User message:\n")
	body.WriteString(userText)
	body.WriteString("\n\n")

	queue := m.reviewQueue()
	body.WriteString(fmt.Sprintf("Review queue: %d item(s)\n", len(queue)))
	for i, item := range queue {
		if i >= 8 {
			body.WriteString(fmt.Sprintf("- ... %d more\n", len(queue)-i))
			break
		}
		body.WriteString(fmt.Sprintf("- %s/%s [%s] target=%s\n", item.HostName, item.SessionName, item.Agent, item.Row.attachTarget))
	}
	body.WriteString("\n")

	if len(m.activities) > 0 {
		body.WriteString("Recent activity:\n")
		start := maxInt(0, len(m.activities)-10)
		for _, activity := range m.activities[start:] {
			body.WriteString(fmt.Sprintf("- %s %s %s: %s\n", activity.Agent, activity.State, activity.Target, activity.Message))
		}
		body.WriteString("\n")
	}

	if len(m.mainMessages) > 0 {
		body.WriteString("Recent room messages:\n")
		start := maxInt(0, len(m.mainMessages)-8)
		for _, message := range m.mainMessages[start:] {
			text := strings.TrimSpace(message.Text)
			if text == "" {
				continue
			}
			body.WriteString(fmt.Sprintf("- %s/%s: %s\n", message.Author, message.Role, text))
		}
	}

	return body.String()
}
