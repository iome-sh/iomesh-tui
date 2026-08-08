package agent

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/agentplugins"
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
		// s1402 mesh streaming lane
		"/onboard next mesh",
		"mesh = streaming org heartbeats",
		"mesh ≠ memory",
		// s1407 Ops Pack pull path
		"/onboard next memory-pull",
		"pull_not_probed",
		"Ops Pack ≠ GPU fleet",
		"pull ≠ freemium hosted palace",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("status missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent dual_write ON / Connected: %s", out)
	}
}

// s1372+s1377+s1382+s1402+s1407+s1413+s1417: AionAgentOnboardingNextLanes residual-honest post-onboard continuum needles.
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
		// s1402 mesh streaming lane
		"4.",
		"I/O Mesh",
		"streaming org heartbeats",
		"dept.*",
		"mesh ≠ memory",
		"not OTel/APM",
		"/onboard next mesh",
		"streams_not_probed",
		// s1407 Ops Pack pull path
		"5.",
		"Memory Ops Pack pull path",
		"iomesh memory pull",
		"mesh → local palace",
		"Ops Pack ≠ GPU fleet",
		"pull_not_probed",
		"/onboard next memory-pull",
		"ops-pack|pull-path|memorypull|ops_pack",
		"pull stays mesh",
		// s1417 agentic integrations product plane 3
		"6.",
		"agentic integrations",
		"product plane 3",
		"MCP list/plan residual-honest",
		"plan_connector_setup",
		"/onboard next agentic",
		"agentic-integrations|integrations|portal-hitl|list-plan|hitl",
		"list_plan_not_connected",
		// portal HITL still
		"7.",
		"portal HITL",
		"agent MCP cannot write installs",
		"catalog ≠ Connected",
		// s1413 human-gates
		"8.",
		"human-gates",
		"/onboard next human-gates",
		"human|gates|apply-gates",
		"PASS ≠ invent human-gate green",
		"PASS ≠ live APPLY",
		"open boxes stay open",
		"Slack HMAC",
		"Stripe Customers:Write",
		"H1/H2 INSTALL_STORE",
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
		// s1432 three product planes cross-link
		"/onboard next planes",
		"three product planes",
		"three-planes|product-planes|product|pillars|three_planes",
		"dual_auth_candidacy_open",
		// pulse stays board (not mesh alias)
		"pulse stays board",
		"never invent pull green",
		"pull ≠ freemium hosted palace",
		"Knowledge Beta→GA cannot invent H1/H2 offline",
		"leave ON_SIGNAL unset",
		"template= ≠ install APPLY",
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
	if strings.Contains(out, "stream green: yes") || strings.Contains(out, "streams Connected") {
		t.Fatalf("must not invent stream green: %s", out)
	}
	if strings.Contains(out, "pull green: yes") {
		t.Fatalf("must not invent pull green: %s", out)
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
		"mesh ≠ memory",
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

// s1402: AionAgentOnboardingNextMeshLane residual-honest mesh streaming lane needles.
func TestAionAgentOnboardingNextMeshLane_HonestyNeedles(t *testing.T) {
	out := AionAgentOnboardingNextMeshLane()
	if out == "" {
		t.Fatal("empty mesh lane")
	}
	for _, want := range []string{
		"onboard next mesh lane",
		"no MCP dial",
		"product plane 1",
		"I/O Mesh",
		"streaming org heartbeats",
		"dept.*",
		"not hosted Memory Palace",
		"not OTel/APM",
		"mesh ≠ memory",
		"Palace sunset",
		"/mesh",
		"iomesh mesh status",
		"iomesh mesh streams",
		"iomesh mesh consumer",
		"streams_not_probed",
		"empty streams honest",
		"never invent stream green",
		"iomesh memory pull",
		"mesh → local palace",
		"dual_write OFF",
		"not freemium hosted palace",
		"not Memory GA",
		"book-demo OFF",
		"residual PASS ≠ live dogfood",
		"PASS ≠ live APPLY",
		"~$88",
		"~$119",
		"Memory Ops Pack",
		"catalog ≠ Connected",
		"portal HITL",
		"agent MCP cannot write installs",
		"/onboard next mesh",
		"stream|streams|heartbeat|heartbeats|pull",
		"NOT pulse",
		"pulse stays",
		"/onboard next status",
		"/onboard next memory",
		"not medical",
		"board/export evidence ≠ invent Connected",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("mesh lane missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent dual_write ON / Connected: %s", out)
	}
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "stream green: yes") {
		t.Fatalf("must not invent Memory GA / stream green: %s", out)
	}
	if strings.Contains(out, "freemium hosted palace ON") || strings.Contains(out, "OTel/APM green") {
		t.Fatalf("must not invent freemium palace / OTel green: %s", out)
	}
}

