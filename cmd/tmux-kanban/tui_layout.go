package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"tmux-kanban/internal/tmuxscan"
)

func (m model) headerHeight() int {
	return 5
}

type tuiMetrics struct {
	hosts    int
	sessions int
	review   int
	skipped  int
	loading  int
	errors   int
}

func (m model) metrics() tuiMetrics {
	metrics := tuiMetrics{
		hosts:   len(m.hosts),
		review:  len(m.reviewQueue()),
		skipped: m.skippedReviewCount(),
	}
	for _, host := range m.hosts {
		if host.loading {
			metrics.loading++
		}
		if host.loaded {
			metrics.sessions += len(host.snapshot.Sessions)
		}
		if host.loaded && host.snapshot.Err != "" {
			metrics.errors++
		}
	}
	return metrics
}

func (m model) renderHeader(width int) string {
	metrics := m.metrics()
	mode := strings.ToUpper(string(m.viewMode))
	if mode == "" {
		mode = "TREE"
	}
	status := strings.TrimSpace(m.status)
	if status == "" {
		status = "ready"
	}

	title := titleStyle.Render("tmux-kanban")
	modeChip := chipStyle(lipgloss.Color("236"), lipgloss.Color("153")).Render(" " + mode + " ")
	summary := headerMetaStyle.Render(fmt.Sprintf("hosts %d | sessions %d | review %d | skipped %d | loading %d | errors %d", metrics.hosts, metrics.sessions, metrics.review, metrics.skipped, metrics.loading, metrics.errors))
	line1 := ansi.Truncate(title+" "+modeChip+" "+summary, width, "...")
	line2 := headerMetaStyle.Render(ansi.Truncate(m.helpLine(), width, "..."))
	line3 := headerStatusStyle.Render(ansi.Truncate("status: "+status, width, "..."))
	return line1 + "\n" + line2 + "\n" + line3 + "\n\n"
}

func (m model) helpLine() string {
	switch {
	case m.snapshotInput.active:
		return "snapshot description | enter save | esc cancel | ctrl+u clear"
	case m.command.active:
		return "command mode | up/down choose | tab complete | enter run | esc cancel | ctrl+u clear"
	case m.compose.active:
		return "message mode | enter send | left/right move | esc cancel | ctrl+u clear"
	case m.control.active:
		return "relay mode | j/k or arrows move remote choice | enter choose | tab next | esc stop"
	case m.viewMode == viewReview:
		return "review view | : command | tab/v tree | j/k move | pgup/pgdn preview | 1-9 choose | h ask Hermes | s skip | u unskip | d snapshot"
	default:
		return "tree view | : command | tab/v review | q quit | r refresh | j/k move | pgup/pgdn preview | enter toggle | a attach | s status | d snapshot"
	}
}

func (m model) renderWorkspace(width int, height int, topRow int, leftCol int) string {
	hostsHeight, previewHeight := splitWorkspaceHeights(height)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderHosts(width, hostsHeight),
		m.renderPreviewPanel(width, previewHeight, topRow+hostsHeight, leftCol),
	)
}

func (m model) renderReviewView(width int, height int, topRow int, leftCol int) string {
	if width >= 140 {
		queueWidth := threeColumnSideWidth(width)
		activityWidth := threeColumnActivityWidth(width, queueWidth)
		queue := m.renderReviewQueue(queueWidth, height)
		previewLeftCol := leftCol + lipgloss.Width(queue) + 2
		preview := m.renderPreviewPanel(maxInt(60, width-queueWidth-activityWidth-4), height, topRow, previewLeftCol)
		activity := m.renderAgentActivity(activityWidth, height)
		return lipgloss.JoinHorizontal(lipgloss.Top, queue, "  ", preview, "  ", activity)
	}

	if width >= 104 {
		queueWidth := twoColumnSideWidth(width)
		queue := m.renderReviewQueue(queueWidth, height)
		previewLeftCol := leftCol + lipgloss.Width(queue) + 2
		preview := m.renderPreviewPanel(maxInt(60, width-queueWidth-2), height, topRow, previewLeftCol)
		return lipgloss.JoinHorizontal(lipgloss.Top, queue, "  ", preview)
	}

	queueHeight := minInt(12, maxInt(8, height/3))
	previewHeight := maxInt(8, height-queueHeight-2)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderReviewQueue(width, queueHeight),
		m.renderPreviewPanel(width, previewHeight, topRow+queueHeight, leftCol),
	)
}

func threeColumnSideWidth(totalWidth int) int {
	return minInt(42, maxInt(38, totalWidth/4))
}

func threeColumnActivityWidth(totalWidth int, leftWidth int) int {
	available := totalWidth - leftWidth - 4
	if available <= 60 {
		return threeColumnSideWidth(totalWidth)
	}

	oneThirdOfPreview := (available + 3) / 4
	width := maxInt(threeColumnSideWidth(totalWidth), oneThirdOfPreview)
	return minInt(width, available-60)
}

