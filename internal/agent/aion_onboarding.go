package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iome-sh/iomesh-tui/internal/agentplugins"
)

// AionAgentOnboardingGuidanceNote residual-honest system note (s1363 + s1368 + s1372 + s1377 + s1382 + s1387 + s1402 + s1407 + s1413 + s1417 + s1432 + s1437).
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
// s1413: /onboard next human-gates residual-honest still-required vs offline APPLY honesty.
// s1417: /onboard next agentic product plane 3 agentic integrations (MCP list/plan + portal HITL).
// s1432: /onboard next planes residual-honest three product planes board (mesh · memory-pull · agentic).
// s1437: /onboard next sales residual-honest sales/buyer claims board (may claim / must not claim).
// Unit-tested for honesty needles. Molds IntegrationsAgentGuidanceNote /
// GtmDraftOnlyAgentGuidanceNote / MemoryAdvancedAgentGuidanceNote.
func AionAgentOnboardingGuidanceNote() string {
	return strings.TrimSpace(`aion agent onboarding (residual-honest TUI ↔ aion CP/MCP · s1363+s1368+s1372+s1377+s1382+s1387+s1402+s1407+s1413+s1417+s1432+s1437):
Point IOMESH/MCP at aion tools — fail-open offline (never invent tool green).

Connector path (integrations portal HITL · product plane 3 agentic integrations):
1. Discover: MCP list_connector_catalog — catalog status ≠ install Connected · catalog ≠ Connected
2. Plan: MCP plan_connector_setup — portal deep links + honesty notes (browser HITL only · template= ≠ install APPLY green)
3. Org installs residual: MCP list_org_connector_installs — fail-open (available=false · installs=null) ≠ empty-as-none · never invent Connected
4. Complete OAuth/install in portal HITL at https://console.iome.sh/integrations — agent MCP cannot write installs

Portal Agent/MCP lane (complementary · s1368 · credential → copy connection → test invoke):
- Portal: mint key / Settings → Agent/MCP → copy MCP connection → test invoke (probe only ≠ Memory GA)
- TUI: configure [[mcp.servers]] streamable HTTP → /onboard · /integrations status
- Console Agent/MCP: https://console.iome.sh/settings/agent (connectors still /integrations)

Memory + operator:
5. Memory: dual_write OFF · local-primary · not Memory GA · optional plugins dogfood ≠ invent Agent Plugins GA (rates ~$88/$119 optional)
6. Operator pulse: /integrations status · /onboard checklist · /onboard portal · portal HITL
7. Post-onboard continuum: /onboard next [plugins|gtm|memory|mesh|memory-pull|agentic|planes|sales|status|export|human-gates] (plugins dogfood · /gtm checklist · aion-memory-mcp local · mesh streaming heartbeats · Ops Pack pull path · agentic integrations MCP list/plan + portal HITL · three product planes board · sales/buyer claims · lane status board · status export receipt · human-gates still-required vs offline)
8. Human gates (s1413): /onboard next human-gates — still human Slack HMAC · Stripe Customers:Write · H1/H2 INSTALL_STORE · D1–D5 · book-demo OFF · ON_SIGNAL unset · offline residual ≠ invent APPLY
9. Agentic integrations (s1417 · product plane 3): /onboard next agentic — MCP list/plan residual-honest · plan_connector_setup → portal deep links · browser HITL only · list_org fail-open ≠ empty-as-none · never invent Connected
10. Three product planes (s1432): /onboard next planes — mesh · memory-pull · agentic residual-honest consolidate · streams_not_probed · pull_not_probed · list_plan_not_connected · dual_auth_candidacy_open · never invent Connected
11. Sales/buyer claims (s1437): /onboard next sales — may claim / must not claim residual-honest · three-planes grounded · never invent Connected / Memory GA / dual-auth live

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
- pull = mesh → local palace egress · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · pull_not_probed honest
- PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open · leave ON_SIGNAL unset
- Knowledge Beta→GA cannot invent H1/H2 offline · local memory / dual_write OFF / agent MCP list/plan do not close human APPLY gates
- agentic: MCP list/plan residual-honest · plan deep links = browser HITL only · template= ≠ install APPLY · list_plan_not_connected · never invent Connected
- planes: mesh · memory-pull · agentic consolidate · never invent stream green / pull green / Connected · dual_auth_candidacy_open
- sales claims: may claim residual-honest only · must not invent Connected / Memory GA / dual-auth live / human-gate green`)
}

