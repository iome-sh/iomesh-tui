package agent

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/mcp"
	"github.com/iome-sh/iomesh-tui/internal/skills"
)

// s1363+s1368+s1372: AionAgentOnboardingGuidanceNote residual-honest needles.
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
		// s1372 post-onboard continuum cross-link
		"/onboard next",
		"drafts only",
		"no auto-send",
		"package load ≠ Memory GA",
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

// s1363+s1368+s1372: AionAgentOnboardingChecklist residual-honest numbered needles.
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
		// s1372 post-onboard continuum cross-link
		"/onboard next",
		"drafts only",
		"no auto-send",
		"package load ≠ Memory GA",
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

// s1368+s1372+s1382: AionAgentOnboardingStatus residual-honest offline static needles.
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
		// s1372 cross-link
		"/onboard next",
		// s1382 cross-link to lane status board
		"/onboard next status",
		// s1387 cross-link to status export receipt
		"/onboard next export",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("status missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent dual_write ON / Connected: %s", out)
	}
}

// s1372+s1377+s1382: AionAgentOnboardingNextLanes residual-honest post-onboard continuum needles.
func TestAionAgentOnboardingNextLanes_HonestyNeedles(t *testing.T) {
	out := AionAgentOnboardingNextLanes()
	if out == "" {
		t.Fatal("empty next lanes")
	}
	for _, want := range []string{
		"onboard next lanes",
		"post-onboard continuum",
		"no MCP dial",
		"1.",
		"iomesh plugins dogfood",
		"offline sample validate",
		"Agent Plugins GA",
		"/onboard next plugins",
		"2.",
		"/gtm checklist",
		"gtm-draft-only-agent",
		"drafts only",
		"no auto-send",
		"human publish",
		"GTM agent GA",
		"/onboard next gtm",
		"3.",
		"aion-memory-mcp",
		"Memory Ops Pack",
		"local-primary",
		"dual_write OFF",
		"package load ≠ Memory GA",
		"freemium palace",
		"/onboard next memory",
		"4.",
		"portal HITL",
		"agent MCP cannot write installs",
		"catalog ≠ Connected",
		"book-demo OFF",
		"not Memory GA",
		"residual PASS ≠ live dogfood",
		"never invent install green",
		"Connected",
		"INSTALL_STORE APPLY",
		"empty-as-none",
		"plugins dogfood",
		"~$88/$119",
		// s1382 cross-link to lane status board
		"/onboard next status",
		"status board",
		// s1387 cross-link to status export receipt
		"/onboard next export",
		"export receipt",
		"board/export evidence ≠ invent Connected",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("next lanes missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent dual_write ON / Connected: %s", out)
	}
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "Agent Plugins GA shipped") {
		t.Fatalf("must not invent Memory/Plugins GA: %s", out)
	}
	if strings.Contains(out, "auto-send enabled") || strings.Contains(out, "GTM agent GA shipped") {
		t.Fatalf("must not invent auto-send / GTM agent GA: %s", out)
	}
}

// s1377: AionAgentOnboardingNextPluginsLane residual-honest plugins dogfood drill needles.
func TestAionAgentOnboardingNextPluginsLane_HonestyNeedles(t *testing.T) {
	out := AionAgentOnboardingNextPluginsLane()
	if out == "" {
		t.Fatal("empty plugins lane")
	}
	for _, want := range []string{
		"onboard next plugins lane",
		"no MCP dial",
		"iomesh plugins dogfood",
		"/plugins dogfood",
		"offline sample validate",
		"examples/agent-plugins",
		"hello-iome",
		"aion-memory-mcp",
		"iomesh plugins list",
		"iomesh plugins validate",
		"iomesh plugins dogfood",
		"plugins dogfood ≠ invent Agent Plugins GA",
		"Agent Plugins GA",
		"residual PASS ≠ live dogfood",
		"never invent install green",
		"Connected",
		"INSTALL_STORE APPLY",
		"catalog ≠ Connected",
		"agent MCP cannot write installs",
		"portal HITL",
		"package load ≠ Memory GA",
		"dual_write OFF",
		"book-demo OFF",
		"not Memory GA",
		"~$88/$119",
		"/onboard next",
		"/plugins status",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("plugins lane missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent dual_write ON / Connected: %s", out)
	}
	if strings.Contains(out, "Agent Plugins GA shipped") || strings.Contains(out, "Memory GA shipped") {
		t.Fatalf("must not invent Plugins/Memory GA: %s", out)
	}
}

