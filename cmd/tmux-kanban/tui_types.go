package main

import (
	"time"

	"tmux-kanban/internal/config"
	"tmux-kanban/internal/core"
	"tmux-kanban/internal/mesh"
	"tmux-kanban/internal/tmuxscan"
)

type model struct {
	cfg           config.Config
	hosts         []hostState
	cursor        int
	expanded      map[string]bool
	statuses      map[string]sessionStatus
	statusStreaks map[string]statusStreak
	reviewTargets map[string]selectedAgentTarget
	preview       previewState
	cache         map[string]previewCacheEntry
	control       agentControlState
	compose       composeState
	command       commandState
	snapshotInput snapshotDescriptionState
	viewMode      viewMode
	hermes        map[string]hermesAdvice
	activities    []agentActivity
	status        string
	width         int
	height        int

	reviewCursor    int
	reviewCursorKey string
	reviewSkipped   map[string]bool

	lastWheelAt        time.Time
	lastWheelDirection int
	skipRender         bool
	scanAnnounce       bool
}

type hostState struct {
	host     config.Host
	snapshot tmuxscan.Snapshot
	loading  bool
	loaded   bool
}

type row struct {
	key          string
	kind         rowKind
	hostIndex    int
	sessionIndex int
	windowIndex  int
	paneIndex    int
	label        string
	attachTarget string
	agent        tmuxscan.AgentKind
}

type previewState struct {
	key        string
	hostIndex  int
	target     string
	loading    bool
	refreshing bool
	lines      []string
	err        string
	capturedAt time.Time
}

type previewCacheEntry struct {
	lines      []string
	err        string
	capturedAt time.Time
}

type hermesAdvice struct {
	loading   bool
	text      string
	err       string
	updatedAt time.Time
}

type rowKind int

const (
	rowHost rowKind = iota
	rowSession
	rowWindow
	rowPane
)

type viewMode string

const (
	viewTree   viewMode = "tree"
	viewReview viewMode = "review"
)

type reviewItem struct {
	SessionKey  string
	HostName    string
	SessionName string
	Agent       tmuxscan.AgentKind
	Row         row
	Target      selectedAgentTarget
}

type scanResult struct {
	index    int
	snapshot tmuxscan.Snapshot
}

type attachFinished struct {
	err error
}

type captureResult struct {
	key     string
	capture tmuxscan.Capture
}

type previewTick struct {
	key string
}

type scanTick struct{}

type agentStatusResult struct {
	key    string
	status sessionStatus
	target selectedAgentTarget
	ok     bool
}

type hermesQueryResult struct {
	key    string
	text   string
	err    string
	auto   bool
	item   reviewItem
	host   config.Host
	lines  []string
	hermes config.HermesConfig
}

type hermesNextStepResult struct {
	key         string
	status      sessionStatus
	text        string
	err         string
	auto        bool
	host        config.Host
	hostName    string
	sessionName string
	target      selectedAgentTarget
	lines       []string
	hermes      config.HermesConfig
}

type memoryUpdateResult struct {
	scope mesh.Scope
	path  string
	text  string
	err   string
}

type sendResult struct {
	action string
	result tmuxscan.SendResult
}

type qqNotifyResult struct {
	result cliNotificationResult
}

type snapshotResult struct {
	path string
	err  string
}

type snapshotDescriptionState struct {
	active    bool
	text      string
	textRunes []rune
	cursor    int
}

type agentActivitySource string

const (
	agentActivitySession agentActivitySource = "session"
	agentActivityReview  agentActivitySource = "review"
)

const (
	maxAgentActivities = 80
)

type agentActivity struct {
	At      time.Time
	Source  agentActivitySource
	Agent   string
	Target  string
	State   string
	Message string
}

type agentControlState struct {
	active    bool
	key       string
	hostIndex int
	target    string
	agent     tmuxscan.AgentKind
}

type composeState struct {
	active    bool
	key       string
	hostIndex int
	target    string
	label     string
	agent     tmuxscan.AgentKind
	text      string
	textRunes []rune
	cursor    int
}

type commandState struct {
	active   bool
	text     string
	selected int
}

type sessionStatus = core.SessionStatus

type statusStreak struct {
	status sessionStatus
	count  int
}

const (
	sessionIdle       sessionStatus = core.StatusIdle
	sessionWorking    sessionStatus = core.StatusWorking
	sessionNeedReview sessionStatus = core.StatusNeedReview
	sessionDone       sessionStatus = core.StatusDone
)

type sessionCard struct {
	Key      string
	Host     string
	Name     string
	Agent    string
	Meta     string
	Selected bool
}

const (
	pollInterval           = 10 * time.Second
	previewRefreshInterval = 1 * time.Second
	previewCaptureHeight   = 120
	wheelThrottleInterval  = 80 * time.Millisecond
)
