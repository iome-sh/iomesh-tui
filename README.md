# iomesh-tui

[![CI](https://github.com/iome-sh/iomesh-tui/actions/workflows/ci.yml/badge.svg)](https://github.com/iome-sh/iomesh-tui/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/github/go-mod/go-version/iome-sh/iomesh-tui)](go.mod)

**I/O Mesh TUI** — a Go coding-agent harness inspired by [xAI Grok Build](https://github.com/xai-org/grok-build), with **I/O Mesh** platform hooks and a **DeepSeek-first** LLM cascade for price-performance.

> **Status:** public open-source **foundation** (pre-1.0). Agent loop, subagents, full-screen TUI, permissions, ACP, skills, MCP, mesh dogfood, optional Gemini/Vertex. Not a multi-tenant remote sandbox — see [SECURITY.md](SECURITY.md).

## Table of contents

- [Why this project](#why-this-project)
- [Quick start](#quick-start)
- [CLI](#cli)
- [Configuration](#configuration)
- [Security](#security)
- [Documentation](#documentation)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

## Why this project

| Concern | Approach |
|---------|----------|
| Sustainable agent economics | Default **DeepSeek V4 Flash**; step-up **V4 Pro**; premium **Grok 4.5** |
| Integration simplicity | OpenAI-compatible HTTP + SSE in pure Go (`internal/router`) |
| Platform fit | Optional I/O Mesh context plane + `dept.*` usage streams (`internal/iomesh`) |
| Familiar agent UX | TUI / headless / ACP, tools, subagents, workspace root, slash commands |

## Quick start

**Requirements:** Go version in [go.mod](go.mod) (CI uses that toolchain).

```bash
git clone https://github.com/iome-sh/iomesh-tui.git
cd iomesh-tui

export DEEPSEEK_API_KEY=…          # required for default cascade
# export XAI_API_KEY=…             # optional Grok fallback
# export GEMINI_API_KEY=…          # optional Gemini AI Studio
# export GOOGLE_CLOUD_PROJECT=…    # optional Vertex
# export VERTEX_API_KEY=…          # GCP access token for Vertex OpenAI-compat

make build
./bin/iomesh models
./bin/iomesh -p "List the top-level packages in this repo"
# optional pins:
# ./bin/iomesh -m gemini-2.5-flash -p "Reply with ok"
# ./bin/iomesh -m vertex-gemini-2.5-flash -p "Reply with ok"
./bin/iomesh                       # full-screen TUI (TTY)
./bin/iomesh --repl                # classic line REPL
./bin/iomesh -c                    # continue latest session
./bin/iomesh sessions | skills | mcp
./bin/iomesh agent serve           # ACP WebSocket (127.0.0.1:7400/acp)
./bin/iomesh mesh dogfood          # mesh smoke (needs IOMESH_ENDPOINT)
make dogfood-unit                  # offline mesh tests
```

Optional: copy [`.env.example`](.env.example) for local env vars (iomesh reads the **process environment**; it does not auto-load `.env` files yet). Copy [`configs/config.example.toml`](configs/config.example.toml) to `~/.iomesh/config.toml` to customize.

### Default model cascade

```text
deepseek-v4-flash  →  deepseek-v4-pro  →  grok-4.5
     (routine)            (plan)          (high-stakes / fallback)
```

**Optional Google runtimes** (not in the default cascade — pin with `-m` / `/model`):

| Logical name | Backend | Auth |
|--------------|---------|------|
| `gemini-2.5-flash` / `gemini-2.5-pro` | Gemini API (AI Studio) OpenAI-compat | `GEMINI_API_KEY` |
| `vertex-gemini-2.5-flash` / `vertex-gemini-2.5-pro` | Vertex AI OpenAI-compat | `GOOGLE_CLOUD_PROJECT` + `VERTEX_API_KEY` |

See [docs/architecture/llm-cascade.md](docs/architecture/llm-cascade.md).

## CLI

```text
iomesh [flags]              full-screen TUI (Bubble Tea)
iomesh --repl               classic line REPL
iomesh -p "prompt"          headless one-shot
iomesh -m <model>           pin logical model
iomesh -C <dir>             workspace root
iomesh --yolo               auto-approve mutating tools (full trust)
iomesh --config <path>      config.toml
iomesh models | sessions | skills | mcp | version
iomesh mesh dogfood         I/O Mesh smoke (needs IOMESH_ENDPOINT)
iomesh agent stdio          ACP JSON-RPC over stdio
iomesh agent serve          ACP WebSocket (default 127.0.0.1:7400/acp)
```

Slash commands (TUI/REPL): `/model`, `/theme`, `/permissions`, `/subagents`, `/save`, `/sessions`, `/load`, `/cost`, `/help`, `/quit`.  
Keys (fullscreen): **Enter** send · **Ctrl+J** newline · **y/n/a** tool approval.

## Configuration

Precedence: **CLI flags** → **environment** (`IOMESH_*`, `DEEPSEEK_API_KEY`, `GEMINI_API_KEY`, …) → **`~/.iomesh/config.toml`** → built-in defaults.

I/O Mesh (optional, fail-open):

```toml
[iomesh]
enabled = true
endpoint = "https://your-mesh.example.com"
tenant = "acme"
emit_dept_streams = true
context_plane = true
```

## Security

Coding agents can read, write, and execute **inside a workspace** when you approve (or pass `--yolo`).

| Control | Behavior |
|---------|----------|
| Path jail | Symlink escape checks, read size caps |
| Tool approval | Mutators prompt y/n/a; headless/ACP **deny** without `--yolo` |
| Shell | Scrubbed env; catastrophic pattern denylist |
| HTTP | `http`/`https` only; redacted error bodies |
| Secrets | Prefer env vars — never commit keys |

**Report vulnerabilities privately** — [SECURITY.md](SECURITY.md) (GitHub private advisory or security@iome.sh). Do **not** open public issues for exploits.

⚠️ **`--yolo` auto-approves mutating tools (write, shell, apply, MCP tools by default). Treat as full trust of the model and tools.**

Details: [docs/security.md](docs/security.md) · [docs/architecture/permissions.md](docs/architecture/permissions.md).

## Documentation

Index: **[docs/README.md](docs/README.md)** (architecture, MCP, ACP, TUI, mesh dogfood, …).

Open-source process: [CONTRIBUTING](CONTRIBUTING.md) · [SUPPORT](SUPPORT.md) · [RELEASING](RELEASING.md) · [CHANGELOG](CHANGELOG.md) · [docs/OPEN_SOURCE_AUDIT.md](docs/OPEN_SOURCE_AUDIT.md).

## Layout

```text
cmd/iomesh/           binary entrypoint
internal/
  router/             LLM select + fallback + OpenAI HTTP/SSE
  config/             TOML + env merge
  agent/              turn loop, tools, events, MCP/skill wiring
  subagent/           explore/plan/gp, parallel, worktrees
  workspace/          rooted FS + path jail
  security/           redaction, env scrub, shell/URL policy
  iomesh/             I/O Mesh client + dogfood
  tui/                full-screen Bubble Tea + classic REPL
  skills/             SKILL.md catalog
  mcp/                MCP stdio/HTTP, resources, prompts, OAuth
  acp/                Agent Client Protocol (stdio + WebSocket)
  session/            transcript persistence
configs/              example config.toml
scripts/              mesh dogfood helper
docs/                 architecture + security
```

## Development

```bash
make check      # fmt-check + vet + test
make test-race
make cover
make vuln       # govulncheck
make build
make ci         # full local gate
```

CI (GitHub Actions on every PR + merge to `main`): **lint**, **test** (race + coverage), **build**, **govulncheck**, aggregate **ci-success**.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) and the [Code of Conduct](CODE_OF_CONDUCT.md). Use [issue templates](https://github.com/iome-sh/iomesh-tui/issues/new/choose) for bugs and features.

## License

MIT — see [LICENSE](LICENSE) and [NOTICE](NOTICE).

Grok Build is a separate project by xAI (Apache-2.0). This repository is an **independent Go implementation** inspired by its product surface, not a fork of the Rust sources.
