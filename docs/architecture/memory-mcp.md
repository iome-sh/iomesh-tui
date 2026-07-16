# Memory Palace + temporal MCP

First-class **Agentic Memory Palace** and **temporal recall** for `iomesh-tui`, without embedding Palace inside the TUI process.

Platform ships `aion-memory-mcp` (stdio) with tools:

| Tool | Purpose |
|------|---------|
| `memory_ingest_turn` | Persist a conversation turn (tiered Palace) |
| `memory_retrieve` | Query memories (optional `session_id`, `since`/`until`, `session_seq`) |
| `memory_timeline` | Temporal timeline slice |
| `memory_search_semantic` | Semantic facts |
| patterns / anomalies / compact | Ops helpers |

Resources: `memory://{tenant}/…` (stats, timeline, session turns, facts).

## Phases

| Phase | Status | Work |
|-------|--------|------|
| **0** | **this doc + config** | Attach `aion-memory-mcp` via existing MCP client; documented example |
| **1** | **this PR** | `[memory]` auto-recall inject, opt-in auto-ingest, `/memory` slash |
| **2+** | planned | Depends on platform M1 (HTTP MCP) + M2 (sync retrieve API) for remote stage; dual-write emit; v0.3.0 |

**Non-goals:** private monorepo imports in public TUI; embedding Qdrant/Palace in-process.

## Phase 0 — attach stdio today

1. Build platform binary (private monorepo):

```bash
# from aion monorepo
go build -o "$HOME/bin/aion-memory-mcp" ./cmd/aion-memory-mcp
```

2. Enable MCP + memory hooks in `~/.iomesh/config.toml`:

```toml
[mcp]
enabled = true

[[mcp.servers]]
name = "memory"
command = "aion-memory-mcp"
args = ["-palace-root", "/data/memory-palaces"]
# env = { "MEMORY_TENANT" = "dept.research", "QDRANT_URL" = "…" }
mutating = true   # ingest tools need approval unless --yolo
# tool_timeout_sec = 60

[memory]
enabled = true
server = "memory"          # must match [[mcp.servers]].name
tenant = "dept.research"   # or MEMORY_TENANT / IOMESH_MEMORY_TENANT
auto_recall = true         # inject <memory-context> each turn (fail-open)
auto_ingest = false        # opt-in: write user+assistant turns after success
# limit = 8
# max_snippet_bytes = 6000
```

3. Verify:

```bash
iomesh mcp --connect
# expect memory_* tools under server "memory"

iomesh   # interactive
# /memory                 → status
# /memory recall <query>  → call memory_retrieve
```

Agent tools also appear as `mcp__memory__memory_retrieve` (etc.) when MCP is attached.

### Env overrides

| Env | Effect |
|-----|--------|
| `IOMESH_MEMORY=1` | Enable `[memory]` hooks |
| `IOMESH_MEMORY_TENANT` / `MEMORY_TENANT` | Default tenant for hooks + slash |
| `IOMESH_MEMORY_AUTO_RECALL=0` | Disable per-turn retrieve inject |
| `IOMESH_MEMORY_AUTO_INGEST=1` | Enable post-turn ingest (still uses MCP tools) |
| `IOMESH_MCP=1` | Enable MCP section |

## Phase 1 — runtime loop

```
user turn
  → [optional] memory_retrieve(query=userText) → <memory-context> system msg
  → LLM + tools
  → [optional auto_ingest] memory_ingest_turn(user) + memory_ingest_turn(assistant)
```

- **Fail-open**: MCP down, empty hits, or tool errors never fail the turn.
- **No Palace import**: only MCP `tools/call` over the existing client.
- **Mutating**: auto-ingest bypasses the interactive approval UI (operator opt-in via `auto_ingest`); interactive `mcp__memory__*` still requires approval when `mutating=true`.

## Slash commands

| Command | Behavior |
|---------|----------|
| `/memory` | Status: enabled, server connected?, flags, tenant |
| `/memory recall [query]` | Retrieve (default query = last user text or `"*"`) |
| `/memory ingest <text>` | Ingest a user turn (requires connected server) |

## Platform gaps (aion backlog)

Tracked in aion `aion-foundation-pending-todos.md`:

| ID | Gap |
|----|-----|
| M1 | Streamable HTTP for `aion-memory-mcp` (remote / Cloud Run) |
| M2 | Sync `POST /v1/memory/retrieve` (SDK recall is async publish today) |
| M3 | SDK temporal envelope fields |
| M4 | Stage warm `aion-memory` path (prod lean absent) |
| M5 | Entitlements fail-closed on MCP |

Phase 0–1 work on **stdio** without M1–M2.

## Package map

| Path | Role |
|------|------|
| `internal/config` | `[memory]` section + env |
| `internal/agent/memory.go` | Recall / ingest helpers |
| `internal/agent/agent.go` | `RunTurn` hooks |
| `internal/tui/tui.go` | `/memory` slash |
| `configs/config.example.toml` | Copy-paste wire-up |

## Honesty

- Local Palace via stdio ≠ multi-tenant Cloud Run Memory Palace.
- “Native Vertex” / G4S claims are separate (see marketing claim matrix); memory is **Palace + MCP**, not Vertex.
- Do not claim temporal pipeline is live unless stage/prod embedding + temporal flags are on.
