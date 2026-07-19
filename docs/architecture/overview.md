# I/O Mesh TUI — Architecture Overview

Go rewrite of [xAI Grok Build](https://github.com/xai-org/grok-build) with a **multi-provider LLM router** (DeepSeek · Grok · Gemini · Vertex; OpenAI-compatible custom endpoints) and optional **I/O Mesh** context / policy / metering hooks. Default auto-cascade remains DeepSeek Flash → Pro → Grok for cost; pin any built-in model with `-m` / `/model`.

## Goals

| Goal | Approach |
|------|----------|
| Cost-efficient coding agent | Default **DeepSeek V4 Flash**; step-up **V4 Pro**; premium **Grok 4.5** |
| OpenAI-compatible, pure Go | `net/http` + JSON SSE; no vendor SDKs in the hot path |
| Grok Build parity (incremental) | TUI / headless / ACP modes; tools; workspace; subagents (later) |
| I/O Mesh integration | Context plane injection + `dept.*` usage streams (fail-open) |

## Package map (Grok Build → Go)

| Grok Build (Rust) | Go package | Status |
|-------------------|------------|--------|
| `xai-grok-pager-bin` | `cmd/iomesh` | Scaffold |
| `xai-grok-pager` | `internal/tui` | Full-screen Bubble Tea + classic REPL fallback |
| `xai-grok-shell` | `internal/agent` | Turn loop + tools |
| `xai-grok-tools` | `internal/agent` tools | read/list/grep/shell/write |
| `xai-grok-workspace` | `internal/workspace` | Rooted FS + path jail |
| config / custom models | `internal/config` + `internal/router` | TOML + cascade |
| — | `internal/iomesh` | Lean platform client (no SDK dep; full client → [iomesh-client-sdk-go](https://github.com/iome-sh/iomesh-client-sdk-go)) |
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
17. ~~Memory Palace MCP Phase 0–1~~ **done** — see [memory-mcp.md](memory-mcp.md) (stdio attach, auto-recall, `/memory`, opt-in ingest)
18. ~~Memory Phase 2~~ **done** — HTTP MCP path + dual-write `MEMORY_INGEST` (v0.3.0); see [memory-mcp.md](memory-mcp.md)
19. ~~Dogfood async MEMORY_RPC recall (session correlation)~~ **done** — see [mesh-dogfood.md](mesh-dogfood.md)
20. ~~Sync HTTP memory retrieve + dogfood `memory_retrieve`~~ **done** — see [mesh-dogfood.md](mesh-dogfood.md) / [memory-mcp.md](memory-mcp.md)
21. ~~Agent auto-recall prefer sync HTTP~~ **done** — mesh `RetrieveMemory` first, MCP fallback; see [memory-mcp.md](memory-mcp.md)
22. ~~Stage warm memory plane / sidecar dogfood~~ **done** — `[memory].endpoint` / `IOMESH_MEMORY_ENDPOINT`; see [mesh-dogfood.md](mesh-dogfood.md)
23. ~~Release v0.4.0~~ **done** — Memory Phase 3+ + dogfood evidence packaged (tag on merge)
24. ~~GoReleaser multi-platform binaries + usage JSON~~ **done** — see [RELEASING.md](../../RELEASING.md) / [mesh-deeper.md](mesh-deeper.md)
25. ~~Release v0.5.0~~ **done** — GoReleaser packaging + usage JSON
26. ~~Remote metering emit path (org/workspace + llm_meter dogfood)~~ **done** — see [mesh-deeper.md](mesh-deeper.md) / [mesh-dogfood.md](mesh-dogfood.md)
27. ~~Release v0.6.0~~ **done** — multi-tenant metering emit packaging
28. ~~Dept emit publish wire parity (SDK + TUI)~~ **done** — `/v1/streams/dept/publish` + SDK `EmitLLMCall` (s284)
29. ~~GoReleaser SPDX SBOM on release assets~~ **done** — see [RELEASING.md](../../RELEASING.md) (s285)
30. Optional: platform remote multi-tenant metering UI; cosign/keyless signing
