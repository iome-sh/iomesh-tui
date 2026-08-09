package agent

import (
	"fmt"
	"strings"
	"sync"
)

// Portal HITL soft offline dogfood session marker (s1562).
// Separate from agentic list/plan soft (s1422) and plugins SoftDogfoodSession* — independent SSOT.
// Session-only: default portal_hitl_soft_not_run. Soft offline pass/fail ≠ invent Connected ·
// ≠ live dogfood · ≠ invent install APPLY · portal HITL still · soft offline ≠ invent Connected.
//
// Lives in agent so AionAgentOnboardingNextPortalHITLLane board + TUI slash share state
// without import cycles (agent cannot import tui).

var portalHITLSoftDogfoodSession struct {
	mu   sync.Mutex
	ran  bool
	pass bool // residual soft offline pass only
}

// Soft dogfood session state labels for portal HITL (honest vocabulary only).
const (
	// PortalHITLSoftNotRun is the default when /onboard next portal-hitl dogfood has not run this session.
	PortalHITLSoftNotRun = "portal_hitl_soft_not_run"
	// PortalHITLSoftPass is session soft offline portal HITL residual PASS (≠ live dogfood · ≠ invent Connected).
	PortalHITLSoftPass = "soft_offline_portal_hitl_session_pass"
	// PortalHITLSoftFail is session soft offline portal HITL residual FAIL (≠ invent red product / Connected).
	PortalHITLSoftFail = "soft_offline_portal_hitl_session_fail"
)

// SetPortalHITLSoftDogfoodSessionState records that soft offline portal HITL dogfood ran this session.
// pass is residual soft offline only — never invents Connected / install APPLY / live dogfood.
func SetPortalHITLSoftDogfoodSessionState(pass bool) {
	portalHITLSoftDogfoodSession.mu.Lock()
	defer portalHITLSoftDogfoodSession.mu.Unlock()
	portalHITLSoftDogfoodSession.ran = true
	portalHITLSoftDogfoodSession.pass = pass
}

// GetPortalHITLSoftDogfoodSessionState returns whether soft portal HITL dogfood ran this session and residual pass.
// Default: ran=false, pass=false → PortalHITLSoftSessionLabel returns portal_hitl_soft_not_run.
func GetPortalHITLSoftDogfoodSessionState() (ran bool, pass bool) {
	portalHITLSoftDogfoodSession.mu.Lock()
	defer portalHITLSoftDogfoodSession.mu.Unlock()
	return portalHITLSoftDogfoodSession.ran, portalHITLSoftDogfoodSession.pass
}

// PortalHITLSoftSessionLabel returns the honest session soft dogfood vocabulary token:
// portal_hitl_soft_not_run | soft_offline_portal_hitl_session_pass | soft_offline_portal_hitl_session_fail.
// Session soft marker ≠ live dogfood · ≠ invent Connected · portal HITL still.
func PortalHITLSoftSessionLabel() string {
	ran, pass := GetPortalHITLSoftDogfoodSessionState()
	if !ran {
		return PortalHITLSoftNotRun
	}
	if pass {
		return PortalHITLSoftPass
	}
	return PortalHITLSoftFail
}

// ResetPortalHITLSoftDogfoodSessionState clears the session marker (tests only).
func ResetPortalHITLSoftDogfoodSessionState() {
	portalHITLSoftDogfoodSession.mu.Lock()
	defer portalHITLSoftDogfoodSession.mu.Unlock()
	portalHITLSoftDogfoodSession.ran = false
	portalHITLSoftDogfoodSession.pass = false
}

// portalHITLSoftDogfoodNeedles are residual-honesty needles required for soft offline pass.
// Offline-only: board content + proven portal path shapes + honesty locks. Never dials MCP.
// Soft offline ≠ invent Connected · ≠ live dogfood · residual PASS ≠ live dogfood · session soft ≠ live dogfood.
var portalHITLSoftDogfoodNeedles = []string{
	// Board identity + journey stage 5 alignment
	"onboard next portal-hitl lane",
	"journey stage 5",
	"portal HITL when connect",
	"no MCP dial",
	// Path: MCP list/plan → browser portal HITL → human OAuth/install
	"MCP list/plan",
	"browser portal HITL",
	"human finishes OAuth/install",
	// Proven portal path shapes (static strings only)
	"/integrations/{id}",
	"/integrations/add?template={id}",
	"/integrations",
	// Install / Connected locks
	"agent MCP cannot write installs",
	"catalog ≠ Connected",
	"never invent Connected",
	"template= ≠ install APPLY",
	// Portal HITL residual
	"portal HITL still",
	"portal_hitl_still",
	// Policy locks
	"dual_write OFF",
	"book-demo OFF",
	"not Memory GA",
	"Edge Memory GA candidacy only",
	"residual PASS ≠ invent Edge Memory GA",
	// Soft / residual honesty
	"residual PASS ≠ live dogfood",
	"soft offline ≠ invent Connected",
	"session soft ≠ live dogfood",
	// Companion portal surfaces
	"console.iome.sh/integrations",
	"console.iome.sh/settings/agent",
	// Free eng serial
	"free eng s1562",
}

