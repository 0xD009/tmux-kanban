package tmuxscan

import (
	"regexp"
	"strings"
)

type AgentScreen struct {
	Choices        []AgentChoice
	SelectedChoice int
	Idle           bool
	Busy           bool
	NeedsReview    bool
}

type AgentChoice struct {
	Number   string
	Label    string
	Selected bool
}

type parsedChoice struct {
	Choice AgentChoice
	Line   int
}

var (
	ansiEscapePattern     = regexp.MustCompile(`\x1b(?:\[[0-9;?]*[ -/]*[@-~]|\][^\a]*(?:\a|\x1b\\))`)
	numberedChoicePattern = regexp.MustCompile(`^(\d{1,2})[.)]\s+(.+)$`)
	bracketChoicePattern  = regexp.MustCompile(`^\[(\d{1,2})\]\s+(.+)$`)
	labeledChoicePattern  = regexp.MustCompile(`^(\d{1,2})\s*[-:]\s+(.+)$`)
	checkboxPattern       = regexp.MustCompile(`^\[([ xX✓])\]\s+(.+)$`)
)

func AnalyzeAgentScreen(lines []string) AgentScreen {
	screen := AgentScreen{SelectedChoice: -1}
	cleanLines := make([]string, 0, len(lines))
	choices := make([]parsedChoice, 0)
	choiceBlockActive := false
	planBlockActive := false

	for _, line := range lines {
		clean := cleanScreenLine(line)
		if clean == "" {
			choiceBlockActive = false
			planBlockActive = false
			continue
		}
		lineIndex := len(cleanLines)
		cleanLines = append(cleanLines, clean)

		if looksPlanTodoHeader(clean) {
			choiceBlockActive = false
			planBlockActive = true
			continue
		}
		if planBlockActive {
			if looksPlanTodoItem(clean) {
				continue
			}
			planBlockActive = false
		}

		if choice, ok := parseChoiceLine(clean); ok {
			choices = append(choices, parsedChoice{Choice: choice, Line: lineIndex})
			choiceBlockActive = true
			continue
		}
		if choiceBlockActive && looksLikeUnmarkedChoiceLabel(clean) {
			choices = append(choices, parsedChoice{
				Choice: AgentChoice{Label: clean},
				Line:   lineIndex,
			})
			continue
		}
		choiceBlockActive = false

		if looksIdle(clean) {
			screen.Idle = true
		}
	}

	currentStart := currentScreenStart(cleanLines, choices)
	screen.Busy = currentScreenBusy(cleanLines, currentStart)
	for _, choice := range choices {
		if choice.Line < currentStart {
			continue
		}
		if choice.Choice.Selected {
			screen.SelectedChoice = len(screen.Choices)
		}
		screen.Choices = append(screen.Choices, choice.Choice)
	}
	if len(screen.Choices) > 0 {
		screen.Idle = true
	}
	if screen.Busy && len(screen.Choices) == 0 {
		screen.Idle = false
	}
	screen.NeedsReview = currentScreenNeedsReview(cleanLines, choices, currentStart)

	return screen
}

func currentScreenBusy(lines []string, currentStart int) bool {
	if len(lines) == 0 {
		return false
	}

	tailStart := len(lines) - 12
	if tailStart < 0 {
		tailStart = 0
	}

	for _, line := range lines[tailStart:] {
		if looksActiveBackgroundTerminal(line) || looksBusy(line) {
			return true
		}
	}

	start := tailStart
	if currentStart > tailStart {
		start = currentStart
	}

	for _, line := range lines[start:] {
		if looksBusy(line) {
			return true
		}
	}
	return false
}

func currentScreenStart(lines []string, choices []parsedChoice) int {
	tailStart := len(lines) - 10
	if tailStart < 0 {
		tailStart = 0
	}
	currentStart := tailStart
	for _, choice := range choices {
		if choice.Line >= tailStart && choiceIsCurrentIdlePrompt(choice.Choice) {
			currentStart = choice.Line
		}
	}
	return currentStart
}

