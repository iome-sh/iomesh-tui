package agent

import (
	"strings"
	"testing"
)

func TestToolCallSoftDogfoodSessionState_DefaultNotRun(t *testing.T) {
	ResetToolCallSoftDogfoodSessionState()
	t.Cleanup(ResetToolCallSoftDogfoodSessionState)

	ran, pass := GetToolCallSoftDogfoodSessionState()
	if ran || pass {
		t.Fatalf("default: ran=%v pass=%v want false/false", ran, pass)
	}
	if got := ToolCallSoftSessionLabel(); got != ToolCallSoftNotRun {
		t.Fatalf("label: got %q want %q", got, ToolCallSoftNotRun)
	}
}

func TestToolCallSoftDogfoodSessionState_PassFail(t *testing.T) {
	ResetToolCallSoftDogfoodSessionState()
	t.Cleanup(ResetToolCallSoftDogfoodSessionState)

	SetToolCallSoftDogfoodSessionState(true)
	ran, pass := GetToolCallSoftDogfoodSessionState()
	if !ran || !pass {
		t.Fatalf("after pass: ran=%v pass=%v", ran, pass)
	}
	if got := ToolCallSoftSessionLabel(); got != ToolCallSoftPass {
		t.Fatalf("pass label: got %q want %q", got, ToolCallSoftPass)
	}

	SetToolCallSoftDogfoodSessionState(false)
	ran, pass = GetToolCallSoftDogfoodSessionState()
	if !ran || pass {
		t.Fatalf("after fail: ran=%v pass=%v", ran, pass)
	}
	if got := ToolCallSoftSessionLabel(); got != ToolCallSoftFail {
		t.Fatalf("fail label: got %q want %q", got, ToolCallSoftFail)
	}

	// Honesty: labels never invent GA / Connected / live dogfood product language.
	for _, label := range []string{ToolCallSoftNotRun, ToolCallSoftPass, ToolCallSoftFail} {
		if strings.Contains(label, "Connected") || strings.Contains(label, "GA") || strings.Contains(label, "live") {
			t.Fatalf("label must not invent Connected/GA/live: %q", label)
		}
	}
}

func TestToolCallSoftDogfoodSessionState_Reset(t *testing.T) {
	SetToolCallSoftDogfoodSessionState(true)
	ResetToolCallSoftDogfoodSessionState()
	if got := ToolCallSoftSessionLabel(); got != ToolCallSoftNotRun {
		t.Fatalf("after reset: got %q want %q", got, ToolCallSoftNotRun)
	}
}

func TestRunDeeperToolCallSoftDogfood_SoftPass(t *testing.T) {
	ResetToolCallSoftDogfoodSessionState()
	t.Cleanup(ResetToolCallSoftDogfoodSessionState)

	out := RunDeeperToolCallSoftDogfood()
	if out == "" {
		t.Fatal("empty soft dogfood output")
	}
	for _, want := range []string{
		"deeper tool-call soft offline dogfood",
		"no MCP dial",
		"never start host",
		"not live dogfood",
		"result: PASS",
		"soft_offline_tool_call_session_pass",
		"memory_ingest_turn",
		"memory_retrieve",
		"memory_search_semantic",
		"memory_list",
		"memory_compact_status",
		"memory_facts_as_of",
		"/onboard next e4",
		"tools=6",
		"iomesh mcp --connect",
		"s1508",
		"s1566",
		"Partial→client-attach-evidence",
		"dual_write OFF",
		"not Memory GA",
		"Edge Memory GA candidacy only",
		"residual PASS ≠ invent Edge Memory GA declared",
		"E10 Open",
		"tip ≠ invent forever-green product dogfood",
		"residual PASS ≠ live dogfood",
		"session soft ≠ live dogfood",
		"soft offline ≠ invent Connected",
		"/onboard next tool-call dogfood",
		"soft|samples|offline|tool-call-soft",
		"bare /onboard next tool-call stays board",
		"free eng s1578",
		"free-floor peer s1580+",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("soft dogfood missing %q in:\n%s", want, out)
		}
	}
	// Must not invent GA declared / dual_write ON / E10 closed / forever-green live green.
	// Note: honest residual phrases contain "invent Edge Memory GA declared" — ban invent-claim shapes only.
	for _, bad := range []string{
		"dual_write ON",
		"Connected: yes",
		"Edge Memory GA is declared",
		"E10 is closed",
		"forever-green product dogfood green",
		"live dogfood green",
		"INSTALL_STORE APPLY success",
	} {
		if strings.Contains(out, bad) {
			t.Fatalf("must not invent %q:\n%s", bad, out)
		}
	}
	// Session marker set to pass
	ran, pass := GetToolCallSoftDogfoodSessionState()
	if !ran || !pass {
		t.Fatalf("after soft pass: ran=%v pass=%v", ran, pass)
	}
	if got := ToolCallSoftSessionLabel(); got != ToolCallSoftPass {
		t.Fatalf("label after run: got %q want %q", got, ToolCallSoftPass)
	}
}

