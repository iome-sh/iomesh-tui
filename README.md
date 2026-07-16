# iomesh-tui

**I/O Mesh TUI** — a Go rewrite of [xAI Grok Build](https://github.com/xai-org/grok-build) with tighter **I/O Mesh** platform integration and a **DeepSeek-first** LLM cascade for price-performance.

> Status: **foundation scaffold** (router, config, agent loop, workspace tools, REPL). Full-screen TUI, ACP, MCP, and subagents are planned next.

## Why this rewrite

| Concern | Approach |
|---------|----------|
| Sustainable agent economics | Default **DeepSeek V4 Flash** ($0.14 / $0.28 per 1M); step-up **V4 Pro**; premium **Grok 4.5** |
| Integration simplicity | OpenAI-compatible HTTP + SSE in pure Go (`internal/router`) |
| Platform fit | Optional I/O Mesh context plane + `dept.*` usage streams (`internal/iomesh`) |
| Grok Build shapes | TUI / headless / ACP modes, tools, **subagents**, workspace root, slash commands |

## Quick start

```bash
# Requirements: Go 1.22+
git clone https://github.com/iome-sh/iomesh-tui.git
cd iomesh-tui

export DEEPSEEK_API_KEY=sk-...
# optional premium fallback:
# export XAI_API_KEY=...

make build
./bin/iomesh models
./bin/iomesh -p "List the top-level packages in this repo"
./bin/iomesh          # interactive REPL scaffold
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
iomesh [flags]              interactive REPL (TUI scaffold)
iomesh -p "prompt"          headless one-shot
iomesh -m <model>           pin logical model
iomesh -C <dir>             workspace root
iomesh --yolo               auto-approve mutating tools
iomesh --config <path>      config.toml
iomesh models               list catalog
iomesh version
```

## Layout

```text
cmd/iomesh/           binary entrypoint
internal/
  router/             LLM select + fallback + OpenAI HTTP/SSE client
  config/             TOML + env merge
  agent/              turn loop, tools, events, spawn_subagent wiring
  subagent/           child sessions (explore/plan/gp, background, caps)
  workspace/          rooted filesystem + grep
  iomesh/             I/O Mesh client (fail-open)
  tui/                interactive REPL scaffold
configs/              example config.toml
docs/architecture/    design notes
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

## Development

```bash
make test
make vet
make build
```

## License

MIT — see [LICENSE](LICENSE).

Grok Build is a separate project by SpaceXAI / xAI (Apache-2.0); this repository is an independent Go implementation inspired by its product surface, not a fork of the Rust sources.
