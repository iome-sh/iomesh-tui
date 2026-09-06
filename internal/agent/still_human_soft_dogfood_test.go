package agent

import (
	"strings"
	"testing"
)

func TestStillHumanSoftDogfoodSessionState_DefaultNotRun(t *testing.T) {
	ResetStillHumanSoftDogfoodSessionState()
	t.Cleanup(ResetStillHumanSoftDogfoodSessionState)

	ran, pass := GetStillHumanSoftDogfoodSessionState()
	if ran || pass {
		t.Fatalf("default: ran=%v pass=%v want false/false", ran, pass)
	}
	if got := StillHumanSoftSessionLabel(); got != StillHumanSoftNotRun {
		t.Fatalf("label: got %q want %q", got, StillHumanSoftNotRun)
	}
}

func TestStillHumanSoftDogfoodSessionState_PassFail(t *testing.T) {
	ResetStillHumanSoftDogfoodSessionState()
	t.Cleanup(ResetStillHumanSoftDogfoodSessionState)

	SetStillHumanSoftDogfoodSessionState(true)
	ran, pass := GetStillHumanSoftDogfoodSessionState()
	if !ran || !pass {
		t.Fatalf("after pass: ran=%v pass=%v", ran, pass)
	}
	if got := StillHumanSoftSessionLabel(); got != StillHumanSoftPass {
		t.Fatalf("pass label: got %q want %q", got, StillHumanSoftPass)
	}

	SetStillHumanSoftDogfoodSessionState(false)
	ran, pass = GetStillHumanSoftDogfoodSessionState()
	if !ran || pass {
		t.Fatalf("after fail: ran=%v pass=%v", ran, pass)
	}
	if got := StillHumanSoftSessionLabel(); got != StillHumanSoftFail {
		t.Fatalf("fail label: got %q want %q", got, StillHumanSoftFail)
	}

	// Honesty: labels never invent GA / Connected / live dogfood product language.
	for _, label := range []string{StillHumanSoftNotRun, StillHumanSoftPass, StillHumanSoftFail} {
		if strings.Contains(label, "Connected") || strings.Contains(label, "GA") || strings.Contains(label, "live") {
			t.Fatalf("label must not invent Connected/GA/live: %q", label)
		}
	}
}

func TestStillHumanSoftDogfoodSessionState_Reset(t *testing.T) {
	SetStillHumanSoftDogfoodSessionState(true)
	ResetStillHumanSoftDogfoodSessionState()
	if got := StillHumanSoftSessionLabel(); got != StillHumanSoftNotRun {
		t.Fatalf("after reset: got %q want %q", got, StillHumanSoftNotRun)
	}
}

func TestRunStillHumanApplySoftDogfood_SoftPass(t *testing.T) {
	ResetStillHumanSoftDogfoodSessionState()
	t.Cleanup(ResetStillHumanSoftDogfoodSessionState)

	out := RunStillHumanApplySoftDogfood()
	if out == "" {
		t.Fatal("empty soft dogfood output")
	}
	for _, want := range []string{
		"still-human APPLY soft offline dogfood",
		"no MCP dial",
		"never start host",
		"not live dogfood",
		"result: PASS",
		"soft_offline_still_human_session_pass",
		"Wave C continuum",
		"still-human APPLY",
		"open boxes stay open",
		"PASS ≠ live APPLY",
		"PASS ≠ invent human-gate green",
		"edge-first",
		"knowledge multi-tenant punted",
		"Slack HMAC punted",
		"portal HITL when connect",
		"book-demo OFF",
		"leave ON_SIGNAL unset",
		"dual_write OFF",
		"not Memory GA",
		"Edge Memory GA candidacy only",
		"E10 Open",
		"residual PASS ≠ invent Edge Memory GA declared",
		"session soft ≠ live dogfood",
		"soft offline ≠ invent Connected",
		"/onboard next human-gates dogfood",
		"soft|samples|offline|still-human-soft|apply-soft",
		"bare /onboard next human-gates stays board",
		"/onboard next wizard",
		"free eng s1574",
		"free-floor peer s1576+",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("soft dogfood missing %q in:\n%s", want, out)
		}
	}
	// Must not invent GA declared / dual_write ON / E10 closed / human-gate green live.
	// Note: honest residual phrases contain "invent Edge Memory GA declared" — ban invent-claim shapes only.
	for _, bad := range []string{
		"dual_write ON",
		"Connected: yes",
		"Edge Memory GA is declared",
		"E10 is closed",
		"human-gate green: yes",
		"live APPLY green",
		"INSTALL_STORE APPLY success",
		"book-demo ON",
	} {
		if strings.Contains(out, bad) {
			t.Fatalf("must not invent %q:\n%s", bad, out)
		}
	}
	// Session marker set to pass
	ran, pass := GetStillHumanSoftDogfoodSessionState()
	if !ran || !pass {
		t.Fatalf("after soft pass: ran=%v pass=%v", ran, pass)
	}
	if got := StillHumanSoftSessionLabel(); got != StillHumanSoftPass {
		t.Fatalf("label after run: got %q want %q", got, StillHumanSoftPass)
	}
}

