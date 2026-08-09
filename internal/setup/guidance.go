package setup

import "strings"

// Portal URLs for residual-honest setup handoff (browser HITL).
const (
	PortalIntegrationsURL  = "https://console.iome.sh/integrations"
	PortalAgentSettingsURL = "https://console.iome.sh/settings/agent"
)

// SetupLifecycleHonestyOneLiner is the bare /setup status honesty line (s1526 P3 + s1530 P5 + s1534 P6 + s1538 P7).
const SetupLifecycleHonestyOneLiner = "dual_write OFF · not Memory GA · catalog ≠ Connected · portal HITL · setup PASS ≠ invent install green · continuous pull opt-in (/setup pull · pull_continuous) · analyze ticks opt-in (/setup analyze · analyze_continuous) · drift report-only (/setup drift) · guided repair (/setup repair · apply --yes only) · CLI iomesh memory pull still valid · /memory digest still valid · pull/analyze/repair ≠ invent Connected · drift ≠ invent install green · repair apply ≠ invent Connected"

// SetupLifecycleAgentGuidanceNote is the residual-honest system note injected on
// AttachMCP (s1526 P3 + s1530 P5 + s1534 P6 + s1538 P7). Steers the LLM: setup init →
// preflight → portal HITL → in-session opt-in continuous pull / analyze ticks / drift
// report / guided repair — without inventing Connected / Memory GA / INSTALL_STORE green.
// Unit-tested for honesty needles.
func SetupLifecycleAgentGuidanceNote() string {
	return strings.TrimSpace(`setup lifecycle (residual-honest agent path · s1526 P3 / s1530 P5 / s1534 P6 / s1538 P7 / skill setup-lifecycle-agent):
1. Init managed config: slash /setup init [profiles] or CLI iomesh setup init — dual_write OFF · secrets env names only · pull_continuous=false · analyze_continuous=false default
2. Preflight probe: /setup preflight (aliases status|check) or iomesh setup preflight — state probe · PASS ≠ invent Connected
3. Portal HITL for OAuth/install: ` + PortalIntegrationsURL + ` · agent settings ` + PortalAgentSettingsURL + ` — agent MCP cannot write installs
4. Continuous pull in-session opt-in: /setup pull start|once|stop|status (loads [memory] pull_* · pull_continuous=true is config opt-in) · CLI iomesh memory pull still valid
5. Analyze ticks in-session opt-in (s1534 P6): /setup analyze start|once|stop|status (loads [memory] analyze_* · analyze_continuous=true is config opt-in · --mode status|digest · --interval N · --window day|week) · /memory digest still valid · analyze tick ≠ invent Connected
6. Drift / maintain report-only (s1534 P6): /setup drift · /setup maintain — FormatDriftText · residual next steps · drift report ≠ invent install green · package wire ≠ Connected
7. Guided repair (s1538 P7): /setup repair · /setup repair plan — PlanRepair from drift (dry plan) · /setup repair apply --yes — ApplyRepairPlan safe steps only (reload_mcp · start_pull · start_analyze) · refuse without --yes · notes for human host/mesh/dual_write · repair apply ≠ invent Connected · dual_write never auto-flipped ON · portal HITL still human
8. Skill: read_skill setup-lifecycle-agent when available · operator slash /setup (alias /setup-lifecycle)

Locks (never violate):
- dual_write OFF · not Memory GA · book-demo OFF
- never invent install green / Connected / INSTALL_STORE APPLY / GA
- catalog status ≠ install Connected
- portal HITL for OAuth/install · secrets as env names only
- setup PASS / local_memory_probe_ok ≠ invent product green
- continuous pull is opt-in only · /setup pull · pull_continuous · CLI iomesh memory pull still valid · pull ≠ invent Connected
- analyze ticks are opt-in only · /setup analyze · analyze_continuous · /memory digest still valid · analyze tick ≠ invent Connected
- drift is report-only · /setup drift · /setup maintain · residual next steps · drift report ≠ invent install green · package wire ≠ Connected
- guided repair needs explicit /setup repair apply --yes · safe steps only · repair apply ≠ invent Connected · no auto-repair without --yes · dual_write never auto-flipped ON · portal HITL still human`)
}

// SetupLifecyclePortalHandoff residual-honest portal URLs for /setup portal.
func SetupLifecyclePortalHandoff() string {
	return strings.TrimSpace(`setup portal handoff (residual-honest · browser HITL · s1526):
  integrations: ` + PortalIntegrationsURL + `
  agent / MCP settings: ` + PortalAgentSettingsURL + `
  agent MCP cannot write installs · OAuth complete in console session
  honesty: ` + SetupLifecycleHonestyOneLiner)
}
