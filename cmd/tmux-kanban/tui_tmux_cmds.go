package main

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"tmux-kanban/internal/agent"
	"tmux-kanban/internal/config"
	tmuxclient "tmux-kanban/internal/tmux"
)

func (m model) currentAgentScreen() agent.Screen {
	if len(m.preview.lines) == 0 {
		return agent.Screen{SelectedChoice: -1}
	}
	return agent.AnalyzeScreen(m.preview.lines)
}

func (m model) shouldPollPreview(key string) bool {
	if m.preview.key != key {
		return false
	}
	selected, ok := m.activePreviewRow()
	return ok && selected.attachTarget != "" && previewKey(selected) == key
}

func previewTickCmd(key string) tea.Cmd {
	return tea.Tick(previewRefreshInterval, func(time.Time) tea.Msg {
		return previewTick{key: key}
	})
}

func scanTickCmd() tea.Cmd {
	return tea.Tick(pollInterval, func(time.Time) tea.Msg {
		return scanTick{}
	})
}

func sendKeysCmd(action string, host config.Host, target string, keys ...string) tea.Cmd {
	return func() tea.Msg {
		client := tmuxclient.DefaultClient{}
		return sendResult{
			action: action,
			result: client.SendKeys(context.Background(), host, target, keys...),
		}
	}
}

func sendKeySequenceCmd(action string, host config.Host, target string, keys ...string) tea.Cmd {
	return func() tea.Msg {
		client := tmuxclient.DefaultClient{}
		return sendResult{
			action: action,
			result: client.SendKeySequence(context.Background(), host, target, keys...),
		}
	}
}

func sendTextCmd(action string, host config.Host, target string, text string, submit bool) tea.Cmd {
	return func() tea.Msg {
		client := tmuxclient.DefaultClient{}
		return sendResult{
			action: action,
			result: client.SendText(context.Background(), host, target, text, submit),
		}
	}
}
