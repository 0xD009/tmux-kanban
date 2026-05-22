package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) executeViewCommand(args []string) (model, tea.Cmd) {
	if len(args) != 1 {
		m.status = "usage: view tree|review"
		return m, nil
	}
	switch strings.ToLower(args[0]) {
	case "tree":
		m.setViewMode(viewTree)
		return m, nil
	case "review", "queue":
		m.setViewMode(viewReview)
		return m, nil
	default:
		m.status = "usage: view tree|review"
		return m, nil
	}
}
