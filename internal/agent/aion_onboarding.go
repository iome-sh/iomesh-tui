package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iome-sh/iomesh-tui/internal/agentplugins"
)

// AionAgentOnboardingGuidanceNote residual-honest system note (s1363 + s1368 + s1372 + s1377 + s1382 + s1387 + s1402 + s1407).
// Injected on AttachMCP after integrations + memory-advanced notes.
// Steers TUI agent ↔ aion CP/MCP onboarding without inventing install green,
// Memory GA, Agent Plugins GA, or dual_write ON.
// s1368: adds explicit portal Agent/MCP handoff lane (mint key → copy MCP →
// test invoke probe only) complementary to integrations portal HITL.
// s1372: cross-link post-onboard continuum → /onboard next operator lanes.
// s1377: /onboard next <lane> drill-down (plugins · gtm · memory).
// s1382: /onboard next status lane status board (pulse|board aliases).
// s1387: /onboard next export status export receipt (aliases receipt|stamp|evidence).
// s1402: /onboard next mesh streaming lane (org heartbeats on dept.* · mesh ≠ memory).
// s1407: /onboard next memory-pull Ops Pack pull path (mesh → local palace egress).
// Unit-tested for honesty needles. Molds IntegrationsAgentGuidanceNote /
// GtmDraftOnlyAgentGuidanceNote / MemoryAdvancedAgentGuidanceNote.
func AionAgentOnboardingGuidanceNote() string {
	return strings.TrimSpace(`aion agent onboarding (residual-honest TUI ↔ aion CP/MCP · s1363+s1368+s1372+s1377+s1382+s1387+s1402+s1407):
Point IOMESH/MCP at aion tools — fail-open offline (never invent tool green).

Connector path (integrations portal HITL):
1. Discover: MCP list_connector_catalog — catalog status ≠ install Connected
2. Plan: MCP plan_connector_setup — portal deep links + honesty notes (browser HITL only)
3. Org installs residual: MCP list_org_connector_installs — fail-open (available=false · installs=null) ≠ empty-as-none · never invent Connected
4. Complete OAuth/install in portal HITL at https://console.iome.sh/integrations — agent MCP cannot write installs

Portal Agent/MCP lane (complementary · s1368 · credential → copy connection → test invoke):
- Portal: mint key / Settings → Agent/MCP → copy MCP connection → test invoke (probe only ≠ Memory GA)
- TUI: configure [[mcp.servers]] streamable HTTP → /onboard · /integrations status
- Console Agent/MCP: https://console.iome.sh/settings/agent (connectors still /integrations)

Memory + operator:
5. Memory: dual_write OFF · local-primary · not Memory GA · optional plugins dogfood ≠ invent Agent Plugins GA (rates ~$88/$119 optional)
6. Operator pulse: /integrations status · /onboard checklist · /onboard portal · portal HITL
7. Post-onboard continuum: /onboard next [plugins|gtm|memory|mesh|memory-pull|status|export] (plugins dogfood · /gtm checklist · aion-memory-mcp local · mesh streaming heartbeats · Ops Pack pull path · portal HITL still · lane status board · status export receipt)

Skill: read_skill aion-agent-onboarding when available

Locks (never violate):
- dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood
- never invent install green / Connected / INSTALL_STORE APPLY
- list_org_connector_installs available=false ≠ empty-as-none
- catalog status ≠ Connected · portal HITL for OAuth/install · agent MCP cannot write installs
- plugins dogfood ≠ invent Agent Plugins GA · rates ~$88/$119 optional
- no invent GA for knowledge/analytical
- test invoke = probe only ≠ Memory GA · mint key ≠ invent install Connected
- drafts only · no auto-send · package load ≠ Memory GA
- board/export evidence ≠ invent Connected
- mesh = streaming org heartbeats · mesh ≠ memory · never invent stream green / Connected · not OTel/APM
- pull = mesh → local palace egress · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · pull_not_probed honest`)
}

