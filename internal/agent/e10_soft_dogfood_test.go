package agent

import (
	"strings"
	"testing"
)

func TestE10SoftDogfoodSessionState_DefaultNotRun(t *testing.T) {
	ResetE10SoftDogfoodSessionState()
	t.Cleanup(ResetE10SoftDogfoodSessionState)

	ran, pass := GetE10SoftDogfoodSessionState()
	if ran || pass {
		t.Fatalf("default: ran=%v pass=%v want false/false", ran, pass)
	}
	if got := E10SoftSessionLabel(); got != E10SoftNotRun {
		t.Fatalf("label: got %q want %q", got, E10SoftNotRun)
	}
}

func TestE10SoftDogfoodSessionState_PassFail(t *testing.T) {
	ResetE10SoftDogfoodSessionState()
	t.Cleanup(ResetE10SoftDogfoodSessionState)

	SetE10SoftDogfoodSessionState(true)
	ran, pass := GetE10SoftDogfoodSessionState()
	if !ran || !pass {
		t.Fatalf("after pass: ran=%v pass=%v", ran, pass)
	}
	if got := E10SoftSessionLabel(); got != E10SoftPass {
		t.Fatalf("pass label: got %q want %q", got, E10SoftPass)
	}

	SetE10SoftDogfoodSessionState(false)
	ran, pass = GetE10SoftDogfoodSessionState()
	if !ran || pass {
		t.Fatalf("after fail: ran=%v pass=%v", ran, pass)
	}
	if got := E10SoftSessionLabel(); got != E10SoftFail {
		t.Fatalf("fail label: got %q want %q", got, E10SoftFail)
	}

	// Honesty: labels never invent GA / Connected / live dogfood product language.
	for _, label := range []string{E10SoftNotRun, E10SoftPass, E10SoftFail} {
		if strings.Contains(label, "Connected") || strings.Contains(label, "GA") || strings.Contains(label, "live") {
			t.Fatalf("label must not invent Connected/GA/live: %q", label)
		}
	}
}

func TestE10SoftDogfoodSessionState_Reset(t *testing.T) {
	SetE10SoftDogfoodSessionState(true)
	ResetE10SoftDogfoodSessionState()
	if got := E10SoftSessionLabel(); got != E10SoftNotRun {
		t.Fatalf("after reset: got %q want %q", got, E10SoftNotRun)
	}
}

func TestRunE10OpenSoftDogfood_SoftPass(t *testing.T) {
	ResetE10SoftDogfoodSessionState()
	t.Cleanup(ResetE10SoftDogfoodSessionState)

	out := RunE10OpenSoftDogfood()
	if out == "" {
		t.Fatal("empty soft residual-check output")
	}
	for _, want := range []string{
		"E10 Open soft offline residual-check",
		"no MCP dial",
		"never start host",
		"not live dogfood",
		"result: PASS",
		"soft_offline_e10_session_pass",
		"E10 Open",
		"E10 Open reaffirm",
		"residual PASS ≠ invent E10 closed",
		"residual PASS ≠ invent Edge Memory GA declared",
		"Edge Memory GA candidacy only",
		"not Memory GA",
		"dual_write OFF",
		"book-demo OFF",
		"founder sign-off only if declaring Edge Memory GA",
		"candidacy allowed without E10",
		"PASS ≠ live APPLY",
		"residual-check",
		"session soft ≠ live dogfood",
		"residual PASS ≠ live dogfood",
		"soft offline ≠ invent Connected",
		"/onboard next e4",
		"/onboard next human-gates",
		"OSS packaging",
		"MIT harness",
		"not control plane",
		"/onboard next e10 dogfood",
		"soft|samples|offline|e10-soft|residual-check",
		"bare /onboard next e10 stays board",
		"free eng s1586",
		"free-floor peer s1588+",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("soft residual-check missing %q in:\n%s", want, out)
		}
	}
	// Must not invent GA declared / dual_write ON / E10 closed / live APPLY green.
	// Note: honest residual phrases contain "invent Edge Memory GA declared" / "invent E10 closed" — ban invent-claim shapes only.
	for _, bad := range []string{
		"dual_write ON",
		"Connected: yes",
		"Edge Memory GA is declared",
		"E10 is closed",
		"live APPLY green",
		"INSTALL_STORE APPLY success",
		"book-demo ON",
		"live dogfood green",
	} {
		if strings.Contains(out, bad) {
			t.Fatalf("must not invent %q:\n%s", bad, out)
		}
	}
	// Session marker set to pass
	ran, pass := GetE10SoftDogfoodSessionState()
	if !ran || !pass {
		t.Fatalf("after soft pass: ran=%v pass=%v", ran, pass)
	}
	if got := E10SoftSessionLabel(); got != E10SoftPass {
		t.Fatalf("label after run: got %q want %q", got, E10SoftPass)
	}
}

