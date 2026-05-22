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

type StartSessionResult struct {
	Host    config.Host
	Session string
	Created bool
	Err     string
}

type CloseSessionResult struct {
	Host    config.Host
	Session string
	Closed  bool
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

	start := StartSession(ctx, host, session, command, args...)
	return EnsureSessionResult{
		Host:    start.Host,
		Session: start.Session,
		Created: start.Created,
		Err:     start.Err,
	}
}

func StartSession(ctx context.Context, host config.Host, session string, command string, args ...string) StartSessionResult {
	result := StartSessionResult{Host: host, Session: strings.TrimSpace(session)}
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

	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	if host.Local {
		if err := exec.CommandContext(ctx, "tmux", "has-session", "-t", exactSessionTarget(session)).Run(); err == nil {
			return result
		}
		newSessionArgs := localNewSessionArgs(session, command, args...)
		output, err := exec.CommandContext(ctx, newSessionArgs[0], newSessionArgs[1:]...).CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			result.Err = "session start timed out"
			return result
		}
		if err != nil {
			result.Err = compactError(err, output)
			return result
		}
		result.Created = true
		return result
	}

	if err := exec.CommandContext(ctx, "ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", host.SSH, "tmux has-session -t "+shellQuote(exactSessionTarget(session))).Run(); err == nil {
		return result
	}
	output, err := exec.CommandContext(ctx, "ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", host.SSH, remoteNewSessionCommand(session, command, args...)).CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		result.Err = "session start timed out"
		return result
	}
	if err != nil {
		result.Err = compactError(err, output)
		return result
	}
	result.Created = true
	return result
}

func CloseSession(ctx context.Context, host config.Host, session string) CloseSessionResult {
	result := CloseSessionResult{Host: host, Session: strings.TrimSpace(session)}
	session = strings.TrimSpace(session)
	if !host.Local && strings.TrimSpace(host.SSH) == "" {
		result.Err = "missing ssh target"
		return result
	}
	if session == "" {
		result.Err = "missing tmux session"
		return result
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if host.Local {
		output, err := exec.CommandContext(ctx, "tmux", "kill-session", "-t", exactSessionTarget(session)).CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			result.Err = "session close timed out"
			return result
		}
		if err != nil {
			result.Err = compactError(err, output)
			return result
		}
		result.Closed = true
		return result
	}

	output, err := exec.CommandContext(ctx, "ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", host.SSH, "tmux kill-session -t "+shellQuote(exactSessionTarget(session))).CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		result.Err = "session close timed out"
		return result
	}
	if err != nil {
		result.Err = compactError(err, output)
		return result
	}
	result.Closed = true
	return result
}

func localNewSessionArgs(session string, command string, args ...string) []string {
	tmuxArgs := []string{"tmux", "new-session", "-d", "-s", session}
	if command := shellCommand(command, args...); command != "" {
		tmuxArgs = append(tmuxArgs, command)
	}
	return tmuxArgs
}

func remoteNewSessionCommand(session string, command string, args ...string) string {
	tmuxCommand := "tmux new-session -d -s " + shellQuote(session)
	if command := shellCommand(command, args...); command != "" {
		tmuxCommand += " " + shellQuote(command)
	}
	return tmuxCommand
}

func shellCommand(command string, args ...string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		return ""
	}
	parts := []string{shellQuote(command)}
	for _, arg := range args {
		if strings.TrimSpace(arg) == "" {
			continue
		}
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func exactSessionTarget(session string) string {
	return "=" + session
}
