package main

import (
	"context"
	"flag"
	"io"
	"os"
	"strings"
	"time"

	"tmux-kanban/internal/agent"
	"tmux-kanban/internal/config"
	debugsnap "tmux-kanban/internal/debug"
	tmuxclient "tmux-kanban/internal/tmux"
	"tmux-kanban/internal/tmuxscan"
)

func cliSnapshot(args []string) error {
	fs := flag.NewFlagSet("snapshot", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	configPath := fs.String("config", "", "path to config yaml")
	dir := fs.String("dir", "", "snapshot output directory")
	height := fs.Int("height", 40, "agent pane capture height")
	includeLines := fs.Bool("lines", true, "include captured pane lines")
	description := fs.String("description", "", "human-readable snapshot description")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	snapshot := buildCLIDebugSnapshot(cfg, *height, *includeLines, *description)
	outputDir := *dir
	if strings.TrimSpace(outputDir) == "" {
		outputDir = debugsnap.ResolveSnapshotDir(cfg)
	}
	path, err := debugsnap.WriteSnapshot(outputDir, snapshot)
	if err != nil {
		return writeJSON(os.Stdout, cliSnapshotResponse{OK: false, Error: err.Error()})
	}
	return writeJSON(os.Stdout, cliSnapshotResponse{OK: true, Path: path})
}

func buildCLIDebugSnapshot(cfg config.Config, height int, includeLines bool, description string) debugsnap.Snapshot {
	client := tmuxclient.DefaultClient{}
	hosts := make([]debugsnap.HostSnapshot, 0, len(cfg.Hosts))
	reviewItems := make([]debugsnap.ReviewItem, 0)
	statuses := map[string]string{}
	targets := map[string]string{}
	errorsOut := make([]string, 0)

	for _, host := range cfg.Hosts {
		snapshot := client.ScanHost(context.Background(), host)
		sessions := make([]interface{}, 0, len(snapshot.Sessions))
		for _, session := range snapshot.Sessions {
			sessions = append(sessions, session)
		}
		if snapshot.Err != "" {
			errorsOut = append(errorsOut, displayHostName(host)+": "+snapshot.Err)
		}
		hosts = append(hosts, debugsnap.HostSnapshot{
			Name:     host.Name,
			SSH:      host.SSH,
			Local:    host.Local,
			Loaded:   snapshot.Err == "",
			Error:    snapshot.Err,
			Sessions: sessions,
		})

		for _, item := range reviewItemsForSnapshot(host, snapshot, height, includeLines) {
			statuses[item.ID] = item.Screen.Status
			if item.Screen.NeedsReview {
				targets[item.ID] = item.Target
				reviewItems = append(reviewItems, debugsnap.ReviewItem{
					SessionKey:   item.ID,
					Host:         item.Host,
					SessionName:  item.SessionName,
					Agent:        item.Agent,
					Target:       item.Target,
					ScreenStatus: item.Screen.Status,
					NeedsReview:  item.Screen.NeedsReview,
					Capture:      reviewItemCapture(item),
				})
			}
		}
	}

	return debugsnap.Snapshot{
		Version:     1,
		CreatedAt:   time.Now(),
		Description: strings.TrimSpace(description),
		Config:      debugsnap.NewConfigSummary(cfg),
		Runtime: debugsnap.RuntimeState{
			ViewMode:        "cli",
			Status:          "snapshot",
			SessionStatuses: statuses,
			ReviewTargets:   targets,
		},
		Hosts:       hosts,
		ReviewQueue: reviewItems,
		Errors:      errorsOut,
	}
}

func reviewItemsForSnapshot(host config.Host, snapshot tmuxscan.Snapshot, height int, includeLines bool) []cliReviewItem {
	items := make([]cliReviewItem, 0)
	hostName := displayHostName(host)
	for _, session := range snapshot.Sessions {
		for _, window := range session.Windows {
			for _, pane := range window.Panes {
				if pane.Agent == tmuxscan.AgentNone {
					continue
				}
				target := tmuxPaneTarget(session.Name, window, pane)
				client := tmuxclient.DefaultClient{}
				capture := client.CapturePane(context.Background(), host, target, height)
				screen := cliScreenFromAgentScreen(agent.AnalyzeScreen(capture.Lines))
				item := cliReviewItem{
					ID:          hostName + "/" + target,
					Host:        hostName,
					SSH:         host.SSH,
					Local:       host.Local,
					SessionID:   session.ID,
					SessionName: session.Name,
					WindowID:    window.ID,
					WindowIndex: window.Index,
					WindowName:  window.Name,
					PaneID:      pane.ID,
					PaneIndex:   pane.Index,
					Target:      target,
					Agent:       string(pane.Agent),
					Screen:      screen,
					Error:       capture.Err,
					Capture:     append([]string(nil), capture.Lines...),
				}
				if !capture.CapturedAt.IsZero() {
					capturedAt := capture.CapturedAt
					item.CapturedAt = &capturedAt
				}
				if includeLines {
					item.Lines = capture.Lines
				}
				items = append(items, item)
			}
		}
	}
	return items
}
