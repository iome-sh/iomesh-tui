---
name: aion-agent-onboarding
description: Residual-honest TUI agent ↔ aion CP/MCP onboarding (portal Agent/MCP mint/copy/probe · list/plan connectors → portal HITL · memory dual_write OFF · not Memory GA · never invent Connected)
---

# Aion agent onboarding (residual-honest)

Builtin playbook for **onboarding a TUI agent session against aion CP/MCP** — residual-honest path only. Molds `connector-integrations-setup` + operator onboarding checklist. **Not** install APPLY, **Not** Memory GA, **Not** Agent Plugins GA, **Not** dual_write ON.

**System note (s1363+s1368+s1372+s1377+s1382+s1387+s1397+s1402+s1407+s1413+s1417+s1422+s1427+s1432+s1437+s1442+s1447+s1542+s1558+s1562+s1566+s1570+s1574):** when MCP is attached (`AttachMCP`), runtime injects residual-honest `<aion-onboarding>` (`AionAgentOnboardingGuidanceNote`) with the same locks. Skill + note stay consistent; skill is the full playbook. Operator slash: `/onboard` (aliases `/aion-onboard` `/agent-onboard`) · `/onboard portal` · `/onboard status` · `/onboard checklist` · `/onboard next` (aliases `after` / `continue` / `lanes`) · `/onboard next [plugins|gtm|memory|mesh|memory-pull|agentic|portal-hitl|e4|planes|sales|demo|operator|setup|journey|wizard|status|export|human-gates]` lane drills (s1377+s1402+s1407+s1417+s1432+s1437+s1442+s1447+s1542+s1558+s1562+s1566+s1570+s1574) · `/onboard next portal-hitl` journey stage-5 portal HITL (s1562 · aliases `hitl`/`portal_hitl`/`portal-dogfood`/`stage5`/`connectors-hitl`) · `/onboard next portal-hitl dogfood` soft offline portal HITL (s1562 · aliases `soft`/`samples`/`offline`/`portal-hitl-soft`) · `/onboard next e4` journey stage-6 E4 client-attach (s1566 · aliases `e4-dogfood`/`client-attach`/`edge-memory-e4`/`e4_attach`) · `/onboard next e4 dogfood` soft offline E4 (s1566 · aliases `soft`/`samples`/`offline`/`e4-soft`) · `/onboard next agentic dogfood` soft offline list/plan (s1422 · aliases `soft`/`samples`/`offline`/`list-plan-soft`) · `/onboard next agentic dual-auth` dual-auth candidacy depth (s1427 · aliases `candidacy`/`list-org`/`org-installs`/`dual_auth`/`dual-auth-candidacy`) · `/onboard next planes` three product planes board (s1432 · aliases `three-planes`/`product-planes`/`product`/`pillars`/`three_planes`) · `/onboard next sales` sales/buyer claims board (s1437 · aliases `claims`/`buyer`/`claim-matrix`/`sales-claims`/`buyer-claims`) · `/onboard next demo` demo readiness board (s1442 · aliases `demo-ready`/`readiness`/`demo-readiness`/`lighthouse`/`landgrab`) · `/onboard next operator` operator readiness matrix (s1447 · aliases `operator-matrix`/`ops-matrix`/`operator-readiness`/`ops-readiness`/`matrix`) · `/onboard next setup` setup lifecycle map (s1542 · stage 4) · `/onboard next journey` edge-user-journey first-run map (s1558 Wave B · aliases `edge-journey`/`user-journey`/`first-run`/`edge_user_journey`) · `/onboard next status` lane status board (aliases `pulse` / `board` · s1382+s1397 session soft dogfood · s1402 mesh row · s1407 memory-pull/ops_pack row · s1417+s1422+s1427 agentic row + list/plan soft + dual_auth_candidacy_open) · `/onboard next export` status export receipt (aliases `receipt` / `stamp` / `evidence` · optional `json` · s1387+s1397+s1402+s1407+s1417+s1422+s1427) · `/onboard next human-gates` still-required vs offline residual (aliases `human` / `gates` / `apply-gates` · s1413).

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

### D. Post-onboard next lanes (operator continuum · s1372+s1377+s1382+s1387+s1397+s1402+s1407+s1413+s1417+s1422+s1427+s1432+s1437+s1442+s1447+s1542+s1558+s1562+s1566 · no MCP dial)

After core onboarding, residual-honest next operator lanes — static offline only; **never invent** product GA from these steps.

- **Overview:** `/onboard next` (aliases `after` / `continue` / `lanes`) → `AionAgentOnboardingNextLanes`.
- **Lane drills (s1377+s1402+s1407+s1417):** `/onboard next <lane>` (also works with parent aliases, e.g. `/onboard after plugins` · `/onboard next mesh` · `/onboard next memory-pull` · `/onboard next agentic`).
- **Three product planes board (s1432):** `/onboard next planes` (aliases `three-planes` / `product-planes` / `product` / `pillars` / `three_planes`) → `AionAgentOnboardingNextThreePlanes` — consolidate mesh · memory-pull · agentic residual-honest · **no invent stream green / pull green / Connected**.
- **Sales/buyer claims board (s1437):** `/onboard next sales` (aliases `claims` / `buyer` / `claim-matrix` / `sales-claims` / `buyer-claims`) → `AionAgentOnboardingNextSalesClaims` — may claim / must not claim residual-honest · three-planes grounded · **never invent Connected / Memory GA / dual-auth live**.
- **Demo readiness board (s1442):** `/onboard next demo` (aliases `demo-ready` / `readiness` / `demo-readiness` / `lighthouse` / `landgrab`) → `AionAgentOnboardingNextDemoReadiness` — Lighthouse beachhead packaging · book-demo **OFF** · Landgrab **NOT READY** · three planes · sales claims · human gates still open · **never invent Connected / book-demo ON / Landgrab READY**.
- **Operator readiness matrix (s1447):** `/onboard next operator` (aliases `operator-matrix` / `ops-matrix` / `operator-readiness` / `ops-readiness` / `matrix`) → `AionAgentOnboardingNextOperatorMatrix` — consolidate demo · sales · planes · human-gates · dual-auth candidacy · policy locks residual-honest · **never invent Connected / GA / dual-auth live**.
- **setup lifecycle map (s1542+s1558 · stage 4 of edge-user-journey · P1–P7 closeout residual):** `/onboard next setup` (aliases `setup-lifecycle` / `lifecycle` / `setup_lifecycle`) → `AionAgentOnboardingNextSetupLane` — init → preflight → reload → portal HITL → pull → analyze → drift → repair plan/apply --yes · **setup_not_probed** · dual_write OFF · package wire ≠ Connected · **repair apply ≠ invent Connected** · dual_write never auto ON · **E10 Open** · offline static ≠ live dogfood · setup closeout residual ≠ invent Edge Memory GA · full first-run map: `/onboard next journey`.
- **edge-user-journey first-run map (s1558 Wave B · 7 stages):** `/onboard next journey` (aliases `edge-journey` / `user-journey` / `first-run` / `edge_user_journey`) → `AionAgentOnboardingNextJourneyLane` — Signup → Download TUI → TUI auth/keys → Setup wizard → Connectors → Local store → Analyze · dual_write OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA · portal HITL · no invent TUI portal SSO · host not auto · free eng **s1558** · free-floor peer **s1560+** mention only · docs `edge-user-journey.md` · `setup-lifecycle.md` · `memory-edge-usage-demo.md` · stage 6 companion `/onboard next e4` (s1566).
- **E4 client-attach stage-6 board + soft dogfood (s1566):** `/onboard next e4` (aliases `e4-dogfood` / `client-attach` / `edge-memory-e4` / `e4_attach`) → `AionAgentOnboardingNextE4Lane` · soft `/onboard next e4 dogfood` (aliases `soft` / `samples` / `offline` / `e4-soft`) → `RunE4SoftDogfood` — journey stage 6 local store / MCP attach · tools=6 · `iomesh mcp --connect` residual · dual_write OFF · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · tip ≠ invent forever-green product dogfood · free eng **s1566** · free-floor peer **s1568+** mention only.
- **first-run wizard residual + soft dogfood (s1570 Wave C):** `/onboard next wizard` (aliases `first-run-wizard` / `guided` / `wave-c` / `wave_c` / `wizard-residual`) → `AionAgentOnboardingNextWizardLane` · soft `/onboard next wizard dogfood` (aliases `soft` / `samples` / `offline` / `wizard-soft`) → `RunFirstRunWizardSoftDogfood` — deeper guided residual map after Wave B journey · session labels `wizard_soft_not_run` · `soft_offline_wizard_session_pass` · `soft_offline_wizard_session_fail` · NOT invent full interactive auto wizard · dual_write OFF · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · portal HITL when connect · agent MCP cannot write installs · catalog ≠ Connected · no invent TUI portal SSO · host not auto · free eng **s1570** · free-floor peer **s1572+** mention only · companion still-human soft `/onboard next human-gates dogfood` (s1574).
- **Agentic soft offline list/plan dogfood (s1422):** `/onboard next agentic dogfood` (aliases `soft` / `samples` / `offline` / `list-plan-soft` as 4th token) → `RunAgenticListPlanSoftDogfood` — independent of plugins soft · **bare `/onboard next agentic` stays board** (not auto dogfood).
- **Agentic dual-auth candidacy (s1427):** `/onboard next agentic dual-auth` (aliases `candidacy` / `list-org` / `org-installs` / `dual_auth` / `dual-auth-candidacy` as 4th token) → `AionAgentOnboardingNextAgenticDualAuthCandidacy` — `list_org` fail-open · **tool ship ≠ dual-auth live** · **do not steal** dogfood soft aliases.
- **Lane status board (s1382+s1397+s1402+s1407+s1417+s1422+s1427):** `/onboard next status` (aliases `pulse` / `board`) → `AionAgentOnboardingNextLaneStatus` — plugins dogfood lane reflects **session soft** marker after `/plugins dogfood` · mesh row is `streams_not_probed` · memory-pull/ops_pack row is `pull_not_probed` · agentic row is `list_plan_not_connected` · `portal_hitl_still` · **`<soft label>`** (s1422) · `dual_auth_candidacy_open` · `list_org_unavailable` (s1427).
- **Status export receipt (s1387+s1397+s1402+s1407+s1417+s1422+s1427):** `/onboard next export` (aliases `receipt` / `stamp` / `evidence`) → `AionAgentOnboardingNextLaneStatusExport` · optional `/onboard next export json` → `AionAgentOnboardingNextLaneStatusExportJSON` — same session soft + mesh + memory-pull + agentic soft + dual-auth candidacy rows · JSON field `agentic_list_plan_soft_state`.
- **Human-gates honesty board + still-human APPLY soft dogfood (s1413+s1546+s1550+s1574 Wave C continuum):** `/onboard next human-gates` (aliases `human` / `gates` / `apply-gates` / `still-human` / `apply-residual`) → `AionAgentHumanGatesHonestyBoard` · soft `/onboard next human-gates dogfood` (aliases `soft` / `samples` / `offline` / `still-human-soft` / `apply-soft`) → `RunStillHumanApplySoftDogfood` — still-human APPLY open inventory residual after Wave A–C continuum · session labels `still_human_soft_not_run` · `soft_offline_still_human_session_pass` · `soft_offline_still_human_session_fail` · PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open · edge-first · knowledge multi-tenant punted · Slack HMAC punted · portal HITL when connect · dual_write OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · free eng **s1574** · free-floor peer **s1576+** mention only.

