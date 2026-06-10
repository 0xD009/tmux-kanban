package main

import "strings"

type commandCandidate struct {
	Text        string
	Display     string
	Description string
}

func (c commandCandidate) Label() string {
	if c.Display != "" {
		return c.Display
	}
	return c.Text
}

func (c commandCandidate) NeedsValue() bool {
	return strings.HasSuffix(c.Text, " ")
}

func commandCandidates(input string) []commandCandidate {
	query := strings.ToLower(strings.TrimSpace(input))
	catalog := commandCandidateCatalog()
	if query == "" {
		return catalog
	}

	candidates := make([]commandCandidate, 0, len(catalog))
	for _, candidate := range catalog {
		if strings.HasPrefix(strings.ToLower(candidate.Text), query) || strings.HasPrefix(strings.ToLower(candidate.Label()), query) {
			candidates = append(candidates, candidate)
		}
	}
	return candidates
}

func commandCandidateCatalog() []commandCandidate {
	return []commandCandidate{
		{Text: "help", Description: "show commands"},
		{Text: "refresh", Description: "scan tmux sessions"},
		{Text: "settings", Description: "show runtime settings"},
		{Text: "view tree", Description: "open tree view"},
		{Text: "session open ", Display: "session open name", Description: "create tmux session"},
		{Text: "session close here", Description: "prepare selected session close"},
		{Text: "session close ", Display: "session close host/session", Description: "prepare session close"},
		{Text: "session close confirm ", Display: "session close confirm host/session", Description: "confirm prepared close"},
		{Text: "memory update pane", Description: "update pane memory with Hermes"},
		{Text: "memory update session", Description: "update session memory with Hermes"},
		{Text: "memory update host", Description: "update host memory with Hermes"},
		{Text: "memory update global", Description: "update global memory with Hermes"},
		{Text: "mesh status", Description: "show mesh settings"},
		{Text: "mesh ", Display: "mesh on/off", Description: "toggle agent mesh"},
		{Text: "mesh shared ", Display: "mesh shared on/off", Description: "toggle shared names"},
		{Text: "mesh default codex", Description: "default mesh agent"},
		{Text: "mesh default claude", Description: "default mesh agent"},
		{Text: "mesh mail ", Display: "mesh mail on/off", Description: "toggle mesh mail"},
		{Text: "mesh skill-root ", Description: "set mesh skill root"},
		{Text: "mesh memory ", Description: "set mesh memory root"},
		{Text: "set qq ", Display: "set qq on/off", Description: "toggle QQ notify"},
		{Text: "set auto_review_audit_qq ", Display: "set auto_review_audit_qq off/uncertain/all", Description: "configure auto review audit copy"},
		{Text: "set terminal_review ", Display: "set terminal_review on/off", Description: "toggle terminal review alert"},
		{Text: "set hermes ", Display: "set hermes on/off", Description: "toggle Hermes"},
		{Text: "set hermes.auto_review ", Display: "set hermes.auto_review on/off", Description: "toggle auto review"},
		{Text: "set hermes.auto_review all ", Display: "set hermes.auto_review all on/off", Description: "set global auto review"},
		{Text: "set hermes.done_advice ", Display: "set hermes.done_advice on/off", Description: "ask Hermes after done"},
		{Text: "set hermes.auto_done ", Display: "set hermes.auto_done on/off", Description: "auto send done advice"},
		{Text: "set hermes.auto_done all ", Display: "set hermes.auto_done all on/off", Description: "set global done auto"},
		{Text: "set hermes.auto_done here ", Display: "set hermes.auto_done here on/off", Description: "scope to selected session"},
		{Text: "set hermes.idle_advice ", Display: "set hermes.idle_advice on/off", Description: "ask Hermes after idle"},
		{Text: "set hermes.auto_idle ", Display: "set hermes.auto_idle on/off", Description: "auto send idle advice"},
		{Text: "set hermes.auto_idle all ", Display: "set hermes.auto_idle all on/off", Description: "set global idle auto"},
		{Text: "set hermes.auto_idle here ", Display: "set hermes.auto_idle here on/off", Description: "scope to selected session"},
		{Text: "set mesh.mail ", Display: "set mesh.mail on/off", Description: "toggle mesh mail"},
		{Text: "set mesh.memory_root ", Description: "set mesh memory root"},
		{Text: "status idle", Description: "mark selected idle"},
		{Text: "status working", Description: "mark selected working"},
		{Text: "status need-review", Description: "mark selected needs review"},
		{Text: "status done", Description: "mark selected done"},
		{Text: "notify ", Description: "send review notification"},
		{Text: "snapshot", Description: "save debug snapshot"},
	}
}
