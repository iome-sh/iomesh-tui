package setup

import (
	"strings"
	"testing"
)

// s1526 P3 / s1530 P5 / s1534 P6: SetupLifecycleAgentGuidanceNote residual-honest needles.
func TestSetupLifecycleAgentGuidanceNote_HonestyNeedles(t *testing.T) {
	out := SetupLifecycleAgentGuidanceNote()
	if out == "" {
		t.Fatal("empty guidance note")
	}
	for _, want := range []string{
		"dual_write OFF",
		"not Memory GA",
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
		"/memory digest",
		"never invent",
		"INSTALL_STORE",
		"secrets",
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
		"Connected",
		"iomesh memory pull",
		"pull_continuous",
		"/setup pull",
		"analyze_continuous",
		"/setup analyze",
		"/setup drift",
		"/memory digest",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("one-liner missing %q: %s", want, s)
		}
	}
}
