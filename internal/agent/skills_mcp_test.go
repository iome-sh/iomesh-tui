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
