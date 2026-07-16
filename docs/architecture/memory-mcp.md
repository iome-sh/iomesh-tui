# I/O Mesh memory + temporal recall via MCP

Plan for wiring **Agentic Memory Palace** (aion `aion-memory-mcp` + temporal pipeline) into **iomesh-tui** as a first-class agent capability.

## Current state (as of v0.2.x)

| Layer | What exists | Gap |
|-------|-------------|-----|
| **aion** `cmd/aion-memory-mcp` | stdio MCP server with temporal-aware tools | No streamable HTTP transport for remote agents |
| **aion** memory pipeline (UC-159) | Async embed, DLQ, temporal envelope, premium recall policy | `aion-memory` **absent** from prod lean fleet |
| **SDK** `iomeshclient` | `PublishMemoryIngest`, `RequestMemoryRecall` (async stream publish) | No **sync** recall RPC; envelope lacks full temporal fields |
| **iomesh-tui** MCP client | stdio + HTTP, tools/resources/prompts | No first-class memory config, auto-ingest, or recall injection |
| **iomesh-tui** mesh client | catalog / context / policy / meter | Not connected to Palace |

### aion-memory-mcp tools (already ship)

| Tool | Role |
|------|------|
| `memory_ingest_turn` | Conversation turn → Palace (`event_time`, `session_seq` for temporal) |
| `memory_ingest_event` | Dept/org event with subject + `event_time` (timeline) |
| `memory_retrieve` | Hybrid recall; `since` / `until` / `session_id` / `session_seq` |
| `memory_search_semantic` | Tier-4 semantic facts |
| `memory_timeline` | Event-time ordered entries |
| `memory_patterns_list` / `memory_anomalies_list` | Temporal pattern/anomaly notes |
| `memory_compact_status` / `memory_trigger_compact` | Compaction control plane |

Resources: `memory://{tenant}/stats|timeline|session/{id}/turns|semantic/facts`.

## Target architecture

```text
┌─────────────────┐     MCP stdio / HTTP      ┌──────────────────────┐
│   iomesh-tui    │ ─────────────────────────► │  aion-memory-mcp     │
│  agent loop     │   tools + resources        │  Palace + temporal   │
└────────┬────────┘                            └──────────┬───────────┘
         │ optional dual path                             │
         │ SDK streams                                    │ optional audit
         ▼                                                ▼
┌─────────────────┐                            ┌──────────────────────┐
│ iomesh-client   │  MEMORY_INGEST / RPC       │  aion broker         │
│ -sdk-go         │ ─────────────────────────► │  + embed consumer    │
└─────────────────┘                            └──────────────────────┘
```

**Default path for the TUI:** MCP (reuse existing client; no private aion imports).  
**Optional path:** pure SDK publish for fire-and-forget ingest when MCP is unavailable.

## Phased delivery

### Phase 0 — Config & docs (iomesh-tui, no aion change) — **shipped**

Example in `configs/config.example.toml` + this doc. Operators can attach memory **today** via stdio:

```toml
[mcp]
enabled = true

[[mcp.servers]]
name = "aion-memory"
command = "aion-memory-mcp"   # or path to binary
args = ["-palace-root", "/data/memory-palaces"]
env = { MEMORY_TENANT = "dept.engineering" }
mutating = true   # model-invoked ingest tools still require approval / --yolo

[memory]
enabled = true
server = "aion-memory"
tenant = "dept.engineering"
auto_recall = true
auto_ingest = false   # opt-in write-back
recall_limit = 8
```

- Temporal args on MCP tools: `event_time`, `session_seq`, `since`/`until`.
- CLI: `iomesh mcp --connect` lists tools/resources; TUI `/memory` for status.
- Env: `IOMESH_MEMORY=1`, `IOMESH_MEMORY_INGEST=1`, `IOMESH_MEMORY_TENANT` / `MEMORY_TENANT`.

### Phase 1 — Agent memory loop (iomesh-tui) — **shipped**

| Feature | Behaviour |
|---------|-----------|
| Config `[memory]` | `enabled`, `server` (default `aion-memory`), `tenant`, `auto_ingest`, `auto_recall`, `recall_limit` |
| **Auto-recall** (fail-open) | Before LLM: `memory_retrieve` with user text + `session_id`; inject `<iomesh-memory>` |
| **Auto-ingest** (opt-in) | After turn: `memory_ingest_turn` for user + assistant with `event_time=now`, `session_seq` monotonic |
| **Slash** `/memory` | Connection + last hit count + flags |
| Tests | `FormatMemoryRecallSnippet` + ConfigureMemory unit tests |

Still **stdio MCP only** until aion HTTP MCP (platform M1).

### Phase 2 — aion platform gaps (required for remote / stage)

Tracked in aion pending TODOs (see platform backlog). Blocking for Cloud Run / multi-tenant:

1. **Streamable HTTP** for `aion-memory-mcp` (same contract as TUI HTTP MCP client).
2. **Sync recall HTTP API** on control plane or sidecar (`POST /v1/memory/retrieve`) so SDK and non-MCP agents get request/response without stream race.
3. **SDK temporal envelope** — extend `MemoryEnvelope` with `event_time`, `session_seq`, `valid_from`/`valid_to` aligned to `domain.MemoryEnvelope`.
4. **Stage dogfood service** — optional `aion-memory` (or CP-embedded MCP HTTP) not present under max-lean; document warm path for memory demos.
5. **Entitlements** — MCP tools should fail closed with clear error when workspace lacks `agent_memory` (plan gate).

### Phase 3 — Temporal-first product UX

| Feature | Notes |
|---------|-------|
| Timeline slash / tool | Surface `memory_timeline` in TUI |
| Session binding | Map iomesh session id → Palace `session_id` |
| Dept event ingest | Optional: mesh catalog subjects → `memory_ingest_event` |
| Dual-write audit | MCP `-enable-audit` → broker `MEMORY_INGEST` for lineage |

### Phase 4 — Metering & release

- Emit `dept.agent.memory_*` via existing mesh meter/emit when recall/ingest succeeds.
- Tag **iomesh-tui v0.3.0** when Phase 1 lands; aion foundation gate only if Phase 2 needs fleet deploy.

## Security / product rules

- Ingest is **mutating** → approval gates apply (not silent by default).
- Fail-open on recall transport errors (agent continues without memory).
- Tenant isolation: never pass empty tenant; default from config/`MEMORY_TENANT`.
- No private monorepo imports in public `iomesh-tui` (MCP + public SDK only).

## Success criteria

1. Local: stdio `aion-memory-mcp` + iomesh-tui auto-recall inject + opt-in ingest (Phase 1).  
2. Stage: HTTP MCP or sync retrieve against warm memory path (Phase 2).  
3. Temporal: `since`/`until`/`event_time` round-trip verified in unit/dogfood tests.  
4. Docs honest: lean fleet without `aion-memory` remains supported offline-first.

## Out of scope (initial)

- Running Palace inside the TUI process.
- Replacing MCP with a proprietary protocol.
- Multi-region Qdrant ops (stays aion provision worker).
