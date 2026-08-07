package agent

import "strings"

// AionAgentOnboardingGuidanceNote residual-honest system note (s1363 + s1368 + s1372 + s1377).
// Injected on AttachMCP after integrations + memory-advanced notes.
// Steers TUI agent ↔ aion CP/MCP onboarding without inventing install green,
// Memory GA, Agent Plugins GA, or dual_write ON.
// s1368: adds explicit portal Agent/MCP handoff lane (mint key → copy MCP →
// test invoke probe only) complementary to integrations portal HITL.
// s1372: cross-link post-onboard continuum → /onboard next operator lanes.
// s1377: /onboard next <lane> drill-down (plugins · gtm · memory).
// Unit-tested for honesty needles. Molds IntegrationsAgentGuidanceNote /
// GtmDraftOnlyAgentGuidanceNote / MemoryAdvancedAgentGuidanceNote.
func AionAgentOnboardingGuidanceNote() string {
	return strings.TrimSpace(`aion agent onboarding (residual-honest TUI ↔ aion CP/MCP · s1363+s1368+s1372+s1377):
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
7. Post-onboard continuum: /onboard next [plugins|gtm|memory] (plugins dogfood · /gtm checklist · aion-memory-mcp local · portal HITL still)

Skill: read_skill aion-agent-onboarding when available

Locks (never violate):
- dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood
- never invent install green / Connected / INSTALL_STORE APPLY
- list_org_connector_installs available=false ≠ empty-as-none
- catalog status ≠ Connected · portal HITL for OAuth/install · agent MCP cannot write installs
- plugins dogfood ≠ invent Agent Plugins GA · rates ~$88/$119 optional
- no invent GA for knowledge/analytical
- test invoke = probe only ≠ Memory GA · mint key ≠ invent install Connected
- drafts only · no auto-send · package load ≠ Memory GA`)
}

