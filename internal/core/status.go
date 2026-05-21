package core

import (
	"strings"
)

type AppState struct {
	Hosts         []HostState
	Statuses      map[string]SessionStatus
	StatusStreaks map[string]StatusStreak
	ReviewTargets map[string]SelectedTarget
	ReviewQueue   ReviewQueue
}

type HostState struct {
	Name    string
	Loading bool
	Loaded  bool
	Error   string
}

type SelectedTarget struct {
	Key       string
	HostIndex int
	Target    string
	Label     string
	Agent     string
}

type SessionStatus string

type StatusStreak struct {
	Status SessionStatus
	Count  int
}

const (
	StatusIdle       SessionStatus = "idle"
	StatusWorking    SessionStatus = "working"
	StatusNeedReview SessionStatus = "need review"
	StatusDone       SessionStatus = "done"
)

func NormalizeStatus(status SessionStatus) SessionStatus {
	switch status {
	case StatusIdle, StatusWorking, StatusNeedReview, StatusDone:
		return status
	default:
		return ""
	}
}

func ParseStatus(value string) (SessionStatus, bool) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.Join(strings.Fields(normalized), "-")
	switch normalized {
	case "idle":
		return StatusIdle, true
	case "working", "work":
		return StatusWorking, true
	case "need-review", "needs-review", "review":
		return StatusNeedReview, true
	case "done", "complete", "completed", "finish", "finished":
		return StatusDone, true
	default:
		return "", false
	}
}

func StatusLabel(status SessionStatus) string {
	switch status {
	case StatusIdle:
		return "idle"
	case StatusWorking:
		return "working"
	case StatusNeedReview:
		return "need review"
	case StatusDone:
		return "done"
	default:
		return "unknown"
	}
}

func NextManualStatus(status SessionStatus) SessionStatus {
	switch NormalizeStatus(status) {
	case StatusIdle:
		return StatusWorking
	case StatusWorking:
		return StatusNeedReview
	case StatusNeedReview:
		return StatusDone
	case StatusDone:
		return StatusIdle
	default:
		return StatusIdle
	}
}

func ApplyPolledStatus(current SessionStatus, hasCurrent bool, polled SessionStatus) SessionStatus {
	next := NormalizeStatus(polled)
	if hasCurrent {
		current = NormalizeStatus(current)
	}
	if hasCurrent && current == StatusDone && next == StatusIdle {
		return StatusDone
	}
	if hasCurrent && current == StatusWorking && next == StatusIdle {
		return StatusDone
	}
	return next
}

func StatusFromAgentScreen(needsReview bool, busy bool, idle bool, choices int) (SessionStatus, bool) {
	switch {
	case needsReview:
		return StatusNeedReview, true
	case busy:
		return StatusWorking, true
	case idle || choices > 0:
		return StatusIdle, true
	default:
		return "", false
	}
}

func StatusPriority(status SessionStatus) int {
	switch NormalizeStatus(status) {
	case StatusDone:
		return 4
	case StatusNeedReview:
		return 3
	case StatusWorking:
		return 2
	case StatusIdle:
		return 1
	default:
		return 0
	}
}
