package agent

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/mcp"
	"github.com/iome-sh/iomesh-tui/internal/skills"
)

// s1363+s1368: AionAgentOnboardingGuidanceNote residual-honest needles.
func TestAionAgentOnboardingGuidanceNote_HonestyNeedles(t *testing.T) {
	out := AionAgentOnboardingGuidanceNote()
	if out == "" {
		t.Fatal("empty guidance note")
	}
	for _, want := range []string{
		"list_connector_catalog",
		"plan_connector_setup",
		"list_org_connector_installs",
		"catalog status ≠ install Connected",
		"available=false",
		"empty-as-none",
		"console.iome.sh/integrations",
		"portal HITL",
		"agent MCP cannot write installs",
		"dual_write OFF",
		"book-demo OFF",
		"not Memory GA",
		"residual PASS ≠ live dogfood",
		"never invent install green",
		"Connected",
		"INSTALL_STORE APPLY",
		"plugins dogfood",
		"Agent Plugins GA",
		"~$88/$119",
		"knowledge/analytical",
		"aion-agent-onboarding",
		"read_skill",
		"/integrations status",
		"/onboard checklist",
		"local-primary",
		"fail-open",
		// s1368 portal Agent/MCP lane
		"console.iome.sh/settings/agent",
		"Agent/MCP",
		"mint key",
		"copy MCP connection",
		"test invoke",
		"probe only",
		"[[mcp.servers]]",
		"streamable HTTP",
		"/onboard portal",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("guidance missing %q in:\n%s", want, out)
		}
	}
	// Must not invent product success language.
	if strings.Contains(out, "Connected: yes") || strings.Contains(out, "dual_write ON") {
		t.Fatalf("must not invent Connected/dual_write ON: %s", out)
	}
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "Agent Plugins GA shipped") {
		t.Fatalf("must not invent Memory/Plugins GA: %s", out)
	}
}

