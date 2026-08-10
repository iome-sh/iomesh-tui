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
		// s1542 setup lifecycle map
		"/onboard next setup",
		"setup lifecycle",
		"setup_not_probed",
		"repair apply ≠ invent Connected",
		"E10 Open",
		// s1546 still-human APPLY reaffirm after setup closeout
		"setup closeout residual ≠ invent APPLY",
		"s1546",
		// s1550 edge-first human-gates residual pin
		"s1550",
		"edge-first",
		"knowledge multi-tenant punted",
		"Slack HMAC punted",
		"portal HITL when connect",
		// s1558 Wave B first-run journey
		"/onboard next journey",
		"edge-user-journey",
		"s1558",
		"Edge Memory GA candidacy only",
		"free eng s1558",
		// s1570 Wave C first-run wizard residual
		"/onboard next wizard",
		"first-run wizard residual",
		"s1570",
		"free eng s1570",
		"free-floor peer s1572+",
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
		// s1546 human-gates companion after setup
		"/onboard next human-gates",
		"setup closeout residual ≠ invent APPLY",
		"still-human APPLY open",
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
		// s1542 setup lifecycle map
		"/onboard next setup",
		"setup_not_probed",
		"package wire ≠ Connected",
		"repair apply ≠ invent Connected",
		"E10 Open",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("status missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent dual_write ON / Connected: %s", out)
	}
}

