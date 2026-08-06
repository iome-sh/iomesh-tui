package iomesh

import (
	"context"
	"encoding/base64"
	"encoding/json"
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
	var gotPath, gotSubject, gotDecoded string
	var gotOrg, gotWS string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotOrg = r.Header.Get("X-IOMesh-Org")
		gotWS = r.Header.Get("X-IOMesh-Workspace")
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotSubject, _ = body["subject"].(string)
		if s, ok := body["payload"].(string); ok {
			raw, _ := base64.StdEncoding.DecodeString(s)
			gotDecoded = string(raw)
		}
		w.WriteHeader(204)
	}))
	defer srv.Close()

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, EmitDeptStreams: true, Tenant: "acme",
		OrgID: "org_a", WorkspaceID: "ws_1",
	}, nil)
	c.RecordLLMCall(router.CallMeta{
		ModelName: "deepseek-v4-flash", ModelID: "deepseek-v4-flash",
		Duration: time.Millisecond, EstimatedUSD: 0.001,
	}, router.Usage{TotalTokens: 10}, nil)

	if gotPath != "/v1/streams/dept/publish" {
		t.Fatalf("path=%q", gotPath)
	}
	if gotSubject != "dept.agent.llm_call" {
		t.Fatalf("subject=%q", gotSubject)
	}
	if !strings.Contains(gotDecoded, "dept.agent.llm_call") {
		t.Fatalf("decoded=%q", gotDecoded)
	}
	if !strings.Contains(gotDecoded, `"org":"org_a"`) && !strings.Contains(gotDecoded, `"org": "org_a"`) {
		t.Fatalf("expected org in payload: %q", gotDecoded)
	}
	if gotOrg != "org_a" || gotWS != "ws_1" {
		t.Fatalf("headers org=%q ws=%q", gotOrg, gotWS)
	}
	if strings.Contains(gotDecoded, "Bearer ") {
		t.Fatal("must not log bearer in payload")
	}
}

func TestHealth(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			gotUA = r.Header.Get("User-Agent")
			w.WriteHeader(200)
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()
	SetUserAgent("iomesh-tui/test")
	t.Cleanup(func() { SetUserAgent("iomesh-tui") })
	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	if err := c.Health(context.Background()); err != nil {
		t.Fatal(err)
	}
	if gotUA != "iomesh-tui/test" {
		t.Fatalf("User-Agent=%q", gotUA)
	}
}

func TestSetUserAgent(t *testing.T) {
	prev := UserAgent()
	t.Cleanup(func() { SetUserAgent(prev) })
	SetUserAgent("  iomesh-tui/9.9.9  ")
	if UserAgent() != "iomesh-tui/9.9.9" {
		t.Fatalf("%q", UserAgent())
	}
	SetUserAgent("") // keep current
	if UserAgent() != "iomesh-tui/9.9.9" {
		t.Fatalf("empty should keep: %q", UserAgent())
	}
}

