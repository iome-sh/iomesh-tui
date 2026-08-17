# iomesh-tui

[![CI](https://github.com/iome-sh/iomesh-tui/actions/workflows/ci.yml/badge.svg)](https://github.com/iome-sh/iomesh-tui/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/github/go-mod/go-version/iome-sh/iomesh-tui)](go.mod)

**I/O Mesh TUI** — a Go coding-agent harness inspired by [xAI Grok Build](https://github.com/xai-org/grok-build), with a **multi-provider LLM router**, tools/subagents, and **local memory** (public MCP host + kernel). Optional **I/O Mesh** client hooks (heartbeats / catalog / pull) when you point at a broker.

Official open-source tooling from [IOMesh](https://iome.sh) (**IOMesh Technology Ltd.**).

### Open source / local-first

**MIT coding-agent harness** with multi-provider LLMs, tools/subagents, and **first-class local memory** (public MCP host + kernel). Optional mesh client hooks if you point at a broker — this repo is **not** a hosted multi-tenant cloud control plane.

| This public repo **is** | This public repo is **not** |
|-------------------------|-----------------------------|
| Local agent loop (TUI / headless / ACP), tools, subagents, skills, permissions | Hosted multi-tenant mesh control plane or cloud admin UI |
| Multi-provider LLM router (API keys + optional **Ollama** local models) | A cloud GPU / managed model product |
| **Local memory** setup, attach, and use via public [`iomesh-memory-mcp`](https://github.com/iome-sh/iomesh-memory-mcp) + [`memory`](https://github.com/iome-sh/memory) | Hosted multi-tenant memory-as-a-service |
| Optional **mesh client** (heartbeats / catalog / policy / pull) when `IOMESH_ENDPOINT` is set | Free access to IOMesh Cloud or private platform backends |

**Local path (default mental model):** install `iomesh` → set an LLM key (or pin Ollama) → `/setup` for managed config + memory host preflight → attach local MCP memory → agent turns with recall/ingest. Mesh and portal are **optional**.

- Local memory docs: [memory-mcp.md](docs/architecture/memory-mcp.md) · [setup-lifecycle.md](docs/architecture/setup-lifecycle.md) · [edge-user-journey.md](docs/architecture/edge-user-journey.md)
- Security model (local sandbox defaults): [SECURITY.md](SECURITY.md)
- Packaging boundary (MIT vs private platform): [oss-packaging-boundary.md](docs/architecture/oss-packaging-boundary.md)

> **Status:** public open-source **v0.77.x** (pre-1.0, **Beta**). Shipped: agent loop · subagents · full-screen TUI · **heartbeat dashboard** · permissions · ACP · skills · MCP client · **local memory attach** · multi-model catalog (DeepSeek · Grok · Gemini · Vertex · Ollama). Optional mesh client when pointed at a broker you run or subscribe to.

[![iomesh-tui /dashboard — landing MeshConsole eval template](docs/assets/dashboard-eval.svg)](docs/architecture/tui.md#dashboard-heartbeat-live-feed)

`/dashboard` is empty until consume (probe broker `/v1` if a mesh client is attached). `/dashboard preview` is the [iome.sh](https://iome.sh) landing MeshConsole **eval template** — not your workspace, not [console.iome.sh](https://console.iome.sh). See [Dashboard](#dashboard-heartbeat-live-feed).

## Table of contents

- [Why this project](#why-this-project)
- [Supported models](#supported-models)
- [Edge install \& docs](#edge-install--docs)
- [Quick start](#quick-start)
- [Dashboard (heartbeat live feed)](#dashboard-heartbeat-live-feed)
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
| Local memory | First-class attach to public `iomesh-memory-mcp` + kernel · `/setup` + `/memory` · dual_write default **OFF** |
| Optional mesh client | Opt-in heartbeats / catalog / policy / pull when `IOMESH_ENDPOINT` points at a broker you control or subscribe to |
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

Local-first path: this MIT harness + public memory kernel/MCP. Optional mesh/portal clients are extra, not required for local agent + local memory.

| Topic | Summary | Doc |
|-------|---------|-----|
| Local memory (MCP host + kernel) | Install/attach `iomesh-memory-mcp` · dual_write default **OFF** | [memory-mcp.md](docs/architecture/memory-mcp.md) |
| Setup lifecycle | Agent-native `/setup` init · preflight · reload · opt-in pull/analyze | [setup-lifecycle.md](docs/architecture/setup-lifecycle.md) |
| User journey | Signup (optional) → TUI → keys → setup → connectors (optional) → local store → analyze | [edge-user-journey.md](docs/architecture/edge-user-journey.md) |
| Usage / demo walkthrough | Operator example end-to-end | [memory-edge-usage-demo.md](docs/architecture/memory-edge-usage-demo.md) |
| Local AI (Ollama) | Local models only · pin via `-m ollama-…` | [llm-cascade.md](docs/architecture/llm-cascade.md) |
| Advanced memory install | Optional ONNX / extra host knobs · lean path does not require Qdrant | [memory-advanced-install.md](docs/architecture/memory-advanced-install.md) |
| Packaging boundary | MIT harness vs private platform surfaces | [oss-packaging-boundary.md](docs/architecture/oss-packaging-boundary.md) |

Optional mesh client docs (broker you run or subscribe to): [mesh smoke](docs/architecture/mesh-dogfood.md).

## Quick start

**Requirements:** Go version in [go.mod](go.mod) (CI uses that toolchain).

```bash
# From source
git clone https://github.com/iome-sh/iomesh-tui.git
cd iomesh-tui
make build

# Or install a released version (Go toolchain)
go install github.com/iome-sh/iomesh-tui/cmd/iomesh@v0.77.0
# Pin matches latest known tag at docs write; GitHub Releases may be newer — see RELEASING.md
# Multi-platform archives: GitHub Releases (GoReleaser on v* tags)
# Or tip of main: go install github.com/iome-sh/iomesh-tui/cmd/iomesh@latest  (pre-1.0 Beta)

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

### First-run (agent)

Local agent + local memory path (no invent Connected / Memory GA · dual_write default **OFF**). **Mesh / Memory Ops Pack not required for first-run** — OSS local-primary only; Ops Pack is an optional later overlay (~$119 pull/retain/support · free eng **s1695**).

1. Set an LLM key (`DEEPSEEK_API_KEY` / `XAI_API_KEY` / …) **or** pin Ollama (`-m ollama-llama3.2`).
2. Run the TUI: `./bin/iomesh` (or `iomesh` if installed).
3. In-session setup: `/setup init` `local-memory` · `/setup preflight` · start `iomesh-memory-mcp` if needed · `/setup reload` (hot-swaps MCP **and** re-scans skills · package wire ≠ Connected). Cold CLI path: `iomesh setup init` → restart `iomesh` · `iomesh setup preflight` (CLI has **no** `setup reload` · free eng s1686). After preflight, same dual path is printed on the report (in-session `/setup reload` vs cold restart · free eng **s1699**).
4. Offline maps when you want a residual-honest board (no MCP dial): `/onboard next journey` · `/onboard next setup` · `/onboard next wizard` · `/onboard next marketing-demo` · `/onboard next memory` (local-primary · Ops Pack not first-run required). Optional peek at the landing heartbeat: `/dashboard preview` (eval template · `/dashboard` stays empty until consume · see [below](#dashboard-heartbeat-live-feed)).

Optional: copy [`.env.example`](.env.example) for local env vars (iomesh reads the **process environment**; it does not auto-load `.env` files yet). Copy [`configs/config.example.toml`](configs/config.example.toml) to `~/.iomesh/config.toml` to customize.

### Default cascade (auto step-up / fallback)

When you do **not** pin `-m` / `/model`, the router escalates by task complexity:

```text
deepseek-v4-flash  →  deepseek-v4-pro  →  grok-4.5
     (routine)            (plan)          (high-stakes / fallback)
```

Pin Google or local Ollama (or any catalog entry) explicitly, e.g. `-m gemini-2.5-flash`, `-m ollama-llama3.2`, or `export IOMESH_DEFAULT_MODEL=vertex-gemini-2.5-flash`.

## Dashboard (heartbeat live feed)

The marketing site widget and this slash are the **same analysis**, different chrome.

| Surface | What it is | What you see |
|---------|------------|--------------|
| [iome.sh](https://iome.sh) MeshConsole | Landing-page demo widget | Eval template (HOME_PROOF seed) |
| **TUI `/dashboard`** | Local harness overlay / REPL snapshot | **Empty** (probe broker `/v1` if mesh attached) |
| **TUI `/dashboard preview`** | Opt-in eval template | iome.sh MeshConsole seed — **not** your org |
| [console.iome.sh](https://console.iome.sh) | Paid Base workspace UI | Your workspace when billed — **not** this slash |

No GIF of a live tenant. Default `/dashboard` stays **empty** until consume (`ListStreams` + `ListStreamMessages` on the broker — **not** cookie-only `GET /v52`). `/dashboard preview` ticks the public seed every 2.6s (same cadence as the site). A recorded tenant GIF would invent Connected.

```text
./bin/iomesh                 # TTY → full-screen
# then:
/dashboard                   # toggle overlay (empty until consume)
/dashboard preview           # opt-in eval template (not your org)
/dashboard focus eng.ops     # tenancy
/heartbeat help              # aliases: /heartbeat /mesh-console
```

Fullscreen keys: **esc** / **q** close · **tab** cycle tenancy · **1–4** jump `sre.incidents` / `eng.ops` / `cs.tickets` / `gtm.pipeline`.

Example (default **EMPTY** — no consume yet; badge **EMPTY** = no mesh client):

```text
● context://mesh · no live heartbeat · consume missing · sre.incidents     EMPTY
▁▁▂█▃▁▁▁▂█▃▁▁▁   analysis  ops 0 · knowledge 0 · analytics 0 · Beta
knowledge Beta empty · analytics Beta empty · not GA

Heartbeat
no consumed messages · mock eval rows hidden
add [iomesh] endpoint="https://hooks.iome.sh" or infer from portal MCP
portal MCP (apiv1.iome.sh/v7/mcp) is catalog — streams are hooks.iome.sh
infer ≠ Connected · do not invent consume
/dashboard preview · eval template on iome.sh (not your org)
```

Example (`/dashboard preview` — eval template, not your org; badge **EVAL**):

```text
● context://mesh · eval template preview · not your org · sre.incidents     EVAL
▁▁▂█▃▁▁▁▂█▃▁▁▁   analysis  ops 3 · knowledge 1 · analytics 1 · Beta

Tenancy            Heartbeat                         Agent tools
▸ sre.incidents    14:02:11  ops        sre.incidents   mesh.ops.pull      ALLOW
  eng.ops          14:02:18  ops        eng.ops         mesh.knowledge.search ALLOW
  cs.tickets       14:02:24  ops        cs.tickets      mesh.gtm.forecast  DENY
  gtm.pipeline     14:02:31  knowledge  sre.incidents
Pulse 18 / min     14:02:39  analytics  gtm.pipeline
```

Honesty: default empty until consume · `/dashboard preview` is eval template not your org · `catalog ≠ Connected` · `dual_write OFF` · knowledge/analytics **Beta** · not Memory GA · not live APPLY · **CLIENT** badge only means a mesh client is configured — listed streams ≠ live pulse · **PULSE** only after ≥1 decoded broker message. Full notes: [tui.md](docs/architecture/tui.md#dashboard-heartbeat-live-feed) · asset: [docs/assets/dashboard-eval.svg](docs/assets/dashboard-eval.svg).

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

Slash commands (TUI/REPL): `/model`, `/theme`, `/dashboard` (heartbeat live feed · aliases `/heartbeat` `/mesh-console`), `/permissions`, `/subagents`, `/setup`, `/onboard`, `/memory`, `/integrations`, `/save`, `/sessions`, `/load`, `/cost`, `/help`, `/quit`.  
Keys (fullscreen): **Enter** send · **Ctrl+J** newline · **y/n/a** tool approval · `/dashboard` overlay (esc close).

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
