package agent

import (
	"strings"
	"testing"
)

func TestPortalHITLSoftDogfoodSessionState_DefaultNotRun(t *testing.T) {
	ResetPortalHITLSoftDogfoodSessionState()
	t.Cleanup(ResetPortalHITLSoftDogfoodSessionState)

	ran, pass := GetPortalHITLSoftDogfoodSessionState()
	if ran || pass {
		t.Fatalf("default: ran=%v pass=%v want false/false", ran, pass)
	}
	if got := PortalHITLSoftSessionLabel(); got != PortalHITLSoftNotRun {
		t.Fatalf("label: got %q want %q", got, PortalHITLSoftNotRun)
	}
}

func TestPortalHITLSoftDogfoodSessionState_PassFail(t *testing.T) {
	ResetPortalHITLSoftDogfoodSessionState()
	t.Cleanup(ResetPortalHITLSoftDogfoodSessionState)

	SetPortalHITLSoftDogfoodSessionState(true)
	ran, pass := GetPortalHITLSoftDogfoodSessionState()
	if !ran || !pass {
		t.Fatalf("after pass: ran=%v pass=%v", ran, pass)
	}
	if got := PortalHITLSoftSessionLabel(); got != PortalHITLSoftPass {
		t.Fatalf("pass label: got %q want %q", got, PortalHITLSoftPass)
	}

	SetPortalHITLSoftDogfoodSessionState(false)
	ran, pass = GetPortalHITLSoftDogfoodSessionState()
	if !ran || pass {
		t.Fatalf("after fail: ran=%v pass=%v", ran, pass)
	}
	if got := PortalHITLSoftSessionLabel(); got != PortalHITLSoftFail {
		t.Fatalf("fail label: got %q want %q", got, PortalHITLSoftFail)
	}

	// Honesty: labels never invent GA / Connected / live dogfood product language.
	for _, label := range []string{PortalHITLSoftNotRun, PortalHITLSoftPass, PortalHITLSoftFail} {
		if strings.Contains(label, "Connected") || strings.Contains(label, "GA") || strings.Contains(label, "live") {
			t.Fatalf("label must not invent Connected/GA/live: %q", label)
		}
	}
}

func TestPortalHITLSoftDogfoodSessionState_Reset(t *testing.T) {
	SetPortalHITLSoftDogfoodSessionState(true)
	ResetPortalHITLSoftDogfoodSessionState()
	if got := PortalHITLSoftSessionLabel(); got != PortalHITLSoftNotRun {
		t.Fatalf("after reset: got %q want %q", got, PortalHITLSoftNotRun)
	}
}

func TestRunPortalHITLSoftDogfood_SoftPass(t *testing.T) {
	ResetPortalHITLSoftDogfoodSessionState()
	t.Cleanup(ResetPortalHITLSoftDogfoodSessionState)

	out := RunPortalHITLSoftDogfood()
	if out == "" {
		t.Fatal("empty soft dogfood output")
	}
	for _, want := range []string{
		"portal HITL soft offline dogfood",
		"no MCP dial",
		"not live dogfood",
		"result: PASS",
		"soft_offline_portal_hitl_session_pass",
		"journey stage 5",
		"/integrations/{id}",
		"/integrations/add?template={id}",
		"/integrations",
		"portal HITL when connect",
		"portal HITL still",
		"soft offline ≠ invent Connected",
		"session soft ≠ live dogfood",
		"residual PASS ≠ live dogfood",
		"agent MCP cannot write installs",
		"catalog ≠ Connected",
		"template= ≠ install APPLY",
		"dual_write OFF",
		"book-demo OFF",
		"not Memory GA",
		"Edge Memory GA candidacy only",
		"console.iome.sh/integrations",
		"console.iome.sh/settings/agent",
		"/onboard next portal-hitl dogfood",
		"soft|samples|offline|portal-hitl-soft",
		"bare /onboard next portal-hitl stays board",
		"free eng s1562",
		"free-floor peer s1564+",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("soft dogfood missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Connected: yes") || strings.Contains(out, "dual_write ON") {
		t.Fatalf("must not invent Connected/dual_write ON:\n%s", out)
	}
	if strings.Contains(out, "INSTALL_STORE APPLY success") || strings.Contains(out, "live dogfood green") {
		t.Fatalf("must not invent APPLY/live green:\n%s", out)
	}
	// Session marker set to pass
	ran, pass := GetPortalHITLSoftDogfoodSessionState()
	if !ran || !pass {
		t.Fatalf("after soft pass: ran=%v pass=%v", ran, pass)
	}
	if got := PortalHITLSoftSessionLabel(); got != PortalHITLSoftPass {
		t.Fatalf("label after run: got %q want %q", got, PortalHITLSoftPass)
	}
}

func TestRunPortalHITLSoftDogfood_IndependentFromAgenticAndPlugins(t *testing.T) {
	ResetPortalHITLSoftDogfoodSessionState()
	ResetAgenticListPlanSoftDogfoodSessionState()
	t.Cleanup(func() {
		ResetPortalHITLSoftDogfoodSessionState()
		ResetAgenticListPlanSoftDogfoodSessionState()
	})

	// Portal HITL soft marker must not reuse agentic or plugins dogfood labels.
	if PortalHITLSoftNotRun == AgenticListPlanSoftNotRun || PortalHITLSoftNotRun == "dogfood_not_run" {
		t.Fatal("portal HITL soft default must not reuse agentic/plugins not_run labels")
	}
	if PortalHITLSoftPass == AgenticListPlanSoftPass || PortalHITLSoftPass == "soft_offline_dogfood_session_pass" {
		t.Fatal("portal HITL soft pass must not reuse agentic/plugins pass labels")
	}
	if PortalHITLSoftFail == AgenticListPlanSoftFail || PortalHITLSoftFail == "soft_offline_dogfood_session_fail" {
		t.Fatal("portal HITL soft fail must not reuse agentic/plugins fail labels")
	}

	// Setting portal HITL soft must not flip agentic soft.
	SetPortalHITLSoftDogfoodSessionState(true)
	if got := PortalHITLSoftSessionLabel(); got != PortalHITLSoftPass {
		t.Fatalf("want portal HITL pass label, got %q", got)
	}
	if got := AgenticListPlanSoftSessionLabel(); got != AgenticListPlanSoftNotRun {
		t.Fatalf("agentic soft must stay not_run, got %q", got)
	}
}

func TestPortalHITLSoftDogfoodNeedles_CoverBoard(t *testing.T) {
	ResetPortalHITLSoftDogfoodSessionState()
	t.Cleanup(ResetPortalHITLSoftDogfoodSessionState)

	board := MeshAgentOnboardingNextPortalHITLLane()
	for _, want := range portalHITLSoftDogfoodNeedles {
		if !strings.Contains(board, want) {
			t.Fatalf("portal-hitl board missing soft-dogfood needle %q in:\n%s", want, board)
		}
	}
}
