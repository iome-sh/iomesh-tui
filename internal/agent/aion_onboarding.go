package agent

import "strings"

// AionAgentOnboardingGuidanceNote residual-honest system note (s1363 + s1368).
// Injected on AttachMCP after integrations + memory-advanced notes.
// Steers TUI agent ↔ aion CP/MCP onboarding without inventing install green,
// Memory GA, Agent Plugins GA, or dual_write ON.
// s1368: adds explicit portal Agent/MCP handoff lane (mint key → copy MCP →
// test invoke probe only) complementary to integrations portal HITL.
// Unit-tested for honesty needles. Molds IntegrationsAgentGuidanceNote /
// GtmDraftOnlyAgentGuidanceNote / MemoryAdvancedAgentGuidanceNote.
func AionAgentOnboardingGuidanceNote() string {
	return strings.TrimSpace(`aion agent onboarding (residual-honest TUI ↔ aion CP/MCP · s1363+s1368):
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

Skill: read_skill aion-agent-onboarding when available

Locks (never violate):
- dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood
- never invent install green / Connected / INSTALL_STORE APPLY
- list_org_connector_installs available=false ≠ empty-as-none
- catalog status ≠ Connected · portal HITL for OAuth/install · agent MCP cannot write installs
- plugins dogfood ≠ invent Agent Plugins GA · rates ~$88/$119 optional
- no invent GA for knowledge/analytical
- test invoke = probe only ≠ Memory GA · mint key ≠ invent install Connected`)
}

// AionAgentOnboardingChecklist residual-honest numbered onboarding checklist (s1363 + s1368).
// Used by /onboard help and /onboard checklist — operator HITL only; never invents
// install green, Memory GA, Agent Plugins GA, dual_write ON, or agent APPLY.
// s1368: portal Agent/MCP handoff steps (mint/copy/probe) + TUI [[mcp.servers]].
func AionAgentOnboardingChecklist() string {
	return strings.TrimSpace(`aion agent onboarding checklist (residual-honest · s1363+s1368 · TUI ↔ aion):
  1. Point IOMESH/MCP at aion tools (fail-open offline)
  2. list_connector_catalog — catalog status ≠ Connected
  3. plan_connector_setup → portal deep links (browser HITL)
  4. list_org_connector_installs residual fail-open (available=false ≠ empty-as-none)
  5. Portal Agent/MCP: mint key → Settings → Agent/MCP → copy MCP connection → test invoke (probe only ≠ Memory GA) at https://console.iome.sh/settings/agent
  6. TUI: [[mcp.servers]] streamable HTTP → /onboard · /integrations status (agent MCP cannot write installs)
  7. Memory dual_write OFF · local-primary · not Memory GA · optional plugins dogfood ≠ Agent Plugins GA
  8. Operator: /integrations status · /onboard checklist · /onboard portal · portal https://console.iome.sh/integrations
  Locks: never invent install green / Connected / INSTALL_STORE APPLY · book-demo OFF · residual PASS ≠ live dogfood · rates ~$88/$119 optional · no invent GA knowledge/analytical · catalog status ≠ Connected · portal HITL`)
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

// AionAgentOnboardingStatus residual-honest static offline status lines for /onboard status (s1368).
// No MCP dial — operator pulse only. Never invents attach green, install Connected, or Memory GA.
func AionAgentOnboardingStatus() string {
	return strings.TrimSpace(`aion onboard status (residual-honest · offline static · s1368):
  MCP attach: expected for full path · fail-open offline (never invent tool green / install green)
  dual_write OFF · local-primary · not Memory GA · book-demo OFF
  portal HITL: Agent/MCP mint/copy/probe @ https://console.iome.sh/settings/agent · connectors @ https://console.iome.sh/integrations
  never invent install green / Connected / INSTALL_STORE APPLY
  list_org fail-open (available=false) ≠ empty-as-none · catalog ≠ Connected
  agent MCP cannot write installs · plugins dogfood ≠ invent Agent Plugins GA
  residual PASS ≠ live dogfood · test invoke = probe only ≠ Memory GA
  slash: /onboard portal · /onboard checklist · /integrations status`)
}
