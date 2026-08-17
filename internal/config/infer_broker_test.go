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

func portalMCPConfig() *Config {
	cfg := &Config{}
	cfg.MCP.Servers = []MCPServerTOML{{
		Name: "io-mesh",
		URL:  "https://apiv1.iome.sh/v7/mcp",
		Headers: map[string]string{
			"X-IOMesh-Tenant": "dept.engineering",
			"X-IOMesh-Org":    "org_example",
		},
	}}
	return cfg
}

func TestInferBrokerFromPortalMCPHeaders(t *testing.T) {
	t.Parallel()
	got := portalMCPConfig().InferBrokerFromPortalMCP()
	if got.Endpoint != "https://hooks.iome.sh" {
		t.Fatalf("endpoint = %q", got.Endpoint)
	}
	if got.Tenant != "dept.engineering" || got.Org != "org_example" {
		t.Fatalf("%+v", got)
	}
}

func TestApplyInferredBroker_Nil(t *testing.T) {
	t.Parallel()
	if got := ApplyInferredBroker(nil); got.Endpoint != "" {
		t.Fatalf("%+v", got)
	}
}

func TestApplyInferredBroker_EmptyInferDoesNotInventEnabled(t *testing.T) {
	t.Parallel()
	cfg := &Config{}
	if got := ApplyInferredBroker(cfg); got.Endpoint != "" {
		t.Fatalf("%+v", got)
	}
	if cfg.IOMesh.Enabled || cfg.IOMesh.Endpoint != "" {
		t.Fatalf("empty infer invented mesh: enabled=%v endpoint=%q", cfg.IOMesh.Enabled, cfg.IOMesh.Endpoint)
	}
}

func TestApplyInferredBroker_FromPortalMCP(t *testing.T) {
	t.Parallel()
	cfg := portalMCPConfig()
	got := ApplyInferredBroker(cfg)
	if got.Endpoint != "https://hooks.iome.sh" {
		t.Fatalf("infer return %+v", got)
	}
	if !cfg.IOMesh.Enabled || cfg.IOMesh.Endpoint != "https://hooks.iome.sh" {
		t.Fatalf("infer endpoint enabled=%v endpoint=%q", cfg.IOMesh.Enabled, cfg.IOMesh.Endpoint)
	}
	if cfg.IOMesh.Tenant != "dept.engineering" || cfg.IOMesh.Org != "org_example" {
		t.Fatalf("infer tenant/org tenant=%q org=%q", cfg.IOMesh.Tenant, cfg.IOMesh.Org)
	}
}

func TestApplyInferredBroker_EndpointWins(t *testing.T) {
	t.Parallel()
	cfg := portalMCPConfig()
	cfg.IOMesh.Enabled = true
	cfg.IOMesh.Endpoint = "https://hooks.example.test"
	cfg.IOMesh.Tenant = "dept.other"
	cfg.IOMesh.Org = "org_other"
	if got := ApplyInferredBroker(cfg); got.Endpoint != "" {
		t.Fatalf("explicit endpoint should skip infer: %+v", got)
	}
	if cfg.IOMesh.Endpoint != "https://hooks.example.test" || cfg.IOMesh.Tenant != "dept.other" || cfg.IOMesh.Org != "org_other" {
		t.Fatalf("endpoint should win: %+v", cfg.IOMesh)
	}
}

func TestApplyInferredBroker_PreservesExistingTenantOrg(t *testing.T) {
	t.Parallel()
	cfg := portalMCPConfig()
	cfg.IOMesh.Tenant = "dept.keep"
	cfg.IOMesh.Org = "org_keep"
	if got := ApplyInferredBroker(cfg); got.Endpoint != "https://hooks.iome.sh" {
		t.Fatalf("%+v", got)
	}
	if cfg.IOMesh.Tenant != "dept.keep" || cfg.IOMesh.Org != "org_keep" {
		t.Fatalf("should keep tenant/org: tenant=%q org=%q", cfg.IOMesh.Tenant, cfg.IOMesh.Org)
	}
}
