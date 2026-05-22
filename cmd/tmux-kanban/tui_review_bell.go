package main

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

var writeTerminalBell = func() {
	_, _ = os.Stdout.Write(needReviewTerminalAlertSequence())
}

const needReviewTerminalTitle = "tmux-kanban: NEED REVIEW"

func needReviewTerminalAlertSequence() []byte {
	return []byte("\a\x1b]1;" + needReviewTerminalTitle + "\x1b\\\x1b]2;" + needReviewTerminalTitle + "\x1b\\")
}

func needReviewBellCmd(hadOld bool, oldStatus sessionStatus, nextStatus sessionStatus, handledByHermes bool) tea.Cmd {
	if handledByHermes || !enteredNeedReview(hadOld, oldStatus, nextStatus) {
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
