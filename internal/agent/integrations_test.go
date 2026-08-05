package agent

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/mcp"
)

// testdataPath resolves internal/agent/testdata relative to this source file.
func testdataPath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "testdata", name)
}

func readTestdata(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(testdataPath(t, name))
	if err != nil {
		t.Fatalf("read testdata %s: %v", name, err)
	}
	return string(b)
}

func TestIntegrationsCatalog_OfflineFailOpen(t *testing.T) {
	rt := &Runtime{} // no MCP
	out, err := rt.IntegrationsCatalog(context.Background(), "")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "fail-open") {
		t.Fatalf("want fail-open: %s", out)
	}
	if !strings.Contains(out, "console.iome.sh/integrations") {
		t.Fatalf("want portal url: %s", out)
	}
	if !strings.Contains(out, "s1237") {
		t.Fatalf("want s1237 mention: %s", out)
	}
	if strings.Contains(strings.ToLower(out), "connected") && strings.Contains(out, "github") {
		t.Fatalf("must not invent catalog rows offline: %s", out)
	}
}

func TestIntegrationsCatalog_InvalidLayer(t *testing.T) {
	rt := &Runtime{}
	_, err := rt.IntegrationsCatalog(context.Background(), "gtm")
	if err == nil {
		t.Fatal("expected invalid layer error")
	}
}

func TestIntegrationsPlan_RequiresID(t *testing.T) {
	rt := &Runtime{}
	_, err := rt.IntegrationsPlan(context.Background(), "  ")
	if err == nil {
		t.Fatal("expected connector_id required")
	}
}

func TestIntegrationsPlan_OfflineFailOpen(t *testing.T) {
	rt := &Runtime{}
	out, err := rt.IntegrationsPlan(context.Background(), "github")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "fail-open") || !strings.Contains(out, "s1237") {
		t.Fatalf("%s", out)
	}
}

func TestIntegrationsSigning_OfflineFailOpen(t *testing.T) {
	rt := &Runtime{}
	out, err := rt.IntegrationsSigning(context.Background(), "operational")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "fail-open") {
		t.Fatalf("want fail-open: %s", out)
	}
	if !strings.Contains(out, "console.iome.sh/integrations") {
		t.Fatalf("want portal: %s", out)
	}
	// must not invent signing secrets offline
	if strings.Contains(out, "PRIMARY_HEADER") && strings.Contains(out, "X-Hub-Signature") {
		t.Fatalf("must not invent signing rows offline: %s", out)
	}
}

// s1242: aion v178 wire shape uses entries + oauth_install_supported (not connectors/oauth).
func TestFormatConnectorCatalog_V178Entries(t *testing.T) {
	raw := `{
		"count": 3,
		"entries": [
			{"id":"github","label":"GitHub","status":"available","mesh_layer":"operational",
			 "ingress_type":"webhook","oauth_install_supported":false,"portal_path":"/integrations/github"},
			{"id":"notion","label":"Notion","status":"beta","mesh_layer":"knowledge",
			 "ingress_type":"oauth","oauth_install_supported":true,"portal_path":"/integrations/notion"},
			{"id":"embeddings","label":"Embeddings","status":"beta","mesh_layer":"analytical",
			 "ingress_type":"api","oauth_install_supported":false,"portal_path":"/integrations/embeddings"}
		]
	}`
	out := formatConnectorCatalog(raw, "")
	if !strings.Contains(out, "github") || !strings.Contains(out, "notion") {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(out, "ID") || !strings.Contains(out, "STATUS") {
		t.Fatalf("missing header: %s", out)
	}
	// oauth_install_supported bool → yes/no
	if !strings.Contains(out, "notion") {
		t.Fatalf("notion missing: %s", out)
	}
	// notion row should show oauth yes
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "notion") {
			if !strings.Contains(line, "yes") {
				t.Fatalf("notion oauth want yes: %s", line)
			}
		}
		if strings.Contains(line, "github") {
			if !strings.Contains(line, "no") {
				t.Fatalf("github oauth want no: %s", line)
			}
		}
	}
	if !strings.Contains(out, "honesty:") {
		t.Fatalf("missing honesty: %s", out)
	}
	// layer filter
	know := formatConnectorCatalog(raw, "knowledge")
	if !strings.Contains(know, "notion") {
		t.Fatalf("filter knowledge: %s", know)
	}
	if strings.Contains(know, "github") {
		t.Fatalf("filter should drop github: %s", know)
	}
}

// Legacy connectors/oauth shape still parses (fail-open compat).
func TestFormatConnectorCatalog_LegacyConnectors(t *testing.T) {
	raw := `{
		"connectors": [
			{"id":"github","status":"available","mesh_layer":"operational","oauth":true},
			{"id":"notion","status":"beta","mesh_layer":"knowledge","ingress_type":"oauth"}
		]
	}`
	out := formatConnectorCatalog(raw, "")
	if !strings.Contains(out, "github") || !strings.Contains(out, "notion") {
		t.Fatalf("%s", out)
	}
}

