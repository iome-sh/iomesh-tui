# LLM models, cascade & price-performance

Research baseline: **July 2026** public pricing and coding-agent benchmarks.

`iomesh-tui` is **multi-provider**: the built-in catalog includes DeepSeek, xAI Grok, Google Gemini (AI Studio), Vertex AI Gemini, and local **Ollama** (OpenAI-compatible `/v1`). All speak OpenAI-compatible chat + tools over pure-Go HTTP/SSE. The **default auto-cascade** still prefers DeepSeek for cost; any catalog model can be the session default (Ollama is pin-only).

List live catalog: `iomesh models`.

## Built-in catalog

| Logical name (`-m`) | Provider | API model id | Env | Cascade role |
|---------------------|----------|--------------|-----|--------------|
| `deepseek-v4-flash` | DeepSeek | `deepseek-v4-flash` | `DEEPSEEK_API_KEY` | **Default** (routine) |
| `deepseek-v4-pro` | DeepSeek | `deepseek-v4-pro` | `DEEPSEEK_API_KEY` | Step-up (plan) |
| `grok-4.5` | xAI | `grok-4.5` | `XAI_API_KEY` | Premium / high-stakes fallback |
| `gemini-2.5-flash` | Google AI Studio | `gemini-2.5-flash` | `GEMINI_API_KEY` | Pin / optional default |
| `gemini-2.5-pro` | Google AI Studio | `gemini-2.5-pro` | `GEMINI_API_KEY` | Pin / optional default |
| `vertex-gemini-2.5-flash` | Vertex AI | `google/gemini-2.5-flash` | `VERTEX_API_KEY` + `GOOGLE_CLOUD_PROJECT` | Pin / optional default |
| `vertex-gemini-2.5-pro` | Vertex AI | `google/gemini-2.5-pro` | `VERTEX_API_KEY` + `GOOGLE_CLOUD_PROJECT` | Pin / optional default |
| `ollama-llama3.2` | Ollama (local) | `llama3.2` | (none; optional `OLLAMA_URL` / `OLLAMA_HOST`) | **Pin-only** (not cascade default) |

### Pricing snapshot (indicative)

| Role | Model | Input / 1M | Output / 1M | Cache hit / 1M | Context |
|------|--------|------------|-------------|----------------|---------|
| Default | DeepSeek V4 Flash | $0.14 | $0.28 | $0.0028 | 1M |
| Step-up | DeepSeek V4 Pro | $0.435 | $0.87 | ~$0.0036 | 1M |
| Premium | Grok 4.5 | ~$2.00 | ~$6.00 | — | 500K |
| Google | Gemini 2.5 Flash | ~$0.30 | ~$2.50 | — | 1M |
| Google | Gemini 2.5 Pro | ~$1.25 | ~$10 | — | 1M |
| Vertex | Gemini Flash / Pro | (GCP list) | (GCP list) | — | 1M |
| Local | Ollama llama3.2 | $0 | $0 | — | ~128K (model-dependent) |

> Legacy DeepSeek ids `deepseek-chat` / `deepseek-reasoner` are deprecated (July 2026). Always use `deepseek-v4-flash` / `deepseek-v4-pro`.

**Why Flash default (when unpinned):** near-frontier coding / agentic quality at strong Score/$; OpenAI-compatible; tool calling; 1M context is economical with cache hits.

**When to escalate (auto):** multi-file refactors and plan mode → Pro; production migrations / security reviews / Flash+Pro failures → Grok.

**Pin Google (or any model):** `-m gemini-2.5-flash`, `-m vertex-gemini-2.5-flash`, `/model …`, or `IOMESH_DEFAULT_MODEL=…`. Use Vertex when burning Google Cloud credits or dogfooding GCP OpenAI-compat.

**Pin local Ollama:** `-m ollama-llama3.2`, `/model ollama-llama3.2`, or `IOMESH_DEFAULT_MODEL=ollama-llama3.2`. Ollama is **not** part of the DeepSeek auto-cascade.

**Custom OpenAI-compatible providers** (OpenAI, Anthropic gateways, other local inference): add `[model.<name>]` with `base_url`, `model`, optional `env_key` in config — same router path as built-ins.

### Gemini API (AI Studio)

```bash
export GEMINI_API_KEY=…   # https://aistudio.google.com/apikey
iomesh -m gemini-2.5-flash -p "Reply with ok"
# or set default:
# export IOMESH_DEFAULT_MODEL=gemini-2.5-flash
```

