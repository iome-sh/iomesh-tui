package setup

import (
	"strings"
	"testing"
)

// s1686: CLI setup init next-step dual path (in-session /setup reload vs cold restart).
func TestSetupInitNextStepLines_HonestyNeedles(t *testing.T) {
	lines := SetupInitNextStepLines()
	if len(lines) == 0 {
		t.Fatal("empty next-step lines")
	}
	out := strings.Join(lines, "\n")
	for _, want := range []string{
		"/setup reload",
		"restart",
		"package wire",
		"CLI has no",
		"setup reload",
		"dual_write OFF",
		"not Memory GA",
		"s1686",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("next-step missing %q in:\n%s", want, out)
		}
	}
	// Must not invent a CLI setup reload subcommand as a real command path.
	if strings.Contains(out, "iomesh setup reload") && !strings.Contains(out, "CLI has no") {
		t.Fatalf("must not advertise CLI setup reload without honesty:\n%s", out)
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Memory GA shipped") {
		t.Fatalf("must not invent dual_write ON / Memory GA shipped:\n%s", out)
	}
}

// s1699: setup preflight next-step dual path (in-session /setup reload vs cold restart).
func TestSetupPreflightNextStepLines_HonestyNeedles(t *testing.T) {
	lines := SetupPreflightNextStepLines()
	if len(lines) == 0 {
		t.Fatal("empty preflight next-step lines")
	}
	out := strings.Join(lines, "\n")
	for _, want := range []string{
		"/setup reload",
		"restart",
		"CLI has no",
		"package wire",
		"dual_write OFF",
		"not Memory GA",
		"s1699",
		"preflight",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("preflight next-step missing %q in:\n%s", want, out)
		}
	}
	// Must not invent a CLI setup reload subcommand as a real command path.
	if strings.Contains(out, "iomesh setup reload") && !strings.Contains(out, "CLI has no") {
		t.Fatalf("must not advertise CLI setup reload without honesty:\n%s", out)
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Memory GA shipped") {
		t.Fatalf("must not invent dual_write ON / Memory GA shipped:\n%s", out)
	}
}

// s1707: setup drift next-step dual path (in-session /setup repair · reload vs cold restart).
func TestSetupDriftNextStepLines_HonestyNeedles(t *testing.T) {
	lines := SetupDriftNextStepLines()
	if len(lines) == 0 {
		t.Fatal("empty drift next-step lines")
	}
	out := strings.Join(lines, "\n")
	for _, want := range []string{
		"/setup reload",
		"/setup repair",
		"restart",
		"CLI has no",
		"package wire",
		"dual_write OFF",
		"not Memory GA",
		"s1707",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("drift next-step missing %q in:\n%s", want, out)
		}
	}
	// Must not invent CLI setup drift/repair/reload as full product without honesty.
	if strings.Contains(out, "iomesh setup drift") && !strings.Contains(out, "CLI has no") {
		t.Fatalf("must not advertise CLI setup drift without honesty:\n%s", out)
	}
	if strings.Contains(out, "iomesh setup repair") && !strings.Contains(out, "CLI has no") {
		t.Fatalf("must not advertise CLI setup repair without honesty:\n%s", out)
	}
	if strings.Contains(out, "iomesh setup reload") && !strings.Contains(out, "CLI has no") {
		t.Fatalf("must not advertise CLI setup reload without honesty:\n%s", out)
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Memory GA shipped") {
		t.Fatalf("must not invent dual_write ON / Memory GA shipped:\n%s", out)
	}
}

// s1707: setup repair next-step dual path (in-session drift/reload vs cold restart).
func TestSetupRepairNextStepLines_HonestyNeedles(t *testing.T) {
	lines := SetupRepairNextStepLines()
	if len(lines) == 0 {
		t.Fatal("empty repair next-step lines")
	}
	out := strings.Join(lines, "\n")
	for _, want := range []string{
		"/setup reload",
		"restart",
		"CLI has no",
		"package wire",
		"dual_write OFF",
		"not Memory GA",
		"s1707",
		"repair apply",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("repair next-step missing %q in:\n%s", want, out)
		}
	}
	// Dual-path slash: /setup reload and/or /setup repair (reload required; repair in CLI honesty).
	if !strings.Contains(out, "/setup reload") && !strings.Contains(out, "/setup repair") {
		t.Fatalf("must include /setup reload and/or /setup repair:\n%s", out)
	}
	// dual_write never auto ON (explicit residual language).
	if !strings.Contains(out, "dual_write never auto ON") {
		t.Fatalf("must pin dual_write never auto ON:\n%s", out)
	}
	if strings.Contains(out, "iomesh setup repair") && !strings.Contains(out, "CLI has no") {
		t.Fatalf("must not advertise CLI setup repair without honesty:\n%s", out)
	}
	if strings.Contains(out, "iomesh setup reload") && !strings.Contains(out, "CLI has no") {
		t.Fatalf("must not advertise CLI setup reload without honesty:\n%s", out)
	}
	// Avoid inventing dual_write ON as product state (allow "never auto ON").
	if strings.Contains(out, "dual_write ON") && !strings.Contains(out, "never auto ON") {
		t.Fatalf("must not invent dual_write ON:\n%s", out)
	}
	if strings.Contains(out, "Memory GA shipped") {
		t.Fatalf("must not invent Memory GA shipped:\n%s", out)
	}
	// repair apply ≠ invent Connected
	if !strings.Contains(out, "repair apply ≠ invent Connected") {
		t.Fatalf("must pin repair apply ≠ invent Connected:\n%s", out)
	}
}

