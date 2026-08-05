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
	if !strings.Contains(out.String(), "/integrations") {
		t.Fatalf("help missing /integrations: %s", out.String())
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

// s1238: /integrations list flag parser for --layer mesh_layer filter.
func TestParseIntegrationsListArgs(t *testing.T) {
	layer, errMsg := parseIntegrationsListArgs([]string{"--layer", "knowledge"})
	if errMsg != "" || layer != "knowledge" {
		t.Fatalf("layer=%q err=%q", layer, errMsg)
	}
	layer2, errMsg2 := parseIntegrationsListArgs([]string{"--mesh-layer=analytical"})
	if errMsg2 != "" || layer2 != "analytical" {
		t.Fatalf("layer=%q err=%q", layer2, errMsg2)
	}
	// bare token form
	layer3, errMsg3 := parseIntegrationsListArgs([]string{"operational"})
	if errMsg3 != "" || layer3 != "operational" {
		t.Fatalf("layer=%q err=%q", layer3, errMsg3)
	}
	// empty defaults
	layer4, errMsg4 := parseIntegrationsListArgs(nil)
	if errMsg4 != "" || layer4 != "" {
		t.Fatalf("empty layer=%q err=%q", layer4, errMsg4)
	}
	_, bad := parseIntegrationsListArgs([]string{"--layer", "gtm"})
	if bad == "" {
		t.Fatal("expected invalid layer")
	}
	_, badFlag := parseIntegrationsListArgs([]string{"--unknown"})
	if badFlag == "" {
		t.Fatal("expected unknown flag")
	}
}

// s1238: /integrations plan arg parser for connector_id.
func TestParseIntegrationsPlanArgs(t *testing.T) {
	id, errMsg := parseIntegrationsPlanArgs([]string{"github"})
	if errMsg != "" || id != "github" {
		t.Fatalf("id=%q err=%q", id, errMsg)
	}
	id2, errMsg2 := parseIntegrationsPlanArgs([]string{"--id", "slack"})
	if errMsg2 != "" || id2 != "slack" {
		t.Fatalf("id=%q err=%q", id2, errMsg2)
	}
	id3, errMsg3 := parseIntegrationsPlanArgs([]string{"--connector-id=notion"})
	if errMsg3 != "" || id3 != "notion" {
		t.Fatalf("id=%q err=%q", id3, errMsg3)
	}
	id4, errMsg4 := parseIntegrationsPlanArgs(nil)
	if errMsg4 != "" || id4 != "" {
		t.Fatalf("empty id=%q err=%q", id4, errMsg4)
	}
	_, bad := parseIntegrationsPlanArgs([]string{"--nope"})
	if bad == "" {
		t.Fatal("expected unknown flag")
	}
	_, multi := parseIntegrationsPlanArgs([]string{"github", "extra"})
	if multi == "" {
		t.Fatal("expected unexpected argument")
	}
}

// s1238: /integrations slash routing — bare/status help + list/plan fail-open offline (no live MCP).
func TestHandleSlash_Integrations(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}

	// bare → help + honesty one-liner
	_, err := handleSlash(&out, adapter, "/integrations")
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, "usage: /integrations") {
		t.Fatalf("help: %s", s)
	}
	if !strings.Contains(s, "catalog + plan + portal HITL") {
		t.Fatalf("honesty: %s", s)
	}
	if !strings.Contains(s, "not full install CRUD") {
		t.Fatalf("not install CRUD: %s", s)
	}

	// aliases
	for _, cmd := range []string{"/integration status", "/connectors help"} {
		out.Reset()
		_, _ = handleSlash(&out, adapter, cmd)
		if !strings.Contains(out.String(), "usage: /integrations") {
			t.Fatalf("%s: %s", cmd, out.String())
		}
	}

	// list without MCP → residual-honest offline (fail-open, no invent green)
	out.Reset()
	_, _ = handleSlash(&out, adapter, "/integrations list --layer knowledge")
	listOut := out.String()
	if !strings.Contains(listOut, "fail-open") {
		t.Fatalf("list offline: %s", listOut)
	}
	if !strings.Contains(listOut, "console.iome.sh/integrations") {
		t.Fatalf("list portal: %s", listOut)
	}
	if !strings.Contains(listOut, "s1237") {
		t.Fatalf("list s1237: %s", listOut)
	}
	// must not invent a catalog table of live connectors offline
	if strings.Contains(listOut, "MESH_LAYER") && strings.Contains(listOut, "github") {
		t.Fatalf("must not invent catalog rows: %s", listOut)
	}

	// plan without MCP → same fail-open path
	out.Reset()
	_, _ = handleSlash(&out, adapter, "/integrations plan github")
	planOut := out.String()
	if !strings.Contains(planOut, "fail-open") || !strings.Contains(planOut, "s1237") {
		t.Fatalf("plan offline: %s", planOut)
	}

	// plan missing id
	out.Reset()
	_, _ = handleSlash(&out, adapter, "/integrations plan")
	if !strings.Contains(out.String(), "usage: /integrations plan") {
		t.Fatalf("plan usage: %s", out.String())
	}

	// bad list layer
	out.Reset()
	_, _ = handleSlash(&out, adapter, "/integrations list --layer gtm")
	if !strings.Contains(out.String(), "invalid") {
		t.Fatalf("bad layer: %s", out.String())
	}

	// unknown subcommand
	out.Reset()
	_, _ = handleSlash(&out, adapter, "/integrations install github")
	if !strings.Contains(out.String(), "unknown subcommand") {
		t.Fatalf("unknown sub: %s", out.String())
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
