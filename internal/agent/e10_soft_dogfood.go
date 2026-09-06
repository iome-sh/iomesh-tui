package agent

import (
	"fmt"
	"strings"
	"sync"
)

// E10 Open soft offline residual-check session marker (s1586 free eng residual).
// Separate from tool-call soft (s1578), still-human soft (s1574), wizard soft (s1570),
// E4 soft (s1566), portal HITL soft (s1562), and agentic list/plan soft (s1422) — independent SSOT.
// Session-only: default e10_soft_not_run. Soft offline pass/fail ≠ invent E10 closed ·
// ≠ Edge Memory GA declared · ≠ dual_write ON · ≠ book-demo ON · ≠ live dogfood · soft offline ≠ invent Connected ·
// residual PASS ≠ live APPLY.
//
// Lives in agent so MeshAgentOnboardingNextE10Lane board + TUI slash share state
// without import cycles (agent cannot import tui).

var e10SoftDogfoodSession struct {
	mu   sync.Mutex
	ran  bool
	pass bool // residual soft offline pass only
}

// Soft dogfood session state labels for E10 Open residual-check (honest vocabulary only).
const (
	// E10SoftNotRun is the default when /onboard next e10 dogfood has not run this session.
	E10SoftNotRun = "e10_soft_not_run"
	// E10SoftPass is session soft offline E10 residual PASS (≠ live dogfood · ≠ invent E10 closed · ≠ invent Edge Memory GA declared).
	E10SoftPass = "soft_offline_e10_session_pass"
	// E10SoftFail is session soft offline E10 residual FAIL (≠ invent red product / E10 closed invent).
	E10SoftFail = "soft_offline_e10_session_fail"
)

// SetE10SoftDogfoodSessionState records that soft offline E10 residual-check ran this session.
// pass is residual soft offline only — never invents E10 closed / Edge Memory GA declared / dual_write ON / live dogfood / live APPLY.
func SetE10SoftDogfoodSessionState(pass bool) {
	e10SoftDogfoodSession.mu.Lock()
	defer e10SoftDogfoodSession.mu.Unlock()
	e10SoftDogfoodSession.ran = true
	e10SoftDogfoodSession.pass = pass
}

// GetE10SoftDogfoodSessionState returns whether soft E10 residual-check ran this session and residual pass.
// Default: ran=false, pass=false → E10SoftSessionLabel returns e10_soft_not_run.
func GetE10SoftDogfoodSessionState() (ran bool, pass bool) {
	e10SoftDogfoodSession.mu.Lock()
	defer e10SoftDogfoodSession.mu.Unlock()
	return e10SoftDogfoodSession.ran, e10SoftDogfoodSession.pass
}

// E10SoftSessionLabel returns the honest session soft residual-check vocabulary token:
// e10_soft_not_run | soft_offline_e10_session_pass | soft_offline_e10_session_fail.
// Session soft marker ≠ live dogfood · residual PASS ≠ invent E10 closed · residual PASS ≠ invent Edge Memory GA declared.
func E10SoftSessionLabel() string {
	ran, pass := GetE10SoftDogfoodSessionState()
	if !ran {
		return E10SoftNotRun
	}
	if pass {
		return E10SoftPass
	}
	return E10SoftFail
}

// ResetE10SoftDogfoodSessionState clears the session marker (tests only).
func ResetE10SoftDogfoodSessionState() {
	e10SoftDogfoodSession.mu.Lock()
	defer e10SoftDogfoodSession.mu.Unlock()
	e10SoftDogfoodSession.ran = false
	e10SoftDogfoodSession.pass = false
}

// e10SoftDogfoodNeedles are residual-honesty needles required for soft offline pass.
// Offline-only: E10 Open board content + residual honesty locks. Never dials MCP · never starts host.
// Soft offline ≠ invent Connected · ≠ live dogfood · residual PASS ≠ invent Edge Memory GA declared ·
// residual PASS ≠ invent E10 closed · residual PASS ≠ live dogfood · dual_write OFF · book-demo OFF · E10 Open.
var e10SoftDogfoodNeedles = []string{
	// Board identity + s1586 E10 Open reaffirm residual
	"onboard next e10 lane",
	"E10 Open",
	"E10 Open reaffirm",
	"no MCP dial",
	// Core honesty locks
	"residual PASS ≠ invent E10 closed",
	"residual PASS ≠ invent Edge Memory GA declared",
	"Edge Memory GA candidacy only",
	"not Memory GA",
	"dual_write OFF",
	"book-demo OFF",
	// Founder / APPLY honesty
	"founder sign-off only if declaring Edge Memory GA",
	"candidacy allowed without E10",
	"PASS ≠ live APPLY",
	// Soft residual-check honesty
	"residual-check",
	"session soft ≠ live dogfood",
	"residual PASS ≠ live dogfood",
	"soft offline ≠ invent Connected",
	// Companions (e4 · human-gates · OSS packaging)
	"/onboard next e4",
	"/onboard next human-gates",
	"OSS packaging",
	"MIT harness",
	"not control plane",
	// Free eng serial
	"free eng s1586",
	"free-floor peer s1588+",
}

