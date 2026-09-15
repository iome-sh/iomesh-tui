package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/iome-sh/iomesh-tui/internal/iomesh"
	"github.com/iome-sh/iomesh-tui/internal/mcp"
	"github.com/iome-sh/iomesh-tui/internal/workspace"
)

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

func TestResolveMemoryIngestSessionID(t *testing.T) {
	if got := ResolveMemoryIngestSessionID("", ""); got != LocalOverlaySessionID {
		t.Fatalf("mint=%q", got)
	}
	if LocalOverlaySessionID != "local-overlay" {
		t.Fatalf("stable needle local-overlay=%q", LocalOverlaySessionID)
	}
	if got := ResolveMemoryIngestSessionID("cfg", "rt"); got != "cfg" {
		t.Fatalf("configured wins: %q", got)
	}
	if got := ResolveMemoryIngestSessionID("", "rt-sess"); got != "rt-sess" {
		t.Fatalf("runtime: %q", got)
	}
}

func TestMemoryIngestTurn_MintsLocalOverlaySessionID(t *testing.T) {
	var gotArgs map[string]any
	cInR, cInW := io.Pipe()
	cOutR, cOutW := io.Pipe()
	go mockMCPIngestTurn(cOutW, cInR, &gotArgs)

	mut := true
	cl := mcp.NewClientForTest(mcp.ServerConfig{Name: "memory", Command: "x", Mutating: &mut}, cInW, cOutR, nil)
	defer cl.Close()
	if err := cl.InitForTest(context.Background()); err != nil {
		t.Fatal(err)
	}
	mgr := mcp.NewManagerEmpty(nil)
	mgr.Attach(cl)

	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, Server: "memory", Tenant: "default", DualWrite: false, PalaceRoot: t.TempDir()},
		mcp:    mgr,
	}
	out, err := rt.MemoryIngestTurn(context.Background(), "user", "Demo note: overlay needle alpha")
	if err != nil {
		t.Fatalf("err=%v out=%q", err, out)
	}
	if gotArgs["session_id"] != LocalOverlaySessionID {
		t.Fatalf("session_id=%v args=%v", gotArgs["session_id"], gotArgs)
	}
	if gotArgs["content"] != "Demo note: overlay needle alpha" {
		t.Fatalf("content=%v", gotArgs["content"])
	}
	if hint, ok := gotArgs["source_hint"]; ok {
		t.Fatalf("local /memory ingest must not invent mesh source_hint; got %v", hint)
	}
	if !strings.Contains(out, "session_id=local-overlay") || !strings.Contains(out, "minted") {
		t.Fatalf("out=%q", out)
	}
	if !strings.Contains(out, "dual_write=false") {
		t.Fatalf("dual_write pin missing: %q", out)
	}
	if !strings.Contains(out, "palace:") || !strings.Contains(out, "ls this path") {
		t.Fatalf("ingest must print palace path: %q", out)
	}
	if strings.Contains(out, "Memory GA") && !strings.Contains(out, "not") {
		t.Fatalf("must not stamp Memory GA: %q", out)
	}
}