| Lane | Slash | Aliases | API helper |
|------|-------|---------|------------|
| plugins | `/onboard next plugins` | `plugin` · `dogfood` | `AionAgentOnboardingNextPluginsLane` |
| gtm | `/onboard next gtm` | `drafts` | `AionAgentOnboardingNextGtmLane` |
| memory | `/onboard next memory` | `mcp` · `palace` | `AionAgentOnboardingNextMemoryLane` |
| mesh | `/onboard next mesh` | `stream` · `streams` · `heartbeat` · `heartbeats` · `pull` | `AionAgentOnboardingNextMeshLane` |
| memory-pull | `/onboard next memory-pull` | `ops-pack` · `pull-path` · `memorypull` · `ops_pack` | `AionAgentOnboardingNextMemoryPullLane` |
| agentic | `/onboard next agentic` | `agentic-integrations` · `integrations` · `list-plan` | `AionAgentOnboardingNextAgenticLane` |
| agentic dual-auth | `/onboard next agentic dual-auth` | `candidacy` · `list-org` · `org-installs` · `dual_auth` · `dual-auth-candidacy` | `AionAgentOnboardingNextAgenticDualAuthCandidacy` |
| portal-hitl | `/onboard next portal-hitl` | `hitl` · `portal_hitl` · `portal-dogfood` · `stage5` · `connectors-hitl` | `AionAgentOnboardingNextPortalHITLLane` |
| portal-hitl dogfood | `/onboard next portal-hitl dogfood` | `soft` · `samples` · `offline` · `portal-hitl-soft` | `RunPortalHITLSoftDogfood` |
| e4 | `/onboard next e4` | `e4-dogfood` · `client-attach` · `edge-memory-e4` · `e4_attach` | `AionAgentOnboardingNextE4Lane` |
| e4 dogfood | `/onboard next e4 dogfood` | `soft` · `samples` · `offline` · `e4-soft` | `RunE4SoftDogfood` |
| planes | `/onboard next planes` | `three-planes` · `product-planes` · `product` · `pillars` · `three_planes` | `AionAgentOnboardingNextThreePlanes` |
| sales | `/onboard next sales` | `claims` · `buyer` · `claim-matrix` · `sales-claims` · `buyer-claims` | `AionAgentOnboardingNextSalesClaims` |
| demo | `/onboard next demo` | `demo-ready` · `readiness` · `demo-readiness` · `lighthouse` · `landgrab` | `AionAgentOnboardingNextDemoReadiness` |
| operator | `/onboard next operator` | `operator-matrix` · `ops-matrix` · `operator-readiness` · `ops-readiness` · `matrix` | `AionAgentOnboardingNextOperatorMatrix` |
| setup | `/onboard next setup` | `setup-lifecycle` · `lifecycle` · `setup_lifecycle` | `AionAgentOnboardingNextSetupLane` |
| journey | `/onboard next journey` | `edge-journey` · `user-journey` · `first-run` · `edge_user_journey` | `AionAgentOnboardingNextJourneyLane` |
| wizard | `/onboard next wizard` | `first-run-wizard` · `guided` · `wave-c` · `wave_c` · `wizard-residual` | `AionAgentOnboardingNextWizardLane` |
| wizard dogfood | `/onboard next wizard dogfood` | `soft` · `samples` · `offline` · `wizard-soft` | `RunFirstRunWizardSoftDogfood` |
| human-gates | `/onboard next human-gates` | `human` · `gates` · `apply-gates` · `still-human` · `apply-residual` | `AionAgentHumanGatesHonestyBoard` |
| human-gates dogfood | `/onboard next human-gates dogfood` | `soft` · `samples` · `offline` · `still-human-soft` · `apply-soft` | `RunStillHumanApplySoftDogfood` |
| status | `/onboard next status` | `pulse` · `board` | `AionAgentOnboardingNextLaneStatus` |
| export | `/onboard next export` | `receipt` · `stamp` · `evidence` | `AionAgentOnboardingNextLaneStatusExport` (+ optional `json`) |

Unknown lane token → overview + usage hint listing `plugins|gtm|memory|mesh|memory-pull|agentic|portal-hitl|e4|planes|sales|demo|operator|setup|journey|wizard|status|export|human-gates`. **Note:** `pulse` is reserved for the status board — it is **not** a mesh alias. **Bare `pull` stays mesh** (s1402) — it is **not** a memory-pull alias (s1407 uses `memory-pull` / `ops-pack` / `pull-path` / `memorypull` / `ops_pack`). **Bare `mcp` stays memory** under `/onboard next` — it is **not** an agentic alias. **Bare `portal` / `agent-mcp` stay portal handoff** under `/onboard` — they are **not** agentic or portal-hitl next-lane aliases (agentic uses `agentic`/`integrations`/`list-plan`; portal HITL uses `portal-hitl`/`hitl` s1562). **`e4` / `client-attach` stay E4 client-attach** (s1566) — not memory lane (memory uses `memory`/`mcp`/`palace`). **`product` / `planes` stay three-planes** — they are **not** sales claims aliases (sales uses `sales` / `claims` / `buyer` / `claim-matrix` / `sales-claims` / `buyer-claims`). **`sales` / `claims` stay sales claims** — they are **not** demo readiness aliases (demo uses `demo` / `demo-ready` / `readiness` / `demo-readiness` / `lighthouse` / `landgrab`; landgrab stays honesty **NOT READY**). **`demo` / `readiness` / `lighthouse` / `landgrab` stay demo board** — they are **not** operator matrix aliases (operator uses `operator` / `operator-matrix` / `ops-matrix` / `operator-readiness` / `ops-readiness` / `matrix`). **`export` / `receipt` stay export** — they are **not** operator matrix aliases.

#### D1. Plugins dogfood lane (`/onboard next plugins`)

1. **`iomesh plugins dogfood`** / **`/plugins dogfood`** — offline sample validate (`examples/agent-plugins/{hello-iome,iomesh-memory-mcp}` product primary · s1517).
   - Steps: `iomesh plugins list` → `validate <path>` → `dogfood` (both in-repo product samples offline).
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

#### D3. Memory local lane (`/onboard next memory` · s1377+s1453+s1458+s1463+s1469+s1478)

3. **local-primary Memory edge** — TUI + Memory MCP + `github.com/iome-sh/memory` kernel + local palace · dual_write **OFF**.
   - Package load ≠ Memory GA · ≠ freemium palace · local-primary only · **Palace sunset**.
   - **Edge OSS Option A (s1453):** product MCP host = **`iomesh-memory-mcp` only** · aion = **private** cloud broker/CP · kernel module `github.com/iome-sh/memory` · s1517 no in-tree residual aion Memory sample.
   - **Public product attach (s1478):** both edge repos **public** · `go install github.com/iome-sh/iomesh-memory-mcp/cmd/iomesh-memory-mcp@main` · `go get github.com/iome-sh/memory@main` · **no GOPRIVATE** · attach streamable HTTP `http://127.0.0.1:8080/mcp` or stdio · docker compose still valid · dual_write **OFF** · not Memory GA · aion broker private · **flip complete residual ≠ invent Memory GA** · **public OSS ≠ invent platform GA** · **PASS ≠ invent full platform sidecar parity**.
   - History: M2 lean attach (s1458) · M3 edge dogfood (s1463) · M4 public flip readiness (s1469) — superseded for operator tip by s1478 public install.
   - **Product-only sample:** `examples/agent-plugins/iomesh-memory-mcp` (s1517 · aion broker private).
   - Mesh **optional for pull only** · companion `/onboard next memory-pull` · `/onboard next e4` (s1566 E4 client-attach soft residual) · `/onboard next operator`.
   - Optional advanced memory via `memory-advanced-agent` skill (opt-in).
   - Operator pulse: `/memory status` · `/onboard status`.
   - **mesh ≠ memory** — memory is local-edge palace; streaming org heartbeats live on the mesh lane (D3b).
   - E4 client attach residual (s1508+s1566): lean host HTTP → `iomesh mcp --connect` · tools=6 stamp residual · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · tip ≠ invent forever-green product dogfood · soft `/onboard next e4 dogfood`.

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