// AionAgentOnboardingChecklist residual-honest numbered onboarding checklist (s1363 + s1368 + s1372 + s1377 + s1382 + s1387 + s1402 + s1407).
// Used by /onboard help and /onboard checklist — operator HITL only; never invents
// install green, Memory GA, Agent Plugins GA, dual_write ON, or agent APPLY.
// s1368: portal Agent/MCP handoff steps (mint/copy/probe) + TUI [[mcp.servers]].
// s1372: cross-link → /onboard next operator lanes (post-onboard continuum).
// s1377: /onboard next [plugins|gtm|memory] lane drills.
// s1382: /onboard next status lane status board.
// s1387: /onboard next export status export receipt.
// s1402: /onboard next mesh streaming lane (org heartbeats).
// s1407: /onboard next memory-pull Ops Pack pull path.
func AionAgentOnboardingChecklist() string {
	return strings.TrimSpace(`aion agent onboarding checklist (residual-honest · s1363+s1368+s1372+s1377+s1382+s1387+s1402+s1407 · TUI ↔ aion):
  1. Point IOMESH/MCP at aion tools (fail-open offline)
  2. list_connector_catalog — catalog status ≠ Connected
  3. plan_connector_setup → portal deep links (browser HITL)
  4. list_org_connector_installs residual fail-open (available=false ≠ empty-as-none)
  5. Portal Agent/MCP: mint key → Settings → Agent/MCP → copy MCP connection → test invoke (probe only ≠ Memory GA) at https://console.iome.sh/settings/agent
  6. TUI: [[mcp.servers]] streamable HTTP → /onboard · /integrations status (agent MCP cannot write installs)
  7. Memory dual_write OFF · local-primary · not Memory GA · optional plugins dogfood ≠ Agent Plugins GA
  8. Operator: /integrations status · /onboard checklist · /onboard portal · portal https://console.iome.sh/integrations
  9. Post-onboard: /onboard next [plugins|gtm|memory|mesh|memory-pull|status|export] (plugins · gtm · memory local · mesh streaming heartbeats · Ops Pack pull path · portal HITL still · lane status board · status export receipt)
  Locks: never invent install green / Connected / INSTALL_STORE APPLY · book-demo OFF · residual PASS ≠ live dogfood · rates ~$88/$119 optional · no invent GA knowledge/analytical · catalog status ≠ Connected · portal HITL · drafts only · no auto-send · package load ≠ Memory GA · board/export evidence ≠ invent Connected · mesh = streaming org heartbeats · mesh ≠ memory · never invent stream green · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · pull_not_probed`)
}

// AionAgentOnboardingPortalHandoff residual-honest short block for /onboard portal (s1368).
// Portal Agent/MCP lane complementary to integrations portal HITL — mint key /
// copy MCP connection / test invoke (probe only) + TUI [[mcp.servers]] attach.
// Never invents install green, Memory GA, or agent write installs.
func AionAgentOnboardingPortalHandoff() string {
	return strings.TrimSpace(`aion portal Agent/MCP handoff (residual-honest · s1368):
Portal half (browser HITL · https://console.iome.sh/settings/agent):
  1. Mint API key / agent principal (settings only · not install APPLY)
  2. Settings → Agent/MCP → copy MCP connection (URL + auth env hint)
  3. Test invoke = probe only ≠ Memory GA · never invent tool green / Connected

TUI half (local config · streamable HTTP):
  4. Configure [[mcp.servers]] with url = streamable HTTP MCP endpoint (+ oauth_token_env if needed)
  5. Restart / reattach MCP → /onboard · /integrations status · /onboard status
  6. Connector OAuth/install still portal HITL at https://console.iome.sh/integrations — agent MCP cannot write installs

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · never invent install green / Connected / INSTALL_STORE APPLY · list_org fail-open ≠ empty-as-none · catalog ≠ Connected · portal HITL · plugins dogfood ≠ invent Agent Plugins GA`)
}

// AionAgentOnboardingStatus residual-honest static offline status lines for /onboard status (s1368 + s1372 + s1377 + s1382 + s1387 + s1402 + s1407).
// No MCP dial — operator pulse only. Never invents attach green, install Connected, or Memory GA.
// s1372: cross-link → /onboard next operator lanes.
// s1377: lane drills via /onboard next [plugins|gtm|memory].
// s1382: cross-link → /onboard next status lane status board.
// s1387: cross-link → /onboard next export status export receipt.
// s1402: cross-link → /onboard next mesh streaming lane.
// s1407: cross-link → /onboard next memory-pull Ops Pack pull path.
func AionAgentOnboardingStatus() string {
	return strings.TrimSpace(`aion onboard status (residual-honest · offline static · s1368+s1372+s1377+s1382+s1387+s1402+s1407):
  MCP attach: expected for full path · fail-open offline (never invent tool green / install green)
  dual_write OFF · local-primary · not Memory GA · book-demo OFF
  portal HITL: Agent/MCP mint/copy/probe @ https://console.iome.sh/settings/agent · connectors @ https://console.iome.sh/integrations
  never invent install green / Connected / INSTALL_STORE APPLY
  list_org fail-open (available=false) ≠ empty-as-none · catalog ≠ Connected
  agent MCP cannot write installs · plugins dogfood ≠ invent Agent Plugins GA
  residual PASS ≠ live dogfood · test invoke = probe only ≠ Memory GA
  mesh = streaming org heartbeats · mesh ≠ memory · never invent stream green
  pull = mesh → local palace egress · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · pull_not_probed
  slash: /onboard portal · /onboard checklist · /onboard next [plugins|gtm|memory|mesh|memory-pull|status|export] · /onboard next status · /onboard next export · /onboard next mesh · /onboard next memory-pull · /integrations status`)
}

