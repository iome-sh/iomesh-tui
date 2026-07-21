# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- **Dogfood wait_ready_attempts evidence** — always emit `wait_ready_attempts` (WaitReady probe cycle count; `0` when wait budget off / step not run) in text and JSON dogfood reports so CI scrapers record how many Health/Ready loops ran without scraping step detail

## [0.47.0] — 2026-07-20

Minor release: mesh wait attempts evidence.

### Added

- **`mesh wait` attempts evidence** — always emit `attempts` (WaitReady probe attempt cycles) in text and `--json` output so CI scrapers record how many Health/Ready loops ran without re-parsing logs

## [0.46.0] — 2026-07-20

Minor release: dogfood wait_ready_result evidence.

### Added

- **Dogfood wait_ready_result evidence** — always emit `wait_ready_result` (`off`|`ok`|`err`|`skip`) in text and JSON dogfood reports so CI scrapers record wait_ready step outcome without scraping the steps array (`off` when wait budget 0 / step not run)

## [0.45.0] — 2026-07-20

Minor release: dogfood wait_ready_interval_ms and wait_require_health evidence.

### Added

- **Dogfood wait preflight knobs evidence** — always emit `wait_ready_interval_ms` (effective poll interval; `0` when wait off, default `500` when wait on and interval unset) and `wait_require_health` (configured bool) in text and JSON dogfood reports so CI scrapers record WaitReady knobs without re-parsing flags

## [0.44.0] — 2026-07-20

Minor release: mesh wait timeout_ms and interval_ms evidence.

### Added

- **`mesh wait` timeout/interval budget evidence** — always emit `timeout_ms` and `interval_ms` (configured WaitReady budget and poll interval) in text and `--json` output so CI scrapers record preflight knobs without re-parsing flags


## [0.43.0] — 2026-07-20

Minor release: mesh wait require_health in result evidence.

### Added

- **`mesh wait` require_health evidence** — always emit `require_health` (boolean) in text and `--json` output so CI scrapers can record whether Health was required without re-parsing flags


## [0.42.0] — 2026-07-20

Minor release: mesh wait elapsed_ms evidence.

### Added

- **`mesh wait` elapsed evidence** — always emit `elapsed_ms` (WaitReady wall time) on PASS/FAIL; optional `--json` `{"ok":true|false,"elapsed_ms":N[, "error":"..."]}` for CI scrapers


## [0.41.0] — 2026-07-20

Minor release: mesh status --strict exit mode.

### Added

- **`mesh status --strict`** — optional exit gate after printing JSON/text: exit `1` only when aggregate `result` is `err`; default remains fail-open (exit `0` on probe failures). Mesh disabled (`skipped`) and `partial` stay exit `0` under `--strict`


## [0.40.0] — 2026-07-20

Minor release: mesh status aggregate result.

### Added

- **`mesh status` aggregate result** — JSON/text always emit `result` (`ok` \| `err` \| `skipped` \| `partial`) from health+ready probes for operator/CI gating without scraping both fields


## [0.39.0] — 2026-07-20

Minor release: mesh status wall-clock duration.

### Added

- **`mesh status` duration** — JSON/text always emit `duration_ms` (wall-clock for the Health+Ready probe path in ms; `0` when mesh disabled or probes skipped) for operator/CI evidence


## [0.38.0] — 2026-07-20

Minor release: dogfood kv list-keys path latency.

### Added

- **Dogfood kv list latency** — top-level `kv_list_ms` (`KVListKeys` latency ms only; always emitted, `0` when kv probe unset / list not run). Distinct from `kv_ms` (full step) and `kv_ensure_ms` (ensure-create only).


## [0.37.0] — 2026-07-20

Minor release: dogfood kv ensure-path latency.

### Added

- **Dogfood kv ensure latency** — top-level `kv_ensure_ms` (`KVCreateBucket` ensure-path latency ms; always emitted, `0` when ensure off / kv probe unset / not attempted). Distinct from `kv_ms` (full step latency).


## [0.36.0] — 2026-07-20

Minor release: mesh status health/ready probe latencies.

### Added

- **`mesh status` Health/Ready latencies** — JSON/text always emit `health_ms` / `ready_ms` (probe wall time ms; `0` when mesh disabled or probes skipped) for operator/CI evidence


