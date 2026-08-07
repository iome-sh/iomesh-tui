package agent

import "strings"

// GtmDraftOnlyAgentGuidanceNote residual-honest system note (s1347).
// Injected on AttachSkills when the skills catalog attaches (builtin always present
// when skills enabled). Molds MemoryAdvancedAgentGuidanceNote / integrations note.
// Does not invent auto-send, suite ops GA, dual_write ON, or Memory GA.
func GtmDraftOnlyAgentGuidanceNote() string {
	return strings.TrimSpace(`gtm draft-only agent (residual-honest · s1347 / skill s1341):
Drafts and plans only — no auto-send · no auto-publish · human publish · human CRM commercial.
Roles: Orchestrator · Content Creator · Campaign Planner · Lead Manager (text drafts/plans only).

Mesh honesty:
- Salesforce = GA CRM among first-party ops (still no auto commercial CRM write)
- HubSpot + GTM suite = Beta multi-tenant install (not invent GA)
- guerrilla social (X / LinkedIn) = Beta global webhooks only · not multi-tenant install plane
- portal HITL for installs · agent MCP list/plan residual · never invent Connected / suite ops GA

Skill: read_skill gtm-draft-only-agent when available

Locks (never violate):
- drafts only · no auto-send · human publish · human CRM commercial
- dual_write OFF · not Memory GA · book-demo OFF
- never invent install green / Connected / INSTALL_STORE APPLY / suite ops GA
- residual PASS ≠ live dogfood publish`)
}

// GtmDraftChecklist residual-honest numbered draft-only checklist (s1358).
// Used by /gtm help and /gtm checklist — operator HITL only; never invents auto-send,
// suite ops GA, dual_write ON, Memory GA, or agent APPLY for connector installs.
func GtmDraftChecklist() string {
	return strings.TrimSpace(`GTM draft-only checklist (residual-honest · s1358 · no auto-send):
  1. Draft content/outreach only — never auto-send email/SNS
  2. Human publish / human CRM commercial
  3. Salesforce = GA CRM; HubSpot + GTM suite Beta multi-tenant; guerrilla global-only
  4. Connector install = portal HITL (not agent APPLY)
  5. dual_write OFF · not Memory GA · book-demo OFF
  6. Skill: read_skill gtm-draft-only-agent`)
}
