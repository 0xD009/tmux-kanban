---
name: tmux-kanban-mesh
description: "Develop tmux-kanban's main session, agent mesh, subagent roles, scoped memory tree, a-mail, review/dispatcher helpers, and cross-session coordination model."
---

# tmux-kanban Mesh

## Concepts

The project is evolving from a tmux monitor into a kanban-aware agent cockpit:

- Main session: a fixed tmux session for a conductor agent, configured by `main_agent`.
- Agent mesh: optional helper-agent scaffold configured by `agent_mesh`.
- Roles: `review-permission`, `review-advice`, `dispatcher`, `session-link`, `memory-summarizer`.
- Scopes: `global -> host -> session -> window -> pane`.
- Memory tree: scoped summaries that agents can read from root to leaf.
- A-mail: scoped messages between helper agents.

## Key Files

- `internal/mesh/mesh.go`: pure role/scope/memory/mail/spec model.
- `internal/config/config.go`: `main_agent` and `agent_mesh` YAML config.
- `internal/debug/snapshot.go`: config/runtime summary in snapshots.
- `cmd/tmux-kanban/tui.go`: runtime commands such as `:main ...` and `:mesh ...`.
- `config.example.yaml` and `README.md`: user-facing examples.

## Design Guardrails

- Keep `agent_mesh.enabled` default-off until real runtime dispatch is deliberate.
- Runtime TUI commands should mutate in-memory config only; do not write `config.yaml` unless the user asks for persistence.
- Prefer pure data structures and tests before starting real Codex/Claude/Hermes subprocesses.
- Do not hard-code a single agent. Config should allow `codex`, `claude-code`, and custom commands.
- Subagent behavior should be scoped: a session helper should not accidentally act on unrelated hosts/sessions.

## Useful Commands

```text
:main start
:main codex
:main claude
:main host local
:main session tmux-kanban-main
:main command codex

:mesh status
:mesh on
:mesh default claude
:mesh shared off
:mesh policy review-advice agent claude
:mesh policy review-advice off
:mesh mail dir ~/.local/state/tmux-kanban/mail
```

## Validation

Mesh changes should usually include pure tests in `internal/mesh` and command tests in `cmd/tmux-kanban/main_test.go`.

```bash
go test ./...
go build -o ./bin/tmux-kanban ./cmd/tmux-kanban
```

