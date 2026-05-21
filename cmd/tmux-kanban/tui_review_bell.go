package main

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

var writeTerminalBell = func() {
	_, _ = os.Stdout.Write([]byte{'\a'})
}

func needReviewBellCmd(hadOld bool, oldStatus sessionStatus, nextStatus sessionStatus) tea.Cmd {
	if !enteredNeedReview(hadOld, oldStatus, nextStatus) {
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
