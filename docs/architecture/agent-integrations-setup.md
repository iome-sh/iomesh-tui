# Agent integrations setup (MCP · residual-honest)

**Pin:** free eng **s1242** (TUI v178 wire parity) + **s1243** (signing surface) · prior **s1238** slash · concurrent aion **s1237** (MCP v178 tools) · residual docs **s1239**.

Agent/TUI path for **connector integrations setup** via MCP tools — not full install CRUD, not OAuth complete, not checklist/API-key mint, not webhook secret mint/rotate.

## Slash command

```text
/integrations [list [--layer operational|knowledge|analytical] | plan <connector_id> | signing [layer|id] | status]
```

Aliases: `/integration`, `/connectors`. Signing aliases: `signing` · `headers` · `signing-headers`.

| Subcommand | MCP tool | Output |
|------------|----------|--------|
| `list` | `list_connector_catalog` | Compact table: id · status · mesh_layer · oauth? |
| `plan <id>` | `plan_connector_setup` | `portal_url` · `oauth_mode_hint` · `signing_headers_tool` · `next_steps` · honesty notes |
| `signing [layer\|id]` | `get_webhook_signing_headers` | Header parity table (discovery only) |
| `status` / bare | — | Help + honesty one-liner |

## Runtime helpers

`internal/agent/integrations.go`:

- `IntegrationsCatalog(ctx, meshLayer)` — MCP `CallTool` `list_connector_catalog`
- `IntegrationsPlan(ctx, connectorID)` — MCP `CallTool` `plan_connector_setup`
- `IntegrationsSigning(ctx, meshLayerOrConnector)` — MCP `CallTool` `get_webhook_signing_headers` (s1243)

All scan connected MCP servers for the bare tool name (same fail-open spirit as memory digest MCP fallback). Prefer Manager bindings; fall back to each client's tool list.

## aion v178 / v30 wire (TUI parse parity · s1242)

**`list_connector_catalog`** returns:

```json
{
  "count": N,
  "entries": [{
    "id": "github",
    "label": "GitHub",
    "status": "available",
    "mesh_layer": "operational",
    "ingress_type": "webhook",
    "webhook_path": "/webhooks/github",
    "summary": "…",
    "oauth_install_supported": false,
    "portal_path": "/integrations/github"
  }]
}
```

TUI parser prefers `entries` (aion v178); still accepts legacy `connectors` / `items` / `catalog`. OAuth column reads `oauth_install_supported` bool (legacy `oauth` any still supported).

**`plan_connector_setup`** returns:

```json
{
  "connector_id": "github",
  "org_id": "",
  "connector": { /* same entry shape */ },
  "portal_url": "https://console.iome.sh/integrations/github",
  "oauth_install_supported": false,
  "oauth_mode_hint": "stub",
  "signing_headers_tool": "get_webhook_signing_headers",
  "next_steps": ["…"],
  "honesty": {
    "browser_hitl_required_for_oauth_complete": true,
    "stub_oauth_not_live": true,
    "pass_not_invent_install_green": true,
    "dual_write_off": true,
    "book_demo_off": true,
    "no_invent_ga": true,
    "agent_mcp_cannot_write_installs": true,
    "session_portal_owns_install_crud": true,
    "notes": ["…"]
  }
}
```

Plan formatter surfaces `portal_url`, `oauth_mode_hint`, `signing_headers_tool`, `next_steps`, and `honesty.notes`.

**`get_webhook_signing_headers`** (aion v30) input: optional `mesh_layer`. Output:

```json
{
  "fleet_enabled": false,
  "fleet_env_var": "…",
  "count": N,
  "entries": [{
    "connector_id": "github",
    "mesh_layer": "operational",
    "scheme": "hmac_sha256",
    "primary_header": "X-Hub-Signature-256",
    "signature_prefix": "sha256=",
    "secret_env_var": "GITHUB_WEBHOOK_SECRET"
  }]
}
```

`secret_env_var` is an operator env **name** (discovery docs) — never a secret value. TUI does not mint or rotate secrets.

When the signing hint is a connector id (not a mesh layer), TUI calls without `mesh_layer` and filters client-side by `connector_id`.

## Residual honesty (required)

| Rule | Meaning |
|------|---------|
| Browser HITL for OAuth complete | Agent path stops at plan + portal URL; human finishes OAuth in browser |
| stub ≠ live | Catalog / plan rows are not proof of Connected install |
| dual_write OFF | Local-primary memory honesty unchanged |
| book-demo OFF | No invent book-a-demo install path |
| no invent GA | Catalog status chips stay honest (available / beta / planned) |
| catalog Beta honesty | Knowledge / analytical layers remain Beta where applicable |
| fail-open when MCP unavailable | Offline message → `https://console.iome.sh/integrations` |
| never invent install green | Plan output always carries honesty notes; no fake “Connected” |
| signing = discovery only | Header parity table only; no secret mint/rotate |

**Agent setup = catalog + plan + signing discovery + portal HITL · not full install CRUD.**

## Fail-open offline copy

When MCP is missing or tools are not connected:

```text
integrations: MCP connector tools unavailable (fail-open).
  portal HITL: https://console.iome.sh/integrations
  aion MCP tools list_connector_catalog / plan_connector_setup (v178/s1237) · get_webhook_signing_headers (v30) …
```

No invented catalog rows. No invented plan success. No invented signing secrets.

## What this is not

- Not portal session cookie install CRUD (`/v12/.../integrations` mutate)
- Not OAuth authorize/callback completion
- Not mesh install secret mint / checklist write
- Not API-key mint (Agent/MCP onboarding stays credential → copy connection → test invoke)
- Not product Memory GA / INSTALL_STORE green

## Peer continuum

| Pin | Repo | Role |
|-----|------|------|
| s1237 | aion | MCP tools `list_connector_catalog` / `plan_connector_setup` (v178) |
| s1238 | iomesh-tui | Slash `/integrations` list/plan/status |
| s1239 | aion | Residual docs / living surfaces |
| **s1242** | **iomesh-tui** | **v178 wire parse/format parity (`entries`, `oauth_install_supported`, honesty object)** |
| **s1243** | **iomesh-tui** | **`/integrations signing` + `IntegrationsSigning` → `get_webhook_signing_headers`** |
| s1244 | aion | Plan deeplink residual (separate) |

## Config

Uses whatever MCP servers are already attached via `[mcp]` (e.g. platform portal/scenario MCP once s1237 tools land). No new TUI config keys required for the fail-open path.

See also: [mcp.md](./mcp.md) · [memory-mcp.md](./memory-mcp.md) · [mesh-deeper.md](./mesh-deeper.md).
