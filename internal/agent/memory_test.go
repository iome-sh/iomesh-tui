package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
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

// s1200: MemoryOpsDigest prefers sync HTTP POST /v1/memory/ops_digest and formats patterns+receipts+honesty.
func TestMemoryOpsDigest_PrefersSyncHTTP(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"window":  "day",
			"horizon": "ops",
			"as_of":   "2026-08-04T12:00:00Z",
			"since":   "2026-08-03T12:00:00Z",
			"honesty": map[string]any{
				"ops_pulse":          "ga_path",
				"knowledge":          "beta",
				"analytical":         "beta",
				"never_invent_ga":    true,
				"dual_write_default": "off",
				"book_demo":          "off",
				"note":               "Ops digests synthesize live pulse",
			},
			"patterns": []map[string]any{
				{"id": "p1", "kind": "burst", "subject": "dept.ops", "summary": "deploy burst", "score": 0.9, "count": 5, "window": "15m"},
			},
			"receipts": []map[string]any{
				{"id": "r1", "event_time": "2026-08-04T10:00:00Z", "summary": "deploy finished customer said outage", "pointer": "https://tickets.example/INC-9"},
			},
			"decision_stub": map[string]any{"pattern": "dept.ops"},
		})
	}))
	defer srv.Close()

	mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: srv.URL, Tenant: "dept.research"}, nil)
	rt := &Runtime{
		mesh: mesh,
		memory: MemoryConfig{
			Enabled: true, Tenant: "dept.research", Server: "memory",
			Limit: 5, SessionID: "sess-digest",
		},
	}
	out, err := rt.MemoryOpsDigest(context.Background())
	if err != nil {
		t.Fatalf("MemoryOpsDigest: %v", err)
	}
	if gotPath != "/v1/memory/ops_digest" {
		t.Fatalf("path=%q", gotPath)
	}
	if gotBody["window"] != "day" || gotBody["horizon"] != "ops" || gotBody["tenant_id"] != "dept.research" {
		t.Fatalf("body=%v", gotBody)
	}
	if !strings.Contains(out, "dept.ops") || !strings.Contains(out, "n=5") || !strings.Contains(out, "window=15m") {
		t.Fatalf("pattern missing: %q", out)
	}
	if !strings.Contains(out, "pointer=https://tickets.example/INC-9") || !strings.Contains(out, "hash=") {
		t.Fatalf("receipt pointer/hash missing: %q", out)
	}
	if strings.Contains(out, "customer said outage") || strings.Contains(out, "deploy finished") {
		t.Fatalf("raw customer receipt text leaked: %q", out)
	}
	if !strings.Contains(out, "honesty:") || !strings.Contains(out, "ga_path") || !strings.Contains(out, "never_invent_ga=true") {
		t.Fatalf("honesty missing: %q", out)
	}
	if !strings.Contains(out, "not Memory GA") || !strings.Contains(out, "catalog list ≠ consume") {
		t.Fatalf("honesty pin missing: %q", out)
	}
	if !strings.Contains(out, "dual_write=off") {
		t.Fatalf("dual_write pin missing: %q", out)
	}
	if strings.Contains(out, "require-sources:") {
		t.Fatalf("default digest without --require-sources must stay existing behavior: %q", out)
	}
}

// s1200: per-call Window/Horizon/Limit override defaults.
func TestMemoryOpsDigest_OptsOverride(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"window": "week", "horizon": "knowledge", "as_of": "now",
			"honesty":  map[string]any{"ops_pulse": "ga_path", "knowledge": "beta", "analytical": "beta", "never_invent_ga": true, "dual_write_default": "off"},
			"patterns": []any{}, "receipts": []any{},
		})
	}))
	defer srv.Close()

	mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: srv.URL, Tenant: "t"}, nil)
	rt := &Runtime{
		mesh:   mesh,
		memory: MemoryConfig{Enabled: true, Tenant: "t", Server: "memory"},
	}
	out, err := rt.MemoryOpsDigest(context.Background(), MemoryOpsDigestOpts{
		Window: "week", Horizon: "knowledge", Limit: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotBody["window"] != "week" || gotBody["horizon"] != "knowledge" || gotBody["limit"] != float64(3) {
		t.Fatalf("body=%v", gotBody)
	}
	if !strings.Contains(out, "window=week") || !strings.Contains(out, "horizon=knowledge") {
		t.Fatalf("out=%q", out)
	}
	if !strings.Contains(out, "insufficient-signal") || !strings.Contains(out, "nothing reliable today") {
		t.Fatalf("expected insufficient-signal on empty patterns: %q", out)
	}
	if !strings.Contains(out, "receipts: (none)") {
		t.Fatalf("empty receipts: %q", out)
	}
}

// s1200: sync 404 + no MCP → error.
func TestMemoryOpsDigest_SyncFailsMCPUnavailable(t *testing.T) {
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
	_, err := rt.MemoryOpsDigest(context.Background())
	if err == nil {
		t.Fatal("expected error when sync 404 and MCP missing")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Fatalf("err=%v", err)
	}
}

// s1200: hooks disabled error.
func TestMemoryOpsDigest_Disabled(t *testing.T) {
	rt := &Runtime{memory: DefaultMemoryConfig()}
	_, err := rt.MemoryOpsDigest(context.Background())
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("err=%v", err)
	}
}

// #373 / #370: classify receipt source_hint — catalog/grant/external never count as mesh|private.
func TestClassifyDigestSourceHint(t *testing.T) {
	cases := []struct {
		hint string
		want string
	}{
		{"mesh", DigestSourceMesh},
		{"mesh_stream", DigestSourceMesh},
		{"MESH-INCIDENTS", DigestSourceMesh},
		{"consume", DigestSourceMesh},
		{"palace_timeline", DigestSourcePrivate},
		{"private_rca", DigestSourcePrivate},
		{"catalog", DigestSourceCatalog},
		{"catalog_only", DigestSourceCatalog},
		{"grant", DigestSourceGrant},
		{"grant_only", DigestSourceGrant},
		{"entitlement", DigestSourceGrant},
		{"external", DigestSourceExternal},
		{"sponsored", DigestSourceExternal},
		{"source=external", DigestSourceExternal},
		{"source=sponsored", DigestSourceExternal},
		{"external_feed", DigestSourceExternal},
		{"sponsored_stream", DigestSourceExternal},
		{"sponsored_consume", DigestSourceExternal},
		{"demand_feed", DigestSourceExternal},
		{"tam_color", DigestSourceExternal},
		{"", ""},
		{"unknown_widget", ""},
	}
	for _, tc := range cases {
		if got := ClassifyDigestSourceHint(tc.hint); got != tc.want {
			t.Fatalf("hint=%q got=%q want=%q", tc.hint, got, tc.want)
		}
	}
}

func TestParseRequireSourcesList(t *testing.T) {
	got, errMsg := ParseRequireSourcesList("mesh,private")
	if errMsg != "" || len(got) != 2 || got[0] != "mesh" || got[1] != "private" {
		t.Fatalf("got=%v err=%q", got, errMsg)
	}
	got2, err2 := ParseRequireSourcesList(" private , mesh , private ")
	if err2 != "" || len(got2) != 2 || got2[0] != "private" || got2[1] != "mesh" {
		t.Fatalf("dedupe got=%v err=%q", got2, err2)
	}
	if _, err := ParseRequireSourcesList("mesh,catalog"); err == "" {
		t.Fatal("expected reject catalog in require-sources")
	}
	if _, err := ParseRequireSourcesList("mesh,external"); err == "" {
		t.Fatal("expected reject external in require-sources")
	}
	if _, err := ParseRequireSourcesList("external"); err == "" {
		t.Fatal("expected reject external-only require-sources")
	}
	if _, err := ParseRequireSourcesList(""); err == "" {
		t.Fatal("expected reject empty")
	}
	for _, junk := range []string{"grant", "mesh,foo", "junk", "mesh,grant"} {
		if _, err := ParseRequireSourcesList(junk); err == "" {
			t.Fatalf("expected reject junk %q", junk)
		}
	}
}

// #373: cite-both ok when mesh + private receipts present.
func TestFormatRequireSourcesCheck_CiteBothOK(t *testing.T) {
	res := &iomesh.MemoryOpsDigestResult{
		Receipts: []iomesh.MemoryOpsDigestReceipt{
			{ID: "m1", Summary: "P2 checkout p95", SourceHint: "mesh_stream"},
			{ID: "p1", Summary: "private RCA note", SourceHint: "palace_timeline"},
			{ID: "c1", Summary: "catalog product list", SourceHint: "catalog"},
		},
	}
	out := FormatRequireSourcesCheck(res, []string{"mesh", "private"})
	if !strings.Contains(out, "require-sources: ok") {
		t.Fatalf("want ok: %q", out)
	}
	if !strings.Contains(out, "mesh=P2 checkout p95") || !strings.Contains(out, "private=private RCA note") {
		t.Fatalf("want cites: %q", out)
	}
	if !strings.Contains(out, "dual_write OFF") || !strings.Contains(out, "not Memory GA") {
		t.Fatalf("honesty pin missing: %q", out)
	}
	if strings.Contains(out, "miss") {
		t.Fatalf("must not miss when both present: %q", out)
	}
}

