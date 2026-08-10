package agent

import (
	"fmt"
	"strings"
	"sync"
)

// Still-human APPLY soft offline dogfood session marker (s1574 Wave C continuum residual).
// Separate from wizard soft (s1570), E4 soft (s1566), portal HITL soft (s1562), and agentic list/plan soft (s1422) — independent SSOT.
// Session-only: default still_human_soft_not_run. Soft offline pass/fail ≠ invent human-gate green · ≠ live APPLY ·
// ≠ Edge Memory GA declared · ≠ E10 closed · ≠ dual_write ON · ≠ live dogfood · soft offline ≠ invent Connected.
//
// Lives in agent so AionAgentHumanGatesHonestyBoard + TUI slash share state
// without import cycles (agent cannot import tui).

var stillHumanSoftDogfoodSession struct {
	mu   sync.Mutex
	ran  bool
	pass bool // residual soft offline pass only
}

// Soft dogfood session state labels for still-human APPLY residual (honest vocabulary only).
const (
	// StillHumanSoftNotRun is the default when /onboard next human-gates dogfood has not run this session.
	StillHumanSoftNotRun = "still_human_soft_not_run"
	// StillHumanSoftPass is session soft offline still-human residual PASS (≠ live dogfood · ≠ invent human-gate green · ≠ live APPLY).
	StillHumanSoftPass = "soft_offline_still_human_session_pass"
	// StillHumanSoftFail is session soft offline still-human residual FAIL (≠ invent red product / human-gate green invent).
	StillHumanSoftFail = "soft_offline_still_human_session_fail"
)

// SetStillHumanSoftDogfoodSessionState records that soft offline still-human dogfood ran this session.
// pass is residual soft offline only — never invents human-gate green / live APPLY / Edge Memory GA declared / dual_write ON / live dogfood / E10 closed.
func SetStillHumanSoftDogfoodSessionState(pass bool) {
	stillHumanSoftDogfoodSession.mu.Lock()
	defer stillHumanSoftDogfoodSession.mu.Unlock()
	stillHumanSoftDogfoodSession.ran = true
	stillHumanSoftDogfoodSession.pass = pass
}

// GetStillHumanSoftDogfoodSessionState returns whether soft still-human dogfood ran this session and residual pass.
// Default: ran=false, pass=false → StillHumanSoftSessionLabel returns still_human_soft_not_run.
func GetStillHumanSoftDogfoodSessionState() (ran bool, pass bool) {
	stillHumanSoftDogfoodSession.mu.Lock()
	defer stillHumanSoftDogfoodSession.mu.Unlock()
	return stillHumanSoftDogfoodSession.ran, stillHumanSoftDogfoodSession.pass
}

// StillHumanSoftSessionLabel returns the honest session soft dogfood vocabulary token:
// still_human_soft_not_run | soft_offline_still_human_session_pass | soft_offline_still_human_session_fail.
// Session soft marker ≠ live dogfood · ≠ invent human-gate green · residual PASS ≠ live APPLY.
func StillHumanSoftSessionLabel() string {
	ran, pass := GetStillHumanSoftDogfoodSessionState()
	if !ran {
		return StillHumanSoftNotRun
	}
	if pass {
		return StillHumanSoftPass
	}
	return StillHumanSoftFail
}

// ResetStillHumanSoftDogfoodSessionState clears the session marker (tests only).
func ResetStillHumanSoftDogfoodSessionState() {
	stillHumanSoftDogfoodSession.mu.Lock()
	defer stillHumanSoftDogfoodSession.mu.Unlock()
	stillHumanSoftDogfoodSession.ran = false
	stillHumanSoftDogfoodSession.pass = false
}

// stillHumanSoftDogfoodNeedles are residual-honesty needles required for soft offline pass.
// Offline-only: human-gates board content + still-human APPLY residual honesty locks. Never dials MCP · never starts host.
// Soft offline ≠ invent Connected · ≠ live dogfood · residual PASS ≠ invent Edge Memory GA declared ·
// residual PASS ≠ live dogfood · PASS ≠ live APPLY · PASS ≠ invent human-gate green · dual_write OFF · E10 Open.
var stillHumanSoftDogfoodNeedles = []string{
	// Board identity + s1574 Wave C continuum residual
	"human-gates honesty board",
	"still-human APPLY",
	"Wave C continuum",
	"free eng s1574",
	// Open boxes honesty
	"open boxes stay open",
	"PASS ≠ live APPLY",
	"PASS ≠ invent human-gate green",
	// edge-first / punted inventory
	"edge-first",
	"knowledge multi-tenant punted",
	"Slack HMAC punted",
	// policy / connect
	"portal HITL when connect",
	"book-demo OFF",
	"leave ON_SIGNAL unset",
	// GA / dual_write locks
	"dual_write OFF",
	"not Memory GA",
	"Edge Memory GA candidacy only",
	"E10 Open",
	"residual PASS ≠ invent Edge Memory GA declared",
	// Soft / free-floor
	"session soft ≠ live dogfood",
	"free-floor peer s1576+",
	// Companion surfaces
	"/onboard next human-gates",
	"/onboard next wizard",
	// Inventory residual reaffirm
	"Stripe",
	"H1/H2",
}

