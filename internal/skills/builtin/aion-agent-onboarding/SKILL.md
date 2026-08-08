---
name: aion-agent-onboarding
description: Residual-honest TUI agent ↔ aion CP/MCP onboarding (portal Agent/MCP mint/copy/probe · list/plan connectors → portal HITL · memory dual_write OFF · not Memory GA · never invent Connected)
---

# Aion agent onboarding (residual-honest)

Builtin playbook for **onboarding a TUI agent session against aion CP/MCP** — residual-honest path only. Molds `connector-integrations-setup` + operator onboarding checklist. **Not** install APPLY, **Not** Memory GA, **Not** Agent Plugins GA, **Not** dual_write ON.

**System note (s1363+s1368+s1372+s1377+s1382+s1387+s1397+s1402+s1407):** when MCP is attached (`AttachMCP`), runtime injects residual-honest `<aion-onboarding>` (`AionAgentOnboardingGuidanceNote`) with the same locks. Skill + note stay consistent; skill is the full playbook. Operator slash: `/onboard` (aliases `/aion-onboard` `/agent-onboard`) · `/onboard portal` · `/onboard status` · `/onboard checklist` · `/onboard next` (aliases `after` / `continue` / `lanes`) · `/onboard next [plugins|gtm|memory|mesh|memory-pull]` lane drills (s1377+s1402+s1407) · `/onboard next status` lane status board (aliases `pulse` / `board` · s1382+s1397 session soft dogfood · s1402 mesh row · s1407 memory-pull/ops_pack row) · `/onboard next export` status export receipt (aliases `receipt` / `stamp` / `evidence` · optional `json` · s1387+s1397+s1402+s1407).

## Workflow — two complementary halves

### A. Portal Agent/MCP half (credential → copy connection → test invoke · s1368)

1. **Mint credential in portal HITL** — open https://console.iome.sh/settings/agent (session cookie).
   - Mint API key / agent principal as needed (settings only).
   - **Not** install APPLY · **not** invent Connected from mint success.

2. **Settings → Agent/MCP → copy MCP connection**
   - Copy streamable HTTP MCP URL + auth env hint from the portal Agent/MCP panel.
   - Connection copy is **handoff only** — not Memory GA, not install green.

3. **Test invoke = probe only ≠ Memory GA**
   - Portal test invoke is a residual-honest **probe** (latency / tool snippet).
   - **Never invent** tool green, Memory Palace GA, or Connected from a probe.

### B. TUI half (`[[mcp.servers]]` streamable HTTP · operator pulse)

4. **Configure TUI MCP attach** — add `[[mcp.servers]]` with streamable HTTP `url` (+ `oauth_token_env` if needed).
   - Example shape (placeholders only — never invent live install green):
     ```toml
     [[mcp.servers]]
     name = "aion"
     url = "https://…/mcp"          # streamable HTTP from portal copy
     oauth_token_env = "AION_TOKEN" # env only — never commit secrets
     ```
   - Restart / reattach MCP. Offline / missing tools → residual-honest **fail-open**; **never invent** tool green or Connected.

5. **Operator pulse** — `/onboard` · `/onboard portal` · `/onboard status` · `/integrations status`.

### C. Connector integrations path (portal HITL · peer to Agent/MCP lane)

6. **Discover** — call MCP `list_connector_catalog` (aion v178).
   - Returns `{count, entries[]}` with `id`, `status`, `mesh_layer`, `oauth_install_supported`, `portal_path`.
   - **Catalog status ≠ install Connected.** Status chips (`available` / `beta` / `planned`) are display honesty only.
   - Never invent install green or org Connected counts from the catalog.
   - **No invent GA for knowledge/analytical** layers.

7. **Plan** — call MCP `plan_connector_setup` with `connector_id`.
   - Surfaces `portal_url`, `portal_add_url`, `deep_links`, `oauth_mode_hint`, `signing_headers_tool`, `next_steps`, `honesty.notes`.
   - Deep links are **browser HITL only** — not install APPLY success.

