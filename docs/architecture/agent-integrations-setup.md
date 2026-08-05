# Agent integrations setup (MCP · residual-honest)

**Pin:** free eng **s1257** (deep-link parity + skill dogfood) · **s1251** (agent skill + system note) · **s1252** (golden fixtures / s1244 deep links) · **s1247** (status pulse) · **s1242** (TUI v178 wire parity) + **s1243** (signing surface) · prior **s1238** slash · concurrent aion **s1237** (MCP v178 tools) · residual docs **s1239** · free eng concurrent **s1256+** / free-floor peer **s1254**.

Agent/TUI path for **connector integrations setup** via MCP tools — not full install CRUD, not OAuth complete, not checklist/API-key mint, not webhook secret mint/rotate.

## Deep-link parity + skill dogfood (s1257)

Residual-honest **round-trip dogfood** for plan deep links and the builtin skill (no live install green claim):

1. **Plan deep-link parity** — golden fixtures (`v178_plan_github.json`, `v178_plan_notion.json`) carry aion **s1244** fields:
   - `portal_url` / `portal_detail_url`
   - `portal_add_url` = `https://console.iome.sh/integrations/add?template={id}`
   - `deep_links` map (`detail`, `add_wizard`, `catalog`, …)
   - `formatConnectorPlan` surfaces `portal_add_url` + `deep_links` with residual labels: **browser HITL only · not install APPLY**
   - `template=` is deep-link shape only — **≠ install APPLY** success
   - Honesty footer always: never invent install green · no `focus=` fantasy params
2. **Builtin skill dogfood** — `LoadWithBuiltin` / `LoadBuiltin` always yields `connector-integrations-setup`; body mentions `list_connector_catalog`, `plan_connector_setup`, portal HITL, browser HITL, never invent install green; description is residual-honest (not install APPLY).
3. **Guidance ↔ skill consistency** — `IntegrationsAgentGuidanceNote()` and skill body share core honesty needles (`list_connector_catalog`, `plan_connector_setup`, portal, never invent install green, dual_write OFF / browser HITL).
4. **Catalog portal_path only** — status pulse / catalog table remain residual-honest when entries carry `portal_path` only (catalog honesty ≠ install Connected).

**Honesty locks (s1257):** deep_links = browser HITL only · template= ≠ install APPLY · skill ≠ invent Connected · dual_write OFF · book-demo OFF · no invent GA.

## Agent skill + system note (s1251)

When MCP is attached (`AttachMCP`), the runtime also injects an **`<integrations>`** system note from `IntegrationsAgentGuidanceNote()`:

1. Discover — `list_connector_catalog` (catalog status ≠ install Connected)
2. Plan — `plan_connector_setup` (portal deep links + honesty notes)
3. Optional — `get_webhook_signing_headers` (discovery only)
4. Complete install/OAuth in **browser portal HITL** at https://console.iome.sh/integrations — agent MCP cannot write installs
5. Operator pulse — slash `/integrations status|list|plan|signing`

**Builtin skill** `connector-integrations-setup` ships via `go:embed` under `internal/skills/builtin/` and is always merged when skills are enabled (`skills.LoadWithBuiltin`), even if user/workspace skill dirs are empty. Agent discovers it via `list_skills` / `read_skill`.

## Slash command

```text
/integrations [list [--layer operational|knowledge|analytical] | plan <connector_id> | signing [layer|id] | status]
```

Aliases: `/integration`, `/connectors`. Signing aliases: `signing` · `headers` · `signing-headers`. Status alias: `st`. Help: `help` · `?` (bare `/integrations` is help).

| Subcommand | MCP tool | Output |
|------------|----------|--------|
| `list` | `list_connector_catalog` | Compact table: id · status · mesh_layer · oauth? |
| `plan <id>` | `plan_connector_setup` | `portal_url` · `oauth_mode_hint` · `signing_headers_tool` · `next_steps` · honesty notes |
| `signing [layer\|id]` | `get_webhook_signing_headers` | Header parity table (discovery only) |
| `status` / `st` | probe + optional list | Residual-honest **operator pulse** (s1247) — not pure help |
| bare / `help` / `?` | — | Usage + honesty one-liner |

## Runtime helpers

`internal/agent/integrations.go`:

- `IntegrationsCatalog(ctx, meshLayer)` — MCP `CallTool` `list_connector_catalog`
- `IntegrationsPlan(ctx, connectorID)` — MCP `CallTool` `plan_connector_setup`
- `IntegrationsSigning(ctx, meshLayerOrConnector)` — MCP `CallTool` `get_webhook_signing_headers` (s1243)
- `IntegrationsStatus(ctx)` — s1247 residual-honest operator pulse (MCP path · tools · catalog honesty) + s1263 org-installs residual honesty

