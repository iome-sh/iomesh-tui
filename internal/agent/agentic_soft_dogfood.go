package agent

import (
	"fmt"
	"strings"
	"sync"
)

// Agentic list/plan soft offline dogfood session marker (s1422).
// Separate from agentplugins SoftDogfoodSession* (plugins lane) — independent SSOT.
// Session-only: default list_plan_soft_not_run. Soft offline pass/fail ≠ invent Connected ·
// ≠ live dogfood · ≠ invent install APPLY · portal HITL still · list_org fail-open ≠ empty-as-none.
//
// Lives in agent so MeshAgentOnboardingNextAgenticLane board + status/export + TUI slash
// share state without import cycles (agent cannot import tui).

var agenticListPlanSoftDogfoodSession struct {
	mu   sync.Mutex
	ran  bool
	pass bool // residual soft offline pass only
}

// Soft dogfood session state labels for agentic list/plan (honest vocabulary only).
const (
	// AgenticListPlanSoftNotRun is the default when /onboard next agentic dogfood has not run this session.
	AgenticListPlanSoftNotRun = "list_plan_soft_not_run"
	// AgenticListPlanSoftPass is session soft offline list/plan residual PASS (≠ live dogfood · ≠ invent Connected).
	AgenticListPlanSoftPass = "soft_offline_list_plan_session_pass"
	// AgenticListPlanSoftFail is session soft offline list/plan residual FAIL (≠ invent red product / Connected).
	AgenticListPlanSoftFail = "soft_offline_list_plan_session_fail"
)

// SetAgenticListPlanSoftDogfoodSessionState records that soft offline list/plan dogfood ran this session.
// pass is residual soft offline only — never invents Connected / install APPLY / live dogfood / dual-auth live.
func SetAgenticListPlanSoftDogfoodSessionState(pass bool) {
	agenticListPlanSoftDogfoodSession.mu.Lock()
	defer agenticListPlanSoftDogfoodSession.mu.Unlock()
	agenticListPlanSoftDogfoodSession.ran = true
	agenticListPlanSoftDogfoodSession.pass = pass
}

// GetAgenticListPlanSoftDogfoodSessionState returns whether soft list/plan dogfood ran this session and residual pass.
// Default: ran=false, pass=false → AgenticListPlanSoftSessionLabel returns list_plan_soft_not_run.
func GetAgenticListPlanSoftDogfoodSessionState() (ran bool, pass bool) {
	agenticListPlanSoftDogfoodSession.mu.Lock()
	defer agenticListPlanSoftDogfoodSession.mu.Unlock()
	return agenticListPlanSoftDogfoodSession.ran, agenticListPlanSoftDogfoodSession.pass
}

// AgenticListPlanSoftSessionLabel returns the honest session soft dogfood vocabulary token:
// list_plan_soft_not_run | soft_offline_list_plan_session_pass | soft_offline_list_plan_session_fail.
// Session soft marker ≠ live dogfood · ≠ invent Connected · portal HITL still · list_plan_not_connected.
func AgenticListPlanSoftSessionLabel() string {
	ran, pass := GetAgenticListPlanSoftDogfoodSessionState()
	if !ran {
		return AgenticListPlanSoftNotRun
	}
	if pass {
		return AgenticListPlanSoftPass
	}
	return AgenticListPlanSoftFail
}

// ResetAgenticListPlanSoftDogfoodSessionState clears the session marker (tests only).
func ResetAgenticListPlanSoftDogfoodSessionState() {
	agenticListPlanSoftDogfoodSession.mu.Lock()
	defer agenticListPlanSoftDogfoodSession.mu.Unlock()
	agenticListPlanSoftDogfoodSession.ran = false
	agenticListPlanSoftDogfoodSession.pass = false
}