8. **Org installs residual snapshot** — call MCP `list_org_connector_installs` with `org_id` when present (aion v179 residual).
   - Residual-honest **fail-open** by default: `available=false`, `status=unavailable`, `installs=null`.
   - **Never invent empty-as-none** — `available=false` / `installs=null` is residual honesty, **not** "none connected".
   - Read-only residual tool only; portal session owns install index.

9. **Complete connector OAuth/install in browser portal HITL** — open https://console.iome.sh/integrations (session cookie).
   - OAuth authorize/callback and install CRUD live in the **console portal**, not agent MCP.
   - Agent MCP **cannot write installs** · never invent INSTALL_STORE APPLY.

10. **Memory residual** — dual_write **OFF** · **local-primary** · **not Memory GA**.
    - Optional advanced memory via `memory-advanced-agent` skill when needed (opt-in only).
    - Optional **plugins dogfood** (in-repo samples / offline validate) ≠ invent **Agent Plugins GA**.
    - Rates **~$88/$119 optional** — commercial framing only; not product GA claim.

11. **Operator pulse** — slash `/integrations status` · `/onboard checklist` · `/onboard portal` · portal HITL.

### D. Post-onboard next lanes (operator continuum · s1372+s1377+s1382+s1387+s1397+s1402+s1407 · no MCP dial)

After core onboarding, residual-honest next operator lanes — static offline only; **never invent** product GA from these steps.

- **Overview:** `/onboard next` (aliases `after` / `continue` / `lanes`) → `AionAgentOnboardingNextLanes`.
- **Lane drills (s1377+s1402+s1407):** `/onboard next <lane>` (also works with parent aliases, e.g. `/onboard after plugins` · `/onboard next mesh` · `/onboard next memory-pull`).
- **Lane status board (s1382+s1397+s1402+s1407):** `/onboard next status` (aliases `pulse` / `board`) → `AionAgentOnboardingNextLaneStatus` — plugins dogfood lane reflects **session soft** marker after `/plugins dogfood` · mesh row is `streams_not_probed` · memory-pull/ops_pack row is `pull_not_probed`.
- **Status export receipt (s1387+s1397+s1402+s1407):** `/onboard next export` (aliases `receipt` / `stamp` / `evidence`) → `AionAgentOnboardingNextLaneStatusExport` · optional `/onboard next export json` → `AionAgentOnboardingNextLaneStatusExportJSON` — same session soft + mesh + memory-pull rows.

| Lane | Slash | Aliases | API helper |
|------|-------|---------|------------|
| plugins | `/onboard next plugins` | `plugin` · `dogfood` | `AionAgentOnboardingNextPluginsLane` |
| gtm | `/onboard next gtm` | `drafts` | `AionAgentOnboardingNextGtmLane` |
| memory | `/onboard next memory` | `mcp` · `palace` | `AionAgentOnboardingNextMemoryLane` |
| mesh | `/onboard next mesh` | `stream` · `streams` · `heartbeat` · `heartbeats` · `pull` | `AionAgentOnboardingNextMeshLane` |
| memory-pull | `/onboard next memory-pull` | `ops-pack` · `pull-path` · `memorypull` · `ops_pack` | `AionAgentOnboardingNextMemoryPullLane` |
| status | `/onboard next status` | `pulse` · `board` | `AionAgentOnboardingNextLaneStatus` |
| export | `/onboard next export` | `receipt` · `stamp` · `evidence` | `AionAgentOnboardingNextLaneStatusExport` (+ optional `json`) |

Unknown lane token → overview + usage hint listing `plugins|gtm|memory|mesh|memory-pull|status|export`. **Note:** `pulse` is reserved for the status board — it is **not** a mesh alias. **Bare `pull` stays mesh** (s1402) — it is **not** a memory-pull alias (s1407 uses `memory-pull` / `ops-pack` / `pull-path` / `memorypull` / `ops_pack`).

#### D1. Plugins dogfood lane (`/onboard next plugins`)

