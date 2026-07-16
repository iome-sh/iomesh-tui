package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/router"
)

func TestApprover_OnceAlwaysDeny(t *testing.T) {
	var phase atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			return
		}
		p := phase.Add(1)
		// Odd calls: request write_file; even: finish.
		if p%2 == 1 {
			_ = json.NewEncoder(w).Encode(router.ChatResponse{
				Choices: []router.Choice{{
					Message: router.Message{
						Role: "assistant",
						ToolCalls: []router.ToolCall{{
							ID: "1", Type: "function",
							Function: router.FunctionCall{
								Name:      "write_file",
								Arguments: `{"path":"x.txt","content":"hi"}`,
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
			Usage: router.Usage{TotalTokens: 1},
		})
	}))
	t.Cleanup(srv.Close)

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

	// Deny
	rt.SetApprover(func(ctx context.Context, tool, args string) (Approval, error) {
		return ApprovalDeny, nil
	})
	var denied bool
	_, _ = rt.RunTurn(context.Background(), "write", func(ev Event) {
		if ev.Type == EventToolDenied {
			denied = true
		}
	})
	if !denied {
		t.Fatal("expected deny")
	}
	if _, err := os.Stat(filepath.Join(dir, "x.txt")); err == nil {
		t.Fatal("file should not exist after deny")
	}

	// Once
	phase.Store(0)
	rt.SetApprover(func(ctx context.Context, tool, args string) (Approval, error) {
		return ApprovalOnce, nil
	})
	_, err = rt.RunTurn(context.Background(), "write again", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "x.txt")); err != nil {
		t.Fatal("file should exist after once")
	}

	// Always then second write without prompt
	phase.Store(0)
	var prompts atomic.Int32
	rt.SetApprover(func(ctx context.Context, tool, args string) (Approval, error) {
		prompts.Add(1)
		return ApprovalAlways, nil
	})
	_ = os.Remove(filepath.Join(dir, "x.txt"))
	_, _ = rt.RunTurn(context.Background(), "write always", nil)
	if prompts.Load() != 1 {
		t.Fatalf("prompts=%d", prompts.Load())
	}
	if !rt.ToolAllowedSession("write_file") {
		t.Fatal("always not stored")
	}
	phase.Store(0)
	prompts.Store(0)
	_ = os.Remove(filepath.Join(dir, "x.txt"))
	// Change content path still write_file
	_, _ = rt.RunTurn(context.Background(), "write again always", nil)
	if prompts.Load() != 0 {
		t.Fatalf("should not prompt after always, prompts=%d", prompts.Load())
	}
}