- Base URL: `https://generativelanguage.googleapis.com/v1beta/openai`
- Auth: `Authorization: Bearer $GEMINI_API_KEY`
- Docs: [OpenAI compatibility](https://ai.google.dev/gemini-api/docs/openai)

### Vertex AI (OpenAI-compatible)

```bash
export GOOGLE_CLOUD_PROJECT=iomesh-stage-001   # required — expanded into base_url
# Auth (pick one):
# A) Auto (default): gcloud ADC helper — no hourly re-export
#    requires: gcloud auth login (or application-default) + aiplatform API enabled
# B) Explicit short-lived token (~1h):
#    export VERTEX_API_KEY=$(gcloud auth print-access-token)
# C) Disable gcloud helper: VERTEX_ADC=0 and set VERTEX_API_KEY yourself
iomesh -m vertex-gemini-2.5-flash -p "Reply with ok"
```

- Base URL template: `https://us-central1-aiplatform.googleapis.com/v1/projects/${GOOGLE_CLOUD_PROJECT}/locations/us-central1/endpoints/openapi`
- Auth: OAuth access token as Bearer (not a Gemini API key). **** in-process cache (~50m) + auto `gcloud auth print-access-token` when env unset; **401 → invalidate + one retry**
- Env: `VERTEX_API_KEY` / `GOOGLE_OAUTH_ACCESS_TOKEN` override; `VERTEX_ADC=0` disables gcloud helper
- Model ids often use publisher prefix: `google/gemini-2.5-flash`

If model ids change, override in `~/.iomesh/config.toml`:

```toml
[model.vertex-gemini-2.5-flash]
model = "google/gemini-2.5-flash"
base_url = "https://us-east4-aiplatform.googleapis.com/v1/projects/${GOOGLE_CLOUD_PROJECT}/locations/us-east4/endpoints/openapi"
env_key = "VERTEX_API_KEY"
```

### Ollama (local OpenAI-compatible `/v1`)

**Beta · local only · not platform GPU · not invent GA.** Cost-max path: I/O Mesh cloud (pull egress) ↔ iomesh-tui ↔ local memory MCP ↔ local AI (Ollama). Cloud mesh ≠ local AI. Dual-write remains optional audit (**default OFF**). Hosted Palace is sunset until scale — primary memory is local Palace.

**s765 (Beta · completeness pin):** local-edge stack complete after s761 Ollama product — docs + unit tests lock inventory; does not invent catalog models or re-claim s761; catalog pin ≠ cascade default · offline unit ≠ live Ollama green. See [memory-mcp.md](./memory-mcp.md#local-edge-stack-s761-product--s765-completeness-pin).

```bash
# Install + serve (https://ollama.com)
ollama serve
ollama pull llama3.2

# Pin built-in catalog entry (no API key required)
iomesh -m ollama-llama3.2 -p "Reply with ok"
# or:
# export IOMESH_DEFAULT_MODEL=ollama-llama3.2
# /model ollama-llama3.2   (in TUI/REPL)
```

| Knob | Effect |
|------|--------|
| Default base | `http://127.0.0.1:11434/v1` |
| `OLLAMA_URL` | Full base override (host root → appends `/v1`, e.g. `http://127.0.0.1:11434`) |
| `OLLAMA_HOST` | Host override (scheme optional; `192.168.1.10:11434` → `http://…/v1`) |
| Auth | None required; empty `Authorization` is fine |

Native Ollama `/api/generate` is **not** used — OpenAI-compatible `/v1/chat/completions` only.

**Custom Ollama models** (other tags / remote Ollama):

```toml
[model.ollama-codellama]
model = "codellama"
base_url = "http://127.0.0.1:11434/v1"
# env_key optional — leave unset for local Ollama
context_window = 128000
cost_tier = 0
input_cost_per_m = 0
output_cost_per_m = 0
capabilities = ["local", "ollama", "fast", "coding", "tool-calling"]
priority = 65
```

## Router behavior in this repo

```go
params := router.SelectParams{
    TaskType:        router.TaskPlan,
    EstimatedTokens: est,
    Complexity:      router.ComplexityPlan, // 0 routine · 1 plan · 2 high-stakes
}
resp, meta, err := r.ExecuteStreamWithFallback(ctx, req, params, onDelta)
```

| Complexity | Preferred capability | Typical pick |
|------------|----------------------|--------------|
| Routine | `fast` | deepseek-v4-flash |
| Plan | `coding` (step-up by cost_tier) | deepseek-v4-pro |
| High-stakes | `premium` | grok-4.5 |

Session override (`-m` / `/model`) always wins until cleared.

## Cost estimator

```text
USD ≈ (input - cached) × input_cost_per_m / 1e6
    + cached × cache_hit_cost_per_m / 1e6
    + output × output_cost_per_m / 1e6
```

Example: Flash, 1M input with 90% cache hit + 10k output ≈ **$0.019**.

## Adding a model

```toml
[model.claude-opus]
model = "claude-opus-4-8"
base_url = "https://api.anthropic.com/v1"
env_key = "ANTHROPIC_API_KEY"
context_window = 1000000
cost_tier = 50
capabilities = ["coding", "premium", "tool-calling"]
priority = 40
extra_headers = { "anthropic-version" = "2023-06-01" }
```

Note: Anthropic Messages API may need an adapter (`api_backend = "messages"`) in a later iteration; OpenAI-compatible gateways work today.

## Verification

Re-check live rates on provider dashboards before production budgeting. Peak-hour DeepSeek pricing may apply; cache discounts dominate high-volume agent workloads.
