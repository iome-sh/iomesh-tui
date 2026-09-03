package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/config"
)

func TestCmdMemory_IngestDirUsage(t *testing.T) {
	if code := cmdMemory([]string{}); code != 2 {
		t.Fatalf("empty exit=%d want 2", code)
	}
	if code := cmdMemory([]string{"ingest-dir"}); code != 2 {
		t.Fatalf("ingest-dir missing path exit=%d want 2", code)
	}
	if code := cmdMemoryIngestDir([]string{"notes"}); code != 2 {
		t.Fatalf("ingest-dir without --yes/--dry-run exit=%d want 2", code)
	}
	if code := cmdMemory([]string{"ingest"}); code != 2 {
		t.Fatalf("ingest missing text exit=%d want 2", code)
	}
	if code := cmdMemoryIngest([]string{"hello overlay"}); code != 2 {
		t.Fatalf("ingest without --yes exit=%d want 2", code)
	}
}

func TestCmdMemoryHelp_IngestDir(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	code := cmdMemory([]string{"help"})
	_ = w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	if code != 0 {
		t.Fatalf("help exit=%d", code)
	}
	got := buf.String()
	for _, want := range []string{"ingest-dir", "local-overlay", "dual_write", "Catalog list ≠ consume"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help missing %q:\n%s", want, got)
		}
	}
}

func TestCmdMemoryIngestDir_DryRun(t *testing.T) {
	root := t.TempDir()
	overlay := root + "/overlay"
	if err := os.MkdirAll(overlay, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(overlay+"/note.md", []byte("alpha needle"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := cmdMemoryIngestDir([]string{"-C", root, "--dry-run", "overlay"}); code != 0 {
		t.Fatalf("dry-run exit=%d", code)
	}
}

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

func TestApplyOrgFlag(t *testing.T) {
	applyOrgFlag(nil, "org_a")
	cfg := &config.Config{}
	cfg.IOMesh.Org = "org_keep"
	applyOrgFlag(cfg, "  ")
	if cfg.IOMesh.Org != "org_keep" {
		t.Fatalf("empty flag must keep config org: %q", cfg.IOMesh.Org)
	}
	applyOrgFlag(cfg, " org_a ")
	if cfg.IOMesh.Org != "org_a" {
		t.Fatalf("flag must overlay org: %q", cfg.IOMesh.Org)
	}
}

func TestCmdSetupInit_MeshPrintOnlyOrgResidual(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	code := cmdSetupInit([]string{"mesh", "--print-only", "--mesh-endpoint", "https://hooks.iome.sh"})
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	if code != 0 {
		t.Fatalf("print-only exit=%d", code)
	}
	got := buf.String()
	for _, want := range []string{
		"[iomesh]",
		"# org =",
		"IOMESH_ORG",
		"fail-open",
		"aion #2721",
		`api_key_env = "IOMESH_TOKEN"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("setup init mesh --print-only missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\norg = ") {
		t.Fatalf("empty org must not persist a live field:\n%s", got)
	}
}

func TestCmdSetupInit_MeshPrintOnlyPersistsOrg(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	code := cmdSetupInit([]string{"mesh", "--print-only", "--mesh-endpoint", "https://hooks.iome.sh", "--mesh-org", "org_a"})
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	if code != 0 {
		t.Fatalf("print-only exit=%d", code)
	}
	got := buf.String()
	if !strings.Contains(got, `org = "org_a"`) {
		t.Fatalf("want persisted org:\n%s", got)
	}
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
