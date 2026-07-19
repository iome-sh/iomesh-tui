# Stage mesh dogfood

Operator smoke for **I/O Mesh** integration from the public `iomesh-tui` harness.

## Checks

### Preflight wait (`mesh wait`)

Operator preflight (s291) polls readiness without running the full dogfood suite:

```bash
iomesh mesh wait [--timeout 30s] [--interval 500ms] [--require-health]
# Exit 0 + "PASS mesh wait: ready" when Ready succeeds; non-zero + FAIL on deadline.
```

`Client.WaitReady` retries `GET /ready` (or `/readyz`) until success or context deadline. With `--require-health`, each attempt requires `GET /health` OK first. Disabled/empty endpoint is offline-first (immediate success). Use before stage dogfood or agent attach when the broker is still warming.

### Dogfood WaitReady soft preflight (s297)

Optional **in-suite** soft preflight after health and before the single-shot ready step:

```bash
iomesh mesh dogfood --wait-ready 10s --wait-interval 500ms --wait-require-health
# wait_ready PASS when Ready (and optional Health) succeed within budget
# timeout → SKIP wait_ready (soft) unless --strict (then FAIL)
# single-shot ready still runs after wait for latency evidence
```

| Flag | Maps to | Default |
|------|---------|---------|
| `--wait-ready dur` | `DogfoodOptions.WaitReady` | `0` (off — no `wait_ready` step) |
| `--wait-interval dur` | `DogfoodOptions.WaitReadyInterval` | `500ms` when wait-ready > 0 and interval is 0 |
| `--wait-require-health` | `DogfoodOptions.WaitRequireHealth` | false |

Effective budget is `min(WaitReady, parent ctx remaining)` via `context.WithTimeout`. Report top-level `wait_ready_ms` is always emitted (configured budget in ms; `0` = off). Outcome is on the `wait_ready` step (`PASS` / `SKIP` / `FAIL`).

### Operator status (`mesh status`)

One-shot operator snapshot (s296) without the full dogfood suite:

```bash
iomesh mesh status [--json] [--endpoint url] [--config path]
# Human: StatusLine + endpoint/tenant/org/workspace/ua + health/ready (ok|err|skipped)
# --json: structured object with the same fields (health/ready fail-open — exit 0 even on probe err)
```

Builds the client like dogfood/wait. Prints `StatusLine` config summary plus optional one-shot `Health` / `Ready` probes (errors shown as `err` + message; never fail the command).

| Step | Request | Soft (default) | Strict (`--strict`) |
|------|---------|----------------|---------------------|
| enabled | config | SKIP if disabled | same |
| health | `GET /health` | **FAIL** if down | **FAIL** |
| wait_ready | poll Ready (optional Health) | only when `--wait-ready` > 0; timeout → **SKIP** | timeout → **FAIL** |
| ready | `GET /ready` or `/readyz` | SKIP if 404 | FAIL if missing/error |
| context | `POST /v1/context/query` | SKIP if empty (fail-open) | FAIL if empty |
| emit | `POST /v1/streams/dept/publish` (`dept.agent.dogfood`) | SKIP on error | FAIL on error |
| llm_meter | `POST /v1/streams/dept/publish` (`dept.agent.llm_call` probe) | SKIP on error (`--skip-emit` / streams off) | FAIL on error |
| policy | `POST /v1/policy/evaluate` | SKIP if mode off / 404 / fail-open | FAIL if mode on and evaluate soft-fails |
| catalog | broker + portal list | SKIP if plane off / fail-open | FAIL if fail-open |
| memory_ingest | `POST /v1/streams/MEMORY_INGEST/publish` | SKIP on error (`--skip-memory` forces SKIP) | FAIL on error |
| memory_recall | `POST /v1/streams/MEMORY_RPC/publish` | SKIP on error (`--skip-memory` forces SKIP) | FAIL on error |
| memory_retrieve | `POST /v1/memory/retrieve` (fallback `/v5`) | SKIP on error (`--skip-memory` forces SKIP) | FAIL on error |

Context requests set `include_lineage` when configured (lineage count shown on PASS detail).

### emit + llm_meter (dept streams / remote metering)

When `emit_dept_streams` is on (default) and not `--skip-emit`:

1. **emit** — `dept.agent.dogfood` probe (generic stage event)
2. **llm_meter** — `dept.agent.llm_call` zero-token probe (same wire as live `RecordLLMCall` for platform remote metering dashboards)

