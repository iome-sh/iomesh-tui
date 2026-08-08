package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/agent"
	"github.com/iome-sh/iomesh-tui/internal/agentplugins"
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
	if !strings.Contains(out.String(), "/gtm") {
		t.Fatalf("help missing /gtm: %s", out.String())
	}
	if !strings.Contains(out.String(), "/onboard") {
		t.Fatalf("help missing /onboard: %s", out.String())
	}
	if !strings.Contains(out.String(), "/plugins") {
		t.Fatalf("help missing /plugins: %s", out.String())
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

// s1311: /memory trigger-compact HITL flag parser.
func TestParseMemoryTriggerCompactArgs(t *testing.T) {
	o, errMsg := parseMemoryTriggerCompactArgs([]string{"--i-confirm"})
	if errMsg != "" {
		t.Fatalf("errMsg=%q", errMsg)
	}
	if !o.Confirm {
		t.Fatalf("want Confirm=true, got %+v", o)
	}
	o2, err2 := parseMemoryTriggerCompactArgs([]string{"--confirm"})
	if err2 != "" || !o2.Confirm {
		t.Fatalf("confirm: %+v err=%q", o2, err2)
	}
	oYes, errYes := parseMemoryTriggerCompactArgs([]string{"--yes"})
	if errYes != "" || !oYes.Confirm {
		t.Fatalf("yes: %+v err=%q", oYes, errYes)
	}
	// Missing confirm still parses cleanly with Confirm=false (HITL gate is MemoryTriggerCompact).
	o3, err3 := parseMemoryTriggerCompactArgs(nil)
	if err3 != "" || o3.Confirm {
		t.Fatalf("nil: %+v err=%q", o3, err3)
	}
	_, badFlag := parseMemoryTriggerCompactArgs([]string{"--unknown"})
	if badFlag == "" {
		t.Fatal("expected unknown flag")
	}
	_, badArg := parseMemoryTriggerCompactArgs([]string{"bare"})
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

// s1301: /memory semantic flag parser for --query/--limit + free tokens.
func TestParseMemorySemanticArgs(t *testing.T) {
	o, errMsg := parseMemorySemanticArgs([]string{"--query", "alice role", "--limit", "5"})
	if errMsg != "" {
		t.Fatalf("errMsg=%q", errMsg)
	}
	if o.Query != "alice role" || o.Limit != 5 {
		t.Fatalf("opts=%+v", o)
	}
	// Free tokens + --limit= form.
	o2, errMsg2 := parseMemorySemanticArgs([]string{"--limit=12", "deploy", "window"})
	if errMsg2 != "" {
		t.Fatalf("errMsg=%q", errMsg2)
	}
	if o2.Query != "deploy window" || o2.Limit != 12 {
		t.Fatalf("opts=%+v", o2)
	}
	// -q alias.
	o3, errMsg3 := parseMemorySemanticArgs([]string{"-q", "sem"})
	if errMsg3 != "" || o3.Query != "sem" {
		t.Fatalf("opts=%+v err=%q", o3, errMsg3)
	}
	// Empty args: zero query (caller shows usage).
	o4, errMsg4 := parseMemorySemanticArgs(nil)
	if errMsg4 != "" || o4.Query != "" || o4.Limit != 0 {
		t.Fatalf("empty opts=%+v err=%q", o4, errMsg4)
	}
	// Invalid limit.
	_, badLim := parseMemorySemanticArgs([]string{"--limit", "nope"})
	if badLim == "" {
		t.Fatal("expected invalid --limit")
	}
	_, badNeg := parseMemorySemanticArgs([]string{"--limit", "-1"})
	if badNeg == "" {
		t.Fatal("expected invalid --limit for negative")
	}
	// Unknown flag rejected.
	_, badFlag := parseMemorySemanticArgs([]string{"--unknown"})
	if badFlag == "" {
		t.Fatal("expected unknown flag")
	}
}

// s1301: /memory ingest-event flag parser for subject/content + optional event flags.
func TestParseMemoryIngestEventArgs(t *testing.T) {
	o, errMsg := parseMemoryIngestEventArgs([]string{
		"--subject", "dept.research.events.github.pr_opened",
		"--content", "PR #12 opened",
		"--event-time", "2026-08-06T02:00:00Z",
		"--session-id", "sess-1",
		"--session-seq", "3",
		"--severity", "info",
		"--source-stream", "github",
	})
	if errMsg != "" {
		t.Fatalf("errMsg=%q", errMsg)
	}
	if o.Subject != "dept.research.events.github.pr_opened" || o.Content != "PR #12 opened" {
		t.Fatalf("opts=%+v", o)
	}
	if o.EventTime != "2026-08-06T02:00:00Z" || o.SessionID != "sess-1" || o.SessionSeq != 3 {
		t.Fatalf("opts=%+v", o)
	}
	if o.Severity != "info" || o.SourceStream != "github" {
		t.Fatalf("opts=%+v", o)
	}
	// = forms + short aliases.
	o2, errMsg2 := parseMemoryIngestEventArgs([]string{
		"-s=subj.x",
		"-c=hello",
		"--event_time=2026-01-01T00:00:00Z",
		"--session_id=s2",
		"--session_seq=9",
		"--source_stream=mesh",
	})
	if errMsg2 != "" {
		t.Fatalf("errMsg=%q", errMsg2)
	}
	if o2.Subject != "subj.x" || o2.Content != "hello" || o2.EventTime != "2026-01-01T00:00:00Z" {
		t.Fatalf("opts=%+v", o2)
	}
	if o2.SessionID != "s2" || o2.SessionSeq != 9 || o2.SourceStream != "mesh" {
		t.Fatalf("opts=%+v", o2)
	}
	// Empty args: zeros (caller shows usage for missing subject/content).
	o3, errMsg3 := parseMemoryIngestEventArgs(nil)
	if errMsg3 != "" || o3.Subject != "" || o3.Content != "" {
		t.Fatalf("empty opts=%+v err=%q", o3, errMsg3)
	}
	// Invalid session-seq.
	_, badSeq := parseMemoryIngestEventArgs([]string{"--session-seq", "nope"})
	if badSeq == "" {
		t.Fatal("expected invalid --session-seq")
	}
	// Unknown flag / bare arg rejected.
	_, badFlag := parseMemoryIngestEventArgs([]string{"--unknown"})
	if badFlag == "" {
		t.Fatal("expected unknown flag")
	}
	_, badArg := parseMemoryIngestEventArgs([]string{"bare"})
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

// s1352: /gtm slash — residual-honest GTM draft-only guidance (no auto-send agency).
func TestHandleSlash_GtmDraftOnly(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}

	_, err := handleSlash(&out, adapter, "/gtm")
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{
		"drafts only",
		"no auto-send",
		"human publish",
		"dual_write OFF",
		"not Memory GA",
		"Salesforce",
		"GA CRM",
		"gtm-draft-only-agent",
		"read_skill",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("/gtm missing %q in:\n%s", want, s)
		}
	}
	// Residual footer (s1352)
	if !strings.Contains(s, "residual:") {
		t.Fatalf("/gtm missing residual footer: %s", s)
	}
	// Must not invent auto-send agency / dual_write ON / suite ops GA.
	if strings.Contains(s, "auto-send ON") || strings.Contains(s, "dual_write ON") {
		t.Fatalf("must not invent auto-send/dual_write ON: %s", s)
	}
	if strings.Contains(s, "suite ops GA shipped") || strings.Contains(s, "Memory GA shipped") {
		t.Fatalf("must not invent suite/Memory GA: %s", s)
	}

	// aliases
	for _, alias := range []string{"/gtm-draft", "/gtm-agent"} {
		out.Reset()
		_, _ = handleSlash(&out, adapter, alias)
		if !strings.Contains(out.String(), "drafts only") || !strings.Contains(out.String(), "gtm-draft-only-agent") {
			t.Fatalf("%s alias: %s", alias, out.String())
		}
	}
}

// s1358: /gtm help and /gtm checklist — residual-honest numbered draft-only checklist.
func TestHandleSlash_GtmHelpChecklist(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}

	needles := []string{
		"1.",
		"Draft content/outreach only",
		"never auto-send",
		"email/SNS",
		"2.",
		"Human publish",
		"human CRM commercial",
		"3.",
		"Salesforce",
		"GA CRM",
		"HubSpot",
		"Beta multi-tenant",
		"guerrilla",
		"global-only",
		"4.",
		"portal HITL",
		"not agent APPLY",
		"5.",
		"dual_write OFF",
		"not Memory GA",
		"book-demo OFF",
		"6.",
		"read_skill",
		"gtm-draft-only-agent",
	}

	for _, line := range []string{"/gtm help", "/gtm checklist", "/gtm-draft help", "/gtm-agent checklist"} {
		out.Reset()
		_, err := handleSlash(&out, adapter, line)
		if err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		s := out.String()
		for _, want := range needles {
			if !strings.Contains(s, want) {
				t.Fatalf("%s missing %q in:\n%s", line, want, s)
			}
		}
		// Checklist mode must not invent auto-send agency.
		if strings.Contains(s, "auto-send ON") || strings.Contains(s, "dual_write ON") {
			t.Fatalf("%s must not invent auto-send/dual_write ON: %s", line, s)
		}
		// Bare guidance residual footer is for /gtm only, not required on checklist.
		// But checklist should not claim suite ops GA shipped.
		if strings.Contains(s, "suite ops GA shipped") || strings.Contains(s, "Memory GA shipped") {
			t.Fatalf("%s must not invent suite/Memory GA: %s", line, s)
		}
	}

	// /help mentions help|checklist subcommands
	out.Reset()
	_, err := handleSlash(&out, adapter, "/help")
	if err != nil {
		t.Fatal(err)
	}
	help := out.String()
	if !strings.Contains(help, "/gtm") || !strings.Contains(help, "checklist") {
		t.Fatalf("/help missing /gtm checklist mention: %s", help)
	}
}