func TestRunE10OpenSoftDogfood_IndependentFromPeers(t *testing.T) {
	ResetE10SoftDogfoodSessionState()
	ResetToolCallSoftDogfoodSessionState()
	ResetStillHumanSoftDogfoodSessionState()
	ResetWizardSoftDogfoodSessionState()
	ResetE4SoftDogfoodSessionState()
	ResetPortalHITLSoftDogfoodSessionState()
	ResetAgenticListPlanSoftDogfoodSessionState()
	t.Cleanup(func() {
		ResetE10SoftDogfoodSessionState()
		ResetToolCallSoftDogfoodSessionState()
		ResetStillHumanSoftDogfoodSessionState()
		ResetWizardSoftDogfoodSessionState()
		ResetE4SoftDogfoodSessionState()
		ResetPortalHITLSoftDogfoodSessionState()
		ResetAgenticListPlanSoftDogfoodSessionState()
	})

	// E10 soft marker must not reuse peer dogfood labels.
	if E10SoftNotRun == ToolCallSoftNotRun || E10SoftNotRun == StillHumanSoftNotRun || E10SoftNotRun == WizardSoftNotRun || E10SoftNotRun == E4SoftNotRun || E10SoftNotRun == PortalHITLSoftNotRun || E10SoftNotRun == AgenticListPlanSoftNotRun || E10SoftNotRun == "dogfood_not_run" {
		t.Fatal("E10 soft default must not reuse peer not_run labels")
	}
	if E10SoftPass == ToolCallSoftPass || E10SoftPass == StillHumanSoftPass || E10SoftPass == WizardSoftPass || E10SoftPass == E4SoftPass || E10SoftPass == PortalHITLSoftPass || E10SoftPass == AgenticListPlanSoftPass || E10SoftPass == "soft_offline_dogfood_session_pass" {
		t.Fatal("E10 soft pass must not reuse peer pass labels")
	}
	if E10SoftFail == ToolCallSoftFail || E10SoftFail == StillHumanSoftFail || E10SoftFail == WizardSoftFail || E10SoftFail == E4SoftFail || E10SoftFail == PortalHITLSoftFail || E10SoftFail == AgenticListPlanSoftFail || E10SoftFail == "soft_offline_dogfood_session_fail" {
		t.Fatal("E10 soft fail must not reuse peer fail labels")
	}

	// Setting E10 soft must not flip peer soft markers.
	SetE10SoftDogfoodSessionState(true)
	if got := E10SoftSessionLabel(); got != E10SoftPass {
		t.Fatalf("want E10 pass label, got %q", got)
	}
	if got := ToolCallSoftSessionLabel(); got != ToolCallSoftNotRun {
		t.Fatalf("tool-call soft must stay not_run, got %q", got)
	}
	if got := StillHumanSoftSessionLabel(); got != StillHumanSoftNotRun {
		t.Fatalf("still-human soft must stay not_run, got %q", got)
	}
	if got := E4SoftSessionLabel(); got != E4SoftNotRun {
		t.Fatalf("E4 soft must stay not_run, got %q", got)
	}
	if got := WizardSoftSessionLabel(); got != WizardSoftNotRun {
		t.Fatalf("wizard soft must stay not_run, got %q", got)
	}
	if got := PortalHITLSoftSessionLabel(); got != PortalHITLSoftNotRun {
		t.Fatalf("portal HITL soft must stay not_run, got %q", got)
	}
	if got := AgenticListPlanSoftSessionLabel(); got != AgenticListPlanSoftNotRun {
		t.Fatalf("agentic soft must stay not_run, got %q", got)
	}
}

func TestE10SoftDogfoodNeedles_CoverBoard(t *testing.T) {
	ResetE10SoftDogfoodSessionState()
	t.Cleanup(ResetE10SoftDogfoodSessionState)

	board := MeshAgentOnboardingNextE10Lane()
	for _, want := range e10SoftDogfoodNeedles {
		if !strings.Contains(board, want) {
			t.Fatalf("e10 board missing soft residual-check needle %q in:\n%s", want, board)
		}
	}
}
