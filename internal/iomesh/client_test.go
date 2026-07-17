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

func TestPublishMemoryIngest_OrgWorkspaceHeaders(t *testing.T) {
	var gotOrg, gotWS, gotTenant string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotOrg = r.Header.Get("X-IOMesh-Org")
		gotWS = r.Header.Get("X-IOMesh-Workspace")
		gotTenant = r.Header.Get("X-IOMesh-Tenant")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"stream":"MEMORY_INGEST","seq":1}`))
	}))
	defer srv.Close()

	c := New(Config{
		Enabled:     true,
		Endpoint:    srv.URL,
		Tenant:      "dept.research",
		OrgID:       "org_dev-org",
		WorkspaceID: "ws_alpha",
	}, nil)
	if _, err := c.PublishMemoryIngest(context.Background(), "dept.research", MemoryEnvelope{Content: "hi"}); err != nil {
		t.Fatalf("PublishMemoryIngest: %v", err)
	}
	if gotOrg != "org_dev-org" {
		t.Fatalf("X-IOMesh-Org=%q", gotOrg)
	}
	if gotWS != "ws_alpha" {
		t.Fatalf("X-IOMesh-Workspace=%q", gotWS)
	}
	if gotTenant != "dept.research" {
		t.Fatalf("X-IOMesh-Tenant=%q", gotTenant)
	}
}

func TestPublishMemoryIngest_OmitsOrgWorkspaceWhenUnset(t *testing.T) {
	var hadOrg, hadWS bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, hadOrg = r.Header["X-IOMesh-Org"]
		_, hadWS = r.Header["X-IOMesh-Workspace"]
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "t"}, nil)
	if _, err := c.PublishMemoryIngest(context.Background(), "t", MemoryEnvelope{Content: "x"}); err != nil {
		t.Fatal(err)
	}
	if hadOrg || hadWS {
		t.Fatalf("expected no org/workspace headers; hadOrg=%v hadWS=%v", hadOrg, hadWS)
	}
}

func TestPublishMemoryRecall_PathSubjectAndSession(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	var gotOrg, gotWS string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotOrg = r.Header.Get("X-IOMesh-Org")
		gotWS = r.Header.Get("X-IOMesh-Workspace")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"stream":  "MEMORY_RPC",
			"seq":     4,
			"subject": gotBody["subject"],
		})
	}))
	defer srv.Close()

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "dept.research",
		OrgID: "org_x", WorkspaceID: "ws_y",
	}, nil)
	ack, err := c.PublishMemoryRecall(context.Background(), "dept.research", "find dogfood notes", 5, "dept.research.mesh-dogfood")
	if err != nil {
		t.Fatalf("PublishMemoryRecall: %v", err)
	}
	if gotPath != "/v1/streams/MEMORY_RPC/publish" {
		t.Fatalf("path=%q", gotPath)
	}
	if gotBody["subject"] != "dept.research.memory.retrieve.request" {
		t.Fatalf("subject=%v", gotBody["subject"])
	}
	if gotOrg != "org_x" || gotWS != "ws_y" {
		t.Fatalf("headers org=%q ws=%q", gotOrg, gotWS)
	}
	payloadB64, _ := gotBody["payload"].(string)
	raw, err := base64.StdEncoding.DecodeString(payloadB64)
	if err != nil {
		t.Fatal(err)
	}
	var env map[string]any
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	if env["type"] != "memory_recall" || env["query"] != "find dogfood notes" {
		t.Fatalf("env=%v", env)
	}
	if env["session_id"] != "dept.research.mesh-dogfood" {
		t.Fatalf("session_id=%v", env["session_id"])
	}
	if env["limit"] != float64(5) {
		t.Fatalf("limit=%v", env["limit"])
	}
	if ack == nil || ack.Seq != 4 {
		t.Fatalf("ack=%+v", ack)
	}
}

func TestRetrieveMemory_SyncHitsAndFallback(t *testing.T) {
	var paths []string
	var gotBody map[string]any
	var gotOrg string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		gotOrg = r.Header.Get("X-IOMesh-Org")
		if r.URL.Path == "/v1/memory/retrieve" {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"memories": []map[string]any{
					{"id": "h1", "summary": "hit one", "full": "full hit one", "score": 0.8},
					{"id": "h2", "summary": "hit two", "full": "full hit two"},
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "dept.x", OrgID: "org_a"}, nil)
	res, err := c.RetrieveMemory(context.Background(), "dept.x", "dogfood", 8, "dept.x.mesh-dogfood")
	if err != nil {
		t.Fatal(err)
	}
	if res.Path != "/v1/memory/retrieve" || len(res.Memories) != 2 {
		t.Fatalf("%+v", res)
	}
	if gotBody["tenant_id"] != "dept.x" || gotBody["session_id"] != "dept.x.mesh-dogfood" {
		t.Fatalf("body=%v", gotBody)
	}
	if gotOrg != "org_a" {
		t.Fatalf("org header=%q", gotOrg)
	}
}

func TestRetrieveMemory_V5Fallback(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path == "/v1/memory/retrieve" {
			w.WriteHeader(404)
			return
		}
		if r.URL.Path == "/v5/memory/retrieve" {
			_ = json.NewEncoder(w).Encode(map[string]any{"memories": []any{}})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "t"}, nil)
	res, err := c.RetrieveMemory(context.Background(), "t", "q", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Path != "/v5/memory/retrieve" || res.Memories == nil {
		t.Fatalf("%+v paths=%v", res, paths)
	}
}
