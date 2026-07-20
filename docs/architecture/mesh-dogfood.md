# Stage mesh dogfood

Operator smoke for **I/O Mesh** integration from the public `iomesh-tui` harness.

## Checks

### Preflight wait (`mesh wait`)

Operator preflight polls readiness without running the full dogfood suite:

```bash
iomesh mesh wait [--timeout 30s] [--interval 500ms] [--require-health]
# Exit 0 + "PASS mesh wait: ready" when Ready succeeds; non-zero + FAIL on deadline.
```

`Client.WaitReady` retries `GET /ready` (or `/readyz`) until success or context deadline. With `--require-health`, each attempt requires `GET /health` OK first. Disabled/empty endpoint is offline-first (immediate success). Use before stage dogfood or agent attach when the broker is still warming.

### Dogfood WaitReady soft preflight

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

Effective budget is `min(WaitReady, parent ctx remaining)` via `context.WithTimeout`. Report top-level `wait_ready_ms` is always emitted (configured budget in ms; `0` = off). Actual wait wall time is `wait_ready_elapsed_ms` (always emitted; `0` when step skipped/absent). Outcome is on the `wait_ready` step (`PASS` / `SKIP` / `FAIL`).

### Operator status (`mesh status`)

One-shot operator snapshot without the full dogfood suite:

```bash
iomesh mesh status [--json] [--endpoint url] [--config path]
# Human: StatusLine + version + endpoint/tenant/org/workspace + plane flags + ua + health/ready + latencies
# --json: structured object with the same fields (health/ready fail-open — exit 0 even on probe err)
```

Builds the client like dogfood/wait. Prints `StatusLine` config summary (includes `version=` when `iomesh.SetProductVersion` is set from main) plus optional one-shot `Health` / `Ready` probes (errors shown as `err` + message; never fail the command). Probe wall times are always emitted as `health_ms` / `ready_ms` (`0` when mesh disabled / probes skipped).

JSON/text fields beyond StatusLine identity:

| Field | Source |
|-------|--------|
| `version` | Binary version (`main.version` / `iomesh version`) |
| `policy_mode` | `[iomesh] policy_mode` (default `off`) |
| `context_plane` | `[iomesh] context_plane` |
| `catalog_plane` | `[iomesh] catalog_plane` |
| `include_lineage` | `[iomesh] include_lineage` |
| `emit_dept` | `[iomesh] emit_dept_streams` |
| `user_agent` | package mesh HTTP User-Agent |
| `health` / `ready` | one-shot probe (`ok` \| `err` \| `skipped`) |
| `health_ms` / `ready_ms` | probe latency ms (**always emitted**, `0` when skipped/disabled) |

