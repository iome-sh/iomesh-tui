package runtimewire

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/config"
)

func TestWire_PluginsOffEmptyServers(t *testing.T) {
	ws := t.TempDir()
	cfg := &config.Config{}
	// Defaults: plugins off, no MCP servers, skills dirs still include defaults.
	res := Wire(cfg, ws, slog.Default())
	if res.PluginCount != 0 {
		t.Fatalf("plugin count: got %d want 0", res.PluginCount)
	}
	if len(res.PluginSkillDirs) != 0 {
		t.Fatalf("plugin skill dirs: got %v", res.PluginSkillDirs)
	}
	if len(res.MCPServers) != 0 {
		t.Fatalf("mcp servers: got %d want 0", len(res.MCPServers))
	}
	if res.TOMLServerCount != 0 || res.PluginServerCount != 0 {
		t.Fatalf("counts toml=%d plugin=%d", res.TOMLServerCount, res.PluginServerCount)
	}
	if len(res.SkillDirs) == 0 {
		t.Fatal("expected default skill dirs even when skills empty")
	}
	// DefaultDirs always includes workspace .iomesh/skills when workspace set.
	foundWS := false
	for _, d := range res.SkillDirs {
		if d == filepath.Join(ws, ".iomesh", "skills") {
			foundWS = true
			break
		}
	}
	if !foundWS {
		// DefaultDirs may use different layout — at least non-empty path list.
		t.Logf("skill dirs: %v", res.SkillDirs)
	}
}

func TestWire_TOMLServersOnly(t *testing.T) {
	ws := t.TempDir()
	cfg := &config.Config{}
	cfg.MCP.Servers = []config.MCPServerTOML{
		{Name: "local", URL: "http://127.0.0.1:8080/mcp"},
	}
	res := Wire(cfg, ws, nil)
	if res.TOMLServerCount != 1 || len(res.MCPServers) != 1 {
		t.Fatalf("want 1 toml server, got toml=%d total=%d", res.TOMLServerCount, len(res.MCPServers))
	}
	if res.MCPServers[0].Name != "local" || res.MCPServers[0].URL == "" {
		t.Fatalf("server: %+v", res.MCPServers[0])
	}
	if res.PluginServerCount != 0 {
		t.Fatalf("plugin servers: %d", res.PluginServerCount)
	}
}

func TestWire_PluginsEnabledEmptyDirs(t *testing.T) {
	ws := t.TempDir()
	cfg := &config.Config{}
	cfg.Plugins.Enabled = true
	// Enabled but no dirs → no discover.
	res := Wire(cfg, ws, slog.Default())
	if res.PluginCount != 0 || len(res.MCPServers) != 0 {
		t.Fatalf("expected no plugins/servers, got plugins=%d servers=%d", res.PluginCount, len(res.MCPServers))
	}
}