1. **`iomesh plugins dogfood`** / **`/plugins dogfood`** — offline sample validate (`examples/agent-plugins/{hello-iome,aion-memory-mcp}`).
   - Steps: `iomesh plugins list` → `validate <path>` → `dogfood` (both in-repo samples offline).
   - Offline validate only · **≠ invent Agent Plugins GA** · soft offline dogfood ≠ invent Agent Plugins GA.
   - residual PASS ≠ live dogfood · rates ~$88/$119 optional · package load ≠ Memory GA.
   - **s1397 session soft marker:** after `/plugins dogfood`, session stores soft residual pass/fail.
     - Default: `dogfood_not_run`
     - Soft pass: `soft_offline_dogfood_session_pass`
     - Soft fail: `soft_offline_dogfood_session_fail`
     - **session soft ≠ live dogfood** · board/export evidence ≠ invent Connected.
     - Tip: re-run `/onboard next status` then `/onboard next export` to refresh residual evidence.

#### D2. GTM draft-only lane (`/onboard next gtm`)

2. **`/gtm checklist` + skill `gtm-draft-only-agent`** — drafts only · no auto-send · human publish.
   - GTM checklist ≠ invent GTM agent GA · no auto-send · human CRM commercial.
   - Companion: `read_skill gtm-draft-only-agent` · slash `/gtm [help|checklist]`.

#### D3. Memory local lane (`/onboard next memory`)

3. **local `aion-memory-mcp` / Memory Ops Pack local-primary** — dual_write **OFF**.
   - Package load ≠ Memory GA · ≠ freemium palace · local-primary only.
   - Optional advanced memory via `memory-advanced-agent` skill (opt-in).
   - Operator pulse: `/memory status` · `/onboard status`.
   - **mesh ≠ memory** — memory is local-edge palace; streaming org heartbeats live on the mesh lane (D3b).

#### D3b. Mesh streaming lane (`/onboard next mesh` · s1402 · product plane 1)

3b. **I/O Mesh = streaming org heartbeats on governed `dept.*`** — residual-honest path only.
   - **Not** hosted Memory Palace · **not** OTel/APM · **not** medical · **Palace sunset**.
   - **mesh ≠ memory** · mesh lane ≠ plugins/gtm lanes · pull ≠ freemium hosted palace.
   - Operator residual soft: `/mesh` · `iomesh mesh status|streams|consumer` — empty streams honest · **never invent stream green / Connected**.
   - Pull honesty (cross-link): `iomesh memory pull` = mesh → local palace **egress** · dual_write **OFF** · not freemium hosted palace · not Memory GA — full Ops Pack drill is **D3c memory-pull**.
   - Rates: mesh base **~$88** · Memory Ops Pack **~$119** pull/retain/support (optional commercial framing only).
   - Honest board vocab: `path_ready` · `residual_only` · `streams_not_probed`.
   - Slash: `/onboard next mesh` (aliases `stream` / `streams` / `heartbeat` / `heartbeats` / `pull`) — **NOT** `pulse` (pulse stays status board). Bare `pull` stays **mesh**, not memory-pull.
   - API: `AionAgentOnboardingNextMeshLane`.

#### D3c. Memory Ops Pack / pull path lane (`/onboard next memory-pull` · s1407)

3c. **Memory Ops Pack pull path** — residual-honest mesh → local palace **egress** only.
   - Path: `iomesh memory pull` · CreateConsumer → fetch → map envelope → local MCP `memory_ingest_turn` → ack.
   - **dual_write OFF** · **not Memory GA** · **Palace sunset** · **pull ≠ freemium hosted palace**.
   - **Ops Pack ≠ GPU fleet** — Memory Ops Pack **~$119** = pull / audit / Extended retain / support (not hosted GPU palace). Mesh base **~$88** is separate.
   - **package load ≠ Ops Pack entitlement** · package load ≠ Memory GA.
   - Honest board vocab: `path_ready` · `residual_only` · `pull_not_probed` — **never invent pull green**.
   - residual PASS ≠ live dogfood · PASS ≠ live APPLY.
   - Slash: `/onboard next memory-pull` (aliases `ops-pack` / `pull-path` / `memorypull` / `ops_pack`) — **bare `pull` stays mesh** (s1402).
   - API: `AionAgentOnboardingNextMemoryPullLane`.
   - Companion: `/onboard next mesh` (streaming heartbeats) · `/onboard next memory` (local-edge attach).