// #373: private-only → explicit miss for mesh; catalog/grant do not satisfy.
func TestFormatRequireSourcesCheck_MissMesh(t *testing.T) {
	res := &iomesh.MemoryOpsDigestResult{
		Receipts: []iomesh.MemoryOpsDigestReceipt{
			{ID: "p1", Summary: "local RCA", SourceHint: "palace_timeline"},
			{ID: "c1", Summary: "portal catalog row", SourceHint: "catalog_only"},
			{ID: "g1", Summary: "grant entitlement", SourceHint: "grant"},
		},
	}
	out := FormatRequireSourcesCheck(res, []string{"mesh", "private"})
	if !strings.Contains(out, "require-sources: miss") || !strings.Contains(out, "missing=mesh") {
		t.Fatalf("want mesh miss: %q", out)
	}
	if !strings.Contains(out, "cited=private") {
		t.Fatalf("want private cited: %q", out)
	}
	if !strings.Contains(out, "catalog/grant do not satisfy cite-both") {
		t.Fatalf("want catalog/grant pin: %q", out)
	}
	if strings.Contains(out, "require-sources: ok") {
		t.Fatalf("must not ok: %q", out)
	}
}

// #373: catalog-only / grant-only never counts as cite-both.
func TestFormatRequireSourcesCheck_CatalogGrantOnly(t *testing.T) {
	res := &iomesh.MemoryOpsDigestResult{
		Receipts: []iomesh.MemoryOpsDigestReceipt{
			{ID: "c1", Summary: "catalog list", SourceHint: "catalog"},
			{ID: "g1", Summary: "grant only", SourceHint: "grant_only"},
		},
	}
	out := FormatRequireSourcesCheck(res, []string{"mesh", "private"})
	if !strings.Contains(out, "miss") || !strings.Contains(out, "missing=mesh,private") {
		t.Fatalf("want both missing: %q", out)
	}
	if !strings.Contains(out, "cited=(none)") {
		t.Fatalf("want cited none: %q", out)
	}
	if !strings.Contains(out, "catalog/grant do not satisfy cite-both") {
		t.Fatalf("want catalog/grant pin: %q", out)
	}
}

// #373: mesh-only → explicit miss for private (catalog/grant pin only when those hints appear).
func TestFormatRequireSourcesCheck_MissPrivate(t *testing.T) {
	res := &iomesh.MemoryOpsDigestResult{
		Receipts: []iomesh.MemoryOpsDigestReceipt{
			{ID: "m1", Summary: "mesh incident INC-9", SourceHint: "mesh_stream"},
		},
	}
	out := FormatRequireSourcesCheck(res, []string{"mesh", "private"})
	if !strings.Contains(out, "require-sources: miss") || !strings.Contains(out, "missing=private") {
		t.Fatalf("want private miss: %q", out)
	}
	if !strings.Contains(out, "cited=mesh") {
		t.Fatalf("want mesh cited: %q", out)
	}
	if strings.Contains(out, "catalog/grant do not satisfy cite-both") {
		t.Fatalf("catalog/grant pin only when those hints present: %q", out)
	}
	if !strings.Contains(out, "dual_write OFF") || !strings.Contains(out, "not Memory GA") {
		t.Fatalf("honesty pin missing: %q", out)
	}
}

// #373: MemoryOpsDigest with --require-sources prefixes miss when palace-only.
func TestMemoryOpsDigest_RequireSourcesMiss(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"window": "day", "horizon": "ops",
			"honesty": map[string]any{
				"ops_pulse": "ga_path", "knowledge": "beta", "analytical": "beta",
				"never_invent_ga": true, "dual_write_default": "off",
			},
			"patterns": []any{},
			"receipts": []map[string]any{
				{"id": "r1", "summary": "palace note", "source_hint": "palace_timeline"},
			},
		})
	}))
	defer srv.Close()

	mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: srv.URL, Tenant: "t"}, nil)
	rt := &Runtime{
		mesh:   mesh,
		memory: MemoryConfig{Enabled: true, Tenant: "t", Server: "memory", DualWrite: false},
	}
	out, err := rt.MemoryOpsDigest(context.Background(), MemoryOpsDigestOpts{
		RequireSources: []string{"mesh", "private"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "require-sources: miss") {
		t.Fatalf("want miss prefix: %q", out)
	}
	if !strings.Contains(out, "missing=mesh") || !strings.Contains(out, "cited=private") {
		t.Fatalf("want private-only cite: %q", out)
	}
	if strings.Contains(out, "palace note") {
		t.Fatalf("receipts must not paste raw customer summary: %q", out)
	}
	if !strings.Contains(out, "source=palace_timeline") {
		t.Fatalf("want digest receipt source hint: %q", out)
	}
	if rt.memory.DualWrite {
		t.Fatal("dual_write must remain OFF")
	}
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "dual_write ON") {
		t.Fatalf("must not invent GA / dual_write ON: %q", out)
	}
}

// #373: MemoryOpsDigest cite-both ok when both source_hints present.
func TestMemoryOpsDigest_RequireSourcesCiteBoth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"window": "day", "horizon": "ops",
			"honesty": map[string]any{
				"ops_pulse": "ga_path", "never_invent_ga": true, "dual_write_default": "off",
			},
			"patterns": []any{},
			"receipts": []map[string]any{
				{"id": "m1", "summary": "mesh incident INC-9", "source_hint": "mesh"},
				{"id": "p1", "summary": "private RCA", "source_hint": "private"},
			},
		})
	}))
	defer srv.Close()

	mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: srv.URL, Tenant: "t"}, nil)
	rt := &Runtime{
		mesh:   mesh,
		memory: MemoryConfig{Enabled: true, Tenant: "t", Server: "memory", DualWrite: false},
	}
	out, err := rt.MemoryOpsDigest(context.Background(), MemoryOpsDigestOpts{
		RequireSources: []string{"mesh", "private"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "require-sources: ok") {
		t.Fatalf("want ok prefix: %q", out)
	}
	if !strings.Contains(out, "mesh=mesh incident INC-9") || !strings.Contains(out, "private=private RCA") {
		t.Fatalf("want both cites: %q", out)
	}
	if !strings.Contains(out, "dual_write OFF") || !strings.Contains(out, "not Memory GA") {
		t.Fatalf("honesty pin missing: %q", out)
	}
	if rt.memory.DualWrite {
		t.Fatal("dual_write must remain OFF")
	}
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "dual_write ON") {
		t.Fatalf("must not invent GA / dual_write ON: %q", out)
	}
}

// #373: catalog-only / grant-only HTTP digest never satisfies cite-both; mesh-only misses private.
func TestMemoryOpsDigest_RequireSourcesCatalogGrantAndMeshOnly(t *testing.T) {
	cases := []struct {
		name     string
		receipts []map[string]any
		want     []string
		not      []string
	}{
		{
			name: "catalog_only",
			receipts: []map[string]any{
				{"id": "c1", "summary": "catalog list", "source_hint": "catalog"},
			},
			want: []string{
				"require-sources: miss", "missing=mesh,private", "cited=(none)",
				"catalog/grant do not satisfy cite-both", "dual_write OFF", "not Memory GA",
			},
			not: []string{"require-sources: ok"},
		},
		{
			name: "grant_only",
			receipts: []map[string]any{
				{"id": "g1", "summary": "grant entitlement", "source_hint": "grant_only"},
			},
			want: []string{
				"require-sources: miss", "missing=mesh,private",
				"catalog/grant do not satisfy cite-both", "dual_write OFF",
			},
			not: []string{"require-sources: ok"},
		},
		{
			name: "mesh_only_missing_private",
			receipts: []map[string]any{
				{"id": "m1", "summary": "mesh incident INC-9", "source_hint": "mesh_stream"},
			},
			want: []string{"require-sources: miss", "missing=private", "cited=mesh", "dual_write OFF"},
			not:  []string{"require-sources: ok", "catalog/grant do not satisfy cite-both"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"window": "day", "horizon": "ops",
					"honesty": map[string]any{
						"ops_pulse": "ga_path", "never_invent_ga": true, "dual_write_default": "off",
					},
					"patterns": []any{},
					"receipts": tc.receipts,
				})
			}))
			defer srv.Close()

			mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: srv.URL, Tenant: "t"}, nil)
			rt := &Runtime{
				mesh:   mesh,
				memory: MemoryConfig{Enabled: true, Tenant: "t", Server: "memory", DualWrite: false},
			}
			out, err := rt.MemoryOpsDigest(context.Background(), MemoryOpsDigestOpts{
				RequireSources: []string{"mesh", "private"},
			})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(strings.TrimSpace(out), "require-sources: miss") {
				t.Fatalf("want miss prefix: %q", out)
			}
			for _, n := range tc.want {
				if !strings.Contains(out, n) {
					t.Fatalf("missing %q in %q", n, out)
				}
			}
			for _, n := range tc.not {
				if strings.Contains(out, n) {
					t.Fatalf("must not contain %q: %q", n, out)
				}
			}
			if rt.memory.DualWrite {
				t.Fatal("dual_write must remain OFF")
			}
		})
	}
}

