package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/agent"
	"github.com/iome-sh/iomesh-tui/internal/router"
)

func TestREPL_ApprovesWriteOnce(t *testing.T) {
	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			return
		}
		n++
		if n == 1 {
			_ = json.NewEncoder(w).Encode(router.ChatResponse{
				Choices: []router.Choice{{
					Message: router.Message{
						Role: "assistant",
						ToolCalls: []router.ToolCall{{
							ID: "1", Type: "function",
							Function: router.FunctionCall{
								Name:      "write_file",
								Arguments: `{"path":"ok.txt","content":"approved"}`,
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
				Message:      router.Message{Role: "assistant", Content: "wrote"},
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
	rt, err := agent.New(agent.Config{Workspace: dir, Yolo: false, SubagentsEnabled: false}, rtr, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Input: user prompt, then "y" for approval, then /quit
	in := strings.NewReader("please write\ny\n/quit\n")
	var out bytes.Buffer
	ctx := context.Background()
	if err := runREPL(ctx, rt, nil, in, &out, nil); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "ok.txt"))
	if err != nil {
		t.Fatalf("file not written: %v\n%s", err, out.String())
	}
	if string(data) != "approved" {
		t.Fatalf("content=%q", data)
	}
	if !strings.Contains(out.String(), "approve tool") {
		t.Fatalf("missing prompt:\n%s", out.String())
	}
}

func TestPrintModelPicker(t *testing.T) {
	models := []router.ModelConfig{
		{Name: "a", BaseURL: "http://127.0.0.1:9", ModelID: "a", APIKey: "k", Priority: 1, MaxContext: 1, CostTier: 1, Capabilities: []string{"fast"}},
		{Name: "b", BaseURL: "http://127.0.0.1:9", ModelID: "b", APIKey: "k", Priority: 2, MaxContext: 1, CostTier: 2, Capabilities: []string{"coding"}},
	}
	rtr, err := router.New(models, "a")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	printModelPicker(&out, rtr)
	if !strings.Contains(out.String(), "1") || !strings.Contains(out.String(), "a") {
		t.Fatal(out.String())
	}
}
