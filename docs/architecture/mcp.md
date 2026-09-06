# MCP client

Minimal [Model Context Protocol](https://modelcontextprotocol.io) client supporting:

| Transport | Config | Notes |
|-----------|--------|--------|
| **stdio** | `command` + `args` | Local process; scrubbed env + explicit `env` |
| **Streamable HTTP** | `url` (+ optional `headers` / OAuth) | POST JSON-RPC; JSON or SSE responses |

If both `url` and `command` are set, **URL wins**.

**Agent Plugins package MCP (s1326 + s1331):** root `mcp.json` closed parse + server map live in `internal/agentplugins`. **s1331 runtime wire:** opt-in `[plugins]` → `runtimewire.Wire` / `ConnectMCP` builds TOML `[[mcp.servers]]` **primary**, then appends plugin-mapped servers (fail-open Discover/map). Map ≠ Connected / process attach green · package wire ≠ install APPLY · not Agent Plugins GA · dual_write OFF · not Memory GA. See [agent-plugins.md](./agent-plugins.md) · [setup-lifecycle.md](./setup-lifecycle.md).

## Methods

| Method | Support |
|--------|---------|
| `initialize` / `notifications/initialized` | required |
| `tools/list` / `tools/call` | required |
| `resources/list` / `resources/read` | optional (fail-open if unsupported) |
| `prompts/list` / `prompts/get` | optional (fail-open if unsupported) |

Session header `Mcp-Session-Id` is stored on HTTP; `DELETE` on close is best-effort.

## Agent tools

| Tool | Mutating | Purpose |
|------|----------|---------|
| `mcp__<server>__<tool>` | per-server (default true) | Server tools |
| `list_mcp_resources` | no | Catalog URIs |
| `read_mcp_resource` | no | Read by URI |
| `list_mcp_prompts` | no | Prompt templates |
| `get_mcp_prompt` | no | Expand prompt + args |

**Agent integrations setup (s1238/s1242/s1243/s1247):** slash `/integrations` calls bare MCP tools `list_connector_catalog` / `plan_connector_setup` (mesh v178) and `get_webhook_signing_headers` (v30) when present on any connected server. TUI parses v178 `entries` + `oauth_install_supported` + honesty object. `/integrations status` (s1247) is a residual-honest operator pulse (MCP path · tools present · catalog honesty counts ≠ install green). Fail-open offline → portal HITL. Residual-honest path only — not install CRUD / not OAuth complete / not secret mint. See [agent-integrations-setup.md](./agent-integrations-setup.md).

## OAuth helpers

For HTTP servers:

```toml
[[mcp.servers]]
name = "remote"
url = "https://mcp.example.com/mcp"
oauth_token_env = "MCP_TOKEN"   # simplest: static Bearer

# Or client_credentials:
# [mcp.servers.oauth]
# token_url = "https://auth.example/oauth/token"
# client_id = "iomesh"
# client_secret_env = "MCP_CLIENT_SECRET"
# scopes = ["mcp"]
```

- Secrets only via **env** (`client_secret_env`, `oauth_token_env`) — never commit tokens.
- Token URL validated with `security.ValidateHTTPURL`.
- `PKCEChallenge` / `NewPKCEVerifier` available for future auth-code browser flows.

## Configuration examples

```toml
[mcp]
enabled = true
# inject_iomesh_context = false   # s1267 opt-in; default false

[[mcp.servers]]
name = "everything"
command = "npx"
args = ["-y", "@modelcontextprotocol/server-everything"]

[[mcp.servers]]
name = "remote"
url = "https://mcp.example.com/mcp"
oauth_token_env = "MCP_TOKEN"
mutating = false
# inject_iomesh_context = true    # per-server override of [mcp] flag
```

- **Fail-open**: bad servers skipped; missing resources/prompts soft-empty.
- **Output**: redacted + truncated (default 20k).

## Multi-tenant context headers (s1267)

Opt-in inject of iomesh multi-tenant context into **HTTP** MCP request headers at `ServerConfig` build time (`config.BuildMCPServerConfig` · `mcp.ApplyIOMeshContextHeaders`). Dial path already sends `cfg.Headers` on each POST.

| Header | Source |
|--------|--------|
| `X-IOMesh-Tenant` | `[iomesh].tenant`, else `[memory].tenant` |
| `X-IOMesh-Org` | `[iomesh].org` |
| `X-IOMesh-Workspace` | `[iomesh].workspace` |

```toml
[iomesh]
tenant = "acme"
org = "org_…"   # paste from console Agent/MCP (org_ + cuid2)
workspace = "ws_alpha"

[mcp]
enabled = true
inject_iomesh_context = true   # global default for all servers

[[mcp.servers]]
name = "mesh-scenario"
url = "https://mcp.example.com/mcp"
# inject_iomesh_context = false  # per-server opt-out
# headers = { "X-IOMesh-Org" = "org_explicit" }  # explicit wins; inject never overwrites
```

**Residual honesty**

| Claim | Truth |
|-------|--------|
| inject ≠ install APPLY / Connected / INSTALL_STORE green | Headers only; no install CRUD |
| inject ≠ dual-auth install list shipped | Peer candidacy only; not claimed here |
| empty values not sent | Never invent tenant/org/workspace |
| default `inject_iomesh_context = false` | Avoids surprising auth/context changes |
| stdio servers | No HTTP headers — inject is a no-op for command transport |
| dual_write / book-demo / GA | OFF · OFF · not invented |

Wire points: agent AttachMCP (`cmd/iomesh`), `iomesh mcp --connect`, memory pull MCP path, ACP session build — all via `BuildMCPServerConfig` / `mcpServerFromTOML`.

## CLI

```bash
iomesh mcp              # list servers
iomesh mcp --connect    # connect + tools/list (+ resource/prompt counts)
```

## Package

- `internal/mcp/client.go` — stdio + resources/prompts APIs  
- `internal/mcp/http.go` — streamable HTTP + SSE  
- `internal/mcp/context_headers.go` — s1267 `ApplyIOMeshContextHeaders`  
- `internal/mcp/oauth.go` — bearer / client_credentials / PKCE  
- `internal/config/mcp_server.go` — `BuildMCPServerConfig` (inject wire)  
- `internal/agent/mcp_tools.go` — agent registration  
