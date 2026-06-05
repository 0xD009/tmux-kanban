package tmuxscan

import "testing"

func TestAnalyzeAgentScreenDetectsNumberedChoices(t *testing.T) {
	screen := AnalyzeAgentScreen([]string{
		"Do you want to allow this command?",
		"  1. Yes",
		"❯ 2. Yes, and don't ask again",
		"  3. No",
	})

	if !screen.Idle {
		t.Fatalf("screen idle = false, want true")
	}
	if len(screen.Choices) != 3 {
		t.Fatalf("got %d choices, want 3", len(screen.Choices))
	}
	if screen.SelectedChoice != 1 {
		t.Fatalf("selected choice = %d, want 1", screen.SelectedChoice)
	}
	if screen.Choices[1].Number != "2" {
		t.Fatalf("choice number = %q, want 2", screen.Choices[1].Number)
	}
	if !screen.NeedsReview {
		t.Fatalf("screen needs review = false, want true")
	}
}

func TestAnalyzeAgentScreenDetectsNumberedChoiceVariants(t *testing.T) {
	screen := AnalyzeAgentScreen([]string{
		"Permission required",
		"❯ [1] Allow once",
		"  2: Deny",
		"  3 - Always allow",
	})

	if len(screen.Choices) != 3 {
		t.Fatalf("got %d choices, want 3", len(screen.Choices))
	}
	if screen.SelectedChoice != 0 {
		t.Fatalf("selected choice = %d, want 0", screen.SelectedChoice)
	}
	if screen.Choices[0].Number != "1" || screen.Choices[0].Label != "Allow once" {
		t.Fatalf("choice[0] = %#v, want [1] Allow once", screen.Choices[0])
	}
	if screen.Choices[1].Number != "2" || screen.Choices[1].Label != "Deny" {
		t.Fatalf("choice[1] = %#v, want 2: Deny", screen.Choices[1])
	}
	if screen.Choices[2].Number != "3" || screen.Choices[2].Label != "Always allow" {
		t.Fatalf("choice[2] = %#v, want 3 - Always allow", screen.Choices[2])
	}
	if !screen.NeedsReview {
		t.Fatalf("screen needs review = false, want true")
	}
}

func TestAnalyzeAgentScreenDetectsUnnumberedChoiceBlock(t *testing.T) {
	screen := AnalyzeAgentScreen([]string{
		"Do you want to continue?",
		"❯ Yes",
		"  No",
	})

	if len(screen.Choices) != 2 {
		t.Fatalf("got %d choices, want 2", len(screen.Choices))
	}
	if screen.SelectedChoice != 0 {
		t.Fatalf("selected choice = %d, want 0", screen.SelectedChoice)
	}
	if screen.Choices[1].Label != "No" {
		t.Fatalf("second choice = %#v, want No", screen.Choices[1])
	}
	if !screen.NeedsReview {
		t.Fatalf("screen needs review = false, want true")
	}
}

func TestAnalyzeAgentScreenDetectsPlanImplementationPrompt(t *testing.T) {
	screen := AnalyzeAgentScreen([]string{
		"─ Worked for 2m 48s ─────────────────────────────────────────",
		"",
		"Implement this plan?",
		"",
		"› 1. Yes, implement this plan          Switch to Default and start coding.",
		"  2. Yes, clear context and implement  Fresh thread. Context: 61% used.",
		"  3. No, stay in Plan mode             Continue planning with the model.",
		"",
		"Press enter to confirm or esc to go back",
	})

	if !screen.NeedsReview {
		t.Fatalf("screen needs review = false, want true")
	}
	if len(screen.Choices) != 3 {
		t.Fatalf("choices = %d, want 3", len(screen.Choices))
	}
	if screen.SelectedChoice != 0 {
		t.Fatalf("selected choice = %d, want 0", screen.SelectedChoice)
	}
	if screen.Choices[0].Number != "1" {
		t.Fatalf("first choice = %#v, want numbered choice 1", screen.Choices[0])
	}
}

