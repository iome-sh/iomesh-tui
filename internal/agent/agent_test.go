package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/router"
)

func TestEstimateTokens(t *testing.T) {
	n := estimateTokens([]router.Message{{Role: "user", Content: strings.Repeat("a", 400)}})
	if n < 100 {
		t.Fatalf("n=%d", n)
	}
}

func TestTruncate(t *testing.T) {
	if truncate("hi", 10) != "hi" {
		t.Fatal()
	}
	if !strings.HasSuffix(truncate(strings.Repeat("x", 20), 5), "…") {
		t.Fatal()
	}
}

func TestRuntime_MutatingToolDeniedWithoutYolo(t *testing.T) {
	var nonStream int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Stream attempts get empty SSE so the agent falls back to non-stream JSON.
		if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			return
		}
		nonStream++
		if nonStream == 1 {
			_ = json.NewEncoder(w).Encode(router.ChatResponse{
				Choices: []router.Choice{{
					Message: router.Message{
						Role: "assistant",
						ToolCalls: []router.ToolCall{{
							ID:   "1",
							Type: "function",
							Function: router.FunctionCall{
								Name:      "write_file",
								Arguments: `{"path":"pwn.txt","content":"x"}`,
							},
						}},
					},
					FinishReason: "tool_calls",
				}},
				Usage: router.Usage{TotalTokens: 1},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(router.ChatResponse{
			Choices: []router.Choice{{
				Message:      router.Message{Role: "assistant", Content: "done"},
				FinishReason: "stop",
			}},
			Usage: router.Usage{TotalTokens: 2},
		})
	}))
	defer srv.Close()

	models := []router.ModelConfig{{
		Name: "m", BaseURL: srv.URL, ModelID: "m", APIKey: "k",
		CostTier: 1, MaxContext: 100000, Capabilities: []string{"fast", "coding"}, Priority: 1,
	}}
	rtr, err := router.New(models, "m")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	rt, err := New(Config{Workspace: dir, Yolo: false, SubagentsEnabled: false}, rtr, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var denied bool
	_, err = rt.RunTurn(context.Background(), "write something", func(ev Event) {
		if ev.Type == EventToolDenied {
			denied = true
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if !denied {
		t.Fatal("expected tool denied without yolo")
	}
	if _, err := os.Stat(filepath.Join(dir, "pwn.txt")); err == nil {
		t.Fatal("pwn.txt must not exist")
	}
}

func TestRuntime_SubagentsRegisterTools(t *testing.T) {
	models := []router.ModelConfig{{
		Name: "m", BaseURL: "http://127.0.0.1:9", ModelID: "m", APIKey: "k",
		CostTier: 1, MaxContext: 1000, Capabilities: []string{"fast"}, Priority: 1,
	}}
	rtr, err := router.New(models, "m")
	if err != nil {
		t.Fatal(err)
	}
	rt, err := New(Config{Workspace: t.TempDir(), SubagentsEnabled: true}, rtr, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if rt.Subagents() == nil || !rt.Subagents().Enabled() {
		t.Fatal("subagents should be enabled")
	}
	names := map[string]bool{}
	for _, s := range rt.tools.Schemas() {
		names[s.Function.Name] = true
	}
	if !names["spawn_subagent"] || !names["spawn_subagents"] || !names["get_subagent_output"] || !names["wait_subagents"] {
		t.Fatalf("missing subagent tools: %v", names)
	}
	if rt.Subagents().MaxConcurrent() != 16 {
		t.Fatalf("default max concurrent=%d", rt.Subagents().MaxConcurrent())
	}
}

func TestNew_RequiresRouter(t *testing.T) {
	if _, err := New(Config{Workspace: t.TempDir()}, nil, nil, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestMessagesCopy(t *testing.T) {
	models := []router.ModelConfig{{
		Name: "m", BaseURL: "http://127.0.0.1:9", ModelID: "m", APIKey: "k",
		CostTier: 1, MaxContext: 1000, Capabilities: []string{"fast"}, Priority: 1,
	}}
	rtr, _ := router.New(models, "m")
	rt, err := New(Config{Workspace: t.TempDir(), SubagentsEnabled: false}, rtr, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	msgs := rt.Messages()
	if len(msgs) < 1 || msgs[0].Role != "system" {
		t.Fatalf("%+v", msgs)
	}
	msgs[0].Content = "mutated"
	if rt.Messages()[0].Content == "mutated" {
		t.Fatal("Messages should return a copy")
	}
}
