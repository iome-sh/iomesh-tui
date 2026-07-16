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
