package config

import "testing"

func TestInferHooksEndpoint(t *testing.T) {
	t.Parallel()
	if got := InferHooksEndpoint("https://apiv1.iome.sh/v7/mcp"); got != "https://hooks.iome.sh" {
		t.Fatalf("prod = %q", got)
	}
	if got := InferHooksEndpoint("https://apiv1.staging.iome.sh/v7/mcp"); got != "https://hooks.staging.iome.sh" {
		t.Fatalf("stage = %q", got)
	}
	if got := InferHooksEndpoint("https://example.invalid/mcp"); got != "" {
		t.Fatalf("unknown = %q", got)
	}
}

func TestInferBrokerFromPortalMCPHeaders(t *testing.T) {
	t.Parallel()
	cfg := &Config{}
	cfg.MCP.Servers = []MCPServerTOML{{
		Name: "io-mesh",
		URL:  "https://apiv1.iome.sh/v7/mcp",
		Headers: map[string]string{
			"X-IOMesh-Tenant": "dept.engineering",
			"X-IOMesh-Org":    "org_example",
		},
	}}
	got := cfg.InferBrokerFromPortalMCP()
	if got.Endpoint != "https://hooks.iome.sh" {
		t.Fatalf("endpoint = %q", got.Endpoint)
	}
	if got.Tenant != "dept.engineering" || got.Org != "org_example" {
		t.Fatalf("%+v", got)
	}
}