// RunPortalHITLSoftDogfood validates residual honesty of the portal HITL path offline (s1562).
// Checks portal-hitl board needles + proven portal path shapes + honesty locks as static strings.
// Never dials MCP · never invents Connected · never invents install APPLY · soft offline ≠ invent Connected.
// Sets session soft marker (pass/fail). Returns residual-honest operator output.
func RunPortalHITLSoftDogfood() string {
	board := AionAgentOnboardingNextPortalHITLLane()
	var missing []string
	for _, want := range portalHITLSoftDogfoodNeedles {
		if !strings.Contains(board, want) {
			missing = append(missing, want)
		}
	}
	pass := len(missing) == 0
	SetPortalHITLSoftDogfoodSessionState(pass)
	label := PortalHITLSoftSessionLabel()

	var b strings.Builder
	b.WriteString("aion onboard next portal HITL soft offline dogfood (residual-honest · s1562 · no MCP dial · not live dogfood):\n")
	b.WriteString("  Path: soft offline residual check of portal-hitl board honesty + proven portal path shapes (journey stage 5)\n")
	b.WriteString("  · never dial MCP · never invent Connected · never invent install APPLY · portal HITL when connect\n")
	b.WriteString("  · soft offline ≠ invent Connected · residual PASS ≠ live dogfood · session soft ≠ live dogfood · portal HITL still\n")
	b.WriteString("  · agent MCP cannot write installs · catalog ≠ Connected · template= ≠ install APPLY\n")
	b.WriteString("\n")
	if pass {
		b.WriteString("  result: PASS (soft offline residual only)\n")
		b.WriteString(fmt.Sprintf("  checked: %d honesty needles + proven portal path shapes present on portal-hitl board\n", len(portalHITLSoftDogfoodNeedles)))
	} else {
		b.WriteString("  result: FAIL (soft offline residual only · ≠ invent red product / Connected)\n")
		b.WriteString(fmt.Sprintf("  missing needles (%d):\n", len(missing)))
		for _, m := range missing {
			b.WriteString(fmt.Sprintf("    - %q\n", m))
		}
	}
	b.WriteString("\n")
	b.WriteString("  Proven portal paths checked (static offline):\n")
	b.WriteString("    · /integrations/{id}\n")
	b.WriteString("    · /integrations/add?template={id}\n")
	b.WriteString("    · /integrations\n")
	b.WriteString("  Companion portal surfaces checked (static offline):\n")
	b.WriteString("    · console.iome.sh/integrations\n")
	b.WriteString("    · console.iome.sh/settings/agent\n")
	b.WriteString("  Honesty locks checked: portal_hitl_still · portal HITL when connect · portal HITL still\n")
	b.WriteString("    · catalog ≠ Connected · template= ≠ install APPLY · agent MCP cannot write installs\n")
	b.WriteString("    · dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only\n")
	b.WriteString("    · residual PASS ≠ invent Edge Memory GA · residual PASS ≠ live dogfood\n")
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  session marker: %s\n", label))
	b.WriteString("  note: soft offline ≠ invent Connected · residual PASS ≠ live dogfood · session soft ≠ live dogfood · portal HITL still\n")
	b.WriteString("  note: residual PASS ≠ live dogfood · PASS ≠ live APPLY · board evidence ≠ invent Connected · free eng s1562\n")
	b.WriteString("  tip: re-run /onboard next status then /onboard next export — companion agentic list/plan soft remains independent (s1422)\n")
	b.WriteString("  slash: /onboard next portal-hitl dogfood (aliases soft|samples|offline|portal-hitl-soft) · bare /onboard next portal-hitl stays board\n")
	b.WriteString("  companion: /onboard next agentic · /onboard next agentic dogfood · /onboard next journey · /onboard portal mint/copy/probe · /integrations list|plan|status\n")
	b.WriteString("\n")
	b.WriteString("Locks: dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open · never invent install green / Connected / INSTALL_STORE APPLY · catalog ≠ Connected · template= ≠ install APPLY · agent MCP cannot write installs · portal HITL when connect · portal HITL still · portal_hitl_still · soft offline ≠ invent Connected · session soft ≠ live dogfood · free eng s1562 · free-floor peer s1564+ mention only")
	return b.String()
}