// AionAgentOnboardingNextLanes residual-honest post-onboard continuum for /onboard next (s1372 + s1377 + s1382 + s1387 + s1402 + s1407).
// Static offline block — no MCP dial. Lists residual-honest operator lanes after
// core onboarding (plugins dogfood · GTM drafts · local memory · mesh streaming · Ops Pack pull · portal HITL still).
// s1377: drill-down via /onboard next plugins|gtm|memory (see lane helpers below).
// s1382: lane status board via /onboard next status (aliases pulse|board).
// s1387: status export receipt via /onboard next export (aliases receipt|stamp|evidence).
// s1402: mesh streaming lane via /onboard next mesh (aliases stream|streams|heartbeat|heartbeats|pull).
// s1407: memory-pull Ops Pack pull path via /onboard next memory-pull (aliases ops-pack|pull-path|memorypull|ops_pack).
// Never invents Agent Plugins GA, Memory GA, auto-send, install Connected, stream green, or pull green.
func AionAgentOnboardingNextLanes() string {
	return strings.TrimSpace(`aion onboard next lanes (residual-honest · post-onboard continuum · s1372+s1377+s1382+s1387+s1402+s1407 · no MCP dial):
  1. iomesh plugins dogfood · /plugins dogfood — offline sample validate (examples/agent-plugins) · ≠ invent Agent Plugins GA
     drill: /onboard next plugins (aliases plugin|dogfood) · slash: /plugins dogfood
  2. /gtm checklist + skill gtm-draft-only-agent — drafts only · no auto-send · human publish · GTM checklist ≠ invent GTM agent GA
     drill: /onboard next gtm (alias drafts)
  3. local aion-memory-mcp / Memory Ops Pack local-primary — dual_write OFF · package load ≠ Memory GA · ≠ freemium palace
     drill: /onboard next memory (aliases mcp|palace)
  4. I/O Mesh streaming org heartbeats on dept.* — mesh ≠ memory · not OTel/APM · not hosted Memory Palace · empty streams honest
     drill: /onboard next mesh (aliases stream|streams|heartbeat|heartbeats|pull) · residual soft: /mesh · iomesh mesh status|streams|consumer
  5. Memory Ops Pack pull path — iomesh memory pull = mesh → local palace egress · dual_write OFF · Ops Pack ≠ GPU fleet · pull_not_probed
     drill: /onboard next memory-pull (aliases ops-pack|pull-path|memorypull|ops_pack) · NOT bare pull (pull stays mesh lane)
  6. portal HITL still required for OAuth/install · agent MCP cannot write installs · catalog ≠ Connected
  status board: /onboard next status (aliases pulse|board) — residual-honest lane states only (never invent connected/ga/apply as success · pulse stays board)
  export receipt: /onboard next export (aliases receipt|stamp|evidence) — offline markdown evidence of board (board/export evidence ≠ invent Connected)

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · never invent install green / Connected / INSTALL_STORE APPLY · list_org fail-open ≠ empty-as-none · plugins dogfood ≠ invent Agent Plugins GA · drafts only · no auto-send · rates ~$88/$119 optional · package load ≠ Memory GA · board/export evidence ≠ invent Connected · mesh = streaming org heartbeats · mesh ≠ memory · never invent stream green / Connected · not OTel/APM · streams_not_probed honest · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · pull_not_probed honest · never invent pull green`)
}

// AionAgentOnboardingNextPluginsLane residual-honest plugins dogfood drill for /onboard next plugins (s1377+s1392).
// Static offline — iomesh plugins dogfood + /plugins dogfood path. Never invents Agent Plugins GA,
// install Connected, dual_write ON, or live dogfood green.
func AionAgentOnboardingNextPluginsLane() string {
	return strings.TrimSpace(`aion onboard next plugins lane (residual-honest · s1377+s1392 · no MCP dial):
  Path: iomesh plugins dogfood · /plugins dogfood — offline sample validate only
  Samples: examples/agent-plugins/{hello-iome,aion-memory-mcp}
  Steps:
    1. iomesh plugins list · /plugins list — closed-manifest discovery map (≠ invent install green / Connected)
    2. iomesh plugins validate <path> · /plugins validate — offline package shape residual
    3. iomesh plugins dogfood · /plugins dogfood — both in-repo samples offline (residual PASS ≠ live dogfood)
  Honesty:
    · plugins dogfood ≠ invent Agent Plugins GA · soft offline dogfood ≠ invent Agent Plugins GA
    · residual PASS ≠ live dogfood · never invent install green / Connected / INSTALL_STORE APPLY
    · catalog ≠ Connected · agent MCP cannot write installs · portal HITL still for OAuth/install
    · package load ≠ Memory GA · rates ~$88/$119 optional
  Slash: /plugins dogfood (aliases soft|samples|offline) · /plugins list · /plugins validate · /plugins status
  Back: /onboard next · companion samples offline only

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · never invent install green / Connected / INSTALL_STORE APPLY · plugins dogfood ≠ invent Agent Plugins GA · catalog ≠ Connected · portal HITL · agent MCP cannot write installs · package load ≠ Memory GA · rates ~$88/$119 optional`)
}

