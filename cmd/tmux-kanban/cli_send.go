package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"strings"

	"tmux-kanban/internal/agent"
	tmuxclient "tmux-kanban/internal/tmux"
)

func cliChoose(args []string) error {
	fs := flag.NewFlagSet("choose", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	configPath := fs.String("config", "", "path to config yaml")
	hostName := fs.String("host", "", "configured host name")
	target := fs.String("target", "", "tmux target, preferably a pane id such as %1")
	choice := fs.String("choice", "", "choice number to select")
	height := fs.Int("height", 40, "capture height before choosing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*choice) == "" {
		return errors.New("missing --choice")
	}

	_, host, err := loadHost(*configPath, *hostName)
	if err != nil {
		return err
	}
	client := tmuxclient.DefaultClient{}
	capture := client.CapturePane(context.Background(), host, *target, *height)
	if capture.Err != "" {
		return writeJSON(os.Stdout, cliSendResponse{
			OK:     false,
			Host:   displayHostName(host),
			Target: *target,
			Action: "choose",
			Error:  capture.Err,
		})
	}

	screen := agent.AnalyzeScreen(capture.Lines)
	keys := choiceKeys(screen, *choice)
	if len(keys) == 0 {
		return writeJSON(os.Stdout, cliSendResponse{
			OK:     false,
			Host:   displayHostName(host),
			Target: *target,
			Action: "choose",
			Error:  "choice is not visible",
		})
	}

	result := client.SendKeySequence(context.Background(), host, *target, keys...)
	return writeJSON(os.Stdout, cliSendResponse{
		OK:     result.Err == "",
		Host:   displayHostName(host),
		Target: *target,
		Action: "choose",
		Keys:   keys,
		Error:  result.Err,
	})
}

func cliSend(args []string) error {
	fs := flag.NewFlagSet("send", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	configPath := fs.String("config", "", "path to config yaml")
	hostName := fs.String("host", "", "configured host name")
	target := fs.String("target", "", "tmux target, preferably a pane id such as %1")
	text := fs.String("text", "", "text to send")
	submit := fs.Bool("submit", true, "send Enter after text")
	if err := fs.Parse(args); err != nil {
		return err
	}

	_, host, err := loadHost(*configPath, *hostName)
	if err != nil {
		return err
	}
	client := tmuxclient.DefaultClient{}
	result := client.SendText(context.Background(), host, *target, *text, *submit)
	return writeJSON(os.Stdout, cliSendResponse{
		OK:     result.Err == "",
		Host:   displayHostName(host),
		Target: *target,
		Action: "send",
		Error:  result.Err,
	})
}

func cliSendKeys(args []string) error {
	fs := flag.NewFlagSet("send-keys", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	configPath := fs.String("config", "", "path to config yaml")
	hostName := fs.String("host", "", "configured host name")
	target := fs.String("target", "", "tmux target, preferably a pane id such as %1")
	keysValue := fs.String("keys", "", "comma-separated tmux keys, e.g. C-c,C-m")
	if err := fs.Parse(args); err != nil {
		return err
	}

	keys := splitKeys(*keysValue)
	if len(keys) == 0 {
		return errors.New("missing --keys")
	}
	_, host, err := loadHost(*configPath, *hostName)
	if err != nil {
		return err
	}
	client := tmuxclient.DefaultClient{}
	result := client.SendKeys(context.Background(), host, *target, keys...)
	return writeJSON(os.Stdout, cliSendResponse{
		OK:     result.Err == "",
		Host:   displayHostName(host),
		Target: *target,
		Action: "send-keys",
		Keys:   keys,
		Error:  result.Err,
	})
}