// s1242: plan fixture with aion v178 honesty object + portal_url + oauth_mode_hint + signing_headers_tool.
func TestFormatConnectorPlan_V178HonestyObject(t *testing.T) {
	raw := `{
		"connector_id": "github",
		"org_id": "",
		"connector": {
			"id":"github","label":"GitHub","status":"available","mesh_layer":"operational",
			"ingress_type":"webhook","oauth_install_supported":false,"portal_path":"/integrations/github"
		},
		"portal_url": "https://console.iome.sh/integrations/github",
		"oauth_install_supported": false,
		"oauth_mode_hint": "",
		"signing_headers_tool": "get_webhook_signing_headers",
		"next_steps": ["Open portal", "Complete OAuth in browser"],
		"honesty": {
			"browser_hitl_required_for_oauth_complete": true,
			"stub_oauth_not_live": true,
			"pass_not_invent_install_green": true,
			"dual_write_off": true,
			"book_demo_off": true,
			"no_invent_ga": true,
			"agent_mcp_cannot_write_installs": true,
			"session_portal_owns_install_crud": true,
			"notes": [
				"Browser HITL required for OAuth complete",
				"Stub OAuth ≠ live provider token exchange",
				"PASS ≠ invent install green · dual_write OFF · book-demo OFF"
			]
		}
	}`
	out := formatConnectorPlan(raw, "github")
	if !strings.Contains(out, "portal_url:") || !strings.Contains(out, "console.iome.sh/integrations/github") {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(out, "Open portal") {
		t.Fatalf("next_steps: %s", out)
	}
	if !strings.Contains(out, "Browser HITL") {
		t.Fatalf("honesty notes: %s", out)
	}
	if !strings.Contains(out, "signing_headers_tool:") || !strings.Contains(out, "get_webhook_signing_headers") {
		t.Fatalf("signing_headers_tool: %s", out)
	}
	if !strings.Contains(out, "never invent install green") && !strings.Contains(out, IntegrationsHonestyOneLiner) {
		t.Fatalf("residual honesty: %s", out)
	}
	// status from nested connector (display-only)
	if !strings.Contains(out, "available") {
		t.Fatalf("status from connector: %s", out)
	}
	// must not invent Connected green
	if strings.Contains(out, "Connected") {
		t.Fatalf("must not invent Connected: %s", out)
	}
}

func TestFormatConnectorPlan_OAuthModeHint(t *testing.T) {
	raw := `{
		"connector_id": "notion",
		"portal_url": "https://console.iome.sh/integrations/notion",
		"oauth_install_supported": true,
		"oauth_mode_hint": "stub",
		"next_steps": ["Open portal"],
		"honesty": {"notes": ["stub ≠ live"]}
	}`
	out := formatConnectorPlan(raw, "notion")
	if !strings.Contains(out, "oauth_mode_hint: stub") {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(out, "oauth_install_supported: true") {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(out, "stub ≠ live") {
		t.Fatalf("%s", out)
	}
}

func TestFormatConnectorPlan_DefaultPortal(t *testing.T) {
	raw := `{"connector_id":"slack"}`
	out := formatConnectorPlan(raw, "slack")
	if !strings.Contains(out, integrationsPortalURL+"/slack") {
		t.Fatalf("default portal: %s", out)
	}
}

// s1243: signing header table from aion v30 wire.
func TestFormatWebhookSigning_V30Entries(t *testing.T) {
	raw := `{
		"fleet_enabled": false,
		"fleet_env_var": "IOMESH_WEBHOOK_SIGN_FLEET",
		"count": 2,
		"entries": [
			{"connector_id":"github","label":"GitHub","mesh_layer":"operational",
			 "scheme":"hmac_sha256","primary_header":"X-Hub-Signature-256",
			 "signature_prefix":"sha256=","secret_env_var":"GITHUB_WEBHOOK_SECRET",
			 "vendor_native_verify":"yes"},
			{"connector_id":"notion","label":"Notion","mesh_layer":"knowledge",
			 "scheme":"hmac_sha256","primary_header":"X-Notion-Signature",
			 "signature_prefix":"","secret_env_var":"NOTION_WEBHOOK_SECRET",
			 "vendor_native_verify":"partial"}
		]
	}`
	out := formatWebhookSigning(raw, "", "")
	if !strings.Contains(out, "github") || !strings.Contains(out, "X-Hub-Signature-256") {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(out, "fleet_env_var:") {
		t.Fatalf("fleet: %s", out)
	}
	if !strings.Contains(out, "discovery only") {
		t.Fatalf("honesty: %s", out)
	}
	// client-side connector filter
	gh := formatWebhookSigning(raw, "github", "github")
	if !strings.Contains(gh, "github") {
		t.Fatalf("filter github: %s", gh)
	}
	if strings.Contains(gh, "notion") {
		t.Fatalf("filter should drop notion: %s", gh)
	}
}

func TestFormatWebhookSigning_Empty(t *testing.T) {
	out := formatWebhookSigning(`{"count":0,"entries":[]}`, "knowledge", "")
	if !strings.Contains(out, "no signing header entries") {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(out, "discovery only") {
		t.Fatalf("%s", out)
	}
}

// s1247: offline Runtime (no MCP) → residual-honest status pulse, no invent counts.
func TestIntegrationsStatus_OfflineFailOpen(t *testing.T) {
	rt := &Runtime{} // no MCP
	out, err := rt.IntegrationsStatus(context.Background())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "s1247") {
		t.Fatalf("want s1247 pulse: %s", out)
	}
	if !strings.Contains(out, "MCP path:") || !strings.Contains(out, "offline") {
		t.Fatalf("want MCP offline: %s", out)
	}
	if !strings.Contains(out, "fail-open") {
		t.Fatalf("want fail-open: %s", out)
	}
	// each expected tool marked offline
	for _, tool := range []string{
		"list_connector_catalog",
		"plan_connector_setup",
		"get_webhook_signing_headers",
	} {
		if !strings.Contains(out, tool) {
			t.Fatalf("want tool %s: %s", tool, out)
		}
	}
	// catalog pulse skipped — no invent counts
	if !strings.Contains(out, "skipped") || !strings.Contains(out, "no invent counts") {
		t.Fatalf("want skipped catalog pulse: %s", out)
	}
	if strings.Contains(out, "total catalog entries:") || strings.Contains(out, "MESH_LAYER") {
		t.Fatalf("must not invent catalog counts offline: %s", out)
	}
	// honesty footer always
	if !strings.Contains(out, "never invent install green") {
		t.Fatalf("honesty footer: %s", out)
	}
	if !strings.Contains(out, "catalog ≠ installs") && !strings.Contains(out, "catalog count ≠ install") {
		t.Fatalf("catalog ≠ install honesty: %s", out)
	}
	if !strings.Contains(out, "browser HITL") {
		t.Fatalf("browser HITL: %s", out)
	}
	if !strings.Contains(out, "dual_write OFF") {
		t.Fatalf("dual_write OFF: %s", out)
	}
	if !strings.Contains(out, "console.iome.sh/integrations") {
		t.Fatalf("portal: %s", out)
	}
	// must not invent install green / Connected success
	if strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent install green: %s", out)
	}
	// s1263: org installs residual honesty always (offline too)
	assertStatusOrgInstallsHonesty(t, out)
}

// s1247: empty MCP manager (Len=0) is residual offline/empty, not invent.
func TestIntegrationsStatus_EmptyManager(t *testing.T) {
	rt := &Runtime{mcp: mcp.NewManagerEmpty(nil)}
	out, err := rt.IntegrationsStatus(context.Background())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	// empty manager → empty or offline path; tools offline; no invent counts
	if !strings.Contains(out, "fail-open") && !strings.Contains(out, "empty") && !strings.Contains(out, "offline") {
		t.Fatalf("want empty/offline residual: %s", out)
	}
	if strings.Contains(out, "count: ") && !strings.Contains(out, "skipped") {
		// only fail if we invented a real pulse count without list tool
		if strings.Contains(out, "by mesh_layer") {
			t.Fatalf("must not invent layer counts: %s", out)
		}
	}
	if !strings.Contains(out, "never invent install green") {
		t.Fatalf("honesty: %s", out)
	}
	// s1263: org installs residual still present on empty manager
	assertStatusOrgInstallsHonesty(t, out)
}

// s1247: formatCatalogPulse from aion v178 entries — catalog honesty only, never install green.
func TestFormatCatalogPulse_V178Entries(t *testing.T) {
	raw := `{
		"count": 3,
		"entries": [
			{"id":"github","label":"GitHub","status":"available","mesh_layer":"operational",
			 "ingress_type":"webhook","oauth_install_supported":false},
			{"id":"notion","label":"Notion","status":"beta","mesh_layer":"knowledge",
			 "ingress_type":"oauth","oauth_install_supported":true},
			{"id":"embeddings","label":"Embeddings","status":"planned","mesh_layer":"analytical",
			 "ingress_type":"api","oauth_install_supported":false}
		]
	}`
	out := formatCatalogPulse(raw)
	if out == "" {
		t.Fatal("expected non-empty pulse")
	}
	if !strings.Contains(out, "total catalog entries: 3") {
		t.Fatalf("count: %s", out)
	}
	if !strings.Contains(out, "catalog honesty") || !strings.Contains(out, "NOT install Connected") {
		t.Fatalf("catalog honesty label: %s", out)
	}
	if !strings.Contains(out, "NOT INSTALL_STORE green") {
		t.Fatalf("INSTALL_STORE residual: %s", out)
	}
	if !strings.Contains(out, "by mesh_layer:") {
		t.Fatalf("mesh_layer line: %s", out)
	}
	if !strings.Contains(out, "operational=1") || !strings.Contains(out, "knowledge=1") || !strings.Contains(out, "analytical=1") {
		t.Fatalf("by mesh_layer: %s", out)
	}
	if !strings.Contains(out, "by catalog status:") {
		t.Fatalf("status line: %s", out)
	}
	if !strings.Contains(out, "available=1") || !strings.Contains(out, "beta=1") || !strings.Contains(out, "planned=1") {
		t.Fatalf("status counts: %s", out)
	}
	// must not claim install Connected / INSTALL_STORE green as success
	if strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not claim install green: %s", out)
	}
	// status chips are honesty only
	if !strings.Contains(out, "available/beta/planned") {
		t.Fatalf("status honesty note: %s", out)
	}
}

func TestFormatCatalogPulse_Empty(t *testing.T) {
	out := formatCatalogPulse(`{"count":0,"entries":[]}`)
	if !strings.Contains(out, "total catalog entries: 0") {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(out, "catalog honesty") {
		t.Fatalf("%s", out)
	}
}

func TestFormatCatalogPulse_NonJSON(t *testing.T) {
	if formatCatalogPulse("not json") != "" {
		t.Fatal("want empty for non-json")
	}
	if formatCatalogPulse("") != "" {
		t.Fatal("want empty for empty")
	}
	if formatCatalogPulse("plain text catalog") != "" {
		t.Fatal("want empty for plain text")
	}
}

func TestFormatCatalogPulse_LegacyConnectors(t *testing.T) {
	raw := `{
		"connectors": [
			{"id":"github","status":"available","mesh_layer":"operational"},
			{"id":"notion","status":"beta","mesh_layer":"knowledge"}
		]
	}`
	out := formatCatalogPulse(raw)
	if !strings.Contains(out, "total catalog entries: 2") {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(out, "operational=1") || !strings.Contains(out, "knowledge=1") {
		t.Fatalf("layers: %s", out)
	}
}

func TestStatusHonestyFooter(t *testing.T) {
	out := statusHonestyFooter()
	for _, want := range []string{
		"never invent install green",
		"catalog ≠ installs",
		"browser HITL",
		"stub ≠ live",
		"dual_write OFF",
		"book-demo OFF",
		"signing discovery only",
		"console.iome.sh/integrations",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in: %s", want, out)
		}
	}
}

// s1247: mock MCP with list_connector_catalog → status shows count, never install green.
func TestIntegrationsStatus_MockCatalogPresent(t *testing.T) {
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
	rt := &Runtime{mcp: mgr}

	out, err := rt.IntegrationsStatus(context.Background())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "MCP path:") || !strings.Contains(out, "available") {
		t.Fatalf("MCP available: %s", out)
	}
	if !strings.Contains(out, "list_connector_catalog") || !strings.Contains(out, "present") {
		t.Fatalf("list tool present: %s", out)
	}
	// plan/signing listed as missing (only list is on mock) — residual-honest
	if !strings.Contains(out, "plan_connector_setup") {
		t.Fatalf("plan tool line: %s", out)
	}
	if !strings.Contains(out, "total catalog entries: 3") {
		t.Fatalf("catalog count from fixture: %s", out)
	}
	if !strings.Contains(out, "operational=1") {
		t.Fatalf("layer counts: %s", out)
	}
	if !strings.Contains(out, "by catalog status:") {
		t.Fatalf("status counts: %s", out)
	}
	if !strings.Contains(out, "catalog honesty") || !strings.Contains(out, "NOT install Connected") {
		t.Fatalf("catalog honesty: %s", out)
	}
	if strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent install Connected: %s", out)
	}
	if !strings.Contains(out, "never invent install green") {
		t.Fatalf("honesty footer: %s", out)
	}
	if !strings.Contains(out, "dual_write OFF") {
		t.Fatalf("dual_write: %s", out)
	}
	// s1263: org installs residual honesty always (online catalog path too)
	assertStatusOrgInstallsHonesty(t, out)
}

// assertStatusOrgInstallsHonesty checks s1263 residual-honest org install snapshot needles.
// Always required on IntegrationsStatus (offline and online) — never invent Connected / empty-as-none.
func assertStatusOrgInstallsHonesty(t *testing.T, out string) {
	t.Helper()
	if !strings.Contains(out, "org installs") {
		t.Fatalf("s1263 want org installs: %s", out)
	}
	if !strings.Contains(out, "unavailable") {
		t.Fatalf("s1263 want unavailable via agent MCP: %s", out)
	}
	if !strings.Contains(out, statusOrgInstallsUnavailableLine) {
		t.Fatalf("s1263 want constant unavailable line: %s", out)
	}
	if !strings.Contains(out, "candidacy open") {
		t.Fatalf("s1263 want dual-auth candidacy open: %s", out)
	}
	if !strings.Contains(out, "never invent Connected") {
		t.Fatalf("s1263 want never invent Connected: %s", out)
	}
	if !strings.Contains(out, "empty-as-none") {
		t.Fatalf("s1263 want empty-as-none residual: %s", out)
	}
	if !strings.Contains(out, "portal session HITL") {
		t.Fatalf("s1263 want portal session HITL: %s", out)
	}
	if !strings.Contains(out, "console.iome.sh/integrations") {
		t.Fatalf("s1263 want portal URL: %s", out)
	}
	// must never invent install rows / Connected green from missing snapshot
	if strings.Contains(out, "Connected: yes") {
		t.Fatalf("s1263 must not invent install green: %s", out)
	}
	if strings.Contains(out, "installs: 0") || strings.Contains(out, "org installs: 0") {
		t.Fatalf("s1263 must not invent empty-as-none installs: %s", out)
	}
	if strings.Contains(out, "dual-auth shipped") || strings.Contains(out, "dual-auth: live") {
		t.Fatalf("s1263 must not claim dual-auth shipped: %s", out)
	}
}

// s1263: offline status always reports org installs unavailable (portal HITL only).
func TestIntegrationsStatus_S1263OrgInstallsOffline(t *testing.T) {
	rt := &Runtime{}
	out, err := rt.IntegrationsStatus(context.Background())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	assertStatusOrgInstallsHonesty(t, out)
	// still residual — no invent install green / catalog counts offline
	if strings.Contains(out, "total catalog entries:") {
		t.Fatalf("must not invent catalog counts offline: %s", out)
	}
	if !strings.Contains(out, "never invent install green") {
		t.Fatalf("honesty footer: %s", out)
	}
}

// s1263: statusOrgInstallsSection is residual-honest and stable for unit needles.
func TestStatusOrgInstallsSection(t *testing.T) {
	out := statusOrgInstallsSection()
	for _, want := range []string{
		statusOrgInstallsUnavailableLine,
		statusOrgInstallsDualAuthLine,
		"portal: " + integrationsPortalURL,
		"org installs",
		"unavailable",
		"candidacy open",
		"never invent Connected",
		"empty-as-none",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in: %s", want, out)
		}
	}
	if strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent install green: %s", out)
	}
}