// AionAgentOnboardingNextGtmLane residual-honest GTM draft-only drill for /onboard next gtm (s1377).
// Static offline — /gtm checklist + gtm-draft-only-agent skill. Never invents auto-send,
// GTM agent GA, suite ops GA, or install Connected.
func AionAgentOnboardingNextGtmLane() string {
	return strings.TrimSpace(`aion onboard next gtm lane (residual-honest · s1377 · no MCP dial):
  Path: /gtm checklist + skill gtm-draft-only-agent — drafts only · no auto-send · human publish
  Steps:
    1. /gtm checklist (or /gtm help) — residual-honest draft-only checklist
    2. read_skill gtm-draft-only-agent — roles draft/plan only (Orchestrator · Content · Campaign · Lead)
    3. Draft content/outreach only — never auto-send email/SNS · human publish · human CRM commercial
  Honesty:
    · drafts only · no auto-send · human publish
    · GTM checklist ≠ invent GTM agent GA · residual PASS ≠ live dogfood publish
    · Salesforce = GA CRM; HubSpot + GTM suite Beta multi-tenant; guerrilla global-only
    · portal HITL for installs · agent MCP cannot write installs · never invent Connected / INSTALL_STORE APPLY
  Back: /onboard next · /gtm [help|checklist]

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · drafts only · no auto-send · human publish · GTM checklist ≠ invent GTM agent GA · never invent install green / Connected / INSTALL_STORE APPLY · catalog ≠ Connected · portal HITL · agent MCP cannot write installs · rates ~$88/$119 optional`)
}

// AionAgentOnboardingNextMemoryLane residual-honest memory local drill for /onboard next memory (s1377).
// Static offline — aion-memory-mcp / Memory Ops Pack local-primary. Never invents Memory GA,
// freemium palace, dual_write ON, or install Connected.
func AionAgentOnboardingNextMemoryLane() string {
	return strings.TrimSpace(`aion onboard next memory lane (residual-honest · s1377 · no MCP dial):
  Path: local aion-memory-mcp / Memory Ops Pack local-primary — dual_write OFF
  Steps:
    1. Attach aion-memory-mcp (local-primary) — package load ≠ Memory GA · ≠ freemium palace
    2. dual_write OFF · local-primary only · not Memory GA
    3. Optional: read_skill memory-advanced-agent (opt-in advanced · still dual_write OFF · not Memory GA)
    4. Operator pulse: /memory status · /onboard status (fail-open offline · never invent tool green)
  Honesty:
    · package load ≠ Memory GA · ≠ freemium palace · dual_write OFF
    · residual PASS ≠ live dogfood · test invoke = probe only ≠ Memory GA
    · never invent install green / Connected / INSTALL_STORE APPLY
    · catalog ≠ Connected · portal HITL · agent MCP cannot write installs
    · mesh ≠ memory · memory lane is local-edge palace, not streaming org heartbeats
  Back: /onboard next · /memory status · portal Agent/MCP https://console.iome.sh/settings/agent

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · package load ≠ Memory GA · ≠ freemium palace · never invent install green / Connected / INSTALL_STORE APPLY · catalog ≠ Connected · portal HITL · agent MCP cannot write installs · rates ~$88/$119 optional · mesh ≠ memory`)
}

// AionAgentOnboardingNextMeshLane residual-honest mesh streaming lane for /onboard next mesh (s1402).
// Static offline — I/O Mesh = streaming org heartbeats / governed dept.* (product plane 1).
// NOT hosted Memory Palace · not OTel/APM · not medical · mesh ≠ memory lane.
// Operator residual soft only (status/streams/consumers) · never invent stream green / Connected.
// Pull honesty: iomesh memory pull = mesh egress into local palace · dual_write OFF · not freemium hosted palace.
// Rates: mesh base ~$88 · Memory Ops Pack ~$119 pull/retain/support.
func AionAgentOnboardingNextMeshLane() string {
	return strings.TrimSpace(`aion onboard next mesh lane (residual-honest · s1402 · no MCP dial · product plane 1):
  Path: I/O Mesh = streaming org heartbeats on governed dept.* — NOT hosted Memory Palace · not OTel/APM
  Separation: mesh ≠ memory · mesh lane ≠ plugins/gtm lanes · pull ≠ freemium hosted palace · Palace sunset
  Steps:
    1. Residual soft operator: /mesh · iomesh mesh status (fail-open offline · never invent stream green / Connected)
    2. Streams residual: iomesh mesh streams — empty streams honest · streams_not_probed until operator probes
    3. Consumers residual: iomesh mesh consumer (durable pull consumers · residual soft · requires --yes when mutating)
    4. Pull honesty: iomesh memory pull = mesh → local palace egress · dual_write OFF · not freemium hosted palace · not Memory GA
  Honesty:
    · mesh = streaming org heartbeats · not OTel/APM · not medical · not hosted Memory Palace
    · mesh ≠ memory · memory lane is local-edge palace; mesh lane is streaming heartbeats
    · never invent stream green / Connected · empty streams honest · streams_not_probed residual
    · residual PASS ≠ live dogfood · PASS ≠ live APPLY · dual_write OFF · book-demo OFF
    · pull ≠ freemium hosted palace · package load ≠ Memory GA · rates ~$88 mesh / ~$119 Memory Ops Pack optional
    · catalog ≠ Connected · portal HITL · agent MCP cannot write installs
  Slash: /onboard next mesh (aliases stream|streams|heartbeat|heartbeats|pull) · NOT pulse (pulse stays /onboard next status board)
  Companion: /mesh · iomesh mesh status|streams|consumer · iomesh memory pull (egress only · dual_write OFF)
  Back: /onboard next · /onboard next status · /onboard next memory (separate local-edge lane)

Locks: dual_write OFF · book-demo OFF · not Memory GA · Palace sunset · residual PASS ≠ live dogfood · PASS ≠ live APPLY · mesh = streaming org heartbeats · mesh ≠ memory · not OTel/APM · not medical · never invent stream green / Connected / INSTALL_STORE APPLY · empty streams honest · streams_not_probed · pull ≠ freemium hosted palace · catalog ≠ Connected · portal HITL · agent MCP cannot write installs · rates ~$88/$119 optional · board/export evidence ≠ invent Connected`)
}

