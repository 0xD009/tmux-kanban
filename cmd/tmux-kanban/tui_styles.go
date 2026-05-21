package main

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("81"))

	headerMetaStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	headerStatusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("229"))

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(1, 2)

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230"))

	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	previewBorderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("109"))

	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("236"))

	inputBoxBorderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("81"))

	inputBoxTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("24"))

	selectedRowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("236")).
				Bold(true)

	hostRowStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("153"))

	sessionRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	windowRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250"))

	paneRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	ruleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("203"))

	warnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("120"))
)