// #373: MCP JSON fallback still classifies source_hint for cite-both.
func TestParseOpsDigestJSON_RequireSources(t *testing.T) {
	raw := `{
		"window": "day",
		"horizon": "ops",
		"honesty": {"ops_pulse": "ga_path", "never_invent_ga": true, "dual_write_default": "off"},
		"patterns": [],
		"receipts": [
			{"id": "c1", "summary": "catalog list", "source_hint": "catalog"},
			{"id": "g1", "summary": "grant only", "source_hint": "grant"}
		]
	}`
	res, formatted := parseOpsDigestJSON(raw, 6000)
	if res == nil || formatted == "" {
		t.Fatalf("parse failed res=%v formatted=%q", res, formatted)
	}
	out := applyRequireSources(formatted, res, []string{"mesh", "private"})
	if !strings.HasPrefix(strings.TrimSpace(out), "require-sources: miss") {
		t.Fatalf("want miss prefix: %q", out)
	}
	if !strings.Contains(out, "catalog/grant do not satisfy cite-both") {
		t.Fatalf("want catalog/grant pin: %q", out)
	}
	if !strings.Contains(out, "dual_write OFF") || !strings.Contains(out, "not Memory GA") {
		t.Fatalf("honesty pin missing: %q", out)
	}

	both := `{
		"window": "day", "horizon": "ops",
		"receipts": [
			{"id": "m1", "summary": "mesh incident INC-9", "source_hint": "mesh"},
			{"id": "p1", "summary": "private RCA", "source_hint": "private"}
		]
	}`
	res2, formatted2 := parseOpsDigestJSON(both, 6000)
	ok := applyRequireSources(formatted2, res2, []string{"mesh", "private"})
	if !strings.HasPrefix(strings.TrimSpace(ok), "require-sources: ok") {
		t.Fatalf("want ok prefix: %q", ok)
	}
	if !strings.Contains(ok, "mesh=mesh incident INC-9") || !strings.Contains(ok, "private=private RCA") {
		t.Fatalf("want both cites: %q", ok)
	}
}

// #370: source=external never satisfies cite-both mesh+private.
func TestFormatRequireSourcesCheck_ExternalNeverCiteBoth(t *testing.T) {
	res := &iomesh.MemoryOpsDigestResult{
		Receipts: []iomesh.MemoryOpsDigestReceipt{
			{ID: "e1", Summary: "sponsored TAM color", SourceHint: "source=external"},
			{ID: "e2", Summary: "demand feed", SourceHint: "sponsored_stream"},
			{ID: "p1", Summary: "private RCA", SourceHint: "palace_timeline"},
		},
	}
	out := FormatRequireSourcesCheck(res, []string{"mesh", "private"})
	if !strings.Contains(out, "require-sources: miss") || !strings.Contains(out, "missing=mesh") {
		t.Fatalf("want mesh miss: %q", out)
	}
	if !strings.Contains(out, "cited=private") {
		t.Fatalf("want private cited: %q", out)
	}
	if !strings.Contains(out, "external/sponsored do not satisfy cite-both") {
		t.Fatalf("want external pin: %q", out)
	}
	if strings.Contains(out, "require-sources: ok") {
		t.Fatalf("must not ok: %q", out)
	}
}

// #370: MemoryOpsDigest httptest — external-only receipts miss cite-both; first-party consume fills mesh.
func TestMemoryOpsDigest_ExternalNeverFillsMesh(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"window": "day", "horizon": "ops",
			"honesty": map[string]any{
				"ops_pulse": "ga_path", "never_invent_ga": true, "dual_write_default": "off",
			},
			"patterns": []map[string]any{
				{"kind": "stall", "delta_kind": "stall", "subject": "dept.ops", "summary": "new checkout stall", "count": 4, "window": "24h"},
			},
			"receipts": []map[string]any{
				{"id": "e1", "summary": "sponsored TAM", "source_hint": "external", "pointer": "https://feed.example/tam"},
				{"id": "p1", "summary": "palace note", "source_hint": "private"},
			},
		})
	}))
	defer srv.Close()

	mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: srv.URL, Tenant: "t"}, nil)
	rt := &Runtime{
		mesh:   mesh,
		memory: MemoryConfig{Enabled: true, Tenant: "t", Server: "memory", DualWrite: false},
	}
	out, err := rt.MemoryOpsDigest(context.Background(), MemoryOpsDigestOpts{
		RequireSources: []string{"mesh", "private"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "require-sources: miss") {
		t.Fatalf("want miss prefix: %q", out)
	}
	if !strings.Contains(out, "missing=mesh") || !strings.Contains(out, "cited=private") {
		t.Fatalf("want private-only cite: %q", out)
	}
	if !strings.Contains(out, "external/sponsored do not satisfy cite-both") {
		t.Fatalf("want external pin: %q", out)
	}
	if !strings.Contains(out, digestExternalPaneLabel) {
		t.Fatalf("want external third pane: %q", out)
	}
	if strings.Contains(out, "sponsored TAM") {
		t.Fatalf("must not paste external raw text: %q", out)
	}
	if rt.memory.DualWrite {
		t.Fatal("dual_write must remain OFF")
	}
}

func TestMemoryOpsDigest_FirstPartyConsumeFillsMesh(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"window": "day", "horizon": "ops",
			"honesty": map[string]any{
				"ops_pulse": "ga_path", "never_invent_ga": true, "dual_write_default": "off",
			},
			"patterns": []map[string]any{
				{"kind": "language", "delta_kind": "language", "subject": "dept.ops", "summary": "new pager language", "count": 2, "window": "24h"},
			},
			"receipts": []map[string]any{
				{"id": "m1", "summary": "consumed GitHub event", "source_hint": "consume", "pointer": "https://tickets.example/INC-1"},
				{"id": "p1", "summary": "private RCA", "source_hint": "private"},
				{"id": "e1", "summary": "sponsored color", "source_hint": "sponsored"},
			},
		})
	}))
	defer srv.Close()

	mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: srv.URL, Tenant: "t"}, nil)
	rt := &Runtime{
		mesh:   mesh,
		memory: MemoryConfig{Enabled: true, Tenant: "t", Server: "memory"},
	}
	out, err := rt.MemoryOpsDigest(context.Background(), MemoryOpsDigestOpts{
		RequireSources: []string{"mesh", "private"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "require-sources: ok") {
		t.Fatalf("want ok prefix: %q", out)
	}
	if !strings.Contains(out, "cited=mesh,private") {
		t.Fatalf("want both cited: %q", out)
	}
	if !strings.Contains(out, "external color is a third pane") {
		t.Fatalf("want third-pane pin when external also present: %q", out)
	}
	if !strings.Contains(out, digestExternalPaneLabel) {
		t.Fatalf("want external pane: %q", out)
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

// s1281: PreferShorterHops pass-through — false/true sent; nil omits (kernel default true).
func TestMemoryRelated_PreferShorterHopsBody(t *testing.T) {
	run := func(t *testing.T, opts MemoryRelatedOpts) map[string]any {
		t.Helper()
		var gotBody map[string]any
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			_ = json.NewEncoder(w).Encode(map[string]any{"memories": []any{}})
		}))
		t.Cleanup(srv.Close)

		mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: srv.URL, Tenant: "t"}, nil)
		rt := &Runtime{
			mesh: mesh,
			memory: MemoryConfig{
				Enabled: true, Tenant: "t", Server: "memory", RelatedMaxHops: 2,
			},
		}
		_, err := rt.MemoryRelated(context.Background(), "person:alice", "", opts)
		if err != nil {
			t.Fatal(err)
		}
		return gotBody
	}

	t.Run("false_legacy_seed_first", func(t *testing.T) {
		b := false
		body := run(t, MemoryRelatedOpts{PreferShorterHops: &b})
		v, ok := body["prefer_shorter_hops"]
		if !ok {
			t.Fatalf("expected prefer_shorter_hops in body: %v", body)
		}
		if v != false {
			t.Fatalf("prefer_shorter_hops=%v want false", v)
		}
	})
	t.Run("true_explicit", func(t *testing.T) {
		b := true
		body := run(t, MemoryRelatedOpts{PreferShorterHops: &b})
		v, ok := body["prefer_shorter_hops"]
		if !ok {
			t.Fatalf("expected prefer_shorter_hops in body: %v", body)
		}
		if v != true {
			t.Fatalf("prefer_shorter_hops=%v want true", v)
		}
	})
	t.Run("nil_omitted_kernel_default", func(t *testing.T) {
		body := run(t, MemoryRelatedOpts{})
		if _, ok := body["prefer_shorter_hops"]; ok {
			t.Fatalf("prefer_shorter_hops must be omitted when nil: %v", body)
		}
	})
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

// s1276: formatFactsAsOfJSON residual-honest fixture {as_of, facts:[{summary,score,...}]}.
func TestFormatFactsAsOfJSON_Fixture(t *testing.T) {
	raw := `{
		"as_of": "2026-08-04T12:00:00Z",
		"facts": [
			{"id": "f1", "summary": "alice role was engineer", "score": 0.91},
			{"id": "f2", "summary": "project alpha active", "full": "longer", "score": 0.7}
		]
	}`
	out := formatFactsAsOfJSON(raw, 6000)
	if out == "" {
		t.Fatal("expected formatted output")
	}
	if !strings.Contains(out, "facts-as-of as_of=2026-08-04T12:00:00Z") {
		t.Fatalf("header: %q", out)
	}
	if !strings.Contains(out, "alice role was engineer") || !strings.Contains(out, "[0.91]") {
		t.Fatalf("fact1: %q", out)
	}
	if !strings.Contains(out, "project alpha active") {
		t.Fatalf("fact2: %q", out)
	}
	if !strings.Contains(out, "bi-temporal lite") || !strings.Contains(out, "not full dual-clock Graphiti") {
		t.Fatalf("honesty pin missing: %q", out)
	}
	if !strings.Contains(out, "not Memory GA") || !strings.Contains(out, "dual_write OFF") {
		t.Fatalf("Memory GA / dual_write pin missing: %q", out)
	}
}

