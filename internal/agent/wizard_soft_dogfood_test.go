package agent

import (
	"strings"
	"testing"
)

func TestWizardSoftDogfoodSessionState_DefaultNotRun(t *testing.T) {
	ResetWizardSoftDogfoodSessionState()
	t.Cleanup(ResetWizardSoftDogfoodSessionState)

	ran, pass := GetWizardSoftDogfoodSessionState()
	if ran || pass {
		t.Fatalf("default: ran=%v pass=%v want false/false", ran, pass)
	}
	if got := WizardSoftSessionLabel(); got != WizardSoftNotRun {
		t.Fatalf("label: got %q want %q", got, WizardSoftNotRun)
	}
}

func TestWizardSoftDogfoodSessionState_PassFail(t *testing.T) {
	ResetWizardSoftDogfoodSessionState()
	t.Cleanup(ResetWizardSoftDogfoodSessionState)

	SetWizardSoftDogfoodSessionState(true)
	ran, pass := GetWizardSoftDogfoodSessionState()
	if !ran || !pass {
		t.Fatalf("after pass: ran=%v pass=%v", ran, pass)
	}
	if got := WizardSoftSessionLabel(); got != WizardSoftPass {
		t.Fatalf("pass label: got %q want %q", got, WizardSoftPass)
	}

	SetWizardSoftDogfoodSessionState(false)
	ran, pass = GetWizardSoftDogfoodSessionState()
	if !ran || pass {
		t.Fatalf("after fail: ran=%v pass=%v", ran, pass)
	}
	if got := WizardSoftSessionLabel(); got != WizardSoftFail {
		t.Fatalf("fail label: got %q want %q", got, WizardSoftFail)
	}

	// Honesty: labels never invent GA / Connected / live dogfood product language.
	for _, label := range []string{WizardSoftNotRun, WizardSoftPass, WizardSoftFail} {
		if strings.Contains(label, "Connected") || strings.Contains(label, "GA") || strings.Contains(label, "live") {
			t.Fatalf("label must not invent Connected/GA/live: %q", label)
		}
	}
}

func TestWizardSoftDogfoodSessionState_Reset(t *testing.T) {
	SetWizardSoftDogfoodSessionState(true)
	ResetWizardSoftDogfoodSessionState()
	if got := WizardSoftSessionLabel(); got != WizardSoftNotRun {
		t.Fatalf("after reset: got %q want %q", got, WizardSoftNotRun)
	}
}

func TestRunFirstRunWizardSoftDogfood_SoftPass(t *testing.T) {
	ResetWizardSoftDogfoodSessionState()
	t.Cleanup(ResetWizardSoftDogfoodSessionState)

	out := RunFirstRunWizardSoftDogfood()
	if out == "" {
		t.Fatal("empty soft dogfood output")
	}
	for _, want := range []string{
		"first-run wizard soft offline dogfood",
		"no MCP dial",
		"never start host",
		"not live dogfood",
		"result: PASS",
		"soft_offline_wizard_session_pass",
		"Wave C",
		"first-run wizard residual",
		"1. Signup",
		"2. Download TUI",
		"3. TUI auth/keys",
		"4. Setup",
		"5. Connectors",
		"6. Local store",
		"7. Analyze",
		"/onboard next setup",
		"/onboard next portal-hitl",
		"/onboard next e4",
		"/onboard next journey",
		"dual_write OFF",
		"book-demo OFF",
		"not Memory GA",
		"Edge Memory GA candidacy only",
		"residual PASS ≠ invent Edge Memory GA declared",
		"E10 Open",
		"portal HITL when connect",
		"agent MCP cannot write installs",
		"catalog ≠ Connected",
		"no invent TUI portal SSO",
		"host not auto",
		"residual PASS ≠ invent full interactive auto wizard",
		"residual PASS ≠ live dogfood",
		"session soft ≠ live dogfood",
		"soft offline ≠ invent Connected",
		"PASS ≠ live APPLY",
		"/onboard next wizard dogfood",
		"soft|samples|offline|wizard-soft",
		"bare /onboard next wizard stays board",
		"free eng s1570",
		"free-floor peer s1572+",
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
		"forever-green interactive wizard green",
		"live dogfood green",
		"INSTALL_STORE APPLY success",
		"TUI portal SSO shipped",
		"auto memory host",
	} {
		if strings.Contains(out, bad) {
			t.Fatalf("must not invent %q:\n%s", bad, out)
		}
	}
	// Session marker set to pass
	ran, pass := GetWizardSoftDogfoodSessionState()
	if !ran || !pass {
		t.Fatalf("after soft pass: ran=%v pass=%v", ran, pass)
	}
	if got := WizardSoftSessionLabel(); got != WizardSoftPass {
		t.Fatalf("label after run: got %q want %q", got, WizardSoftPass)
	}
}

func TestRunFirstRunWizardSoftDogfood_IndependentFromPeers(t *testing.T) {
	ResetWizardSoftDogfoodSessionState()
	ResetE4SoftDogfoodSessionState()
	ResetPortalHITLSoftDogfoodSessionState()
	ResetAgenticListPlanSoftDogfoodSessionState()
	t.Cleanup(func() {
		ResetWizardSoftDogfoodSessionState()
		ResetE4SoftDogfoodSessionState()
		ResetPortalHITLSoftDogfoodSessionState()
		ResetAgenticListPlanSoftDogfoodSessionState()
	})

	// Wizard soft marker must not reuse E4 / portal HITL / agentic / plugins dogfood labels.
	if WizardSoftNotRun == E4SoftNotRun || WizardSoftNotRun == PortalHITLSoftNotRun || WizardSoftNotRun == AgenticListPlanSoftNotRun || WizardSoftNotRun == "dogfood_not_run" {
		t.Fatal("wizard soft default must not reuse E4/portal HITL/agentic/plugins not_run labels")
	}
	if WizardSoftPass == E4SoftPass || WizardSoftPass == PortalHITLSoftPass || WizardSoftPass == AgenticListPlanSoftPass || WizardSoftPass == "soft_offline_dogfood_session_pass" {
		t.Fatal("wizard soft pass must not reuse E4/portal HITL/agentic/plugins pass labels")
	}
	if WizardSoftFail == E4SoftFail || WizardSoftFail == PortalHITLSoftFail || WizardSoftFail == AgenticListPlanSoftFail || WizardSoftFail == "soft_offline_dogfood_session_fail" {
		t.Fatal("wizard soft fail must not reuse E4/portal HITL/agentic/plugins fail labels")
	}

	// Setting wizard soft must not flip peer soft markers.
	SetWizardSoftDogfoodSessionState(true)
	if got := WizardSoftSessionLabel(); got != WizardSoftPass {
		t.Fatalf("want wizard pass label, got %q", got)
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

func TestWizardSoftDogfoodNeedles_CoverBoard(t *testing.T) {
	ResetWizardSoftDogfoodSessionState()
	t.Cleanup(ResetWizardSoftDogfoodSessionState)

	board := AionAgentOnboardingNextWizardLane()
	for _, want := range wizardSoftDogfoodNeedles {
		if !strings.Contains(board, want) {
			t.Fatalf("wizard board missing soft-dogfood needle %q in:\n%s", want, board)
		}
	}
}
