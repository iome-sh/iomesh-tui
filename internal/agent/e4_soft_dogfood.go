package agent

import (
	"fmt"
	"strings"
	"sync"
)

// E4 client-attach soft offline dogfood session marker (s1566).
// Separate from portal HITL soft (s1562) and agentic list/plan soft (s1422) — independent SSOT.
// Session-only: default e4_soft_not_run. Soft offline pass/fail ≠ invent Edge Memory GA declared ·
// ≠ forever-green product dogfood · ≠ E10 closed · ≠ dual_write ON · ≠ live dogfood · soft offline ≠ invent Connected.
//
// Lives in agent so MeshAgentOnboardingNextE4Lane board + TUI slash share state
// without import cycles (agent cannot import tui).

var e4SoftDogfoodSession struct {
	mu   sync.Mutex
	ran  bool
	pass bool // residual soft offline pass only
}

// Soft dogfood session state labels for E4 client-attach (honest vocabulary only).
const (
	// E4SoftNotRun is the default when /onboard next e4 dogfood has not run this session.
	E4SoftNotRun = "e4_soft_not_run"
	// E4SoftPass is session soft offline E4 residual PASS (≠ live dogfood · ≠ invent Edge Memory GA declared).
	E4SoftPass = "soft_offline_e4_session_pass"
	// E4SoftFail is session soft offline E4 residual FAIL (≠ invent red product / forever-green dogfood).
	E4SoftFail = "soft_offline_e4_session_fail"
)

// SetE4SoftDogfoodSessionState records that soft offline E4 dogfood ran this session.
// pass is residual soft offline only — never invents Edge Memory GA declared / dual_write ON / live dogfood / E10 closed.
func SetE4SoftDogfoodSessionState(pass bool) {
	e4SoftDogfoodSession.mu.Lock()
	defer e4SoftDogfoodSession.mu.Unlock()
	e4SoftDogfoodSession.ran = true
	e4SoftDogfoodSession.pass = pass
}

// GetE4SoftDogfoodSessionState returns whether soft E4 dogfood ran this session and residual pass.
// Default: ran=false, pass=false → E4SoftSessionLabel returns e4_soft_not_run.
func GetE4SoftDogfoodSessionState() (ran bool, pass bool) {
	e4SoftDogfoodSession.mu.Lock()
	defer e4SoftDogfoodSession.mu.Unlock()
	return e4SoftDogfoodSession.ran, e4SoftDogfoodSession.pass
}

// E4SoftSessionLabel returns the honest session soft dogfood vocabulary token:
// e4_soft_not_run | soft_offline_e4_session_pass | soft_offline_e4_session_fail.
// Session soft marker ≠ live dogfood · ≠ invent Edge Memory GA declared · residual PASS ≠ live dogfood.
func E4SoftSessionLabel() string {
	ran, pass := GetE4SoftDogfoodSessionState()
	if !ran {
		return E4SoftNotRun
	}
	if pass {
		return E4SoftPass
	}
	return E4SoftFail
}

// ResetE4SoftDogfoodSessionState clears the session marker (tests only).
func ResetE4SoftDogfoodSessionState() {
	e4SoftDogfoodSession.mu.Lock()
	defer e4SoftDogfoodSession.mu.Unlock()
	e4SoftDogfoodSession.ran = false
	e4SoftDogfoodSession.pass = false
}

// e4SoftDogfoodNeedles are residual-honesty needles required for soft offline pass.
// Offline-only: board content + E4 attach path honesty locks. Never dials MCP · never starts host.
// Soft offline ≠ invent Connected · ≠ live dogfood · residual PASS ≠ invent Edge Memory GA declared ·
// residual PASS ≠ live dogfood · tip ≠ invent forever-green product dogfood · E10 Open · dual_write OFF.
var e4SoftDogfoodNeedles = []string{
	// Board identity + journey stage 6 alignment
	"onboard next e4 lane",
	"journey stage 6",
	"E4 client attach",
	"no MCP dial",
	// Attach path + tools stamp residual
	"client attach",
	"tools=6",
	"iomesh mcp --connect",
	// Product host + local-primary
	"iomesh-memory-mcp",
	"local-primary",
	// Policy / GA locks
	"dual_write OFF",
	"not Memory GA",
	"Edge Memory GA candidacy only",
	"residual PASS ≠ invent Edge Memory GA declared",
	"E10 Open",
	// Soft / residual honesty
	"tip ≠ invent forever-green product dogfood",
	"residual PASS ≠ live dogfood",
	"session soft ≠ live dogfood",
	"soft offline ≠ invent Connected",
	// Evidence stamp
	"docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md",
	// Free eng serial
	"free eng s1566",
}

