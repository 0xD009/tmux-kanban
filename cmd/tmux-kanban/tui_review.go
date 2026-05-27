package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"tmux-kanban/internal/core"
	"tmux-kanban/internal/tmuxscan"
)

func (m *model) toggleViewMode() {
	if m.viewMode == viewReview {
		m.setViewMode(viewTree)
	} else {
		m.setViewMode(viewReview)
	}
}

func (m model) reviewItems() []reviewItem {
	items := make([]reviewItem, 0)
	for hostIndex, state := range m.hosts {
		if !state.loaded || state.snapshot.Err != "" {
			continue
		}
		for sessionIndex, session := range state.snapshot.Sessions {
			sessionKeyValue := sessionStatusKey(state.host, session)
			if m.sessionStatusForKey(sessionKeyValue) != sessionNeedReview {
				continue
			}

			target, ok := m.reviewTargetForSession(sessionKeyValue, hostIndex, session)
			if !ok {
				continue
			}
			items = append(items, reviewItem{
				SessionKey:  sessionKeyValue,
				HostName:    state.host.Name,
				SessionName: session.Name,
				Agent:       target.agent,
				Target:      target,
				Row: row{
					key:          target.key,
					kind:         rowPane,
					hostIndex:    hostIndex,
					sessionIndex: sessionIndex,
					label:        state.host.Name + "/" + session.Name,
					attachTarget: target.target,
					agent:        target.agent,
				},
			})
		}
	}
	return items
}

func (m model) reviewTargetForSession(sessionKeyValue string, hostIndex int, session tmuxscan.Session) (selectedAgentTarget, bool) {
	if target, ok := m.reviewTargets[sessionKeyValue]; ok && target.target != "" && target.hostIndex == hostIndex && targetExistsInSession(target.target, session) {
		return target, true
	}
	return firstAgentTargetInSession(hostIndex, sessionKey(hostIndex, session.ID), session)
}

func (m model) reviewQueue() []reviewItem {
	items := m.reviewItems()
	if len(m.reviewSkipped) == 0 {
		return items
	}

	queue := make([]reviewItem, 0, len(items))
	for _, item := range items {
		if !m.reviewSkipped[item.SessionKey] {
			queue = append(queue, item)
		}
	}
	return queue
}

func reviewQueueKeys(items []reviewItem) []string {
	keys := make([]string, 0, len(items))
	for _, item := range items {
		keys = append(keys, item.SessionKey)
	}
	return keys
}

func (m model) skippedReviewCount() int {
	if len(m.reviewSkipped) == 0 {
		return 0
	}

	count := 0
	for _, item := range m.reviewItems() {
		if m.reviewSkipped[item.SessionKey] {
			count++
		}
	}
	return count
}

func (m model) currentReviewItem() (reviewItem, bool) {
	items := m.reviewQueue()
	if len(items) == 0 {
		return reviewItem{}, false
	}
	return items[m.reviewCursorIndex(items)], true
}

func (m model) reviewCursorIndex(items []reviewItem) int {
	return core.ReviewQueue{Cursor: m.reviewCursor, CursorKey: m.reviewCursorKey}.CursorIndex(reviewQueueKeys(items))
}

func (m *model) clampReviewCursor() {
	items := m.reviewQueue()
	oldKey := m.reviewCursorKey
	queue := core.ReviewQueue{Cursor: m.reviewCursor, CursorKey: m.reviewCursorKey}.Clamp(reviewQueueKeys(items))
	m.reviewCursor = queue.Cursor
	m.reviewCursorKey = queue.CursorKey
	if oldKey != "" && oldKey != m.reviewCursorKey && m.viewMode == viewReview {
		m.preview = previewState{}
	}
}

func (m *model) moveReviewCursor(delta int) {
	items := m.reviewQueue()
	queue, moved := (core.ReviewQueue{Cursor: m.reviewCursor, CursorKey: m.reviewCursorKey}).Move(reviewQueueKeys(items), delta)
	if moved {
		m.preview = previewState{}
		m.resetPreviewScroll()
	}
	m.reviewCursor = queue.Cursor
	m.reviewCursorKey = queue.CursorKey
}

