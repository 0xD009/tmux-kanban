package hermeslog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Entry struct {
	At           time.Time         `json:"at"`
	Flow         string            `json:"flow"`
	Event        string            `json:"event"`
	Mode         string            `json:"mode,omitempty"`
	Trigger      string            `json:"trigger,omitempty"`
	Status       string            `json:"status,omitempty"`
	Host         string            `json:"host,omitempty"`
	Session      string            `json:"session,omitempty"`
	Target       string            `json:"target,omitempty"`
	Agent        string            `json:"agent,omitempty"`
	Scope        string            `json:"scope,omitempty"`
	Conditions   map[string]string `json:"conditions,omitempty"`
	Advice       string            `json:"advice,omitempty"`
	ParsedAction string            `json:"parsed_action,omitempty"`
	Choice       string            `json:"choice,omitempty"`
	Message      string            `json:"message,omitempty"`
	Accepted     bool              `json:"accepted,omitempty"`
	Modified     bool              `json:"modified,omitempty"`
	ModifiedPath string            `json:"modified_path,omitempty"`
	Error        string            `json:"error,omitempty"`
}

func Append(path string, entry Entry) (string, error) {
	path = ResolvePath(path)
	if path == "" {
		return "", fmt.Errorf("resolve Hermes work log path")
	}
	if entry.At.IsZero() {
		entry.At = time.Now()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return path, fmt.Errorf("create Hermes work log dir: %w", err)
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return path, fmt.Errorf("encode Hermes work log entry: %w", err)
	}
	data = append(data, '\n')
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return path, fmt.Errorf("open Hermes work log: %w", err)
	}
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return path, fmt.Errorf("write Hermes work log: %w", err)
	}
	return path, nil
}

func ResolvePath(path string) string {
	path = expandPath(path)
	if strings.TrimSpace(path) != "" {
		return path
	}
	return DefaultPath()
}

func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return filepath.Join(os.TempDir(), "tmux-kanban", "hermes-worklog.jsonl")
	}
	return filepath.Join(home, ".local", "state", "tmux-kanban", "hermes-worklog.jsonl")
}

func expandPath(path string) string {
	path = strings.TrimSpace(os.ExpandEnv(path))
	if path == "" {
		return ""
	}
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}
