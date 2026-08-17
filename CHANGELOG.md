# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- **TUI integrate-app UX (s2055)** — infer broker when building the TUI/ACP mesh client (`runtimewire.NewMesh` + `config.ApplyInferredBroker`; same mapping as CLI). `/setup reload` hot-swaps mesh via `Runtime.ReplaceMesh` when `[iomesh]` or inferred hooks change. Dashboard `consume missing` without a mesh client tells the operator to add `[iomesh]` or infer from portal MCP (does not invent consume). Slash `/setup init` accepts `--mesh-endpoint` · `--mesh-tenant` · `--platform-mcp-url` (writes hooks, not `/v7/mcp`; apiv1 infers `hooks.iome.sh`). After mesh/platform-mcp init: `IOMESH_TOKEN` → reload → `iomesh mesh streams --create --yes` → `--messages` (create ≠ PULSE). After `--create` and empty `FormatStreams`: inbox is empty until the first durable event from the app/console tap; mesh pub is ephemeral and does not fill `/dashboard`. Mesh disabled on `pub`/`status`/`wait` reuses the streams hooks-vs-catalog hint. Infer ≠ Connected · catalog MCP ≠ hooks streams. HITL stays OPEN. Docs: [tui.md](docs/architecture/tui.md) · [setup-lifecycle.md](docs/architecture/setup-lifecycle.md) · [mesh-deeper.md](docs/architecture/mesh-deeper.md).

- **Mesh streams create (s2038)** — lean `CreateStream` (`POST /v1/streams`; console defaults `OPERATIONAL_EVENTS` / `dept.{tenant}.events.github`; `retention=limits` · `max_age_sec=604800` · `max_msgs=1000000`; **no** `retention_tier` on the create wire). 201 decodes `StreamInfo`; **409 Conflict = success** (idempotent — `GetStream` or name-only). CLI `iomesh mesh streams --create --yes` (`--name` defaults `OPERATIONAL_EVENTS`; `--subject` override; incompatible with `--delete` / `--messages`). Text: `PASS mesh streams create` + `FormatStreamDetail`; JSON: `StreamInfoPrint`. Create ≠ PULSE (listed stream + 0 messages is still empty). HITL stays OPEN. Do not invent unpaid 403 (raw `http 403`). Dashboard empty CTA adds `or: iomesh mesh streams --create --yes` (console Settings path kept). Docs: [mesh-deeper.md](docs/architecture/mesh-deeper.md).

### Changed

- **Onboard next mesh integrate-app steps (s2057)** — `/onboard next mesh` now lists the shipped s2055 CLI path: add `[iomesh]` or infer from portal MCP (infer ≠ Connected) → `IOMESH_TOKEN` → `iomesh mesh streams --create --yes` (create ≠ PULSE) → wait for a durable event from the app/console tap → `--messages` / `/dashboard` (mesh pub ephemeral ≠ consume). Residual-honest: `streams_not_probed` · never invent stream green · catalog ≠ Connected · mesh ≠ memory. Existing lock dump `~$88/$119` stays. Does not invent Memory GA.

- **Dashboard three-pillar kinds (s2046)** — consume classifies `Kind` from subject tokens (same as CP): `events.docs` / `events.documents` / `notion` / `confluence` / `sharepoint` / `google_drive` / `.docs.` → knowledge; `metric.` / `.cdc` / `embedding` / `warehouse` / `.dbt` → analytics; else ops. GitHub stays ops. When knowledge or analytics count is 0, `/dashboard` adds `knowledge Beta empty · analytics Beta empty · not GA`. Does not invent events or GA. C2 stays OPEN.
- **Start-here / guidance rates (s2045)** — operator-visible Memory line + short guidance lock now say `Base ~$132 · Memory Ops Pack hidden public · local memory free OSS` (not `~$88/$119` as first-run). Residual lane lock dumps stay `~$88/$119`. Pack hidden · no freemium · `~$132` Base.
- **`mesh streams --messages` SUBJECT column** — 28 → 36 so `dept.engineering.events.github` is not truncated to `git…`.
- **`mesh streams --messages` preview** — text table peels stacked persist base64 and unwraps `observation.payload` to `event_type · repository` (not `eyJ…`). JSON `--messages` still emits raw payload. Pretty preview ≠ Connected.
- **Mesh CLI infer broker (all commands)** — `applyInferredBroker` runs on `mesh wait|status|catalog|streams|consumer|pub|kv|smoke` and `memory pull`, not only `mesh streams`. Portal MCP (`apiv1.iome.sh/v7/mcp`) still maps to `hooks.iome.sh` (+ tenant/org from MCP headers). `--endpoint` wins. Empty infer does not invent Enabled. Infer ≠ Connected.
- **Mesh streams infer broker** — `iomesh mesh streams` without `[iomesh]` still listed if `--endpoint` was set, but `--messages` said `mesh disabled`. Portal MCP (`apiv1.iome.sh/v7/mcp`) is catalog, not the stream broker. CLI now infers `https://hooks.iome.sh` (+ tenant/org from MCP headers) and prints a residual-honest hint. Setup `platform-mcp` writes `[iomesh]` when the portal URL is apiv1. Infer ≠ Connected.