func TestSetProductVersion(t *testing.T) {
	prev := ProductVersion()
	t.Cleanup(func() {
		productVersion = prev
	})
	productVersion = "" // isolate
	SetProductVersion("  0.27.0  ")
	if ProductVersion() != "0.27.0" {
		t.Fatalf("%q", ProductVersion())
	}
	SetProductVersion("") // keep current
	if ProductVersion() != "0.27.0" {
		t.Fatalf("empty should keep: %q", ProductVersion())
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

func TestRetrieveMemory_PrefersMemoryEndpointSidecar(t *testing.T) {
	// Mesh broker has no memory routes; sidecar does.
	broker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer broker.Close()

	var hitSidecar bool
	sidecar := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/memory/retrieve" {
			hitSidecar = true
			_ = json.NewEncoder(w).Encode(map[string]any{
				"memories": []map[string]any{{"summary": "warm palace hit"}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer sidecar.Close()

	c := New(Config{
		Enabled: true, Endpoint: broker.URL, Tenant: "dept.x",
		MemoryEndpoint: sidecar.URL,
	}, nil)
	if !c.MemoryEndpointConfigured() || c.MemoryBaseURL() != strings.TrimRight(sidecar.URL, "/") {
		t.Fatalf("base=%q configured=%v", c.MemoryBaseURL(), c.MemoryEndpointConfigured())
	}
	res, err := c.RetrieveMemory(context.Background(), "dept.x", "warm", 4, "sess")
	if err != nil {
		t.Fatal(err)
	}
	if !hitSidecar || len(res.Memories) != 1 || res.Memories[0].Summary != "warm palace hit" {
		t.Fatalf("hit=%v res=%+v", hitSidecar, res)
	}
}

func TestRetrieveMemory_SidecarOnlyWithoutMeshEndpoint(t *testing.T) {
	// Sidecar alone is enough for SyncMemoryReady / RetrieveMemory.
	sidecar := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"memories": []any{}})
	}))
	defer sidecar.Close()

	c := New(Config{MemoryEndpoint: sidecar.URL, Tenant: "t"}, nil)
	if c.Enabled() {
		t.Fatal("mesh should not be enabled without Endpoint")
	}
	if !c.SyncMemoryReady() {
		t.Fatal("expected SyncMemoryReady with sidecar only")
	}
	res, err := c.RetrieveMemory(context.Background(), "t", "q", 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Memories == nil {
		t.Fatal("expected empty slice")
	}
}

// s1068: temporal since/until/session_seq must appear in the JSON body when set.
func TestRetrieveMemoryWithOptions_TemporalFiltersInBody(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/memory/retrieve" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{"memories": []any{}})
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "dept.x"}, nil)
	res, err := c.RetrieveMemoryWithOptions(context.Background(), "dept.x", MemoryRetrieveOptions{
		Query:      "windowed",
		Limit:      5,
		SessionID:  "sess-1",
		SessionSeq: 3,
		Since:      "2026-07-01T00:00:00Z",
		Until:      "2026-07-31T23:59:59Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || res.Path != "/v1/memory/retrieve" {
		t.Fatalf("res=%+v", res)
	}
	if gotBody["tenant_id"] != "dept.x" || gotBody["query"] != "windowed" {
		t.Fatalf("body base=%v", gotBody)
	}
	if gotBody["session_id"] != "sess-1" {
		t.Fatalf("session_id=%v", gotBody["session_id"])
	}
	if gotBody["session_seq"] != float64(3) {
		t.Fatalf("session_seq=%v", gotBody["session_seq"])
	}
	if gotBody["since"] != "2026-07-01T00:00:00Z" {
		t.Fatalf("since=%v", gotBody["since"])
	}
	if gotBody["until"] != "2026-07-31T23:59:59Z" {
		t.Fatalf("until=%v", gotBody["until"])
	}
	if gotBody["type"] != "memory_recall" {
		t.Fatalf("type=%v", gotBody["type"])
	}
	if gotBody["limit"] != float64(5) {
		t.Fatalf("limit=%v", gotBody["limit"])
	}
}

// s1068: empty temporal fields must not appear in the JSON body (omitempty parity).
func TestRetrieveMemoryWithOptions_OmitsEmptyTemporal(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{"memories": []any{}})
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "t"}, nil)
	_, err := c.RetrieveMemoryWithOptions(context.Background(), "t", MemoryRetrieveOptions{
		Query: "q", Limit: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"since", "until", "session_seq"} {
		if _, ok := gotBody[k]; ok {
			t.Fatalf("unexpected key %q in body: %v", k, gotBody)
		}
	}
	// Thin wrapper still works.
	_, err = c.RetrieveMemory(context.Background(), "t", "q2", 2, "sid")
	if err != nil {
		t.Fatal(err)
	}
}

// s1135: multi-hop related posts to /v1/memory/related with seed/query/max_hops and parses hop_distance.
func TestRetrieveMemoryRelated_SyncHitsAndBody(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.URL.Path != "/v1/memory/related" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"memories": []map[string]any{
				{"id": "h1", "summary": "hop1 fact", "score": 0.9, "hop_distance": 1},
				{"id": "h2", "summary": "hop2 fact", "hop_distance": 2},
			},
		})
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "dept.x"}, nil)
	res, err := c.RetrieveMemoryRelated(context.Background(), "dept.x", MemoryRelatedOptions{
		SeedEntity: "person:alice",
		Query:      "related notes",
		MaxHops:    2,
		Limit:      5,
		SessionID:  "sess-rel",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/memory/related" || res.Path != "/v1/memory/related" {
		t.Fatalf("path got=%q res=%q", gotPath, res.Path)
	}
	if gotBody["tenant_id"] != "dept.x" || gotBody["seed_entity"] != "person:alice" {
		t.Fatalf("body=%v", gotBody)
	}
	if gotBody["query"] != "related notes" || gotBody["max_hops"] != float64(2) {
		t.Fatalf("body hops/query=%v", gotBody)
	}
	if gotBody["session_id"] != "sess-rel" || gotBody["type"] != "memory_related" {
		t.Fatalf("body session/type=%v", gotBody)
	}
	if len(res.Memories) != 2 {
		t.Fatalf("hits=%+v", res.Memories)
	}
	if res.Memories[0].HopDistance != 1 || res.Memories[1].HopDistance != 2 {
		t.Fatalf("hop_distance=%+v", res.Memories)
	}
	if res.Memories[0].Summary != "hop1 fact" {
		t.Fatalf("summary=%q", res.Memories[0].Summary)
	}
}