// mockIntegrationsMCP serves tools/list + list_connector_catalog with aion v178 entries.
func mockIntegrationsMCP(w io.WriteCloser, r io.Reader) {
	defer w.Close()
	dec := json.NewDecoder(r)
	catalog := `{
		"count": 3,
		"entries": [
			{"id":"github","label":"GitHub","status":"available","mesh_layer":"operational",
			 "ingress_type":"webhook","oauth_install_supported":false,"portal_path":"/integrations/github"},
			{"id":"notion","label":"Notion","status":"beta","mesh_layer":"knowledge",
			 "ingress_type":"oauth","oauth_install_supported":true,"portal_path":"/integrations/notion"},
			{"id":"embeddings","label":"Embeddings","status":"beta","mesh_layer":"analytical",
			 "ingress_type":"api","oauth_install_supported":false,"portal_path":"/integrations/embeddings"}
		]
	}`
	for {
		var req map[string]any
		if err := dec.Decode(&req); err != nil {
			return
		}
		id := req["id"]
		method, _ := req["method"].(string)
		if method == "notifications/initialized" || id == nil {
			continue
		}
		var result any
		switch method {
		case "initialize":
			result = map[string]any{"protocolVersion": "2024-11-05", "serverInfo": map[string]string{"name": "aion", "version": "1"}}
		case "tools/list":
			result = map[string]any{"tools": []map[string]any{{
				"name": "list_connector_catalog", "description": "Catalog (v178)", "inputSchema": map[string]any{
					"type": "object", "properties": map[string]any{},
				},
			}}}
		case "tools/call":
			name := ""
			if params, ok := req["params"].(map[string]any); ok {
				name, _ = params["name"].(string)
			}
			text := "unknown tool"
			if name == "list_connector_catalog" {
				text = catalog
			}
			result = map[string]any{"content": []map[string]any{{"type": "text", "text": text}}}
		default:
			result = map[string]any{}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
	}
}

// --- s1251: agent guidance note ---

func TestIntegrationsAgentGuidanceNote_HonestyNeedles(t *testing.T) {
	out := IntegrationsAgentGuidanceNote()
	if out == "" {
		t.Fatal("empty guidance note")
	}
	for _, want := range []string{
		"list_connector_catalog",
		"plan_connector_setup",
		"get_webhook_signing_headers",
		"console.iome.sh/integrations",
		"browser HITL",
		"never invent install green",
		"Connected",
		"INSTALL_STORE",
		"stub OAuth",
		"dual_write OFF",
		"book-demo OFF",
		"catalog Beta",
		"agent MCP cannot write installs",
		"/integrations status",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("guidance missing %q in:\n%s", want, out)
		}
	}
	// Must not invent install success language as a claim
	if strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent install green claim: %s", out)
	}
}

