package main

import tea "github.com/charmbracelet/bubbletea"

func (m model) executeViewCommand(args []string) (model, tea.Cmd) {
	m.status = "usage: view tree"
	return m, nil
}
