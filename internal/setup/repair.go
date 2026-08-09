package setup

import (
	"context"
	"fmt"
	"strings"
)

// RepairHonestyFooter residual-honest footer.
const RepairHonestyFooter = "dual_write OFF · not Memory GA · repair apply ≠ invent Connected · package wire ≠ Connected · portal HITL still human"

// RepairKind classifies a guided repair step.
type RepairKind string

const (
	RepairReloadMCP      RepairKind = "reload_mcp"       // safe
	RepairStartPull      RepairKind = "start_pull"       // safe if pull_continuous + mesh
	RepairStartAnalyze   RepairKind = "start_analyze"    // safe if analyze_continuous
	RepairNoteDualWrite  RepairKind = "note_dual_write"  // never auto-apply flip to ON; note set dual_write=false
	RepairNoteMemoryHost RepairKind = "note_memory_host" // human start host
	RepairNoteMeshConfig RepairKind = "note_mesh_config" // human configure [iomesh]
	RepairNoteNoop       RepairKind = "note_noop"        // residual-honest: no safe repair needed
)

// RepairStep is one residual-honest guided step (plan or apply result).
type RepairStep struct {
	Kind    RepairKind
	Reason  string
	Safe    bool // true = may auto-apply with --yes
	Applied bool
	Skipped bool
	Result  string
	Err     string
}

// RepairPlan is ordered residual-honest steps from a DriftReport.
type RepairPlan struct {
	Steps   []RepairStep
	DryRun  bool
	Applied int
	Failed  int
	Skipped int
}

// PlanRepair builds ordered residual-honest steps from a DriftReport (no side effects).
// Order: dual_write note → memory host note → reload_mcp → mesh note → start_pull → start_analyze.
// Never invents Connected / Memory GA / dual_write ON; notes never auto-flip honesty flags.
func PlanRepair(rep DriftReport) RepairPlan {
	plan := RepairPlan{Steps: []RepairStep{}}

	// 1. dual_write note (never auto-apply)
	if rep.DualWriteConfig {
		plan.Steps = append(plan.Steps, RepairStep{
			Kind:   RepairNoteDualWrite,
			Reason: "dual_write=true in config · residual-honest local-primary prefers dual_write=false · set dual_write=false manually (never auto-flip ON)",
			Safe:   false,
		})
	}

	// 2. memory host note (human start host)
	// reload won't fix missing host alone when attached but server not OK
	if rep.MemoryEnabled && !rep.MemoryServerOK && rep.MCPAttached {
		plan.Steps = append(plan.Steps, RepairStep{
			Kind:   RepairNoteMemoryHost,
			Reason: "memory enabled but MCP memory server not OK · start iomesh-memory-mcp host (human) · reload alone does not invent Connected",
			Safe:   false,
		})
	}

	// 3. reload_mcp (safe when memory wants MCP but not attached)
	if rep.MemoryEnabled && !rep.MCPAttached {
		plan.Steps = append(plan.Steps, RepairStep{
			Kind:   RepairReloadMCP,
			Reason: "memory enabled but MCP not attached · safe apply: reload MCP servers",
			Safe:   true,
		})
	}

	// 4. mesh config note (human configure [iomesh])
	if rep.PullContinuousConfig && !rep.MeshEnabled {
		plan.Steps = append(plan.Steps, RepairStep{
			Kind:   RepairNoteMeshConfig,
			Reason: "pull_continuous=true but mesh disabled · configure [iomesh] enabled + endpoint manually · then /setup reload · never invent Connected",
			Safe:   false,
		})
	}

	// 5. start_pull (safe when pull_continuous + mesh + not running)
	if rep.PullContinuousConfig && !rep.PullRunning && rep.MeshEnabled {
		plan.Steps = append(plan.Steps, RepairStep{
			Kind:   RepairStartPull,
			Reason: "pull_continuous=true but pull not running · mesh enabled · safe apply: start continuous pull (≠ invent Connected)",
			Safe:   true,
		})
	}

	// 6. start_analyze (safe when analyze_continuous + not running)
	if rep.AnalyzeContinuousConfig && !rep.AnalyzeRunning {
		plan.Steps = append(plan.Steps, RepairStep{
			Kind:   RepairStartAnalyze,
			Reason: "analyze_continuous=true but analyze not running · safe apply: start analyze tick (≠ invent Connected)",
			Safe:   true,
		})
	}

	// No findings / nothing actionable → residual-honest noop note
	if len(plan.Steps) == 0 {
		reason := "no safe repair needed"
		if rep.OK && len(rep.Findings) == 0 {
			reason = "no safe repair needed · drift OK / no findings · dual_write OFF residual · ≠ invent Connected"
		} else if len(rep.Findings) == 0 {
			reason = "no safe repair needed · empty findings · residual-honest · ≠ invent Connected"
		} else {
			reason = "no safe auto-apply repair for current findings · human/host steps only · ≠ invent Connected"
		}
		plan.Steps = append(plan.Steps, RepairStep{
			Kind:   RepairNoteNoop,
			Reason: reason,
			Safe:   false,
		})
	}

	return plan
}

