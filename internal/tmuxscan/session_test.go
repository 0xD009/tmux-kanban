package tmuxscan

import "testing"

func TestLocalNewSessionArgsBuildsShellCommand(t *testing.T) {
	got := localNewSessionArgs("tmux-kanban-main", "codex", "--profile", "kanban agent")
	want := []string{"tmux", "new-session", "-d", "-s", "tmux-kanban-main", "'codex' '--profile' 'kanban agent'"}
	if len(got) != len(want) {
		t.Fatalf("args length = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("arg[%d] = %q, want %q; args=%#v", i, got[i], want[i], got)
		}
	}
}

func TestRemoteNewSessionCommandQuotesSessionAndCommand(t *testing.T) {
	got := remoteNewSessionCommand("main session", "claude", "--model", "sonnet")
	want := "tmux new-session -d -s 'main session' ''\\''claude'\\'' '\\''--model'\\'' '\\''sonnet'\\'''"
	if got != want {
		t.Fatalf("remote command = %q, want %q", got, want)
	}
}