| Step | Request | Soft (default) | Strict (`--strict`) |
|------|---------|----------------|---------------------|
| enabled | config | SKIP if disabled | same |
| health | `GET /health` | **FAIL** if down | **FAIL** |
| wait_ready | poll Ready (optional Health) | only when `--wait-ready` > 0; timeout → **SKIP** | timeout → **FAIL** |
| ready | `GET /ready` or `/readyz` | SKIP if 404 | FAIL if missing/error |
| context | `POST /v1/context/query` | SKIP if empty (fail-open) | FAIL if empty |
| emit | `POST /v1/streams/dept/publish` (`dept.agent.dogfood`) | SKIP on error | FAIL on error |
| llm_meter | `POST /v1/streams/dept/publish` (`dept.agent.llm_call` probe) | SKIP on error (`--skip-emit` / streams off) | FAIL on error |
| pub | `POST /v1/pub` (ephemeral; optional `--pub-subject`) | SKIP if unset (`pub probe unset`); soft SKIP on error | FAIL on error when subject set |
| policy | `POST /v1/policy/evaluate` | SKIP if mode off / 404 / fail-open; top-level `policy_mode` / `policy_source` / `policy_allow` evidence | FAIL if mode on and evaluate soft-fails |
| catalog | broker + portal list | SKIP if plane off / fail-open | FAIL if fail-open |
| streams | `GET /v1/streams` (`ListStreams`) | SKIP on error (`--skip-streams` forces SKIP); empty list is PASS `n=0` | FAIL on error |
| consumer | `POST .../consumers` create; optional fetch | SKIP if stream+name unset; soft SKIP on create/fetch error; 409 = success; no ack | FAIL on create/fetch error when stream+name set |
| kv | `GET /v1/kv/{bucket}` (`KVListKeys`); optional `POST` create when `--kv-ensure` | SKIP if `--kv-bucket` unset (`kv probe unset`); soft SKIP on list error; ensure create is always soft fail-open | FAIL on list error when bucket set (ensure create never alone fails) |
| memory_ingest | `POST /v1/streams/MEMORY_INGEST/publish` | SKIP on error (`--skip-memory` forces SKIP) | FAIL on error |
| memory_recall | `POST /v1/streams/MEMORY_RPC/publish` | SKIP on error (`--skip-memory` forces SKIP) | FAIL on error |
| memory_retrieve | `POST /v1/memory/retrieve` (fallback `/v5`) | SKIP on error (`--skip-memory` forces SKIP) | FAIL on error |

Context requests set `include_lineage` when configured (lineage count shown on PASS detail).

### policy (evaluate evidence)

Runs when mesh is enabled. Mode from Client `[iomesh] policy_mode` / `IOMESH_POLICY_MODE` (default `off`):

- **mode off**: step **SKIP** `policy mode off`; top-level `policy_mode=off`, `policy_source=off`, `policy_allow` **omitted**
- **advisory / enforce**: `POST /v1/policy/evaluate` with action `dogfood.probe`; sets `policy_source` + `policy_allow` from `PolicyDecision`
  - `source=mesh` → **PASS** (allow or deny — both are valid evaluate evidence)
  - `source=unavailable` (404) / `fail-open` → **SKIP** soft; **FAIL** when `--strict`
- Top-level fields enable CI greps without scraping step detail

### streams (list probe)

Runs **after** `catalog` and **before** `kv` / `memory_*` whenever mesh is enabled (not gated on a plane flag). Non-destructive `ListStreams` (`GET /v1/streams`):

- Soft mode: transport/HTTP errors → **SKIP** (fail-open); `--strict` → **FAIL**
- Empty list is still **PASS** with `n=0` (valid discovery)
- `--skip-streams` forces **SKIP** with detail `skipped (--skip-streams)`; `streams_count=0`
- **PASS detail**: `n=N` plus truncated `names=…` (up to 8 names)
- Top-level report field `streams_count` always emitted (CI evidence without scraping step detail)

### consumer (soft create + optional fetch/delete probe)

Optional best-effort durable consumer create after `streams` and before `kv` / `memory_*`. Non-destructive relative to ack (never acks/nacks):

| Flag / option | Maps to | Default |
|---------------|---------|---------|
| `--consumer-stream S` | `DogfoodOptions.ConsumerStream` | empty with name empty → step **SKIP** `consumer probe unset` |
| `--consumer-name C` | `DogfoodOptions.ConsumerName` | both stream+name required together |
| `--consumer-filter F` | `DogfoodOptions.ConsumerFilter` | empty (optional filter_subject) |
| `--consumer-fetch` | `DogfoodOptions.ConsumerFetch` | false; when true, soft fetch after successful create |
| `--consumer-delete` | `DogfoodOptions.ConsumerDelete` | false; when true, best-effort `DeleteConsumer` cleanup after successful create |

