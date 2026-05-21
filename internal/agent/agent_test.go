package agent

import "testing"

func TestDetectCodexThroughWrapper(t *testing.T) {
	got := Detect("node", []Process{{Command: "node", Args: "node /usr/bin/codex exec"}})
	if got != Codex {
		t.Fatalf("Detect() = %q, want codex", got)
	}
}

func TestAnalyzeScreenDelegatesChoiceParsing(t *testing.T) {
	screen := AnalyzeScreen([]string{
		"Do you want to allow this command?",
		"❯ 1. Allow",
		"  2. Deny",
	})
	if !screen.NeedsReview || len(screen.Choices) != 2 {
		t.Fatalf("screen = %#v, want needs review with two choices", screen)
	}
}