// FormatRepairPlan residual-honest human text for /setup repair (plan/dry-run).
func FormatRepairPlan(plan RepairPlan) string {
	var b strings.Builder
	fmt.Fprintf(&b, "setup repair plan (guided · residual-honest · not Memory GA · ≠ invent Connected)\n")
	if plan.DryRun {
		fmt.Fprintf(&b, "  mode: dry-run (no apply)\n")
	} else {
		fmt.Fprintf(&b, "  mode: plan (apply only with explicit --yes · safe steps only)\n")
	}
	fmt.Fprintf(&b, "  steps: %d\n", len(plan.Steps))
	if len(plan.Steps) == 0 {
		fmt.Fprintf(&b, "  (none)\n")
	} else {
		for i, s := range plan.Steps {
			safeTag := "note"
			if s.Safe {
				safeTag = "safe"
			}
			fmt.Fprintf(&b, "  %d. [%s] %s — %s\n", i+1, safeTag, s.Kind, s.Reason)
		}
	}
	fmt.Fprintf(&b, "  note: safe steps may apply with --yes · notes are manual/human · dual_write never auto-flipped ON\n")
	fmt.Fprintf(&b, "  note: repair apply success ≠ invent Connected / install green · package wire ≠ Connected\n")
	fmt.Fprintf(&b, "  honesty: %s\n", RepairHonestyFooter)
	return b.String()
}

// FormatRepairResult after apply attempt.
func FormatRepairResult(plan RepairPlan) string {
	var b strings.Builder
	mode := "apply"
	if plan.DryRun {
		mode = "dry-run"
	}
	fmt.Fprintf(&b, "setup repair result (%s · residual-honest · not Memory GA · ≠ invent Connected)\n", mode)
	fmt.Fprintf(&b, "  applied=%d failed=%d skipped=%d steps=%d\n",
		plan.Applied, plan.Failed, plan.Skipped, len(plan.Steps))
	for i, s := range plan.Steps {
		status := "pending"
		switch {
		case s.Applied:
			status = "applied"
		case s.Skipped:
			status = "skipped"
		case s.Err != "":
			status = "failed"
		case plan.DryRun && s.Safe:
			status = "dry-run"
		case !s.Safe:
			status = "skipped"
		}
		fmt.Fprintf(&b, "  %d. [%s] %s", i+1, status, s.Kind)
		if s.Result != "" {
			fmt.Fprintf(&b, " — %s", s.Result)
		}
		if s.Err != "" {
			fmt.Fprintf(&b, " err=%s", s.Err)
		}
		fmt.Fprintf(&b, "\n")
	}
	fmt.Fprintf(&b, "  note: repair apply ≠ invent Connected · package wire ≠ Connected · dual_write stays OFF residual\n")
	fmt.Fprintf(&b, "  honesty: %s\n", RepairHonestyFooter)
	return b.String()
}