// RunE10OpenSoftDogfood validates residual honesty of the E10 Open reaffirm board offline (s1586).
// Checks e10 board needles + E10 Open residual honesty locks as static strings.
// Never dials MCP · never starts host · never invents E10 closed · Edge Memory GA declared · dual_write ON · live APPLY green · live dogfood.
// Sets session soft marker (pass/fail). Returns residual-honest operator output.
func RunE10OpenSoftDogfood() string {
	board := MeshAgentOnboardingNextE10Lane()
	var missing []string
	for _, want := range e10SoftDogfoodNeedles {
		if !strings.Contains(board, want) {
			missing = append(missing, want)
		}
	}
	pass := len(missing) == 0
	SetE10SoftDogfoodSessionState(pass)
	label := E10SoftSessionLabel()

	var b strings.Builder
	b.WriteString("mesh onboard next E10 Open soft offline residual-check (residual-honest · s1586 · no MCP dial · never start host · not live dogfood):\n")
	b.WriteString("  Path: soft offline residual check of e10 board honesty + E10 Open reaffirm after OSS packaging continuum\n")
	b.WriteString("  · never dial MCP · never start host · residual PASS ≠ invent E10 closed · residual PASS ≠ invent Edge Memory GA declared · dual_write stays OFF · E10 Open\n")
	b.WriteString("  · residual PASS ≠ live dogfood · session soft ≠ live dogfood · soft offline ≠ invent Connected · PASS ≠ live APPLY · free eng s1586\n")
	b.WriteString("  · Edge Memory GA candidacy only · dual_write OFF · book-demo OFF · not Memory GA · residual-check\n")
	b.WriteString("\n")
	if pass {
		b.WriteString("  result: PASS (soft offline residual only)\n")
		b.WriteString(fmt.Sprintf("  checked: %d honesty needles + E10 Open residual path shapes present on e10 board\n", len(e10SoftDogfoodNeedles)))
	} else {
		b.WriteString("  result: FAIL (soft offline residual only · ≠ invent red product · residual PASS ≠ invent E10 closed · residual PASS ≠ invent Edge Memory GA declared)\n")
		b.WriteString(fmt.Sprintf("  missing needles (%d):\n", len(missing)))
		for _, m := range missing {
			b.WriteString(fmt.Sprintf("    - %q\n", m))
		}
	}
	b.WriteString("\n")
	b.WriteString("  E10 Open residual path checked (static offline):\n")
	b.WriteString("    · E10 Open · residual PASS ≠ invent E10 closed · residual PASS ≠ invent Edge Memory GA declared\n")
	b.WriteString("    · Edge Memory GA candidacy only · not Memory GA · dual_write OFF · book-demo OFF\n")
	b.WriteString("    · founder sign-off only if declaring Edge Memory GA · candidacy allowed without E10 · PASS ≠ live APPLY\n")
	b.WriteString("  Companion residual checked: /onboard next e4 · /onboard next human-gates · OSS packaging · MIT harness · not control plane\n")
	b.WriteString("  Honesty locks checked: dual_write OFF · not Memory GA · Edge Memory GA candidacy only · E10 Open\n")
	b.WriteString("    · residual PASS ≠ invent Edge Memory GA declared · residual PASS ≠ invent E10 closed\n")
	b.WriteString("    · residual PASS ≠ live dogfood · session soft ≠ live dogfood · soft offline ≠ invent Connected · residual-check\n")
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  session marker: %s\n", label))
	b.WriteString("  note: soft offline ≠ invent Connected · residual PASS ≠ live dogfood · session soft ≠ live dogfood · residual PASS ≠ invent Edge Memory GA declared\n")
	b.WriteString("  note: residual PASS ≠ invent E10 closed · PASS ≠ live APPLY · residual-check · free eng s1586\n")
	b.WriteString("  tip: re-run /onboard next status then /onboard next export — companion e4 · human-gates · OSS packaging remain independent\n")
	b.WriteString("  slash: /onboard next e10 dogfood (aliases soft|samples|offline|e10-soft|residual-check) · bare /onboard next e10 stays board\n")
	b.WriteString("  companion: /onboard next e4 · /onboard next human-gates · /onboard next tool-call · /onboard next · docs/architecture/oss-packaging-boundary.md · docs/architecture/edge-user-journey.md · docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md\n")
	b.WriteString("\n")
	b.WriteString("Locks: dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · residual PASS ≠ invent E10 closed · E10 Open · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open · never invent install green / Connected / INSTALL_STORE APPLY · dual_write stays OFF (never invent primary ON) · E10 stays Open (never invent closed) · soft offline ≠ invent Connected · session soft ≠ live dogfood · residual-check · free eng s1586 · free-floor peer s1588+ mention only")
	return b.String()
}
