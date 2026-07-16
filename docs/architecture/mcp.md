# MCP client (stdio)

Minimal [Model Context Protocol](https://modelcontextprotocol.io) client over **stdio JSON-RPC**.

## Supported methods

- `initialize` / `notifications/initialized`
- `tools/list`
- `tools/call`

HTTP/SSE transports are not implemented yet.

## Configuration

```toml
[mcp]
enabled = true

[[mcp.servers]]
name = "everything"
command = "npx"
args = ["-y", "@modelcontextprotocol/server-everything"]
# mutating = true   # default: tools require approval
# env = { "FOO" = "bar" }
# startup_timeout_sec = 30
# tool_timeout_sec = 120
```

- **Fail-open**: a single server that fails to start is skipped; others still connect.
- **Env**: child process starts from scrubbed env + explicit `env` map only (no ambient API keys).
- **Output**: tool results redacted and truncated (default 20_000 bytes).

## Agent tool names

```text
mcp__<server>__<tool>
```

Example: server `github` tool `create_issue` → `mcp__github__create_issue`.

Servers default to **mutating=true** so tools go through the same y/n/a approval path as `write_file` / `run_shell`. Set `mutating = false` for read-only servers.

## CLI

```bash
iomesh mcp              # list configured servers
iomesh mcp --connect    # spawn + tools/list (slow; requires enabled=true)
```

## Package

- `internal/mcp` — client, manager, bindings
- `internal/agent/mcp_tools.go` — registration into the tool registry