// s1363: /onboard slash — residual-honest TUI agent ↔ aion onboarding guidance.
func TestHandleSlash_Onboard(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}

	_, err := handleSlash(&out, adapter, "/onboard")
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{
		"list_connector_catalog",
		"plan_connector_setup",
		"list_org_connector_installs",
		"dual_write OFF",
		"not Memory GA",
		"portal HITL",
		"never invent install green",
		"aion-agent-onboarding",
		"read_skill",
		"console.iome.sh/integrations",
		// s1368 portal Agent/MCP lane in guidance
		"console.iome.sh/settings/agent",
		"Agent/MCP",
		"[[mcp.servers]]",
		"streamable HTTP",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("/onboard missing %q in:\n%s", want, s)
		}
	}
	// Residual footer (s1363)
	if !strings.Contains(s, "residual:") {
		t.Fatalf("/onboard missing residual footer: %s", s)
	}
	if strings.Contains(s, "dual_write ON") || strings.Contains(s, "Connected: yes") {
		t.Fatalf("must not invent dual_write ON / Connected: %s", s)
	}
	if strings.Contains(s, "Memory GA shipped") || strings.Contains(s, "Agent Plugins GA shipped") {
		t.Fatalf("must not invent Memory/Plugins GA: %s", s)
	}

	// aliases
	for _, alias := range []string{"/aion-onboard", "/agent-onboard"} {
		out.Reset()
		_, _ = handleSlash(&out, adapter, alias)
		if !strings.Contains(out.String(), "list_connector_catalog") || !strings.Contains(out.String(), "aion-agent-onboarding") {
			t.Fatalf("%s alias: %s", alias, out.String())
		}
	}
}

// s1363: /onboard help and /onboard checklist — residual-honest numbered onboarding checklist.
func TestHandleSlash_OnboardHelpChecklist(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}

	needles := []string{
		"1.",
		"IOMESH/MCP",
		"fail-open",
		"2.",
		"list_connector_catalog",
		"catalog status ≠ Connected",
		"3.",
		"plan_connector_setup",
		"4.",
		"list_org_connector_installs",
		"available=false ≠ empty-as-none",
		"5.",
		"dual_write OFF",
		"not Memory GA",
		"6.",
		"/integrations status",
		"console.iome.sh/integrations",
		"never invent install green",
		"INSTALL_STORE APPLY",
		"book-demo OFF",
		"residual PASS ≠ live dogfood",
		// s1368
		"console.iome.sh/settings/agent",
		"Agent/MCP",
		"[[mcp.servers]]",
		"/onboard portal",
	}

	for _, line := range []string{"/onboard help", "/onboard checklist", "/aion-onboard help", "/agent-onboard checklist"} {
		out.Reset()
		_, err := handleSlash(&out, adapter, line)
		if err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		s := out.String()
		for _, want := range needles {
			if !strings.Contains(s, want) {
				t.Fatalf("%s missing %q in:\n%s", line, want, s)
			}
		}
		if strings.Contains(s, "dual_write ON") || strings.Contains(s, "Memory GA shipped") {
			t.Fatalf("%s must not invent dual_write ON / Memory GA: %s", line, s)
		}
	}

	// Unknown sub → guidance + usage
	out.Reset()
	_, err := handleSlash(&out, adapter, "/onboard bogon")
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, "list_connector_catalog") || !strings.Contains(s, "usage: /onboard") {
		t.Fatalf("/onboard bogon want guidance+usage: %s", s)
	}
	if !strings.Contains(s, "residual:") {
		t.Fatalf("/onboard bogon missing residual footer: %s", s)
	}

	// /help mentions /onboard checklist
	out.Reset()
	_, err = handleSlash(&out, adapter, "/help")
	if err != nil {
		t.Fatal(err)
	}
	help := out.String()
	if !strings.Contains(help, "/onboard") || !strings.Contains(help, "checklist") {
		t.Fatalf("/help missing /onboard checklist mention: %s", help)
	}
	if !strings.Contains(help, "portal") || !strings.Contains(help, "status") {
		t.Fatalf("/help missing /onboard portal|status mention: %s", help)
	}
	// s1372: /help mentions /onboard next
	if !strings.Contains(help, "next") {
		t.Fatalf("/help missing /onboard next mention: %s", help)
	}
	// s1377+s1382+s1387+s1402: /help mentions next [plugins|gtm|memory|mesh|status|export] lanes
	if !strings.Contains(help, "plugins") || !strings.Contains(help, "gtm") || !strings.Contains(help, "memory") {
		// help line lists next [plugins|gtm|memory|…] — ensure lane tokens appear near onboard
		if !strings.Contains(help, "next [plugins|gtm|memory") && !strings.Contains(help, "plugins|gtm|memory") {
			t.Fatalf("/help missing /onboard next lane drill mention: %s", help)
		}
	}
	if !strings.Contains(help, "mesh") {
		t.Fatalf("/help missing /onboard next mesh mention: %s", help)
	}
	if !strings.Contains(help, "export") {
		t.Fatalf("/help missing /onboard next export mention: %s", help)
	}
}

// s1368: /onboard portal (and aliases) — residual-honest portal Agent/MCP handoff.
func TestHandleSlash_OnboardPortal(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}

	needles := []string{
		"portal Agent/MCP",
		"console.iome.sh/settings/agent",
		"Mint API key",
		"copy MCP connection",
		"Test invoke",
		"probe only",
		"not Memory GA",
		"[[mcp.servers]]",
		"streamable HTTP",
		"/integrations status",
		"console.iome.sh/integrations",
		"agent MCP cannot write installs",
		"dual_write OFF",
		"never invent install green",
		"INSTALL_STORE APPLY",
		"residual PASS ≠ live dogfood",
	}

	for _, line := range []string{"/onboard portal", "/aion-onboard portal", "/agent-onboard agent-mcp", "/onboard mcp"} {
		out.Reset()
		_, err := handleSlash(&out, adapter, line)
		if err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		s := out.String()
		for _, want := range needles {
			if !strings.Contains(s, want) {
				t.Fatalf("%s missing %q in:\n%s", line, want, s)
			}
		}
		if !strings.Contains(s, "residual:") {
			t.Fatalf("%s missing residual footer: %s", line, s)
		}
		if strings.Contains(s, "dual_write ON") || strings.Contains(s, "Connected: yes") {
			t.Fatalf("%s must not invent dual_write ON / Connected: %s", line, s)
		}
	}
}

// s1368: /onboard status — residual-honest offline static status (no MCP dial).
func TestHandleSlash_OnboardStatus(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}

	_, err := handleSlash(&out, adapter, "/onboard status")
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{
		"MCP attach",
		"fail-open offline",
		"dual_write OFF",
		"not Memory GA",
		"portal HITL",
		"console.iome.sh/settings/agent",
		"console.iome.sh/integrations",
		"never invent install green",
		"agent MCP cannot write installs",
		"residual PASS ≠ live dogfood",
		"/onboard portal",
		// s1372 cross-link
		"/onboard next",
		// s1382 cross-link to lane status board
		"/onboard next status",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("/onboard status missing %q in:\n%s", want, s)
		}
	}
	if strings.Contains(s, "dual_write ON") || strings.Contains(s, "Connected: yes") {
		t.Fatalf("must not invent dual_write ON / Connected: %s", s)
	}
}

// s1372: /onboard next (and aliases) — residual-honest post-onboard operator lanes overview.
func TestHandleSlash_OnboardNext(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}

	needles := []string{
		"onboard next lanes",
		"post-onboard continuum",
		"iomesh plugins dogfood",
		"offline sample validate",
		"Agent Plugins GA",
		"/gtm checklist",
		"gtm-draft-only-agent",
		"drafts only",
		"no auto-send",
		"human publish",
		"aion-memory-mcp",
		"local-primary",
		"dual_write OFF",
		"package load ≠ Memory GA",
		"freemium palace",
		"portal HITL",
		"agent MCP cannot write installs",
		"catalog ≠ Connected",
		"not Memory GA",
		"residual PASS ≠ live dogfood",
		"never invent install green",
		"INSTALL_STORE APPLY",
		// s1377 drill cross-links on overview
		"/onboard next plugins",
		"/onboard next gtm",
		"/onboard next memory",
		// s1382 status board cross-link
		"/onboard next status",
		// s1387 status export receipt cross-link
		"/onboard next export",
		"board/export evidence ≠ invent Connected",
	}

	for _, line := range []string{"/onboard next", "/aion-onboard after", "/agent-onboard continue", "/onboard lanes"} {
		out.Reset()
		_, err := handleSlash(&out, adapter, line)
		if err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		s := out.String()
		for _, want := range needles {
			if !strings.Contains(s, want) {
				t.Fatalf("%s missing %q in:\n%s", line, want, s)
			}
		}
		if !strings.Contains(s, "residual:") {
			t.Fatalf("%s missing residual footer: %s", line, s)
		}
		if strings.Contains(s, "dual_write ON") || strings.Contains(s, "Connected: yes") {
			t.Fatalf("%s must not invent dual_write ON / Connected: %s", line, s)
		}
		if strings.Contains(s, "Memory GA shipped") || strings.Contains(s, "Agent Plugins GA shipped") {
			t.Fatalf("%s must not invent Memory/Plugins GA: %s", line, s)
		}
	}
}

// s1377: /onboard next plugins|plugin|dogfood — residual-honest plugins dogfood lane drill.
func TestHandleSlash_OnboardNextPluginsLane(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}

	needles := []string{
		"onboard next plugins lane",
		"iomesh plugins dogfood",
		"/plugins dogfood",
		"offline sample validate",
		"examples/agent-plugins",
		"Agent Plugins GA",
		"plugins dogfood ≠ invent Agent Plugins GA",
		"residual PASS ≠ live dogfood",
		"dual_write OFF",
		"not Memory GA",
		"package load ≠ Memory GA",
		"never invent install green",
		"INSTALL_STORE APPLY",
		"catalog ≠ Connected",
		"portal HITL",
		"agent MCP cannot write installs",
	}

	for _, line := range []string{
		"/onboard next plugins",
		"/onboard next plugin",
		"/onboard next dogfood",
		"/aion-onboard after plugins",
		"/agent-onboard continue dogfood",
		"/onboard lanes plugins",
	} {
		out.Reset()
		_, err := handleSlash(&out, adapter, line)
		if err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		s := out.String()
		for _, want := range needles {
			if !strings.Contains(s, want) {
				t.Fatalf("%s missing %q in:\n%s", line, want, s)
			}
		}
		if !strings.Contains(s, "residual:") {
			t.Fatalf("%s missing residual footer: %s", line, s)
		}
		if strings.Contains(s, "dual_write ON") || strings.Contains(s, "Agent Plugins GA shipped") {
			t.Fatalf("%s must not invent dual_write ON / Plugins GA: %s", line, s)
		}
	}
}

