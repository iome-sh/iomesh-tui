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
- **Brief ACK (#371):** compose strip shows `brief: UNREAD` until `/dashboard ack`. Unacked ≠ known (fail-open). Optional local palace write (`brief-ack.json` beside user config). ACK does not send/pay/ship.
- **Market-telling / voc_brief (#372):** `/gtm brief` writes a named palace artifact (`source=agent-brief`, tenant `gtm/founder`) — local palace SoR, not a git file. Ledger: shipped / moved / killed vs falsified + contradiction vs yesterday. Cadence `daily|weekly|on_threshold` (daily refused below volume floor). One RevOps support-theme recipe, same metadata as incidents, ≤3 first-party sources. dual_write OFF · not Memory GA · no Slack persist · CRM ≠ Connected. Hands (win-back, price change) stay off this plane.

```text
iomesh> /dashboard
iomesh> /dashboard preview
iomesh> /dashboard focus eng.ops
iomesh> /dashboard ack
iomesh> /heartbeat help
iomesh> /gtm brief write --kind voc_brief --hypothesis support-theme --confidence 0.7 --falsify theme-vanishes-next-window
iomesh> /gtm brief ledger killed H2 --vs-yesterday shipped
iomesh> /gtm brief cadence daily --volume 3
```

## Package

- `internal/tui/fullscreen.go` — Bubble Tea model + textarea input + dashboard overlay
- `internal/tui/dashboard.go` — MeshConsole heartbeat live-feed analysis
- `internal/tui/market_telling.go` — palace `market_telling` / `voc_brief` (#372) · ledger · cadence · one RevOps support-theme recipe
- `internal/tui/theme.go` — named palettes
- `internal/tui/tui.go` — classic REPL + shared slash helpers (`/memory`, `/integrations`, `/dashboard`, `/gtm brief`, …)

**s1238/s1242/s1243:** `/integrations` (`list` / `plan` / `signing` / `status`) — connector catalog + setup plan + signing header discovery via MCP (v178 wire parity); residual-honest fail-open. See [agent-integrations-setup.md](./agent-integrations-setup.md).
