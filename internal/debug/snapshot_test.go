package debug

import (
	"encoding/json"
	"os"
	"testing"

	"tmux-kanban/internal/config"
)

func TestWriteSnapshotSerializesDiagnosticPackage(t *testing.T) {
	dir := t.TempDir()
	snapshot := Snapshot{
		Description: "FARI marked done while the pane still said Working",
		Config: NewConfigSummary(config.Config{
			Hosts: []config.Host{{Name: "local", Local: true}},
		}),
		Runtime: RuntimeState{
			ViewMode: "review",
			SessionStatuses: map[string]string{
				"local:$1": "need review",
			},
		},
		ReviewQueue: []ReviewItem{{
			SessionKey:  "local:$1",
			Host:        "local",
			SessionName: "agents",
			Agent:       "codex",
			Target:      "%1",
			Capture:     []string{"Do you want to allow this command?"},
		}},
	}

	path, err := WriteSnapshot(dir, snapshot)
	if err != nil {
		t.Fatalf("WriteSnapshot() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}

	var decoded Snapshot
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("snapshot JSON decode failed: %v", err)
	}
	if decoded.Version != 1 {
		t.Fatalf("version = %d, want 1", decoded.Version)
	}
	if decoded.Description != "FARI marked done while the pane still said Working" {
		t.Fatalf("description = %q, want diagnostic note", decoded.Description)
	}
	if len(decoded.ReviewQueue) != 1 || decoded.ReviewQueue[0].Capture[0] == "" {
		t.Fatalf("review queue = %#v, want captured review item", decoded.ReviewQueue)
	}
}
