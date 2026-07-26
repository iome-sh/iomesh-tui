# Deeper I/O Mesh (lineage · policy · metering)

Extends the offline-first mesh client beyond health/context/emit dogfood.

## Lineage-aware context

When `include_lineage = true` (default) the client POSTs:

```http
POST /v1/context/query
{ "tenant", "workspace", "query", "limit", "include_lineage": true }
```

Response shapes accepted:

```json
{ "text": "…", "lineage": [{ "id", "product", "subject", "source", "freshness" }] }
```

or `{ "items": [ { "text", "lineage" }, … ] }`.

Injected system block:

```xml
<iomesh-context>
…text…
<iomesh-lineage>
- dp-1 · subject=dept.eng.events · source=kafka · freshness=2m
</iomesh-lineage>
</iomesh-context>
```

Fail-open: empty string on transport/HTTP/decode errors.

## Policy gates (Rego / broker evaluate)

Config / env:

| Value | Behaviour |
|-------|-----------|
| `off` (default) | no remote calls |
| `advisory` | `POST /v1/policy/evaluate`; log `EventMeshPolicy` + dept emit; **never block** |
| `enforce` | same, but **deny tool** when mesh returns `allow:false` |

Fail-open: transport errors, non-OK (except explicit 200 deny), and **404** (`source=unavailable`) never block tools.

Agent order: **mesh policy → interactive approval → execute**.

## Local metering “dashboard”

`Client` implements `router.MetricsSink` and keeps an **in-process** rollup (`UsageMeter`):

- `iomesh mesh usage` — print table for the **current process** (empty in a fresh CLI)
- `iomesh mesh usage --json` — same snapshot as indented JSON (stage scrapers / CI)
- After agent LLM calls, totals accumulate; emit still goes to `dept.agent.llm_call` when mesh is enabled
- **Not** a remote multi-tenant dashboard UI — that lives on the platform. This CLI surface is operator-local cost telemetry only.

## Remote metering path (platform dashboards)

When `[iomesh]` is enabled and `emit_dept_streams = true` (default):