func TestListIngestDirFiles_TextAndSkipBinary(t *testing.T) {
	root := t.TempDir()
	overlay := filepath.Join(root, "overlay")
	if err := os.MkdirAll(filepath.Join(overlay, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(overlay, "alpha.md"), []byte("Project alpha ships Friday"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(overlay, "nested", "beta.txt"), []byte("owner is Alice"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(overlay, "blob.bin"), []byte{0x00, 0x01, 0x02}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(overlay, "empty.txt"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := ListIngestDirFiles(ws, "overlay", 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Files) != 2 {
		t.Fatalf("files=%d skipped=%v", len(plan.Files), plan.Skipped)
	}
	seen := map[string]bool{}
	for _, f := range plan.Files {
		seen[f.Rel] = true
		if f.Text == "" {
			t.Fatalf("empty text for %s", f.Rel)
		}
	}
	if !seen["overlay/alpha.md"] || !seen["overlay/nested/beta.txt"] {
		t.Fatalf("rels=%v", seen)
	}
	skipBlob := false
	skipEmpty := false
	for _, s := range plan.Skipped {
		if strings.Contains(s, "blob.bin") {
			skipBlob = true
		}
		if strings.Contains(s, "empty.txt") {
			skipEmpty = true
		}
	}
	if !skipBlob || !skipEmpty {
		t.Fatalf("skipped=%v", plan.Skipped)
	}

	text := FormatIngestDirPlan(plan, LocalOverlaySessionID, true, true, MemoryIngestDirOpts{})
	for _, want := range []string{"ingest-dir", "dry-run", "local-overlay", "source_hint=private", "dual_write=off", "catalog list ≠ consume", "private overlay"} {
		if !strings.Contains(text, want) {
			t.Fatalf("plan missing %q: %s", want, text)
		}
	}
	if strings.Contains(text, "Memory GA") {
		t.Fatalf("must not stamp Memory GA: %s", text)
	}
	if strings.Contains(text, "source_hint=mesh") {
		t.Fatalf("must not stamp mesh: %s", text)
	}
}

func TestListIngestDirFiles_PathJail(t *testing.T) {
	ws, err := workspace.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, err = ListIngestDirFiles(ws, "/etc", 4)
	if err == nil {
		t.Fatal("expected path jail")
	}
}

func TestIngestDirCaps_D2(t *testing.T) {
	if DefaultIngestDirLimit != 128 {
		t.Fatalf("DefaultIngestDirLimit=%d want 128", DefaultIngestDirLimit)
	}
	if MaxIngestDirFileBytes != 64<<10 {
		t.Fatalf("MaxIngestDirFileBytes=%d want 64KiB", MaxIngestDirFileBytes)
	}
	if maxIngestDirSkipReported < 64 {
		t.Fatalf("maxIngestDirSkipReported=%d want ≥64", maxIngestDirSkipReported)
	}
}

func TestListIngestDirFiles_AllowlistPDFAndNoExt(t *testing.T) {
	root := t.TempDir()
	overlay := filepath.Join(root, "overlay")
	if err := os.MkdirAll(overlay, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(overlay, "ok.md"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(overlay, "data.JSON"), []byte(`{"k":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(overlay, "scan.pdf"), []byte("%PDF-1.4 fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(overlay, "blob.bin"), []byte{0x00, 0x01, 0x02}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(overlay, "noext"), []byte{0x00, 0xff}, 0o644); err != nil {
		t.Fatal(err)
	}
	oversize := bytes.Repeat([]byte("a"), MaxIngestDirFileBytes+1)
	if err := os.WriteFile(filepath.Join(overlay, "big.md"), oversize, 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := ListIngestDirFiles(ws, "overlay", 0)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, f := range plan.Files {
		seen[filepath.Base(f.Rel)] = true
	}
	if !seen["ok.md"] || !seen["data.JSON"] {
		t.Fatalf("allowlist miss files=%v skipped=%v", seen, plan.Skipped)
	}
	if seen["scan.pdf"] || seen["blob.bin"] || seen["noext"] || seen["big.md"] {
		t.Fatalf("should skip pdf/bin/noext/oversize: %v", seen)
	}
	var pdf, bin, noext, big bool
	for _, s := range plan.Skipped {
		if strings.Contains(s, "scan.pdf") {
			pdf = true
			if !strings.Contains(s, "export text first") {
				t.Fatalf("pdf skip must say export text first: %s", s)
			}
			if strings.Contains(s, "not utf-8") {
				t.Fatalf("pdf skip must be distinct from not utf-8: %s", s)
			}
		}
		if strings.Contains(s, "blob.bin") {
			bin = true
		}
		if strings.Contains(s, "noext") {
			noext = true
		}
		if strings.Contains(s, "big.md") {
			big = true
		}
	}
	if !pdf || !bin || !noext || !big {
		t.Fatalf("skipped=%v", plan.Skipped)
	}
}

func TestNormalizeMemoryIngestDirOpts_SourceHintAndTags(t *testing.T) {
	var opts MemoryIngestDirOpts
	if err := NormalizeMemoryIngestDirOpts(&opts); err != nil {
		t.Fatal(err)
	}
	if opts.SourceHint != IngestDirSourceHintPrivate {
		t.Fatalf("default source_hint=%q", opts.SourceHint)
	}
	mesh := MemoryIngestDirOpts{SourceHint: "mesh"}
	err := NormalizeMemoryIngestDirOpts(&mesh)
	if err == nil || !strings.Contains(err.Error(), "mesh") {
		t.Fatalf("mesh must be rejected: %v", err)
	}
	for _, bad := range []string{"catalog", "grant"} {
		o := MemoryIngestDirOpts{SourceHint: bad}
		if err := NormalizeMemoryIngestDirOpts(&o); err == nil {
			t.Fatalf("expected reject %q", bad)
		}
	}
	dept := MemoryIngestDirOpts{Department: "Support", Scenario: "support"}
	if err := NormalizeMemoryIngestDirOpts(&dept); err != nil {
		t.Fatal(err)
	}
	if dept.Department != "support" || dept.Scenario != "support" {
		t.Fatalf("lowercase: %+v", dept)
	}
	tags := IngestDirTags(dept)
	if len(tags) != 2 || tags[0] != "dept:support" || tags[1] != "scenario:support" {
		t.Fatalf("tags=%v", tags)
	}
	sid, minted := ResolveIngestDirSessionID(dept, "cfg-session", "rt-session")
	if sid != "local-overlay:support" || !minted {
		t.Fatalf("department mint sid=%q minted=%v", sid, minted)
	}
	withSess := dept
	withSess.SessionID = "explicit"
	sid2, minted2 := ResolveIngestDirSessionID(withSess, "", "")
	if sid2 != "explicit" || minted2 {
		t.Fatalf("explicit session sid=%q minted=%v", sid2, minted2)
	}
	badID := MemoryIngestDirOpts{Department: "no spaces"}
	if err := NormalizeMemoryIngestDirOpts(&badID); err == nil {
		t.Fatal("expected invalid department")
	}
}

func TestListIngestDirFiles_SupportDeptRCAKit(t *testing.T) {
	if DefaultIngestDirLimit != 128 || MaxIngestDirFileBytes != 64<<10 {
		t.Fatalf("D2 ingest-dir caps: limit=%d want 128 bytes=%d want 64KiB", DefaultIngestDirLimit, MaxIngestDirFileBytes)
	}
	root := moduleRoot(t)
	rel := filepath.Join("examples", "dept-rca", "support")
	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := ListIngestDirFiles(ws, rel, 0)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, f := range plan.Files {
		seen[filepath.Base(f.Rel)] = f.Text
		if f.Size > MaxIngestDirFileBytes {
			t.Fatalf("%s exceeds 64 KiB: %d", f.Rel, f.Size)
		}
		if !utf8.ValidString(f.Text) {
			t.Fatalf("%s is not utf-8", f.Rel)
		}
		if strings.Contains(f.Text, "source_hint=mesh") {
			t.Fatalf("kit must not stamp mesh on files: %s", f.Rel)
		}
	}
	for _, name := range []string{"README.md", "ticket-export.md", "policy.md", "macro.md"} {
		body, ok := seen[name]
		if !ok {
			t.Fatalf("kit missing %s; files=%v skipped=%v", name, seen, plan.Skipped)
		}
		if strings.Contains(body, "leftover_is_bind") && name == "README.md" {
			t.Fatal("kit README must not mention leftover_is_bind")
		}
	}
	ticket := seen["ticket-export.md"]
	policy := seen["policy.md"]
	macro := seen["macro.md"]
	readme := seen["README.md"]
	for _, want := range []string{"ZD-1001", "unused-seat", "2026-06-15T14:22:00Z"} {
		if !strings.Contains(ticket, want) {
			t.Fatalf("ticket-export missing %q", want)
		}
	}
	if !strings.Contains(policy, "14-day") || !strings.Contains(policy, "2026-01-01") {
		t.Fatal("policy must be 14-day unused-seat refund from 2026-01-01")
	}
	if !strings.Contains(macro, "policy.md") {
		t.Fatal("macro must point at policy.md")
	}
	for _, want := range []string{
		"iomesh memory ingest-dir --yes examples/dept-rca/support",
		"--dry-run",
		"private overlay",
		"dept.support.events.*",
		"/memory digest --require-sources mesh,private",
		"2026-06-15T14:22:00Z",
		"not E-G1",
		"not Memory GA",
	} {
		if !strings.Contains(readme, want) {
			t.Fatalf("kit README missing %q", want)
		}
	}
	text := FormatIngestDirPlan(plan, LocalOverlaySessionID, true, true, MemoryIngestDirOpts{})
	for _, want := range []string{"ingest-dir dry-run", "private overlay", "dual_write=off", "source_hint=private"} {
		if !strings.Contains(text, want) {
			t.Fatalf("dry-run missing %q: %s", want, text)
		}
	}
}

func TestListIngestDirFiles_OpsDeptRCAKit(t *testing.T) {
	if DefaultIngestDirLimit != 128 || MaxIngestDirFileBytes != 64<<10 {
		t.Fatalf("D2 ingest-dir caps: limit=%d want 128 bytes=%d want 64KiB", DefaultIngestDirLimit, MaxIngestDirFileBytes)
	}
	root := moduleRoot(t)
	rel := filepath.Join("examples", "dept-rca", "ops")
	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := ListIngestDirFiles(ws, rel, 0)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, f := range plan.Files {
		seen[filepath.Base(f.Rel)] = f.Text
		if f.Size > MaxIngestDirFileBytes {
			t.Fatalf("%s exceeds 64 KiB: %d", f.Rel, f.Size)
		}
		if !utf8.ValidString(f.Text) {
			t.Fatalf("%s is not utf-8", f.Rel)
		}
		if strings.Contains(f.Text, "source_hint=mesh") {
			t.Fatalf("kit must not stamp mesh on files: %s", f.Rel)
		}
	}
	for _, name := range []string{"README.md", "page.md", "runbook.md", "deploy-note.md"} {
		body, ok := seen[name]
		if !ok {
			t.Fatalf("kit missing %s; files=%v skipped=%v", name, seen, plan.Skipped)
		}
		if strings.Contains(body, "leftover_is_bind") && name == "README.md" {
			t.Fatal("kit README must not mention leftover_is_bind")
		}
	}
	page := seen["page.md"]
	runbook := seen["runbook.md"]
	readme := seen["README.md"]
	for _, want := range []string{"PD-HMAC-5xx", "2026-06-15T14:08:00Z"} {
		if !strings.Contains(page, want) {
			t.Fatalf("page missing %q", want)
		}
	}
	if !strings.Contains(runbook, "HMAC 200 is not a consume receipt") {
		t.Fatal("runbook must state HMAC 200 is not a consume receipt")
	}
	kit := readme + page + runbook + seen["deploy-note.md"]
	for _, want := range []string{
		"PD-HMAC-5xx",
		"2026-06-15T14:08:00Z",
		"HMAC 200 is not a consume receipt",
		"ingest-dir --yes examples/dept-rca/ops",
		"private overlay",
		"dept.ops.events.*",
		"not E-G1",
		"not Memory GA",
	} {
		if !strings.Contains(kit, want) {
			t.Fatalf("ops kit missing %q", want)
		}
	}
	for _, want := range []string{
		"iomesh memory ingest-dir --yes examples/dept-rca/ops",
		"--dry-run",
		"private overlay",
		"dept.ops.events.*",
		"/memory digest --require-sources mesh,private",
		"2026-06-15T14:08:00Z",
		"not E-G1",
		"not Memory GA",
	} {
		if !strings.Contains(readme, want) {
			t.Fatalf("kit README missing %q", want)
		}
	}
	text := FormatIngestDirPlan(plan, LocalOverlaySessionID, true, true, MemoryIngestDirOpts{})
	for _, want := range []string{"ingest-dir dry-run", "private overlay", "dual_write=off", "source_hint=private"} {
		if !strings.Contains(text, want) {
			t.Fatalf("dry-run missing %q: %s", want, text)
		}
	}
}

func TestListIngestDirFiles_SalesDeptRCAKit(t *testing.T) {
	if DefaultIngestDirLimit != 128 || MaxIngestDirFileBytes != 64<<10 {
		t.Fatalf("D2 ingest-dir caps: limit=%d want 128 bytes=%d want 64KiB", DefaultIngestDirLimit, MaxIngestDirFileBytes)
	}
	root := moduleRoot(t)
	rel := filepath.Join("examples", "dept-rca", "sales")
	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := ListIngestDirFiles(ws, rel, 0)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, f := range plan.Files {
		seen[filepath.Base(f.Rel)] = f.Text
		if f.Size > MaxIngestDirFileBytes {
			t.Fatalf("%s exceeds 64 KiB: %d", f.Rel, f.Size)
		}
		if !utf8.ValidString(f.Text) {
			t.Fatalf("%s is not utf-8", f.Rel)
		}
		if strings.Contains(f.Text, "source_hint=mesh") {
			t.Fatalf("kit must not stamp mesh on files: %s", f.Rel)
		}
	}
	for _, name := range []string{"README.md", "call-notes.md", "qbr.md", "list-price.md"} {
		body, ok := seen[name]
		if !ok {
			t.Fatalf("kit missing %s; files=%v skipped=%v", name, seen, plan.Skipped)
		}
		if strings.Contains(body, "leftover_is_bind") && name == "README.md" {
			t.Fatal("kit README must not mention leftover_is_bind")
		}
	}
	notes := seen["call-notes.md"]
	readme := seen["README.md"]
	for _, want := range []string{"OPP-1001", "2026-02-20T16:00:00Z"} {
		if !strings.Contains(notes, want) {
			t.Fatalf("call-notes missing %q", want)
		}
	}
	if !strings.Contains(seen["list-price.md"], "2026-03-01") {
		t.Fatal("list-price must be effective until 2026-03-01")
	}
	kit := readme + notes + seen["qbr.md"] + seen["list-price.md"]
	for _, want := range []string{
		"OPP-1001",
		"2026-02-20T16:00:00Z",
		"2026-03-01",
		"2026-02-28T18:00:00Z",
		"ingest-dir --yes examples/dept-rca/sales",
		"private overlay",
		"dept.sales.events.*",
		"not E-G1",
		"not Memory GA",
	} {
		if !strings.Contains(kit, want) {
			t.Fatalf("sales kit missing %q", want)
		}
	}
	for _, bad := range []string{"$400k", "$400K"} {
		if strings.Contains(kit, bad) {
			t.Fatalf("sales kit must not invent ARR %q", bad)
		}
	}
	for _, name := range []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis", "Wilson", "Anderson"} {
		if strings.Contains(kit, name) {
			t.Fatalf("sales kit must not use customer last name %q", name)
		}
	}
	for _, want := range []string{
		"iomesh memory ingest-dir --yes examples/dept-rca/sales",
		"--dry-run",
		"private overlay",
		"dept.sales.events.*",
		"/memory digest --require-sources mesh,private",
		"2026-02-28T18:00:00Z",
		"not E-G1",
		"not Memory GA",
	} {
		if !strings.Contains(readme, want) {
			t.Fatalf("kit README missing %q", want)
		}
	}
	text := FormatIngestDirPlan(plan, LocalOverlaySessionID, true, true, MemoryIngestDirOpts{})
	for _, want := range []string{"ingest-dir dry-run", "private overlay", "dual_write=off", "source_hint=private"} {
		if !strings.Contains(text, want) {
			t.Fatalf("dry-run missing %q: %s", want, text)
		}
	}
}

func TestListIngestDirFiles_CSDeptRCAKit(t *testing.T) {
	if DefaultIngestDirLimit != 128 || MaxIngestDirFileBytes != 64<<10 {
		t.Fatalf("D2 ingest-dir caps: limit=%d want 128 bytes=%d want 64KiB", DefaultIngestDirLimit, MaxIngestDirFileBytes)
	}
	root := moduleRoot(t)
	rel := filepath.Join("examples", "dept-rca", "customer_success")
	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := ListIngestDirFiles(ws, rel, 0)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, f := range plan.Files {
		seen[filepath.Base(f.Rel)] = f.Text
		if f.Size > MaxIngestDirFileBytes {
			t.Fatalf("%s exceeds 64 KiB: %d", f.Rel, f.Size)
		}
		if !utf8.ValidString(f.Text) {
			t.Fatalf("%s is not utf-8", f.Rel)
		}
		if strings.Contains(f.Text, "source_hint=mesh") {
			t.Fatalf("kit must not stamp mesh on files: %s", f.Rel)
		}
	}
	for _, name := range []string{"README.md", "health-note.md", "renewal.md", "playbook.md"} {
		body, ok := seen[name]
		if !ok {
			t.Fatalf("kit missing %s; files=%v skipped=%v", name, seen, plan.Skipped)
		}
		if strings.Contains(body, "leftover_is_bind") && name == "README.md" {
			t.Fatal("kit README must not mention leftover_is_bind")
		}
	}
	health := seen["health-note.md"]
	readme := seen["README.md"]
	for _, want := range []string{"ACC-1001", "2026-08-15"} {
		if !strings.Contains(health, want) {
			t.Fatalf("health-note missing %q", want)
		}
	}
	if !strings.Contains(seen["renewal.md"], "2026-09-01") {
		t.Fatal("renewal must have window 2026-09-01")
	}
	if !strings.Contains(seen["playbook.md"], "health-note.md") {
		t.Fatal("playbook must point at health-note.md")
	}
	kit := readme + health + seen["renewal.md"] + seen["playbook.md"]
	for _, want := range []string{
		"ACC-1001",
		"2026-08-15",
		"2026-09-01",
		"2026-08-31T18:00:00Z",
		"ingest-dir --yes examples/dept-rca/customer_success",
		"private overlay",
		"dept.customer_success.events.*",
		"not E-G1",
		"not Memory GA",
	} {
		if !strings.Contains(kit, want) {
			t.Fatalf("cs kit missing %q", want)
		}
	}
	for _, bad := range []string{"$400k", "$400K"} {
		if strings.Contains(kit, bad) {
			t.Fatalf("cs kit must not invent ARR %q", bad)
		}
	}
	if i := strings.Index(kit, "$"); i >= 0 && i+1 < len(kit) {
		c := kit[i+1]
		if c >= '0' && c <= '9' {
			end := i + 8
			if end > len(kit) {
				end = len(kit)
			}
			t.Fatalf("cs kit must not invent ARR dollar amounts: %q", kit[i:end])
		}
	}
	for _, name := range []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis", "Wilson", "Anderson"} {
		if strings.Contains(kit, name) {
			t.Fatalf("cs kit must not use customer last name %q", name)
		}
	}
	for _, want := range []string{
		"iomesh memory ingest-dir --yes examples/dept-rca/customer_success",
		"--dry-run",
		"private overlay",
		"dept.customer_success.events.*",
		"/memory digest --require-sources mesh,private",
		"2026-08-31T18:00:00Z",
		"not E-G1",
		"not Memory GA",
	} {
		if !strings.Contains(readme, want) {
			t.Fatalf("kit README missing %q", want)
		}
	}
	text := FormatIngestDirPlan(plan, LocalOverlaySessionID, true, true, MemoryIngestDirOpts{})
	for _, want := range []string{"ingest-dir dry-run", "private overlay", "dual_write=off", "source_hint=private"} {
		if !strings.Contains(text, want) {
			t.Fatalf("dry-run missing %q: %s", want, text)
		}
	}
}

func TestMemoryIngestDir_DryRunNoMCP(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes", "n.md"), []byte("needle"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, Server: "memory", DualWrite: false},
		ws:     ws,
	}
	out, err := rt.MemoryIngestDir(context.Background(), MemoryIngestDirOpts{Path: "notes", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "ingest-dir dry-run") || !strings.Contains(out, "notes/n.md") {
		t.Fatalf("out=%q", out)
	}
	if !strings.Contains(out, "session_id=local-overlay") || !strings.Contains(out, "dual_write=off") {
		t.Fatalf("honesty: %q", out)
	}
	if !strings.Contains(out, "source_hint=private") {
		t.Fatalf("source_hint missing: %q", out)
	}
}

func TestMemoryIngestDir_MeshOrgHeaderWhenDualWrite(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes", "alpha.md"), []byte("Project alpha ships Friday"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatal(err)
	}

	var gotOrg string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotOrg = r.Header.Get("X-IOMesh-Org")
		_ = json.NewEncoder(w).Encode(map[string]any{"stream": "MEMORY_INGEST", "seq": 1})
	}))
	defer srv.Close()

	mesh := iomesh.New(iomesh.Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "dept.engineering", OrgID: "org_a",
	}, nil)
	rt := &Runtime{
		mesh: mesh,
		memory: MemoryConfig{
			Enabled: true, DualWrite: true, Tenant: "dept.engineering", Server: "memory",
		},
		ws: ws,
	}
	out, err := rt.MemoryIngestDir(context.Background(), MemoryIngestDirOpts{Path: "notes"})
	if err != nil {
		t.Fatalf("err=%v out=%q", err, out)
	}
	if gotOrg != "org_a" {
		t.Fatalf("ingest-dir mesh path must send X-IOMesh-Org=org_a; got %q", gotOrg)
	}
	if !strings.Contains(out, "ingest-dir") || !strings.Contains(out, "ingested=1") {
		t.Fatalf("out=%q", out)
	}
}

func TestMemoryIngestTurn_OrgHeaderWhenDualWrite(t *testing.T) {
	var gotOrg string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotOrg = r.Header.Get("X-IOMesh-Org")
		_ = json.NewEncoder(w).Encode(map[string]any{"stream": "MEMORY_INGEST", "seq": 2})
	}))
	defer srv.Close()

	mesh := iomesh.New(iomesh.Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "acme", OrgID: "org_a",
	}, nil)
	rt := &Runtime{
		mesh:   mesh,
		memory: MemoryConfig{Enabled: true, DualWrite: true, Tenant: "acme", Server: "memory"},
	}
	if _, err := rt.MemoryIngestTurn(context.Background(), "user", "note for org_a"); err != nil {
		t.Fatal(err)
	}
	if gotOrg != "org_a" {
		t.Fatalf("X-IOMesh-Org=%q", gotOrg)
	}
}

func TestMemoryIngestTurn_OmitsOrgWhenUnset(t *testing.T) {
	var gotOrg string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotOrg = r.Header.Get("X-IOMesh-Org")
		_ = json.NewEncoder(w).Encode(map[string]any{"stream": "MEMORY_INGEST", "seq": 1})
	}))
	defer srv.Close()

	mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: srv.URL, Tenant: "acme"}, nil)
	rt := &Runtime{
		mesh:   mesh,
		memory: MemoryConfig{Enabled: true, DualWrite: true, Tenant: "acme", Server: "memory"},
	}
	if _, err := rt.MemoryIngestTurn(context.Background(), "user", "note"); err != nil {
		t.Fatal(err)
	}
	if gotOrg != "" {
		t.Fatalf("empty org must omit X-IOMesh-Org (fail-open); got %q", gotOrg)
	}
}

func TestMemoryIngestDir_MockMCP(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes", "alpha.md"), []byte("Project alpha ships Friday"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatal(err)
	}

	var gotArgs map[string]any
	cInR, cInW := io.Pipe()
	cOutR, cOutW := io.Pipe()
	go mockMCPIngestTurn(cOutW, cInR, &gotArgs)

	mut := true
	cl := mcp.NewClientForTest(mcp.ServerConfig{Name: "memory", Command: "x", Mutating: &mut}, cInW, cOutR, nil)
	defer cl.Close()
	if err := cl.InitForTest(context.Background()); err != nil {
		t.Fatal(err)
	}
	mgr := mcp.NewManagerEmpty(nil)
	mgr.Attach(cl)

	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, Server: "memory", Tenant: "default", DualWrite: false},
		mcp:    mgr,
		ws:     ws,
	}
	out, err := rt.MemoryIngestDir(context.Background(), MemoryIngestDirOpts{Path: "notes"})
	if err != nil {
		t.Fatalf("err=%v out=%q", err, out)
	}
	if gotArgs["session_id"] != LocalOverlaySessionID {
		t.Fatalf("session_id=%v", gotArgs["session_id"])
	}
	if gotArgs["source_hint"] != IngestDirSourceHintPrivate {
		t.Fatalf("ingest-dir must pass source_hint=private; got %v", gotArgs["source_hint"])
	}
	if _, ok := gotArgs["tags"]; ok {
		t.Fatalf("tags must be omitted when empty; got %v", gotArgs["tags"])
	}
	content, _ := gotArgs["content"].(string)
	if !strings.Contains(content, "file: notes/alpha.md") || !strings.Contains(content, "Project alpha ships Friday") {
		t.Fatalf("content=%q", content)
	}
	if !strings.Contains(content, "source_hint: private") {
		t.Fatalf("content header missing source_hint: %q", content)
	}
	if strings.Contains(strings.ToLower(content), "source_hint: mesh") || gotArgs["source_hint"] == "mesh" {
		t.Fatalf("must not stamp mesh: args=%v content=%q", gotArgs, content)
	}
	if !strings.Contains(out, "ingest-dir") || !strings.Contains(out, "ingested=1") {
		t.Fatalf("out=%q", out)
	}
	if !strings.Contains(out, "dual_write=false") {
		t.Fatalf("dual_write pin: %q", out)
	}
}

func TestMemoryIngestDir_DepartmentTagsAndSession(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes", "alpha.md"), []byte("Project alpha ships Friday"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatal(err)
	}

	var gotArgs map[string]any
	cInR, cInW := io.Pipe()
	cOutR, cOutW := io.Pipe()
	go mockMCPIngestTurn(cOutW, cInR, &gotArgs)

	mut := true
	cl := mcp.NewClientForTest(mcp.ServerConfig{Name: "memory", Command: "x", Mutating: &mut}, cInW, cOutR, nil)
	defer cl.Close()
	if err := cl.InitForTest(context.Background()); err != nil {
		t.Fatal(err)
	}
	mgr := mcp.NewManagerEmpty(nil)
	mgr.Attach(cl)

	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, Server: "memory", Tenant: "default", DualWrite: false},
		mcp:    mgr,
		ws:     ws,
	}
	opts := MemoryIngestDirOpts{Path: "notes", Department: "support", Scenario: "support"}
	out, err := rt.MemoryIngestDir(context.Background(), opts)
	if err != nil {
		t.Fatalf("err=%v out=%q", err, out)
	}
	if gotArgs["source_hint"] != IngestDirSourceHintPrivate {
		t.Fatalf("source_hint=%v", gotArgs["source_hint"])
	}
	gotTags := ingestDirTagStrings(gotArgs["tags"])
	if len(gotTags) != 2 || gotTags[0] != "dept:support" || gotTags[1] != "scenario:support" {
		t.Fatalf("tags=%v (%T)", gotArgs["tags"], gotArgs["tags"])
	}
	if gotArgs["session_id"] != "local-overlay:support" {
		t.Fatalf("session_id=%v", gotArgs["session_id"])
	}
	content, _ := gotArgs["content"].(string)
	if !strings.Contains(content, "tags: dept:support, scenario:support") {
		t.Fatalf("content header tags: %q", content)
	}
	if !strings.Contains(out, "dept:support") || !strings.Contains(out, "source_hint=private") {
		t.Fatalf("out=%q", out)
	}

	dry, err := rt.MemoryIngestDir(context.Background(), MemoryIngestDirOpts{
		Path: "notes", DryRun: true, Department: "support", Scenario: "support",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ingest-dir dry-run", "dept:support", "scenario:support", "source_hint=private", "local-overlay:support"} {
		if !strings.Contains(dry, want) {
			t.Fatalf("dry-run missing %q: %s", want, dry)
		}
	}
}

func TestMemoryIngestDir_DryRunRejectsMeshHint(t *testing.T) {
	rt := &Runtime{memory: MemoryConfig{Enabled: true}}
	_, err := rt.MemoryIngestDir(context.Background(), MemoryIngestDirOpts{Path: "notes", DryRun: true, SourceHint: "mesh"})
	if err == nil || !strings.Contains(err.Error(), "mesh") {
		t.Fatalf("mesh source-hint must fail: %v", err)
	}
}

func TestCallIngestDirMCP_RetriesWithoutTags(t *testing.T) {
	calls := 0
	call := func(_ context.Context, name string, args map[string]any) (string, error) {
		calls++
		if name != "memory_ingest_turn" {
			t.Fatalf("tool=%s", name)
		}
		if _, ok := args["tags"]; ok {
			return "", fmt.Errorf(`mcp tool error: additional properties 'tags' not allowed`)
		}
		if args["source_hint"] != IngestDirSourceHintPrivate {
			t.Fatalf("retry must keep source_hint=private: %v", args["source_hint"])
		}
		return "ok", nil
	}
	args := map[string]any{
		"source_hint": IngestDirSourceHintPrivate,
		"tags":        []string{"dept:support"},
		"content":     "file: x.md\nsource_hint: private\ntags: dept:support\n\nhi",
	}
	out, err := CallIngestDirMCP(context.Background(), call, args)
	if err != nil || out != "ok" {
		t.Fatalf("out=%q err=%v", out, err)
	}
	if calls != 2 {
		t.Fatalf("calls=%d want 2", calls)
	}
	if _, ok := args["tags"]; ok {
		t.Fatal("tags should be stripped on retry")
	}
}

func ingestDirTagStrings(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			out = append(out, fmt.Sprint(x))
		}
		return out
	default:
		return nil
	}
}

func mockMCPIngestTurn(w io.WriteCloser, r io.Reader, got *map[string]any) {
	defer w.Close()
	dec := json.NewDecoder(r)
	for {
		var req map[string]any
		if err := dec.Decode(&req); err != nil {
			return
		}
		id := req["id"]
		method, _ := req["method"].(string)
		if method == "notifications/initialized" || id == nil {
			continue
		}
		var result any
		switch method {
		case "initialize":
			result = map[string]any{"protocolVersion": "2024-11-05", "serverInfo": map[string]string{"name": "memory", "version": "1"}}
		case "tools/list":
			result = map[string]any{"tools": []map[string]any{{
				"name": "memory_ingest_turn", "description": "ingest turn",
				"inputSchema": map[string]any{"type": "object"},
			}}}
		case "tools/call":
			if params, _ := req["params"].(map[string]any); params != nil {
				if args, ok := params["arguments"].(map[string]any); ok && got != nil {
					*got = args
				}
			}
			payload := `{"memory_id":"mem_test","tier":1,"tenant":"default","audited":false,"dual_write":"off"}`
			result = map[string]any{"content": []map[string]any{{"type": "text", "text": payload}}}
		}
		line, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
		_, _ = w.Write(append(line, '\n'))
	}
}
