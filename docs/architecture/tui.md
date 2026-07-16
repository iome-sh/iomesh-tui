# Full-screen TUI

Interactive front-end built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Modes

| Mode | Entry | When |
|------|--------|------|
| **Full-screen** | `iomesh` (default on TTY) | Alt-screen, scrollback, multi-line input, themes |
| **Classic REPL** | `iomesh --repl` or non-TTY stdout | Line-oriented; used in tests/pipes |

## Layout

```text
┌─ header: brand · model · theme · status / last cost ─────┐
│                                                          │
│  scrollable transcript (viewport)                        │
│                                                          │
├──────────────────────────────────────────────────────────┤
│  ❯ multi-line textarea (grows 3–8 rows)                  │
│  workspace · status · enter send · ctrl+j ⏎              │
└──────────────────────────────────────────────────────────┘
```

During **approval**, the footer becomes a focused y/n/a bar.

## Keys

| Key | Action |
|-----|--------|
| **Enter** | Send prompt or slash command |
| **Ctrl+J** / Alt+Enter / Shift+Enter | Insert newline (multi-line edit) |
| PgUp / Ctrl+U | Scroll transcript up |
| PgDn / Ctrl+D | Scroll transcript down |
| Ctrl+C | Quit |
| y / n / a | Approval once / deny / always |

## Themes

Config `[ui] theme = "…"` or slash `/theme <name>` at runtime:

| Name | Look |
|------|------|
| `default` | Cyan/amber (original) |
| `mono` | Grayscale |
| `high-contrast` | Bright, strong error/approve |
| `dim` | Soft / muted |

```bash
# config.toml
[ui]
theme = "mono"
```

```text
iomesh> /theme
iomesh> /theme high-contrast
```

## Package

- `internal/tui/fullscreen.go` — Bubble Tea model + textarea input
- `internal/tui/theme.go` — named palettes
- `internal/tui/tui.go` — classic REPL + shared slash helpers