// s1526 P3 / s1530 P5 / s1534 P6 / s1558 Wave B: SetupLifecycleAgentGuidanceNote residual-honest needles.
func TestSetupLifecycleAgentGuidanceNote_HonestyNeedles(t *testing.T) {
	out := SetupLifecycleAgentGuidanceNote()
	if out == "" {
		t.Fatal("empty guidance note")
	}
	for _, want := range []string{
		"dual_write OFF",
		"not Memory GA",
		"Edge Memory GA candidacy only",
		"residual PASS ≠ invent Edge Memory GA",
		"Connected",
		"portal HITL",
		"setup-lifecycle-agent",
		"read_skill",
		"/setup",
		"iomesh memory pull",
		"pull_continuous",
		"/setup pull",
		"analyze_continuous",
		"/setup analyze",
		"/setup drift",
		"/setup repair",
		"repair apply",
		"/memory digest",
		"never invent",
		"INSTALL_STORE",
		"secrets",
		// s1558 Wave B first-run journey mapping
		"s1558",
		"free eng s1558",
		"Signup",
		"Download TUI",
		"TUI auth/keys",
		"Setup wizard",
		"Connectors",
		"Local store",
		"Analyze",
		"/onboard next journey",
		"edge-user-journey",
		"host not auto",
		"no invent TUI portal SSO",
		"free-floor peer s1560+",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("guidance missing %q in:\n%s", want, out)
		}
	}
	// Must not invent product green language.
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "dual_write ON") {
		t.Fatalf("must not invent Memory GA / dual_write ON: %s", out)
	}
	if strings.Contains(out, "Connected shipped") || strings.Contains(out, "INSTALL_STORE green shipped") {
		t.Fatalf("must not invent Connected/INSTALL_STORE green: %s", out)
	}
}

func TestSetupLifecyclePortalHandoff_HonestyNeedles(t *testing.T) {
	out := SetupLifecyclePortalHandoff()
	for _, want := range []string{
		PortalIntegrationsURL,
		PortalAgentSettingsURL,
		"dual_write OFF",
		"not Memory GA",
		"Connected",
		"browser HITL",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("portal handoff missing %q in:\n%s", want, out)
		}
	}
}

func TestSetupLifecycleHonestyOneLiner(t *testing.T) {
	s := SetupLifecycleHonestyOneLiner
	for _, want := range []string{
		"dual_write OFF",
		"not Memory GA",
		"Edge Memory GA candidacy only",
		"Connected",
		"iomesh memory pull",
		"pull_continuous",
		"/setup pull",
		"analyze_continuous",
		"/setup analyze",
		"/setup drift",
		"/setup repair",
		"repair apply ≠ invent Connected",
		"/memory digest",
		"stage 4 of edge-user-journey",
		"free eng s1558",
		"/onboard next journey",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("one-liner missing %q: %s", want, s)
		}
	}
}

// s1558 Wave B: SetupLifecycleFirstRunJourneyOneLiner residual-honest needles.
func TestSetupLifecycleFirstRunJourneyOneLiner(t *testing.T) {
	s := SetupLifecycleFirstRunJourneyOneLiner
	for _, want := range []string{
		"edge-user-journey 7 stages",
		"free eng s1558",
		"Signup",
		"Download TUI",
		"TUI auth/keys",
		"Setup wizard",
		"Connectors",
		"Local store",
		"Analyze",
		"dual_write OFF",
		"not Memory GA",
		"Edge Memory GA candidacy only",
		"residual PASS ≠ invent Edge Memory GA",
		"portal HITL",
		"host not auto",
		"no invent TUI portal SSO",
		"free-floor peer s1560+",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("first-run one-liner missing %q: %s", want, s)
		}
	}
	if strings.Contains(s, "dual_write ON") || strings.Contains(s, "Memory GA shipped") {
		t.Fatalf("must not invent dual_write ON / Memory GA shipped: %s", s)
	}
}