## [0.35.0] — 2026-07-20

Minor release: dogfood soft consumer delete probe.

### Added

- **Dogfood soft consumer delete cleanup** — optional `DogfoodOptions.ConsumerDelete` / CLI `--consumer-delete` best-effort `DeleteConsumer` after successful create (and optional fetch) in the consumer step; soft fail-open unless `--strict`; top-level `consumer_delete_probed` + `consumer_delete_ok` always emitted


## [0.34.0] — 2026-07-20

Minor release: mesh consumer delete.

### Added

- **Mesh consumer delete** — lean `DeleteConsumer` (`DELETE /v1/streams/{stream}/consumers/{name}`; 204/2xx success; path-escaped stream+name) + CLI `iomesh mesh consumer delete --stream S --name C --yes [--json]`


## [0.33.0] — 2026-07-20

Minor release: dogfood step pass/fail/skip counts.

### Added

- **Dogfood step counts** — top-level `steps_pass` / `steps_fail` / `steps_skip` (PASS/FAIL/SKIP step tallies; always emitted). Mesh-disabled early return sets `steps_skip=1`. CI can gate without scraping `summary` or the `steps` array


## [0.32.0] — 2026-07-20

Minor release: dogfood wait_ready elapsed latency.

### Added

- **Dogfood wait_ready elapsed latency** — top-level `wait_ready_elapsed_ms` (wait_ready step latency ms; always emitted, `0` when skipped/absent). Distinct from `wait_ready_ms` configured budget.


## [0.31.0] — 2026-07-20

Minor release: dogfood consumer/kv step latencies.

### Added

- **Dogfood consumer/kv latencies** — top-level `consumer_ms` / `kv_ms` (step latency ms; always emitted, `0` when skipped/absent)

## [0.30.0] — 2026-07-20

Minor release: dogfood llm_meter/pub/memory step latencies.

### Added

- **Dogfood llm_meter/pub/memory latencies** — top-level `llm_meter_ms` / `pub_ms` / `memory_ingest_ms` / `memory_recall_ms` / `memory_retrieve_ms` (step latency ms; always emitted, `0` when skipped/absent)

## [0.29.0] — 2026-07-19

Minor release: dogfood emit/policy/duration latencies and disabled StatusLine version.

### Added

- **Dogfood emit/policy/duration latencies** — top-level `emit_ms` / `policy_ms` (step latency ms; always emitted, `0` when skipped/absent) and `duration_ms` (wall-clock Finished−Started ms; always emitted, `>= 0`)
- **StatusLine version when mesh disabled** — `/mesh` / `StatusLine` appends `version=` when `ProductVersion` is set, including offline-first disabled clients

## [0.28.0] — 2026-07-19

Minor release: dogfood step latencies and StatusLine version.

### Added

- **Dogfood step latencies** — top-level `context_ms` / `streams_ms` / `catalog_ms` (step latency ms; always emitted, `0` when skipped/absent)
- **StatusLine product version** — `iomesh.SetProductVersion` / `ProductVersion` (wired from main like User-Agent); `StatusLine` appends `version=` when set; dogfood report `version` defaults from `ProductVersion` when `DogfoodOptions.Version` is empty

## [0.27.0] — 2026-07-19

Minor release: dogfood version and health/ready latency fields.

### Added

- **Dogfood version + probe latency** — top-level `version` (from `DogfoodOptions.Version` / CLI binary version; always emitted, empty when unset) and `health_ms` / `ready_ms` (step latency ms; always emitted, `0` when skipped/absent)

## [0.26.0] — 2026-07-19

Minor release: dogfood consumer identity and richer mesh status.

### Added

- **Dogfood consumer identity** — top-level `consumer_stream` / `consumer_name` / `consumer_filter` when both stream+name are configured for the soft consumer probe (set even if create fails; omitted when unset)
- **Richer `mesh status`** — JSON and text include binary `version`, `policy_mode`, `context_plane`, `catalog_plane`, `include_lineage`, and `emit_dept` from config

## [0.25.0] — 2026-07-19

Minor release: soft dogfood consumer create and fetch probe.

