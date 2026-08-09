# Agent Plugins package client (s1326 + s1331 runtime wire + s1336 CLI + s1337/s1346 samples + s1357 dogfood)

**Pin:** free eng **s1326** — Agent Plugins **v1.0.0 package client** (discover + validate).  
**Pin:** free eng **s1331** — **opt-in runtime wire** of package skills + MCP into existing Skills / MCP runtimes.  
**Pin:** free eng **s1336** — operator DX CLI `iomesh plugins list|validate`.
**Pin:** free eng **s1337** — residual-honest **sample package** [`examples/agent-plugins/hello-iome`](../../examples/agent-plugins/hello-iome) (skills-only dogfood).
**Pin:** free eng **s1346** — residual-honest **sample package** [`examples/agent-plugins/aion-memory-mcp`](../../examples/agent-plugins/aion-memory-mcp) (stdio map of private platform residual `aion-memory-mcp` · not product naming · not Memory GA).
**Pin:** free eng **s1357** — offline residual-honest `iomesh plugins dogfood` (validates **both** product samples; no MCP dial · PATH residual for binary).
**Pin:** free eng **s1478** — product sample [`examples/agent-plugins/iomesh-memory-mcp`](../../examples/agent-plugins/iomesh-memory-mcp) (public product host stdio map · dogfood primary with hello-iome).

Residual-honest: **package wire ≠ invent Agent Plugins GA**. Not Memory GA. Discover/load success ≠ Connected / install APPLY green. dual_write **OFF** (unchanged). book-demo **OFF**. Sample package ≠ GA. list/validate/dogfood ≠ invent Agent Plugins GA. dogfood PASS ≠ Connected / Memory GA · PATH residual for `iomesh-memory-mcp` binary.

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
| Operator CLI `iomesh plugins list\|validate` | **done** (s1336 · format helpers in `cli_format.go`) |
| Offline samples dogfood `iomesh plugins dogfood` | **done** (s1357 · both samples · discover/validate only · no MCP dial) |
| Approval gates on mutating MCP tools | **still apply** (plugin servers default Mutating=true) |
| Install / marketplace / enable UX | **out of scope** |
| Full Agent Plugins client GA | **not claimed** |
| Sample skills-only package (`hello-iome`) | **done** (s1337 · dogfood primary · opt-in `[plugins]`) |
| Sample product stdio memory map (`iomesh-memory-mcp`) | **done** (s1478 · public product map · binary on PATH for connect · not Memory GA · dual_write OFF) |
| Sample residual stdio memory map (`aion-memory-mcp`) | **done** (s1346 · private platform residual · not product naming · not Memory GA · dual_write OFF) |

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


## Sample packages

### `hello-iome` (s1337 · skills-only)

In-repo dogfood package (skills-only; no MCP server, no secrets):

- Path: [`examples/agent-plugins/hello-iome`](../../examples/agent-plugins/hello-iome)
- Enable via opt-in `[plugins]` — see that package's [README](../../examples/agent-plugins/hello-iome/README.md)
- Loading requires `enabled = true` + `dirs` pointing at the package root (or parent of package roots)
- Discover/map of the sample ≠ install Connected / Agent Plugins GA

### `iomesh-memory-mcp` (s1478 · product stdio map · dogfood primary)

In-repo **product** dogfood package that **maps** public product edge Memory MCP via stdio (no secrets; binary not shipped):

- Path: [`examples/agent-plugins/iomesh-memory-mcp`](../../examples/agent-plugins/iomesh-memory-mcp)
- `mcp.json`: server key `memory`, type `stdio`, command `iomesh-memory-mcp` (skill `iomesh-memory-local`)
- Public install: `go install github.com/iome-sh/iomesh-memory-mcp/cmd/iomesh-memory-mcp@main` · **no GOPRIVATE**
- Operator must put binary on **PATH**; connect is fail-open if missing
- Mapped runtime name: `iomesh-memory-mcp-memory` (`<manifest.name>-<serverName>`)
- Enable via opt-in `[plugins]` — see that package's [README](../../examples/agent-plugins/iomesh-memory-mcp/README.md)
- Discover/map success ≠ process Connected / install APPLY / **Memory GA** · dual_write **OFF** · not freemium hosted palace
- TOML `[[mcp.servers]]` remains the **primary** attach path; package map is portable dogfood

### `aion-memory-mcp` (s1346 · residual private stdio map · not product)

Residual in-repo sample that **maps** private monorepo platform Memory MCP via stdio (not product edge naming):

