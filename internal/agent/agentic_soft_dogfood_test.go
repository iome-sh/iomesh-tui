package agent

import (
	"strings"
	"testing"
)

func TestAgenticListPlanSoftDogfoodSessionState_DefaultNotRun(t *testing.T) {
	ResetAgenticListPlanSoftDogfoodSessionState()
	t.Cleanup(ResetAgenticListPlanSoftDogfoodSessionState)

	ran, pass := GetAgenticListPlanSoftDogfoodSessionState()
	if ran || pass {
		t.Fatalf("default: ran=%v pass=%v want false/false", ran, pass)
	}
	if got := AgenticListPlanSoftSessionLabel(); got != AgenticListPlanSoftNotRun {
		t.Fatalf("label: got %q want %q", got, AgenticListPlanSoftNotRun)
	}
}

func TestAgenticListPlanSoftDogfoodSessionState_PassFail(t *testing.T) {
	ResetAgenticListPlanSoftDogfoodSessionState()
	t.Cleanup(ResetAgenticListPlanSoftDogfoodSessionState)

	SetAgenticListPlanSoftDogfoodSessionState(true)
	ran, pass := GetAgenticListPlanSoftDogfoodSessionState()
	if !ran || !pass {
		t.Fatalf("after pass: ran=%v pass=%v", ran, pass)
	}
	if got := AgenticListPlanSoftSessionLabel(); got != AgenticListPlanSoftPass {
		t.Fatalf("pass label: got %q want %q", got, AgenticListPlanSoftPass)
	}

	SetAgenticListPlanSoftDogfoodSessionState(false)
	ran, pass = GetAgenticListPlanSoftDogfoodSessionState()
	if !ran || pass {
		t.Fatalf("after fail: ran=%v pass=%v", ran, pass)
	}
	if got := AgenticListPlanSoftSessionLabel(); got != AgenticListPlanSoftFail {
		t.Fatalf("fail label: got %q want %q", got, AgenticListPlanSoftFail)
	}

	// Honesty: labels never invent GA / Connected / live dogfood product language.
	for _, label := range []string{AgenticListPlanSoftNotRun, AgenticListPlanSoftPass, AgenticListPlanSoftFail} {
		if strings.Contains(label, "Connected") || strings.Contains(label, "GA") || strings.Contains(label, "live") {
			t.Fatalf("label must not invent Connected/GA/live: %q", label)
		}
	}
}

func TestAgenticListPlanSoftDogfoodSessionState_Reset(t *testing.T) {
	SetAgenticListPlanSoftDogfoodSessionState(true)
	ResetAgenticListPlanSoftDogfoodSessionState()
	if got := AgenticListPlanSoftSessionLabel(); got != AgenticListPlanSoftNotRun {
		t.Fatalf("after reset: got %q want %q", got, AgenticListPlanSoftNotRun)
	}
}

func TestRunAgenticListPlanSoftDogfood_SoftPass(t *testing.T) {
	ResetAgenticListPlanSoftDogfoodSessionState()
	t.Cleanup(ResetAgenticListPlanSoftDogfoodSessionState)

	out := RunAgenticListPlanSoftDogfood()
	if out == "" {
		t.Fatal("empty soft dogfood output")
	}
	for _, want := range []string{
		"agentic list/plan soft offline dogfood",
		"no MCP dial",
		"not live dogfood",
		"result: PASS",
		"soft_offline_list_plan_session_pass",
		"/integrations/{id}",
		"/integrations/add?template={id}",
		"/integrations",
		"list_plan_not_connected",
		"portal_hitl_still",
		"soft offline list/plan ≠ live dogfood",
		"≠ invent Connected",
		"portal HITL still",
		"list_org fail-open ≠ empty-as-none",
		"session soft ≠ live dogfood",
		"re-run /onboard next status then /onboard next export",
		"/onboard next agentic dogfood",
		"soft|samples|offline|list-plan-soft",
		"bare /onboard next agentic stays board",
		"dual_write OFF",
		"template= ≠ install APPLY",
		"agent MCP cannot write installs",
		"does not claim dual-auth live for list_org",
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
	ran, pass := GetAgenticListPlanSoftDogfoodSessionState()
	if !ran || !pass {
		t.Fatalf("after soft pass: ran=%v pass=%v", ran, pass)
	}
	if got := AgenticListPlanSoftSessionLabel(); got != AgenticListPlanSoftPass {
		t.Fatalf("label after run: got %q want %q", got, AgenticListPlanSoftPass)
	}
}

func TestRunAgenticListPlanSoftDogfood_IndependentFromPlugins(t *testing.T) {
	ResetAgenticListPlanSoftDogfoodSessionState()
	t.Cleanup(ResetAgenticListPlanSoftDogfoodSessionState)

	// Agentic soft marker must not reuse plugins dogfood labels.
	if AgenticListPlanSoftNotRun == "dogfood_not_run" {
		t.Fatal("agentic soft default must not reuse plugins dogfood_not_run")
	}
	if AgenticListPlanSoftPass == "soft_offline_dogfood_session_pass" {
		t.Fatal("agentic soft pass must not reuse plugins soft_offline_dogfood_session_pass")
	}
	if AgenticListPlanSoftFail == "soft_offline_dogfood_session_fail" {
		t.Fatal("agentic soft fail must not reuse plugins soft_offline_dogfood_session_fail")
	}

	// Setting agentic soft must not affect default not_run independence of label namespace.
	SetAgenticListPlanSoftDogfoodSessionState(true)
	if got := AgenticListPlanSoftSessionLabel(); got != AgenticListPlanSoftPass {
		t.Fatalf("want agentic pass label, got %q", got)
	}
	if strings.Contains(AgenticListPlanSoftSessionLabel(), "dogfood_not_run") {
		t.Fatalf("agentic label must not contain plugins dogfood_not_run: %q", AgenticListPlanSoftSessionLabel())
	}
}