func TestAnalyzeAgentScreenDetectsReviewChoiceWhenPromptScrolledOff(t *testing.T) {
	screen := AnalyzeAgentScreen([]string{
		"  $ CUDA_VISIBLE_DEVICES=1 PYTHONPATH=/repo:$PYTHONPATH python run_fari.py",
		"  --wm_type TR --n 20 --steps 150 --validation_steps 50",
		"  --modelid_target /very/long/path/that/wraps/the/review/prompt/offscreen",
		"  --image_mse_weight 0.2 --tv_weight 0.001 --image_reg_interval 5",
		"",
		"› 1. Yes, proceed (y)",
		"  2. Yes, and don't ask again for commands that start with `CUDA_VISIBLE_DEVICES=1` (p)",
		"  3. No, and tell Codex what to do differently (esc)",
		"",
		"Press enter to confirm or esc to cancel",
	})

	if !screen.NeedsReview {
		t.Fatalf("screen needs review = false, want true")
	}
	if len(screen.Choices) != 3 {
		t.Fatalf("choices = %d, want 3", len(screen.Choices))
	}
	if screen.SelectedChoice != 0 {
		t.Fatalf("selected choice = %d, want 0", screen.SelectedChoice)
	}
}

func TestAnalyzeAgentScreenDetectsApprovalBeforeClaudeTodo(t *testing.T) {
	screen := AnalyzeAgentScreen([]string{
		"Read file",
		"",
		"  Read(/tmp/n6.png)",
		"",
		"Do you want to proceed?",
		"❯ 1. Yes",
		"  2. Yes, allow reading from tmp/ during this session",
		"  3. No",
		"",
		"Esc to cancel · Tab to amend",
		"",
		"  6 tasks (4 done, 1 in progress, 1 open)",
		"  ◼ Build natural-style PPT slides 3-9",
		"  ◻ Build natural-style Word report",
		"  ✔ Redesign PPT with stronger visual system",
		"  ✔ Redesign Word report typography & layout",
		"  ✔ Expand Word doc with deeper analysis",
		"  … +1 completed",
	})

	if !screen.NeedsReview {
		t.Fatalf("screen needs review = false, want true")
	}
	if len(screen.Choices) != 3 {
		t.Fatalf("choices = %d, want 3", len(screen.Choices))
	}
	if screen.SelectedChoice != 0 {
		t.Fatalf("selected choice = %d, want 0", screen.SelectedChoice)
	}
}

func TestAnalyzeAgentScreenDetectsCheckboxChoices(t *testing.T) {
	screen := AnalyzeAgentScreen([]string{
		"> [x] Edit file",
		"  [ ] Run tests",
	})

	if len(screen.Choices) != 2 {
		t.Fatalf("got %d choices, want 2", len(screen.Choices))
	}
	if !screen.Choices[0].Selected {
		t.Fatalf("first choice selected = false, want true")
	}
}

func TestAnalyzeAgentScreenDetectsBusy(t *testing.T) {
	screen := AnalyzeAgentScreen([]string{
		"working... esc to interrupt",
	})

	if screen.Idle {
		t.Fatalf("screen idle = true, want false")
	}
	if !screen.Busy {
		t.Fatalf("screen busy = false, want true")
	}
}

func TestAnalyzeAgentScreenDetectsBackgroundTerminalAsBusy(t *testing.T) {
	screen := AnalyzeAgentScreen([]string{
		"1 background terminal running · /ps to view · /stop to close",
		"",
		"› Write tests for @filename",
		"",
		"gpt-5.5 medium · ~/repo",
	})

	if !screen.Busy {
		t.Fatalf("screen busy = false, want true")
	}
	if !screen.Idle {
		t.Fatalf("screen idle = false, want true because prompt is still visible")
	}
}