// s1377: /onboard next gtm|drafts — residual-honest GTM draft-only lane drill.
func TestHandleSlash_OnboardNextGtmLane(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}

	needles := []string{
		"onboard next gtm lane",
		"/gtm checklist",
		"gtm-draft-only-agent",
		"drafts only",
		"no auto-send",
		"human publish",
		"GTM agent GA",
		"GTM checklist ≠ invent GTM agent GA",
		"dual_write OFF",
		"not Memory GA",
		"portal HITL",
		"agent MCP cannot write installs",
		"never invent install green",
		"INSTALL_STORE APPLY",
	}

	for _, line := range []string{
		"/onboard next gtm",
		"/onboard next drafts",
		"/aion-onboard after gtm",
		"/agent-onboard continue drafts",
		"/onboard lanes gtm",
	} {
		out.Reset()
		_, err := handleSlash(&out, adapter, line)
		if err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		s := out.String()
		for _, want := range needles {
			if !strings.Contains(s, want) {
				t.Fatalf("%s missing %q in:\n%s", line, want, s)
			}
		}
		if !strings.Contains(s, "residual:") {
			t.Fatalf("%s missing residual footer: %s", line, s)
		}
		if strings.Contains(s, "auto-send enabled") || strings.Contains(s, "GTM agent GA shipped") {
			t.Fatalf("%s must not invent auto-send / GTM agent GA: %s", line, s)
		}
	}
}

// s1377+s1453+s1458: /onboard next memory|mcp|palace — residual-honest memory local + edge OSS + M2 lean attach drill.
func TestHandleSlash_OnboardNextMemoryLane(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}

	needles := []string{
		"onboard next memory lane",
		"aion-memory-mcp",
		"iomesh-memory-mcp",
		"github.com/iome-sh/iomesh-memory-mcp",
		"http://127.0.0.1:8080/mcp",
		"stdio",
		"M2 lean",
		"scaffold/M2",
		"PASS ≠ invent full platform sidecar parity",
		"aion broker private",
		"OSS path ≠ invent public flip complete",
		"Memory Ops Pack",
		"local-primary",
		"dual_write OFF",
		"package load ≠ Memory GA",
		"freemium palace",
		"not Memory GA",
		"Palace sunset",
		"/memory status",
		"residual PASS ≠ live dogfood",
		"portal HITL",
		"agent MCP cannot write installs",
		"never invent install green",
		"INSTALL_STORE APPLY",
		"console.iome.sh/settings/agent",
	}

	for _, line := range []string{
		"/onboard next memory",
		"/onboard next mcp",
		"/onboard next palace",
		"/aion-onboard after memory",
		"/agent-onboard continue palace",
		"/onboard lanes mcp",
	} {
		out.Reset()
		_, err := handleSlash(&out, adapter, line)
		if err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		s := out.String()
		for _, want := range needles {
			if !strings.Contains(s, want) {
				t.Fatalf("%s missing %q in:\n%s", line, want, s)
			}
		}
		if !strings.Contains(s, "residual:") {
			t.Fatalf("%s missing residual footer: %s", line, s)
		}
		if strings.Contains(s, "dual_write ON") || strings.Contains(s, "Memory GA shipped") {
			t.Fatalf("%s must not invent dual_write ON / Memory GA: %s", line, s)
		}
	}
}

// s1402: /onboard next mesh|stream|streams|heartbeat|heartbeats|pull — residual-honest mesh streaming lane.
// pulse stays status board (not a mesh alias).
func TestHandleSlash_OnboardNextMeshLane(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}

	needles := []string{
		"onboard next mesh lane",
		"I/O Mesh",
		"streaming org heartbeats",
		"dept.*",
		"mesh ≠ memory",
		"not OTel/APM",
		"streams_not_probed",
		"never invent stream green",
		"dual_write OFF",
		"not Memory GA",
		"iomesh memory pull",
		"not freemium hosted palace",
		"Palace sunset",
		"residual PASS ≠ live dogfood",
		"portal HITL",
		"agent MCP cannot write installs",
		"catalog ≠ Connected",
	}

	for _, line := range []string{
		"/onboard next mesh",
		"/onboard next stream",
		"/onboard next streams",
		"/onboard next heartbeat",
		"/onboard next heartbeats",
		"/onboard next pull",
		"/aion-onboard after mesh",
		"/agent-onboard continue streams",
		"/onboard lanes heartbeat",
	} {
		out.Reset()
		_, err := handleSlash(&out, adapter, line)
		if err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		s := out.String()
		for _, want := range needles {
			if !strings.Contains(s, want) {
				t.Fatalf("%s missing %q in:\n%s", line, want, s)
			}
		}
		if !strings.Contains(s, "residual:") {
			t.Fatalf("%s missing residual footer: %s", line, s)
		}
		if strings.Contains(s, "dual_write ON") || strings.Contains(s, "Memory GA shipped") {
			t.Fatalf("%s must not invent dual_write ON / Memory GA: %s", line, s)
		}
		if strings.Contains(s, "stream green: yes") || strings.Contains(s, "Connected: yes") {
			t.Fatalf("%s must not invent stream/Connected green: %s", line, s)
		}
		// Mesh drill must not be confused with status board body title.
		if strings.Contains(s, "onboard next lane status (") {
			t.Fatalf("%s must not emit status board: %s", line, s)
		}
	}

	// pulse stays status board — not mesh lane.
	out.Reset()
	_, err := handleSlash(&out, adapter, "/onboard next pulse")
	if err != nil {
		t.Fatal(err)
	}
	pulseOut := out.String()
	if !strings.Contains(pulseOut, "onboard next lane status") {
		t.Fatalf("pulse must stay status board, got:\n%s", pulseOut)
	}
	if strings.Contains(pulseOut, "onboard next mesh lane") {
		t.Fatalf("pulse must not drill mesh lane:\n%s", pulseOut)
	}

	// bare pull stays mesh (s1407 memory-pull must not steal pull).
	out.Reset()
	_, err = handleSlash(&out, adapter, "/onboard next pull")
	if err != nil {
		t.Fatal(err)
	}
	pullOut := out.String()
	if !strings.Contains(pullOut, "onboard next mesh lane") {
		t.Fatalf("bare pull must map to mesh lane, got:\n%s", pullOut)
	}
	if strings.Contains(pullOut, "onboard next memory-pull lane") {
		t.Fatalf("bare pull must not drill memory-pull lane:\n%s", pullOut)
	}
}

// s1407: /onboard next memory-pull|ops-pack|pull-path|memorypull|ops_pack — residual-honest Ops Pack pull path.
// bare pull stays mesh (asserted in mesh test).
func TestHandleSlash_OnboardNextMemoryPullLane(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}

	needles := []string{
		"onboard next memory-pull lane",
		"Ops Pack pull path",
		"iomesh memory pull",
		"mesh → local palace",
		"dual_write OFF",
		"not Memory GA",
		"pull_not_probed",
		"never invent pull green",
		"Ops Pack ≠ GPU fleet",
		"not freemium hosted palace",
		"Palace sunset",
		"package load ≠ Ops Pack entitlement",
		"residual PASS ≠ live dogfood",
		"PASS ≠ live APPLY",
		"mesh ≠ memory",
		"portal HITL",
		"agent MCP cannot write installs",
		"catalog ≠ Connected",
	}

	for _, line := range []string{
		"/onboard next memory-pull",
		"/onboard next ops-pack",
		"/onboard next pull-path",
		"/onboard next memorypull",
		"/onboard next ops_pack",
		"/aion-onboard after memory-pull",
		"/agent-onboard continue ops-pack",
		"/onboard lanes pull-path",
	} {
		out.Reset()
		_, err := handleSlash(&out, adapter, line)
		if err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		s := out.String()
		for _, want := range needles {
			if !strings.Contains(s, want) {
				t.Fatalf("%s missing %q in:\n%s", line, want, s)
			}
		}
		if !strings.Contains(s, "residual:") {
			t.Fatalf("%s missing residual footer: %s", line, s)
		}
		if strings.Contains(s, "dual_write ON") || strings.Contains(s, "Memory GA shipped") {
			t.Fatalf("%s must not invent dual_write ON / Memory GA: %s", line, s)
		}
		if strings.Contains(s, "pull green: yes") || strings.Contains(s, "Connected: yes") {
			t.Fatalf("%s must not invent pull/Connected green: %s", line, s)
		}
		// Must not be mesh lane body title.
		if strings.Contains(s, "onboard next mesh lane") {
			t.Fatalf("%s must not emit mesh lane body: %s", line, s)
		}
		if strings.Contains(s, "onboard next lane status (") {
			t.Fatalf("%s must not emit status board: %s", line, s)
		}
	}
}

// s1377+s1382+s1387+s1402+s1407+s1413+s1417+s1432+s1437+s1442+s1447: unknown /onboard next <lane> → overview + usage hint listing lanes.
func TestHandleSlash_OnboardNextUnknownLane(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}

	_, err := handleSlash(&out, adapter, "/onboard next unknown-lane")
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{
		"onboard next lanes",
		"post-onboard continuum",
		"usage:",
		"plugins",
		"gtm",
		"memory",
		"mesh",
		"memory-pull",
		"agentic",
		"planes",
		"sales",
		"demo",
		"operator",
		"status",
		"export",
		"human-gates",
		"residual:",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("unknown next lane missing %q in:\n%s", want, s)
		}
	}
	if strings.Contains(s, "onboard next plugins lane") {
		t.Fatalf("unknown next lane must not drill into plugins lane: %s", s)
	}
	if strings.Contains(s, "onboard next mesh lane") {
		t.Fatalf("unknown next lane must not drill into mesh lane: %s", s)
	}
	if strings.Contains(s, "onboard next memory-pull lane") {
		t.Fatalf("unknown next lane must not drill into memory-pull lane: %s", s)
	}
	if strings.Contains(s, "onboard next agentic lane") {
		t.Fatalf("unknown next lane must not drill into agentic lane: %s", s)
	}
	if strings.Contains(s, "onboard next three product planes") {
		t.Fatalf("unknown next lane must not drill into three planes board: %s", s)
	}
	if strings.Contains(s, "onboard next sales / buyer claims") {
		t.Fatalf("unknown next lane must not drill into sales claims board: %s", s)
	}
	if strings.Contains(s, "onboard next demo readiness") {
		t.Fatalf("unknown next lane must not drill into demo readiness board: %s", s)
	}
	if strings.Contains(s, "onboard next operator readiness matrix") {
		t.Fatalf("unknown next lane must not drill into operator readiness matrix: %s", s)
	}
	if strings.Contains(s, "human-gates honesty board") {
		t.Fatalf("unknown next lane must not drill into human-gates board: %s", s)
	}
	if strings.Contains(s, "onboard next lane status") {
		t.Fatalf("unknown next lane must not drill into status board: %s", s)
	}
	if strings.Contains(s, "evidence_kind=onboard_next_lane_status_export") {
		t.Fatalf("unknown next lane must not emit export receipt: %s", s)
	}
}

