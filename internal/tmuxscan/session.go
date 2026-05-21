package tmuxscan

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"tmux-kanban/internal/config"
)

type EnsureSessionResult struct {
	Host    config.Host
	Session string
	Created bool
	Err     string
}

func EnsureSession(ctx context.Context, host config.Host, session string, command string, args ...string) EnsureSessionResult {
	result := EnsureSessionResult{Host: host, Session: session}
	session = strings.TrimSpace(session)
	command = strings.TrimSpace(command)
	if !host.Local && strings.TrimSpace(host.SSH) == "" {
		result.Err = "missing ssh target"
		return result
	}
	if session == "" {
		result.Err = "missing tmux session"
		return result
	}
	if command == "" {
		result.Err = "missing main agent command"
		return result
	}

	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	if host.Local {
		if err := exec.CommandContext(ctx, "tmux", "has-session", "-t", session).Run(); err == nil {
			return result
		}
		newSessionArgs := localNewSessionArgs(session, command, args...)
		output, err := exec.CommandContext(ctx, newSessionArgs[0], newSessionArgs[1:]...).CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			result.Err = "main session start timed out"
			return result
		}
		if err != nil {
			result.Err = compactError(err, output)
			return result
		}
		result.Created = true
		return result
	}

	if err := exec.CommandContext(ctx, "ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", host.SSH, "tmux has-session -t "+shellQuote(session)).Run(); err == nil {
		return result
	}
	output, err := exec.CommandContext(ctx, "ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", host.SSH, remoteNewSessionCommand(session, command, args...)).CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		result.Err = "main session start timed out"
		return result
	}
	if err != nil {
		result.Err = compactError(err, output)
		return result
	}
	result.Created = true
	return result
}

func localNewSessionArgs(session string, command string, args ...string) []string {
	return []string{"tmux", "new-session", "-d", "-s", session, shellCommand(command, args...)}
}

func remoteNewSessionCommand(session string, command string, args ...string) string {
	return "tmux new-session -d -s " + shellQuote(session) + " " + shellQuote(shellCommand(command, args...))
}

func shellCommand(command string, args ...string) string {
	parts := []string{shellQuote(command)}
	for _, arg := range args {
		if strings.TrimSpace(arg) == "" {
			continue
		}
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}
