package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/agent"
	"github.com/iome-sh/iomesh-tui/internal/router"
)

func testRuntime(t *testing.T) *agent.Runtime {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			return
		}
		_ = json.NewEncoder(w).Encode(router.ChatResponse{
			Choices: []router.Choice{{
				Message:      router.Message{Role: "assistant", Content: "pong"},
				FinishReason: "stop",
			}},
			Usage: router.Usage{TotalTokens: 3},
		})
	}))
	t.Cleanup(srv.Close)
	models := []router.ModelConfig{{
		Name: "m", BaseURL: srv.URL, ModelID: "m", APIKey: "k",
		CostTier: 1, MaxContext: 10000, Capabilities: []string{"fast", "coding"}, Priority: 1,
	}}
	rtr, err := router.New(models, "m")
	if err != nil {
		t.Fatal(err)
	}
	rt, err := agent.New(agent.Config{
		Workspace: t.TempDir(), SubagentsEnabled: true, Yolo: false,
	}, rtr, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return rt
}

func TestHandleSlash_ModelsAndCost(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}
	quit, err := handleSlash(&out, adapter, "/models")
	if quit || err != nil {
		t.Fatalf("quit=%v err=%v", quit, err)
	}
	if !strings.Contains(out.String(), "m") {
		t.Fatalf("%s", out.String())
	}
	out.Reset()
	_, _ = handleSlash(&out, adapter, "/cost")
	if !strings.Contains(out.String(), "estimate") {
		t.Fatalf("%s", out.String())
	}
	out.Reset()
	_, _ = handleSlash(&out, adapter, "/model m")
	if !strings.Contains(out.String(), "override") {
		t.Fatalf("%s", out.String())
	}
	out.Reset()
	_, _ = handleSlash(&out, adapter, "/model default")
	if !strings.Contains(out.String(), "cleared") {
		t.Fatalf("%s", out.String())
	}
	out.Reset()
	_, _ = handleSlash(&out, adapter, "/subagents")
	if !strings.Contains(out.String(), "no subagents") && !strings.Contains(out.String(), "sa-") {
		t.Fatalf("%s", out.String())
	}
	out.Reset()
	quit, _ = handleSlash(&out, adapter, "/quit")
	if !quit {
		t.Fatal("expected quit")
	}
	out.Reset()
	_, _ = handleSlash(&out, adapter, "/help")
	if !strings.Contains(out.String(), "/models") {
		t.Fatal(out.String())
	}
}

func TestRunREPL_Quit(t *testing.T) {
	rt := testRuntime(t)
	in := strings.NewReader("/quit\n")
	var out bytes.Buffer
	err := runREPL(context.Background(), rt, nil, in, &out, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "iomesh") {
		t.Fatalf("%s", out.String())
	}
}

func TestModelPickerNumber(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}
	_, _ = handleSlash(&out, adapter, "/model 1")
	if !strings.Contains(out.String(), "override") {
		t.Fatal(out.String())
	}
}

func TestTruncate(t *testing.T) {
	if truncate("hi", 10) != "hi" {
		t.Fatal()
	}
	if !strings.Contains(truncate("hello\nworld", 5), "…") {
		t.Fatal()
	}
}
