package main

import (
	"context"
	"flag"
	"io"
	"os"
	"time"

	"tmux-kanban/internal/config"
	tmuxclient "tmux-kanban/internal/tmux"
)

func cliReviewList(args []string) error {
	fs := flag.NewFlagSet("review-list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	configPath := fs.String("config", "", "path to config yaml")
	all := fs.Bool("all", false, "include idle/working/unknown agent panes")
	height := fs.Int("height", 40, "capture height")
	includeLines := fs.Bool("lines", false, "include captured pane lines")
	notify := fs.Bool("notify", false, "send a QQ notification when needs_review items are found and notification.qq_enabled is true")
	intent := fs.String("intent", "", "user intent to include in the Hermes QQ prompt")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	response := collectReviewList(cfg, *all, *height, *includeLines)
	if *notify {
		notification := notifyQQForReviewItems(cfg, response.Items, *intent)
		response.Notification = &notification
		if notification.Error != "" {
			response.OK = false
		}
	}
	return writeJSON(os.Stdout, response)
}

func cliNotifyReview(args []string) error {
	fs := flag.NewFlagSet("notify-review", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	configPath := fs.String("config", "", "path to config yaml")
	height := fs.Int("height", 40, "capture height")
	includeLines := fs.Bool("lines", false, "include captured pane lines")
	intent := fs.String("intent", "", "user intent to include in the Hermes QQ prompt")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	response := collectReviewList(cfg, false, *height, *includeLines)
	notification := notifyQQForReviewItems(cfg, response.Items, *intent)
	response.Notification = &notification
	if notification.Error != "" {
		response.OK = false
	}
	return writeJSON(os.Stdout, response)
}

func collectReviewList(cfg config.Config, all bool, height int, includeLines bool) cliReviewListResponse {
	response := cliReviewListResponse{
		OK:      true,
		Items:   []cliReviewItem{},
		Scanned: time.Now(),
	}

	client := tmuxclient.DefaultClient{}
	for _, host := range cfg.Hosts {
		snapshot := client.ScanHost(context.Background(), host)
		hostName := displayHostName(host)
		if snapshot.Err != "" {
			response.Errors = append(response.Errors, hostName+": "+snapshot.Err)
			continue
		}

		for _, item := range reviewItemsForSnapshot(host, snapshot, height, includeLines) {
			if all || item.Screen.NeedsReview {
				response.Items = append(response.Items, item)
			}
		}
	}
	if len(response.Errors) > 0 {
		response.OK = false
	}
	return response
}
