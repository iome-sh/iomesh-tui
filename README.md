# iomesh-tui

[![CI](https://github.com/iome-sh/iomesh-tui/actions/workflows/ci.yml/badge.svg)](https://github.com/iome-sh/iomesh-tui/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/github/go-mod/go-version/iome-sh/iomesh-tui)](go.mod)

**I/O Mesh TUI** — a Go coding-agent harness inspired by [xAI Grok Build](https://github.com/xai-org/grok-build), with a **multi-provider LLM router** and optional **I/O Mesh** client hooks (publish/pull heartbeats, local memory attach).

Official open-source tooling from [IOMesh](https://iome.sh) (**IOMesh Technology Ltd.**).

### Open source / local-primary

**MIT OSS agent harness + optional mesh client surface — not the hosted multi-tenant mesh control plane.**

| This public repo is | This public repo is **not** |
|---------------------|-----------------------------|
| Local agent loop (TUI / headless / ACP), tools, subagents, skills, MCP client | Hosted multi-tenant control plane, broker fleet, portal, billing, install-store CRUD |
| Optional **mesh client** hooks against a broker you run or subscribe to | Free access to IOMesh Cloud / `aion` platform services |
| **Local-primary** memory via public [`memory`](https://github.com/iome-sh/memory) + [`iomesh-memory-mcp`](https://github.com/iome-sh/iomesh-memory-mcp) | Freemium hosted Memory Palace / platform GPU / invent **Memory GA** |
| Optional mesh **pull** into local palace; dual_write default **OFF** | Push-to-cloud-palace product path |

The TUI is a **local agent on the org pulse plane**: optional mesh hooks publish/pull organizational **heartbeats / pulses** (`dept.*` work events — not host/APM metrics · public copy = heartbeat/pulse only). Optional **Ollama** local AI ≠ platform GPU. Platform **$119** language (if any) = **Memory Ops Pack** (pull / retain / audit / support) — not cloud GPU palace. dual_write default OFF · hosted Palace sunset · no invent GA · **Beta** / pre-1.0. Not a multi-tenant remote sandbox — see [SECURITY.md](SECURITY.md).

Buyer claim pin: [memory-mcp.md](docs/architecture/memory-mcp.md#buyer-claim-pin-s774) · org-pulse edge framing: [mesh smoke / org-pulse](docs/architecture/mesh-dogfood.md#org-pulse-edge-framing-s785-pin).

> **Status:** public open-source **v0.71.x** (pre-1.0, **Beta**). Shipped in this harness: agent loop · subagents · full-screen TUI · permissions · ACP · skills · MCP client · local Memory Palace attach · multi-model catalog (DeepSeek · Grok · Gemini · Vertex · Ollama local). Optional client mesh smoke / catalog / policy / metering hooks when pointed at a broker. **Internal roadmap** items (serial residual stamps, platform install-plane, knowledge multi-tenant INSTALL_STORE, live fleet APPLY) live in private platform docs — they are **not** claims that this MIT repo is the control plane.

## Table of contents

- [Why this project](#why-this-project)
- [Supported models](#supported-models)
- [Edge install \& docs](#edge-install--docs)
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
| Multi-provider agents | Built-in **DeepSeek**, **xAI Grok**, **Gemini**, **Vertex Gemini**, **Ollama** (local pin); pin any logical name or add OpenAI-compatible endpoints |
| Sustainable defaults | Auto-cascade prefers **DeepSeek V4 Flash → Pro → Grok 4.5** for price/performance (override anytime; Ollama is pin-only) |
| Integration simplicity | Pure-Go OpenAI-compatible HTTP + SSE (`internal/router`) |
| Optional mesh client | Opt-in client hooks for heartbeats / policy / catalog when `IOMESH_ENDPOINT` (or config) points at a broker — **not** shipping the multi-tenant control plane in this repo |
| Familiar agent UX | TUI / headless / ACP, tools, subagents, workspace root, slash commands |

## Supported models

Built-in catalog (`iomesh models`). **Default cascade** uses the first three rows for automatic step-up/fallback; Google and local Ollama entries are first-class pins (`-m` / `/model` / `IOMESH_DEFAULT_MODEL`).

| Logical name | Provider | API model id | Auth env | Role |
|--------------|----------|--------------|----------|------|
| `deepseek-v4-flash` | DeepSeek | `deepseek-v4-flash` | `DEEPSEEK_API_KEY` | **Default** cascade (routine) |
| `deepseek-v4-pro` | DeepSeek | `deepseek-v4-pro` | `DEEPSEEK_API_KEY` | Cascade step-up (plan) |
| `grok-4.5` | xAI | `grok-4.5` | `XAI_API_KEY` | Cascade premium / fallback |
| `gemini-2.5-flash` | Google AI Studio | `gemini-2.5-flash` | `GEMINI_API_KEY` | Opt-in pin |
| `gemini-2.5-pro` | Google AI Studio | `gemini-2.5-pro` | `GEMINI_API_KEY` | Opt-in pin |
| `vertex-gemini-2.5-flash` | Vertex AI | `google/gemini-2.5-flash` | `GOOGLE_CLOUD_PROJECT` + ADC/`gcloud` token (or `VERTEX_API_KEY`) | Opt-in pin |
| `vertex-gemini-2.5-pro` | Vertex AI | `google/gemini-2.5-pro` | same | Opt-in pin |
| `ollama-llama3.2` | Ollama (local) | `llama3.2` | none (`OLLAMA_URL` / `OLLAMA_HOST` optional) | Local pin ($0; not cascade default) |

Any other **OpenAI-compatible** chat endpoint can be added under `[model.<name>]` in config (OpenAI, Anthropic-compatible gateways, other Ollama tags, llama.cpp/vLLM, etc.). Details: [docs/architecture/llm-cascade.md](docs/architecture/llm-cascade.md).

## Edge install & docs

Public **edge** path only — MIT harness + public memory kernel/MCP. Serial stamps in deep docs are **internal residual/roadmap labels**, not a claim that this repo hosts the multi-tenant control plane.

| Topic | Honesty | Doc |
|-------|---------|-----|
| Local AI (Ollama) | Local only · not platform GPU · not invent GA | [memory-mcp.md](docs/architecture/memory-mcp.md#local-primary-lt-honesty-s768-pin) |
| Local Docker Memory MCP | Product host [`iomesh-memory-mcp`](https://github.com/iome-sh/iomesh-memory-mcp) · dual_write OFF · not Memory GA · aion/broker **private** | [memory-mcp.md Local-edge Docker](docs/architecture/memory-mcp.md#local-edge-docker-memory-mcp-s1308--product-host-preferred--s1517) |
| Public edge install | `go install …/iomesh-memory-mcp@main` · `go get github.com/iome-sh/memory@main` · **no GOPRIVATE** · public OSS ≠ invent platform GA | [memory-mcp.md Edge OSS](docs/architecture/memory-mcp.md#edge-oss-option-a-s1453--m2-lean-attach-s1458--m3-edge-dogfood-s1463--m4-public-flip-readiness-s1469--s1478-public-product-attach) |
| Advanced Memory install | ONNX optional · Qdrant not required for lean TUI | [memory-advanced-install.md](docs/architecture/memory-advanced-install.md) |
| Usage / demo walkthrough | Signup optional · integrations list/plan + portal HITL · local memory not fully automatic | [memory-edge-usage-demo.md](docs/architecture/memory-edge-usage-demo.md) |
| Setup lifecycle | Agent-native `/setup` · residual-honest · dual_write OFF | [setup-lifecycle.md](docs/architecture/setup-lifecycle.md) |

**Internal roadmap (private platform · not this MIT surface):** multi-tenant mesh control plane, connector install-store fleet APPLY, knowledge multi-tenant INSTALL_STORE (punted for edge-first), live Cloud Run/image residual gates, portal billing fleet. Those live in private `aion` / ops residual docs — **not** shipped as open control-plane code here.

## Quick start

**Requirements:** Go version in [go.mod](go.mod) (CI uses that toolchain).

```bash
# From source
git clone https://github.com/iome-sh/iomesh-tui.git
cd iomesh-tui
make build

# Or install a released version (Go toolchain)
go install github.com/iome-sh/iomesh-tui/cmd/iomesh@v0.28.0
# Multi-platform archives: GitHub Releases (GoReleaser on v* tags) — see RELEASING.md

export DEEPSEEK_API_KEY=…          # required for default cascade
# export XAI_API_KEY=…             # optional Grok fallback
# export GEMINI_API_KEY=…          # optional Gemini AI Studio
# export GOOGLE_CLOUD_PROJECT=…    # required for Vertex models
# # Vertex auth: auto gcloud print-access-token (cache ~50m) or VERTEX_API_KEY override
# # Local Ollama (pin-only; no API key): ollama serve && ollama pull llama3.2
# # export OLLAMA_URL=http://127.0.0.1:11434/v1   # optional override

./bin/iomesh models
./bin/iomesh -p "List the top-level packages in this repo"
# optional pins:
# ./bin/iomesh -m gemini-2.5-flash -p "Reply with ok"
# ./bin/iomesh -m vertex-gemini-2.5-flash -p "Reply with ok"
# ./bin/iomesh -m ollama-llama3.2 -p "Reply with ok"
./bin/iomesh                       # full-screen TUI (TTY)
./bin/iomesh --repl                # classic line REPL
./bin/iomesh -c                    # continue latest session
./bin/iomesh sessions | skills | mcp
./bin/iomesh agent serve           # ACP WebSocket (127.0.0.1:7400/acp)
./bin/iomesh mesh smoke            # mesh smoke (needs IOMESH_ENDPOINT)
./bin/iomesh mesh usage --json     # local process meter (JSON)
make smoke-unit                    # offline mesh tests (alias: dogfood-unit)
```

Optional: copy [`.env.example`](.env.example) for local env vars (iomesh reads the **process environment**; it does not auto-load `.env` files yet). Copy [`configs/config.example.toml`](configs/config.example.toml) to `~/.iomesh/config.toml` to customize.

### Default cascade (auto step-up / fallback)

When you do **not** pin `-m` / `/model`, the router escalates by task complexity:

```text
deepseek-v4-flash  →  deepseek-v4-pro  →  grok-4.5
     (routine)            (plan)          (high-stakes / fallback)
```

Pin Google or local Ollama (or any catalog entry) explicitly, e.g. `-m gemini-2.5-flash`, `-m ollama-llama3.2`, or `export IOMESH_DEFAULT_MODEL=vertex-gemini-2.5-flash`.

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
iomesh mesh smoke           I/O Mesh smoke (needs IOMESH_ENDPOINT; legacy: dogfood|probe)
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

Index: **[docs/README.md](docs/README.md)** (architecture, MCP, ACP, TUI, mesh smoke, …).

Open-source process: [CONTRIBUTING](CONTRIBUTING.md) · [SUPPORT](SUPPORT.md) · [RELEASING](RELEASING.md) · [CHANGELOG](CHANGELOG.md) · [docs/OPEN_SOURCE_AUDIT.md](docs/OPEN_SOURCE_AUDIT.md).

**Boundary reminder:** deep docs may mention optional mesh client behavior or residual serial stamps for maintainers. That is **edge/client documentation**, not a promise that this repository is the hosted multi-tenant mesh control plane.

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
  iomesh/             I/O Mesh client + smoke suite
  tui/                full-screen Bubble Tea + classic REPL
  skills/             SKILL.md catalog
  mcp/                MCP stdio/HTTP, resources, prompts, OAuth
  acp/                Agent Client Protocol (stdio + WebSocket)
  session/            transcript persistence
configs/              example config.toml
scripts/              mesh smoke helper
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

[MIT](LICENSE) © 2026 [IOMesh Technology Ltd.](https://iome.sh) — see [NOTICE](NOTICE).

**Maintained by** [IOMesh Technology Ltd.](https://iome.sh) · Product: [iome.sh](https://iome.sh) · Support: [SUPPORT.md](SUPPORT.md)

Grok Build is a separate project by xAI (Apache-2.0). This repository is an **independent Go implementation** inspired by its product surface, not a fork of the Rust sources.