// s1582: OSS packaging residual honesty one-liner needles.
func TestOSSPackagingHonestyOneLiner_Needles(t *testing.T) {
	out := OSSPackagingHonestyOneLiner
	if out == "" {
		t.Fatal("empty OSS packaging one-liner")
	}
	for _, want := range []string{
		"MIT OSS harness",
		"not control plane",
		"dual_write OFF",
		"not Memory GA",
		"Edge Memory GA candidacy only",
		"book-demo OFF",
		"residual PASS ≠ invent control plane in MIT repo",
		"residual-check",
		"session soft ≠ live dogfood",
		"≠ invent platform green",
		"free eng s1582",
		"free-floor peer s1584+",
		"oss-packaging-boundary.md",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("OSS packaging one-liner missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "book-demo ON") {
		t.Fatalf("must not invent dual_write/book-demo ON: %s", out)
	}
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent Memory GA / Connected: %s", out)
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
		// s1582 OSS packaging residual groups
		"Edge OSS path",
		"Platform residual honesty",
		"optional · anti-claims · offline residual checks",
		"residual-check",
		"not control plane",
		"OSS harness",
		"residual PASS ≠ invent control plane in MIT repo",
		"free eng s1582",
		"free-floor peer s1584+",
		"oss-packaging-boundary.md",
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
		"iomesh-memory-mcp", // product host (s1517: residual aion sample removed)
		"iomesh-memory-mcp",
		"github.com/iome-sh/iomesh-memory-mcp",
		"public product attach",
		"s1478",
		"go install",
		"no GOPRIVATE",
		"docker compose still valid",
		"aion broker private",
		"flip complete residual ≠ invent Memory GA",
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
		"agentic-integrations|integrations|list-plan",
		"/onboard next portal-hitl",
		"hitl|portal_hitl|portal-dogfood|stage5|connectors-hitl",
		"free eng s1562",
		"list_plan_not_connected",
		// portal HITL still
		"7.",
		"portal HITL",
		"agent MCP cannot write installs",
		"catalog ≠ Connected",
		// s1566 E4 client attach journey stage 6
		"8.",
		"E4 client attach",
		"/onboard next e4",
		"e4-dogfood|client-attach|edge-memory-e4|e4_attach",
		"free eng s1566",
		"tools=6",
		"iomesh mcp --connect",
		// s1413 human-gates
		"9.",
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
		// s1437 sales/buyer claims cross-link
		"/onboard next sales",
		"sales/buyer claims",
		"claims|buyer|claim-matrix|sales-claims|buyer-claims",
		"may claim / must not claim",
		"three-planes grounded",
		// s1442 demo readiness cross-link
		"/onboard next demo",
		"demo readiness",
		"demo-ready|readiness|demo-readiness|lighthouse|landgrab",
		"Landgrab NOT READY",
		"book-demo OFF",
		"residual PASS ≠ logos met",
		// s1447 operator readiness matrix cross-link
		"/onboard next operator",
		"operator readiness matrix",
		"operator-matrix|ops-matrix|operator-readiness|ops-readiness|matrix",
		"still_human",
		"policy_off",
		"not_ready",
		// s1542 setup lifecycle P1–P7 closeout residual
		"/onboard next setup",
		"setup lifecycle map",
		"setup-lifecycle|lifecycle|setup_lifecycle",
		"setup_not_probed",
		"repair apply ≠ invent Connected",
		"dual_write never auto ON",
		"E10 Open",
		"package wire ≠ Connected",
		// s1558 Wave B edge-user-journey first-run
		"/onboard next journey",
		"edge-user-journey first-run map",
		"edge-journey|user-journey|first-run|edge_user_journey",
		"free eng s1558",
		"Edge Memory GA candidacy only",
		"residual PASS ≠ invent Edge Memory GA",
		"no invent TUI portal SSO",
		"host not auto",
		"free-floor peer s1560+",
		// s1570 Wave C first-run wizard residual
		"/onboard next wizard",
		"first-run wizard residual",
		"first-run-wizard|guided|wave-c|wave_c|wizard-residual",
		"free eng s1570",
		"free-floor peer s1572+",
		// s1586 E10 Open reaffirm residual-check (Platform residual honesty)
		"/onboard next e10",
		"E10 Open reaffirm",
		"e10-open|edge-memory-e10|ga-signoff|e10_open",
		"residual PASS ≠ invent E10 closed",
		"free eng s1586",
		"free-floor peer s1588+",
		// s1590 marketing demo path (Edge OSS / demo-oriented)
		"/onboard next marketing-demo",
		"marketing demo path",
		"marketing|sales-demo|demo-script|gtm-demo",
		"local agent + local memory",
		"mesh optional",
		"free eng s1590",
		"free-floor peer s1592+",
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
		"iomesh plugins smoke",
		"/plugins smoke",
		"iomesh plugins dogfood", // legacy alias residual
		"offline sample validate",
		"examples/agent-plugins",
		"hello-iome",
		"iomesh-memory-mcp",
		"iomesh plugins list",
		"iomesh plugins validate",
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

// s1377+s1453+s1458+s1463+s1469+s1478+s1508+s1517: AionAgentOnboardingNextMemoryLane residual-honest memory local + edge OSS + public product attach + E4 client attach + product-only sample needles.
func TestAionAgentOnboardingNextMemoryLane_HonestyNeedles(t *testing.T) {
	out := AionAgentOnboardingNextMemoryLane()
	if out == "" {
		t.Fatal("empty memory lane")
	}
	for _, want := range []string{
		"onboard next memory lane",
		"no MCP dial",
		"s1377+s1453+s1458+s1463+s1469+s1478+s1508+s1517",
		"local-primary",
		"github.com/iome-sh/memory",
		"github.com/iome-sh/iomesh-memory-mcp",
		"iomesh-memory-mcp",
		"s1517", // product-only memory sample (iomesh-memory-mcp); aion residual sample removed
		"product-only memory sample",
		"aion broker private",
		"aion still private",
		"public",
		"s1478",
		"s1508",
		"E4 MCP client attach",
		"Edge Memory GA candidacy only",
		"residual PASS ≠ invent Edge Memory GA declared",
		"E10 Open",
		"tip ≠ invent forever-green product dogfood",
		"no GOPRIVATE",
		"go install",
		"go get github.com/iome-sh/memory@main",
		"github.com/iome-sh/iomesh-memory-mcp/cmd/iomesh-memory-mcp@main",
		"Edge OSS",
		"Option A",
		"streamable HTTP",
		"http://127.0.0.1:8080/mcp",
		"stdio",
		"docker compose",
		"iomesh-memory-mcp:local",
		"healthz",
		"edge-dogfood-gate",
		"offline dogfood tip ≠ invent live dogfood as green",
		"flip complete residual ≠ invent Memory GA",
		"public OSS ≠ invent platform GA",
		"PASS ≠ invent full platform sidecar parity",
		"tool parity may be lean",
		"Palace sunset",
		"mesh optional for pull",
		"/onboard next memory-pull",
		"/onboard next operator",
		"dual_write OFF",
		"package load ≠ Memory GA",
		"freemium palace",
		"not Memory GA",
		"memory-advanced-agent",
		"/memory status",
		"/onboard status",
		"fail-open offline",
		"residual PASS ≠ live dogfood",
		"PASS ≠ live APPLY",
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
		"product-only memory sample",
		"iomesh-memory-mcp only",
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
	if strings.Contains(out, "full platform sidecar parity: yes") || strings.Contains(out, "platform sidecar parity complete") {
		t.Fatalf("must not invent full platform sidecar parity: %s", out)
	}
	if strings.Contains(out, "live dogfood green: yes") || strings.Contains(out, "live dogfood: green") {
		t.Fatalf("must not invent live dogfood green: %s", out)
	}
	// Edge packs are public; do not keep the obsolete "repos still private" claim.
	if strings.Contains(out, "repos still private") {
		t.Fatalf("must not claim edge repos still private after s1478 public flip: %s", out)
	}
	// Positive claims only — honesty needles contain "≠ invent Edge Memory GA declared" / "E10 Open".
	if strings.Contains(out, "Edge Memory GA declared: yes") || strings.Contains(out, "Edge Memory GA: shipped") || strings.Contains(out, "Edge Memory GA shipped") {
		t.Fatalf("must not invent Edge Memory GA declared: %s", out)
	}
	if strings.Contains(out, "E10 closed") || strings.Contains(out, "E10: closed") {
		t.Fatalf("must not invent E10 closed: %s", out)
	}
	if strings.Contains(out, "forever-green product dogfood: yes") || strings.Contains(out, "forever-green: yes") {
		t.Fatalf("must not invent forever-green product dogfood: %s", out)
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

// s1542+s1558: AionAgentOnboardingNextSetupLane residual-honest setup lifecycle P1–P7 closeout map needles.
func TestAionAgentOnboardingNextSetupLane_HonestyNeedles(t *testing.T) {
	out := AionAgentOnboardingNextSetupLane()
	if out == "" {
		t.Fatal("empty setup lane")
	}
	for _, want := range []string{
		"onboard next setup lane",
		"no MCP dial",
		"setup lifecycle P1–P7 closeout residual",
		"stage 4 of edge-user-journey",
		"s1558 Wave B",
		"/setup init",
		"iomesh setup init",
		"dual_write OFF",
		"managed fragment",
		"start memory host",
		"/setup preflight",
		"PASS ≠ invent Connected",
		"/setup reload",
		"package wire ≠ Connected",
		"/setup portal",
		"portal HITL",
		"/setup pull start",
		"pull ≠ invent Connected",
		"/setup analyze start",
		"tick ≠ invent green",
		"/setup drift",
		"report-only",
		"/setup repair plan",
		"/setup repair apply --yes",
		"safe steps only",
		"dual_write never auto ON",
		"repair apply ≠ invent Connected",
		"/memory digest",
		"not Memory GA",
		"catalog ≠ Connected",
		"still-human APPLY open",
		"E10 Open",
		"setup_not_probed",
		"offline static lane ≠ live dogfood",
		"setup closeout residual ≠ invent Edge Memory GA",
		"Edge Memory GA candidacy only",
		"free eng s1558",
		"/onboard next setup",
		"setup-lifecycle|lifecycle|setup_lifecycle",
		"/onboard next journey",
		"/onboard next memory",
		"/onboard next memory-pull",
		"/onboard next human-gates",
		"/onboard next operator",
		"docs/architecture/setup-lifecycle.md",
		"docs/architecture/edge-user-journey.md",
		"docs/architecture/memory-edge-usage-demo.md",
		"agent MCP cannot write installs",
		"never invent install green",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("setup lane missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent dual_write ON / Connected: %s", out)
	}
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "Edge Memory GA declared") {
		t.Fatalf("must not invent Memory GA / Edge Memory GA declared: %s", out)
	}
	if strings.Contains(out, "E10 closed") || strings.Contains(out, "APPLY green") {
		t.Fatalf("must not invent E10 closed / APPLY green: %s", out)
	}
}

// s1570 Wave C: AionAgentOnboardingNextWizardLane residual-honest guided first-run wizard residual needles.
func TestAionAgentOnboardingNextWizardLane_HonestyNeedles(t *testing.T) {
	ResetWizardSoftDogfoodSessionState()
	t.Cleanup(ResetWizardSoftDogfoodSessionState)

	out := AionAgentOnboardingNextWizardLane()
	if out == "" {
		t.Fatal("empty wizard lane")
	}
	for _, want := range []string{
		"onboard next wizard lane",
		"no MCP dial",
		"s1570 Wave C",
		"first-run wizard residual",
		"Wave C",
		"free eng s1570",
		"free-floor peer s1572+",
		// 7 stages
		"1. Signup",
		"2. Download TUI",
		"3. TUI auth/keys",
		"4. Setup",
		"5. Connectors",
		"6. Local store",
		"7. Analyze",
		"console.iome.sh",
		"optional pure local",
		"go install",
		"Ollama",
		"no invent TUI portal SSO",
		"/onboard next setup",
		"/setup init",
		"/setup preflight",
		"/onboard next portal-hitl",
		"/integrations list|plan|status",
		"portal HITL when connect",
		"/onboard next e4",
		"iomesh-memory-mcp",
		"host not auto",
		"/setup analyze",
		"/memory digest",
		// honesty locks
		"dual_write OFF",
		"not Memory GA",
		"Edge Memory GA candidacy only",
		"residual PASS ≠ invent Edge Memory GA declared",
		"E10 Open",
		"agent MCP cannot write installs",
		"catalog ≠ Connected",
		"book-demo OFF",
		"residual PASS ≠ invent full interactive auto wizard",
		"wizard_soft_not_run",
		"/onboard next wizard dogfood",
		"soft|samples|offline|wizard-soft",
		// slash + companions
		"/onboard next wizard",
		"first-run-wizard|guided|wave-c|wave_c|wizard-residual",
		"/onboard next journey",
		"/onboard next setup",
		"/onboard next portal-hitl",
		"/onboard next e4",
		"/onboard next human-gates",
		"docs/architecture/edge-user-journey.md",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("wizard lane missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent dual_write ON / Connected: %s", out)
	}
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "Edge Memory GA is declared") {
		t.Fatalf("must not invent Memory GA / Edge Memory GA is declared: %s", out)
	}
	if strings.Contains(out, "TUI portal SSO shipped") || strings.Contains(out, "auto memory host") {
		t.Fatalf("must not invent SSO / auto host: %s", out)
	}
	if strings.Contains(out, "full interactive auto wizard shipped") {
		t.Fatalf("must not invent full interactive wizard: %s", out)
	}
}

// s1558 Wave B: AionAgentOnboardingNextJourneyLane residual-honest 7-stage first-run map needles.
func TestAionAgentOnboardingNextJourneyLane_HonestyNeedles(t *testing.T) {
	out := AionAgentOnboardingNextJourneyLane()
	if out == "" {
		t.Fatal("empty journey lane")
	}
	for _, want := range []string{
		"onboard next journey lane",
		"no MCP dial",
		"s1558 Wave B",
		"edge-user-journey first-run map",
		"free eng s1558",
		"free-floor peer s1560+",
		// 7 stages
		"1. Signup",
		"2. Download TUI",
		"3. TUI auth/keys",
		"4. Setup wizard",
		"5. Connectors",
		"6. Local store",
		"7. Analyze",
		"console.iome.sh",
		"optional pure local",
		"go install",
		"Ollama",
		"no invent TUI portal SSO",
		"/setup",
		"/onboard next setup",
		"iomesh setup",
		"/integrations list|plan|status",
		"/onboard next portal-hitl",
		"/onboard next agentic",
		"portal HITL",
		"/onboard next portal-hitl dogfood",
		"iomesh-memory-mcp",
		"host not auto",
		"/onboard next memory",
		"/onboard next e4",
		"/onboard next e4 dogfood",
		"/memory digest",
		"/setup analyze",
		// honesty locks
		"dual_write OFF",
		"not Memory GA",
		"Edge Memory GA candidacy only",
		"residual PASS ≠ invent Edge Memory GA",
		"agent MCP cannot write installs",
		"catalog ≠ Connected",
		"book-demo OFF",
		// residual gaps
		"no SSO invent",
		"host not auto",
		// docs
		"docs/architecture/edge-user-journey.md",
		"docs/architecture/setup-lifecycle.md",
		"docs/architecture/memory-edge-usage-demo.md",
		"docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md",
		// slash + companions
		"/onboard next journey",
		"edge-journey|user-journey|first-run|edge_user_journey",
		"/onboard next setup",
		"/onboard next portal-hitl",
		"/onboard next e4",
		"/onboard next agentic",
		"/onboard next memory",
		"/onboard next human-gates",
		"/onboard next operator",
		"/onboard next wizard",
		"s1570 Wave C",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("journey lane missing %q in:\n%s", want, out)
		}
	}
	// Must not invent product success language.
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent dual_write ON / Connected: %s", out)
	}
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "Edge Memory GA is declared") {
		t.Fatalf("must not invent Memory GA / Edge Memory GA is declared: %s", out)
	}
	if strings.Contains(out, "TUI portal SSO shipped") || strings.Contains(out, "auto memory host") {
		t.Fatalf("must not invent SSO / auto host: %s", out)
	}
	if strings.Contains(out, "book-demo ON") || strings.Contains(out, "INSTALL_STORE APPLY green") {
		t.Fatalf("must not invent book-demo ON / APPLY green: %s", out)
	}
}

