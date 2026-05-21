# tmux-kanban

Local TUI for tracking kanban cards, reviewing agent panes, and coordinating work across local or remote tmux sessions.

## Capabilities

- Scan local and SSH tmux hosts, group sessions/windows/panes into a kanban-style cockpit, and attach directly to the selected target.
- Detect `codex` and `claude-code` panes, infer whether they are idle, working, waiting for review, or done, and keep a focused review queue.
- Capture live pane previews, relay choices or text back into agent panes, and save diagnostic snapshots for debugging stale state or misclassification.
- Expose the same review, capture, choice, send, notification, and snapshot operations as JSON CLI commands so other agents can drive the cockpit.
- Provide a Main Room coordination surface plus an optional agent-mesh scaffold for review advice, dispatching, scoped memory, and a-mail style handoffs.

## Local Config

Keep personal hostnames, SSH targets, notification settings, local Hermes paths, and snapshot directories in `config.yaml`. The repo only tracks `config.example.yaml`; copy it and edit locally:

```bash
cp config.example.yaml config.yaml
```

Hosts can be remote SSH targets or local tmux:

```yaml
hosts:
  - name: local
    local: true
  - name: gpu-a
    ssh: user@gpu-a
```

## Controls

- `r` scans configured hosts for tmux sessions.
- `:` opens the command prompt.
- `j` / `k` or arrow keys move the cursor.
- `enter` / `space` expands or collapses a host, session, or window.
- `s` cycles the selected session through `idle`, `working`, `need review`, and `done`.
- Selecting a session, window, or pane opens a live `tmux capture-pane` preview under it. The active preview refreshes about once per second while session/status scanning stays on the slower polling interval.
- `a` attaches to the selected session, window, or pane target.
- Select a `codex` or `claude-code` session, window, or pane and press `x` to relay selection keys to its first agent pane.
- Select a `codex` or `claude-code` session, window, or pane and press `m` to send its first agent pane a message.
- `g` opens Main Room, a chat-style coordination channel. It is local by default; when `hermes.enabled` is on, room messages ask Hermes for a reply.
- `tab` / `v` switches between the tree view and focused review queue.
- In review view, `h` asks Hermes for advice on the current `need review` item, `1-9` chooses, `s` skips, and `u` restores skipped items. Choosing or skipping advances to the next queued item while refreshes keep the current item stable.
- `d` saves a diagnostic snapshot for debugging disappearing sessions or state transitions.
- `q` quits.

## TUI Commands

Press `:` to run runtime commands inside the TUI:

```text
:help
:refresh
:view tree
:view review
:view main
:main start
:main hide
:mesh status
:mesh on
:mesh default claude
:mesh shared off
:mesh skill-root ./mesh-skills
:mesh policy review-advice backend claude
:mesh policy review-advice agent claude
:mesh policy review-advice skill review-advice
:mesh policy review-advice off
:mesh mail dir ~/.local/state/tmux-kanban/mail
:set qq on
:set qq off
:set hermes on
:set hermes.auto_review on
:set mesh.mail on
:set mesh.memory_root ~/.local/state/tmux-kanban/memory
:status idle
:status working
:status need-review
:notify optional message for Hermes
:snapshot
```

These commands affect the running TUI process only. They do not rewrite `config.yaml`. `:notify` uses the same QQ notification path as the CLI and still requires `notification.qq_enabled: true`.

## Main Room

Main Room is a chat-style coordination surface for user messages, review events, Hermes replies, and session activity. It currently has no built-in main-session agent backend: pressing `g` or running `:main start` only opens the room. When `hermes.enabled` is on, Main Room messages ask Hermes directly and show its reply in the room. When Hermes is off, messages stay local until a harness backend is wired in.

The future harness should speak a small structured protocol and can use the JSON CLI abilities below to inspect the review queue, capture panes, send messages, make choices, and save snapshots. It can also call `capabilities` to discover this contract in machine-readable JSON.

```yaml
main_agent:
  enabled: false
  host: local
  session: tmux-kanban-main
  agent: codex
  command: codex
  args: []
```

Runtime commands do not rewrite `config.yaml`.

## Agent Mesh

`agent_mesh` is the scaffold for per-session and cross-session helper agents. It is disabled by default while the runtime is still being wired up, but the model is in place:

- `review-permission` can own approval policy for a session.
- `review-advice` can summarize what it thinks humans should do.
- `dispatcher` can turn advice into a message or task for another agent.
- `session-link` can coordinate windows in one session or sessions on one host.
- `memory_root` is reserved for a hierarchical memory tree: global -> host -> session -> window -> pane.
- `mail` is the a-mail channel for one helper agent to leave scoped messages for another.

