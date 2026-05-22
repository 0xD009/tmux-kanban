package main

import "time"

const qqNotificationTarget = "qqbot"

type cliChoice struct {
	Number   string `json:"number,omitempty"`
	Label    string `json:"label"`
	Selected bool   `json:"selected"`
}

type cliScreen struct {
	Choices        []cliChoice `json:"choices,omitempty"`
	SelectedChoice int         `json:"selected_choice"`
	Idle           bool        `json:"idle"`
	Busy           bool        `json:"busy"`
	NeedsReview    bool        `json:"needs_review"`
	Status         string      `json:"status"`
}

type cliReviewItem struct {
	ID          string     `json:"id"`
	Host        string     `json:"host"`
	SSH         string     `json:"ssh,omitempty"`
	Local       bool       `json:"local"`
	SessionID   string     `json:"session_id"`
	SessionName string     `json:"session_name"`
	WindowID    string     `json:"window_id"`
	WindowIndex string     `json:"window_index"`
	WindowName  string     `json:"window_name"`
	PaneID      string     `json:"pane_id"`
	PaneIndex   string     `json:"pane_index"`
	Target      string     `json:"target"`
	Agent       string     `json:"agent"`
	Screen      cliScreen  `json:"screen"`
	Error       string     `json:"error,omitempty"`
	CapturedAt  *time.Time `json:"captured_at,omitempty"`
	Lines       []string   `json:"lines,omitempty"`
	Capture     []string   `json:"-"`
}

type cliReviewListResponse struct {
	OK           bool                   `json:"ok"`
	Items        []cliReviewItem        `json:"items"`
	Errors       []string               `json:"errors,omitempty"`
	Notification *cliNotificationResult `json:"notification,omitempty"`
	Scanned      time.Time              `json:"scanned_at"`
}

type cliNotificationResult struct {
	Enabled          bool   `json:"enabled"`
	Attempted        bool   `json:"attempted"`
	Sent             bool   `json:"sent"`
	Target           string `json:"target"`
	NeedsReviewCount int    `json:"needs_review_count"`
	Reason           string `json:"reason,omitempty"`
	Error            string `json:"error,omitempty"`
	HermesOutput     string `json:"hermes_output,omitempty"`
}

type cliCaptureResponse struct {
	OK         bool      `json:"ok"`
	Host       string    `json:"host"`
	Target     string    `json:"target"`
	Lines      []string  `json:"lines,omitempty"`
	Screen     cliScreen `json:"screen"`
	CapturedAt time.Time `json:"captured_at"`
	Error      string    `json:"error,omitempty"`
}

type cliSendResponse struct {
	OK     bool     `json:"ok"`
	Host   string   `json:"host"`
	Target string   `json:"target"`
	Action string   `json:"action"`
	Keys   []string `json:"keys,omitempty"`
	Error  string   `json:"error,omitempty"`
}

type cliSessionResponse struct {
	OK              bool   `json:"ok"`
	Host            string `json:"host"`
	Session         string `json:"session"`
	Action          string `json:"action"`
	Created         bool   `json:"created,omitempty"`
	Closed          bool   `json:"closed,omitempty"`
	RequiredConfirm string `json:"required_confirm,omitempty"`
	Error           string `json:"error,omitempty"`
}

type cliSnapshotResponse struct {
	OK    bool   `json:"ok"`
	Path  string `json:"path,omitempty"`
	Error string `json:"error,omitempty"`
}

type cliCapabilitiesResponse struct {
	OK          bool                   `json:"ok"`
	MainAgent   cliMainAgentCapability `json:"main_agent"`
	Skills      []string               `json:"skills"`
	CLICommands []string               `json:"cli_commands"`
	Summary     string                 `json:"summary"`
}

type cliMainAgentCapability struct {
	Enabled bool     `json:"enabled"`
	Host    string   `json:"host"`
	Session string   `json:"session"`
	Agent   string   `json:"agent"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
}