#### D3d. Agentic integrations lane (`/onboard next agentic` · s1417+s1422+s1427 · product plane 3)

3d. **Agentic integrations (product plane 3)** — residual-honest MCP list/plan + portal HITL continuum.
   - MCP **list**: `list_connector_catalog` / list connectors residual-honest · **catalog ≠ Connected** · catalog status ≠ install Connected.
   - MCP **plan**: `plan_connector_setup` → proven portal deep links only · **browser HITL** · `template=` ≠ install APPLY green · deep_links = browser HITL only.
   - Proven deep-link paths: `/integrations/{id}` · `/integrations/add?template={id}` · `/integrations`.
   - Org residual: `list_org_connector_installs` fail-open · `available=false` default residual · installs=null · **list_org fail-open ≠ empty-as-none** · **never invent Connected**.
   - **portal HITL** for OAuth/install @ https://console.iome.sh/integrations — **agent MCP cannot write installs**.
   - Companion (not this lane): Agent/MCP mint/copy/probe @ https://console.iome.sh/settings/agent · `/onboard portal` · human-gates still-required vs offline.
   - **Portal HITL polish (s1422):** proven paths only · `template=` ≠ install APPLY · deep_links = browser HITL only · complementary `/onboard portal` mint/copy/probe (probe only) · OAuth/install still portal HITL · dual_write OFF · catalog ≠ Connected.
   - Honest board vocab: `path_ready` · `residual_only` · `portal_hitl_still` · `list_plan_not_connected` · soft label (s1422) · `dual_auth_candidacy_open` (s1427).
   - dual_write **OFF** · book-demo **OFF** · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open · rates ~$88/$119 optional.
   - Does **not** claim dual-auth live for list_org · residual soft path only · **tool ship ≠ dual-auth live**.
   - Slash: `/onboard next agentic` (aliases `agentic-integrations` / `integrations` / `list-plan`) — **NOT** bare `mcp` (memory) · **NOT** bare `portal`/`agent-mcp` (portal handoff) · **NOT** bare `pull` (mesh).
   - Dual-auth depth tip: `/onboard next agentic dual-auth` (s1427) · soft dogfood tip: `/onboard next agentic dogfood` (s1422) · portal HITL stage-5 tip: `/onboard next portal-hitl` (s1562) · soft `/onboard next portal-hitl dogfood`.
   - API: `AionAgentOnboardingNextAgenticLane`.
   - Companion: `/onboard next portal-hitl` (s1562) · `/onboard portal` · `/integrations list|plan|status` · `/onboard next human-gates` · `/onboard next status` · `/onboard next export`.

#### D3d-soft. Agentic list/plan soft offline dogfood (`/onboard next agentic dogfood` · s1422)

3d-soft. **Soft offline list/plan residual dogfood** — offline honesty check of agentic board + proven portal path shapes.
   - **No MCP dial** · never invent Connected · never invent install APPLY · never claim dual-auth live.
   - Session soft marker (**independent** of plugins `SoftDogfoodSession*`):
     - Default: `list_plan_soft_not_run`
     - Soft pass: `soft_offline_list_plan_session_pass`
     - Soft fail: `soft_offline_list_plan_session_fail`
   - **soft offline list/plan ≠ live dogfood** · **session soft ≠ live dogfood** · portal HITL still · list_org fail-open ≠ empty-as-none · board/export ≠ invent Connected.
   - Slash: `/onboard next agentic dogfood` (aliases 4th token `soft` / `samples` / `offline` / `list-plan-soft`) — **bare `/onboard next agentic` stays board** (not auto dogfood).
   - **Bare `/onboard next dogfood` stays plugins lane** — does not steal for agentic.
   - Tip: re-run `/onboard next status` then `/onboard next export` so agentic lane reflects session soft.
   - API: `RunAgenticListPlanSoftDogfood` · session SSOT in `agent` package (`AgenticListPlanSoftSessionLabel`).

#### D3d-dual-auth. Agentic dual-auth candidacy (`/onboard next agentic dual-auth` · s1427)

3d-dual-auth. **Dual-auth candidacy depth** for org installs snapshot residual — static offline only.
   - MCP tool residual: `list_org_connector_installs` · `available=false` · `status=unavailable` · `installs=null`.
   - **never invent empty-as-none** — `installs=null` not `[]` · `available=false` ≠ "none connected".
   - **dual_auth_candidacy_open** · **list_org_unavailable** · **tool ship ≠ dual-auth live** · **PASS ≠ invent dual-auth shipped** · never invent dual-auth live.
   - **portal session owns install index** · session-cookie + org membership only · **agent MCP cannot write installs**.
   - catalog ≠ Connected · never invent Connected · portal HITL @ https://console.iome.sh/integrations.
   - dual_write **OFF** · book-demo **OFF** · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open · rates ~$88/$119 optional.
   - Honest vocab: `path_ready` · `residual_only` · `dual_auth_candidacy_open` · `list_org_unavailable`.
   - Slash: `/onboard next agentic dual-auth` (aliases 4th token `candidacy` / `list-org` / `org-installs` / `dual_auth` / `dual-auth-candidacy`).
   - **Do not steal:** `dogfood` / `soft` / `samples` / `offline` / `list-plan-soft` stay soft dogfood (s1422).
   - Bare `/onboard next agentic` stays main agentic board.
   - API: `AionAgentOnboardingNextAgenticDualAuthCandidacy`.
   - Companion: `/onboard next agentic` · `/onboard next agentic dogfood` · `/onboard portal` · `/onboard next status`.

#### D3e. Three product planes board (`/onboard next planes` · s1432)

3e. **Three product planes residual-honest consolidate** — offline static board for mesh · memory-pull · agentic narrative planes.
   - **Plane 1 Mesh:** streaming org heartbeats on `dept.*` · **mesh ≠ memory** · `streams_not_probed` · not OTel/APM · never invent stream green · `/onboard next mesh`.
   - **Plane 2 Memory-pull / Ops Pack:** mesh → local palace egress · **dual_write OFF** · **Ops Pack ≠ GPU** · `pull_not_probed` · never invent pull green · `/onboard next memory-pull`.
   - **Plane 3 Agentic integrations:** MCP list/plan · portal HITL · `list_plan_not_connected` · `dual_auth_candidacy_open` · never invent Connected · `/onboard next agentic` · dual-auth · dogfood.
   - Rates **~$88 mesh / ~$119 Memory Ops Pack** optional · book-demo **OFF** · human-gates still open (tip only) · residual PASS ≠ live dogfood · PASS ≠ live APPLY.
   - Honest vocab: `path_ready` · `residual_only` · `streams_not_probed` · `pull_not_probed` · `portal_hitl_still` · `list_plan_not_connected` · `dual_auth_candidacy_open`.
   - Slash: `/onboard next planes` (aliases `three-planes` / `product-planes` / `product` / `pillars` / `three_planes`).
   - **Do not steal:** `pulse` / `board` stay status · bare `pull` stays mesh · bare `mcp` stays memory.
   - API: `AionAgentOnboardingNextThreePlanes`.
   - Companion: `/onboard next mesh` · `/onboard next memory-pull` · `/onboard next agentic` · `/onboard next agentic dual-auth` · `/onboard next agentic dogfood` · `/onboard next sales` · `/onboard next status` · `/onboard next export` · `/onboard next human-gates`.

#### D3f. Sales / buyer claims board (`/onboard next sales` · s1437)

3f. **Sales / buyer claims residual-honest matrix** — offline static board for founders/sales operators (may claim vs must not claim).
   - **May claim:** I/O Mesh = streaming org heartbeats / pulse (not OTel/APM) · Mesh base **~$88** · Memory Ops Pack **~$119** local-primary pull/retain/support · **dual_write OFF** · local-primary · Palace sunset · Salesforce **GA CRM** · HubSpot + GTM suite **Beta multi-tenant** · guerrilla global-only · knowledge/analytical **Beta** · agentic list/plan residual · portal HITL · catalog ≠ Connected · three planes via `/onboard next planes`.
   - **Must not claim:** invent Connected / install APPLY green / INSTALL_STORE green · Memory GA · dual_write ON · freemium hosted palace · Ops Pack = GPU fleet · book-demo ON (book-demo **OFF**) · dual-auth live · empty-as-none invent · agent MCP write installs · invent Knowledge/Analytics GA · invent human-gate green (Slack HMAC · Stripe Customers:Write · H1/H2 still human).
   - Honesty locks: drafts only for GTM · book-demo **OFF** · dual_write **OFF** · not Memory GA · never invent Connected · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open · rates ~$88/$119 optional · `dual_auth_candidacy_open` · tool ship ≠ dual-auth live · mesh ≠ memory · no invent GA.
   - Slash: `/onboard next sales` (aliases `claims` / `buyer` / `claim-matrix` / `sales-claims` / `buyer-claims`).
   - **Do not steal:** `product` / `planes` stay three-planes · `gtm` / `drafts` stay GTM lane · `pulse` / `board` stay status board.
   - API: `AionAgentOnboardingNextSalesClaims`.
   - Companion: `/onboard next planes` · `/onboard next mesh` · `/onboard next memory-pull` · `/onboard next agentic` · `/onboard next agentic dual-auth` · `/onboard next human-gates` · `/onboard next status` · `/onboard next export` · `/onboard next gtm`.

