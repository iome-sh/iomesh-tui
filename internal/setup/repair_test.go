package setup

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestPlanRepair_DualWriteNoteOnly(t *testing.T) {
	rep := DriftReport{
		ConfigPresent:   true,
		DualWriteConfig: true,
		DualWriteHonest: false,
		MemoryEnabled:   true,
		MCPAttached:     true,
		MemoryServerOK:  true,
		Findings:        []string{"dual_write=true in config"},
		OK:              false,
	}
	plan := PlanRepair(rep)
	if len(plan.Steps) == 0 {
		t.Fatal("expected at least note_dual_write step")
	}
	found := false
	for _, s := range plan.Steps {
		if s.Kind == RepairNoteDualWrite {
			found = true
			if s.Safe {
				t.Fatal("note_dual_write must Safe=false (never auto-apply dual_write flip)")
			}
			if !strings.Contains(s.Reason, "dual_write") {
				t.Fatalf("reason should mention dual_write: %s", s.Reason)
			}
		}
		// Must never plan a safe step that flips dual_write
		if s.Kind == RepairNoteDualWrite && s.Safe {
			t.Fatal("dual_write note must not be safe")
		}
	}
	if !found {
		t.Fatalf("expected note_dual_write: %+v", plan.Steps)
	}
	// No invent green in format
	text := FormatRepairPlan(plan)
	assertHonestyFooter(t, text)
	if strings.Contains(text, "Connected: yes") || strings.Contains(text, "Memory GA: yes") {
		t.Fatalf("must not invent green:\n%s", text)
	}
}

func TestPlanRepair_PullContinuousMeshStartPull(t *testing.T) {
	rep := DriftReport{
		ConfigPresent:        true,
		DualWriteConfig:      false,
		DualWriteHonest:      true,
		MemoryEnabled:        true,
		PullContinuousConfig: true,
		MCPAttached:          true,
		MemoryServerOK:       true,
		MeshEnabled:          true,
		PullRunning:          false,
		Findings:             []string{"pull_continuous=true but pull not running"},
		OK:                   false,
	}
	plan := PlanRepair(rep)
	found := false
	for _, s := range plan.Steps {
		if s.Kind == RepairStartPull {
			found = true
			if !s.Safe {
				t.Fatal("start_pull must Safe=true when pull_continuous + mesh + not running")
			}
		}
	}
	if !found {
		t.Fatalf("expected start_pull Safe=true: %+v", plan.Steps)
	}
	text := FormatRepairPlan(plan)
	assertHonestyFooter(t, text)
}

func TestPlanRepair_PullContinuousNoMeshNote(t *testing.T) {
	rep := DriftReport{
		ConfigPresent:        true,
		DualWriteHonest:      true,
		PullContinuousConfig: true,
		MeshEnabled:          false,
		PullRunning:          false,
		Findings:             []string{"pull_continuous=true but mesh disabled"},
		OK:                   false,
	}
	plan := PlanRepair(rep)
	var meshNote, startPull bool
	for _, s := range plan.Steps {
		if s.Kind == RepairNoteMeshConfig {
			meshNote = true
			if s.Safe {
				t.Fatal("note_mesh_config must Safe=false")
			}
		}
		if s.Kind == RepairStartPull {
			startPull = true
		}
	}
	if !meshNote {
		t.Fatalf("expected note_mesh_config: %+v", plan.Steps)
	}
	if startPull {
		t.Fatal("must not plan start_pull when mesh disabled")
	}
}

func TestPlanRepair_ReloadMCP(t *testing.T) {
	rep := DriftReport{
		ConfigPresent:   true,
		DualWriteHonest: true,
		MemoryEnabled:   true,
		MCPAttached:     false,
		MemoryServerOK:  true,
		Findings:        []string{"memory enabled but MCP not attached"},
		OK:              false,
	}
	plan := PlanRepair(rep)
	found := false
	for _, s := range plan.Steps {
		if s.Kind == RepairReloadMCP {
			found = true
			if !s.Safe {
				t.Fatal("reload_mcp must Safe=true")
			}
		}
	}
	if !found {
		t.Fatalf("expected reload_mcp: %+v", plan.Steps)
	}
}

