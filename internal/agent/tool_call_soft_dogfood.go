package agent

import (
	"fmt"
	"strings"
	"sync"
)

// Deeper tool-call soft offline dogfood session marker (s1578 free eng residual).
// Separate from still-human soft (s1574), wizard soft (s1570), E4 soft (s1566), portal HITL soft (s1562),
// and agentic list/plan soft (s1422) — independent SSOT.
// Session-only: default tool_call_soft_not_run. Soft offline pass/fail ≠ invent Edge Memory GA declared ·
// ≠ forever-green product dogfood · ≠ E10 closed · ≠ dual_write ON · ≠ live dogfood · soft offline ≠ invent Connected.
// Depth residual after E4 attach tools=6: operator map for ingest → retrieve → list → as-of/status (soft offline only).
//
// Lives in agent so MeshAgentOnboardingNextToolCallLane board + TUI slash share state
// without import cycles (agent cannot import tui).

var toolCallSoftDogfoodSession struct {
	mu   sync.Mutex
	ran  bool
	pass bool // residual soft offline pass only
}

// Soft dogfood session state labels for deeper tool-call residual (honest vocabulary only).
const (
	// ToolCallSoftNotRun is the default when /onboard next tool-call dogfood has not run this session.
	ToolCallSoftNotRun = "tool_call_soft_not_run"
	// ToolCallSoftPass is session soft offline tool-call residual PASS (≠ live dogfood · ≠ invent Edge Memory GA declared).
	ToolCallSoftPass = "soft_offline_tool_call_session_pass"
	// ToolCallSoftFail is session soft offline tool-call residual FAIL (≠ invent red product / forever-green dogfood).
	ToolCallSoftFail = "soft_offline_tool_call_session_fail"
)

// SetToolCallSoftDogfoodSessionState records that soft offline tool-call dogfood ran this session.
// pass is residual soft offline only — never invents Edge Memory GA declared / dual_write ON / live dogfood / E10 closed.
func SetToolCallSoftDogfoodSessionState(pass bool) {
	toolCallSoftDogfoodSession.mu.Lock()
	defer toolCallSoftDogfoodSession.mu.Unlock()
	toolCallSoftDogfoodSession.ran = true
	toolCallSoftDogfoodSession.pass = pass
}

// GetToolCallSoftDogfoodSessionState returns whether soft tool-call dogfood ran this session and residual pass.
// Default: ran=false, pass=false → ToolCallSoftSessionLabel returns tool_call_soft_not_run.
func GetToolCallSoftDogfoodSessionState() (ran bool, pass bool) {
	toolCallSoftDogfoodSession.mu.Lock()
	defer toolCallSoftDogfoodSession.mu.Unlock()
	return toolCallSoftDogfoodSession.ran, toolCallSoftDogfoodSession.pass
}

// ToolCallSoftSessionLabel returns the honest session soft dogfood vocabulary token:
// tool_call_soft_not_run | soft_offline_tool_call_session_pass | soft_offline_tool_call_session_fail.
// Session soft marker ≠ live dogfood · ≠ invent Edge Memory GA declared · residual PASS ≠ live dogfood.
func ToolCallSoftSessionLabel() string {
	ran, pass := GetToolCallSoftDogfoodSessionState()
	if !ran {
		return ToolCallSoftNotRun
	}
	if pass {
		return ToolCallSoftPass
	}
	return ToolCallSoftFail
}

// ResetToolCallSoftDogfoodSessionState clears the session marker (tests only).
func ResetToolCallSoftDogfoodSessionState() {
	toolCallSoftDogfoodSession.mu.Lock()
	defer toolCallSoftDogfoodSession.mu.Unlock()
	toolCallSoftDogfoodSession.ran = false
	toolCallSoftDogfoodSession.pass = false
}

// toolCallSoftDogfoodNeedles are residual-honesty needles required for soft offline pass.
// Offline-only: tool-call board content + deeper tool path honesty locks. Never dials MCP · never starts host.
// Soft offline ≠ invent Connected · ≠ live dogfood · residual PASS ≠ invent Edge Memory GA declared ·
// residual PASS ≠ live dogfood · tip ≠ invent forever-green product dogfood · E10 Open · dual_write OFF.
var toolCallSoftDogfoodNeedles = []string{
	// Board identity + depth after E4 attach
	"onboard next tool-call lane",
	"deeper tool-call residual",
	"journey stage 6/7",
	"no MCP dial",
	// Tool path honesty (ingest → retrieve → list → as-of)
	"memory_ingest_turn",
	"memory_retrieve",
	"memory_search_semantic",
	"memory_list",
	"memory_compact_status",
	"memory_facts_as_of",
	// Companion E4 attach stamp residual (s1508/s1566)
	"/onboard next e4",
	"tools=6",
	"iomesh mcp --connect",
	"s1508",
	"s1566",
	// Product host + Partial stamp residual
	"iomesh-memory-mcp",
	"Partial→client-attach-evidence",
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
	// Free eng serial
	"free eng s1578",
	"free-floor peer s1580+",
}

