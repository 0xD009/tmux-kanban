---
name: tmux-kanban-ui
description: "Work on tmux-kanban's Bubble Tea TUI, including layout, preview panels, input/message boxes, keybindings, kanban/session/review views, agent activity, and terminal UX polish."
---

# tmux-kanban UI

## Entry Points

Work in:

- `cmd/tmux-kanban/tui.go`
- `internal/ui/keymap.go`
- `internal/ui/input.go` if present
- `cmd/tmux-kanban/main_test.go`

Run from:

```bash
cd /Users/0xd009/Documents/Projects/ideas/tmux-kanban
```

## Current UI Model

- Tree view: left kanban, middle sessions/preview, right agent activity on wide screens.
- Review view: review queue, preview, right agent activity on wide screens.
- Main session: `g` or `:main start` pins preview to the configured cockpit session.
- Preview refresh is cached and polls about once per second.
- Message and command input share the preview-bottom input box and must keep cursor movement smooth.

## UX Rules

- Keep panel bottoms aligned; tests often assert `lipgloss.Height`.
- Avoid flashing tags or loading text during background polling.
- Do not expose raw tmux pane IDs like `%1` in user-facing status if a friendly label can be resolved.
- Keep Chinese input/paste behavior intact: composition should not stutter, and cursor movement should stay local when possible.
- Do not move the message box to a global bottom bar unless the user explicitly asks.
- Do not reintroduce tmux popup.

## Validation

Use focused tests for render helpers and key flows, then full verification:

```bash
go test ./...
go build -o ./bin/tmux-kanban ./cmd/tmux-kanban
```

