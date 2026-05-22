package main

import (
	"context"
	"fmt"
	"strings"

	"tmux-kanban/internal/agent"
	"tmux-kanban/internal/config"
	"tmux-kanban/internal/hermeslog"
)

func notifyQQForReviewItems(cfg config.Config, items []cliReviewItem, intent string) cliNotificationResult {
	needsReview := needsReviewItems(items)
	result := cliNotificationResult{
		Enabled:          cfg.Notification.QQEnabled,
		Target:           qqNotificationTarget,
		NeedsReviewCount: len(needsReview),
	}
	if len(needsReview) == 0 {
		result.Reason = "no needs_review items"
		return result
	}
	if !cfg.Notification.QQEnabled {
		result.Reason = "notification.qq_enabled is false"
		return result
	}

	ctx, cancel := hermesTimeoutContext(context.Background(), cfg.Hermes)
	defer cancel()

	result.Attempted = true
	prompt := hermesQQNotificationPrompt(needsReview, intent)
	output, err := runHermesOneshot(ctx, cfg.Hermes, prompt)
	if err != nil {
		result.Error = err.Error()
		appendHermesWorkLogForConfig(cfg, hermeslog.Entry{
			Flow:    "notify",
			Event:   "error",
			Mode:    "manual",
			Trigger: "notify-review-cli",
			Conditions: map[string]string{
				"needs_review_count": fmt.Sprintf("%d", len(needsReview)),
				"intent":             intent,
			},
			Error: err.Error(),
		})
		return result
	}

	result.Sent = true
	result.Reason = "sent"
	result.HermesOutput = clipString(output, 400)
	appendHermesWorkLogForConfig(cfg, hermeslog.Entry{
		Flow:     "notify",
		Event:    "sent",
		Mode:     "manual",
		Trigger:  "notify-review-cli",
		Advice:   output,
		Accepted: true,
		Modified: true,
		Target:   qqNotificationTarget,
		Conditions: map[string]string{
			"needs_review_count": fmt.Sprintf("%d", len(needsReview)),
			"intent":             intent,
		},
	})
	return result
}

func needsReviewItems(items []cliReviewItem) []cliReviewItem {
	out := make([]cliReviewItem, 0, len(items))
	for _, item := range items {
		if item.Screen.NeedsReview {
			out = append(out, item)
		}
	}
	return out
}

func hermesQQNotificationPrompt(items []cliReviewItem, intent string) string {
	intent = strings.TrimSpace(intent)
	if intent == "" {
		intent = "Notify the user that tmux-kanban found Codex or Claude Code sessions waiting for human review."
	}

	var body strings.Builder
	body.WriteString("You are Hermes running in oneshot mode. You do not have the user's current QQ chat context.\n")
	body.WriteString("Use send_message(target=\"")
	body.WriteString(qqNotificationTarget)
	body.WriteString("\", message=...) exactly once to notify the user.\n")
	body.WriteString("Send one concise Chinese QQ message. Do not send multiple messages.\n")
	body.WriteString("Only notify because the CLI found current needs_review items.\n\n")
	body.WriteString("User intent:\n")
	body.WriteString(intent)
	body.WriteString("\n\n")
	body.WriteString("Review items:\n")
	for i, item := range items {
		body.WriteString(fmt.Sprintf("\nItem %d:\n", i+1))
		body.WriteString("- Host: " + item.Host + "\n")
		body.WriteString("- Target: " + item.Target + "\n")
		body.WriteString("- Agent: " + item.Agent + "\n")
		body.WriteString("- Session: " + item.SessionName + "\n")
		body.WriteString("- Window: " + item.WindowIndex + " " + item.WindowName + "\n")
		body.WriteString("- Pane: " + item.PaneIndex + "\n")
		if len(item.Screen.Choices) > 0 {
			body.WriteString("Choices:\n")
			for choiceIndex, choice := range item.Screen.Choices {
				number := choice.Number
				if number == "" {
					number = fmt.Sprintf("%d", choiceIndex+1)
				}
				marker := ""
				if choice.Selected {
					marker = " (currently selected)"
				}
				body.WriteString(fmt.Sprintf("- %s: %s%s\n", number, choice.Label, marker))
			}
		}
		body.WriteString("Pane capture:\n")
		body.WriteString("```text\n")
		body.WriteString(strings.Join(tailPreviewLines(reviewItemCapture(item), 160, 40), "\n"))
		body.WriteString("\n```\n")
	}
	body.WriteString("\nThe QQ message should include host, target, agent, and the visible choice summary if choices are present.\n")
	return body.String()
}

func reviewItemCapture(item cliReviewItem) []string {
	if len(item.Capture) > 0 {
		return item.Capture
	}
	return item.Lines
}

func cliScreenFromAgentScreen(screen agent.Screen) cliScreen {
	choices := make([]cliChoice, 0, len(screen.Choices))
	for _, choice := range screen.Choices {
		choices = append(choices, cliChoice{
			Number:   choice.Number,
			Label:    choice.Label,
			Selected: choice.Selected,
		})
	}
	status, ok := sessionStatusFromAgentScreen(screen)
	statusText := "unknown"
	if ok {
		statusText = statusLabel(status)
	}
	return cliScreen{
		Choices:        choices,
		SelectedChoice: screen.SelectedChoice,
		Idle:           screen.Idle,
		Busy:           screen.Busy,
		NeedsReview:    screen.NeedsReview,
		Status:         statusText,
	}
}
