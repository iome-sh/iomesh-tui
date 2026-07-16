package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDialHTTP_JSON(t *testing.T) {
	var gotSession string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", 405)
			return
		}
		var req rpcRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if r.Header.Get("Accept") == "" {
			t.Error("missing Accept")
		}
		w.Header().Set("Mcp-Session-Id", "sess-1")
		w.Header().Set("Content-Type", "application/json")
		var result any
		switch req.Method {
		case "initialize":
			result = initializeResult{ProtocolVersion: ProtocolVersion}
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
			return
		case "tools/list":
			result = toolsListResult{Tools: []Tool{{
				Name:        "echo",
				Description: "Echo",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}}}`),
			}}}
		case "tools/call":
			var p callToolParams
			_ = json.Unmarshal(req.Params, &p)
			text := ""
			if p.Arguments != nil {
				if v, ok := p.Arguments["text"].(string); ok {
					text = v
				}
			}
			result = callToolResult{Content: []contentPart{{Type: "text", Text: "echo:" + text}}}
		default:
			_ = json.NewEncoder(w).Encode(rpcResponse{
				JSONRPC: "2.0", ID: req.ID,
				Error: &rpcError{Code: -32601, Message: "unknown"},
			})
			return
		}
		raw, _ := json.Marshal(result)
		_ = json.NewEncoder(w).Encode(rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: raw})
		gotSession = "sess-1"
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c, err := DialHTTP(ctx, ServerConfig{Name: "http-mock", URL: srv.URL, Mutating: boolPtr(false)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if !c.isHTTP() {
		t.Fatal("expected http client")
	}
	if gotSession == "" {
		t.Log("session header path exercised via client state")
	}
	if c.sessionID != "sess-1" {
		t.Fatalf("sessionID=%q", c.sessionID)
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
		t.Fatalf("%q", out)
	}
}

func TestDialHTTP_SSE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Method == "notifications/initialized" {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Mcp-Session-Id", "sse-1")
		flusher, _ := w.(http.Flusher)
		var result any
		switch req.Method {
		case "initialize":
			result = initializeResult{ProtocolVersion: ProtocolVersion}
		case "tools/list":
			result = toolsListResult{Tools: []Tool{{Name: "ping", Description: "p", InputSchema: json.RawMessage(`{}`)}}}
		case "tools/call":
			result = callToolResult{Content: []contentPart{{Type: "text", Text: "pong"}}}
		}
		raw, _ := json.Marshal(result)
		payload, _ := json.Marshal(rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: raw})
		_, _ = w.Write([]byte("event: message\ndata: " + string(payload) + "\n\n"))
		if flusher != nil {
			flusher.Flush()
		}
	}))
	t.Cleanup(srv.Close)

	ctx := context.Background()
	c, err := DialHTTP(ctx, ServerConfig{Name: "sse", URL: srv.URL}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	out, err := c.CallTool(ctx, "ping", nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "pong" {
		t.Fatalf("%q", out)
	}
}

func TestDialHTTP_RejectsBadScheme(t *testing.T) {
	_, err := DialHTTP(context.Background(), ServerConfig{Name: "x", URL: "file:///etc/passwd"}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestManager_HTTPAndStdio(t *testing.T) {
	// HTTP only manager path
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		var req rpcRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Method == "notifications/initialized" {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		var result any
		switch req.Method {
		case "initialize":
			result = initializeResult{ProtocolVersion: ProtocolVersion}
		case "tools/list":
			result = toolsListResult{Tools: []Tool{{Name: "t", Description: "d", InputSchema: json.RawMessage(`{}`)}}}
		case "tools/call":
			result = callToolResult{Content: []contentPart{{Type: "text", Text: "ok"}}}
		}
		raw, _ := json.Marshal(result)
		_ = json.NewEncoder(w).Encode(rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: raw})
	}))
	t.Cleanup(srv.Close)

	mgr := NewManager(context.Background(), []ServerConfig{
		{Name: "remote", URL: srv.URL, Mutating: boolPtr(false)},
		{Name: "broken", Command: ""}, // skipped
	}, nil)
	defer mgr.Close()
	if mgr.Len() != 1 {
		t.Fatalf("len=%d", mgr.Len())
	}
	q := ToolQualifiedName("remote", "t")
	out, err := mgr.Call(context.Background(), q, map[string]any{})
	if err != nil || out != "ok" {
		t.Fatalf("out=%q err=%v", out, err)
	}
}

func TestReadSSEResponse(t *testing.T) {
	body := "event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":3,\"result\":{\"ok\":true}}\n\n"
	resp, err := readSSEResponse(strings.NewReader(body), 3)
	if err != nil {
		t.Fatal(err)
	}
	if string(resp.Result) != `{"ok":true}` {
		t.Fatalf("%s", resp.Result)
	}
}