// s1562: AionAgentOnboardingNextPortalHITLLane residual-honest journey stage-5 portal HITL needles.
func TestAionAgentOnboardingNextPortalHITLLane_HonestyNeedles(t *testing.T) {
	ResetPortalHITLSoftDogfoodSessionState()
	t.Cleanup(ResetPortalHITLSoftDogfoodSessionState)

	out := AionAgentOnboardingNextPortalHITLLane()
	if out == "" {
		t.Fatal("empty portal-hitl lane")
	}
	for _, want := range []string{
		"onboard next portal-hitl lane",
		"no MCP dial",
		"journey stage 5",
		"portal HITL when connect",
		"MCP list/plan",
		"browser portal HITL",
		"human finishes OAuth/install",
		"/integrations/{id}",
		"/integrations/add?template={id}",
		"/integrations",
		"agent MCP cannot write installs",
		"catalog ≠ Connected",
		"never invent Connected",
		"template= ≠ install APPLY",
		"portal HITL still",
		"portal_hitl_still",
		"dual_write OFF",
		"book-demo OFF",
		"not Memory GA",
		"Edge Memory GA candidacy only",
		"residual PASS ≠ invent Edge Memory GA",
		"residual PASS ≠ live dogfood",
		"soft offline ≠ invent Connected",
		"session soft ≠ live dogfood",
		"console.iome.sh/integrations",
		"console.iome.sh/settings/agent",
		"portal_hitl_soft_not_run",
		"/onboard next portal-hitl dogfood",
		"soft|samples|offline|portal-hitl-soft",
		"hitl|portal_hitl|portal-dogfood|stage5|connectors-hitl",
		"/onboard next agentic",
		"/onboard next journey",
		"/onboard next agentic dogfood",
		"free eng s1562",
		"free-floor peer s1564+",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("portal-hitl lane missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent dual_write ON / Connected: %s", out)
	}
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "INSTALL_STORE APPLY success") {
		t.Fatalf("must not invent Memory GA / APPLY green: %s", out)
	}
	if strings.Contains(out, "live dogfood green") || strings.Contains(out, "book-demo ON") {
		t.Fatalf("must not invent live dogfood green / book-demo ON: %s", out)
	}
}

