package main

import (
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/config"
)

func TestCmdMeshStreams_CreateRequiresYes(t *testing.T) {
	if code := cmdMeshStreams([]string{"--create"}); code != 2 {
		t.Fatalf("missing --yes exit=%d want 2", code)
	}
}

func TestCmdMeshStreams_CreateIncompatibleWithDeleteAndMessages(t *testing.T) {
	if code := cmdMeshStreams([]string{"--create", "--yes", "--delete", "--name", "X"}); code != 2 {
		t.Fatalf("create+delete exit=%d want 2", code)
	}
	if code := cmdMeshStreams([]string{"--create", "--yes", "--messages", "--name", "X"}); code != 2 {
		t.Fatalf("create+messages exit=%d want 2", code)
	}
}

func portalMCPConfig() *config.Config {
	cfg := &config.Config{}
	cfg.MCP.Servers = []config.MCPServerTOML{{
		Name: "io-mesh",
		URL:  "https://apiv1.iome.sh/v7/mcp",
		Headers: map[string]string{
			"X-IOMesh-Tenant": "dept.engineering",
			"X-IOMesh-Org":    "org_example",
		},
	}}
	return cfg
}

func TestApplyInferredBroker_Nil(t *testing.T) {
	applyInferredBroker(nil)
}

func TestApplyInferredBroker_EmptyInferDoesNotInventEnabled(t *testing.T) {
	cfg := &config.Config{}
	applyInferredBroker(cfg)
	if cfg.IOMesh.Enabled || cfg.IOMesh.Endpoint != "" {
		t.Fatalf("empty infer invented mesh: enabled=%v endpoint=%q", cfg.IOMesh.Enabled, cfg.IOMesh.Endpoint)
	}
}

func TestApplyInferredBroker_FromPortalMCP(t *testing.T) {
	cfg := portalMCPConfig()
	applyInferredBroker(cfg)
	if !cfg.IOMesh.Enabled || cfg.IOMesh.Endpoint != "https://hooks.iome.sh" {
		t.Fatalf("infer endpoint enabled=%v endpoint=%q", cfg.IOMesh.Enabled, cfg.IOMesh.Endpoint)
	}
	if cfg.IOMesh.Tenant != "dept.engineering" || cfg.IOMesh.Org != "org_example" {
		t.Fatalf("infer tenant/org tenant=%q org=%q", cfg.IOMesh.Tenant, cfg.IOMesh.Org)
	}
}

func TestApplyInferredBroker_EndpointWins(t *testing.T) {
	cfg := portalMCPConfig()
	cfg.IOMesh.Enabled = true
	cfg.IOMesh.Endpoint = "https://hooks.example.test"
	cfg.IOMesh.Tenant = "dept.other"
	cfg.IOMesh.Org = "org_other"
	applyInferredBroker(cfg)
	if cfg.IOMesh.Endpoint != "https://hooks.example.test" || cfg.IOMesh.Tenant != "dept.other" || cfg.IOMesh.Org != "org_other" {
		t.Fatalf("endpoint should win: %+v", cfg.IOMesh)
	}
}

func TestApplyInferredBroker_PreservesExistingTenantOrg(t *testing.T) {
	cfg := portalMCPConfig()
	cfg.IOMesh.Tenant = "dept.keep"
	cfg.IOMesh.Org = "org_keep"
	applyInferredBroker(cfg)
	if cfg.IOMesh.Endpoint != "https://hooks.iome.sh" {
		t.Fatalf("endpoint = %q", cfg.IOMesh.Endpoint)
	}
	if cfg.IOMesh.Tenant != "dept.keep" || cfg.IOMesh.Org != "org_keep" {
		t.Fatalf("should keep tenant/org: tenant=%q org=%q", cfg.IOMesh.Tenant, cfg.IOMesh.Org)
	}
}
