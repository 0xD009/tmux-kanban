package main

import (
	"context"
	"flag"
	"io"
	"os"

	"tmux-kanban/internal/agent"
	tmuxclient "tmux-kanban/internal/tmux"
)

func cliCapture(args []string) error {
	fs := flag.NewFlagSet("capture", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	configPath := fs.String("config", "", "path to config yaml")
	hostName := fs.String("host", "", "configured host name")
	target := fs.String("target", "", "tmux target, preferably a pane id such as %1")
	height := fs.Int("height", 40, "capture height")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, host, err := loadHost(*configPath, *hostName)
	if err != nil {
		return err
	}
	_ = cfg
	client := tmuxclient.DefaultClient{}
	capture := client.CapturePane(context.Background(), host, *target, *height)
	screen := cliScreenFromAgentScreen(agent.AnalyzeScreen(capture.Lines))
	response := cliCaptureResponse{
		OK:         capture.Err == "",
		Host:       displayHostName(host),
		Target:     *target,
		Lines:      capture.Lines,
		Screen:     screen,
		CapturedAt: capture.CapturedAt,
		Error:      capture.Err,
	}
	return writeJSON(os.Stdout, response)
}
