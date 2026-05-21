package agent

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

	"tmux-kanban/internal/config"
)

type HermesReviewer struct {
	Config config.HermesConfig
	Prompt func(ReviewRequest) string
}

func (r HermesReviewer) Review(ctx context.Context, request ReviewRequest) (ReviewAdvice, error) {
	if r.Prompt == nil {
		return ReviewAdvice{}, errors.New("Hermes prompt builder is not configured")
	}
	ctx, cancel := HermesTimeoutContext(ctx, r.Config)
	defer cancel()

	text, err := RunHermesOneshot(ctx, r.Config, r.Prompt(request))
	if err != nil {
		return ReviewAdvice{}, err
	}
	return ReviewAdvice{Text: text}, nil
}

func HermesTimeoutContext(parent context.Context, cfg config.HermesConfig) (context.Context, context.CancelFunc) {
	timeout := cfg.TimeoutSeconds
	if timeout <= 0 {
		timeout = 120
	}
	return context.WithTimeout(parent, time.Duration(timeout)*time.Second)
}

func RunHermesOneshot(ctx context.Context, cfg config.HermesConfig, prompt string) (string, error) {
	command := strings.TrimSpace(cfg.Command)
	if command == "" {
		return "", errors.New("Hermes command is not configured")
	}

	args := append([]string{}, cfg.Args...)
	if len(args) == 0 {
		args = []string{"--oneshot"}
	}
	args = append(args, prompt)

	cmd := exec.CommandContext(ctx, command, args...)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", errors.New("Hermes query timed out")
	}
	if err != nil {
		return "", errors.New(compactCommandError(err, output))
	}

	text := strings.TrimSpace(string(output))
	if text == "" {
		text = "<empty Hermes response>"
	}
	return text, nil
}

func compactCommandError(err error, output []byte) string {
	text := strings.TrimSpace(string(output))
	if text == "" {
		return err.Error()
	}
	return err.Error() + ": " + clipString(text, 240)
}

func clipString(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit] + "..."
}
