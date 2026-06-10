package main

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

var writeTerminalBell = func() {
	_, _ = os.Stdout.Write(needReviewTerminalAlertSequence())
}

var writeReviewTerminalTitle = func(active bool) {
	_, _ = os.Stdout.Write(reviewTerminalTitleSequence(active))
}

const needReviewTerminalTitle = "tmux-kanban: NEED REVIEW"
const defaultTerminalTitle = "tmux-kanban"

func needReviewTerminalAlertSequence() []byte {
	return []byte("\a\x1b]1;" + needReviewTerminalTitle + "\x1b\\\x1b]2;" + needReviewTerminalTitle + "\x1b\\")
}

func reviewTerminalTitleSequence(active bool) []byte {
	title := defaultTerminalTitle
	if active {
		title = needReviewTerminalTitle
	}
	return []byte("\x1b]1;" + title + "\x1b\\\x1b]2;" + title + "\x1b\\")
}

func needReviewBellCmd(bellEnabled bool, hadOld bool, oldStatus sessionStatus, nextStatus sessionStatus, handledByHermes bool) tea.Cmd {
	if !bellEnabled || handledByHermes || !enteredNeedReview(hadOld, oldStatus, nextStatus) {
		return nil
	}
	return func() tea.Msg {
		writeTerminalBell()
		return nil
	}
}

func enteredNeedReview(hadOld bool, oldStatus sessionStatus, nextStatus sessionStatus) bool {
	if normalizeSessionStatus(nextStatus) != sessionNeedReview {
		return false
	}
	if !hadOld {
		return true
	}
	return normalizeSessionStatus(oldStatus) != sessionNeedReview
}

func (m *model) syncReviewTerminalTitleCmd() tea.Cmd {
	if !m.cfg.Notification.TerminalReview {
		if !m.reviewTitleActive {
			return nil
		}
		m.reviewTitleActive = false
		return func() tea.Msg {
			writeReviewTerminalTitle(false)
			return nil
		}
	}

	active := len(m.reviewQueue()) > 0
	if m.reviewTitleActive == active {
		return nil
	}
	m.reviewTitleActive = active
	return func() tea.Msg {
		writeReviewTerminalTitle(active)
		return nil
	}
}