// s1377: AionAgentOnboardingNextGtmLane residual-honest GTM draft-only drill needles.
func TestAionAgentOnboardingNextGtmLane_HonestyNeedles(t *testing.T) {
	out := AionAgentOnboardingNextGtmLane()
	if out == "" {
		t.Fatal("empty gtm lane")
	}
	for _, want := range []string{
		"onboard next gtm lane",
		"no MCP dial",
		"/gtm checklist",
		"gtm-draft-only-agent",
		"drafts only",
		"no auto-send",
		"human publish",
		"human CRM commercial",
		"read_skill gtm-draft-only-agent",
		"GTM checklist ≠ invent GTM agent GA",
		"GTM agent GA",
		"Salesforce",
		"HubSpot",
		"guerrilla",
		"portal HITL",
		"agent MCP cannot write installs",
		"never invent install green",
		"Connected",
		"INSTALL_STORE APPLY",
		"catalog ≠ Connected",
		"dual_write OFF",
		"book-demo OFF",
		"not Memory GA",
		"residual PASS ≠ live dogfood",
		"~$88/$119",
		"/onboard next",
		"/gtm",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("gtm lane missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent dual_write ON / Connected: %s", out)
	}
	if strings.Contains(out, "auto-send enabled") || strings.Contains(out, "GTM agent GA shipped") {
		t.Fatalf("must not invent auto-send / GTM agent GA: %s", out)
	}
}

// s1377: AionAgentOnboardingNextMemoryLane residual-honest memory local drill needles.
func TestAionAgentOnboardingNextMemoryLane_HonestyNeedles(t *testing.T) {
	out := AionAgentOnboardingNextMemoryLane()
	if out == "" {
		t.Fatal("empty memory lane")
	}
	for _, want := range []string{
		"onboard next memory lane",
		"no MCP dial",
		"aion-memory-mcp",
		"Memory Ops Pack",
		"local-primary",
		"dual_write OFF",
		"package load ≠ Memory GA",
		"freemium palace",
		"not Memory GA",
		"memory-advanced-agent",
		"/memory status",
		"/onboard status",
		"fail-open offline",
		"residual PASS ≠ live dogfood",
		"probe only",
		"never invent install green",
		"Connected",
		"INSTALL_STORE APPLY",
		"catalog ≠ Connected",
		"portal HITL",
		"agent MCP cannot write installs",
		"book-demo OFF",
		"console.iome.sh/settings/agent",
		"~$88/$119",
		"/onboard next",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("memory lane missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent dual_write ON / Connected: %s", out)
	}
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "freemium palace GA") {
		t.Fatalf("must not invent Memory GA / freemium palace: %s", out)
	}
}

// s1382: AionAgentOnboardingNextLaneStatus residual-honest lane status board needles.
func TestAionAgentOnboardingNextLaneStatus_HonestyNeedles(t *testing.T) {
	out := AionAgentOnboardingNextLaneStatus()
	if out == "" {
		t.Fatal("empty next lane status board")
	}
	// All four lanes named + honest vocabulary + locks (no invent Connected/GA/APPLY success).
	for _, want := range []string{
		"onboard next lane status",
		"no MCP dial",
		"not live dogfood",
		// lanes
		"plugins:",
		"gtm:",
		"memory:",
		"portal:",
		// honest state vocabulary (never invent connected/ga/apply as success)
		"dogfood_not_run",
		"path_ready",
		"skill_ready",
		"residual_only",
		"portal_hitl_still",
		// plugins honesty
		"Agent Plugins GA",
		"plugins dogfood ≠ invent Agent Plugins GA",
		"examples/agent-plugins",
		// gtm honesty
		"drafts only",
		"no auto-send",
		"GTM agent GA",
		"GTM checklist ≠ invent GTM agent GA",
		// memory honesty
		"dual_write OFF",
		"package load ≠ Memory GA",
		"local-primary",
		"freemium palace",
		"not Memory GA",
		// portal honesty
		"agent MCP cannot write installs",
		"catalog ≠ Connected",
		"portal HITL",
		// locks footer
		"book-demo OFF",
		"residual PASS ≠ live dogfood",
		"never invent install green",
		"Connected",
		"INSTALL_STORE APPLY",
		"~$88/$119",
		// cross-links
		"/onboard next status",
		"/onboard next export",
		"/onboard next plugins",
		"/onboard next gtm",
		"/onboard next memory",
		"/onboard next",
		"/onboard status",
		// s1392: /plugins dogfood soft offline cross-link
		"/plugins dogfood",
		"/plugins status",
		// s1387: board → export cross-link honesty
		"board/export evidence ≠ invent Connected",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("next lane status missing %q in:\n%s", want, out)
		}
	}
	// Soft samples state must be one of the honest values (from module-root soft check).
	if !strings.Contains(out, "samples_ok") && !strings.Contains(out, "samples_missing") {
		t.Fatalf("next lane status must report samples_ok or samples_missing:\n%s", out)
	}
	// Must not invent product success language.
	if strings.Contains(out, "Connected: yes") || strings.Contains(out, "dual_write ON") {
		t.Fatalf("must not invent Connected/dual_write ON: %s", out)
	}
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "Agent Plugins GA shipped") {
		t.Fatalf("must not invent Memory/Plugins GA: %s", out)
	}
	if strings.Contains(out, "GTM agent GA shipped") || strings.Contains(out, "auto-send enabled") {
		t.Fatalf("must not invent GTM agent GA / auto-send: %s", out)
	}
	if strings.Contains(out, "INSTALL_STORE APPLY success") || strings.Contains(out, "APPLY: ok") {
		t.Fatalf("must not invent APPLY success: %s", out)
	}
	// Never claim dogfood was run / live dogfood green from this static board.
	if strings.Contains(out, "dogfood_run") || strings.Contains(out, "dogfood PASS live") {
		t.Fatalf("must not invent dogfood run/live PASS: %s", out)
	}
}

