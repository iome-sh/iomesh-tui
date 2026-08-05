package config

import (
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/mcp"
)

// s1267: building ServerConfig with inject flag sets headers from iomesh context.
func TestBuildMCPServerConfig_InjectSetsHeaders(t *testing.T) {
	cfg := Default()
	cfg.MCP.InjectIOMeshContext = true
	cfg.IOMesh.Tenant = "acme"
	cfg.IOMesh.Org = "org_dev"
	cfg.IOMesh.Workspace = "ws_alpha"
	s := MCPServerTOML{
		Name: "scenario",
		URL:  "https://mcp.example.com/mcp",
	}
	sc := cfg.BuildMCPServerConfig(s)
	if sc.Headers[mcp.HeaderIOMeshTenant] != "acme" {
		t.Fatalf("tenant header=%v", sc.Headers)
	}
	if sc.Headers[mcp.HeaderIOMeshOrg] != "org_dev" {
		t.Fatalf("org header=%v", sc.Headers)
	}
	if sc.Headers[mcp.HeaderIOMeshWorkspace] != "ws_alpha" {
		t.Fatalf("workspace header=%v", sc.Headers)
	}
}

// s1267: does not overwrite pre-set X-IOMesh-Org in server headers.
func TestBuildMCPServerConfig_NoOverwriteExplicitOrg(t *testing.T) {
	cfg := Default()
	cfg.MCP.InjectIOMeshContext = true
	cfg.IOMesh.Tenant = "acme"
	cfg.IOMesh.Org = "org_from_iomesh"
	cfg.IOMesh.Workspace = "ws_cfg"
	s := MCPServerTOML{
		Name: "scenario",
		URL:  "https://mcp.example.com/mcp",
		Headers: map[string]string{
			mcp.HeaderIOMeshOrg: "org_explicit",
		},
	}
	sc := cfg.BuildMCPServerConfig(s)
	if sc.Headers[mcp.HeaderIOMeshOrg] != "org_explicit" {
		t.Fatalf("must not overwrite Org: %v", sc.Headers)
	}
	if sc.Headers[mcp.HeaderIOMeshTenant] != "acme" {
		t.Fatalf("tenant still injected: %v", sc.Headers)
	}
	if sc.Headers[mcp.HeaderIOMeshWorkspace] != "ws_cfg" {
		t.Fatalf("workspace still injected: %v", sc.Headers)
	}
	// Config map must not be mutated.
	if s.Headers[mcp.HeaderIOMeshTenant] != "" {
		t.Fatal("must not mutate original server Headers map")
	}
}

// s1267: default inject=false leaves headers unchanged.
func TestBuildMCPServerConfig_DefaultInjectOff(t *testing.T) {
	cfg := Default()
	cfg.IOMesh.Tenant = "acme"
	cfg.IOMesh.Org = "org_dev"
	s := MCPServerTOML{
		Name: "scenario",
		URL:  "https://mcp.example.com/mcp",
		Headers: map[string]string{
			"Authorization": "Bearer x",
		},
	}
	sc := cfg.BuildMCPServerConfig(s)
	if len(sc.Headers) != 1 || sc.Headers["Authorization"] != "Bearer x" {
		t.Fatalf("default inject=false must leave headers unchanged: %v", sc.Headers)
	}
	if _, ok := sc.Headers[mcp.HeaderIOMeshTenant]; ok {
		t.Fatal("must not inject tenant when inject_iomesh_context=false")
	}
}

// s1267: per-server false overrides global true.
func TestBuildMCPServerConfig_PerServerOverrideOff(t *testing.T) {
	cfg := Default()
	cfg.MCP.InjectIOMeshContext = true
	cfg.IOMesh.Tenant = "acme"
	off := false
	s := MCPServerTOML{
		Name:                "scenario",
		URL:                 "https://mcp.example.com/mcp",
		InjectIOMeshContext: &off,
	}
	sc := cfg.BuildMCPServerConfig(s)
	if len(sc.Headers) != 0 {
		t.Fatalf("per-server false: %v", sc.Headers)
	}
}

// s1267: memory tenant fallback when [iomesh].tenant empty.
func TestBuildMCPServerConfig_MemoryTenantFallback(t *testing.T) {
	cfg := Default()
	cfg.MCP.InjectIOMeshContext = true
	cfg.Memory.Tenant = "dept.research"
	s := MCPServerTOML{Name: "m", URL: "http://127.0.0.1:8080/mcp"}
	sc := cfg.BuildMCPServerConfig(s)
	if sc.Headers[mcp.HeaderIOMeshTenant] != "dept.research" {
		t.Fatalf("headers=%v", sc.Headers)
	}
}

// s1267: empty context values not sent (no invent tenant).
func TestBuildMCPServerConfig_EmptyNotSent(t *testing.T) {
	cfg := Default()
	cfg.MCP.InjectIOMeshContext = true
	// no tenant/org/workspace configured
	s := MCPServerTOML{Name: "m", URL: "http://127.0.0.1:8080/mcp"}
	sc := cfg.BuildMCPServerConfig(s)
	if sc.Headers != nil && len(sc.Headers) > 0 {
		t.Fatalf("empty context must not invent headers: %v", sc.Headers)
	}
}
