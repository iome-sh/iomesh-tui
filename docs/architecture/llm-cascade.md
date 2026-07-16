# LLM models, cascade & price-performance

Research baseline: **July 2026** public pricing and coding-agent benchmarks.

`iomesh-tui` is **multi-provider**: the built-in catalog includes DeepSeek, xAI Grok, Google Gemini (AI Studio), and Vertex AI Gemini. All speak OpenAI-compatible chat + tools over pure-Go HTTP/SSE. The **default auto-cascade** still prefers DeepSeek for cost; any catalog model can be the session default.

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

### Pricing snapshot (indicative)

| Role | Model | Input / 1M | Output / 1M | Cache hit / 1M | Context |
|------|--------|------------|-------------|----------------|---------|
| Default | DeepSeek V4 Flash | $0.14 | $0.28 | $0.0028 | 1M |
| Step-up | DeepSeek V4 Pro | $0.435 | $0.87 | ~$0.0036 | 1M |
| Premium | Grok 4.5 | ~$2.00 | ~$6.00 | — | 500K |
| Google | Gemini 2.5 Flash | ~$0.30 | ~$2.50 | — | 1M |
| Google | Gemini 2.5 Pro | ~$1.25 | ~$10 | — | 1M |
| Vertex | Gemini Flash / Pro | (GCP list) | (GCP list) | — | 1M |

> Legacy DeepSeek ids `deepseek-chat` / `deepseek-reasoner` are deprecated (July 2026). Always use `deepseek-v4-flash` / `deepseek-v4-pro`.

**Why Flash default (when unpinned):** near-frontier coding / agentic quality at strong Score/$; OpenAI-compatible; tool calling; 1M context is economical with cache hits.

**When to escalate (auto):** multi-file refactors and plan mode → Pro; production migrations / security reviews / Flash+Pro failures → Grok.

**Pin Google (or any model):** `-m gemini-2.5-flash`, `-m vertex-gemini-2.5-flash`, `/model …`, or `IOMESH_DEFAULT_MODEL=…`. Use Vertex when burning Google Cloud credits or dogfooding GCP OpenAI-compat.

**Custom OpenAI-compatible providers** (OpenAI, Anthropic gateways, local inference): add `[model.<name>]` with `base_url`, `model`, `env_key` in config — same router path as built-ins.

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
export VERTEX_LOCATION=us-central1             # built-in URL uses us-central1; override base_url for other regions
export VERTEX_API_KEY=$(gcloud auth print-access-token)  # ~1h; alias: GOOGLE_OAUTH_ACCESS_TOKEN
iomesh -m vertex-gemini-2.5-flash -p "Reply with ok"
```

- Base URL template: `https://us-central1-aiplatform.googleapis.com/v1/projects/${GOOGLE_CLOUD_PROJECT}/locations/us-central1/endpoints/openapi`
- Auth: short-lived OAuth access token as Bearer (not a Gemini API key)
- Model ids often use publisher prefix: `google/gemini-2.5-flash`

If model ids change, override in `~/.iomesh/config.toml`:

```toml
[model.vertex-gemini-2.5-flash]
model = "google/gemini-2.5-flash"
base_url = "https://us-east4-aiplatform.googleapis.com/v1/projects/${GOOGLE_CLOUD_PROJECT}/locations/us-east4/endpoints/openapi"
env_key = "VERTEX_API_KEY"
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
