package tmuxscan

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"tmux-kanban/internal/config"
)

const remoteListCommand = `tmux list-sessions -F 'S\t#{session_id}\t#{session_name}\t#{session_windows}\t#{session_attached}' 2>/dev/null || true; tmux list-windows -a -F 'W\t#{session_id}\t#{window_id}\t#{window_index}\t#{window_name}\t#{window_active}' 2>/dev/null || true; tmux list-panes -a -F 'P\t#{session_id}\t#{window_id}\t#{pane_id}\t#{pane_index}\t#{pane_pid}\t#{pane_current_command}\t#{pane_current_path}\t#{pane_active}' 2>/dev/null || true; pane_pids=$(tmux list-panes -a -F '#{pane_pid}' 2>/dev/null | tr '\n' ' '); if [ -n "$pane_pids" ]; then ps -eo pid=,ppid=,comm=,args= 2>/dev/null | awk -v roots="$pane_pids" 'BEGIN { n=split(roots, r, " "); for (i=1; i<=n; i++) if (r[i] != "") root[r[i]]=1 } { pid=$1; ppid=$2; comm=$3; line=$0; sub(/^[[:space:]]*[0-9]+[[:space:]]+[0-9]+[[:space:]]+[^[:space:]]+[[:space:]]*/, "", line); parent[pid]=ppid; command[pid]=comm; args[pid]=line; seen[pid]=1 } END { for (pid in seen) { cur=pid; depth=0; found=""; while (cur != "" && cur != "0" && depth < 64) { if (cur in root) { found=cur; break } cur=parent[cur]; depth++ } if (found != "" && pid != found) { gsub(/\t/, " ", args[pid]); gsub(/\r/, " ", args[pid]); print "R\t" found "\t" pid "\t" command[pid] "\t" args[pid] } } }'; fi`

func AttachCommand(host config.Host, target string) *exec.Cmd {
	if host.Local {
		return exec.Command("tmux", "attach-session", "-t", target)
	}

	remoteCommand := "tmux attach-session -t " + shellQuote(target)
	return exec.Command("ssh", "-t", host.SSH, remoteCommand)
}

func CapturePane(ctx context.Context, host config.Host, target string, height int) Capture {
	capture := Capture{Host: host, Target: target, CapturedAt: time.Now()}
	if !host.Local && strings.TrimSpace(host.SSH) == "" {
		capture.Err = "missing ssh target"
		return capture
	}
	if strings.TrimSpace(target) == "" {
		capture.Err = "missing tmux target"
		return capture
	}
	if height <= 0 {
		height = 20
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := captureCommand(ctx, host, target, height)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		capture.Err = "preview timed out"
		return capture
	}
	if err != nil {
		capture.Err = compactError(err, output)
		return capture
	}

	capture.Lines = normalizeCaptureLines(string(output))
	return capture
}

func listCommand(ctx context.Context, host config.Host) *exec.Cmd {
	if host.Local {
		return exec.CommandContext(ctx, "sh", "-lc", remoteListCommand)
	}
	return exec.CommandContext(ctx, "ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", host.SSH, remoteListCommand)
}

func captureCommand(ctx context.Context, host config.Host, target string, height int) *exec.Cmd {
	if host.Local {
		return exec.CommandContext(ctx, "tmux", "capture-pane", "-e", "-p", "-S", fmt.Sprintf("-%d", height), "-t", target)
	}

	remoteCommand := fmt.Sprintf("tmux capture-pane -e -p -S -%d -t %s", height, shellQuote(target))
	return exec.CommandContext(ctx, "ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", host.SSH, remoteCommand)
}