### Added

- **Dogfood soft consumer probe** — optional `DogfoodOptions.ConsumerStream` + `ConsumerName` / CLI `--consumer-stream` + `--consumer-name` best-effort `CreateConsumer` (201 or idempotent 409); optional `--consumer-filter` and `--consumer-fetch` (batch=1, max_wait 500ms, empty OK, no ack); step SKIP when unset; top-level `consumer_probed` + `consumer_ok` + `consumer_fetch_ok` always emitted

## [0.24.0] — 2026-07-19

Minor release: consumer ack and nack CLI.

### Added

- **Mesh consumer ack/nack** — lean `ConsumerAck` / `ConsumerNack` (`POST /v1/streams/{stream}/consumers/{name}/ack|nack`; body `{"seqs":[...]}`; optional `ack_floor` on response) + CLI `iomesh mesh consumer ack|nack --stream S --name C --seq N [--seq N...] --yes`

## [0.23.0] — 2026-07-19

Minor release: consumer create CLI and soft dogfood pub probe.

### Added

- **Mesh consumer create/fetch** — lean `CreateConsumer` / `ConsumerFetch` / `FormatConsumerInfo` (`POST /v1/streams/{stream}/consumers`; 201 full info, 409 idempotent name-only; fetch default batch=1 max_wait 2s) + CLI `iomesh mesh consumer create|fetch --stream S --name C --yes`
- **Dogfood soft pub probe** — optional `DogfoodOptions.PubSubject` / CLI `--pub-subject SUBJECT` ephemeral `Pub` with fixed dogfood payload after emit; step SKIP when unset; top-level `pub_probed` + `pub_ok` always emitted

## [0.22.0] — 2026-07-19

Minor release: dogfood kv-ensure and ephemeral mesh pub.

### Added

- **Dogfood `--kv-ensure`** — with `--kv-bucket`, best-effort `KVCreateBucket` before list-keys (soft fail-open; step detail `ensure=ok|skip|soft-fail`; top-level `kv_ensured` always emitted)
- **Ephemeral mesh pub** — lean `Pub` (`POST /v1/pub`; body `{subject, payload string, headers?}` SDK wire) + CLI `iomesh mesh pub --subject S --payload STR|--payload-file F --yes`

## [0.21.0] — 2026-07-19

Minor release: KV create-bucket lean client and CLI.

### Added

- **Mesh KV create-bucket** — lean `KVCreateBucket` (`POST /v1/kv/{bucket}`; empty body; 201 decodes `KVBucketInfo`; 409 Conflict treated as success) + CLI `--create-bucket --yes` (mutually exclusive with list/get/put/delete)

## [0.20.0] — 2026-07-19

Minor release: gated KV put/delete and soft dogfood kv probe.

### Added

- **Gated mesh KV put/delete** — lean `KVPut` / `KVDelete` (`PUT|DELETE /v1/kv/{bucket}/{key}`; body `{"value": base64}` on put) + CLI `--put KEY --value|--value-file --yes` / `--delete KEY --yes` (mutating ops require `--yes`; mutually exclusive with list/get)
- **Dogfood soft kv probe** — optional `DogfoodOptions.KVBucket` / CLI `--kv-bucket NAME` list-keys only (non-destructive); step SKIP when unset; top-level `kv_bucket` (omitempty) + `kv_key_count` (always)

## [0.19.0] — 2026-07-19

Minor release: dogfood policy evidence + mesh kv CLI.

### Added

- **Dogfood policy evidence** — top-level `policy_mode` (always), `policy_source`, and `policy_allow` (when evaluated) on dogfood JSON/text reports for CI without scraping step detail
- **Mesh KV read CLI** — lean `KVGet` / `KVListKeys` (`GET /v1/kv/{bucket}/{key}`, `GET /v1/kv/{bucket}?prefix=`) + `iomesh mesh kv --bucket NAME --list|--get KEY` (explicit errors; no SDK dep)

## [0.18.0] — 2026-07-19

Minor release: stream message list CLI.

### Added

- **Stream message list CLI** — lean `ListStreamMessages` (`GET /v1/streams/{name}/messages?from_seq=&to_seq=&limit=`) + `iomesh mesh streams --messages --name NAME` (default `--limit 20`; `--from-seq` / `--to-seq` / `--json`; incompatible with `--delete`; base64 payload decoded for table display)

