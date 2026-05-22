---
name: tmux-kanban-dev
description: "Develop the local tmux-kanban Go project. Use when changing architecture, core state logic, CLI commands, config loading, debug snapshots, tests, or general code in this repository."
---

# tmux-kanban Dev

## Repo

Default checkout:

```bash
cd /Users/0xd009/Documents/Projects/ideas/tmux-kanban
```

Important files and packages:

- `cmd/tmux-kanban/main.go`: small entrypoint.
- `cmd/tmux-kanban/tui.go`: Bubble Tea TUI implementation.
- `cmd/tmux-kanban/cli.go`: JSON CLI commands for agents.
- `internal/core`: pure session status and review queue logic.
- `internal/agent`: agent screen analysis wrappers and external reviewer interfaces.
- `internal/tmux` and `internal/tmuxscan`: tmux client boundary and command implementation.
- `internal/config`: YAML config and defaults.
- `internal/debug`: snapshot schema and writer.
- `internal/ui`: shared key/input primitives.
- `internal/mesh`: agent mesh scaffold.

## Workflow

1. Inspect with `rg` before editing. The codebase is mid-refactor, so do not assume old `main.go` contains behavior.
2. Keep changes incremental and package-local. Prefer pure helpers in `internal/core`, `internal/agent`, or `internal/mesh` over growing `tui.go` when behavior is testable outside UI.
3. Preserve existing UX unless the user explicitly asks to change it. Do not restore removed tmux popup behavior.
4. For manual edits, use `apply_patch`.
5. After Go changes, run:

```bash
gofmt -w <changed-go-files>
go test ./...
go build -o ./bin/tmux-kanban ./cmd/tmux-kanban
```

## Testing Bias

- Pure state changes need table tests.
- TUI command/key changes need tests through `executeCommand`, `Update`, or small render helpers.
- tmux command construction needs tests without requiring a live tmux server.
- Snapshot/config changes should be reflected in `internal/debug` or `internal/config` tests.

