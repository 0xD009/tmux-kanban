package main

import (
	"reflect"
	"strings"
	"testing"

	"tmux-kanban/internal/config"
	"tmux-kanban/internal/tmuxscan"
)

func TestChoiceKeysUsesRemoteMovementWhenSelectionKnown(t *testing.T) {
	screen := tmuxscan.AgentScreen{
		SelectedChoice: 0,
		Choices: []tmuxscan.AgentChoice{
			{Number: "1", Label: "Allow", Selected: true},
			{Number: "2", Label: "Deny"},
			{Number: "3", Label: "Always allow"},
		},
	}

	got := choiceKeys(screen, "3")
	want := []string{"Down", "Down", "C-m"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("choiceKeys() = %#v, want %#v", got, want)
	}
}

func TestChoiceKeysFallsBackToMovementFromFirstChoice(t *testing.T) {
	screen := tmuxscan.AgentScreen{
		SelectedChoice: -1,
		Choices: []tmuxscan.AgentChoice{
			{Number: "1", Label: "Allow"},
			{Number: "2", Label: "Deny"},
			{Number: "3", Label: "Always allow"},
		},
	}

	got := choiceKeys(screen, "3")
	want := []string{"Down", "Down", "C-m"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("choiceKeys() = %#v, want %#v", got, want)
	}
}

func TestChoiceKeysCanMoveUnnumberedChoiceWhenSelectionUnknown(t *testing.T) {
	screen := tmuxscan.AgentScreen{
		SelectedChoice: -1,
		Choices: []tmuxscan.AgentChoice{
			{Label: "Allow"},
			{Label: "Deny"},
		},
	}

	got := choiceKeys(screen, "2")
	want := []string{"Down", "C-m"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("choiceKeys() = %#v, want %#v", got, want)
	}
}

func TestSelectedAgentTargetFallsThroughFromSession(t *testing.T) {
	m := model{
		hosts: []hostState{{
			host: config.Host{Name: "local", SSH: "local"},
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{{
				ID:   "$1",
				Name: "agents",
				Windows: []tmuxscan.Window{
					{
						ID:    "@1",
						Index: "0",
						Panes: []tmuxscan.Pane{
							{ID: "%1", Index: "0", Agent: tmuxscan.AgentCodex},
						},
					},
					{
						ID:     "@2",
						Index:  "1",
						Active: true,
						Panes: []tmuxscan.Pane{
							{ID: "%2", Index: "0", Active: true, Agent: tmuxscan.AgentClaude},
						},
					},
				},
			}}},
			loaded: true,
		}},
		expanded: map[string]bool{"host:0": true},
		cursor:   0,
	}

	target, ok := m.selectedAgentTarget()
	if !ok {
		t.Fatalf("selectedAgentTarget() ok = false, want true")
	}
	if target.target != "%2" {
		t.Fatalf("target = %q, want %%2", target.target)
	}
	if target.agent != tmuxscan.AgentClaude {
		t.Fatalf("agent = %q, want %q", target.agent, tmuxscan.AgentClaude)
	}
}

func TestAgentTargetDisplayLabelUsesHumanReadablePaneLocation(t *testing.T) {
	m := model{
		hosts: []hostState{{
			host: config.Host{Name: "local", SSH: "local"},
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{{
				ID:   "$1",
				Name: "agents",
				Windows: []tmuxscan.Window{{
					ID:    "@1",
					Index: "2",
					Panes: []tmuxscan.Pane{{
						ID:    "%1",
						Index: "3",
						Agent: tmuxscan.AgentCodex,
					}},
				}},
			}}},
			loaded: true,
		}},
	}
	target := selectedAgentTarget{
		hostIndex: 0,
		target:    "%1",
		agent:     tmuxscan.AgentCodex,
	}

	label := m.agentTargetDisplayLabel(target)
	if label != "local/agents:2.3 (codex)" {
		t.Fatalf("label = %q, want local/agents:2.3 (codex)", label)
	}
}