## [0.17.0] — 2026-07-19

Minor release: streams_names dogfood sample + gated streams delete; public-surface hygiene.

### Changed

- **Public-surface hygiene** — no private ledger serials / monorepo paths in docs; CONTRIBUTING public repository policy; OPEN_SOURCE_AUDIT residual honesty

### Added

- **Dogfood `streams_names` sample** — top-level JSON/text array of up to 8 stream names from last `ListStreams` (always emitted; empty on skip/error) for CI greps without step-detail scrape
- **Gated `mesh streams --delete`** — lean `DeleteStream` (`DELETE /v1/streams/{name}`) + CLI requires `--name` and `--yes` (destructive; explicit errors)

## [0.16.0] — 2026-07-19

Minor release: dogfood streams list evidence.

### Added

- **Dogfood streams list evidence** — soft `streams` step (`ListStreams` / `GET /v1/streams`) after catalog; top-level `streams_count` always emitted in JSON/text; CLI `--skip-streams`

## [0.15.0] — 2026-07-19

Minor release: mesh streams list/get CLI.

### Added

- **Mesh stream discovery** — lean `ListStreams` / `GetStream` (`GET /v1/streams`, `GET /v1/streams/{name}`) + CLI `iomesh mesh streams [--name] [--json]` (explicit errors; no SDK dep)

## [0.14.0] — 2026-07-19

Minor release: dogfood WaitReady soft preflight.

### Added

- **Dogfood WaitReady soft preflight** — optional `DogfoodOptions.WaitReady` / CLI `--wait-ready` polls Ready (optional Health) before the single-shot ready step; timeout SKIP unless `--strict`; report always emits `wait_ready_ms` (0=off)

## [0.13.0] — 2026-07-19

Minor release: dogfood context evidence + mesh status CLI.

### Added

- **Dogfood context plane evidence** — `DogfoodReport.context_chars` / `context_lineage_count` top-level JSON + text fields for CI without scraping step detail
- **`iomesh mesh status [--json]`** — operator snapshot of StatusLine fields + one-shot Health/Ready (fail-open display)

## [0.12.0] — 2026-07-19

Minor release: dogfood catalog_source/count evidence.

### Added

- **Dogfood catalog evidence** — `DogfoodReport.catalog_source` / `catalog_count` top-level JSON + text fields for CI without scraping step detail

## [0.11.0] — 2026-07-19

Minor release: WaitReady + mesh wait CLI.

### Added

- **WaitReady + `iomesh mesh wait`** — poll mesh `Ready` (optional `Health`) until OK or deadline for operator preflight

## [0.10.0] — 2026-07-19

Minor release: dogfood/StatusLine user_agent evidence.

### Added

- **Dogfood / StatusLine `user_agent` evidence** — `DogfoodReport.user_agent` + text/JSON report fields and `StatusLine` `ua=` token surface package mesh HTTP User-Agent for operator/CI evidence

## [0.9.0] — 2026-07-19

Minor release: mesh HTTP User-Agent + local release-snapshot skip-sign.

### Added

- **Mesh HTTP User-Agent** — `iomesh-tui/<version>` on outbound mesh requests (`iomesh.SetUserAgent`); Health/Ready use same auth path

### Fixed

- **`make release-snapshot`** — passes `--skip=sign` so local dry-runs do not require cosign OIDC

## [0.8.0] — 2026-07-19

Minor release: keyless cosign signatures on release checksums (GitHub OIDC).

### Added

- **Keyless cosign on release checksums** — `checksums.txt.sig` + `.pem` via GitHub OIDC (no long-lived keys); RELEASING verify snippet

## [0.7.0] — 2026-07-19

Minor release: SPDX SBOM assets on GoReleaser multi-platform releases.

### Added

- **GoReleaser SPDX SBOM** — per-archive `*.sbom.spdx.json` on `v*` releases (syft); RELEASING notes optional cosign

## [0.6.1] — 2026-07-19

Patch release: dept stream emit wire fix for live metering on platform brokers.

### Fixed

