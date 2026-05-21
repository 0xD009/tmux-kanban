package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"tmux-kanban/internal/config"
)

func main() {
	if len(os.Args) > 1 && isCLICommand(os.Args[1]) {
		if err := runCLI(os.Args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	configPath := flag.String("config", "", "path to config yaml")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	output := cursorAwareOutput{file: os.Stdout}
	if _, err := tea.NewProgram(initialModel(cfg), tea.WithAltScreen(), tea.WithMouseCellMotion(), tea.WithOutput(output)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