- Path: [`examples/agent-plugins/aion-memory-mcp`](../../examples/agent-plugins/aion-memory-mcp)
- `mcp.json`: server key `memory`, type `stdio`, command `aion-memory-mcp` (optional skill `aion-memory-local`)
- Operator must install the private monorepo binary and put it on **PATH**; connect is fail-open if missing
- Mapped runtime name: `aion-memory-mcp-memory` (`<manifest.name>-<serverName>`)
- **Not** required for `iomesh plugins dogfood` (product samples are hello-iome + iomesh-memory-mcp)
- Discover/map success ≠ process Connected / install APPLY / **Memory GA** · dual_write **OFF**

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

## CLI operator DX (s1336 + s1357 dogfood)

```bash
iomesh plugins              # alias: list
iomesh plugins list         # table: NAME VERSION SKILLS MCP WARN ROOT
iomesh plugins validate     # OK/FAIL per package; exit 1 on fatal or zero OK when dirs set
iomesh plugins dogfood      # offline validate both in-repo samples (s1357)
iomesh plugins help
```

### Flags

| Flag | Role |
|------|------|
| `--config path` | config.toml (default user config) — list/validate |
| `-dir path` | package root **or** parent of roots; **repeatable**; comma-separated OK — list/validate |
| `-module-root path` | module root containing `examples/agent-plugins/*` (default: walk up from cwd for `go.mod`) — dogfood |

`-dir` **supplements** `[plugins].dirs` for one-shot list/validate without enabling runtime wire. Empty dirs + plugins default-off → residual-honest opt-in message (not a fake “plugins green”).

### Behavior

| Subcommand | Behavior |
|------------|----------|
| **list** | `DiscoverAll` on merged dirs; stdout table; stderr per-dir / per-plugin warnings + residual honesty footer. Fail-open (empty table / residual footer, exit 0). |
| **validate** | `ValidateDirs` (Discover per package root); stdout `OK` / `FAIL` lines; stderr plugin warnings + honesty. **Exit 1** if any fatal FAIL **or** zero plugins OK when dirs were specified. |
| **dogfood** | Resolves both product sample dirs (`hello-iome` + `iomesh-memory-mcp`) under module root; `ValidateDirs` per sample; stdout `OK`/`FAIL` + summary; stderr PATH residual + honesty. **Exit 1** if any fatal, missing sample, or not both expected samples OK. **No** MCP Dial / process connect · **does not** require `iomesh-memory-mcp` on PATH. Residual `aion-memory-mcp` sample is optional (not dogfood-required). |

Helpers: `SamplePluginRelPaths` / `DefaultSamplePluginDirs` / `FindModuleRoot` / `DogfoodSamples` / `DogfoodPass` in `internal/agentplugins/dogfood.go` (unit-tested).

### Honesty (CLI)

- list/validate/dogfood ≠ invent Agent Plugins GA
- dual_write **OFF** · Discover ≠ Connected · not Memory GA · book-demo **OFF**
- dogfood PASS ≠ invent Agent Plugins GA · PATH residual for binary · connect skip
- CLI success ≠ runtime wire / MCP attach / install APPLY green
- `[plugins]` remains opt-in (`enabled=false` default); CLI can inspect packages via `-dir` without enabling

Pure format helpers live in `internal/agentplugins/cli_format.go` (unit-tested). Dogfood helpers: `dogfood.go`.

## Residual honesty locks

| Claim | Truth |
|-------|--------|
| package client candidacy | discover/validate + opt-in runtime wire + operator CLI + samples dogfood |
| ≠ Agent Plugins GA | no marketplace/install UX · no product “plugins green” |
| ≠ Memory GA | orthogonal surface · iomesh-memory-mcp / aion-memory-mcp samples are map only |
| dual_write | **OFF** (unchanged default; not a package concern) |
| book-demo | **OFF** |
| fail-open | per dir / component / entry (list); validate/dogfood surfaces fatals |
| secrets | client-owned · never from portable `headers`/`env` package fields |
| MCP risk | higher than skills — approval gates still apply (mutating default true) |
| TOML MCP | still primary attach path; plugins append |
| install / Connected green | **not invented** from Discover, list, validate, dogfood, or map success |
| PATH residual | dogfood does **not** require `iomesh-memory-mcp` binary; connect remains separate |

Peer: [Agent Plugins specification](https://agent-plugins.org/specification) v1.0.0 · related local docs [skills.md](./skills.md) · [mcp.md](./mcp.md).

**Note (s1341):** builtin skills such as `gtm-draft-only-agent` ship via `go:embed` under `internal/skills/builtin/` — they are **not** Agent Plugins packages (orthogonal residual surface; skills loader remains primary for builtins).

## Tests

```bash
go test ./internal/agentplugins/...
go build ./cmd/iomesh/
# from module root (offline; no PATH binary required):
./bin/iomesh plugins dogfood
```
