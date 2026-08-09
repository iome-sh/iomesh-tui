package setup

import "strings"

// Portal URLs for residual-honest setup handoff (browser HITL).
const (
	PortalIntegrationsURL  = "https://console.iome.sh/integrations"
	PortalAgentSettingsURL = "https://console.iome.sh/settings/agent"
)

// SetupLifecycleHonestyOneLiner is the bare /setup status honesty line (s1526 P3).
const SetupLifecycleHonestyOneLiner = "dual_write OFF · not Memory GA · catalog ≠ Connected · portal HITL · setup PASS ≠ invent install green · continuous pull still CLI iomesh memory pull"

// SetupLifecycleAgentGuidanceNote is the residual-honest system note injected on
// AttachMCP (s1526 P3). Steers the LLM: setup init → preflight → portal HITL →
// CLI pull — without inventing Connected / Memory GA / INSTALL_STORE green.
// Unit-tested for honesty needles.
func SetupLifecycleAgentGuidanceNote() string {
	return strings.TrimSpace(`setup lifecycle (residual-honest agent path · s1526 P3 / skill setup-lifecycle-agent):
1. Init managed config: slash /setup init [profiles] or CLI iomesh setup init — dual_write OFF · secrets env names only
2. Preflight probe: /setup preflight (aliases status|check) or iomesh setup preflight — state probe · PASS ≠ invent Connected
3. Portal HITL for OAuth/install: ` + PortalIntegrationsURL + ` · agent settings ` + PortalAgentSettingsURL + ` — agent MCP cannot write installs
4. Continuous pull still CLI: iomesh memory pull (in-session pull later PR) · analyze via /memory digest
5. Skill: read_skill setup-lifecycle-agent when available · operator slash /setup (alias /setup-lifecycle)

Locks (never violate):
- dual_write OFF · not Memory GA · book-demo OFF
- never invent install green / Connected / INSTALL_STORE APPLY / GA
- catalog status ≠ install Connected
- portal HITL for OAuth/install · secrets as env names only
- setup PASS / local_memory_probe_ok ≠ invent product green
- continuous pull not in-session yet · use iomesh memory pull`)
}

// SetupLifecyclePortalHandoff residual-honest portal URLs for /setup portal.
func SetupLifecyclePortalHandoff() string {
	return strings.TrimSpace(`setup portal handoff (residual-honest · browser HITL · s1526):
  integrations: ` + PortalIntegrationsURL + `
  agent / MCP settings: ` + PortalAgentSettingsURL + `
  agent MCP cannot write installs · OAuth complete in console session
  honesty: ` + SetupLifecycleHonestyOneLiner)
}