// s1276: empty facts is residual-honest empty — never invent memories.
func TestFormatFactsAsOf_EmptyHonest(t *testing.T) {
	out := formatFactsAsOf("2026-01-01T00:00:00Z", nil, 6000)
	if !strings.Contains(out, "facts: (none)") {
		t.Fatalf("empty: %q", out)
	}
	if !strings.Contains(out, factsAsOfHonestyFooter) {
		t.Fatalf("honesty: %q", out)
	}
	// Empty facts array from JSON.
	out2 := formatFactsAsOfJSON(`{"as_of":"2026-06-01T00:00:00Z","facts":[]}`, 6000)
	if !strings.Contains(out2, "facts: (none)") || !strings.Contains(out2, "as_of=2026-06-01T00:00:00Z") {
		t.Fatalf("empty json: %q", out2)
	}
}

// s1276: offline MCP → residual-honest fail-open message (not invent empty success).
func TestMemoryFactsAsOf_OfflineFailOpen(t *testing.T) {
	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, Server: "memory", Tenant: "dept.research"},
		mcp:    mcp.NewManagerEmpty(nil),
	}
	out, err := rt.MemoryFactsAsOf(context.Background(), MemoryFactsAsOfOpts{
		AsOf: "2026-08-04T12:00:00Z",
	})
	if err != nil {
		t.Fatalf("expected fail-open nil err, got %v", err)
	}
	if !strings.Contains(out, "unavailable") || !strings.Contains(out, "not connected") {
		t.Fatalf("offline: %q", out)
	}
	if !strings.Contains(out, "MCP-first") || !strings.Contains(out, "bi-temporal lite") {
		t.Fatalf("honesty offline: %q", out)
	}
	if !strings.Contains(out, "empty ≠ invent memories") {
		t.Fatalf("empty≠invent pin: %q", out)
	}
	// Must not look like successful empty listing alone.
	if strings.Contains(out, "facts: (none)") && !strings.Contains(out, "unavailable") {
		t.Fatalf("must not invent empty-success: %q", out)
	}
}

// s1276: requires as_of; hooks disabled error; bad RFC3339 rejected.
func TestMemoryFactsAsOf_Validation(t *testing.T) {
	rt := &Runtime{memory: MemoryConfig{Enabled: true, Server: "memory"}}
	_, err := rt.MemoryFactsAsOf(context.Background(), MemoryFactsAsOfOpts{})
	if err == nil || !strings.Contains(err.Error(), "as_of required") {
		t.Fatalf("err=%v", err)
	}
	_, err = rt.MemoryFactsAsOf(context.Background(), MemoryFactsAsOfOpts{AsOf: "not-a-time"})
	if err == nil || !strings.Contains(err.Error(), "RFC3339") {
		t.Fatalf("bad as_of err=%v", err)
	}
	rt2 := &Runtime{memory: DefaultMemoryConfig()}
	_, err = rt2.MemoryFactsAsOf(context.Background(), MemoryFactsAsOfOpts{AsOf: "2026-08-04T12:00:00Z"})
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("disabled err=%v", err)
	}
}

// s1276: formatFactsAsOfJSON returns empty on non-JSON (caller may pass through).
func TestFormatFactsAsOfJSON_NonJSON(t *testing.T) {
	if got := formatFactsAsOfJSON("not json", 100); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := formatFactsAsOfJSON("", 100); got != "" {
		t.Fatalf("got %q", got)
	}
}

// s1282: formatSupersedeJSON residual-honest fixture {entity, as_of, superseded_count}.
func TestFormatSupersedeJSON_Fixture(t *testing.T) {
	raw := `{"entity":"person:alice","as_of":"2026-08-04T12:00:00Z","superseded_count":3}`
	out := formatSupersedeJSON(raw)
	if out == "" {
		t.Fatal("expected formatted output")
	}
	if !strings.Contains(out, "supersede entity=person:alice") {
		t.Fatalf("entity: %q", out)
	}
	if !strings.Contains(out, "as_of=2026-08-04T12:00:00Z") {
		t.Fatalf("as_of: %q", out)
	}
	if !strings.Contains(out, "superseded_count: 3") {
		t.Fatalf("count: %q", out)
	}
	if !strings.Contains(out, "A3 lite supersede") || !strings.Contains(out, "not NLP contradiction") {
		t.Fatalf("honesty pin missing: %q", out)
	}
	if !strings.Contains(out, "not full dual-clock Graphiti") || !strings.Contains(out, "not Memory GA") {
		t.Fatalf("Graphiti/GA pin missing: %q", out)
	}
	if !strings.Contains(out, "dual_write OFF") || !strings.Contains(out, "mutating") {
		t.Fatalf("dual_write/mutating pin missing: %q", out)
	}
	// Zero count is honest (real wire), not invent offline.
	out0 := formatSupersedeJSON(`{"entity":"org:acme","as_of":"2026-01-01T00:00:00Z","superseded_count":0}`)
	if !strings.Contains(out0, "superseded_count: 0") || !strings.Contains(out0, "org:acme") {
		t.Fatalf("zero count: %q", out0)
	}
}

// s1282: formatSupersedeJSON returns empty on non-JSON (caller may pass through).
func TestFormatSupersedeJSON_NonJSON(t *testing.T) {
	if got := formatSupersedeJSON("not json"); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := formatSupersedeJSON(""); got != "" {
		t.Fatalf("got %q", got)
	}
}

// s1282: without Confirm → residual-honest refusal string; must NOT require MCP (no call).
func TestMemorySupersede_RefuseWithoutConfirm(t *testing.T) {
	// Empty MCP manager: if Confirm were true we'd go offline-fail-open; without Confirm
	// we must refuse before any MCP readiness check path mutates or invents count.
	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, Server: "memory", Tenant: "dept.research"},
		mcp:    mcp.NewManagerEmpty(nil),
	}
	out, err := rt.MemorySupersede(context.Background(), MemorySupersedeOpts{
		Entity:  "person:alice",
		AsOf:    "2026-08-04T12:00:00Z",
		Confirm: false,
	})
	if err != nil {
		t.Fatalf("expected residual-honest refusal nil err, got %v", err)
	}
	if !strings.Contains(out, "refused") || !strings.Contains(out, "HITL") {
		t.Fatalf("refusal: %q", out)
	}
	if !strings.Contains(out, "--i-confirm") {
		t.Fatalf("must mention --i-confirm: %q", out)
	}
	if !strings.Contains(out, "person:alice") {
		t.Fatalf("entity: %q", out)
	}
	if !strings.Contains(out, supersedeHonestyFooter) && !strings.Contains(out, "A3 lite supersede") {
		t.Fatalf("honesty: %q", out)
	}
	// Must not look like success with a count.
	if strings.Contains(out, "superseded_count:") {
		t.Fatalf("must not invent superseded_count on refuse: %q", out)
	}
	if strings.Contains(out, "unavailable") {
		// Refuse must short-circuit before offline messaging.
		t.Fatalf("must not reach offline path without Confirm: %q", out)
	}
}

// s1282: Confirm + offline MCP → residual-honest fail-open (not invent count).
func TestMemorySupersede_OfflineFailOpen(t *testing.T) {
	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, Server: "memory", Tenant: "dept.research"},
		mcp:    mcp.NewManagerEmpty(nil),
	}
	out, err := rt.MemorySupersede(context.Background(), MemorySupersedeOpts{
		Entity:  "person:alice",
		AsOf:    "2026-08-04T12:00:00Z",
		Confirm: true,
	})
	if err != nil {
		t.Fatalf("expected fail-open nil err, got %v", err)
	}
	if !strings.Contains(out, "unavailable") || !strings.Contains(out, "not connected") {
		t.Fatalf("offline: %q", out)
	}
	if !strings.Contains(out, "MCP-first") || !strings.Contains(out, "A3 lite supersede") {
		t.Fatalf("honesty offline: %q", out)
	}
	if !strings.Contains(out, "not inventing superseded_count") {
		t.Fatalf("not-invent pin: %q", out)
	}
	// Must not look like successful supersede with a count.
	if strings.Contains(out, "superseded_count:") {
		t.Fatalf("must not invent supersede success offline: %q", out)
	}
}

