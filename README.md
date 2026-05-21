# tmux-kanban

Terminal kanban for supervising long-running coding agents across local and remote tmux sessions.

[中文文档](README.zh-CN.md)

![tmux-kanban terminal cockpit showing session board, tmux explorer, live preview, and agent activity](docs/images/tmux-kanban-showcase.png)

tmux-kanban is a local TUI for tracking coding-agent sessions, reviewing permission prompts, and coordinating work across multiple SSH machines. It is built mainly for Codex and Claude Code. Because I currently use Codex more heavily, the Codex path is likely better exercised, but Claude Code is a first-class target in the design.

The project solves a very specific pain point for me: I do persistent work on several SSH servers. On each machine, tmux is already a good primitive for keeping work alive. The bad part is constantly switching between machines and tmux sessions just to notice that an agent is waiting, press Enter, pick an option, or send a short message. I often forget one session, which slows the whole system down.

From an efficiency point of view, that felt like human-bound vibe coding: the agents were waiting for me. My goal is agent-bound vibe coding instead: most of the time, I should be waiting for agents.

Codex and Claude Code both have remote-control features, but in my own setup those connections have not been stable enough, especially Codex remote control during the days that pushed me to build this. tmux-kanban is my terminal-native answer to that: it does not rely on the Codex or Claude Code SDKs. It uses tmux itself as the control plane.

### How It Works

The core implementation is intentionally pragmatic:

- It scans local and SSH tmux hosts, then groups sessions, windows, and panes into a kanban-style cockpit.
- It detects Codex and Claude Code panes mostly through terminal text pattern matching.
- It infers whether a session is idle, working, waiting for review, or done.
- It polls panes with `tmux capture-pane` for live previews. The latency is visible, but acceptable for my workflow.
- It can attach directly to a target with `a`, send a quick message with `m`, and choose visible options with `1-9`.
- It can relay keys and messages back into an agent pane without opening the tmux session manually.

I have not tested the full matrix of tmux window-splitting workflows yet, because I rarely use split windows myself. The common session/window/pane paths are the ones I use and care about today.

### Capabilities

- Local and remote tmux host scanning.
- Session board with `idle`, `working`, `need review`, and `done` states.
- Focused review queue for Codex and Claude Code prompts.
- Live terminal preview for selected sessions, windows, and panes.
- Direct attach, quick message sending, key relay, and numbered choice selection.
- JSON CLI commands for review listing, capture, choose, send, notify, and snapshot.
- Optional Hermes integration for advice, mobile workflows, and social-media notification hooks.
- Diagnostic snapshots designed for agent-assisted debugging.
- Experimental agent-mesh scaffolding for memory, review advice, and future task dispatch.

### Hermes And The CLI

tmux-kanban also includes a JSON CLI because I want to work remotely from a phone through Hermes and a small skill layer. In that direction, tmux-kanban gives Hermes core abilities such as listing review items, capturing panes, sending messages, choosing options, and saving snapshots.

Hermes also gives abilities back to tmux-kanban:

- In review mode, `h` asks Hermes for advice on the current item.
- With the right settings, tmux-kanban can ask Hermes automatically when a session enters review.
- Hermes replies can be accepted automatically when they are explicit enough, such as `CHOOSE <number>` or `SKIP`.
- If a problem needs me personally, Hermes can notify me through a social channel such as QQ.

Strictly speaking, the current "review" flow is closer to permission approval than code review. It would be easy to delegate approval to Codex or Claude Code too, but I have not wired that up yet, partly because those agents cannot easily ask me for help through my social channels.

### Memory And Future Dispatch

The current system is still under my direct control. It can approve choices and coordinate sessions, but I am not yet letting agents dispatch arbitrary work to other agents. That change would not be very large technically, but I do not want these projects to run outside my control, and I also do not want to burn enough tokens to keep a fleet of agents working day and night.

That said, I am preparing for more autonomous dispatch. The main design idea is scoped memory: guidance can exist at multiple granularities, from global project notes down to host, session, window, and pane context.

```text
memory_root/
  global.md
  hosts/<host>/memory.md
  hosts/<host>/sessions/<session>/memory.md
  hosts/<host>/sessions/<session>/panes/<pane>/memory.md
  hosts/<host>/sessions/<session>/windows/<window>/memory.md
  hosts/<host>/sessions/<session>/windows/<window>/panes/<pane>/memory.md
```

For now, this memory mainly informs review advice. Later it can guide task dispatch, summarization, and cross-session coordination.

