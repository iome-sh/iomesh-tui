# I/O Mesh TUI — Architecture Overview

Go rewrite of [xAI Grok Build](https://github.com/xai-org/grok-build) with first-class **I/O Mesh** platform hooks and a **DeepSeek-first** LLM cascade.

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
| `xai-grok-pager` | `internal/tui` | REPL scaffold (fullscreen TUI next) |
| `xai-grok-shell` | `internal/agent` | Turn loop + tools |
| `xai-grok-tools` | `internal/agent` tools | read/list/grep/shell/write |
| `xai-grok-workspace` | `internal/workspace` | Rooted FS + path jail |
| config / custom models | `internal/config` + `internal/router` | TOML + cascade |
| — | `internal/iomesh` | Platform client |
| subagents | `internal/subagent` + agent tools | explore/plan/gp + background |
| MCP / skills / sandbox | TBD | Planned |

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

1. Full-screen TUI (Bubble Tea / custom renderer) with scrollback + permissions
2. ACP `agent stdio` JSON-RPC
3. ~~Subagent orchestration~~ **done** — see [subagents.md](subagents.md)
4. MCP client + skills loader
5. Session persistence + compaction
6. Git worktree isolation for subagents
7. Deeper I/O Mesh: lineage-aware context, Rego policy gates, metering dashboards
