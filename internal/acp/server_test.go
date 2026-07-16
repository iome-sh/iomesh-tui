package acp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/router"
)

func TestACP_InitializeSessionPrompt(t *testing.T) {
	// Fake LLM: empty stream then non-stream text; tool call for spawn visibility optional.
	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			return
		}
		n++
		if n == 1 {
			// Ask for list_dir tool to exercise tool_call stream
			_ = json.NewEncoder(w).Encode(router.ChatResponse{
				Choices: []router.Choice{{
					Message: router.Message{
						Role: "assistant",
						ToolCalls: []router.ToolCall{{
							ID: "1", Type: "function",
							Function: router.FunctionCall{Name: "list_dir", Arguments: `{"path":"."}`},
						}},
					},
					FinishReason: "tool_calls",
				}},
				Usage: router.Usage{TotalTokens: 5},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(router.ChatResponse{
			Choices: []router.Choice{{
				Message:      router.Message{Role: "assistant", Content: "hello-acp"},
				FinishReason: "stop",
			}},
			Usage: router.Usage{TotalTokens: 3},
		})
	}))
	t.Cleanup(srv.Close)

	// Workspace + config with test model
	ws := t.TempDir()
	cfgPath := filepath.Join(ws, "config.toml")
	cfgBody := `
[models]
default = "test-m"

[model.test-m]
model = "test-m"
base_url = "` + srv.URL + `"
api_key = "k"
context_window = 100000
cost_tier = 1
capabilities = ["fast", "coding"]
priority = 1

[subagents]
enabled = true
`
	if err := os.WriteFile(cfgPath, []byte(cfgBody), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	s := New(Options{
		ConfigPath: cfgPath,
		Workspace:  ws,
		Yolo:       true,
		Version:    "test",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- s.Run(ctx, inR, &out)
	}()

	write := func(obj any) {
		b, _ := json.Marshal(obj)
		_, _ = inW.Write(append(b, '\n'))
	}

	write(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{"protocolVersion": ProtocolVersion},
	})
	write(map[string]any{
		"jsonrpc": "2.0", "id": 2, "method": "session/new",
		"params": map[string]any{"cwd": ws},
	})

	// Read until we get session id
	sc := bufio.NewScanner(strings.NewReader(""))
	// Poll out buffer
	var sessionID string
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && sessionID == "" {
		lines := strings.Split(out.String(), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			var resp map[string]any
			if json.Unmarshal([]byte(line), &resp) != nil {
				continue
			}
			if result, ok := resp["result"].(map[string]any); ok {
				if sid, ok := result["sessionId"].(string); ok {
					sessionID = sid
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if sessionID == "" {
		t.Fatalf("no session id in output: %s", out.String())
	}

	write(map[string]any{
		"jsonrpc": "2.0", "id": 3, "method": "session/prompt",
		"params": map[string]any{
			"sessionId": sessionID,
			"prompt":    []map[string]string{{"type": "text", "text": "list files"}},
		},
	})

	deadline = time.Now().Add(5 * time.Second)
	var sawText, sawTool, sawEnd bool
	for time.Now().Before(deadline) {
		text := out.String()
		if strings.Contains(text, "hello-acp") || strings.Contains(text, "agent_message_chunk") {
			sawText = true
		}
		if strings.Contains(text, "tool_call") || strings.Contains(text, "list_dir") {
			sawTool = true
		}
		if strings.Contains(text, "end_turn") {
			sawEnd = true
		}
		if sawText && sawTool && sawEnd {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	_ = inW.Close()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}

	if !sawEnd {
		t.Fatalf("missing end_turn in:\n%s", out.String())
	}
	if !sawTool {
		t.Fatalf("missing tool stream in:\n%s", out.String())
	}
	if !sawText {
		t.Fatalf("missing message chunk in:\n%s", out.String())
	}
	_ = sc
}

func TestJoinPrompt(t *testing.T) {
	got := joinPrompt([]promptContent{{Type: "text", Text: "a"}, {Text: "b"}})
	if got != "ab" {
		t.Fatal(got)
	}
}

func TestToolKind_Subagents(t *testing.T) {
	if toolKind("spawn_subagents") != "other" {
		t.Fatal()
	}
	if toolKind("apply_worktree") != "edit" {
		t.Fatal()
	}
	if toolKind("read_file") != "read" {
		t.Fatal()
	}
}