// s1407: AionAgentOnboardingNextMemoryPullLane residual-honest Ops Pack pull path needles.
func TestAionAgentOnboardingNextMemoryPullLane_HonestyNeedles(t *testing.T) {
	out := AionAgentOnboardingNextMemoryPullLane()
	if out == "" {
		t.Fatal("empty memory-pull lane")
	}
	for _, want := range []string{
		"onboard next memory-pull lane",
		"no MCP dial",
		"Ops Pack pull path",
		"iomesh memory pull",
		"mesh → local palace",
		"CreateConsumer",
		"memory_ingest_turn",
		"dual_write OFF",
		"not freemium hosted palace",
		"not Memory GA",
		"Palace sunset",
		"Ops Pack ≠ GPU fleet",
		"~$119",
		"~$88",
		"Memory Ops Pack",
		"package load ≠ Ops Pack entitlement",
		"package load ≠ Memory GA",
		"pull_not_probed",
		"never invent pull green",
		"residual PASS ≠ live dogfood",
		"PASS ≠ live APPLY",
		"book-demo OFF",
		"mesh ≠ memory",
		"catalog ≠ Connected",
		"portal HITL",
		"agent MCP cannot write installs",
		"/onboard next memory-pull",
		"ops-pack|pull-path|memorypull|ops_pack",
		"bare pull stays mesh",
		"/onboard next mesh",
		"/onboard next memory",
		"/onboard next status",
		"board/export evidence ≠ invent Connected",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("memory-pull lane missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent dual_write ON / Connected: %s", out)
	}
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "pull green: yes") {
		t.Fatalf("must not invent Memory GA / pull green: %s", out)
	}
	if strings.Contains(out, "GPU fleet green") || strings.Contains(out, "freemium hosted palace ON") {
		t.Fatalf("must not invent GPU fleet / freemium palace: %s", out)
	}
}

// s1417+s1422: AionAgentOnboardingNextAgenticLane residual-honest product plane 3 agentic integrations needles.
func TestAionAgentOnboardingNextAgenticLane_HonestyNeedles(t *testing.T) {
	ResetAgenticListPlanSoftDogfoodSessionState()
	t.Cleanup(ResetAgenticListPlanSoftDogfoodSessionState)

	out := AionAgentOnboardingNextAgenticLane()
	if out == "" {
		t.Fatal("empty agentic lane")
	}
	for _, want := range []string{
		"onboard next agentic lane",
		"no MCP dial",
		"product plane 3",
		"agentic integrations",
		"MCP list/plan residual-honest",
		"plan_connector_setup",
		"portal deep links",
		"browser HITL only",
		"template= ≠ install APPLY",
		"deep_links = browser HITL only",
		"/integrations/{id}",
		"/integrations/add?template={id}",
		"/integrations",
		"list_org fail-open ≠ empty-as-none",
		"available=false default residual",
		"agent MCP cannot write installs",
		"catalog ≠ Connected",
		"never invent Connected",
		"console.iome.sh/integrations",
		"console.iome.sh/settings/agent",
		"dual_write OFF",
		"book-demo OFF",
		"not Memory GA",
		"residual PASS ≠ live dogfood",
		"PASS ≠ live APPLY",
		"open boxes stay open",
		"rates ~$88/$119 optional",
		"path_ready",
		"residual_only",
		"portal_hitl_still",
		"list_plan_not_connected",
		"/onboard next agentic",
		"agentic-integrations|integrations|portal-hitl|list-plan|hitl",
		"bare mcp",
		"memory lane",
		"portal handoff",
		"does not claim dual-auth live for list_org",
		"board/export evidence ≠ invent Connected",
		// s1422 portal HITL polish + soft dogfood
		"Portal HITL polish",
		"list_plan_soft_not_run",
		"/onboard next agentic dogfood",
		"soft|samples|offline|list-plan-soft",
		"soft offline list/plan ≠ live dogfood",
		"session soft ≠ live dogfood",
		"soft offline ≠ live dogfood",
		"/onboard portal mint/copy/probe",
		// s1427 dual-auth candidacy tip on main board
		"dual_auth_candidacy_open",
		"/onboard next agentic dual-auth",
		"candidacy|list-org|org-installs|dual_auth|dual-auth-candidacy",
		"tool ship ≠ dual-auth live",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("agentic lane missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent dual_write ON / Connected: %s", out)
	}
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "INSTALL_STORE APPLY success") {
		t.Fatalf("must not invent Memory GA / INSTALL_STORE APPLY: %s", out)
	}
	if strings.Contains(out, "install green: yes") || strings.Contains(out, "list_org Connected") {
		t.Fatalf("must not invent install green / list_org Connected: %s", out)
	}
}

