---
name: tmux-kanban-config
description: "Write, edit, or review tmux-kanban configuration. Use when changing config.yaml, config.example.yaml, internal/config defaults or YAML loading, runtime :set/:settings behavior, Hermes scoped settings, agent_mesh/main_agent config, snapshot config summaries, or config documentation/tests."
---

# tmux-kanban Config

## Scope

Use this skill for changes to:

- `config.yaml` or `config.example.yaml`
- `internal/config/config.go` and `internal/config/config_test.go`
- Runtime config commands in `cmd/tmux-kanban/tui_command_*.go`
- Config output in `:settings`, CLI capabilities, or debug snapshots
- README config examples and command docs

## Principles

- `config.example.yaml` is the tracked public example. Personal values belong in local `config.yaml`; preserve existing private hostnames, paths, and notification settings unless the user asks to change them.
- Runtime TUI commands mutate only the in-memory config for the current process. Do not imply that `:set ...`, `:mesh ...`, or similar commands persist to `config.yaml`.
- Keep config defaults conservative. Autonomous behavior should default off unless there is an explicit existing default to preserve.
- When adding a YAML field, update the Go struct, defaults/load normalization, example YAML, tests, debug snapshot summary if relevant, and README command/config docs if user-facing.
- Prefer explicit config names over abbreviations. Keep aliases in runtime commands only when they improve ergonomics without changing the YAML schema.
- Do not reintroduce Main Room config or commands. `main_agent` may still identify an optional conductor session for CLI/skill use, but the TUI no longer has a Main Room view.

## Current Schema Map

- `hosts`: tmux scan targets. Each host can be local or SSH-backed.
- `kanban.columns`: display columns; empty falls back to defaults.
- `main_agent`: optional conductor session metadata used by skills/CLI and board filtering.
- `agent_mesh`: helper-agent scaffold, role policies, skill root, memory root, and mail settings.
- `hermes`: Hermes command, timeout, work log, review automation, done/idle advice, done/idle auto-send, and scoped overrides.
- `notification.qq_enabled`: enables QQ notification paths.
- `debug.snapshot_dir`: optional diagnostic snapshot directory.

## Hermes Scopes

Hermes config has global booleans plus ordered `scopes`.

- Later matching scopes override earlier matching scopes.
- `host: all` is a scoped wildcard for hosts.
- `session: all` or an empty session matches all sessions for the matching host.
- `session: host/session` can include a host selector inside the session field.
- Runtime commands support global, host, session, and current-selection settings; reflect any new boolean in both YAML and scoped runtime command handling.

When adding a scoped Hermes field, update:

- `config.HermesConfig`
- `config.HermesScopeConfig`
- `HermesConfig.Resolve`
- `HermesScopeConfig.Apply`
- Runtime `setHermesScopedBool` command handling
- Config tests and command tests
- Snapshot `HermesSummary` and docs

## Editing Workflow

1. Inspect existing references with `rg` before editing field names or command names.
2. Keep YAML field names stable unless the user explicitly asks for a migration.
3. Preserve backward-compatible load behavior when practical by filling defaults in `Load`.
4. Keep `config.example.yaml` realistic but non-private.
5. Update tests near the behavior: `internal/config` for load/defaults, `cmd/tmux-kanban` for runtime commands, `internal/debug` for snapshots.

## Validation

After config-related Go changes, run:

```bash
gofmt -w <changed-go-files>
go test ./...
go build -o ./bin/tmux-kanban ./cmd/tmux-kanban
```

For YAML-only skill or documentation edits, at least inspect the edited YAML and run a focused `rg` for stale command names or field names.
