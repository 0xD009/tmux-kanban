package tmux

import (
	"context"
	"os/exec"

	"tmux-kanban/internal/config"
	"tmux-kanban/internal/tmuxscan"
)

type Client interface {
	ScanHost(ctx context.Context, host config.Host) tmuxscan.Snapshot
	CapturePane(ctx context.Context, host config.Host, target string, height int) tmuxscan.Capture
	SendKeys(ctx context.Context, host config.Host, target string, keys ...string) tmuxscan.SendResult
	SendKeySequence(ctx context.Context, host config.Host, target string, keys ...string) tmuxscan.SendResult
	SendText(ctx context.Context, host config.Host, target string, text string, submit bool) tmuxscan.SendResult
	EnsureSession(ctx context.Context, host config.Host, session string, command string, args ...string) tmuxscan.EnsureSessionResult
	StartSession(ctx context.Context, host config.Host, session string, command string, args ...string) tmuxscan.StartSessionResult
	CloseSession(ctx context.Context, host config.Host, session string) tmuxscan.CloseSessionResult
	AttachCommand(host config.Host, target string) *exec.Cmd
}

type DefaultClient struct{}

func (DefaultClient) ScanHost(ctx context.Context, host config.Host) tmuxscan.Snapshot {
	return tmuxscan.ScanHost(ctx, host)
}

func (DefaultClient) CapturePane(ctx context.Context, host config.Host, target string, height int) tmuxscan.Capture {
	return tmuxscan.CapturePane(ctx, host, target, height)
}

func (DefaultClient) SendKeys(ctx context.Context, host config.Host, target string, keys ...string) tmuxscan.SendResult {
	return tmuxscan.SendKeys(ctx, host, target, keys...)
}

func (DefaultClient) SendKeySequence(ctx context.Context, host config.Host, target string, keys ...string) tmuxscan.SendResult {
	return tmuxscan.SendKeySequence(ctx, host, target, keys...)
}

func (DefaultClient) SendText(ctx context.Context, host config.Host, target string, text string, submit bool) tmuxscan.SendResult {
	return tmuxscan.SendText(ctx, host, target, text, submit)
}

func (DefaultClient) EnsureSession(ctx context.Context, host config.Host, session string, command string, args ...string) tmuxscan.EnsureSessionResult {
	return tmuxscan.EnsureSession(ctx, host, session, command, args...)
}

func (DefaultClient) StartSession(ctx context.Context, host config.Host, session string, command string, args ...string) tmuxscan.StartSessionResult {
	return tmuxscan.StartSession(ctx, host, session, command, args...)
}

func (DefaultClient) CloseSession(ctx context.Context, host config.Host, session string) tmuxscan.CloseSessionResult {
	return tmuxscan.CloseSession(ctx, host, session)
}

func (DefaultClient) AttachCommand(host config.Host, target string) *exec.Cmd {
	return tmuxscan.AttachCommand(host, target)
}
