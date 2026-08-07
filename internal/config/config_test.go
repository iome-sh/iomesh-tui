package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/router"
)

func TestDefault_DeepSeekPrimary(t *testing.T) {
	cfg := Default()
	if cfg.Models.Default != router.DefaultModelName {
		t.Fatalf("default=%q", cfg.Models.Default)
	}
	if len(cfg.Catalog) < 3 {
		t.Fatalf("catalog len=%d", len(cfg.Catalog))
	}
	if !cfg.Subagents.Enabled {
		t.Fatal("subagents default on")
	}
}

// TestDefault_DualWriteOff pins s768 local-primary honesty (+ s771 naming · s774 buyer-claim · s785 org-pulse peers):
// [memory].dual_write defaults OFF (optional mesh audit — not primary cloud palace).
// s771: "Memory Palace" / $119 = local MCP + Memory Ops Pack naming honesty, not hosted GPU.
// s774: MIT OSS TUI agent harness ≠ hosted multi-tenant mesh CP; local-primary buyer claim pin.
// s785: org-pulse edge framing — local agent on org pulse plane; dual_write still OFF (docs pin peer).
func TestDefault_DualWriteOff(t *testing.T) {
	// s768: dual_write default OFF (local-primary honesty); s771 naming + s774 buyer-claim + s785 org-pulse peers
	cfg := Default()
	if cfg.Memory.DualWrite {
		t.Fatalf("s768/s771/s774/s785 honesty: Memory.DualWrite must default false, got %+v", cfg.Memory)
	}
	if cfg.Memory.Enabled {
		t.Fatalf("memory enabled must default false, got %+v", cfg.Memory)
	}
}

// TestDefault_PluginsOff pins s1331 residual honesty: [plugins] opt-in only.
// package wire ≠ Agent Plugins GA · dual_write remains OFF.
func TestDefault_PluginsOff(t *testing.T) {
	cfg := Default()
	if cfg.Plugins.Enabled {
		t.Fatalf("s1331 honesty: Plugins.Enabled must default false, got %+v", cfg.Plugins)
	}
	if len(cfg.Plugins.Dirs) != 0 {
		t.Fatalf("plugins dirs must default empty, got %+v", cfg.Plugins.Dirs)
	}
	if cfg.Memory.DualWrite {
		t.Fatal("dual_write must stay OFF when plugins section is present")
	}
}

func TestLoad_PluginsSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[plugins]
enabled = true
dirs = ["/opt/plugins/a", "~/plugins/b"]
data_dir = "/var/plugin-data"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Plugins.Enabled {
		t.Fatal("plugins.enabled")
	}
	if len(cfg.Plugins.Dirs) != 2 || cfg.Plugins.Dirs[0] != "/opt/plugins/a" {
		t.Fatalf("dirs=%v", cfg.Plugins.Dirs)
	}
	if cfg.Plugins.DataDir != "/var/plugin-data" {
		t.Fatalf("data_dir=%q", cfg.Plugins.DataDir)
	}
}

func TestLoad_MergeModelOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[models]
default = "deepseek-v4-pro"

[model.deepseek-v4-flash]
temperature = 0.2
extra_headers = { "X-Team" = "platform" }

[router]
max_attempts = 5

[iomesh]
enabled = true
endpoint = "https://mesh.example.com"
tenant = "acme"
org = "org_dev-org"
workspace = "ws_alpha"

[subagents]
enabled = true
max_depth = 3
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Models.Default != "deepseek-v4-pro" {
		t.Fatalf("default=%q", cfg.Models.Default)
	}
	if cfg.Router.MaxAttempts != 5 {
		t.Fatalf("max_attempts=%d", cfg.Router.MaxAttempts)
	}
	if !cfg.IOMesh.Enabled || cfg.IOMesh.Tenant != "acme" {
		t.Fatalf("iomesh=%+v", cfg.IOMesh)
	}
	if cfg.IOMesh.Org != "org_dev-org" || cfg.IOMesh.Workspace != "ws_alpha" {
		t.Fatalf("iomesh org/workspace=%+v", cfg.IOMesh)
	}
	if cfg.Subagents.MaxDepth != 3 {
		t.Fatalf("max_depth=%d", cfg.Subagents.MaxDepth)
	}
	r, err := cfg.NewRouter()
	if err != nil {
		t.Fatal(err)
	}
	m, ok := r.Model("deepseek-v4-flash")
	if !ok {
		t.Fatal("missing flash")
	}
	if m.ExtraHeaders["X-Team"] != "platform" {
		t.Fatalf("headers=%v", m.ExtraHeaders)
	}
}

