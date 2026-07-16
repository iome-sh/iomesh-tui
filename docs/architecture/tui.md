# Full-screen TUI

Interactive front-end built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Modes

| Mode | Entry | When |
|------|--------|------|
| **Full-screen** | `iomesh` (default on TTY) | Alt-screen, scrollback, streaming, approval overlay |
| **Classic REPL** | `iomesh --repl` or non-TTY stdout | Line-oriented; used in tests/pipes |

## Layout

```text
┌─ header: brand · model · status / last cost ─────────────┐
│                                                          │
│  scrollable transcript (viewport)                        │
│  · user messages                                         │
│  · streamed assistant text                               │
│  · tool start/end/denied                                 │
│  · cost lines                                            │
│                                                          │
├──────────────────────────────────────────────────────────┤
│  ❯ input                                                 │
│  workspace · status                                      │
└──────────────────────────────────────────────────────────┘
```

During **approval**, the footer becomes a focused y/n/a bar (mutators: write, shell, apply/remove worktree).

## Keys

| Key | Action |
|-----|--------|
| Enter | Send prompt or slash command |
| PgUp / Ctrl+U | Scroll transcript up |
| PgDn / Ctrl+D | Scroll transcript down |
| Ctrl+C | Quit |
| y / n / a | Approval once / deny / always (when prompted) |

Slash commands match the REPL: `/model`, `/models`, `/subagents`, `/permissions`, `/save`, `/sessions`, `/load`, `/cost`, `/help`, `/quit`.

## Package

- `internal/tui/fullscreen.go` — Bubble Tea model + `RunFullscreen`
- `internal/tui/tui.go` — classic `runREPL`, shared slash/model helpers
- Approvals share `agent.Approver` with the REPL path