// s1251: AttachMCP injects <integrations> system note when MCP manager is present.
func TestAttachMCP_InjectsIntegrationsGuidance(t *testing.T) {
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
	if !strings.Contains(sys, "<integrations>") {
		t.Fatalf("want <integrations> system note: %s", sys)
	}
	if !strings.Contains(sys, "</integrations>") {
		t.Fatalf("want closed integrations tag: %s", sys)
	}
	// Same needles as the guidance note
	for _, want := range []string{
		"list_connector_catalog",
		"plan_connector_setup",
		"never invent install green",
		"browser HITL",
		"dual_write OFF",
		"book-demo OFF",
	} {
		if !strings.Contains(sys, want) {
			t.Fatalf("integrations note missing %q: %s", want, sys)
		}
	}
	// MCP note still present
	if !strings.Contains(sys, "<mcp>") {
		t.Fatalf("want <mcp> note too: %s", sys)
	}
}

// --- s1252: golden fixtures expansion ---

func TestFormatConnectorCatalog_GoldenFixture(t *testing.T) {
	raw := readTestdata(t, "v178_catalog_entries.json")
	out := formatConnectorCatalog(raw, "")
	if out == "" {
		t.Fatal("empty catalog format")
	}
	// Round-trip: table includes oauth_install_supported → yes/no
	for _, want := range []string{"github", "notion", "embeddings", "ID", "STATUS", "MESH_LAYER", "OAUTH"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q: %s", want, out)
		}
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "notion") {
			if !strings.Contains(line, "yes") {
				t.Fatalf("notion oauth want yes: %s", line)
			}
		}
		if strings.Contains(line, "github") {
			if !strings.Contains(line, "no") {
				t.Fatalf("github oauth want no: %s", line)
			}
		}
	}
	if !strings.Contains(out, "honesty:") {
		t.Fatalf("honesty footer: %s", out)
	}
	if strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent install green: %s", out)
	}
	// layer filter round-trip
	know := formatConnectorCatalog(raw, "knowledge")
	if !strings.Contains(know, "notion") || strings.Contains(know, "github") {
		t.Fatalf("knowledge filter: %s", know)
	}
}