// s1387: AionAgentOnboardingNextLaneStatusExport residual-honest markdown export receipt needles.
func TestAionAgentOnboardingNextLaneStatusExport_HonestyNeedles(t *testing.T) {
	out := AionAgentOnboardingNextLaneStatusExport()
	if out == "" {
		t.Fatal("empty next lane status export receipt")
	}
	for _, want := range []string{
		// evidence header
		"evidence_kind=onboard_next_lane_status_export",
		"offline_static",
		"not_live_dogfood",
		"serial=s1387",
		"format=markdown",
		"export receipt",
		// lanes + s1382 vocabulary only
		"plugins:",
		"gtm:",
		"memory:",
		"portal:",
		"dogfood_not_run",
		"path_ready",
		"skill_ready",
		"residual_only",
		"portal_hitl_still",
		// honesty locks
		"dual_write OFF",
		"book-demo OFF",
		"not Memory GA",
		"residual PASS ≠ live dogfood",
		"never invent install green",
		"Connected",
		"INSTALL_STORE APPLY",
		"catalog ≠ Connected",
		"portal HITL",
		"agent MCP cannot write installs",
		"plugins dogfood ≠ invent Agent Plugins GA",
		"drafts only",
		"no auto-send",
		"GTM checklist ≠ invent GTM agent GA",
		"package load ≠ Memory GA",
		"board/export evidence ≠ invent Connected",
		"~$88/$119",
		// does not run dogfood / dial MCP
		"does NOT run plugins dogfood",
		"does NOT dial MCP",
		// slash
		"/onboard next export",
		"/onboard next status",
		// s1392: /plugins dogfood soft offline cross-link
		"/plugins dogfood",
		"/plugins status",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("export receipt missing %q in:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "samples_ok") && !strings.Contains(out, "samples_missing") {
		t.Fatalf("export receipt must report samples_ok or samples_missing:\n%s", out)
	}
	// Must not invent product success language.
	if strings.Contains(out, "Connected: yes") || strings.Contains(out, "dual_write ON") {
		t.Fatalf("must not invent Connected/dual_write ON: %s", out)
	}
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "Agent Plugins GA shipped") {
		t.Fatalf("must not invent Memory/Plugins GA: %s", out)
	}
	if strings.Contains(out, "INSTALL_STORE APPLY success") || strings.Contains(out, "APPLY: ok") {
		t.Fatalf("must not invent APPLY success: %s", out)
	}
	if strings.Contains(out, "dogfood_run") || strings.Contains(out, "dogfood PASS live") {
		t.Fatalf("must not invent dogfood run/live PASS: %s", out)
	}
}

// s1387: AionAgentOnboardingNextLaneStatusExportJSON residual-honest JSON export needles.
func TestAionAgentOnboardingNextLaneStatusExportJSON_HonestyNeedles(t *testing.T) {
	out := AionAgentOnboardingNextLaneStatusExportJSON()
	if out == "" {
		t.Fatal("empty next lane status export JSON")
	}
	for _, want := range []string{
		`"evidence_kind": "onboard_next_lane_status_export"`,
		`"offline_static": true`,
		`"not_live_dogfood": true`,
		`"serial": "s1387"`,
		`"format": "json"`,
		`"dogfood_not_run": true`,
		"path_ready",
		"skill_ready",
		"residual_only",
		"portal_hitl_still",
		"dual_write OFF",
		"not Memory GA",
		"board/export evidence ≠ invent Connected",
		"never invent install green / Connected / INSTALL_STORE APPLY",
		"/onboard next export",
		"/plugins dogfood",
		"/plugins status",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("export JSON missing %q in:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "samples_ok") && !strings.Contains(out, "samples_missing") {
		t.Fatalf("export JSON must report samples_ok or samples_missing:\n%s", out)
	}
	if strings.Contains(out, "Connected: yes") || strings.Contains(out, "dual_write ON") {
		t.Fatalf("must not invent Connected/dual_write ON: %s", out)
	}
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "INSTALL_STORE APPLY success") {
		t.Fatalf("must not invent Memory GA / APPLY success: %s", out)
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