// AionAgentOnboardingNextMemoryPullLane residual-honest Memory Ops Pack pull path for
// /onboard next memory-pull (s1407). Static offline — pull = mesh → local palace egress
// (iomesh memory pull · CreateConsumer → fetch → map envelope → local MCP memory_ingest_turn → ack).
// NOT freemium hosted palace · dual_write OFF · not Memory GA · Palace sunset · Ops Pack ≠ GPU fleet.
// Ops Pack ~$119 = pull / audit / Extended retain / support — not hosted GPU palace.
// Mesh base ~$88 is separate · mesh ≠ memory · bare pull alias stays mesh lane (s1402).
// Honest residual: path_ready · residual_only · pull_not_probed (never invent pull green).
func AionAgentOnboardingNextMemoryPullLane() string {
	return strings.TrimSpace(`aion onboard next memory-pull lane (residual-honest · s1407 · no MCP dial · Ops Pack pull path):
  Path: iomesh memory pull = mesh → local palace egress — CreateConsumer → fetch → map envelope → local MCP memory_ingest_turn → ack
  Product: Memory Ops Pack ~$119 = pull / audit / Extended retain / support — NOT GPU fleet · not freemium hosted palace · Palace sunset
  Separation: mesh ≠ memory · mesh base ~$88 separate · pull ≠ freemium hosted palace · dual_write OFF · package load ≠ Ops Pack entitlement
  Steps:
    1. Residual soft: iomesh memory pull --dry-run / config [memory] pull_stream · pull_consumer · pull_filter (fail-open offline · never invent pull green)
    2. Durable consumer residual: CreateConsumer on mesh stream (requires --yes when mutating · residual soft only)
    3. Fetch → map envelope → local MCP memory_ingest_turn → ack (dual_write OFF · local-primary only)
    4. Operator pulse: /onboard next status · /onboard next export — board shows pull_not_probed until operator probes
  Honesty:
    · pull = mesh → local palace egress · dual_write OFF · not freemium hosted palace · not Memory GA
    · Ops Pack ≠ GPU fleet · package load ≠ Ops Pack entitlement · package load ≠ Memory GA
    · residual PASS ≠ live dogfood · PASS ≠ live APPLY · never invent pull green / Connected
    · pull_not_probed residual honest · board/export evidence ≠ invent Connected
    · mesh ≠ memory · rates ~$88 mesh / ~$119 Memory Ops Pack optional · book-demo OFF
    · catalog ≠ Connected · portal HITL · agent MCP cannot write installs
  Slash: /onboard next memory-pull (aliases ops-pack|pull-path|memorypull|ops_pack) · bare pull stays mesh lane (s1402)
  Companion: iomesh memory pull · /onboard next mesh (streaming heartbeats · product plane 1) · /onboard next memory (local-edge attach)
  Back: /onboard next · /onboard next status · /onboard next export

Locks: dual_write OFF · book-demo OFF · not Memory GA · Palace sunset · residual PASS ≠ live dogfood · PASS ≠ live APPLY · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · pull_not_probed · never invent pull green / Connected / INSTALL_STORE APPLY · package load ≠ Ops Pack entitlement · package load ≠ Memory GA · mesh ≠ memory · rates ~$88/$119 optional · board/export evidence ≠ invent Connected · catalog ≠ Connected · portal HITL · agent MCP cannot write installs`)
}