// s1566: AionAgentOnboardingNextE4Lane residual-honest journey stage-6 E4 client-attach needles.
func TestAionAgentOnboardingNextE4Lane_HonestyNeedles(t *testing.T) {
	ResetE4SoftDogfoodSessionState()
	t.Cleanup(ResetE4SoftDogfoodSessionState)

	out := AionAgentOnboardingNextE4Lane()
	if out == "" {
		t.Fatal("empty e4 lane")
	}
	for _, want := range []string{
		"onboard next e4 lane",
		"no MCP dial",
		"journey stage 6",
		"E4 client attach",
		"client attach",
		"tools=6",
		"iomesh mcp --connect",
		"iomesh-memory-mcp",
		"local-primary",
		"docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md",
		"dual_write OFF",
		"book-demo OFF",
		"not Memory GA",
		"Edge Memory GA candidacy only",
		"residual PASS ≠ invent Edge Memory GA declared",
		"E10 Open",
		"tip ≠ invent forever-green product dogfood",
		"residual PASS ≠ live dogfood",
		"PASS ≠ live APPLY",
		"soft offline ≠ invent Connected",
		"session soft ≠ live dogfood",
		"e4_soft_not_run",
		"/onboard next e4 dogfood",
		"soft|samples|offline|e4-soft",
		"e4-dogfood|client-attach|edge-memory-e4|e4_attach",
		"/onboard next memory",
		"/onboard next journey",
		"/onboard next tool-call",
		"free eng s1566",
		"free-floor peer s1568+",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("e4 lane missing %q in:\n%s", want, out)
		}
	}
	for _, bad := range []string{
		"dual_write ON",
		"Connected: yes",
		"Edge Memory GA is declared",
		"E10 is closed",
		"forever-green product dogfood green",
		"live dogfood green",
		"INSTALL_STORE APPLY success",
		"book-demo ON",
		"Memory GA shipped",
	} {
		if strings.Contains(out, bad) {
			t.Fatalf("must not invent %q:\n%s", bad, out)
		}
	}
}