func (m *model) advanceReviewCursorAfter(currentKey string) {
	items := m.reviewQueue()
	queue := core.ReviewQueue{Cursor: m.reviewCursor, CursorKey: m.reviewCursorKey}.AdvanceAfter(reviewQueueKeys(items), currentKey)
	m.reviewCursor = queue.Cursor
	m.reviewCursorKey = queue.CursorKey
}

func (m *model) skipReviewItem() {
	item, ok := m.currentReviewItem()
	if !ok {
		m.status = "review queue is empty"
		return
	}
	if m.reviewSkipped == nil {
		m.reviewSkipped = map[string]bool{}
	}
	m.reviewSkipped[item.SessionKey] = true
	m.advanceReviewCursorAfter(item.SessionKey)
	m.preview = previewState{}
	m.status = "skipped " + item.HostName + "/" + item.SessionName
	m.addAgentActivity(agentActivity{
		Source:  agentActivityReview,
		Agent:   "review queue",
		Target:  item.HostName + "/" + item.SessionName,
		State:   "skipped",
		Message: "review item skipped",
	})
}

func (m *model) unskipReviewItems() {
	m.reviewSkipped = map[string]bool{}
	m.clampReviewCursor()
	m.preview = previewState{}
	m.status = "review queue restored"
	m.addAgentActivity(agentActivity{
		Source:  agentActivityReview,
		Agent:   "review queue",
		Target:  "all skipped review items",
		State:   "restored",
		Message: "skipped items restored",
	})
}

func (m *model) clearHermesAdvice(key string) {
	if m.hermes == nil {
		return
	}
	delete(m.hermes, key)
}

func (m model) hermesQueryStillCurrent(msg hermesQueryResult) bool {
	if m.sessionStatusForKey(msg.key) != sessionNeedReview {
		return false
	}
	target, ok := m.reviewTargets[msg.key]
	if !ok {
		return false
	}
	return target.target == msg.item.Row.attachTarget
}

func hermesActivityAnswer(text string) string {
	lines := compactTextLines(text, 180, 4)
	if len(lines) == 0 {
		return "<empty Hermes response>"
	}
	return strings.Join(lines, "\n")
}

func (m *model) queryHermesForReviewItem() tea.Cmd {
	item, ok := m.currentReviewItem()
	if !ok {
		m.status = "review queue is empty"
		return nil
	}
	hermesCfg, ok := m.hermesConfigForReviewItem(item)
	if !ok {
		m.status = "review item host is not available"
		return nil
	}
	if !hermesCfg.Enabled {
		m.status = "Hermes is disabled in config"
		return nil
	}
	if strings.TrimSpace(hermesCfg.Command) == "" {
		m.status = "Hermes command is not configured"
		return nil
	}

	if m.hermes == nil {
		m.hermes = map[string]hermesAdvice{}
	}
	m.hermes[item.SessionKey] = hermesAdvice{loading: true}
	m.status = "asking Hermes about " + item.HostName + "/" + item.SessionName
	m.addAgentActivity(agentActivity{
		Source:  agentActivityReview,
		Agent:   "Hermes",
		Target:  item.HostName + "/" + item.SessionName,
		State:   "thinking",
		Message: "review requested",
	})
	host := m.hosts[item.Row.hostIndex].host
	return hermesQueryCmd(configWithHermes(m.cfg, hermesCfg), item, host, false)
}

func (m model) reviewItemByKey(key string) (reviewItem, bool) {
	for _, item := range m.reviewItems() {
		if item.SessionKey == key {
			return item, true
		}
	}
	return reviewItem{}, false
}

func (m model) activePreviewRow() (row, bool) {
	if m.viewMode == viewReview {
		item, ok := m.currentReviewItem()
		if !ok {
			return row{}, false
		}
		return item.Row, true
	}
	if target, _, ok := m.selectedReviewAgentTarget(); ok {
		if targetRow, ok := m.rowForAgentTarget(target); ok {
			return targetRow, true
		}
	}
	return m.selectedRow()
}

func (m model) activeAgentTarget() (selectedAgentTarget, bool) {
	if m.viewMode == viewReview {
		item, ok := m.currentReviewItem()
		if !ok {
			return selectedAgentTarget{}, false
		}
		return item.Target, true
	}
	if target, _, ok := m.selectedReviewAgentTarget(); ok {
		return target, true
	}
	return m.selectedAgentTarget()
}
