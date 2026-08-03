package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/iomesh"
	"github.com/iome-sh/iomesh-tui/internal/mcp"
	"github.com/iome-sh/iomesh-tui/internal/router"
)

func TestTruncateBytes(t *testing.T) {
	if got := truncateBytes("hello", 100); got != "hello" {
		t.Fatalf("got %q", got)
	}
	long := strings.Repeat("a", 100)
	got := truncateBytes(long, 20)
	if len(got) > 20 {
		t.Fatalf("len=%d got=%q", len(got), got)
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("expected ellipsis: %q", got)
	}
}

func TestMemoryStatusDisabled(t *testing.T) {
	rt := &Runtime{memory: DefaultMemoryConfig()}
	s := rt.MemoryStatusLine()
	if !strings.Contains(s, "disabled") {
		t.Fatalf("status=%q", s)
	}
}

func TestAttachMemorySystemNote(t *testing.T) {
	rtr, err := router.New(router.DefaultModels(), router.DefaultModelName)
	if err != nil {
		t.Fatal(err)
	}
	rt, err := New(Config{Workspace: t.TempDir(), SubagentsEnabled: false}, rtr, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	rt.AttachMemory(MemoryConfig{
		Enabled:    true,
		Server:     "memory",
		Tenant:     "acme",
		AutoRecall: true,
	})
	msgs := rt.Messages()
	if len(msgs) == 0 || !strings.Contains(msgs[0].Content, "<memory>") {
		t.Fatalf("expected memory system note: %+v", msgs)
	}
}

func TestMemoryRecallRequiresPath(t *testing.T) {
	rt := &Runtime{memory: MemoryConfig{Enabled: true, Server: "memory"}}
	_, err := rt.MemoryRecall(context.Background(), "q")
	if err == nil {
		t.Fatal("expected error without mcp or mesh")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Fatalf("err=%v", err)
	}
}

func TestMaybeInjectMemoryRecall_NoClient(t *testing.T) {
	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, AutoRecall: true, Server: "memory"},
		mcp:    mcp.NewManagerEmpty(nil),
	}
	var events []Event
	rt.maybeInjectMemoryRecall(context.Background(), "hello", func(e Event) {
		events = append(events, e)
	})
	if len(events) != 0 {
		t.Fatalf("expected fail-open no events, got %v", events)
	}
}

func TestMemoryRecall_PrefersSyncHTTP(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"memories": []map[string]any{
				{"id": "1", "summary": "prior decision: use sync retrieve", "score": 0.91},
				{"id": "2", "full": "full-only hit"},
			},
		})
	}))
	defer srv.Close()

	mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: srv.URL, Tenant: "dept.research"}, nil)
	rt := &Runtime{
		mesh: mesh,
		memory: MemoryConfig{
			Enabled: true, AutoRecall: true, Tenant: "dept.research",
			Server: "memory", Limit: 5, SessionID: "sess-sync",
		},
		// MCP intentionally absent — sync path alone must work.
	}
	out, err := rt.MemoryRecall(context.Background(), "what did we decide")
	if err != nil {
		t.Fatalf("MemoryRecall: %v", err)
	}
	if gotPath != "/v1/memory/retrieve" {
		t.Fatalf("path=%q", gotPath)
	}
	if gotBody["tenant_id"] != "dept.research" || gotBody["session_id"] != "sess-sync" {
		t.Fatalf("body=%v", gotBody)
	}
	if !strings.Contains(out, "prior decision") || !strings.Contains(out, "0.91") {
		t.Fatalf("out=%q", out)
	}
	if !strings.Contains(out, "full-only hit") {
		t.Fatalf("expected full fallback: %q", out)
	}
}

// s1068: config RecallSince/Until/SessionSeq flow into sync retrieve body.
func TestMemoryRecall_TemporalConfigOptions(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"memories": []map[string]any{{"summary": "in window"}},
		})
	}))
	defer srv.Close()

	mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: srv.URL, Tenant: "dept.x"}, nil)
	rt := &Runtime{
		mesh: mesh,
		memory: MemoryConfig{
			Enabled: true, Tenant: "dept.x", Server: "memory", Limit: 4,
			SessionID: "s1", RecallSince: "2026-07-01T00:00:00Z",
			RecallUntil: "2026-07-31T23:59:59Z", RecallSessionSeq: 7,
		},
	}
	out, err := rt.MemoryRecall(context.Background(), "window query")
	if err != nil {
		t.Fatalf("MemoryRecall: %v", err)
	}
	if !strings.Contains(out, "in window") {
		t.Fatalf("out=%q", out)
	}
	if gotBody["since"] != "2026-07-01T00:00:00Z" || gotBody["until"] != "2026-07-31T23:59:59Z" {
		t.Fatalf("temporal body=%v", gotBody)
	}
	if gotBody["session_seq"] != float64(7) {
		t.Fatalf("session_seq=%v", gotBody["session_seq"])
	}

	// Per-call opts override config.
	gotBody = nil
	out, err = rt.MemoryRecallWithOpts(context.Background(), "override", MemoryRecallOpts{
		Since: "2026-08-01T00:00:00Z", Until: "2026-08-02T00:00:00Z",
		SessionSeq: 1, SessionSeqSet: true,
	})
	if err != nil {
		t.Fatalf("MemoryRecallWithOpts: %v", err)
	}
	if gotBody["since"] != "2026-08-01T00:00:00Z" || gotBody["until"] != "2026-08-02T00:00:00Z" {
		t.Fatalf("override body=%v", gotBody)
	}
	if gotBody["session_seq"] != float64(1) {
		t.Fatalf("override session_seq=%v", gotBody["session_seq"])
	}
	_ = out
}

