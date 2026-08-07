package agentplugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writePluginJSON(t *testing.T, root, name string) {
	t.Helper()
	body := `{"$schema":"https://agent-plugins.org/schemas/1.0.0/plugin.schema.json","name":"` + name + `"}`
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "plugin.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverAll_PackageRootsAndParent(t *testing.T) {
	base := t.TempDir()
	// Parent with two children.
	parent := filepath.Join(base, "plugins")
	a := filepath.Join(parent, "alpha")
	b := filepath.Join(parent, "beta")
	writePluginJSON(t, a, "alpha")
	writePluginJSON(t, b, "beta")
	// Direct package root.
	solo := filepath.Join(base, "solo-plugin")
	writePluginJSON(t, solo, "solo")
	// Bad dir (missing).
	missing := filepath.Join(base, "nope")

	plugins, warns := DiscoverAll([]string{parent, solo, missing, ""})
	if len(plugins) != 3 {
		t.Fatalf("plugins=%d warns=%v", len(plugins), warns)
	}
	// Sorted by name: alpha, beta, solo
	if plugins[0].Manifest.Name != "alpha" || plugins[1].Manifest.Name != "beta" || plugins[2].Manifest.Name != "solo" {
		t.Fatalf("order: %v %v %v", plugins[0].Manifest.Name, plugins[1].Manifest.Name, plugins[2].Manifest.Name)
	}
	// missing → warning
	foundMissing := false
	for _, w := range warns {
		if strings.Contains(w, "nope") {
			foundMissing = true
			break
		}
	}
	if !foundMissing {
		t.Fatalf("expected missing warning, got %v", warns)
	}
}