1. Each LLM call → local `UsageMeter` **and** `POST /v1/streams/dept/publish` (subject = `dept.agent.llm_call`, base64 JSON envelope — same wire as [iomesh-client-sdk-go](https://github.com/iome-sh/iomesh-client-sdk-go) `EmitLLMCall`)
2. Request headers: `X-IOMesh-Org` / `X-IOMesh-Workspace` when `[iomesh] org` / `workspace` are set (PlanGate / multi-tenant attribution)
3. Envelope payload includes `tenant`, `org`, `workspace`, token counts, `est_usd`, model ids (errors redacted)

Stage smoke:

```bash
export IOMESH_ENDPOINT=…
export IOMESH_ORG=org_…
export IOMESH_WORKSPACE=ws_…
iomesh mesh dogfood --json
# steps emit + llm_meter PASS with org= / workspace= / session_id=
```

Dogfood step **`llm_meter`** publishes a zero-token probe event of the same shape so CI can prove the remote path without an LLM round-trip. Soft-SKIP on transport unless `--strict`.

## Config

```toml
[iomesh]
include_lineage = true
policy_mode = "off"   # off | advisory | enforce
emit_dept_streams = true
# org = "org_…"        # X-IOMesh-Org on dept emit + memory
# workspace = "ws_…"   # X-IOMesh-Workspace
```

Env: `IOMESH_INCLUDE_LINEAGE`, `IOMESH_POLICY_MODE`, `IOMESH_ORG`, `IOMESH_WORKSPACE`.

## Dogfood

`iomesh mesh dogfood` adds:

- **policy** when mode ≠ off (SKIP on 404/fail-open unless `--strict`)
- **llm_meter** when dept streams enabled (same soft/strict matrix as **emit**)

## Catalog composition + portal federation

When `catalog_plane = true` (default), discovery tries **broker then portal** (I/O Mesh control plane / portal edge):

| Order | Path | Source label |
|------|------|--------------|
| 1 | `GET /v1/catalog/data-products` | `mesh` |
| 2 | `GET /v1/catalog/products` | `mesh` |
| 3 | `GET /v17/portal/catalog/data-products` | `portal` |
| 4 | `GET /v16/portal/catalog/marketing/data-products` | `portal` |

Portal JSON fields (`mesh_layer`, `subject_pattern`, `sample_subjects`, `summary`) normalize into the shared product shape.

| Surface | Behaviour |
|---------|-----------|
| CLI `iomesh mesh catalog [--query q]` | Operator table (`source=mesh\|portal`) |
| TUI `/catalog [query]` | Same |
| Agent `list_mesh_catalog` / `get_mesh_catalog_product` / `mesh_status` | Read-only |
| `inject_catalog = true` | Per-turn `<iomesh-catalog>` system block (opt-in) |
| Dogfood **catalog** step | PASS for mesh **or** portal; soft-skip on 404 |
| Dogfood `--json` | Machine-readable report for stage CI |

## Stream discovery (operator list/get/delete/messages)

Lean client surface (no SDK dependency; wire parity with [iomesh-client-sdk-go](https://github.com/iome-sh/iomesh-client-sdk-go) `StreamInfo` / message list intent):

| Method | HTTP | Notes |
|--------|------|-------|
| `ListStreams` | `GET /v1/streams` | Accepts JSON array or `{"streams":[...]}`; **explicit errors** (not fail-open empty) |
| `GetStream(name)` | `GET /v1/streams/{name}` | Path-escaped name; empty name / 404 → error |
| `DeleteStream(name)` | `DELETE /v1/streams/{name}` | Path-escaped name; 2xx/204 success; empty name / non-2xx → error |
| `ListStreamMessages(name, opts)` | `GET /v1/streams/{name}/messages` | Query `from_seq` / `to_seq` / `limit`; path-escaped name; empty name / non-2xx (incl. 403 replay gate) → error; base64 payload decoded |

Wire `StreamInfo` stays lean (`omitempty` on optional knobs including `retention_tier`). CLI print surfaces always-emit for scrapers (s699 retention knobs + s702 `retention_tier`):

| Surface | Behaviour |
|---------|-----------|
| `FormatStreamDetail` / get text | Always prints `description`, `retention`, `retention_tier`, `partitions`, `max_msgs`, `max_age_sec`, … (empty/`0` when unset) |
| `StreamInfoPrint` / get+list `--json` | Same knobs without omitempty gaps; list path maps via `NewStreamInfoPrint` |
| `FormatStreams` table | Columns include MAX_MSGS, MAX_AGE, RETENTION, **TIER** (empty when broker omits) |

`retention_tier` is decoded from the broker wire only (hot\|temp\|extended\|archive when present). **Never invent** tier from `max_age_sec` alone — empty string is honest when the broker omits. Beta · offline unit ≠ live APPLY · peer aion s701.

```bash
iomesh mesh streams                  # table of all streams
iomesh mesh streams --name EVENTS    # multi-line detail
iomesh mesh streams --json           # JSON array (print DTO always-emit)
iomesh mesh streams --name EVENTS --json
# Message inspection (requires --name; default --limit 20; not dogfood):
iomesh mesh streams --messages --name EVENTS
iomesh mesh streams --messages --name EVENTS --from-seq 1 --to-seq 100 --limit 50
iomesh mesh streams --messages --name EVENTS --json
# DESTRUCTIVE — requires both --name and --yes (incompatible with --messages):
iomesh mesh streams --delete --name TEMP --yes
```

Mesh disabled / empty endpoint → error `mesh disabled` (non-zero CLI exit). Dogfood probes list only (`streams` step + `streams_count` / `streams_names`); delete and message list are CLI-only. Message list does not enable broker replay flags and is not auto-probed by dogfood.

## KV (operator list/get/put/delete/create-bucket)

Lean KV surface (no SDK dependency; wire parity with SDK `KVEntry` / `Get` / `ListKeys` / `Put` / `Delete` / `CreateBucket`):

| Method | HTTP | Notes |
|--------|------|-------|
| `KVListKeys(bucket, prefix)` | `GET /v1/kv/{bucket}?prefix=` | Path-escaped bucket; optional `prefix` query; accepts `{"keys":[...]}` or bare array; **explicit errors** |
| `KVGet(bucket, key)` | `GET /v1/kv/{bucket}/{key}` | Path-escaped bucket/key; empty args / non-2xx → error; JSON `value` base64-decoded into `[]byte` |
| `KVPut(bucket, key, value)` | `PUT /v1/kv/{bucket}/{key}` | Body `{"value": base64}`; returns revision `uint64`; empty args / non-2xx → error |
| `KVDelete(bucket, key)` | `DELETE /v1/kv/{bucket}/{key}` | 2xx/204 success; empty args / non-2xx → error |
| `KVCreateBucket(name)` | `POST /v1/kv/{bucket}` | Empty body; 201 decodes `KVBucketInfo`; **409 Conflict = success** (idempotent, returns `{Name}`); empty name / other non-2xx → error |

```bash
iomesh mesh kv --bucket config --list
iomesh mesh kv --bucket config --list --prefix app
iomesh mesh kv --bucket config --get app.json
iomesh mesh kv --bucket config --list --json
iomesh mesh kv --bucket config --get app.json --json
# Mutating — requires --yes:
iomesh mesh kv --bucket config --put app.json --value '{"ok":true}' --yes
iomesh mesh kv --bucket config --put app.json --value-file ./app.json --yes
iomesh mesh kv --bucket config --delete tmp.key --yes
iomesh mesh kv --bucket config --create-bucket --yes
iomesh mesh kv --bucket config --create-bucket --yes --json
```

`--bucket` required; exactly one of `--list` / `--get` / `--put` / `--delete` / `--create-bucket`. `--put` requires `--value` or `--value-file` and `--yes`. `--delete` and `--create-bucket` require `--yes`. Create-bucket is idempotent (409 already-exists → success). Mesh disabled → error `mesh disabled` (non-zero CLI exit). Dogfood soft-probes list-keys when `--kv-bucket` / `DogfoodOptions.KVBucket` is set; optional `--kv-ensure` best-effort creates the bucket first (soft fail-open). Put/delete/create-bucket remain CLI-only.

## Ephemeral pub (`mesh pub`)

Lean fire-and-forget publish (no stream name; not durable stream append). Wire parity with SDK `Pub` / `POST /v1/pub`:

| Method | HTTP | Notes |
|--------|------|-------|
| `Pub(subject, payload, headers)` | `POST /v1/pub` | Body `{"subject","payload" as **raw string** (not base64), `headers?`}; empty subject / non-2xx → error; mesh disabled → `mesh disabled` |

```bash
iomesh mesh pub --subject dept.agent.ping --payload '{"ok":true}' --yes
iomesh mesh pub --subject dept.agent.ping --payload-file ./evt.json --yes
iomesh mesh pub --subject dept.agent.ping --payload hello --yes --json
```

Requires `--subject` and `--payload` or `--payload-file` and **`--yes`**. Success prints `PASS mesh pub subject=… bytes=N` (or JSON `{subject,ok,bytes}`). Distinct from stream `POST /v1/streams/{name}/publish` (dept emit / memory ingest use that path).

Dogfood soft-probes the same path when `--pub-subject` / `DogfoodOptions.PubSubject` is set (fixed payload `{"source":"iomesh-tui-dogfood"}`; soft SKIP on error unless `--strict`).

## Durable pull consumers (`mesh consumer`)

Lean consumer surface (no SDK dependency; wire parity with SDK `CreateConsumer` / fetch / ack / `DeleteConsumer` intent):

| Method | HTTP | Notes |
|--------|------|-------|
| `CreateConsumer(stream, name, filter)` | `POST /v1/streams/{stream}/consumers` | Body `{name, filter_subject?}`; path-escaped stream; **201** decodes `ConsumerInfo`; **409 Conflict = success** (idempotent, returns `{Stream, Name}`); empty stream/name / other non-2xx → error |
| `ConsumerFetch(stream, name, batch, maxWait)` | `POST /v1/streams/{stream}/consumers/{name}/fetch` | Body `{batch, max_wait_ms}`; default maxWait 2s; returns `[]StreamMessage` (base64 or raw payload); empty args / batch≤0 / non-2xx → error |
| `ConsumerAck(stream, name, seqs...)` | `POST .../consumers/{name}/ack` | Body `{"seqs":[...]}`; path-escaped stream+name; returns optional `ack_floor` (0 if empty body); empty stream/name/seqs / non-2xx → error |
| `ConsumerNack(stream, name, seqs...)` | `POST .../consumers/{name}/nack` | Same shape as ack |
| `DeleteConsumer(stream, name)` | `DELETE .../consumers/{name}` | Path-escaped stream+name; **204/2xx** success; empty stream/name / non-2xx → error |
| `FormatConsumerInfo` / `FormatConsumerInfoWithAuth` | — | multi-line operator view; always emits `filter_subject` / `pull_role` / `pull_allow_suffix` (empty when unset; s696) |
| `NewConsumerInfoPrint` | — | CLI create JSON DTO with always-emit pull identity (does not pollute wire `ConsumerInfo`) |
| `NewConsumerFetchPrint` / `FormatConsumerFetch` | — | CLI fetch text/JSON envelope; always emits `stream` / `name` / `pull_role` / `pull_allow_suffix` / `batch` / `max_wait_ms` / `count` / `messages` (s708; empty role honest) |
| `NewConsumerDeletePrint` / `FormatConsumerDelete` | — | CLI delete text/JSON; always emits `{ok,stream,name,pull_role,pull_allow_suffix}` (s708) |

```bash
iomesh mesh consumer create --stream EVENTS --name worker-1 --yes
iomesh mesh consumer create --stream EVENTS --name worker-1 --filter 'dept.events.>' --yes --json
iomesh mesh consumer create --stream EVENTS --name agent-1 --role agent --yes   # s681: X-IOMesh-Role + default filter tenant.events.>
iomesh mesh consumer create --stream EVENTS --name mem-1 --role memory --yes    # s687: default filter tenant.memory.>
iomesh mesh consumer fetch --stream EVENTS --name worker-1 --batch 1 --yes
iomesh mesh consumer fetch --stream EVENTS --name agent-1 --role agent --batch 1 --yes   # s684: role headers on fetch path
iomesh mesh consumer fetch --stream EVENTS --name worker-1 --batch 5 --yes --json
iomesh mesh consumer ack  --stream EVENTS --name worker-1 --seq 1 --seq 2 --yes
iomesh mesh consumer nack --stream EVENTS --name worker-1 --seq 3 --yes
iomesh mesh consumer delete --stream EVENTS --name worker-1 --yes
iomesh mesh consumer delete --stream EVENTS --name worker-1 --yes --json
```

Requires `--stream`, `--name`, and **`--yes`**. Create is idempotent (409 already-exists → success). Optional `--role` / `--pull-allow-suffix` (or `[memory].pull_role` / `pull_allow_suffix`) set auth headers on create (s681), fetch (s684; aion validates role on fetch), and ack/nack/delete (defense-in-depth); empty `--filter` on create gets role-aware default via `DefaultMemoryPullFilterForRole` (s681 / s678 / s687 — `memory` → `tenant.memory.>`). **s696:** create text (`FormatConsumerInfoWithAuth`) and JSON (`ConsumerInfoPrint`) always emit `pull_role` / `pull_allow_suffix` next to `filter_subject` (empty when unset) so CI scrapers see pull auth identity without omitempty gaps. **s708:** fetch text/JSON (`ConsumerFetchPrint`) always emits pull identity + knobs (`batch` / `max_wait_ms`) + `count` + `messages` (not raw `[]StreamMessage`); delete text/JSON (`ConsumerDeletePrint`) always emits `{ok,stream,name,pull_role,pull_allow_suffix}` (empty role honest). Peer create s696 + memory-pull s705; peer aion s707 gate completeness. **Beta** federated ACL — fail-open without role; dual_write default OFF; not full mesh RBAC GA; offline unit ≠ live APPLY; does not invent fetch/delete success from identity. Fetch long-polls up to 2s. Ack/nack require at least one `--seq` (repeatable; CSV ok). Mesh disabled → error `mesh disabled` (non-zero CLI exit). Soft consumer probe in dogfood always-emits `pull_role` / `pull_allow_suffix` identity (s687).

## Packages

- `internal/iomesh/client.go` — QueryContext, lineage format, meter hook
- `internal/iomesh/policy.go` — EvaluatePolicy
- `internal/iomesh/meter.go` — UsageMeter / FormatUsage
- `internal/iomesh/catalog.go` — ListCatalog / FormatCatalog / CatalogSnippet
- `internal/iomesh/streams.go` — ListStreams / GetStream / DeleteStream / FormatStreams
- `internal/iomesh/streams_messages.go` — ListStreamMessages / FormatStreamMessages
- `internal/iomesh/consumers.go` — CreateConsumer / ConsumerFetch / ConsumerAck / ConsumerNack / DeleteConsumer / FormatConsumerInfo / FormatConsumerInfoWithAuth / NewConsumerInfoPrint / NewConsumerFetchPrint / FormatConsumerFetch / NewConsumerDeletePrint / FormatConsumerDelete
- `internal/iomesh/kv.go` — KVGet / KVListKeys / KVPut / KVDelete / KVCreateBucket / FormatKVEntry / FormatKVKeys / FormatKVBucketInfo
- `internal/iomesh/pub.go` — Pub (ephemeral `POST /v1/pub`)
- `internal/agent` — policy before tool execute; mesh catalog tools; `EventMeshPolicy`