func TestMaybeInjectMemoryRecall_SyncHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/memory/retrieve" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"memories": []map[string]any{
				{"summary": "injected from sidecar"},
			},
		})
	}))
	defer srv.Close()

	mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: srv.URL, Tenant: "acme"}, nil)
	rt := &Runtime{
		mesh: mesh,
		memory: MemoryConfig{
			Enabled: true, AutoRecall: true, Tenant: "acme",
			Server: "memory", MaxSnippetBytes: 6000,
		},
	}
	var events []Event
	rt.maybeInjectMemoryRecall(context.Background(), "hello", func(e Event) {
		events = append(events, e)
	})
	if len(events) != 1 || events[0].Type != EventMemoryRecall {
		t.Fatalf("events=%v", events)
	}
	if len(rt.messages) != 1 || !strings.Contains(rt.messages[0].Content, "<memory-context>") {
		t.Fatalf("messages=%+v", rt.messages)
	}
	if !strings.Contains(rt.messages[0].Content, "injected from sidecar") {
		t.Fatalf("content=%q", rt.messages[0].Content)
	}
}

func TestMemoryRecall_SyncFailsFallsBackMCPUnavailable(t *testing.T) {
	// Mesh enabled but returns 404 on retrieve (broker-only); no MCP → error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: srv.URL, Tenant: "t"}, nil)
	rt := &Runtime{
		mesh:   mesh,
		memory: MemoryConfig{Enabled: true, Server: "memory", Tenant: "t"},
		mcp:    mcp.NewManagerEmpty(nil),
	}
	_, err := rt.MemoryRecall(context.Background(), "q")
	if err == nil {
		t.Fatal("expected error when sync 404 and MCP missing")
	}
}

func TestMemoryStatusLine_SyncHTTP(t *testing.T) {
	mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: "http://127.0.0.1:9", Tenant: "t"}, nil)
	rt := &Runtime{
		mesh:   mesh,
		memory: MemoryConfig{Enabled: true, Server: "memory", DualWrite: true, AutoRecall: true},
	}
	s := rt.MemoryStatusLine()
	if !strings.Contains(s, "sync_http=true") || !strings.Contains(s, "mcp=false") {
		t.Fatalf("status=%q", s)
	}
}

func TestFormatMemoryHits(t *testing.T) {
	if got := formatMemoryHits(nil, 0); got != "" {
		t.Fatalf("nil=%q", got)
	}
	got := formatMemoryHits([]iomesh.MemoryHit{
		{Summary: "a", Score: 0.5},
		{Full: "b only"},
		{Summary: "  "}, // skipped
	}, 0)
	if !strings.Contains(got, "[0.50] a") || !strings.Contains(got, "b only") || !strings.Contains(got, "---") {
		t.Fatalf("got=%q", got)
	}
}

// s1135: hop_distance rendered when present.
func TestFormatMemoryHits_HopDistance(t *testing.T) {
	got := formatMemoryHits([]iomesh.MemoryHit{
		{Summary: "near", HopDistance: 1, Score: 0.8},
		{Summary: "far", HopDistance: 2},
	}, 0)
	if !strings.Contains(got, "[hop=1] [0.80] near") || !strings.Contains(got, "[hop=2] far") {
		t.Fatalf("got=%q", got)
	}
}

func TestDefaultMemoryConfig(t *testing.T) {
	d := DefaultMemoryConfig()
	if d.Enabled || d.Server != "memory" || !d.AutoRecall || d.AutoIngest || d.DualWrite {
		t.Fatalf("%+v", d)
	}
}

