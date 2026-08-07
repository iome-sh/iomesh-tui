package agent

import "strings"

// AionAgentOnboardingGuidanceNote residual-honest system note (s1363).
// Injected on AttachMCP after integrations + memory-advanced notes.
// Steers TUI agent ↔ aion CP/MCP onboarding without inventing install green,
// Memory GA, Agent Plugins GA, or dual_write ON.
// Unit-tested for honesty needles. Molds IntegrationsAgentGuidanceNote /
// GtmDraftOnlyAgentGuidanceNote / MemoryAdvancedAgentGuidanceNote.
func AionAgentOnboardingGuidanceNote() string {
	return strings.TrimSpace(`aion agent onboarding (residual-honest TUI ↔ aion CP/MCP · s1363):
Point IOMESH/MCP at aion tools — fail-open offline (never invent tool green).
1. Discover: MCP list_connector_catalog — catalog status ≠ install Connected
2. Plan: MCP plan_connector_setup — portal deep links + honesty notes (browser HITL only)
3. Org installs residual: MCP list_org_connector_installs — fail-open (available=false · installs=null) ≠ empty-as-none · never invent Connected
4. Complete OAuth/install in portal HITL at https://console.iome.sh/integrations — agent MCP cannot write installs
5. Memory: dual_write OFF · local-primary · not Memory GA · optional plugins dogfood ≠ invent Agent Plugins GA (rates ~$88/$119 optional)
6. Operator pulse: /integrations status · /onboard checklist · portal HITL

Skill: read_skill aion-agent-onboarding when available

Locks (never violate):
- dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood
- never invent install green / Connected / INSTALL_STORE APPLY
- list_org_connector_installs available=false ≠ empty-as-none
- catalog status ≠ Connected · portal HITL for OAuth/install · agent MCP cannot write installs
- plugins dogfood ≠ invent Agent Plugins GA · rates ~$88/$119 optional
- no invent GA for knowledge/analytical`)
}

// AionAgentOnboardingChecklist residual-honest numbered onboarding checklist (s1363).
// Used by /onboard help and /onboard checklist — operator HITL only; never invents
// install green, Memory GA, Agent Plugins GA, dual_write ON, or agent APPLY.
func AionAgentOnboardingChecklist() string {
	return strings.TrimSpace(`aion agent onboarding checklist (residual-honest · s1363 · TUI ↔ aion):
  1. Point IOMESH/MCP at aion tools (fail-open offline)
  2. list_connector_catalog — catalog status ≠ Connected
  3. plan_connector_setup → portal deep links (browser HITL)
  4. list_org_connector_installs residual fail-open (available=false ≠ empty-as-none)
  5. Memory dual_write OFF · local-primary · not Memory GA · optional plugins dogfood ≠ Agent Plugins GA
  6. Operator: /integrations status · /onboard checklist · portal https://console.iome.sh/integrations
  Locks: never invent install green / Connected / INSTALL_STORE APPLY · book-demo OFF · residual PASS ≠ live dogfood · rates ~$88/$119 optional · no invent GA knowledge/analytical`)
}