- Both stream and name empty → **SKIP** `consumer probe unset` (no network)
- Only one of stream/name set → **SKIP** `consumer probe needs stream and name`
- When both set: `CreateConsumer` (`POST /v1/streams/{stream}/consumers`); **201** or idempotent **409** = success (`consumer_ok=true`)
- Soft mode: create transport/HTTP errors → **SKIP** (`consumer soft-fail: …`); `--strict` → **FAIL**
- **`--consumer-fetch`**: after create success, `ConsumerFetch` batch=1 max_wait=500ms; empty message list is still **PASS**; fetch errors soft **SKIP** (or **FAIL** when strict); never ack
- **`--consumer-delete`**: after successful create (and optional fetch), best-effort `DeleteConsumer` (`DELETE /v1/streams/{stream}/consumers/{name}`); delete only runs when create succeeded. Soft mode: delete errors → **SKIP** (`delete soft-fail: …`) with `consumer_delete_ok=false`; `--strict` → **FAIL**. `consumer_delete_probed=true` only when the delete attempt ran
- **PASS detail**: `stream=S name=C create=ok [filter=F] [fetch=n=N] [delete=ok]`
- Top-level `consumer_probed`, `consumer_ok`, `consumer_fetch_ok`, `consumer_delete_probed`, `consumer_delete_ok` always emitted (`consumer_probed` true only when both set and create attempt ran; `consumer_fetch_ok` true only when fetch requested and succeeded; delete flags false when flag off / create failed / probe unset)
- Top-level `consumer_stream` / `consumer_name` / `consumer_filter` set when both stream+name provided for the probe (**even if create fails**); omitted when unset/partial

### kv (soft list-keys probe)

Optional non-destructive `KVListKeys` after `streams` / `consumer` and before `memory_*`:

| Flag / option | Maps to | Default |
|---------------|---------|---------|
| `--kv-bucket NAME` | `DogfoodOptions.KVBucket` | empty → step **SKIP** `kv probe unset` (no network) |
| `--kv-ensure` | `DogfoodOptions.KVEnsure` | false; only meaningful with `--kv-bucket` |

- When bucket set: `GET /v1/kv/{bucket}` list-keys only (never put/delete via dogfood)
- **`--kv-ensure`**: best-effort `KVCreateBucket` (`POST /v1/kv/{bucket}`) before list; idempotent 409 = success. Soft fail-open: create errors never fail the step alone (even under `--strict`); list still runs. Step detail notes `ensure=ok|skip|soft-fail`
- Soft mode (list): transport/HTTP errors → **SKIP** (`kv soft-fail: …`); `--strict` → **FAIL**
- Empty key list is still **PASS** with `n=0`
- **PASS detail**: `bucket=NAME n=N ensure=…`
- Top-level `kv_bucket` omitted when unset; `kv_key_count` and `kv_ensured` always emitted (`kv_ensured` true only if ensure create attempted and succeeded)

### emit + llm_meter (dept streams / remote metering)

When `emit_dept_streams` is on (default) and not `--skip-emit`:

1. **emit** — `dept.agent.dogfood` probe (generic stage event)
2. **llm_meter** — `dept.agent.llm_call` zero-token probe (same wire as live `RecordLLMCall` for platform remote metering dashboards)

### pub (soft ephemeral probe)

Optional non-destructive `Pub` after emit/llm_meter (independent of dept emit flags):

| Flag / option | Maps to | Default |
|---------------|---------|---------|
| `--pub-subject SUBJECT` | `DogfoodOptions.PubSubject` | empty → step **SKIP** `pub probe unset` (no network) |

- When subject set: `POST /v1/pub` with fixed payload `{"source":"iomesh-tui-dogfood"}` (raw string wire, same as CLI `mesh pub`)
- Soft mode: transport/HTTP errors → **SKIP** (`pub soft-fail: …`); `--strict` → **FAIL**
- **PASS detail**: `POST /v1/pub subject=… bytes=N`
- Top-level `pub_probed` and `pub_ok` always emitted (`pub_probed` true only when subject set and attempt ran; `pub_ok` true only on success)


