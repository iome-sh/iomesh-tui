package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/config"
)

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	fn()
	_ = w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func usagePrimaryBlock(got string) string {
	i := strings.Index(got, "Usage:")
	j := strings.Index(got, "\nAdvanced")
	if i < 0 {
		return got
	}
	if j < 0 || j <= i {
		return got[i:]
	}
	return got[i:j]
}

func TestPrintUsage_TTFHPrimaryAndAdvanced(t *testing.T) {
	got := captureStderr(t, printUsage)
	primary := usagePrimaryBlock(got)
	for _, want := range []string{
		"iomesh [flags]",
		`iomesh --repl`,
		`iomesh -p "prompt"`,
		"iomesh setup init|preflight",
		"iomesh memory ingest",
		"iomesh ttfh [--unit]",
		"iomesh mesh smoke",
		"iomesh models | sessions | mcp | version",
	} {
		if !strings.Contains(primary, want) {
			t.Fatalf("primary Usage missing %q:\n%s", want, primary)
		}
	}
	for _, hide := range []string{
		"iomesh memory pull",
		"iomesh plugins",
		"iomesh agent stdio",
		"iomesh agent serve",
		"iomesh mesh pub",
		"iomesh mesh consumer",
		"iomesh mesh wait",
		"iomesh mesh status",
		"iomesh mesh usage",
		"iomesh memory ingest-dir",
		"iomesh skills",
	} {
		if strings.Contains(primary, hide) {
			t.Fatalf("primary Usage must not advertise %q:\n%s", hide, primary)
		}
	}
	adv := got
	if i := strings.Index(got, "\nAdvanced"); i >= 0 {
		adv = got[i:]
		if j := strings.Index(adv, "\nFlags:"); j >= 0 {
			adv = adv[:j]
		}
	}
	if !strings.Contains(adv, "Advanced") {
		t.Fatalf("printUsage missing Advanced section:\n%s", got)
	}
	for _, want := range []string{
		"iomesh mesh pub",
		"iomesh mesh consumer create|delete|ack|nack",
		"iomesh mesh wait",
		"iomesh mesh status",
		"iomesh mesh usage",
		"iomesh memory pull",
		"iomesh memory ingest-dir",
		"iomesh plugins list|validate|smoke",
		"iomesh agent stdio|serve",
		"iomesh skills",
	} {
		if !strings.Contains(adv, want) {
			t.Fatalf("Advanced missing %q:\n%s", want, adv)
		}
	}
	if strings.Contains(got, "/gtm") || strings.Contains(got, "/plugins") {
		t.Fatalf("printUsage must keep /gtm /plugins slash hidden:\n%s", got)
	}
}

func TestReadmeCLIFence_TTFHPrimary(t *testing.T) {
	b, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	readme := string(b)
	start := strings.Index(readme, "## CLI")
	if start < 0 {
		t.Fatal("README missing ## CLI")
	}
	rest := readme[start:]
	end := strings.Index(rest[4:], "\n## ")
	if end < 0 {
		t.Fatal("README CLI section has no following heading")
	}
	cli := rest[:end+4]
	fenceStart := strings.Index(cli, "```text")
	fenceEnd := strings.Index(cli, "```\n")
	if fenceStart < 0 || fenceEnd <= fenceStart {
		t.Fatalf("README ## CLI missing text fence:\n%s", cli)
	}
	// Fence body is between ```text and the closing ``` that follows it.
	body := cli[fenceStart+len("```text") : fenceEnd]
	primary := body
	if i := strings.Index(body, "\nAdvanced:"); i >= 0 {
		primary = body[:i]
	}
	for _, want := range []string{
		"iomesh [flags]",
		"iomesh --repl",
		`iomesh -p "prompt"`,
		"iomesh setup init|preflight",
		"iomesh memory ingest",
		"iomesh ttfh",
		"iomesh mesh smoke",
		"iomesh models | sessions | mcp | version",
	} {
		if !strings.Contains(primary, want) {
			t.Fatalf("README CLI primary missing %q:\n%s", want, primary)
		}
	}
	if !strings.Contains(body, "Advanced: mesh consumer/pub · memory pull · plugins · agent serve") {
		t.Fatalf("README CLI missing Advanced one-liner:\n%s", body)
	}
	for _, hide := range []string{
		"iomesh memory pull",
		"iomesh plugins",
		"iomesh agent stdio",
		"iomesh agent serve",
		"iomesh skills",
	} {
		if strings.Contains(primary, hide) {
			t.Fatalf("README CLI primary must not advertise %q:\n%s", hide, primary)
		}
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestCmdTTFH_Unit(t *testing.T) {
	t.Setenv("IOMESH_ENDPOINT", "")
	t.Setenv("IOMESH_MEMORY_DUAL_WRITE", "")
	t.Setenv("IOMESH_CONFIG", t.TempDir()+"/missing.toml")

	var code int
	got := captureStdout(t, func() {
		code = cmdTTFH([]string{"--unit"})
	})
	if code != 0 {
		t.Fatalf("cmdTTFH --unit exit=%d want 0", code)
	}
	if !strings.Contains(got, "time-to-first-heartbeat") && !strings.Contains(strings.ToLower(got), "ttfh") {
		t.Fatalf("missing ttfh / time-to-first-heartbeat:\n%s", got)
	}
	for _, want := range []string{
		"dual_write OFF",
		"catalog ≠ Connected",
		"EMPTY",
		"knowledge Beta empty",
		"/memory digest --require-sources mesh,private",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("cmdTTFH --unit missing %q:\n%s", want, got)
		}
	}
	for _, bad := range []string{
		"Connected: yes",
		"dual_write ON",
		"Memory GA shipped",
		"analysis  ops 0",
	} {
		if strings.Contains(got, bad) {
			t.Fatalf("cmdTTFH --unit must not invent %q:\n%s", bad, got)
		}
	}
	if strings.Contains(got, "optional: iomesh mesh smoke") {
		t.Fatalf("--unit must not print live mesh hint:\n%s", got)
	}
}

func TestCmdTTFH_EndpointHintNoDial(t *testing.T) {
	t.Setenv("IOMESH_ENDPOINT", "https://hooks.iome.sh")
	t.Setenv("IOMESH_MEMORY_DUAL_WRITE", "")
	t.Setenv("IOMESH_CONFIG", t.TempDir()+"/missing.toml")

	var code int
	got := captureStdout(t, func() {
		code = cmdTTFH(nil)
	})
	if code != 0 {
		t.Fatalf("cmdTTFH exit=%d want 0", code)
	}
	want := "optional: iomesh mesh smoke (fail-open · never invent Connected · PULSE only after ≥1 decoded broker message)"
	if !strings.Contains(got, want) {
		t.Fatalf("endpoint without --unit missing mesh hint:\n%s", got)
	}
	if strings.Contains(got, "Connected: yes") || strings.Contains(got, "dual_write ON") {
		t.Fatalf("must not invent Connected / dual_write ON:\n%s", got)
	}
}

func TestCmdTTFH_BadFlags(t *testing.T) {
	if code := cmdTTFH([]string{"--nope"}); code != 2 {
		t.Fatalf("unknown flag exit=%d want 2", code)
	}
	if code := cmdTTFH([]string{"live"}); code != 2 {
		t.Fatalf("unknown arg exit=%d want 2", code)
	}
}

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
		"broker empty-org fail-open",
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