// s1432: AionAgentOnboardingNextThreePlanes residual-honest three product planes board needles.
func TestAionAgentOnboardingNextThreePlanes_HonestyNeedles(t *testing.T) {
	out := AionAgentOnboardingNextThreePlanes()
	if out == "" {
		t.Fatal("empty three product planes board")
	}
	for _, want := range []string{
		"onboard next three product planes",
		"no MCP dial",
		"s1432",
		// plane 1 mesh
		"plane 1",
		"Mesh",
		"product plane 1",
		"streaming org heartbeats",
		"dept.*",
		"mesh ≠ memory",
		"not OTel/APM",
		"streams_not_probed",
		"never invent stream green",
		"/onboard next mesh",
		// plane 2 memory-pull
		"plane 2",
		"Memory-pull",
		"Ops Pack",
		"product plane 2",
		"mesh → local palace egress",
		"dual_write OFF",
		"Ops Pack ≠ GPU",
		"pull_not_probed",
		"never invent pull green",
		"/onboard next memory-pull",
		// plane 3 agentic
		"plane 3",
		"Agentic integrations",
		"product plane 3",
		"MCP list/plan residual-honest",
		"portal_hitl_still",
		"list_plan_not_connected",
		"dual_auth_candidacy_open",
		"agent MCP cannot write installs",
		"never invent Connected",
		"tool ship ≠ dual-auth live",
		"/onboard next agentic",
		"/onboard next agentic dual-auth",
		"/onboard next agentic dogfood",
		// honest vocab shared
		"path_ready",
		"residual_only",
		// rates + gates
		"~$88",
		"~$119",
		"book-demo OFF",
		"not Memory GA",
		"residual PASS ≠ live dogfood",
		"PASS ≠ live APPLY",
		"open boxes stay open",
		// slash + aliases
		"/onboard next planes",
		"three-planes|product-planes|product|pillars|three_planes",
		"pulse|board",
		// cross-links
		"/onboard next status",
		"/onboard next export",
		"/onboard next human-gates",
		// do not steal
		"bare pull stays mesh",
		"NOT bare mcp",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("three planes board missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent dual_write ON / Connected: %s", out)
	}
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "stream green: yes") {
		t.Fatalf("must not invent Memory GA / stream green: %s", out)
	}
	if strings.Contains(out, "pull green: yes") || strings.Contains(out, "dual-auth live: yes") {
		t.Fatalf("must not invent pull green / dual-auth live: %s", out)
	}
	if strings.Contains(out, "INSTALL_STORE APPLY success") {
		t.Fatalf("must not invent INSTALL_STORE APPLY success: %s", out)
	}
}