// RunDeeperToolCallSoftDogfood validates residual honesty of the deeper tool-call path offline (s1578).
// Checks tool-call board needles + tool path honesty locks as static strings.
// Never dials MCP · never starts host · never invents Edge Memory GA declared · dual_write ON · E10 closed · forever-green product dogfood.
// Sets session soft marker (pass/fail). Returns residual-honest operator output.
func RunDeeperToolCallSoftDogfood() string {
	board := MeshAgentOnboardingNextToolCallLane()
	var missing []string
	for _, want := range toolCallSoftDogfoodNeedles {
		if !strings.Contains(board, want) {
			missing = append(missing, want)
		}
	}
	pass := len(missing) == 0
	SetToolCallSoftDogfoodSessionState(pass)
	label := ToolCallSoftSessionLabel()

	var b strings.Builder
	b.WriteString("mesh onboard next deeper tool-call soft offline dogfood (residual-honest · s1578 · no MCP dial · never start host · not live dogfood):\n")
	b.WriteString("  Path: soft offline residual check of tool-call board honesty + deeper tool path after E4 attach (ingest→retrieve→list→as-of)\n")
	b.WriteString("  · never dial MCP · never start host · residual PASS ≠ invent Edge Memory GA declared · dual_write stays OFF · E10 Open\n")
	b.WriteString("  · tip ≠ invent forever-green product dogfood · residual PASS ≠ live dogfood · session soft ≠ live dogfood · soft offline ≠ invent Connected\n")
	b.WriteString("  · residual PASS ≠ invent Edge Memory GA declared · Edge Memory GA candidacy only · dual_write OFF · not Memory GA · free eng s1578\n")
	b.WriteString("\n")
	if pass {
		b.WriteString("  result: PASS (soft offline residual only)\n")
		b.WriteString(fmt.Sprintf("  checked: %d honesty needles + deeper tool-call path shapes present on tool-call board\n", len(toolCallSoftDogfoodNeedles)))
	} else {
		b.WriteString("  result: FAIL (soft offline residual only · ≠ invent red product · residual PASS ≠ invent Edge Memory GA declared)\n")
		b.WriteString(fmt.Sprintf("  missing needles (%d):\n", len(missing)))
		for _, m := range missing {
			b.WriteString(fmt.Sprintf("    - %q\n", m))
		}
	}
	b.WriteString("\n")
	b.WriteString("  Deeper tool-call residual path checked (static offline):\n")
	b.WriteString("    · memory_ingest_turn · memory_retrieve · memory_search_semantic\n")
	b.WriteString("    · memory_list · memory_compact_status · memory_facts_as_of\n")
	b.WriteString("    · companion /onboard next e4 · tools=6 · iomesh mcp --connect (s1508/s1566 attach stamp residual)\n")
	b.WriteString("    · Partial→client-attach-evidence · deeper tool-call residual candidacy (not forever-green full product dogfood)\n")
	b.WriteString("  Honesty locks checked: dual_write OFF · not Memory GA · Edge Memory GA candidacy only\n")
	b.WriteString("    · residual PASS ≠ invent Edge Memory GA declared · E10 Open · tip ≠ invent forever-green product dogfood\n")
	b.WriteString("    · residual PASS ≠ live dogfood · session soft ≠ live dogfood · soft offline ≠ invent Connected\n")
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  session marker: %s\n", label))
	b.WriteString("  note: soft offline ≠ invent Connected · residual PASS ≠ live dogfood · session soft ≠ live dogfood · residual PASS ≠ invent Edge Memory GA declared\n")
	b.WriteString("  note: residual PASS ≠ live dogfood · tip ≠ invent forever-green product dogfood · free eng s1578\n")
	b.WriteString("  tip: re-run /onboard next status then /onboard next export — companion e4 lane + memory lane + journey stage 6/7 remain independent\n")
	b.WriteString("  slash: /onboard next tool-call dogfood (aliases soft|samples|offline|tool-call-soft) · bare /onboard next tool-call stays board\n")
	b.WriteString("  companion: /onboard next e4 · /onboard next memory · /onboard next journey · /memory status · docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md\n")
	b.WriteString("\n")
	b.WriteString("Locks: dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · tip ≠ invent forever-green product dogfood · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open · never invent install green / Connected / INSTALL_STORE APPLY · dual_write stays OFF (never invent primary ON) · E10 stays Open (never invent closed) · soft offline ≠ invent Connected · session soft ≠ live dogfood · free eng s1578 · free-floor peer s1580+ mention only")
	return b.String()
}
