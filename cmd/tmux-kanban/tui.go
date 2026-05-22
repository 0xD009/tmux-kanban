package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"tmux-kanban/internal/config"
	"tmux-kanban/internal/ui"
)

func initialModel(cfg config.Config) model {
	hosts := make([]hostState, len(cfg.Hosts))
	expanded := map[string]bool{}
	for i, host := range cfg.Hosts {
		hosts[i] = hostState{host: host, loading: true}
		expanded[hostKey(i)] = true
	}

	return model{
		cfg:           cfg,
		hosts:         hosts,
		expanded:      expanded,
		statuses:      map[string]sessionStatus{},
		statusStreaks: map[string]statusStreak{},
		reviewTargets: map[string]selectedAgentTarget{},
		cache:         map[string]previewCacheEntry{},
		viewMode:      viewTree,
		hermes:        map[string]hermesAdvice{},
		status:        "scanning remote tmux sessions...",
		width:         100,
		height:        32,
		reviewSkipped: map[string]bool{},
	}
}

func (m model) Init() tea.Cmd {
	_, cmd := m.startScanModel()
	return tea.Batch(cmd, scanTickCmd())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.skipRender = false
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.MouseMsg:
		if !m.command.active && !m.compose.active && !m.control.active && !m.snapshotInput.active {
			m.handleMouse(msg, time.Now())
		}
	case tea.KeyMsg:
		if m.snapshotInput.active {
			return m.updateSnapshotDescription(msg)
		}
		if m.command.active {
			return m.updateCommand(msg)
		}
		if m.compose.active {
			return m.updateCompose(msg)
		}
		if m.control.active {
			return m.updateAgentControl(msg)
		}

		switch msg.String() {
		case "ctrl+c", ui.KeyQuit:
			return m, tea.Quit
		case ui.KeyCommand:
			m.beginCommand()
			return m, tea.ShowCursor
		case ui.KeyRefresh:
			return m.startScan()
		case ui.KeyToggleView, ui.KeyToggleView2:
			m.toggleViewMode()
		case "enter", " ":
			if m.viewMode == viewTree {
				m.toggleSelected()
			}
		case "up", "k":
			if m.viewMode == viewReview {
				m.moveReviewCursor(-1)
			} else {
				m.moveCursor(-1)
			}
		case "down", "j":
			if m.viewMode == viewReview {
				m.moveReviewCursor(1)
			} else {
				m.moveCursor(1)
			}
		case ui.KeyAttach:
			return m, m.attachSelected()
		case ui.KeyStatus:
			if m.viewMode == viewReview {
				m.skipReviewItem()
			} else {
				return m, m.cycleSelectedSessionStatus()
			}
		case ui.KeyUnskip:
			if m.viewMode == viewReview {
				m.unskipReviewItems()
			}
		case ui.KeyHermes:
			if m.viewMode == viewReview {
				return m, m.queryHermesForReviewItem()
			}
		case ui.KeyRelay:
			m.beginAgentControl()
		case ui.KeyMessage:
			m.beginCompose()
			if m.compose.active {
				return m, tea.ShowCursor
			}
			return m, nil
		case ui.KeySnapshot, ui.KeySnapshot2:
			m.beginSnapshotDescription()
			return m, tea.ShowCursor
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			if cmd := m.sendChoice(msg.String()); cmd != nil {
				return m, cmd
			}
		}
	case scanResult:
		m.carrySessionStatuses(msg.index, msg.snapshot)
		m.hosts[msg.index].snapshot = msg.snapshot
		m.hosts[msg.index].loading = false
		m.hosts[msg.index].loaded = true
		if m.scanAnnounce {
			m.status = m.scanStatus()
		}
		cmds := m.agentStatusCmds(msg.index)
		cmds = append(cmds, m.ensurePreview())
		return m, tea.Batch(cmds...)
	case attachFinished:
		if msg.err != nil {
			m.status = "attach failed: " + msg.err.Error()
		} else {
			m.status = "returned from tmux"
		}
	case captureResult:
		cached := m.cachePreview(msg.key, msg.capture)
		if m.preview.key == msg.key {
			m.preview.loading = false
			m.preview.refreshing = false
			m.preview.lines = append([]string(nil), cached.lines...)
			m.preview.err = cached.err
			m.preview.capturedAt = cached.capturedAt
		}
		var cmds []tea.Cmd
		if msg.capture.Err == "" {
			cmds = append(cmds, m.applyPreviewAgentStatus(msg.key, msg.capture.Lines))
		}
		if m.shouldPollPreview(msg.key) {
			cmds = append(cmds, previewTickCmd(msg.key))
			return m, tea.Batch(cmds...)
		}
		if len(cmds) > 0 {
			cmds = append(cmds, m.ensurePreview())
			return m, tea.Batch(cmds...)
		}
	case previewTick:
		if m.shouldPollPreview(msg.key) && !m.preview.loading && !m.preview.refreshing {
			selected, ok := m.activePreviewRow()
			if !ok {
				return m, nil
			}
			host := m.hosts[selected.hostIndex].host
			if len(m.preview.lines) == 0 && m.preview.err == "" {
				m.preview.loading = true
			} else {
				m.preview.refreshing = true
			}
			return m, captureCmd(msg.key, host, selected.attachTarget)
		}
	case scanTick:
		nextTick := scanTickCmd()
		if m.anyHostLoading() {
			return m, nextTick
		}

		next, scanCmd := m.startBackgroundScanModel()
		return next, tea.Batch(scanCmd, nextTick)
	case agentStatusResult:
		if msg.ok {
			oldStatus, hadOldStatus := m.statuses[msg.key]
			oldReviewKey := m.reviewCursorKey
			m.applyAgentStatusResult(msg)
			nextStatus := m.sessionStatusForKey(msg.key)
			if m.shouldLogPolledStatusChange(hadOldStatus, oldStatus, nextStatus) {
				m.addAgentActivity(agentActivity{
					Source:  agentActivitySession,
					Agent:   string(msg.target.agent),
					Target:  m.agentTargetDisplayLabel(msg.target),
					State:   statusLabel(nextStatus),
					Message: "status changed",
				})
			}
			if m.viewMode == viewReview {
				m.clampReviewCursor()
				if oldReviewKey != m.reviewCursorKey {
					m.preview = previewState{}
				}
			}
			autoCmd := m.autoHermesReviewCmd(hadOldStatus, oldStatus, nextStatus, msg.key)
			nextStepCmd := m.autoHermesNextStepCmd(hadOldStatus, oldStatus, nextStatus, msg.key)
			bellCmd := needReviewBellCmd(hadOldStatus, oldStatus, nextStatus, autoCmd != nil)
			if bellCmd != nil || autoCmd != nil || nextStepCmd != nil {
				return m, tea.Batch(bellCmd, autoCmd, nextStepCmd, m.ensurePreview())
			}
		}
	case hermesQueryResult:
		if !m.hermesQueryStillCurrent(msg) {
			return m, nil
		}
		if m.hermes == nil {
			m.hermes = map[string]hermesAdvice{}
		}
		advice := hermesAdvice{updatedAt: time.Now()}
		if msg.err != "" {
			advice.err = msg.err
			m.status = "Hermes query failed: " + clipString(msg.err, 80)
			entry := reviewHermesWorkLogEntry(msg.item, displayHostName(msg.host), autoModeLabel(msg.auto), "error")
			entry.Error = msg.err
			addEffectiveHermesConditions(&entry, msg.hermes)
			m.appendHermesWorkLog(entry)
			m.addAgentActivity(agentActivity{
				Source:  agentActivityReview,
				Agent:   "Hermes",
				Target:  m.reviewItemLabelByKey(msg.key),
				State:   "error",
				Message: clipString(msg.err, 80),
			})
		} else {
			advice.text = msg.text
			m.status = "Hermes replied"
			entry := reviewHermesWorkLogEntry(msg.item, displayHostName(msg.host), autoModeLabel(msg.auto), "reply")
			entry.Advice = msg.text
			addEffectiveHermesConditions(&entry, msg.hermes)
			m.appendHermesWorkLog(entry)
			m.addAgentActivity(agentActivity{
				Source:  agentActivityReview,
				Agent:   "Hermes",
				Target:  m.reviewItemLabelByKey(msg.key),
				State:   "replied",
				Message: hermesActivityAnswer(msg.text),
			})
		}
		m.hermes[msg.key] = advice
		if msg.auto && msg.err == "" {
			if cmd := m.applyHermesAutoReview(msg.item, displayHostName(msg.host), msg.lines, msg.text); cmd != nil {
				return m, cmd
			}
		}
	case hermesNextStepResult:
		if !m.hermesNextStepStillCurrent(msg) {
			return m, nil
		}
		if m.hermes == nil {
			m.hermes = map[string]hermesAdvice{}
		}
		advice := hermesAdvice{updatedAt: time.Now()}
		if msg.err != "" {
			advice.err = msg.err
			m.status = "Hermes next-step query failed: " + clipString(msg.err, 80)
			entry := nextStepHermesWorkLogEntry(msg, "error")
			entry.Error = msg.err
			addEffectiveHermesConditions(&entry, msg.hermes)
			m.appendHermesWorkLog(entry)
			m.addAgentActivity(agentActivity{
				Source:  agentActivitySession,
				Agent:   "Hermes",
				Target:  hermesNextStepTargetLabel(msg),
				State:   "error",
				Message: clipString(msg.err, 80),
			})
		} else {
			advice.text = msg.text
			m.status = "Hermes suggested next step"
			entry := nextStepHermesWorkLogEntry(msg, "reply")
			entry.Advice = msg.text
			addEffectiveHermesConditions(&entry, msg.hermes)
			m.appendHermesWorkLog(entry)
			m.addAgentActivity(agentActivity{
				Source:  agentActivitySession,
				Agent:   "Hermes",
				Target:  hermesNextStepTargetLabel(msg),
				State:   "replied",
				Message: hermesActivityAnswer(msg.text),
			})
		}
		m.hermes[msg.key] = advice
		if msg.auto && msg.err == "" {
			if cmd := m.applyHermesAutoNextStep(msg); cmd != nil {
				return m, cmd
			}
		}
	case memoryUpdateResult:
		if msg.err != "" {
			m.status = "memory update failed: " + clipString(msg.err, 100)
			entry := memoryHermesWorkLogEntry(msg.scope, "error")
			entry.Advice = msg.text
			entry.Error = msg.err
			m.appendHermesWorkLog(entry)
			m.addAgentActivity(agentActivity{
				Source:  agentActivitySession,
				Agent:   "Hermes",
				Target:  memoryScopeLabel(msg.scope),
				State:   "error",
				Message: clipString(msg.err, 80),
			})
		} else {
			m.status = "memory updated: " + msg.path
			entry := memoryHermesWorkLogEntry(msg.scope, "memory_write")
			entry.Advice = msg.text
			entry.Modified = true
			entry.ModifiedPath = msg.path
			m.appendHermesWorkLog(entry)
			m.addAgentActivity(agentActivity{
				Source:  agentActivitySession,
				Agent:   "Hermes",
				Target:  memoryScopeLabel(msg.scope),
				State:   "memory",
				Message: "wrote " + msg.path,
			})
		}
	case sessionOperationResult:
		target := sessionCloseConfirmationToken(msg.host, msg.session)
		if msg.err != "" {
			m.status = msg.action + " failed: " + clipString(msg.err, 100)
			m.addAgentActivity(agentActivity{
				Source:  agentActivitySession,
				Agent:   "tmux",
				Target:  target,
				State:   "error",
				Message: msg.action + ": " + clipString(msg.err, 80),
			})
			return m, nil
		}
		switch msg.action {
		case "session-open":
			if msg.created {
				m.status = "session opened: " + target
			} else {
				m.status = "session already exists: " + target
			}
		case "session-close":
			if msg.closed {
				m.status = "session closed: " + target
			} else {
				m.status = "session close skipped: " + target
			}
			m.preview = previewState{}
		}
		m.addAgentActivity(agentActivity{
			Source:  agentActivitySession,
			Agent:   "tmux",
			Target:  target,
			State:   "session",
			Message: msg.action,
		})
		next, cmd := m.startScanModelWithAnnounce(false)
		return next, cmd
	case sendResult:
		targetLabel := m.sendResultDisplayLabel(msg.result)
		if msg.result.Err != "" {
			m.status = msg.action + " failed: " + msg.result.Err
			m.addAgentActivity(agentActivity{
				Source:  agentActivitySession,
				Agent:   "session",
				Target:  targetLabel,
				State:   "error",
				Message: msg.action + ": " + clipString(msg.result.Err, 80),
			})
		} else {
			m.status = msg.action + " sent to " + targetLabel
			m.preview = previewState{}
			m.addAgentActivity(agentActivity{
				Source:  agentActivitySession,
				Agent:   "session",
				Target:  targetLabel,
				State:   "sent",
				Message: msg.action,
			})
		}
	case qqNotifyResult:
		result := msg.result
		switch {
		case result.Error != "":
			m.status = "QQ notification failed: " + clipString(result.Error, 80)
		case result.Sent:
			m.status = fmt.Sprintf("QQ notification sent (%d need review)", result.NeedsReviewCount)
		default:
			m.status = "QQ notification skipped: " + result.Reason
		}
	case snapshotResult:
		if msg.err != "" {
			m.status = "snapshot failed: " + clipString(msg.err, 100)
		} else {
			m.status = "snapshot saved: " + msg.path
		}
	}

	return m, m.ensurePreview()
}