func TestPlanRepair_MemoryHostNote(t *testing.T) {
	rep := DriftReport{
		ConfigPresent:   true,
		DualWriteHonest: true,
		MemoryEnabled:   true,
		MCPAttached:     true,
		MemoryServerOK:  false,
		Findings:        []string{"memory enabled but MCP memory server not OK"},
		OK:              false,
	}
	plan := PlanRepair(rep)
	found := false
	for _, s := range plan.Steps {
		if s.Kind == RepairNoteMemoryHost {
			found = true
			if s.Safe {
				t.Fatal("note_memory_host must Safe=false")
			}
		}
	}
	if !found {
		t.Fatalf("expected note_memory_host: %+v", plan.Steps)
	}
}

func TestPlanRepair_AnalyzeStart(t *testing.T) {
	rep := DriftReport{
		ConfigPresent:           true,
		DualWriteHonest:         true,
		AnalyzeContinuousConfig: true,
		AnalyzeRunning:          false,
		OK:                      false,
		Findings:                []string{"analyze_continuous=true but analyze not running"},
	}
	plan := PlanRepair(rep)
	found := false
	for _, s := range plan.Steps {
		if s.Kind == RepairStartAnalyze {
			found = true
			if !s.Safe {
				t.Fatal("start_analyze must Safe=true")
			}
		}
	}
	if !found {
		t.Fatalf("expected start_analyze: %+v", plan.Steps)
	}
}

func TestPlanRepair_Order(t *testing.T) {
	// All conditions true to verify order
	rep := DriftReport{
		ConfigPresent:           true,
		DualWriteConfig:         true,
		DualWriteHonest:         false,
		MemoryEnabled:           true,
		MCPAttached:             false, // reload
		MemoryServerOK:          false, // host note only if MCPAttached — so host note off
		PullContinuousConfig:    true,
		MeshEnabled:             false, // mesh note; no start_pull
		PullRunning:             false,
		AnalyzeContinuousConfig: true,
		AnalyzeRunning:          false,
		OK:                      false,
		Findings:                []string{"many"},
	}
	// Force memory host note: attached but server not OK
	rep.MCPAttached = true
	// But then reload won't fire (!MCPAttached). Split: attach false for reload, server not ok needs attach.
	// Spec order: dual_write → memory host → reload → mesh → start_pull → start_analyze
	// memory host needs MCPAttached; reload needs !MCPAttached — mutually exclusive in one plan.
	// Use host note path (attached) + dual_write + mesh + analyze.
	plan := PlanRepair(rep)
	var kinds []RepairKind
	for _, s := range plan.Steps {
		kinds = append(kinds, s.Kind)
	}
	// Expected: note_dual_write, note_memory_host, note_mesh_config, start_analyze
	// (no reload because MCPAttached; no start_pull because !MeshEnabled)
	wantOrder := []RepairKind{
		RepairNoteDualWrite,
		RepairNoteMemoryHost,
		RepairNoteMeshConfig,
		RepairStartAnalyze,
	}
	if len(kinds) != len(wantOrder) {
		t.Fatalf("kinds=%v want=%v", kinds, wantOrder)
	}
	for i := range wantOrder {
		if kinds[i] != wantOrder[i] {
			t.Fatalf("order mismatch at %d: got %v want %v (full=%v)", i, kinds[i], wantOrder[i], kinds)
		}
	}
}