// RepairExecutor is implemented by TUI/runtime adapter for safe applies.
type RepairExecutor interface {
	ReloadMCP(ctx context.Context) error
	StartPull(ctx context.Context) error // uses config pull_* continuous start
	StartAnalyze(ctx context.Context) error
}

// ApplyRepairPlan applies only Safe steps when dryRun=false.
// Requires explicit apply (caller sets dryRun=false after operator --yes).
// Unsafe/note steps: mark Skipped with residual-honest Result, never invent green.
// Fail-open per step: record Err, continue.
func ApplyRepairPlan(ctx context.Context, plan RepairPlan, ex RepairExecutor, dryRun bool) RepairPlan {
	if ctx == nil {
		ctx = context.Background()
	}
	plan.DryRun = dryRun
	plan.Applied = 0
	plan.Failed = 0
	plan.Skipped = 0

	out := make([]RepairStep, len(plan.Steps))
	copy(out, plan.Steps)

	for i := range out {
		s := &out[i]
		s.Applied = false
		s.Skipped = false
		s.Err = ""
		// Keep Reason; clear prior Result unless we set below.

		if !s.Safe {
			s.Skipped = true
			s.Result = residualSkipResult(s.Kind)
			plan.Skipped++
			continue
		}

		if dryRun {
			// Safe but dry-run: do not apply.
			s.Applied = false
			s.Skipped = false
			s.Result = "dry-run · not applied · safe step would run with --yes · ≠ invent Connected"
			// Count as neither applied nor failed; skipped count only for notes.
			continue
		}

		if ex == nil {
			s.Err = "no executor"
			s.Result = "failed · no RepairExecutor · ≠ invent Connected"
			plan.Failed++
			continue
		}

		var err error
		switch s.Kind {
		case RepairReloadMCP:
			err = ex.ReloadMCP(ctx)
		case RepairStartPull:
			err = ex.StartPull(ctx)
		case RepairStartAnalyze:
			err = ex.StartAnalyze(ctx)
		default:
			// Unknown safe kind — treat as skip residual-honest.
			s.Skipped = true
			s.Result = "skipped · unknown safe kind · ≠ invent Connected"
			plan.Skipped++
			continue
		}

		if err != nil {
			s.Err = err.Error()
			s.Result = "failed · residual-honest · ≠ invent Connected"
			plan.Failed++
			continue
		}
		s.Applied = true
		s.Result = safeApplyResult(s.Kind)
		plan.Applied++
	}

	plan.Steps = out
	return plan
}

func residualSkipResult(kind RepairKind) string {
	switch kind {
	case RepairNoteDualWrite:
		return "skipped · manual: set dual_write=false · never auto-flip dual_write ON · ≠ invent Connected"
	case RepairNoteMemoryHost:
		return "skipped · human: start iomesh-memory-mcp host · repair ≠ invent Memory GA / Connected"
	case RepairNoteMeshConfig:
		return "skipped · human: configure [iomesh] enabled + endpoint · package wire ≠ Connected"
	case RepairNoteNoop:
		return "skipped · no safe repair needed · residual-honest · ≠ invent Connected"
	default:
		return "skipped · note/manual step · residual-honest · ≠ invent Connected"
	}
}

func safeApplyResult(kind RepairKind) string {
	switch kind {
	case RepairReloadMCP:
		return "applied reload_mcp · MCP reload attempted · ≠ invent Connected / Memory GA"
	case RepairStartPull:
		return "applied start_pull · continuous pull start attempted · ≠ invent Connected"
	case RepairStartAnalyze:
		return "applied start_analyze · analyze tick start attempted · ≠ invent Connected"
	default:
		return "applied · residual-honest · ≠ invent Connected"
	}
}