// AionAgentOnboardingNextLaneStatus residual-honest post-onboard lane status board for
// /onboard next status (aliases pulse|board) — free eng s1382 (+ s1387 export · s1397 session soft dogfood · s1402 mesh · s1407 memory-pull).
// Default path: no MCP dial · no invent install green / Connected / GA / APPLY / stream green / pull green.
// Honest state vocabulary only: path_ready · samples_ok · samples_missing ·
// dogfood_not_run · soft_offline_dogfood_session_pass|fail · skill_ready · residual_only ·
// streams_not_probed · pull_not_probed · portal_hitl_still.
// Optional soft check: sample package dirs via agentplugins.
// Plugins dogfood state: default dogfood_not_run; after /plugins dogfood session marker
// soft_offline_dogfood_session_pass|fail (session soft ≠ live dogfood · ≠ invent Agent Plugins GA).
// Mesh: path_ready · residual_only · streams_not_probed (never invent live green / Connected).
// Memory-pull: path_ready · residual_only · pull_not_probed (never invent pull green).
// Never claims dogfood PASS live, Agent Plugins GA, Memory GA, or install Connected.
func AionAgentOnboardingNextLaneStatus() string {
	samplesState := nextLanePluginsSamplesSoftState()
	dogfoodState := nextLanePluginsDogfoodSessionState()
	// samples_ok|samples_missing is path soft-check only ≠ residual PASS / live dogfood.
	// dogfoodState is session soft marker only (default dogfood_not_run) — ≠ live dogfood.
	return strings.TrimSpace(fmt.Sprintf(`aion onboard next lane status (residual-honest · s1382+s1397+s1402+s1407 · no MCP dial · not live dogfood):
  board (honest vocabulary only — never invent connected / ga / apply / stream green / pull green as success):

  plugins: %s · %s · path_ready
    · offline samples soft-check only (examples/agent-plugins) · ≠ invent Agent Plugins GA
    · session soft marker ≠ live dogfood · soft offline dogfood ≠ invent Agent Plugins GA
    · residual PASS ≠ live dogfood · never invent install green / Connected / INSTALL_STORE APPLY
    · board/export evidence ≠ invent Connected · drill: /onboard next plugins (aliases plugin|dogfood)

  gtm: skill_ready · path_ready · residual_only
    · /gtm checklist + skill gtm-draft-only-agent path ready
    · drafts only · no auto-send · human publish · GTM checklist ≠ invent GTM agent GA
    · drill: /onboard next gtm (alias drafts)

  memory: path_ready · residual_only
    · dual_write OFF · local-primary · package load ≠ Memory GA · ≠ freemium palace
    · not Memory GA · book-demo OFF · rates ~$88/$119 optional · mesh ≠ memory
    · drill: /onboard next memory (aliases mcp|palace)

  mesh: path_ready · residual_only · streams_not_probed
    · I/O Mesh = streaming org heartbeats on dept.* · not OTel/APM · not hosted Memory Palace
    · mesh ≠ memory · empty streams honest · never invent stream green / Connected
    · pull = mesh → local palace egress · dual_write OFF · rates ~$88 mesh / ~$119 Memory Ops Pack optional
    · drill: /onboard next mesh (aliases stream|streams|heartbeat|heartbeats|pull) · NOT pulse (pulse stays this board)

  memory-pull: path_ready · residual_only · pull_not_probed
    · Ops Pack pull path · iomesh memory pull = mesh → local palace egress · dual_write OFF
    · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · never invent pull green
    · package load ≠ Ops Pack entitlement · package load ≠ Memory GA · Palace sunset
    · rates ~$88 mesh / ~$119 Memory Ops Pack optional · residual PASS ≠ live dogfood · PASS ≠ live APPLY
    · drill: /onboard next memory-pull (aliases ops-pack|pull-path|memorypull|ops_pack) · bare pull stays mesh

  portal: portal_hitl_still
    · agent MCP cannot write installs · catalog ≠ Connected · portal HITL still for OAuth/install
    · list_org fail-open ≠ empty-as-none · never invent Connected / INSTALL_STORE APPLY
    · Agent/MCP mint/copy/probe @ https://console.iome.sh/settings/agent · connectors @ https://console.iome.sh/integrations

  slash: /onboard next status (aliases pulse|board) · /onboard next export (aliases receipt|stamp|evidence) · /onboard next mesh · /onboard next memory-pull · /onboard next · /onboard status · /integrations status · /plugins dogfood · /plugins status · /mesh
  export receipt: /onboard next export — offline markdown evidence of this board (board/export evidence ≠ invent Connected)
  plugins soft offline: /plugins dogfood (aliases soft|samples|offline) — soft offline ≠ live dogfood · ≠ invent Agent Plugins GA · session soft refreshes this board

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · never invent install green / Connected / INSTALL_STORE APPLY · catalog ≠ Connected · portal HITL · agent MCP cannot write installs · plugins dogfood ≠ invent Agent Plugins GA · session soft ≠ live dogfood · drafts only · no auto-send · package load ≠ Memory GA · rates ~$88/$119 optional · board/export evidence ≠ invent Connected · mesh = streaming org heartbeats · mesh ≠ memory · never invent stream green · streams_not_probed · not OTel/APM · pull_not_probed · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · never invent pull green`, samplesState, dogfoodState))
}