In spirit, my longer-term vision overlaps a little with [openai/symphony](https://github.com/openai/symphony): moving from supervising coding agents directly toward managing work at a higher level. The difference is that tmux-kanban is much more personal and tmux-centered. I started this project before noticing Symphony had been open sourced, which was a funny coincidence.

### Snapshots For Agent Debugging

Snapshots are meant to make behavior debuggable without requiring me to inspect every line of code myself. A snapshot records the config summary, host/session topology, review queue, status maps, current preview, and recent scan errors. That gives a coding agent enough evidence to investigate why a session was marked idle, working, waiting for review, or done.

This matters because the project is intentionally agent-assisted. I chose Go partly because I know a little Go, but in practice I often debug by asking agents to inspect snapshots and tests rather than reading the whole codebase manually.

### Name

The name is admittedly plain. A friend already complained about it. Naming is hard; the current name at least says what it does.

### Roadmap

The near-term plan is mostly maintenance: fix bugs found in real use, make the status detection less brittle, and continue cleaning up the code structure after the first working version. For my own workflow, there are not many urgent new features left; the tool already covers the main pain point I built it for.

If I have time, I may improve the Codex and Claude Code integrations, support more tmux layouts, and make the mesh/memory pieces more useful. But this is exactly the sort of sentence that often turns into a quiet TODO forever, so treat it as direction rather than a promise.

### Quick Start

```bash
go run ./cmd/tmux-kanban
```

Use a config file:

```bash
cp config.example.yaml config.yaml
go run ./cmd/tmux-kanban --config ./config.yaml
```

Build a binary:

```bash
go build -o ./bin/tmux-kanban ./cmd/tmux-kanban
```

### Local Config

Keep personal hostnames, SSH targets, notification settings, local Hermes paths, and snapshot directories in `config.yaml`. The repo only tracks `config.example.yaml`.

```yaml
hosts:
  - name: local
    local: true
  - name: gpu-a
    ssh: user@gpu-a
```

### Controls

- `r` scans configured hosts for tmux sessions.
- `:` opens the command prompt.
- `j` / `k` or arrow keys move the cursor.
- `enter` / `space` expands or collapses a host, session, or window.
- `s` cycles the selected session through `idle`, `working`, `need review`, and `done`.
- `a` attaches to the selected session, window, or pane target.
- `m` sends a message to the first detected agent pane for the selected target.
- `x` relays selection keys to the first detected agent pane for the selected target.
- `g` opens Main Room, a local coordination channel.
- `tab` / `v` switches between tree view and the focused review queue.
- In review view, `h` asks Hermes for advice, `1-9` chooses, `s` skips, and `u` restores skipped items.
- `d` saves a diagnostic snapshot.
- `q` quits.

### TUI Commands

Press `:` to open the command prompt. Commands support completion suggestions; use `up` / `down` or `ctrl+p` / `ctrl+n` to move through suggestions, `tab` to accept one, `enter` to run it, and `esc` or `ctrl+c` to cancel.

These commands are runtime controls. They affect the current TUI process only and do not rewrite `config.yaml`.

General navigation and state:

```text
:help
:refresh
:view tree
:view review
:view main
:status idle
:status working
:status need-review
:status done
:snapshot
```

`:refresh` rescans configured tmux hosts. `:view` switches between the tree, review queue, and Main Room. `:status` manually overrides the selected session's state. `:snapshot` saves a diagnostic JSON snapshot; if no description is provided, the TUI prompts for one.

Main Room commands:

```text
:main start
:main hide
:main status
:main codex
:main claude
:main host local
:main session tmux-kanban-main
:main command codex
```

Main Room is a coordination surface and preview target for a configured conductor session. Some commands already change the runtime config and preview target, but the broader "main agent harness" is still experimental.

Hermes, QQ, and runtime settings:

```text
:settings
:set qq on
:set qq off
:set hermes on
:set hermes.auto_review on
:set main.agent claude
:set main.host local
:set main.session tmux-kanban-main
:notify optional message for Hermes
```

`:notify` uses the configured Hermes/QQ notification path and still requires `notification.qq_enabled: true`. Hermes auto review is intentionally conservative: automatic choices require explicit Hermes replies such as `CHOOSE <number>` or `SKIP`.

Agent mesh commands:

```text
:mesh status
:mesh on
:mesh off
:mesh default claude
:mesh shared off
:mesh skill-root ./mesh-skills
:mesh memory ~/.local/state/tmux-kanban/memory
:mesh policy review-advice backend claude
:mesh policy review-advice agent claude
:mesh policy review-advice skill review-advice
:mesh policy review-advice off
:mesh mail dir ~/.local/state/tmux-kanban/mail
:set mesh.mail on
:set mesh.memory_root ~/.local/state/tmux-kanban/memory
```

The mesh commands currently expose the role, backend, skill, mail, and memory configuration model at runtime. The memory and review-advice pieces are useful today, while full autonomous task dispatch is still a scaffold rather than a finished workflow.

### Agent CLI

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

`review-list` returns current `need review` panes by default. Add `--all` to list every detected Codex or Claude Code pane with its inferred state.

### Architecture

- `cmd/tmux-kanban`: TUI and JSON CLI entrypoints.
- `internal/core`: pure status and review queue logic.
- `internal/agent`: agent-facing screen analysis, choices, targets, and reviewer concepts.
- `internal/mesh`: role, scope, memory tree, and a-mail scaffolding.
- `internal/tmux`: tmux client boundary.
- `internal/tmuxscan`: tmux command parsing and screen detection.
- `internal/debug`: diagnostic snapshot writer.
- `internal/ui`: shared TUI key/input primitives.
