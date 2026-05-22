package hermeslog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAppendWritesJSONL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs", "hermes.jsonl")
	written, err := Append(path, Entry{
		Flow:         "review",
		Event:        "auto_action",
		Mode:         "auto",
		Host:         "local",
		Session:      "agents",
		ParsedAction: "choose",
		Choice:       "1",
		Accepted:     true,
	})
	if err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if written != path {
		t.Fatalf("path = %q, want %q", written, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var entry Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("Unmarshal() error = %v; data = %q", err, string(data))
	}
	if entry.Flow != "review" || entry.ParsedAction != "choose" || !entry.Accepted {
		t.Fatalf("entry = %#v", entry)
	}
}