func currentScreenNeedsReview(lines []string, choices []parsedChoice, currentStart int) bool {
	hasReviewPrompt := false
	for _, line := range lines[currentStart:] {
		if looksNeedsReview(line) {
			hasReviewPrompt = true
			break
		}
	}

	currentChoices := 0
	hasSelectedReviewChoice := false
	for _, choice := range choices {
		if choice.Line >= currentStart {
			currentChoices++
			if choice.Choice.Selected && selectedChoiceNeedsReview(choice.Choice) {
				hasSelectedReviewChoice = true
			}
		}
	}
	return (hasReviewPrompt && currentChoices > 0) || hasSelectedReviewChoice
}

func cleanScreenLine(line string) string {
	line = ansiEscapePattern.ReplaceAllString(line, "")
	line = strings.ReplaceAll(line, "\r", "")
	line = strings.TrimSpace(line)
	line = strings.Trim(line, "│")
	return strings.TrimSpace(line)
}

func parseChoiceLine(line string) (AgentChoice, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return AgentChoice{}, false
	}

	selected := false
	for _, marker := range []string{"❯", "›", "▸", "▶", "➜", "→", ">", "*", "●", "◉", "◦", "○", "-"} {
		if strings.HasPrefix(line, marker) {
			selected = marker != "-" && marker != "○" && marker != "◦"
			line = strings.TrimSpace(strings.TrimPrefix(line, marker))
			break
		}
	}

	if match := checkboxPattern.FindStringSubmatch(line); len(match) == 3 {
		return AgentChoice{
			Label:    strings.TrimSpace(match[2]),
			Selected: selected || match[1] != " ",
		}, true
	}

	if match := numberedChoicePattern.FindStringSubmatch(line); len(match) == 3 {
		return AgentChoice{
			Number:   match[1],
			Label:    strings.TrimSpace(match[2]),
			Selected: selected,
		}, true
	}

	if match := bracketChoicePattern.FindStringSubmatch(line); len(match) == 3 {
		return AgentChoice{
			Number:   match[1],
			Label:    strings.TrimSpace(match[2]),
			Selected: selected,
		}, true
	}

	if match := labeledChoicePattern.FindStringSubmatch(line); len(match) == 3 {
		return AgentChoice{
			Number:   match[1],
			Label:    strings.TrimSpace(match[2]),
			Selected: selected,
		}, true
	}

	if selected && looksLikeMenuLabel(line) {
		return AgentChoice{
			Label:    line,
			Selected: true,
		}, true
	}

	return AgentChoice{}, false
}

func looksLikeUnmarkedChoiceLabel(line string) bool {
	if !looksLikeMenuLabel(line) {
		return false
	}
	lower := strings.ToLower(strings.TrimSpace(line))
	for _, reject := range []string{" · ", "~/", "gpt-", "claude ", "tokens", "cwd:", "model:"} {
		if strings.Contains(lower, reject) {
			return false
		}
	}
	if choiceNeedsReview(AgentChoice{Label: line}) {
		return true
	}
	for _, exact := range []string{
		"yes",
		"no",
		"allow",
		"deny",
		"cancel",
		"continue",
		"approve",
		"reject",
		"new task",
		"recent sessions",
	} {
		if lower == exact {
			return true
		}
	}
	return false
}

func looksLikeMenuLabel(line string) bool {
	if line == "" || len([]rune(line)) > 96 {
		return false
	}

	lower := strings.ToLower(line)
	for _, prefix := range []string{"press ", "ctrl", "esc", "enter ", "shift", "tab ", "q ", "q:"} {
		if strings.HasPrefix(lower, prefix) {
			return false
		}
	}

	return true
}

