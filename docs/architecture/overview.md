# I/O Mesh TUI — Architecture Overview

Go rewrite of [xAI Grok Build](https://github.com/xai-org/grok-build) with a **multi-provider LLM router** (DeepSeek · Grok · Gemini · Vertex; OpenAI-compatible custom endpoints) and optional **I/O Mesh** context / policy / metering hooks. Default auto-cascade remains DeepSeek Flash → Pro → Grok for cost; pin any built-in model with `-m` / `/model`.

## Goals

| Goal | Approach |
|------|----------|
| Cost-efficient coding agent | Default **DeepSeek V4 Flash**; step-up **V4 Pro**; premium **Grok 4.5** |
| OpenAI-compatible, pure Go | `net/http` + JSON SSE; no vendor SDKs in the hot path |
| Grok Build parity (incremental) | TUI / headless / ACP modes; tools; workspace; subagents (later) |
| I/O Mesh integration | Context plane injection + `dept.*` usage / **org event streams (heartbeats / pulses)** (fail-open) · TUI as local agent on the org pulse plane |

## Package map (Grok Build → Go)

| Grok Build (Rust) | Go package | Status |
|-------------------|------------|--------|
| `xai-grok-pager-bin` | `cmd/iomesh` | Scaffold |
| `xai-grok-pager` | `internal/tui` | Full-screen Bubble Tea + classic REPL fallback |
| `xai-grok-shell` | `internal/agent` | Turn loop + tools |
| `xai-grok-tools` | `internal/agent` tools | read/list/grep/shell/write |
| `xai-grok-workspace` | `internal/workspace` | Rooted FS + path jail |
| config / custom models | `internal/config` + `internal/router` | TOML + cascade |
| — | `internal/iomesh` | Lean platform client (no SDK dep; full client surface → [iomesh-client-sdk-go](https://github.com/iome-sh/iomesh-client-sdk-go) or [iomesh-client-sdk-python](https://github.com/iome-sh/iomesh-client-sdk-python) peers) |
| subagents | `internal/subagent` + agent tools | explore/plan/gp + parallel + worktree |
| MCP / skills | `internal/mcp` + `internal/skills` | stdio MCP + SKILL.md loader |

## LLM fallback router

```
SelectModel(task, tokens, complexity)
        │
        ▼
┌───────────────────┐
│ deepseek-v4-flash │  routine / subagent / cheap
└─────────┬─────────┘
          │ plan / multi-file
          ▼
┌───────────────────┐
│ deepseek-v4-pro   │  step-up coding
└─────────┬─────────┘
          │ high-stakes / failure
          ▼
┌───────────────────┐
│ grok-4.5          │  premium fallback
└───────────────────┘
```

- **Selection heuristics**: capability tags (`fast`, `coding`, `premium`), `cost_tier`, context window fit, session `/model` override.
- **Fallback**: on rate-limit, 5xx, or network errors, walk the priority-sorted chain (max attempts configurable).
- **Observability**: per-call duration, tokens, estimated USD (including cache-hit rates), optional `dept.agent.llm_call` emit.

See `internal/router/` for implementation and tests.

## Runtime flow

```
CLI / TUI / ACP
      │
      ▼
 agent.Runtime.RunTurn
      │  (optional iomesh.ContextSnippet)
      ▼
 router.ExecuteStreamWithFallback
      │
      ▼
 tool loop (read_file, grep, run_shell, …)
      │
      ▼
 workspace effects + events → UI
```

## Configuration precedence

1. CLI flags (`-m`, `-p`, `--yolo`, `-C`)
2. Environment (`IOMESH_*`, `DEEPSEEK_API_KEY`, `XAI_API_KEY`)
3. `~/.iomesh/config.toml` (or `$IOMESH_CONFIG`)
4. Built-in defaults (`router.DefaultModels`)

## Next milestones

1. ~~Full-screen Bubble Tea TUI~~ **done** — see [tui.md](tui.md) (scrollback, streaming, approval overlay)
2. ~~Interactive permissions + model picker~~ **done** — see [permissions.md](permissions.md)
3. ~~ACP `agent stdio`~~ **done** — see [acp.md](acp.md)
4. ~~Subagent orchestration~~ **done** — see [subagents.md](subagents.md)
5. ~~Git worktree isolation~~ **done** — `isolation=worktree`
6. ~~Worktree apply/merge helper~~ **done** — `apply_worktree` / `diff_worktree`
7. ~~Session persistence + compaction~~ **done** — `.iomesh/sessions`, resume subagent catalog
8. ~~MCP client + skills loader~~ **done** — see [mcp.md](mcp.md), [skills.md](skills.md)
9. ~~ACP WebSocket serve~~ **done** — `iomesh agent serve` (see [acp.md](acp.md))
10. ~~MCP HTTP/SSE (streamable HTTP)~~ **done** — see [mcp.md](mcp.md)
11. ~~Stage mesh dogfood~~ **done** — see [mesh-dogfood.md](mesh-dogfood.md)
12. ~~TUI polish (multi-line edit, themes)~~ **done** — see [tui.md](tui.md)
13. ~~MCP resources/prompts + OAuth helpers~~ **done** — see [mcp.md](mcp.md)
14. ~~Deeper I/O Mesh: lineage-aware context, Rego policy gates, local metering~~ **done** — see [mesh-deeper.md](mesh-deeper.md)
15. ~~Mesh catalog composition + TUI cost/mesh slash cmds~~ **done** — see [mesh-deeper.md](mesh-deeper.md)
16. ~~Portal catalog federation + dogfood JSON~~ **done** — see [mesh-deeper.md](mesh-deeper.md)
17. ~~Memory Palace MCP Phase 0–1~~ **done** — local-primary MCP palace path (not hosted cloud GPU); see [memory-mcp.md](memory-mcp.md) (stdio attach, auto-recall, `/memory`, opt-in ingest · [naming honesty s771](memory-mcp.md#naming-honesty-s771-pin) · [buyer claim pin s774](memory-mcp.md#buyer-claim-pin-s774) — MIT TUI ≠ hosted mesh CP · [org-pulse edge framing s785](mesh-dogfood.md#org-pulse-edge-framing-s785-pin))
18. ~~Memory Phase 2~~ **done** — HTTP MCP path + optional dual-write `MEMORY_INGEST` (default OFF; v0.3.0); see [memory-mcp.md](memory-mcp.md)
19. ~~Dogfood async MEMORY_RPC recall (session correlation)~~ **done** — see [mesh-dogfood.md](mesh-dogfood.md)
20. ~~Sync HTTP memory retrieve + dogfood `memory_retrieve`~~ **done** — see [mesh-dogfood.md](mesh-dogfood.md) / [memory-mcp.md](memory-mcp.md)
21. ~~Agent auto-recall prefer sync HTTP~~ **done** — mesh `RetrieveMemory` first, MCP fallback; see [memory-mcp.md](memory-mcp.md)
22. ~~Stage warm memory plane / sidecar dogfood~~ **done** — `[memory].endpoint` / `IOMESH_MEMORY_ENDPOINT`; see [mesh-dogfood.md](mesh-dogfood.md)
23. ~~Release v0.4.0~~ **done** — Memory Phase 3+ + dogfood evidence packaged (tag on merge)
24. ~~GoReleaser multi-platform binaries + usage JSON~~ **done** — see [RELEASING.md](../../RELEASING.md) / [mesh-deeper.md](mesh-deeper.md)
25. ~~Release v0.5.0~~ **done** — GoReleaser packaging + usage JSON
26. ~~Remote metering emit path (org/workspace + llm_meter dogfood)~~ **done** — see [mesh-deeper.md](mesh-deeper.md) / [mesh-dogfood.md](mesh-dogfood.md)
27. ~~Release v0.6.0~~ **done** — multi-tenant metering emit packaging
28. ~~Dept emit publish wire parity (SDK + TUI)~~ **done** — `/v1/streams/dept/publish` + SDK `EmitLLMCall`
29. ~~GoReleaser SPDX SBOM on release assets~~ **done** — see [RELEASING.md](../../RELEASING.md)
30. ~~Keyless cosign on release checksums~~ **done** — see [RELEASING.md](../../RELEASING.md)
31. ~~Mesh User-Agent + local release-snapshot --skip=sign~~ **done** — SDK Health/Ready parity sibling
32. ~~Dogfood / StatusLine User-Agent evidence~~ **done** — report `user_agent` + StatusLine `ua=` for CI/ops; see [mesh-dogfood.md](mesh-dogfood.md)
33. ~~WaitReady + `iomesh mesh wait` preflight~~ **done** — poll Ready (optional Health) until OK or deadline; see [mesh-dogfood.md](mesh-dogfood.md)
34. ~~Dogfood catalog plane evidence~~ **done** — report `catalog_source` / `catalog_count` for CI; see [mesh-dogfood.md](mesh-dogfood.md)
35. ~~Dogfood context plane evidence + `mesh status`~~ **done** — report `context_chars` / `context_lineage_count` + operator status CLI; see [mesh-dogfood.md](mesh-dogfood.md)
36. ~~Dogfood WaitReady soft preflight~~ **done** — optional `--wait-ready` inside mesh dogfood + report `wait_ready_ms`; see [mesh-dogfood.md](mesh-dogfood.md)
37. ~~Lean mesh stream discovery + `mesh streams` CLI~~ **done** — `ListStreams` / `GetStream` + `iomesh mesh streams`; see [mesh-deeper.md](mesh-deeper.md)
38. ~~Dogfood streams list evidence~~ **done** — soft `streams` step + report `streams_count`; see [mesh-dogfood.md](mesh-dogfood.md)
39. ~~Dogfood `streams_names` + gated streams delete~~ **done** — report name sample + `mesh streams --delete --name --yes`; see [mesh-dogfood.md](mesh-dogfood.md) / [mesh-deeper.md](mesh-deeper.md)
40. ~~Lean stream message list + `mesh streams --messages`~~ **done** — `ListStreamMessages` + CLI message inspection; see [mesh-deeper.md](mesh-deeper.md)
41. ~~Dogfood policy evidence~~ **done** — report `policy_mode` / `policy_source` / `policy_allow`; see [mesh-dogfood.md](mesh-dogfood.md)
42. ~~Lean mesh KV read + `mesh kv` CLI~~ **done** — `KVGet` / `KVListKeys` + `iomesh mesh kv`; see [mesh-deeper.md](mesh-deeper.md)
43. ~~Gated mesh KV put/delete + soft dogfood kv probe~~ **done** — `KVPut` / `KVDelete` + `--yes` CLI gates + `--kv-bucket` list-keys evidence; see [mesh-deeper.md](mesh-deeper.md) / [mesh-dogfood.md](mesh-dogfood.md)
44. ~~Lean mesh KV create-bucket~~ **done** — `KVCreateBucket` + `iomesh mesh kv --create-bucket --yes` (idempotent 409); see [mesh-deeper.md](mesh-deeper.md)
45. ~~Dogfood kv-ensure + ephemeral mesh pub~~ **done** — `--kv-ensure` / `kv_ensured` + `Pub` / `iomesh mesh pub --yes`; see [mesh-dogfood.md](mesh-dogfood.md) / [mesh-deeper.md](mesh-deeper.md)
46. ~~Agent-native setup lifecycle `/setup` (s1525–s1542)~~ **done** — init/preflight/portal/reload/pull/analyze/drift/repair; dual_write OFF · package wire ≠ Connected · not Memory GA; see [setup-lifecycle.md](setup-lifecycle.md)
47. ~~Edge user journey + soft residual-check lanes (s1554–s1586 continuum)~~ **done** — journey/wizard/portal-hitl/e4/tool-call/human-gates/e10 soft residual-checks; E10 Open · book-demo OFF · residual PASS ≠ invent Connected/GA; see [edge-user-journey.md](edge-user-journey.md)
48. ~~OSS packaging boundary + marketing demo path + Python SDK peer (s1582/s1590/s1666)~~ **done** — MIT harness vs private control plane honesty · `/onboard next marketing-demo` · Go+Python client SDK peers; see [oss-packaging-boundary.md](oss-packaging-boundary.md) · [marketing-demo-path.md](marketing-demo-path.md)
49. ~~Easy first-run + skills reload (s1670)~~ **done** — `/setup reload` re-scans skills · README first-run honesty; skills re-scan ≠ invent Connected
50. ~~Release v0.72.0 packaging cut~~ **done** — setup lifecycle + edge first-run continuum + residual soft checks + OSS packaging honesty
51. ~~Setup first-run residual continuum + release v0.73.0 (s1686/s1695/s1699)~~ **done** — CLI init dual-path next-step · preflight dual-path next-step · Memory Ops Pack local-primary honesty (not first-run required) · dual_write OFF · package wire ≠ Connected · Ops Pack optional · CLI has no setup reload
52. ~~Setup drift/repair dual-path next-step honesty (s1707)~~ **done** — `FormatDriftText` / `FormatRepairPlan` / `FormatRepairResult` dual path · in-session slash vs cold restart · CLI has no setup drift/repair/reload · dual_write OFF · package wire ≠ Connected · repair apply ≠ invent Connected; see [setup-lifecycle.md](setup-lifecycle.md)
53. ~~Setup reload/pull/analyze next-step honesty (s1711)~~ **done** — `SetupReloadNextStepLines` · `SetupPullNextStepLines` · `SetupAnalyzeNextStepLines` · reload in-session only (CLI has no setup reload) · pull dual path slash vs CLI `iomesh memory pull` · analyze dual path slash vs `/memory digest` · dual_write OFF · package wire ≠ Connected · pull ≠ invent Connected · analyze tick ≠ invent Connected · not Memory GA; see [setup-lifecycle.md](setup-lifecycle.md)
54. ~~Release v0.74.0 packaging cut (s1715)~~ **done** — setup Format*/next-step honesty continuum packaged: drift/repair dual-path (s1707) + reload/pull/analyze next-step (s1711) · dual_write OFF · package wire ≠ Connected · not Memory GA · CLI has no setup drift/repair/reload
55. ~~Setup init slash parity + portal next-step + PLATFORM_RESIDUAL label (s1723)~~ **done** — slash `/setup init` uses `SetupInitNextStepLines` (CLI parity with s1686) · `/setup portal` appends `SetupPortalNextStepLines` (browser HITL → preflight/reload dual path · agent MCP cannot write installs · catalog ≠ Connected) · optional `IOMESH_PLATFORM_RESIDUAL=1` via `PlatformResidualEnvOn` labels only (does not hide Edge OSS lanes · residual PASS ≠ invent control plane) · dual_write OFF · package wire ≠ Connected · not Memory GA; see [setup-lifecycle.md](setup-lifecycle.md) · [oss-packaging-boundary.md](oss-packaging-boundary.md)
56. ~~Integrations residual-honest next-step after list/plan/status/signing (s1727)~~ **done** — `IntegrationsNextStepLines` on catalog/plan/signing/status honesty footers + offline/tool-missing · portal HITL → `/setup preflight|reload` dual path · agent MCP cannot write installs · catalog ≠ Connected · template= ≠ install APPLY · dual_write OFF · not Memory GA; see [agent-integrations-setup.md](agent-integrations-setup.md)
57. ~~Release v0.75.0 packaging cut (s1731)~~ **done** — residual next-step honesty continuum packaged: setup init slash parity + portal next-step + IOMESH_PLATFORM_RESIDUAL label (s1723) · integrations list/plan/status/signing next-step (s1727) · dual_write OFF · package wire ≠ Connected · catalog ≠ Connected · not Memory GA
58. ~~Onboard residual-honest next-step after status/checklist/next/portal (s1825)~~ **done** — `OnboardNextStepLines` / `AionAgentOnboardingNextStepLines` dual path: TUI/session → `/setup preflight|reload` · optional `/integrations list` · `/onboard next portal-hitl|setup|memory`; cold start → restart iomesh · `iomesh setup preflight` · dual_write OFF · package wire ≠ Connected · catalog ≠ Connected · agent MCP cannot write installs · not Memory GA; see [oss-packaging-boundary.md](oss-packaging-boundary.md)
59. ~~Memory residual-honest next-step after status/help/digest (s1831)~~ **done** — `MemoryNextStepLines` dual path: TUI/session → `/setup preflight|reload` · optional `/memory digest` · `/onboard next memory|memory-pull`; cold start → restart iomesh · `iomesh setup preflight` · optional `iomesh memory pull` · dual_write OFF · not Memory GA · local-primary · package wire ≠ Connected · soft ≠ invent live dogfood; see [memory-mcp.md](memory-mcp.md)
60. Optional: platform remote multi-tenant metering UI
