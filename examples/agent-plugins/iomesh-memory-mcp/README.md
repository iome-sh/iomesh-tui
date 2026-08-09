# iomesh-memory-mcp (sample Agent Plugin · product edge)

Residual-honest **stdio map** sample package for operator dogfood of iomesh-tui Agent Plugins package client against the **public product edge** Memory MCP binary (`iomesh-memory-mcp`).

**Honesty locks**

| Claim | Truth |
|-------|--------|
| sample package | ≠ invent **Agent Plugins GA** |
| MCP map success | ≠ process **Connected** / install APPLY green |
| Memory | ≠ **Memory GA** · local-primary edge only |
| dual_write | **OFF** (unchanged) |
| freemium hosted palace | **not** claimed |
| secrets in package JSON | **none** — portable map only |
| public install | `go install …@main` · **no GOPRIVATE** / PAT |
| connect | requires **`iomesh-memory-mcp` on PATH** · fail-open if missing |

This package **maps** a stdio server entry named `memory` → command `iomesh-memory-mcp`. It does **not** ship the binary, mint secrets, or invent attach green. Runtime attach still fail-opens inside `mcp.NewManager` when the binary is absent.

## Layout

```text
iomesh-memory-mcp/
├── plugin.json                         # Agent Plugins 1.0.0 closed manifest
├── mcp.json                            # stdio map: memory → iomesh-memory-mcp (no secrets)
├── skills/
│   └── iomesh-memory-local/
│       └── SKILL.md                    # residual-honest public product edge playbook
└── README.md                           # this file
```

## Prerequisites (operator · s1478 public)

Both product edge modules are **public** — no `GOPRIVATE` / PAT:

```bash
go install github.com/iome-sh/iomesh-memory-mcp/cmd/iomesh-memory-mcp@main
# optional kernel tip:
go get github.com/iome-sh/memory@main
```

Or clone [`github.com/iome-sh/iomesh-memory-mcp`](https://github.com/iome-sh/iomesh-memory-mcp) and `go build` / `docker compose up --build`.

Ensure `iomesh-memory-mcp` is on **PATH** for the TUI process (or prefer TOML `[[mcp.servers]]` HTTP/stdio). Connect is **fail-open**: Discover/map of this package succeeds even if the binary is missing.

No secrets belong in `plugin.json` or `mcp.json`.

## Enable (opt-in)

```toml
# Agent Plugins package wire — opt-in (default enabled=false).
# dual_write OFF · package wire ≠ Agent Plugins GA · Discover ≠ install green · not Memory GA.
[plugins]
enabled = true
dirs = ["/absolute/path/to/iomesh-tui/examples/agent-plugins/iomesh-memory-mcp"]
# Optional: parent of package roots also works (discovers immediate children with plugin.json).
# dirs = ["/absolute/path/to/iomesh-tui/examples/agent-plugins"]
```

When plugins are enabled:

- Skill dirs append after `[skills].dirs` + builtin.
- Plugin MCP servers **append after** TOML `[[mcp.servers]]` (TOML remains primary).
- Mapped server name: `iomesh-memory-mcp-memory` (`<pluginManifest.name>-<serverName>`).

### Alternative: TOML-only attach (primary path)

```toml
[mcp]
enabled = true

[[mcp.servers]]
name = "iomesh-memory-mcp"
url = "http://127.0.0.1:8080/mcp"
allow_loopback = true
mutating = true

# Alternate stdio:
# command = "iomesh-memory-mcp"
# args = ["-palace-root", "./data/memory-palaces", "-tenant", "default"]

[memory]
enabled = true
server = "iomesh-memory-mcp"
tenant = "default"
dual_write = false   # OFF · not primary palace · package load ≠ Memory GA
```

## Verify discover (library only)

From the iomesh-tui module root:

```bash
go test ./internal/agentplugins/ -run TestDiscover_IomeshMemoryMCPExample -count=1
iomesh plugins validate -dir examples/agent-plugins/iomesh-memory-mcp
iomesh plugins list -dir examples/agent-plugins/iomesh-memory-mcp
iomesh plugins dogfood   # hello-iome + iomesh-memory-mcp product samples
```

## Non-goals

- No marketplace / install UX
- No invent “plugins Connected” or Memory GA
- No secrets in portable package fields
- No freemium hosted palace claim
- No auto dual_write
- No bundled binary
- s1517: product-only sample in TUI (no residual aion Memory sample tree)

See [docs/architecture/agent-plugins.md](../../../docs/architecture/agent-plugins.md), [memory-mcp.md](../../../docs/architecture/memory-mcp.md), and the end-to-end [memory-edge-usage-demo.md](../../../docs/architecture/memory-edge-usage-demo.md) (s1513).