func TestFormatConnectorPlan_GoldenFixtureDeepLinks(t *testing.T) {
	raw := readTestdata(t, "v178_plan_github.json")
	out := formatConnectorPlan(raw, "github")
	if out == "" {
		t.Fatal("empty plan format")
	}
	// portal_url + honesty notes + deep links (s1244)
	for _, want := range []string{
		"portal_url:",
		"console.iome.sh/integrations/github",
		"portal_add_url:",
		"template=github",
		"deep_links:",
		"add_wizard:",
		"catalog:",
		"signing_headers_tool:",
		"get_webhook_signing_headers",
		"Browser HITL",
		"never invent install green",
		"available", // catalog status honesty
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("plan missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent Connected: %s", out)
	}
	// never invent focus= fantasy
	if strings.Contains(out, "focus=") {
		t.Fatalf("must not invent focus= deep links: %s", out)
	}
}

func TestFormatConnectorPlan_GoldenFixtureNotionStub(t *testing.T) {
	raw := readTestdata(t, "v178_plan_notion.json")
	out := formatConnectorPlan(raw, "notion")
	if !strings.Contains(out, "oauth_mode_hint: stub") {
		t.Fatalf("stub hint: %s", out)
	}
	if !strings.Contains(out, "portal_add_url:") || !strings.Contains(out, "template=notion") {
		t.Fatalf("portal_add_url: %s", out)
	}
	if !strings.Contains(out, "deep_links:") {
		t.Fatalf("deep_links: %s", out)
	}
	if !strings.Contains(out, "stub ≠ live") {
		t.Fatalf("honesty: %s", out)
	}
	if !strings.Contains(out, "oauth_install_supported: true") {
		t.Fatalf("oauth_install_supported: %s", out)
	}
	if strings.Contains(out, "Connected") && strings.Contains(out, "yes") {
		// status "beta" is ok; invent Connected green is not
		if strings.Contains(out, "Connected: yes") {
			t.Fatalf("must not invent install green: %s", out)
		}
	}
}