func twoColumnSideWidth(totalWidth int) int {
	return minInt(42, maxInt(38, totalWidth/3))
}

func (m model) renderReviewQueue(width int, height int) string {
	innerHeight := panelInnerHeight(height)
	lineWidth := maxInt(18, width-8)
	items := m.reviewQueue()
	skipped := m.skippedReviewCount()

	lines := []string{
		panelTitleStyle.Render("Review Queue"),
		headerMetaStyle.Render(fmt.Sprintf("%d queued | %d skipped", len(items), skipped)),
		ruleStyle.Render(strings.Repeat("-", lineWidth)),
		"",
	}

	if len(items) == 0 {
		message := "queue empty"
		if skipped > 0 {
			message = "all review items skipped"
		}
		lines = append(lines, mutedStyle.Render(message))
		lines = append(lines, mutedStyle.Render("u  restore skipped items"))
		return renderFixedPanel(width, innerHeight, lines)
	}

	cursor := m.reviewCursorIndex(items)
	for i, item := range items {
		prefix := fmt.Sprintf("%02d ", i+1)
		if i == cursor {
			prefix = "> "
		}
		agentBadge := ""
		if item.Agent != tmuxscan.AgentNone {
			agentBadge = " " + agentBadgeStyle(item.Agent)
		}
		label := fmt.Sprintf("%s%s/%s%s", prefix, item.HostName, item.SessionName, agentBadge)
		lines = append(lines, renderListLine(label, lineWidth, i == cursor, false))
	}

	return renderFixedPanel(width, innerHeight, lines)
}

func (m model) renderAgentActivity(width int, height int) string {
	innerHeight := panelInnerHeight(height)
	lineWidth := maxInt(18, width-8)
	lines := []string{
		panelTitleStyle.Render("Agent Activity"),
		headerMetaStyle.Render("session agents + review agent"),
		ruleStyle.Render(strings.Repeat("-", lineWidth)),
		"",
	}
	headerLines := len(lines)

	if len(m.activities) == 0 {
		lines = append(lines, mutedStyle.Render("no activity yet"))
		lines = append(lines, mutedStyle.Render("status, review, sends"))
		lines = append(lines, mutedStyle.Render("will appear here"))
		return renderFixedPanel(width, innerHeight, lines)
	}

	content := make([]string, 0, len(m.activities)*3)
	for i := len(m.activities) - 1; i >= 0; i-- {
		activity := m.activities[i]
		content = append(content, renderAgentActivityHeader(activity, lineWidth))
		target := strings.TrimSpace(activity.Target)
		if target == "" {
			target = "unknown target"
		}
		targetPrefix := "-> "
		if isHermesAnswerActivity(activity) {
			targetPrefix = "Q: "
		}
		content = append(content, mutedStyle.Render(ansi.Truncate(targetPrefix+target, lineWidth, "...")))
		if message := strings.TrimSpace(activity.Message); message != "" {
			content = append(content, renderAgentActivityMessageLines(activity, lineWidth, 3)...)
		}
	}

	contentHeight := maxInt(0, innerHeight-headerLines)
	if len(content) > contentHeight {
		start := clampInt(m.activityScroll, 0, len(content)-contentHeight)
		content = content[start : start+contentHeight]
	}
	lines = append(lines, content...)
	return renderFixedPanel(width, innerHeight, lines)
}

func renderAgentActivityMessageLines(activity agentActivity, width int, maxLines int) []string {
	if maxLines <= 0 {
		return nil
	}
	message := strings.TrimSpace(activity.Message)
	if message == "" {
		return nil
	}
	if !isHermesAnswerActivity(activity) {
		return []string{previewBorderStyle.Render(ansi.Truncate(message, width, "..."))}
	}

	answerWidth := maxInt(1, width-3)
	rawLines := compactTextLines(message, answerWidth, minInt(3, maxLines))
	lines := make([]string, 0, len(rawLines))
	for i, line := range rawLines {
		prefix := "   "
		if i == 0 {
			prefix = "A: "
		}
		lines = append(lines, previewBorderStyle.Render(ansi.Truncate(prefix+line, width, "...")))
	}
	return lines
}

func isHermesAnswerActivity(activity agentActivity) bool {
	return strings.EqualFold(strings.TrimSpace(activity.Agent), "Hermes") && strings.EqualFold(strings.TrimSpace(activity.State), "replied")
}

func splitWorkspaceHeights(height int) (int, int) {
	if height <= 1 {
		return maxInt(0, height), 0
	}

	minHostsHeight := 8
	minPreviewHeight := 8
	previewHeight := maxInt(14, (height*5)/6)
	maxPreviewHeight := height - minHostsHeight
	if maxPreviewHeight < 1 {
		maxPreviewHeight = 1
	}
	if previewHeight > maxPreviewHeight {
		previewHeight = maxPreviewHeight
	}
	if height >= minHostsHeight+minPreviewHeight && previewHeight < minPreviewHeight {
		previewHeight = minPreviewHeight
	}

	return height - previewHeight, previewHeight
}
