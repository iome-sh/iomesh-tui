package setup

import (
	"fmt"
	"strings"

	"github.com/iome-sh/iomesh-tui/internal/config"
)

// DriftHonestyFooter is residual-honest footer on every drift report (report-only · no invent green).
const DriftHonestyFooter = "dual_write OFF · not Memory GA · drift report ≠ invent install green · package wire ≠ Connected"

// DriftSnapshot is a residual-honest runtime probe filled by TUI/agent.
// Compatible field set with agent.Runtime.DriftSnapshot() when that lands (s1534 P6b).
// No auto-repair — surfaces fill this; BuildDriftReport only compares config intent vs snap.
type DriftSnapshot struct {
	MCPAttached    bool
	MCPServerCount int
	MemoryServerOK bool
	MemoryServer   string
	MeshEnabled    bool
	PullRunning    bool
	PullConsumer   string
	AnalyzeRunning bool
	DualWrite      bool
	MemoryEnabled  bool
}

// DriftReport is a report-only maintenance drift probe (s1534 P6b).
// OK = dual_write honest + no critical contradictions; OK ≠ invent Connected / install green.
// Findings list residual-honest mismatches; Notes suggest next steps (no auto-repair).
type DriftReport struct {
	// config intent
	ConfigPresent           bool
	DualWriteConfig         bool
	DualWriteHonest         bool // must be false dual_write for OK
	MemoryEnabled           bool
	PullContinuousConfig    bool
	AnalyzeContinuousConfig bool
	// runtime snap
	MCPAttached    bool
	MCPServerCount int
	MemoryServerOK bool
	MeshEnabled    bool
	PullRunning    bool
	AnalyzeRunning bool
	// mismatches + residual
	Findings []string
	OK       bool
	Notes    []string
}

// BuildDriftReport compares config intent with a runtime DriftSnapshot (report-only).
// Never invents Connected / Memory GA / install green. No auto-repair.
func BuildDriftReport(cfg *config.Config, snap DriftSnapshot) DriftReport {
	rep := DriftReport{
		MCPAttached:    snap.MCPAttached,
		MCPServerCount: snap.MCPServerCount,
		MemoryServerOK: snap.MemoryServerOK,
		MeshEnabled:    snap.MeshEnabled,
		PullRunning:    snap.PullRunning,
		AnalyzeRunning: snap.AnalyzeRunning,
		Findings:       []string{},
		Notes:          []string{},
	}

	meshConfigEnabled := false
	if cfg != nil {
		rep.ConfigPresent = true
		rep.DualWriteConfig = cfg.Memory.DualWrite
		rep.DualWriteHonest = !cfg.Memory.DualWrite
		rep.MemoryEnabled = cfg.Memory.Enabled
		rep.PullContinuousConfig = cfg.Memory.PullContinuous
		rep.AnalyzeContinuousConfig = cfg.Memory.AnalyzeContinuous
		meshConfigEnabled = cfg.IOMesh.Enabled
		// Report mesh as config OR runtime (either plane can show enabled).
		rep.MeshEnabled = snap.MeshEnabled || meshConfigEnabled
	} else {
		// No config: dual_write honest-by-absence; still not OK (config absent).
		rep.DualWriteHonest = true
		rep.MemoryEnabled = snap.MemoryEnabled
		rep.MeshEnabled = snap.MeshEnabled
	}

	// --- Findings (residual-honest mismatches) ---
	if rep.DualWriteConfig {
		rep.Findings = append(rep.Findings,
			"dual_write=true in config (BAD honesty · residual-honest local-primary prefers dual_write=false)")
	}
	if rep.MemoryEnabled && !snap.MemoryServerOK {
		rep.Findings = append(rep.Findings,
			"memory enabled but MCP memory server not OK · start iomesh-memory-mcp or run /setup preflight")
	}
	if rep.MemoryEnabled && !snap.MCPAttached && snap.MCPServerCount == 0 {
		rep.Findings = append(rep.Findings,
			"memory enabled but MCP not attached / no servers · next: /setup reload or start memory host")
	}
	if rep.PullContinuousConfig && !snap.PullRunning {
		rep.Findings = append(rep.Findings,
			"pull_continuous=true but pull not running · next: /setup pull start")
	}
	if rep.AnalyzeContinuousConfig && !snap.AnalyzeRunning {
		rep.Findings = append(rep.Findings,
			"analyze_continuous=true but analyze not running · next: /setup analyze start")
	}
	// Mesh pull wanted but mesh disabled (config + runtime both off).
	if rep.PullContinuousConfig && !snap.MeshEnabled && !meshConfigEnabled {
		rep.Findings = append(rep.Findings,
			"pull_continuous=true but mesh disabled · configure [iomesh] enabled + endpoint, then /setup reload")
	}

	// --- OK: dual_write honest + no critical contradictions (≠ invent Connected) ---
	rep.OK = rep.DualWriteHonest && rep.ConfigPresent
	if rep.MemoryEnabled && !snap.MemoryServerOK {
		rep.OK = false
	}
	// Intent vs runtime continuous mismatches are contradictions for maintenance probe.
	if rep.PullContinuousConfig && !snap.PullRunning {
		rep.OK = false
	}
	if rep.AnalyzeContinuousConfig && !snap.AnalyzeRunning {
		rep.OK = false
	}
	if rep.PullContinuousConfig && !snap.MeshEnabled && !meshConfigEnabled {
		rep.OK = false
	}

	// --- Notes: next steps only (no auto-repair) ---
	rep.Notes = append(rep.Notes,
		"drift is report-only · no auto-repair · dual_write OFF · not Memory GA",
		"next steps when mismatched: /setup reload · /setup pull start · /setup analyze start · start memory host",
		"package wire / drift PASS ≠ invent Connected · CLI iomesh memory pull still valid",
	)
	if !rep.ConfigPresent {
		rep.Notes = append(rep.Notes, "no config — run: iomesh setup init local-memory or /setup init")
	}
	if rep.PullContinuousConfig && !snap.PullRunning {
		rep.Notes = append(rep.Notes, "pull: /setup pull start (or once) after mesh + pull_consumer configured")
	}
	if rep.AnalyzeContinuousConfig && !snap.AnalyzeRunning {
		rep.Notes = append(rep.Notes, "analyze: /setup analyze start (opt-in) · analyze_continuous default false")
	}
	if rep.MemoryEnabled && !snap.MemoryServerOK {
		rep.Notes = append(rep.Notes, "memory host: start iomesh-memory-mcp · /setup preflight")
	}

	return rep
}

