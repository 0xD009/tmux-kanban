package main

import (
	"flag"
	"io"
	"os"

	"tmux-kanban/internal/config"
)

func cliCapabilities(args []string) error {
	fs := flag.NewFlagSet("capabilities", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	configPath := fs.String("config", "", "path to config yaml")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	return writeJSON(os.Stdout, collectCapabilities(cfg))
}

func collectCapabilities(cfg config.Config) cliCapabilitiesResponse {
	args := append([]string(nil), cfg.MainAgent.Args...)
	if args == nil {
		args = []string{}
	}
	return cliCapabilitiesResponse{
		OK: true,
		MainAgent: cliMainAgentCapability{
			Enabled: cfg.MainAgent.Enabled,
			Host:    cfg.MainAgent.Host,
			Session: cfg.MainAgent.Session,
			Agent:   normalizeConfigAgent(cfg.MainAgent.Agent),
			Command: cfg.MainAgent.Command,
			Args:    args,
		},
		Skills:      []string{mainSessionSkillName},
		CLICommands: append([]string(nil), mainSessionCLIAbilities...),
		Summary:     mainSessionCapabilitySummary(),
	}
}