// TestDefaultMemoryConfig_DualWriteOff pins s768 local-primary honesty (+ s771 naming · s774 buyer-claim · s785 org-pulse peers):
// dual_write is optional mesh audit only and defaults OFF (not primary cloud palace).
// s771: "Memory Palace" / $119 = local MCP + Memory Ops Pack naming honesty, not hosted GPU.
// s774: MIT OSS TUI agent harness ≠ hosted multi-tenant mesh CP; local-primary buyer claim pin.
// s785: org-pulse edge framing — local agent on org pulse plane; dual_write still OFF (docs pin peer).
func TestDefaultMemoryConfig_DualWriteOff(t *testing.T) {
	// s768: dual_write default OFF (local-primary honesty); s771 naming + s774 buyer-claim + s785 org-pulse peers
	d := DefaultMemoryConfig()
	if d.DualWrite {
		t.Fatalf("s768/s771/s774/s785 honesty: DualWrite must default false, got %+v", d)
	}
}

func TestMaybeAutoIngest_DualWriteOnly(t *testing.T) {
	var mu sync.Mutex
	var publishes []map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/streams/MEMORY_INGEST/publish" {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		publishes = append(publishes, body)
		mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"stream": "MEMORY_INGEST", "seq": 1, "subject": body["subject"],
		})
	}))
	defer srv.Close()

	mesh := iomesh.New(iomesh.Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "dept.research",
	}, nil)

	rt := &Runtime{
		mesh: mesh,
		memory: MemoryConfig{
			Enabled:    true,
			AutoIngest: true,
			DualWrite:  true,
			Tenant:     "dept.research",
			Server:     "memory",
			SessionID:  "sess-dual",
		},
		sessionID: "sess-dual",
	}

	var events []Event
	rt.maybeAutoIngest(context.Background(), "hello user", "hello assistant", func(e Event) {
		events = append(events, e)
	})

	mu.Lock()
	defer mu.Unlock()
	if len(publishes) != 2 {
		t.Fatalf("publishes=%d events=%v", len(publishes), events)
	}
	// Monotonic session_seq 1 then 2.
	var seqs []int
	for _, p := range publishes {
		if p["subject"] != "dept.research.memory.ingest.turn" {
			t.Fatalf("subject=%v", p["subject"])
		}
		raw, err := base64.StdEncoding.DecodeString(p["payload"].(string))
		if err != nil {
			t.Fatal(err)
		}
		var env map[string]any
		if err := json.Unmarshal(raw, &env); err != nil {
			t.Fatal(err)
		}
		if env["type"] != "memory_ingest" {
			t.Fatalf("type=%v", env["type"])
		}
		if env["session_id"] != "sess-dual" {
			t.Fatalf("session_id=%v", env["session_id"])
		}
		seqs = append(seqs, int(env["session_seq"].(float64)))
	}
	if seqs[0] != 1 || seqs[1] != 2 {
		t.Fatalf("seqs=%v", seqs)
	}
	// dual_write events emitted
	var dualN int
	for _, e := range events {
		if strings.Contains(e.Text, "dual_write") {
			dualN++
		}
	}
	if dualN != 2 {
		t.Fatalf("dual events=%v", events)
	}
}

func TestMemoryIngestTurn_DualWriteWithoutMCP(t *testing.T) {
	var gotSubject string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotSubject, _ = body["subject"].(string)
		_ = json.NewEncoder(w).Encode(map[string]any{"stream": "MEMORY_INGEST", "seq": 1})
	}))
	defer srv.Close()

	mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: srv.URL, Tenant: "acme"}, nil)
	rt := &Runtime{
		mesh: mesh,
		memory: MemoryConfig{
			Enabled: true, DualWrite: true, Tenant: "acme", Server: "memory",
		},
	}
	out, err := rt.MemoryIngestTurn(context.Background(), "user", "note for palace")
	if err != nil {
		t.Fatalf("err=%v out=%q", err, out)
	}
	if !strings.Contains(out, "dual_write") {
		t.Fatalf("out=%q", out)
	}
	if gotSubject != "acme.memory.ingest.turn" {
		t.Fatalf("subject=%q", gotSubject)
	}
}

func TestMemoryStatusLine_DualWrite(t *testing.T) {
	rt := &Runtime{memory: MemoryConfig{Enabled: true, Server: "memory", DualWrite: true, AutoRecall: true}}
	s := rt.MemoryStatusLine()
	if !strings.Contains(s, "dual_write=true") {
		t.Fatalf("status=%q", s)
	}
	if !strings.Contains(s, "sync_http=false") {
		t.Fatalf("expected sync_http=false without mesh: %q", s)
	}
}