// s1282: requires entity; hooks disabled; bad as_of rejected; Confirm gate is string not error.
func TestMemorySupersede_Validation(t *testing.T) {
	rt := &Runtime{memory: MemoryConfig{Enabled: true, Server: "memory"}}
	_, err := rt.MemorySupersede(context.Background(), MemorySupersedeOpts{Confirm: true})
	if err == nil || !strings.Contains(err.Error(), "entity required") {
		t.Fatalf("err=%v", err)
	}
	_, err = rt.MemorySupersede(context.Background(), MemorySupersedeOpts{
		Entity: "person:alice", AsOf: "not-a-time", Confirm: true,
	})
	if err == nil || !strings.Contains(err.Error(), "RFC3339") {
		t.Fatalf("bad as_of err=%v", err)
	}
	rt2 := &Runtime{memory: DefaultMemoryConfig()}
	_, err = rt2.MemorySupersede(context.Background(), MemorySupersedeOpts{
		Entity: "person:alice", Confirm: true,
	})
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("disabled err=%v", err)
	}
}

// s1282: Confirm + mock MCP success formats superseded_count (full path).
func TestMemorySupersede_MockMCPSuccess(t *testing.T) {
	// Lightweight stdio mock that answers tools/call memory_supersede_entity.
	// Mirrors skills_mcp_test mock pattern.
	cInR, cInW := io.Pipe()
	cOutR, cOutW := io.Pipe()
	go mockMCPSupersede(cOutW, cInR)

	mut := true // supersede is mutating
	cl := mcp.NewClientForTest(mcp.ServerConfig{Name: "memory", Command: "x", Mutating: &mut}, cInW, cOutR, nil)
	defer cl.Close()
	if err := cl.InitForTest(context.Background()); err != nil {
		t.Fatal(err)
	}
	mgr := mcp.NewManagerEmpty(nil)
	mgr.Attach(cl)

	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, Server: "memory", Tenant: "dept.research"},
		mcp:    mgr,
	}
	out, err := rt.MemorySupersede(context.Background(), MemorySupersedeOpts{
		Entity:  "person:alice",
		AsOf:    "2026-08-04T12:00:00Z",
		Confirm: true,
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "superseded_count: 2") {
		t.Fatalf("count: %q", out)
	}
	if !strings.Contains(out, "person:alice") || !strings.Contains(out, "A3 lite supersede") {
		t.Fatalf("format: %q", out)
	}
}

// mockMCPSupersede is a minimal MCP server for memory_supersede_entity success fixture.
func mockMCPSupersede(w io.WriteCloser, r io.Reader) {
	defer w.Close()
	dec := json.NewDecoder(r)
	for {
		var req map[string]any
		if err := dec.Decode(&req); err != nil {
			return
		}
		id := req["id"]
		method, _ := req["method"].(string)
		if method == "notifications/initialized" || id == nil {
			continue
		}
		var result any
		switch method {
		case "initialize":
			result = map[string]any{"protocolVersion": "2024-11-05", "serverInfo": map[string]string{"name": "memory", "version": "1"}}
		case "tools/list":
			result = map[string]any{"tools": []map[string]any{{
				"name": "memory_supersede_entity", "description": "A3 lite supersede",
				"inputSchema": map[string]any{"type": "object"},
			}}}
		case "tools/call":
			// Always return residual-honest success wire for supersede.
			payload := `{"entity":"person:alice","as_of":"2026-08-04T12:00:00Z","superseded_count":2}`
			result = map[string]any{"content": []map[string]any{{"type": "text", "text": payload}}}
		}
		line, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
		_, _ = w.Write(append(line, '\n'))
	}
}

// s1287: formatPatternsJSON residual-honest fixture {patterns:[{subject,kind,count,score,summary}]}.
func TestFormatPatternsJSON_Fixture(t *testing.T) {
	raw := `{
		"patterns": [
			{"id": "p1", "kind": "recurrence", "subject": "deploy", "count": 5, "score": 0.82, "summary": "deploy recurs in window", "window": "24h"},
			{"id": "p2", "kind": "recurrence", "subject": "oncall", "count": 3, "score": 0.6}
		]
	}`
	out := formatPatternsJSON(raw, 6000)
	if out == "" {
		t.Fatal("expected formatted output")
	}
	if !strings.Contains(out, "patterns (2):") {
		t.Fatalf("header: %q", out)
	}
	if !strings.Contains(out, "subject=deploy") || !strings.Contains(out, "kind=recurrence") {
		t.Fatalf("pattern1 fields: %q", out)
	}
	if !strings.Contains(out, "score=0.82") || !strings.Contains(out, "count=5") {
		t.Fatalf("pattern1 score/count: %q", out)
	}
	if !strings.Contains(out, "deploy recurs in window") {
		t.Fatalf("summary: %q", out)
	}
	if !strings.Contains(out, "subject=oncall") {
		t.Fatalf("pattern2: %q", out)
	}
	if !strings.Contains(out, "ops pulse Beta") || !strings.Contains(out, "not medical diagnosis") {
		t.Fatalf("honesty pin missing: %q", out)
	}
	if !strings.Contains(out, "not OTel host metrics") || !strings.Contains(out, "not invent GA window engine") {
		t.Fatalf("OTel/GA pin missing: %q", out)
	}
	if !strings.Contains(out, "dual_write OFF") || !strings.Contains(out, "not Memory GA") {
		t.Fatalf("dual_write/Memory GA pin missing: %q", out)
	}
}

// s1287: formatAnomaliesJSON residual-honest fixture {anomalies:[...]}.
func TestFormatAnomaliesJSON_Fixture(t *testing.T) {
	raw := `{
		"anomalies": [
			{"id": "a1", "kind": "burst", "subject": "error.rate", "count": 12, "score": 0.95, "summary": "burst vs baseline", "window": "15m"}
		]
	}`
	out := formatAnomaliesJSON(raw, 6000)
	if out == "" {
		t.Fatal("expected formatted output")
	}
	if !strings.Contains(out, "anomalies (1):") {
		t.Fatalf("header: %q", out)
	}
	if !strings.Contains(out, "subject=error.rate") || !strings.Contains(out, "kind=burst") {
		t.Fatalf("anomaly fields: %q", out)
	}
	if !strings.Contains(out, "score=0.95") || !strings.Contains(out, "burst vs baseline") {
		t.Fatalf("score/summary: %q", out)
	}
	if !strings.Contains(out, "ops pulse Beta") || !strings.Contains(out, "suggestive only") {
		t.Fatalf("honesty pin: %q", out)
	}
	if !strings.Contains(out, "not medical diagnosis") || !strings.Contains(out, "not Memory GA") {
		t.Fatalf("medical/Memory GA pin: %q", out)
	}
}

// s1287: empty patterns/anomalies is residual-honest empty — never invent signals.
func TestFormatPatternsAnomalies_EmptyHonest(t *testing.T) {
	out := formatPatterns(nil, 6000)
	if !strings.Contains(out, "patterns: (none)") {
		t.Fatalf("empty patterns: %q", out)
	}
	if !strings.Contains(out, pulseHonestyFooter) {
		t.Fatalf("honesty: %q", out)
	}
	out2 := formatPatternsJSON(`{"patterns":[]}`, 6000)
	if !strings.Contains(out2, "patterns: (none)") {
		t.Fatalf("empty json patterns: %q", out2)
	}
	outA := formatAnomalies(nil, 6000)
	if !strings.Contains(outA, "anomalies: (none)") {
		t.Fatalf("empty anomalies: %q", outA)
	}
	if !strings.Contains(outA, pulseHonestyFooter) {
		t.Fatalf("honesty anomalies: %q", outA)
	}
	outA2 := formatAnomaliesJSON(`{"anomalies":[]}`, 6000)
	if !strings.Contains(outA2, "anomalies: (none)") {
		t.Fatalf("empty json anomalies: %q", outA2)
	}
}

// s1287: offline MCP → residual-honest fail-open (not invent empty success).
func TestMemoryPatterns_OfflineFailOpen(t *testing.T) {
	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, Server: "memory", Tenant: "dept.research"},
		mcp:    mcp.NewManagerEmpty(nil),
	}
	out, err := rt.MemoryPatterns(context.Background(), MemoryPatternsOpts{Limit: 5})
	if err != nil {
		t.Fatalf("expected fail-open nil err, got %v", err)
	}
	if !strings.Contains(out, "unavailable") || !strings.Contains(out, "not connected") {
		t.Fatalf("offline: %q", out)
	}
	if !strings.Contains(out, "MCP-first") || !strings.Contains(out, "ops pulse Beta") {
		t.Fatalf("honesty offline: %q", out)
	}
	if !strings.Contains(out, "empty ≠ invent patterns") {
		t.Fatalf("empty≠invent pin: %q", out)
	}
	// Must not look like successful empty listing alone.
	if strings.Contains(out, "patterns: (none)") && !strings.Contains(out, "unavailable") {
		t.Fatalf("must not invent empty-success: %q", out)
	}
}

// s1287: offline MCP anomalies fail-open.
func TestMemoryAnomalies_OfflineFailOpen(t *testing.T) {
	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, Server: "memory", Tenant: "dept.research"},
		mcp:    mcp.NewManagerEmpty(nil),
	}
	out, err := rt.MemoryAnomalies(context.Background(), MemoryAnomaliesOpts{})
	if err != nil {
		t.Fatalf("expected fail-open nil err, got %v", err)
	}
	if !strings.Contains(out, "unavailable") || !strings.Contains(out, "not connected") {
		t.Fatalf("offline: %q", out)
	}
	if !strings.Contains(out, "MCP-first") || !strings.Contains(out, "ops pulse Beta") {
		t.Fatalf("honesty offline: %q", out)
	}
	if !strings.Contains(out, "empty ≠ invent anomalies") {
		t.Fatalf("empty≠invent pin: %q", out)
	}
	if strings.Contains(out, "anomalies: (none)") && !strings.Contains(out, "unavailable") {
		t.Fatalf("must not invent empty-success: %q", out)
	}
}