// s1363+s1368: AionAgentOnboardingChecklist residual-honest numbered needles.
func TestAionAgentOnboardingChecklist_HonestyNeedles(t *testing.T) {
	out := AionAgentOnboardingChecklist()
	if out == "" {
		t.Fatal("empty checklist")
	}
	for _, want := range []string{
		"1.",
		"IOMESH/MCP",
		"fail-open",
		"2.",
		"list_connector_catalog",
		"catalog status ≠ Connected",
		"3.",
		"plan_connector_setup",
		"portal deep links",
		"4.",
		"list_org_connector_installs",
		"available=false ≠ empty-as-none",
		"5.",
		"dual_write OFF",
		"local-primary",
		"not Memory GA",
		"plugins dogfood",
		"Agent Plugins GA",
		"6.",
		"/integrations status",
		"/onboard checklist",
		"console.iome.sh/integrations",
		"never invent install green",
		"Connected",
		"INSTALL_STORE APPLY",
		"book-demo OFF",
		"residual PASS ≠ live dogfood",
		"~$88/$119",
		"knowledge/analytical",
		// s1368 portal Agent/MCP
		"console.iome.sh/settings/agent",
		"Agent/MCP",
		"mint key",
		"copy MCP connection",
		"test invoke",
		"probe only",
		"[[mcp.servers]]",
		"streamable HTTP",
		"/onboard portal",
		"agent MCP cannot write installs",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("checklist missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Memory GA shipped") {
		t.Fatalf("must not invent dual_write ON / Memory GA: %s", out)
	}
}

// s1368: AionAgentOnboardingPortalHandoff residual-honest needles.
func TestAionAgentOnboardingPortalHandoff_HonestyNeedles(t *testing.T) {
	out := AionAgentOnboardingPortalHandoff()
	if out == "" {
		t.Fatal("empty portal handoff")
	}
	for _, want := range []string{
		"portal Agent/MCP",
		"console.iome.sh/settings/agent",
		"Mint API key",
		"Agent/MCP",
		"copy MCP connection",
		"Test invoke",
		"probe only",
		"not Memory GA",
		"[[mcp.servers]]",
		"streamable HTTP",
		"/onboard",
		"/integrations status",
		"console.iome.sh/integrations",
		"agent MCP cannot write installs",
		"dual_write OFF",
		"book-demo OFF",
		"residual PASS ≠ live dogfood",
		"never invent install green",
		"Connected",
		"INSTALL_STORE APPLY",
		"empty-as-none",
		"catalog ≠ Connected",
		"portal HITL",
		"plugins dogfood",
		"Agent Plugins GA",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("portal handoff missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent dual_write ON / Connected: %s", out)
	}
	if strings.Contains(out, "Memory GA shipped") {
		t.Fatalf("must not invent Memory GA: %s", out)
	}
}

// s1368: AionAgentOnboardingStatus residual-honest offline static needles.
func TestAionAgentOnboardingStatus_HonestyNeedles(t *testing.T) {
	out := AionAgentOnboardingStatus()
	if out == "" {
		t.Fatal("empty status")
	}
	for _, want := range []string{
		"MCP attach",
		"fail-open offline",
		"never invent tool green",
		"dual_write OFF",
		"local-primary",
		"not Memory GA",
		"book-demo OFF",
		"portal HITL",
		"console.iome.sh/settings/agent",
		"console.iome.sh/integrations",
		"never invent install green",
		"Connected",
		"INSTALL_STORE APPLY",
		"empty-as-none",
		"catalog ≠ Connected",
		"agent MCP cannot write installs",
		"plugins dogfood",
		"Agent Plugins GA",
		"residual PASS ≠ live dogfood",
		"probe only",
		"/onboard portal",
		"/onboard checklist",
		"/integrations status",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("status missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent dual_write ON / Connected: %s", out)
	}
}

// s1363: AttachMCP injects <aion-onboarding> system note when MCP manager is present.
// Mirrors TestAttachMCP_InjectsIntegrationsGuidance / InjectsMemoryAdvancedGuidance.
func TestAttachMCP_InjectsAionOnboardingGuidance(t *testing.T) {
	cInR, cInW := io.Pipe()
	cOutR, cOutW := io.Pipe()
	go mockIntegrationsMCP(cOutW, cInR)

	mut := false
	cl := mcp.NewClientForTest(mcp.ServerConfig{Name: "aion-scenario", Command: "x", Mutating: &mut}, cInW, cOutR, nil)
	defer cl.Close()
	if err := cl.InitForTest(context.Background()); err != nil {
		t.Fatal(err)
	}

	mgr := mcp.NewManagerEmpty(nil)
	mgr.Attach(cl)
	rt := testRT(t, t.TempDir())
	rt.AttachMCP(mgr)

	sys := rt.Messages()[0].Content
	if !strings.Contains(sys, "<aion-onboarding>") {
		t.Fatalf("want <aion-onboarding> system note: %s", sys)
	}
	if !strings.Contains(sys, "</aion-onboarding>") {
		t.Fatalf("want closed aion-onboarding tag: %s", sys)
	}
	for _, want := range []string{
		"list_connector_catalog",
		"plan_connector_setup",
		"dual_write OFF",
		"not Memory GA",
		"never invent install green",
		"portal HITL",
		"aion-agent-onboarding",
		// s1368 needles in injected note
		"console.iome.sh/settings/agent",
		"Agent/MCP",
		"[[mcp.servers]]",
	} {
		if !strings.Contains(sys, want) {
			t.Fatalf("aion-onboarding note missing %q: %s", want, sys)
		}
	}
	// Co-inject with integrations + memory-advanced (s1251 + s1291 + s1363).
	if !strings.Contains(sys, "<integrations>") {
		t.Fatalf("want <integrations> note too: %s", sys)
	}
	if !strings.Contains(sys, "<memory-advanced>") {
		t.Fatalf("want <memory-advanced> note too: %s", sys)
	}
	if !strings.Contains(sys, "<mcp>") {
		t.Fatalf("want <mcp> note too: %s", sys)
	}

	// Builtin skill name is loadable (companion skill always present when skills on).
	cat, err := skills.LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cat.Get("aion-agent-onboarding"); !ok {
		t.Fatalf("want builtin aion-agent-onboarding; names=%v", cat.Names())
	}
}