func TestPlanRepair_EmptyFindingsResidualHonest(t *testing.T) {
	rep := DriftReport{
		ConfigPresent:   true,
		DualWriteHonest: true,
		MemoryEnabled:   true,
		MCPAttached:     true,
		MemoryServerOK:  true,
		Findings:        nil,
		OK:              true,
	}
	plan := PlanRepair(rep)
	if len(plan.Steps) == 0 {
		t.Fatal("empty findings should still yield residual-honest noop note or empty-with-format")
	}
	// Prefer single noop note
	if plan.Steps[0].Kind != RepairNoteNoop && plan.Steps[0].Safe {
		t.Fatalf("empty plan step should not be a surprise safe apply: %+v", plan.Steps[0])
	}
	text := FormatRepairPlan(plan)
	assertHonestyFooter(t, text)
	if !strings.Contains(text, "no safe repair needed") && !strings.Contains(plan.Steps[0].Reason, "no safe repair needed") {
		// Format may list the step reason
		if !strings.Contains(text, string(RepairNoteNoop)) && !strings.Contains(text, "none") {
			t.Fatalf("expected residual empty guidance:\n%s", text)
		}
	}
	resultText := FormatRepairResult(plan)
	assertHonestyFooter(t, resultText)
}

func TestApplyRepairPlan_DryRunLeavesAppliedFalse(t *testing.T) {
	rep := DriftReport{
		ConfigPresent:           true,
		DualWriteHonest:         true,
		PullContinuousConfig:    true,
		MeshEnabled:             true,
		PullRunning:             false,
		MemoryEnabled:           true,
		MCPAttached:             false,
		AnalyzeContinuousConfig: true,
		AnalyzeRunning:          false,
	}
	plan := PlanRepair(rep)
	ex := &mockRepairExecutor{}
	out := ApplyRepairPlan(context.Background(), plan, ex, true)
	if !out.DryRun {
		t.Fatal("DryRun must be true")
	}
	if out.Applied != 0 {
		t.Fatalf("dry-run Applied must be 0, got %d", out.Applied)
	}
	for _, s := range out.Steps {
		if s.Applied {
			t.Fatalf("dry-run step must Applied=false: %+v", s)
		}
	}
	if ex.reloadCalls != 0 || ex.pullCalls != 0 || ex.analyzeCalls != 0 {
		t.Fatalf("dry-run must not call executor: %+v", ex)
	}
	text := FormatRepairResult(out)
	assertHonestyFooter(t, text)
}

func TestApplyRepairPlan_MockAppliesSafeSkipsNotes(t *testing.T) {
	rep := DriftReport{
		ConfigPresent:           true,
		DualWriteConfig:         true, // note → skip
		DualWriteHonest:         false,
		MemoryEnabled:           true,
		MCPAttached:             false, // reload safe
		MemoryServerOK:          true,
		PullContinuousConfig:    true,
		MeshEnabled:             true,
		PullRunning:             false, // start_pull safe
		AnalyzeContinuousConfig: true,
		AnalyzeRunning:          false, // start_analyze safe
	}
	plan := PlanRepair(rep)
	ex := &mockRepairExecutor{}
	out := ApplyRepairPlan(context.Background(), plan, ex, false)
	if out.DryRun {
		t.Fatal("apply mode DryRun=false")
	}
	if out.Applied < 1 {
		t.Fatalf("expected safe applies, applied=%d steps=%+v", out.Applied, out.Steps)
	}
	if out.Skipped < 1 {
		t.Fatalf("expected dual_write note skipped, skipped=%d", out.Skipped)
	}
	if ex.reloadCalls != 1 {
		t.Fatalf("reload calls=%d want 1", ex.reloadCalls)
	}
	if ex.pullCalls != 1 {
		t.Fatalf("pull calls=%d want 1", ex.pullCalls)
	}
	if ex.analyzeCalls != 1 {
		t.Fatalf("analyze calls=%d want 1", ex.analyzeCalls)
	}

	var dualSkipped, reloadApplied bool
	for _, s := range out.Steps {
		if s.Kind == RepairNoteDualWrite {
			if !s.Skipped || s.Applied {
				t.Fatalf("dual_write note must be skipped not applied: %+v", s)
			}
			dualSkipped = true
			if !strings.Contains(s.Result, "dual_write") {
				t.Fatalf("skip result should mention dual_write: %s", s.Result)
			}
		}
		if s.Kind == RepairReloadMCP {
			if !s.Applied || s.Err != "" {
				t.Fatalf("reload_mcp should apply: %+v", s)
			}
			reloadApplied = true
		}
	}
	if !dualSkipped || !reloadApplied {
		t.Fatalf("expected dual skipped + reload applied: %+v", out.Steps)
	}

	text := FormatRepairResult(out)
	assertHonestyFooter(t, text)
	// Must not invent Connected green
	if strings.Contains(text, "Connected: yes") || strings.Contains(text, "install green") && strings.Contains(text, "invent install green: yes") {
		t.Fatalf("must not invent green:\n%s", text)
	}
	if !strings.Contains(text, "≠ invent Connected") {
		t.Fatalf("honesty residual required:\n%s", text)
	}
}