func TestRunDeeperToolCallSoftDogfood_IndependentFromPeers(t *testing.T) {
	ResetToolCallSoftDogfoodSessionState()
	ResetStillHumanSoftDogfoodSessionState()
	ResetWizardSoftDogfoodSessionState()
	ResetE4SoftDogfoodSessionState()
	ResetPortalHITLSoftDogfoodSessionState()
	ResetAgenticListPlanSoftDogfoodSessionState()
	t.Cleanup(func() {
		ResetToolCallSoftDogfoodSessionState()
		ResetStillHumanSoftDogfoodSessionState()
		ResetWizardSoftDogfoodSessionState()
		ResetE4SoftDogfoodSessionState()
		ResetPortalHITLSoftDogfoodSessionState()
		ResetAgenticListPlanSoftDogfoodSessionState()
	})

	// Tool-call soft marker must not reuse peer dogfood labels.
	if ToolCallSoftNotRun == StillHumanSoftNotRun || ToolCallSoftNotRun == WizardSoftNotRun || ToolCallSoftNotRun == E4SoftNotRun || ToolCallSoftNotRun == PortalHITLSoftNotRun || ToolCallSoftNotRun == AgenticListPlanSoftNotRun || ToolCallSoftNotRun == "dogfood_not_run" {
		t.Fatal("tool-call soft default must not reuse still-human/wizard/E4/portal HITL/agentic/plugins not_run labels")
	}
	if ToolCallSoftPass == StillHumanSoftPass || ToolCallSoftPass == WizardSoftPass || ToolCallSoftPass == E4SoftPass || ToolCallSoftPass == PortalHITLSoftPass || ToolCallSoftPass == AgenticListPlanSoftPass || ToolCallSoftPass == "soft_offline_dogfood_session_pass" {
		t.Fatal("tool-call soft pass must not reuse peer pass labels")
	}
	if ToolCallSoftFail == StillHumanSoftFail || ToolCallSoftFail == WizardSoftFail || ToolCallSoftFail == E4SoftFail || ToolCallSoftFail == PortalHITLSoftFail || ToolCallSoftFail == AgenticListPlanSoftFail || ToolCallSoftFail == "soft_offline_dogfood_session_fail" {
		t.Fatal("tool-call soft fail must not reuse peer fail labels")
	}

	// Setting tool-call soft must not flip peer soft markers.
	SetToolCallSoftDogfoodSessionState(true)
	if got := ToolCallSoftSessionLabel(); got != ToolCallSoftPass {
		t.Fatalf("want tool-call pass label, got %q", got)
	}
	if got := StillHumanSoftSessionLabel(); got != StillHumanSoftNotRun {
		t.Fatalf("still-human soft must stay not_run, got %q", got)
	}
	if got := WizardSoftSessionLabel(); got != WizardSoftNotRun {
		t.Fatalf("wizard soft must stay not_run, got %q", got)
	}
	if got := E4SoftSessionLabel(); got != E4SoftNotRun {
		t.Fatalf("E4 soft must stay not_run, got %q", got)
	}
	if got := PortalHITLSoftSessionLabel(); got != PortalHITLSoftNotRun {
		t.Fatalf("portal HITL soft must stay not_run, got %q", got)
	}
	if got := AgenticListPlanSoftSessionLabel(); got != AgenticListPlanSoftNotRun {
		t.Fatalf("agentic soft must stay not_run, got %q", got)
	}
}

func TestToolCallSoftDogfoodNeedles_CoverBoard(t *testing.T) {
	ResetToolCallSoftDogfoodSessionState()
	t.Cleanup(ResetToolCallSoftDogfoodSessionState)

	board := AionAgentOnboardingNextToolCallLane()
	for _, want := range toolCallSoftDogfoodNeedles {
		if !strings.Contains(board, want) {
			t.Fatalf("tool-call board missing soft-dogfood needle %q in:\n%s", want, board)
		}
	}
}