// s1578: AionAgentOnboardingNextToolCallLane residual-honest deeper tool-call needles.
func TestAionAgentOnboardingNextToolCallLane_HonestyNeedles(t *testing.T) {
	ResetToolCallSoftDogfoodSessionState()
	t.Cleanup(ResetToolCallSoftDogfoodSessionState)

	out := AionAgentOnboardingNextToolCallLane()
	if out == "" {
		t.Fatal("empty tool-call lane")
	}
	for _, want := range []string{
		"onboard next tool-call lane",
		"no MCP dial",
		"deeper tool-call residual",
		"journey stage 6/7",
		"memory_ingest_turn",
		"memory_retrieve",
		"memory_search_semantic",
		"memory_list",
		"memory_compact_status",
		"memory_facts_as_of",
		"Partial→client-attach-evidence",
		"/onboard next e4",
		"tools=6",
		"iomesh mcp --connect",
		"s1508",
		"s1566",
		"iomesh-memory-mcp",
		"docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md",
		"dual_write OFF",
		"book-demo OFF",
		"not Memory GA",
		"Edge Memory GA candidacy only",
		"residual PASS ≠ invent Edge Memory GA declared",
		"E10 Open",
		"tip ≠ invent forever-green product dogfood",
		"residual PASS ≠ live dogfood",
		"PASS ≠ live APPLY",
		"soft offline ≠ invent Connected",
		"session soft ≠ live dogfood",
		"tool_call_soft_not_run",
		"/onboard next tool-call dogfood",
		"soft|samples|offline|tool-call-soft",
		"tool-calls|deeper-e4|e4-tools|ingest-retrieve|tool_call",
		"/onboard next memory",
		"/onboard next journey",
		"/onboard next e10",
		"free eng s1578",
		"free-floor peer s1580+",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("tool-call lane missing %q in:\n%s", want, out)
		}
	}
	for _, bad := range []string{
		"dual_write ON",
		"Connected: yes",
		"Edge Memory GA is declared",
		"E10 is closed",
		"forever-green product dogfood green",
		"live dogfood green",
		"INSTALL_STORE APPLY success",
		"book-demo ON",
		"Memory GA shipped",
	} {
		if strings.Contains(out, bad) {
			t.Fatalf("must not invent %q:\n%s", bad, out)
		}
	}
}

// s1586: AionAgentOnboardingNextE10Lane residual-honest E10 Open reaffirm residual-check needles.
func TestAionAgentOnboardingNextE10Lane_HonestyNeedles(t *testing.T) {
	ResetE10SoftDogfoodSessionState()
	t.Cleanup(ResetE10SoftDogfoodSessionState)

	out := AionAgentOnboardingNextE10Lane()
	if out == "" {
		t.Fatal("empty e10 lane")
	}
	for _, want := range []string{
		"onboard next e10 lane",
		"no MCP dial",
		"E10 Open",
		"E10 Open reaffirm",
		"residual-check",
		"residual PASS ≠ invent E10 closed",
		"residual PASS ≠ invent Edge Memory GA declared",
		"Edge Memory GA candidacy only",
		"not Memory GA",
		"dual_write OFF",
		"book-demo OFF",
		"founder sign-off only if declaring Edge Memory GA",
		"candidacy allowed without E10",
		"PASS ≠ live APPLY",
		"session soft ≠ live dogfood",
		"residual PASS ≠ live dogfood",
		"soft offline ≠ invent Connected",
		"e10_soft_not_run",
		"/onboard next e10 dogfood",
		"soft|samples|offline|e10-soft|residual-check",
		"e10-open|edge-memory-e10|ga-signoff|e10_open",
		"/onboard next e4",
		"/onboard next human-gates",
		"OSS packaging",
		"MIT harness",
		"not control plane",
		"free eng s1586",
		"free-floor peer s1588+",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("e10 lane missing %q in:\n%s", want, out)
		}
	}
	for _, bad := range []string{
		"dual_write ON",
		"Connected: yes",
		"Edge Memory GA is declared",
		"E10 is closed",
		"live APPLY green",
		"live dogfood green",
		"INSTALL_STORE APPLY success",
		"book-demo ON",
		"Memory GA shipped",
	} {
		if strings.Contains(out, bad) {
			t.Fatalf("must not invent %q:\n%s", bad, out)
		}
	}
}