Both set `session_id={tenant}.mesh-dogfood` for correlation with memory_*. PASS detail appends `org=` / `workspace=` when Client OrgID/WorkspaceID are set (headers `X-IOMesh-Org` / `X-IOMesh-Workspace` on the POST). Soft: transport/HTTP errors → **SKIP**; `--strict` → **FAIL**.

### memory_ingest (dual-write probe)

Included **by default** when mesh is enabled (not gated on agent `[memory].dual_write`). Calls the same lean path as Phase 2 dual-write (`PublishMemoryIngest`):

- Subject: `{tenant}.memory.ingest.turn`
- Envelope: `type=memory_ingest`, `role=tool`, `content=iomesh-tui dual-write dogfood`, `event_time=now`, `session_seq=1`, `session_id={tenant}.mesh-dogfood` (or `mesh-dogfood` when tenant unset)
- Soft mode: publish/transport errors → **SKIP** (fail-open); `--strict` → **FAIL**
- Offline / mesh disabled: whole report is SKIP (no memory step)
- **PASS detail** includes stream, subject, and seq when available. When Client `[iomesh] org` / `workspace` (`OrgID` / `WorkspaceID`) are set, detail also appends `org=…` and/or `workspace=…` as operator-visible evidence that dual-write publish used those headers (`X-IOMesh-Org` / `X-IOMesh-Workspace`). Empty values are omitted (no `org=` token). Detail always includes temporal correlation from the envelope sent: `session_seq=N` and `session_id=…` when non-empty. Detail **always** ends with `dual_write=true` or `dual_write=false` from Client `[memory].dual_write` / `IOMESH_MEMORY_DUAL_WRITE` (report evidence only — does not gate the probe), so human-readable reports and step logs show mode without relying only on top-level JSON.

### memory_recall (async MEMORY_RPC probe — )

Runs **after** `memory_ingest` (same `--skip-memory` gate). Calls `PublishMemoryRecall` (SDK-parity fire-and-forget; **not** sync hits):

- Stream: `MEMORY_RPC` · subject: `{tenant}.memory.retrieve.request`
- Payload: `type=memory_recall`, `tenant_id`, `query=iomesh-tui dual-write dogfood`, `limit=8`, **`session_id` identical to ingest** (`{tenant}.mesh-dogfood`)
- Soft: transport errors → **SKIP**; `--strict` → **FAIL**
- **PASS detail**: stream, subject, seq, optional `org=`/`workspace=`, `session_id=…`, `dual_write=true|false`