- **Dashboard consume honesty (#342 / #344 / this)** — `/dashboard` defaults **empty** (no mock P2 / Pulse 18). `/dashboard preview` is the iome.sh **eval template**, not your org. When a mesh client is attached, `/dashboard` probes broker `GET /v1/streams` + `GET /v1/streams/{name}/messages` (same path as `iomesh mesh streams --messages`) — **not** cookie-only `GET /v52`. Fail-open reasons: `no_streams` · `empty_stream` · `replay_disabled` · `broker_unavailable`. PULSE-shaped rows only when ≥1 message was decoded. Never invent PULSE from eval seed or stream list alone. Mesh auth: `IOMESH_TOKEN` then `IOMESH_API_KEY`; Org/Workspace headers on all mesh requests including stream list/messages. Stacked persist payloads are peeled (std + URL-safe base64, max 4) then unwrapped (`observation.payload.event_type` + `repository`, or `title`/`summary`) so titles are `create · iome-sh/aion` instead of `eyJ…`. `/memory` empty MCP path is `mcp-manager-empty (0 servers) · fail-open` (same as `/integrations`). Does not invent P2 checkout. HITL stays OPEN. catalog ≠ Connected.

## [0.77.0] — 2026-08-15

Minor release: landing-page heartbeat dashboard in the TUI (`/dashboard`) plus README showcase. **Beta** · dual_write OFF · catalog ≠ Connected · not Memory GA · not live APPLY · eval template ≠ Connected.

### Added

- **README dashboard showcase (s1990)** — README hero + [Dashboard](README.md#dashboard-heartbeat-live-feed) section: [docs/assets/dashboard-eval.svg](docs/assets/dashboard-eval.svg) (versionable eval-template image, not a tenant GIF), TUI vs iome.sh MeshConsole vs console.iome.sh table, `/dashboard` example. Same honesty as s1989. Docs: [tui.md](docs/architecture/tui.md).
- **TUI heartbeat dashboard (s1989)** — `/dashboard` (aliases `/heartbeat` `/mesh-console`) ports the iome.sh landing MeshConsole: tenancy · pulse · heartbeat live feed · agent tools ALLOW/DENY · kind analysis. REPL prints a snapshot; fullscreen toggles a ticking overlay (esc/q close · tab/1–4 tenancy). Feed is the public **evaluation template**, not a Connected workspace. Badge EVAL (no mesh client) / CLIENT (client configured — still template until a real stream is pulled). `catalog ≠ Connected` · `dual_write OFF` · knowledge/analytics **Beta** · not Memory GA · not live APPLY · demo feed ≠ fleet-GA. Docs: [tui.md](docs/architecture/tui.md).

### Changed

- **TUI agent / MCP / integrations start-here (s1982)** — `/onboard`, `/onboard portal|status|checklist|next`, `/onboard next wizard`, and the injected `<aion-onboarding>` / `<integrations>` notes now lead with a five-step path (portal MCP copy → TUI `[[mcp.servers]]` attach → `/integrations list|plan` → portal HITL → `/setup` / wizard). Residual boards stay below as operator notes. `/help` and `/integrations` help match. Peer of console **s1981**. Never invent Connected / Memory GA / install APPLY.

### Added

- **Skills residual-honest next-step after list_skills/read_skill (s1837)** — after agent tools `list_skills` / `read_skill` (no dedicated `/skills` slash; skills load via catalog + tools + `/setup reload` re-scan s1670), residual next-step footers via `SkillsNextStepLines`: dual path — if TUI/session running → `/setup preflight` · `/setup reload` (skills re-scan · package wire ≠ Connected) · optional `list_skills` tool · `/onboard next setup`; cold start → restart `iomesh` · `iomesh setup preflight`. Appended to successful tool return strings only (errors stay bare · never invent success). Peer of plugins next-step (s1829) · memory next-step (s1831) · onboard next-step (s1825) · integrations next-step (s1727). skills re-scan ≠ invent Connected · package wire ≠ Connected · dual_write **OFF** · not Agent Plugins GA · not Memory GA · free eng **s1837**. Docs: [skills.md](docs/architecture/skills.md).

## [0.76.0] — 2026-08-12

Minor release: residual next-step honesty continuum after v0.75.0 — onboard status/checklist/next/portal next-step (s1825) · plugins list/validate/smoke/status next-step (s1829) · memory status/help/digest next-step (s1831). **Beta** · dual_write OFF · package wire ≠ Connected · catalog ≠ Connected · Discover ≠ Connected · not Memory GA · not invent Agent Plugins GA · packaging ≠ invent GA · E10 Open · book-demo OFF.

### Added

- **Memory residual-honest next-step after status/help/digest (s1831)** — after bare `/memory` (help) · `/memory status` (`MemoryAdvancedStatus`) · `/memory digest`, residual next-step footers via `MemoryNextStepLines`: dual path — if TUI/session running → `/setup preflight` · `/setup reload` · optional `/memory digest` · `/onboard next memory|memory-pull`; cold start → restart `iomesh` · `iomesh setup preflight` · optional `iomesh memory pull`. Peer of onboard next-step (s1825) · integrations next-step (s1727) · setup next-step continuum (s1686–s1723). Advanced slash surfaces remain honesty-footer fragmented (not all re-wired). dual_write **OFF** · not Memory GA · local-primary · package wire ≠ Connected · soft ≠ invent live dogfood · free eng **s1831**. Never invent Connected / Memory GA from memory slash alone. Docs: [memory-mcp.md](docs/architecture/memory-mcp.md).
- **Plugins residual-honest next-step after list/validate/smoke/status (s1829)** — after `/plugins` list|validate|smoke|status|help, residual next-step footers via `PluginsNextStepLines` (in `internal/agentplugins`, peer of ResidualSlashHonesty): dual path — if TUI/session running → `/setup preflight` · `/setup reload` (skills/MCP re-scan · package wire ≠ Connected) · optional `/onboard next plugins|status`; cold start → restart `iomesh` · `iomesh setup preflight` · optional `iomesh plugins smoke`. Appended after ResidualSlashHonesty (and ResidualDogfoodHonesty error paths) on all `/plugins` handler footers. Peer of integrations next-step (s1727) · onboard next-step (s1825). Docs: [agent-plugins.md](docs/architecture/agent-plugins.md). Discover ≠ Connected · soft offline smoke ≠ invent Agent Plugins GA · package load ≠ Memory GA · dual_write **OFF** · free eng **s1829**.
- **Onboard residual-honest next-step after status/checklist/next/portal (s1825)** — after `/onboard status` · `/onboard checklist` · `/onboard next` lanes map · `/onboard portal`, residual next-step footers via `OnboardNextStepLines` (alias `AionAgentOnboardingNextStepLines`): dual path — if TUI/session running → `/setup preflight` · `/setup reload` · optional `/integrations list` · `/onboard next portal-hitl|setup|memory`; cold start → restart `iomesh` · `iomesh setup preflight`. Peer of integrations next-step (s1727) · setup next-step continuum (s1686–s1723). dual_write **OFF** · package wire ≠ Connected · catalog ≠ Connected · agent MCP cannot write installs · not Memory GA · free eng **s1825**. Never invent Connected / Memory GA from onboard maps alone.

## [0.75.0] — 2026-08-11

Minor release: residual next-step honesty continuum after v0.74.0 — setup init slash parity + portal next-step + IOMESH_PLATFORM_RESIDUAL label (s1723) · integrations list/plan/status/signing next-step (s1727). **Beta** · dual_write OFF · package wire ≠ Connected · catalog ≠ Connected · agent MCP cannot write installs · template= ≠ install APPLY · IOMESH_PLATFORM_RESIDUAL labels only (does not hide lanes) · residual PASS ≠ invent control plane · not Memory GA · E10 Open · book-demo OFF.

### Added

- **Integrations residual-honest next-step after list/plan/status/signing (s1727)** — after `/integrations` list|plan|status|signing (and offline/tool-missing fail-open), residual next-step footers via `IntegrationsNextStepLines`: browser portal HITL for OAuth/install (agent MCP **cannot write installs**) → in-session `/setup preflight` · `/setup reload` · optional `/onboard next portal-hitl`; cold start → restart iomesh · `iomesh setup preflight`. Appended to catalog/plan/signing/status honesty footers + offline messages. Peer of setup next-step continuum (s1686–s1723). Docs: [agent-integrations-setup.md](docs/architecture/agent-integrations-setup.md) · skill `connector-integrations-setup`. catalog ≠ Connected · template= ≠ install APPLY · dual_write **OFF** · not Memory GA · free eng **s1727**.
- **Setup init slash next-step parity + portal next-step + IOMESH_PLATFORM_RESIDUAL label (s1723)** — slash `/setup init` uses `SetupInitNextStepLines` (CLI parity with s1686). `/setup portal` appends `SetupPortalNextStepLines` (browser HITL → preflight/reload dual path · agent MCP cannot write installs · catalog ≠ Connected). Optional `IOMESH_PLATFORM_RESIDUAL=1` env labels platform residual honesty only (does **not** hide Edge OSS lanes · residual PASS ≠ invent control plane). dual_write **OFF** · package wire ≠ Connected · not Memory GA · free eng **s1723**.

## [0.74.0] — 2026-08-11

Minor release: setup Format*/next-step honesty continuum after v0.73.0 — drift/repair dual-path next-step (s1707) + reload/pull/analyze next-step (s1711). **Beta** · dual_write OFF · package wire ≠ Connected · not Memory GA · CLI has no setup drift/repair/reload · pull ≠ invent Connected · analyze tick ≠ invent Connected · repair apply ≠ invent Connected · E10 Open · book-demo OFF.

### Added

- **Setup reload/pull/analyze next-step honesty (s1711)** — after `/setup reload` and `/setup pull` / `/setup analyze` outputs, residual-honest next-step footers via `SetupReloadNextStepLines` · `SetupPullNextStepLines` · `SetupAnalyzeNextStepLines`. **Reload** in-session only (CLI has **no** setup reload) → optional `/setup pull|analyze start` · `/setup drift` · `/memory digest`. **Pull** dual path: in-session slash (`/setup pull status|start|once|stop`) vs CLI `iomesh memory pull`. **Analyze** dual path: in-session slash (`/setup analyze status|start|once|stop`) vs `/memory digest`. Peers of s1686 init / s1699 preflight / s1707 drift/repair. Docs: [setup-lifecycle.md](docs/architecture/setup-lifecycle.md) · skill `setup-lifecycle-agent`. dual_write **OFF** · package wire ≠ Connected · pull ≠ invent Connected · analyze tick ≠ invent Connected · not Memory GA · free eng **s1711**.
- **Setup drift/repair dual-path next-step honesty (s1707)** — after `/setup drift` / `/setup maintain` (`FormatDriftText`) and `/setup repair` plan/result (`FormatRepairPlan` / `FormatRepairResult`), residual-honest dual path: TUI/session running → in-session slash (`/setup repair` · `/setup reload` · optional pull/analyze · re-run drift); cold start → restart `iomesh` · `iomesh setup preflight` (CLI has **no** setup drift/repair/reload). Helpers `setup.SetupDriftNextStepLines` · `setup.SetupRepairNextStepLines` unit-tested · peers of s1686 init / s1699 preflight. Docs: [setup-lifecycle.md](docs/architecture/setup-lifecycle.md) · skill `setup-lifecycle-agent`. dual_write **OFF** · never auto ON · package wire ≠ Connected · repair apply ≠ invent Connected · not Memory GA · free eng **s1707**.

## [0.73.0] — 2026-08-11

Minor release: setup first-run residual continuum after v0.72.0 — CLI init dual-path next-step, preflight dual-path next-step, Memory Ops Pack local-primary honesty (not first-run required). **Beta** · dual_write OFF · package wire ≠ Connected · not Memory GA · Ops Pack optional · CLI has no setup reload · E10 Open · book-demo OFF.

### Added

- **Setup preflight dual-path next-step honesty (s1699)** — after `iomesh setup preflight` / `/setup preflight` report (`FormatPreflightText`), residual-honest dual path: preflight ok + TUI/session running → `/setup reload` (hot-swap MCP + skills · package wire ≠ Connected); host/secrets missing → start host · set env · re-run preflight; cold start → restart `iomesh` (CLI has **no** `setup reload`). Helper `setup.SetupPreflightNextStepLines` unit-tested · peer of s1686 init next-step. Docs: [setup-lifecycle.md](docs/architecture/setup-lifecycle.md) · skill `setup-lifecycle-agent` · README first-run. dual_write **OFF** · package wire ≠ Connected · not Memory GA · catalog ≠ Connected · PASS ≠ invent install green · free eng **s1699**.
- **Ops Pack local-primary honesty / not first-run required (s1695)** — residual-honest first-run + memory-lane Memory Ops Pack framing: **OSS first-run complete without mesh** · **Ops Pack not first-run required** · **Memory Ops Pack optional** commercial overlay (~$119 pull/retain/support · local-primary · TUI OSS + mesh pull entitlement) · mesh optional · dual_write **OFF** · not Memory GA · not freemium hosted palace · Ops Pack ≠ GPU fleet · package load ≠ Ops Pack entitlement. Surfaces: `/onboard next memory` · `/onboard next memory-pull` · README First-run · [memory-mcp.md](docs/architecture/memory-mcp.md) buyer claim pin · [edge-user-journey.md](docs/architecture/edge-user-journey.md) stages 6–7 · skill `aion-agent-onboarding` honesty line. free eng **s1695**.
- **CLI setup init next-step reload/restart honesty (s1686)** — after `iomesh setup init` write, residual-honest dual path: TUI/session already running → `/setup preflight` · `/setup reload` (hot-swap MCP + skills · package wire ≠ Connected); cold start → restart `iomesh` · `iomesh setup preflight`. CLI has **no** `iomesh setup reload` subcommand (in-session only). Helper `setup.SetupInitNextStepLines` unit-tested. Docs: [setup-lifecycle.md](docs/architecture/setup-lifecycle.md) · skill `setup-lifecycle-agent` · README first-run. dual_write **OFF** · package wire ≠ Connected · not Memory GA · catalog ≠ Connected · free eng **s1686**.

## [0.72.0] — 2026-08-11

Minor release: agent-native setup lifecycle + edge first-run continuum + residual-honest soft checks + OSS packaging / marketing-demo / Python SDK peer honesty + easy first-run skills reload (s1525–s1670 and post-0.71 Unreleased mesh/print residual on main). **Beta** · dual_write OFF · package wire ≠ Connected · not Memory GA · not Agent Plugins GA · E10 Open · book-demo OFF.

### Added

- **Easy first-run + skills reload residual (s1670)** — README install pin **v0.72.0** + **First-run (agent)** path (`/setup init` · preflight · memory host · `/setup reload` · onboard maps journey/setup/wizard/marketing-demo). `/setup reload` re-scans skills via `runtimewire.Wire` SkillDirs + `skills.LoadWithBuiltin` + `Runtime.ReplaceSkills` (including plugin skill dirs when `[plugins]` enabled) — process restart no longer required for skill-only path changes. Docs honesty flip: [skills.md](docs/architecture/skills.md) / [mcp.md](docs/architecture/mcp.md) residual-honest s1331 package wire truth · [setup-lifecycle.md](docs/architecture/setup-lifecycle.md) reload row · [docs/README](docs/README.md) agent-plugins index. dual_write **OFF** · package wire ≠ Connected · not Agent Plugins GA · skills re-scan ≠ invent Connected · not Memory GA · free eng **s1670**.
- **Python client SDK peer honesty (s1666)** — public TUI docs peer **Go and Python** official client SDKs for full mesh client surface ([iomesh-client-sdk-go](https://github.com/iome-sh/iomesh-client-sdk-go) · [iomesh-client-sdk-python](https://github.com/iome-sh/iomesh-client-sdk-python)); **iomesh-tui stays lean** (no Go SDK module dep · no Python SDK packaging). Python **Beta** · tip **v0.10.x** · **v0.10 ≠ invent 1.0** · **GitHub release ≠ invent PyPI green**. Docs: [memory-mcp.md](docs/architecture/memory-mcp.md) · [overview.md](docs/architecture/overview.md) · [marketing-demo-path.md](docs/architecture/marketing-demo-path.md) · light [mesh-deeper.md](docs/architecture/mesh-deeper.md). dual_write **OFF** · not Memory GA · book-demo **OFF** · MIT edge Beta · free eng **s1666** · free-floor peer mention only.
- **Marketing demo path (s1590)** — plain-language `/onboard next marketing-demo` operator script for **videos and sales** of local agent + local memory (aliases `marketing` · `sales-demo` · `demo-script` · `gtm-demo`) via `AionAgentOnboardingNextMarketingDemoLane`. Steps: install/build iomesh → LLM key or Ollama → `/setup init` local-memory + preflight → start/attach `iomesh-memory-mcp` → `/memory` ingest + recall · mesh **optional** only if configured. Listed under Edge OSS / demo-oriented continuum (not buried as residual dogfood). dual_write **OFF** · local memory · not Memory GA · never invent Connected · book-demo OFF. Does **not** steal bare `demo` (demo readiness s1442) · bare `sales` (claims) · bare `gtm` (drafts). Docs: [marketing-demo-path.md](docs/architecture/marketing-demo-path.md) · skill continuum stamp. free eng **s1590** · free-floor peer **s1592+** mention only.
- **E10 Open reaffirm residual-check (s1586)** — residual-honest **E10 Open** reaffirm after OSS packaging continuum (Platform residual honesty group). Board `/onboard next e10` (aliases `e10-open` · `edge-memory-e10` · `ga-signoff` · `e10_open`) via `AionAgentOnboardingNextE10Lane` — pin **E10 Open** · residual PASS ≠ invent E10 closed · residual PASS ≠ invent Edge Memory GA declared · Edge Memory GA candidacy only · founder sign-off only if declaring Edge Memory GA · candidacy allowed without E10 · Live APPLY still human · PASS ≠ live APPLY · dual_write OFF · book-demo OFF · not Memory GA · residual-check. Soft residual-check `/onboard next e10 dogfood` (aliases `soft`/`samples`/`offline`/`e10-soft`/`residual-check`) via `RunE10OpenSoftDogfood` · session labels `e10_soft_not_run` · `soft_offline_e10_session_pass|fail` (independent of tool-call soft s1578 · still-human soft s1574 · wizard soft s1570 · E4 soft s1566 · portal HITL soft s1562 · agentic list/plan soft s1422) · **never dial MCP / never start host**. Cross-links: e4 · human-gates · tool-call · OSS packaging (MIT harness ≠ control plane) · skill continuum · [edge-user-journey.md](docs/architecture/edge-user-journey.md) · [oss-packaging-boundary.md](docs/architecture/oss-packaging-boundary.md) · [EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md](docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md). Optional note only: `IOMESH_PLATFORM_RESIDUAL=1` future/opt-in documentation (does not hide lanes). Honesty: dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · residual PASS ≠ invent E10 closed · E10 Open · residual PASS ≠ live dogfood · session soft ≠ live dogfood · soft offline ≠ invent Connected · free eng **s1586** · free-floor peer **s1588+** mention only.
- **OSS packaging residual (s1582)** — residual-honest **public MIT packaging boundary** so OSS readers see Edge harness first; platform residual honesty is labeled optional anti-claim rails (not control plane). New SSOT [docs/architecture/oss-packaging-boundary.md](docs/architecture/oss-packaging-boundary.md) (MIT harness vs private control plane table · Edge OSS path · residual-check glossary · CHANGELOG serial reading · free eng **s1582** · free-floor peer **s1584+** mention only). README Edge install table + short **Platform residual honesty (optional)** subsection · `docs/README` index. Operator surface: `OSSPackagingHonestyOneLiner` on bare `/onboard` · `/onboard next` continuum split into **Edge OSS path** vs **Platform residual honesty (optional · anti-claims · offline residual checks)** · user-facing **residual-check** alongside slash token `dogfood` (compat). Soft residual-check harnesses **kept** (anti-claims). Honesty: MIT OSS · dual_write OFF · not Memory GA · Edge Memory GA candidacy only · book-demo OFF · residual PASS ≠ invent control plane in MIT repo · session soft ≠ live dogfood · free eng **s1582** · free-floor peer **s1584+** mention only.
- **Deeper tool-call soft dogfood residual (s1578)** — residual-honest optional deeper tool path soft dogfood after E4 attach (tools=6 / `iomesh mcp --connect` stamp residual). Board `/onboard next tool-call` (aliases `tool-calls` · `deeper-e4` · `e4-tools` · `ingest-retrieve` · `tool_call`) via `AionAgentOnboardingNextToolCallLane` — journey **stage 6/7** depth · operator map ingest → retrieve → list → as-of/status · tool names `memory_ingest_turn` · `memory_retrieve` · `memory_search_semantic` · `memory_list` · `memory_compact_status` · `memory_facts_as_of` · Partial→client-attach-evidence · companion E4 (s1508/s1566). Soft offline dogfood `/onboard next tool-call dogfood` (aliases `soft`/`samples`/`offline`/`tool-call-soft`) via `RunDeeperToolCallSoftDogfood` · session labels `tool_call_soft_not_run` · `soft_offline_tool_call_session_pass|fail` (independent of still-human soft s1574 · wizard soft s1570 · E4 soft s1566 · portal HITL soft s1562 · agentic list/plan soft s1422) · **never dial MCP / never start host**. Cross-links: E4 lane · memory lane · journey stage 6/7 · skill continuum · [EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md](docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md) · [edge-user-journey.md](docs/architecture/edge-user-journey.md). Honesty: dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · tip ≠ invent forever-green product dogfood · residual PASS ≠ live dogfood · session soft ≠ live dogfood · soft offline ≠ invent Connected · free eng **s1578** · free-floor peer **s1580+** mention only.
- **Still-human APPLY soft dogfood residual (s1574)** — residual-honest Wave C continuum reaffirm that **still-human APPLY** open boxes stay open after Wave A–C (journey · wizard · portal-hitl · e4 soft). Soft offline dogfood `/onboard next human-gates dogfood` (aliases `soft`/`samples`/`offline`/`still-human-soft`/`apply-soft`) via `RunStillHumanApplySoftDogfood` · session labels `still_human_soft_not_run` · `soft_offline_still_human_session_pass|fail` (independent of wizard soft s1570 · E4 soft s1566 · portal HITL soft s1562 · agentic list/plan soft s1422) · **never dial MCP / never start host**. Board aliases: `still-human` · `apply-residual`. Open inventory residual-honest reaffirm: Slack HMAC punted/still open · Stripe Customers:Write residual · H1/H2 knowledge INSTALL_STORE punted/not launch gate · OAuth Connected still portal HITL · book-demo OFF · dual_write OFF · E10 Open. Companion: `/onboard next human-gates` · wizard · journey · setup · portal-hitl · e4. Docs: skill `aion-agent-onboarding` · [edge-user-journey.md](docs/architecture/edge-user-journey.md) · [setup-lifecycle.md](docs/architecture/setup-lifecycle.md). Honesty: dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open · portal HITL when connect · residual PASS ≠ live dogfood · session soft ≠ live dogfood · soft offline ≠ invent Connected · free eng **s1574** · free-floor peer **s1576+** mention only.
- **Wave C first-run wizard residual (s1570)** — residual-honest `/onboard next wizard` lane (aliases `first-run-wizard` · `guided` · `wave-c` · `wave_c` · `wizard-residual`) via `AionAgentOnboardingNextWizardLane` — deeper guided first-run residual map after Wave B journey (stages 1–7 with primary residual-honest next actions + honesty non-claims) · **not** invent full interactive auto wizard UX. Soft offline dogfood `/onboard next wizard dogfood` (aliases `soft`/`samples`/`offline`/`wizard-soft`) via `RunFirstRunWizardSoftDogfood` · session labels `wizard_soft_not_run` · `soft_offline_wizard_session_pass|fail` (independent of E4 soft s1566 · portal HITL soft s1562 · agentic list/plan soft s1422) · **never dial MCP / never start host**. Companion: journey · setup · portal-hitl · e4 · human-gates. Note: `wizard` alias moved from setup lane to Wave C wizard lane (setup keeps `setup-lifecycle` · `lifecycle` · `setup_lifecycle`). Docs: [edge-user-journey.md](docs/architecture/edge-user-journey.md) Wave C residual · [setup-lifecycle.md](docs/architecture/setup-lifecycle.md). Honesty: dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · portal HITL when connect · agent MCP cannot write installs · catalog ≠ Connected · no invent TUI portal SSO · host not auto · residual PASS ≠ invent full interactive auto wizard · residual PASS ≠ live dogfood · session soft ≠ live dogfood · soft offline ≠ invent Connected · free eng **s1570** · free-floor peer **s1572+** mention only.
- **E4 client-attach soft dogfood residual (s1566)** — residual-honest `/onboard next e4` lane (aliases `e4-dogfood` · `client-attach` · `edge-memory-e4` · `e4_attach`) via `AionAgentOnboardingNextE4Lane` — edge-user-journey **stage 6** local store / MCP attach · E4 client attach · tools=6 · `iomesh mcp --connect` residual · `iomesh-memory-mcp` · local-primary · evidence [EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md](docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md). Soft offline dogfood `/onboard next e4 dogfood` (aliases `soft`/`samples`/`offline`/`e4-soft`) via `RunE4SoftDogfood` · session labels `e4_soft_not_run` · `soft_offline_e4_session_pass|fail` (independent of portal HITL soft s1562 · agentic list/plan soft s1422) · **never dial MCP / never start host**. Cross-links: journey stage 6 · memory companion · skill continuum · [edge-user-journey.md](docs/architecture/edge-user-journey.md). Honesty: dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · tip ≠ invent forever-green product dogfood · residual PASS ≠ live dogfood · session soft ≠ live dogfood · soft offline ≠ invent Connected · PASS ≠ live APPLY · free eng **s1566** · free-floor peer **s1568+** mention only.
- **Portal HITL soft dogfood residual (s1562)** — residual-honest `/onboard next portal-hitl` lane (aliases `hitl` · `portal_hitl` · `portal-dogfood` · `stage5` · `connectors-hitl`) via `AionAgentOnboardingNextPortalHITLLane` — edge-user-journey **stage 5** connectors / portal HITL when connect · path MCP list/plan → browser portal HITL → human OAuth/install · proven paths `/integrations/{id}` · `/integrations/add?template={id}` · `/integrations`. Soft offline dogfood `/onboard next portal-hitl dogfood` (aliases `soft`/`samples`/`offline`/`portal-hitl-soft`) via `RunPortalHITLSoftDogfood` · session labels `portal_hitl_soft_not_run` · `soft_offline_portal_hitl_session_pass|fail` (independent of agentic list/plan soft s1422). Cross-links: journey stage 5 · agentic companion · skill continuum · [edge-user-journey.md](docs/architecture/edge-user-journey.md). Honesty: dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA · portal HITL when connect · agent MCP cannot write installs · catalog ≠ Connected · soft offline ≠ invent Connected · session soft ≠ live dogfood · residual PASS ≠ live dogfood · PASS ≠ live APPLY · free eng **s1562** · free-floor peer **s1564+** mention only.
- **Wave B first-run journey polish (s1558)** — residual-honest `/onboard next journey` lane (aliases `edge-journey` · `user-journey` · `first-run` · `edge_user_journey`) via `AionAgentOnboardingNextJourneyLane` — 7-stage edge-user-journey first-run map with primary slash/CLI for stages 3–7 · honesty one-liner locks · residual gaps. Setup lane polished as **stage 4** of journey · companion `/onboard next journey`. Setup guidance first-run mapping (stages 1–7 · in-session focus 4–7) · `SetupLifecycleFirstRunJourneyOneLiner` · skill `setup-lifecycle-agent` Wave B stamp · continuum lists (NextLanes / guidance / checklist / status). Docs: [edge-user-journey.md](docs/architecture/edge-user-journey.md) Wave B residual · [setup-lifecycle.md](docs/architecture/setup-lifecycle.md). Honesty: dual_write OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA · portal HITL · agent MCP cannot write installs · catalog ≠ Connected · book-demo OFF · no invent TUI portal SSO · host not auto · free eng **s1558** · free-floor peer **s1560+** mention only.
- **Edge user journey SSOT (s1554)** — new [docs/architecture/edge-user-journey.md](docs/architecture/edge-user-journey.md): residual-honest **7-stage** edge-first product narrative (Signup → Download TUI → TUI auth/keys → Setup wizard → Connectors/events on mesh → Local store → Analyze). Stage table · ownership map · honesty locks · residual gaps (no TUI portal SSO invent · memory host not auto on signup · portal HITL for installs). Cross-links: memory-edge-usage-demo phase mapping · setup-lifecycle stage-4 pointer · docs/README + light README edge-install row. Wave A **docs only** · free-floor peer **s1556+** mention only. Honesty: drafts only · dual_write OFF · not Memory GA · Edge Memory GA candidacy only · knowledge multi-tenant punted · Slack HMAC punted · H1/H2 not launch gate · portal HITL when connect · book-demo OFF · agent MCP cannot write installs · catalog ≠ Connected · residual PASS ≠ invent Edge Memory GA · rates ~$88/$119 · aion private · Palace sunset.
- **Edge-first human-gates residual pin (s1550)** — rewrite `AionAgentHumanGatesHonestyBoard` for **edge-first** launch residual: local TUI + `iomesh-memory-mcp` + optional mesh pull · **dual_write OFF** · knowledge multi-tenant INSTALL_STORE **punted** (H1/H2 not launch gate) · Slack HMAC **punted for now** · integrations path = TUI agent MCP list/plan + portal HITL when OAuth needed · Stripe key/ACL largely closed (ACL residual only if Dashboard regresses) · still portal HITL when connect · book-demo OFF · E10 only if claiming Edge Memory GA · Edge Memory GA **candidacy only**. Sections: architecture · still_human_or_policy · punted_or_demoted · offline_residual_only/shipped_or_policy · operator. Light cross-links: guidance human-gates step · operator matrix row 4 · TUI residual footer · skill `aion-agent-onboarding` · [docs/architecture/setup-lifecycle.md](docs/architecture/setup-lifecycle.md). Honesty: dual_write OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA · PASS ≠ invent Connected · agent MCP cannot write installs · portal HITL when connect · knowledge multi-tenant punted · Slack HMAC punted · H1/H2 not launch gate · open policy boxes stay honest.
- **Still-human APPLY reaffirm after setup closeout (s1546)** — residual-honest reaffirm on `AionAgentHumanGatesHonestyBoard` that setup lifecycle P1–P7 / `/onboard next setup` residual complete **≠** invent human-gate green · live APPLY · OAuth Connected · E10 sign-off. Adds **After setup closeout residual (s1542/s1546)** section · `setup_not_probed` still honest offline · companion `/onboard next setup` · `/setup portal`. Light cross-links: guidance note human-gates step · operator matrix row 4 · portal handoff · TUI residual footer. Honesty: dual_write OFF · not Memory GA · book-demo OFF · open boxes stay open · **E10 Open** · PASS ≠ invent human-gate green · PASS ≠ live APPLY · residual PASS ≠ invent Edge Memory GA · setup closeout residual ≠ invent APPLY. Docs: skill `aion-agent-onboarding` · [docs/architecture/setup-lifecycle.md](docs/architecture/setup-lifecycle.md).
- **Setup lifecycle closeout residual (s1542)** — residual-honest `/onboard next setup` lane (aliases `setup-lifecycle` · `wizard` · `lifecycle` · `setup_lifecycle`) consolidates setup P1–P7 map story via `AionAgentOnboardingNextSetupLane` · wired into guidance/checklist/portal handoff/NextLanes · status/export boards with **`setup_not_probed`** · TUI slash residual footer · drift notes point at guided `/setup repair plan` · `/setup repair apply --yes` (safe steps only · dual_write never auto ON) instead of pure “no auto-repair”. Honesty: dual_write OFF · not Memory GA · package wire ≠ Connected · repair apply ≠ invent Connected · portal HITL · still-human APPLY open · **E10 Open** · offline static ≠ live dogfood · setup closeout residual ≠ invent Edge Memory GA. Docs: [docs/architecture/setup-lifecycle.md](docs/architecture/setup-lifecycle.md) · [docs/architecture/memory-edge-usage-demo.md](docs/architecture/memory-edge-usage-demo.md) · skills `setup-lifecycle-agent` · `aion-agent-onboarding`.
- **Setup lifecycle guided repair surfaces (s1538 P7)** — slash `/setup repair` (bare/`plan` → `PlanRepair` + `FormatRepairPlan` from current drift · no side effects) · `/setup repair apply --yes` → `ApplyRepairPlan` safe steps only via TUI `setupRepairExecutor` (`ReloadMCP` = ConnectMCP+ReplaceMCP · `StartPull` · `StartAnalyze`) · **refuse without `--yes`** (no auto-repair) · notes for human host/mesh/dual_write never auto-applied · post-apply residual drift reprint · guidance/skill/preflight honesty flip: **repair apply ≠ invent Connected** · dual_write never auto-flipped ON · portal HITL still human · package wire ≠ Connected · dual_write OFF · not Memory GA. Docs: [docs/architecture/setup-lifecycle.md](docs/architecture/setup-lifecycle.md) · demo polish [docs/architecture/memory-edge-usage-demo.md](docs/architecture/memory-edge-usage-demo.md).
- **Setup lifecycle analyze ticks + drift surfaces (s1534 P6)** — slash `/setup analyze` (`status` · `start` · `once` · `stop`; bare = status; flags `--mode status|digest` · `--interval N` · `--window day|week` · `--config path`) builds `AnalyzeTickConfig` from `[memory] analyze_*` · slash start sets Enabled · `/setup drift` / `/setup maintain` → `FormatDriftText(BuildDriftReport(cfg, DriftSnapshot()))` report-only (residual next steps) · guidance/skill honesty flip: **analyze_continuous** opt-in · `/memory digest` still valid · analyze tick ≠ invent Connected · drift report ≠ invent install green · package wire ≠ Connected · dual_write OFF · not Memory GA. Docs: [docs/architecture/setup-lifecycle.md](docs/architecture/setup-lifecycle.md).
- **Setup lifecycle continuous pull surfaces (s1530 P5)** — slash `/setup pull` (`status` · `start` · `once` · `stop`; bare = status) builds `ContinuousPullConfig` from `[memory] pull_*` (default stream `EVENTS`) · `pull_continuous = false` in setup local-memory fragment · guidance/skill honesty flip to **in-session opt-in** continuous pull · CLI `iomesh memory pull` still valid · dual_write OFF · not Memory GA · pull ≠ invent Connected. Docs: [docs/architecture/setup-lifecycle.md](docs/architecture/setup-lifecycle.md).
- **Setup lifecycle hot MCP reload foundation (s1526 P4)** — `internal/runtimewire` (config→skill dirs + MCP ServerConfig: plugins DiscoverAll when enabled · SkillDirs · TOML primary then MCPServersFromPlugins · BuildMCPServerConfig inject) shared by agent bootstrap, `iomesh mcp --connect`, skills list, ACP. `Runtime.ReplaceMCP` closes previous manager, unregisters `mcp__*` / MCP meta tools, re-AttachMCP (system notes upsert by tag). Slash `/setup reload` → `ConnectMCP` + `ReplaceMCP`. dual_write OFF · package wire ≠ Connected. Docs: [docs/architecture/setup-lifecycle.md](docs/architecture/setup-lifecycle.md).
- **Setup lifecycle agent-native surfaces (s1526 P3)** — slash `/setup` (alias `/setup-lifecycle`: `init` · `preflight|status|check` · `portal` · `reload` via P4) · builtin skill `setup-lifecycle-agent` (`go:embed`) · system note `<setup-lifecycle>` on `AttachMCP` via `setup.SetupLifecycleAgentGuidanceNote()`. Residual-honest: dual_write OFF · not Memory GA · catalog ≠ Connected · portal HITL · setup PASS ≠ invent install green · continuous pull still CLI `iomesh memory pull`. Docs: [docs/architecture/setup-lifecycle.md](docs/architecture/setup-lifecycle.md).
- **Setup lifecycle foundation (s1525 P1–P2)** — agent-native setup path: `internal/config` managed-block write (`WriteSetupManagedFragment` refuses `dual_write=true`) · `iomesh setup init` profiles (`local-memory|plugins|mesh|platform-mcp|all`) · `iomesh setup preflight [--json]` residual-honest probes (never invent Connected/Memory GA). Docs: [docs/architecture/setup-lifecycle.md](docs/architecture/setup-lifecycle.md).
- **Advanced Memory install ladder (s1525)** — residual-honest operator guide to maximize TUI Memory benefit: baseline host + auto_recall · advanced slash · optional ONNX (`MEMORY_ONNX_MODEL_PATH` on host) · Qdrant documented as **not required** / lean host `qdrant=off`. See [docs/architecture/memory-advanced-install.md](docs/architecture/memory-advanced-install.md) · sample plugin README.
- **Memory edge usage / demo example (s1513)** — residual-honest operator walkthrough: portal signup (optional) → TUI install → MCP integrations list/plan + portal HITL → local memory install (kernel + `iomesh-memory-mcp`; **not fully automatic**) → TUI attach → show `/memory` + `iomesh mcp --connect` usage. **Honesty:** dual_write OFF · not Memory GA · Edge Memory GA candidacy only · E10 Open · install ≠ signup auto-provision · catalog ≠ Connected · aion broker private. See [docs/architecture/memory-edge-usage-demo.md](docs/architecture/memory-edge-usage-demo.md).
- **E4 MCP client attach dogfood (s1508)** — residual-honest full MCP client attach dogfood tip (docs + evidence stamp + light `/onboard next memory` mention). Runbook: lean `iomesh-memory-mcp` HTTP host → temp `[[mcp.servers]]` URL → `./bin/iomesh mcp --connect` observed **connected=1 · tools=6** (`memory_ingest_turn`, `memory_retrieve`, `memory_search_semantic`, `memory_list`, `memory_compact_status`, `memory_facts_as_of`). **Pinned stamp:** UTC `2026-08-09T06:23:34Z` · TUI tip `6b3958a90a01d2c8f50ee161c8dc1009637b64f1` · MCP tip `f46afe2462ebaa94890b30296b1a19d03d6853da` (`f46afe2`). **Honesty:** local-primary · dual_write OFF · **Edge Memory GA candidacy only** · residual PASS ≠ invent Edge Memory GA declared · not bare Memory GA · not hosted Memory GA · aion broker private · **E10 Open** · tip ≠ invent forever-green product dogfood · tip ≠ invent E10. See [docs/architecture/memory-mcp.md](docs/architecture/memory-mcp.md) · [docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md](docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md).
- **Public edge Memory product attach continuum (s1478)** — residual-honest TUI tip after both edge repos public (`github.com/iome-sh/memory` + `github.com/iome-sh/iomesh-memory-mcp`). Surfaces: `/onboard next memory` lane + residual print · product sample plugin `examples/agent-plugins/iomesh-memory-mcp` · dogfood primary samples `hello-iome` + `iomesh-memory-mcp` · docs/README/config. Install: `go install github.com/iome-sh/iomesh-memory-mcp/cmd/iomesh-memory-mcp@main` · `go get github.com/iome-sh/memory@main` · **no GOPRIVATE** · attach HTTP `http://127.0.0.1:8080/mcp` or stdio · docker compose still valid. **Honesty:** dual_write OFF · not Memory GA · aion broker **still private** · flip complete residual ≠ invent Memory GA · public OSS ≠ invent platform GA · residual monorepo `aion-memory-mcp` ≠ product naming. See [docs/architecture/memory-mcp.md](docs/architecture/memory-mcp.md).
- **M4 public flip readiness tip for edge Memory OSS (s1469)** — residual-honest TUI readiness tip only (docs + `/onboard next memory` lane + tests). Order: kernel `github.com/iome-sh/memory` **first** · then `iomesh-memory-mcp` · readiness docs/gates in those repos (mention only). **Honesty:** readiness tip ≠ invent public flip complete · residual PASS ≠ invent public flip · dual_write OFF · not Memory GA · aion broker private · M5 signing later after flip · keeps Option A (s1453) + M2 lean attach (s1458) + M3 edge dogfood (s1463). See [docs/architecture/memory-mcp.md](docs/architecture/memory-mcp.md). *(Superseded for operator tip by s1478 public product attach — edge packs now public.)*
- **M3 edge dogfood tip for iomesh-memory-mcp (s1463)** — residual-honest TUI↔product edge Memory MCP dogfood surfaces (docs + `/onboard next memory` lane + config example + tests). Path: build/run from `github.com/iome-sh/iomesh-memory-mcp` · `docker compose up --build` → image `iomesh-memory-mcp:local` · attach `http://127.0.0.1:8080/mcp` · healthz · stdio alternate · peer mcp `make edge-dogfood-gate` (mention only). **Honesty:** dual_write OFF · not Memory GA · host/kernel **public as of s1478** · aion broker private · offline dogfood tip ≠ invent live dogfood as green · PASS ≠ invent full platform sidecar parity · residual PASS ≠ invent public flip · M3 after M2 scaffold · M4 public flip later deliberate · keeps Option A (s1453) + M2 lean attach (s1458). See [docs/architecture/memory-mcp.md](docs/architecture/memory-mcp.md).
- **Memory facts-as-of bi-temporal lite (s1276)** — opt-in validity listing via MCP `memory_facts_as_of` (aion Beta K4 lite); slash `/memory facts-as-of|facts|as-of --as-of <RFC3339> [--entity …] [--query …] [--limit N]`. MCP-first (no lean HTTP `/memory/facts_as_of` route today). Residual-honest format + offline fail-open. **Honesty:** bi-temporal lite · not full dual-clock Graphiti · not Memory GA · dual_write OFF · empty ≠ invent memories · not auto-recall
- **MCP opt-in iomesh multi-tenant context headers (s1267)** — `[mcp] inject_iomesh_context` (default **false**) or per `[[mcp.servers]] inject_iomesh_context` merges non-empty `X-IOMesh-Tenant` / `X-IOMesh-Org` / `X-IOMesh-Workspace` into HTTP MCP `ServerConfig.Headers` at build time (`config.BuildMCPServerConfig` · `mcp.ApplyIOMeshContextHeaders`). Tenant from `[iomesh].tenant` else `[memory].tenant`; org/workspace from `[iomesh]`. Never overwrites explicit server headers (case-insensitive); empty values not sent; stdio has no HTTP headers (inject no-op). Wire: agent AttachMCP, `iomesh mcp --connect`, memory pull MCP, ACP. Residual honesty: inject ≠ install APPLY/Connected/INSTALL_STORE green · ≠ dual-auth install list · dual_write OFF · book-demo OFF · no invent GA. See [docs/architecture/mcp.md](docs/architecture/mcp.md).
- **streams print JSON completeness pin (s759)** — docs + unit tests lock the complete streams print JSON surface after s720 StreamMessagesPrint + s723 StreamMessagePrint + s726 StreamDeletePrint: always-emit messages envelope keys `{stream,from_seq,to_seq,limit,count,messages}` (nested `messages[]` item keys `{stream,seq,subject,partition,payload,headers,timestamp}` with empty/0/`""`/`{}` honest; `messages []` not null), delete keys `{ok,name}` (no pull_role invent). Completeness pin only — **does not** invent new DTO fields or re-claim s720/s723/s726 product bodies. Peer aion s758 residual. Beta · offline unit ≠ live APPLY · dual_write default OFF · envelope ≠ invent message success · item ≠ invent message success · DTO ≠ invent stream gone · wire StreamMessage lean · not full mesh RBAC GA
- **mutate/print JSON completeness pin (s756)** — docs + unit tests lock the complete mutate/print JSON surface after s738 UsagePrint + s732 PubPrint + s729 KVPutPrint/KVDeletePrint: always-emit usage keys `{started,as_of,calls,errors,tokens,est_usd,by_model}` (zero-time `""` never `0001-01-01`; `by_model []` not null), pub keys `{ok,subject,bytes}` (no payload), put keys `{ok,bucket,key,revision}`, delete keys `{ok,bucket,key}`. Completeness pin only — **does not** invent new DTO fields or re-claim s729/s732/s738 product bodies. Peer aion s755 residual. Beta · offline unit ≠ live APPLY · dual_write default OFF · DTO ≠ invent usage/pub/kv success · local process meter ≠ remote dashboard · ephemeral pub ≠ durable stream publish · s714 ≠ mutate residual · not full mesh RBAC GA
- **catalog print JSON completeness pin (s753)** — docs + unit tests lock the complete catalog print JSON surface after s735 CatalogPrint (list) + s744 CatalogProductPrint (detail): always-emit list keys `{source,detail,query,count,products}`, detail keys `{source,detail,id,found,product}`, nested `DataProductPrint` subjects/lineage `[]` not null. Completeness pin only — **does not** invent new DTO fields or re-claim s735/s744 product bodies. Peer aion s752 residual. Beta catalog · offline unit ≠ live APPLY · dual_write default OFF · DTO ≠ invent catalog/product success · `found=false` honest · s735 list ≠ product detail · not full mesh RBAC GA · portal federation not invent GA
- **Format\*JSON helper completeness pin (s750)** — docs + unit tests lock the complete s741 Format\*JSON helper surface: `FormatStreamMessagesJSON` · `FormatStreamInfoJSON` · `FormatStreamInfoListJSON` · `FormatConsumerInfoJSON` · `FormatKVBucketInfoJSON` · `FormatKVEntryJSON` · `FormatKVKeysJSON` (keys present · trailing newline · nil list → `[]` not null). Completeness pin only — **does not** invent new helpers/fields or re-claim s741 product body. Peer aion s749 residual. Beta · offline unit ≠ live APPLY · dual_write default OFF · helper completeness ≠ invent new DTO fields · CLI prefer Format\*JSON · empty/`[]` honest · not full mesh RBAC GA
- **`iomesh memory pull` process-evidence completeness pin (s747)** — docs + unit tests lock the complete `MemoryPullStatsPrint` surface: identity (s705) + knobs/counters + process evidence (s717) with always-emit `result` / `exit_code` / `endpoint` / `org` / `workspace` on empty and populated JSON paths. Completeness pin only — **does not** invent new DTO fields or re-claim s717 product body. Peer aion s746 residual. Beta · offline unit ≠ live APPLY · dual_write default OFF · process evidence ≠ invent pull success · empty honest · not full mesh RBAC GA
- **`mesh catalog --id` CatalogProductPrint always-emit (s744)** — `CatalogProductPrint` / `NewCatalogProductPrint` / `FormatCatalogProductJSON` always emit single-product detail envelope `{source,detail,id,found,product}` with nested `DataProductPrint` (empty string / `0` / `[]` / `false` honest; subjects/lineage never null; `found=false` when missing — empty product fields, no invent). CLI `iomesh mesh catalog --id ID [--json]` uses `GetCatalogProduct`; `--json` → CatalogProductPrint; text → `FormatProductDetail`. List path (`--id` omitted) stays CatalogPrint (s735). Exit 1 when `Source=="off"`; fail-open not-found keeps exit 0 (`found=false` honesty). Wire `DataProduct` / `GetCatalogProduct` tags unchanged. Mold CatalogPrint s735 + PubPrint s732. Peer aion s743 residual. Beta catalog · offline unit ≠ live APPLY · dual_write default OFF · fail-open source honest · not full mesh RBAC GA · portal federation not invent GA · DTO ≠ invent catalog/product success · s735 list ≠ product detail residual · no invent GA
- **Format*JSON helper completeness for existing always-emit print DTOs (s741)** — adds `FormatStreamMessagesJSON` / `FormatStreamInfoJSON` / `FormatStreamInfoListJSON` (nil → `[]` not null) / `FormatConsumerInfoJSON` / `FormatKVBucketInfoJSON` / `FormatKVEntryJSON` / `FormatKVKeysJSON` (MarshalIndent + trailing newline + marshal-error fallback mold like `FormatPubJSON`). CLI `cmd/iomesh` `--json` success paths for streams `--messages` / get / list, consumer create, kv create-bucket / get / list now prefer these helpers over ad-hoc `json.MarshalIndent`. Does **not** invent new DTO fields or re-claim s714/s720/s723/s726/s729/s732/s735/s738 product always-emit bodies — helper surface + CLI wire only. Peer aion s740 residual. Beta · offline unit ≠ live APPLY · dual_write default OFF · empty/0/`[]` honest · not full mesh RBAC GA
- **`mesh usage --json` UsagePrint always-emit (s738)** — `ModelUsagePrint` / `UsagePrint` / `NewUsagePrint` / `FormatUsageJSON` always emit `{started,as_of,calls,errors,tokens,est_usd,by_model[]}` and per-model `{model,calls,errors,prompt_tokens,completion_tokens,total_tokens,est_usd,duration_ms}` (zero times → `""` never `"0001-01-01T00:00:00Z"`; by_model empty `[]` not null; empty/`0` honest) so CI scrapers get a stable usage JSON surface without omitempty / zero-time gaps. CLI `iomesh mesh usage [--json]` keeps call site `FormatUsageJSON(UsageSnapshot)` (internally maps via `NewUsagePrint`); text `FormatUsage` unchanged. Wire `UsageSnapshot` / `ModelUsage` stay as in-process rollup. Also fixes stale mesh-help Flags (pub) `--json` line to PubPrint always-emit (s732 honesty lag). Mold CatalogPrint s735 + PubPrint s732 + KVPutPrint s729. Peer aion s737 residual. Beta · offline unit ≠ live APPLY · dual_write default OFF · not full mesh RBAC GA · local process meter ≠ remote dashboard · empty-time honest · DTO ≠ invent usage/meter success
- **`mesh catalog --json` CatalogPrint always-emit (s735)** — `DataProductPrint` / `CatalogPrint` / `NewDataProductPrint` / `NewCatalogPrint` / `FormatCatalogJSON` always emit list envelope `{source,detail,query,count,products}` and per-product `{id,name,title,description,subject,layer,status,department,subjects,lineage}` (empty string / `0` / `[]` honest; subjects/lineage never null) so CI scrapers get a stable catalog JSON surface without omitempty gaps. CLI `iomesh mesh catalog [--query q] [--json]` prints CatalogPrint on `--json`; text `FormatCatalog` unchanged. Wire `DataProduct` stays lean omitempty; `CatalogResult` stays untagged. Exit 1 when `Source=="off"` unchanged. Mold PubPrint s732 + StreamMessagesPrint s720 + KVKeysPrint s714. Peer aion s734 residual. Beta catalog · offline unit ≠ live APPLY · dual_write default OFF · fail-open source honest · not full mesh RBAC GA · portal federation not invent GA · DTO ≠ invent catalog/product success · wire omitempty ≠ print always-emit
- **`mesh pub` PubPrint always-emit (s732)** — `PubPrint` / `NewPubPrint` / `FormatPub` / `FormatPubJSON` always emit `{ok,subject,bytes}` (bytes `0` honest when unset; empty subject honest) so CI scrapers get a stable envelope on ephemeral pub success without omitempty gaps / ad-hoc `map[string]any`. CLI `--subject S --payload STR|--payload-file F --yes [--json]` prints the DTO on success only (FAIL stays stderr); wire `Pub` stays lean error-return. No `pull_role` invent and no payload echo. Ephemeral `POST /v1/pub` ≠ durable stream publish. Mold StreamDeletePrint s726 + KVPutPrint s729. Peer aion s731 residual. Beta · offline unit ≠ live APPLY · empty/0 honest · dual_write default OFF · not full mesh RBAC GA · DTO ≠ invent pub success when HTTP failed
- **`mesh kv --put` / `--delete` JSON always-emit print DTOs (s729)** — closes s714 mutate half-gap. `KVPutPrint` / `NewKVPutPrint` / `FormatKVPut` / `FormatKVPutJSON` always emit `{ok,bucket,key,revision}` (revision `0` honest when unset; empty bucket/key honest). `KVDeletePrint` / `NewKVDeletePrint` / `FormatKVDelete` / `FormatKVDeleteJSON` always emit `{ok,bucket,key}`. CLI `--put … --yes [--json]` / `--delete … --yes [--json]` prints the DTO on success only (FAIL stays stderr); wire `KVPut` / `KVDelete` stay lean. No `pull_role` invent and no value echo on put JSON. Mold StreamDeletePrint s726 + s714 read DTOs. Peer aion s728 residual. Beta · offline unit ≠ live APPLY · empty/0 honest · dual_write default OFF · not full mesh RBAC GA · DTO ≠ invent mutate success when HTTP failed
- **`mesh streams --delete` StreamDeletePrint always-emit (s726)** — `StreamDeletePrint` / `NewStreamDeletePrint` / `FormatStreamDelete` / `FormatStreamDeleteJSON` always emit `{ok,name}` (empty string honest when unset) so CI scrapers get a stable envelope on delete success without omitempty gaps. CLI `--delete --name N --yes [--json]` prints the DTO on success only (FAIL stays stderr); wire `DeleteStream` stays lean error-return. No `pull_role` invent (stream delete ≠ consumer pull-auth). Mold ConsumerDeletePrint s708. Peer aion s725 residual. Beta · offline unit ≠ live APPLY · empty honest · dual_write default OFF · not full mesh RBAC GA · DTO ≠ invent delete success when HTTP failed
- **`StreamMessagePrint` nested always-emit for scrapers (s723)** — residual after s720 outer envelope: nested `messages[]` now map via `StreamMessagePrint` / `NewStreamMessagePrint` so scrapers always see `stream` (`""` honest), `seq`, `subject`, `partition` (`0` honest), `payload` (base64 []byte wire behaviour; nil → empty), `headers` (nil → `{}`), and `timestamp` as string (`""` when zero; RFC3339 UTC when set — KVEntryPrint mold). `StreamMessagesPrint.Messages` and `ConsumerFetchPrint.Messages` are `[]StreamMessagePrint` (converted on New*). Wire `StreamMessage` stays lean omitempty. Mold StreamInfoPrint s699/s702 / KVEntryPrint s714. Peer aion s722 residual. Beta · offline unit ≠ live APPLY · empty/0/`""`/`{}` honest · dual_write default OFF · not full mesh RBAC GA · does not invent message success from fields alone
- **`mesh streams --messages --json` always-emit print envelope (s720)** — `StreamMessagesPrint` / `NewStreamMessagesPrint` / `FormatStreamMessagesPrint` always emit `stream`, `from_seq` / `to_seq` / `limit` (0 honest when unset), `count`, and `messages` (empty array when none) so CI scrapers get a stable envelope rather than a bare `[]StreamMessage`. CLI `--messages --json` marshals the print DTO; text path includes knobs in the header. Wire `StreamMessage` stays lean omitempty. Mold KVKeysPrint s714 / ConsumerFetchPrint s708. Peer aion s719 residual. Nested message always-emit closed by s723. Beta · offline unit ≠ live APPLY · empty/0 honest · dual_write default OFF · not full mesh RBAC GA · does not invent message success from knobs alone
- **`iomesh memory pull` process evidence always-emit (s717)** — `MemoryPullStatsPrint` / `FormatMemoryPullStats` / `FormatMemoryPullStatsJSON` always emit process evidence on top of s705 identity+knobs+counters: `endpoint` / `org` / `workspace` (empty string honest when unset from `[iomesh]`), `result` (`ok`|`err`), `exit_code` (0 success / 1 hard or soft fail), `duration_ms` (0 if not timed), `ack` knob. CLI sets result/exit on success, hard-fail (non-cancel), soft-fail (`errors>0 && ingested==0 && !dryRun`); emits print DTO on early fail paths when possible (mesh disabled / MCP missing). Peer aion s716 residual after s705 identity. Beta · offline unit ≠ live APPLY · dual_write default OFF · process evidence ≠ invent pull success · not full mesh RBAC GA
- **`mesh kv` create-bucket/get/list `--json` always-emit print DTOs (s714)** — closes KV JSON half-gap after text FormatKV always-emit (s560). `KVBucketInfoPrint` / create-bucket `--json` always emit `name`, `history` (0), `max_bytes` / `ttl_seconds` (0 when wire `*int64` nil). `KVEntryPrint` / get `--json` always emit `bucket`, `key`, `value` (base64; empty when nil), `revision`, `created_at` (`""` when zero; RFC3339 when set). `KVKeysPrint` list envelope always emits `{bucket, prefix, count, keys}` (not bare string array). Wire `KVBucketInfo` / `KVEntry` stay lean omitempty. Mold StreamInfoPrint s699/s702. Peer aion s713 lifecycle completeness. Beta · offline unit ≠ live APPLY · empty/0 honest · dual_write default OFF · does not invent KV success from knobs alone
- **`mesh consumer ack|nack` pull identity always-emit (s711)** — `ConsumerAckPrint` / `FormatConsumerAck` / `FormatConsumerAckJSON` always emit identity (`ok`, `op` ack|nack, `stream`, `name`, `pull_role`, `pull_allow_suffix` — empty string honest when unset), `seqs`, `ack_floor`, and `count`. CLI ack/nack `--json` marshals the print DTO; text PASS always prints pull identity lines (peer create FormatConsumerInfo s696 + fetch/delete s708 + memory-pull s705). Uses resolved s684 Client/flag role+suffix. Peer aion s710 residual. Beta · offline unit ≠ live APPLY · dual_write default OFF · fail-open empty role · not full mesh RBAC GA · does not invent ack success from identity fields alone
- **`mesh consumer fetch` + `delete` pull identity always-emit (s708)** — `ConsumerFetchPrint` / `FormatConsumerFetch` / `FormatConsumerFetchJSON` always emit identity (`stream`, `name`, `pull_role`, `pull_allow_suffix` — empty string honest when unset), knobs (`batch`, `max_wait_ms`), `count`, and `messages` (wire messages stay lean). CLI fetch `--json` marshals the print DTO (not raw `[]StreamMessage`). `ConsumerDeletePrint` / delete text+JSON always emit `{ok,stream,name,pull_role,pull_allow_suffix}`. Text paths always print pull_role/suffix summary lines (peer create FormatConsumerInfo s696 + memory-pull s705). Uses resolved s684 Client/flag role+suffix. Peer aion s707 gate completeness. Beta · offline unit ≠ live APPLY · dual_write default OFF · fail-open empty role · not full mesh RBAC GA · does not invent fetch/delete success from identity fields alone
- **`iomesh memory pull` identity + stats always-emit + `--json` (s705)** — `MemoryPullStatsPrint` / `FormatMemoryPullStats` / `FormatMemoryPullStatsJSON` always emit identity (`stream`, `consumer`, `filter_subject`, `pull_role`, `pull_allow_suffix`, `tenant` — empty string honest when unset), knobs (`dry_run`, `dual_write` report-only default false, `batch`, `max_wait_ms`, `once`), and counters (`create_ok` / loops / fetched / ingested / skipped / acked / errors / `last_error`). CLI `--json` marshals the print DTO; text PASS/summary always prints identity lines (not only stderr start log). Peer create FormatConsumerInfo s696 + status/wait pull identity continuum; peer aion s704 sales claim suite/retention honesty. Beta · offline unit ≠ live APPLY · dual_write default OFF · fail-open empty role/tenant · not full mesh RBAC GA · does not invent pull success from identity fields alone
- **`mesh stream` retention_tier decode + always-emit (s702)** — wire `StreamInfo.RetentionTier` (`json:"retention_tier,omitempty"`) decodes broker product tier (hot|temp|extended|archive). `StreamInfoPrint` / `FormatStreamDetail` / list `--json` always emit `retention_tier` (empty string when broker omits — never invent from `max_age` alone). `FormatStreams` list table adds TIER column. Closes s699 list-JSON half-gap (marshals `[]StreamInfoPrint`). Peer aion s701 mesh-stream-retention residual. Beta · offline unit ≠ live APPLY · does not invent freemium unlimited retain · dual_write default OFF · not full mesh RBAC GA
- **`mesh stream` get/detail FormatStreamDetail retention knobs always-emit (s699)** — `FormatStreamDetail` always emits `description` / `retention` (empty when unset), `partitions` / `max_msgs` / `max_age_sec` (numeric `0` when unset / `*int64` nil) for CI scrapers. `StreamInfoPrint` + mesh stream get `--json` always-emits the same knobs without omitempty gaps (wire `StreamInfo` stays lean). `FormatStreams` list table always prints MAX_MSGS / MAX_AGE columns (`0` when nil). Superseded for `retention_tier` by s702. Beta · offline unit ≠ live APPLY · peer aion s698 cost-max residual suite continuum
- **`mesh consumer create` FormatConsumerInfo pull_role identity always-emit (s696)** — `FormatConsumerInfoWithAuth` / create text+JSON always emit `pull_role` / `pull_allow_suffix` (empty string when unset) next to `filter_subject` from resolved s681/s684 auth. CLI print DTO keeps wire `ConsumerInfo` free of auth fields. CI scrapers can key stable identity without omitempty gaps. Beta federated ACL; fail-open empty; dual_write default OFF; not full mesh RBAC GA; peer aion s695 sales claim continuum
- **`mesh wait` pull_role identity always-emit (s693)** — always emit `pull_role` / `pull_allow_suffix` (empty string when unset) in mesh wait text and JSON from Client Config (`[memory].pull_role` / `pull_allow_suffix` wired onto Client like status s690). CI scrapers can key stable identity without omitempty gaps. Beta federated ACL; fail-open empty; dual_write default OFF; not full mesh RBAC GA; peer aion s692 Ops Pack floors residual gate continuum
- **`mesh status` pull_role identity always-emit (s690)** — always emit `pull_role` / `pull_allow_suffix` (empty string when unset) in mesh status text and JSON from Client Config (`[memory].pull_role` / `pull_allow_suffix` wired onto Client like dogfood s687). CI scrapers can key stable identity without omitempty gaps. Beta federated ACL; fail-open empty; dual_write default OFF; not full mesh RBAC GA; peer aion s689 residual gate continuum
- **Memory role default filter + dogfood pull identity (s687)** — `DefaultMemoryPullFilterForRole` role=`memory` → `tenant.memory.>` when tenant set (peer aion s686). Dogfood report always-emits `pull_role` / `pull_allow_suffix` from Client Config (empty string when unset); dogfood CLI wires `[memory].pull_role` / `pull_allow_suffix` onto Client so consumer create/fetch send headers + role-aware empty-filter default. Help text lists `memory` next to agent|viewer|…. Beta; fail-open without role; dual_write default OFF; not full mesh RBAC GA
- **`iomesh mesh consumer fetch` role + allow-suffix headers (s684)** — `--role` / `--pull-allow-suffix` (config fallbacks `[memory].pull_role` / `pull_allow_suffix`) set client auth headers on fetch (and ack/nack/delete for defense-in-depth). Same path as create s681; logs effective role/suffix on fetch when set. Beta; fail-open without role; not full mesh RBAC GA; peer aion s683 continuum
- **`iomesh mesh consumer create` role + default filter (s681)** — `--role` / `--pull-allow-suffix` (config fallbacks `[memory].pull_role` / `pull_allow_suffix`) set client auth headers; empty `--filter` uses role-aware `DefaultMemoryPullFilterForRole` (same as memory pull s678). Beta; fail-open without role; not full mesh RBAC GA
- **`iomesh memory pull` (s652 cost-max M1)** — durable mesh consumer → map envelopes → local MCP `memory_ingest_turn` (optional `--dry-run` / `--once`); config `[memory] pull_*`; dual_write remains optional audit default OFF; hosted Palace sunset until scale ([docs/architecture/memory-mcp.md](docs/architecture/memory-mcp.md))
- **Ops heartbeat digest export (s1200)** — opt-in ops digest via lean HTTP `POST /v1|/v5/memory/ops_digest` (`ExportOpsDigest`) with MCP `ops_digest_export` fallback; slash `/memory digest [--window day|week] [--horizon ops|knowledge|analytical|all] [--limit N]`. Human-readable patterns + receipts + honesty line. **Honesty:** ops GA-path · knowledge/analytical Beta · never invent GA · dual_write default OFF · not product Memory GA · not full graph RAG · fail-open

### Changed

- **README open-source boundary (local-first)** — refresh “this public repo is / is not” for OSS readers: first-class **local memory** (MCP host + kernel), multi-provider LLMs / Ollama, optional mesh client. Drop residual-internal “freemium palace / invent Memory GA” framing from the public is/is-not table. Simplify Edge install docs table and “Why this project”. Align packaging SSOT table. dual_write default OFF unchanged.
- **README install pin + slash surface (s1670)** — `go install …@v0.72.0` (releases may be newer; `@latest` Beta note) · slash list includes `/setup` · `/onboard` · `/memory` · `/integrations`.
- **README OSS boundary (edge-first honesty)** — lead with **MIT OSS agent harness + optional mesh client surface — not the hosted multi-tenant mesh control plane**; table of what this public repo is / is not; status line separates shipped harness from **internal roadmap** (private control plane, install-store fleet APPLY, knowledge multi-tenant INSTALL_STORE punted for edge-first); edge install docs table; residual serial stamps framed as deep-doc labels not control-plane claims. dual_write OFF · not Memory GA · public OSS ≠ invent platform GA.
- **Public mesh/plugins smoke rename (s1521)** — public CLI prefers `iomesh mesh smoke` and `iomesh plugins smoke` (also `check`); `dogfood` / `probe` remain **legacy aliases**. README, CONTRIBUTING, Makefile (`smoke` / `smoke-unit`), and mesh smoke docs updated so public copy no longer leads with internal “dogfood”. dual_write OFF · not Memory GA · smoke ≠ invent Connected/GA.
- **Drop residual aion Memory sample (s1517)** — remove in-tree `examples/agent-plugins/aion-memory-mcp` and retarget product Memory attach/docs/onboard/config/skills to **`iomesh-memory-mcp` only**. Dogfood samples remain `hello-iome` + `iomesh-memory-mcp`. **Honesty:** dual_write OFF · not Memory GA · aion broker/CP private · no invent platform GA · s1517 product-only sample. See [docs/architecture/memory-mcp.md](docs/architecture/memory-mcp.md) · [docs/architecture/agent-plugins.md](docs/architecture/agent-plugins.md).

## [0.71.0] — 2026-08-03

Minor release: multi-hop related memory recall + temporal retrieve options + short-TTL recall cache.

### Added

- **Multi-hop related memory recall (s1135)** — opt-in multi-hop lite related recall via lean HTTP `POST /v1|/v5/memory/related` (`RetrieveMemoryRelated`) with MCP `memory_related` fallback; slash/CLI `/memory related --seed … [--query …] [--max-hops N]`. Default auto-recall remains single-hop. **Honesty:** multi-hop lite ≠ full graph RAG · not product Memory GA · dual_write default OFF · hop ranking lite · fail-open
- **Memory recall short-TTL cache + efficiency (s1069)** — process-local fail-open short-TTL cache for sync `RetrieveMemory` (`[memory] recall_cache_ttl_ms`, default 3000; `0` disables), keyed by tenant+session+query+limit+since/until; snippet early-stop at `max_snippet_bytes`; auto-recall always-emits retrieve latency (`Nms` / `Nms cache`). Not product Memory GA · dual_write default OFF
- **Temporal retrieve since/until/session_seq (s1068)** — wire platform sidecar temporal filters on `POST /v1/memory/retrieve` via `MemoryRetrieveOptions` / `RetrieveMemoryWithOptions`; `[memory] recall_since` / `recall_until` / `recall_session_seq` + env overrides; `/memory recall --since/--until/--session-seq` flags; MCP fallback forwards the same keys. Fail-open unchanged · dual_write default OFF · does not invent temporal pipeline GA

## [0.70.0] — 2026-07-22

Minor release: FormatProductDetail always-emit for optional knobs.

### Changed

- **FormatProductDetail always-emit** — always emit status/department/description/lineage/subjects (empty/(none) honest when unset) for operator/CI scrapers

## [0.69.0] — 2026-07-22

Minor release: Format stream/consumer always-emit for optional knobs and filter_subject.

### Changed

- **FormatStreamDetail always-emit** — always emit optional stream knobs (description, retention, partitions, max_msgs, max_age_sec, created_at, subjects; empty/zero/blank honest when unset)
- **FormatConsumerInfo filter_subject always-emit** — always emit `filter_subject` (empty when unset)

## [0.68.0] — 2026-07-22

Minor release: FormatKV always-emit for entry `created_at` and bucket knobs.

### Changed

- **FormatKVEntry `created_at` always-emit** — always emit `created_at` (RFC3339 UTC when set; blank when zero/unset) so operator/CI scrapers can key a stable field without omitempty gaps; peers mesh/stream format always-emit continuum
- **FormatKVBucketInfo optional knobs always-emit** — always emit `history`, `max_bytes`, `ttl_seconds` (`0` / blank when unset; `*int64` nil prints blank after the colon rather than omitting the line) for operator/CI scrapers

## [0.67.0] — 2026-07-22

Minor release: dogfood step latency_ms always-emit.

### Added

- **Dogfood step `latency_ms` always-emit** — always emit per-step `latency_ms` (int milliseconds; `0` when zero / not timed) in dogfood JSON reports alongside existing string `latency` so CI scrapers who marshal steps natively get a stable numeric field without omitempty gaps; set whenever step `Latency` is set (`stepTimed`); text report unchanged (duration still shown in parens when timed)

## [0.66.0] — 2026-07-22

Minor release: mesh wait error always-emit.

### Added

- **`mesh wait` error always-emit** — always emit `error` (empty string when OK) in text and JSON so CI scrapers can key on a stable field without omitempty gaps; text always prints `error:` after identity; peers result / exit_code / identity always-emit continuum

## [0.65.0] — 2026-07-22

Minor release: dogfood step detail and latency always-emit.

### Added

- **Dogfood step detail/latency always-emit** — always emit per-step `detail` (empty string when unset) and `latency` (duration string; empty string when zero) in dogfood JSON reports so CI scrapers can key on stable step fields without omitempty gaps; text report already prints steps with empty detail; peers identity / probe-err / policy_allow always-emit continuum

## [0.64.0] — 2026-07-21

Minor release: dogfood policy_allow always-emit.

### Added

- **Dogfood policy_allow always-emit** — always emit top-level `policy_allow` as string `"true"` | `"false"` | `""` (empty when policy mode off / not evaluated / mesh disabled before step) in dogfood text and JSON reports so CI scrapers can key on a stable field without omitempty gaps; text always prints `policy_allow:` after `policy_source:`; empty-honest when unevaluated (does not invent a decision); peers health_err / ready_err / policy_source / memory_endpoint always-emit continuum

## [0.63.0] — 2026-07-21

Minor release: dogfood kv/consumer identity always-emit.

### Added

- **Dogfood kv/consumer soft-probe identity always-emit** — always emit top-level `kv_bucket`, `consumer_stream`, `consumer_name`, and `consumer_filter` (empty string when unset / partial / probe not configured) in dogfood text and JSON reports so CI scrapers can key on stable identity fields without omitempty gaps; text always prints all four lines; empty identity does not invent probe success (pair with `kv_key_count` / `kv_ensured` / `consumer_probed` / `consumer_ok`); peers identity / memory_endpoint / catalog_source always-emit continuum

## [0.62.0] — 2026-07-21

Minor release: dogfood catalog_source and policy_source always-emit.

### Added

- **Dogfood catalog_source / policy_source always-emit** — always emit top-level `catalog_source` and `policy_source` (empty string when unset / mesh disabled before step) in dogfood text and JSON reports so CI scrapers can key on stable source fields without omitempty gaps; text always prints `catalog_source:` and `policy_source:` lines; peers identity / memory_endpoint / health_err always-emit continuum

## [0.61.0] — 2026-07-21

Minor release: dogfood memory_endpoint always-emit evidence.

### Added

- **Dogfood memory_endpoint always-emit** — always emit top-level `memory_endpoint` (empty string when `[memory].endpoint` / `IOMESH_MEMORY_ENDPOINT` unset) in dogfood text and JSON reports so CI scrapers can key on a stable field without omitempty gaps; text always prints `memory_endpoint:` after identity; empty-honest when unset (does not invent memory plane readiness); peers identity always-emit mold and SDK SUMMARY+RESULT `base_url` continuum

## [0.60.0] — 2026-07-21

Minor release: dogfood probe-err always-emit evidence.

### Added

- **Dogfood probe-err always-emit** — always emit top-level `health_err` and `ready_err` (empty string when health/ready PASS, clean skip without err detail, or mesh disabled) in dogfood text and JSON reports so CI scrapers can key on stable probe-error fields without omitempty gaps; text uses dedicated `health_err:` / `ready_err:` lines after `health_ms` / `ready_ms`; peers mesh status probe-err and SDK ConnectionStatus always-emit continuum

## [0.59.0] — 2026-07-21

Minor release: mesh status probe-err always-emit evidence.

### Added

- **`mesh status` probe-err always-emit** — always emit `health_err` and `ready_err` (empty string when probe OK / skipped) in text and JSON so CI scrapers can key on stable probe-error fields without omitempty gaps; text uses dedicated `health_err:` / `ready_err:` lines (detail no longer inlined on `health:` / `ready:`); peers SDK ConnectionStatus always-emit continuum

## [0.58.0] — 2026-07-21

Minor release: mesh wait identity always-emit evidence.

### Added

- **`mesh wait` identity always-emit** — always emit `endpoint`, `tenant`, `org`, and `workspace` (empty string when unset) in text and JSON so CI scrapers can key on stable identity fields without omitempty gaps; peers mesh status identity continuum; does not invent readiness from identity
- **Dogfood identity always-emit** — always emit `tenant`, `org`, and `workspace` (empty string when unset) in dogfood text and JSON reports so CI scrapers can key on stable identity fields without omitempty gaps; peers mesh status identity always-emit continuum

## [0.57.0] — 2026-07-21

Minor release: mesh status identity always-emit evidence.

### Added

- **`mesh status` identity always-emit** — always emit `endpoint`, `tenant`, `org`, and `workspace` (empty string when unset) in text and JSON so CI scrapers can key on stable identity fields without omitempty gaps; peers dogfood/mesh wait always-emit continuum

## [0.56.0] — 2026-07-20

Minor release: dogfood always-emit user_agent evidence.

### Added

- **Dogfood user_agent always-emit** — always emit `user_agent` (package mesh HTTP User-Agent via `iomesh.UserAgent()`, default `iomesh-tui`; empty string when unset) in dogfood text and JSON reports so CI scrapers record the outbound mesh UA without re-parsing flags; peers mesh wait/status always-emit continuum

## [0.55.0] — 2026-07-20

Minor release: mesh wait result evidence.

### Added

- **`mesh wait` result evidence** — always emit `result` (`ok` when wait OK / `err` when wait fails; derived from OK / waitErr only) in text and `--json` output so CI scrapers peer mesh status `result` continuum without inventing readiness schema

## [0.54.0] — 2026-07-20

Minor release: mesh wait user_agent evidence.

### Added

- **`mesh wait` user_agent evidence** — always emit `user_agent` (package mesh HTTP User-Agent via `iomesh.UserAgent()`, default `iomesh-tui`) in text and `--json` output so CI scrapers record the outbound mesh UA without re-parsing flags; peers mesh status/dogfood

## [0.53.0] — 2026-07-20

Minor release: mesh wait version evidence.

### Added

- **`mesh wait` version evidence** — always emit `version` (package product/binary version via `ProductVersion()`, empty string when unset) in text and `--json` output so CI scrapers record the CLI build without shell probes; peers mesh status/dogfood

## [0.52.0] — 2026-07-20

Minor release: mesh wait exit_code evidence.

### Added

- **`mesh wait` exit_code evidence** — always emit `exit_code` (int process exit matching `cmdMeshWait`: `0` when `ok=true`, `1` when `ok=false`) in text and `--json` output so CI scrapers record the intended exit without shell `$?`

## [0.51.0] — 2026-07-20

Minor release: dogfood exit_code evidence.

### Added

- **Dogfood exit_code evidence** — always emit `exit_code` (int process exit matching `cmdMeshDogfood`: `0` when `ok=true`, `1` when `ok=false`) in text and JSON dogfood reports so CI scrapers record the intended exit without shell `$?`

## [0.50.0] — 2026-07-20

Minor release: mesh status exit_code evidence.

### Added

- **`mesh status` exit_code evidence** — always emit `exit_code` (int process exit that `MeshStatusExitCode(strict, result)` would return: `0` fail-open / non-err, `1` only when `--strict` and aggregate `result=err`) in text and JSON so CI scrapers record the intended exit without shell `$?`

## [0.49.0] — 2026-07-20

Minor release: mesh status strict evidence.

### Added

- **`mesh status` strict evidence** — always emit `strict` (configured `--strict` exit-gate bool; `false` when unset) in text and JSON so CI scrapers record whether `result=err` would exit `1` without re-parsing flags

## [0.48.0] — 2026-07-20

Minor release: dogfood wait_ready_attempts evidence.

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
