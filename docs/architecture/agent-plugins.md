# Agent Plugins package client (s1326)

**Pin:** free eng **s1326** — first Agent Plugins **v1.0.0 package client slice** in `iomesh-tui`.

Residual-honest: **discovery + closed validation only**. Not full Agent Plugins client GA. Not Memory GA. Package client candidacy ≠ product install green.

## What this is

| Surface | Status |
|---------|--------|
| `plugin.json` closed validation | **done** (`internal/agentplugins`) |
| Fixed-location skill discovery (`skills/<name>/SKILL.md`) | **done** (map `SkillRef` only) |
| Fixed-location `mcp.json` parse + server map | **done** (structs only) |
| `${PLUGIN_ROOT}` / `${PLUGIN_DATA}` expand helpers | **done** (args/env/cwd) |
| Path confinement (`./` + stay in plugin root) | **done** |
| Runtime attach of MCP processes from package | **not yet** |
| Wiring package skills into agent `list_skills` | **not yet** |
| Install / marketplace / enable UX | **out of scope** |

Package API entrypoint:

```go
import "github.com/iome-sh/iomesh-tui/internal/agentplugins"

p, err := agentplugins.Discover("/path/to/plugin")
// p.Manifest, p.Skills, p.MCPServers, p.Warnings
```

## Package layout (Agent Plugins 1.0.0)

```text
my-plugin/
├── plugin.json          # required
├── skills/
│   └── summarize/
│       └── SKILL.md     # immediate children only — no deep recurse
└── mcp.json             # optional
```

### `plugin.json`

- Closed top-level keys: `$schema`, `name`, `version`, `description`, `author`, `homepage`, `repository`, `license`, `keywords`, `extensions`
- Unknown keys: **warn + ignore** (not fatal)
- Fatal: unreadable / invalid JSON · missing or invalid `name`
- When `$schema` is present it **must** be exactly  
  `https://agent-plugins.org/schemas/1.0.0/plugin.schema.json` (unsupported → reject plugin)
- `name`: 1–64, `[a-z0-9.-]`, start/end alnum, no `--` or `..`

### Skills

- Only immediate children of `skills/` that contain a regular file `SKILL.md`
- Missing `skills/` is OK
- Invalid / escaped skill paths skipped (fail-open per skill)

### `mcp.json`

- Schema id: `https://agent-plugins.org/schemas/1.0.0/mcp.schema.json`
- Top-level: `$schema` + `mcpServers` only
- Per entry `type`: `stdio` | `streamable-http` | `sse`
- Invalid file → **MCP component disabled** for that plugin; skills still load
- Invalid / unsupported entry → **skip that server**
- `command`: bare executable or `./` plugin-relative — **no** placeholder expansion
- Placeholders `${PLUGIN_ROOT}` / `${PLUGIN_DATA}` only in **args / env values / cwd**
- `env` must not set `PLUGIN_ROOT` or `PLUGIN_DATA` (client-owned)
- Headers / env from package are **visible package data** — not a portable secret mechanism; **secrets never from portable fields** (client-owned auth only)
- Missing `mcp.json` is OK

## Relationship to existing paths

| Path | Role after s1326 |
|------|------------------|
| TOML `[[mcp.servers]]` | **Remains primary** MCP attach until package load is wired |
| `[skills]` dirs + builtin | **Remains primary** skill load until package skills are merged |
| `internal/agentplugins` | Validation/discovery library only |

Do not invent “plugins installed / Connected / MCP attach green” from package discovery alone.

## Optional future config (not wired)

Document-only sketch — **not** read by main in s1326:

```toml
# [plugins]
# enabled = false
# dirs = ["~/.iomesh/plugins", ".iomesh/plugins"]
```

## Residual honesty locks

| Claim | Truth |
|-------|--------|
| package client candidacy | discovery/validation slice only |
| ≠ Memory GA | orthogonal surface |
| dual_write | **OFF** (unchanged default; not a package concern) |
| book-demo | **OFF** |
| fail-open | per component / per entry |
| secrets | client-owned · never from portable `headers`/`env` package fields |
| MCP risk | higher than skills — map structs only; no process attach yet |
| TOML MCP | still primary attach path |
| install green | **not invented** from Discover success |

Peer: [Agent Plugins specification](https://agent-plugins.org/specification) v1.0.0 · related local docs [skills.md](./skills.md) · [mcp.md](./mcp.md).

## Tests

```bash
go test ./internal/agentplugins/...
```