// AionAgentOnboardingNextLaneStatusExport residual-honest markdown status export receipt
// for /onboard next export (aliases receipt|stamp|evidence) — free eng s1387 (+ s1397 session soft · s1402 mesh · s1407 memory-pull).
// Offline evidence of the s1382 lane status board vocabulary; plugins dogfood lane reflects
// session soft marker from /plugins dogfood when present (default dogfood_not_run).
// Mesh lane: path_ready · residual_only · streams_not_probed (never invent stream green).
// Memory-pull lane: path_ready · residual_only · pull_not_probed (never invent pull green).
// Does NOT run plugins dogfood itself, dial MCP, or invent install green / Connected / GA / APPLY.
// Header pins evidence_kind=onboard_next_lane_status_export · offline_static ·
// not_live_dogfood · serial s1387.
// board/export evidence ≠ invent Connected · session soft ≠ live dogfood.
func AionAgentOnboardingNextLaneStatusExport() string {
	samplesState := nextLanePluginsSamplesSoftState()
	dogfoodState := nextLanePluginsDogfoodSessionState()
	return strings.TrimSpace(fmt.Sprintf(`# aion onboard next lane status export receipt

evidence_kind=onboard_next_lane_status_export
offline_static=true
not_live_dogfood=true
serial=s1387
format=markdown

## board (honest vocabulary only — s1382+s1397+s1402+s1407 reuse · never invent connected / ga / apply / stream green / pull green as success)

plugins: %s · %s · path_ready
  · offline samples soft-check only (examples/agent-plugins) · ≠ invent Agent Plugins GA
  · session soft marker ≠ live dogfood · soft offline dogfood ≠ invent Agent Plugins GA
  · residual PASS ≠ live dogfood · never invent install green / Connected / INSTALL_STORE APPLY
  · board/export evidence ≠ invent Connected · drill: /onboard next plugins (aliases plugin|dogfood)

gtm: skill_ready · path_ready · residual_only
  · /gtm checklist + skill gtm-draft-only-agent path ready
  · drafts only · no auto-send · human publish · GTM checklist ≠ invent GTM agent GA
  · drill: /onboard next gtm (alias drafts)

memory: path_ready · residual_only
  · dual_write OFF · local-primary · package load ≠ Memory GA · ≠ freemium palace
  · not Memory GA · book-demo OFF · rates ~$88/$119 optional · mesh ≠ memory
  · drill: /onboard next memory (aliases mcp|palace)

mesh: path_ready · residual_only · streams_not_probed
  · I/O Mesh = streaming org heartbeats on dept.* · not OTel/APM · not hosted Memory Palace
  · mesh ≠ memory · empty streams honest · never invent stream green / Connected
  · pull = mesh → local palace egress · dual_write OFF · rates ~$88 mesh / ~$119 Memory Ops Pack optional
  · drill: /onboard next mesh (aliases stream|streams|heartbeat|heartbeats|pull)

memory-pull: path_ready · residual_only · pull_not_probed
  · Ops Pack pull path · iomesh memory pull = mesh → local palace egress · dual_write OFF
  · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · never invent pull green
  · package load ≠ Ops Pack entitlement · package load ≠ Memory GA · Palace sunset
  · rates ~$88 mesh / ~$119 Memory Ops Pack optional · residual PASS ≠ live dogfood · PASS ≠ live APPLY
  · drill: /onboard next memory-pull (aliases ops-pack|pull-path|memorypull|ops_pack) · bare pull stays mesh

portal: portal_hitl_still
  · agent MCP cannot write installs · catalog ≠ Connected · portal HITL still for OAuth/install
  · list_org fail-open ≠ empty-as-none · never invent Connected / INSTALL_STORE APPLY
  · Agent/MCP mint/copy/probe @ https://console.iome.sh/settings/agent · connectors @ https://console.iome.sh/integrations

## honesty

- dual_write OFF · book-demo OFF · not Memory GA
- residual PASS ≠ live dogfood · never invent install green / Connected / INSTALL_STORE APPLY
- catalog ≠ Connected · portal HITL · agent MCP cannot write installs
- plugins dogfood ≠ invent Agent Plugins GA · session soft ≠ live dogfood · drafts only · no auto-send
- GTM checklist ≠ invent GTM agent GA · package load ≠ Memory GA
- board/export evidence ≠ invent Connected · rates ~$88/$119 optional
- mesh = streaming org heartbeats · mesh ≠ memory · never invent stream green · streams_not_probed · not OTel/APM
- pull_not_probed · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · never invent pull green
- this receipt does NOT run plugins dogfood · does NOT dial MCP · does NOT invent green
- session soft marker (if present) ≠ live dogfood · ≠ invent Agent Plugins GA · board ≠ invent Connected

## slash

/onboard next export (aliases receipt|stamp|evidence) · optional /onboard next export json
/onboard next status (aliases pulse|board) · /onboard next mesh · /onboard next memory-pull · /onboard next · /onboard status
/plugins dogfood (aliases soft|samples|offline) · /plugins status — soft offline ≠ live dogfood · ≠ invent Agent Plugins GA

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · never invent install green / Connected / INSTALL_STORE APPLY · catalog ≠ Connected · portal HITL · agent MCP cannot write installs · plugins dogfood ≠ invent Agent Plugins GA · session soft ≠ live dogfood · drafts only · no auto-send · package load ≠ Memory GA · board/export evidence ≠ invent Connected · rates ~$88/$119 optional · mesh = streaming org heartbeats · mesh ≠ memory · never invent stream green · streams_not_probed · pull_not_probed · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · never invent pull green`, samplesState, dogfoodState))
}

// nextLaneStatusExportDTO is the offline JSON shape for AionAgentOnboardingNextLaneStatusExportJSON (s1387+s1397).
// Honest vocabulary only — never invents Connected/GA/APPLY success.
// Plugins dogfood session state (s1397): dogfood_not_run default; soft_offline_dogfood_session_pass|fail after /plugins dogfood.
type nextLaneStatusExportDTO struct {
	EvidenceKind        string            `json:"evidence_kind"`
	OfflineStatic       bool              `json:"offline_static"`
	NotLiveDogfood      bool              `json:"not_live_dogfood"`
	Serial              string            `json:"serial"`
	Format              string            `json:"format"`
	Lanes               map[string]string `json:"lanes"`
	SamplesState        string            `json:"samples_state"`
	PluginsDogfoodState string            `json:"plugins_dogfood_state"`
	DogfoodNotRun       bool              `json:"dogfood_not_run"`
	HonestyLocks        []string          `json:"honesty_locks"`
	Slash               []string          `json:"slash"`
	Note                string            `json:"note"`
}

