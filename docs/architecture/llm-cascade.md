# LLM cascade & price-performance notes

Research baseline: **July 2026** public pricing and coding-agent benchmarks.

## Primary recommendation

| Role | Model | API model id | Input / 1M | Output / 1M | Cache hit / 1M | Context |
|------|--------|--------------|------------|-------------|----------------|---------|
| **Default** | DeepSeek V4 Flash | `deepseek-v4-flash` | $0.14 | $0.28 | $0.0028 | 1M |
| **Step-up** | DeepSeek V4 Pro | `deepseek-v4-pro` | $0.435 | $0.87 | ~$0.0036 | 1M |
| **Premium** | Grok 4.5 | `grok-4.5` | ~$2.00 | ~$6.00 | — | 500K |

> Legacy DeepSeek ids `deepseek-chat` / `deepseek-reasoner` are deprecated (July 2026). Always use `deepseek-v4-flash` / `deepseek-v4-pro`.

**Why Flash default:** near-frontier coding / agentic quality at the best Score/$ on cost-adjusted leaderboards; OpenAI-compatible; strong tool calling; 1M context makes repo-scale prompts economical especially with cache hits.

**When to escalate:** multi-file refactors and plan mode → Pro; production migrations / security reviews / Flash+Pro failures → Grok (or Claude if configured).

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
