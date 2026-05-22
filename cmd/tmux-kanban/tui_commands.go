package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"tmux-kanban/internal/config"
)

func (m *model) beginCommand() {
	m.command = commandState{active: true}
	m.status = "command mode (:help)"
}

func (m model) updateCommand(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.command = commandState{}
		m.status = "command canceled"
		return m, tea.HideCursor
	case "ctrl+c":
		m.command = commandState{}
		m.status = "command canceled"
		return m, tea.HideCursor
	case "ctrl+u":
		m.command.text = ""
		m.command.selected = 0
		return m, nil
	case "up", "ctrl+p":
		m.moveCommandSelection(-1)
		return m, nil
	case "down", "ctrl+n":
		m.moveCommandSelection(1)
		return m, nil
	case "tab":
		if candidate, ok := m.selectedCommandCandidate(); ok {
			m.command.text = candidate.Text
			m.command.selected = 0
		}
		return m, nil
	case "backspace", "ctrl+h":
		runes := []rune(m.command.text)
		if len(runes) > 0 {
			m.command.text = string(runes[:len(runes)-1])
		}
		m.clampCommandSelection()
		return m, nil
	case "enter":
		if next, ok := m.completeCommandCandidateForValue(); ok {
			return next, nil
		}
		input := m.commandInputForEnter()
		m.command = commandState{}
		if input == "" {
			m.status = "command canceled"
			return m, tea.HideCursor
		}
		next, cmd := m.executeCommand(input)
		if next.snapshotInput.active {
			return next, tea.ShowCursor
		}
		return next, tea.Batch(tea.HideCursor, cmd)
	default:
		if len(msg.Runes) > 0 {
			m.command.text += string(inputRunesFromKey(msg, false))
			m.clampCommandSelection()
		}
		return m, nil
	}
}

func (m *model) moveCommandSelection(delta int) {
	candidates := commandCandidates(m.command.text)
	if len(candidates) == 0 {
		m.command.selected = 0
		return
	}
	m.command.selected = (m.command.selected + delta) % len(candidates)
	if m.command.selected < 0 {
		m.command.selected += len(candidates)
	}
}

func (m *model) clampCommandSelection() {
	candidates := commandCandidates(m.command.text)
	if len(candidates) == 0 {
		m.command.selected = 0
		return
	}
	m.command.selected = clampInt(m.command.selected, 0, len(candidates)-1)
}

func (m model) selectedCommandCandidate() (commandCandidate, bool) {
	candidates := commandCandidates(m.command.text)
	if len(candidates) == 0 {
		return commandCandidate{}, false
	}
	index := clampInt(m.command.selected, 0, len(candidates)-1)
	return candidates[index], true
}

func (m model) commandInputForEnter() string {
	input := strings.TrimSpace(m.command.text)
	if input == "" {
		return ""
	}
	candidate, ok := m.selectedCommandCandidate()
	if !ok {
		return input
	}
	lowerInput := strings.ToLower(input)
	lowerCandidate := strings.ToLower(candidate.Text)
	if lowerInput != lowerCandidate && strings.HasPrefix(lowerCandidate, lowerInput) {
		return candidate.Text
	}
	return input
}

func (m model) completeCommandCandidateForValue() (model, bool) {
	input := strings.TrimSpace(m.command.text)
	candidate, ok := m.selectedCommandCandidate()
	if !ok || !candidate.NeedsValue() {
		return m, false
	}

	candidateText := strings.TrimSpace(candidate.Text)
	if input == "" || input == candidateText {
		return m, false
	}
	if len(strings.Fields(input)) > len(strings.Fields(candidateText)) {
		return m, false
	}

	lowerInput := strings.ToLower(input)
	lowerCandidate := strings.ToLower(candidateText)
	lowerLabel := strings.ToLower(strings.TrimSpace(candidate.Label()))
	if strings.HasPrefix(lowerCandidate, lowerInput) || strings.HasPrefix(lowerLabel, lowerInput) {
		m.command.text = candidate.Text
		m.command.selected = 0
		return m, true
	}
	return m, false
}

func (m model) executeCommand(input string) (model, tea.Cmd) {
	fields := strings.Fields(input)
	if len(fields) == 0 {
		m.status = "command canceled"
		return m, nil
	}

	name := strings.ToLower(fields[0])
	args := fields[1:]
	switch name {
	case "help", "?":
		m.status = "commands: refresh | view tree/review | memory update scope | mesh on/off/status/policy | set qq/hermes/hermes.auto_review/hermes.done_advice/hermes.auto_done/hermes.idle_advice/hermes.auto_idle/mesh.* | status idle/working/need-review/done | notify | snapshot"
	case "refresh", "scan", "r":
		return m.startScanModel()
	case "tree":
		m.setViewMode(viewTree)
	case "review":
		m.setViewMode(viewReview)
	case "mesh", "agent-mesh":
		m.executeMeshCommand(args)
	case "memory", "mem":
		return m.executeMemoryCommand(args)
	case "view":
		return m.executeViewCommand(args)
	case "set":
		return m, m.executeSetCommand(args)
	case "settings":
		m.status = fmt.Sprintf("settings: qq=%s hermes=%s hermes.auto_review=%s hermes.done_advice=%s hermes.auto_done=%s hermes.idle_advice=%s hermes.auto_idle=%s hermes.scopes=%d mesh=%s view=%s", onOff(m.cfg.Notification.QQEnabled), onOff(m.cfg.Hermes.Enabled), onOff(m.cfg.Hermes.AutoReview), onOff(m.cfg.Hermes.DoneAdvice), onOff(m.cfg.Hermes.AutoDone), onOff(m.cfg.Hermes.IdleAdvice), onOff(m.cfg.Hermes.AutoIdle), len(m.cfg.Hermes.Scopes), onOff(m.cfg.AgentMesh.Enabled), m.viewMode)
	case "qq":
		m.executeBoolSettingCommand("qq", args, func(value bool) {
			m.cfg.Notification.QQEnabled = value
			m.status = "QQ notification " + onOff(value)
		})
	case "hermes":
		m.setHermesScopedBool(args, func(value bool) {
			m.cfg.Hermes.Enabled = value
			m.status = "Hermes " + onOff(value)
		}, func(scope *config.HermesScopeConfig, value bool) {
			scope.Enabled = boolSettingPtr(value)
		}, "hermes")
	case "status":
		return m, m.executeStatusCommand(args)
	case "notify":
		intent := commandRemainder(input, fields[0])
		return m, m.notifyQQForReviewQueue(intent)
	case "snapshot":
		description := strings.TrimSpace(commandRemainder(input, fields[0]))
		if description == "" {
			m.beginSnapshotDescription()
			return m, tea.ShowCursor
		}
		return m, m.saveSnapshotCmd(description)
	default:
		m.status = "unknown command: " + fields[0]
	}
	return m, nil
}
