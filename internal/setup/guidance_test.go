package setup

import (
	"strings"
	"testing"
)

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