All scan connected MCP servers for the bare tool name (same fail-open spirit as memory digest MCP fallback). Prefer Manager bindings; fall back to each client's tool list.

## Status pulse (s1247 · s1263)

`/integrations status` reports an **operator pulse**, not help text:

1. **MCP path** — available (N servers) · connected-empty · offline fail-open
2. **Tools present** — for each of `list_connector_catalog`, `plan_connector_setup`, `get_webhook_signing_headers`: `present` / `missing` / `offline` (lightweight discovery only; same binding/Tools scan as `callMCPToolByName`, no invent)
3. **Catalog pulse** (`formatCatalogPulse`) — only when `list_connector_catalog` is present and the call returns parseable JSON (v178 `entries`; legacy keys still accepted):
   - `total catalog entries: N  (catalog honesty — NOT install Connected / NOT INSTALL_STORE green)`
   - `by mesh_layer: operational=A knowledge=B analytical=C`
   - `by catalog status: available=X beta=Y planned=Z`
   - Catalog status chips are display honesty only — **not** Connected/installed; **catalog count ≠ install count**
4. **Org installs residual honesty (s1263)** — always (online and offline); no install MCP tool exists on agent path:
   - `org installs: unavailable via agent MCP (portal session HITL only)`
   - `dual-auth read snapshot: candidacy open · never invent Connected / empty-as-none`
   - `portal: https://console.iome.sh/integrations`
   - Does **not** call any install MCP tool · does **not** invent install rows · does **not** claim dual-auth shipped (peer aion s1261 candidacy only)
5. **Honesty footer always** (`statusHonestyFooter`) — never invent install green · catalog ≠ installs · browser HITL · stub ≠ live · dual_write OFF · book-demo OFF · signing discovery only · portal `https://console.iome.sh/integrations`

Offline / empty MCP → residual message, **no invented counts**. Never invents org install Connected / INSTALL_STORE green / GA / empty-as-none installs.

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

Plan formatter surfaces `portal_url`, `portal_add_url`, `deep_links` (s1244 proven console routes), `oauth_mode_hint`, `signing_headers_tool`, `next_steps`, and `honesty.notes`. Deep links are **browser HITL only** — never invent `focus=` fantasy query params or install APPLY green.

Golden fixtures (s1252): `internal/agent/testdata/v178_catalog_entries.json`, `v178_plan_github.json`, `v178_plan_notion.json`.

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
| never invent install green | Plan/status output always carries honesty notes; no fake “Connected” |
| signing = discovery only | Header parity table only; no secret mint/rotate |
| catalog count ≠ install count | Status pulse catalog inventory is honesty only, not Connected installs |
| status = operator pulse | MCP path + tool presence + catalog honesty; not pure help (s1247) |

**Agent setup = catalog + plan + signing discovery + status pulse + portal HITL · not full install CRUD.**

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
| s1242 | iomesh-tui | v178 wire parse/format parity (`entries`, `oauth_install_supported`, honesty object) |
| s1243 | iomesh-tui | `/integrations signing` + `IntegrationsSigning` → `get_webhook_signing_headers` |
| s1244 | aion | Plan deeplink residual (`portal_add_url` · `deep_links`) |
| **s1247** | **iomesh-tui** | **`/integrations status` residual-honest operator pulse (`IntegrationsStatus` · `formatCatalogPulse`)** |
| **s1251** | **iomesh-tui** | **Agent skill + `<integrations>` system note (`IntegrationsAgentGuidanceNote` · builtin `connector-integrations-setup`)** |
| **s1252** | **iomesh-tui** | **Golden fixtures + plan deep_links display parity** |
| **s1257** | **iomesh-tui** | **Deep-link parity dogfood + skill residual-honest tests (template= · portal_add_url · guidance↔skill needles)** |
| **s1259** | free-floor peer | free-floor continuum peer (not dual-auth ship) |
| **s1261** | aion | dual-auth org install read snapshot **candidacy** only (not claimed shipped in TUI) |
| **s1263** | **iomesh-tui** | **Status org-installs residual honesty (`statusOrgInstallsSection` · always unavailable via agent MCP · portal HITL · never invent Connected / empty-as-none)** · free eng concurrent s1261+ |

## Config

Uses whatever MCP servers are already attached via `[mcp]` (e.g. platform portal/scenario MCP once s1237 tools land). No new TUI config keys required for the fail-open path.

See also: [mcp.md](./mcp.md) · [memory-mcp.md](./memory-mcp.md) · [mesh-deeper.md](./mesh-deeper.md).