// s1427: AionAgentOnboardingNextAgenticDualAuthCandidacy residual-honest dual-auth candidacy needles.
func TestAionAgentOnboardingNextAgenticDualAuthCandidacy_HonestyNeedles(t *testing.T) {
	out := AionAgentOnboardingNextAgenticDualAuthCandidacy()
	if out == "" {
		t.Fatal("empty dual-auth candidacy board")
	}
	for _, want := range []string{
		"onboard next agentic dual-auth candidacy",
		"no MCP dial",
		"product plane 3",
		"dual_auth_candidacy_open",
		"list_org_connector_installs",
		"available=false",
		"status=unavailable",
		"installs=null",
		"never invent empty-as-none",
		"installs=null not []",
		"tool ship ≠ dual-auth live",
		"PASS ≠ invent dual-auth shipped",
		"never invent dual-auth live",
		"portal session owns install index",
		"session-cookie + org membership only",
		"agent MCP cannot write installs",
		"catalog ≠ Connected",
		"never invent Connected",
		"console.iome.sh/integrations",
		"dual_write OFF",
		"book-demo OFF",
		"not Memory GA",
		"residual PASS ≠ live dogfood",
		"PASS ≠ live APPLY",
		"open boxes stay open",
		"rates ~$88/$119 optional",
		"path_ready",
		"residual_only",
		"list_org_unavailable",
		"/onboard next agentic dual-auth",
		"candidacy|list-org|org-installs|dual_auth|dual-auth-candidacy",
		"/onboard next agentic",
		"/onboard next agentic dogfood",
		"/onboard portal",
		"/onboard next status",
		"dogfood|soft|samples|offline|list-plan-soft",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("dual-auth candidacy missing %q in:\n%s", want, out)
		}
	}
	// Anti-inventions
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent dual_write ON / Connected: %s", out)
	}
	if strings.Contains(out, "dual-auth live shipped") || strings.Contains(out, "dual-auth: live") {
		t.Fatalf("must not invent dual-auth live: %s", out)
	}
	if strings.Contains(out, "installs: []") || strings.Contains(out, `"installs":[]`) {
		t.Fatalf("must not invent empty-as-none installs=[]: %s", out)
	}
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "INSTALL_STORE APPLY success") {
		t.Fatalf("must not invent Memory GA / INSTALL_STORE APPLY: %s", out)
	}
	// Must not be main agentic board body or soft dogfood runner.
	if strings.Contains(out, "onboard next agentic lane (") {
		t.Fatalf("dual-auth board must not be main agentic lane body: %s", out)
	}
	if strings.Contains(out, "agentic list/plan soft offline dogfood") {
		t.Fatalf("dual-auth board must not be soft dogfood runner: %s", out)
	}
}

