package setup

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/config"
)

func TestPreflight_NoConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.toml")
	rep, err := Preflight(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if rep.OK {
		t.Fatal("ok should be false without config")
	}
	if rep.State != "not_started" {
		t.Fatalf("state=%s", rep.State)
	}
	if strings.Contains(strings.ToLower(rep.Honesty), "connected") && strings.Contains(rep.Honesty, "invent Connected") {
		// honesty must mention never invent Connected
	}
	if !strings.Contains(rep.Honesty, "Connected") {
		t.Fatalf("honesty missing Connected needle: %s", rep.Honesty)
	}
}

func TestPreflight_MemoryHealthz(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","dual_write":"off","not_memory_ga":true,"embeddings":"hash","qdrant":"off"}`))
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	frag := `
[mcp]
enabled = true
[[mcp.servers]]
name = "iomesh-memory-mcp"
url = "` + srv.URL + `/mcp"
allow_loopback = true
mutating = true
[memory]
enabled = true
server = "iomesh-memory-mcp"
dual_write = false
`
	if err := config.WriteSetupManagedFragment(path, frag); err != nil {
		t.Fatal(err)
	}
	// WriteSetupManagedFragment only writes managed block - need valid file for Load
	// Load can parse managed content if it's valid TOML - but markers are comments so OK
	// Actually full file is markers + toml - Load unmarshals whole file; comments ignored.
	rep, err := Preflight(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.MemoryHealthOK {
		t.Fatalf("health: %+v err=%s", rep, rep.MemoryHealthErr)
	}
	if !rep.OK {
		t.Fatalf("ok false: %+v notes=%v", rep, rep.Notes)
	}
	if rep.State != "local_memory_probe_ok" {
		t.Fatalf("state=%s", rep.State)
	}
	if rep.DualWrite {
		t.Fatal("dual_write must be false")
	}
}

func TestPreflight_MeshOrgEmptyNote(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	opt := DefaultInitOptions()
	opt.MeshEndpoint = "https://hooks.iome.sh"
	opt.MeshTenant = "dept.engineering"
	frag, err := BuildManagedFragment([]Profile{ProfileMesh}, opt)
	if err != nil {
		t.Fatal(err)
	}
	if err := config.WriteSetupManagedFragment(path, frag); err != nil {
		t.Fatal(err)
	}
	rep, err := Preflight(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if rep.MeshOrg != "" {
		t.Fatalf("org must be empty honest, got %q", rep.MeshOrg)
	}
	text := FormatPreflightText(rep)
	jsonOut := FormatPreflightJSON(rep)
	if !strings.Contains(jsonOut, `"org": ""`) && !strings.Contains(jsonOut, `"org":""`) {
		t.Fatalf("preflight JSON must always-emit empty org:\n%s", jsonOut)
	}
	note := false
	for _, n := range rep.Notes {
		if strings.Contains(n, "org empty") && strings.Contains(n, "fail-open") {
			note = true
		}
	}
	if !note {
		t.Fatalf("want N=2 isolation note when mesh-on + org-empty: %+v", rep.Notes)
	}
	if !strings.Contains(text, `org=""`) && !strings.Contains(text, `org="`) {
		t.Fatalf("text must surface org residual:\n%s", text)
	}
	if strings.Contains(text, "Connected: yes") {
		t.Fatal(text)
	}
}

func TestPreflight_MeshOrgSetNoFailOpenNote(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	opt := DefaultInitOptions()
	opt.MeshEndpoint = "https://hooks.iome.sh"
	opt.MeshOrg = "org_a"
	frag, err := BuildManagedFragment([]Profile{ProfileMesh}, opt)
	if err != nil {
		t.Fatal(err)
	}
	if err := config.WriteSetupManagedFragment(path, frag); err != nil {
		t.Fatal(err)
	}
	rep, err := Preflight(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if rep.MeshOrg != "org_a" {
		t.Fatalf("org=%q", rep.MeshOrg)
	}
	for _, n := range rep.Notes {
		if strings.Contains(n, "org empty") {
			t.Fatalf("must not warn fail-open when org set: %s", n)
		}
	}
	if !strings.Contains(FormatPreflightJSON(rep), `"org": "org_a"`) && !strings.Contains(FormatPreflightJSON(rep), `"org":"org_a"`) {
		t.Fatalf("json org:\n%s", FormatPreflightJSON(rep))
	}
}

func TestWriteInitAndPreflight_LocalMemoryOffline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	opt := DefaultInitOptions()
	opt.MemoryHTTPURL = "http://127.0.0.1:1/mcp" // nothing listening
	frag, err := BuildManagedFragment([]Profile{ProfileLocalMemory}, opt)
	if err != nil {
		t.Fatal(err)
	}
	if err := config.WriteSetupManagedFragment(path, frag); err != nil {
		t.Fatal(err)
	}
	rep, err := Preflight(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if rep.OK {
		t.Fatal("expected ok false when healthz down")
	}
	if !rep.ConfigPresent {
		t.Fatal("config should be present")
	}
	// ensure no invent Connected as success
	text := FormatPreflightText(rep)
	if strings.Contains(text, "Connected: yes") {
		t.Fatal(text)
	}
	_ = os.Remove(path)
}

// s1699: FormatPreflightText appends dual-path next-step after report body.
func TestFormatPreflightText_DualPathNextStep(t *testing.T) {
	// Minimal nil-safe path uses empty report; still appends next-step block.
	text := FormatPreflightText(&PreflightReport{
		State:   "not_started",
		OK:      false,
		Honesty: "dual_write OFF · not Memory GA · catalog ≠ Connected",
		Notes:   []string{},
	})
	for _, want := range []string{
		"/setup reload",
		"restart",
		"CLI has no",
		"package wire",
		"dual_write OFF",
		"not Memory GA",
		"s1699",
		"preflight",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("FormatPreflightText missing dual-path needle %q in:\n%s", want, text)
		}
	}
	if strings.Contains(text, "Connected: yes") {
		t.Fatalf("must not invent Connected green:\n%s", text)
	}
	if strings.Contains(text, "dual_write ON") || strings.Contains(text, "Memory GA shipped") {
		t.Fatalf("must not invent dual_write ON / Memory GA shipped:\n%s", text)
	}
}
