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
| `/dashboard` | Toggle landing-page heartbeat live-feed overlay |
| Esc / q | Close dashboard overlay (or dismiss help) |
| Tab / 1–4 | Dashboard: cycle or jump tenancy |
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

## Dashboard (heartbeat live feed)

Landing-page MeshConsole, in the TUI. Same tenancy / pulse / heartbeat / agent-tools analysis as [iome.sh](https://iome.sh) (`/dashboard`, aliases `/heartbeat` `/mesh-console`).

```text
● context://mesh · sre.incidents · policy-gated MCP          EVAL
──────────────── pulse ────────────────────────────────────────
▁▁▂█▃▁▁▁▂█▃▁▁▁
analysis  ops n · knowledge n · analytics n  ·  Beta
Tenancy          Heartbeat                     Agent tools
sre.incidents    14:02:11 ops sre.incidents    mesh.ops.pull
eng.ops          P2 opened — checkout p95      ALLOW
…                …                             DENY
Pulse 18
events / min
```

- REPL prints a snapshot. Fullscreen toggles a ticking overlay (2.6s, same cadence as the site).
- Feed is the public **evaluation template** (HOME_PROOF / MeshConsole seed). Not your workspace. `catalog ≠ Connected`.
- Badge **EVAL** with no mesh client; **CLIENT** only means a mesh client is configured — still a template feed until you pull a real stream.
- Knowledge / analytics stay **Beta**. `dual_write OFF`. Not Memory GA. Not live APPLY.

```text
iomesh> /dashboard
iomesh> /dashboard focus eng.ops
iomesh> /heartbeat help
```

## Package

- `internal/tui/fullscreen.go` — Bubble Tea model + textarea input + dashboard overlay
- `internal/tui/dashboard.go` — MeshConsole heartbeat live-feed analysis
- `internal/tui/theme.go` — named palettes
- `internal/tui/tui.go` — classic REPL + shared slash helpers (`/memory`, `/integrations`, `/dashboard`, …)

**s1238/s1242/s1243:** `/integrations` (`list` / `plan` / `signing` / `status`) — connector catalog + setup plan + signing header discovery via MCP (v178 wire parity); residual-honest fail-open. See [agent-integrations-setup.md](./agent-integrations-setup.md).