#### D4. Portal HITL still (all lanes)

4. **portal HITL still required for OAuth/install** — agent MCP **cannot write installs**.
   - catalog ≠ Connected · list_org fail-open ≠ empty-as-none · never invent Connected / INSTALL_STORE APPLY.
   - Portal: https://console.iome.sh/integrations · Agent/MCP: https://console.iome.sh/settings/agent.

#### D5. Lane status board (`/onboard next status` · s1382+s1397+s1402+s1407)

5. **Residual-honest lane status board** — offline pulse of plugins · gtm · memory · mesh · memory-pull · portal.
   - **No MCP dial** · never invent install green / Connected / GA / APPLY / stream green / pull green as success.
   - Honest state vocabulary only: `path_ready` · `samples_ok` · `samples_missing` · `dogfood_not_run` · `soft_offline_dogfood_session_pass` · `soft_offline_dogfood_session_fail` · `skill_ready` · `residual_only` · `streams_not_probed` · `pull_not_probed` · `portal_hitl_still`.
   - **plugins:** soft-check of sample dirs (`examples/agent-plugins`) · default `dogfood_not_run` · after `/plugins dogfood` session soft pass/fail (s1397) · **session soft ≠ live dogfood** · ≠ invent Agent Plugins GA · board ≠ invent Connected.
   - **gtm:** skill/checklist path ready · drafts only · no auto-send · ≠ invent GTM agent GA.
   - **memory:** dual_write OFF · package load ≠ Memory GA · local-primary ≠ freemium palace · mesh ≠ memory.
   - **mesh (s1402):** `path_ready` · `residual_only` · `streams_not_probed` · streaming org heartbeats · not OTel/APM · never invent stream green / Connected · empty streams honest.
   - **memory-pull (s1407):** `path_ready` · `residual_only` · `pull_not_probed` · Ops Pack pull path · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · never invent pull green.
   - **portal:** `portal_hitl_still` · agent MCP cannot write installs.
   - Slash: `/onboard next status` (aliases `pulse` / `board`) · also linked from `/onboard next` overview and `/onboard status`.
   - Export receipt: `/onboard next export` (s1387+s1397+s1402+s1407) — offline markdown evidence of this board (includes session soft + mesh + memory-pull rows).

#### D6. Status export receipt (`/onboard next export` · s1387+s1397+s1402+s1407)

6. **Residual-honest status export receipt** — offline markdown (or optional JSON) evidence of the s1382 lane status board (plugins lane includes s1397 session soft marker when set · mesh row s1402 · memory-pull/ops_pack row s1407).
   - Header: `evidence_kind=onboard_next_lane_status_export` · `offline_static` · `not_live_dogfood` · serial `s1387`.
   - Reuses honest vocabulary: `path_ready` · `samples_ok`/`samples_missing` · `dogfood_not_run` · `soft_offline_dogfood_session_pass|fail` · `skill_ready` · `residual_only` · `streams_not_probed` · `pull_not_probed` · `portal_hitl_still`.
   - JSON field `plugins_dogfood_state` mirrors session soft label; `dogfood_not_run` is true only when session soft has not run; mesh lane value is `path_ready · residual_only · streams_not_probed`; memory-pull and ops_pack lanes are `path_ready · residual_only · pull_not_probed`.
   - **Does NOT** run plugins dogfood · **does NOT** dial MCP · **does NOT** invent install green / Connected / GA / APPLY / stream green / pull green.
   - **session soft ≠ live dogfood** · **board/export evidence ≠ invent Connected** — a stamped receipt is offline residual evidence only.
   - Markdown: `/onboard next export` (aliases `receipt` / `stamp` / `evidence`) → `AionAgentOnboardingNextLaneStatusExport`.
   - JSON: `/onboard next export json` → `AionAgentOnboardingNextLaneStatusExportJSON`.
   - Cross-linked from lane status board footer, `/onboard next` overview, and `/plugins dogfood` tip.

## Honesty locks

