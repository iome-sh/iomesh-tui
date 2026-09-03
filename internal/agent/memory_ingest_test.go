package agent

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/mcp"
	"github.com/iome-sh/iomesh-tui/internal/workspace"
)

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
		memory: MemoryConfig{Enabled: true, Server: "memory", Tenant: "default", DualWrite: false},
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
	if !strings.Contains(out, "session_id=local-overlay") || !strings.Contains(out, "minted") {
		t.Fatalf("out=%q", out)
	}
	if !strings.Contains(out, "dual_write=false") {
		t.Fatalf("dual_write pin missing: %q", out)
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

	text := FormatIngestDirPlan(plan, LocalOverlaySessionID, true, true)
	for _, want := range []string{"ingest-dir", "dry-run", "local-overlay", "dual_write=off", "catalog list ≠ consume", "private overlay"} {
		if !strings.Contains(text, want) {
			t.Fatalf("plan missing %q: %s", want, text)
		}
	}
	if strings.Contains(text, "Memory GA") && !strings.Contains(text, "not Memory GA") {
		t.Fatalf("must not stamp Memory GA: %s", text)
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
	content, _ := gotArgs["content"].(string)
	if !strings.Contains(content, "file: notes/alpha.md") || !strings.Contains(content, "Project alpha ships Friday") {
		t.Fatalf("content=%q", content)
	}
	if !strings.Contains(out, "ingest-dir") || !strings.Contains(out, "ingested=1") {
		t.Fatalf("out=%q", out)
	}
	if !strings.Contains(out, "dual_write=false") {
		t.Fatalf("dual_write pin: %q", out)
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