Both set `session_id={tenant}.mesh-dogfood` for correlation with memory_*. PASS detail appends `org=` / `workspace=` when Client OrgID/WorkspaceID are set (headers `X-IOMesh-Org` / `X-IOMesh-Workspace` on the POST). Soft: transport/HTTP errors → **SKIP**; `--strict` → **FAIL**.

### memory_ingest (dual-write probe)

Included **by default** when mesh is enabled (not gated on agent `[memory].dual_write`). Calls the same lean path as Phase 2 dual-write (`PublishMemoryIngest`):

- Subject: `{tenant}.memory.ingest.turn`
- Envelope: `type=memory_ingest`, `role=tool`, `content=iomesh-tui dual-write dogfood`, `event_time=now`, `session_seq=1`, `session_id={tenant}.mesh-dogfood` (or `mesh-dogfood` when tenant unset)
- Soft mode: publish/transport errors → **SKIP** (fail-open); `--strict` → **FAIL**
- Offline / mesh disabled: whole report is SKIP (no memory step)
- **PASS detail** includes stream, subject, and seq when available. When Client `[iomesh] org` / `workspace` (`OrgID` / `WorkspaceID`) are set, detail also appends `org=…` and/or `workspace=…` as operator-visible evidence that dual-write publish used those headers (`X-IOMesh-Org` / `X-IOMesh-Workspace`). Empty values are omitted (no `org=` token). Detail always includes temporal correlation from the envelope sent: `session_seq=N` and `session_id=…` when non-empty (s243). Detail **always** ends with `dual_write=true` or `dual_write=false` from Client `[memory].dual_write` / `IOMESH_MEMORY_DUAL_WRITE` (report evidence only — does not gate the probe), so human-readable reports and step logs show mode without relying only on top-level JSON.

### memory_recall (async MEMORY_RPC probe — s247)

Runs **after** `memory_ingest` (same `--skip-memory` gate). Calls `PublishMemoryRecall` (SDK-parity fire-and-forget; **not** sync hits):

- Stream: `MEMORY_RPC` · subject: `{tenant}.memory.retrieve.request`
- Payload: `type=memory_recall`, `tenant_id`, `query=iomesh-tui dual-write dogfood`, `limit=8`, **`session_id` identical to ingest** (`{tenant}.mesh-dogfood`)
- Soft: transport errors → **SKIP**; `--strict` → **FAIL**
- **PASS detail**: stream, subject, seq, optional `org=`/`workspace=`, `session_id=…`, `dual_write=true|false`

### memory_retrieve (sync HTTP — Phase 3 / s251 + sidecar s269)

Runs **after** async `memory_recall`. Calls `RetrieveMemory` (request/response against **memory sidecar** HTTP, not `MEMORY_RPC`):

- Path: `POST /v1/memory/retrieve` then `/v5/memory/retrieve` on 404
- **Base URL**: `[memory].endpoint` / `IOMESH_MEMORY_ENDPOINT` / `MEMORY_SIDECAR_URL` when set (stage **warm plane**); else mesh `IOMESH_ENDPOINT` (broker-only often 404 → soft SKIP)
- Body: `tenant_id`, `type=memory_recall`, `query=iomesh-tui dual-write dogfood`, `limit=8`, **`session_id` same as ingest**
- Soft: transport/HTTP errors → **SKIP**; `--strict` → **FAIL**
- Empty `memories: []` is still **PASS** with `hits=0` (valid 200)
- **PASS detail**: `POST /v1/memory/retrieve hits=N [org=] [workspace=] session_id=… memory_base=sidecar|mesh dual_write=…` (no `MEMORY_RPC`)

Final line: `RESULT=PASS …` or `RESULT=FAIL …`.

### Stage warm memory plane

When the mesh broker does not terminate `/v1/memory/retrieve`, point dogfood (and agent auto-recall) at the sidecar:

```bash
export IOMESH_ENDPOINT=https://mesh.stage.example   # health, streams, catalog
export IOMESH_MEMORY_ENDPOINT=http://127.0.0.1:8765 # or stage memory sidecar URL
# aion-compatible alias: MEMORY_SIDECAR_URL=…

iomesh mesh dogfood --json
# top-level memory_endpoint set; memory_retrieve detail ends with memory_base=sidecar
```

CLI override: `iomesh mesh dogfood --memory-endpoint http://127.0.0.1:8765`.

## JSON report (`--json`)

`iomesh mesh dogfood --json` / `FormatReportJSON` emits indented JSON for stage CI evidence. Top-level fields:

