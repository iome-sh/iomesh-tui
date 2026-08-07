# aion-memory-mcp (sample Agent Plugin)

Residual-honest **stdio map** sample package for operator dogfood of iomesh-tui Agent Plugins package client (s1326) + opt-in runtime wire (s1331) against the **local-primary** Memory MCP binary.

**Honesty locks**

| Claim | Truth |
|-------|--------|
| sample package | ≠ invent **Agent Plugins GA** |
| MCP map success | ≠ process **Connected** / install APPLY green |
| Memory | ≠ **Memory GA** · local-primary edge only |
| dual_write | **OFF** (unchanged) |
| freemium hosted palace | **not** claimed |
| secrets in package JSON | **none** — portable map only |
| connect | requires **`aion-memory-mcp` on PATH** · fail-open if missing |

This package **maps** a stdio server entry named `memory` → command `aion-memory-mcp`. It does **not** ship the binary, mint secrets, or invent attach green. Runtime attach still fail-opens inside `mcp.NewManager` when the binary is absent.

## Layout

```text
aion-memory-mcp/
├── plugin.json                         # Agent Plugins 1.0.0 closed manifest
├── mcp.json                            # stdio map: memory → aion-memory-mcp (no secrets)
├── skills/
│   └── aion-memory-local/
│       └── SKILL.md                    # residual-honest local memory edge playbook
└── README.md                           # this file
```

## Prerequisites (operator)

1. Install / build the platform **`aion-memory-mcp`** binary (from aion / release artifacts — not invented here).
2. Ensure it is on **`PATH`** for the TUI process (or replace the command via a local fork of this package / prefer TOML `[[mcp.servers]]`).
3. Connect is **fail-open**: Discover/map of this package succeeds even if the binary is missing; attach will fail at runtime without inventing Connected green.

No secrets belong in `plugin.json` or `mcp.json`. Client-owned env (`PLUGIN_ROOT` / `PLUGIN_DATA`) is injected by the package runtime wire when plugins are enabled.

## Enable (opt-in)

In your iomesh-tui config (e.g. `~/.config/iomesh/config.toml` or workspace config), add:

```toml
# Agent Plugins package wire — opt-in (default enabled=false).
# dual_write OFF · package wire ≠ Agent Plugins GA · Discover ≠ install green · not Memory GA.
[plugins]
enabled = true
dirs = ["/absolute/path/to/iomesh-tui/examples/agent-plugins/aion-memory-mcp"]
# Optional: parent of package roots also works (discovers immediate children with plugin.json).
# dirs = ["/absolute/path/to/iomesh-tui/examples/agent-plugins"]
```

Replace the path with your clone of this repo. Absolute paths are least ambiguous for dogfood.

Restart the TUI (or start a new session) after changing config. When plugins are enabled:

- Skill dirs append after `[skills].dirs` + builtin.
- Plugin MCP servers **append after** TOML `[[mcp.servers]]` (TOML remains primary).
- Mapped server name: `aion-memory-mcp-memory` (`<pluginManifest.name>-<serverName>`).

### Alternative: TOML-only attach (primary path)

You do **not** need this package to attach local memory MCP. Prefer TOML when you already run a binary:

```toml
[mcp]
enabled = true

[[mcp.servers]]
name = "memory"
command = "aion-memory-mcp"
# args optional — see docs/architecture/memory-mcp.md
```

Package map is portable discover/dogfood; TOML remains the primary attach path.

## Verify discover (library only)

From the iomesh-tui module root:

```bash
go test ./internal/agentplugins/ -run TestDiscover_AionMemoryMCPExample -count=1
iomesh plugins validate -dir examples/agent-plugins/aion-memory-mcp
iomesh plugins list -dir examples/agent-plugins/aion-memory-mcp
```

Or in Go:

```go
p, err := agentplugins.Discover("examples/agent-plugins/aion-memory-mcp") // from module root
// p.Manifest.Name == "aion-memory-mcp"
// one skill aion-memory-local; one MCP server type stdio command aion-memory-mcp
cfgs, _ := agentplugins.MCPServersFromPlugins([]*agentplugins.Plugin{p}, dataDirRoot)
// cfgs[0].Name == "aion-memory-mcp-memory"; Command == "aion-memory-mcp"
// map success ≠ process Connected
```

## Non-goals

- No marketplace / install UX
- No invent “plugins Connected” or Memory GA
- No secrets in portable package fields
- No freemium hosted palace claim
- No auto dual_write
- No bundled `aion-memory-mcp` binary

See [docs/architecture/agent-plugins.md](../../../docs/architecture/agent-plugins.md) and [memory-mcp.md](../../../docs/architecture/memory-mcp.md).
