# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- **Mesh dogfood `memory_ingest` step** — exercises Phase 2 dual-write via `PublishMemoryIngest` (`POST /v1/streams/MEMORY_INGEST/publish`); included by default when mesh enabled (fail-open → SKIP unless `--strict`); CLI `--skip-memory` to omit ([docs/architecture/mesh-dogfood.md](docs/architecture/mesh-dogfood.md))
- **MEMORY_INGEST dual-write org/workspace headers** — optional `[iomesh] org` / `workspace` (`IOMESH_ORG` / `MEMORY_ORG` / `IOMESH_WORKSPACE`) set `X-IOMesh-Org` + `X-IOMesh-Workspace` on `PublishMemoryIngest` (M5 entitlements parity)
- **Dogfood `memory_ingest` org/workspace evidence** — PASS detail appends `org=` / `workspace=` when Client OrgID/WorkspaceID are configured (omitted when unset)
- **Dogfood JSON `org` / `workspace` fields** — `DogfoodReport` + `FormatReportJSON` carry Client OrgID/WorkspaceID as top-level `org` / `workspace` (`omitempty`) for stage CI / multi-tenant gate parsing ([docs/architecture/mesh-dogfood.md](docs/architecture/mesh-dogfood.md))
- **Dogfood JSON `dual_write` field** — `DogfoodReport` + `FormatReportJSON` always emit top-level `dual_write` bool from Client cfg (wired from `[memory].dual_write` / `IOMESH_MEMORY_DUAL_WRITE` in `mesh dogfood` CLI); default `false`; does not gate the `memory_ingest` probe ([docs/architecture/mesh-dogfood.md](docs/architecture/mesh-dogfood.md))
- **Dogfood `memory_ingest` dual_write detail** — PASS detail always appends `dual_write=true|false` from Client cfg so human-readable reports show mode without relying only on top-level JSON
- **Dogfood `memory_ingest` session correlation detail** — probe envelope sets stable `session_id` (`{tenant}.mesh-dogfood` or `mesh-dogfood`) + `session_seq=1`; PASS detail appends `session_seq=` and `session_id=` when set (temporal correlation evidence without scraping payload)
- **Dogfood `memory_recall` step** — async `MEMORY_RPC` publish via `PublishMemoryRecall` (same `session_id` as ingest for temporal correlation); PASS detail includes `MEMORY_RPC`, `session_id=`, `dual_write=` (s247)
- **Sync memory retrieve** — `RetrieveMemory` → `POST /v1/memory/retrieve` (fallback `/v5`); dogfood step `memory_retrieve` with `hits=N` + correlated `session_id=` (s251); empty hits still PASS
- **Agent auto-recall prefer sync HTTP** — when `[iomesh]` mesh client is enabled, auto-recall and `/memory recall` use `RetrieveMemory` first; MCP `memory_retrieve` on failure/unavailability; status shows `sync_http=` / `mcp=` (s252)
- **Memory sidecar / stage warm plane** — optional `[memory].endpoint` (`IOMESH_MEMORY_ENDPOINT` / `MEMORY_SIDECAR_URL` / `--memory-endpoint`) used as base for `RetrieveMemory` + dogfood `memory_retrieve`; JSON `memory_endpoint` + PASS `memory_base=sidecar|mesh` (s269)

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

[Unreleased]: https://github.com/iome-sh/iomesh-tui/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.3.0
[0.2.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.2.0
[0.1.0]: https://github.com/iome-sh/iomesh-tui/releases/tag/v0.1.0
