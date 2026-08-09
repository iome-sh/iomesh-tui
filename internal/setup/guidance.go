package setup

import "strings"

// Portal URLs for residual-honest setup handoff (browser HITL).
const (
	PortalIntegrationsURL  = "https://console.iome.sh/integrations"
	PortalAgentSettingsURL = "https://console.iome.sh/settings/agent"
)

// SetupLifecycleHonestyOneLiner is the bare /setup status honesty line (s1526 P3 + s1530 P5).
const SetupLifecycleHonestyOneLiner = "dual_write OFF · not Memory GA · catalog ≠ Connected · portal HITL · setup PASS ≠ invent install green · continuous pull opt-in (/setup pull · pull_continuous) · CLI iomesh memory pull still valid · pull ≠ invent Connected"

// SetupLifecycleAgentGuidanceNote is the residual-honest system note injected on
// AttachMCP (s1526 P3 + s1530 P5). Steers the LLM: setup init → preflight → portal HITL →
// in-session opt-in continuous pull or CLI — without inventing Connected / Memory GA /
// INSTALL_STORE green. Unit-tested for honesty needles.
func SetupLifecycleAgentGuidanceNote() string {
	return strings.TrimSpace(`setup lifecycle (residual-honest agent path · s1526 P3 / s1530 P5 / skill setup-lifecycle-agent):
1. Init managed config: slash /setup init [profiles] or CLI iomesh setup init — dual_write OFF · secrets env names only · pull_continuous=false default
2. Preflight probe: /setup preflight (aliases status|check) or iomesh setup preflight — state probe · PASS ≠ invent Connected
3. Portal HITL for OAuth/install: ` + PortalIntegrationsURL + ` · agent settings ` + PortalAgentSettingsURL + ` — agent MCP cannot write installs
4. Continuous pull in-session opt-in: /setup pull start|once|stop|status (loads [memory] pull_* · pull_continuous=true is config opt-in) · CLI iomesh memory pull still valid · analyze via /memory digest (auto-ticks later)
5. Skill: read_skill setup-lifecycle-agent when available · operator slash /setup (alias /setup-lifecycle)

Locks (never violate):
- dual_write OFF · not Memory GA · book-demo OFF
- never invent install green / Connected / INSTALL_STORE APPLY / GA
- catalog status ≠ install Connected
- portal HITL for OAuth/install · secrets as env names only
- setup PASS / local_memory_probe_ok ≠ invent product green
- continuous pull is opt-in only · /setup pull · pull_continuous · CLI iomesh memory pull still valid · pull ≠ invent Connected`)
}

// SetupLifecyclePortalHandoff residual-honest portal URLs for /setup portal.
func SetupLifecyclePortalHandoff() string {
	return strings.TrimSpace(`setup portal handoff (residual-honest · browser HITL · s1526):
  integrations: ` + PortalIntegrationsURL + `
  agent / MCP settings: ` + PortalAgentSettingsURL + `
  agent MCP cannot write installs · OAuth complete in console session
  honesty: ` + SetupLifecycleHonestyOneLiner)
}