// s1437: /onboard next sales|claims|buyer|claim-matrix|sales-claims|buyer-claims —
// residual-honest sales/buyer claims board (may claim / must not claim).
// product/planes stay three-planes; gtm stays drafts; pulse stays status.
func TestHandleSlash_OnboardNextSalesClaims(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}

	needles := []string{
		"onboard next sales / buyer claims",
		"no MCP dial",
		"s1437",
		"MAY CLAIM",
		"MUST NOT CLAIM",
		"streaming org heartbeats",
		"dual_write OFF",
		"not Memory GA",
		"Salesforce = GA CRM",
		"HubSpot + GTM suite Beta multi-tenant",
		"never invent Connected",
		"dual_auth_candidacy_open",
		"tool ship ≠ dual-auth live",
		"book-demo OFF",
		"residual PASS ≠ live dogfood",
		"PASS ≠ live APPLY",
		"three-planes grounded",
		"/onboard next planes",
	}
	for _, line := range []string{
		"/onboard next sales",
		"/onboard next claims",
		"/onboard next buyer",
		"/onboard next claim-matrix",
		"/onboard next sales-claims",
		"/onboard next buyer-claims",
		"/aion-onboard after sales",
		"/agent-onboard continue claims",
	} {
		out.Reset()
		_, err := handleSlash(&out, adapter, line)
		if err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		s := out.String()
		for _, want := range needles {
			if !strings.Contains(s, want) {
				t.Fatalf("%s missing %q in:\n%s", line, want, s)
			}
		}
		// Must not be three-planes / status / gtm-only boards.
		if strings.Contains(s, "onboard next three product planes (") {
			t.Fatalf("%s must not be three planes board: %s", line, s)
		}
		if strings.Contains(s, "onboard next lane status (") {
			t.Fatalf("%s must not be status board: %s", line, s)
		}
		if strings.Contains(s, "onboard next gtm lane") {
			t.Fatalf("%s must not be gtm-only lane: %s", line, s)
		}
	}

	// product stays three-planes (not sales claims).
	out.Reset()
	_, err := handleSlash(&out, adapter, "/onboard next product")
	if err != nil {
		t.Fatal(err)
	}
	productOut := out.String()
	if !strings.Contains(productOut, "onboard next three product planes") {
		t.Fatalf("product must stay three planes: %s", productOut)
	}
	if strings.Contains(productOut, "onboard next sales / buyer claims") {
		t.Fatalf("product must not open sales claims: %s", productOut)
	}

	// planes stays three-planes (not sales claims).
	out.Reset()
	_, err = handleSlash(&out, adapter, "/onboard next planes")
	if err != nil {
		t.Fatal(err)
	}
	planesOut := out.String()
	if !strings.Contains(planesOut, "onboard next three product planes") {
		t.Fatalf("planes must stay three planes: %s", planesOut)
	}
	if strings.Contains(planesOut, "onboard next sales / buyer claims") {
		t.Fatalf("planes must not open sales claims: %s", planesOut)
	}

	// gtm stays GTM draft lane (not sales claims).
	out.Reset()
	_, err = handleSlash(&out, adapter, "/onboard next gtm")
	if err != nil {
		t.Fatal(err)
	}
	gtmOut := out.String()
	if !strings.Contains(gtmOut, "onboard next gtm lane") {
		t.Fatalf("gtm must stay gtm lane: %s", gtmOut)
	}
	if strings.Contains(gtmOut, "onboard next sales / buyer claims") {
		t.Fatalf("gtm must not open sales claims: %s", gtmOut)
	}

	// pulse stays status board (not sales claims).
	out.Reset()
	_, err = handleSlash(&out, adapter, "/onboard next pulse")
	if err != nil {
		t.Fatal(err)
	}
	pulseOut := out.String()
	if !strings.Contains(pulseOut, "onboard next lane status") {
		t.Fatalf("pulse must stay status board: %s", pulseOut)
	}
	if strings.Contains(pulseOut, "onboard next sales / buyer claims") {
		t.Fatalf("pulse must not open sales claims: %s", pulseOut)
	}
}

// s1442: /onboard next demo|demo-ready|readiness|demo-readiness|lighthouse|landgrab —
// residual-honest demo readiness board (Lighthouse · book-demo OFF · Landgrab NOT READY).
// sales/claims stay sales claims; product/planes stay three-planes; pulse stays status; gtm stays drafts.
func TestHandleSlash_OnboardNextDemoReadiness(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}

	needles := []string{
		"onboard next demo readiness",
		"no MCP dial",
		"s1442",
		"Lighthouse beachhead",
		"book-demo OFF",
		"Landgrab NOT READY",
		"dual_write OFF",
		"not Memory GA",
		"never invent Connected",
		"residual PASS ≠ live dogfood",
		"PASS ≠ live APPLY",
		"residual PASS ≠ logos met",
		"/onboard next planes",
		"/onboard next sales",
		"/onboard next human-gates",
		"founder-led walkthrough only when scheduled",
	}
	for _, line := range []string{
		"/onboard next demo",
		"/onboard next demo-ready",
		"/onboard next readiness",
		"/onboard next demo-readiness",
		"/onboard next lighthouse",
		"/onboard next landgrab",
		"/aion-onboard after demo",
		"/agent-onboard continue lighthouse",
	} {
		out.Reset()
		_, err := handleSlash(&out, adapter, line)
		if err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		s := out.String()
		for _, want := range needles {
			if !strings.Contains(s, want) {
				t.Fatalf("%s missing %q in:\n%s", line, want, s)
			}
		}
		// Must not be sales / three-planes / status / gtm-only boards.
		if strings.Contains(s, "onboard next sales / buyer claims (") {
			t.Fatalf("%s must not be sales claims board: %s", line, s)
		}
		if strings.Contains(s, "onboard next three product planes (") {
			t.Fatalf("%s must not be three planes board: %s", line, s)
		}
		if strings.Contains(s, "onboard next lane status (") {
			t.Fatalf("%s must not be status board: %s", line, s)
		}
		if strings.Contains(s, "onboard next gtm lane") {
			t.Fatalf("%s must not be gtm-only lane: %s", line, s)
		}
	}

	// sales stays sales claims (not demo readiness).
	out.Reset()
	_, err := handleSlash(&out, adapter, "/onboard next sales")
	if err != nil {
		t.Fatal(err)
	}
	salesOut := out.String()
	if !strings.Contains(salesOut, "onboard next sales / buyer claims") {
		t.Fatalf("sales must stay sales claims: %s", salesOut)
	}
	if strings.Contains(salesOut, "onboard next demo readiness") {
		t.Fatalf("sales must not open demo readiness: %s", salesOut)
	}

	// claims stays sales claims (not demo readiness).
	out.Reset()
	_, err = handleSlash(&out, adapter, "/onboard next claims")
	if err != nil {
		t.Fatal(err)
	}
	claimsOut := out.String()
	if !strings.Contains(claimsOut, "onboard next sales / buyer claims") {
		t.Fatalf("claims must stay sales claims: %s", claimsOut)
	}
	if strings.Contains(claimsOut, "onboard next demo readiness") {
		t.Fatalf("claims must not open demo readiness: %s", claimsOut)
	}

	// product stays three-planes (not demo readiness).
	out.Reset()
	_, err = handleSlash(&out, adapter, "/onboard next product")
	if err != nil {
		t.Fatal(err)
	}
	productOut := out.String()
	if !strings.Contains(productOut, "onboard next three product planes") {
		t.Fatalf("product must stay three planes: %s", productOut)
	}
	if strings.Contains(productOut, "onboard next demo readiness") {
		t.Fatalf("product must not open demo readiness: %s", productOut)
	}

	// planes stays three-planes (not demo readiness).
	out.Reset()
	_, err = handleSlash(&out, adapter, "/onboard next planes")
	if err != nil {
		t.Fatal(err)
	}
	planesOut := out.String()
	if !strings.Contains(planesOut, "onboard next three product planes") {
		t.Fatalf("planes must stay three planes: %s", planesOut)
	}
	if strings.Contains(planesOut, "onboard next demo readiness") {
		t.Fatalf("planes must not open demo readiness: %s", planesOut)
	}

	// gtm stays GTM draft lane (not demo readiness).
	out.Reset()
	_, err = handleSlash(&out, adapter, "/onboard next gtm")
	if err != nil {
		t.Fatal(err)
	}
	gtmOut := out.String()
	if !strings.Contains(gtmOut, "onboard next gtm lane") {
		t.Fatalf("gtm must stay gtm lane: %s", gtmOut)
	}
	if strings.Contains(gtmOut, "onboard next demo readiness") {
		t.Fatalf("gtm must not open demo readiness: %s", gtmOut)
	}

	// pulse stays status board (not demo readiness).
	out.Reset()
	_, err = handleSlash(&out, adapter, "/onboard next pulse")
	if err != nil {
		t.Fatal(err)
	}
	pulseOut := out.String()
	if !strings.Contains(pulseOut, "onboard next lane status") {
		t.Fatalf("pulse must stay status board: %s", pulseOut)
	}
	if strings.Contains(pulseOut, "onboard next demo readiness") {
		t.Fatalf("pulse must not open demo readiness: %s", pulseOut)
	}
}