func TestSelectedPaneAgentTargetCarriesDisplayLabel(t *testing.T) {
	m := model{
		hosts: []hostState{{
			host: config.Host{Name: "local", SSH: "local"},
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{{
				ID:   "$1",
				Name: "agents",
				Windows: []tmuxscan.Window{{
					ID:    "@1",
					Index: "2",
					Panes: []tmuxscan.Pane{{
						ID:    "%1",
						Index: "3",
						Agent: tmuxscan.AgentCodex,
					}},
				}},
			}}},
			loaded: true,
		}},
		expanded: map[string]bool{
			"host:0":                      true,
			"host:0:session:$1":           true,
			"host:0:session:$1:window:@1": true,
		},
		cursor: 2,
	}

	target, ok := m.selectedAgentTarget()
	if !ok {
		t.Fatalf("selectedAgentTarget() ok = false, want true")
	}
	if target.label != "agents:2.3 (codex)" {
		t.Fatalf("target label = %q, want agents:2.3 (codex)", target.label)
	}
	if label := m.agentTargetDisplayLabel(target); label != "local/agents:2.3 (codex)" {
		t.Fatalf("display label = %q, want local/agents:2.3 (codex)", label)
	}
}

func TestComposeInputPrefixDoesNotRepeatTarget(t *testing.T) {
	label := composeInputPrefix(composeState{target: "%1", label: "local/agents:2.3 (codex)"})
	if strings.Contains(label, "%1") {
		t.Fatalf("label = %q, want no tmux pane id", label)
	}
	if label != "" {
		t.Fatalf("label = %q, want empty message input prefix", label)
	}
}

func TestComposeInputPrefixDoesNotExposeRawPaneIDWhenLabelMissing(t *testing.T) {
	label := composeInputPrefix(composeState{target: "%1"})
	if strings.Contains(label, "%1") {
		t.Fatalf("label = %q, want no raw tmux pane id", label)
	}
}

func TestSelectedAgentTargetFallsThroughFromWindow(t *testing.T) {
	m := model{
		hosts: []hostState{{
			host: config.Host{Name: "local", SSH: "local"},
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{{
				ID:   "$1",
				Name: "agents",
				Windows: []tmuxscan.Window{{
					ID:    "@1",
					Index: "0",
					Panes: []tmuxscan.Pane{
						{ID: "%1", Index: "0", Agent: tmuxscan.AgentCodex},
					},
				}},
			}}},
			loaded: true,
		}},
		expanded: map[string]bool{
			"host:0":            true,
			"host:0:session:$1": true,
		},
		cursor: 1,
	}

	target, ok := m.selectedAgentTarget()
	if !ok {
		t.Fatalf("selectedAgentTarget() ok = false, want true")
	}
	if target.target != "%1" {
		t.Fatalf("target = %q, want %%1", target.target)
	}
	if target.agent != tmuxscan.AgentCodex {
		t.Fatalf("agent = %q, want %q", target.agent, tmuxscan.AgentCodex)
	}
}

func TestCycleSelectedSessionStatusFromPane(t *testing.T) {
	m := model{
		hosts: []hostState{{
			host: config.Host{Name: "local", SSH: "local"},
			snapshot: tmuxscan.Snapshot{Sessions: []tmuxscan.Session{{
				ID:   "$1",
				Name: "agents",
				Windows: []tmuxscan.Window{{
					ID:    "@1",
					Index: "0",
					Panes: []tmuxscan.Pane{
						{ID: "%1", Index: "0", Agent: tmuxscan.AgentCodex},
					},
				}},
			}}},
			loaded: true,
		}},
		expanded: map[string]bool{
			"host:0":                      true,
			"host:0:session:$1":           true,
			"host:0:session:$1:window:@1": true,
		},
		statuses: map[string]sessionStatus{},
		cursor:   2,
	}

	m.cycleSelectedSessionStatus()
	ref, ok := m.selectedSessionRef()
	if !ok {
		t.Fatalf("selectedSessionRef() ok = false, want true")
	}
	if got := m.sessionStatusForKey(ref.Key); got != sessionWorking {
		t.Fatalf("status = %q, want %q", got, sessionWorking)
	}

	m.cycleSelectedSessionStatus()
	if got := m.sessionStatusForKey(ref.Key); got != sessionNeedReview {
		t.Fatalf("status = %q, want %q", got, sessionNeedReview)
	}

	m.cycleSelectedSessionStatus()
	if got := m.sessionStatusForKey(ref.Key); got != sessionDone {
		t.Fatalf("status = %q, want %q", got, sessionDone)
	}

	m.cycleSelectedSessionStatus()
	if got := m.sessionStatusForKey(ref.Key); got != sessionIdle {
		t.Fatalf("status = %q, want %q", got, sessionIdle)
	}
}
