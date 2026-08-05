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
	if !strings.Contains(out.String(), "/memory") {
		t.Fatalf("help missing /memory: %s", out.String())
	}
	out.Reset()
	_, _ = handleSlash(&out, adapter, "/memory")
	if !strings.Contains(out.String(), "memory:") {
		t.Fatalf("memory status: %q", out.String())
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

// s1068: /memory recall flag parser for temporal since/until/session_seq.
func TestParseMemoryRecallArgs(t *testing.T) {
	q, o := parseMemoryRecallArgs([]string{
		"--since", "2026-07-01T00:00:00Z",
		"--until=2026-07-31T23:59:59Z",
		"--session-seq", "3",
		"what", "did", "we", "decide",
	})
	if q != "what did we decide" {
		t.Fatalf("query=%q", q)
	}
	if o.Since != "2026-07-01T00:00:00Z" || o.Until != "2026-07-31T23:59:59Z" {
		t.Fatalf("opts=%+v", o)
	}
	if !o.SessionSeqSet || o.SessionSeq != 3 {
		t.Fatalf("session_seq=%+v", o)
	}
	q2, o2 := parseMemoryRecallArgs([]string{"plain", "query"})
	if q2 != "plain query" || o2.Since != "" || o2.SessionSeqSet {
		t.Fatalf("plain q=%q opts=%+v", q2, o2)
	}
}

// s1135: /memory related flag parser for seed / query / max-hops.
func TestParseMemoryRelatedArgs(t *testing.T) {
	seed, q, o, errMsg := parseMemoryRelatedArgs([]string{
		"--seed", "person:alice",
		"--query", "teammate notes",
		"--max-hops", "2",
		"--limit=5",
	})
	if errMsg != "" {
		t.Fatalf("errMsg=%q", errMsg)
	}
	if seed != "person:alice" || q != "teammate notes" {
		t.Fatalf("seed=%q q=%q", seed, q)
	}
	if o.MaxHops != 2 || o.Limit != 5 {
		t.Fatalf("opts=%+v", o)
	}
	// --seed= form + free tokens as query.
	seed2, q2, o2, errMsg2 := parseMemoryRelatedArgs([]string{
		"--seed-entity=org:acme",
		"free", "query", "words",
	})
	if errMsg2 != "" || seed2 != "org:acme" || q2 != "free query words" {
		t.Fatalf("seed=%q q=%q err=%q", seed2, q2, errMsg2)
	}
	if o2.MaxHops != 0 {
		t.Fatalf("opts=%+v", o2)
	}
	_, _, _, bad := parseMemoryRelatedArgs([]string{"--max-hops", "nope"})
	if bad == "" {
		t.Fatal("expected invalid --max-hops")
	}
}

// s1200: /memory digest flag parser for window / horizon / limit.
func TestParseMemoryDigestArgs(t *testing.T) {
	o, errMsg := parseMemoryDigestArgs([]string{
		"--window", "week",
		"--horizon=knowledge",
		"--limit", "7",
		"--as-of", "2026-08-04T00:00:00Z",
	})
	if errMsg != "" {
		t.Fatalf("errMsg=%q", errMsg)
	}
	if o.Window != "week" || o.Horizon != "knowledge" || o.Limit != 7 {
		t.Fatalf("opts=%+v", o)
	}
	if o.AsOf != "2026-08-04T00:00:00Z" {
		t.Fatalf("as_of=%q", o.AsOf)
	}
	// defaults when empty.
	o2, errMsg2 := parseMemoryDigestArgs(nil)
	if errMsg2 != "" || o2.Window != "" || o2.Horizon != "" || o2.Limit != 0 {
		t.Fatalf("empty opts=%+v err=%q", o2, errMsg2)
	}
	_, badWin := parseMemoryDigestArgs([]string{"--window", "month"})
	if badWin == "" {
		t.Fatal("expected invalid --window")
	}
	_, badHor := parseMemoryDigestArgs([]string{"--horizon", "gtm"})
	if badHor == "" {
		t.Fatal("expected invalid --horizon")
	}
	_, badLim := parseMemoryDigestArgs([]string{"--limit", "nope"})
	if badLim == "" {
		t.Fatal("expected invalid --limit")
	}
	_, badFlag := parseMemoryDigestArgs([]string{"--unknown"})
	if badFlag == "" {
		t.Fatal("expected unknown flag")
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