#### D3g. Demo readiness board (`/onboard next demo` · s1442)

3g. **Demo readiness residual-honest package** — offline static board for founders/operators (Lighthouse packaging · book-demo OFF · Landgrab NOT READY).
   - **Packaging:** Lighthouse beachhead · B2B SaaS · book-demo **OFF** · secondary CTA See pricing · leave ON_SIGNAL unset · rates ~$88/~$119 optional.
   - **Landgrab:** **NOT READY** / empty-honest · residual PASS ≠ logos met · do not invent book-demo ON · landgrab alias stays honesty NOT READY.
   - **Three planes companion:** `/onboard next planes` — mesh · memory-pull · agentic residual-honest · never invent Connected.
   - **Sales claims companion:** `/onboard next sales` — may claim / must not claim residual-honest.
   - **Human gates still open:** Slack HMAC · Stripe Customers:Write · H1/H2 INSTALL_STORE · K-D* · tip `/onboard next human-gates` · open boxes stay open.
   - **Demo path residual:** founder-led walkthrough only when scheduled · operator runbook ≠ public /demo booking live.
   - Honesty locks: dual_write **OFF** · book-demo **OFF** · Landgrab **NOT READY** · not Memory GA · never invent Connected · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open · rates ~$88/$119 optional · dual_auth_candidacy_open.
   - Slash: `/onboard next demo` (aliases `demo-ready` / `readiness` / `demo-readiness` / `lighthouse` / `landgrab`).
   - **Do not steal:** `sales` / `claims` stay sales claims · `product` / `planes` stay three-planes · `pulse` / `board` stay status board · `gtm` / `drafts` stay GTM lane.
   - Companion: `/onboard next planes` · `/onboard next sales` · `/onboard next human-gates` · `/onboard next mesh` · `/onboard next memory-pull` · `/onboard next agentic` · `/onboard next status` · `/onboard next export`.

#### D3i. Setup lifecycle map (`/onboard next setup` · s1542+s1558 · stage 4 of edge-user-journey · P1–P7 closeout residual)

3i. **Setup lifecycle residual-honest map** — offline static consolidation of setup P1–P7 story (init · preflight · reload · portal · pull · analyze · drift · guided repair) · **stage 4** of edge-user-journey.
   - Steps: `/setup init` · start memory host · `/setup preflight` (PASS ≠ invent Connected) · `/setup reload` (package wire ≠ Connected) · `/setup portal` HITL · optional `/setup pull start` · optional `/setup analyze start` · `/setup drift` report-only · `/setup repair plan` · `/setup repair apply --yes` (safe steps only · dual_write never auto ON) · `/memory digest` still valid.
   - Honest board vocab: `path_ready` · `residual_only` · **`setup_not_probed`** — never invent Connected / install green.
   - **repair apply ≠ invent Connected** · dual_write **OFF** · not Memory GA · Edge Memory GA candidacy only · portal HITL · catalog ≠ Connected · **E10 Open** · still-human APPLY open · free eng **s1558**.
   - offline static lane ≠ live dogfood · setup closeout residual ≠ invent Edge Memory GA.
   - Slash: `/onboard next setup` (aliases `setup-lifecycle` / `lifecycle` / `setup_lifecycle`).
   - API: `AionAgentOnboardingNextSetupLane`.
   - Companion: `/onboard next journey` (full first-run map) · `/onboard next memory` · `/onboard next memory-pull` · `/onboard next human-gates` · `/onboard next operator` · skill `setup-lifecycle-agent` · docs `docs/architecture/setup-lifecycle.md` · `docs/architecture/edge-user-journey.md` · `docs/architecture/memory-edge-usage-demo.md`.

#### D3j. Edge-user-journey first-run map (`/onboard next journey` · s1558 Wave B)

3j. **7-stage edge-user-journey first-run residual-honest map** — offline static first-run operator surface after Wave A docs SSOT (s1554).
   - Stages: **1 Signup** (portal · optional pure local) · **2 Download TUI** · **3 TUI auth/keys** (LLM/Ollama · no invent TUI portal SSO) · **4 Setup wizard** (`/setup` · `/onboard next setup`) · **5 Connectors** (`/integrations` list/plan · `/onboard next portal-hitl` · portal HITL · soft dogfood residual s1562) · **6 Local store** (`iomesh-memory-mcp` · host not auto · `/onboard next e4` soft residual s1566) · **7 Analyze** (`/memory digest` · `/setup analyze`).
   - Honesty: dual_write **OFF** · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA · portal HITL · agent MCP cannot write installs · catalog ≠ Connected · book-demo **OFF** · free eng **s1558** · free-floor peer **s1560+** mention only.
   - Residual gaps: no SSO invent · host not auto · portal HITL still human · dual_write OFF · Edge Memory GA candidacy only.
   - Slash: `/onboard next journey` (aliases `edge-journey` / `user-journey` / `first-run` / `edge_user_journey`).
   - API: `AionAgentOnboardingNextJourneyLane`.
   - Companion: `/onboard next wizard` (s1570 Wave C) · `/onboard next wizard dogfood` · `/onboard next setup` · `/onboard next portal-hitl` (stage 5 · s1562) · `/onboard next portal-hitl dogfood` · `/onboard next e4` (stage 6 · s1566) · `/onboard next e4 dogfood` · `/onboard next agentic` · `/onboard next memory` · `/onboard next human-gates` · `/onboard next operator` · docs `docs/architecture/edge-user-journey.md` · `docs/architecture/setup-lifecycle.md` · `docs/architecture/memory-edge-usage-demo.md` · `docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md`.


#### D3k. Portal HITL stage-5 board + soft dogfood (`/onboard next portal-hitl` · s1562)

3k. **Portal HITL connectors (journey stage 5)** — residual-honest offline board for MCP list/plan → browser portal HITL → human OAuth/install.
   - Path: MCP list/plan residual-honest · proven deep links `/integrations/{id}` · `/integrations/add?template={id}` · `/integrations` · human finishes OAuth/install in portal HITL when connect.
   - **agent MCP cannot write installs** · **catalog ≠ Connected** · `template=` ≠ install APPLY · portal HITL still · dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA.
   - Soft offline dogfood: `/onboard next portal-hitl dogfood` (aliases 4th token `soft` / `samples` / `offline` / `portal-hitl-soft`) → `RunPortalHITLSoftDogfood`.
     - Default session: `portal_hitl_soft_not_run`
     - Soft pass: `soft_offline_portal_hitl_session_pass`
     - Soft fail: `soft_offline_portal_hitl_session_fail`
     - **soft offline ≠ invent Connected** · **session soft ≠ live dogfood** · residual PASS ≠ live dogfood · independent of agentic list/plan soft (s1422) and plugins soft.
   - Slash: `/onboard next portal-hitl` (aliases `hitl` / `portal_hitl` / `portal-dogfood` / `stage5` / `connectors-hitl`) — **bare stays board** (not auto dogfood).
   - API: `AionAgentOnboardingNextPortalHITLLane` · `RunPortalHITLSoftDogfood` · `PortalHITLSoftSessionLabel`.
   - Companion: `/onboard next agentic` · `/onboard next agentic dogfood` · `/onboard next journey` · `/onboard portal` · `/integrations list|plan|status`.
   - free eng **s1562** · free-floor peer **s1564+** mention only (do not rewrite free-floor).

#### D3l. E4 client-attach stage-6 board + soft dogfood (`/onboard next e4` · s1566)

3l. **E4 client attach (journey stage 6 local store / MCP attach)** — residual-honest offline board for lean `iomesh-memory-mcp` → `iomesh mcp --connect` · tools=6 stamp residual.
   - Path: local-primary `iomesh-memory-mcp` · E4 client attach · tools=6 · `iomesh mcp --connect` residual · evidence `docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md`.
   - **dual_write OFF** · **not Memory GA** · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · **E10 Open** · tip ≠ invent forever-green product dogfood · residual PASS ≠ live dogfood · PASS ≠ live APPLY · book-demo OFF.
   - Soft offline dogfood: `/onboard next e4 dogfood` (aliases 4th token `soft` / `samples` / `offline` / `e4-soft`) → `RunE4SoftDogfood`.
     - Default session: `e4_soft_not_run`
     - Soft pass: `soft_offline_e4_session_pass`
     - Soft fail: `soft_offline_e4_session_fail`
     - **never dial MCP · never start host** · soft offline ≠ invent Connected · session soft ≠ live dogfood · residual PASS ≠ live dogfood · residual PASS ≠ invent Edge Memory GA declared · independent of portal HITL soft (s1562) and agentic list/plan soft (s1422).
   - Slash: `/onboard next e4` (aliases `e4-dogfood` / `client-attach` / `edge-memory-e4` / `e4_attach`) — **bare stays board** (not auto dogfood).
   - API: `AionAgentOnboardingNextE4Lane` · `RunE4SoftDogfood` · `E4SoftSessionLabel`.
   - Companion: `/onboard next memory` · `/onboard next journey` · `/onboard next memory-pull` · `/memory status`.
   - free eng **s1566** · free-floor peer **s1568+** mention only (do not rewrite free-floor).