```yaml
agent_mesh:
  enabled: false
  shared_short_agent: true
  default_agent: codex
  skill_root: mesh-skills
  memory_root: ""
  policies:
    - name: review-advice
      role: review-advice
      scope: session
      backend: codex
      skill: review-advice
      agent: codex
      enabled: true
  mail:
    enabled: true
    dir: ""
```

`backend` can be `builtin`, `codex`, `claude-code`, `hermes`, or `command`. Codex/Claude backends should be constrained by the role skills under `mesh-skills/<skill>/SKILL.md`; tmux-kanban treats those as role instructions, not as permission to execute actions directly.

Hermes review advice reads the `review-advice` skill file and scoped local memory when `agent_mesh.memory_root` is set. Memory is markdown, read from root to leaf, and only affects advice:

```text
memory_root/
  global.md
  hosts/<host>/memory.md
  hosts/<host>/sessions/<session>/memory.md
  hosts/<host>/sessions/<session>/panes/<pane>/memory.md
  hosts/<host>/sessions/<session>/windows/<window>/memory.md
  hosts/<host>/sessions/<session>/windows/<window>/panes/<pane>/memory.md
```

Set `hermes.enabled: true` plus `hermes.auto_review: true`, or run `:set hermes on` and `:set hermes.auto_review on`, to ask Hermes automatically when a session first enters `need review`. Auto review only accepts explicit Hermes replies that start with `CHOOSE <number>` or `SKIP`; `ASK`, unclear replies, stale sessions, or invisible choices remain for human review.

## Agent CLI

The same binary exposes JSON commands for external agents:

```bash
./bin/tmux-kanban capabilities --config ./config.yaml
./bin/tmux-kanban review-list --config ./config.yaml
./bin/tmux-kanban review-list --config ./config.yaml --all --lines
./bin/tmux-kanban review-list --config ./config.yaml --notify --intent "tell me when an agent needs review"
./bin/tmux-kanban notify-review --config ./config.yaml --intent "daily agent review check"
./bin/tmux-kanban capture --config ./config.yaml --host local --target android:0.0
./bin/tmux-kanban choose --config ./config.yaml --host local --target android:0.0 --choice 1
./bin/tmux-kanban send --config ./config.yaml --host local --target android:0.0 --text "continue"
./bin/tmux-kanban send-keys --config ./config.yaml --host local --target android:0.0 --keys C-c,C-m
./bin/tmux-kanban snapshot --config ./config.yaml
```

`review-list` returns current `need review` panes by default. Add `--all` to list every detected `codex` / `claude-code` pane with its inferred state. Each item includes `host`, `target`, `agent`, and detected choices, so another agent can call `capture`, `choose`, or `send` without opening the TUI.

QQ notification is opt-in and side-effect free by default. Set `notification.qq_enabled: true`, then run `review-list --notify` or `notify-review` to ask the configured Hermes oneshot command to call `send_message(target="qqbot", message=...)`. Notifications are only attempted when current `needs_review` items are non-empty; otherwise the command only prints JSON. The Hermes prompt includes host, target, agent, pane capture, detected choices, and the `--intent` value because oneshot calls do not inherit QQ chat context.

```yaml
hermes:
  enabled: false
  auto_review: false
  command: hermes
  args:
    - --oneshot
  timeout_seconds: 120

notification:
  qq_enabled: false

debug:
  snapshot_dir: ""
```

Snapshots default to `~/.local/state/tmux-kanban/snapshots`. Each snapshot is JSON and includes config summary, host/session state, review queue, status maps, current preview, and recent scan errors.

## Architecture

- `internal/core` owns pure session status and review queue logic.
- `internal/agent` owns agent-facing concepts such as screen analysis, choices, targets, and external reviewer interfaces.
- `internal/mesh` owns the agent mesh scaffold: roles, scopes, memory tree, a-mail, and per-scope agent specs.
- `internal/tmux` exposes a tmux client boundary for scanning, capture, send, and attach operations.
- `internal/ui` keeps shared TUI input/keymap primitives.
- `internal/debug` writes diagnostic snapshots shared by TUI and CLI.

## Quick Start

```bash
go run ./cmd/tmux-kanban
```

Use a config file:

```bash
go run ./cmd/tmux-kanban --config ./config.example.yaml
```

Build a binary:

```bash
go build -o ./bin/tmux-kanban ./cmd/tmux-kanban
```