func looksPlanTodoHeader(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))
	lower = strings.Trim(lower, "#:：- ")
	switch lower {
	case "plan", "todo", "todos", "to do", "tasks", "task list", "checklist", "current plan", "implementation plan":
		return true
	default:
		return strings.HasPrefix(lower, "plan ") || strings.HasPrefix(lower, "todo ") || strings.HasPrefix(lower, "tasks ")
	}
}

func looksPlanTodoItem(line string) bool {
	trimmed := strings.TrimSpace(line)
	lower := strings.ToLower(trimmed)
	if checkboxPattern.MatchString(trimmed) {
		return true
	}
	for _, prefix := range []string{"- [", "* [", "• [", "☐", "☑", "✓ ", "✔ "} {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	if numberedChoicePattern.MatchString(trimmed) || labeledChoicePattern.MatchString(trimmed) {
		return true
	}
	return strings.HasPrefix(lower, "- ") || strings.HasPrefix(lower, "* ") || strings.HasPrefix(lower, "• ")
}

func looksIdle(line string) bool {
	lower := strings.ToLower(line)
	if strings.HasPrefix(line, ">") || strings.HasPrefix(line, "›") {
		return true
	}
	if strings.Contains(lower, "press enter") || strings.Contains(lower, "choose") || strings.Contains(lower, "select") {
		return true
	}
	return strings.HasSuffix(line, ">") || strings.HasSuffix(line, "›")
}

func choiceNeedsReview(choice AgentChoice) bool {
	label := strings.ToLower(strings.TrimSpace(choice.Label))
	if label == "" {
		return false
	}

	for _, exact := range []string{
		"allow",
		"allow once",
		"always allow",
		"approve",
		"confirm",
		"continue",
		"proceed",
		"yes",
		"yes, and don't ask again",
		"no",
		"deny",
		"reject",
		"accept",
	} {
		if label == exact {
			return true
		}
	}

	for _, token := range []string{
		"allow ",
		"approve ",
		"confirm ",
		"continue ",
		"proceed ",
		"don't ask again",
		"run command",
		"run this command",
		"execute command",
		"execute this command",
	} {
		if strings.Contains(label, token) {
			return true
		}
	}
	return false
}

func selectedChoiceNeedsReview(choice AgentChoice) bool {
	label := strings.ToLower(strings.TrimSpace(choice.Label))
	if label == "" {
		return false
	}

	for _, exact := range []string{
		"allow",
		"allow once",
		"always allow",
		"approve",
		"continue",
		"proceed",
		"yes",
		"yes, and don't ask again",
		"no",
		"deny",
		"reject",
		"accept",
	} {
		if label == exact {
			return true
		}
	}

	for _, prefix := range []string{
		"yes,",
		"yes ",
		"allow ",
		"approve ",
		"continue ",
		"proceed ",
		"no,",
		"no ",
		"deny ",
		"reject ",
		"accept ",
	} {
		if strings.HasPrefix(label, prefix) {
			return true
		}
	}
	return false
}

func choiceIsCurrentIdlePrompt(choice AgentChoice) bool {
	return choice.Selected && choice.Number == "" && !choiceNeedsReview(choice)
}

func looksNeedsReview(line string) bool {
	lower := strings.ToLower(line)
	if looksPlanTodoHeader(line) || looksPlanTodoItem(line) {
		return false
	}
	for _, token := range []string{
		"do you want to",
		"would you like to",
		"implement this plan?",
		"permission required",
		"requires approval",
		"approval required",
		"allow this command",
		"allow command?",
		"confirm?",
		"continue?",
		"proceed?",
		"are you sure",
		"run this command?",
		"execute this command?",
	} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func looksBusy(line string) bool {
	lower := strings.ToLower(line)
	for _, token := range []string{
		"esc to interrupt",
		"ctrl+c to interrupt",
		"running...",
		"running (",
		"thinking...",
		"thinking (",
		"working...",
		"working (",
		"executing...",
		"executing (",
	} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func looksActiveBackgroundTerminal(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "background terminal running") ||
		strings.Contains(lower, "background terminals running")
}