### memory_retrieve (sync HTTP — Phase 3 /  + sidecar )

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
# legacy / platform-compatible alias: MEMORY_SIDECAR_URL=…

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
| `catalog_source` | string | Last catalog probe source (`mesh` \| `portal` \| `fail-open` \| `off`); omitted when empty (mesh disabled before catalog step) |
| `catalog_count` | int | Product count from last `ListCatalog` (**always emitted**, `0` when none/off). Top-level CI evidence — no step-detail scrape |
| `context_chars` | int | `len(FormatContextSnippet)` from last context probe (**always emitted**, `0` when skip/off/empty) |
| `context_lineage_count` | int | `len(res.Lineage)` from last `QueryContext` (**always emitted**, `0` when skip/off/empty) |
| `streams_count` | int | `len(ListStreams)` from last streams probe (**always emitted**, `0` on skip/error/disabled) |
| `streams_names` | string[] | Short sample of stream names from last `ListStreams` (max 8; **always emitted** as JSON array, `[]` on skip/error/disabled). Full count stays in `streams_count` |
| `kv_bucket` | string | Soft kv probe bucket (`DogfoodOptions.KVBucket` / `--kv-bucket`); **omitted** when empty (probe unset) |
| `kv_key_count` | int | `len(KVListKeys)` from last kv probe (**always emitted**, `0` on skip/error/unset) |
| `kv_ensured` | bool | True only when `--kv-ensure` create was attempted and succeeded (**always emitted**, `false` when unset/skip/soft-fail) |
| `pub_probed` | bool | True when `--pub-subject` was set and a Pub attempt ran (**always emitted**, `false` when unset) |
| `pub_ok` | bool | True when soft pub probe succeeded (**always emitted**, `false` when unset/skip/fail) |
| `consumer_stream` | string | Probe stream when both stream+name configured (**omitted** when unset/partial; set even if create fails) |
| `consumer_name` | string | Probe consumer name when both stream+name configured (**omitted** when unset/partial) |
| `consumer_filter` | string | Optional filter_subject when set with stream+name (**omitted** when empty) |
| `consumer_probed` | bool | True when `--consumer-stream` + `--consumer-name` set and create attempt ran (**always emitted**, `false` when unset) |
| `consumer_ok` | bool | True when soft consumer create succeeded (201 or 409) (**always emitted**, `false` when unset/skip/fail) |
| `consumer_fetch_ok` | bool | True when optional soft fetch ran without error (**always emitted**, `false` when not requested/fail/unset) |
| `consumer_delete_probed` | bool | True when `--consumer-delete` set, create succeeded, and `DeleteConsumer` attempt ran (**always emitted**, `false` when flag off / create failed / unset) |
| `consumer_delete_ok` | bool | True when soft `DeleteConsumer` returned nil (**always emitted**, `false` when not requested / not attempted / error) |
| `wait_ready_ms` | int | Configured WaitReady budget in ms (**always emitted**, `0` = off / no preflight). Outcome on `wait_ready` step detail |
| `wait_ready_elapsed_ms` | int | Wait_ready step latency in ms (**always emitted**, `0` when step skipped/absent / mesh disabled). Distinct from `wait_ready_ms` budget |
| `health_ms` | int | Health step latency in ms (**always emitted**, `0` when step skipped/absent / mesh disabled) |
| `ready_ms` | int | Ready step latency in ms (**always emitted**, `0` when step skipped/absent / mesh disabled) |
| `context_ms` | int | Context step latency in ms (**always emitted**, `0` when step skipped/absent / mesh disabled) |
| `streams_ms` | int | Streams step latency in ms (**always emitted**, `0` when step skipped/absent / mesh disabled) |
| `catalog_ms` | int | Catalog step latency in ms (**always emitted**, `0` when step skipped/absent / mesh disabled) |
| `emit_ms` | int | Emit step latency in ms (**always emitted**, `0` when step skipped/absent / mesh disabled) |
| `llm_meter_ms` | int | LLM meter step latency in ms (**always emitted**, `0` when step skipped/absent / mesh disabled) |
| `pub_ms` | int | Soft pub step latency in ms (**always emitted**, `0` when step skipped/absent / mesh disabled) |
| `policy_ms` | int | Policy step latency in ms (**always emitted**, `0` when step skipped/absent / mesh disabled) |
| `consumer_ms` | int | Consumer step latency in ms (**always emitted**, `0` when step skipped/absent / mesh disabled) |
| `kv_ms` | int | KV step latency in ms (**always emitted**, `0` when step skipped/absent / mesh disabled) |
| `memory_ingest_ms` | int | Memory ingest step latency in ms (**always emitted**, `0` when step skipped/absent / mesh disabled) |
| `memory_recall_ms` | int | Memory recall step latency in ms (**always emitted**, `0` when step skipped/absent / mesh disabled) |
| `memory_retrieve_ms` | int | Memory retrieve step latency in ms (**always emitted**, `0` when step skipped/absent / mesh disabled) |
| `duration_ms` | int | Total wall-clock duration of the dogfood run (`Finished−Started`) in ms (**always emitted**, `>= 0`) |
| `steps_pass` | int | Count of PASS steps (**always emitted**, `0` when none). Top-level CI evidence without scraping `summary` or `steps` |
| `steps_fail` | int | Count of FAIL steps (**always emitted**, `0` when none) |
| `steps_skip` | int | Count of SKIP steps (**always emitted**, `0` when none; mesh-disabled early return emits `1` for the single `enabled` SKIP) |
| `policy_mode` | string | Configured policy mode (`off` \| `advisory` \| `enforce`; **always emitted**, default `off`) |
| `policy_source` | string | Last policy probe source (`mesh` \| `fail-open` \| `unavailable` \| `off`); `off` when mode off; omitted when mesh disabled before policy step |
| `policy_allow` | bool | Evaluate decision when policy ran; **omitted** when mode off / skipped without evaluate |
| `memory_endpoint` | string | Optional memory sidecar base (`[memory].endpoint` / `IOMESH_MEMORY_ENDPOINT`); omitted when empty (retrieve uses mesh `endpoint`) |
| `version` | string | CLI/binary version from `DogfoodOptions.Version`, else package `ProductVersion()` (**always emitted**, `""` when unset). CLI wires package `version` |
| `user_agent` | string | Package mesh HTTP User-Agent (`iomesh-tui/<version>` via `iomesh.UserAgent`); always set for CI evidence — not scraped from server |
| `strict` | bool | `--strict` |
| `ok` | bool | no FAIL steps |
| `summary` | string | e.g. `PASS (pass=N skip=M)` |
| `result` | string | `PASS` \| `FAIL` \| `SKIP` (summary prefix) |
| `started` / `finished` | RFC3339 | probe window |
| `steps` | array | `{name,status,detail?,latency?}` |