| Lock | Meaning |
|------|---------|
| dual_write OFF | Local-primary memory honesty; do not claim dual_write ON |
| book-demo OFF | No invent book-a-demo install path |
| not Memory GA | Onboarding ≠ invent Memory Palace / graph RAG product green; test invoke = probe only |
| residual PASS ≠ live dogfood | Offline skill / gate PASS is not live install, live AAA, or live APPLY |
| never invent install green | Plan/list/status never claim Connected / INSTALL_STORE APPLY success |
| available=false ≠ empty-as-none | `list_org_connector_installs` fail-open residual ≠ "no installs" |
| catalog ≠ Connected | Catalog status is display honesty only, not install Connected |
| portal HITL | Human finishes OAuth / install / key mint in console session |
| agent MCP cannot write installs | Portal session owns install plane; MCP is residual list/plan |
| plugins dogfood ≠ Agent Plugins GA | Sample/offline dogfood is not product GA; rates ~$88/$119 optional |
| drafts only · no auto-send | GTM / post-onboard lanes never auto-send or auto-publish; human publish |
| package load ≠ Memory GA | Loading aion-memory-mcp / Ops Pack local ≠ invent Memory Palace GA / freemium palace |
| GTM checklist ≠ GTM agent GA | `/gtm checklist` residual-honest draft path only — not invent GTM agent GA |
| board/export evidence ≠ invent Connected | Lane status board + export receipt are offline residual evidence only — never invent Connected / GA / APPLY |
| session soft ≠ live dogfood | `/plugins dogfood` session marker on status/export is soft offline residual only — not live dogfood · not Agent Plugins GA |
| mesh = streaming org heartbeats | I/O Mesh product plane 1 = governed `dept.*` org heartbeats — not OTel/APM · not medical · not hosted Memory Palace |
| mesh ≠ memory | Mesh streaming lane is separate from local-edge memory lane; pull = egress into local palace only |
| never invent stream green | Empty streams honest · `streams_not_probed` residual · never invent Connected / live stream green from board/export |
| pull ≠ freemium hosted palace | `iomesh memory pull` is mesh → local palace egress · dual_write OFF · Palace sunset · not freemium hosted palace |
| Ops Pack ≠ GPU fleet | Memory Ops Pack ~$119 = pull / audit / Extended retain / support — not hosted GPU palace; mesh base ~$88 separate |
| pull_not_probed | Ops Pack pull path residual honest until operator probes · never invent pull green from board/export |
| package load ≠ Ops Pack entitlement | Loading packages / residual path ≠ invent Ops Pack commercial entitlement |
| no invent GA knowledge/analytical | Do not invent GA for knowledge or analytical mesh layers |

## Non-goals (never do)

- Do **not** invent install green / Connected / INSTALL_STORE APPLY / GA.
- Do **not** invent empty-as-none installs from unavailable / `installs=null` / `available=false`.
- Do **not** complete OAuth without browser HITL (stub OAuth ≠ live).
- Do **not** claim dual_write ON, book-demo ON, Memory GA, or Agent Plugins GA.
- Do **not** treat catalog Beta/available/planned as Connected/installed.
- Do **not** treat residual PASS as live dogfood publish or live APPLY green.
- Do **not** invent GA for knowledge/analytical connectors or digests.
- Do **not** invent freemium palace / dual-write audit as product green.
- Do **not** treat portal test invoke / mint key as Memory GA or install Connected.
- Do **not** invent Agent Plugins GA from `iomesh plugins dogfood` offline validate.
- Do **not** invent GTM agent GA from `/gtm checklist` / draft-only skill.
- Do **not** invent Memory GA / freemium palace from local `aion-memory-mcp` package load.
- Do **not** invent Connected / GA / APPLY from board or export receipt evidence stamps.
- Do **not** treat session soft dogfood pass/fail as live dogfood or Agent Plugins GA.
- Do **not** invent stream green / Connected / live mesh from residual soft status/streams (empty streams honest · `streams_not_probed`).
- Do **not** conflate mesh streaming heartbeats with local-edge memory lane (mesh ≠ memory · not OTel/APM · not freemium hosted palace).
- Do **not** treat `iomesh memory pull` as freemium hosted palace or dual_write ON.
- Do **not** invent pull green / Ops Pack entitlement / GPU fleet from residual soft memory-pull path (`pull_not_probed` · Ops Pack ≠ GPU fleet).
- Do **not** steal bare `pull` under `/onboard next` for memory-pull — bare `pull` stays mesh (s1402); memory-pull uses `memory-pull` / `ops-pack` / `pull-path` / `memorypull` / `ops_pack`.