// agenticListPlanSoftDogfoodNeedles are residual-honesty needles required for soft offline pass.
// Offline-only: board content + proven portal path shapes + honesty locks. Never dials MCP.
// Soft offline ≠ invent Connected · ≠ live dogfood · portal HITL still · list_org fail-open ≠ empty-as-none.
var agenticListPlanSoftDogfoodNeedles = []string{
	// Board identity + product plane
	"onboard next agentic lane",
	"product plane 3",
	"agentic integrations",
	"MCP list/plan residual-honest",
	"no MCP dial",
	// Proven portal path shapes (static strings only)
	"/integrations/{id}",
	"/integrations/add?template={id}",
	"/integrations",
	// Plan / HITL honesty
	"plan_connector_setup",
	"browser HITL only",
	"template= ≠ install APPLY",
	"deep_links = browser HITL only",
	// Org residual
	"list_org fail-open ≠ empty-as-none",
	// Install / Connected locks
	"agent MCP cannot write installs",
	"catalog ≠ Connected",
	"never invent Connected",
	// Honest residual vocab
	"path_ready",
	"residual_only",
	"portal_hitl_still",
	"list_plan_not_connected",
	// Policy locks
	"dual_write OFF",
	"book-demo OFF",
	"not Memory GA",
	"residual PASS ≠ live dogfood",
	"PASS ≠ live APPLY",
	// Companion portal surfaces
	"console.iome.sh/integrations",
	"console.iome.sh/settings/agent",
	// Dual-auth honesty
	"does not claim dual-auth live for list_org",
}

// RunAgenticListPlanSoftDogfood validates residual honesty of the agentic list/plan path offline (s1422).
// Checks agentic board needles + proven portal path shapes + honesty locks as static strings.
// Never dials MCP · never invents Connected · never invents install APPLY · never claims dual-auth live.
// Sets session soft marker (pass/fail). Returns residual-honest operator output.
func RunAgenticListPlanSoftDogfood() string {
	board := MeshAgentOnboardingNextAgenticLane()
	var missing []string
	for _, want := range agenticListPlanSoftDogfoodNeedles {
		if !strings.Contains(board, want) {
			missing = append(missing, want)
		}
	}
	pass := len(missing) == 0
	SetAgenticListPlanSoftDogfoodSessionState(pass)
	label := AgenticListPlanSoftSessionLabel()

	var b strings.Builder
	b.WriteString("mesh onboard next agentic list/plan soft offline dogfood (residual-honest · s1422 · no MCP dial · not live dogfood):\n")
	b.WriteString("  Path: soft offline residual check of agentic board honesty + proven portal path shapes\n")
	b.WriteString("  · never dial MCP · never invent Connected · never invent install APPLY · never claim dual-auth live\n")
	b.WriteString("  · soft offline list/plan ≠ live dogfood · portal HITL still · list_org fail-open ≠ empty-as-none\n")
	b.WriteString("  · session soft ≠ live dogfood · catalog ≠ Connected · template= ≠ install APPLY · agent MCP cannot write installs\n")
	b.WriteString("\n")
	if pass {
		b.WriteString("  result: PASS (soft offline residual only)\n")
		b.WriteString(fmt.Sprintf("  checked: %d honesty needles + proven portal path shapes present on agentic board\n", len(agenticListPlanSoftDogfoodNeedles)))
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
	b.WriteString("  Honesty locks checked: list_plan_not_connected · portal_hitl_still · path_ready · residual_only\n")
	b.WriteString("    · catalog ≠ Connected · template= ≠ install APPLY · agent MCP cannot write installs\n")
	b.WriteString("    · list_org fail-open ≠ empty-as-none · dual_write OFF · book-demo OFF · not Memory GA\n")
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  session marker: %s\n", label))
	b.WriteString("  note: soft offline list/plan ≠ live dogfood · ≠ invent Connected · portal HITL still · list_org fail-open ≠ empty-as-none\n")
	b.WriteString("  note: session soft ≠ live dogfood · residual PASS ≠ live dogfood · PASS ≠ live APPLY · board/export evidence ≠ invent Connected\n")
	b.WriteString("  tip: re-run /onboard next status then /onboard next export — session soft list/plan refreshes agentic lane (≠ invent Connected · ≠ live dogfood · portal HITL still)\n")
	b.WriteString("  slash: /onboard next agentic dogfood (aliases soft|samples|offline|list-plan-soft) · bare /onboard next agentic stays board\n")
	b.WriteString("  companion: /onboard portal mint/copy/probe · /integrations list|plan|status · /onboard next human-gates\n")
	b.WriteString("\n")
	b.WriteString("Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open · never invent install green / Connected / INSTALL_STORE APPLY · catalog ≠ Connected · list_org fail-open ≠ empty-as-none · plan deep links = browser HITL only · template= ≠ install APPLY · agent MCP cannot write installs · portal HITL · list_plan_not_connected · portal_hitl_still · path_ready · residual_only · soft offline ≠ live dogfood · session soft ≠ live dogfood · rates ~$88/$119 optional · board/export evidence ≠ invent Connected · does not claim dual-auth live for list_org")
	return b.String()
}
