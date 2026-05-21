package main

import "strings"

const mainSessionSkillName = "tmux-kanban-review"

var mainSessionCLIAbilities = []string{
	"capabilities",
	"review-list",
	"capture",
	"choose",
	"send",
	"send-keys",
	"snapshot",
	"notify-review",
}

func mainSessionCapabilityBadges() string {
	return " [conductor] [skill] [cli]"
}

func mainSessionCapabilityShort() string {
	return "conductor | skill+JSON CLI"
}

func mainSessionCapabilitySummary() string {
	return "conductor | skill " + mainSessionSkillName + " | CLI " + strings.Join(mainSessionCLIAbilities, "/")
}