// s1432: /onboard next planes|three-planes|product-planes|product|pillars|three_planes —
// residual-honest three product planes board (mesh · memory-pull · agentic).
// bare pulse stays status; bare pull stays mesh; bare mcp stays memory.
// s1447: /onboard next operator|operator-matrix|ops-matrix|operator-readiness|ops-readiness|matrix —
// residual-honest operator readiness matrix (demo · sales · planes · human-gates).
// demo/readiness/lighthouse/landgrab stay demo; sales/claims stay sales; product/planes stay three-planes;
// pulse stays status; export stays export.
func TestHandleSlash_OnboardNextOperatorMatrix(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}

	needles := []string{
		"onboard next operator readiness matrix",
		"no MCP dial",
		"s1447",
		"book-demo OFF",
		"Landgrab NOT READY",
		"dual_write OFF",
		"not Memory GA",
		"never invent Connected",
		"dual_auth_candidacy_open",
		"tool ship ≠ dual-auth live",
		"residual PASS ≠ live dogfood",
		"PASS ≠ live APPLY",
		"residual PASS ≠ logos met",
		"residual_only",
		"path_ready",
		"still_human",
		"policy_off",
		"not_ready",
		"portal_hitl_still",
		"/onboard next demo",
		"/onboard next sales",
		"/onboard next planes",
		"/onboard next human-gates",
		"/onboard next export",
	}
	for _, line := range []string{
		"/onboard next operator",
		"/onboard next operator-matrix",
		"/onboard next ops-matrix",
		"/onboard next operator-readiness",
		"/onboard next ops-readiness",
		"/onboard next matrix",
		"/aion-onboard after operator",
		"/agent-onboard continue matrix",
	} {
		out.Reset()
		_, err := handleSlash(&out, adapter, line)
		if err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		s := out.String()
		for _, want := range needles {
			if !strings.Contains(s, want) {
				t.Fatalf("%s missing %q in:\n%s", line, want, s)
			}
		}
		// Must not be demo / sales / three-planes / status / export boards alone.
		if strings.Contains(s, "onboard next demo readiness (") {
			t.Fatalf("%s must not be demo readiness board: %s", line, s)
		}
		if strings.Contains(s, "onboard next sales / buyer claims (") {
			t.Fatalf("%s must not be sales claims board: %s", line, s)
		}
		if strings.Contains(s, "onboard next three product planes (") {
			t.Fatalf("%s must not be three planes board: %s", line, s)
		}
		if strings.Contains(s, "onboard next lane status (") {
			t.Fatalf("%s must not be status board: %s", line, s)
		}
		if strings.Contains(s, "evidence_kind=onboard_next_lane_status_export") {
			t.Fatalf("%s must not be export receipt: %s", line, s)
		}
	}

	// demo stays demo readiness (not operator matrix).
	out.Reset()
	_, err := handleSlash(&out, adapter, "/onboard next demo")
	if err != nil {
		t.Fatal(err)
	}
	demoOut := out.String()
	if !strings.Contains(demoOut, "onboard next demo readiness") {
		t.Fatalf("demo must stay demo readiness: %s", demoOut)
	}
	if strings.Contains(demoOut, "onboard next operator readiness matrix") {
		t.Fatalf("demo must not open operator matrix: %s", demoOut)
	}

	// readiness stays demo (not operator matrix).
	out.Reset()
	_, err = handleSlash(&out, adapter, "/onboard next readiness")
	if err != nil {
		t.Fatal(err)
	}
	readinessOut := out.String()
	if !strings.Contains(readinessOut, "onboard next demo readiness") {
		t.Fatalf("readiness must stay demo readiness: %s", readinessOut)
	}
	if strings.Contains(readinessOut, "onboard next operator readiness matrix") {
		t.Fatalf("readiness must not open operator matrix: %s", readinessOut)
	}

	// landgrab stays demo (not operator matrix).
	out.Reset()
	_, err = handleSlash(&out, adapter, "/onboard next landgrab")
	if err != nil {
		t.Fatal(err)
	}
	landgrabOut := out.String()
	if !strings.Contains(landgrabOut, "onboard next demo readiness") {
		t.Fatalf("landgrab must stay demo readiness: %s", landgrabOut)
	}
	if strings.Contains(landgrabOut, "onboard next operator readiness matrix") {
		t.Fatalf("landgrab must not open operator matrix: %s", landgrabOut)
	}

	// sales stays sales claims (not operator matrix).
	out.Reset()
	_, err = handleSlash(&out, adapter, "/onboard next sales")
	if err != nil {
		t.Fatal(err)
	}
	salesOut := out.String()
	if !strings.Contains(salesOut, "onboard next sales / buyer claims") {
		t.Fatalf("sales must stay sales claims: %s", salesOut)
	}
	if strings.Contains(salesOut, "onboard next operator readiness matrix") {
		t.Fatalf("sales must not open operator matrix: %s", salesOut)
	}

	// planes stays three-planes (not operator matrix).
	out.Reset()
	_, err = handleSlash(&out, adapter, "/onboard next planes")
	if err != nil {
		t.Fatal(err)
	}
	planesOut := out.String()
	if !strings.Contains(planesOut, "onboard next three product planes") {
		t.Fatalf("planes must stay three planes: %s", planesOut)
	}
	if strings.Contains(planesOut, "onboard next operator readiness matrix") {
		t.Fatalf("planes must not open operator matrix: %s", planesOut)
	}

	// pulse stays status board (not operator matrix).
	out.Reset()
	_, err = handleSlash(&out, adapter, "/onboard next pulse")
	if err != nil {
		t.Fatal(err)
	}
	pulseOut := out.String()
	if !strings.Contains(pulseOut, "onboard next lane status") {
		t.Fatalf("pulse must stay status board: %s", pulseOut)
	}
	if strings.Contains(pulseOut, "onboard next operator readiness matrix") {
		t.Fatalf("pulse must not open operator matrix: %s", pulseOut)
	}

	// export stays export receipt (not operator matrix).
	out.Reset()
	_, err = handleSlash(&out, adapter, "/onboard next export")
	if err != nil {
		t.Fatal(err)
	}
	exportOut := out.String()
	if !strings.Contains(exportOut, "evidence_kind=onboard_next_lane_status_export") {
		t.Fatalf("export must stay export receipt: %s", exportOut)
	}
	if strings.Contains(exportOut, "onboard next operator readiness matrix") {
		t.Fatalf("export must not open operator matrix: %s", exportOut)
	}
}

func TestHandleSlash_OnboardNextThreePlanes(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}

	needles := []string{
		"onboard next three product planes",
		"no MCP dial",
		"plane 1",
		"streaming org heartbeats",
		"streams_not_probed",
		"mesh ≠ memory",
		"plane 2",
		"pull_not_probed",
		"Ops Pack ≠ GPU",
		"plane 3",
		"list_plan_not_connected",
		"dual_auth_candidacy_open",
		"never invent Connected",
		"dual_write OFF",
		"not Memory GA",
		"residual PASS ≠ live dogfood",
		"PASS ≠ live APPLY",
	}
	for _, line := range []string{
		"/onboard next planes",
		"/onboard next three-planes",
		"/onboard next product-planes",
		"/onboard next product",
		"/onboard next pillars",
		"/onboard next three_planes",
		"/aion-onboard after planes",
		"/agent-onboard continue three-planes",
	} {
		out.Reset()
		_, err := handleSlash(&out, adapter, line)
		if err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		s := out.String()
		for _, want := range needles {
			if !strings.Contains(s, want) {
				t.Fatalf("%s missing %q in:\n%s", line, want, s)
			}
		}
		// Must not be status/mesh/memory/agentic-only boards.
		if strings.Contains(s, "onboard next lane status (") {
			t.Fatalf("%s must not be status board: %s", line, s)
		}
		if strings.Contains(s, "onboard next mesh lane") {
			t.Fatalf("%s must not be mesh-only lane: %s", line, s)
		}
		if strings.Contains(s, "onboard next agentic lane (") {
			t.Fatalf("%s must not be agentic-only lane: %s", line, s)
		}
	}

	// bare pulse under /onboard next stays status board (not three-planes).
	out.Reset()
	_, err := handleSlash(&out, adapter, "/onboard next pulse")
	if err != nil {
		t.Fatal(err)
	}
	pulseOut := out.String()
	if !strings.Contains(pulseOut, "onboard next lane status") {
		t.Fatalf("bare pulse must stay status board: %s", pulseOut)
	}
	if strings.Contains(pulseOut, "onboard next three product planes") {
		t.Fatalf("bare pulse must not open three planes board: %s", pulseOut)
	}

	// bare pull stays mesh (not three-planes).
	out.Reset()
	_, err = handleSlash(&out, adapter, "/onboard next pull")
	if err != nil {
		t.Fatal(err)
	}
	pullOut := out.String()
	if !strings.Contains(pullOut, "onboard next mesh lane") {
		t.Fatalf("bare pull must stay mesh: %s", pullOut)
	}
	if strings.Contains(pullOut, "onboard next three product planes") {
		t.Fatalf("bare pull must not open three planes board: %s", pullOut)
	}

	// bare mcp stays memory (not three-planes).
	out.Reset()
	_, err = handleSlash(&out, adapter, "/onboard next mcp")
	if err != nil {
		t.Fatal(err)
	}
	mcpOut := out.String()
	if !strings.Contains(mcpOut, "onboard next memory lane") {
		t.Fatalf("bare mcp must stay memory: %s", mcpOut)
	}
	if strings.Contains(mcpOut, "onboard next three product planes") {
		t.Fatalf("bare mcp must not open three planes board: %s", mcpOut)
	}
}

// s1417: /onboard next agentic|agentic-integrations|integrations|portal-hitl|list-plan|hitl —
// residual-honest product plane 3 agentic integrations (MCP list/plan + portal HITL).
// bare mcp stays memory; bare portal stays portal handoff; bare pull stays mesh.
func TestHandleSlash_OnboardNextAgenticLane(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}
	agent.ResetAgenticListPlanSoftDogfoodSessionState()
	t.Cleanup(agent.ResetAgenticListPlanSoftDogfoodSessionState)

	needles := []string{
		"onboard next agentic lane",
		"product plane 3",
		"agentic integrations",
		"MCP list/plan residual-honest",
		"plan_connector_setup",
		"portal deep links",
		"browser HITL only",
		"template= ≠ install APPLY",
		"list_org fail-open ≠ empty-as-none",
		"catalog ≠ Connected",
		"agent MCP cannot write installs",
		"never invent Connected",
		"list_plan_not_connected",
		"portal_hitl_still",
		"dual_write OFF",
		"not Memory GA",
		"book-demo OFF",
		"residual PASS ≠ live dogfood",
		"PASS ≠ live APPLY",
		"console.iome.sh/integrations",
		"console.iome.sh/settings/agent",
		// s1422 portal HITL polish + soft tip on board
		"Portal HITL polish",
		"list_plan_soft_not_run",
		"/onboard next agentic dogfood",
	}

	for _, line := range []string{
		"/onboard next agentic",
		"/onboard next agentic-integrations",
		"/onboard next integrations",
		"/onboard next portal-hitl",
		"/onboard next list-plan",
		"/onboard next hitl",
		"/aion-onboard after agentic",
		"/agent-onboard continue integrations",
		"/onboard lanes portal-hitl",
	} {
		out.Reset()
		_, err := handleSlash(&out, adapter, line)
		if err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		s := out.String()
		for _, want := range needles {
			if !strings.Contains(s, want) {
				t.Fatalf("%s missing %q in:\n%s", line, want, s)
			}
		}
		if !strings.Contains(s, "residual:") {
			t.Fatalf("%s missing residual footer: %s", line, s)
		}
		// Bare agentic must be board, not auto soft dogfood runner.
		if strings.Contains(s, "agentic list/plan soft offline dogfood") {
			t.Fatalf("%s must not auto-run soft dogfood (bare agentic = board):\n%s", line, s)
		}
		if strings.Contains(s, "dual_write ON") || strings.Contains(s, "Memory GA shipped") {
			t.Fatalf("%s must not invent dual_write ON / Memory GA: %s", line, s)
		}
		if strings.Contains(s, "Connected: yes") || strings.Contains(s, "INSTALL_STORE APPLY success") {
			t.Fatalf("%s must not invent Connected/APPLY green: %s", line, s)
		}
		// Must not be memory lane or portal handoff body titles.
		if strings.Contains(s, "onboard next memory lane") {
			t.Fatalf("%s must not emit memory lane body: %s", line, s)
		}
		if strings.Contains(s, "portal Agent/MCP handoff") {
			t.Fatalf("%s must not emit portal handoff body: %s", line, s)
		}
		if strings.Contains(s, "onboard next lane status (") {
			t.Fatalf("%s must not emit status board: %s", line, s)
		}
	}

	// bare mcp under /onboard next stays memory lane (not agentic).
	out.Reset()
	_, err := handleSlash(&out, adapter, "/onboard next mcp")
	if err != nil {
		t.Fatal(err)
	}
	mcpOut := out.String()
	if !strings.Contains(mcpOut, "onboard next memory lane") {
		t.Fatalf("bare mcp under next must map to memory lane, got:\n%s", mcpOut)
	}
	if strings.Contains(mcpOut, "onboard next agentic lane") {
		t.Fatalf("bare mcp must not drill agentic lane:\n%s", mcpOut)
	}

	// bare portal under /onboard stays portal handoff (not agentic).
	out.Reset()
	_, err = handleSlash(&out, adapter, "/onboard portal")
	if err != nil {
		t.Fatal(err)
	}
	portalOut := out.String()
	if !strings.Contains(portalOut, "portal Agent/MCP handoff") {
		t.Fatalf("bare portal must map to portal handoff, got:\n%s", portalOut)
	}
	if strings.Contains(portalOut, "onboard next agentic lane") {
		t.Fatalf("bare portal must not drill agentic lane:\n%s", portalOut)
	}
}

