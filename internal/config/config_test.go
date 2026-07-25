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