// s1287: hooks disabled error for patterns/anomalies.
func TestMemoryPatternsAnomalies_Disabled(t *testing.T) {
	rt := &Runtime{memory: DefaultMemoryConfig()}
	_, err := rt.MemoryPatterns(context.Background(), MemoryPatternsOpts{})
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("patterns disabled err=%v", err)
	}
	_, err = rt.MemoryAnomalies(context.Background(), MemoryAnomaliesOpts{})
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("anomalies disabled err=%v", err)
	}
}

// s1287: non-JSON returns empty (caller may pass through).
func TestFormatPatternsAnomaliesJSON_NonJSON(t *testing.T) {
	if got := formatPatternsJSON("not json", 100); got != "" {
		t.Fatalf("patterns got %q", got)
	}
	if got := formatPatternsJSON("", 100); got != "" {
		t.Fatalf("patterns empty got %q", got)
	}
	if got := formatAnomaliesJSON("not json", 100); got != "" {
		t.Fatalf("anomalies got %q", got)
	}
	if got := formatAnomaliesJSON("", 100); got != "" {
		t.Fatalf("anomalies empty got %q", got)
	}
}

// s1296: formatTimelineJSON residual-honest fixture {entries:[{id,summary,event_time|timestamp,...}]}.
func TestFormatTimelineJSON_Fixture(t *testing.T) {
	raw := `{
		"entries": [
			{"id": "e1", "summary": "deploy finished", "event_time": "2026-08-04T10:00:00Z", "score": 0.9},
			{"id": "e2", "summary": "incident opened", "timestamp": "2026-08-04T09:00:00Z"}
		]
	}`
	out := formatTimelineJSON(raw, 6000)
	if out == "" {
		t.Fatal("expected formatted output")
	}
	if !strings.Contains(out, "timeline") {
		t.Fatalf("header: %q", out)
	}
	if !strings.Contains(out, "deploy finished") || !strings.Contains(out, "2026-08-04T10:00:00Z") {
		t.Fatalf("entry1: %q", out)
	}
	if !strings.Contains(out, "incident opened") || !strings.Contains(out, "2026-08-04T09:00:00Z") {
		t.Fatalf("entry2: %q", out)
	}
	if !strings.Contains(out, "[0.90]") && !strings.Contains(out, "[0.9]") {
		// Score formatting uses %.2f → [0.90]
		if !strings.Contains(out, "0.9") {
			t.Fatalf("score: %q", out)
		}
	}
	if !strings.Contains(out, "temporal timeline") || !strings.Contains(out, "filters before limit") {
		t.Fatalf("honesty pin missing: %q", out)
	}
	if !strings.Contains(out, "not Memory GA") || !strings.Contains(out, "dual_write OFF") {
		t.Fatalf("Memory GA / dual_write pin missing: %q", out)
	}
	if !strings.Contains(out, "MCP-first") {
		t.Fatalf("MCP-first pin missing: %q", out)
	}
}

// s1296: empty timeline is residual-honest empty — never invent entries.
func TestFormatTimeline_EmptyHonest(t *testing.T) {
	out := formatTimeline(nil, 6000)
	if !strings.Contains(out, "entries: (none)") {
		t.Fatalf("empty: %q", out)
	}
	if !strings.Contains(out, timelineHonestyFooter) {
		t.Fatalf("honesty: %q", out)
	}
	out2 := formatTimelineJSON(`{"entries":[]}`, 6000)
	if !strings.Contains(out2, "entries: (none)") {
		t.Fatalf("empty json: %q", out2)
	}
}

// s1296: offline MCP timeline → residual-honest fail-open (not invent empty success).
func TestMemoryTimeline_OfflineFailOpen(t *testing.T) {
	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, Server: "memory", Tenant: "dept.research"},
		mcp:    mcp.NewManagerEmpty(nil),
	}
	out, err := rt.MemoryTimeline(context.Background(), MemoryTimelineOpts{Limit: 5})
	if err != nil {
		t.Fatalf("expected fail-open nil err, got %v", err)
	}
	if !strings.Contains(out, "unavailable") || !strings.Contains(out, "not connected") {
		t.Fatalf("offline: %q", out)
	}
	if !strings.Contains(out, "MCP-first") || !strings.Contains(out, "temporal timeline") {
		t.Fatalf("honesty offline: %q", out)
	}
	if !strings.Contains(out, "empty ≠ invent memories") {
		t.Fatalf("empty≠invent pin: %q", out)
	}
	if strings.Contains(out, "entries: (none)") && !strings.Contains(out, "unavailable") {
		t.Fatalf("must not invent empty-success: %q", out)
	}
}

// s1296: hooks disabled error for timeline.
func TestMemoryTimeline_Disabled(t *testing.T) {
	rt := &Runtime{memory: DefaultMemoryConfig()}
	_, err := rt.MemoryTimeline(context.Background(), MemoryTimelineOpts{})
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("timeline disabled err=%v", err)
	}
}

// s1296: non-JSON timeline returns empty (caller may pass through).
func TestFormatTimelineJSON_NonJSON(t *testing.T) {
	if got := formatTimelineJSON("not json", 100); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := formatTimelineJSON("", 100); got != "" {
		t.Fatalf("got %q", got)
	}
}

// s1296: formatCompactStatusJSON residual-honest fixture (PascalCase stats + last_compaction).
func TestFormatCompactStatusJSON_Fixture(t *testing.T) {
	// aion wire: palace.MemoryStats has no json tags → PascalCase nested fields.
	raw := `{
		"stats": {
			"WorkingCount": 2,
			"ContextualCount": 5,
			"ArchivalCount": 1,
			"SemanticCount": 3,
			"TotalEntries": 11,
			"LastCompaction": "0001-01-01T00:00:00Z"
		},
		"last_compaction": "2026-08-01T12:00:00Z"
	}`
	out := formatCompactStatusJSON(raw, 6000)
	if out == "" {
		t.Fatal("expected formatted output")
	}
	if !strings.Contains(out, "compact-status") {
		t.Fatalf("header: %q", out)
	}
	if !strings.Contains(out, "working=2") || !strings.Contains(out, "contextual=5") {
		t.Fatalf("tiers: %q", out)
	}
	if !strings.Contains(out, "archival=1") || !strings.Contains(out, "semantic=3") {
		t.Fatalf("tiers2: %q", out)
	}
	if !strings.Contains(out, "total=11") {
		t.Fatalf("total: %q", out)
	}
	if !strings.Contains(out, "last_compaction: 2026-08-01T12:00:00Z") {
		t.Fatalf("last_compaction: %q", out)
	}
	if !strings.Contains(out, "Palace tier counts residual") || !strings.Contains(out, "not auto-compact product") {
		t.Fatalf("honesty pin missing: %q", out)
	}
	if !strings.Contains(out, "not Memory GA") || !strings.Contains(out, "dual_write OFF") {
		t.Fatalf("Memory GA / dual_write pin missing: %q", out)
	}
	// snake_case nested stats also accepted.
	out2 := formatCompactStatusJSON(`{"stats":{"working_count":1,"semantic_count":4},"last_compaction":""}`, 6000)
	if !strings.Contains(out2, "working=1") || !strings.Contains(out2, "semantic=4") {
		t.Fatalf("snake: %q", out2)
	}
	if !strings.Contains(out2, "last_compaction: (none)") {
		t.Fatalf("empty last: %q", out2)
	}
}

// s1296: offline MCP compact-status → residual-honest fail-open (not invent green).
func TestMemoryCompactStatus_OfflineFailOpen(t *testing.T) {
	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, Server: "memory", Tenant: "dept.research"},
		mcp:    mcp.NewManagerEmpty(nil),
	}
	out, err := rt.MemoryCompactStatus(context.Background(), MemoryCompactStatusOpts{})
	if err != nil {
		t.Fatalf("expected fail-open nil err, got %v", err)
	}
	if !strings.Contains(out, "unavailable") || !strings.Contains(out, "not connected") {
		t.Fatalf("offline: %q", out)
	}
	if !strings.Contains(out, "MCP-first") || !strings.Contains(out, "Palace tier counts residual") {
		t.Fatalf("honesty offline: %q", out)
	}
	if !strings.Contains(out, "empty ≠ invent compaction green") {
		t.Fatalf("empty≠invent pin: %q", out)
	}
	// Must not look like successful tier listing alone.
	if strings.Contains(out, "working=") && !strings.Contains(out, "unavailable") {
		t.Fatalf("must not invent tier success: %q", out)
	}
}

// s1296: hooks disabled error for compact-status.
func TestMemoryCompactStatus_Disabled(t *testing.T) {
	rt := &Runtime{memory: DefaultMemoryConfig()}
	_, err := rt.MemoryCompactStatus(context.Background(), MemoryCompactStatusOpts{})
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("compact-status disabled err=%v", err)
	}
}

