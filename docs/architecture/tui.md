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
| `/dashboard` | Toggle overlay (empty until consume; probe if mesh attached) |
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

Landing-page MeshConsole chrome, in the TUI (`/dashboard`, aliases `/heartbeat` `/mesh-console`). Default `/dashboard` is **empty until consume**. `/dashboard preview` is the public **evaluation template** (HOME_PROOF / MeshConsole seed) — not your org. README showcase: [dashboard-eval.svg](../assets/dashboard-eval.svg).

```text
● context://mesh · no live heartbeat · consume missing · sre.incidents     EMPTY
──────────────── pulse ────────────────────────────────────────
▁▁▂█▃▁▁▁▂█▃▁▁▁
analysis  ops 0 · knowledge 0 · analytics 0  ·  Beta
knowledge Beta empty · analytics Beta empty · not GA
Heartbeat
no consumed messages · mock eval rows hidden
next: add [iomesh] endpoint="https://hooks.iome.sh" · then consume GitHub
portal MCP (apiv1.iome.sh/v7/mcp) is catalog — streams are hooks.iome.sh
or infer from portal MCP · infer ≠ Connected · do not invent consume
```

- REPL `/dashboard` (no args): empty snapshot; **probe** if a mesh client is attached (`ListStreams` then `ListStreamMessages` on the first 4 names — same path as `iomesh mesh streams --messages` / broker `GET /v1/streams/{name}/messages`). **Not** portal `GET /v52` (cookie-only).
- When **no mesh client** (`consume missing`): tell the operator to add `[iomesh]` or infer hooks from portal MCP. Infer ≠ Connected. Do **not** invent consume.
- Fullscreen toggles an overlay. Tick is a no-op unless `/dashboard preview`. On open (non-preview) the same consume probe runs once if `Mesh()` is available. `/setup reload` hot-swaps mesh when `[iomesh]` or inferred hooks change.
- Fail-open reasons: `no_streams` · `empty_stream` · `replay_disabled` · `broker_unavailable`. Errors → empty + reason, never the eval seed.
- **PULSE**-shaped rows only when ≥1 broker message was decoded. Never invent PULSE from eval or from a stream list alone. Create stream ≠ PULSE. Mesh pub is ephemeral and does not fill `/dashboard`.
- Badge **EMPTY** (no mesh, no consume) / **CLIENT** (mesh attached, no consumed rows) / **PULSE** (≥1 decoded message) / **EVAL** (`/dashboard preview` only).
- Knowledge / analytics stay **Beta**. Empty knowledge or analytics adds `knowledge Beta empty · analytics Beta empty · not GA`. `catalog ≠ Connected`. `dual_write OFF`. Not Memory GA. Not live APPLY. Kind from subject ≠ GA.

```text
iomesh> /dashboard
iomesh> /dashboard preview
iomesh> /dashboard focus eng.ops
iomesh> /heartbeat help
```

## Package

- `internal/tui/fullscreen.go` — Bubble Tea model + textarea input + dashboard overlay
- `internal/tui/dashboard.go` — MeshConsole heartbeat live-feed analysis
- `internal/tui/theme.go` — named palettes
- `internal/tui/tui.go` — classic REPL + shared slash helpers (`/memory`, `/integrations`, `/dashboard`, …)

**s1238/s1242/s1243:** `/integrations` (`list` / `plan` / `signing` / `status`) — connector catalog + setup plan + signing header discovery via MCP (v178 wire parity); residual-honest fail-open. See [agent-integrations-setup.md](./agent-integrations-setup.md).
