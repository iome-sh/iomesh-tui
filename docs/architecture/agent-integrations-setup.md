# Agent integrations setup (MCP · residual-honest)

**Pin:** free eng **s1238** (TUI) · concurrent aion **s1237** (MCP tools) · residual docs **s1239**.

Agent/TUI path for **connector integrations setup** via MCP tools — not full install CRUD, not OAuth complete, not checklist/API-key mint.

## Slash command

```text
/integrations [list [--layer operational|knowledge|analytical] | plan <connector_id> | status]
```

Aliases: `/integration`, `/connectors`.

| Subcommand | MCP tool | Output |
|------------|----------|--------|
| `list` | `list_connector_catalog` | Compact table: id · status · mesh_layer · oauth? |
| `plan <id>` | `plan_connector_setup` | `portal_url` · `next_steps` · honesty notes |
| `status` / bare | — | Help + honesty one-liner |

## Runtime helpers

`internal/agent/integrations.go`:

- `IntegrationsCatalog(ctx, meshLayer)` — MCP `CallTool` `list_connector_catalog`
- `IntegrationsPlan(ctx, connectorID)` — MCP `CallTool` `plan_connector_setup`

Both scan connected MCP servers for the bare tool name (same fail-open spirit as memory digest MCP fallback). Prefer Manager bindings; fall back to each client's tool list.

## Residual honesty (required)

| Rule | Meaning |
|------|---------|
| Browser HITL for OAuth complete | Agent path stops at plan + portal URL; human finishes OAuth in browser |
| stub ≠ live | Catalog / plan rows are not proof of Connected install |
| dual_write OFF | Local-primary memory honesty unchanged |
| no invent GA | Catalog status chips stay honest (available / beta / planned) |
| catalog Beta honesty | Knowledge / analytical layers remain Beta where applicable |
| fail-open when MCP unavailable | Offline message → `https://console.iome.sh/integrations` · aion tools ship in s1237 |
| never invent install green | Plan output always carries honesty notes; no fake “Connected” |

**Agent setup = catalog + plan + portal HITL · not full install CRUD.**

## Fail-open offline copy

When MCP is missing or tools are not connected:

```text
integrations: MCP connector tools unavailable (fail-open).
  portal HITL: https://console.iome.sh/integrations
  aion MCP tools list_connector_catalog / plan_connector_setup ship in s1237 …
```

No invented catalog rows. No invented plan success.

## What this is not

- Not portal session cookie install CRUD (`/v12/.../integrations` mutate)
- Not OAuth authorize/callback completion
- Not mesh install secret mint / checklist write
- Not API-key mint (Agent/MCP onboarding stays credential → copy connection → test invoke)
- Not product Memory GA / INSTALL_STORE green

## Peer continuum

| Pin | Repo | Role |
|-----|------|------|
| s1237 | aion | MCP tools `list_connector_catalog` / `plan_connector_setup` |
| **s1238** | **iomesh-tui** | **This slash + Runtime helpers** |
| s1239 | aion | Residual docs / living surfaces |

## Config

Uses whatever MCP servers are already attached via `[mcp]` (e.g. platform portal/scenario MCP once s1237 tools land). No new TUI config keys required for the fail-open path.

See also: [mcp.md](./mcp.md) · [memory-mcp.md](./memory-mcp.md) · [mesh-deeper.md](./mesh-deeper.md).