## Related

- Builtin skill always available when skills enabled (**s1363+s1368+s1372+s1377+s1382+s1387+s1397+s1402+s1407** · molds s1251 connector + s1288 memory-advanced + s1341 gtm-draft-only).
- System note inject on `AttachMCP`: `<aion-onboarding>` via `AionAgentOnboardingGuidanceNote` (s1363+s1368+s1372+s1377+s1382+s1387+s1402+s1407).
- Portal handoff block: `AionAgentOnboardingPortalHandoff` · slash `/onboard portal` (aliases `agent-mcp` / `mcp`).
- Offline status: `AionAgentOnboardingStatus` · slash `/onboard status` (no MCP dial) · cross-links `/onboard next status` (s1382+s1397+s1402+s1407) · `/onboard next export` (s1387+s1397+s1402+s1407) · `/onboard next mesh` (s1402) · `/onboard next memory-pull` (s1407).
- Post-onboard next lanes overview: `AionAgentOnboardingNextLanes` · slash `/onboard next` (aliases `after` / `continue` / `lanes`) — plugins · gtm · memory local · mesh streaming · Ops Pack pull path · portal HITL still (s1372+s1402+s1407) · status board (s1382+s1397+s1402+s1407) · export receipt (s1387+s1397+s1402+s1407).
- Lane drills (s1377+s1402+s1407): `AionAgentOnboardingNextPluginsLane` · `AionAgentOnboardingNextGtmLane` · `AionAgentOnboardingNextMemoryLane` · `AionAgentOnboardingNextMeshLane` · `AionAgentOnboardingNextMemoryPullLane` · slash `/onboard next [plugins|gtm|memory|mesh|memory-pull]` (aliases plugin|dogfood · drafts · mcp|palace · stream|streams|heartbeat|heartbeats|pull · ops-pack|pull-path|memorypull|ops_pack).
- Lane status board (s1382+s1397+s1402+s1407): `AionAgentOnboardingNextLaneStatus` · slash `/onboard next status` (aliases `pulse` / `board`) — honest vocabulary only · no invent Connected/GA/APPLY/stream green/pull green · dogfood_not_run default · session soft pass/fail after `/plugins dogfood` · mesh `streams_not_probed` · memory-pull `pull_not_probed` · session soft ≠ live dogfood.
- Status export receipt (s1387+s1397+s1402+s1407): `AionAgentOnboardingNextLaneStatusExport` · slash `/onboard next export` (aliases `receipt` / `stamp` / `evidence`) · optional `AionAgentOnboardingNextLaneStatusExportJSON` via `/onboard next export json` — evidence_kind=onboard_next_lane_status_export · offline_static · not_live_dogfood · plugins_dogfood_state session soft · mesh streams_not_probed · memory-pull/ops_pack pull_not_probed · board/export evidence ≠ invent Connected.
- Companion builtin: `connector-integrations-setup` (list/plan → portal HITL).
- Companion builtin: `memory-advanced-agent` (opt-in advanced memory · dual_write OFF · not Memory GA).
- Companion builtin: `gtm-draft-only-agent` (drafts only · human publish · no auto-send).
- Slash residual honesty: `/onboard [help|checklist|portal|status|next]` · `/onboard next [plugins|gtm|memory|mesh|memory-pull|status|export]` · `/plugins dogfood|status` · `/integrations list|plan|status|signing` · `/memory status` · `/mesh` · `/gtm [help|checklist]`.
- Skills are **not** Agent Plugins — plugins dogfood ≠ invent Agent Plugins GA · session soft ≠ live dogfood · mesh ≠ memory · Ops Pack ≠ GPU fleet.