#### D3h. Operator readiness matrix (`/onboard next operator` · s1447)

3h. **Operator readiness residual-honest matrix** — offline static board consolidating demo · sales · planes · human-gates for founders/operators.
   - **Row 1 Demo readiness:** Lighthouse · book-demo **OFF** · Landgrab **NOT READY** · companion `/onboard next demo` · residual PASS ≠ logos met.
   - **Row 2 Sales claims:** may claim / must not claim residual-honest · companion `/onboard next sales` · never invent Connected / Memory GA / dual-auth live.
   - **Row 3 Three planes:** mesh · memory-pull · agentic residual-honest · companion `/onboard next planes` · streams_not_probed · pull_not_probed · list_plan_not_connected.
   - **Row 4 Human gates:** edge-first · knowledge multi-tenant punted · Slack HMAC punted · portal HITL when connect · companion `/onboard next human-gates` · dual_write OFF · not Memory GA · Edge Memory GA candidacy only.
   - **Row 5 Agentic dual-auth:** `dual_auth_candidacy_open` · tool ship ≠ dual-auth live · list_org unavailable · companion `/onboard next agentic dual-auth`.
   - **Row 6 Policy locks:** dual_write **OFF** · not Memory GA · leave ON_SIGNAL unset · rates ~$88/$119 optional · no invent GA.
   - **Row 7 Export tip:** `/onboard next export` offline evidence · board/export evidence ≠ invent Connected.
   - Honest vocab: `residual_only` · `path_ready` · `still_human` · `policy_off` · `not_ready` · `portal_hitl_still`.
   - Honesty locks: book-demo **OFF** · Landgrab **NOT READY** · dual_write **OFF** · not Memory GA · never invent Connected · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open · dual_auth_candidacy_open.
   - Slash: `/onboard next operator` (aliases `operator-matrix` / `ops-matrix` / `operator-readiness` / `ops-readiness` / `matrix`).
   - **Do not steal:** `demo` / `readiness` / `lighthouse` / `landgrab` stay demo · `sales` / `claims` stay sales · `product` / `planes` stay three-planes · `pulse` / `board` stay status · `export` / `receipt` stay export.
   - API: `AionAgentOnboardingNextOperatorMatrix`.
   - Companion: `/onboard next demo` · `/onboard next sales` · `/onboard next planes` · `/onboard next human-gates` · `/onboard next agentic dual-auth` · `/onboard next status` · `/onboard next export`.

#### D4. Portal HITL still (all lanes)

4. **portal HITL still required for OAuth/install** — agent MCP **cannot write installs**.
   - catalog ≠ Connected · list_org fail-open ≠ empty-as-none · never invent Connected / INSTALL_STORE APPLY.
   - Portal: https://console.iome.sh/integrations · Agent/MCP: https://console.iome.sh/settings/agent.
   - Full agentic list/plan drill: **D3d** `/onboard next agentic` (s1417) · stage-5 portal HITL board: **D3k** `/onboard next portal-hitl` (s1562) · soft dogfood residual `/onboard next portal-hitl dogfood`.

#### D5. Lane status board (`/onboard next status` · s1382+s1397+s1402+s1407+s1417+s1422+s1427)

5. **Residual-honest lane status board** — offline pulse of plugins · gtm · memory · mesh · memory-pull · agentic · portal.
   - **No MCP dial** · never invent install green / Connected / GA / APPLY / stream green / pull green as success.
   - Honest state vocabulary only: `path_ready` · `samples_ok` · `samples_missing` · `dogfood_not_run` · `soft_offline_dogfood_session_pass` · `soft_offline_dogfood_session_fail` · `skill_ready` · `residual_only` · `streams_not_probed` · `pull_not_probed` · `portal_hitl_still` · `list_plan_not_connected` · `list_plan_soft_not_run` · `soft_offline_list_plan_session_pass` · `soft_offline_list_plan_session_fail` · `dual_auth_candidacy_open` · `list_org_unavailable`.
   - **plugins:** soft-check of sample dirs (`examples/agent-plugins`) · default `dogfood_not_run` · after `/plugins dogfood` session soft pass/fail (s1397) · **session soft ≠ live dogfood** · ≠ invent Agent Plugins GA · board ≠ invent Connected.
   - **gtm:** skill/checklist path ready · drafts only · no auto-send · ≠ invent GTM agent GA.
   - **memory:** dual_write OFF · package load ≠ Memory GA · local-primary ≠ freemium palace · mesh ≠ memory · **s1453+s1458+s1463+s1469+s1478** edge OSS tip (`iomesh-memory-mcp` · public product attach · go install · no GOPRIVATE · docker compose still valid · dual_write OFF · not Memory GA · aion broker private · flip complete residual ≠ invent Memory GA · public OSS ≠ invent platform GA · PASS ≠ invent full platform sidecar parity · mesh optional for pull · Palace sunset).
   - **mesh (s1402):** `path_ready` · `residual_only` · `streams_not_probed` · streaming org heartbeats · not OTel/APM · never invent stream green / Connected · empty streams honest.
   - **memory-pull (s1407):** `path_ready` · `residual_only` · `pull_not_probed` · Ops Pack pull path · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · never invent pull green.
   - **agentic (s1417+s1422+s1427):** `path_ready` · `residual_only` · `portal_hitl_still` · `list_plan_not_connected` · `<soft label>` · `dual_auth_candidacy_open` · `list_org_unavailable` · product plane 3 MCP list/plan residual-honest · never invent Connected · plan deep links = browser HITL only · template= ≠ install APPLY · session soft list/plan ≠ live dogfood · soft offline ≠ invent Connected · **tool ship ≠ dual-auth live**.
   - **portal:** `portal_hitl_still` · agent MCP cannot write installs.
   - Slash: `/onboard next status` (aliases `pulse` / `board`) · also linked from `/onboard next` overview and `/onboard status`.
   - Export receipt: `/onboard next export` (s1387+s1397+s1402+s1407+s1417+s1422+s1427) — offline markdown evidence of this board (includes session soft + mesh + memory-pull + agentic soft + dual-auth candidacy rows).

#### D6. Status export receipt (`/onboard next export` · s1387+s1397+s1402+s1407+s1417+s1422+s1427)

6. **Residual-honest status export receipt** — offline markdown (or optional JSON) evidence of the s1382 lane status board (plugins lane includes s1397 session soft marker when set · mesh row s1402 · memory-pull/ops_pack row s1407 · agentic row s1417+s1422 soft · dual-auth candidacy s1427).
   - Header: `evidence_kind=onboard_next_lane_status_export` · `offline_static` · `not_live_dogfood` · serial `s1387`.
   - Reuses honest vocabulary: `path_ready` · `samples_ok`/`samples_missing` · `dogfood_not_run` · `soft_offline_dogfood_session_pass|fail` · `skill_ready` · `residual_only` · `streams_not_probed` · `pull_not_probed` · `portal_hitl_still` · `list_plan_not_connected` · `list_plan_soft_not_run` · `soft_offline_list_plan_session_pass|fail` · `dual_auth_candidacy_open` · `list_org_unavailable`.
   - JSON field `plugins_dogfood_state` mirrors plugins session soft label; `dogfood_not_run` is true only when plugins session soft has not run; JSON field `agentic_list_plan_soft_state` mirrors agentic list/plan soft (independent · s1422); mesh lane value is `path_ready · residual_only · streams_not_probed`; memory-pull and ops_pack lanes are `path_ready · residual_only · pull_not_probed`; agentic lane is `path_ready · residual_only · portal_hitl_still · list_plan_not_connected · dual_auth_candidacy_open · list_org_unavailable · <soft label>`.
   - **Does NOT** run plugins dogfood · **does NOT** run agentic list/plan dogfood · **does NOT** dial MCP · **does NOT** invent install green / Connected / GA / APPLY / stream green / pull green / dual-auth live.
   - **session soft ≠ live dogfood** · **soft offline list/plan ≠ invent Connected** · **tool ship ≠ dual-auth live** · **board/export evidence ≠ invent Connected** — a stamped receipt is offline residual evidence only.
   - Markdown: `/onboard next export` (aliases `receipt` / `stamp` / `evidence`) → `AionAgentOnboardingNextLaneStatusExport`.
   - JSON: `/onboard next export json` → `AionAgentOnboardingNextLaneStatusExportJSON`.
   - Cross-linked from lane status board footer, `/onboard next` overview, `/plugins dogfood` tip, `/onboard next agentic dogfood` tip, and `/onboard next agentic dual-auth` tip.
   - Optional tip: `/onboard next agentic` (s1417) · `/onboard next agentic dogfood` (s1422) · `/onboard next agentic dual-auth` (s1427) · `/onboard next human-gates` (s1413) for still-required human APPLY residuals (does not invent green from this receipt).

#### D7. Human-gates honesty board + still-human APPLY soft dogfood (`/onboard next human-gates` · s1413+s1546+s1550+s1574 Wave C continuum)

