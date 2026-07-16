# MCP client

Minimal [Model Context Protocol](https://modelcontextprotocol.io) client supporting:

| Transport | Config | Notes |
|-----------|--------|--------|
| **stdio** | `command` + `args` | Local process; scrubbed env + explicit `env` |
| **Streamable HTTP** | `url` (+ optional `headers`) | POST JSON-RPC; `application/json` or `text/event-stream` (SSE) responses |

If both `url` and `command` are set, **URL wins**.

## Supported methods

- `initialize` / `notifications/initialized`
- `tools/list`
- `tools/call`

Session header `Mcp-Session-Id` is stored and sent on subsequent HTTP requests; `DELETE` on close is best-effort.

## Configuration

```toml
[mcp]
enabled = true

# Stdio
[[mcp.servers]]
name = "everything"
command = "npx"
args = ["-y", "@modelcontextprotocol/server-everything"]
# mutating = true
# env = { "FOO" = "bar" }

# Streamable HTTP
[[mcp.servers]]
name = "remote"
url = "https://mcp.example.com/mcp"
# headers = { "Authorization" = "Bearer …" }
# allow_loopback = true
# mutating = false
```

- **Fail-open**: a single server that fails to connect is skipped; others still connect.
- **HTTP URL policy**: `http`/`https` only via `security.ValidateHTTPURL` (loopback allowed by default).
- **Output**: tool results redacted and truncated (default 20_000 bytes).

## Agent tool names

```text
mcp__<server>__<tool>
```

Servers default to **mutating=true** so tools go through y/n/a approval. Set `mutating = false` for read-only servers.

## CLI

```bash
iomesh mcp              # list configured servers (shows stdio vs http)
iomesh mcp --connect    # connect + tools/list (requires enabled=true)
```

## Package

- `internal/mcp/client.go` — stdio client
- `internal/mcp/http.go` — streamable HTTP + SSE parse
- `internal/mcp/manager.go` — multi-server, fail-open
- `internal/agent/mcp_tools.go` — agent registry binding
