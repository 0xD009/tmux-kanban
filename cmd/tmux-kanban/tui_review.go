package main

import (
	"strings"

	"tmux-kanban/internal/tmuxscan"
)

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

func (m model) reviewItemByKey(key string) (reviewItem, bool) {
	for _, item := range m.reviewItems() {
		if item.SessionKey == key {
			return item, true
		}
	}
	return reviewItem{}, false
}

func (m model) activePreviewRow() (row, bool) {
	if target, _, ok := m.selectedReviewAgentTarget(); ok {
		if targetRow, ok := m.rowForAgentTarget(target); ok {
			return targetRow, true
		}
	}
	return m.selectedRow()
}

func (m model) activeAgentTarget() (selectedAgentTarget, bool) {
	if target, _, ok := m.selectedReviewAgentTarget(); ok {
		return target, true
	}
	return m.selectedAgentTarget()
}