`org` / `workspace` are structured multi-tenant evidence for operators and multi-tenant CI: parse top-level JSON without scraping step detail. Detail strings on `memory_ingest` (and related memory steps) may also include `org=` / `workspace=` when set.

`dual_write` is structured dual-write **mode** evidence for CI: parse top-level JSON instead of grepping detail strings. The same mode is also always present on the `memory_ingest` PASS detail string as `dual_write=true|false` for human logs. The CLI wires `cfg.Memory.DualWrite` into `iomesh.Config.DualWrite` when running `mesh dogfood`.

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
iomesh mesh dogfood --wait-ready 10s --wait-interval 500ms   # soft ready preflight
iomesh mesh dogfood --endpoint "$IOMESH_ENDPOINT" --tenant acme
iomesh mesh dogfood --memory-endpoint "$IOMESH_MEMORY_ENDPOINT"
iomesh mesh dogfood --skip-context --skip-emit --skip-memory --skip-streams   # health-only-ish
iomesh mesh dogfood --kv-bucket config   # soft KV list-keys probe + kv_bucket / kv_key_count / kv_ensured evidence
iomesh mesh dogfood --kv-bucket config --kv-ensure   # best-effort create bucket before list (soft fail-open)
iomesh mesh dogfood --pub-subject dept.agent.ping   # soft ephemeral Pub + pub_probed / pub_ok evidence
iomesh mesh catalog              # broker then portal paths
iomesh mesh streams [--name] [--json] [--delete --yes]  # lean list/get/delete (delete destructive); dogfood probes list + streams_names
iomesh mesh kv --bucket NAME --list|--get|--put|--delete|--create-bucket  # put/delete/create-bucket require --yes
iomesh mesh pub --subject S --payload STR|--payload-file F --yes  # ephemeral POST /v1/pub
iomesh mesh consumer create --stream S --name C [--filter F] --yes  # durable pull consumer (409 idempotent)
iomesh mesh consumer fetch --stream S --name C [--batch N] --yes    # long-poll fetch (default batch 1, 2s)
iomesh mesh consumer ack  --stream S --name C --seq N [--seq N...] --yes  # ack sequences
iomesh mesh consumer nack --stream S --name C --seq N [--seq N...] --yes  # nack sequences
iomesh mesh consumer delete --stream S --name C --yes               # DELETE durable consumer (204/2xx)
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
