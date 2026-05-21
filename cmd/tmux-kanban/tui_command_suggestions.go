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
		{Text: "view review", Description: "open review queue"},
		{Text: "view main", Description: "open main room"},
		{Text: "main start", Description: "open main room"},
		{Text: "main hide", Description: "hide main room"},
		{Text: "main codex", Description: "use Codex main agent"},
		{Text: "main claude", Description: "use Claude main agent"},
		{Text: "main status", Description: "show main settings"},
		{Text: "main host ", Description: "set main host"},
		{Text: "main session ", Description: "set main session"},
		{Text: "main command ", Description: "set main command"},
		{Text: "mesh status", Description: "show mesh settings"},
		{Text: "mesh ", Display: "mesh on/off", Description: "toggle agent mesh"},
		{Text: "mesh shared ", Display: "mesh shared on/off", Description: "toggle shared names"},
		{Text: "mesh default codex", Description: "default mesh agent"},
		{Text: "mesh default claude", Description: "default mesh agent"},
		{Text: "mesh mail ", Display: "mesh mail on/off", Description: "toggle mesh mail"},
		{Text: "mesh skill-root ", Description: "set mesh skill root"},
		{Text: "mesh memory ", Description: "set mesh memory root"},
		{Text: "set qq ", Display: "set qq on/off", Description: "toggle QQ notify"},
		{Text: "set hermes ", Display: "set hermes on/off", Description: "toggle Hermes"},
		{Text: "set hermes.auto_review ", Display: "set hermes.auto_review on/off", Description: "toggle auto review"},
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