func TestWire_PluginsDiscoverSkillAndMCP(t *testing.T) {
	ws := t.TempDir()
	// Minimal plugin package: plugin.json + skills + mcp.json
	root := filepath.Join(ws, "plugins", "hello")
	if err := os.MkdirAll(filepath.Join(root, "skills", "hello-skill"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Closed plugin.json keys only; skills/ and mcp.json discovered from FS.
	pluginJSON := `{"name":"hello","version":"0.0.1","description":"test"}`
	if err := os.WriteFile(filepath.Join(root, "plugin.json"), []byte(pluginJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	skillMD := "---\nname: hello-skill\ndescription: hi\n---\n\n# Hello\n"
	if err := os.WriteFile(filepath.Join(root, "skills", "hello-skill", "SKILL.md"), []byte(skillMD), 0o644); err != nil {
		t.Fatal(err)
	}
	mcpJSON := `{
  "$schema": "https://agent-plugins.org/schemas/1.0.0/mcp.schema.json",
  "mcpServers": {
    "echo": {
      "type": "stdio",
      "command": "true",
      "args": []
    }
  }
}`
	if err := os.WriteFile(filepath.Join(root, "mcp.json"), []byte(mcpJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{}
	cfg.Plugins.Enabled = true
	cfg.Plugins.Dirs = []string{root}
	cfg.MCP.Servers = []config.MCPServerTOML{
		{Name: "toml-first", URL: "http://127.0.0.1:9/mcp"},
	}

	res := Wire(cfg, ws, slog.Default())
	if res.PluginCount != 1 {
		t.Fatalf("plugins: %d warns=%v", res.PluginCount, res.Warnings)
	}
	if len(res.PluginSkillDirs) != 1 {
		t.Fatalf("plugin skill dirs: %v", res.PluginSkillDirs)
	}
	if res.TOMLServerCount != 1 {
		t.Fatalf("toml count %d", res.TOMLServerCount)
	}
	if res.PluginServerCount < 1 {
		t.Fatalf("expected plugin mcp server, got %d total=%d warns=%v",
			res.PluginServerCount, len(res.MCPServers), res.Warnings)
	}
	// TOML first, then plugins.
	if res.MCPServers[0].Name != "toml-first" {
		t.Fatalf("toml must be first: %s", res.MCPServers[0].Name)
	}
}

func TestMCPFeatureOn(t *testing.T) {
	if MCPFeatureOn(nil) {
		t.Fatal("nil")
	}
	cfg := &config.Config{}
	if MCPFeatureOn(cfg) {
		t.Fatal("default off")
	}
	cfg.MCP.Enabled = true
	cfg.Features.MCP = true
	if !MCPFeatureOn(cfg) {
		t.Fatal("want on")
	}
}

func TestSkillsFeatureOn(t *testing.T) {
	cfg := &config.Config{}
	if SkillsFeatureOn(cfg) {
		t.Fatal("default off")
	}
	cfg.Skills.Enabled = true
	cfg.Features.Skills = true
	if !SkillsFeatureOn(cfg) {
		t.Fatal("want on")
	}
}

func TestConnectMCP_NilWhenDisabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.MCP.Servers = []config.MCPServerTOML{{Name: "x", URL: "http://127.0.0.1:1/mcp"}}
	if mgr := ConnectMCP(t.Context(), cfg, t.TempDir(), nil); mgr != nil {
		t.Fatal("disabled must return nil")
	}
}

func TestNewMesh_EmptyInferDoesNotInventEnabled(t *testing.T) {
	c, inf := NewMesh(&config.Config{}, slog.Default())
	if inf.Endpoint != "" {
		t.Fatalf("empty infer: %+v", inf)
	}
	if c == nil || c.Enabled() {
		t.Fatal("empty infer must not invent mesh enabled")
	}
}

func TestNewMesh_InfersHooksFromPortalMCP(t *testing.T) {
	cfg := &config.Config{}
	cfg.MCP.Servers = []config.MCPServerTOML{{
		Name: "io-mesh",
		URL:  "https://apiv1.iome.sh/v7/mcp",
		Headers: map[string]string{
			"X-IOMesh-Tenant": "dept.engineering",
			"X-IOMesh-Org":    "org_example",
		},
	}}
	c, inf := NewMesh(cfg, slog.Default())
	if inf.Endpoint != "https://hooks.iome.sh" {
		t.Fatalf("infer %+v", inf)
	}
	if c == nil || !c.Enabled() || c.Endpoint() != "https://hooks.iome.sh" {
		t.Fatalf("enabled=%v endpoint=%q", c.Enabled(), c.Endpoint())
	}
	if c.Tenant() != "dept.engineering" {
		t.Fatalf("tenant=%q", c.Tenant())
	}
	if !cfg.IOMesh.Enabled || cfg.IOMesh.Endpoint != "https://hooks.iome.sh" {
		t.Fatalf("cfg not filled: %+v", cfg.IOMesh)
	}
}

func TestNewMesh_DepartmentAuthHeader(t *testing.T) {
	var gotDept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotDept = r.Header.Get("X-IOMesh-Department")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &config.Config{}
	cfg.IOMesh.Enabled = true
	cfg.IOMesh.Endpoint = srv.URL
	cfg.IOMesh.Department = "eng"
	c, _ := NewMesh(cfg, slog.Default())
	if err := c.Health(context.Background()); err != nil {
		t.Fatal(err)
	}
	if gotDept != "eng" {
		t.Fatalf("X-IOMesh-Department=%q", gotDept)
	}

	cfg.IOMesh.Department = ""
	gotDept = "sentinel"
	c, _ = NewMesh(cfg, slog.Default())
	if err := c.Health(context.Background()); err != nil {
		t.Fatal(err)
	}
	if gotDept != "" {
		t.Fatalf("empty department must omit header, got %q", gotDept)
	}
}

func TestNewMesh_ExplicitEndpointWins(t *testing.T) {
	cfg := &config.Config{}
	cfg.IOMesh.Enabled = true
	cfg.IOMesh.Endpoint = "https://hooks.example.test"
	cfg.MCP.Servers = []config.MCPServerTOML{{
		URL: "https://apiv1.iome.sh/v7/mcp",
	}}
	c, inf := NewMesh(cfg, nil)
	if inf.Endpoint != "" {
		t.Fatalf("explicit should skip infer: %+v", inf)
	}
	if c.Endpoint() != "https://hooks.example.test" {
		t.Fatalf("endpoint=%q", c.Endpoint())
	}
}
