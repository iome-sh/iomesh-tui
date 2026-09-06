package agent

import (
	"fmt"
	"strings"
	"sync"
)

// First-run wizard soft offline dogfood session marker (s1570 Wave C).
// Separate from E4 soft (s1566), portal HITL soft (s1562), and agentic list/plan soft (s1422) — independent SSOT.
// Session-only: default wizard_soft_not_run. Soft offline pass/fail ≠ invent Edge Memory GA declared ·
// ≠ forever-green interactive wizard UX · ≠ dual_write ON · ≠ live dogfood · soft offline ≠ invent Connected ·
// ≠ invent TUI portal SSO · host not auto.
//
// Lives in agent so MeshAgentOnboardingNextWizardLane board + TUI slash share state
// without import cycles (agent cannot import tui).

var wizardSoftDogfoodSession struct {
	mu   sync.Mutex
	ran  bool
	pass bool // residual soft offline pass only
}

// Soft dogfood session state labels for first-run wizard residual (honest vocabulary only).
const (
	// WizardSoftNotRun is the default when /onboard next wizard dogfood has not run this session.
	WizardSoftNotRun = "wizard_soft_not_run"
	// WizardSoftPass is session soft offline wizard residual PASS (≠ live dogfood · ≠ invent Edge Memory GA declared).
	WizardSoftPass = "soft_offline_wizard_session_pass"
	// WizardSoftFail is session soft offline wizard residual FAIL (≠ invent red product / full interactive wizard GA).
	WizardSoftFail = "soft_offline_wizard_session_fail"
)

// SetWizardSoftDogfoodSessionState records that soft offline wizard dogfood ran this session.
// pass is residual soft offline only — never invents Edge Memory GA declared / dual_write ON / live dogfood / SSO / auto-host.
func SetWizardSoftDogfoodSessionState(pass bool) {
	wizardSoftDogfoodSession.mu.Lock()
	defer wizardSoftDogfoodSession.mu.Unlock()
	wizardSoftDogfoodSession.ran = true
	wizardSoftDogfoodSession.pass = pass
}

// GetWizardSoftDogfoodSessionState returns whether soft wizard dogfood ran this session and residual pass.
// Default: ran=false, pass=false → WizardSoftSessionLabel returns wizard_soft_not_run.
func GetWizardSoftDogfoodSessionState() (ran bool, pass bool) {
	wizardSoftDogfoodSession.mu.Lock()
	defer wizardSoftDogfoodSession.mu.Unlock()
	return wizardSoftDogfoodSession.ran, wizardSoftDogfoodSession.pass
}

// WizardSoftSessionLabel returns the honest session soft dogfood vocabulary token:
// wizard_soft_not_run | soft_offline_wizard_session_pass | soft_offline_wizard_session_fail.
// Session soft marker ≠ live dogfood · ≠ invent Edge Memory GA declared · residual PASS ≠ live dogfood.
func WizardSoftSessionLabel() string {
	ran, pass := GetWizardSoftDogfoodSessionState()
	if !ran {
		return WizardSoftNotRun
	}
	if pass {
		return WizardSoftPass
	}
	return WizardSoftFail
}

// ResetWizardSoftDogfoodSessionState clears the session marker (tests only).
func ResetWizardSoftDogfoodSessionState() {
	wizardSoftDogfoodSession.mu.Lock()
	defer wizardSoftDogfoodSession.mu.Unlock()
	wizardSoftDogfoodSession.ran = false
	wizardSoftDogfoodSession.pass = false
}

// wizardSoftDogfoodNeedles are residual-honesty needles required for soft offline pass.
// Offline-only: board content + first-run wizard residual honesty locks. Never dials MCP · never starts host.
// Soft offline ≠ invent Connected · ≠ live dogfood · residual PASS ≠ invent Edge Memory GA declared ·
// residual PASS ≠ live dogfood · dual_write OFF · no invent TUI portal SSO · host not auto · E10 Open.
var wizardSoftDogfoodNeedles = []string{
	// Board identity + Wave C residual
	"onboard next wizard lane",
	"Wave C",
	"first-run wizard residual",
	"free eng s1570",
	// All 7 stages by name
	"1. Signup",
	"2. Download TUI",
	"3. TUI auth/keys",
	"4. Setup",
	"5. Connectors",
	"6. Local store",
	"7. Analyze",
	// Next-action companions
	"/onboard next setup",
	"/onboard next portal-hitl",
	"/onboard next e4",
	"/onboard next journey",
	// Policy / GA locks
	"dual_write OFF",
	"not Memory GA",
	"Edge Memory GA candidacy only",
	"E10 Open",
	// Connect / install honesty
	"portal HITL when connect",
	"agent MCP cannot write installs",
	"catalog ≠ Connected",
	// Residual honesty
	"residual PASS ≠ invent Edge Memory GA declared",
	"no invent TUI portal SSO",
	"host not auto",
	// Soft / free-floor
	"session soft ≠ live dogfood",
	"free-floor peer s1572+",
}