// s1427: /onboard next agentic dual-auth|candidacy|list-org|org-installs|dual_auth|dual-auth-candidacy —
// residual-honest dual-auth candidacy depth (list_org fail-open · tool ship ≠ dual-auth live).
// dogfood|soft|samples|offline|list-plan-soft stay soft dogfood; bare agentic stays main board.
func TestHandleSlash_OnboardNextAgenticDualAuthCandidacy(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}
	agent.ResetAgenticListPlanSoftDogfoodSessionState()
	t.Cleanup(agent.ResetAgenticListPlanSoftDogfoodSessionState)

	needles := []string{
		"onboard next agentic dual-auth candidacy",
		"dual_auth_candidacy_open",
		"list_org_connector_installs",
		"available=false",
		"status=unavailable",
		"installs=null",
		"never invent empty-as-none",
		"tool ship ≠ dual-auth live",
		"never invent dual-auth live",
		"portal session owns install index",
		"agent MCP cannot write installs",
		"catalog ≠ Connected",
		"never invent Connected",
		"list_org_unavailable",
		"path_ready",
		"residual_only",
		"dual_write OFF",
		"not Memory GA",
		"book-demo OFF",
		"residual PASS ≠ live dogfood",
		"PASS ≠ live APPLY",
		"open boxes stay open",
		"console.iome.sh/integrations",
	}

	for _, line := range []string{
		"/onboard next agentic dual-auth",
		"/onboard next agentic candidacy",
		"/onboard next agentic list-org",
		"/onboard next agentic org-installs",
		"/onboard next agentic dual_auth",
		"/onboard next agentic dual-auth-candidacy",
		"/onboard next agentic-integrations dual-auth",
		"/onboard next integrations candidacy",
		"/aion-onboard after agentic list-org",
		"/agent-onboard continue portal-hitl org-installs",
	} {
		out.Reset()
		_, err := handleSlash(&out, adapter, line)
		if err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		s := out.String()
		for _, want := range needles {
			if !strings.Contains(s, want) {
				t.Fatalf("%s missing %q in:\n%s", line, want, s)
			}
		}
		if !strings.Contains(s, "residual:") {
			t.Fatalf("%s missing residual footer: %s", line, s)
		}
		// Must not be soft dogfood runner or main agentic board body title alone without dual-auth framing.
		if strings.Contains(s, "agentic list/plan soft offline dogfood") {
			t.Fatalf("%s must not run soft dogfood:\n%s", line, s)
		}
		if strings.Contains(s, "onboard next agentic lane (") {
			t.Fatalf("%s must not emit main agentic board body:\n%s", line, s)
		}
		if strings.Contains(s, "dual_write ON") || strings.Contains(s, "Connected: yes") {
			t.Fatalf("%s must not invent dual_write ON / Connected:\n%s", line, s)
		}
		if strings.Contains(s, "dual-auth: live") || strings.Contains(s, "dual-auth live shipped") {
			t.Fatalf("%s must not invent dual-auth live:\n%s", line, s)
		}
	}

	// Bare agentic still main board (not dual-auth board).
	out.Reset()
	_, err := handleSlash(&out, adapter, "/onboard next agentic")
	if err != nil {
		t.Fatal(err)
	}
	bare := out.String()
	if !strings.Contains(bare, "onboard next agentic lane") {
		t.Fatalf("bare agentic want main board:\n%s", bare)
	}
	if strings.Contains(bare, "onboard next agentic dual-auth candidacy") {
		t.Fatalf("bare agentic must not emit dual-auth board body:\n%s", bare)
	}
	// Soft dogfood aliases still soft dogfood (not dual-auth).
	out.Reset()
	_, err = handleSlash(&out, adapter, "/onboard next agentic dogfood")
	if err != nil {
		t.Fatal(err)
	}
	dog := out.String()
	if !strings.Contains(dog, "agentic list/plan soft offline dogfood") {
		t.Fatalf("dogfood alias must stay soft dogfood:\n%s", dog)
	}
	if strings.Contains(dog, "onboard next agentic dual-auth candidacy") {
		t.Fatalf("dogfood must not emit dual-auth board:\n%s", dog)
	}
}

// s1422: /onboard next agentic dogfood|soft|offline|list-plan-soft|samples — soft offline list/plan dogfood.
func TestHandleSlash_OnboardNextAgenticListPlanSoftDogfood(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}
	agent.ResetAgenticListPlanSoftDogfoodSessionState()
	resetPluginsSlashSession()
	t.Cleanup(func() {
		agent.ResetAgenticListPlanSoftDogfoodSessionState()
		resetPluginsSlashSession()
	})

	needles := []string{
		"agentic list/plan soft offline dogfood",
		"no MCP dial",
		"not live dogfood",
		"result: PASS",
		"soft_offline_list_plan_session_pass",
		"/integrations/{id}",
		"/integrations/add?template={id}",
		"list_plan_not_connected",
		"soft offline list/plan ≠ live dogfood",
		"≠ invent Connected",
		"portal HITL still",
		"session soft ≠ live dogfood",
		"re-run /onboard next status then /onboard next export",
		"dual_write OFF",
		"agent MCP cannot write installs",
	}

	for _, line := range []string{
		"/onboard next agentic dogfood",
		"/onboard next agentic soft",
		"/onboard next agentic offline",
		"/onboard next agentic list-plan-soft",
		"/onboard next agentic samples",
		"/onboard next agentic-integrations dogfood",
		"/onboard next integrations soft",
		"/aion-onboard after agentic offline",
		"/agent-onboard continue portal-hitl list-plan-soft",
	} {
		agent.ResetAgenticListPlanSoftDogfoodSessionState()
		out.Reset()
		_, err := handleSlash(&out, adapter, line)
		if err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		s := out.String()
		for _, want := range needles {
			if !strings.Contains(s, want) {
				t.Fatalf("%s missing %q in:\n%s", line, want, s)
			}
		}
		if !strings.Contains(s, "residual:") {
			t.Fatalf("%s missing residual footer: %s", line, s)
		}
		// Must not be the static board body title alone without dogfood runner framing.
		if strings.Contains(s, "Connected: yes") || strings.Contains(s, "dual_write ON") {
			t.Fatalf("%s must not invent Connected/dual_write ON:\n%s", line, s)
		}
		// Must not steal plugins dogfood path.
		if strings.Contains(s, "onboard next plugins lane") {
			t.Fatalf("%s must not emit plugins lane:\n%s", line, s)
		}
		// Session marker set
		if got := agent.AgenticListPlanSoftSessionLabel(); got != agent.AgenticListPlanSoftPass {
			t.Fatalf("%s session label: got %q want %q", line, got, agent.AgenticListPlanSoftPass)
		}
	}

	// Status/export reflect soft after dogfood
	out.Reset()
	_, err := handleSlash(&out, adapter, "/onboard next status")
	if err != nil {
		t.Fatal(err)
	}
	statusOut := out.String()
	if !strings.Contains(statusOut, "soft_offline_list_plan_session_pass") {
		t.Fatalf("status after dogfood want soft pass:\n%s", statusOut)
	}
	if !strings.Contains(statusOut, "list_plan_not_connected") {
		t.Fatalf("status still want list_plan_not_connected:\n%s", statusOut)
	}
	// Plugins soft independent default
	if !strings.Contains(statusOut, "dogfood_not_run") {
		t.Fatalf("plugins soft must stay dogfood_not_run when only agentic soft ran:\n%s", statusOut)
	}

	out.Reset()
	_, err = handleSlash(&out, adapter, "/onboard next export json")
	if err != nil {
		t.Fatal(err)
	}
	js := out.String()
	if !strings.Contains(js, `"agentic_list_plan_soft_state": "soft_offline_list_plan_session_pass"`) {
		t.Fatalf("export json want agentic_list_plan_soft_state pass:\n%s", js)
	}

	// bare /onboard next dogfood still plugins lane (does not steal for agentic)
	agent.ResetAgenticListPlanSoftDogfoodSessionState()
	out.Reset()
	_, err = handleSlash(&out, adapter, "/onboard next dogfood")
	if err != nil {
		t.Fatal(err)
	}
	pluginsOut := out.String()
	if !strings.Contains(pluginsOut, "onboard next plugins lane") {
		t.Fatalf("bare next dogfood must stay plugins lane:\n%s", pluginsOut)
	}
	if strings.Contains(pluginsOut, "agentic list/plan soft offline dogfood") {
		t.Fatalf("bare next dogfood must not run agentic soft dogfood:\n%s", pluginsOut)
	}
}

