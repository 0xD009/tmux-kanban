package main

import (
	"fmt"
	"os"
)

func isCLICommand(arg string) bool {
	switch arg {
	case "capabilities", "review-list", "notify-review", "capture", "choose", "send", "send-keys", "snapshot", "help":
		return true
	default:
		return false
	}
}

func runCLI(args []string) error {
	if len(args) == 0 || args[0] == "help" {
		printCLIUsage(os.Stdout)
		return nil
	}

	switch args[0] {
	case "capabilities":
		return cliCapabilities(args[1:])
	case "review-list":
		return cliReviewList(args[1:])
	case "notify-review":
		return cliNotifyReview(args[1:])
	case "capture":
		return cliCapture(args[1:])
	case "choose":
		return cliChoose(args[1:])
	case "send":
		return cliSend(args[1:])
	case "send-keys":
		return cliSendKeys(args[1:])
	case "snapshot":
		return cliSnapshot(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}