func TestAnalyzeAgentScreenTreatsPromptAfterWorkingAsBusy(t *testing.T) {
	screen := AnalyzeAgentScreen([]string{
		"• Ran go test ./...",
		"",
		"◦ Working (43m 52s • esc to interrupt)",
		"",
		"› Run /review on my current changes",
		"",
		"gpt-5.5 xhigh · ~/repo",
	})

	if !screen.Busy {
		t.Fatalf("screen busy = false, want true while the current Codex status says working")
	}
	if !screen.Idle {
		t.Fatalf("screen idle = false, want true from visible prompt")
	}
	if screen.NeedsReview {
		t.Fatalf("screen needs review = true, want false")
	}
}

func TestAnalyzeAgentScreenTreatsCurrentWorkingStatusAsBusyWithDraftPrompt(t *testing.T) {
	screen := AnalyzeAgentScreen([]string{
		"• Waited for background terminal",
		"",
		"• Working (24m 57s • esc to interrupt)",
		"",
		"› Improve documentation in @filename",
		"",
		"gpt-5.5 medium · ~/repo",
	})

	if !screen.Busy {
		t.Fatalf("screen busy = false, want true")
	}
	if screen.NeedsReview {
		t.Fatalf("screen needs review = true, want false")
	}
}

func TestAnalyzeAgentScreenIgnoresHistoricalBusyWithIdlePrompt(t *testing.T) {
	screen := AnalyzeAgentScreen([]string{
		"◦ Working (1m 2s • esc to interrupt)",
		"old transcript 1",
		"old transcript 2",
		"old transcript 3",
		"old transcript 4",
		"old transcript 5",
		"old transcript 6",
		"old transcript 7",
		"old transcript 8",
		"old transcript 9",
		"old transcript 10",
		"old transcript 11",
		"old transcript 12",
		"old transcript 13",
		"old transcript 14",
		"old transcript 15",
		"› Explain this codebase",
		"gpt-5.5 high · ~/repo",
	})

	if screen.Busy {
		t.Fatalf("screen busy = true, want false")
	}
	if !screen.Idle {
		t.Fatalf("screen idle = false, want true")
	}
	if screen.NeedsReview {
		t.Fatalf("screen needs review = true, want false")
	}
}

func TestAnalyzeAgentScreenDoesNotTreatPromptTextContainingWorkingAsBusy(t *testing.T) {
	screen := AnalyzeAgentScreen([]string{
		"• 已验证测试通过",
		"",
		"› working结束的应该进入done啊，这点没有实现好",
		"",
		"gpt-5.5 xhigh · ~/repo",
	})

	if screen.Busy {
		t.Fatalf("screen busy = true, want false")
	}
	if !screen.Idle {
		t.Fatalf("screen idle = false, want true")
	}
}

func TestAnalyzeAgentScreenDoesNotTreatIdleMenuAsReview(t *testing.T) {
	screen := AnalyzeAgentScreen([]string{
		"Welcome back",
		"› New task",
		"  Recent sessions",
		"  ? for shortcuts",
	})

	if len(screen.Choices) == 0 {
		t.Fatalf("choices = 0, want idle menu choice parsed")
	}
	if !screen.Idle {
		t.Fatalf("screen idle = false, want true")
	}
	if screen.NeedsReview {
		t.Fatalf("screen needs review = true, want false")
	}
	if len(screen.Choices) != 2 || screen.Choices[0].Label != "New task" || screen.Choices[1].Label != "Recent sessions" {
		t.Fatalf("choices = %#v, want current idle menu choices only", screen.Choices)
	}
}

func TestAnalyzeAgentScreenIgnoresOldApprovalTranscript(t *testing.T) {
	screen := AnalyzeAgentScreen([]string{
		"Do you want to allow this command?",
		"  1. Yes",
		"❯ 2. Yes, and don't ask again",
		"  3. No",
		"✔ You approved codex to run tmux this time",
		"Ran tmux ls",
		"Everything completed",
		"────────────────────────────────",
		"你好啊",
		"› Find and fix a bug in @filename",
		"gpt-5.5 high · ~/nate",
	})

	if !screen.Idle {
		t.Fatalf("screen idle = false, want true")
	}
	if screen.NeedsReview {
		t.Fatalf("screen needs review = true, want false")
	}
	if len(screen.Choices) != 1 || screen.Choices[0].Label != "Find and fix a bug in @filename" {
		t.Fatalf("choices = %#v, want current idle prompt only", screen.Choices)
	}
}