- **Dept emit wire** — `Emit` / `llm_meter` / `RecordLLMCall` use `POST /v1/streams/dept/publish` with base64 JSON payload (broker stream API + SDK parity); previous path lacked `/publish`

### Changed

- Docs: Public Go SDK cross-link — operators pointed to [iomesh-client-sdk-go](https://github.com/iome-sh/iomesh-client-sdk-go) (M2 sync retrieve, M3 temporal envelope, `WithWorkspace`); TUI remains lean / no-SDK-dep ([docs/architecture/memory-mcp.md](docs/architecture/memory-mcp.md), [docs/architecture/overview.md](docs/architecture/overview.md))

## [0.6.0] — 2026-07-18

Minor release: remote multi-tenant metering emit path (org/workspace headers + dogfood `llm_meter`). Compatible with `v0.5.x` configs.

### Added

- **Remote metering emit path** — `dept.agent.llm_call` and all dept emit set `X-IOMesh-Org` / `X-IOMesh-Workspace` when configured; LLM payload includes `tenant`/`org`/`workspace` for multi-tenant platform dashboards ([docs/architecture/mesh-deeper.md](docs/architecture/mesh-deeper.md))
- **Dogfood `llm_meter` step** — zero-token `dept.agent.llm_call` probe after `emit` (same soft/strict + `--skip-emit` gate); PASS detail `org=`/`workspace=`/`session_id=`

## [0.5.0] — 2026-07-18

Minor release: GoReleaser multi-platform binaries on `v*` tags and JSON local usage export. Compatible with `v0.4.x` configs.

### Added

- **GoReleaser** — multi-platform `iomesh` binaries on `v*` tags (`.goreleaser.yaml` + `.github/workflows/release.yml`); `make release-snapshot` for local dry-run ([RELEASING.md](RELEASING.md))
- **`iomesh mesh usage --json`** — JSON usage snapshot for scrapers; documents platform remote dashboards vs local meter ([docs/architecture/mesh-deeper.md](docs/architecture/mesh-deeper.md))

## [0.4.0] — 2026-07-18

Minor release: Memory Phase 3+ (sync HTTP retrieve, agent auto-recall prefer sidecar, stage warm-plane dogfood) and full mesh memory dogfood evidence. Compatible with existing `v0.3.x` configs (new flags default off / empty).

### Added

- **Mesh dogfood `memory_ingest` step** — exercises Phase 2 dual-write via `PublishMemoryIngest` (`POST /v1/streams/MEMORY_INGEST/publish`); included by default when mesh enabled (fail-open → SKIP unless `--strict`); CLI `--skip-memory` to omit ([docs/architecture/mesh-dogfood.md](docs/architecture/mesh-dogfood.md))
- **MEMORY_INGEST dual-write org/workspace headers** — optional `[iomesh] org` / `workspace` (`IOMESH_ORG` / `MEMORY_ORG` / `IOMESH_WORKSPACE`) set `X-IOMesh-Org` + `X-IOMesh-Workspace` on `PublishMemoryIngest` (M5 entitlements parity)
- **Dogfood `memory_ingest` org/workspace evidence** — PASS detail appends `org=` / `workspace=` when Client OrgID/WorkspaceID are configured (omitted when unset)
- **Dogfood JSON `org` / `workspace` fields** — `DogfoodReport` + `FormatReportJSON` carry Client OrgID/WorkspaceID as top-level `org` / `workspace` (`omitempty`) for stage CI / multi-tenant gate parsing ([docs/architecture/mesh-dogfood.md](docs/architecture/mesh-dogfood.md))
- **Dogfood JSON `dual_write` field** — `DogfoodReport` + `FormatReportJSON` always emit top-level `dual_write` bool from Client cfg (wired from `[memory].dual_write` / `IOMESH_MEMORY_DUAL_WRITE` in `mesh dogfood` CLI); default `false`; does not gate the `memory_ingest` probe ([docs/architecture/mesh-dogfood.md](docs/architecture/mesh-dogfood.md))
- **Dogfood `memory_ingest` dual_write detail** — PASS detail always appends `dual_write=true|false` from Client cfg so human-readable reports show mode without relying only on top-level JSON
- **Dogfood `memory_ingest` session correlation detail** — probe envelope sets stable `session_id` (`{tenant}.mesh-dogfood` or `mesh-dogfood`) + `session_seq=1`; PASS detail appends `session_seq=` and `session_id=` when set (temporal correlation evidence without scraping payload)
- **Dogfood `memory_recall` step** — async `MEMORY_RPC` publish via `PublishMemoryRecall` (same `session_id` as ingest for temporal correlation); PASS detail includes `MEMORY_RPC`, `session_id=`, `dual_write=`
- **Sync memory retrieve** — `RetrieveMemory` → `POST /v1/memory/retrieve` (fallback `/v5`); dogfood step `memory_retrieve` with `hits=N` + correlated `session_id=`; empty hits still PASS
- **Agent auto-recall prefer sync HTTP** — when mesh and/or memory sidecar is configured, auto-recall and `/memory recall` use `RetrieveMemory` first; MCP `memory_retrieve` on failure/unavailability; status shows `sync_http=` / `mcp=`
- **Memory sidecar / stage warm plane** — optional `[memory].endpoint` (`IOMESH_MEMORY_ENDPOINT` / `MEMORY_SIDECAR_URL` / `--memory-endpoint`) used as base for `RetrieveMemory` + dogfood `memory_retrieve`; JSON `memory_endpoint` + PASS `memory_base=sidecar|mesh`

### Config / env (new)

| Key | Default | Notes |
|-----|---------|--------|
| `[memory].endpoint` / `IOMESH_MEMORY_ENDPOINT` / `MEMORY_SIDECAR_URL` | empty | Sync retrieve base (sidecar); else mesh endpoint |
| `--memory-endpoint` (dogfood) | empty | CLI override for sidecar base |
| `[iomesh] org` / `workspace` (dogfood headers) | empty | `X-IOMesh-Org` / `X-IOMesh-Workspace` on memory publish |

## [0.3.0] — 2026-07-16

Minor release: Memory Palace Phase 2 (HTTP MCP + dual-write `MEMORY_INGEST`) and catalog federation polish. Compatible with existing `v0.2.x` configs (new flags default off).

### Added

- **Memory Palace MCP (Phase 0–2)** — attach `aion-memory-mcp` via stdio or streamable HTTP MCP; `[memory]` auto-recall inject + opt-in auto-ingest; TUI `/memory` slash ([docs/architecture/memory-mcp.md](docs/architecture/memory-mcp.md))
- **Memory dual-write** — optional `dual_write` / `IOMESH_MEMORY_DUAL_WRITE` publishes async `memory_ingest` envelopes to mesh `MEMORY_INGEST` (temporal fields: `event_time`, `session_seq`, `session_id`, `role`, `content`); fail-open; no SDK dependency
- Portal catalog federation: after broker `/v1/catalog/*`, try `/v17/portal/catalog/data-products` and marketing catalog; normalize portal fields
- Agent tool `get_mesh_catalog_product`; dogfood catalog PASS for `source=portal`
- `iomesh mesh dogfood --json` for stage CI evidence

### Config / env (new)

| Key | Default | Notes |
|-----|---------|--------|
| `dual_write` / `IOMESH_MEMORY_DUAL_WRITE` | false | Mesh `MEMORY_INGEST` dual-write when `[iomesh]` enabled |

## [0.2.0] — 2026-07-16

Minor release: deeper I/O Mesh integration, multi-model honesty, catalog composition, and Vertex ADC ergonomics. Compatible with existing `v0.1.x` configs (new flags default fail-open / off where enforcement matters).

### Added

- **Mesh lineage context** — `include_lineage` on context plane queries; `<iomesh-lineage>` prompt block ([#22](https://github.com/iome-sh/iomesh-tui/pull/22))
- **Mesh policy gates** — `policy_mode` = `off` \| `advisory` \| `enforce` → `POST /v1/policy/evaluate` (fail-open on transport/404) ([#22](https://github.com/iome-sh/iomesh-tui/pull/22))
- **Local usage meter** — process `UsageMeter` via MetricsSink; `iomesh mesh usage`; headless `-p` stderr rollup ([#22](https://github.com/iome-sh/iomesh-tui/pull/22))
- **Mesh catalog plane** — `list_mesh_catalog` / `mesh_status` tools, `iomesh mesh catalog`, TUI `/catalog` `/mesh`, optional `inject_catalog`, dogfood catalog step ([#26](https://github.com/iome-sh/iomesh-tui/pull/26))
- **TUI `/cost`** — session process usage + sample estimate ([#26](https://github.com/iome-sh/iomesh-tui/pull/26))
- **Vertex ADC auto-refresh** — cached access token + `gcloud` refresh on 401 ([#25](https://github.com/iome-sh/iomesh-tui/pull/25), [#27](https://github.com/iome-sh/iomesh-tui/pull/27))
- Docs: [mesh-deeper.md](docs/architecture/mesh-deeper.md); multi-model catalog tables in README / llm-cascade

### Changed

- Multi-model positioning (DeepSeek · Grok · Gemini · Vertex) — default cascade still Flash → Pro → Grok when unpinned ([#24](https://github.com/iome-sh/iomesh-tui/pull/24))
- Org branding: **IOMesh Technology Ltd.** + [iome.sh](https://iome.sh) in LICENSE/NOTICE/README; GitHub About homepage ([#23](https://github.com/iome-sh/iomesh-tui/pull/23))
- CI GitHub Actions pins: `checkout` / `setup-go` / `upload-artifact` v7 ([#18](https://github.com/iome-sh/iomesh-tui/pull/18)–[#20](https://github.com/iome-sh/iomesh-tui/pull/20))

### Config / env (new)

| Key | Default | Notes |
|-----|---------|--------|
| `include_lineage` / `IOMESH_INCLUDE_LINEAGE` | true | Context plane lineage |
| `policy_mode` / `IOMESH_POLICY_MODE` | off | advisory \| enforce |
| `catalog_plane` / `IOMESH_CATALOG_PLANE` | true | Data-product discovery |
| `inject_catalog` / `IOMESH_INJECT_CATALOG` | false | Per-turn catalog inject |

## [0.1.0] — 2026-07-16

First public tagged release of the I/O Mesh TUI coding agent.

### Added

- DeepSeek-first LLM cascade + pure-Go OpenAI-compatible router (DeepSeek, OpenAI, Anthropic, Gemini / Vertex OpenAI-compat)
- Agent loop with workspace tools, path jail, shell policy, and secret scrubbing
- Subagents (parallel runs, git worktree isolation, apply/merge)
- Session persistence and interactive permissions
- Full-screen Bubble Tea TUI and headless `-p` prompt mode
- ACP over stdio and WebSocket (`iomesh agent serve`)
- Skills loader and MCP client (stdio/HTTP: tools, resources, prompts, OAuth helpers)
- Stage I/O Mesh mesh dogfood probe (`iomesh mesh dogfood` / `make dogfood`)
- Open-source launch pack: LICENSE, SECURITY, SUPPORT, CONTRIBUTING, RELEASING, NOTICE, issue/PR templates, Dependabot
- CI: lint, test, race, coverage artifact, govulncheck, build

### Security

- Residual-risk documentation for public operators ([SECURITY.md](SECURITY.md), [docs/security.md](docs/security.md))
- ACP loopback Origin hardening; path-jail and scrubbing defaults documented

[Unreleased]: https://github.com/iome-sh/iomesh-tui/compare/v0.28.0...HEAD
[0.28.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.28.0
[0.27.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.27.0
[0.26.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.26.0
[0.25.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.25.0
[0.24.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.24.0
[0.23.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.23.0
[0.22.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.22.0
[0.21.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.21.0
[0.20.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.20.0
[0.19.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.19.0
[0.18.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.18.0
[0.17.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.17.0
[0.16.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.16.0
[0.15.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.15.0
[0.14.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.14.0
[0.13.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.13.0
[0.12.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.12.0
[0.11.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.11.0
[0.10.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.10.0
[0.9.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.9.0
[0.8.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.8.0
[0.7.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.7.0
[0.6.1]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.6.1
[0.6.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.6.0
[0.5.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.5.0
[0.4.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.4.0
[0.3.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.3.0
[0.2.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.2.0
[0.1.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.1.0
