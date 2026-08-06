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
// s1281: --prefer-shorter-hops / --legacy-sort / --no-prefer-shorter-hops hop ranking flags.
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
	if o.PreferShorterHops != nil {
		t.Fatalf("PreferShorterHops must be nil when omitted: %+v", o)
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

	// s1281: bare --prefer-shorter-hops → true
	_, _, o3, err3 := parseMemoryRelatedArgs([]string{"--seed", "person:x", "--prefer-shorter-hops"})
	if err3 != "" {
		t.Fatalf("err3=%q", err3)
	}
	if o3.PreferShorterHops == nil || !*o3.PreferShorterHops {
		t.Fatalf("bare --prefer-shorter-hops want true, got %+v", o3.PreferShorterHops)
	}

	// s1281: --prefer-shorter-hops=false → false
	_, _, o4, err4 := parseMemoryRelatedArgs([]string{"--seed", "person:x", "--prefer-shorter-hops=false"})
	if err4 != "" {
		t.Fatalf("err4=%q", err4)
	}
	if o4.PreferShorterHops == nil || *o4.PreferShorterHops {
		t.Fatalf("prefer-shorter-hops=false want false, got %+v", o4.PreferShorterHops)
	}

	// s1281: --prefer_shorter_hops true (space-separated)
	_, _, o5, err5 := parseMemoryRelatedArgs([]string{"--seed", "person:x", "--prefer_shorter_hops", "true"})
	if err5 != "" {
		t.Fatalf("err5=%q", err5)
	}
	if o5.PreferShorterHops == nil || !*o5.PreferShorterHops {
		t.Fatalf("prefer_shorter_hops true want true, got %+v", o5.PreferShorterHops)
	}

	// s1281: --legacy-sort → false
	_, _, o6, err6 := parseMemoryRelatedArgs([]string{"--seed", "person:x", "--legacy-sort"})
	if err6 != "" {
		t.Fatalf("err6=%q", err6)
	}
	if o6.PreferShorterHops == nil || *o6.PreferShorterHops {
		t.Fatalf("--legacy-sort want false, got %+v", o6.PreferShorterHops)
	}

	// s1281: --no-prefer-shorter-hops → false
	_, _, o7, err7 := parseMemoryRelatedArgs([]string{"--seed", "person:x", "--no-prefer-shorter-hops"})
	if err7 != "" {
		t.Fatalf("err7=%q", err7)
	}
	if o7.PreferShorterHops == nil || *o7.PreferShorterHops {
		t.Fatalf("--no-prefer-shorter-hops want false, got %+v", o7.PreferShorterHops)
	}

	// s1281: invalid bool value
	_, _, _, badBool := parseMemoryRelatedArgs([]string{"--prefer-shorter-hops=maybe"})
	if badBool == "" {
		t.Fatal("expected invalid --prefer-shorter-hops")
	}

	// Bare flag must not swallow free query tokens that are not bool-like.
	seed8, q8, o8, err8 := parseMemoryRelatedArgs([]string{
		"--seed", "person:x",
		"--prefer-shorter-hops",
		"free", "query",
	})
	if err8 != "" || seed8 != "person:x" || q8 != "free query" {
		t.Fatalf("seed=%q q=%q err=%q", seed8, q8, err8)
	}
	if o8.PreferShorterHops == nil || !*o8.PreferShorterHops {
		t.Fatalf("bare flag + free query: PreferShorterHops=%+v", o8.PreferShorterHops)
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

// s1282: /memory supersede flag parser for --entity / --as-of / --i-confirm HITL.
func TestParseMemorySupersedeArgs(t *testing.T) {
	o, errMsg := parseMemorySupersedeArgs([]string{
		"--entity", "person:alice",
		"--as-of", "2026-08-04T12:00:00Z",
		"--i-confirm",
	})
	if errMsg != "" {
		t.Fatalf("errMsg=%q", errMsg)
	}
	if o.Entity != "person:alice" || o.AsOf != "2026-08-04T12:00:00Z" || !o.Confirm {
		t.Fatalf("opts=%+v", o)
	}
	// --entity= / --as_of= / --confirm aliases.
	o2, errMsg2 := parseMemorySupersedeArgs([]string{
		"--entity=org:acme",
		"--as_of=2026-01-01T00:00:00Z",
		"--confirm",
	})
	if errMsg2 != "" {
		t.Fatalf("errMsg=%q", errMsg2)
	}
	if o2.Entity != "org:acme" || o2.AsOf != "2026-01-01T00:00:00Z" || !o2.Confirm {
		t.Fatalf("opts=%+v", o2)
	}
	// --yes alias.
	oYes, errYes := parseMemorySupersedeArgs([]string{"-e", "x", "--yes"})
	if errYes != "" || !oYes.Confirm || oYes.Entity != "x" {
		t.Fatalf("yes opts=%+v err=%q", oYes, errYes)
	}
	// Missing confirm still parses cleanly with Confirm=false (HITL gate is MemorySupersede).
	o3, errMsg3 := parseMemorySupersedeArgs([]string{"--entity", "person:bob"})
	if errMsg3 != "" {
		t.Fatalf("errMsg=%q", errMsg3)
	}
	if o3.Entity != "person:bob" || o3.Confirm {
		t.Fatalf("missing confirm must parse Confirm=false: %+v", o3)
	}
	// Empty args.
	o4, errMsg4 := parseMemorySupersedeArgs(nil)
	if errMsg4 != "" || o4.Entity != "" || o4.Confirm {
		t.Fatalf("empty opts=%+v err=%q", o4, errMsg4)
	}
	// Unknown flag rejected.
	_, badFlag := parseMemorySupersedeArgs([]string{"--entity", "x", "--unknown"})
	if badFlag == "" {
		t.Fatal("expected unknown flag")
	}
	// Unexpected bare argument rejected.
	_, badArg := parseMemorySupersedeArgs([]string{"bare-entity"})
	if badArg == "" {
		t.Fatal("expected unexpected argument")
	}
}

// s1287: /memory patterns flag parser for --limit.
func TestParseMemoryPatternsArgs(t *testing.T) {
	o, errMsg := parseMemoryPatternsArgs([]string{"--limit", "5"})
	if errMsg != "" {
		t.Fatalf("errMsg=%q", errMsg)
	}
	if o.Limit != 5 {
		t.Fatalf("opts=%+v", o)
	}
	// --limit= form.
	o2, errMsg2 := parseMemoryPatternsArgs([]string{"--limit=12"})
	if errMsg2 != "" || o2.Limit != 12 {
		t.Fatalf("opts=%+v err=%q", o2, errMsg2)
	}
	// Empty args: Limit 0 (caller uses config default).
	o3, errMsg3 := parseMemoryPatternsArgs(nil)
	if errMsg3 != "" || o3.Limit != 0 {
		t.Fatalf("empty opts=%+v err=%q", o3, errMsg3)
	}
	// Invalid limit.
	_, badLim := parseMemoryPatternsArgs([]string{"--limit", "nope"})
	if badLim == "" {
		t.Fatal("expected invalid --limit")
	}
	_, badNeg := parseMemoryPatternsArgs([]string{"--limit", "-1"})
	if badNeg == "" {
		t.Fatal("expected invalid --limit for negative")
	}
	// Unknown flag rejected.
	_, badFlag := parseMemoryPatternsArgs([]string{"--unknown"})
	if badFlag == "" {
		t.Fatal("expected unknown flag")
	}
	// Bare arg rejected.
	_, badArg := parseMemoryPatternsArgs([]string{"bare"})
	if badArg == "" {
		t.Fatal("expected unexpected argument")
	}
}

// s1287: /memory anomalies flag parser for --limit.
func TestParseMemoryAnomaliesArgs(t *testing.T) {
	o, errMsg := parseMemoryAnomaliesArgs([]string{"--limit", "3"})
	if errMsg != "" {
		t.Fatalf("errMsg=%q", errMsg)
	}
	if o.Limit != 3 {
		t.Fatalf("opts=%+v", o)
	}
	o2, errMsg2 := parseMemoryAnomaliesArgs([]string{"--limit=8"})
	if errMsg2 != "" || o2.Limit != 8 {
		t.Fatalf("opts=%+v err=%q", o2, errMsg2)
	}
	o3, errMsg3 := parseMemoryAnomaliesArgs(nil)
	if errMsg3 != "" || o3.Limit != 0 {
		t.Fatalf("empty opts=%+v err=%q", o3, errMsg3)
	}
	_, badLim := parseMemoryAnomaliesArgs([]string{"--limit", "nope"})
	if badLim == "" {
		t.Fatal("expected invalid --limit")
	}
	_, badFlag := parseMemoryAnomaliesArgs([]string{"--window", "day"})
	if badFlag == "" {
		t.Fatal("expected unknown flag")
	}
	_, badArg := parseMemoryAnomaliesArgs([]string{"extra"})
	if badArg == "" {
		t.Fatal("expected unexpected argument")
	}
}

// s1296: /memory timeline flag parser for --since/--until/--session-id/--query/--limit.
func TestParseMemoryTimelineArgs(t *testing.T) {
	o, errMsg := parseMemoryTimelineArgs([]string{
		"--since", "2026-08-01T00:00:00Z",
		"--until", "2026-08-04T12:00:00Z",
		"--session-id", "sess-1",
		"--query", "deploy",
		"--limit", "5",
	})
	if errMsg != "" {
		t.Fatalf("errMsg=%q", errMsg)
	}
	if o.Since != "2026-08-01T00:00:00Z" || o.Until != "2026-08-04T12:00:00Z" {
		t.Fatalf("opts=%+v", o)
	}
	if o.SessionID != "sess-1" || o.Query != "deploy" || o.Limit != 5 {
		t.Fatalf("opts=%+v", o)
	}
	// --session_id= form + free tokens as query + --limit=.
	o2, errMsg2 := parseMemoryTimelineArgs([]string{
		"--since=2026-01-01T00:00:00Z",
		"--session_id=sess-2",
		"--limit=12",
		"free", "query", "words",
	})
	if errMsg2 != "" {
		t.Fatalf("errMsg=%q", errMsg2)
	}
	if o2.Since != "2026-01-01T00:00:00Z" || o2.SessionID != "sess-2" || o2.Limit != 12 {
		t.Fatalf("opts=%+v", o2)
	}
	if o2.Query != "free query words" {
		t.Fatalf("query=%q", o2.Query)
	}
	// Empty args: all zero (caller uses defaults).
	o3, errMsg3 := parseMemoryTimelineArgs(nil)
	if errMsg3 != "" || o3.Limit != 0 || o3.Query != "" {
		t.Fatalf("empty opts=%+v err=%q", o3, errMsg3)
	}
	// Invalid limit.
	_, badLim := parseMemoryTimelineArgs([]string{"--limit", "nope"})
	if badLim == "" {
		t.Fatal("expected invalid --limit")
	}
	_, badNeg := parseMemoryTimelineArgs([]string{"--limit", "-1"})
	if badNeg == "" {
		t.Fatal("expected invalid --limit for negative")
	}
	// Unknown flag rejected.
	_, badFlag := parseMemoryTimelineArgs([]string{"--unknown"})
	if badFlag == "" {
		t.Fatal("expected unknown flag")
	}
}

// s1276: /memory facts-as-of flag parser for --as-of / --entity / --query / --limit.
func TestParseMemoryFactsAsOfArgs(t *testing.T) {
	o, errMsg := parseMemoryFactsAsOfArgs([]string{
		"--as-of", "2026-08-04T12:00:00Z",
		"--entity", "person:alice",
		"--query", "role",
		"--limit", "5",
		"--session-id=sess-1",
	})
	if errMsg != "" {
		t.Fatalf("errMsg=%q", errMsg)
	}
	if o.AsOf != "2026-08-04T12:00:00Z" || o.Entity != "person:alice" {
		t.Fatalf("opts=%+v", o)
	}
	if o.Query != "role" || o.Limit != 5 || o.SessionID != "sess-1" {
		t.Fatalf("opts=%+v", o)
	}
	// --as_of= form + free tokens as query.
	o2, errMsg2 := parseMemoryFactsAsOfArgs([]string{
		"--as_of=2026-01-01T00:00:00Z",
		"--entity=org:acme",
		"free", "query", "words",
	})
	if errMsg2 != "" {
		t.Fatalf("errMsg=%q", errMsg2)
	}
	if o2.AsOf != "2026-01-01T00:00:00Z" || o2.Entity != "org:acme" {
		t.Fatalf("opts=%+v", o2)
	}
	if o2.Query != "free query words" {
		t.Fatalf("query=%q", o2.Query)
	}
	_, badLim := parseMemoryFactsAsOfArgs([]string{"--as-of", "2026-08-04T12:00:00Z", "--limit", "nope"})
	if badLim == "" {
		t.Fatal("expected invalid --limit")
	}
	_, badFlag := parseMemoryFactsAsOfArgs([]string{"--as-of", "2026-08-04T12:00:00Z", "--unknown"})
	if badFlag == "" {
		t.Fatal("expected unknown flag")
	}
	// Empty args: AsOf empty (caller enforces required).
	o3, errMsg3 := parseMemoryFactsAsOfArgs(nil)
	if errMsg3 != "" || o3.AsOf != "" {
		t.Fatalf("empty opts=%+v err=%q", o3, errMsg3)
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

// s1243: /integrations signing arg parser for mesh_layer or connector_id.
func TestParseIntegrationsSigningArgs(t *testing.T) {
	hint, errMsg := parseIntegrationsSigningArgs([]string{"knowledge"})
	if errMsg != "" || hint != "knowledge" {
		t.Fatalf("hint=%q err=%q", hint, errMsg)
	}
	hint2, errMsg2 := parseIntegrationsSigningArgs([]string{"--layer", "operational"})
	if errMsg2 != "" || hint2 != "operational" {
		t.Fatalf("hint=%q err=%q", hint2, errMsg2)
	}
	hint3, errMsg3 := parseIntegrationsSigningArgs([]string{"--id=github"})
	if errMsg3 != "" || hint3 != "github" {
		t.Fatalf("hint=%q err=%q", hint3, errMsg3)
	}
	// bare connector id (not a mesh layer)
	hint4, errMsg4 := parseIntegrationsSigningArgs([]string{"slack"})
	if errMsg4 != "" || hint4 != "slack" {
		t.Fatalf("hint=%q err=%q", hint4, errMsg4)
	}
	hint5, errMsg5 := parseIntegrationsSigningArgs(nil)
	if errMsg5 != "" || hint5 != "" {
		t.Fatalf("empty hint=%q err=%q", hint5, errMsg5)
	}
	_, bad := parseIntegrationsSigningArgs([]string{"--layer", "gtm"})
	if bad == "" {
		t.Fatal("expected invalid layer")
	}
	_, badFlag := parseIntegrationsSigningArgs([]string{"--unknown"})
	if badFlag == "" {
		t.Fatal("expected unknown flag")
	}
	_, multi := parseIntegrationsSigningArgs([]string{"github", "extra"})
	if multi == "" {
		t.Fatal("expected unexpected argument")
	}
}

// s1238/s1247: /integrations slash routing — bare/help + status pulse + list/plan fail-open offline.
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
	// help describes status as operator pulse (s1247), not "this help"
	if !strings.Contains(s, "operator pulse") {
		t.Fatalf("status help line: %s", s)
	}

	// help/? still pure help
	out.Reset()
	_, _ = handleSlash(&out, adapter, "/connectors help")
	if !strings.Contains(out.String(), "usage: /integrations") {
		t.Fatalf("help alias: %s", out.String())
	}

	// s1247: status ≠ pure help only — residual-honest operator pulse (offline fail-open)
	out.Reset()
	_, _ = handleSlash(&out, adapter, "/integration status")
	statusOut := out.String()
	if strings.Contains(statusOut, "usage: /integrations") && !strings.Contains(statusOut, "operator pulse") {
		t.Fatalf("status must not be pure help only: %s", statusOut)
	}
	if !strings.Contains(statusOut, "s1247") || !strings.Contains(statusOut, "operator pulse") {
		t.Fatalf("status pulse: %s", statusOut)
	}
	if !strings.Contains(statusOut, "MCP path:") {
		t.Fatalf("status MCP path: %s", statusOut)
	}
	if !strings.Contains(statusOut, "list_connector_catalog") {
		t.Fatalf("status tools: %s", statusOut)
	}
	if !strings.Contains(statusOut, "fail-open") || !strings.Contains(statusOut, "offline") {
		t.Fatalf("status offline residual: %s", statusOut)
	}
	if !strings.Contains(statusOut, "never invent install green") {
		t.Fatalf("status honesty: %s", statusOut)
	}
	// st alias
	out.Reset()
	_, _ = handleSlash(&out, adapter, "/integrations st")
	if !strings.Contains(out.String(), "operator pulse") {
		t.Fatalf("st alias: %s", out.String())
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

	// s1243: signing without MCP → fail-open offline (discovery only; no invent secrets)
	out.Reset()
	_, _ = handleSlash(&out, adapter, "/integrations signing knowledge")
	signOut := out.String()
	if !strings.Contains(signOut, "fail-open") {
		t.Fatalf("signing offline: %s", signOut)
	}
	if !strings.Contains(signOut, "console.iome.sh/integrations") {
		t.Fatalf("signing portal: %s", signOut)
	}
	// alias headers
	out.Reset()
	_, _ = handleSlash(&out, adapter, "/integrations headers github")
	if !strings.Contains(out.String(), "fail-open") {
		t.Fatalf("headers alias: %s", out.String())
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