func TestDiscoverAll_FailOpenBadPlugin(t *testing.T) {
	base := t.TempDir()
	good := filepath.Join(base, "good")
	writePluginJSON(t, good, "good")
	bad := filepath.Join(base, "bad")
	if err := os.MkdirAll(bad, 0o755); err != nil {
		t.Fatal(err)
	}
	// Invalid name in plugin.json.
	if err := os.WriteFile(filepath.Join(bad, "plugin.json"), []byte(`{"name":"BAD_NAME"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	plugins, warns := DiscoverAll([]string{good, bad})
	if len(plugins) != 1 || plugins[0].Manifest.Name != "good" {
		t.Fatalf("plugins=%+v warns=%v", plugins, warns)
	}
	if len(warns) == 0 {
		t.Fatal("expected warning for bad plugin")
	}
}

func TestSkillDirs(t *testing.T) {
	root := t.TempDir()
	writePluginJSON(t, root, "sk")
	sk := filepath.Join(root, "skills", "summarize")
	if err := os.MkdirAll(sk, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sk, "SKILL.md"), []byte("# s"), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	dirs := SkillDirs([]*Plugin{p, nil, p})
	if len(dirs) != 1 {
		t.Fatalf("dirs=%v", dirs)
	}
	want := filepath.Join(root, "skills")
	// Discover resolves root via EvalSymlinks on macOS (/var → /private/var).
	if resolved, err := filepath.EvalSymlinks(want); err == nil {
		want = resolved
	}
	if dirs[0] != want {
		t.Fatalf("got %q want %q", dirs[0], want)
	}
	// No skills → empty.
	bareRoot := t.TempDir()
	writePluginJSON(t, bareRoot, "bare")
	bp, err := Discover(bareRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(SkillDirs([]*Plugin{bp})) != 0 {
		t.Fatal("expected empty skill dirs")
	}
}

func TestMCPServersFromPlugins_StdioAndHTTP(t *testing.T) {
	root := t.TempDir()
	writePluginJSON(t, root, "demo")
	// Bundled binary for ./ confinement.
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	mcpBody := `{
		"$schema": "https://agent-plugins.org/schemas/1.0.0/mcp.schema.json",
		"mcpServers": {
			"local": {
				"type": "stdio",
				"command": "./bin/server",
				"args": ["--cfg", "${PLUGIN_ROOT}/cfg.json"],
				"env": {"CACHE": "${PLUGIN_DATA}/cache"},
				"cwd": "${PLUGIN_ROOT}"
			},
			"remote": {
				"type": "streamable-http",
				"url": "https://mcp.example.com/mcp",
				"headers": {"X-Plugin": "${PLUGIN_ROOT}"}
			},
			"npx": {
				"type": "stdio",
				"command": "npx",
				"args": ["-y", "pkg"]
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(root, "mcp.json"), []byte(mcpBody), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.MCPServers) != 3 {
		t.Fatalf("servers=%d warns=%v", len(p.MCPServers), p.Warnings)
	}

	dataRoot := filepath.Join(t.TempDir(), "plugin-data")
	cfgs, warns := MCPServersFromPlugins([]*Plugin{p}, dataRoot)
	if len(warns) != 0 {
		t.Fatalf("warns=%v", warns)
	}
	if len(cfgs) != 3 {
		t.Fatalf("cfgs=%d", len(cfgs))
	}

	// Index by name.
	byName := map[string]int{}
	for i, c := range cfgs {
		byName[c.Name] = i
	}
	for _, n := range []string{"demo-local", "demo-remote", "demo-npx"} {
		if _, ok := byName[n]; !ok {
			t.Fatalf("missing %s in %+v", n, byName)
		}
	}

	local := cfgs[byName["demo-local"]]
	// Command resolved absolute under root.
	if !strings.HasSuffix(local.Command, filepath.Join("bin", "server")) && !strings.HasSuffix(local.Command, "bin/server") {
		t.Fatalf("command=%q", local.Command)
	}
	if strings.HasPrefix(local.Command, "./") {
		t.Fatalf("command still relative: %q", local.Command)
	}
	// Args expanded.
	if len(local.Args) != 2 || !strings.HasSuffix(local.Args[1], "cfg.json") {
		t.Fatalf("args=%v", local.Args)
	}
	// Env: PLUGIN_ROOT/DATA injected + CACHE expanded.
	if local.Env["PLUGIN_ROOT"] == "" || local.Env["PLUGIN_DATA"] == "" {
		t.Fatalf("env inject missing: %+v", local.Env)
	}
	if !strings.Contains(local.Env["CACHE"], "cache") {
		t.Fatalf("CACHE=%q", local.Env["CACHE"])
	}
	if local.Env["PLUGIN_DATA"] != filepath.Join(dataRoot, "demo") && !strings.HasSuffix(local.Env["PLUGIN_DATA"], filepath.Join("plugin-data", "demo")) {
		// Allow symlink resolution of data path.
		if !strings.Contains(local.Env["PLUGIN_DATA"], "demo") {
			t.Fatalf("PLUGIN_DATA=%q", local.Env["PLUGIN_DATA"])
		}
	}
	// PLUGIN_DATA dir created.
	if st, err := os.Stat(local.Env["PLUGIN_DATA"]); err != nil || !st.IsDir() {
		t.Fatalf("PLUGIN_DATA dir: %v", err)
	}
	// Mutating default nil (fail-closed true at runtime).
	if local.Mutating != nil {
		t.Fatalf("mutating should be nil default, got %v", *local.Mutating)
	}
	// Cwd expanded to plugin root.
	if local.Cwd != p.Root {
		t.Fatalf("cwd=%q root=%q", local.Cwd, p.Root)
	}

	remote := cfgs[byName["demo-remote"]]
	if remote.URL != "https://mcp.example.com/mcp" {
		t.Fatalf("url=%q", remote.URL)
	}
	if remote.Headers["X-Plugin"] != p.Root {
		t.Fatalf("header expand: %q want %q", remote.Headers["X-Plugin"], p.Root)
	}
	if remote.Command != "" {
		t.Fatal("http should not set command")
	}

	npx := cfgs[byName["demo-npx"]]
	if npx.Command != "npx" {
		t.Fatalf("npx command=%q", npx.Command)
	}
	if npx.Env["PLUGIN_ROOT"] != p.Root {
		t.Fatalf("npx PLUGIN_ROOT=%q", npx.Env["PLUGIN_ROOT"])
	}
}

func TestMCPServersFromPlugins_FailOpenEmptyAndNil(t *testing.T) {
	cfgs, warns := MCPServersFromPlugins(nil, "")
	if len(cfgs) != 0 || len(warns) != 0 {
		t.Fatalf("%v %v", cfgs, warns)
	}
	// Plugin with no MCP.
	root := t.TempDir()
	writePluginJSON(t, root, "empty")
	p, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	cfgs, warns = MCPServersFromPlugins([]*Plugin{p}, t.TempDir())
	if len(cfgs) != 0 || len(warns) != 0 {
		t.Fatalf("%v %v", cfgs, warns)
	}
}

func TestDefaultPluginDataRoot(t *testing.T) {
	got := DefaultPluginDataRoot("/ws", "")
	if got != filepath.Join("/ws", ".iomesh", "plugin-data") {
		t.Fatal(got)
	}
	got = DefaultPluginDataRoot("/ws", "/custom/data")
	if got != "/custom/data" {
		t.Fatal(got)
	}
}

func TestPluginServerName(t *testing.T) {
	if pluginServerName("demo", "local") != "demo-local" {
		t.Fatal(pluginServerName("demo", "local"))
	}
	if pluginServerName("", "x") != "x" {
		t.Fatal("empty plugin")
	}
}
