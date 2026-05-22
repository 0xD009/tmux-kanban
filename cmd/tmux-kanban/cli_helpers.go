package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"tmux-kanban/internal/config"
)

func loadHost(configPath string, hostName string) (config.Config, config.Host, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return config.Config{}, config.Host{}, err
	}
	host, ok := findHost(cfg, hostName)
	if !ok {
		if strings.TrimSpace(hostName) == "" {
			return cfg, config.Host{}, errors.New("missing --host")
		}
		return cfg, config.Host{}, fmt.Errorf("host %q not found", hostName)
	}
	return cfg, host, nil
}

func findHost(cfg config.Config, hostName string) (config.Host, bool) {
	for _, host := range cfg.Hosts {
		if host.Name == hostName || host.SSH == hostName {
			return host, true
		}
	}
	return config.Host{}, false
}

func displayHostName(host config.Host) string {
	return config.HostDisplayName(host)
}

func splitKeys(value string) []string {
	parts := strings.Split(value, ",")
	keys := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			keys = append(keys, part)
		}
	}
	return keys
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func printCLIUsage(w io.Writer) {
	fmt.Fprintln(w, `tmux-kanban CLI commands:
  capabilities --config ./config.yaml
  review-list   --config ./config.yaml [--all] [--lines] [--notify]
  notify-review --config ./config.yaml [--lines] [--intent "..."]
  capture     --config ./config.yaml --host local --target %1
  choose      --config ./config.yaml --host local --target %1 --choice 1
  send        --config ./config.yaml --host local --target %1 --text "hello" [--submit=false]
  send-keys   --config ./config.yaml --host local --target %1 --keys C-c,C-m
  session-open  --config ./config.yaml --host local --name work [--command codex -- --profile kanban]
  session-close --config ./config.yaml --host local --name work --confirm local/work
  snapshot    --config ./config.yaml [--description "..."] [--dir ~/.local/state/tmux-kanban/snapshots]

All commands print JSON to stdout. The main session is a conductor with the tmux-kanban-review skill and these JSON CLI abilities.`)
}