7. **Residual-honest human-gates status** — edge-first launch residual vs punted/demoted vs offline residual only / shipped-or-policy · still-human APPLY open inventory reaffirm after Wave A–C continuum.
   - Slash: `/onboard next human-gates` (aliases `human` / `gates` / `apply-gates` / `still-human` / `apply-residual`) → `AionAgentHumanGatesHonestyBoard`.
   - Soft offline dogfood: `/onboard next human-gates dogfood` (aliases 4th token `soft` / `samples` / `offline` / `still-human-soft` / `apply-soft`) → `RunStillHumanApplySoftDogfood`.
     - Default session: `still_human_soft_not_run`
     - Soft pass: `soft_offline_still_human_session_pass`
     - Soft fail: `soft_offline_still_human_session_fail`
     - **never dial MCP · never start host** · soft offline ≠ invent Connected · session soft ≠ live dogfood · residual PASS ≠ live dogfood · PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open · residual PASS ≠ invent Edge Memory GA declared · independent of wizard/E4/portal HITL/agentic soft.
   - **architecture (locked):** edge-first local memory · dual_write OFF · hosted palace sunset · knowledge multi-tenant INSTALL_STORE punted · H1/H2 not launch gate · Slack HMAC punted · integrations path = TUI agent MCP list/plan + portal HITL.
   - **still_human_or_policy (edge-first launch residual · still-human APPLY open):** portal HITL OAuth/connect when customer must Connect · catalog ≠ Connected · book-demo OFF · leave ON_SIGNAL unset · dual_write OFF · not Memory GA · Edge Memory GA candidacy only · E10 Open · residual PASS ≠ invent Edge Memory GA declared · E10 founder sign-off only if claiming Edge Memory GA · optional E4 edge dogfood if tightening claim beyond candidacy.
   - **open inventory residual-honest (s1574):** Slack HMAC punted/still open as policy · Stripe Customers:Write residual · H1/H2 knowledge INSTALL_STORE punted/not launch gate · OAuth Connected still portal HITL · book-demo OFF · dual_write OFF · E10 Open.
   - **punted_or_demoted:** Slack HMAC rotate punted · H1/H2 knowledge INSTALL_STORE punted with multi-tenant · knowledge D1–D5 deferred · Stripe key/ACL largely closed (ACL residual only if Dashboard regresses).
   - **offline_residual_only / shipped_or_policy:** setup lifecycle residual complete ≠ invent Connected / Memory GA · Wave A–C continuum residual ≠ invent human-gate green / live APPLY · agent MCP list/plan residual-honest · dual_write OFF · not Memory GA · Edge Memory GA candidacy only · local memory / dual_write OFF do NOT invent Edge Memory GA declared.
   - Operator: `/onboard next human-gates` · `/onboard next human-gates dogfood` · `/onboard next wizard` · `/onboard next setup` · `/integrations list|plan|status` · **never invent Connected / INSTALL_STORE green / Edge Memory GA / book-demo ON / human-gate green**.
   - free eng **s1574** · free-floor peer **s1576+** mention only (do not rewrite free-floor).

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
| package load ≠ Memory GA | Loading iomesh-memory-mcp / Ops Pack local ≠ invent Memory Palace GA / freemium palace |
| iomesh-memory-mcp | Product edge MCP host only (s1453 Option A · s1517) |
| public product attach (s1478) | Both edge repos public · go install · no GOPRIVATE · HTTP `http://127.0.0.1:8080/mcp` or stdio · docker compose still valid |
| flip complete residual ≠ invent Memory GA | Public edge packs ≠ invent Memory GA / freemium palace |
| public OSS ≠ invent platform GA | Public MIT edge modules ≠ invent multi-tenant platform Memory GA |
| PASS ≠ invent full platform sidecar parity | Lean tool surface may lag platform residual · attach residual ≠ invent full parity |
| aion broker private | aion stays private cloud broker/CP · not OSS edge pack |
| s1517 product-only | In-tree residual aion Memory sample removed · aion broker private |
| mesh optional for pull | Mesh is optional feed via pull only · not required for local-primary Memory |
| GTM checklist ≠ GTM agent GA | `/gtm checklist` residual-honest draft path only — not invent GTM agent GA |
| board/export evidence ≠ invent Connected | Lane status board + export receipt are offline residual evidence only — never invent Connected / GA / APPLY |
| session soft ≠ live dogfood | `/plugins dogfood` or `/onboard next agentic dogfood` session marker on status/export is soft offline residual only — not live dogfood · not Agent Plugins GA · not invent Connected |
| soft offline list/plan ≠ invent Connected | Agentic list/plan soft offline dogfood (s1422) never invents Connected / install APPLY / dual-auth live |
| soft offline ≠ invent Connected | Portal HITL soft offline dogfood (s1562) · E4 soft offline dogfood (s1566) never invent Connected / install APPLY / live dogfood |
| residual PASS ≠ invent Edge Memory GA declared | E4 soft residual (s1566) · attach stamp · tip never invent Edge Memory GA declared · candidacy only |
| E10 Open | Founder/GTM Edge Memory GA sign-off remains open · residual PASS ≠ invent E10 closed |
| tip ≠ invent forever-green product dogfood | One observed E4 stamp · soft residual PASS ≠ invent forever-green full product dogfood |
| residual PASS ≠ live dogfood | Soft offline residual PASS is not live dogfood · not live APPLY · not invent Connected |
| tool ship ≠ dual-auth live | MCP tool `list_org_connector_installs` shipping residual ≠ invent dual-auth product live (s1427) |
| dual_auth_candidacy_open | Org installs dual-auth is residual candidacy only · never invent dual-auth shipped |
| list_org_unavailable | list_org fail-open residual · available=false · status=unavailable · installs=null · never invent empty-as-none |
| mesh = streaming org heartbeats | I/O Mesh product plane 1 = governed `dept.*` org heartbeats — not OTel/APM · not medical · not hosted Memory Palace |
| mesh ≠ memory | Mesh streaming lane is separate from local-edge memory lane; pull = egress into local palace only |
| never invent stream green | Empty streams honest · `streams_not_probed` residual · never invent Connected / live stream green from board/export |
| pull ≠ freemium hosted palace | `iomesh memory pull` is mesh → local palace egress · dual_write OFF · Palace sunset · not freemium hosted palace |
| Ops Pack ≠ GPU fleet | Memory Ops Pack ~$119 = pull / audit / Extended retain / support — not hosted GPU palace; mesh base ~$88 separate |
| pull_not_probed | Ops Pack pull path residual honest until operator probes · never invent pull green from board/export |
| package load ≠ Ops Pack entitlement | Loading packages / residual path ≠ invent Ops Pack commercial entitlement |
| list_plan_not_connected | Agentic MCP list/plan residual · catalog ≠ Connected · never invent install Connected from list/plan |
| plan deep links = browser HITL only | plan_connector_setup deep links are browser HITL · template= ≠ install APPLY green |
| no invent GA knowledge/analytical | Do not invent GA for knowledge or analytical mesh layers |
| PASS ≠ invent human-gate green | Offline residual / board / skill PASS never invents Slack HMAC green · Stripe Write · INSTALL_STORE APPLY · H1/H2 green |
| PASS ≠ live APPLY | Residual gate PASS is not live image tip / signed dogfood APPLY |
| open boxes stay open | Human launch residuals stay open until human evidence — do not invent checked green offline |
| Knowledge Beta→GA cannot invent H1/H2 offline | Exit-Beta candidacy cannot invent H1/H2 INSTALL_STORE image APPLY from offline residual |
| leave ON_SIGNAL unset | No invent warm APPLY / bootstrap warm green from residual board |
| local memory / dual_write OFF / agent MCP list/plan do not close human APPLY gates | Offline agent paths never close Slack HMAC · Stripe · H1/H2 · D1–D5 |

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
- Do **not** invent Memory GA / freemium palace from local `iomesh-memory-mcp` package load.
- Do **not** invent Memory GA / platform GA from public product attach residual (`iomesh-memory-mcp` go install · no GOPRIVATE · dual_write OFF · flip complete residual ≠ invent Memory GA · public OSS ≠ invent platform GA · aion broker private · s1478).
- Do **not** invent full platform sidecar parity / Memory GA from lean edge host attach residual (`iomesh-memory-mcp` · dual_write OFF · PASS ≠ invent full platform sidecar parity).
- Do **not** invent live dogfood green / Memory GA from M3 edge dogfood tip residual (`iomesh-memory-mcp` compose/HTTP · offline dogfood tip ≠ invent live dogfood as green · s1463).
- Do **not** invent Connected / GA / APPLY from board or export receipt evidence stamps.
- Do **not** treat session soft dogfood pass/fail as live dogfood or Agent Plugins GA.
- Do **not** invent stream green / Connected / live mesh from residual soft status/streams (empty streams honest · `streams_not_probed`).
- Do **not** conflate mesh streaming heartbeats with local-edge memory lane (mesh ≠ memory · not OTel/APM · not freemium hosted palace).
- Do **not** treat `iomesh memory pull` as freemium hosted palace or dual_write ON.
- Do **not** invent pull green / Ops Pack entitlement / GPU fleet from residual soft memory-pull path (`pull_not_probed` · Ops Pack ≠ GPU fleet).
- Do **not** steal bare `pull` under `/onboard next` for memory-pull — bare `pull` stays mesh (s1402); memory-pull uses `memory-pull` / `ops-pack` / `pull-path` / `memorypull` / `ops_pack`.
- Do **not** invent human-gate green from offline residual / human-gates board / residual gate PASS (Slack HMAC · Stripe Customers:Write · H1/H2 INSTALL_STORE · D1–D5 stay open).
- Do **not** invent live APPLY / INSTALL_STORE green / book-demo ON / ON_SIGNAL set from residual soft status.
- Do **not** treat local memory · dual_write OFF · agent MCP list/plan as closing human APPLY gates.
- Do **not** invent Connected / install green / INSTALL_STORE APPLY from residual-honest agentic MCP list/plan or plan deep links (`list_plan_not_connected` · template= ≠ install APPLY · deep_links = browser HITL only).
- Do **not** treat agentic list/plan soft offline dogfood pass/fail as live dogfood or invent Connected (s1422 · `list_plan_soft_not_run` default · soft offline ≠ invent Connected).
- Do **not** treat portal HITL soft offline dogfood pass/fail as live dogfood or invent Connected (s1562 · `portal_hitl_soft_not_run` default · soft offline ≠ invent Connected · session soft ≠ live dogfood · residual PASS ≠ live dogfood).
- Do **not** treat E4 soft offline dogfood pass/fail as live dogfood or invent Edge Memory GA declared / forever-green product dogfood / E10 closed / dual_write ON (s1566 · `e4_soft_not_run` default · never dial MCP · never start host · residual PASS ≠ invent Edge Memory GA declared · tip ≠ invent forever-green product dogfood · residual PASS ≠ live dogfood).
- Do **not** steal bare `portal` under `/onboard` for portal-hitl next lane — bare `portal` stays portal handoff; portal HITL next uses `portal-hitl` / `hitl` / `stage5` / `connectors-hitl` (s1562).
- Do **not** steal bare `mcp` / `palace` under `/onboard next` for E4 — those stay memory lane; E4 uses `e4` / `e4-dogfood` / `client-attach` / `edge-memory-e4` / `e4_attach` (s1566).
- Do **not** steal bare `mcp` under `/onboard next` for agentic — bare `mcp` stays memory; agentic uses `agentic` / `agentic-integrations` / `integrations` / `list-plan`.
- Do **not** steal bare `portal` / `agent-mcp` under `/onboard` for agentic — those stay portal handoff; portal HITL next-lane uses `portal-hitl`/`hitl` (s1562).
- Do **not** steal bare `/onboard next dogfood` for agentic soft — bare `dogfood` stays plugins lane; agentic soft uses `/onboard next agentic dogfood`.
- Do **not** invent dual-auth live / dual-auth shipped from residual candidacy or tool ship (`tool ship ≠ dual-auth live` · `dual_auth_candidacy_open` · `list_org_unavailable` · s1427).
- Do **not** invent empty-as-none installs from `list_org_connector_installs` fail-open (`available=false` · `status=unavailable` · `installs=null` not `[]`).
- Do **not** steal dogfood soft 4th tokens for dual-auth — `dogfood`/`soft`/`samples`/`offline`/`list-plan-soft` stay soft dogfood (s1422); dual-auth uses `dual-auth`/`candidacy`/`list-org`/`org-installs`/`dual_auth`/`dual-auth-candidacy`.
- Do **not** invent stream green / pull green / Connected from the three product planes board (s1432 · `streams_not_probed` · `pull_not_probed` · `list_plan_not_connected` · `dual_auth_candidacy_open`).
- Do **not** steal `pulse`/`board` for three planes — those stay status board; planes uses `planes`/`three-planes`/`product-planes`/`product`/`pillars`/`three_planes`.
- Do **not** invent Connected / Memory GA / dual-auth live / human-gate green / freemium palace / Ops Pack GPU from the sales/buyer claims board (s1437 · may claim residual-honest only · must not invent GA).
- Do **not** steal `product`/`planes` for sales claims — those stay three-planes; sales uses `sales`/`claims`/`buyer`/`claim-matrix`/`sales-claims`/`buyer-claims`.
- Do **not** steal `gtm`/`drafts` for sales claims — those stay GTM draft lane.
- Do **not** steal `pulse`/`board` for sales claims — those stay status board.
- Do **not** invent book-demo ON / Landgrab READY / Connected / Memory GA / logos met from the demo readiness board (s1442 · Lighthouse packaging residual-honest · Landgrab **NOT READY** · residual PASS ≠ logos met).
- Do **not** steal `sales`/`claims` for demo readiness — those stay sales claims; demo uses `demo`/`demo-ready`/`readiness`/`demo-readiness`/`lighthouse`/`landgrab`.
- Do **not** steal `product`/`planes` for demo readiness — those stay three-planes.
- Do **not** steal `pulse`/`board` for demo readiness — those stay status board.
- Do **not** steal `gtm`/`drafts` for demo readiness — those stay GTM draft lane.
- Do **not** invent Connected / Memory GA / dual-auth live / human-gate green / book-demo ON / Landgrab READY from the operator readiness matrix (s1447 · residual_only · path_ready · still_human · policy_off · not_ready · portal_hitl_still).
- Do **not** steal `demo`/`readiness`/`lighthouse`/`landgrab` for operator matrix — those stay demo board; operator uses `operator`/`operator-matrix`/`ops-matrix`/`operator-readiness`/`ops-readiness`/`matrix`.
- Do **not** steal `sales`/`claims` for operator matrix — those stay sales claims.
- Do **not** steal `product`/`planes` for operator matrix — those stay three-planes.
- Do **not** steal `pulse`/`board` for operator matrix — those stay status board.
- Do **not** steal `export`/`receipt` for operator matrix — those stay export receipt.