// s1296: non-JSON compact status returns empty (caller may pass through).
func TestFormatCompactStatusJSON_NonJSON(t *testing.T) {
	if got := formatCompactStatusJSON("not json", 100); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := formatCompactStatusJSON("", 100); got != "" {
		t.Fatalf("got %q", got)
	}
}

// s1301: formatSemanticJSON residual-honest fixture {facts:[{id,summary,full,score}]}.
func TestFormatSemanticJSON_Fixture(t *testing.T) {
	raw := `{
		"facts": [
			{"id": "f1", "summary": "alice is engineer", "score": 0.91},
			{"id": "f2", "full": "deploy window is Tuesday"}
		]
	}`
	out := formatSemanticJSON(raw, "alice role", 6000)
	if out == "" {
		t.Fatal("expected formatted output")
	}
	if !strings.Contains(out, "semantic query=alice role") {
		t.Fatalf("header: %q", out)
	}
	if !strings.Contains(out, "alice is engineer") || !strings.Contains(out, "f1") {
		t.Fatalf("fact1: %q", out)
	}
	if !strings.Contains(out, "deploy window is Tuesday") || !strings.Contains(out, "f2") {
		t.Fatalf("fact2: %q", out)
	}
	if !strings.Contains(out, "tier-4 semantic facts residual") {
		t.Fatalf("honesty pin missing: %q", out)
	}
	if !strings.Contains(out, "not Memory GA") || !strings.Contains(out, "dual_write OFF") {
		t.Fatalf("Memory GA / dual_write pin missing: %q", out)
	}
	if !strings.Contains(out, "MCP-first") || !strings.Contains(out, "empty ≠ invent") {
		t.Fatalf("MCP-first / empty pin missing: %q", out)
	}
}

// s1301: empty semantic is residual-honest empty — never invent facts.
func TestFormatSemantic_EmptyHonest(t *testing.T) {
	out := formatSemantic("q", nil, 6000)
	if !strings.Contains(out, "facts: (none)") {
		t.Fatalf("empty: %q", out)
	}
	if !strings.Contains(out, semanticHonestyFooter) {
		t.Fatalf("honesty: %q", out)
	}
	out2 := formatSemanticJSON(`{"facts":[]}`, "q", 6000)
	if !strings.Contains(out2, "facts: (none)") {
		t.Fatalf("empty json: %q", out2)
	}
}

// s1301: offline MCP semantic → residual-honest fail-open (not invent empty success).
func TestMemorySearchSemantic_OfflineFailOpen(t *testing.T) {
	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, Server: "memory", Tenant: "dept.research"},
		mcp:    mcp.NewManagerEmpty(nil),
	}
	out, err := rt.MemorySearchSemantic(context.Background(), MemorySemanticOpts{Query: "alice", Limit: 5})
	if err != nil {
		t.Fatalf("expected fail-open nil err, got %v", err)
	}
	if !strings.Contains(out, "unavailable") || !strings.Contains(out, "not connected") {
		t.Fatalf("offline: %q", out)
	}
	if !strings.Contains(out, "MCP-first") || !strings.Contains(out, "tier-4 semantic facts residual") {
		t.Fatalf("honesty offline: %q", out)
	}
	if !strings.Contains(out, "empty ≠ invent memories") {
		t.Fatalf("empty≠invent pin: %q", out)
	}
	if strings.Contains(out, "facts: (none)") && !strings.Contains(out, "unavailable") {
		t.Fatalf("must not invent empty-success: %q", out)
	}
}

// s1301: hooks disabled + missing query for semantic.
func TestMemorySearchSemantic_DisabledAndQueryRequired(t *testing.T) {
	rt := &Runtime{memory: DefaultMemoryConfig()}
	_, err := rt.MemorySearchSemantic(context.Background(), MemorySemanticOpts{Query: "x"})
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("semantic disabled err=%v", err)
	}
	rt2 := &Runtime{memory: MemoryConfig{Enabled: true, Server: "memory"}}
	_, err2 := rt2.MemorySearchSemantic(context.Background(), MemorySemanticOpts{})
	if err2 == nil || !strings.Contains(err2.Error(), "query required") {
		t.Fatalf("query required err=%v", err2)
	}
}

// s1301: non-JSON semantic returns empty (caller may pass through).
func TestFormatSemanticJSON_NonJSON(t *testing.T) {
	if got := formatSemanticJSON("not json", "q", 100); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := formatSemanticJSON("", "q", 100); got != "" {
		t.Fatalf("got %q", got)
	}
}

// s1301: formatIngestEventJSON residual-honest fixture {memory_id,tier,event_time,audited}.
func TestFormatIngestEventJSON_Fixture(t *testing.T) {
	raw := `{
		"memory_id": "pz5rzwg3ugzt5fbmsyibinvm",
		"tier": 1,
		"event_time": "2026-08-06T02:30:28Z",
		"audited": true
	}`
	out := formatIngestEventJSON(raw, "dept.research.events.test", 6000)
	if out == "" {
		t.Fatal("expected formatted output")
	}
	if !strings.Contains(out, "ingest-event subject=dept.research.events.test") {
		t.Fatalf("header: %q", out)
	}
	if !strings.Contains(out, "memory_id: pz5rzwg3ugzt5fbmsyibinvm") {
		t.Fatalf("memory_id: %q", out)
	}
	if !strings.Contains(out, "tier: 1") {
		t.Fatalf("tier: %q", out)
	}
	if !strings.Contains(out, "event_time: 2026-08-06T02:30:28Z") {
		t.Fatalf("event_time: %q", out)
	}
	if !strings.Contains(out, "audited: true") {
		t.Fatalf("audited: %q", out)
	}
	if !strings.Contains(out, "s138 T1 temporal event telemetry") {
		t.Fatalf("honesty pin missing: %q", out)
	}
	if !strings.Contains(out, "not conversation turn") || !strings.Contains(out, "not Memory GA") {
		t.Fatalf("turn / Memory GA pin missing: %q", out)
	}
	if !strings.Contains(out, "dual_write OFF") || !strings.Contains(out, "MCP-first") {
		t.Fatalf("dual_write / MCP-first pin missing: %q", out)
	}
	// Empty wire still residual — never invent memory_id.
	out2 := formatIngestEventJSON(`{}`, "subj", 6000)
	if !strings.Contains(out2, "memory_id: (none from wire)") {
		t.Fatalf("empty wire memory_id: %q", out2)
	}
	if !strings.Contains(out2, ingestEventHonestyFooter) {
		t.Fatalf("empty wire honesty: %q", out2)
	}
}

// s1301: offline MCP ingest-event → residual-honest fail-open (never invent memory_id).
func TestMemoryIngestEvent_OfflineFailOpen(t *testing.T) {
	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, Server: "memory", Tenant: "dept.research"},
		mcp:    mcp.NewManagerEmpty(nil),
	}
	out, err := rt.MemoryIngestEvent(context.Background(), MemoryIngestEventOpts{
		Subject: "dept.research.events.probe",
		Content: "hello event",
	})
	if err != nil {
		t.Fatalf("expected fail-open nil err, got %v", err)
	}
	if !strings.Contains(out, "unavailable") || !strings.Contains(out, "not connected") {
		t.Fatalf("offline: %q", out)
	}
	if !strings.Contains(out, "MCP-first") || !strings.Contains(out, "s138 T1 temporal event telemetry") {
		t.Fatalf("honesty offline: %q", out)
	}
	if !strings.Contains(out, "never invent memory_id") {
		t.Fatalf("never invent pin: %q", out)
	}
	if strings.Contains(out, "memory_id: ") && !strings.Contains(out, "unavailable") {
		// offline path must not invent a successful memory_id line without unavailable.
		if strings.Contains(out, "memory_id: pz") || strings.Contains(out, "memory_id: mem") {
			t.Fatalf("must not invent memory_id: %q", out)
		}
	}
}

// s1301: hooks disabled + subject/content required for ingest-event.
func TestMemoryIngestEvent_DisabledAndRequired(t *testing.T) {
	rt := &Runtime{memory: DefaultMemoryConfig()}
	_, err := rt.MemoryIngestEvent(context.Background(), MemoryIngestEventOpts{Subject: "s", Content: "c"})
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("ingest-event disabled err=%v", err)
	}
	rt2 := &Runtime{memory: MemoryConfig{Enabled: true, Server: "memory"}}
	_, err2 := rt2.MemoryIngestEvent(context.Background(), MemoryIngestEventOpts{Content: "c"})
	if err2 == nil || !strings.Contains(err2.Error(), "subject required") {
		t.Fatalf("subject required err=%v", err2)
	}
	_, err3 := rt2.MemoryIngestEvent(context.Background(), MemoryIngestEventOpts{Subject: "s"})
	if err3 == nil || !strings.Contains(err3.Error(), "content required") {
		t.Fatalf("content required err=%v", err3)
	}
}

// s1301: non-JSON ingest-event returns empty (caller may pass through).
func TestFormatIngestEventJSON_NonJSON(t *testing.T) {
	if got := formatIngestEventJSON("not json", "s", 100); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := formatIngestEventJSON("", "s", 100); got != "" {
		t.Fatalf("got %q", got)
	}
}

