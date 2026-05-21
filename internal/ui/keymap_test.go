package ui

import "testing"

func TestInputBarPrompt(t *testing.T) {
	tests := []struct {
		bar  InputBar
		want string
	}{
		{bar: InputBar{Mode: InputCommand}, want: ":"},
		{bar: InputBar{Mode: InputMessage, Label: "local/agents:0.1"}, want: "message to local/agents:0.1: "},
		{bar: InputBar{Mode: InputMessage, Target: "%1"}, want: "message to selected target: "},
	}

	for _, tt := range tests {
		if got := tt.bar.Prompt(); got != tt.want {
			t.Fatalf("Prompt() = %q, want %q", got, tt.want)
		}
	}
}
