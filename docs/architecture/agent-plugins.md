# Agent Plugins package client (s1326 + s1331 runtime wire)

**Pin:** free eng **s1326** — Agent Plugins **v1.0.0 package client** (discover + validate).  
**Pin:** free eng **s1331** — **opt-in runtime wire** of package skills + MCP into existing Skills / MCP runtimes.

Residual-honest: **package wire ≠ invent Agent Plugins GA**. Not Memory GA. Discover/load success ≠ Connected / install APPLY green. dual_write **OFF** (unchanged). book-demo **OFF**.

## What this is

| Surface | Status |
|---------|--------|
| `plugin.json` closed validation | **done** (`internal/agentplugins`) |
| Fixed-location skill discovery (`skills/<name>/SKILL.md`) | **done** |
| Fixed-location `mcp.json` parse + server map | **done** |
| `${PLUGIN_ROOT}` / `${PLUGIN_DATA}` expand helpers | **done** (args/env/cwd · never `command`) |
| Path confinement (`./` + stay in plugin root) | **done** |
| Opt-in `[plugins]` config | **done** (s1331 · default `enabled=false`) |
| Merge plugin skill dirs into skills catalog | **done** (s1331 · fail-open) |
| Map plugin MCP → `mcp.ServerConfig` (append after TOML) | **done** (s1331 · fail-open) |
| Approval gates on mutating MCP tools | **still apply** (plugin servers default Mutating=true) |
| Install / marketplace / enable UX | **out of scope** |
| Full Agent Plugins client GA | **not claimed** |

Package API entrypoint:

```go
import "github.com/iome-sh/iomesh-tui/internal/agentplugins"

p, err := agentplugins.Discover("/path/to/plugin")
// p.Manifest, p.Skills, p.MCPServers, p.Warnings

// s1331 runtime helpers (unit-tested):
plugins, warns := agentplugins.DiscoverAll(cfg.Plugins.Dirs)
skillDirs := agentplugins.SkillDirs(plugins) // pluginRoot/skills for LoadDirs
servers, mw := agentplugins.MCPServersFromPlugins(plugins, dataDirRoot)
// callers: TOML servers first, then append plugin servers
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

| Path | Role after s1331 |
|------|------------------|
| TOML `[[mcp.servers]]` | **Remains primary** MCP attach; plugin servers **append after** TOML |
| `[skills]` dirs + builtin | **Remains primary**; plugin `skills/` dirs **append** when plugins enabled |
| `internal/agentplugins` | Validation/discovery + runtime map helpers |
| Approval / yolo | Unchanged — plugin MCP tools default **mutating=true** (fail-closed) |

Do not invent “plugins installed / Connected / MCP attach green” from package discovery or map alone. Process connect still fail-open inside `mcp.NewManager`.

## Config (s1331 · opt-in)

```toml
# Agent Plugins package wire — opt-in (default enabled=false).
# Each dirs entry is a package root (plugin.json) or a parent of package roots.
# data_dir: root for per-plugin PLUGIN_DATA (default: <workspace>/.iomesh/plugin-data/<name>).
# Secrets never from portable plugin fields — only ${PLUGIN_ROOT}/${PLUGIN_DATA} expand.
# dual_write OFF · book-demo OFF · package wire ≠ Agent Plugins GA.
[plugins]
enabled = false
# dirs = ["~/.iomesh/plugins/my-plugin", "/opt/iomesh/plugins"]
# data_dir = "~/.iomesh/plugin-data"
```

### Runtime mapping details

- **Skills:** `SkillDirs` returns each `pluginRoot/skills` that has discovered `SKILL.md` children; passed to `skills.LoadWithBuiltin` after `[skills].dirs`.
- **MCP names:** `<pluginManifest.name>-<serverName>` (stable uniqueness across plugins).
- **stdio:** `command` never placeholder-expanded; `./` resolved under plugin root; `args`/`env`/`cwd` expanded; client injects absolute `PLUGIN_ROOT` + `PLUGIN_DATA` env; `PLUGIN_DATA` dir created (`MkdirAll`) before map return.
- **HTTP/SSE:** URL + headers (PLUGIN_* expanded in header **values** only). No secret invent.
- **Order:** TOML servers first, then plugin-mapped servers. Plugins-only MCP allowed when `[mcp] enabled` and TOML list empty.
- **Fail-open:** per dir / per plugin / per MCP server entry.

## Residual honesty locks

| Claim | Truth |
|-------|--------|
| package client candidacy | discover/validate + opt-in runtime wire |
| ≠ Agent Plugins GA | no marketplace/install UX · no product “plugins green” |
| ≠ Memory GA | orthogonal surface |
| dual_write | **OFF** (unchanged default; not a package concern) |
| book-demo | **OFF** |
| fail-open | per dir / component / entry |
| secrets | client-owned · never from portable `headers`/`env` package fields |
| MCP risk | higher than skills — approval gates still apply (mutating default true) |
| TOML MCP | still primary attach path; plugins append |
| install / Connected green | **not invented** from Discover or map success |

Peer: [Agent Plugins specification](https://agent-plugins.org/specification) v1.0.0 · related local docs [skills.md](./skills.md) · [mcp.md](./mcp.md).

## Tests

```bash
go test ./internal/agentplugins/...
```
