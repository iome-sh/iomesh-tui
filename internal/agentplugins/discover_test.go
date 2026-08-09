package agentplugins

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// moduleRoot walks up from this test file to the directory containing go.mod.
func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if st, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !st.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above test file")
		}
		dir = parent
	}
}

func writePlugin(t *testing.T, root, name string) {
	t.Helper()
	body := `{"$schema":"https://agent-plugins.org/schemas/1.0.0/plugin.schema.json","name":"` + name + `"}`
	if err := os.WriteFile(filepath.Join(root, "plugin.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscover_SkillsOnly(t *testing.T) {
	root := t.TempDir()
	writePlugin(t, root, "skills-only")
	// Immediate skill.
	sk := filepath.Join(root, "skills", "summarize")
	if err := os.MkdirAll(sk, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sk, "SKILL.md"), []byte("---\nname: summarize\n---\n# S\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Nested deeper skill must NOT be discovered.
	deep := filepath.Join(root, "skills", "summarize", "nested", "inner")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deep, "SKILL.md"), []byte("# nested"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Dir without SKILL.md ignored.
	if err := os.MkdirAll(filepath.Join(root, "skills", "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Second skill.
	sk2 := filepath.Join(root, "skills", "deploy")
	if err := os.MkdirAll(sk2, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sk2, "SKILL.md"), []byte("# deploy"), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if p.Manifest.Name != "skills-only" {
		t.Fatal(p.Manifest.Name)
	}
	if len(p.Skills) != 2 {
		t.Fatalf("skills=%d %+v", len(p.Skills), p.Skills)
	}
	// Sorted by name: deploy, summarize
	if p.Skills[0].Name != "deploy" || p.Skills[1].Name != "summarize" {
		t.Fatal(p.Skills)
	}
	if len(p.MCPServers) != 0 {
		t.Fatal("no mcp expected")
	}
	if !strings.Contains(p.Summary(), `plugin "skills-only"`) {
		t.Fatal(p.Summary())
	}
}

func TestDiscover_MissingSkillsAndMCP_OK(t *testing.T) {
	root := t.TempDir()
	writePlugin(t, root, "bare")
	p, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Skills) != 0 || len(p.MCPServers) != 0 {
		t.Fatalf("%+v", p)
	}
}

func TestDiscover_WithMCP(t *testing.T) {
	root := t.TempDir()
	writePlugin(t, root, "with-mcp")
	mcp := `{
		"$schema": "https://agent-plugins.org/schemas/1.0.0/mcp.schema.json",
		"mcpServers": {
			"npx-tool": {"type": "stdio", "command": "npx", "args": ["-y", "pkg"]},
			"remote": {"type": "streamable-http", "url": "https://mcp.example.com/mcp"}
		}
	}`
	if err := os.WriteFile(filepath.Join(root, "mcp.json"), []byte(mcp), 0o644); err != nil {
		t.Fatal(err)
	}
	// skill
	sk := filepath.Join(root, "skills", "help")
	_ = os.MkdirAll(sk, 0o755)
	_ = os.WriteFile(filepath.Join(sk, "SKILL.md"), []byte("# help"), 0o644)

	p, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Skills) != 1 || p.Skills[0].Name != "help" {
		t.Fatal(p.Skills)
	}
	if len(p.MCPServers) != 2 {
		t.Fatalf("mcp=%+v warnings=%v", p.MCPServers, p.Warnings)
	}
}

func TestDiscover_MCPFailOpen_SkillsStillLoad(t *testing.T) {
	root := t.TempDir()
	writePlugin(t, root, "mcp-bad")
	// Invalid mcp.json should disable MCP only.
	if err := os.WriteFile(filepath.Join(root, "mcp.json"), []byte(`not-json`), 0o644); err != nil {
		t.Fatal(err)
	}
	sk := filepath.Join(root, "skills", "ok")
	_ = os.MkdirAll(sk, 0o755)
	_ = os.WriteFile(filepath.Join(sk, "SKILL.md"), []byte("# ok"), 0o644)

	p, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Skills) != 1 {
		t.Fatal(p.Skills)
	}
	if len(p.MCPServers) != 0 {
		t.Fatal(p.MCPServers)
	}
	if len(p.Warnings) == 0 {
		t.Fatal("want mcp warning")
	}
}

func TestDiscover_FatalMissingManifest(t *testing.T) {
	if _, err := Discover(t.TempDir()); err == nil {
		t.Fatal("expected error")
	}
}

func TestDiscover_FatalBadName(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "plugin.json"), []byte(`{"name":"Bad Name"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Discover(root); err == nil {
		t.Fatal("expected bad name error")
	}
}

func TestDiscover_UnknownManifestKeyWarn(t *testing.T) {
	root := t.TempDir()
	body := `{"name":"warn-plugin","inline_skills":true}`
	if err := os.WriteFile(filepath.Join(root, "plugin.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range p.Warnings {
		if strings.Contains(w, "inline_skills") {
			found = true
		}
	}
	if !found {
		t.Fatalf("warnings=%v", p.Warnings)
	}
}

func TestDiscover_SkillsNotDir(t *testing.T) {
	root := t.TempDir()
	writePlugin(t, root, "skills-file")
	if err := os.WriteFile(filepath.Join(root, "skills"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Skills) != 0 {
		t.Fatal(p.Skills)
	}
	found := false
	for _, w := range p.Warnings {
		if strings.Contains(w, "not a directory") {
			found = true
		}
	}
	if !found {
		t.Fatalf("warnings=%v", p.Warnings)
	}
}

// TestDiscover_HelloIomeExample pins s1337 residual-honest sample package under
// examples/agent-plugins/hello-iome (skills-only · no invent GA · dual_write OFF).
func TestDiscover_HelloIomeExample(t *testing.T) {
	root := filepath.Join(moduleRoot(t), "examples", "agent-plugins", "hello-iome")
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		t.Fatalf("sample package missing at %s: %v", root, err)
	}

	p, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover(%s): %v", root, err)
	}
	if p.Manifest.Name != "hello-iome" {
		t.Fatalf("name=%q", p.Manifest.Name)
	}
	if p.Manifest.Schema != PluginSchemaID {
		t.Fatalf("schema=%q want %q", p.Manifest.Schema, PluginSchemaID)
	}
	if len(p.Skills) != 1 || p.Skills[0].Name != "hello-iome" {
		t.Fatalf("skills=%+v", p.Skills)
	}
	if len(p.MCPServers) != 0 {
		t.Fatalf("sample is skills-only; mcp=%+v", p.MCPServers)
	}
	// Residual honesty: Discover success is not install/Connected invent.
	if !strings.Contains(p.Summary(), `plugin "hello-iome"`) {
		t.Fatal(p.Summary())
	}
}

// TestDiscover_IomeshMemoryMCPExample pins s1478 product sample package under
// examples/agent-plugins/iomesh-memory-mcp (public product stdio map · not Memory GA ·
// dual_write OFF · package load ≠ Connected · binary on PATH required for connect).
func TestDiscover_IomeshMemoryMCPExample(t *testing.T) {
	root := filepath.Join(moduleRoot(t), "examples", "agent-plugins", "iomesh-memory-mcp")
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		t.Fatalf("sample package missing at %s: %v", root, err)
	}

	p, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover(%s): %v", root, err)
	}
	if p.Manifest.Name != "iomesh-memory-mcp" {
		t.Fatalf("name=%q", p.Manifest.Name)
	}
	if p.Manifest.Schema != PluginSchemaID {
		t.Fatalf("schema=%q want %q", p.Manifest.Schema, PluginSchemaID)
	}
	if len(p.Skills) != 1 || p.Skills[0].Name != "iomesh-memory-local" {
		t.Fatalf("skills=%+v", p.Skills)
	}
	if len(p.MCPServers) != 1 {
		t.Fatalf("want 1 mcp server; got %+v warnings=%v", p.MCPServers, p.Warnings)
	}
	srv := p.MCPServers[0]
	if srv.Name != "memory" {
		t.Fatalf("mcp name=%q", srv.Name)
	}
	if srv.Type != TransportStdio {
		t.Fatalf("mcp type=%q want %q", srv.Type, TransportStdio)
	}
	if srv.Command != "iomesh-memory-mcp" {
		t.Fatalf("mcp command=%q", srv.Command)
	}
	if len(srv.Env) != 0 || len(srv.Headers) != 0 {
		t.Fatalf("sample must not ship secrets env/headers: env=%v headers=%v", srv.Env, srv.Headers)
	}

	dataRoot := filepath.Join(t.TempDir(), "plugin-data")
	cfgs, warns := MCPServersFromPlugins([]*Plugin{p}, dataRoot)
	if len(warns) != 0 {
		t.Fatalf("map warnings=%v", warns)
	}
	if len(cfgs) != 1 {
		t.Fatalf("cfgs=%d", len(cfgs))
	}
	if cfgs[0].Name != "iomesh-memory-mcp-memory" {
		t.Fatalf("mapped name=%q", cfgs[0].Name)
	}
	if cfgs[0].Command != "iomesh-memory-mcp" {
		t.Fatalf("mapped command=%q", cfgs[0].Command)
	}
	if cfgs[0].Env["PLUGIN_ROOT"] == "" || cfgs[0].Env["PLUGIN_DATA"] == "" {
		t.Fatalf("PLUGIN_ROOT/DATA inject missing: %+v", cfgs[0].Env)
	}
	if !strings.Contains(p.Summary(), `plugin "iomesh-memory-mcp"`) {
		t.Fatal(p.Summary())
	}
}

// TestDiscover_AionMemoryMCPExample pins s1346 residual private sample package under
// examples/agent-plugins/aion-memory-mcp (stdio map · not product naming · not Memory GA ·
// dual_write OFF · package load ≠ Connected · binary on PATH required for connect).
func TestDiscover_AionMemoryMCPExample(t *testing.T) {
	root := filepath.Join(moduleRoot(t), "examples", "agent-plugins", "aion-memory-mcp")
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		t.Fatalf("sample package missing at %s: %v", root, err)
	}

	p, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover(%s): %v", root, err)
	}
	if p.Manifest.Name != "aion-memory-mcp" {
		t.Fatalf("name=%q", p.Manifest.Name)
	}
	if p.Manifest.Schema != PluginSchemaID {
		t.Fatalf("schema=%q want %q", p.Manifest.Schema, PluginSchemaID)
	}
	if len(p.Skills) != 1 || p.Skills[0].Name != "aion-memory-local" {
		t.Fatalf("skills=%+v", p.Skills)
	}
	if len(p.MCPServers) != 1 {
		t.Fatalf("want 1 mcp server; got %+v warnings=%v", p.MCPServers, p.Warnings)
	}
	srv := p.MCPServers[0]
	if srv.Name != "memory" {
		t.Fatalf("mcp name=%q", srv.Name)
	}
	if srv.Type != TransportStdio {
		t.Fatalf("mcp type=%q want %q", srv.Type, TransportStdio)
	}
	if srv.Command != "aion-memory-mcp" {
		t.Fatalf("mcp command=%q", srv.Command)
	}
	// No secrets in portable package fields (env/headers empty for this sample).
	if len(srv.Env) != 0 || len(srv.Headers) != 0 {
		t.Fatalf("sample must not ship secrets env/headers: env=%v headers=%v", srv.Env, srv.Headers)
	}

	// Map without process attach — success ≠ Connected / install green.
	dataRoot := filepath.Join(t.TempDir(), "plugin-data")
	cfgs, warns := MCPServersFromPlugins([]*Plugin{p}, dataRoot)
	if len(warns) != 0 {
		t.Fatalf("map warnings=%v", warns)
	}
	if len(cfgs) != 1 {
		t.Fatalf("cfgs=%d", len(cfgs))
	}
	if cfgs[0].Name != "aion-memory-mcp-memory" {
		t.Fatalf("mapped name=%q", cfgs[0].Name)
	}
	if cfgs[0].Command != "aion-memory-mcp" {
		t.Fatalf("mapped command=%q", cfgs[0].Command)
	}
	if cfgs[0].Env["PLUGIN_ROOT"] == "" || cfgs[0].Env["PLUGIN_DATA"] == "" {
		t.Fatalf("PLUGIN_ROOT/DATA inject missing: %+v", cfgs[0].Env)
	}
	// Residual honesty: Discover success is not install/Connected invent.
	if !strings.Contains(p.Summary(), `plugin "aion-memory-mcp"`) {
		t.Fatal(p.Summary())
	}
}