// RunE4SoftDogfood validates residual honesty of the E4 client-attach path offline (s1566).
// Checks e4 board needles + E4 attach path honesty locks as static strings.
// Never dials MCP · never starts host · never invents Edge Memory GA declared · dual_write ON · E10 closed · forever-green product dogfood.
// Sets session soft marker (pass/fail). Returns residual-honest operator output.
func RunE4SoftDogfood() string {
	board := MeshAgentOnboardingNextE4Lane()
	var missing []string
	for _, want := range e4SoftDogfoodNeedles {
		if !strings.Contains(board, want) {
			missing = append(missing, want)
		}
	}
	pass := len(missing) == 0
	SetE4SoftDogfoodSessionState(pass)
	label := E4SoftSessionLabel()

	var b strings.Builder
	b.WriteString("mesh onboard next E4 client-attach soft offline dogfood (residual-honest · s1566 · no MCP dial · never start host · not live dogfood):\n")
	b.WriteString("  Path: soft offline residual check of e4 board honesty + E4 client attach path (journey stage 6 local store / MCP attach)\n")
	b.WriteString("  · never dial MCP · never start host · residual PASS ≠ invent Edge Memory GA declared · dual_write stays OFF · E10 Open\n")
	b.WriteString("  · tip ≠ invent forever-green product dogfood · residual PASS ≠ live dogfood · session soft ≠ live dogfood · soft offline ≠ invent Connected\n")
	b.WriteString("  · residual PASS ≠ invent Edge Memory GA declared · Edge Memory GA candidacy only · dual_write OFF · not Memory GA · local-primary\n")
	b.WriteString("\n")
	if pass {
		b.WriteString("  result: PASS (soft offline residual only)\n")
		b.WriteString(fmt.Sprintf("  checked: %d honesty needles + E4 client attach path shapes present on e4 board\n", len(e4SoftDogfoodNeedles)))
	} else {
		b.WriteString("  result: FAIL (soft offline residual only · ≠ invent red product · residual PASS ≠ invent Edge Memory GA declared)\n")
		b.WriteString(fmt.Sprintf("  missing needles (%d):\n", len(missing)))
		for _, m := range missing {
			b.WriteString(fmt.Sprintf("    - %q\n", m))
		}
	}
	b.WriteString("\n")
	b.WriteString("  E4 client attach path checked (static offline):\n")
	b.WriteString("    · E4 client attach · tools=6 · iomesh mcp --connect\n")
	b.WriteString("    · iomesh-memory-mcp · local-primary\n")
	b.WriteString("    · docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md\n")
	b.WriteString("  Honesty locks checked: dual_write OFF · not Memory GA · Edge Memory GA candidacy only\n")
	b.WriteString("    · residual PASS ≠ invent Edge Memory GA declared · E10 Open · tip ≠ invent forever-green product dogfood\n")
	b.WriteString("    · residual PASS ≠ live dogfood · session soft ≠ live dogfood · soft offline ≠ invent Connected\n")
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  session marker: %s\n", label))
	b.WriteString("  note: soft offline ≠ invent Connected · residual PASS ≠ live dogfood · session soft ≠ live dogfood · residual PASS ≠ invent Edge Memory GA declared\n")
	b.WriteString("  note: residual PASS ≠ live dogfood · PASS ≠ live APPLY · tip ≠ invent forever-green product dogfood · free eng s1566\n")
	b.WriteString("  tip: re-run /onboard next status then /onboard next export — companion memory lane + journey stage 6 remain independent\n")
	b.WriteString("  slash: /onboard next e4 dogfood (aliases soft|samples|offline|e4-soft) · bare /onboard next e4 stays board\n")
	b.WriteString("  companion: /onboard next memory · /onboard next journey · /onboard next memory-pull · /memory status · docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md\n")
	b.WriteString("\n")
	b.WriteString("Locks: dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · tip ≠ invent forever-green product dogfood · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open · never invent install green / Connected / INSTALL_STORE APPLY · dual_write stays OFF (never invent primary ON) · E10 stays Open (never invent closed) · soft offline ≠ invent Connected · session soft ≠ live dogfood · free eng s1566 · free-floor peer s1568+ mention only")
	return b.String()
}