func TestApplyRepairPlan_FailOpenContinues(t *testing.T) {
	rep := DriftReport{
		ConfigPresent:           true,
		DualWriteHonest:         true,
		MemoryEnabled:           true,
		MCPAttached:             false,
		PullContinuousConfig:    true,
		MeshEnabled:             true,
		PullRunning:             false,
		AnalyzeContinuousConfig: true,
		AnalyzeRunning:          false,
	}
	plan := PlanRepair(rep)
	ex := &mockRepairExecutor{reloadErr: errors.New("reload boom")}
	out := ApplyRepairPlan(context.Background(), plan, ex, false)
	if out.Failed < 1 {
		t.Fatalf("expected failed step, got failed=%d", out.Failed)
	}
	// Pull and analyze should still run after reload failure
	if ex.pullCalls != 1 || ex.analyzeCalls != 1 {
		t.Fatalf("fail-open: pull=%d analyze=%d want 1 each (reload failed first)", ex.pullCalls, ex.analyzeCalls)
	}
	if out.Applied < 2 {
		t.Fatalf("expected later safe steps applied, applied=%d", out.Applied)
	}
}

func TestFormatRepairPlan_AlwaysHonestyFooter(t *testing.T) {
	text := FormatRepairPlan(RepairPlan{})
	assertHonestyFooter(t, text)
	text2 := FormatRepairResult(RepairPlan{})
	assertHonestyFooter(t, text2)
}

func TestApplyRepairPlan_NilExecutor(t *testing.T) {
	plan := RepairPlan{Steps: []RepairStep{
		{Kind: RepairReloadMCP, Reason: "test", Safe: true},
		{Kind: RepairNoteDualWrite, Reason: "note", Safe: false},
	}}
	out := ApplyRepairPlan(context.Background(), plan, nil, false)
	if out.Failed != 1 {
		t.Fatalf("nil executor safe step should fail, failed=%d", out.Failed)
	}
	if out.Skipped != 1 {
		t.Fatalf("note should skip, skipped=%d", out.Skipped)
	}
}

func assertHonestyFooter(t *testing.T, text string) {
	t.Helper()
	needles := []string{
		"dual_write OFF",
		"not Memory GA",
		"repair apply ≠ invent Connected",
		"package wire ≠ Connected",
	}
	for _, n := range needles {
		if !strings.Contains(text, n) {
			t.Fatalf("format missing honesty needle %q:\n%s", n, text)
		}
	}
	if !strings.Contains(text, RepairHonestyFooter) && !strings.Contains(text, "portal HITL still human") {
		// Full footer preferred
		if !strings.Contains(text, "honesty:") {
			t.Fatalf("missing honesty footer:\n%s", text)
		}
	}
}

// mockRepairExecutor records safe apply calls.
type mockRepairExecutor struct {
	reloadCalls  int
	pullCalls    int
	analyzeCalls int
	reloadErr    error
	pullErr      error
	analyzeErr   error
}

func (m *mockRepairExecutor) ReloadMCP(ctx context.Context) error {
	m.reloadCalls++
	return m.reloadErr
}

func (m *mockRepairExecutor) StartPull(ctx context.Context) error {
	m.pullCalls++
	return m.pullErr
}

func (m *mockRepairExecutor) StartAnalyze(ctx context.Context) error {
	m.analyzeCalls++
	return m.analyzeErr
}
