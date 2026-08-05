package agent

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/mcp"
)

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