func (m model) View() string {
	if m.skipRender {
		if view, ok := cachedTUIView(); ok {
			return view
		}
	}

	beginTUIViewRender()
	clearTUIInputCursor()
	var out strings.Builder

	headerWidth := maxInt(60, m.width-2)
	out.WriteString(m.renderHeader(headerWidth))

	contentWidth := maxInt(60, m.width-4)
	contentTopRow := m.headerHeight()
	contentHeight := maxInt(18, m.height-contentTopRow)
	if m.viewMode == viewReview {
		out.WriteString(m.renderReviewView(contentWidth, contentHeight, contentTopRow, 1))
		out.WriteString("\n")
		view := out.String()
		finishTUIViewRender(view)
		return view
	}

	if contentWidth >= 140 {
		kanbanWidth := threeColumnSideWidth(contentWidth)
		activityWidth := threeColumnActivityWidth(contentWidth, kanbanWidth)
		kanban := m.renderKanban(kanbanWidth, contentHeight)
		workspaceLeftCol := lipgloss.Width(kanban) + 3
		workspaceWidth := maxInt(60, contentWidth-kanbanWidth-activityWidth-4)
		workspace := m.renderWorkspace(workspaceWidth, contentHeight, contentTopRow, workspaceLeftCol)
		activity := m.renderAgentActivity(activityWidth, contentHeight)
		out.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, kanban, "  ", workspace, "  ", activity))
	} else if m.width >= 104 {
		kanbanWidth := twoColumnSideWidth(contentWidth)
		kanban := m.renderKanban(kanbanWidth, contentHeight)
		workspaceLeftCol := lipgloss.Width(kanban) + 3
		workspace := m.renderWorkspace(maxInt(60, contentWidth-kanbanWidth-2), contentHeight, contentTopRow, workspaceLeftCol)
		out.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, kanban, "  ", workspace))
	} else {
		kanbanHeight := 0
		totalPanelHeight := contentHeight - 2
		if contentHeight >= 34 {
			totalPanelHeight = contentHeight - 4
			kanbanHeight = maxInt(8, totalPanelHeight/5)
		}
		hostHeight, previewHeight := splitWorkspaceHeights(totalPanelHeight - kanbanHeight)

		out.WriteString(m.renderHosts(contentWidth, hostHeight))
		out.WriteString("\n\n")
		out.WriteString(m.renderPreviewPanel(contentWidth, previewHeight, contentTopRow+hostHeight+2, 1))
		if kanbanHeight > 0 {
			out.WriteString("\n\n")
			out.WriteString(m.renderKanban(contentWidth, kanbanHeight))
		}
	}
	out.WriteString("\n")

	view := out.String()
	finishTUIViewRender(view)
	return view
}

func hostKey(index int) string {
	return fmt.Sprintf("host:%d", index)
}

func sessionKey(hostIndex int, sessionID string) string {
	return fmt.Sprintf("host:%d:session:%s", hostIndex, sessionID)
}

func windowKey(hostIndex int, sessionID string, windowID string) string {
	return fmt.Sprintf("host:%d:session:%s:window:%s", hostIndex, sessionID, windowID)
}

func previewKey(selected row) string {
	return selected.key + ":preview:" + selected.attachTarget
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func clampInt(value int, low int, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func panelInnerHeight(outerHeight int) int {
	return maxInt(1, outerHeight-4)
}

func clipString(value string, width int) string {
	if width <= 0 {
		return ""
	}

	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width <= 3 {
		return string(runes[:width])
	}
	return string(runes[:width-3]) + "..."
}