func TestLoad_MemorySection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[memory]
enabled = true
server = "palace"
tenant = "acme"
endpoint = "http://127.0.0.1:8765"
auto_recall = true
auto_ingest = true
dual_write = true
limit = 12
pull_role = "agent"
pull_allow_suffix = "ops,memory"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Memory.Enabled || cfg.Memory.Server != "palace" || cfg.Memory.Tenant != "acme" {
		t.Fatalf("memory=%+v", cfg.Memory)
	}
	if !cfg.Memory.AutoIngest || !cfg.Memory.DualWrite || cfg.Memory.Limit != 12 {
		t.Fatalf("memory flags=%+v", cfg.Memory)
	}
	if cfg.Memory.Endpoint != "http://127.0.0.1:8765" {
		t.Fatalf("endpoint=%q", cfg.Memory.Endpoint)
	}
	if cfg.Memory.PullRole != "agent" || cfg.Memory.PullAllowSuffix != "ops,memory" {
		t.Fatalf("pull role/suffix=%q %q", cfg.Memory.PullRole, cfg.Memory.PullAllowSuffix)
	}
}

func TestEnv_MemoryPullRoleAndSuffix(t *testing.T) {
	t.Setenv("IOMESH_MEMORY_PULL_ROLE", "custom")
	t.Setenv("IOMESH_MEMORY_PULL_ALLOW_SUFFIX", "a,b")
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Memory.PullRole != "custom" || cfg.Memory.PullAllowSuffix != "a,b" {
		t.Fatalf("got role=%q suffix=%q", cfg.Memory.PullRole, cfg.Memory.PullAllowSuffix)
	}
}

func TestLoad_MemoryEndpointEnv(t *testing.T) {
	t.Setenv("IOMESH_MEMORY_ENDPOINT", "http://sidecar:8765")
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Memory.Endpoint != "http://sidecar:8765" {
		t.Fatalf("got %q", cfg.Memory.Endpoint)
	}
	t.Setenv("IOMESH_MEMORY_ENDPOINT", "")
	t.Setenv("MEMORY_SIDECAR_URL", "http://legacy:9000")
	cfg2, err := Load(filepath.Join(t.TempDir(), "nope2.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg2.Memory.Endpoint != "http://legacy:9000" {
		t.Fatalf("sidecar env=%q", cfg2.Memory.Endpoint)
	}
}

func TestLoad_MissingFileUsesDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Models.Default != router.DefaultModelName {
		t.Fatal(cfg.Models.Default)
	}
}

func TestLoad_CustomModel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[models]
default = "local-llama"

[model.local-llama]
model = "llama-3"
base_url = "http://127.0.0.1:8080/v1"
env_key = "LOCAL_KEY"
context_window = 128000
cost_tier = 0.5
capabilities = ["fast", "coding"]
priority = 5
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	r, err := cfg.NewRouter()
	if err != nil {
		t.Fatal(err)
	}
	if r.DefaultModel() != "local-llama" {
		t.Fatal(r.DefaultModel())
	}
	m, ok := r.Model("local-llama")
	if !ok || m.ModelID != "llama-3" {
		t.Fatalf("%+v", m)
	}
}

func TestLoad_RejectsFileSchemeBaseURL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[models]
default = "evil"

[model.evil]
model = "x"
base_url = "file:///etc"
priority = 1
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cfg.NewRouter(); err == nil {
		t.Fatal("expected file:// rejection")
	}
}

func TestEnv_SubagentsOff(t *testing.T) {
	t.Setenv("IOMESH_SUBAGENTS", "0")
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Subagents.Enabled {
		t.Fatal("expected disabled via env")
	}
}

func TestEnv_IOMeshOrgWorkspace(t *testing.T) {
	t.Setenv("IOMESH_ORG", "org_from_env")
	t.Setenv("IOMESH_WORKSPACE", "ws_from_env")
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.IOMesh.Org != "org_from_env" || cfg.IOMesh.Workspace != "ws_from_env" {
		t.Fatalf("iomesh org/workspace=%+v", cfg.IOMesh)
	}
}

