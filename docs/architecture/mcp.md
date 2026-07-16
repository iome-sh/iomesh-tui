# MCP client

Minimal [Model Context Protocol](https://modelcontextprotocol.io) client supporting:

| Transport | Config | Notes |
|-----------|--------|--------|
| **stdio** | `command` + `args` | Local process; scrubbed env + explicit `env` |
| **Streamable HTTP** | `url` (+ optional `headers` / OAuth) | POST JSON-RPC; JSON or SSE responses |

If both `url` and `command` are set, **URL wins**.

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

[[mcp.servers]]
name = "everything"
command = "npx"
args = ["-y", "@modelcontextprotocol/server-everything"]

[[mcp.servers]]
name = "remote"
url = "https://mcp.example.com/mcp"
oauth_token_env = "MCP_TOKEN"
mutating = false
```

- **Fail-open**: bad servers skipped; missing resources/prompts soft-empty.
- **Output**: redacted + truncated (default 20k).

## CLI

```bash
iomesh mcp              # list servers
iomesh mcp --connect    # connect + tools/list (+ resource/prompt counts)
```

## Package

- `internal/mcp/client.go` — stdio + resources/prompts APIs  
- `internal/mcp/http.go` — streamable HTTP + SSE  
- `internal/mcp/oauth.go` — bearer / client_credentials / PKCE  
- `internal/agent/mcp_tools.go` — agent registration  
