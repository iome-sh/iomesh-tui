package agent

import (
	"context"
	"strings"
	"testing"
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

func TestFormatConnectorCatalog_Table(t *testing.T) {
	raw := `{
		"connectors": [
			{"id":"github","status":"available","mesh_layer":"operational","oauth":true},
			{"id":"notion","status":"beta","mesh_layer":"knowledge","ingress_type":"oauth"},
			{"id":"embeddings","status":"beta","mesh_layer":"analytical","oauth":false}
		]
	}`
	out := formatConnectorCatalog(raw, "")
	if !strings.Contains(out, "github") || !strings.Contains(out, "notion") {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(out, "ID") || !strings.Contains(out, "STATUS") {
		t.Fatalf("missing header: %s", out)
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

func TestFormatConnectorPlan_PortalAndHonesty(t *testing.T) {
	raw := `{
		"connector_id": "github",
		"portal_url": "https://console.iome.sh/integrations/github",
		"next_steps": ["Open portal", "Complete OAuth in browser"],
		"honesty_notes": ["Browser HITL for OAuth", "stub ≠ live"]
	}`
	out := formatConnectorPlan(raw, "github")
	if !strings.Contains(out, "portal_url:") || !strings.Contains(out, "console.iome.sh/integrations/github") {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(out, "Open portal") {
		t.Fatalf("next_steps: %s", out)
	}
	if !strings.Contains(out, "Browser HITL") {
		t.Fatalf("honesty: %s", out)
	}
	if !strings.Contains(out, "never invent install green") && !strings.Contains(out, IntegrationsHonestyOneLiner) {
		t.Fatalf("residual honesty: %s", out)
	}
}

func TestFormatConnectorPlan_DefaultPortal(t *testing.T) {
	raw := `{"connector_id":"slack"}`
	out := formatConnectorPlan(raw, "slack")
	if !strings.Contains(out, integrationsPortalURL+"/slack") {
		t.Fatalf("default portal: %s", out)
	}
}