func TestEnv_MemoryOrgFallback(t *testing.T) {
	t.Setenv("IOMESH_ORG", "")
	t.Setenv("MEMORY_ORG", "org_memory")
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.IOMesh.Org != "org_memory" {
		t.Fatalf("expected MEMORY_ORG fallback, got %q", cfg.IOMesh.Org)
	}
}

func TestLoad_InvalidTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	if err := os.WriteFile(path, []byte("[[[not valid"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected parse error")
	}
}

// s1267: inject_iomesh_context config parse + WantsInject / IOMeshMCPContext helpers.
func TestLoad_MCPInjectIOMeshContext(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[mcp]
enabled = true
inject_iomesh_context = true

[[mcp.servers]]
name = "http-a"
url = "http://127.0.0.1:8080/mcp"
headers = { "X-IOMesh-Org" = "org_explicit" }

[[mcp.servers]]
name = "http-b"
url = "http://127.0.0.1:8081/mcp"
inject_iomesh_context = false

[[mcp.servers]]
name = "stdio"
command = "echo"
inject_iomesh_context = true

[iomesh]
tenant = "acme"
org = "org_from_iomesh"
workspace = "ws_alpha"

[memory]
tenant = "memory_only_fallback"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.MCP.InjectIOMeshContext {
		t.Fatalf("global inject default parse: %+v", cfg.MCP)
	}
	if len(cfg.MCP.Servers) != 3 {
		t.Fatalf("servers=%d", len(cfg.MCP.Servers))
	}
	// Server A inherits global true
	if !cfg.MCP.WantsInjectIOMeshContext(cfg.MCP.Servers[0]) {
		t.Fatal("server A should inherit global inject=true")
	}
	// Server B explicit false
	if cfg.MCP.WantsInjectIOMeshContext(cfg.MCP.Servers[1]) {
		t.Fatal("server B inject_iomesh_context=false must win")
	}
	// Server C explicit true
	if !cfg.MCP.WantsInjectIOMeshContext(cfg.MCP.Servers[2]) {
		t.Fatal("server C inject_iomesh_context=true")
	}
	if cfg.MCP.Servers[0].Headers["X-IOMesh-Org"] != "org_explicit" {
		t.Fatalf("headers parse: %v", cfg.MCP.Servers[0].Headers)
	}
	tenant, org, ws := cfg.IOMeshMCPContext()
	if tenant != "acme" || org != "org_from_iomesh" || ws != "ws_alpha" {
		t.Fatalf("IOMeshMCPContext=%q %q %q", tenant, org, ws)
	}
}

func TestDefault_MCPInjectIOMeshContextOff(t *testing.T) {
	cfg := Default()
	if cfg.MCP.InjectIOMeshContext {
		t.Fatal("s1267 honesty: inject_iomesh_context must default false")
	}
	s := MCPServerTOML{Name: "x"}
	if cfg.MCP.WantsInjectIOMeshContext(s) {
		t.Fatal("default wants inject false")
	}
}

func TestIOMeshMCPContext_MemoryTenantFallback(t *testing.T) {
	cfg := Default()
	cfg.Memory.Tenant = "dept.research"
	tenant, org, ws := cfg.IOMeshMCPContext()
	if tenant != "dept.research" {
		t.Fatalf("memory tenant fallback=%q", tenant)
	}
	if org != "" || ws != "" {
		t.Fatalf("empty org/ws expected, got %q %q", org, ws)
	}
	cfg.IOMesh.Tenant = "iomesh_wins"
	tenant, _, _ = cfg.IOMeshMCPContext()
	if tenant != "iomesh_wins" {
		t.Fatalf("iomesh tenant preferred=%q", tenant)
	}
}

func TestLoad_MCPInjectIOMeshContext_DefaultFalse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[mcp]
enabled = true
[[mcp.servers]]
name = "remote"
url = "https://mcp.example.com/mcp"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MCP.InjectIOMeshContext {
		t.Fatal("unset inject must parse as false")
	}
	if cfg.MCP.WantsInjectIOMeshContext(cfg.MCP.Servers[0]) {
		t.Fatal("default inject off for servers")
	}
}
