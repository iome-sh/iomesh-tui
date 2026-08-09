---
name: setup-lifecycle-agent
description: Residual-honest agent-native setup lifecycle (init/preflight · dual_write OFF · not Memory GA · catalog ≠ Connected · portal HITL · not invent install green)
---

# Setup lifecycle agent (residual-honest · s1526 P3)

Agent-native path to **bootstrap** local TUI config planes via managed fragment write + preflight probes — **not** invent Connected / Memory GA / INSTALL_STORE green.

Prefer slash `/setup` (alias `/setup-lifecycle`) or CLI `iomesh setup` when the operator is at a terminal. Use this skill when planning setup steps in chat.

## Workflow

1. **Init managed config** — write residual-honest fragment into user `config.toml`.
   - CLI: `iomesh setup init [profiles]`
   - Slash: `/setup init [profiles] [--stdio] [--print-only] [--plugins-dir path]`
   - Profiles: `local-memory` (default) · `plugins` · `mesh` · `platform-mcp` · `all`
   - Managed block markers: `# BEGIN iomesh-setup-managed` … `# END iomesh-setup-managed`
   - **dual_write = false always** — setup path refuses `dual_write = true`
   - Secrets as **env names only** (`api_key_env`, `oauth_token_env`) — never commit secret values
   - After write: start memory host if local-memory · set env vars · **restart TUI** (hot MCP reattach is a later PR)

2. **Preflight probe** — residual-honest state, never invent install green.
   - CLI: `iomesh setup preflight [--json]`
   - Slash: `/setup preflight` (aliases `status` · `check`)
   - States: `not_started` · `config_present` / `config_written` · `awaiting_memory_host` · `local_memory_probe_ok`
   - **PASS ≠ invent Connected / INSTALL_STORE green / Memory GA**

3. **Portal HITL** — OAuth / connector install still browser session.
   - Slash: `/setup portal`
   - URLs: https://console.iome.sh/integrations · https://console.iome.sh/settings/agent
   - Agent MCP **cannot write installs**

4. **Continuous pull** — still CLI until later PRs.
   - `iomesh memory pull` (requires mesh + consumer configured)
   - **Not** in-session continuous pull yet · **not** `/setup pull` product green

5. **Analyze** — ops pulse after data exists.
   - `/memory digest` (and advanced memory skill) — residual · not Memory GA

## Honesty locks (never violate)

| Lock | Rule |
|------|------|
| dual_write OFF | Managed fragment + setup path never force dual_write ON |
| not Memory GA | Preflight / init never stamp Memory GA |
| catalog ≠ Connected | Setup PASS / probe_ok ≠ invent install Connected |
| portal HITL | OAuth / INSTALL_STORE APPLY stay browser |
| secrets env names only | `api_key_env` / `oauth_token_env` — no secret values in config |
| continuous pull CLI | `iomesh memory pull` until in-session PR |
| never invent green | No Connected / INSTALL_STORE green / Memory GA from setup alone |

## Non-goals (never do)

- Do **not** invent Connected / INSTALL_STORE APPLY green from setup PASS.
- Do **not** claim Memory GA or dual_write ON.
- Do **not** mint OAuth tokens or write connector installs from agent MCP.
- Do **not** claim in-session continuous pull is shipped (CLI only for now).
- Do **not** treat catalog Beta/available as org Connected counts.

## Operator surfaces

| Surface | Action |
|---------|--------|
| Slash `/setup` | help · init · preflight · portal |
| Alias `/setup-lifecycle` | same |
| CLI `iomesh setup` | init · preflight (s1525 P1–P2) |
| System note | `<setup-lifecycle>` on AttachMCP |
| Skill | `read_skill setup-lifecycle-agent` |

## Related

- Docs: `docs/architecture/setup-lifecycle.md`
- Builtin always merged when skills enabled (`go:embed`)
- Integrations residual path: skill `connector-integrations-setup` · `/integrations`
- Memory advanced residual: skill `memory-advanced-agent` · `/memory`
- Onboarding continuum: skill `aion-agent-onboarding` · `/onboard`
