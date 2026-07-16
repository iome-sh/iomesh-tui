package mcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestResourcesAndPrompts_HTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			result = toolsListResult{Tools: []Tool{{Name: "noop", Description: "n", InputSchema: json.RawMessage(`{}`)}}}
		case "resources/list":
			result = resourcesListResult{Resources: []Resource{
				{URI: "file:///readme", Name: "readme", Description: "README"},
			}}
		case "resources/read":
			result = resourcesReadResult{Contents: []contentPart{{Type: "text", Text: "# Hello resource"}}}
		case "prompts/list":
			result = promptsListResult{Prompts: []Prompt{
				{Name: "review", Description: "Code review"},
			}}
		case "prompts/get":
			raw, _ := json.Marshal("Please review carefully.")
			result = promptsGetResult{
				Description: "review",
				Messages:    []promptMessage{{Role: "user", Content: raw}},
			}
		case "tools/call":
			result = callToolResult{Content: []contentPart{{Type: "text", Text: "ok"}}}
		default:
			_ = json.NewEncoder(w).Encode(rpcResponse{
				JSONRPC: "2.0", ID: req.ID,
				Error: &rpcError{Code: -32601, Message: "unknown " + req.Method},
			})
			return
		}
		raw, _ := json.Marshal(result)
		_ = json.NewEncoder(w).Encode(rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: raw})
	}))
	t.Cleanup(srv.Close)

	ctx := context.Background()
	c, err := DialHTTP(ctx, ServerConfig{Name: "r", URL: srv.URL, Mutating: boolPtr(false)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	if len(c.Resources()) != 1 || c.Resources()[0].URI != "file:///readme" {
		t.Fatalf("%+v", c.Resources())
	}
	text, err := c.ReadResource(ctx, "file:///readme")
	if err != nil || !strings.Contains(text, "Hello resource") {
		t.Fatalf("%q %v", text, err)
	}
	if len(c.Prompts()) != 1 {
		t.Fatalf("%+v", c.Prompts())
	}
	pt, err := c.GetPrompt(ctx, "review", nil)
	if err != nil || !strings.Contains(pt, "review carefully") {
		t.Fatalf("%q %v", pt, err)
	}

	mgr := NewManagerEmpty(nil)
	mgr.Attach(c)
	out, err := mgr.ListAllResources(ctx, "", false)
	if err != nil || !strings.Contains(out, "file:///readme") {
		t.Fatalf("%q %v", out, err)
	}
	out, err = mgr.ListAllPrompts(ctx, "r", false)
	if err != nil || !strings.Contains(out, "review") {
		t.Fatalf("%q %v", out, err)
	}
}

func TestOAuth_StaticTokenEnv(t *testing.T) {
	var sawAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		var req rpcRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Method == "notifications/initialized" {
			w.WriteHeader(202)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		var result any
		switch req.Method {
		case "initialize":
			result = initializeResult{ProtocolVersion: ProtocolVersion}
		case "tools/list":
			result = toolsListResult{Tools: nil}
		case "resources/list", "prompts/list":
			_ = json.NewEncoder(w).Encode(rpcResponse{
				JSONRPC: "2.0", ID: req.ID,
				Error: &rpcError{Code: -32601, Message: "nope"},
			})
			return
		}
		raw, _ := json.Marshal(result)
		_ = json.NewEncoder(w).Encode(rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: raw})
	}))
	t.Cleanup(srv.Close)

	t.Setenv("MCP_TEST_TOKEN", "secret-tok")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c, err := DialHTTP(ctx, ServerConfig{
		Name: "auth", URL: srv.URL, AccessTokenEnv: "MCP_TEST_TOKEN",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if sawAuth != "Bearer secret-tok" {
		t.Fatalf("auth=%q", sawAuth)
	}
}

func TestOAuth_ClientCredentials(t *testing.T) {
	var tokenHits, mcpHits int
	auth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenHits++
		b, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(b), "client_credentials") {
			http.Error(w, "bad grant", 400)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "cc-token",
			"expires_in":   3600,
			"token_type":   "Bearer",
		})
	}))
	t.Cleanup(auth.Close)

	mcpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mcpHits++
		if r.Header.Get("Authorization") != "Bearer cc-token" {
			http.Error(w, "unauthorized", 401)
			return
		}
		var req rpcRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Method == "notifications/initialized" {
			w.WriteHeader(202)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		var result any
		switch req.Method {
		case "initialize":
			result = initializeResult{ProtocolVersion: ProtocolVersion}
		case "tools/list":
			result = toolsListResult{}
		default:
			_ = json.NewEncoder(w).Encode(rpcResponse{
				JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32601, Message: "skip"},
			})
			return
		}
		raw, _ := json.Marshal(result)
		_ = json.NewEncoder(w).Encode(rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: raw})
	}))
	t.Cleanup(mcpSrv.Close)

	t.Setenv("MCP_CC_SECRET", "shh")
	ctx := context.Background()
	c, err := DialHTTP(ctx, ServerConfig{
		Name: "cc", URL: mcpSrv.URL,
		OAuth: &OAuthConfig{
			TokenURL:        auth.URL,
			ClientID:        "cid",
			ClientSecretEnv: "MCP_CC_SECRET",
			Scopes:          []string{"mcp"},
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if tokenHits < 1 || mcpHits < 1 {
		t.Fatalf("tokenHits=%d mcpHits=%d", tokenHits, mcpHits)
	}
}

func TestPKCE(t *testing.T) {
	v, err := NewPKCEVerifier()
	if err != nil || len(v) < 40 {
		t.Fatalf("%q %v", v, err)
	}
	ch, method := PKCEChallenge(v)
	if method != "S256" || ch == "" || ch == v {
		t.Fatalf("%s %s", ch, method)
	}
}