// s1422: status/export reflect agentic list/plan session soft after dogfood.
func TestAionAgentOnboardingNextLaneStatus_AgenticListPlanSoftDogfood(t *testing.T) {
	ResetAgenticListPlanSoftDogfoodSessionState()
	agentplugins.ResetSoftDogfoodSessionState()
	t.Cleanup(func() {
		ResetAgenticListPlanSoftDogfoodSessionState()
		agentplugins.ResetSoftDogfoodSessionState()
	})

	// Default: list_plan_soft_not_run on agentic row
	out := AionAgentOnboardingNextLaneStatus()
	if !strings.Contains(out, "list_plan_soft_not_run") {
		t.Fatalf("default status want list_plan_soft_not_run:\n%s", out)
	}
	if !strings.Contains(out, "list_plan_not_connected") {
		t.Fatalf("default status still want list_plan_not_connected:\n%s", out)
	}
	if strings.Contains(out, "soft_offline_list_plan_session_pass") || strings.Contains(out, "soft_offline_list_plan_session_fail") {
		t.Fatalf("default status must not show agentic soft pass/fail:\n%s", out)
	}
	// Plugins default independent
	if !strings.Contains(out, "dogfood_not_run") {
		t.Fatalf("plugins default dogfood_not_run still required:\n%s", out)
	}

	// Soft pass via runner
	_ = RunAgenticListPlanSoftDogfood()
	out = AionAgentOnboardingNextLaneStatus()
	if !strings.Contains(out, "soft_offline_list_plan_session_pass") {
		t.Fatalf("after pass want soft_offline_list_plan_session_pass:\n%s", out)
	}
	if strings.Contains(out, "· list_plan_soft_not_run") {
		t.Fatalf("after pass agentic lane must not hardcode list_plan_soft_not_run as state:\n%s", out)
	}
	for _, want := range []string{
		"session soft list/plan ≠ live dogfood",
		"soft offline ≠ invent Connected",
		"list_plan_not_connected",
		"portal_hitl_still",
		"not live dogfood",
		"/onboard next agentic dogfood",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("pass status missing honesty %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Connected: yes") || strings.Contains(out, "live dogfood green") {
		t.Fatalf("pass status must not invent Connected/live:\n%s", out)
	}
	// Plugins soft still independent default
	if !strings.Contains(out, "dogfood_not_run") {
		t.Fatalf("agentic soft must not steal plugins dogfood marker:\n%s", out)
	}

	// Soft fail
	SetAgenticListPlanSoftDogfoodSessionState(false)
	out = AionAgentOnboardingNextLaneStatus()
	if !strings.Contains(out, "soft_offline_list_plan_session_fail") {
		t.Fatalf("after fail want soft_offline_list_plan_session_fail:\n%s", out)
	}
	if strings.Contains(out, "soft_offline_list_plan_session_pass") {
		t.Fatalf("after fail must not show pass:\n%s", out)
	}

	// Export markdown + JSON follow session
	exp := AionAgentOnboardingNextLaneStatusExport()
	if !strings.Contains(exp, "soft_offline_list_plan_session_fail") {
		t.Fatalf("export after fail want session fail:\n%s", exp)
	}
	if !strings.Contains(exp, "session soft list/plan ≠ live dogfood") {
		t.Fatalf("export missing agentic session soft honesty:\n%s", exp)
	}
	if strings.Contains(exp, "Connected: yes") {
		t.Fatalf("export must not invent Connected:\n%s", exp)
	}

	js := AionAgentOnboardingNextLaneStatusExportJSON()
	if !strings.Contains(js, `"agentic_list_plan_soft_state": "soft_offline_list_plan_session_fail"`) {
		t.Fatalf("export JSON want agentic_list_plan_soft_state fail:\n%s", js)
	}
	if !strings.Contains(js, "soft_offline_list_plan_session_fail") {
		t.Fatalf("export JSON agentic lane want soft fail:\n%s", js)
	}
	if !strings.Contains(js, "session soft ≠ live dogfood") {
		t.Fatalf("export JSON missing session soft honesty:\n%s", js)
	}
	if strings.Contains(js, "Connected: yes") {
		t.Fatalf("export JSON must not invent Connected:\n%s", js)
	}
}

// s1413: AionAgentHumanGatesHonestyBoard residual-honest still-required vs offline needles.
func TestAionAgentHumanGatesHonestyBoard_HonestyNeedles(t *testing.T) {
	out := AionAgentHumanGatesHonestyBoard()
	if out == "" {
		t.Fatal("empty human-gates honesty board")
	}
	for _, want := range []string{
		"human-gates honesty board",
		"no MCP dial",
		"not live APPLY",
		// sections
		"still_human",
		"offline_residual_only",
		"shipped_or_policy",
		"do_not_close",
		// still human APPLY residuals
		"Slack HMAC",
		"Stripe Customers:Write",
		"H1/H2 INSTALL_STORE",
		"D1–D5",
		"book-demo OFF",
		"ON_SIGNAL",
		"leave ON_SIGNAL unset",
		// offline residual only
		"residual gates",
		"soft dogfood",
		"agent MCP list/plan",
		"dry-run",
		"dry-run ≠ APPLY",
		// shipped / policy
		"GitHub App HMAC",
		"dogfood-proven",
		"dual_write OFF",
		"Palace sunset",
		"analytical NO-install intentional",
		// do not close human APPLY
		"do NOT close human APPLY gates",
		"local memory",
		// honesty locks
		"PASS ≠ invent human-gate green",
		"PASS ≠ live APPLY",
		"open boxes stay open",
		"Knowledge Beta→GA cannot invent H1/H2 offline",
		"not Memory GA",
		"never invent APPLY",
		"catalog ≠ Connected",
		"portal HITL",
		"agent MCP cannot write installs",
		"board/export evidence ≠ invent Connected",
		"rates ~$88/$119 optional",
		// operator
		"make human-gates-status",
		"/onboard next human-gates",
		"human|gates|apply-gates",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("human-gates board missing %q in:\n%s", want, out)
		}
	}
	// Forbidden invent tokens
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "book-demo ON") {
		t.Fatalf("must not invent dual_write ON / book-demo ON: %s", out)
	}
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent Memory GA / Connected green: %s", out)
	}
	if strings.Contains(out, "H1/H2 green: yes") || strings.Contains(out, "INSTALL_STORE APPLY: done") {
		t.Fatalf("must not invent H1/H2 / INSTALL_STORE green: %s", out)
	}
	if strings.Contains(out, "human-gate green: yes") || strings.Contains(out, "ON_SIGNAL=1") {
		t.Fatalf("must not invent human-gate green / ON_SIGNAL set: %s", out)
	}
}

