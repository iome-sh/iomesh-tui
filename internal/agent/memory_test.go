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

func TestMemoryRecallRequiresMCP(t *testing.T) {
	rt := &Runtime{memory: MemoryConfig{Enabled: true, Server: "memory"}}
	_, err := rt.MemoryRecall(context.Background(), "q")
	if err == nil {
		t.Fatal("expected error without mcp")
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

func TestDefaultMemoryConfig(t *testing.T) {
	d := DefaultMemoryConfig()
	if d.Enabled || d.Server != "memory" || !d.AutoRecall || d.AutoIngest || d.DualWrite {
		t.Fatalf("%+v", d)
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