// s1413: /onboard next human-gates|human|gates|apply-gates — residual-honest still-required vs offline.
func TestHandleSlash_OnboardNextHumanGates(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}

	needles := []string{
		"human-gates honesty board",
		"still_human",
		"offline_residual_only",
		"shipped_or_policy",
		"Slack HMAC",
		"Stripe Customers:Write",
		"H1/H2 INSTALL_STORE",
		"D1–D5",
		"book-demo OFF",
		"ON_SIGNAL",
		"dual_write OFF",
		"not Memory GA",
		"PASS ≠ invent human-gate green",
		"PASS ≠ live APPLY",
		"open boxes stay open",
		"Knowledge Beta→GA cannot invent H1/H2 offline",
		"dry-run ≠ APPLY",
		"analytical NO-install intentional",
		"do NOT close human APPLY gates",
		"make human-gates-status",
		"never invent APPLY",
		"Palace sunset",
		"agent MCP cannot write installs",
	}

	for _, line := range []string{
		"/onboard next human-gates",
		"/onboard next human",
		"/onboard next gates",
		"/onboard next apply-gates",
		"/aion-onboard after human-gates",
		"/agent-onboard continue gates",
		"/onboard lanes human",
	} {
		out.Reset()
		_, err := handleSlash(&out, adapter, line)
		if err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		s := out.String()
		for _, want := range needles {
			if !strings.Contains(s, want) {
				t.Fatalf("%s missing %q in:\n%s", line, want, s)
			}
		}
		if !strings.Contains(s, "residual:") {
			t.Fatalf("%s missing residual footer: %s", line, s)
		}
		if strings.Contains(s, "dual_write ON") || strings.Contains(s, "book-demo ON") {
			t.Fatalf("%s must not invent dual_write ON / book-demo ON: %s", line, s)
		}
		if strings.Contains(s, "Memory GA shipped") || strings.Contains(s, "Connected: yes") {
			t.Fatalf("%s must not invent Memory GA / Connected: %s", line, s)
		}
		// Must not be status board body title.
		if strings.Contains(s, "onboard next lane status (") {
			t.Fatalf("%s must not emit status board: %s", line, s)
		}
		if strings.Contains(s, "onboard next mesh lane") {
			t.Fatalf("%s must not emit mesh lane body: %s", line, s)
		}
	}
}

// s1382: /onboard next status|pulse|board — residual-honest lane status board.
func TestHandleSlash_OnboardNextLaneStatus(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}
	resetPluginsSlashSession()
	agent.ResetAgenticListPlanSoftDogfoodSessionState()
	t.Cleanup(func() {
		resetPluginsSlashSession()
		agent.ResetAgenticListPlanSoftDogfoodSessionState()
	})

	needles := []string{
		"onboard next lane status",
		"no MCP dial",
		"plugins:",
		"gtm:",
		"memory:",
		"mesh:",
		"memory-pull:",
		"agentic:",
		"list_plan_soft_not_run",
		"portal:",
		"dogfood_not_run",
		"path_ready",
		"skill_ready",
		"residual_only",
		"streams_not_probed",
		"pull_not_probed",
		"portal_hitl_still",
		"list_plan_not_connected",
		"mesh ≠ memory",
		"dual_write OFF",
		"not Memory GA",
		"package load ≠ Memory GA",
		"drafts only",
		"no auto-send",
		"Agent Plugins GA",
		"agent MCP cannot write installs",
		"catalog ≠ Connected",
		"portal HITL",
		"never invent install green",
		"INSTALL_STORE APPLY",
		"residual PASS ≠ live dogfood",
		"session soft ≠ live dogfood",
		"/onboard next export",
		"/onboard next memory-pull",
		"/onboard next agentic",
		"Ops Pack ≠ GPU fleet",
		"never invent pull green",
		"board/export evidence ≠ invent Connected",
		"product plane 3",
	}

	for _, line := range []string{
		"/onboard next status",
		"/onboard next pulse",
		"/onboard next board",
		"/aion-onboard after status",
		"/agent-onboard continue pulse",
		"/onboard lanes board",
	} {
		out.Reset()
		_, err := handleSlash(&out, adapter, line)
		if err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		s := out.String()
		for _, want := range needles {
			if !strings.Contains(s, want) {
				t.Fatalf("%s missing %q in:\n%s", line, want, s)
			}
		}
		if !strings.Contains(s, "residual:") {
			t.Fatalf("%s missing residual footer: %s", line, s)
		}
		if !strings.Contains(s, "samples_ok") && !strings.Contains(s, "samples_missing") {
			t.Fatalf("%s must report samples_ok or samples_missing:\n%s", line, s)
		}
		if strings.Contains(s, "dual_write ON") || strings.Contains(s, "Connected: yes") {
			t.Fatalf("%s must not invent dual_write ON / Connected: %s", line, s)
		}
		if strings.Contains(s, "Memory GA shipped") || strings.Contains(s, "Agent Plugins GA shipped") {
			t.Fatalf("%s must not invent Memory/Plugins GA: %s", line, s)
		}
		if strings.Contains(s, "INSTALL_STORE APPLY success") {
			t.Fatalf("%s must not invent APPLY success: %s", line, s)
		}
	}
}

// s1387: /onboard next export|receipt|stamp|evidence — residual-honest status export receipt.
func TestHandleSlash_OnboardNextLaneStatusExport(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}
	resetPluginsSlashSession()
	agent.ResetAgenticListPlanSoftDogfoodSessionState()
	t.Cleanup(func() {
		resetPluginsSlashSession()
		agent.ResetAgenticListPlanSoftDogfoodSessionState()
	})

	needles := []string{
		"evidence_kind=onboard_next_lane_status_export",
		"offline_static",
		"not_live_dogfood",
		"serial=s1387",
		"export receipt",
		"plugins:",
		"gtm:",
		"memory:",
		"mesh:",
		"memory-pull:",
		"agentic:",
		"portal:",
		"dogfood_not_run",
		"list_plan_soft_not_run",
		"path_ready",
		"skill_ready",
		"residual_only",
		"streams_not_probed",
		"pull_not_probed",
		"portal_hitl_still",
		"list_plan_not_connected",
		"dual_write OFF",
		"not Memory GA",
		"mesh ≠ memory",
		"board/export evidence ≠ invent Connected",
		"never invent install green",
		"INSTALL_STORE APPLY",
		"agent MCP cannot write installs",
		"catalog ≠ Connected",
		"portal HITL",
		"package load ≠ Memory GA",
		"drafts only",
		"no auto-send",
		"residual PASS ≠ live dogfood",
		"session soft ≠ live dogfood",
		"Ops Pack ≠ GPU fleet",
		"never invent pull green",
		"/onboard next memory-pull",
		"/onboard next agentic",
		"/onboard next agentic dogfood",
		"product plane 3",
	}

	for _, line := range []string{
		"/onboard next export",
		"/onboard next receipt",
		"/onboard next stamp",
		"/onboard next evidence",
		"/aion-onboard after export",
		"/agent-onboard continue receipt",
		"/onboard lanes stamp",
	} {
		out.Reset()
		_, err := handleSlash(&out, adapter, line)
		if err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		s := out.String()
		for _, want := range needles {
			if !strings.Contains(s, want) {
				t.Fatalf("%s missing %q in:\n%s", line, want, s)
			}
		}
		if !strings.Contains(s, "residual:") {
			t.Fatalf("%s missing residual footer: %s", line, s)
		}
		if !strings.Contains(s, "samples_ok") && !strings.Contains(s, "samples_missing") {
			t.Fatalf("%s must report samples_ok or samples_missing:\n%s", line, s)
		}
		if strings.Contains(s, "dual_write ON") || strings.Contains(s, "Connected: yes") {
			t.Fatalf("%s must not invent dual_write ON / Connected: %s", line, s)
		}
		if strings.Contains(s, "Memory GA shipped") || strings.Contains(s, "Agent Plugins GA shipped") {
			t.Fatalf("%s must not invent Memory/Plugins GA: %s", line, s)
		}
		if strings.Contains(s, "INSTALL_STORE APPLY success") {
			t.Fatalf("%s must not invent APPLY success: %s", line, s)
		}
	}
}

// s1387: /onboard next export json — residual-honest JSON status export receipt.
func TestHandleSlash_OnboardNextLaneStatusExportJSON(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}
	resetPluginsSlashSession()
	agent.ResetAgenticListPlanSoftDogfoodSessionState()
	t.Cleanup(func() {
		resetPluginsSlashSession()
		agent.ResetAgenticListPlanSoftDogfoodSessionState()
	})

	for _, line := range []string{
		"/onboard next export json",
		"/onboard next receipt json",
		"/aion-onboard after evidence json",
	} {
		out.Reset()
		_, err := handleSlash(&out, adapter, line)
		if err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		s := out.String()
		for _, want := range []string{
			`"evidence_kind": "onboard_next_lane_status_export"`,
			`"offline_static": true`,
			`"not_live_dogfood": true`,
			`"serial": "s1387"`,
			`"format": "json"`,
			`"dogfood_not_run": true`,
			`"plugins_dogfood_state": "dogfood_not_run"`,
			`"agentic_list_plan_soft_state": "list_plan_soft_not_run"`,
			"path_ready",
			"portal_hitl_still",
			"pull_not_probed",
			"list_plan_not_connected",
			`"memory-pull":`,
			`"ops_pack":`,
			`"agentic":`,
			"board/export evidence ≠ invent Connected",
			"session soft ≠ live dogfood",
			"residual:",
			"s1387",
			"/onboard next agentic",
			"/onboard next agentic dogfood",
		} {
			if !strings.Contains(s, want) {
				t.Fatalf("%s missing %q in:\n%s", line, want, s)
			}
		}
		if strings.Contains(s, "Connected: yes") || strings.Contains(s, "dual_write ON") {
			t.Fatalf("%s must not invent Connected/dual_write ON: %s", line, s)
		}
		if strings.Contains(s, "INSTALL_STORE APPLY success") {
			t.Fatalf("%s must not invent APPLY success: %s", line, s)
		}
	}
}