| Field | Type | Notes |
|-------|------|-------|
| `endpoint` | string | Mesh base URL |
| `tenant` | string | omitted when empty |
| `org` | string | Client `[iomesh] org` / `IOMESH_ORG` (PlanGate); omitted when empty |
| `workspace` | string | Client `[iomesh] workspace` / `IOMESH_WORKSPACE`; omitted when empty. **Not** the context-plane path (`DogfoodOptions.Workspace`) |
| `dual_write` | bool | Agent `[memory].dual_write` / `IOMESH_MEMORY_DUAL_WRITE` from Client cfg (**always emitted**, default `false`). Report-only — does **not** gate the `memory_ingest` probe |
| `catalog_source` | string | Last catalog probe source (`mesh` \| `portal` \| `fail-open` \| `off`); omitted when empty (mesh disabled before catalog step) (s292) |
| `catalog_count` | int | Product count from last `ListCatalog` (**always emitted**, `0` when none/off). Top-level CI evidence — no step-detail scrape (s292) |
| `context_chars` | int | `len(FormatContextSnippet)` from last context probe (**always emitted**, `0` when skip/off/empty) (s296) |
| `context_lineage_count` | int | `len(res.Lineage)` from last `QueryContext` (**always emitted**, `0` when skip/off/empty) (s296) |
| `wait_ready_ms` | int | Configured WaitReady budget in ms (**always emitted**, `0` = off / no preflight) (s297). Outcome on `wait_ready` step detail |
| `memory_endpoint` | string | Optional memory sidecar base (`[memory].endpoint` / `IOMESH_MEMORY_ENDPOINT`); omitted when empty (retrieve uses mesh `endpoint`) |
| `user_agent` | string | Package mesh HTTP User-Agent (`iomesh-tui/<version>` via `iomesh.UserAgent()`); always set for CI evidence (s290) — not scraped from server |
| `strict` | bool | `--strict` |
| `ok` | bool | no FAIL steps |
| `summary` | string | e.g. `PASS (pass=N skip=M)` |
| `result` | string | `PASS` \| `FAIL` \| `SKIP` (summary prefix) |
| `started` / `finished` | RFC3339 | probe window |
| `steps` | array | `{name,status,detail?,latency?}` |

`org` / `workspace` are structured multi-tenant evidence for operators and aion gates (parseable without scraping step detail). s235 still embeds the same values in the `memory_ingest` PASS detail string when set.

`dual_write` is structured dual-write **mode** evidence for CI (s239): parse top-level JSON instead of grepping detail strings. The same mode is also always present on the `memory_ingest` PASS detail string as `dual_write=true|false` (s241) for human logs. The CLI wires `cfg.Memory.DualWrite` into `iomesh.Config.DualWrite` when running `mesh dogfood`.

## CLI

```bash
export IOMESH_ENDPOINT=https://mesh.stage.example
# Or control-plane / portal edge for catalog federation:
# export IOMESH_ENDPOINT=https://cp.stage.example
export IOMESH_API_KEY=…          # optional
export IOMESH_TENANT=acme        # optional
# Warm memory plane (optional; required for memory_retrieve PASS when broker has no /v1/memory/*):
# export IOMESH_MEMORY_ENDPOINT=http://127.0.0.1:8765

iomesh mesh dogfood
iomesh mesh dogfood --strict
iomesh mesh dogfood --json       # stage CI evidence
iomesh mesh dogfood --wait-ready 10s --wait-interval 500ms   # soft ready preflight (s297)
iomesh mesh dogfood --endpoint "$IOMESH_ENDPOINT" --tenant acme
iomesh mesh dogfood --memory-endpoint "$IOMESH_MEMORY_ENDPOINT"
iomesh mesh dogfood --skip-context --skip-emit --skip-memory   # health-only-ish
iomesh mesh catalog              # broker then portal paths
iomesh mesh status [--json]      # operator snapshot (StatusLine + Health/Ready)
```

## Script / Make

```bash
./scripts/mesh_dogfood.sh              # live
./scripts/mesh_dogfood.sh --strict
./scripts/mesh_dogfood.sh --unit       # go test ./internal/iomesh (CI)

make dogfood
make dogfood-unit
```

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | `RESULT=PASS` or mesh disabled SKIP (offline-first) |
| 1 | any hard FAIL |

## Package

- `internal/iomesh/dogfood.go` — `Client.Dogfood`, `Ready`, `EmitErr`, `PublishMemoryIngest` step, `FormatReport`
- `internal/iomesh/memory.go` — dual-write publish wire format
- `cmd/iomesh` — `mesh dogfood|probe|status|wait|catalog|usage`