// AionAgentOnboardingChecklist residual-honest numbered onboarding checklist (s1363 + s1368 + s1372 + s1377).
// Used by /onboard help and /onboard checklist — operator HITL only; never invents
// install green, Memory GA, Agent Plugins GA, dual_write ON, or agent APPLY.
// s1368: portal Agent/MCP handoff steps (mint/copy/probe) + TUI [[mcp.servers]].
// s1372: cross-link → /onboard next operator lanes (post-onboard continuum).
// s1377: /onboard next [plugins|gtm|memory] lane drills.
func AionAgentOnboardingChecklist() string {
	return strings.TrimSpace(`aion agent onboarding checklist (residual-honest · s1363+s1368+s1372+s1377 · TUI ↔ aion):
  1. Point IOMESH/MCP at aion tools (fail-open offline)
  2. list_connector_catalog — catalog status ≠ Connected
  3. plan_connector_setup → portal deep links (browser HITL)
  4. list_org_connector_installs residual fail-open (available=false ≠ empty-as-none)
  5. Portal Agent/MCP: mint key → Settings → Agent/MCP → copy MCP connection → test invoke (probe only ≠ Memory GA) at https://console.iome.sh/settings/agent
  6. TUI: [[mcp.servers]] streamable HTTP → /onboard · /integrations status (agent MCP cannot write installs)
  7. Memory dual_write OFF · local-primary · not Memory GA · optional plugins dogfood ≠ Agent Plugins GA
  8. Operator: /integrations status · /onboard checklist · /onboard portal · portal https://console.iome.sh/integrations
  9. Post-onboard: /onboard next [plugins|gtm|memory] (plugins · gtm · memory local · portal HITL still)
  Locks: never invent install green / Connected / INSTALL_STORE APPLY · book-demo OFF · residual PASS ≠ live dogfood · rates ~$88/$119 optional · no invent GA knowledge/analytical · catalog status ≠ Connected · portal HITL · drafts only · no auto-send · package load ≠ Memory GA`)
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

// AionAgentOnboardingStatus residual-honest static offline status lines for /onboard status (s1368 + s1372 + s1377).
// No MCP dial — operator pulse only. Never invents attach green, install Connected, or Memory GA.
// s1372: cross-link → /onboard next operator lanes.
// s1377: lane drills via /onboard next [plugins|gtm|memory].
func AionAgentOnboardingStatus() string {
	return strings.TrimSpace(`aion onboard status (residual-honest · offline static · s1368+s1372+s1377):
  MCP attach: expected for full path · fail-open offline (never invent tool green / install green)
  dual_write OFF · local-primary · not Memory GA · book-demo OFF
  portal HITL: Agent/MCP mint/copy/probe @ https://console.iome.sh/settings/agent · connectors @ https://console.iome.sh/integrations
  never invent install green / Connected / INSTALL_STORE APPLY
  list_org fail-open (available=false) ≠ empty-as-none · catalog ≠ Connected
  agent MCP cannot write installs · plugins dogfood ≠ invent Agent Plugins GA
  residual PASS ≠ live dogfood · test invoke = probe only ≠ Memory GA
  slash: /onboard portal · /onboard checklist · /onboard next [plugins|gtm|memory] · /integrations status`)
}

// AionAgentOnboardingNextLanes residual-honest post-onboard continuum for /onboard next (s1372 + s1377).
// Static offline block — no MCP dial. Lists residual-honest operator lanes after
// core onboarding (plugins dogfood · GTM drafts · local memory · portal HITL still).
// s1377: drill-down via /onboard next plugins|gtm|memory (see lane helpers below).
// Never invents Agent Plugins GA, Memory GA, auto-send, or install Connected.
func AionAgentOnboardingNextLanes() string {
	return strings.TrimSpace(`aion onboard next lanes (residual-honest · post-onboard continuum · s1372+s1377 · no MCP dial):
  1. iomesh plugins dogfood — offline sample validate (examples/agent-plugins) · ≠ invent Agent Plugins GA
     drill: /onboard next plugins (aliases plugin|dogfood)
  2. /gtm checklist + skill gtm-draft-only-agent — drafts only · no auto-send · human publish · GTM checklist ≠ invent GTM agent GA
     drill: /onboard next gtm (alias drafts)
  3. local aion-memory-mcp / Memory Ops Pack local-primary — dual_write OFF · package load ≠ Memory GA · ≠ freemium palace
     drill: /onboard next memory (aliases mcp|palace)
  4. portal HITL still required for OAuth/install · agent MCP cannot write installs · catalog ≠ Connected

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · never invent install green / Connected / INSTALL_STORE APPLY · list_org fail-open ≠ empty-as-none · plugins dogfood ≠ invent Agent Plugins GA · drafts only · no auto-send · rates ~$88/$119 optional · package load ≠ Memory GA`)
}

// AionAgentOnboardingNextPluginsLane residual-honest plugins dogfood drill for /onboard next plugins (s1377).
// Static offline — iomesh plugins dogfood path only. Never invents Agent Plugins GA,
// install Connected, dual_write ON, or live dogfood green.
func AionAgentOnboardingNextPluginsLane() string {
	return strings.TrimSpace(`aion onboard next plugins lane (residual-honest · s1377 · no MCP dial):
  Path: iomesh plugins dogfood — offline sample validate only
  Samples: examples/agent-plugins/{hello-iome,aion-memory-mcp}
  Steps:
    1. iomesh plugins list — closed-manifest discovery map (≠ invent install green / Connected)
    2. iomesh plugins validate <path> — offline package shape residual
    3. iomesh plugins dogfood — both in-repo samples offline (residual PASS ≠ live dogfood)
  Honesty:
    · plugins dogfood ≠ invent Agent Plugins GA
    · residual PASS ≠ live dogfood · never invent install green / Connected / INSTALL_STORE APPLY
    · catalog ≠ Connected · agent MCP cannot write installs · portal HITL still for OAuth/install
    · package load ≠ Memory GA · rates ~$88/$119 optional
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
  Back: /onboard next · /memory status · portal Agent/MCP https://console.iome.sh/settings/agent

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · package load ≠ Memory GA · ≠ freemium palace · never invent install green / Connected / INSTALL_STORE APPLY · catalog ≠ Connected · portal HITL · agent MCP cannot write installs · rates ~$88/$119 optional`)
}
