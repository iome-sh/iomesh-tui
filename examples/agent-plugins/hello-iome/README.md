# hello-iome (sample Agent Plugin)

Residual-honest **skills-only** sample package for operator dogfood of iomesh-tui Agent Plugins package client (s1326) + opt-in runtime wire (s1331).

**Honesty locks**

| Claim | Truth |
|-------|--------|
| sample package | ≠ invent **Agent Plugins GA** |
| skills | playbooks only · no secrets in `plugin.json` · no auto-send outbound |
| Memory | ≠ Memory GA (orthogonal) |
| dual_write | **OFF** (unchanged) |
| Discover / map success | ≠ install APPLY / Connected green |
| loading | requires **opt-in** `[plugins]` |

This package does **not** ship an MCP server (no network secrets, no process attach invent).

## Layout

```text
hello-iome/
├── plugin.json              # Agent Plugins 1.0.0 closed manifest
├── skills/
│   └── hello-iome/
│       └── SKILL.md         # residual-honest welcome / mesh orientation
└── README.md                # this file
```

## Enable (opt-in)

In your iomesh-tui config (e.g. `~/.config/iomesh/config.toml` or workspace config), add:

```toml
# Agent Plugins package wire — opt-in (default enabled=false).
# dual_write OFF · package wire ≠ Agent Plugins GA · Discover ≠ install green.
[plugins]
enabled = true
dirs = ["/absolute/path/to/iomesh-tui/examples/agent-plugins/hello-iome"]
# Optional: parent of package roots also works (discovers immediate children with plugin.json).
# dirs = ["/absolute/path/to/iomesh-tui/examples/agent-plugins"]
```

Replace the path with your clone of this repo. Relative paths are resolved by the config loader where supported; absolute paths are least ambiguous for dogfood.

Restart the TUI (or start a new session) after changing config. Plugin skill dirs append after `[skills].dirs` + builtin when plugins are enabled.

## Verify discover (library only)

From the iomesh-tui module root:

```bash
go test ./internal/agentplugins/ -run TestDiscover_HelloIomeExample -count=1
```

Or in Go:

```go
p, err := agentplugins.Discover("examples/agent-plugins/hello-iome") // from module root
// p.Manifest.Name == "hello-iome"; one skill; no MCP servers
```

## Non-goals

- No marketplace / install UX
- No invent “plugins Connected”
- No MCP server requiring secrets or network
- No CLI for plugins (sibling work)

See [docs/architecture/agent-plugins.md](../../../docs/architecture/agent-plugins.md).
