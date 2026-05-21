package main

import (
	"context"

	"tmux-kanban/internal/agent"
	"tmux-kanban/internal/config"
)

func hermesTimeoutContext(parent context.Context, cfg config.HermesConfig) (context.Context, context.CancelFunc) {
	return agent.HermesTimeoutContext(parent, cfg)
}

func runHermesOneshot(ctx context.Context, cfg config.HermesConfig, prompt string) (string, error) {
	return agent.RunHermesOneshot(ctx, cfg, prompt)
}