## Related

- Builtin skill always available when skills enabled (**s1363+s1368+s1372+s1377+s1382+s1387+s1397+s1402+s1407+s1413+s1417+s1422+s1427+s1432+s1437+s1442+s1447** · molds s1251 connector + s1288 memory-advanced + s1341 gtm-draft-only).
- System note inject on `AttachMCP`: `<aion-onboarding>` via `AionAgentOnboardingGuidanceNote` (s1363+s1368+s1372+s1377+s1382+s1387+s1402+s1407+s1413+s1417+s1432+s1437+s1442+s1447).
- Portal handoff block: `AionAgentOnboardingPortalHandoff` · slash `/onboard portal` (aliases `agent-mcp` / `mcp`).
- Offline status: `AionAgentOnboardingStatus` · slash `/onboard status` (no MCP dial) · cross-links `/onboard next status` (s1382+s1397+s1402+s1407+s1417+s1422+s1427) · `/onboard next export` (s1387+s1397+s1402+s1407+s1417+s1422+s1427) · `/onboard next mesh` (s1402) · `/onboard next memory-pull` (s1407) · `/onboard next agentic` (s1417) · `/onboard next agentic dogfood` (s1422) · `/onboard next agentic dual-auth` (s1427) · `/onboard next planes` (s1432) · `/onboard next sales` (s1437) · `/onboard next demo` (s1442) · `/onboard next operator` (s1447) · `/onboard next human-gates` (s1413).
- Post-onboard next lanes overview: `AionAgentOnboardingNextLanes` · slash `/onboard next` (aliases `after` / `continue` / `lanes`) — plugins · gtm · memory local · mesh streaming · Ops Pack pull path · agentic integrations product plane 3 · three product planes board (s1432) · sales/buyer claims (s1437) · demo readiness (s1442) · operator readiness matrix (s1447) · portal HITL still (s1372+s1402+s1407+s1417) · human-gates still-required vs offline (s1413) · status board (s1382+s1397+s1402+s1407+s1417+s1422+s1427) · export receipt (s1387+s1397+s1402+s1407+s1417+s1422+s1427).
- Lane drills (s1377+s1402+s1407+s1417): `AionAgentOnboardingNextPluginsLane` · `AionAgentOnboardingNextGtmLane` · `AionAgentOnboardingNextMemoryLane` · `AionAgentOnboardingNextMeshLane` · `AionAgentOnboardingNextMemoryPullLane` · `AionAgentOnboardingNextAgenticLane` · slash `/onboard next [plugins|gtm|memory|mesh|memory-pull|agentic]` (aliases plugin|dogfood · drafts · mcp|palace · stream|streams|heartbeat|heartbeats|pull · ops-pack|pull-path|memorypull|ops_pack · agentic-integrations|integrations|list-plan).
- Three product planes board (s1432): `AionAgentOnboardingNextThreePlanes` · slash `/onboard next planes` (aliases three-planes|product-planes|product|pillars|three_planes) · mesh · memory-pull · agentic residual-honest · streams_not_probed · pull_not_probed · list_plan_not_connected · dual_auth_candidacy_open · never invent Connected · do not steal pulse/board/pull/mcp.
- Sales/buyer claims board (s1437): `AionAgentOnboardingNextSalesClaims` · slash `/onboard next sales` (aliases claims|buyer|claim-matrix|sales-claims|buyer-claims) · may claim / must not claim residual-honest · three-planes grounded · never invent Connected / Memory GA / dual-auth live · do not steal product/planes/gtm/pulse/board.
- Demo readiness board (s1442): `AionAgentOnboardingNextDemoReadiness` · slash `/onboard next demo` (aliases demo-ready|readiness|demo-readiness|lighthouse|landgrab) · Lighthouse beachhead packaging · book-demo OFF · Landgrab NOT READY · three planes · sales claims · human gates still open · residual PASS ≠ logos met · never invent Connected / book-demo ON · do not steal sales/claims/product/planes/pulse/board/gtm/drafts.
- Operator readiness matrix (s1447): `AionAgentOnboardingNextOperatorMatrix` · slash `/onboard next operator` (aliases operator-matrix|ops-matrix|operator-readiness|ops-readiness|matrix) · demo · sales · planes · human-gates · dual-auth candidacy · policy locks residual-honest · residual_only · path_ready · still_human · policy_off · not_ready · portal_hitl_still · never invent Connected / GA · do not steal demo/readiness/lighthouse/landgrab/sales/claims/product/planes/pulse/board/export/receipt.
- Setup lifecycle map (s1542+s1558 · stage 4): `AionAgentOnboardingNextSetupLane` · slash `/onboard next setup` (aliases setup-lifecycle|lifecycle|setup_lifecycle) · P1–P7 closeout residual · setup_not_probed · package wire ≠ Connected · repair apply ≠ invent Connected · dual_write never auto ON · E10 Open · offline static ≠ live dogfood · setup closeout residual ≠ invent Edge Memory GA · companion `/setup` slash + skill `setup-lifecycle-agent` · full first-run `/onboard next journey`.
- Edge-user-journey first-run map (s1558 Wave B): `AionAgentOnboardingNextJourneyLane` · slash `/onboard next journey` (aliases edge-journey|user-journey|first-run|edge_user_journey) · 7 stages residual-honest · dual_write OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA · portal HITL · no invent TUI portal SSO · host not auto · free eng s1558 · free-floor peer s1560+ mention only · docs edge-user-journey · setup-lifecycle · memory-edge-usage-demo.
- Portal HITL stage-5 board (s1562): `AionAgentOnboardingNextPortalHITLLane` · slash `/onboard next portal-hitl` (aliases hitl|portal_hitl|portal-dogfood|stage5|connectors-hitl) · journey stage 5 · portal HITL when connect · soft dogfood residual · free eng s1562 · free-floor peer s1564+ mention only.
- Portal HITL soft offline dogfood (s1562): `RunPortalHITLSoftDogfood` · slash `/onboard next portal-hitl dogfood` (aliases soft|samples|offline|portal-hitl-soft) · default `portal_hitl_soft_not_run` · soft offline ≠ invent Connected · session soft ≠ live dogfood · bare portal-hitl stays board · independent of agentic list/plan soft.
- E4 client-attach stage-6 board (s1566): `AionAgentOnboardingNextE4Lane` · slash `/onboard next e4` (aliases e4-dogfood|client-attach|edge-memory-e4|e4_attach) · journey stage 6 · E4 client attach · tools=6 · iomesh mcp --connect residual · free eng s1566 · free-floor peer s1568+ mention only.
- First-run wizard residual (s1570 Wave C): `AionAgentOnboardingNextWizardLane` · slash `/onboard next wizard` (aliases first-run-wizard|guided|wave-c|wave_c|wizard-residual) · soft `/onboard next wizard dogfood` (aliases soft|samples|offline|wizard-soft) → `RunFirstRunWizardSoftDogfood` · default `wizard_soft_not_run` · after run `soft_offline_wizard_session_pass` | `soft_offline_wizard_session_fail` · residual PASS ≠ invent full interactive auto wizard · residual PASS ≠ invent Edge Memory GA declared · free eng s1570 · free-floor peer s1572+ mention only · bare wizard stays board · independent of E4/portal HITL/agentic soft · companion still-human soft s1574.
- E4 soft offline dogfood (s1566): `RunE4SoftDogfood` · slash `/onboard next e4 dogfood` (aliases soft|samples|offline|e4-soft) · default `e4_soft_not_run` · never dial MCP · never start host · residual PASS ≠ invent Edge Memory GA declared · E10 Open · tip ≠ invent forever-green product dogfood · residual PASS ≠ live dogfood · bare e4 stays board · independent of portal HITL soft and agentic list/plan soft.
- Agentic list/plan soft offline dogfood (s1422): `RunAgenticListPlanSoftDogfood` · slash `/onboard next agentic dogfood` (aliases soft|samples|offline|list-plan-soft) · session SSOT independent of plugins · default `list_plan_soft_not_run` · soft offline ≠ invent Connected · bare agentic stays board.
- Agentic dual-auth candidacy (s1427): `AionAgentOnboardingNextAgenticDualAuthCandidacy` · slash `/onboard next agentic dual-auth` (aliases candidacy|list-org|org-installs|dual_auth|dual-auth-candidacy) · dual_auth_candidacy_open · list_org_unavailable · tool ship ≠ dual-auth live · never invent empty-as-none · dogfood soft aliases not stolen.
- Human-gates honesty board (s1413+s1546+s1550+s1574 Wave C continuum): `AionAgentHumanGatesHonestyBoard` · slash `/onboard next human-gates` (aliases `human` / `gates` / `apply-gates` / `still-human` / `apply-residual`) — still-human APPLY open · edge-first · knowledge multi-tenant punted · Slack HMAC punted · portal HITL when connect · dual_write OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA · residual PASS ≠ invent Edge Memory GA declared · E10 Open · PASS ≠ invent Connected · PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open · H1/H2 not launch gate · agent MCP cannot write installs · book-demo OFF · leave ON_SIGNAL unset · free eng s1574.
- Still-human APPLY soft offline dogfood (s1574): `RunStillHumanApplySoftDogfood` · slash `/onboard next human-gates dogfood` (aliases soft|samples|offline|still-human-soft|apply-soft) · default `still_human_soft_not_run` · after run `soft_offline_still_human_session_pass` | `soft_offline_still_human_session_fail` · never dial MCP · never start host · PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open · residual PASS ≠ invent Edge Memory GA declared · free eng s1574 · free-floor peer s1576+ mention only · bare human-gates stays board · independent of wizard/E4/portal HITL/agentic soft.
- Lane status board (s1382+s1397+s1402+s1407+s1417+s1422+s1427): `AionAgentOnboardingNextLaneStatus` · slash `/onboard next status` (aliases `pulse` / `board`) — honest vocabulary only · no invent Connected/GA/APPLY/stream green/pull green · dogfood_not_run default · session soft pass/fail after `/plugins dogfood` · mesh `streams_not_probed` · memory-pull `pull_not_probed` · agentic `list_plan_not_connected` · agentic soft `list_plan_soft_not_run` default · dual_auth_candidacy_open · list_org_unavailable · session soft ≠ live dogfood · tool ship ≠ dual-auth live · tip `/onboard next planes` (s1432) · tip `/onboard next sales` (s1437) · tip `/onboard next demo` (s1442) · tip `/onboard next operator` (s1447).
- Status export receipt (s1387+s1397+s1402+s1407+s1417+s1422+s1427): `AionAgentOnboardingNextLaneStatusExport` · slash `/onboard next export` (aliases `receipt` / `stamp` / `evidence`) · optional `AionAgentOnboardingNextLaneStatusExportJSON` via `/onboard next export json` — evidence_kind=onboard_next_lane_status_export · offline_static · not_live_dogfood · plugins_dogfood_state session soft · agentic_list_plan_soft_state session soft (s1422) · mesh streams_not_probed · memory-pull/ops_pack pull_not_probed · agentic list_plan_not_connected · dual_auth_candidacy_open · list_org_unavailable · board/export evidence ≠ invent Connected · human-gates tip (s1413) · agentic tip (s1417) · agentic dogfood tip (s1422) · dual-auth tip (s1427) · planes tip (s1432) · sales tip (s1437).
- Companion builtin: `connector-integrations-setup` (list/plan → portal HITL).
- Companion builtin: `memory-advanced-agent` (opt-in advanced memory · dual_write OFF · not Memory GA).
- Companion builtin: `gtm-draft-only-agent` (drafts only · human publish · no auto-send).
- Slash residual honesty: `/onboard [help|checklist|portal|status|next]` · `/onboard next [plugins|gtm|memory|mesh|memory-pull|agentic|portal-hitl|e4|planes|sales|demo|operator|setup|journey|wizard|status|export|human-gates]` · `/onboard next portal-hitl dogfood` · `/onboard next e4 dogfood` · `/onboard next wizard dogfood` · `/onboard next human-gates dogfood` · `/onboard next agentic dogfood` · `/onboard next agentic dual-auth` · `/plugins dogfood|status` · `/integrations list|plan|status|signing` · `/memory status` · `/mesh` · `/gtm [help|checklist]` · `/setup [init|preflight|portal|reload|pull|analyze|drift|repair]`.
- Skills are **not** Agent Plugins — plugins dogfood ≠ invent Agent Plugins GA · session soft ≠ live dogfood · mesh ≠ memory · Ops Pack ≠ GPU fleet · human-gates offline ≠ invent APPLY · still-human soft ≠ invent human-gate green / live APPLY · agentic list/plan ≠ invent Connected · soft offline list/plan ≠ invent Connected · E4 soft ≠ invent Edge Memory GA declared / forever-green product dogfood · tool ship ≠ dual-auth live · three planes ≠ invent stream/pull/Connected green · sales claims ≠ invent Connected/Memory GA/dual-auth live · operator matrix ≠ invent Connected/GA/dual-auth live/book-demo ON · edge-user-journey residual map ≠ invent SSO/auto host/Memory GA/Connected.
