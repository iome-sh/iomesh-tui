# iomesh-tui

**I/O Mesh TUI** — a Go rewrite of [xAI Grok Build](https://github.com/xai-org/grok-build) with tighter **I/O Mesh** platform integration and a **DeepSeek-first** LLM cascade for price-performance.

> Status: **foundation** (router, agent, subagents, full-screen TUI, permissions, ACP stdio+WS, skills, **MCP stdio + streamable HTTP**). Mesh dogfood and TUI polish are next. Hardened for open-source readiness (path jail, secret scrubbing, CI).

## Why this rewrite

| Concern | Approach |
|---------|----------|
| Sustainable agent economics | Default **DeepSeek V4 Flash** ($0.14 / $0.28 per 1M); step-up **V4 Pro**; premium **Grok 4.5** |
| Integration simplicity | OpenAI-compatible HTTP + SSE in pure Go (`internal/router`) |
| Platform fit | Optional I/O Mesh context plane + `dept.*` usage streams (`internal/iomesh`) |
| Grok Build shapes | TUI / headless / ACP modes, tools, **subagents**, workspace root, slash commands |

## Quick start

```bash
# Requirements: Go 1.26.5+ (see go.mod)
git clone https://github.com/iome-sh/iomesh-tui.git
cd iomesh-tui

export DEEPSEEK_API_KEY=sk-...
# optional premium fallback:
# export XAI_API_KEY=...

make build
./bin/iomesh models
./bin/iomesh -p "List the top-level packages in this repo"
./bin/iomesh -c       # continue latest session (transcript + subagents)
./bin/iomesh sessions
./bin/iomesh          # full-screen TUI (Bubble Tea)
./bin/iomesh --repl   # classic line REPL
./bin/iomesh skills   # list SKILL.md catalogs
./bin/iomesh mcp      # list configured MCP servers
./bin/iomesh agent serve   # ACP WebSocket on 127.0.0.1:7400/acp
```

Copy [`configs/config.example.toml`](configs/config.example.toml) to `~/.iomesh/config.toml` to customize.

## Default model cascade

```
deepseek-v4-flash  →  deepseek-v4-pro  →  grok-4.5
     (routine)            (plan)          (high-stakes / fallback)
```

- Heuristic routing by task complexity and context size  
- Automatic fallback on rate limits / 5xx / network errors  
- Cost estimates (including cache hits) logged per call  
- Override: `iomesh -m deepseek-v4-pro` or `/model grok-4.5` in the REPL  

See [docs/architecture/llm-cascade.md](docs/architecture/llm-cascade.md).

## CLI

```text
iomesh [flags]              full-screen TUI (Bubble Tea)
iomesh --repl               classic line REPL
iomesh -p "prompt"          headless one-shot
iomesh -m <model>           pin logical model
iomesh -C <dir>             workspace root
iomesh --yolo               auto-approve mutating tools
iomesh --config <path>      config.toml
iomesh models               list catalog
iomesh version

# TUI / REPL slash commands
# /model <name|#>   pin model (or /models, /model default)
# /permissions      session always-allow tools
# /subagents        list child agents + worktree flag
# /save /sessions /load /cost /quit
```

## Layout

```text
cmd/iomesh/           binary entrypoint
internal/
  router/             LLM select + fallback + OpenAI HTTP/SSE client
  config/             TOML + env merge
  agent/              turn loop, tools, events, spawn_subagent wiring
  subagent/           child sessions (explore/plan/gp, background, caps)
  workspace/          rooted filesystem + path jail
  security/           redaction, env scrub, shell/URL policy
  iomesh/             I/O Mesh client (fail-open)
  tui/                full-screen Bubble Tea + classic REPL
  skills/             SKILL.md catalog loader
  mcp/                MCP stdio JSON-RPC client
configs/              example config.toml
docs/architecture/    design notes
docs/security.md      threat model
```

## I/O Mesh integration

```toml
[iomesh]
enabled = true
endpoint = "https://your-mesh.example.com"
tenant = "acme"
emit_dept_streams = true
context_plane = true
```

When enabled:

- **Context plane** — optional governed operational context injected into the system prompt (fail-open on errors)
- **Dept streams** — `dept.agent.llm_call` events with tokens, model, estimated USD

Offline / local use needs no mesh configuration.

## Security

Coding agents can read, write, and execute within a workspace. Key controls:

- **Path jail** with symlink escape checks and read size caps
- **Tool approval**: mutating tools (`write_file`, `run_shell`, `apply_worktree`, MCP tools by default, …) prompt y/n/a; headless/ACP deny without `--yolo` / `--always-approve` (fail-closed)
- **Skills / MCP**: project+user `SKILL.md` catalogs; opt-in MCP via **stdio** or **streamable HTTP/SSE** (`mcp__server__tool`)
- **Shell**: approval/`--yolo` required; API keys scrubbed from child env; dangerous pattern denylist
- **HTTP**: `http`/`https` only for model/mesh URLs; redacted error bodies
- Prefer env vars for secrets — never commit keys

See [SECURITY.md](SECURITY.md), [docs/security.md](docs/security.md), and [docs/architecture/permissions.md](docs/architecture/permissions.md). Report vulnerabilities privately (do not open public issues for exploits).

⚠️ **`--yolo` auto-approves mutating tools (write + shell + apply). Treat as full trust.**

## Development

```bash
make check      # fmt-check + vet + test
make test-race
make cover
make vuln       # govulncheck
make build
```

CI (GitHub Actions on every PR + merge to `main`): **lint**, **test** (race + coverage), **build**, **govulncheck**, aggregate **ci-success**. See [CONTRIBUTING.md](CONTRIBUTING.md#ci-on-pr-and-merge).

Contributions: [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT — see [LICENSE](LICENSE).

Grok Build is a separate project by SpaceXAI / xAI (Apache-2.0); this repository is an independent Go implementation inspired by its product surface, not a fork of the Rust sources.
