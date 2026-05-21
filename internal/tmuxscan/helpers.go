package tmuxscan

import (
	"strconv"
	"strings"
)

func splitRecord(line string) []string {
	if strings.Contains(line, "\t") {
		return strings.Split(line, "\t")
	}
	return strings.Split(line, `\t`)
}

func compactError(err error, output []byte) string {
	message := strings.TrimSpace(string(output))
	if message == "" {
		return err.Error()
	}

	message = strings.Join(strings.Fields(message), " ")
	if len(message) > 180 {
		message = message[:180] + "..."
	}
	return message
}

func parseBool(value string) bool {
	return value == "1" || strings.EqualFold(value, "true")
}

func parseInt(value string) int {
	number, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return number
}

func numericLess(left, right string) bool {
	leftNumber, leftErr := strconv.Atoi(left)
	rightNumber, rightErr := strconv.Atoi(right)
	if leftErr == nil && rightErr == nil {
		return leftNumber < rightNumber
	}
	return left < right
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func normalizeCaptureLines(output string) []string {
	output = strings.ReplaceAll(output, "\r\n", "\n")
	output = strings.TrimRight(output, "\n")
	if output == "" {
		return []string{"<empty pane>"}
	}

	lines := strings.Split(output, "\n")
	for len(lines) > 1 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	return lines
}
