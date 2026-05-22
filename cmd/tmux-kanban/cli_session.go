package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"strings"

	"tmux-kanban/internal/config"
	tmuxclient "tmux-kanban/internal/tmux"
)

func cliSessionOpen(args []string) error {
	fs := flag.NewFlagSet("session-open", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	configPath := fs.String("config", "", "path to config yaml")
	hostName := fs.String("host", "", "configured host name")
	sessionName := fs.String("name", "", "initial tmux session name")
	command := fs.String("command", "", "optional command to run in the session")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*sessionName) == "" {
		return errors.New("missing --name")
	}

	_, host, err := loadHost(*configPath, *hostName)
	if err != nil {
		return err
	}
	client := tmuxclient.DefaultClient{}
	result := client.StartSession(context.Background(), host, *sessionName, *command, fs.Args()...)
	return writeJSON(os.Stdout, cliSessionResponse{
		OK:      result.Err == "",
		Host:    displayHostName(host),
		Session: strings.TrimSpace(*sessionName),
		Action:  "session-open",
		Created: result.Created,
		Error:   result.Err,
	})
}

func cliSessionClose(args []string) error {
	fs := flag.NewFlagSet("session-close", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	configPath := fs.String("config", "", "path to config yaml")
	hostName := fs.String("host", "", "configured host name")
	sessionName := fs.String("name", "", "tmux session name to close")
	confirm := fs.String("confirm", "", "confirmation token in the form host/session")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*sessionName) == "" {
		return errors.New("missing --name")
	}

	_, host, err := loadHost(*configPath, *hostName)
	if err != nil {
		return err
	}
	requiredConfirm := sessionCloseConfirmationToken(host, *sessionName)
	if strings.TrimSpace(*confirm) != requiredConfirm {
		return writeJSON(os.Stdout, cliSessionResponse{
			OK:              false,
			Host:            displayHostName(host),
			Session:         strings.TrimSpace(*sessionName),
			Action:          "session-close",
			RequiredConfirm: requiredConfirm,
			Error:           "confirmation required",
		})
	}

	client := tmuxclient.DefaultClient{}
	result := client.CloseSession(context.Background(), host, *sessionName)
	return writeJSON(os.Stdout, cliSessionResponse{
		OK:              result.Err == "",
		Host:            displayHostName(host),
		Session:         strings.TrimSpace(*sessionName),
		Action:          "session-close",
		Closed:          result.Closed,
		RequiredConfirm: requiredConfirm,
		Error:           result.Err,
	})
}

func sessionCloseConfirmationToken(host config.Host, session string) string {
	return displayHostName(host) + "/" + strings.TrimSpace(session)
}
