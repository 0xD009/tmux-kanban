# Snapshot Schema Notes

## Top-Level Fields

- `version`: Snapshot schema version.
- `created_at`: Snapshot creation time.
- `description`: Human-entered note describing why the snapshot was saved. Use it as the debugging hypothesis.
- `config`: Summarized tmux-kanban config.
- `runtime`: TUI/CLI runtime state.
- `hosts`: Host scans and tmux sessions.
- `review_queue`: Items classified as current need-review work.
- `agent_activities`: Recent TUI activities such as status changes, choices, messages, Hermes results.
- `preview`: TUI preview capture for the selected row.
- `errors`: Host scan or snapshot errors.

## Runtime Fields

- `view_mode`: `tree`, `review`, or `cli`.
- `status`: Current TUI status line, or `snapshot` for CLI snapshots.
- `session_statuses`: Classification map.
  - TUI keys often look like `nebula:$27`.
  - CLI keys often look like `nebula/%31`.
- `review_targets`: Map from session key to pane target for current review items.
- `skipped_review`: Review keys hidden by the user.
- `review_cursor` and `review_cursor_key`: Current review selection.

## Review Item Fields

- `session_key`: Key matching runtime state.
- `host`: Host name.
- `session_name`: tmux session name.
- `agent`: Detected agent, usually `codex` or `claude-code`.
- `target`: tmux target, often a pane like `%31`.
- `screen_status`: Classifier output.
- `needs_review`: Whether the screen was actionable review.
- `capture`: Captured lines when included.

## Preview Fields

- `key`: TUI preview key for selected row.
- `host_index`: Host index in `.hosts`.
- `target`: tmux target captured by preview.
- `loading` / `refreshing`: Preview state at snapshot time.
- `error`: Capture error.
- `captured_at`: Time of preview capture.
- `lines`: Captured terminal lines.

## Useful jq Commands

Latest TUI/CLI summary:

```bash
jq '{created_at, description, view_mode: .runtime.view_mode, status: .runtime.status, session_statuses: .runtime.session_statuses, review_queue_len: (.review_queue|length), preview: {target: .preview.target, captured_at: .preview.captured_at, error: .preview.error}}' <snapshot.json>
```

Activities:

```bash
jq '.agent_activities[]? | {at, source, agent, target, state, message}' <snapshot.json>
```

Review queue:

```bash
jq '.review_queue[]? | {session_key, host, session_name, agent, target, screen_status, needs_review}' <snapshot.json>
```

Map host sessions:

```bash
jq '.hosts[] | {host: .name, sessions: [.sessions[] | {id, name, windows: [.windows[] | {id, index, name, panes}] }]}' <snapshot.json>
```

Find one session by key suffix:

```bash
jq '.hosts[] | select(.name=="nebula") | .sessions[] | select(.id=="$27")' <snapshot.json>
```

Preview tail:

```bash
jq -r '.preview.lines[-40:][]?' <snapshot.json>
```
