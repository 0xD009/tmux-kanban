package tmux

import (
	"path/filepath"
	"testing"

	"tmux-kanban/internal/config"
)

func TestDefaultClientAttachCommand(t *testing.T) {
	client := DefaultClient{}
	cmd := client.AttachCommand(config.Host{Name: "local", Local: true}, "main")
	if filepath.Base(cmd.Path) != "tmux" {
		t.Fatalf("command path = %q, want tmux", cmd.Path)
	}
	if len(cmd.Args) != 4 || cmd.Args[1] != "attach-session" || cmd.Args[3] != "main" {
		t.Fatalf("args = %#v, want tmux attach-session -t main", cmd.Args)
	}
}