func TestFormatCatalogPulse_GoldenFixture(t *testing.T) {
	raw := readTestdata(t, "v178_catalog_entries.json")
	out := formatCatalogPulse(raw)
	if !strings.Contains(out, "total catalog entries: 3") {
		t.Fatalf("count: %s", out)
	}
	if !strings.Contains(out, "operational=1") || !strings.Contains(out, "knowledge=1") || !strings.Contains(out, "analytical=1") {
		t.Fatalf("layers: %s", out)
	}
	if !strings.Contains(out, "available=1") || !strings.Contains(out, "beta=1") || !strings.Contains(out, "planned=1") {
		t.Fatalf("status: %s", out)
	}
	if !strings.Contains(out, "catalog honesty") || !strings.Contains(out, "NOT install Connected") {
		t.Fatalf("honesty: %s", out)
	}
}

// s1252: IntegrationsStatus offline still residual; golden catalog not invented offline.
func TestIntegrationsStatus_OfflineNoGoldenInvent(t *testing.T) {
	rt := &Runtime{}
	out, err := rt.IntegrationsStatus(context.Background())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if strings.Contains(out, "total catalog entries: 3") {
		t.Fatalf("must not invent golden catalog counts offline: %s", out)
	}
	if !strings.Contains(out, "never invent install green") {
		t.Fatalf("honesty: %s", out)
	}
	if !strings.Contains(out, "dual_write OFF") || !strings.Contains(out, "book-demo OFF") {
		t.Fatalf("residual locks: %s", out)
	}
}

// --- s1257: deep-link parity + residual-honest dogfood ---

