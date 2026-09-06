package agent

import (
	"strings"
	"testing"
)

func TestE4SoftDogfoodSessionState_DefaultNotRun(t *testing.T) {
	ResetE4SoftDogfoodSessionState()
	t.Cleanup(ResetE4SoftDogfoodSessionState)

	ran, pass := GetE4SoftDogfoodSessionState()
	if ran || pass {
		t.Fatalf("default: ran=%v pass=%v want false/false", ran, pass)
	}
	if got := E4SoftSessionLabel(); got != E4SoftNotRun {
		t.Fatalf("label: got %q want %q", got, E4SoftNotRun)
	}
}

func TestE4SoftDogfoodSessionState_PassFail(t *testing.T) {
	ResetE4SoftDogfoodSessionState()
	t.Cleanup(ResetE4SoftDogfoodSessionState)

	SetE4SoftDogfoodSessionState(true)
	ran, pass := GetE4SoftDogfoodSessionState()
	if !ran || !pass {
		t.Fatalf("after pass: ran=%v pass=%v", ran, pass)
	}
	if got := E4SoftSessionLabel(); got != E4SoftPass {
		t.Fatalf("pass label: got %q want %q", got, E4SoftPass)
	}

	SetE4SoftDogfoodSessionState(false)
	ran, pass = GetE4SoftDogfoodSessionState()
	if !ran || pass {
		t.Fatalf("after fail: ran=%v pass=%v", ran, pass)
	}
	if got := E4SoftSessionLabel(); got != E4SoftFail {
		t.Fatalf("fail label: got %q want %q", got, E4SoftFail)
	}

	// Honesty: labels never invent GA / Connected / live dogfood product language.
	for _, label := range []string{E4SoftNotRun, E4SoftPass, E4SoftFail} {
		if strings.Contains(label, "Connected") || strings.Contains(label, "GA") || strings.Contains(label, "live") {
			t.Fatalf("label must not invent Connected/GA/live: %q", label)
		}
	}
}

func TestE4SoftDogfoodSessionState_Reset(t *testing.T) {
	SetE4SoftDogfoodSessionState(true)
	ResetE4SoftDogfoodSessionState()
	if got := E4SoftSessionLabel(); got != E4SoftNotRun {
		t.Fatalf("after reset: got %q want %q", got, E4SoftNotRun)
	}
}

func TestRunE4SoftDogfood_SoftPass(t *testing.T) {
	ResetE4SoftDogfoodSessionState()
	t.Cleanup(ResetE4SoftDogfoodSessionState)

	out := RunE4SoftDogfood()
	if out == "" {
		t.Fatal("empty soft dogfood output")
	}
	for _, want := range []string{
		"E4 client-attach soft offline dogfood",
		"no MCP dial",
		"never start host",
		"not live dogfood",
		"result: PASS",
		"soft_offline_e4_session_pass",
		"journey stage 6",
		"E4 client attach",
		"tools=6",
		"iomesh mcp --connect",
		"iomesh-memory-mcp",
		"local-primary",
		"docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md",
		"dual_write OFF",
		"book-demo OFF",
		"not Memory GA",
		"Edge Memory GA candidacy only",
		"residual PASS ≠ invent Edge Memory GA declared",
		"E10 Open",
		"tip ≠ invent forever-green product dogfood",
		"residual PASS ≠ live dogfood",
		"session soft ≠ live dogfood",
		"soft offline ≠ invent Connected",
		"PASS ≠ live APPLY",
		"/onboard next e4 dogfood",
		"soft|samples|offline|e4-soft",
		"bare /onboard next e4 stays board",
		"free eng s1566",
		"free-floor peer s1568+",
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
	ran, pass := GetE4SoftDogfoodSessionState()
	if !ran || !pass {
		t.Fatalf("after soft pass: ran=%v pass=%v", ran, pass)
	}
	if got := E4SoftSessionLabel(); got != E4SoftPass {
		t.Fatalf("label after run: got %q want %q", got, E4SoftPass)
	}
}

func TestRunE4SoftDogfood_IndependentFromPortalHITLAndAgentic(t *testing.T) {
	ResetE4SoftDogfoodSessionState()
	ResetPortalHITLSoftDogfoodSessionState()
	ResetAgenticListPlanSoftDogfoodSessionState()
	t.Cleanup(func() {
		ResetE4SoftDogfoodSessionState()
		ResetPortalHITLSoftDogfoodSessionState()
		ResetAgenticListPlanSoftDogfoodSessionState()
	})

	// E4 soft marker must not reuse portal HITL or agentic dogfood labels.
	if E4SoftNotRun == PortalHITLSoftNotRun || E4SoftNotRun == AgenticListPlanSoftNotRun || E4SoftNotRun == "dogfood_not_run" {
		t.Fatal("E4 soft default must not reuse portal HITL/agentic/plugins not_run labels")
	}
	if E4SoftPass == PortalHITLSoftPass || E4SoftPass == AgenticListPlanSoftPass || E4SoftPass == "soft_offline_dogfood_session_pass" {
		t.Fatal("E4 soft pass must not reuse portal HITL/agentic/plugins pass labels")
	}
	if E4SoftFail == PortalHITLSoftFail || E4SoftFail == AgenticListPlanSoftFail || E4SoftFail == "soft_offline_dogfood_session_fail" {
		t.Fatal("E4 soft fail must not reuse portal HITL/agentic/plugins fail labels")
	}

	// Setting E4 soft must not flip portal HITL or agentic soft.
	SetE4SoftDogfoodSessionState(true)
	if got := E4SoftSessionLabel(); got != E4SoftPass {
		t.Fatalf("want E4 pass label, got %q", got)
	}
	if got := PortalHITLSoftSessionLabel(); got != PortalHITLSoftNotRun {
		t.Fatalf("portal HITL soft must stay not_run, got %q", got)
	}
	if got := AgenticListPlanSoftSessionLabel(); got != AgenticListPlanSoftNotRun {
		t.Fatalf("agentic soft must stay not_run, got %q", got)
	}
}

func TestE4SoftDogfoodNeedles_CoverBoard(t *testing.T) {
	ResetE4SoftDogfoodSessionState()
	t.Cleanup(ResetE4SoftDogfoodSessionState)

	board := MeshAgentOnboardingNextE4Lane()
	for _, want := range e4SoftDogfoodNeedles {
		if !strings.Contains(board, want) {
			t.Fatalf("e4 board missing soft-dogfood needle %q in:\n%s", want, board)
		}
	}
}