func TestAnalyzeAgentScreenDoesNotExposeIdlePromptChoicesWhileBusy(t *testing.T) {
	screen := AnalyzeAgentScreen([]string{
		"• Working (45s • esc to interrupt)",
		"",
		"› Implement {feature}",
		"",
		"gpt-5.5 high · ~/repo",
	})

	if !screen.Busy {
		t.Fatalf("screen busy = false, want true")
	}
	if !screen.Idle {
		t.Fatalf("screen idle = false, want true because prompt is visible")
	}
	if len(screen.Choices) != 0 {
		t.Fatalf("choices = %#v, want no choices while busy", screen.Choices)
	}
	if screen.SelectedChoice != -1 {
		t.Fatalf("selected choice = %d, want -1", screen.SelectedChoice)
	}
	if screen.NeedsReview {
		t.Fatalf("screen needs review = true, want false")
	}
}

func TestAnalyzeAgentScreenDoesNotTreatPlanTodoAsReview(t *testing.T) {
	screen := AnalyzeAgentScreen([]string{
		"Plan:",
		"- [x] Inspect current parser",
		"- [ ] Continue hardening session detection",
		"- [ ] Apply changes to tmux sender",
		"",
		"Todo",
		"1. Confirm current behavior",
		"2. Run tests",
		"3. Continue polishing",
	})

	if screen.NeedsReview {
		t.Fatalf("screen needs review = true, want false")
	}
	if len(screen.Choices) != 0 {
		t.Fatalf("choices = %#v, want no interactive choices from plan/todo", screen.Choices)
	}
}

func TestAnalyzeAgentScreenDoesNotTreatChecklistWithoutPromptAsReview(t *testing.T) {
	screen := AnalyzeAgentScreen([]string{
		"[ ] Continue implementation",
		"[ ] Apply changes",
		"[x] Confirm tests pass",
	})

	if screen.NeedsReview {
		t.Fatalf("screen needs review = true, want false")
	}
}

func TestSendKeysRemoteCommandQuotesTargetAndKeys(t *testing.T) {
	command := sendKeysRemoteCommand("main:0.1", "Down", "C-m")
	want := "tmux send-keys -t 'main:0.1' -- 'Down' 'C-m'"
	if command != want {
		t.Fatalf("command = %q, want %q", command, want)
	}
}

func TestSendKeySequenceRemoteCommandAddsDelayBetweenKeys(t *testing.T) {
	command := sendKeySequenceRemoteCommand("%1", "Down", "C-m")
	want := "tmux send-keys -t '%1' -- 'Down' && sleep 0.06 && tmux send-keys -t '%1' -- 'C-m'"
	if command != want {
		t.Fatalf("command = %q, want %q", command, want)
	}
}

func TestSendTextRemoteCommandUsesPasteBuffer(t *testing.T) {
	command := sendTextRemoteCommand("main:0.1", "tmux-kanban-test", "hello 'world'\nnext", true)
	want := "tmux set-buffer -b 'tmux-kanban-test' -- 'hello '\\''world'\\''\nnext' && tmux paste-buffer -b 'tmux-kanban-test' -t 'main:0.1' && tmux delete-buffer -b 'tmux-kanban-test' && sleep 0.08 && tmux send-keys -t 'main:0.1' C-m"
	if command != want {
		t.Fatalf("command = %q, want %q", command, want)
	}
}

func TestSendTextRemoteCommandCanSubmitOnly(t *testing.T) {
	command := sendTextRemoteCommand("main:0.1", "tmux-kanban-test", "", true)
	want := "tmux send-keys -t 'main:0.1' C-m"
	if command != want {
		t.Fatalf("command = %q, want %q", command, want)
	}
}