// s1135: 404 on v1 falls back to v5/memory/related.
func TestRetrieveMemoryRelated_V5Fallback(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path == "/v1/memory/related" {
			w.WriteHeader(404)
			return
		}
		if r.URL.Path == "/v5/memory/related" {
			_ = json.NewEncoder(w).Encode(map[string]any{"memories": []any{}})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "t"}, nil)
	res, err := c.RetrieveMemoryRelated(context.Background(), "t", MemoryRelatedOptions{
		SeedEntity: "person:bob",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Path != "/v5/memory/related" || res.Memories == nil {
		t.Fatalf("%+v paths=%v", res, paths)
	}
}

// s1135: requires seed_entity or query.
func TestRetrieveMemoryRelated_RequiresSeedOrQuery(t *testing.T) {
	c := New(Config{Enabled: true, Endpoint: "http://127.0.0.1:9", Tenant: "t"}, nil)
	_, err := c.RetrieveMemoryRelated(context.Background(), "t", MemoryRelatedOptions{})
	if err == nil || !strings.Contains(err.Error(), "seed_entity or query") {
		t.Fatalf("err=%v", err)
	}
}

// s1135: default max_hops=2 and limit=10 when unset.
func TestRetrieveMemoryRelated_DefaultHopsAndLimit(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{"memories": []any{}})
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "t"}, nil)
	_, err := c.RetrieveMemoryRelated(context.Background(), "t", MemoryRelatedOptions{
		Query: "seed from query only",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotBody["max_hops"] != float64(2) {
		t.Fatalf("max_hops=%v", gotBody["max_hops"])
	}
	if gotBody["limit"] != float64(10) {
		t.Fatalf("limit=%v", gotBody["limit"])
	}
	if _, ok := gotBody["seed_entity"]; ok {
		t.Fatalf("unexpected seed_entity: %v", gotBody)
	}
	// s1281 honesty: nil PreferShorterHops omits field (kernel default true).
	if _, ok := gotBody["prefer_shorter_hops"]; ok {
		t.Fatalf("prefer_shorter_hops must be omitted when nil: %v", gotBody)
	}
}

// s1281: PreferShorterHops false/true sent on body; nil omits (aion s1277 parity).
func TestRetrieveMemoryRelated_PreferShorterHops(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{"memories": []any{}})
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "t"}, nil)
	f := false
	_, err := c.RetrieveMemoryRelated(context.Background(), "t", MemoryRelatedOptions{
		SeedEntity:        "person:alice",
		PreferShorterHops: &f,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotBody["prefer_shorter_hops"] != false {
		t.Fatalf("prefer_shorter_hops=%v want false body=%v", gotBody["prefer_shorter_hops"], gotBody)
	}

	tr := true
	_, err = c.RetrieveMemoryRelated(context.Background(), "t", MemoryRelatedOptions{
		SeedEntity:        "person:alice",
		PreferShorterHops: &tr,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotBody["prefer_shorter_hops"] != true {
		t.Fatalf("prefer_shorter_hops=%v want true body=%v", gotBody["prefer_shorter_hops"], gotBody)
	}
}

// s1200: ops digest posts to /v1/memory/ops_digest with window/horizon/limit and parses patterns+receipts+honesty.
func TestExportOpsDigest_SyncHitsAndBody(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.URL.Path != "/v1/memory/ops_digest" {
			http.NotFound(w, r)
			return
		}
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
				{"id": "p1", "kind": "burst", "subject": "dept.ops", "summary": "spike in deploys", "score": 0.9, "count": 5},
			},
			"receipts": []map[string]any{
				{"id": "r1", "event_time": "2026-08-04T10:00:00Z", "summary": "deploy finished", "source_hint": "palace_timeline"},
			},
			"decision_stub": map[string]any{
				"pattern":      "dept.ops",
				"receipts_ref": []string{"r1"},
			},
		})
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "dept.x"}, nil)
	res, err := c.ExportOpsDigest(context.Background(), "dept.x", MemoryOpsDigestOptions{
		Window:  "day",
		Horizon: "ops",
		Limit:   5,
		AsOf:    "2026-08-04T12:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/memory/ops_digest" || res.Path != "/v1/memory/ops_digest" {
		t.Fatalf("path got=%q res=%q", gotPath, res.Path)
	}
	if gotBody["tenant_id"] != "dept.x" || gotBody["window"] != "day" || gotBody["horizon"] != "ops" {
		t.Fatalf("body=%v", gotBody)
	}
	if gotBody["limit"] != float64(5) || gotBody["as_of"] != "2026-08-04T12:00:00Z" {
		t.Fatalf("body limit/as_of=%v", gotBody)
	}
	if len(res.Patterns) != 1 || res.Patterns[0].Summary != "spike in deploys" {
		t.Fatalf("patterns=%+v", res.Patterns)
	}
	if len(res.Receipts) != 1 || res.Receipts[0].ID != "r1" {
		t.Fatalf("receipts=%+v", res.Receipts)
	}
	if !res.Honesty.NeverInventGA || res.Honesty.OpsPulse != "ga_path" || res.Honesty.Knowledge != "beta" {
		t.Fatalf("honesty=%+v", res.Honesty)
	}
	if res.DecisionStub.Pattern != "dept.ops" {
		t.Fatalf("stub=%+v", res.DecisionStub)
	}
}

// s1200: 404 on v1 falls back to v5/memory/ops_digest.
func TestExportOpsDigest_V5Fallback(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path == "/v1/memory/ops_digest" {
			w.WriteHeader(404)
			return
		}
		if r.URL.Path == "/v5/memory/ops_digest" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"window": "week", "horizon": "ops", "as_of": "now",
				"honesty":  map[string]any{"ops_pulse": "ga_path", "knowledge": "beta", "analytical": "beta", "never_invent_ga": true, "dual_write_default": "off", "book_demo": "off"},
				"patterns": []any{}, "receipts": []any{},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "t"}, nil)
	res, err := c.ExportOpsDigest(context.Background(), "t", MemoryOpsDigestOptions{Window: "week"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Path != "/v5/memory/ops_digest" || res.Patterns == nil || res.Receipts == nil {
		t.Fatalf("%+v paths=%v", res, paths)
	}
}

// s1200: defaults window=day horizon=ops; validates window/horizon; tenant required.
func TestExportOpsDigest_DefaultsAndValidation(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"window": "day", "horizon": "ops", "as_of": "now",
			"honesty":  map[string]any{"ops_pulse": "ga_path"},
			"patterns": []any{}, "receipts": []any{},
		})
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "dept.defaults"}, nil)
	res, err := c.ExportOpsDigest(context.Background(), "dept.defaults", MemoryOpsDigestOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if gotBody["window"] != "day" || gotBody["horizon"] != "ops" {
		t.Fatalf("defaults body=%v", gotBody)
	}
	if _, ok := gotBody["limit"]; ok {
		t.Fatalf("limit should omit when 0: %v", gotBody)
	}
	if res.Window != "day" {
		t.Fatalf("res=%+v", res)
	}

	_, err = c.ExportOpsDigest(context.Background(), "t", MemoryOpsDigestOptions{Window: "month"})
	if err == nil || !strings.Contains(err.Error(), "window") {
		t.Fatalf("window err=%v", err)
	}
	_, err = c.ExportOpsDigest(context.Background(), "t", MemoryOpsDigestOptions{Horizon: "gtm"})
	if err == nil || !strings.Contains(err.Error(), "horizon") {
		t.Fatalf("horizon err=%v", err)
	}
	c2 := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	_, err = c2.ExportOpsDigest(context.Background(), "", MemoryOpsDigestOptions{})
	if err == nil || !strings.Contains(err.Error(), "tenant_id") {
		t.Fatalf("tenant err=%v", err)
	}
}