// AionAgentOnboardingNextLaneStatusExportJSON residual-honest JSON status export receipt
// for /onboard next export json (s1387+s1397+s1402+s1407). Same honesty as markdown; offline only.
// Reflects session soft dogfood marker when set by /plugins dogfood (≠ live dogfood).
// Mesh lane: path_ready · residual_only · streams_not_probed (never invent stream green).
// Memory-pull / ops_pack lane: path_ready · residual_only · pull_not_probed (never invent pull green).
// Does NOT run plugins dogfood itself, dial MCP, or invent install green / Connected / GA / APPLY.
func AionAgentOnboardingNextLaneStatusExportJSON() string {
	samplesState := nextLanePluginsSamplesSoftState()
	dogfoodState := nextLanePluginsDogfoodSessionState()
	ran, _ := agentplugins.GetSoftDogfoodSessionState()
	dto := nextLaneStatusExportDTO{
		EvidenceKind:   "onboard_next_lane_status_export",
		OfflineStatic:  true,
		NotLiveDogfood: true,
		Serial:         "s1387",
		Format:         "json",
		Lanes: map[string]string{
			"plugins":     fmt.Sprintf("%s · %s · path_ready", samplesState, dogfoodState),
			"gtm":         "skill_ready · path_ready · residual_only",
			"memory":      "path_ready · residual_only",
			"mesh":        "path_ready · residual_only · streams_not_probed",
			"memory-pull": "path_ready · residual_only · pull_not_probed",
			"ops_pack":    "path_ready · residual_only · pull_not_probed",
			"portal":      "portal_hitl_still",
		},
		SamplesState:        samplesState,
		PluginsDogfoodState: dogfoodState,
		// dogfood_not_run true only when session soft dogfood has not run (s1397).
		// When ran, still not_live_dogfood=true — session soft ≠ live dogfood.
		DogfoodNotRun: !ran,
		HonestyLocks: []string{
			"dual_write OFF",
			"book-demo OFF",
			"not Memory GA",
			"residual PASS ≠ live dogfood",
			"session soft ≠ live dogfood",
			"never invent install green / Connected / INSTALL_STORE APPLY",
			"catalog ≠ Connected",
			"portal HITL",
			"agent MCP cannot write installs",
			"plugins dogfood ≠ invent Agent Plugins GA",
			"soft offline dogfood ≠ invent Agent Plugins GA",
			"drafts only",
			"no auto-send",
			"GTM checklist ≠ invent GTM agent GA",
			"package load ≠ Memory GA",
			"board/export evidence ≠ invent Connected",
			"rates ~$88/$119 optional",
			"mesh = streaming org heartbeats",
			"mesh ≠ memory",
			"never invent stream green / Connected",
			"streams_not_probed",
			"not OTel/APM",
			"pull_not_probed",
			"pull ≠ freemium hosted palace",
			"Ops Pack ≠ GPU fleet",
			"never invent pull green",
			"package load ≠ Ops Pack entitlement",
		},
		Slash: []string{
			"/onboard next export",
			"/onboard next export json",
			"/onboard next status",
			"/onboard next mesh",
			"/onboard next memory-pull",
			"/plugins dogfood",
			"/plugins status",
		},
		Note: "offline residual-honest lane board evidence; plugins_dogfood_state is session soft marker only (default dogfood_not_run); mesh lane is path_ready · residual_only · streams_not_probed (never invent stream green); memory-pull/ops_pack lane is path_ready · residual_only · pull_not_probed (never invent pull green); session soft ≠ live dogfood; board/export evidence ≠ invent Connected; does NOT run plugins dogfood or dial MCP; soft offline dogfood ≠ invent Agent Plugins GA; mesh ≠ memory; pull ≠ freemium hosted palace; Ops Pack ≠ GPU fleet",
	}
	b, err := json.MarshalIndent(dto, "", "  ")
	if err != nil {
		// Fail-open residual: never invent green; return a static honesty stub.
		return `{"evidence_kind":"onboard_next_lane_status_export","offline_static":true,"not_live_dogfood":true,"serial":"s1387","error":"marshal_failed","note":"board/export evidence ≠ invent Connected"}`
	}
	return string(b)
}

// nextLanePluginsSamplesSoftState soft-checks in-repo sample package dirs (s1382/s1387/s1392).
// samples_ok when both hello-iome + aion-memory-mcp dirs exist under module root;
// samples_missing otherwise (including when module root is not found).
// Does not run dogfood, Dial MCP, or invent Agent Plugins GA / Connected.
func nextLanePluginsSamplesSoftState() string {
	return agentplugins.SamplesSoftState("")
}

// nextLanePluginsDogfoodSessionState returns session soft dogfood vocabulary (s1397).
// Default dogfood_not_run; after /plugins dogfood: soft_offline_dogfood_session_pass|fail.
// Session soft ≠ live dogfood · ≠ invent Agent Plugins GA · board ≠ invent Connected.
func nextLanePluginsDogfoodSessionState() string {
	return agentplugins.SoftDogfoodSessionLabel()
}