// s1382: AionAgentOnboardingNextLaneStatus residual-honest lane status board needles.
func TestAionAgentOnboardingNextLaneStatus_HonestyNeedles(t *testing.T) {
	agentplugins.ResetSoftDogfoodSessionState()
	ResetAgenticListPlanSoftDogfoodSessionState()
	t.Cleanup(func() {
		agentplugins.ResetSoftDogfoodSessionState()
		ResetAgenticListPlanSoftDogfoodSessionState()
	})

	out := AionAgentOnboardingNextLaneStatus()
	if out == "" {
		t.Fatal("empty next lane status board")
	}
	// All lanes named + honest vocabulary + locks (no invent Connected/GA/APPLY/stream/pull green success).
	for _, want := range []string{
		"onboard next lane status",
		"no MCP dial",
		"not live dogfood",
		// lanes
		"plugins:",
		"gtm:",
		"memory:",
		"mesh:",
		"memory-pull:",
		"agentic:",
		"portal:",
		// honest state vocabulary (never invent connected/ga/apply/stream/pull green as success)
		"dogfood_not_run",
		"list_plan_soft_not_run",
		"path_ready",
		"skill_ready",
		"residual_only",
		"streams_not_probed",
		"pull_not_probed",
		"portal_hitl_still",
		"list_plan_not_connected",
		// plugins honesty
		"Agent Plugins GA",
		"plugins dogfood ≠ invent Agent Plugins GA",
		"session soft marker ≠ live dogfood",
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
		"mesh ≠ memory",
		// mesh honesty (s1402)
		"streaming org heartbeats",
		"not OTel/APM",
		"never invent stream green",
		"/onboard next mesh",
		// memory-pull honesty (s1407)
		"Ops Pack pull path",
		"Ops Pack ≠ GPU fleet",
		"never invent pull green",
		"/onboard next memory-pull",
		"package load ≠ Ops Pack entitlement",
		// agentic honesty (s1417+s1427)
		"product plane 3",
		"MCP list/plan residual-honest",
		"plan_connector_setup",
		"/onboard next agentic",
		"template= ≠ install APPLY",
		"dual_auth_candidacy_open",
		"list_org_unavailable",
		"tool ship ≠ dual-auth live",
		"/onboard next agentic dual-auth",
		// portal honesty
		"agent MCP cannot write installs",
		"catalog ≠ Connected",
		"portal HITL",
		// locks footer
		"book-demo OFF",
		"residual PASS ≠ live dogfood",
		"session soft ≠ live dogfood",
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
	if strings.Contains(out, "stream green: yes") || strings.Contains(out, "streams Connected") {
		t.Fatalf("must not invent stream green: %s", out)
	}
	if strings.Contains(out, "pull green: yes") {
		t.Fatalf("must not invent pull green: %s", out)
	}
}

// s1397: status/export reflect session soft dogfood pass/fail (≠ live dogfood · ≠ invent GA/Connected).
func TestAionAgentOnboardingNextLaneStatus_SessionSoftDogfood(t *testing.T) {
	agentplugins.ResetSoftDogfoodSessionState()
	t.Cleanup(agentplugins.ResetSoftDogfoodSessionState)

	// Default: dogfood_not_run
	out := AionAgentOnboardingNextLaneStatus()
	if !strings.Contains(out, "dogfood_not_run") {
		t.Fatalf("default status want dogfood_not_run:\n%s", out)
	}
	if strings.Contains(out, "soft_offline_dogfood_session_pass") || strings.Contains(out, "soft_offline_dogfood_session_fail") {
		t.Fatalf("default status must not show session pass/fail:\n%s", out)
	}

	// Soft pass → session pass label
	agentplugins.SetSoftDogfoodSessionState(true)
	out = AionAgentOnboardingNextLaneStatus()
	if !strings.Contains(out, "soft_offline_dogfood_session_pass") {
		t.Fatalf("after pass want soft_offline_dogfood_session_pass:\n%s", out)
	}
	// Avoid matching residual honesty prose that still mentions "dogfood_not_run default".
	if strings.Contains(out, "· dogfood_not_run ·") {
		t.Fatalf("after pass plugins lane must not hardcode dogfood_not_run:\n%s", out)
	}
	for _, want := range []string{
		"session soft marker ≠ live dogfood",
		"soft offline dogfood ≠ invent Agent Plugins GA",
		"board/export evidence ≠ invent Connected",
		"not live dogfood",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("pass status missing honesty %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Connected: yes") || strings.Contains(out, "Agent Plugins GA shipped") || strings.Contains(out, "live dogfood green") {
		t.Fatalf("pass status must not invent Connected/GA/live:\n%s", out)
	}

	// Soft fail
	agentplugins.SetSoftDogfoodSessionState(false)
	out = AionAgentOnboardingNextLaneStatus()
	if !strings.Contains(out, "soft_offline_dogfood_session_fail") {
		t.Fatalf("after fail want soft_offline_dogfood_session_fail:\n%s", out)
	}
	if strings.Contains(out, "soft_offline_dogfood_session_pass") {
		t.Fatalf("after fail must not show pass:\n%s", out)
	}
	if strings.Contains(out, "Connected: yes") || strings.Contains(out, "Agent Plugins GA shipped") {
		t.Fatalf("fail status must not invent Connected/GA:\n%s", out)
	}

	// Export markdown + JSON follow session
	exp := AionAgentOnboardingNextLaneStatusExport()
	if !strings.Contains(exp, "soft_offline_dogfood_session_fail") {
		t.Fatalf("export after fail want session fail:\n%s", exp)
	}
	if !strings.Contains(exp, "session soft ≠ live dogfood") {
		t.Fatalf("export missing session soft honesty:\n%s", exp)
	}
	if strings.Contains(exp, "Connected: yes") || strings.Contains(exp, "Agent Plugins GA shipped") {
		t.Fatalf("export must not invent Connected/GA:\n%s", exp)
	}

	js := AionAgentOnboardingNextLaneStatusExportJSON()
	if !strings.Contains(js, `"plugins_dogfood_state": "soft_offline_dogfood_session_fail"`) {
		t.Fatalf("export JSON want plugins_dogfood_state fail:\n%s", js)
	}
	if !strings.Contains(js, `"dogfood_not_run": false`) {
		t.Fatalf("export JSON after session run want dogfood_not_run false:\n%s", js)
	}
	if !strings.Contains(js, "session soft ≠ live dogfood") {
		t.Fatalf("export JSON missing session soft honesty:\n%s", js)
	}
	if strings.Contains(js, "Connected: yes") || strings.Contains(js, "Agent Plugins GA shipped") {
		t.Fatalf("export JSON must not invent Connected/GA:\n%s", js)
	}

	// Pass on export
	agentplugins.SetSoftDogfoodSessionState(true)
	exp = AionAgentOnboardingNextLaneStatusExport()
	if !strings.Contains(exp, "soft_offline_dogfood_session_pass") {
		t.Fatalf("export after pass want session pass:\n%s", exp)
	}
	js = AionAgentOnboardingNextLaneStatusExportJSON()
	if !strings.Contains(js, `"plugins_dogfood_state": "soft_offline_dogfood_session_pass"`) {
		t.Fatalf("export JSON want plugins_dogfood_state pass:\n%s", js)
	}
}

// s1387: AionAgentOnboardingNextLaneStatusExport residual-honest markdown export receipt needles.
func TestAionAgentOnboardingNextLaneStatusExport_HonestyNeedles(t *testing.T) {
	agentplugins.ResetSoftDogfoodSessionState()
	ResetAgenticListPlanSoftDogfoodSessionState()
	t.Cleanup(func() {
		agentplugins.ResetSoftDogfoodSessionState()
		ResetAgenticListPlanSoftDogfoodSessionState()
	})

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
		// lanes + s1382+s1402+s1407+s1417+s1422 vocabulary only
		"plugins:",
		"gtm:",
		"memory:",
		"mesh:",
		"memory-pull:",
		"agentic:",
		"portal:",
		"dogfood_not_run",
		"list_plan_soft_not_run",
		"path_ready",
		"skill_ready",
		"residual_only",
		"streams_not_probed",
		"pull_not_probed",
		"portal_hitl_still",
		"list_plan_not_connected",
		// honesty locks
		"dual_write OFF",
		"book-demo OFF",
		"not Memory GA",
		"residual PASS ≠ live dogfood",
		"session soft ≠ live dogfood",
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
		// s1402 mesh honesty
		"mesh = streaming org heartbeats",
		"mesh ≠ memory",
		"never invent stream green",
		"not OTel/APM",
		// s1407 memory-pull honesty
		"Ops Pack ≠ GPU fleet",
		"never invent pull green",
		"pull ≠ freemium hosted palace",
		// s1417 agentic honesty
		"product plane 3",
		"plan_connector_setup",
		"/onboard next agentic",
		"template= ≠ install APPLY",
		// does not run dogfood / dial MCP
		"does NOT run plugins dogfood",
		"does NOT dial MCP",
		// slash
		"/onboard next export",
		"/onboard next status",
		"/onboard next mesh",
		"/onboard next memory-pull",
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
	agentplugins.ResetSoftDogfoodSessionState()
	ResetAgenticListPlanSoftDogfoodSessionState()
	t.Cleanup(func() {
		agentplugins.ResetSoftDogfoodSessionState()
		ResetAgenticListPlanSoftDogfoodSessionState()
	})

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
		`"plugins_dogfood_state": "dogfood_not_run"`,
		`"agentic_list_plan_soft_state": "list_plan_soft_not_run"`,
		"path_ready",
		"skill_ready",
		"residual_only",
		"streams_not_probed",
		"pull_not_probed",
		"portal_hitl_still",
		"list_plan_not_connected",
		`"mesh":`,
		`"memory-pull":`,
		`"ops_pack":`,
		`"agentic":`,
		"dual_write OFF",
		"not Memory GA",
		"session soft ≠ live dogfood",
		"soft offline list/plan ≠ live dogfood",
		"board/export evidence ≠ invent Connected",
		"never invent install green / Connected / INSTALL_STORE APPLY",
		"mesh = streaming org heartbeats",
		"mesh ≠ memory",
		"never invent stream green / Connected",
		"Ops Pack ≠ GPU fleet",
		"never invent pull green",
		"pull ≠ freemium hosted palace",
		"plan deep links = browser HITL only",
		"template= ≠ install APPLY",
		"/onboard next export",
		"/onboard next mesh",
		"/onboard next agentic dogfood",
		"/onboard next memory-pull",
		"/onboard next agentic",
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
	if strings.Contains(out, "stream green: yes") {
		t.Fatalf("must not invent stream green: %s", out)
	}
	if strings.Contains(out, "pull green: yes") {
		t.Fatalf("must not invent pull green: %s", out)
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
