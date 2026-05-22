---
name: tmux-kanban-snapshot-debug
description: Diagnose tmux-kanban debug snapshots and status misclassification bugs. Use when Codex is asked to inspect a tmux-kanban snapshot, explain why an agent/session was marked idle/working/need review/done, compare snapshot state with live tmux panes, debug review queue or preview inconsistencies, or use snapshot descriptions to reproduce classification problems.
---

# tmux-kanban Snapshot Debug

## Default Paths

Work from the repo unless the user gives another checkout:

```bash
cd /Users/0xd009/Documents/Projects/ideas/tmux-kanban
```

Snapshots usually live in:

```bash
/Users/0xd009/.local/state/tmux-kanban/snapshots
```

Resolved snapshots should be moved under:

```bash
/Users/0xd009/.local/state/tmux-kanban/snapshots/archive
```

Prefer the repo-local binary:

```bash
./bin/tmux-kanban
```

If source was changed, rebuild both common entrypoints before verifying:

```bash
go build -o bin/tmux-kanban ./cmd/tmux-kanban
go build -o tmux-kanban ./cmd/tmux-kanban
```

## Debug Workflow

1. Identify the exact snapshot.

If the user says "latest", use `ls -lt` on the snapshot directory and inspect the newest file by timestamp. Do not assume the newest named file is the relevant one without checking.

2. Read the human note first.

Check top-level `description`. Treat it as the user's claim or observation, not as proof.

```bash
jq '{created_at, description, runtime: .runtime, review_queue_len: (.review_queue|length)}' <snapshot.json>
```

3. Determine snapshot source.

- TUI snapshots usually have `runtime.view_mode` as `tree` or `review` and status keys like `nebula:$27`.
- CLI snapshots usually have `runtime.view_mode: "cli"` and status keys like `nebula/%31`.

This matters because session-key snapshots must be mapped through `.hosts[].sessions[]`, while pane-key snapshots already name the target pane.

4. Map sessions and panes.

For TUI snapshots, map `host:$session_id` to session name and panes:

```bash
jq '.hosts[] | {host: .name, sessions: [.sessions[] | {id, name, windows}]}' <snapshot.json>
```

For a specific host:

```bash
jq '.hosts[] | select(.name=="nebula") | .sessions[] | {id, name, windows}' <snapshot.json>
```

5. Compare the state layers.

Check, in this order:

- `runtime.session_statuses`: what the UI/CLI believed.
- `review_queue`: what was considered actionable review.
- `runtime.review_targets`: pane target remembered for review items.
- `preview`: what the TUI had captured for the selected target.
- `agent_activities`: status transition history and user actions like sent choices/messages.
- `hosts[].sessions[]`: current topology as of the snapshot.

6. Capture live panes with the current binary when investigating a misclassification.

Use the pane target from the snapshot. For example:

```bash
./bin/tmux-kanban capture --config ./config.yaml --host nebula --target %31 --height 120
./bin/tmux-kanban capture --config ./config.yaml --host nebula --target %32 --height 120
```

Trust the JSON `screen` fields over visual guesses:

- `screen.status`
- `screen.busy`
- `screen.idle`
- `screen.needs_review`
- `screen.choices`

7. Generate a fresh CLI snapshot to separate old TUI state from current parser behavior.

```bash
./bin/tmux-kanban snapshot --config ./config.yaml --description "debug: checking <issue>"
```

If the fresh CLI snapshot is correct but the TUI snapshot is wrong, suspect an old running TUI process, stale in-memory state, or TUI-only status carryover. If the fresh CLI snapshot is also wrong, suspect parser/classifier logic.

8. Archive each snapshot after its issue is resolved.

Once a snapshot has been diagnosed, fixed, and verified, move that exact snapshot into an archive subdirectory. Resolve and archive snapshots one at a time so the active snapshot directory only contains unresolved evidence.

```bash
mkdir -p /Users/0xd009/.local/state/tmux-kanban/snapshots/archive/<YYYYMMDD-HHMMSS>
mv /Users/0xd009/.local/state/tmux-kanban/snapshots/<snapshot-file>.json /Users/0xd009/.local/state/tmux-kanban/snapshots/archive/<YYYYMMDD-HHMMSS>/
```

## Common Diagnoses

**Marked `done` while pane is still working**

Likely causes:

- A parser missed a live `Working (...)`, `Running (...)`, `Thinking (...)`, or background terminal line.
- The TUI saw `working -> idle` and converted it to `done`.
- A running TUI process predates a source/binary fix.

Required evidence:

- Snapshot `runtime.session_statuses` for the affected key.
- Any `agent_activities` showing transition into `done`.
- Live `capture` JSON for the affected pane.
- Binary timestamps from `ls -l bin/tmux-kanban tmux-kanban` if staleness is possible.

**Not detected as `need review`**

Likely causes:

- The review prompt scrolled above the captured tail.
- Only a selected choice is visible, and the choice label is not recognized as a review action.
- Checklist or plan text was intentionally ignored to avoid false positives.

Required evidence:

- `screen.choices` and `screen.selected_choice`.
- Tail lines around the visible prompt.
- Whether the prompt text is visible or only choices remain.

**TUI snapshot and CLI snapshot disagree**

Likely causes:

- TUI uses session IDs and carries state over time; CLI uses current pane captures.
- Old TUI process has not been restarted after rebuild.
- TUI preview is stale or pointed at a different row.

Required evidence:

- `runtime.view_mode`, `runtime.status`, and status key format.
- `preview.key`, `preview.target`, and `preview.captured_at`.
- Fresh CLI snapshot with description.

## Response Format

When reporting back, keep it evidence-first:

1. State the exact snapshot file and `created_at`.
2. Quote or summarize the `description`.
3. List affected keys and mapped session/pane names.
4. Explain the transition or inconsistency.
5. Say whether live capture confirms or contradicts the snapshot.
6. Name the likely fix area, such as parser, status state machine, preview sync, or stale TUI process.
7. If the issue was fixed and verified, state the archive path where the snapshot was moved.

Use file and line references only after inspecting source. Do not claim a code fix unless tests and a rebuilt binary verify it.

## Extra Reference

For field meanings and useful jq snippets, read `references/snapshot-schema.md`.