// s1590: AionAgentOnboardingNextMarketingDemoLane plain-language marketing demo path needles.
func TestAionAgentOnboardingNextMarketingDemoLane_HonestyNeedles(t *testing.T) {
	out := AionAgentOnboardingNextMarketingDemoLane()
	if out == "" {
		t.Fatal("empty marketing-demo lane")
	}
	for _, want := range []string{
		"onboard next marketing-demo path",
		"plain-language operator script",
		"local agent + local memory",
		"videos/sales",
		// script steps
		"Install / build iomesh",
		"LLM key or Ollama",
		"/setup init",
		"local-memory",
		"preflight",
		"iomesh-memory-mcp",
		"/memory",
		"ingest",
		"recall",
		"mesh optional",
		// local memory honesty
		"local memory",
		"local-primary",
		"dual_write OFF",
		"not Memory GA",
		"never invent Connected",
		"book-demo OFF",
		// aliases + non-steal
		"marketing|sales-demo|demo-script|gtm-demo",
		"NOT bare demo",
		"NOT bare sales",
		"NOT bare gtm",
		"/onboard next marketing-demo",
		"free eng s1590",
		"free-floor peer s1592+",
		"docs/architecture/marketing-demo-path.md",
		// s1594 sales talk track (optional spoken bullets)
		"Sales talk track",
		"tool-marketing",
		"claims catalog",
		"demoable vs do-not-claim",
		"Win-back",
		"closed-lost",
		"sales process",
		"auto-CRM",
		"s1594",
		// s1598 GTM claim-support (SEO / publish / CRM honesty)
		"s1598",
		"Search Console",
		"no auto rank claims",
		"Hermes handoff",
		"does not auto-post",
		"closed-loop metrics",
		"not the CRM",
		// s1602 operator GTM boundary (credentials · Hermes outside TUI · CRM human-gated)
		"s1602",
		"operator machine",
		"outside the public TUI",
		"no social tokens",
		"commercial CRM writes stay human",
		// s1606 GTM wave 6 (Hermes network · HubSpot/Twenty operator-box · no tokens in public harness)
		"s1606",
		"Hermes network dispatch",
		"operator webhook",
		"private runner",
		"HubSpot / Twenty",
		"operator-box OAuth",
		"human approve",
		"No social or CRM tokens in the public harness",
		// s1610 GTM wave 7 (Hermes mock dogfood · HubSpot dual control · sales-loop outbox · not fleet GA)
		"s1610",
		"Hermes dogfood",
		"operator mock",
		"not Hermes control plane",
		"HubSpot dual control",
		"write allow flags",
		"Sales-loop mesh outbox",
		"local envelope",
		"mesh GTM fleet GA",
		// s1614 GTM wave 8 (real Hermes daemon path · live dual-control dogfood · operator GTM status)
		"s1614",
		"Real Hermes daemon",
		"TUI does not host it",
		"Live HubSpot / Twenty",
		"dual control + tokens",
		"Operator GTM status",
		"private tooling",
		"not a product dashboard claim",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("marketing-demo lane missing %q in:\n%s", want, out)
		}
	}
	for _, bad := range []string{
		"dual_write ON",
		"Connected: yes",
		"Memory GA shipped",
		"book-demo ON",
		"INSTALL_STORE APPLY success",
		"live dogfood green",
		"mesh GTM fleet GA shipped",
		"freemium palace",
	} {
		if strings.Contains(out, bad) {
			t.Fatalf("must not invent %q:\n%s", bad, out)
		}
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
		"agentic-integrations|integrations|list-plan",
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
		// s1562 companion portal HITL residual
		"/onboard next portal-hitl",
		"/onboard next portal-hitl dogfood",
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

// s1437: AionAgentOnboardingNextSalesClaims residual-honest sales/buyer claims board needles.
func TestAionAgentOnboardingNextSalesClaims_HonestyNeedles(t *testing.T) {
	out := AionAgentOnboardingNextSalesClaims()
	if out == "" {
		t.Fatal("empty sales claims board")
	}
	for _, want := range []string{
		"onboard next sales / buyer claims",
		"no MCP dial",
		"s1437",
		"three-planes grounded",
		// may claim
		"MAY CLAIM",
		"streaming org heartbeats",
		"not OTel/APM",
		"mesh ≠ memory",
		"~$88",
		"~$119",
		"dual_write OFF",
		"local-primary",
		"Palace sunset",
		"Salesforce = GA CRM",
		"HubSpot + GTM suite Beta multi-tenant",
		"guerrilla global-only",
		"knowledge / analytical = Beta",
		"no invent GA knowledge/analytical",
		"MCP list/plan residual-honest",
		"catalog ≠ Connected",
		"list_plan_not_connected",
		"/onboard next planes",
		// must not claim
		"MUST NOT CLAIM",
		"never invent Connected",
		"INSTALL_STORE APPLY",
		"not Memory GA",
		"Ops Pack ≠ GPU fleet",
		"book-demo OFF",
		"dual_auth_candidacy_open",
		"tool ship ≠ dual-auth live",
		"agent MCP cannot write installs",
		"PASS ≠ invent human-gate green",
		"Slack HMAC",
		"Stripe Customers:Write",
		"H1/H2 INSTALL_STORE",
		"open boxes stay open",
		"residual PASS ≠ live dogfood",
		"PASS ≠ live APPLY",
		// three-planes companion
		"plane 1 mesh",
		"plane 2 memory-pull",
		"plane 3 agentic",
		"streams_not_probed",
		"pull_not_probed",
		// slash + aliases
		"/onboard next sales",
		"claims|buyer|claim-matrix|sales-claims|buyer-claims",
		// do not steal
		"NOT product|planes",
		"NOT gtm|drafts",
		"NOT pulse|board",
		// cross-links
		"/onboard next mesh",
		"/onboard next memory-pull",
		"/onboard next agentic",
		"/onboard next human-gates",
		"/onboard next status",
		"drafts only",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("sales claims board missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent dual_write ON / Connected: %s", out)
	}
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "book-demo ON") {
		t.Fatalf("must not invent Memory GA shipped / book-demo ON: %s", out)
	}
	if strings.Contains(out, "dual-auth live: yes") || strings.Contains(out, "INSTALL_STORE APPLY success") {
		t.Fatalf("must not invent dual-auth live / INSTALL_STORE APPLY success: %s", out)
	}
	if strings.Contains(out, "Knowledge GA shipped") || strings.Contains(out, "Analytics GA shipped") {
		t.Fatalf("must not invent Knowledge/Analytics GA: %s", out)
	}
	// Must-not section should still surface the forbidden-claim list (rephrased).
	for _, want := range []string{
		"invent dual_write as ON",
		"invent book-demo as ON",
		"invent dual-auth as live",
		"Ops Pack as GPU fleet",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("sales claims must-not section missing %q in:\n%s", want, out)
		}
	}
}

