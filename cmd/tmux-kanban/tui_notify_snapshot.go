package main

import (
	"context"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"tmux-kanban/internal/agent"
	"tmux-kanban/internal/config"
	debugsnap "tmux-kanban/internal/debug"
	tmuxclient "tmux-kanban/internal/tmux"
)

func (m *model) notifyQQForReviewQueue(intent string) tea.Cmd {
	items := m.reviewQueue()
	if len(items) == 0 {
		m.status = "review queue is empty"
		return nil
	}
	m.status = "sending QQ notification..."
	return qqNotifyCmd(m.cfg, items, m.hosts, intent)
}

func (m *model) saveSnapshotCmd(description string) tea.Cmd {
	m.status = "saving snapshot..."
	snapshot := m.debugSnapshot(description)
	dir := debugsnap.ResolveSnapshotDir(m.cfg)
	return func() tea.Msg {
		path, err := debugsnap.WriteSnapshot(dir, snapshot)
		if err != nil {
			return snapshotResult{err: err.Error()}
		}
		return snapshotResult{path: path}
	}
}

func (m model) debugSnapshot(description string) debugsnap.Snapshot {
	items := m.reviewQueue()
	reviewItems := make([]debugsnap.ReviewItem, 0, len(items))
	for _, item := range items {
		reviewItems = append(reviewItems, debugsnap.ReviewItem{
			SessionKey:   item.SessionKey,
			Host:         item.HostName,
			SessionName:  item.SessionName,
			Agent:        string(item.Agent),
			Target:       item.Row.attachTarget,
			ScreenStatus: statusLabel(sessionNeedReview),
			NeedsReview:  true,
		})
	}

	hosts := make([]debugsnap.HostSnapshot, 0, len(m.hosts))
	errors := make([]string, 0)
	for _, host := range m.hosts {
		sessions := make([]interface{}, 0, len(host.snapshot.Sessions))
		for _, session := range host.snapshot.Sessions {
			sessions = append(sessions, session)
		}
		hostSnapshot := debugsnap.HostSnapshot{
			Name:     host.host.Name,
			SSH:      host.host.SSH,
			Local:    host.host.Local,
			Loading:  host.loading,
			Loaded:   host.loaded,
			Error:    host.snapshot.Err,
			Sessions: sessions,
		}
		if host.snapshot.Err != "" {
			errors = append(errors, displayHostName(host.host)+": "+host.snapshot.Err)
		}
		hosts = append(hosts, hostSnapshot)
	}

	statuses := make(map[string]string, len(m.statuses))
	for key, status := range m.statuses {
		statuses[key] = statusLabel(status)
	}
	targets := make(map[string]string, len(m.reviewTargets))
	for key, target := range m.reviewTargets {
		targets[key] = target.target
	}
	skipped := make([]string, 0, len(m.reviewSkipped))
	for key, skippedValue := range m.reviewSkipped {
		if skippedValue {
			skipped = append(skipped, key)
		}
	}
	sort.Strings(skipped)

	activities := make([]debugsnap.AgentActivity, 0, len(m.activities))
	for _, activity := range m.activities {
		activities = append(activities, debugsnap.AgentActivity{
			At:      activity.At,
			Source:  string(activity.Source),
			Agent:   activity.Agent,
			Target:  activity.Target,
			State:   activity.State,
			Message: activity.Message,
		})
	}

	return debugsnap.Snapshot{
		Version:     1,
		CreatedAt:   time.Now(),
		Description: strings.TrimSpace(description),
		Config:      debugsnap.NewConfigSummary(m.cfg),
		Runtime: debugsnap.RuntimeState{
			ViewMode:        string(m.viewMode),
			MainActive:      m.mainActive,
			Status:          m.status,
			SessionStatuses: statuses,
			ReviewTargets:   targets,
			SkippedReview:   skipped,
			ReviewCursor:    m.reviewCursor,
			ReviewCursorKey: m.reviewCursorKey,
		},
		Hosts:       hosts,
		ReviewQueue: reviewItems,
		Activities:  activities,
		Preview: debugsnap.PreviewState{
			Key:        m.preview.key,
			HostIndex:  m.preview.hostIndex,
			Target:     m.preview.target,
			Loading:    m.preview.loading,
			Refreshing: m.preview.refreshing,
			Error:      m.preview.err,
			CapturedAt: m.preview.capturedAt,
			Lines:      append([]string(nil), m.preview.lines...),
		},
		Errors: errors,
	}
}

func qqNotifyCmd(cfg config.Config, items []reviewItem, hosts []hostState, intent string) tea.Cmd {
	return func() tea.Msg {
		cliItems := make([]cliReviewItem, 0, len(items))
		for _, item := range items {
			cliItem := cliReviewItem{
				ID:          item.HostName + "/" + item.Row.attachTarget,
				Host:        item.HostName,
				SessionName: item.SessionName,
				Target:      item.Row.attachTarget,
				Agent:       string(item.Agent),
				Screen: cliScreen{
					NeedsReview: true,
					Status:      statusLabel(sessionNeedReview),
				},
			}

			if cfg.Notification.QQEnabled && item.Row.hostIndex >= 0 && item.Row.hostIndex < len(hosts) {
				client := tmuxclient.DefaultClient{}
				capture := client.CapturePane(context.Background(), hosts[item.Row.hostIndex].host, item.Row.attachTarget, 40)
				cliItem.Error = capture.Err
				cliItem.Capture = capture.Lines
				cliItem.Screen = cliScreenFromAgentScreen(agent.AnalyzeScreen(capture.Lines))
				if !capture.CapturedAt.IsZero() {
					capturedAt := capture.CapturedAt
					cliItem.CapturedAt = &capturedAt
				}
			}

			cliItems = append(cliItems, cliItem)
		}
		return qqNotifyResult{result: notifyQQForReviewItems(cfg, cliItems, intent)}
	}
}
