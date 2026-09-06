# Deeper I/O Mesh (lineage · policy · metering)

Extends the offline-first mesh client beyond health/context/emit dogfood.

**Org event streams / heartbeats (docs framing):** mesh streams and `dept.*` publishes are organizational **heartbeats / pulses** on the org pulse plane — signed work events agents and operators share — not host/APM MELT. Public lexicon = **heartbeat / pulse** only. TUI remains a lean local-edge client (publish/pull · local-primary memory); not the hosted multi-tenant control plane. See [mesh-dogfood.md org-pulse edge framing (s785)](mesh-dogfood.md#org-pulse-edge-framing-s785-pin).

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
- `iomesh mesh usage --json` — `UsagePrint` always-emit indented JSON (stage scrapers / CI)
- After agent LLM calls, totals accumulate; emit still goes to `dept.agent.llm_call` when mesh is enabled
- **Not** a remote multi-tenant dashboard UI — that lives on the platform. This CLI surface is operator-local cost telemetry only.

**s738 usage print always-emit** (mold CatalogPrint s735 + PubPrint s732 + KVPutPrint s729; peer aion s737 residual): `iomesh mesh usage --json` marshals `UsagePrint` (via `FormatUsageJSON` → `NewUsagePrint`) always-emitting `{started,as_of,calls,errors,tokens,est_usd,by_model[]}` with nested `ModelUsagePrint` rows (`model,calls,errors,prompt_tokens,completion_tokens,total_tokens,est_usd,duration_ms`). Zero `time.Time` maps to `""` (never `"0001-01-01T00:00:00Z"`); `by_model` is always `[]` not null; empty/`0` honest. Wire `UsageSnapshot` / `ModelUsage` stay as in-process rollup (`time.Time` fields). Text `FormatUsage` unchanged. Call sites keep `FormatUsageJSON(UsageSnapshot)`.

**Honesty:** Beta · offline unit ≠ live APPLY · dual_write default OFF · not full mesh RBAC GA · empty/`0`/`[]` honest · zero time → empty string · **local process meter ≠ remote dashboard** · DTO ≠ invent usage/meter success · peer aion s737 residual · no invent GA.

**s756 (Beta · completeness pin):** mutate/print JSON **complete** — usage `UsagePrint` (s738) + pub `PubPrint` (s732) + kv put/delete `KVPutPrint`/`KVDeletePrint` (s729) locked by docs + unit tests (always-emit keys: usage `{started,as_of,calls,errors,tokens,est_usd,by_model}` with zero-time `""` and `by_model []`; pub `{ok,subject,bytes}` no payload; put `{ok,bucket,key,revision}`; delete `{ok,bucket,key}`). Completeness pin **s756** · peer aion **s755**. Completeness pin **does not** invent new DTO fields · **does not** re-claim s729/s732/s738 product bodies · DTO ≠ invent usage/pub/kv success · local process meter ≠ remote dashboard · ephemeral pub ≠ durable stream publish · s714 ≠ mutate residual · dual_write OFF · not full mesh RBAC GA · offline unit ≠ live APPLY.

```bash
iomesh mesh usage              # text table (process lifetime; often empty in fresh CLI)
iomesh mesh usage --json       # UsagePrint always-emit (s738; completeness pin s756)
```

## Remote metering path (platform dashboards)

When `[iomesh]` is enabled and `emit_dept_streams = true` (default):

1. Each LLM call → local `UsageMeter` **and** `POST /v1/streams/dept/publish` (subject = `dept.agent.llm_call`, base64 JSON envelope — same wire as [iomesh-client-sdk-go](https://github.com/iome-sh/iomesh-client-sdk-go) `EmitLLMCall` / Python SDK peer)
2. Request headers: `X-IOMesh-Org` / `X-IOMesh-Workspace` / `X-IOMesh-Department` when `[iomesh] org` / `workspace` / `department` are set (PlanGate / multi-tenant attribution)
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
# department = "eng"   # X-IOMesh-Department on mesh HTTP auth + MCP inject
```

Env: `IOMESH_INCLUDE_LINEAGE`, `IOMESH_POLICY_MODE`, `IOMESH_ORG`, `IOMESH_WORKSPACE`, `IOMESH_DEPARTMENT`.

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

Wire `DataProduct` stays lean (`omitempty` on optional fields). CLI print surfaces always-emit for scrapers (s735 CatalogPrint list + s744 CatalogProductPrint detail; mold PubPrint s732 + StreamMessagesPrint s720 + KVKeysPrint s714; peer aion s734 / s743 residual):

| Surface | Behaviour |
|---------|-----------|
| CLI `iomesh mesh catalog [--query q]` | Operator table (`source=mesh\|portal\|fail-open\|off`) |
| CLI `iomesh mesh catalog --json` | `CatalogPrint` always-emit `{source,detail,query,count,products[]}` (s735); nested rows are `DataProductPrint` |
| CLI `iomesh mesh catalog --id ID` | Text `FormatProductDetail` for one product via `GetCatalogProduct` |
| CLI `iomesh mesh catalog --id ID --json` | `CatalogProductPrint` always-emit `{source,detail,id,found,product}` (s744); nested `product` is `DataProductPrint` |
| `DataProductPrint` / nested product JSON | Always emits `id` / `name` / `title` / `description` / `subject` / `layer` / `status` / `department` / `subjects` / `lineage` (empty string / `[]` honest; never null arrays) |
| TUI `/catalog [query]` | Text table (same as non-JSON CLI list) |
| Agent `list_mesh_catalog` / `get_mesh_catalog_product` / `mesh_status` | Read-only |
| `inject_catalog = true` | Per-turn `<iomesh-catalog>` system block (opt-in) |
| Dogfood **catalog** step | PASS for mesh **or** portal; soft-skip on 404 |
| Dogfood `--json` | Machine-readable report for stage CI |

**s735 catalog print always-emit** (mold PubPrint s732 + StreamMessagesPrint s720 + KVKeysPrint s714; peer aion s734 residual): `iomesh mesh catalog [--query q] [--json]` prints `CatalogPrint` always-emitting `{source,detail,query,count,products}` with nested `DataProductPrint` rows (empty/`0`/`[]` honest; subjects/lineage never null). Text `FormatCatalog` unchanged. Wire `DataProduct` stays lean omitempty; `CatalogResult` stays untagged. Exit 1 when `Source=="off"` unchanged. **Beta catalog** · offline unit ≠ live APPLY · dual_write default OFF · fail-open source honest · not full mesh RBAC GA · portal federation not invent GA · DTO ≠ invent catalog/product success · wire omitempty ≠ print always-emit.

**s744 catalog product detail print always-emit** (mold CatalogPrint s735 + PubPrint s732; peer aion s743 residual): `iomesh mesh catalog --id ID [--json]` fetches one product via `GetCatalogProduct` (portal detail routes, then list filter fallback). `--json` prints `CatalogProductPrint` always-emitting `{source,detail,id,found,product}` with nested `DataProductPrint` (empty/`0`/`[]`/`false` honest; subjects/lineage never null). `found=false` when product missing / fail-open not-found / off — nested product is empty fields + `[]` arrays (no invent). Text path uses `FormatProductDetail` unchanged. List path (`--id` omitted) stays s735 CatalogPrint. Exit 1 when `Source=="off"`; fail-open not-found keeps exit 0 so scrapers see `found=false` without treating operator disable as success. Wire `DataProduct` / `GetCatalogProduct` tags unchanged. **Beta catalog** · offline unit ≠ live APPLY · dual_write default OFF · fail-open source honest · not full mesh RBAC GA · portal federation not invent GA · DTO ≠ invent catalog/product success · s735 list ≠ product detail residual · no invent GA.

**s753 (Beta · completeness pin):** catalog print JSON **complete** — list `CatalogPrint` (s735) + product detail `CatalogProductPrint` (s744) locked by docs + unit tests (always-emit keys: list `{source,detail,query,count,products}`; detail `{source,detail,id,found,product}`; nested `DataProductPrint` subjects/lineage `[]` not null). Completeness pin **s753** · peer aion **s752**. Completeness pin **does not** invent new DTO fields · **does not** re-claim s735/s744 product bodies · DTO ≠ invent catalog/product success · `found=false` honest · s735 list ≠ product detail · dual_write OFF · not full mesh RBAC GA · offline unit ≠ live APPLY · Beta catalog · portal federation not invent GA.

```bash
iomesh mesh catalog
iomesh mesh catalog --query operational
iomesh mesh catalog --json                    # CatalogPrint always-emit (s735; completeness pin s753)
iomesh mesh catalog --query ops --json        # query echoed; count + products[]
iomesh mesh catalog --id ops-incidents        # FormatProductDetail text
iomesh mesh catalog --id ops-incidents --json # CatalogProductPrint always-emit (s744; completeness pin s753)
```

## Stream discovery (operator list/get/delete/messages/create)

Operator discovery over **org event streams / heartbeats** (help-blurb level framing — list/get/messages/delete wire unchanged; create is console-default mutate). Lean client surface (no SDK dependency; wire parity with [iomesh-client-sdk-go](https://github.com/iome-sh/iomesh-client-sdk-go) `StreamInfo` / message list intent, or [Python SDK peer](https://github.com/iome-sh/iomesh-client-sdk-python)):

| Method | HTTP | Notes |
|--------|------|-------|
| `ListStreams` | `GET /v1/streams` | Accepts JSON array or `{"streams":[...]}`; **explicit errors** (not fail-open empty) |
| `GetStream(name)` | `GET /v1/streams/{name}` | Path-escaped name; empty name / 404 → error |
| `CreateStream(cfg)` | `POST /v1/streams` | Body `{name,subjects,retention,max_age_sec,max_msgs,description?}` — **no `retention_tier`**; 201 decodes `StreamInfo`; **409 Conflict = success** (idempotent: `GetStream` or `{Name}`); empty name / no subjects / other non-2xx → error. Defaults: `DefaultOperationalEventsCreate(tenant)` → `OPERATIONAL_EVENTS` + `dept.{tenant}.events.github` (empty tenant → `dept.engineering.events.github`). Create ≠ PULSE. Do not invent unpaid 403. |
| `DeleteStream(name)` | `DELETE /v1/streams/{name}` | Path-escaped name; 2xx/204 success; empty name / non-2xx → error |
| `ListStreamMessages(name, opts)` | `GET /v1/streams/{name}/messages` | Query `from_seq` / `to_seq` / `limit`; path-escaped name; empty name / non-2xx (incl. 403 replay gate) → error; base64 payload decoded |

Wire `StreamInfo` stays lean (`omitempty` on optional knobs including `retention_tier`). CLI print surfaces always-emit for scrapers (s699 retention knobs + s702 `retention_tier`):

| Surface | Behaviour |
|---------|-----------|
| `FormatStreamDetail` / get text | Always prints `description`, `retention`, `retention_tier`, `partitions`, `max_msgs`, `max_age_sec`, … (empty/`0` when unset) |
| `StreamInfoPrint` / get+list `--json` | Same knobs without omitempty gaps; list path maps via `NewStreamInfoPrint` |
| `FormatStreams` table | Columns include MAX_MSGS, MAX_AGE, RETENTION, **TIER** (empty when broker omits). Empty list appends inbox CTA (`StreamsInboxNextStepLines`: first durable event · mesh pub ≠ `/dashboard`) |
| `StreamMessagesPrint` / `--messages --json` | Envelope `{stream, from_seq, to_seq, limit, count, messages}` (not bare array; s720; 0 honest); nested `messages[]` are `StreamMessagePrint` (s723); completeness pin s759 |
| `StreamMessagePrint` / nested message JSON | Always emits `stream`, `seq`, `subject`, `partition`, `payload`, `headers`, `timestamp` (s723; empty/0/`""`/`{}` honest; completeness pin s759) |
| `FormatStreamMessagesPrint` / `--messages` text | Header includes knobs + count; table of messages (empty → `(no messages)`) |
| `StreamDeletePrint` / `--delete --json` | Always emits `{ok,name}` on success only (s726; empty name honest; completeness pin s759); FAIL stays stderr; no pull_role invent |
| `FormatStreamDelete` / `--delete` text | PASS + always-emit `name:` line (s726) |

`retention_tier` is decoded from the broker wire only (hot\|temp\|extended\|archive when present). **Never invent** tier from `max_age_sec` alone — empty string is honest when the broker omits. Beta · offline unit ≠ live APPLY · peer aion s701.

**s720 messages envelope** (mold KVKeysPrint s714 / ConsumerFetchPrint s708; peer aion s719 residual): `--messages --json` marshals `StreamMessagesPrint` always-emitting scraper keys without omitempty gaps. Wire `StreamMessage` stays lean. Empty/0 knobs are honest; does not invent message success from knobs alone. dual_write default OFF · not full mesh RBAC GA.

**s723 nested message always-emit** (mold StreamInfoPrint s699/s702 / KVEntryPrint s714; peer aion s722 residual): closes the half-gap where outer envelope always-emitted but nested `messages[]` still used lean wire omitempty. `NewStreamMessagePrint` maps each wire message so scrapers always see `stream` (`""`), `partition` (`0`), `headers` (`{}`), `timestamp` (`""` or RFC3339). Same nested type used by `ConsumerFetchPrint.Messages`. Wire `StreamMessage` stays lean. Beta · offline unit ≠ live APPLY · empty/0/`""`/`{}` honest · dual_write default OFF · not full mesh RBAC GA · does not invent message success from fields alone.

**s726 stream delete print always-emit** (mold ConsumerDeletePrint s708; peer aion s725 residual): `--delete --name N --yes [--json]` prints `StreamDeletePrint` always-emitting `{ok,name}` (empty name honest). Success path only — FAIL stays stderr. Wire `DeleteStream` stays lean (error return only). No `pull_role` invent (stream delete ≠ consumer pull-auth). Beta · offline unit ≠ live APPLY · dual_write default OFF · not full mesh RBAC GA · DTO ≠ invent delete success when HTTP failed.

**s759 (Beta · completeness pin):** streams print JSON **complete** — messages envelope `StreamMessagesPrint` (s720) + nested item `StreamMessagePrint` (s723) + delete `StreamDeletePrint` (s726) locked by docs + unit tests (always-emit keys: messages envelope `{stream,from_seq,to_seq,limit,count,messages}` with nested item `{stream,seq,subject,partition,payload,headers,timestamp}` empty/0/`""`/`{}` honest; delete `{ok,name}`). Completeness pin **s759** · peer aion **s758**. Completeness pin **does not** invent new DTO fields · **does not** re-claim s720/s723/s726 product bodies · envelope ≠ invent message success · item ≠ invent message success · DTO ≠ invent stream gone · dual_write OFF · not full mesh RBAC GA · offline unit ≠ live APPLY · wire StreamMessage lean.

```bash
iomesh mesh streams                  # table of all streams
iomesh mesh streams --name EVENTS    # multi-line detail
iomesh mesh streams --json           # JSON array (print DTO always-emit)
iomesh mesh streams --name EVENTS --json
# Message inspection (requires --name; default --limit 20; not dogfood):
iomesh mesh streams --messages --name EVENTS
iomesh mesh streams --messages --name EVENTS --from-seq 1 --to-seq 100 --limit 50
iomesh mesh streams --messages --name EVENTS --json   # StreamMessagesPrint envelope (s720; completeness pin s759)
# Create — requires --yes (incompatible with --delete / --messages); 409 = already exists:
iomesh mesh streams --create --yes
iomesh mesh streams --create --yes --name OPERATIONAL_EVENTS --subject dept.engineering.events.github
iomesh mesh streams --create --yes --json   # StreamInfoPrint (create ≠ PULSE)
# DESTRUCTIVE — requires both --name and --yes (incompatible with --messages / --create):
iomesh mesh streams --delete --name TEMP --yes
iomesh mesh streams --delete --name TEMP --yes --json   # StreamDeletePrint {ok,name} (s726; completeness pin s759)
```

Mesh disabled / empty endpoint → error `mesh disabled` (non-zero CLI exit) plus hooks-vs-catalog hint (`MeshDisabledHooksHint`; also printed on `mesh pub` / `mesh status` / `mesh wait` when disabled). Dogfood probes list only (`streams` step + `streams_count` / `streams_names`); create, delete, and message list are CLI-only. Create uses console defaults (`OPERATIONAL_EVENTS`, Temp 7d `limits`, **no** `retention_tier` on the wire). Text `--create` prints `StreamsInboxNextStepLines` (empty inbox until the first durable event from the app or console tap · mesh pub ephemeral ≠ `/dashboard` consume). Create ≠ PULSE (a listed stream with 0 messages is still empty). HITL stays OPEN. Message list does not enable broker replay flags and is not auto-probed by dogfood.

## KV (operator list/get/put/delete/create-bucket)

Lean KV surface (no SDK dependency; wire parity with SDK `KVEntry` / `Get` / `ListKeys` / `Put` / `Delete` / `CreateBucket`):

| Method | HTTP | Notes |
|--------|------|-------|
| `KVListKeys(bucket, prefix)` | `GET /v1/kv/{bucket}?prefix=` | Path-escaped bucket; optional `prefix` query; accepts `{"keys":[...]}` or bare array; **explicit errors** |
| `KVGet(bucket, key)` | `GET /v1/kv/{bucket}/{key}` | Path-escaped bucket/key; empty args / non-2xx → error; JSON `value` base64-decoded into `[]byte` |
| `KVPut(bucket, key, value)` | `PUT /v1/kv/{bucket}/{key}` | Body `{"value": base64}`; returns revision `uint64`; empty args / non-2xx → error |
| `KVDelete(bucket, key)` | `DELETE /v1/kv/{bucket}/{key}` | 2xx/204 success; empty args / non-2xx → error |
| `KVCreateBucket(name)` | `POST /v1/kv/{bucket}` | Empty body; 201 decodes `KVBucketInfo`; **409 Conflict = success** (idempotent, returns `{Name}`); empty name / other non-2xx → error |

Wire `KVBucketInfo` / `KVEntry` stay lean (`omitempty` on optional knobs). CLI print surfaces always-emit for scrapers (s714 read + s729 put/delete; peer text FormatKV s560 + StreamInfoPrint s699/s702 mold + StreamDeletePrint s726):

| Surface | Behaviour |
|---------|-----------|
| `FormatKVBucketInfo` / create-bucket text | Always prints `history`, `max_bytes`, `ttl_seconds` (blank when `*int64` nil) |
| `KVBucketInfoPrint` / create-bucket `--json` | Always emits `name`, `history` (0), `max_bytes` / `ttl_seconds` (0 when nil) — no omitempty |
| `FormatKVEntry` / get text | Always prints `created_at` (blank when zero) |
| `KVEntryPrint` / get `--json` | Always emits `bucket`, `key`, `value` (base64), `revision`, `created_at` (`""` when zero) |
| `KVKeysPrint` / list `--json` | Envelope `{bucket, prefix, count, keys}` (not bare string array) |
| `KVPutPrint` / `--put --json` | Always emits `{ok,bucket,key,revision}` on success only (s729; revision `0` honest; no value echo; no pull_role) |
| `FormatKVPut` / `--put` text | PASS + always-emit `bucket` / `key` / `revision` lines (s729) |
| `KVDeletePrint` / `--delete --json` | Always emits `{ok,bucket,key}` on success only (s729; empty honest; FAIL stays stderr) |
| `FormatKVDelete` / `--delete` text | PASS + always-emit `bucket` / `key` lines (s729) |

**s729 kv put/delete print always-emit** (mold StreamDeletePrint s726 + s714 read DTOs; peer aion s728 residual): closes s714 mutate half-gap. `--put … --yes [--json]` / `--delete … --yes [--json]` print DTOs on success only — FAIL stays stderr. Wire `KVPut` / `KVDelete` stay lean. No `pull_role` invent and no value echo on put JSON. Beta · offline unit ≠ live APPLY · empty/0 honest · dual_write default OFF · not full mesh RBAC GA · DTO ≠ invent mutate success when HTTP failed.

**s756 (Beta · completeness pin):** mutate/print JSON **complete** — includes kv put/delete `KVPutPrint`/`KVDeletePrint` (s729) with usage `UsagePrint` (s738) + pub `PubPrint` (s732). Completeness pin **s756** · peer aion **s755**. Completeness pin **does not** invent new DTO fields · **does not** re-claim s729 product body · DTO ≠ invent mutate success · s714 ≠ mutate residual · dual_write OFF · not full mesh RBAC GA · offline unit ≠ live APPLY.

Beta · offline unit ≠ live APPLY · empty/0 honest · dual_write default OFF · peer aion s713/s728 · does not invent KV success from knobs alone.

```bash
iomesh mesh kv --bucket config --list
iomesh mesh kv --bucket config --list --prefix app
iomesh mesh kv --bucket config --get app.json
iomesh mesh kv --bucket config --list --json
iomesh mesh kv --bucket config --get app.json --json
# Mutating — requires --yes:
iomesh mesh kv --bucket config --put app.json --value '{"ok":true}' --yes
iomesh mesh kv --bucket config --put app.json --value '{"ok":true}' --yes --json   # KVPutPrint {ok,bucket,key,revision} (s729; completeness pin s756)
iomesh mesh kv --bucket config --put app.json --value-file ./app.json --yes
iomesh mesh kv --bucket config --delete tmp.key --yes
iomesh mesh kv --bucket config --delete tmp.key --yes --json   # KVDeletePrint {ok,bucket,key} (s729; completeness pin s756)
iomesh mesh kv --bucket config --create-bucket --yes
iomesh mesh kv --bucket config --create-bucket --yes --json
```

`--bucket` required; exactly one of `--list` / `--get` / `--put` / `--delete` / `--create-bucket`. `--put` requires `--value` or `--value-file` and `--yes`. `--delete` and `--create-bucket` require `--yes`. Create-bucket is idempotent (409 already-exists → success). Mesh disabled → error `mesh disabled` (non-zero CLI exit). Dogfood soft-probes list-keys when `--kv-bucket` / `DogfoodOptions.KVBucket` is set; optional `--kv-ensure` best-effort creates the bucket first (soft fail-open). Put/delete/create-bucket remain CLI-only.

## Ephemeral pub (`mesh pub`)

Lean fire-and-forget publish (no stream name; not durable stream append). Wire parity with SDK `Pub` / `POST /v1/pub`:

| Method | HTTP | Notes |
|--------|------|-------|
| `Pub(subject, payload, headers)` | `POST /v1/pub` | Body `{"subject","payload" as **raw string** (not base64), `headers?`}; empty subject / non-2xx → error; mesh disabled → `mesh disabled` |

Wire `Pub` stays lean (error return only). CLI print surfaces always-emit for scrapers (mold StreamDeletePrint s726 + KVPutPrint s729; peer aion s731 residual):

| Surface | Behaviour |
|---------|-----------|
| `PubPrint` / `--json` | Always emits `{ok,subject,bytes}` on success only (s732; bytes `0` honest; empty subject honest); FAIL stays stderr; no pull_role invent; no payload echo |
| `FormatPub` / text | PASS + always-emit `subject:` / `bytes:` lines (s732) |

**s732 pub print always-emit** (mold StreamDeletePrint s726 + KVPutPrint s729; peer aion s731 residual): `--subject S --payload STR|--payload-file F --yes [--json]` prints `PubPrint` always-emitting `{ok,subject,bytes}` (empty/0 honest). Success path only — FAIL stays stderr. Wire `Pub` stays lean (error return only). No `pull_role` invent and no payload echo. Ephemeral `POST /v1/pub` ≠ durable stream publish. Beta · offline unit ≠ live APPLY · dual_write default OFF · not full mesh RBAC GA · DTO ≠ invent pub success when HTTP failed.

**s756 (Beta · completeness pin):** mutate/print JSON **complete** — includes pub `PubPrint` (s732) with usage `UsagePrint` (s738) + kv put/delete `KVPutPrint`/`KVDeletePrint` (s729). Completeness pin **s756** · peer aion **s755**. Completeness pin **does not** invent new DTO fields · **does not** re-claim s732 product body · DTO ≠ invent pub success · ephemeral pub ≠ durable stream publish · dual_write OFF · not full mesh RBAC GA · offline unit ≠ live APPLY.

```bash
iomesh mesh pub --subject dept.agent.ping --payload '{"ok":true}' --yes
iomesh mesh pub --subject dept.agent.ping --payload-file ./evt.json --yes
iomesh mesh pub --subject dept.agent.ping --payload hello --yes --json   # PubPrint {ok,subject,bytes} (s732; completeness pin s756)
```

Requires `--subject` and `--payload` or `--payload-file` and **`--yes`**. Success prints `PubPrint` always-emit text/JSON `{ok,subject,bytes}` (s732; empty/0 honest; completeness pin s756). Distinct from stream `POST /v1/streams/{name}/publish` (dept emit / memory ingest use that path).

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
| `NewConsumerFetchPrint` / `FormatConsumerFetch` | — | CLI fetch text/JSON envelope; always emits `stream` / `name` / `pull_role` / `pull_allow_suffix` / `batch` / `max_wait_ms` / `count` / `messages` (s708; empty role honest; nested `StreamMessagePrint` s723) |
| `NewConsumerAckPrint` / `FormatConsumerAck` | — | CLI ack|nack text/JSON; always emits `{ok,op,stream,name,pull_role,pull_allow_suffix,seqs,ack_floor,count}` (s711; empty role honest) |
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
iomesh mesh consumer ack  --stream EVENTS --name worker-1 --seq 1 --seq 2 --yes --json
iomesh mesh consumer nack --stream EVENTS --name worker-1 --seq 3 --yes
iomesh mesh consumer nack --stream EVENTS --name agent-1 --role agent --seq 3 --yes --json
iomesh mesh consumer delete --stream EVENTS --name worker-1 --yes
iomesh mesh consumer delete --stream EVENTS --name worker-1 --yes --json
```

Requires `--stream`, `--name`, and **`--yes`**. Create is idempotent (409 already-exists → success). Optional `--role` / `--pull-allow-suffix` (or `[memory].pull_role` / `pull_allow_suffix`) set auth headers on create (s681), fetch (s684; aion validates role on fetch), and ack/nack/delete (defense-in-depth); empty `--filter` on create gets role-aware default via `DefaultMemoryPullFilterForRole` (s681 / s678 / s687 — `memory` → `tenant.memory.>`). **s696:** create text (`FormatConsumerInfoWithAuth`) and JSON (`ConsumerInfoPrint`) always emit `pull_role` / `pull_allow_suffix` next to `filter_subject` (empty when unset) so CI scrapers see pull auth identity without omitempty gaps. **s708:** fetch text/JSON (`ConsumerFetchPrint`) always emits pull identity + knobs (`batch` / `max_wait_ms`) + `count` + `messages` (not raw `[]StreamMessage`); delete text/JSON (`ConsumerDeletePrint`) always emits `{ok,stream,name,pull_role,pull_allow_suffix}` (empty role honest). **s711:** ack|nack text/JSON (`ConsumerAckPrint`) always emits `{ok,op,stream,name,pull_role,pull_allow_suffix,seqs,ack_floor,count}` (empty role honest). Peer create s696 + fetch/delete s708 + memory-pull s705; peer aion s710 residual. **Beta** federated ACL — fail-open without role; dual_write default OFF; not full mesh RBAC GA; offline unit ≠ live APPLY; does not invent fetch/delete/ack success from identity. Fetch long-polls up to 2s. Ack/nack require at least one `--seq` (repeatable; CSV ok). Mesh disabled → error `mesh disabled` (non-zero CLI exit). Soft consumer probe in dogfood always-emits `pull_role` / `pull_allow_suffix` identity (s687).

## Packages

- `internal/iomesh/client.go` — QueryContext, lineage format, meter hook
- `internal/iomesh/policy.go` — EvaluatePolicy
- `internal/iomesh/meter.go` — UsageMeter / FormatUsage / NewUsagePrint / FormatUsageJSON (UsagePrint always-emit s738; completeness pin s756)
- `internal/iomesh/catalog.go` — ListCatalog / GetCatalogProduct / FormatCatalog / NewCatalogPrint / FormatCatalogJSON / NewCatalogProductPrint / FormatCatalogProductJSON / FormatProductDetail / CatalogSnippet (CatalogPrint s735 + CatalogProductPrint s744; completeness pin s753)
- `internal/iomesh/streams.go` — ListStreams / GetStream / CreateStream / DefaultOperationalEventsCreate / DeleteStream / FormatStreams / NewStreamInfoPrint / FormatStreamInfoJSON / FormatStreamInfoListJSON / NewStreamDeletePrint / FormatStreamDelete / FormatStreamDeleteJSON (StreamDeletePrint s726; completeness pin s759)
- `internal/iomesh/streams_messages.go` — ListStreamMessages / FormatStreamMessages / NewStreamMessagePrint / NewStreamMessagesPrint / FormatStreamMessagesPrint / FormatStreamMessagesJSON (StreamMessagesPrint s720 + StreamMessagePrint s723; completeness pin s759)
- `internal/iomesh/consumers.go` — CreateConsumer / ConsumerFetch / ConsumerAck / ConsumerNack / DeleteConsumer / FormatConsumerInfo / FormatConsumerInfoWithAuth / NewConsumerInfoPrint / FormatConsumerInfoJSON / NewConsumerFetchPrint / FormatConsumerFetch / FormatConsumerFetchJSON / NewConsumerAckPrint / FormatConsumerAck / NewConsumerDeletePrint / FormatConsumerDelete
- `internal/iomesh/kv.go` — KVGet / KVListKeys / KVPut / KVDelete / KVCreateBucket / FormatKVEntry / FormatKVKeys / FormatKVBucketInfo / NewKVBucketInfoPrint / FormatKVBucketInfoJSON / NewKVEntryPrint / FormatKVEntryJSON / NewKVKeysPrint / FormatKVKeysJSON / NewKVPutPrint / FormatKVPut / FormatKVPutJSON / NewKVDeletePrint / FormatKVDelete / FormatKVDeleteJSON (KVPutPrint/KVDeletePrint s729; completeness pin s756)
- `internal/iomesh/pub.go` — Pub / NewPubPrint / FormatPub / FormatPubJSON (ephemeral `POST /v1/pub`; PubPrint always-emit s732; completeness pin s756)
- `internal/agent` — policy before tool execute; mesh catalog tools; `EventMeshPolicy`

**s741 Format\*JSON helper completeness** (peer aion s740 residual): CLI `--json` success paths prefer package `Format*JSON` helpers over ad-hoc `json.MarshalIndent`. Helpers share mold (MarshalIndent + trailing newline + marshal-error fallback). Print DTOs already always-emit from prior serials (s714/s696/s699/s702/s720/…); this serial does **not** invent new DTO fields or re-claim product always-emit bodies — helper surface + CLI wire only. `FormatStreamInfoListJSON(nil)` emits `[]` not null. Beta · offline unit ≠ live APPLY · dual_write default OFF · empty/0/`[]` honest · not full mesh RBAC GA.

**s750 (Beta · completeness pin):** Format\*JSON helper completeness **complete** — prior always-emit Format\*JSON continuum + **s741 residual helpers** (`FormatStreamMessagesJSON` · `FormatStreamInfoJSON` · `FormatStreamInfoListJSON` · `FormatConsumerInfoJSON` · `FormatKVBucketInfoJSON` · `FormatKVEntryJSON` · `FormatKVKeysJSON`) locked by docs + unit tests (keys present · trailing newline · nil list → `[]` not null). Completeness pin **s750** · peer aion **s749**. Completeness pin **does not** invent new DTO fields · **does not** re-claim s741 product body · CLI prefer Format\*JSON · dual_write OFF · not full mesh RBAC GA · offline unit ≠ live APPLY · helper completeness ≠ invent new DTO fields.

**s753 (Beta · completeness pin):** catalog print JSON **complete** — list CatalogPrint (s735) + product CatalogProductPrint (s744) + nested DataProductPrint locked by docs + unit tests. Completeness pin **s753** · peer aion **s752**. Completeness pin **does not** invent new DTO fields · **does not** re-claim s735/s744 product bodies · DTO ≠ invent catalog/product success · `found=false` honest · dual_write OFF · not full mesh RBAC GA · offline unit ≠ live APPLY · Beta catalog.

**s756 (Beta · completeness pin):** mutate/print JSON **complete** — usage UsagePrint (s738) + pub PubPrint (s732) + kv put/delete KVPutPrint/KVDeletePrint (s729) locked by docs + unit tests. Completeness pin **s756** · peer aion **s755**. Completeness pin **does not** invent new DTO fields · **does not** re-claim s729/s732/s738 product bodies · DTO ≠ invent usage/pub/kv success · local process meter ≠ remote dashboard · ephemeral pub ≠ durable stream publish · s714 ≠ mutate residual · dual_write OFF · not full mesh RBAC GA · offline unit ≠ live APPLY.

**s759 (Beta · completeness pin):** streams print JSON **complete** — messages envelope StreamMessagesPrint (s720) + nested StreamMessagePrint (s723) + delete StreamDeletePrint (s726) locked by docs + unit tests. Completeness pin **s759** · peer aion **s758**. Completeness pin **does not** invent new DTO fields · **does not** re-claim s720/s723/s726 product bodies · envelope ≠ invent message success · item ≠ invent message success · DTO ≠ invent stream gone · dual_write OFF · not full mesh RBAC GA · offline unit ≠ live APPLY · wire StreamMessage lean.
