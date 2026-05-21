package tmuxscan

import (
	"path/filepath"
	"testing"

	"tmux-kanban/internal/config"
)

func TestParseTmuxOutput(t *testing.T) {
	output := "" +
		"S\t$1\tmain\t2\t1\n" +
		"W\t$1\t@1\t0\tshell\t1\n" +
		"W\t$1\t@2\t1\tlogs\t0\n" +
		"P\t$1\t@1\t%1\t0\t100\tzsh\t/home/user\t1\n" +
		"P\t$1\t@2\t%2\t0\t200\ttail\t/var/log\t1\n"

	sessions, err := Parse(output)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	if sessions[0].Name != "main" {
		t.Fatalf("session name = %q, want main", sessions[0].Name)
	}
	if len(sessions[0].Windows) != 2 {
		t.Fatalf("got %d windows, want 2", len(sessions[0].Windows))
	}
	if sessions[0].Windows[1].Panes[0].Command != "tail" {
		t.Fatalf("pane command = %q, want tail", sessions[0].Windows[1].Panes[0].Command)
	}
	if sessions[0].Windows[0].Panes[0].PID != "100" {
		t.Fatalf("pane pid = %q, want 100", sessions[0].Windows[0].Panes[0].PID)
	}
}

func TestParseDetectsCodexProcess(t *testing.T) {
	output := "" +
		"S\t$1\tagents\t1\t0\n" +
		"W\t$1\t@1\t0\tmain\t1\n" +
		"P\t$1\t@1\t%1\t0\t100\tzsh\t/home/user\t1\n" +
		"R\t100\t101\tnode\tnode /usr/local/bin/codex app-server\n"

	sessions, err := Parse(output)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	pane := sessions[0].Windows[0].Panes[0]
	if pane.Agent != AgentCodex {
		t.Fatalf("pane agent = %q, want %q", pane.Agent, AgentCodex)
	}
}

func TestParseDetectsClaudeCodeProcess(t *testing.T) {
	output := "" +
		"S\t$1\tagents\t1\t0\n" +
		"W\t$1\t@1\t0\tmain\t1\n" +
		"P\t$1\t@1\t%1\t0\t200\tzsh\t/home/user\t1\n" +
		"R\t200\t201\tnode\tnode /home/user/.npm/_npx/123/node_modules/@anthropic-ai/claude-code/cli.js\n"

	sessions, err := Parse(output)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	pane := sessions[0].Windows[0].Panes[0]
	if pane.Agent != AgentClaude {
		t.Fatalf("pane agent = %q, want %q", pane.Agent, AgentClaude)
	}
}

func TestDetectAgentIgnoresPlainArguments(t *testing.T) {
	pane := Pane{
		Command: "grep",
		Processes: []Process{
			{PID: "10", Command: "grep", Args: "grep codex notes.txt"},
		},
	}

	if agent := DetectAgent(pane); agent != AgentNone {
		t.Fatalf("agent = %q, want none", agent)
	}
}

func TestNormalizeCaptureLines(t *testing.T) {
	lines := normalizeCaptureLines("\n\nhello\nworld\n\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}
	if lines[0] != "hello" {
		t.Fatalf("first line = %q, want hello", lines[0])
	}
	if lines[1] != "world" {
		t.Fatalf("last line = %q, want world", lines[1])
	}
}

func TestAttachCommandUsesLocalTmuxForLocalHost(t *testing.T) {
	cmd := AttachCommand(config.Host{Name: "local", Local: true}, "main:0")
	if filepath.Base(cmd.Path) != "tmux" {
		t.Fatalf("command path = %q, want tmux", cmd.Path)
	}
	want := []string{"tmux", "attach-session", "-t", "main:0"}
	if len(cmd.Args) != len(want) {
		t.Fatalf("args = %#v, want %#v", cmd.Args, want)
	}
	for i := range want {
		if cmd.Args[i] != want[i] {
			t.Fatalf("args = %#v, want %#v", cmd.Args, want)
		}
	}
}

func TestAttachCommandUsesSSHForRemoteHost(t *testing.T) {
	cmd := AttachCommand(config.Host{Name: "gpu", SSH: "gpu-a"}, "main")
	if filepath.Base(cmd.Path) != "ssh" {
		t.Fatalf("command path = %q, want ssh", cmd.Path)
	}
	if len(cmd.Args) < 4 || cmd.Args[1] != "-t" || cmd.Args[2] != "gpu-a" {
		t.Fatalf("args = %#v, want ssh -t gpu-a ...", cmd.Args)
	}
}