// s1397: after /plugins dogfood, /onboard next status + export reflect session soft marker.
func TestHandleSlash_OnboardNextStatusExport_SessionSoftDogfood(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}
	resetPluginsSlashSession()
	agent.ResetAgenticListPlanSoftDogfoodSessionState()
	t.Cleanup(func() {
		resetPluginsSlashSession()
		agent.ResetAgenticListPlanSoftDogfoodSessionState()
	})

	// Default status/export: dogfood_not_run
	out.Reset()
	_, err := handleSlash(&out, adapter, "/onboard next status")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "dogfood_not_run") {
		t.Fatalf("default status want dogfood_not_run:\n%s", out.String())
	}

	// Run soft offline dogfood (sets session marker)
	out.Reset()
	_, err = handleSlash(&out, adapter, "/plugins dogfood")
	if err != nil {
		t.Fatal(err)
	}
	dogfoodOut := out.String()
	for _, want := range []string{
		"/onboard next status",
		"/onboard next export",
		"session soft ≠ live dogfood",
		"Agent Plugins GA",
	} {
		if !strings.Contains(dogfoodOut, want) {
			t.Fatalf("dogfood tip/honesty missing %q in:\n%s", want, dogfoodOut)
		}
	}
	if strings.Contains(dogfoodOut, "Agent Plugins GA shipped") || strings.Contains(dogfoodOut, "Connected: yes") {
		t.Fatalf("dogfood must not invent GA/Connected:\n%s", dogfoodOut)
	}

	// Status board should show session soft marker (pass expected when samples present).
	out.Reset()
	_, err = handleSlash(&out, adapter, "/onboard next status")
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, "soft_offline_dogfood_session") {
		t.Fatalf("after dogfood, status want soft_offline_dogfood_session_*:\n%s", s)
	}
	if strings.Contains(s, "· dogfood_not_run ·") {
		t.Fatalf("after dogfood, plugins lane must not hardcode dogfood_not_run:\n%s", s)
	}
	for _, want := range []string{
		"session soft ≠ live dogfood",
		"board/export evidence ≠ invent Connected",
		"not live dogfood",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("status after dogfood missing %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "Connected: yes") || strings.Contains(s, "Agent Plugins GA shipped") || strings.Contains(s, "live dogfood green") {
		t.Fatalf("status after dogfood must not invent Connected/GA/live:\n%s", s)
	}

	// Export markdown
	out.Reset()
	_, err = handleSlash(&out, adapter, "/onboard next export")
	if err != nil {
		t.Fatal(err)
	}
	s = out.String()
	if !strings.Contains(s, "soft_offline_dogfood_session") {
		t.Fatalf("after dogfood, export want soft_offline_dogfood_session_*:\n%s", s)
	}
	if strings.Contains(s, "Connected: yes") || strings.Contains(s, "Agent Plugins GA shipped") {
		t.Fatalf("export after dogfood must not invent Connected/GA:\n%s", s)
	}

	// Export JSON
	out.Reset()
	_, err = handleSlash(&out, adapter, "/onboard next export json")
	if err != nil {
		t.Fatal(err)
	}
	s = out.String()
	if !strings.Contains(s, `"plugins_dogfood_state": "soft_offline_dogfood_session_`) {
		t.Fatalf("export JSON after dogfood want plugins_dogfood_state session marker:\n%s", s)
	}
	if !strings.Contains(s, `"dogfood_not_run": false`) {
		t.Fatalf("export JSON after dogfood want dogfood_not_run false:\n%s", s)
	}
	if !strings.Contains(s, `"not_live_dogfood": true`) {
		t.Fatalf("export JSON must keep not_live_dogfood true:\n%s", s)
	}
	if strings.Contains(s, "Connected: yes") || strings.Contains(s, "Agent Plugins GA shipped") {
		t.Fatalf("export JSON after dogfood must not invent Connected/GA:\n%s", s)
	}
}

// s1392: residual-honest /plugins slash soft offline dogfood (help·list·validate·dogfood·status).
func TestHandleSlash_Plugins(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}
	resetPluginsSlashSession()
	t.Cleanup(resetPluginsSlashSession)

	// bare /plugins → help
	_, err := handleSlash(&out, adapter, "/plugins")
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{
		"usage: /plugins",
		"list",
		"validate",
		"dogfood",
		"status",
		"soft offline dogfood ≠ invent Agent Plugins GA",
		"Discover ≠ Connected",
		"dual_write OFF",
		"not Memory GA",
		"residual PASS ≠ live dogfood",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("bare /plugins missing %q in:\n%s", want, s)
		}
	}
	if strings.Contains(s, "Agent Plugins GA shipped") || strings.Contains(s, "Connected: yes") {
		t.Fatalf("must not invent GA/Connected: %s", s)
	}

	// help aliases
	for _, line := range []string{"/plugins help", "/plugin ?", "/plugins help"} {
		out.Reset()
		_, err := handleSlash(&out, adapter, line)
		if err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		if !strings.Contains(out.String(), "usage: /plugins") {
			t.Fatalf("%s want help: %s", line, out.String())
		}
	}

	// list with no dirs → residual offline opt-in message (fail-open)
	out.Reset()
	_, err = handleSlash(&out, adapter, "/plugins list")
	if err != nil {
		t.Fatal(err)
	}
	s = out.String()
	for _, want := range []string{
		"[plugins] is opt-in",
		"/plugins dogfood",
		"Agent Plugins GA",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("list empty missing %q in:\n%s", want, s)
		}
	}
	if strings.Contains(s, "Connected: yes") {
		t.Fatalf("list must not invent Connected: %s", s)
	}

	// validate with no dirs → residual offline
	out.Reset()
	_, err = handleSlash(&out, adapter, "/plugins validate")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "[plugins] is opt-in") {
		t.Fatalf("validate empty: %s", out.String())
	}

	// status default dogfood_not_run + samples soft state
	out.Reset()
	_, err = handleSlash(&out, adapter, "/plugins status")
	if err != nil {
		t.Fatal(err)
	}
	s = out.String()
	for _, want := range []string{
		"plugins status",
		"dogfood_not_run",
		"not live dogfood",
		"soft offline dogfood ≠ invent Agent Plugins GA",
		"/plugins dogfood",
		"dual_write OFF",
		"package load ≠ Memory GA",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("status missing %q in:\n%s", want, s)
		}
	}
	if !strings.Contains(s, "samples_ok") && !strings.Contains(s, "samples_missing") {
		t.Fatalf("status must report samples soft state: %s", s)
	}
	if strings.Contains(s, "dogfood PASS live") || strings.Contains(s, "Agent Plugins GA shipped") {
		t.Fatalf("status must not invent live dogfood/GA: %s", s)
	}

	// status alias
	out.Reset()
	_, _ = handleSlash(&out, adapter, "/plugin st")
	if !strings.Contains(out.String(), "dogfood_not_run") {
		t.Fatalf("st alias: %s", out.String())
	}

	// dogfood soft offline (uses FindModuleRoot from cwd — repo has samples)
	out.Reset()
	_, err = handleSlash(&out, adapter, "/plugins dogfood")
	if err != nil {
		t.Fatal(err)
	}
	s = out.String()
	for _, want := range []string{
		"dogfood",
		"no MCP dial",
		"Agent Plugins GA",
		"dual_write OFF",
		"Discover ≠ Connected",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("dogfood missing %q in:\n%s", want, s)
		}
	}
	// Must not invent live dogfood green / GA product language.
	if strings.Contains(s, "Agent Plugins GA shipped") || strings.Contains(s, "live dogfood green") {
		t.Fatalf("dogfood must not invent GA/live green: %s", s)
	}
	// ResidualDogfoodHonesty footer present.
	if !strings.Contains(s, "PATH residual") && !strings.Contains(s, "dogfood PASS ≠ invent Agent Plugins GA") {
		// Accept either residual dogfood honesty phrasing.
		if !strings.Contains(s, "soft offline dogfood ≠ invent Agent Plugins GA") {
			t.Fatalf("dogfood missing honesty footer: %s", s)
		}
	}

	// dogfood aliases
	for _, line := range []string{"/plugins soft", "/plugins samples", "/plugin offline"} {
		out.Reset()
		_, err := handleSlash(&out, adapter, line)
		if err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		if !strings.Contains(out.String(), "dogfood") {
			t.Fatalf("%s want dogfood output: %s", line, out.String())
		}
	}

	// After dogfood, status may show session soft marker (≠ live dogfood).
	// Note prose still mentions "dogfood_not_run default" — check the dogfood: state line.
	out.Reset()
	_, _ = handleSlash(&out, adapter, "/plugins status")
	s = out.String()
	if !strings.Contains(s, "dogfood: soft_offline_dogfood_session") {
		t.Fatalf("after dogfood, want dogfood: soft_offline_dogfood_session_* marker:\n%s", s)
	}
	if strings.Contains(s, "dogfood: dogfood_not_run") {
		t.Fatalf("after dogfood, dogfood state line should not be dogfood_not_run:\n%s", s)
	}
	if strings.Contains(s, "live dogfood green") || strings.Contains(s, "Agent Plugins GA shipped") {
		t.Fatalf("session marker must not invent live GA: %s", s)
	}

	// unknown subcommand → help
	out.Reset()
	_, _ = handleSlash(&out, adapter, "/plugins bogon")
	if !strings.Contains(out.String(), "unknown subcommand") || !strings.Contains(out.String(), "usage: /plugins") {
		t.Fatalf("unknown sub: %s", out.String())
	}

	// /help mentions /plugins
	out.Reset()
	_, _ = handleSlash(&out, adapter, "/help")
	if !strings.Contains(out.String(), "/plugins") {
		t.Fatalf("/help missing /plugins: %s", out.String())
	}
}

// s1392: /plugins list|validate with real sample dir (Discover ≠ Connected).
func TestHandleSlash_PluginsListValidateSample(t *testing.T) {
	rt := testRuntime(t)
	var out bytes.Buffer
	adapter := runtimeAdapter{rt: rt}

	root, err := agentplugins.FindModuleRoot("")
	if err != nil {
		t.Skipf("module root not found (offline residual skip): %v", err)
	}
	sample := filepath.Join(root, "examples", "agent-plugins", "hello-iome")

	out.Reset()
	_, err = handleSlash(&out, adapter, "/plugins list "+sample)
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	// Discovered row expected for in-repo sample — never invent Connected green.
	if !strings.Contains(s, "hello-iome") {
		t.Fatalf("list want hello-iome row: %s", s)
	}
	if strings.Contains(s, "Connected: yes") {
		t.Fatalf("list must not invent Connected: %s", s)
	}
	if !strings.Contains(s, "honesty:") {
		t.Fatalf("list missing honesty footer: %s", s)
	}

	out.Reset()
	_, err = handleSlash(&out, adapter, "/plugins validate "+sample)
	if err != nil {
		t.Fatal(err)
	}
	s = out.String()
	if !strings.Contains(s, "OK") {
		t.Fatalf("validate want OK for sample: %s", s)
	}
	if strings.Contains(s, "Connected: yes") {
		t.Fatalf("validate must not invent Connected: %s", s)
	}
	if !strings.Contains(s, "honesty:") {
		t.Fatalf("validate missing honesty: %s", s)
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