// AionAgentOnboardingChecklist residual-honest numbered onboarding checklist (s1363 + s1368 + s1372 + s1377 + s1382 + s1387 + s1402 + s1407 + s1413 + s1417 + s1432 + s1437).
// Used by /onboard help and /onboard checklist — operator HITL only; never invents
// install green, Memory GA, Agent Plugins GA, dual_write ON, or agent APPLY.
// s1368: portal Agent/MCP handoff steps (mint/copy/probe) + TUI [[mcp.servers]].
// s1372: cross-link → /onboard next operator lanes (post-onboard continuum).
// s1377: /onboard next [plugins|gtm|memory] lane drills.
// s1382: /onboard next status lane status board.
// s1387: /onboard next export status export receipt.
// s1402: /onboard next mesh streaming lane (org heartbeats).
// s1407: /onboard next memory-pull Ops Pack pull path.
// s1413: /onboard next human-gates residual-honest still-required vs offline.
// s1417: /onboard next agentic product plane 3 agentic integrations (MCP list/plan + portal HITL).
// s1432: /onboard next planes residual-honest three product planes board (mesh · memory-pull · agentic).
// s1437: /onboard next sales residual-honest sales/buyer claims board (may / must-not).
func AionAgentOnboardingChecklist() string {
	return strings.TrimSpace(`aion agent onboarding checklist (residual-honest · s1363+s1368+s1372+s1377+s1382+s1387+s1402+s1407+s1413+s1417+s1432+s1437 · TUI ↔ aion):
  1. Point IOMESH/MCP at aion tools (fail-open offline)
  2. list_connector_catalog — catalog status ≠ Connected
  3. plan_connector_setup → portal deep links (browser HITL · template= ≠ install APPLY)
  4. list_org_connector_installs residual fail-open (available=false ≠ empty-as-none)
  5. Portal Agent/MCP: mint key → Settings → Agent/MCP → copy MCP connection → test invoke (probe only ≠ Memory GA) at https://console.iome.sh/settings/agent
  6. TUI: [[mcp.servers]] streamable HTTP → /onboard · /integrations status (agent MCP cannot write installs)
  7. Memory dual_write OFF · local-primary · not Memory GA · optional plugins dogfood ≠ Agent Plugins GA
  8. Operator: /integrations status · /onboard checklist · /onboard portal · portal https://console.iome.sh/integrations
  9. Post-onboard: /onboard next [plugins|gtm|memory|mesh|memory-pull|agentic|planes|sales|status|export|human-gates] (plugins · gtm · memory local · mesh streaming heartbeats · Ops Pack pull path · agentic integrations MCP list/plan + portal HITL · three product planes board · sales/buyer claims · lane status board · status export receipt · human-gates still-required vs offline)
  10. Human gates: /onboard next human-gates — still human Slack HMAC · Stripe Customers:Write · H1/H2 INSTALL_STORE · D1–D5 · book-demo OFF · ON_SIGNAL unset · PASS ≠ invent human-gate green · never invent APPLY
  11. Agentic integrations (product plane 3): /onboard next agentic — MCP list/plan residual-honest · plan_connector_setup → portal deep links · browser HITL only · list_org fail-open ≠ empty-as-none · catalog ≠ Connected · never invent Connected
  12. Three product planes (s1432): /onboard next planes — mesh · memory-pull · agentic residual-honest consolidate · streams_not_probed · pull_not_probed · list_plan_not_connected · dual_auth_candidacy_open · never invent Connected
  13. Sales/buyer claims (s1437): /onboard next sales — may claim / must not claim residual-honest · three-planes grounded · never invent Connected / Memory GA / dual-auth live
  Locks: never invent install green / Connected / INSTALL_STORE APPLY · book-demo OFF · residual PASS ≠ live dogfood · PASS ≠ live APPLY · rates ~$88/$119 optional · no invent GA knowledge/analytical · catalog status ≠ Connected · portal HITL · drafts only · no auto-send · package load ≠ Memory GA · board/export evidence ≠ invent Connected · mesh = streaming org heartbeats · mesh ≠ memory · never invent stream green · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · pull_not_probed · open boxes stay open · Knowledge Beta→GA cannot invent H1/H2 offline · leave ON_SIGNAL unset · list_plan_not_connected · plan deep links = browser HITL only · template= ≠ install APPLY · dual_auth_candidacy_open · sales claims residual-honest only`)
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

// AionAgentOnboardingStatus residual-honest static offline status lines for /onboard status (s1368 + s1372 + s1377 + s1382 + s1387 + s1402 + s1407 + s1413 + s1417 + s1432 + s1437).
// No MCP dial — operator pulse only. Never invents attach green, install Connected, or Memory GA.
// s1372: cross-link → /onboard next operator lanes.
// s1377: lane drills via /onboard next [plugins|gtm|memory].
// s1382: cross-link → /onboard next status lane status board.
// s1387: cross-link → /onboard next export status export receipt.
// s1402: cross-link → /onboard next mesh streaming lane.
// s1407: cross-link → /onboard next memory-pull Ops Pack pull path.
// s1413: cross-link → /onboard next human-gates residual-honest still-required vs offline.
// s1417: cross-link → /onboard next agentic product plane 3 agentic integrations.
// s1432: cross-link → /onboard next planes three product planes board.
// s1437: cross-link → /onboard next sales sales/buyer claims board.
func AionAgentOnboardingStatus() string {
	return strings.TrimSpace(`aion onboard status (residual-honest · offline static · s1368+s1372+s1377+s1382+s1387+s1402+s1407+s1413+s1417+s1432+s1437):
  MCP attach: expected for full path · fail-open offline (never invent tool green / install green)
  dual_write OFF · local-primary · not Memory GA · book-demo OFF · leave ON_SIGNAL unset
  portal HITL: Agent/MCP mint/copy/probe @ https://console.iome.sh/settings/agent · connectors @ https://console.iome.sh/integrations
  never invent install green / Connected / INSTALL_STORE APPLY · PASS ≠ invent human-gate green · PASS ≠ live APPLY
  list_org fail-open (available=false) ≠ empty-as-none · catalog ≠ Connected · list_plan_not_connected
  agent MCP cannot write installs · plugins dogfood ≠ invent Agent Plugins GA
  residual PASS ≠ live dogfood · test invoke = probe only ≠ Memory GA
  mesh = streaming org heartbeats · mesh ≠ memory · never invent stream green
  pull = mesh → local palace egress · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · pull_not_probed
  agentic: product plane 3 · MCP list/plan residual-honest · plan deep links = browser HITL only · template= ≠ install APPLY · portal_hitl_still
  three planes: /onboard next planes — mesh · memory-pull · agentic residual-honest consolidate · streams_not_probed · pull_not_probed · list_plan_not_connected · dual_auth_candidacy_open
  sales claims: /onboard next sales — may claim / must not claim residual-honest · three-planes grounded · never invent Connected / Memory GA
  human-gates: still human Slack HMAC · Stripe Customers:Write · H1/H2 INSTALL_STORE · D1–D5 · open boxes stay open · Knowledge Beta→GA cannot invent H1/H2 offline
  slash: /onboard portal · /onboard checklist · /onboard next [plugins|gtm|memory|mesh|memory-pull|agentic|planes|sales|status|export|human-gates] · /onboard next status · /onboard next export · /onboard next mesh · /onboard next memory-pull · /onboard next agentic · /onboard next planes · /onboard next sales · /onboard next human-gates · /integrations status`)
}

// AionAgentOnboardingNextLanes residual-honest post-onboard continuum for /onboard next (s1372 + s1377 + s1382 + s1387 + s1402 + s1407 + s1413 + s1417 + s1432 + s1437).
// Static offline block — no MCP dial. Lists residual-honest operator lanes after
// core onboarding (plugins dogfood · GTM drafts · local memory · mesh streaming · Ops Pack pull · agentic integrations · portal HITL still · human gates · sales claims).
// s1377: drill-down via /onboard next plugins|gtm|memory (see lane helpers below).
// s1382: lane status board via /onboard next status (aliases pulse|board).
// s1387: status export receipt via /onboard next export (aliases receipt|stamp|evidence).
// s1402: mesh streaming lane via /onboard next mesh (aliases stream|streams|heartbeat|heartbeats|pull).
// s1407: memory-pull Ops Pack pull path via /onboard next memory-pull (aliases ops-pack|pull-path|memorypull|ops_pack).
// s1413: human-gates honesty board via /onboard next human-gates (aliases human|gates|apply-gates).
// s1417: agentic integrations product plane 3 via /onboard next agentic (aliases agentic-integrations|integrations|portal-hitl|list-plan|hitl).
// s1432: three product planes board via /onboard next planes (aliases three-planes|product-planes|product|pillars|three_planes).
// s1437: sales/buyer claims board via /onboard next sales (aliases claims|buyer|claim-matrix|sales-claims|buyer-claims).
// Never invents Agent Plugins GA, Memory GA, auto-send, install Connected, stream green, pull green, or human-gate green.
func AionAgentOnboardingNextLanes() string {
	return strings.TrimSpace(`aion onboard next lanes (residual-honest · post-onboard continuum · s1372+s1377+s1382+s1387+s1402+s1407+s1413+s1417+s1432+s1437 · no MCP dial):
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
  6. agentic integrations (product plane 3) — MCP list/plan residual-honest · plan_connector_setup → portal deep links · browser HITL only · catalog ≠ Connected · agent MCP cannot write installs
     drill: /onboard next agentic (aliases agentic-integrations|integrations|portal-hitl|list-plan|hitl) · NOT bare mcp (memory lane) · NOT bare portal (portal handoff)
  7. portal HITL still required for OAuth/install · agent MCP cannot write installs · catalog ≠ Connected · list_plan_not_connected
  8. human-gates still-required vs offline residual — Slack HMAC · Stripe Customers:Write · H1/H2 INSTALL_STORE · D1–D5 · book-demo OFF · ON_SIGNAL unset
     drill: /onboard next human-gates (aliases human|gates|apply-gates) · PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open
  three product planes: /onboard next planes (aliases three-planes|product-planes|product|pillars|three_planes) — mesh · memory-pull · agentic residual-honest consolidate · streams_not_probed · pull_not_probed · list_plan_not_connected · dual_auth_candidacy_open · never invent Connected (s1432)
  sales/buyer claims: /onboard next sales (aliases claims|buyer|claim-matrix|sales-claims|buyer-claims) — may claim / must not claim residual-honest · three-planes grounded · never invent Connected / Memory GA / dual-auth live (s1437) · NOT product/planes (those stay three-planes) · NOT gtm (drafts) · NOT pulse/board (status)
  status board: /onboard next status (aliases pulse|board) — residual-honest lane states only (never invent connected/ga/apply as success · pulse stays board)
  export receipt: /onboard next export (aliases receipt|stamp|evidence) — offline markdown evidence of board (board/export evidence ≠ invent Connected)
  human-gates board: /onboard next human-gates — still human vs offline residual only vs shipped/policy (local memory / dual_write OFF / agent MCP list/plan do not close human APPLY gates)
  agentic lane: /onboard next agentic — product plane 3 MCP list/plan + portal HITL · list_plan_not_connected · never invent Connected / install green

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY · PASS ≠ invent human-gate green · open boxes stay open · leave ON_SIGNAL unset · Knowledge Beta→GA cannot invent H1/H2 offline · never invent install green / Connected / INSTALL_STORE APPLY · list_org fail-open ≠ empty-as-none · plugins dogfood ≠ invent Agent Plugins GA · drafts only · no auto-send · rates ~$88/$119 optional · package load ≠ Memory GA · board/export evidence ≠ invent Connected · mesh = streaming org heartbeats · mesh ≠ memory · never invent stream green / Connected · not OTel/APM · streams_not_probed honest · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · pull_not_probed honest · never invent pull green · list_plan_not_connected · dual_auth_candidacy_open · plan deep links = browser HITL only · template= ≠ install APPLY · sales claims residual-honest only`)
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

// AionAgentOnboardingNextAgenticLane residual-honest product plane 3 agentic integrations
// for /onboard next agentic (s1417 + s1422 soft dogfood + portal HITL polish + s1427 dual-auth candidacy tip).
// Static offline — no MCP dial on this board.
// MCP list (catalog / list connectors residual-honest · catalog ≠ Connected) ·
// MCP plan (plan_connector_setup → proven portal deep links only · browser HITL) ·
// portal HITL for OAuth/install (agent MCP cannot write installs).
// Proven deep-link paths: /integrations/{id} · /integrations/add?template={id} · /integrations.
// template= ≠ install APPLY green · deep_links = browser HITL only.
// list_org fail-open ≠ empty-as-none · available=false default residual · never invent Connected.
// Complements (does not replace) /onboard portal mint/copy/probe and human-gates lane.
// s1422: soft offline list/plan dogfood via /onboard next agentic dogfood (session soft ≠ live dogfood).
// s1422: Portal HITL polish block (proven paths · mint/copy/probe complementary · OAuth/install still portal).
// s1427: dual-auth candidacy depth via /onboard next agentic dual-auth (tool ship ≠ dual-auth live).
// Honest residual: path_ready · residual_only · portal_hitl_still · list_plan_not_connected · list_plan_soft_not_run default.
// Aliases: agentic-integrations|integrations|portal-hitl|list-plan|hitl.
// DO NOT steal: bare mcp (memory lane) · bare portal/agent-mcp (portal handoff) · bare pull (mesh).
func AionAgentOnboardingNextAgenticLane() string {
	softLabel := AgenticListPlanSoftSessionLabel()
	return strings.TrimSpace(fmt.Sprintf(`aion onboard next agentic lane (residual-honest · s1417+s1422+s1427 · no MCP dial · product plane 3 · agentic integrations):
  Path: MCP list/plan residual-honest + portal HITL for OAuth/install — agent MCP cannot write installs
  Product plane 3: agentic integrations continuum (post-onboard next lane · complements /onboard portal mint/copy/probe · does not replace human-gates)
  Steps:
    1. MCP list — list_connector_catalog / list connectors residual-honest · catalog ≠ Connected · catalog status ≠ install Connected
    2. MCP plan — plan_connector_setup → proven portal deep links only · browser HITL · template= ≠ install APPLY green
    3. Proven deep_links (browser HITL only): /integrations/{id} · /integrations/add?template={id} · /integrations
    4. Org residual: list_org_connector_installs fail-open · available=false default residual · installs=null · list_org fail-open ≠ empty-as-none · never invent Connected
    5. Portal HITL OAuth/install @ https://console.iome.sh/integrations — agent MCP cannot write installs · deep_links = browser HITL only
    6. Agent/MCP mint/copy/probe companion @ https://console.iome.sh/settings/agent (complementary · not this lane's primary · /onboard portal)
    7. Dual-auth candidacy depth (s1427): /onboard next agentic dual-auth — list_org fail-open · dual_auth_candidacy_open · tool ship ≠ dual-auth live
  Portal HITL polish (s1422 · proven paths only · browser HITL):
    · Proven deep-link paths: /integrations/{id} · /integrations/add?template={id} · /integrations
    · template= ≠ install APPLY · deep_links = browser HITL only (plan deep links never invent install green)
    · Complementary: /onboard portal mint/copy/probe @ https://console.iome.sh/settings/agent (probe only ≠ Memory GA)
    · OAuth/install still portal HITL @ https://console.iome.sh/integrations · agent MCP cannot write installs
    · dual_write OFF · catalog ≠ Connected · list_org fail-open ≠ empty-as-none · never invent Connected
  Soft offline list/plan dogfood (s1422 · session soft ≠ live dogfood):
    · session soft state: %s (default list_plan_soft_not_run · after run soft_offline_list_plan_session_pass|fail)
    · slash: /onboard next agentic dogfood (aliases soft|samples|offline|list-plan-soft) · bare /onboard next agentic stays this board
    · soft offline list/plan ≠ live dogfood · ≠ invent Connected · portal HITL still · list_org fail-open ≠ empty-as-none
    · tip: after dogfood re-run /onboard next status then /onboard next export so agentic lane reflects session soft
  Dual-auth candidacy depth (s1427 · residual-only · tool ship ≠ dual-auth live):
    · dual_auth_candidacy_open · list_org_unavailable · never invent dual-auth live
    · slash: /onboard next agentic dual-auth (aliases candidacy|list-org|org-installs|dual_auth|dual-auth-candidacy)
    · NOT dogfood|soft|samples|offline|list-plan-soft (those stay soft dogfood s1422)
  Honesty:
    · product plane 3 · agentic integrations · MCP list/plan residual-honest
    · plan_connector_setup · portal deep links · browser HITL only
    · template= ≠ install APPLY green · deep_links = browser HITL only
    · list_org fail-open ≠ empty-as-none · available=false default residual
    · agent MCP cannot write installs · catalog ≠ Connected · never invent Connected / install green
    · portal HITL @ https://console.iome.sh/integrations · Agent/MCP mint/copy/probe @ https://console.iome.sh/settings/agent
    · dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY
    · open boxes stay open · rates ~$88/$119 optional · board/export evidence ≠ invent Connected
    · does not claim dual-auth live for list_org · residual soft path only · tool ship ≠ dual-auth live
    · path_ready · residual_only · portal_hitl_still · list_plan_not_connected · dual_auth_candidacy_open · %s (honest vocab)
    · soft offline ≠ live dogfood · session soft ≠ live dogfood
  Slash: /onboard next agentic (aliases agentic-integrations|integrations|portal-hitl|list-plan|hitl)
    · dogfood: /onboard next agentic dogfood (aliases soft|samples|offline|list-plan-soft)
    · dual-auth: /onboard next agentic dual-auth (aliases candidacy|list-org|org-installs|dual_auth|dual-auth-candidacy)
    · NOT bare mcp (mcp stays memory lane under /onboard next) · NOT bare portal|agent-mcp (portal handoff) · NOT bare pull (mesh)
  Companion: /onboard portal (mint/copy/probe) · /integrations list|plan|status · /onboard next human-gates · /onboard next status · /onboard next export
  Back: /onboard next · /onboard next status · /onboard status

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open · never invent install green / Connected / INSTALL_STORE APPLY · catalog ≠ Connected · list_org fail-open ≠ empty-as-none · plan deep links = browser HITL only · template= ≠ install APPLY · agent MCP cannot write installs · portal HITL · list_plan_not_connected · portal_hitl_still · path_ready · residual_only · soft offline ≠ live dogfood · session soft ≠ live dogfood · rates ~$88/$119 optional · board/export evidence ≠ invent Connected · does not claim dual-auth live for list_org · tool ship ≠ dual-auth live · dual_auth_candidacy_open`, softLabel, softLabel))
}

// AionAgentOnboardingNextAgenticDualAuthCandidacy residual-honest dual-auth candidacy depth
// for /onboard next agentic dual-auth (s1427 · product plane 3). Static offline — no MCP dial.
// Surfaces list_org_connector_installs residual fail-open shape + dual_auth_candidacy_open honesty.
// tool ship ≠ dual-auth live · PASS ≠ invent dual-auth shipped · never invent empty-as-none
// (installs=null not []) · portal session owns install index · agent MCP cannot write installs.
// Honest vocab: path_ready · residual_only · dual_auth_candidacy_open · list_org_unavailable.
// Aliases (4th token): dual-auth|candidacy|list-org|org-installs|dual_auth|dual-auth-candidacy.
// DO NOT steal: dogfood|soft|samples|offline|list-plan-soft stay soft dogfood (s1422).
// Bare /onboard next agentic stays main agentic board.
func AionAgentOnboardingNextAgenticDualAuthCandidacy() string {
	return strings.TrimSpace(`aion onboard next agentic dual-auth candidacy (residual-honest · s1427 · no MCP dial · product plane 3 · dual_auth_candidacy_open):
  Path: list_org_connector_installs residual fail-open + dual-auth candidacy honesty — tool ship ≠ dual-auth live
  Product plane 3 depth: agentic org installs snapshot residual (post agentic lane s1417 · after soft dogfood s1422)
  Snapshot residual (fail-open shape · never invent live dual-auth):
    · MCP tool: list_org_connector_installs
    · available=false · status=unavailable · installs=null
    · never invent empty-as-none — installs=null not [] · available=false ≠ "none connected"
    · dual_auth_candidacy_open · list_org_unavailable · residual soft path only
  Ownership:
    · portal session owns install index · session-cookie + org membership only
    · agent MCP cannot write installs · catalog ≠ Connected · never invent Connected
    · portal HITL @ https://console.iome.sh/integrations for OAuth/install CRUD
  Honesty:
    · dual_auth_candidacy_open · never invent dual-auth live
    · tool ship ≠ dual-auth live · PASS ≠ invent dual-auth shipped
    · list_org_connector_installs · available=false · status=unavailable · installs=null
    · never invent empty-as-none (installs=null not [])
    · agent MCP cannot write installs · catalog ≠ Connected · never invent Connected
    · portal session owns install index · session-cookie + org membership only
    · portal HITL @ https://console.iome.sh/integrations
    · dual_write OFF · book-demo OFF · not Memory GA
    · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open
    · rates ~$88/$119 optional · board/export evidence ≠ invent Connected
    · path_ready · residual_only · dual_auth_candidacy_open · list_org_unavailable (honest vocab)
  Slash: /onboard next agentic dual-auth (aliases candidacy|list-org|org-installs|dual_auth|dual-auth-candidacy)
    · NOT dogfood|soft|samples|offline|list-plan-soft (those stay soft dogfood s1422)
    · bare /onboard next agentic stays main agentic board
  Companion: /onboard next agentic · /onboard next agentic dogfood · /onboard portal · /onboard next status
  Back: /onboard next agentic · /onboard next status · /onboard status

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open · never invent dual-auth live · tool ship ≠ dual-auth live · PASS ≠ invent dual-auth shipped · never invent empty-as-none · installs=null not [] · available=false · status=unavailable · dual_auth_candidacy_open · list_org_unavailable · agent MCP cannot write installs · catalog ≠ Connected · never invent Connected · portal session owns install index · session-cookie + org membership only · portal HITL · rates ~$88/$119 optional · board/export evidence ≠ invent Connected · path_ready · residual_only`)
}

// AionAgentOnboardingNextThreePlanes residual-honest consolidated three product planes board
// for /onboard next planes (s1432). Static offline — no MCP dial · no invent stream green /
// pull green / Connected / GA / APPLY. Surfaces product narrative planes:
//
//	1 Mesh (streaming org heartbeats on dept.*)
//	2 Memory-pull / Ops Pack (mesh → local palace egress)
//	3 Agentic integrations (MCP list/plan · portal HITL · dual-auth candidacy open)
//
// Honest vocab: path_ready · residual_only · streams_not_probed · pull_not_probed ·
// portal_hitl_still · list_plan_not_connected · dual_auth_candidacy_open.
// Aliases: three-planes|product-planes|product|pillars|three_planes.
// Do NOT steal: pulse|board (status) · pull (mesh) · mcp (memory).
// Cross-links: /onboard next mesh · memory-pull · agentic · agentic dual-auth · agentic dogfood ·
// status · export · human-gates.
func AionAgentOnboardingNextThreePlanes() string {
	return strings.TrimSpace(`aion onboard next three product planes (residual-honest · s1432 · no MCP dial · offline static · not live dogfood · not live APPLY):
  board (honest vocabulary only — never invent stream green / pull green / Connected / GA / APPLY as success):

  plane 1 · Mesh (I/O Mesh streaming · product plane 1):
    · path_ready · residual_only · streams_not_probed
    · streaming org heartbeats on dept.* · mesh ≠ memory · not OTel/APM · not medical · Palace sunset
    · empty streams honest · never invent stream green / Connected
    · rates ~$88 mesh optional (commercial framing only · not product GA claim)
    · drill: /onboard next mesh (aliases stream|streams|heartbeat|heartbeats|pull) · NOT pulse (pulse stays status board)

  plane 2 · Memory-pull / Ops Pack (customer-edge memory egress · product plane 2):
    · path_ready · residual_only · pull_not_probed
    · mesh → local palace egress · iomesh memory pull · dual_write OFF · not Memory GA
    · Ops Pack ≠ GPU fleet · pull ≠ freemium hosted palace · package load ≠ Ops Pack entitlement · package load ≠ Memory GA
    · never invent pull green / Connected · rates ~$119 Memory Ops Pack optional (commercial framing only)
    · drill: /onboard next memory-pull (aliases ops-pack|pull-path|memorypull|ops_pack) · bare pull stays mesh

  plane 3 · Agentic integrations (MCP list/plan + portal HITL · product plane 3):
    · path_ready · residual_only · portal_hitl_still · list_plan_not_connected · dual_auth_candidacy_open
    · MCP list/plan residual-honest · plan_connector_setup → portal deep links · browser HITL only
    · template= ≠ install APPLY · catalog ≠ Connected · list_org fail-open ≠ empty-as-none
    · agent MCP cannot write installs · never invent Connected · tool ship ≠ dual-auth live
    · dual_auth_candidacy_open · list_org_unavailable · never invent dual-auth live
    · drill: /onboard next agentic (aliases agentic-integrations|integrations|portal-hitl|list-plan|hitl)
    · soft dogfood: /onboard next agentic dogfood (aliases soft|samples|offline|list-plan-soft) · soft offline ≠ live dogfood
    · dual-auth depth: /onboard next agentic dual-auth (aliases candidacy|list-org|org-installs|dual_auth|dual-auth-candidacy)
    · NOT bare mcp (memory) · NOT bare portal (portal handoff) · NOT bare pull (mesh)

  cross-plane honesty (all planes):
    · dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY
    · open boxes stay open · human-gates still open (tip only) · never invent GA
    · rates ~$88 mesh / ~$119 Memory Ops Pack optional · board/export evidence ≠ invent Connected
    · mesh ≠ memory · never invent stream green / pull green / Connected
    · human-gates tip: /onboard next human-gates (aliases human|gates|apply-gates) · PASS ≠ invent human-gate green

  slash: /onboard next planes (aliases three-planes|product-planes|product|pillars|three_planes)
    · NOT pulse|board (those stay status board) · NOT bare pull (mesh) · NOT bare mcp (memory)
  companion drills: /onboard next mesh · /onboard next memory-pull · /onboard next agentic · /onboard next agentic dual-auth · /onboard next agentic dogfood
  companion boards: /onboard next status (aliases pulse|board) · /onboard next export · /onboard next human-gates · /onboard next sales · /onboard next · /onboard status
  sales claims tip: /onboard next sales (aliases claims|buyer|claim-matrix|sales-claims|buyer-claims) — may claim / must not claim residual-honest (s1437)

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open · mesh ≠ memory · never invent stream green / pull green / Connected / INSTALL_STORE APPLY · streams_not_probed · pull_not_probed · Ops Pack ≠ GPU fleet · pull ≠ freemium hosted palace · agent MCP cannot write installs · list_plan_not_connected · portal_hitl_still · dual_auth_candidacy_open · tool ship ≠ dual-auth live · never invent dual-auth live · catalog ≠ Connected · plan deep links = browser HITL only · template= ≠ install APPLY · rates ~$88/$119 optional · no invent GA · board/export evidence ≠ invent Connected · path_ready · residual_only · leave ON_SIGNAL unset · Knowledge Beta→GA cannot invent H1/H2 offline`)
}

// AionAgentOnboardingNextSalesClaims residual-honest sales / buyer claims board for
// /onboard next sales (s1437). Static offline — no MCP dial · no invent Connected /
// Memory GA / dual-auth live / human-gate green / freemium palace / Ops Pack GPU.
// Grounded in three product planes (mesh · memory-pull · agentic) via companion
// /onboard next planes. Surfaces may-claim vs must-not-claim for founders/sales ops.
//
// May claim (residual-honest): mesh heartbeats · rates ~$88/~$119 local-primary ·
// dual_write OFF · Palace sunset · Salesforce GA CRM · HubSpot+GTM Beta multi-tenant ·
// knowledge/analytical Beta · agentic list/plan residual · portal HITL · catalog ≠ Connected.
// Must not claim: invent Connected / INSTALL_STORE green · Memory GA · dual_write ON ·
// freemium palace · Ops Pack = GPU · book-demo ON · dual-auth live · empty-as-none ·
// agent MCP write installs · Knowledge/Analytics GA · human-gate green.
//
// Aliases: claims|buyer|claim-matrix|sales-claims|buyer-claims.
// Do NOT steal: product|planes (three-planes) · gtm|drafts (GTM lane) · pulse|board (status).
// Cross-links: /onboard next planes · mesh · memory-pull · agentic · human-gates · status · export.
func AionAgentOnboardingNextSalesClaims() string {
	return strings.TrimSpace(`aion onboard next sales / buyer claims (residual-honest · s1437 · no MCP dial · offline static · not live dogfood · not live APPLY · three-planes grounded):
  board for founders/sales operators — may claim vs must not claim only (never invent Connected / GA / APPLY as success):

  MAY CLAIM (residual-honest · commercial framing only):
    · I/O Mesh = streaming org heartbeats / pulse on dept.* — not OTel/APM replacement · not medical · mesh ≠ memory
    · Mesh base ~$88 · Memory Ops Pack ~$119 (local-primary pull/retain/support · not freemium hosted GPU palace)
    · dual_write OFF · local-primary · Palace sunset · package load ≠ Memory GA
    · Salesforce = GA CRM · HubSpot + GTM suite Beta multi-tenant · guerrilla global-only
    · knowledge / analytical = Beta (not invent GA) · no invent GA knowledge/analytical
    · agentic: MCP list/plan residual-honest · portal HITL · catalog ≠ Connected framing · list_plan_not_connected
    · three product planes via /onboard next planes (mesh · memory-pull · agentic) — streams_not_probed · pull_not_probed · dual_auth_candidacy_open
    · drafts only for GTM · human publish · rates ~$88/$119 optional

  MUST NOT CLAIM (honesty locks — never invent):
    · invent Connected / install APPLY green / INSTALL_STORE green · never invent Connected / INSTALL_STORE APPLY
    · invent Memory GA · invent dual_write as ON · freemium hosted palace · Ops Pack as GPU fleet · Ops Pack ≠ GPU fleet
    · invent book-demo as ON (book-demo OFF) · leave ON_SIGNAL unset
    · invent dual-auth as live · empty-as-none invent · agent MCP write installs · tool ship ≠ dual-auth live · dual_auth_candidacy_open
    · invent Knowledge/Analytics as GA · Knowledge Beta→GA cannot invent H1/H2 offline
    · invent human-gate green (Slack HMAC · Stripe Customers:Write · H1/H2 INSTALL_STORE still human) · PASS ≠ invent human-gate green · open boxes stay open
    · residual PASS ≠ live dogfood · PASS ≠ live APPLY · board/export evidence ≠ invent Connected

  three-planes companion (product narrative ground):
    · plane 1 mesh: streaming org heartbeats · mesh ≠ memory · streams_not_probed · /onboard next mesh
    · plane 2 memory-pull: mesh → local palace egress · dual_write OFF · pull_not_probed · /onboard next memory-pull
    · plane 3 agentic: list/plan residual · portal HITL · list_plan_not_connected · dual_auth_candidacy_open · /onboard next agentic
    · consolidate: /onboard next planes (aliases three-planes|product-planes|product|pillars|three_planes)

  slash: /onboard next sales (aliases claims|buyer|claim-matrix|sales-claims|buyer-claims)
    · NOT product|planes (those stay three-planes board) · NOT gtm|drafts (GTM draft lane) · NOT pulse|board (status board)
  companion: /onboard next planes · /onboard next mesh · /onboard next memory-pull · /onboard next agentic · /onboard next agentic dual-auth
  companion boards: /onboard next status · /onboard next export · /onboard next human-gates · /onboard next gtm · /onboard next · /onboard status

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open · never invent Connected / INSTALL_STORE APPLY · mesh ≠ memory · dual_auth_candidacy_open · tool ship ≠ dual-auth live · never invent dual-auth live · drafts only · no auto-send · rates ~$88/$119 optional · Ops Pack ≠ GPU fleet · pull ≠ freemium hosted palace · agent MCP cannot write installs · catalog ≠ Connected · list_plan_not_connected · no invent GA · board/export evidence ≠ invent Connected · leave ON_SIGNAL unset · Knowledge Beta→GA cannot invent H1/H2 offline · PASS ≠ invent human-gate green`)
}

// AionAgentOnboardingNextLaneStatus residual-honest post-onboard lane status board for
// /onboard next status (aliases pulse|board) — free eng s1382 (+ s1387 export · s1397 session soft dogfood · s1402 mesh · s1407 memory-pull · s1417 agentic).
// Default path: no MCP dial · no invent install green / Connected / GA / APPLY / stream green / pull green.
// Honest state vocabulary only: path_ready · samples_ok · samples_missing ·
// dogfood_not_run · soft_offline_dogfood_session_pass|fail · skill_ready · residual_only ·
// streams_not_probed · pull_not_probed · portal_hitl_still · list_plan_not_connected.
// Optional soft check: sample package dirs via agentplugins.
// Plugins dogfood state: default dogfood_not_run; after /plugins dogfood session marker
// soft_offline_dogfood_session_pass|fail (session soft ≠ live dogfood · ≠ invent Agent Plugins GA).
// Mesh: path_ready · residual_only · streams_not_probed (never invent live green / Connected).
// Memory-pull: path_ready · residual_only · pull_not_probed (never invent pull green).
// Agentic (s1417+s1422+s1427): path_ready · residual_only · portal_hitl_still · list_plan_not_connected · <soft label>
// · dual_auth_candidacy_open · list_org_unavailable (never invent dual-auth live · tool ship ≠ dual-auth live).
// (never invent Connected · session soft ≠ live dogfood · soft offline ≠ invent Connected).
// Never claims dogfood PASS live, Agent Plugins GA, Memory GA, or install Connected.
func AionAgentOnboardingNextLaneStatus() string {
	samplesState := nextLanePluginsSamplesSoftState()
	dogfoodState := nextLanePluginsDogfoodSessionState()
	agenticSoft := AgenticListPlanSoftSessionLabel()
	// samples_ok|samples_missing is path soft-check only ≠ residual PASS / live dogfood.
	// dogfoodState is session soft marker only (default dogfood_not_run) — ≠ live dogfood.
	// agenticSoft is independent session soft for list/plan (default list_plan_soft_not_run).
	return strings.TrimSpace(fmt.Sprintf(`aion onboard next lane status (residual-honest · s1382+s1397+s1402+s1407+s1417+s1422+s1427 · no MCP dial · not live dogfood):
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

  agentic: path_ready · residual_only · portal_hitl_still · list_plan_not_connected · dual_auth_candidacy_open · list_org_unavailable · %s
    · product plane 3 agentic integrations · MCP list/plan residual-honest · never invent Connected
    · plan_connector_setup → portal deep links · browser HITL only · template= ≠ install APPLY green
    · list_org fail-open ≠ empty-as-none · agent MCP cannot write installs · catalog ≠ Connected
    · portal HITL @ https://console.iome.sh/integrations · Agent/MCP @ https://console.iome.sh/settings/agent
    · session soft list/plan ≠ live dogfood · soft offline ≠ invent Connected · portal HITL still
    · dual_auth_candidacy_open · list_org_unavailable · tool ship ≠ dual-auth live · never invent dual-auth live
    · drill: /onboard next agentic (aliases agentic-integrations|integrations|portal-hitl|list-plan|hitl)
    · soft dogfood: /onboard next agentic dogfood (aliases soft|samples|offline|list-plan-soft)
    · dual-auth depth: /onboard next agentic dual-auth (aliases candidacy|list-org|org-installs|dual_auth|dual-auth-candidacy)
    · NOT bare mcp (memory) · NOT bare portal (portal handoff)

  portal: portal_hitl_still
    · agent MCP cannot write installs · catalog ≠ Connected · portal HITL still for OAuth/install
    · list_org fail-open ≠ empty-as-none · never invent Connected / INSTALL_STORE APPLY
    · Agent/MCP mint/copy/probe @ https://console.iome.sh/settings/agent · connectors @ https://console.iome.sh/integrations

  slash: /onboard next status (aliases pulse|board) · /onboard next export (aliases receipt|stamp|evidence) · /onboard next mesh · /onboard next memory-pull · /onboard next agentic (aliases agentic-integrations|integrations|portal-hitl|list-plan|hitl) · /onboard next agentic dogfood · /onboard next agentic dual-auth · /onboard next planes (aliases three-planes|product-planes|product|pillars|three_planes) · /onboard next sales (aliases claims|buyer|claim-matrix|sales-claims|buyer-claims) · /onboard next human-gates (aliases human|gates|apply-gates) · /onboard next · /onboard status · /integrations status · /plugins dogfood · /plugins status · /mesh
  export receipt: /onboard next export — offline markdown evidence of this board (board/export evidence ≠ invent Connected)
  three product planes: /onboard next planes — mesh · memory-pull · agentic residual-honest consolidate · streams_not_probed · pull_not_probed · list_plan_not_connected · dual_auth_candidacy_open · never invent Connected (s1432)
  sales/buyer claims: /onboard next sales (aliases claims|buyer|claim-matrix|sales-claims|buyer-claims) — may claim / must not claim residual-honest · three-planes grounded · never invent Connected / Memory GA (s1437)
  human-gates: /onboard next human-gates — still human vs offline residual · PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open (s1413)
  plugins soft offline: /plugins dogfood (aliases soft|samples|offline) — soft offline ≠ live dogfood · ≠ invent Agent Plugins GA · session soft refreshes this board
  agentic soft offline: /onboard next agentic dogfood (aliases soft|samples|offline|list-plan-soft) — soft offline list/plan ≠ live dogfood · ≠ invent Connected · session soft refreshes agentic lane
  agentic dual-auth candidacy: /onboard next agentic dual-auth (aliases candidacy|list-org|org-installs|dual_auth|dual-auth-candidacy) — dual_auth_candidacy_open · list_org_unavailable · tool ship ≠ dual-auth live (s1427)

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY · PASS ≠ invent human-gate green · open boxes stay open · never invent install green / Connected / INSTALL_STORE APPLY · catalog ≠ Connected · portal HITL · agent MCP cannot write installs · plugins dogfood ≠ invent Agent Plugins GA · session soft ≠ live dogfood · soft offline ≠ live dogfood · drafts only · no auto-send · package load ≠ Memory GA · rates ~$88/$119 optional · board/export evidence ≠ invent Connected · mesh = streaming org heartbeats · mesh ≠ memory · never invent stream green · streams_not_probed · not OTel/APM · pull_not_probed · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · never invent pull green · list_plan_not_connected · plan deep links = browser HITL only · template= ≠ install APPLY · dual_auth_candidacy_open · list_org_unavailable · tool ship ≠ dual-auth live · never invent dual-auth live · Knowledge Beta→GA cannot invent H1/H2 offline · leave ON_SIGNAL unset`, samplesState, dogfoodState, agenticSoft))
}

// AionAgentOnboardingNextLaneStatusExport residual-honest markdown status export receipt
// for /onboard next export (aliases receipt|stamp|evidence) — free eng s1387 (+ s1397 session soft · s1402 mesh · s1407 memory-pull · s1417 agentic · s1422 agentic list/plan soft).
// Offline evidence of the s1382 lane status board vocabulary; plugins dogfood lane reflects
// session soft marker from /plugins dogfood when present (default dogfood_not_run).
// Mesh lane: path_ready · residual_only · streams_not_probed (never invent stream green).
// Memory-pull lane: path_ready · residual_only · pull_not_probed (never invent pull green).
// Agentic lane (s1417+s1422): path_ready · residual_only · portal_hitl_still · list_plan_not_connected · <soft label>.
// Does NOT run plugins/agentic dogfood itself, dial MCP, or invent install green / Connected / GA / APPLY.
// Header pins evidence_kind=onboard_next_lane_status_export · offline_static ·
// not_live_dogfood · serial s1387.
// board/export evidence ≠ invent Connected · session soft ≠ live dogfood.
func AionAgentOnboardingNextLaneStatusExport() string {
	samplesState := nextLanePluginsSamplesSoftState()
	dogfoodState := nextLanePluginsDogfoodSessionState()
	agenticSoft := AgenticListPlanSoftSessionLabel()
	return strings.TrimSpace(fmt.Sprintf(`# aion onboard next lane status export receipt

evidence_kind=onboard_next_lane_status_export
offline_static=true
not_live_dogfood=true
serial=s1387
format=markdown

## board (honest vocabulary only — s1382+s1397+s1402+s1407+s1417+s1422 reuse · never invent connected / ga / apply / stream green / pull green as success)

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

agentic: path_ready · residual_only · portal_hitl_still · list_plan_not_connected · dual_auth_candidacy_open · list_org_unavailable · %s
  · product plane 3 agentic integrations · MCP list/plan residual-honest · never invent Connected
  · plan_connector_setup → portal deep links · browser HITL only · template= ≠ install APPLY green
  · list_org fail-open ≠ empty-as-none · agent MCP cannot write installs · catalog ≠ Connected
  · portal HITL @ https://console.iome.sh/integrations · Agent/MCP @ https://console.iome.sh/settings/agent
  · session soft list/plan ≠ live dogfood · soft offline ≠ invent Connected · portal HITL still
  · dual_auth_candidacy_open · list_org_unavailable · tool ship ≠ dual-auth live · never invent dual-auth live
  · drill: /onboard next agentic (aliases agentic-integrations|integrations|portal-hitl|list-plan|hitl)
  · soft dogfood: /onboard next agentic dogfood (aliases soft|samples|offline|list-plan-soft)
  · dual-auth depth: /onboard next agentic dual-auth (aliases candidacy|list-org|org-installs|dual_auth|dual-auth-candidacy)

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
- list_plan_not_connected · plan deep links = browser HITL only · template= ≠ install APPLY
- soft offline list/plan ≠ live dogfood · ≠ invent Connected · portal HITL still
- dual_auth_candidacy_open · list_org_unavailable · tool ship ≠ dual-auth live · never invent dual-auth live
- this receipt does NOT run plugins dogfood · does NOT run agentic list/plan dogfood · does NOT dial MCP · does NOT invent green
- session soft marker (if present) ≠ live dogfood · ≠ invent Agent Plugins GA · board ≠ invent Connected

## slash

/onboard next export (aliases receipt|stamp|evidence) · optional /onboard next export json
/onboard next status (aliases pulse|board) · /onboard next mesh · /onboard next memory-pull · /onboard next agentic · /onboard next agentic dogfood · /onboard next agentic dual-auth · /onboard next human-gates · /onboard next · /onboard status
/onboard next agentic (aliases agentic-integrations|integrations|portal-hitl|list-plan|hitl) — product plane 3 MCP list/plan + portal HITL · list_plan_not_connected · never invent Connected (s1417)
/onboard next agentic dogfood (aliases soft|samples|offline|list-plan-soft) — soft offline list/plan residual · session soft ≠ live dogfood · ≠ invent Connected (s1422)
/onboard next agentic dual-auth (aliases candidacy|list-org|org-installs|dual_auth|dual-auth-candidacy) — dual_auth_candidacy_open · list_org_unavailable · tool ship ≠ dual-auth live (s1427)
/onboard next human-gates (aliases human|gates|apply-gates) — still human vs offline residual · PASS ≠ invent human-gate green · PASS ≠ live APPLY (s1413)
/plugins dogfood (aliases soft|samples|offline) · /plugins status — soft offline ≠ live dogfood · ≠ invent Agent Plugins GA

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY · PASS ≠ invent human-gate green · open boxes stay open · never invent install green / Connected / INSTALL_STORE APPLY · catalog ≠ Connected · portal HITL · agent MCP cannot write installs · plugins dogfood ≠ invent Agent Plugins GA · session soft ≠ live dogfood · soft offline ≠ live dogfood · drafts only · no auto-send · package load ≠ Memory GA · board/export evidence ≠ invent Connected · rates ~$88/$119 optional · mesh = streaming org heartbeats · mesh ≠ memory · never invent stream green · streams_not_probed · pull_not_probed · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · never invent pull green · list_plan_not_connected · plan deep links = browser HITL only · template= ≠ install APPLY · dual_auth_candidacy_open · list_org_unavailable · tool ship ≠ dual-auth live · never invent dual-auth live · Knowledge Beta→GA cannot invent H1/H2 offline · leave ON_SIGNAL unset`, samplesState, dogfoodState, agenticSoft))
}

// nextLaneStatusExportDTO is the offline JSON shape for AionAgentOnboardingNextLaneStatusExportJSON (s1387+s1397+s1422).
// Honest vocabulary only — never invents Connected/GA/APPLY success.
// Plugins dogfood session state (s1397): dogfood_not_run default; soft_offline_dogfood_session_pass|fail after /plugins dogfood.
// Agentic list/plan soft state (s1422): list_plan_soft_not_run default; soft_offline_list_plan_session_pass|fail after /onboard next agentic dogfood.
type nextLaneStatusExportDTO struct {
	EvidenceKind             string            `json:"evidence_kind"`
	OfflineStatic            bool              `json:"offline_static"`
	NotLiveDogfood           bool              `json:"not_live_dogfood"`
	Serial                   string            `json:"serial"`
	Format                   string            `json:"format"`
	Lanes                    map[string]string `json:"lanes"`
	SamplesState             string            `json:"samples_state"`
	PluginsDogfoodState      string            `json:"plugins_dogfood_state"`
	DogfoodNotRun            bool              `json:"dogfood_not_run"`
	AgenticListPlanSoftState string            `json:"agentic_list_plan_soft_state"`
	HonestyLocks             []string          `json:"honesty_locks"`
	Slash                    []string          `json:"slash"`
	Note                     string            `json:"note"`
}

// AionAgentOnboardingNextLaneStatusExportJSON residual-honest JSON status export receipt
// for /onboard next export json (s1387+s1397+s1402+s1407+s1413+s1417+s1422). Same honesty as markdown; offline only.
// Reflects session soft dogfood marker when set by /plugins dogfood (≠ live dogfood).
// Reflects agentic list/plan soft marker when set by /onboard next agentic dogfood (≠ invent Connected).
// Mesh lane: path_ready · residual_only · streams_not_probed (never invent stream green).
// Memory-pull / ops_pack lane: path_ready · residual_only · pull_not_probed (never invent pull green).
// Agentic lane (s1417+s1422): path_ready · residual_only · portal_hitl_still · list_plan_not_connected · <soft label>.
// s1413: slash tip + honesty locks for human-gates board (still ≠ invent human-gate green / live APPLY).
// Does NOT run plugins/agentic dogfood itself, dial MCP, or invent install green / Connected / GA / APPLY.
func AionAgentOnboardingNextLaneStatusExportJSON() string {
	samplesState := nextLanePluginsSamplesSoftState()
	dogfoodState := nextLanePluginsDogfoodSessionState()
	agenticSoft := AgenticListPlanSoftSessionLabel()
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
			"agentic":     fmt.Sprintf("path_ready · residual_only · portal_hitl_still · list_plan_not_connected · %s", agenticSoft),
			"portal":      "portal_hitl_still",
		},
		SamplesState:        samplesState,
		PluginsDogfoodState: dogfoodState,
		// dogfood_not_run true only when session soft dogfood has not run (s1397).
		// When ran, still not_live_dogfood=true — session soft ≠ live dogfood.
		DogfoodNotRun:            !ran,
		AgenticListPlanSoftState: agenticSoft,
		HonestyLocks: []string{
			"dual_write OFF",
			"book-demo OFF",
			"not Memory GA",
			"residual PASS ≠ live dogfood",
			"session soft ≠ live dogfood",
			"soft offline list/plan ≠ live dogfood",
			"soft offline ≠ invent Connected",
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
			"list_plan_not_connected",
			"plan deep links = browser HITL only",
			"template= ≠ install APPLY",
			"PASS ≠ invent human-gate green",
			"PASS ≠ live APPLY",
			"open boxes stay open",
			"Knowledge Beta→GA cannot invent H1/H2 offline",
			"leave ON_SIGNAL unset",
		},
		Slash: []string{
			"/onboard next export",
			"/onboard next export json",
			"/onboard next status",
			"/onboard next mesh",
			"/onboard next memory-pull",
			"/onboard next agentic",
			"/onboard next agentic dogfood",
			"/onboard next human-gates",
			"/plugins dogfood",
			"/plugins status",
		},
		Note: "offline residual-honest lane board evidence; plugins_dogfood_state is session soft marker only (default dogfood_not_run); agentic_list_plan_soft_state is independent session soft for list/plan (default list_plan_soft_not_run · s1422); mesh lane is path_ready · residual_only · streams_not_probed (never invent stream green); memory-pull/ops_pack lane is path_ready · residual_only · pull_not_probed (never invent pull green); agentic lane is path_ready · residual_only · portal_hitl_still · list_plan_not_connected · soft label (never invent Connected · product plane 3 MCP list/plan + portal HITL · s1417+s1422); session soft ≠ live dogfood; soft offline list/plan ≠ invent Connected; board/export evidence ≠ invent Connected; does NOT run plugins dogfood or agentic list/plan dogfood or dial MCP; soft offline dogfood ≠ invent Agent Plugins GA; mesh ≠ memory; pull ≠ freemium hosted palace; Ops Pack ≠ GPU fleet; human-gates: PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open (s1413 tip only)",
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

// AionAgentHumanGatesHonestyBoard residual-honest human-gates status section for operators
// (/onboard next human-gates · aliases human|gates|apply-gates) — free eng s1413.
// Static offline — no MCP dial · no gcloud · never invents human-gate green or live APPLY.
// Separates: still human APPLY · offline residual only · shipped/policy.
// Explicit: local memory / dual_write OFF / residual-honest agent MCP list/plan do NOT close human APPLY gates.
// Operator tip: re-run make human-gates-status / residual gate on aion · never invent APPLY.
// Honesty locks: PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open ·
// dual_write OFF · book-demo OFF · leave ON_SIGNAL unset · Knowledge Beta→GA cannot invent H1/H2 offline ·
// not Memory GA · rates ~$88/$119 optional · analytical NO-install intentional.
func AionAgentHumanGatesHonestyBoard() string {
	return strings.TrimSpace(`aion human-gates honesty board (residual-honest · s1413 · no MCP dial · not live APPLY):
  board (honest vocabulary only — never invent human-gate green / Connected / APPLY / H1/H2 green as success):

  still_human (open boxes stay open · human APPLY only):
    · Slack HMAC rotate — Signing Secret still human if not rotated · make broker-hmac-rotate dry-run default · SM name present ≠ real App secrets
    · Stripe Customers:Write — Dashboard ACL on live restricted key · SM present ≠ Write granted · Checkout may use customer_email fallback
    · H1/H2 INSTALL_STORE image APPLY — CP image first · broker second · VPC/rqlite · Knowledge Beta→GA cannot invent H1/H2 offline
    · knowledge live dogfood D1–D5 — after H1/H2 only · dry-run ≠ APPLY · fixture ≠ live dogfood · catalog knowledge stays Beta
    · book-demo OFF (to turn ON needs separate launch gates) · leave ON_SIGNAL unset (no invent warm APPLY)

  offline_residual_only (PASS ≠ invent human-gate green · PASS ≠ live APPLY):
    · residual gates (make human-gates-hmac-stripe-install-store-residual-gate · mesh INSTALL_STORE residual · knowledge dry-run residual)
    · soft dogfood / offline samples · session soft ≠ live dogfood · residual PASS ≠ live dogfood
    · agent MCP list/plan · residual-honest list_org fail-open · catalog ≠ Connected · agent MCP cannot write installs
    · dry-run paths (broker-hmac-rotate dry-run · stage install dogfood dry-run · knowledge dry-run) · dry-run ≠ APPLY

  shipped_or_policy (do not re-claim as closing open human boxes):
    · GitHub App HMAC may be dogfood-proven (signed ping 200) — Slack still human until rotated
    · dual_write OFF · Palace sunset · local-primary memory · not Memory GA · package load ≠ Memory GA
    · analytical NO-install intentional (dbt/warehouse INSTALL_STORE not shipped · embeddings N/A by design)
    · rates ~$88 mesh / ~$119 Memory Ops Pack optional — commercial framing only · not product GA claim

  do_not_close (explicit · offline residual never invents APPLY):
    · local memory / dual_write OFF / residual-honest agent MCP list/plan do NOT close human APPLY gates
    · board/export evidence ≠ invent Connected · offline PASS ≠ invent human-gate green · open boxes stay open

  operator:
    · re-run make human-gates-status (aion s191 · read-only · NO APPLY) · residual gate on aion
    · never invent APPLY · never invent Slack rotate done · never invent Stripe Write · never invent INSTALL_STORE green
    · slash: /onboard next human-gates (aliases human|gates|apply-gates) · /onboard next status · /onboard next export
    · companion: /onboard next · /onboard status · /integrations status · portal HITL https://console.iome.sh/integrations

Locks: dual_write OFF · book-demo OFF · leave ON_SIGNAL unset · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY · PASS ≠ invent human-gate green · open boxes stay open · Knowledge Beta→GA cannot invent H1/H2 offline · never invent install green / Connected / INSTALL_STORE APPLY · catalog ≠ Connected · portal HITL · agent MCP cannot write installs · dry-run ≠ APPLY · rates ~$88/$119 optional · analytical NO-install intentional · board/export evidence ≠ invent Connected`)
}