func TestAttachMemorySystemNote_DualWrite(t *testing.T) {
	rtr, err := router.New(router.DefaultModels(), router.DefaultModelName)
	if err != nil {
		t.Fatal(err)
	}
	rt, err := New(Config{Workspace: t.TempDir(), SubagentsEnabled: false}, rtr, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	rt.AttachMemory(MemoryConfig{Enabled: true, DualWrite: true, Server: "memory"})
	msgs := rt.Messages()
	if len(msgs) == 0 || !strings.Contains(msgs[0].Content, "dual_write=true") {
		t.Fatalf("expected dual_write in system note: %+v", msgs)
	}
}

// s1135: MemoryRelated prefers sync HTTP POST /v1/memory/related.
func TestMemoryRelated_PrefersSyncHTTP(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"memories": []map[string]any{
				{"id": "1", "summary": "alice teammate note", "score": 0.88, "hop_distance": 1},
				{"id": "2", "summary": "two hops out", "hop_distance": 2},
			},
		})
	}))
	defer srv.Close()

	mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: srv.URL, Tenant: "dept.research"}, nil)
	rt := &Runtime{
		mesh: mesh,
		memory: MemoryConfig{
			Enabled: true, Tenant: "dept.research", Server: "memory",
			Limit: 5, SessionID: "sess-rel", RelatedMaxHops: 2,
		},
	}
	out, err := rt.MemoryRelated(context.Background(), "person:alice", "teammate")
	if err != nil {
		t.Fatalf("MemoryRelated: %v", err)
	}
	if gotPath != "/v1/memory/related" {
		t.Fatalf("path=%q", gotPath)
	}
	if gotBody["seed_entity"] != "person:alice" || gotBody["query"] != "teammate" {
		t.Fatalf("body=%v", gotBody)
	}
	if gotBody["max_hops"] != float64(2) || gotBody["session_id"] != "sess-rel" {
		t.Fatalf("body hops/session=%v", gotBody)
	}
	if !strings.Contains(out, "alice teammate") || !strings.Contains(out, "[hop=1]") {
		t.Fatalf("out=%q", out)
	}
	if !strings.Contains(out, "[hop=2]") {
		t.Fatalf("expected hop2: %q", out)
	}
}

// s1135: per-call MaxHops overrides config RelatedMaxHops.
func TestMemoryRelated_MaxHopsOverride(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{"memories": []any{}})
	}))
	defer srv.Close()

	mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: srv.URL, Tenant: "t"}, nil)
	rt := &Runtime{
		mesh: mesh,
		memory: MemoryConfig{
			Enabled: true, Tenant: "t", Server: "memory", RelatedMaxHops: 2,
		},
	}
	_, err := rt.MemoryRelated(context.Background(), "person:bob", "", MemoryRelatedOpts{MaxHops: 1, Limit: 3})
	if err != nil {
		t.Fatal(err)
	}
	if gotBody["max_hops"] != float64(1) || gotBody["limit"] != float64(3) {
		t.Fatalf("body=%v", gotBody)
	}
}

// s1135: sync 404 + no MCP → error (MCP fallback path when unavailable).
func TestMemoryRelated_SyncFailsMCPUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: srv.URL, Tenant: "t"}, nil)
	rt := &Runtime{
		mesh:   mesh,
		memory: MemoryConfig{Enabled: true, Server: "memory", Tenant: "t"},
		mcp:    mcp.NewManagerEmpty(nil),
	}
	_, err := rt.MemoryRelated(context.Background(), "person:alice", "")
	if err == nil {
		t.Fatal("expected error when sync 404 and MCP missing")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Fatalf("err=%v", err)
	}
}

// s1135: requires seed or query; hooks disabled error.
func TestMemoryRelated_RequiresSeedOrQuery(t *testing.T) {
	rt := &Runtime{memory: MemoryConfig{Enabled: true, Server: "memory"}}
	_, err := rt.MemoryRelated(context.Background(), "", "")
	if err == nil || !strings.Contains(err.Error(), "seed_entity or query") {
		t.Fatalf("err=%v", err)
	}
	rt2 := &Runtime{memory: DefaultMemoryConfig()}
	_, err = rt2.MemoryRelated(context.Background(), "person:alice", "")
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("err=%v", err)
	}
}

// s1135: default RelatedMaxHops=2; auto-recall path unchanged (MemoryRecall still used).
func TestDefaultMemoryConfig_RelatedMaxHops(t *testing.T) {
	d := DefaultMemoryConfig()
	if d.RelatedMaxHops != 2 {
		t.Fatalf("RelatedMaxHops=%d want 2", d.RelatedMaxHops)
	}
	// Honesty pin: multi-hop is opt-in — RelatedMaxHops does not enable auto multi-hop.
	if !d.AutoRecall {
		t.Fatal("AutoRecall still default true (single-hop MemoryRecall)")
	}
}