// TestFormatConnectorPlan_S1257DeepLinkParityGithub asserts aion s1244 field round-trip
// for github golden fixture: portal_url / portal_detail_url / portal_add_url
// (template={id}) / deep_links map, and honesty footer never invents install green.
func TestFormatConnectorPlan_S1257DeepLinkParityGithub(t *testing.T) {
	raw := readTestdata(t, "v178_plan_github.json")

	// Fixture itself carries s1244 shape (JSON round-trip dogfood).
	var p map[string]any
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("fixture JSON: %v", err)
	}
	for _, key := range []string{"portal_url", "portal_detail_url", "portal_add_url", "deep_links"} {
		if _, ok := p[key]; !ok {
			t.Fatalf("fixture missing s1244 field %q", key)
		}
	}
	addURL, _ := p["portal_add_url"].(string)
	if !strings.Contains(addURL, "/integrations/add?template=github") {
		t.Fatalf("portal_add_url template= shape: %q", addURL)
	}
	dl, ok := p["deep_links"].(map[string]any)
	if !ok || len(dl) == 0 {
		t.Fatalf("deep_links map: %#v", p["deep_links"])
	}
	for _, k := range []string{"detail", "add_wizard", "catalog", "portal_add_url"} {
		v, _ := dl[k].(string)
		if strings.TrimSpace(v) == "" {
			t.Fatalf("deep_links[%q] empty: %#v", k, dl)
		}
	}
	addWizard, _ := dl["add_wizard"].(string)
	if !strings.Contains(addWizard, "template=github") {
		t.Fatalf("add_wizard template=: %q", addWizard)
	}

	out := formatConnectorPlan(raw, "github")
	// Formatter surfaces portal_add_url + deep_links with residual labels.
	for _, want := range []string{
		"portal_url:",
		"https://console.iome.sh/integrations/github",
		"portal_add_url:",
		"/integrations/add?template=github",
		"template=github",
		"browser HITL add wizard",
		"not install APPLY",
		"deep_links:",
		"browser HITL only",
		"not install green",
		"detail:",
		"add_wizard:",
		"catalog:",
		"portal_detail_url:",
		"signing_headers_tool:",
		"get_webhook_signing_headers",
		"Browser HITL",
		"never invent install green",
		"available", // catalog status honesty only
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("plan missing %q:\n%s", want, out)
		}
	}
	// Honesty residual locks — never invent install green / Connected success.
	// Note: residual copy may say "not install APPLY success" — that is honest.
	// Only fail on affirmative invent claims.
	for _, bad := range []string{"Connected: yes", "INSTALL_STORE: green", "install APPLY: success"} {
		if strings.Contains(out, bad) {
			t.Fatalf("must not invent %q:\n%s", bad, out)
		}
	}
	// Must not claim APPLY success without the residual "not" framing.
	if strings.Contains(out, "install APPLY success") && !strings.Contains(out, "not install APPLY") {
		t.Fatalf("must not invent install APPLY success:\n%s", out)
	}
	// Proven console routes only — never invent focus= fantasy.
	if strings.Contains(out, "focus=") {
		t.Fatalf("must not invent focus= deep links:\n%s", out)
	}
	// template= is deep-link shape, not install APPLY claim.
	if !strings.Contains(out, "template=github") {
		t.Fatalf("template=github deep-link shape:\n%s", out)
	}
}

// TestFormatConnectorPlan_S1257DeepLinkParityNotion same parity for notion
// (stub oauth · template=notion · honesty residual).
func TestFormatConnectorPlan_S1257DeepLinkParityNotion(t *testing.T) {
	raw := readTestdata(t, "v178_plan_notion.json")

	var p map[string]any
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("fixture JSON: %v", err)
	}
	addURL, _ := p["portal_add_url"].(string)
	if !strings.Contains(addURL, "/integrations/add?template=notion") {
		t.Fatalf("portal_add_url template= shape: %q", addURL)
	}

	out := formatConnectorPlan(raw, "notion")
	for _, want := range []string{
		"portal_url:",
		"console.iome.sh/integrations/notion",
		"portal_add_url:",
		"template=notion",
		"deep_links:",
		"add_wizard:",
		"oauth_mode_hint: stub",
		"oauth_install_supported: true",
		"stub ≠ live",
		"never invent install green",
		"browser HITL",
		"not install APPLY",
	} {
		if !strings.Contains(out, want) {
			// case: "browser HITL" may appear as "Browser HITL" — tolerate either
			if want == "browser HITL" && strings.Contains(strings.ToLower(out), "browser hitl") {
				continue
			}
			t.Fatalf("notion plan missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent install green:\n%s", out)
	}
	if strings.Contains(out, "focus=") {
		t.Fatalf("must not invent focus=:\n%s", out)
	}
	// template=notion deep-link ≠ install APPLY
	if !strings.Contains(out, "template=notion") {
		t.Fatalf("template=notion:\n%s", out)
	}
}

// TestFormatConnectorPlan_S1257DeepLinksFromMapOnly: when only deep_links is set
// (no top-level portal_url / portal_add_url), formatter still surfaces routes.
func TestFormatConnectorPlan_S1257DeepLinksFromMapOnly(t *testing.T) {
	raw := `{
		"connector_id": "github",
		"connector": {"id":"github","status":"available","mesh_layer":"operational"},
		"deep_links": {
			"detail": "https://console.iome.sh/integrations/github",
			"add_wizard": "https://console.iome.sh/integrations/add?template=github",
			"catalog": "https://console.iome.sh/integrations"
		},
		"next_steps": ["Open portal"],
		"honesty": {"notes": ["Browser HITL required for OAuth complete", "never invent install green"]}
	}`
	out := formatConnectorPlan(raw, "github")
	if !strings.Contains(out, "portal_url:") || !strings.Contains(out, "console.iome.sh/integrations/github") {
		t.Fatalf("portal from deep_links.detail: %s", out)
	}
	if !strings.Contains(out, "portal_add_url:") || !strings.Contains(out, "template=github") {
		t.Fatalf("portal_add from deep_links.add_wizard: %s", out)
	}
	if !strings.Contains(out, "deep_links:") || !strings.Contains(out, "add_wizard:") {
		t.Fatalf("deep_links block: %s", out)
	}
	if !strings.Contains(out, "never invent install green") {
		t.Fatalf("honesty: %s", out)
	}
	if strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent Connected: %s", out)
	}
}

