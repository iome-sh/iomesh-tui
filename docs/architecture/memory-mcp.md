# Memory Palace + temporal MCP

First-class **Agentic Memory Palace** and **temporal recall** for `iomesh-tui`, without embedding Palace inside the TUI process.

Platform ships `aion-memory-mcp` (stdio **and** streamable HTTP) with tools:

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
| **0** | **done** | Attach `aion-memory-mcp` via existing MCP client; documented example |
| **1** | **done** | `[memory]` auto-recall inject, opt-in auto-ingest, `/memory` slash |
| **2** | **done (v0.3.0)** | HTTP MCP primary path + optional dual-write to mesh `MEMORY_INGEST` |
| **3 partial** | **done (dogfood)** | Async `MEMORY_RPC` recall probe in mesh dogfood (`PublishMemoryRecall`, same `session_id` as ingest) |
| **3+** | planned | Pure SDK **sync** retrieve HTTP when operators want non-MCP request/response hits |

**Non-goals:** private monorepo imports in public TUI; embedding Qdrant/Palace in-process; dependency on `iomesh-client-sdk-go`.

## Phase 0–1 — MCP hooks (stdio or HTTP)

### Preferred: streamable HTTP (platform M1)

```bash
aion-memory-mcp -http-addr :8080 -palace-root /data/memory-palaces
# MCP endpoint: http://127.0.0.1:8080/mcp
# health:       http://127.0.0.1:8080/healthz
```

```toml
[mcp]
enabled = true

[[mcp.servers]]
name = "memory"
url = "http://127.0.0.1:8080/mcp"
allow_loopback = true
mutating = true

[memory]
enabled = true
server = "memory"          # must match [[mcp.servers]].name
tenant = "dept.research"   # or MEMORY_TENANT / IOMESH_MEMORY_TENANT
auto_recall = true         # inject <memory-context> each turn (fail-open)
auto_ingest = false        # opt-in: write user+assistant turns after success
# dual_write = false       # opt-in: also publish MEMORY_INGEST (needs [iomesh])
# limit = 8
# max_snippet_bytes = 6000
```

Env: `MEMORY_MCP_HTTP_ADDR` / `AION_MEMORY_MCP_HTTP_ADDR`, path `MEMORY_MCP_HTTP_PATH` (default `/mcp`).

### Alternate: stdio

```bash
# from aion monorepo
go build -o "$HOME/bin/aion-memory-mcp" ./cmd/aion-memory-mcp
```

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
server = "memory"
tenant = "dept.research"
auto_recall = true
auto_ingest = false
```

### Verify

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
| `IOMESH_MEMORY_AUTO_INGEST=1` | Enable post-turn ingest (MCP and/or dual-write) |
| `IOMESH_MEMORY_DUAL_WRITE=1` | Also publish async `MEMORY_INGEST` when mesh enabled |
| `IOMESH_MCP=1` | Enable MCP section |

## Phase 1 — runtime loop

```
user turn
  → [optional] memory_retrieve(query=userText) → <memory-context> system msg
  → LLM + tools
  → [optional auto_ingest]
        memory_ingest_turn(user) + memory_ingest_turn(assistant)   # MCP when connected
        + PublishMemoryIngest → MEMORY_INGEST                      # when dual_write
```

- **Fail-open**: MCP down, empty hits, dual-write errors, or tool errors never fail the turn.
- **No Palace import**: only MCP `tools/call` over the existing client (+ lean HTTP publish).
- **Mutating**: auto-ingest bypasses the interactive approval UI (operator opt-in via `auto_ingest`); interactive `mcp__memory__*` still requires approval when `mutating=true`.

## Phase 2 — dual-write MEMORY_INGEST (v0.3.0)

When `dual_write = true` (or `IOMESH_MEMORY_DUAL_WRITE=1`) **and** `[iomesh]` mesh client is enabled:

1. After MCP ingest (or **instead of** MCP when no server is connected), publish an async envelope to  
   `POST /v1/streams/MEMORY_INGEST/publish`.
2. Subject: `{tenant}.memory.ingest.turn`
3. Payload (base64 JSON, same wire as public SDK):

```json
{
  "type": "memory_ingest",
  "session_id": "…",
  "role": "user|assistant",
  "content": "…",
  "event_time": "2026-07-16T12:00:00Z",
  "session_seq": 1
}
```

- `session_seq` is **monotonic per Runtime** (process lifetime of the agent session).
- Tenant from `[memory].tenant`, else mesh tenant.
- Dual-write is **independent** of MCP success (fail-open both ways).
- Useful for durable stream consumers / temporal pipelines without embedding Palace in the TUI.

Requires mesh:

```toml
[iomesh]
enabled = true
endpoint = "https://mesh.example"
tenant = "dept.research"
# api_key_env = "IOMESH_API_KEY"

[memory]
enabled = true
auto_ingest = true
dual_write = true
tenant = "dept.research"
```

### Dogfood dual-write probe

`iomesh mesh dogfood` includes a **memory_ingest** step by default when mesh is enabled. It exercises the same `PublishMemoryIngest` path (not MCP Palace write):

```bash
iomesh mesh dogfood --tenant dept.research
# soft: FAIL on publish → SKIP unless --strict
iomesh mesh dogfood --strict
iomesh mesh dogfood --skip-memory   # omit the step
```

See [mesh-dogfood.md](mesh-dogfood.md) for soft vs strict matrix. Unit coverage: `go test ./internal/iomesh` (httptest mock returns 200 on `/v1/streams/MEMORY_INGEST/publish`).

## Slash commands

| Command | Behavior |
|---------|----------|
| `/memory` | Status: enabled, server connected?, flags (incl. `dual_write`), tenant |
| `/memory recall [query]` | Retrieve (default query = last user text or `"*"`) |
| `/memory ingest <text>` | Ingest a user turn (MCP and/or dual-write) |

## Platform gaps (aion backlog)

Tracked in aion `aion-foundation-pending-todos.md`:

| ID | Gap |
|----|-----|
| M1 | Streamable HTTP for `aion-memory-mcp` — **shipped** (TUI HTTP path ready) |
| M2 | Sync `POST /v5/memory/retrieve` (SDK) — optional non-MCP clients |
| M3 | SDK temporal envelope fields — **shipped** (TUI dual-write mirrors subset) |
| M4 | Stage warm `aion-memory` path (prod lean absent) |
| M5 | Entitlements fail-closed on MCP |

## Package map

| Path | Role |
|------|------|
| `internal/config` | `[memory]` section + env (`dual_write`) |
| `internal/iomesh/memory.go` | `PublishMemoryIngest` lean HTTP (no SDK dep) |
| `internal/agent/memory.go` | Recall / ingest / dual-write helpers |
| `internal/agent/agent.go` | `RunTurn` hooks |
| `internal/tui/tui.go` | `/memory` slash |
| `configs/config.example.toml` | Copy-paste wire-up |

## Honesty

- Local Palace via stdio/HTTP MCP ≠ multi-tenant Cloud Run Memory Palace with full entitlements.
- Dual-write is **best-effort** stream publish; it does not guarantee Palace persistence by itself.
- “Native Vertex” / G4S claims are separate (see marketing claim matrix); memory is **Palace + MCP**, not Vertex.
- Do not claim temporal pipeline is live unless stage/prod embedding + temporal flags are on.