// s1311: without Confirm → residual-honest refusal string; must NOT require MCP (no call).
func TestMemoryTriggerCompact_RefuseWithoutConfirm(t *testing.T) {
	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, Server: "memory", Tenant: "dept.research"},
		mcp:    mcp.NewManagerEmpty(nil),
	}
	out, err := rt.MemoryTriggerCompact(context.Background(), MemoryTriggerCompactOpts{Confirm: false})
	if err != nil {
		t.Fatalf("expected residual-honest refusal nil err, got %v", err)
	}
	if !strings.Contains(out, "refused") || !strings.Contains(out, "HITL") {
		t.Fatalf("refusal: %q", out)
	}
	if !strings.Contains(out, "--i-confirm") {
		t.Fatalf("must mention --i-confirm: %q", out)
	}
	if !strings.Contains(out, "trigger-compact") {
		t.Fatalf("surface: %q", out)
	}
	if !strings.Contains(out, triggerCompactHonestyFooter) && !strings.Contains(out, "RecMem advisory") {
		t.Fatalf("honesty: %q", out)
	}
	// Must not look like success with triggered/cluster_size.
	if strings.Contains(out, "triggered:") || strings.Contains(out, "cluster_size:") {
		t.Fatalf("must not invent triggered/cluster_size on refuse: %q", out)
	}
	if strings.Contains(out, "unavailable") {
		t.Fatalf("must not reach offline path without Confirm: %q", out)
	}
}

// s1311: Confirm + offline MCP → residual-honest fail-open (not invent triggered).
func TestMemoryTriggerCompact_OfflineFailOpen(t *testing.T) {
	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, Server: "memory", Tenant: "dept.research"},
		mcp:    mcp.NewManagerEmpty(nil),
	}
	out, err := rt.MemoryTriggerCompact(context.Background(), MemoryTriggerCompactOpts{Confirm: true})
	if err != nil {
		t.Fatalf("expected fail-open nil err, got %v", err)
	}
	if !strings.Contains(out, "unavailable") || !strings.Contains(out, "not connected") {
		t.Fatalf("offline: %q", out)
	}
	if !strings.Contains(out, "MCP-first") || !strings.Contains(out, "RecMem advisory") {
		t.Fatalf("honesty offline: %q", out)
	}
	if !strings.Contains(out, "not inventing triggered/cluster_size") {
		t.Fatalf("not-invent pin: %q", out)
	}
	if strings.Contains(out, "triggered:") || strings.Contains(out, "cluster_size:") {
		t.Fatalf("must not invent trigger success offline: %q", out)
	}
}

// s1311: hooks disabled error for trigger-compact.
func TestMemoryTriggerCompact_Disabled(t *testing.T) {
	rt := &Runtime{memory: DefaultMemoryConfig()}
	_, err := rt.MemoryTriggerCompact(context.Background(), MemoryTriggerCompactOpts{Confirm: true})
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("disabled err=%v", err)
	}
}

// s1311: formatTriggerCompactJSON residual-honest fixture {triggered, cluster_size}.
func TestFormatTriggerCompactJSON_Fixture(t *testing.T) {
	raw := `{"triggered":true,"cluster_size":7}`
	out := formatTriggerCompactJSON(raw)
	if out == "" {
		t.Fatal("expected formatted output")
	}
	if !strings.Contains(out, "trigger-compact") {
		t.Fatalf("header: %q", out)
	}
	if !strings.Contains(out, "triggered: true") {
		t.Fatalf("triggered: %q", out)
	}
	if !strings.Contains(out, "cluster_size: 7") {
		t.Fatalf("cluster_size: %q", out)
	}
	if !strings.Contains(out, "RecMem advisory") || !strings.Contains(out, "not invent compaction green") {
		t.Fatalf("honesty pin missing: %q", out)
	}
	if !strings.Contains(out, "dual_write OFF") || !strings.Contains(out, "not Memory GA") {
		t.Fatalf("dual_write/Memory GA pin missing: %q", out)
	}
	if !strings.Contains(out, "mutating HITL") {
		t.Fatalf("HITL pin missing: %q", out)
	}
	// Empty / non-JSON → empty (caller may pass through).
	if got := formatTriggerCompactJSON(""); got != "" {
		t.Fatalf("empty: %q", got)
	}
	if got := formatTriggerCompactJSON("not json"); got != "" {
		t.Fatalf("non-json: %q", got)
	}
	if got := formatTriggerCompactJSON(`{}`); got != "" {
		t.Fatalf("empty object must not invent false/0 success: %q", got)
	}
}

// s1311: Confirm + mock MCP success formats triggered + cluster_size.
func TestMemoryTriggerCompact_MockMCPSuccess(t *testing.T) {
	cInR, cInW := io.Pipe()
	cOutR, cOutW := io.Pipe()
	go mockMCPTriggerCompact(cOutW, cInR)

	mut := true // trigger-compact is mutating
	cl := mcp.NewClientForTest(mcp.ServerConfig{Name: "memory", Command: "x", Mutating: &mut}, cInW, cOutR, nil)
	defer cl.Close()
	if err := cl.InitForTest(context.Background()); err != nil {
		t.Fatal(err)
	}
	mgr := mcp.NewManagerEmpty(nil)
	mgr.Attach(cl)

	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, Server: "memory", Tenant: "dept.research"},
		mcp:    mgr,
	}
	out, err := rt.MemoryTriggerCompact(context.Background(), MemoryTriggerCompactOpts{Confirm: true})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "triggered: true") {
		t.Fatalf("triggered: %q", out)
	}
	if !strings.Contains(out, "cluster_size: 12") {
		t.Fatalf("cluster_size: %q", out)
	}
	if !strings.Contains(out, "RecMem advisory") {
		t.Fatalf("honesty: %q", out)
	}
}

// mockMCPTriggerCompact is a minimal MCP server for memory_trigger_compact success fixture.
func mockMCPTriggerCompact(w io.WriteCloser, r io.Reader) {
	defer w.Close()
	dec := json.NewDecoder(r)
	for {
		var req map[string]any
		if err := dec.Decode(&req); err != nil {
			return
		}
		id := req["id"]
		method, _ := req["method"].(string)
		if method == "notifications/initialized" || id == nil {
			continue
		}
		var result any
		switch method {
		case "initialize":
			result = map[string]any{"protocolVersion": "2024-11-05", "serverInfo": map[string]string{"name": "memory", "version": "1"}}
		case "tools/list":
			result = map[string]any{"tools": []map[string]any{{
				"name": "memory_trigger_compact", "description": "RecMem compact advisory",
				"inputSchema": map[string]any{"type": "object"},
			}}}
		case "tools/call":
			payload := `{"triggered":true,"cluster_size":12}`
			result = map[string]any{"content": []map[string]any{{"type": "text", "text": payload}}}
		}
		line, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
		_, _ = w.Write(append(line, '\n'))
	}
}

// s1311: MemoryAdvancedStatus offline residual inventory needles.
func TestMemoryAdvancedStatus_OfflineResidual(t *testing.T) {
	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, Server: "memory", Tenant: "dept.research"},
		mcp:    mcp.NewManagerEmpty(nil),
	}
	out, err := rt.MemoryAdvancedStatus(context.Background())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	for _, want := range []string{
		"memory advanced status",
		"s1311",
		"advanced tools:",
		"memory_related",
		"memory_facts_as_of",
		"memory_supersede_entity",
		"memory_timeline",
		"memory_compact_status",
		"memory_search_semantic",
		"memory_ingest_event",
		"memory_patterns_list",
		"memory_anomalies_list",
		"ops_digest_export",
		"memory_trigger_compact",
		"offline",
		"dual_write",
		"not Memory GA",
		"/integrations status",
		"trigger-compact requires HITL",
		"fail-open",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("advanced status missing %q in:\n%s", want, out)
		}
	}
	// Must not invent product green.
	if strings.Contains(out, "Memory GA green") || strings.Contains(out, "Memory GA shipped") {
		t.Fatalf("must not invent Memory GA claim: %s", out)
	}
	if !strings.Contains(out, "mcp-manager-empty (0 servers) · fail-open") {
		t.Fatalf("want mcp-manager-empty: %s", out)
	}
	if strings.Contains(out, "connected-empty") {
		t.Fatalf("must not use connected-empty: %s", out)
	}
}

func TestMemoryAdvancedStatus_EmptyManagerNotConnectedEmpty(t *testing.T) {
	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, Server: "memory"},
		mcp:    mcp.NewManagerEmpty(nil),
	}
	out, err := rt.MemoryAdvancedStatus(context.Background())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "mcp-manager-empty (0 servers) · fail-open") {
		t.Fatalf("want mcp-manager-empty: %s", out)
	}
	if strings.Contains(out, "connected-empty") {
		t.Fatalf("must not use connected-empty: %s", out)
	}
}

// s1311: MemoryAdvancedStatus with hooks disabled still residual (no invent).
func TestMemoryAdvancedStatus_Disabled(t *testing.T) {
	rt := &Runtime{memory: DefaultMemoryConfig()}
	out, err := rt.MemoryAdvancedStatus(context.Background())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "hooks disabled") {
		t.Fatalf("disabled: %q", out)
	}
	if !strings.Contains(out, "memory_trigger_compact") {
		t.Fatalf("inventory still listed: %q", out)
	}
	if !strings.Contains(out, "not Memory GA") {
		t.Fatalf("honesty: %q", out)
	}
}