// TestFormatCatalogPulse_S1257PortalPathOnly: catalog entries with portal_path only
// still residual-honest — count/layer honesty, never install green.
func TestFormatCatalogPulse_S1257PortalPathOnly(t *testing.T) {
	raw := `{
		"count": 2,
		"entries": [
			{"id":"github","label":"GitHub","status":"available","mesh_layer":"operational",
			 "portal_path":"/integrations/github"},
			{"id":"notion","label":"Notion","status":"beta","mesh_layer":"knowledge",
			 "portal_path":"/integrations/notion"}
		]
	}`
	out := formatCatalogPulse(raw)
	if out == "" {
		t.Fatal("expected non-empty pulse")
	}
	if !strings.Contains(out, "total catalog entries: 2") {
		t.Fatalf("count: %s", out)
	}
	if !strings.Contains(out, "operational=1") || !strings.Contains(out, "knowledge=1") {
		t.Fatalf("layers: %s", out)
	}
	if !strings.Contains(out, "catalog honesty") || !strings.Contains(out, "NOT install Connected") {
		t.Fatalf("catalog honesty: %s", out)
	}
	if strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent install green: %s", out)
	}
	// portal_path is catalog field only — pulse must not invent install APPLY
	if strings.Contains(out, "INSTALL_STORE APPLY") || strings.Contains(out, "install green: yes") {
		t.Fatalf("must not invent install APPLY: %s", out)
	}
}

// TestFormatConnectorCatalog_S1257PortalPathHonesty: catalog table with portal_path
// entries still carries honesty footer (portal_path is not install Connected).
func TestFormatConnectorCatalog_S1257PortalPathHonesty(t *testing.T) {
	raw := readTestdata(t, "v178_catalog_entries.json")
	out := formatConnectorCatalog(raw, "")
	if !strings.Contains(out, "github") || !strings.Contains(out, "notion") {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(out, "honesty:") {
		t.Fatalf("honesty footer: %s", out)
	}
	if !strings.Contains(out, "never invent install green") {
		t.Fatalf("honesty needle: %s", out)
	}
	if strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent install green: %s", out)
	}
}

// TestS1257PlanHonestyFooterNeverInventInstallGreen pins residual footer copy
// on plan formatter when honesty notes are sparse.
func TestS1257PlanHonestyFooterNeverInventInstallGreen(t *testing.T) {
	raw := `{"connector_id":"github","portal_url":"https://console.iome.sh/integrations/github"}`
	out := formatConnectorPlan(raw, "github")
	if !strings.Contains(out, "never invent install green") && !strings.Contains(out, IntegrationsHonestyOneLiner) {
		t.Fatalf("residual honesty footer: %s", out)
	}
	if strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent Connected: %s", out)
	}
	// default portal is HITL deep-link shape, not APPLY
	if !strings.Contains(out, "portal_url:") {
		t.Fatalf("portal_url: %s", out)
	}
}

// --- s1257: guidance note ↔ builtin skill honesty consistency (lightweight) ---

// TestS1257GuidanceNoteSkillHonestyConsistency loads the embedded skill body and
// asserts IntegrationsAgentGuidanceNote() shares core honesty needles.
// Keeps system note and skill from drifting on residual locks.
func TestS1257GuidanceNoteSkillHonestyConsistency(t *testing.T) {
	// Load skill the same way AttachSkills does (LoadWithBuiltin / LoadBuiltin).
	// Avoid import cycle: re-parse fixture-equivalent needles from guidance +
	// require the skill file content via relative path under internal/skills.
	//
	// We read the embedded skill markdown from disk (source of go:embed) so this
	// package does not import internal/skills (agent may already be depended on
	// by skills/tui — keep agent self-contained).
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	skillPath := filepath.Join(filepath.Dir(file), "..", "skills", "builtin", "connector-integrations-setup", "SKILL.md")
	skillBytes, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read skill: %v", err)
	}
	skillText := string(skillBytes)
	guidance := IntegrationsAgentGuidanceNote()

	// Core honesty needles that BOTH must carry (s1257 residual lock set).
	needles := []string{
		"list_connector_catalog",
		"plan_connector_setup",
		"portal",
		"never invent install green",
	}
	// At least one of dual_write OFF / browser HITL must appear in each.
	softNeedles := []string{
		"dual_write OFF",
		"browser HITL",
	}

	for _, n := range needles {
		if !strings.Contains(guidance, n) {
			t.Fatalf("guidance missing %q:\n%s", n, guidance)
		}
		if !strings.Contains(skillText, n) {
			t.Fatalf("skill missing %q:\n%s", n, skillText)
		}
	}
	// Soft: each source must have at least one residual soft needle.
	guidanceSoft := false
	skillSoft := false
	for _, n := range softNeedles {
		if strings.Contains(guidance, n) {
			guidanceSoft = true
		}
		if strings.Contains(skillText, n) {
			skillSoft = true
		}
	}
	if !guidanceSoft {
		t.Fatalf("guidance missing dual_write OFF / browser HITL:\n%s", guidance)
	}
	if !skillSoft {
		t.Fatalf("skill missing dual_write OFF / browser HITL:\n%s", skillText)
	}
	// Neither invents install green claims.
	for _, src := range []struct {
		name string
		text string
	}{
		{"guidance", guidance},
		{"skill", skillText},
	} {
		if strings.Contains(src.text, "Connected: yes") {
			t.Fatalf("%s invents install green", src.name)
		}
	}
}
