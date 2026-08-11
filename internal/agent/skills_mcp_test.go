package agent

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/mcp"
	"github.com/iome-sh/iomesh-tui/internal/router"
	"github.com/iome-sh/iomesh-tui/internal/skills"
)

func TestAttachSkills_ToolsAndPrompt(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".iomesh", "skills", "demo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: demo\ndescription: Demo skill\n---\n\n# Demo\n\nHello skill.\n"), 0o644)

	cat, err := skills.LoadDirs(filepath.Join(root, ".iomesh", "skills"))
	if err != nil {
		t.Fatal(err)
	}
	rt := testRT(t, root)
	rt.AttachSkills(cat)
	if rt.Skills() == nil || rt.Skills().Len() != 1 {
		t.Fatal("skills not attached")
	}
	sys := rt.Messages()[0].Content
	if !strings.Contains(sys, "demo") {
		t.Fatalf("system prompt missing skills: %s", sys)
	}
	out, err := rt.tools.Execute(context.Background(), "list_skills", `{}`)
	if err != nil || !strings.Contains(out, "demo") {
		t.Fatalf("list_skills: %q err=%v", out, err)
	}
	out, err = rt.tools.Execute(context.Background(), "read_skill", `{"name":"demo"}`)
	if err != nil || !strings.Contains(out, "Hello skill") {
		t.Fatalf("read_skill: %q err=%v", out, err)
	}
}

