package mcp

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockServer implements a minimal MCP server on pipes.
func mockServer(t *testing.T, clientWriter io.WriteCloser, clientReader io.Reader) {
	t.Helper()
	go func() {
		defer clientWriter.Close()
		dec := json.NewDecoder(clientReader)
		for {
			var req rpcRequest
			if err := dec.Decode(&req); err != nil {
				return
			}
			var resp rpcResponse
			resp.JSONRPC = "2.0"
			resp.ID = req.ID
			switch req.Method {
			case "initialize":
				resp.Result, _ = json.Marshal(initializeResult{
					ProtocolVersion: ProtocolVersion,
					ServerInfo: struct {
						Name    string `json:"name"`
						Version string `json:"version"`
					}{Name: "mock", Version: "1"},
				})
			case "notifications/initialized":
				continue
			case "tools/list":
				schema := json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`)
				resp.Result, _ = json.Marshal(toolsListResult{
					Tools: []Tool{{
						Name:        "echo",
						Description: "Echo text",
						InputSchema: schema,
					}},
				})
			case "tools/call":
				var p callToolParams
				_ = json.Unmarshal(req.Params, &p)
				text := ""
				if p.Arguments != nil {
					if v, ok := p.Arguments["text"].(string); ok {
						text = v
					}
				}
				resp.Result, _ = json.Marshal(callToolResult{
					Content: []contentPart{{Type: "text", Text: "echo:" + text}},
				})
			default:
				resp.Error = &rpcError{Code: -32601, Message: "method not found"}
			}
			line, _ := json.Marshal(resp)
			line = append(line, '\n')
			_, _ = clientWriter.Write(line)
		}
	}()
}

func TestClient_InitializeListCall(t *testing.T) {
	// client stdin -> server reads; server writes -> client stdout
	cInR, cInW := io.Pipe()   // client writes requests to cInW; server reads cInR
	cOutR, cOutW := io.Pipe() // server writes to cOutW; client reads cOutR

	mockServer(t, cOutW, cInR)

	cfg := ServerConfig{Name: "mock", Command: "unused", Mutating: boolPtr(false)}
	c := NewClientForTest(cfg, cInW, cOutR, nil)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.initialize(ctx); err != nil {
		t.Fatal(err)
	}
	if err := c.refreshTools(ctx); err != nil {
		t.Fatal(err)
	}
	tools := c.Tools()
	if len(tools) != 1 || tools[0].Name != "echo" {
		t.Fatalf("%+v", tools)
	}
	out, err := c.CallTool(ctx, "echo", map[string]any{"text": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "echo:hi" {
		t.Fatalf("out=%q", out)
	}
}

func TestManager_AttachAndCall(t *testing.T) {
	cInR, cInW := io.Pipe()
	cOutR, cOutW := io.Pipe()
	mockServer(t, cOutW, cInR)

	cfg := ServerConfig{Name: "mock", Command: "x"}
	c := NewClientForTest(cfg, cInW, cOutR, nil)
	ctx := context.Background()
	if err := c.initialize(ctx); err != nil {
		t.Fatal(err)
	}
	if err := c.refreshTools(ctx); err != nil {
		t.Fatal(err)
	}

	m := NewManagerEmpty(nil)
	m.Attach(c)
	defer m.Close()

	q := ToolQualifiedName("mock", "echo")
	if !strings.HasPrefix(q, "mcp__") {
		t.Fatal(q)
	}
	out, err := m.Call(ctx, q, map[string]any{"text": "z"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "echo:z" {
		t.Fatal(out)
	}
}

func TestSplitQualified(t *testing.T) {
	s, tool, ok := SplitQualified("mcp__github__create_issue")
	if !ok || s != "github" || tool != "create_issue" {
		t.Fatalf("%s %s %v", s, tool, ok)
	}
	_, _, ok = SplitQualified("write_file")
	if ok {
		t.Fatal("expected fail")
	}
}

func TestDialStdio_MissingCommand(t *testing.T) {
	_, err := DialStdio(context.Background(), ServerConfig{Name: "x"}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConcurrentCalls(t *testing.T) {
	cInR, cInW := io.Pipe()
	cOutR, cOutW := io.Pipe()
	mockServer(t, cOutW, cInR)
	c := NewClientForTest(ServerConfig{Name: "m", Command: "x"}, cInW, cOutR, nil)
	defer c.Close()
	ctx := context.Background()
	_ = c.initialize(ctx)
	_ = c.refreshTools(ctx)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			out, err := c.CallTool(ctx, "echo", map[string]any{"text": "n"})
			if err != nil || !strings.Contains(out, "echo:") {
				t.Errorf("n=%d out=%q err=%v", n, out, err)
			}
		}(i)
	}
	wg.Wait()
}

func boolPtr(b bool) *bool { return &b }