// FormatDriftText returns a residual-honest human report (always includes honesty footer).
func FormatDriftText(rep DriftReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "setup drift (report-only · residual-honest · not Memory GA · ≠ invent Connected)\n")
	fmt.Fprintf(&b, "  ok: %v  (dual_write honest + no critical contradictions · never invent install green)\n", rep.OK)
	fmt.Fprintf(&b, "  config: present=%v dual_write=%v honest_off=%v memory.enabled=%v\n",
		rep.ConfigPresent, rep.DualWriteConfig, rep.DualWriteHonest, rep.MemoryEnabled)
	fmt.Fprintf(&b, "  config: pull_continuous=%v analyze_continuous=%v\n",
		rep.PullContinuousConfig, rep.AnalyzeContinuousConfig)
	fmt.Fprintf(&b, "  runtime: mcp_attached=%v mcp_servers=%d memory_server_ok=%v mesh=%v\n",
		rep.MCPAttached, rep.MCPServerCount, rep.MemoryServerOK, rep.MeshEnabled)
	fmt.Fprintf(&b, "  runtime: pull_running=%v analyze_running=%v\n",
		rep.PullRunning, rep.AnalyzeRunning)
	if len(rep.Findings) == 0 {
		fmt.Fprintf(&b, "  findings: (none)\n")
	} else {
		fmt.Fprintf(&b, "  findings:\n")
		for _, f := range rep.Findings {
			fmt.Fprintf(&b, "    - %s\n", f)
		}
	}
	for _, n := range rep.Notes {
		fmt.Fprintf(&b, "  note: %s\n", n)
	}
	fmt.Fprintf(&b, "  honesty: %s\n", DriftHonestyFooter)
	return b.String()
}

// CheckDrift is a thin alias for BuildDriftReport (config + agent-compatible snapshot).
func CheckDrift(cfg *config.Config, snap DriftSnapshot) DriftReport {
	return BuildDriftReport(cfg, snap)
}