// s1442: AionAgentOnboardingNextDemoReadiness residual-honest demo readiness board needles.
func TestAionAgentOnboardingNextDemoReadiness_HonestyNeedles(t *testing.T) {
	out := AionAgentOnboardingNextDemoReadiness()
	if out == "" {
		t.Fatal("empty demo readiness board")
	}
	for _, want := range []string{
		"onboard next demo readiness",
		"no MCP dial",
		"s1442",
		// packaging
		"Lighthouse beachhead",
		"B2B SaaS",
		"book-demo OFF",
		"See pricing",
		"leave ON_SIGNAL unset",
		// Landgrab
		"Landgrab NOT READY",
		"empty-honest",
		"residual PASS ≠ logos met",
		"do not invent book-demo as ON",
		// three planes
		"/onboard next planes",
		"mesh · memory-pull · agentic",
		"streams_not_probed",
		"pull_not_probed",
		"list_plan_not_connected",
		"dual_auth_candidacy_open",
		// sales claims
		"/onboard next sales",
		"may claim / must not claim",
		// human gates
		"Slack HMAC",
		"Stripe Customers:Write",
		"H1/H2 INSTALL_STORE",
		"K-D*",
		"/onboard next human-gates",
		"PASS ≠ invent human-gate green",
		"open boxes stay open",
		// honesty locks
		"dual_write OFF",
		"not Memory GA",
		"never invent Connected",
		"residual PASS ≠ live dogfood",
		"PASS ≠ live APPLY",
		"rates ~$88/$119 optional",
		// demo path residual
		"founder-led walkthrough only when scheduled",
		"operator runbook ≠ public /demo booking live",
		// slash + aliases
		"/onboard next demo",
		"demo-ready|readiness|demo-readiness|lighthouse|landgrab",
		// do not steal
		"NOT sales|claims",
		"NOT product|planes",
		"NOT pulse|board",
		"NOT gtm|drafts",
		// cross-links
		"/onboard next mesh",
		"/onboard next memory-pull",
		"/onboard next agentic",
		"/onboard next status",
		"/onboard next export",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("demo readiness board missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent dual_write ON / Connected: %s", out)
	}
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "book-demo ON") {
		t.Fatalf("must not invent Memory GA shipped / book-demo ON: %s", out)
	}
	if strings.Contains(out, "Landgrab READY: yes") || strings.Contains(out, "Landgrab: READY") {
		t.Fatalf("must not invent Landgrab READY: %s", out)
	}
	if strings.Contains(out, "dual-auth live: yes") || strings.Contains(out, "INSTALL_STORE APPLY success") {
		t.Fatalf("must not invent dual-auth live / INSTALL_STORE APPLY success: %s", out)
	}
}

// s1427: AionAgentOnboardingNextAgenticDualAuthCandidacy residual-honest dual-auth candidacy needles.
// s1447: AionAgentOnboardingNextOperatorMatrix residual-honest operator readiness matrix needles.
func TestAionAgentOnboardingNextOperatorMatrix_HonestyNeedles(t *testing.T) {
	out := AionAgentOnboardingNextOperatorMatrix()
	if out == "" {
		t.Fatal("empty operator readiness matrix")
	}
	for _, want := range []string{
		"onboard next operator readiness matrix",
		"no MCP dial",
		"s1447",
		// honest vocab
		"residual_only",
		"path_ready",
		"still_human",
		"policy_off",
		"not_ready",
		"portal_hitl_still",
		// row 1 demo
		"Demo readiness",
		"Lighthouse beachhead",
		"book-demo OFF",
		"Landgrab NOT READY",
		"residual PASS ≠ logos met",
		"/onboard next demo",
		// row 2 sales
		"Sales claims",
		"may claim / must not claim",
		"/onboard next sales",
		// row 3 planes
		"Three product planes",
		"mesh · memory-pull · agentic",
		"streams_not_probed",
		"pull_not_probed",
		"list_plan_not_connected",
		"/onboard next planes",
		// row 4 human gates (s1550 edge-first)
		"Human gates",
		"edge-first",
		"knowledge multi-tenant punted",
		"Slack HMAC punted",
		"portal HITL when connect",
		"/onboard next human-gates",
		"dual_write OFF",
		"not Memory GA",
		"Edge Memory GA candidacy only",
		"residual PASS ≠ invent Edge Memory GA",
		"PASS ≠ invent Connected",
		"H1/H2 not launch gate",
		// s1546 setup closeout residual ≠ invent APPLY
		"setup closeout residual ≠ invent APPLY",
		"s1546",
		"E10",
		// row 5 dual-auth
		"dual_auth_candidacy_open",
		"list_org_unavailable",
		"tool ship ≠ dual-auth live",
		"/onboard next agentic dual-auth",
		// row 6 policy
		"dual_write OFF",
		"not Memory GA",
		"leave ON_SIGNAL unset",
		"rates ~$88",
		"~$119",
		// row 7 export
		"/onboard next export",
		"board/export evidence ≠ invent Connected",
		// honesty locks
		"never invent Connected",
		"residual PASS ≠ live dogfood",
		"PASS ≠ live APPLY",
		// slash + aliases
		"/onboard next operator",
		"operator-matrix|ops-matrix|operator-readiness|ops-readiness|matrix",
		// do not steal
		"NOT demo|readiness|lighthouse|landgrab",
		"NOT sales|claims",
		"NOT product|planes",
		"NOT pulse|board",
		"NOT export|receipt",
		// companions
		"/onboard next agentic",
		"/onboard next status",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("operator readiness matrix missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent dual_write ON / Connected: %s", out)
	}
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "book-demo ON") {
		t.Fatalf("must not invent Memory GA shipped / book-demo ON: %s", out)
	}
	if strings.Contains(out, "Landgrab READY: yes") || strings.Contains(out, "Landgrab: READY") {
		t.Fatalf("must not invent Landgrab READY: %s", out)
	}
	if strings.Contains(out, "dual-auth live: yes") || strings.Contains(out, "INSTALL_STORE APPLY success") {
		t.Fatalf("must not invent dual-auth live / INSTALL_STORE APPLY success: %s", out)
	}
}

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