// RunStillHumanApplySoftDogfood validates residual honesty of still-human APPLY boxes offline (s1574).
// Checks human-gates board needles + still-human APPLY residual honesty locks as static strings.
// Never dials MCP · never starts host · never invents human-gate green · live APPLY · Edge Memory GA declared ·
// dual_write ON · E10 closed · live dogfood.
// Sets session soft marker (pass/fail). Returns residual-honest operator output.
func RunStillHumanApplySoftDogfood() string {
	board := AionAgentHumanGatesHonestyBoard()
	var missing []string
	for _, want := range stillHumanSoftDogfoodNeedles {
		if !strings.Contains(board, want) {
			missing = append(missing, want)
		}
	}
	pass := len(missing) == 0
	SetStillHumanSoftDogfoodSessionState(pass)
	label := StillHumanSoftSessionLabel()

	var b strings.Builder
	b.WriteString("aion onboard next still-human APPLY soft offline dogfood (residual-honest · s1574 Wave C continuum · no MCP dial · never start host · not live dogfood):\n")
	b.WriteString("  Path: soft offline residual check of human-gates board honesty + still-human APPLY open inventory after Wave A–C continuum\n")
	b.WriteString("  · never dial MCP · never start host · residual PASS ≠ invent Edge Memory GA declared · dual_write stays OFF · E10 Open\n")
	b.WriteString("  · PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open · residual PASS ≠ live dogfood · session soft ≠ live dogfood · soft offline ≠ invent Connected\n")
	b.WriteString("  · residual PASS ≠ invent Edge Memory GA declared · Edge Memory GA candidacy only · dual_write OFF · not Memory GA · free eng s1574\n")
	b.WriteString("\n")
	if pass {
		b.WriteString("  result: PASS (soft offline residual only)\n")
		b.WriteString(fmt.Sprintf("  checked: %d honesty needles + still-human APPLY residual path shapes present on human-gates board\n", len(stillHumanSoftDogfoodNeedles)))
	} else {
		b.WriteString("  result: FAIL (soft offline residual only · ≠ invent red product · residual PASS ≠ invent Edge Memory GA declared · PASS ≠ invent human-gate green)\n")
		b.WriteString(fmt.Sprintf("  missing needles (%d):\n", len(missing)))
		for _, m := range missing {
			b.WriteString(fmt.Sprintf("    - %q\n", m))
		}
	}
	b.WriteString("\n")
	b.WriteString("  Still-human APPLY residual path checked (static offline):\n")
	b.WriteString("    · still-human APPLY · open boxes stay open · PASS ≠ live APPLY · PASS ≠ invent human-gate green\n")
	b.WriteString("    · edge-first · knowledge multi-tenant punted · Slack HMAC punted · Stripe residual · H1/H2 residual\n")
	b.WriteString("    · portal HITL when connect · book-demo OFF · leave ON_SIGNAL unset\n")
	b.WriteString("  Honesty locks checked: dual_write OFF · not Memory GA · Edge Memory GA candidacy only · E10 Open\n")
	b.WriteString("    · residual PASS ≠ invent Edge Memory GA declared · residual PASS ≠ invent Edge Memory GA declared\n")
	b.WriteString("    · residual PASS ≠ live dogfood · session soft ≠ live dogfood · soft offline ≠ invent Connected\n")
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  session marker: %s\n", label))
	b.WriteString("  note: soft offline ≠ invent Connected · residual PASS ≠ live dogfood · session soft ≠ live dogfood · residual PASS ≠ invent Edge Memory GA declared\n")
	b.WriteString("  note: PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open · free eng s1574\n")
	b.WriteString("  tip: re-run /onboard next status then /onboard next export — companion wizard (Wave C) + human-gates board remain independent\n")
	b.WriteString("  slash: /onboard next human-gates dogfood (aliases soft|samples|offline|still-human-soft|apply-soft) · bare /onboard next human-gates stays board\n")
	b.WriteString("  companion: /onboard next human-gates · /onboard next wizard · /onboard next journey · /onboard next setup · /onboard next portal-hitl · /onboard next e4 · docs/architecture/edge-user-journey.md\n")
	b.WriteString("\n")
	b.WriteString("Locks: dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · portal HITL when connect · agent MCP cannot write installs · catalog ≠ Connected · PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open · never invent install green / Connected / INSTALL_STORE APPLY · dual_write stays OFF (never invent primary ON) · E10 stays Open (never invent closed) · soft offline ≠ invent Connected · session soft ≠ live dogfood · free eng s1574 · free-floor peer s1576+ mention only")
	return b.String()
}