// RunFirstRunWizardSoftDogfood validates residual honesty of the first-run wizard path offline (s1570 Wave C).
// Checks wizard board needles + first-run wizard residual honesty locks as static strings.
// Never dials MCP · never starts host · never invents Edge Memory GA declared · dual_write ON ·
// TUI portal SSO · auto host · forever-green interactive wizard UX · live dogfood.
// Sets session soft marker (pass/fail). Returns residual-honest operator output.
func RunFirstRunWizardSoftDogfood() string {
	board := MeshAgentOnboardingNextWizardLane()
	var missing []string
	for _, want := range wizardSoftDogfoodNeedles {
		if !strings.Contains(board, want) {
			missing = append(missing, want)
		}
	}
	pass := len(missing) == 0
	SetWizardSoftDogfoodSessionState(pass)
	label := WizardSoftSessionLabel()

	var b strings.Builder
	b.WriteString("mesh onboard next first-run wizard soft offline dogfood (residual-honest · s1570 Wave C · no MCP dial · never start host · not live dogfood):\n")
	b.WriteString("  Path: soft offline residual check of wizard board honesty + guided first-run residual map (7 stages)\n")
	b.WriteString("  · never dial MCP · never start host · residual PASS ≠ invent Edge Memory GA declared · dual_write stays OFF · E10 Open\n")
	b.WriteString("  · no invent TUI portal SSO · host not auto · residual PASS ≠ invent full interactive auto wizard · session soft ≠ live dogfood · soft offline ≠ invent Connected\n")
	b.WriteString("  · residual PASS ≠ invent Edge Memory GA declared · Edge Memory GA candidacy only · dual_write OFF · not Memory GA · free eng s1570\n")
	b.WriteString("\n")
	if pass {
		b.WriteString("  result: PASS (soft offline residual only)\n")
		b.WriteString(fmt.Sprintf("  checked: %d honesty needles + first-run wizard residual path shapes present on wizard board\n", len(wizardSoftDogfoodNeedles)))
	} else {
		b.WriteString("  result: FAIL (soft offline residual only · ≠ invent red product · residual PASS ≠ invent Edge Memory GA declared)\n")
		b.WriteString(fmt.Sprintf("  missing needles (%d):\n", len(missing)))
		for _, m := range missing {
			b.WriteString(fmt.Sprintf("    - %q\n", m))
		}
	}
	b.WriteString("\n")
	b.WriteString("  First-run wizard residual path checked (static offline):\n")
	b.WriteString("    · Wave C · first-run wizard residual · free eng s1570\n")
	b.WriteString("    · 1. Signup · 2. Download TUI · 3. TUI auth/keys · 4. Setup · 5. Connectors · 6. Local store · 7. Analyze\n")
	b.WriteString("    · companions: /onboard next setup · /onboard next portal-hitl · /onboard next e4 · /onboard next journey\n")
	b.WriteString("  Honesty locks checked: dual_write OFF · not Memory GA · Edge Memory GA candidacy only · E10 Open\n")
	b.WriteString("    · portal HITL when connect · agent MCP cannot write installs · catalog ≠ Connected\n")
	b.WriteString("    · residual PASS ≠ invent Edge Memory GA declared · no invent TUI portal SSO · host not auto\n")
	b.WriteString("    · residual PASS ≠ live dogfood · session soft ≠ live dogfood · soft offline ≠ invent Connected\n")
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  session marker: %s\n", label))
	b.WriteString("  note: soft offline ≠ invent Connected · residual PASS ≠ live dogfood · session soft ≠ live dogfood · residual PASS ≠ invent Edge Memory GA declared\n")
	b.WriteString("  note: residual PASS ≠ invent full interactive auto wizard · residual PASS ≠ live dogfood · free eng s1570\n")
	b.WriteString("  tip: re-run /onboard next status then /onboard next export — companion journey (Wave B) + setup · portal-hitl · e4 remain independent\n")
	b.WriteString("  slash: /onboard next wizard dogfood (aliases soft|samples|offline|wizard-soft) · bare /onboard next wizard stays board\n")
	b.WriteString("  companion: /onboard next journey · /onboard next setup · /onboard next portal-hitl · /onboard next e4 · /onboard next human-gates · docs/architecture/edge-user-journey.md\n")
	b.WriteString("\n")
	b.WriteString("Locks: dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · portal HITL when connect · agent MCP cannot write installs · catalog ≠ Connected · no invent TUI portal SSO · host not auto · residual PASS ≠ invent full interactive auto wizard · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open · never invent install green / Connected / INSTALL_STORE APPLY · dual_write stays OFF (never invent primary ON) · E10 stays Open (never invent closed) · soft offline ≠ invent Connected · session soft ≠ live dogfood · free eng s1570 · free-floor peer s1572+ mention only")
	return b.String()
}
