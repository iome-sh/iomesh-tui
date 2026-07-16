package iomesh

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/router"
)

func TestDisabledIsNoop(t *testing.T) {
	c := New(Config{Enabled: false}, nil)
	if c.Enabled() {
		t.Fatal()
	}
	if snip := c.ContextSnippet(context.Background(), "/ws", "q"); snip != "" {
		t.Fatal(snip)
	}
	c.Emit(context.Background(), DeptEvent{Type: "dept.test"})
	if err := c.Health(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidEndpointDisables(t *testing.T) {
	c := New(Config{Enabled: true, Endpoint: "file:///etc/passwd"}, nil)
	if c.Enabled() {
		t.Fatal("file URL must disable client")
	}
}

func TestContextSnippet_SuccessAndFailOpen(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/context/query" {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["include_lineage"] != true {
			t.Errorf("expected include_lineage, got %v", body["include_lineage"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"text": "mesh-ctx",
			"lineage": []map[string]string{
				{"id": "dp-1", "subject": "dept.eng.events", "source": "kafka", "freshness": "2m"},
			},
		})
	}))
	defer srv.Close()

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, ContextPlane: true, Tenant: "t", IncludeLineage: true,
	}, nil)
	if !c.Enabled() {
		t.Fatal("should enable")
	}
	got := c.ContextSnippet(context.Background(), "/ws", "q")
	if !strings.Contains(got, "mesh-ctx") || !strings.Contains(got, "dp-1") || !strings.Contains(got, "<iomesh-lineage>") {
		t.Fatalf("got %q", got)
	}

	// Fail-open on 500
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer bad.Close()
	c2 := New(Config{Enabled: true, Endpoint: bad.URL, ContextPlane: true}, nil)
	if got := c2.ContextSnippet(context.Background(), "/ws", "q"); got != "" {
		t.Fatalf("fail-open got %q", got)
	}
}

func TestFormatContextSnippet_TextOnly(t *testing.T) {
	if got := FormatContextSnippet(ContextResult{Text: "  hello  "}); got != "hello" {
		t.Fatalf("%q", got)
	}
}

func TestEmitAndRecordLLMCall(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(204)
	}))
	defer srv.Close()

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, EmitDeptStreams: true, Tenant: "acme",
	}, nil)
	c.RecordLLMCall(router.CallMeta{
		ModelName: "deepseek-v4-flash", ModelID: "deepseek-v4-flash",
		Duration: time.Millisecond, EstimatedUSD: 0.001,
	}, router.Usage{TotalTokens: 10}, nil)

	// Allow async-ish emit to complete (RecordLLMCall is sync with timeout).
	if !strings.Contains(gotBody, "dept.agent.llm_call") {
		t.Fatalf("body=%q", gotBody)
	}
	if strings.Contains(gotBody, "Bearer ") {
		t.Fatal("must not log bearer in payload")
	}
}

func TestHealth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(200)
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()
	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	if err := c.Health(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestPublishMemoryIngest_PathSubjectAndPayload(t *testing.T) {
	var gotPath string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"stream":  "MEMORY_INGEST",
			"seq":     9,
			"subject": gotBody["subject"],
		})
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "dept.research"}, nil)
	ack, err := c.PublishMemoryIngest(context.Background(), "dept.research", MemoryEnvelope{
		Role:       "user",
		Content:    "remember this",
		SessionID:  "sess-1",
		EventTime:  "2026-07-16T12:00:00Z",
		SessionSeq: 3,
	})
	if err != nil {
		t.Fatalf("PublishMemoryIngest: %v", err)
	}
	if gotPath != "/v1/streams/MEMORY_INGEST/publish" {
		t.Fatalf("path=%q", gotPath)
	}
	if gotBody["subject"] != "dept.research.memory.ingest.turn" {
		t.Fatalf("subject=%v", gotBody["subject"])
	}
	payloadB64, _ := gotBody["payload"].(string)
	if payloadB64 == "" {
		t.Fatal("expected base64 payload")
	}
	raw, err := base64.StdEncoding.DecodeString(payloadB64)
	if err != nil {
		t.Fatalf("b64: %v", err)
	}
	var env map[string]any
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	if env["type"] != "memory_ingest" {
		t.Fatalf("type=%v", env["type"])
	}
	if env["event_time"] != "2026-07-16T12:00:00Z" {
		t.Fatalf("event_time=%v", env["event_time"])
	}
	if env["session_seq"] != float64(3) {
		t.Fatalf("session_seq=%v", env["session_seq"])
	}
	if env["session_id"] != "sess-1" || env["role"] != "user" || env["content"] != "remember this" {
		t.Fatalf("env=%v", env)
	}
	if ack == nil || ack.Seq != 9 {
		t.Fatalf("ack=%+v", ack)
	}
}

func TestPublishMemoryIngest_RequiresContentAndTenant(t *testing.T) {
	c := New(Config{Enabled: true, Endpoint: "http://127.0.0.1:9"}, nil)
	if _, err := c.PublishMemoryIngest(context.Background(), "", MemoryEnvelope{Content: "x"}); err == nil {
		t.Fatal("expected tenant error")
	}
	if _, err := c.PublishMemoryIngest(context.Background(), "t", MemoryEnvelope{}); err == nil {
		t.Fatal("expected content error")
	}
	off := New(Config{Enabled: false}, nil)
	if _, err := off.PublishMemoryIngest(context.Background(), "t", MemoryEnvelope{Content: "x"}); err == nil {
		t.Fatal("expected disabled error")
	}
}