func TestRunStillHumanApplySoftDogfood_IndependentFromPeers(t *testing.T) {
	ResetStillHumanSoftDogfoodSessionState()
	ResetWizardSoftDogfoodSessionState()
	ResetE4SoftDogfoodSessionState()
	ResetPortalHITLSoftDogfoodSessionState()
	ResetAgenticListPlanSoftDogfoodSessionState()
	t.Cleanup(func() {
		ResetStillHumanSoftDogfoodSessionState()
		ResetWizardSoftDogfoodSessionState()
		ResetE4SoftDogfoodSessionState()
		ResetPortalHITLSoftDogfoodSessionState()
		ResetAgenticListPlanSoftDogfoodSessionState()
	})

	// Still-human soft marker must not reuse wizard / E4 / portal HITL / agentic / plugins dogfood labels.
	if StillHumanSoftNotRun == WizardSoftNotRun || StillHumanSoftNotRun == E4SoftNotRun || StillHumanSoftNotRun == PortalHITLSoftNotRun || StillHumanSoftNotRun == AgenticListPlanSoftNotRun || StillHumanSoftNotRun == "dogfood_not_run" {
		t.Fatal("still-human soft default must not reuse wizard/E4/portal HITL/agentic/plugins not_run labels")
	}
	if StillHumanSoftPass == WizardSoftPass || StillHumanSoftPass == E4SoftPass || StillHumanSoftPass == PortalHITLSoftPass || StillHumanSoftPass == AgenticListPlanSoftPass || StillHumanSoftPass == "soft_offline_dogfood_session_pass" {
		t.Fatal("still-human soft pass must not reuse wizard/E4/portal HITL/agentic/plugins pass labels")
	}
	if StillHumanSoftFail == WizardSoftFail || StillHumanSoftFail == E4SoftFail || StillHumanSoftFail == PortalHITLSoftFail || StillHumanSoftFail == AgenticListPlanSoftFail || StillHumanSoftFail == "soft_offline_dogfood_session_fail" {
		t.Fatal("still-human soft fail must not reuse wizard/E4/portal HITL/agentic/plugins fail labels")
	}

	// Setting still-human soft must not flip peer soft markers.
	SetStillHumanSoftDogfoodSessionState(true)
	if got := StillHumanSoftSessionLabel(); got != StillHumanSoftPass {
		t.Fatalf("want still-human pass label, got %q", got)
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

func TestStillHumanSoftDogfoodNeedles_CoverBoard(t *testing.T) {
	ResetStillHumanSoftDogfoodSessionState()
	t.Cleanup(ResetStillHumanSoftDogfoodSessionState)

	board := MeshAgentHumanGatesHonestyBoard()
	for _, want := range stillHumanSoftDogfoodNeedles {
		if !strings.Contains(board, want) {
			t.Fatalf("human-gates board missing soft-dogfood needle %q in:\n%s", want, board)
		}
	}
}
