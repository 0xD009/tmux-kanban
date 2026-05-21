package tmuxscan

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"tmux-kanban/internal/config"
)

type SendResult struct {
	Host   config.Host
	Target string
	Err    string
}

const (
	pasteSubmitDelay = 80 * time.Millisecond
	keySequenceDelay = 60 * time.Millisecond
)

func SendKeys(ctx context.Context, host config.Host, target string, keys ...string) SendResult {
	result := SendResult{Host: host, Target: target}
	if !host.Local && strings.TrimSpace(host.SSH) == "" {
		result.Err = "missing ssh target"
		return result
	}
	if strings.TrimSpace(target) == "" {
		result.Err = "missing tmux target"
		return result
	}
	if len(keys) == 0 {
		result.Err = "missing keys"
		return result
	}

	if host.Local {
		return runLocalTmux(ctx, host, target, append([]string{"send-keys", "-t", target, "--"}, keys...))
	}

	return runRemoteTmux(ctx, host, target, sendKeysRemoteCommand(target, keys...))
}

func SendKeySequence(ctx context.Context, host config.Host, target string, keys ...string) SendResult {
	result := SendResult{Host: host, Target: target}
	if !host.Local && strings.TrimSpace(host.SSH) == "" {
		result.Err = "missing ssh target"
		return result
	}
	if strings.TrimSpace(target) == "" {
		result.Err = "missing tmux target"
		return result
	}
	if len(keys) == 0 {
		result.Err = "missing keys"
		return result
	}

	if host.Local {
		for i, key := range keys {
			result = runLocalTmux(ctx, host, target, []string{"send-keys", "-t", target, "--", key})
			if result.Err != "" {
				return result
			}
			if i < len(keys)-1 {
				time.Sleep(keySequenceDelay)
			}
		}
		return result
	}

	return runRemoteTmux(ctx, host, target, sendKeySequenceRemoteCommand(target, keys...))
}

func SendText(ctx context.Context, host config.Host, target string, text string, submit bool) SendResult {
	result := SendResult{Host: host, Target: target}
	if !host.Local && strings.TrimSpace(host.SSH) == "" {
		result.Err = "missing ssh target"
		return result
	}
	if strings.TrimSpace(target) == "" {
		result.Err = "missing tmux target"
		return result
	}
	if text == "" && !submit {
		result.Err = "missing text"
		return result
	}

	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	bufferName := sendTextBufferName()

	if host.Local {
		if text != "" {
			result := runLocalTmux(ctx, host, target, []string{"set-buffer", "-b", bufferName, "--", text})
			if result.Err != "" {
				return result
			}
			result = runLocalTmux(ctx, host, target, []string{"paste-buffer", "-b", bufferName, "-t", target})
			cleanup := runLocalTmux(ctx, host, target, []string{"delete-buffer", "-b", bufferName})
			if result.Err != "" {
				return result
			}
			if cleanup.Err != "" {
				return cleanup
			}
		}
		if !submit {
			return result
		}
		if text != "" {
			time.Sleep(pasteSubmitDelay)
		}
		return runLocalTmux(ctx, host, target, []string{"send-keys", "-t", target, "C-m"})
	}

	return runRemoteTmux(ctx, host, target, sendTextRemoteCommand(target, bufferName, text, submit))
}

func sendTextBufferName() string {
	return fmt.Sprintf("tmux-kanban-send-%d", time.Now().UnixNano())
}

func sendTextRemoteCommand(target string, bufferName string, text string, submit bool) string {
	commands := make([]string, 0, 4)
	if text != "" {
		commands = append(commands,
			"tmux set-buffer -b "+shellQuote(bufferName)+" -- "+shellQuote(text),
			"tmux paste-buffer -b "+shellQuote(bufferName)+" -t "+shellQuote(target),
			"tmux delete-buffer -b "+shellQuote(bufferName),
		)
	}
	if submit {
		if text != "" {
			commands = append(commands, pasteSubmitDelayCommand())
		}
		commands = append(commands, "tmux send-keys -t "+shellQuote(target)+" C-m")
	}
	return strings.Join(commands, " && ")
}

func pasteSubmitDelayCommand() string {
	return fmt.Sprintf("sleep %.2f", pasteSubmitDelay.Seconds())
}

func sendKeysRemoteCommand(target string, keys ...string) string {
	parts := []string{"tmux", "send-keys", "-t", shellQuote(target), "--"}
	for _, key := range keys {
		parts = append(parts, shellQuote(key))
	}
	return strings.Join(parts, " ")
}

func sendKeySequenceRemoteCommand(target string, keys ...string) string {
	commands := make([]string, 0, len(keys)*2)
	for i, key := range keys {
		commands = append(commands, sendKeysRemoteCommand(target, key))
		if i < len(keys)-1 {
			commands = append(commands, keySequenceDelayCommand())
		}
	}
	return strings.Join(commands, " && ")
}

func keySequenceDelayCommand() string {
	return fmt.Sprintf("sleep %.2f", keySequenceDelay.Seconds())
}

func runLocalTmux(ctx context.Context, host config.Host, target string, args []string) SendResult {
	result := SendResult{Host: host, Target: target}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "tmux", args...)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		result.Err = "send timed out"
		return result
	}
	if err != nil {
		result.Err = compactError(err, output)
		return result
	}

	return result
}

func runRemoteTmux(ctx context.Context, host config.Host, target string, remoteCommand string) SendResult {
	result := SendResult{Host: host, Target: target}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", host.SSH, remoteCommand)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		result.Err = "send timed out"
		return result
	}
	if err != nil {
		result.Err = compactError(err, output)
		return result
	}

	return result
}