// s1413+s1546+s1550+s1574: AionAgentHumanGatesHonestyBoard residual-honest edge-first needles.
// s1550: edge-first pin — knowledge multi-tenant punted · Slack HMAC punted · portal HITL when connect.
// s1574: Wave C continuum still-human APPLY soft residual reaffirm · open boxes stay open.
func TestAionAgentHumanGatesHonestyBoard_HonestyNeedles(t *testing.T) {
	ResetStillHumanSoftDogfoodSessionState()
	t.Cleanup(ResetStillHumanSoftDogfoodSessionState)

	out := AionAgentHumanGatesHonestyBoard()
	if out == "" {
		t.Fatal("empty human-gates honesty board")
	}
	for _, want := range []string{
		"human-gates honesty board",
		"no MCP dial",
		"s1550",
		"s1574",
		"edge-first",
		"Wave C continuum",
		"still-human APPLY",
		// sections
		"architecture",
		"still_human_or_policy",
		"punted_or_demoted",
		"offline_residual_only",
		"shipped_or_policy",
		"open inventory residual-honest",
		// architecture locked
		"dual_write OFF",
		"knowledge multi-tenant",
		"H1/H2 not launch gate",
		"Slack HMAC punted",
		"portal HITL",
		"agent cannot write installs",
		// still human or policy
		"catalog ≠ Connected",
		"book-demo OFF",
		"leave ON_SIGNAL unset",
		"Edge Memory GA",
		"E10 Open",
		"PASS ≠ invent human-gate green",
		"PASS ≠ live APPLY",
		"open boxes stay open",
		// punted
		"knowledge multi-tenant punted",
		"H1/H2",
		"INSTALL_STORE",
		"Stripe",
		// offline / shipped
		"agent MCP list/plan",
		"not Memory GA",
		"Edge Memory GA candidacy only",
		"residual PASS ≠ invent Edge Memory GA",
		"residual PASS ≠ invent Edge Memory GA declared",
		"PASS ≠ invent Connected",
		// soft dogfood
		"still_human_soft_not_run",
		"/onboard next human-gates dogfood",
		"soft|samples|offline|still-human-soft|apply-soft",
		"free eng s1574",
		"free-floor peer s1576+",
		// operator
		"/onboard next human-gates",
		"/onboard next setup",
		"/onboard next wizard",
		"/integrations list|plan|status",
		"never invent Connected",
		"human|gates|apply-gates|still-human|apply-residual",
		// locks
		"agent MCP cannot write installs",
		"portal HITL when connect",
		"Slack HMAC punted",
		"open policy boxes stay honest",
		"session soft ≠ live dogfood",
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
	if strings.Contains(out, "Edge Memory GA declared: yes") {
		t.Fatalf("must not invent Edge Memory GA declared: %s", out)
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
		"setup:",
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
		"setup_not_probed",
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
		// memory honesty (+ s1453+s1458+s1463+s1469+s1478+s1508 edge OSS + public product attach + E4)
		"dual_write OFF",
		"package load ≠ Memory GA",
		"local-primary",
		"freemium palace",
		"not Memory GA",
		"mesh ≠ memory",
		"Palace sunset",
		"mesh optional for pull",
		"iomesh-memory-mcp",
		"public product attach",
		"go install",
		"no GOPRIVATE",
		"docker compose still valid",
		"offline dogfood tip ≠ invent live dogfood as green",
		"flip complete residual ≠ invent Memory GA",
		"public OSS ≠ invent platform GA",
		"PASS ≠ invent full platform sidecar parity",
		"aion broker private",
		"s1508",
		"Edge Memory GA candidacy only",
		"residual PASS ≠ invent Edge Memory GA declared",
		"E10 Open",
		"tip ≠ invent forever-green product dogfood",
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
		// setup honesty (s1542)
		"setup lifecycle P1–P7",
		"/onboard next setup",
		"setup-lifecycle|lifecycle|setup_lifecycle",
		"package wire ≠ Connected",
		"repair apply ≠ invent Connected",
		"dual_write never auto ON",
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
		"setup:",
		"agentic:",
		"portal:",
		"dogfood_not_run",
		"list_plan_soft_not_run",
		"path_ready",
		"skill_ready",
		"residual_only",
		"streams_not_probed",
		"pull_not_probed",
		"setup_not_probed",
		"portal_hitl_still",
		"list_plan_not_connected",
		// memory edge OSS tip (s1453+s1458+s1463+s1469+s1478+s1508 public product attach + E4)
		"iomesh-memory-mcp",
		"public product attach",
		"go install",
		"no GOPRIVATE",
		"docker compose still valid",
		"offline dogfood tip ≠ invent live dogfood as green",
		"flip complete residual ≠ invent Memory GA",
		"public OSS ≠ invent platform GA",
		"PASS ≠ invent full platform sidecar parity",
		"aion broker private",
		"s1508",
		"Edge Memory GA candidacy only",
		"residual PASS ≠ invent Edge Memory GA declared",
		"E10 Open",
		"tip ≠ invent forever-green product dogfood",
		"Palace sunset",
		"mesh optional for pull",
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
		"setup_not_probed",
		"portal_hitl_still",
		"list_plan_not_connected",
		`"mesh":`,
		`"memory-pull":`,
		`"ops_pack":`,
		`"setup":`,
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
		"package wire ≠ Connected",
		"repair apply ≠ invent Connected",
		"plan deep links = browser HITL only",
		"template= ≠ install APPLY",
		"/onboard next export",
		"/onboard next mesh",
		"/onboard next agentic dogfood",
		"/onboard next memory-pull",
		"/onboard next setup",
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