func TestAttachMCP_Tools(t *testing.T) {
	cInR, cInW := io.Pipe()
	cOutR, cOutW := io.Pipe()
	go mockMCPAgent(cOutW, cInR)

	mut := false
	cl := mcp.NewClientForTest(mcp.ServerConfig{Name: "mock", Command: "x", Mutating: &mut}, cInW, cOutR, nil)
	defer cl.Close()
	if err := cl.InitForTest(context.Background()); err != nil {
		t.Fatal(err)
	}

	mgr := mcp.NewManagerEmpty(nil)
	mgr.Attach(cl)
	rt := testRT(t, t.TempDir())
	rt.AttachMCP(mgr)
	if rt.MCP() == nil {
		t.Fatal("mcp not attached")
	}
	if !strings.Contains(rt.Messages()[0].Content, "MCP") {
		t.Fatal(rt.Messages()[0].Content)
	}
	name := mcp.ToolQualifiedName("mock", "echo")
	if rt.tools.IsMutating(name) {
		t.Fatal("expected non-mutating")
	}
	out, err := rt.tools.Execute(context.Background(), name, `{"text":"ok"}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "echo:ok" {
		t.Fatalf("%q", out)
	}
}

func TestAttachMCP_EmptyNoop(t *testing.T) {
	rt := testRT(t, t.TempDir())
	rt.AttachMCP(mcp.NewManagerEmpty(nil))
	if rt.MCP() != nil {
		t.Fatal("empty should not attach")
	}
}

// s1526 P4: ReplaceMCP nil-safe detach + hot-swap tools without process restart.
func TestReplaceMCP_NilSafeDetach(t *testing.T) {
	rt := testRT(t, t.TempDir())
	// No prior MCP — nil is fine.
	rt.ReplaceMCP(nil)
	if rt.MCP() != nil {
		t.Fatal("expected nil MCP after ReplaceMCP(nil)")
	}
	// Empty manager detaches (same as nil).
	rt.ReplaceMCP(mcp.NewManagerEmpty(nil))
	if rt.MCP() != nil {
		t.Fatal("empty manager must detach")
	}
	sys := rt.Messages()[0].Content
	if !strings.Contains(sys, "detached") {
		t.Fatalf("expected detached mcp note: %s", sys)
	}
}

func TestReplaceMCP_HotSwapTools(t *testing.T) {
	// First mock server.
	cInR1, cInW1 := io.Pipe()
	cOutR1, cOutW1 := io.Pipe()
	go mockMCPAgent(cOutW1, cInR1)
	mut := false
	cl1 := mcp.NewClientForTest(mcp.ServerConfig{Name: "mock-a", Command: "x", Mutating: &mut}, cInW1, cOutR1, nil)
	defer cl1.Close()
	if err := cl1.InitForTest(context.Background()); err != nil {
		t.Fatal(err)
	}
	mgr1 := mcp.NewManagerEmpty(nil)
	mgr1.Attach(cl1)

	rt := testRT(t, t.TempDir())
	rt.AttachMCP(mgr1)
	nameA := mcp.ToolQualifiedName("mock-a", "echo")
	if _, err := rt.tools.Execute(context.Background(), nameA, `{"text":"a"}`); err != nil {
		t.Fatalf("tool a before replace: %v", err)
	}
	sys1 := rt.Messages()[0].Content
	if strings.Count(sys1, "<integrations>") != 1 {
		t.Fatalf("expected single integrations note after attach, got %d", strings.Count(sys1, "<integrations>"))
	}

	// Second mock server — ReplaceMCP should unregister mock-a tools and attach mock-b.
	cInR2, cInW2 := io.Pipe()
	cOutR2, cOutW2 := io.Pipe()
	go mockMCPAgent(cOutW2, cInR2)
	cl2 := mcp.NewClientForTest(mcp.ServerConfig{Name: "mock-b", Command: "x", Mutating: &mut}, cInW2, cOutR2, nil)
	defer cl2.Close()
	if err := cl2.InitForTest(context.Background()); err != nil {
		t.Fatal(err)
	}
	mgr2 := mcp.NewManagerEmpty(nil)
	mgr2.Attach(cl2)

	rt.ReplaceMCP(mgr2)
	if rt.MCP() == nil {
		t.Fatal("mcp not attached after replace")
	}
	nameB := mcp.ToolQualifiedName("mock-b", "echo")
	if _, err := rt.tools.Execute(context.Background(), nameB, `{"text":"b"}`); err != nil {
		t.Fatalf("tool b after replace: %v", err)
	}
	// Old mcp__ tool must be gone.
	if _, err := rt.tools.Execute(context.Background(), nameA, `{"text":"a"}`); err == nil {
		t.Fatal("expected mock-a tool unregistered after ReplaceMCP")
	}
	// System notes upsert — no duplicate integrations/mcp tags.
	sys2 := rt.Messages()[0].Content
	if strings.Count(sys2, "<integrations>") != 1 {
		t.Fatalf("integrations note duplicated: %d", strings.Count(sys2, "<integrations>"))
	}
	if strings.Count(sys2, "<mcp>") != 1 {
		t.Fatalf("mcp note count: %d", strings.Count(sys2, "<mcp>"))
	}
	if !strings.Contains(sys2, "1 server") {
		t.Fatalf("mcp note should reflect new manager: %s", sys2)
	}

	// Detach via ReplaceMCP(nil) after attach.
	rt.ReplaceMCP(nil)
	if rt.MCP() != nil {
		t.Fatal("expected nil after detach")
	}
	if _, err := rt.tools.Execute(context.Background(), nameB, `{"text":"b"}`); err == nil {
		t.Fatal("expected tools cleared on detach")
	}
}

func TestUnregisterMCPTools_Idempotent(t *testing.T) {
	reg := ToolRegistry{funcs: map[string]toolFunc{}, meta: map[string]toolMeta{}}
	// No panic on empty.
	reg.UnregisterMCPTools()
	reg.UnregisterMCPTools()
}

// s1670: ReplaceSkills hot-swaps catalog tools without process restart.
func TestReplaceSkills_HotSwap(t *testing.T) {
	root := t.TempDir()
	dirA := filepath.Join(root, "skills-a", "skill-a")
	dirB := filepath.Join(root, "skills-b", "skill-b")
	if err := os.MkdirAll(dirA, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dirB, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dirA, "SKILL.md"), []byte("---\nname: skill-a\ndescription: Skill A\n---\n\n# A\n\nBody A.\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dirB, "SKILL.md"), []byte("---\nname: skill-b\ndescription: Skill B\n---\n\n# B\n\nBody B.\n"), 0o644)

	catA, err := skills.LoadDirs(filepath.Join(root, "skills-a"))
	if err != nil {
		t.Fatal(err)
	}
	catB, err := skills.LoadDirs(filepath.Join(root, "skills-b"))
	if err != nil {
		t.Fatal(err)
	}

	rt := testRT(t, root)
	rt.AttachSkills(catA)
	out, err := rt.tools.Execute(context.Background(), "list_skills", `{}`)
	if err != nil || !strings.Contains(out, "skill-a") {
		t.Fatalf("list_skills A: %q err=%v", out, err)
	}
	if strings.Contains(out, "skill-b") {
		t.Fatalf("list_skills should not list B yet: %q", out)
	}
	sys1 := rt.Messages()[0].Content
	// defaultSystemPrompt also mentions <skills> as a word; count closed note tags for upsert.
	if strings.Count(sys1, "</skills>") != 1 {
		t.Fatalf("expected single skills note after attach, got %d", strings.Count(sys1, "</skills>"))
	}
	if strings.Count(sys1, "<gtm-draft-only>") != 1 {
		t.Fatalf("expected single gtm-draft-only note, got %d", strings.Count(sys1, "<gtm-draft-only>"))
	}

	// Hot-swap to skill B catalog — skill-a tools must go.
	rt.ReplaceSkills(catB)
	if rt.Skills() == nil || rt.Skills().Len() != 1 {
		t.Fatal("skills not attached after replace")
	}
	out, err = rt.tools.Execute(context.Background(), "list_skills", `{}`)
	if err != nil || !strings.Contains(out, "skill-b") {
		t.Fatalf("list_skills B after replace: %q err=%v", out, err)
	}
	if strings.Contains(out, "skill-a") {
		t.Fatalf("skill-a must be gone after ReplaceSkills: %q", out)
	}
	readOut, err := rt.tools.Execute(context.Background(), "read_skill", `{"name":"skill-b"}`)
	if err != nil || !strings.Contains(readOut, "Body B") {
		t.Fatalf("read_skill B: %q err=%v", readOut, err)
	}
	if _, err := rt.tools.Execute(context.Background(), "read_skill", `{"name":"skill-a"}`); err == nil {
		t.Fatal("expected skill-a unknown after ReplaceSkills")
	}
	sys2 := rt.Messages()[0].Content
	if strings.Count(sys2, "</skills>") != 1 {
		t.Fatalf("skills note duplicated: %d", strings.Count(sys2, "</skills>"))
	}
	if strings.Count(sys2, "<gtm-draft-only>") != 1 {
		t.Fatalf("gtm-draft-only note duplicated: %d", strings.Count(sys2, "<gtm-draft-only>"))
	}
	if !strings.Contains(sys2, "skill-b") {
		t.Fatalf("skills note should reflect new catalog: %s", sys2)
	}
	if strings.Contains(sys2, "skill-a") {
		t.Fatalf("skills note still mentions skill-a after replace: %s", sys2)
	}

	// Detach via ReplaceSkills(nil).
	rt.ReplaceSkills(nil)
	if rt.Skills() != nil {
		t.Fatal("expected nil skills after detach")
	}
	if _, err := rt.tools.Execute(context.Background(), "list_skills", `{}`); err == nil {
		t.Fatal("expected list_skills unregistered after ReplaceSkills(nil)")
	}
	if _, err := rt.tools.Execute(context.Background(), "read_skill", `{"name":"skill-b"}`); err == nil {
		t.Fatal("expected read_skill unregistered after ReplaceSkills(nil)")
	}
	sys3 := rt.Messages()[0].Content
	if !strings.Contains(sys3, "detached") {
		t.Fatalf("expected detached skills note: %s", sys3)
	}
}

func TestUnregisterSkillsTools_Idempotent(t *testing.T) {
	reg := ToolRegistry{funcs: map[string]toolFunc{}, meta: map[string]toolMeta{}}
	reg.UnregisterSkillsTools()
	reg.UnregisterSkillsTools()
}

func testRT(t *testing.T, workspace string) *Runtime {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(router.ChatResponse{
			Choices: []router.Choice{{Message: router.Message{Role: "assistant", Content: "x"}, FinishReason: "stop"}},
		})
	}))
	t.Cleanup(srv.Close)
	rtr, err := router.New([]router.ModelConfig{{
		Name: "m", BaseURL: srv.URL, ModelID: "m", APIKey: "k",
		CostTier: 1, MaxContext: 1e5, Capabilities: []string{"fast"}, Priority: 1,
	}}, "m")
	if err != nil {
		t.Fatal(err)
	}
	rt, err := New(Config{Workspace: workspace, SubagentsEnabled: false}, rtr, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return rt
}

func mockMCPAgent(w io.WriteCloser, r io.Reader) {
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
			result = map[string]any{"protocolVersion": "2024-11-05", "serverInfo": map[string]string{"name": "m", "version": "1"}}
		case "tools/list":
			result = map[string]any{"tools": []map[string]any{{
				"name": "echo", "description": "Echo", "inputSchema": map[string]any{
					"type": "object", "properties": map[string]any{"text": map[string]any{"type": "string"}},
				},
			}}}
		case "tools/call":
			// Echo text from arguments when present.
			text := "ok"
			if params, ok := req["params"].(map[string]any); ok {
				if args, ok := params["arguments"].(map[string]any); ok {
					if v, ok := args["text"].(string); ok {
						text = v
					}
				}
			}
			result = map[string]any{"content": []map[string]any{{"type": "text", "text": "echo:" + text}}}
		}
		line, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
		line = append(line, '\n')
		_, _ = w.Write(line)
	}
}
