package acp

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/iome-sh/iomesh-tui/internal/router"
)

func TestAuthorize(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/acp?token=secret", nil)
	if !authorize(r, "secret") {
		t.Fatal("query token")
	}
	r2 := httptest.NewRequest(http.MethodGet, "/acp", nil)
	r2.Header.Set("Authorization", "Bearer secret")
	if !authorize(r2, "secret") {
		t.Fatal("bearer")
	}
	r3 := httptest.NewRequest(http.MethodGet, "/acp", nil)
	if authorize(r3, "secret") {
		t.Fatal("expected deny")
	}
	if !authorize(r3, "") {
		t.Fatal("empty token allows")
	}
}

func TestRunWebSocket_InitializeSessionPrompt(t *testing.T) {
	llm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			return
		}
		_ = json.NewEncoder(w).Encode(router.ChatResponse{
			Choices: []router.Choice{{
				Message:      router.Message{Role: "assistant", Content: "ws-hi"},
				FinishReason: "stop",
			}},
			Usage: router.Usage{TotalTokens: 2},
		})
	}))
	t.Cleanup(llm.Close)

	wsDir := t.TempDir()
	cfgPath := filepath.Join(wsDir, "config.toml")
	cfgBody := `
[models]
default = "test-m"
[model.test-m]
model = "test-m"
base_url = "` + llm.URL + `"
api_key = "k"
context_window = 100000
cost_tier = 1
capabilities = ["fast", "coding"]
priority = 1
[subagents]
enabled = false
`
	if err := os.WriteFile(cfgPath, []byte(cfgBody), 0o644); err != nil {
		t.Fatal(err)
	}

	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/acp" {
			http.NotFound(w, r)
			return
		}
		if !authorize(r, "tok") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"*"}})
		if err != nil {
			t.Error(err)
			return
		}
		acpSrv := New(Options{
			ConfigPath: cfgPath,
			Workspace:  wsDir,
			Yolo:       true,
			Version:    "test",
		})
		_ = acpSrv.RunWebSocket(r.Context(), c)
		_ = c.Close(websocket.StatusNormalClosure, "")
	}))
	t.Cleanup(httpSrv.Close)

	ctx := context.Background()
	// Unauthorized HTTP
	resp, err := http.Get(httpSrv.URL + "/acp")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}

	wsURL := "ws" + strings.TrimPrefix(httpSrv.URL, "http") + "/acp?token=tok"
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	writeJSON := func(obj any) {
		t.Helper()
		b, err := json.Marshal(obj)
		if err != nil {
			t.Fatal(err)
		}
		if err := conn.Write(ctx, websocket.MessageText, b); err != nil {
			t.Fatal(err)
		}
	}
	readJSON := func() map[string]any {
		t.Helper()
		ctxR, cancel := context.WithTimeout(ctx, 8*time.Second)
		defer cancel()
		_, data, err := conn.Read(ctxR)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatalf("%v raw=%s", err, data)
		}
		return m
	}

	writeJSON(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{"protocolVersion": ProtocolVersion},
	})
	init := readJSON()
	if init["error"] != nil {
		t.Fatalf("%v", init)
	}
	if init["result"] == nil {
		t.Fatalf("%v", init)
	}

	writeJSON(map[string]any{
		"jsonrpc": "2.0", "id": 2, "method": "session/new",
		"params": map[string]any{"cwd": wsDir},
	})
	newRes := readJSON()
	if newRes["error"] != nil {
		t.Fatalf("%v", newRes)
	}
	sid, _ := newRes["result"].(map[string]any)["sessionId"].(string)
	if sid == "" {
		t.Fatalf("%v", newRes)
	}

	writeJSON(map[string]any{
		"jsonrpc": "2.0", "id": 3, "method": "session/prompt",
		"params": map[string]any{
			"sessionId": sid,
			"prompt":    []map[string]string{{"type": "text", "text": "hi"}},
		},
	})

	deadline := time.Now().Add(15 * time.Second)
	var sawResult bool
	for time.Now().Before(deadline) {
		msg := readJSON()
		if msg["id"] != nil {
			if msg["error"] != nil {
				t.Fatalf("%v", msg)
			}
			sawResult = true
			break
		}
	}
	if !sawResult {
		t.Fatal("no prompt result")
	}
}

func TestListenAndServe_HealthAndShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	_ = l.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- ListenAndServe(ctx, ServeOptions{
			Listen:         addr,
			Path:           "/acp",
			AllowAnyOrigin: true,
			Options:        Options{Version: "t", Yolo: true},
		})
	}()

	var ok bool
	for i := 0; i < 50; i++ {
		resp, err := http.Get("http://" + addr + "/healthz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				ok = true
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !ok {
		t.Fatal("healthz not ready")
	}
	resp, err := http.Get("http://" + addr + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown timeout")
	}
}
