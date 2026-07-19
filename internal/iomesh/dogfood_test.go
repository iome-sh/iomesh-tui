package iomesh

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func mockMeshServer(t *testing.T, opts struct {
	failHealth bool
	noReady    bool
	emptyCtx   bool
	failEmit   bool
	failMemory bool
	noMemory   bool
	failRecall bool
	noRecall   bool
}) *httptest.Server {
	t.Helper()
	var emits atomic.Int32
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			if opts.failHealth {
				w.WriteHeader(503)
				return
			}
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case r.URL.Path == "/ready" || r.URL.Path == "/readyz":
			if opts.noReady {
				w.WriteHeader(404)
				return
			}
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ready"))
		case r.URL.Path == "/v1/context/query" && r.Method == http.MethodPost:
			if opts.emptyCtx {
				_ = json.NewEncoder(w).Encode(map[string]string{"text": ""})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"text": "stage-context: ops green"})
		case r.URL.Path == "/v1/streams/dept/publish" && r.Method == http.MethodPost:
			emits.Add(1)
			if opts.failEmit {
				w.WriteHeader(500)
				return
			}
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			subject, _ := body["subject"].(string)
			payloadB64, _ := body["payload"].(string)
			raw, err := base64.StdEncoding.DecodeString(payloadB64)
			if err != nil || subject == "" {
				w.WriteHeader(400)
				return
			}
			// Accept generic dogfood emit and llm_meter probe (dept.agent.llm_call).
			if !strings.Contains(string(raw), "dept.agent.dogfood") &&
				!strings.Contains(string(raw), "dept.agent.llm_call") &&
				!strings.Contains(subject, "dept.agent.") {
				w.WriteHeader(400)
				return
			}
			w.WriteHeader(204)
		case r.URL.Path == "/v1/streams/MEMORY_INGEST/publish" && r.Method == http.MethodPost:
			if opts.noMemory {
				w.WriteHeader(404)
				return
			}
			if opts.failMemory {
				w.WriteHeader(500)
				return
			}
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			subject, _ := body["subject"].(string)
			if subject == "" || !strings.HasSuffix(subject, ".memory.ingest.turn") {
				w.WriteHeader(400)
				return
			}
			payloadB64, _ := body["payload"].(string)
			raw, err := base64.StdEncoding.DecodeString(payloadB64)
			if err != nil {
				w.WriteHeader(400)
				return
			}
			var env map[string]any
			if err := json.Unmarshal(raw, &env); err != nil {
				w.WriteHeader(400)
				return
			}
			if env["type"] != "memory_ingest" || env["content"] == "" {
				w.WriteHeader(400)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"stream":  "MEMORY_INGEST",
				"seq":     1,
				"subject": subject,
			})
		case r.URL.Path == "/v1/streams/MEMORY_RPC/publish" && r.Method == http.MethodPost:
			if opts.noRecall {
				w.WriteHeader(404)
				return
			}
			if opts.failRecall {
				w.WriteHeader(500)
				return
			}
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			subject, _ := body["subject"].(string)
			if subject == "" || !strings.HasSuffix(subject, ".memory.retrieve.request") {
				w.WriteHeader(400)
				return
			}
			payloadB64, _ := body["payload"].(string)
			raw, err := base64.StdEncoding.DecodeString(payloadB64)
			if err != nil {
				w.WriteHeader(400)
				return
			}
			var env map[string]any
			if err := json.Unmarshal(raw, &env); err != nil {
				w.WriteHeader(400)
				return
			}
			if env["type"] != "memory_recall" || env["query"] == "" {
				w.WriteHeader(400)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"stream":  "MEMORY_RPC",
				"seq":     2,
				"subject": subject,
			})
		case (r.URL.Path == "/v1/memory/retrieve" || r.URL.Path == "/v5/memory/retrieve") && r.Method == http.MethodPost:
			if opts.noRecall {
				// reuse noRecall for missing sync retrieve surface
				w.WriteHeader(404)
				return
			}
			if opts.failRecall {
				w.WriteHeader(500)
				return
			}
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			tenant, _ := body["tenant_id"].(string)
			query, _ := body["query"].(string)
			if tenant == "" || query == "" {
				w.WriteHeader(400)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"memories": []map[string]any{
					{
						"id":      "mem_dogfood_1",
						"summary": "iomesh-tui dual-write dogfood",
						"full":    "iomesh-tui dual-write dogfood",
						"score":   0.9,
					},
				},
			})
		case r.URL.Path == "/v1/catalog/data-products" || r.URL.Path == "/v1/catalog/products":
			// Default catalog products for dogfood catalog evidence (s292).
			_ = json.NewEncoder(w).Encode(map[string]any{
				"products": []map[string]string{
					{"id": "ops-incidents", "layer": "operational", "subject": "dept.sre.incidents", "title": "Incidents"},
					{"id": "crm-contacts", "layer": "knowledge", "subject": "dept.sales.contacts", "name": "CRM"},
				},
			})
		case r.URL.Path == "/v1/streams" && r.Method == http.MethodGet:
			// Default streams list for dogfood streams step (s300) — empty OK for full-pass.
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestDogfood_FullPass(t *testing.T) {
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{})
	t.Cleanup(srv.Close)

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "stage",
		ContextPlane: true, EmitDeptStreams: true,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{Strict: true, Workspace: "/ws"})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	out := FormatReport(rep)
	if !strings.Contains(out, "RESULT=PASS") {
		t.Fatal(out)
	}
	if !strings.Contains(out, "health") || !strings.Contains(out, "context") {
		t.Fatal(out)
	}
	var memOK, recallOK, retrieveOK, llmMeterOK bool
	for _, s := range rep.Steps {
		if s.Name == "llm_meter" && s.Status == StepPass {
			llmMeterOK = true
			if !strings.Contains(s.Detail, "dept.agent.llm_call") || !strings.Contains(s.Detail, "session_id=stage.mesh-dogfood") {
				t.Fatalf("llm_meter detail: %s", s.Detail)
			}
		}
		if s.Name == "memory_ingest" && s.Status == StepPass {
			memOK = true
			if !strings.Contains(s.Detail, "MEMORY_INGEST") || !strings.Contains(s.Detail, "stage.memory.ingest.turn") {
				t.Fatalf("memory detail: %s", s.Detail)
			}
			// Temporal correlation fields from envelope (s243).
			if !strings.Contains(s.Detail, "session_seq=1") {
				t.Fatalf("expected session_seq= in PASS detail: %s", s.Detail)
			}
			if !strings.Contains(s.Detail, "session_id=stage.mesh-dogfood") {
				t.Fatalf("expected session_id= in PASS detail: %s", s.Detail)
			}
		}
		if s.Name == "memory_recall" && s.Status == StepPass {
			recallOK = true
			if !strings.Contains(s.Detail, "MEMORY_RPC") || !strings.Contains(s.Detail, "stage.memory.retrieve.request") {
				t.Fatalf("recall detail: %s", s.Detail)
			}
			// Same session_id as ingest for temporal correlation (s247).
			if !strings.Contains(s.Detail, "session_id=stage.mesh-dogfood") {
				t.Fatalf("expected correlated session_id in recall PASS detail: %s", s.Detail)
			}
			if !strings.Contains(s.Detail, "dual_write=") {
				t.Fatalf("expected dual_write= in recall PASS detail: %s", s.Detail)
			}
		}
		if s.Name == "memory_retrieve" && s.Status == StepPass {
			retrieveOK = true
			if !strings.Contains(s.Detail, "POST /v1/memory/retrieve") || !strings.Contains(s.Detail, "hits=") {
				t.Fatalf("retrieve detail: %s", s.Detail)
			}
			if !strings.Contains(s.Detail, "session_id=stage.mesh-dogfood") {
				t.Fatalf("expected session_id in retrieve PASS: %s", s.Detail)
			}
			if !strings.Contains(s.Detail, "memory_base=mesh") {
				t.Fatalf("expected memory_base=mesh without sidecar: %s", s.Detail)
			}
			if strings.Contains(s.Detail, "MEMORY_RPC") {
				t.Fatalf("sync retrieve must not use MEMORY_RPC: %s", s.Detail)
			}
		}
	}
	if !memOK || !recallOK || !retrieveOK || !llmMeterOK {
		t.Fatal(FormatReport(rep))
	}
	js := FormatReportJSON(rep)
	if !strings.Contains(js, `"result": "PASS"`) || !strings.Contains(js, `"ok": true`) {
		t.Fatal(js)
	}
}

func TestDogfood_LLMMeterOrgHeaders(t *testing.T) {
	var gotOrg, gotWS string
	var llmBodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health", r.URL.Path == "/ready", r.URL.Path == "/readyz":
			w.WriteHeader(200)
		case r.URL.Path == "/v1/context/query":
			_ = json.NewEncoder(w).Encode(map[string]string{"text": "ctx"})
		case r.URL.Path == "/v1/streams/dept/publish" && r.Method == http.MethodPost:
			gotOrg = r.Header.Get("X-IOMesh-Org")
			gotWS = r.Header.Get("X-IOMesh-Workspace")
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if s, ok := body["payload"].(string); ok {
				raw, _ := base64.StdEncoding.DecodeString(s)
				llmBodies = append(llmBodies, string(raw))
			}
			w.WriteHeader(204)
		case strings.Contains(r.URL.Path, "MEMORY") || strings.Contains(r.URL.Path, "memory"):
			// soft skip memory paths if hit
			if strings.HasSuffix(r.URL.Path, "/publish") {
				_ = json.NewEncoder(w).Encode(map[string]any{"stream": "MEMORY_INGEST", "seq": 1})
				return
			}
			if strings.Contains(r.URL.Path, "retrieve") {
				_ = json.NewEncoder(w).Encode(map[string]any{"memories": []any{}})
				return
			}
			http.NotFound(w, r)
		default:
			// catalog etc
			if strings.Contains(r.URL.Path, "catalog") {
				_ = json.NewEncoder(w).Encode(map[string]any{"products": []any{}})
				return
			}
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "stage",
		OrgID: "org_meter", WorkspaceID: "ws_m",
		ContextPlane: true, EmitDeptStreams: true, CatalogPlane: false,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{SkipMemory: true})
	if !rep.OK {
		t.Fatal(FormatReport(rep))
	}
	var found bool
	for _, s := range rep.Steps {
		if s.Name == "llm_meter" {
			found = true
			if s.Status != StepPass {
				t.Fatalf("llm_meter=%s %s", s.Status, s.Detail)
			}
			if !strings.Contains(s.Detail, "org=org_meter") || !strings.Contains(s.Detail, "workspace=ws_m") {
				t.Fatalf("detail=%s", s.Detail)
			}
		}
	}
	if !found {
		t.Fatal(FormatReport(rep))
	}
	if gotOrg != "org_meter" || gotWS != "ws_m" {
		t.Fatalf("headers org=%q ws=%q", gotOrg, gotWS)
	}
	joined := strings.Join(llmBodies, "\n")
	if !strings.Contains(joined, "dept.agent.llm_call") {
		t.Fatalf("bodies=%v", llmBodies)
	}
}

func TestDogfood_MemorySidecarEndpoint(t *testing.T) {
	// Broker serves health/streams but not sync retrieve; sidecar serves retrieve.
	broker := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{noRecall: true}) // 404 on broker /v1/memory/retrieve

	var sidecarHits int
	sidecar := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/memory/retrieve" && r.Method == http.MethodPost {
			sidecarHits++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"memories": []map[string]any{
					{"id": "w1", "summary": "warm hit", "score": 0.7},
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer sidecar.Close()

	c := New(Config{
		Enabled: true, Endpoint: broker.URL, Tenant: "stage",
		MemoryEndpoint: sidecar.URL,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{})
	if !rep.OK {
		t.Fatal(FormatReport(rep))
	}
	wantBase := strings.TrimRight(sidecar.URL, "/")
	if rep.MemoryEndpoint != wantBase {
		t.Fatalf("memory_endpoint=%q want=%q", rep.MemoryEndpoint, wantBase)
	}
	var retrieveDetail string
	for _, s := range rep.Steps {
		if s.Name == "memory_retrieve" {
			if s.Status != StepPass {
				t.Fatalf("retrieve=%s detail=%s", s.Status, s.Detail)
			}
			retrieveDetail = s.Detail
		}
	}
	if sidecarHits != 1 {
		t.Fatalf("sidecar hits=%d", sidecarHits)
	}
	if !strings.Contains(retrieveDetail, "hits=1") || !strings.Contains(retrieveDetail, "memory_base=sidecar") {
		t.Fatalf("detail=%q", retrieveDetail)
	}
	js := FormatReportJSON(rep)
	if !strings.Contains(js, `"memory_endpoint"`) {
		t.Fatal(js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "memory_endpoint:") {
		t.Fatal(text)
	}
}

func TestDogfood_MemoryIngestSessionDetail(t *testing.T) {
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{})
	t.Cleanup(srv.Close)

	// Tenant-prefixed session_id for multi-tenant evidence (s243).
	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "acme",
		EmitDeptStreams: true,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{Strict: true})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	var found bool
	for _, s := range rep.Steps {
		if s.Name == "memory_ingest" && s.Status == StepPass {
			found = true
			if !strings.Contains(s.Detail, "session_seq=") {
				t.Fatalf("PASS detail must contain session_seq=: %s", s.Detail)
			}
			if !strings.Contains(s.Detail, "session_id=acme.mesh-dogfood") {
				t.Fatalf("PASS detail must contain session_id when envelope has id: %s", s.Detail)
			}
			// dual_write still present after session fields (s241).
			if !strings.Contains(s.Detail, "dual_write=false") {
				t.Fatalf("expected dual_write= in PASS detail: %s", s.Detail)
			}
		}
	}
	if !found {
		t.Fatal(FormatReport(rep))
	}
}

func TestDogfood_HealthFail(t *testing.T) {
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{failHealth: true})
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL, ContextPlane: true, EmitDeptStreams: true}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{})
	if rep.OK {
		t.Fatal("expected fail")
	}
	if !strings.Contains(rep.Summary, "FAIL") {
		t.Fatal(rep.Summary)
	}
}

func TestDogfood_SoftContextSkip(t *testing.T) {
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{emptyCtx: true, noReady: true})
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL, ContextPlane: true, EmitDeptStreams: true}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{Strict: false})
	if !rep.OK {
		t.Fatalf("soft mode should pass: %s\n%s", rep.Summary, FormatReport(rep))
	}
	// context should be SKIP
	var found bool
	for _, s := range rep.Steps {
		if s.Name == "context" && s.Status == StepSkip {
			found = true
		}
	}
	if !found {
		t.Fatal(FormatReport(rep))
	}
}

func TestDogfood_StrictContextFail(t *testing.T) {
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{emptyCtx: true})
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL, ContextPlane: true, EmitDeptStreams: true}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{Strict: true})
	if rep.OK {
		t.Fatal("strict should fail empty context")
	}
}

func TestDogfood_Disabled(t *testing.T) {
	c := New(Config{Enabled: false}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{})
	if !rep.OK || !strings.Contains(rep.Summary, "SKIP") {
		t.Fatalf("%+v", rep)
	}
	// No memory_ingest step when mesh is disabled (early return).
	for _, s := range rep.Steps {
		if s.Name == "memory_ingest" {
			t.Fatalf("unexpected memory_ingest step when disabled: %+v", s)
		}
	}
}

func TestDogfood_MemoryIngestSoftSkip(t *testing.T) {
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{failMemory: true})
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "stage", EmitDeptStreams: true}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{Strict: false})
	if !rep.OK {
		t.Fatalf("soft memory fail should not fail report: %s\n%s", rep.Summary, FormatReport(rep))
	}
	var found bool
	for _, s := range rep.Steps {
		if s.Name == "memory_ingest" && s.Status == StepSkip {
			found = true
			if !strings.Contains(s.Detail, "soft-fail") {
				t.Fatalf("detail=%s", s.Detail)
			}
		}
	}
	if !found {
		t.Fatal(FormatReport(rep))
	}
}

func TestDogfood_MemoryIngestStrictFail(t *testing.T) {
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{failMemory: true})
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "stage", EmitDeptStreams: true}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{Strict: true})
	if rep.OK {
		t.Fatal("strict should fail memory_ingest")
	}
	var found bool
	for _, s := range rep.Steps {
		if s.Name == "memory_ingest" && s.Status == StepFail {
			found = true
		}
	}
	if !found {
		t.Fatal(FormatReport(rep))
	}
}

func TestDogfood_SkipMemory(t *testing.T) {
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{failMemory: true, failRecall: true}) // would fail if not skipped
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "stage", EmitDeptStreams: true}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{Strict: true, SkipMemory: true})
	if !rep.OK {
		t.Fatalf("skip-memory should pass: %s\n%s", rep.Summary, FormatReport(rep))
	}
	var foundIngest, foundRecall, foundRetrieve bool
	for _, s := range rep.Steps {
		if s.Name == "memory_ingest" && s.Status == StepSkip {
			foundIngest = true
			if !strings.Contains(s.Detail, "skip-memory") {
				t.Fatalf("detail=%s", s.Detail)
			}
		}
		if s.Name == "memory_recall" && s.Status == StepSkip {
			foundRecall = true
			if !strings.Contains(s.Detail, "skip-memory") {
				t.Fatalf("detail=%s", s.Detail)
			}
		}
		if s.Name == "memory_retrieve" && s.Status == StepSkip {
			foundRetrieve = true
			if !strings.Contains(s.Detail, "skip-memory") {
				t.Fatalf("detail=%s", s.Detail)
			}
		}
	}
	if !foundIngest || !foundRecall || !foundRetrieve {
		t.Fatal(FormatReport(rep))
	}
}

func TestDogfood_MemoryRetrieveSyncHits(t *testing.T) {
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{})
	t.Cleanup(srv.Close)

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "acme",
		OrgID: "org_sync", WorkspaceID: "ws_sync", DualWrite: false,
		EmitDeptStreams: true,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{Strict: true})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	var found bool
	for _, s := range rep.Steps {
		if s.Name == "memory_retrieve" && s.Status == StepPass {
			found = true
			if !strings.Contains(s.Detail, "hits=1") {
				t.Fatalf("want hits=1: %s", s.Detail)
			}
			if !strings.Contains(s.Detail, "session_id=acme.mesh-dogfood") {
				t.Fatalf("session: %s", s.Detail)
			}
			if !strings.Contains(s.Detail, "org=org_sync") || !strings.Contains(s.Detail, "workspace=ws_sync") {
				t.Fatalf("org/ws: %s", s.Detail)
			}
			if !strings.Contains(s.Detail, "dual_write=false") {
				t.Fatalf("dual_write: %s", s.Detail)
			}
		}
	}
	if !found {
		t.Fatal(FormatReport(rep))
	}
}

func TestDogfood_MemoryRetrieveSoftSkip(t *testing.T) {
	// failRecall makes MEMORY_RPC and /v1/memory/retrieve return 500.
	// memory_recall soft-skips; memory_retrieve should also soft-skip.
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{failRecall: true})
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "stage", EmitDeptStreams: true}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{Strict: false})
	if !rep.OK {
		t.Fatalf("soft retrieve fail should not fail report: %s\n%s", rep.Summary, FormatReport(rep))
	}
	var found bool
	for _, s := range rep.Steps {
		if s.Name == "memory_retrieve" && s.Status == StepSkip {
			found = true
			if !strings.Contains(s.Detail, "soft-fail") {
				t.Fatalf("detail=%s", s.Detail)
			}
		}
	}
	if !found {
		t.Fatal(FormatReport(rep))
	}
}

func TestDogfood_MemoryRecallSessionDetail(t *testing.T) {
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{})
	t.Cleanup(srv.Close)

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "acme",
		OrgID: "org_dev", WorkspaceID: "ws_1", DualWrite: true,
		EmitDeptStreams: true,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{Strict: true})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	var ingestID, recallID string
	for _, s := range rep.Steps {
		if s.Name == "memory_ingest" && s.Status == StepPass {
			if !strings.Contains(s.Detail, "session_id=acme.mesh-dogfood") {
				t.Fatalf("ingest: %s", s.Detail)
			}
			ingestID = "acme.mesh-dogfood"
		}
		if s.Name == "memory_recall" && s.Status == StepPass {
			if !strings.Contains(s.Detail, "MEMORY_RPC") {
				t.Fatalf("recall stream: %s", s.Detail)
			}
			if !strings.Contains(s.Detail, "session_id=acme.mesh-dogfood") {
				t.Fatalf("recall session: %s", s.Detail)
			}
			if !strings.Contains(s.Detail, "org=org_dev") || !strings.Contains(s.Detail, "workspace=ws_1") {
				t.Fatalf("recall org/ws: %s", s.Detail)
			}
			if !strings.Contains(s.Detail, "dual_write=true") {
				t.Fatalf("recall dual_write: %s", s.Detail)
			}
			recallID = "acme.mesh-dogfood"
		}
	}
	if ingestID == "" || recallID == "" || ingestID != recallID {
		t.Fatalf("correlation ingest=%q recall=%q\n%s", ingestID, recallID, FormatReport(rep))
	}
}

func TestDogfood_MemoryRecallSoftSkip(t *testing.T) {
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{failRecall: true})
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "stage", EmitDeptStreams: true}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{Strict: false})
	if !rep.OK {
		t.Fatalf("soft recall fail should not fail report: %s\n%s", rep.Summary, FormatReport(rep))
	}
	var found bool
	for _, s := range rep.Steps {
		if s.Name == "memory_recall" && s.Status == StepSkip {
			found = true
			if !strings.Contains(s.Detail, "soft-fail") {
				t.Fatalf("detail=%s", s.Detail)
			}
		}
	}
	if !found {
		t.Fatal(FormatReport(rep))
	}
}

func TestDogfood_MemoryRecallStrictFail(t *testing.T) {
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{failRecall: true})
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "stage", EmitDeptStreams: true}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{Strict: true})
	if rep.OK {
		t.Fatal("strict should fail memory_recall")
	}
	var found bool
	for _, s := range rep.Steps {
		if s.Name == "memory_recall" && s.Status == StepFail {
			found = true
		}
	}
	if !found {
		t.Fatal(FormatReport(rep))
	}
}

func TestDogfood_MemoryIngestOrgWorkspaceEvidence(t *testing.T) {
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{})
	t.Cleanup(srv.Close)

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "stage",
		EmitDeptStreams: true,
		OrgID:           "org_dev-org",
		WorkspaceID:     "ws_alpha",
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{Strict: true})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if rep.Org != "org_dev-org" || rep.Workspace != "ws_alpha" {
		t.Fatalf("report org/workspace: org=%q workspace=%q", rep.Org, rep.Workspace)
	}
	var found bool
	for _, s := range rep.Steps {
		if s.Name == "memory_ingest" && s.Status == StepPass {
			found = true
			if !strings.Contains(s.Detail, "org=org_dev-org") {
				t.Fatalf("expected org evidence in detail: %s", s.Detail)
			}
			if !strings.Contains(s.Detail, "workspace=ws_alpha") {
				t.Fatalf("expected workspace evidence in detail: %s", s.Detail)
			}
			if !strings.Contains(s.Detail, "MEMORY_INGEST") || !strings.Contains(s.Detail, "seq=") {
				t.Fatalf("expected stream/seq in detail: %s", s.Detail)
			}
		}
	}
	if !found {
		t.Fatal(FormatReport(rep))
	}
	// Structured JSON fields for stage CI / aion gates (s237).
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if parsed["org"] != "org_dev-org" {
		t.Fatalf("json org: %v\n%s", parsed["org"], js)
	}
	if parsed["workspace"] != "ws_alpha" {
		t.Fatalf("json workspace: %v\n%s", parsed["workspace"], js)
	}
}

func TestDogfood_MemoryIngestOmitsEmptyOrgWorkspace(t *testing.T) {
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{})
	t.Cleanup(srv.Close)

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "stage",
		EmitDeptStreams: true,
		// OrgID / WorkspaceID intentionally unset
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{Strict: true})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if rep.Org != "" || rep.Workspace != "" {
		t.Fatalf("unset org/workspace must be empty: org=%q workspace=%q", rep.Org, rep.Workspace)
	}
	var found bool
	for _, s := range rep.Steps {
		if s.Name == "memory_ingest" && s.Status == StepPass {
			found = true
			if strings.Contains(s.Detail, "org=") {
				t.Fatalf("unset OrgID must not print org=: %s", s.Detail)
			}
			if strings.Contains(s.Detail, "workspace=") {
				t.Fatalf("unset WorkspaceID must not print workspace=: %s", s.Detail)
			}
			if !strings.Contains(s.Detail, "MEMORY_INGEST") {
				t.Fatalf("detail: %s", s.Detail)
			}
		}
	}
	if !found {
		t.Fatal(FormatReport(rep))
	}
	// omitempty: top-level org/workspace keys absent when unset.
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if _, ok := parsed["org"]; ok {
		t.Fatalf("unset org must be omitted from JSON: %s", js)
	}
	if _, ok := parsed["workspace"]; ok {
		t.Fatalf("unset workspace must be omitted from JSON: %s", js)
	}
}

func TestDogfood_JSONDualWriteTrue(t *testing.T) {
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{})
	t.Cleanup(srv.Close)

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "stage",
		EmitDeptStreams: true,
		DualWrite:       true,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{Strict: true})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if !rep.DualWrite {
		t.Fatal("expected DualWrite true on report")
	}
	// memory_ingest still runs (not gated on DualWrite); PASS detail includes dual_write=true (s241).
	var memOK bool
	for _, s := range rep.Steps {
		if s.Name == "memory_ingest" && s.Status == StepPass {
			memOK = true
			if !strings.Contains(s.Detail, "dual_write=true") {
				t.Fatalf("expected dual_write=true in PASS detail: %s", s.Detail)
			}
		}
	}
	if !memOK {
		t.Fatal(FormatReport(rep))
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if parsed["dual_write"] != true {
		t.Fatalf("json dual_write: %v\n%s", parsed["dual_write"], js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "dual_write: true") {
		t.Fatalf("text report missing dual_write: %s", text)
	}
}

func TestDogfood_JSONDualWriteFalse(t *testing.T) {
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{})
	t.Cleanup(srv.Close)

	// DualWrite intentionally unset (default false).
	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "stage",
		EmitDeptStreams: true,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{Strict: true})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if rep.DualWrite {
		t.Fatal("expected DualWrite false on report when unset")
	}
	// PASS detail always includes dual_write=false when unset (s241).
	var memOK bool
	for _, s := range rep.Steps {
		if s.Name == "memory_ingest" && s.Status == StepPass {
			memOK = true
			if !strings.Contains(s.Detail, "dual_write=false") {
				t.Fatalf("expected dual_write=false in PASS detail: %s", s.Detail)
			}
		}
	}
	if !memOK {
		t.Fatal(FormatReport(rep))
	}
	// Always emit dual_write key (unlike omitempty org/workspace).
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	v, ok := parsed["dual_write"]
	if !ok {
		t.Fatalf("dual_write must always be present in JSON: %s", js)
	}
	if v != false {
		t.Fatalf("json dual_write: %v\n%s", v, js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "dual_write: false") {
		t.Fatalf("text report missing dual_write: %s", text)
	}
}

func TestDogfood_CatalogEvidence(t *testing.T) {
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{})
	t.Cleanup(srv.Close)

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "stage",
		EmitDeptStreams: true,
		CatalogPlane:    true,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{Strict: true})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if rep.CatalogSource != "mesh" && rep.CatalogSource != "portal" {
		t.Fatalf("CatalogSource: %q want mesh|portal", rep.CatalogSource)
	}
	if rep.CatalogCount <= 0 {
		t.Fatalf("CatalogCount: %d want > 0", rep.CatalogCount)
	}
	var catOK bool
	for _, s := range rep.Steps {
		if s.Name == "catalog" && s.Status == StepPass {
			catOK = true
		}
	}
	if !catOK {
		t.Fatal(FormatReport(rep))
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if parsed["catalog_source"] != rep.CatalogSource {
		t.Fatalf("json catalog_source: %v\n%s", parsed["catalog_source"], js)
	}
	// JSON numbers decode as float64.
	if n, ok := parsed["catalog_count"].(float64); !ok || int(n) != rep.CatalogCount {
		t.Fatalf("json catalog_count: %v want %d\n%s", parsed["catalog_count"], rep.CatalogCount, js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "catalog_source: "+rep.CatalogSource) {
		t.Fatalf("text report missing catalog_source:\n%s", text)
	}
	if !strings.Contains(text, fmt.Sprintf("catalog_count: %d", rep.CatalogCount)) {
		t.Fatalf("text report missing catalog_count:\n%s", text)
	}
}

func TestDogfood_CatalogEvidenceOff(t *testing.T) {
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{})
	t.Cleanup(srv.Close)

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "stage",
		EmitDeptStreams: true,
		CatalogPlane:    false,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if rep.CatalogSource != "off" {
		t.Fatalf("CatalogSource: %q want off", rep.CatalogSource)
	}
	if rep.CatalogCount != 0 {
		t.Fatalf("CatalogCount: %d want 0", rep.CatalogCount)
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if parsed["catalog_source"] != "off" {
		t.Fatalf("json catalog_source: %v\n%s", parsed["catalog_source"], js)
	}
	if n, ok := parsed["catalog_count"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json catalog_count: %v want 0\n%s", parsed["catalog_count"], js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "catalog_source: off") {
		t.Fatalf("text report missing catalog_source off:\n%s", text)
	}
	if !strings.Contains(text, "catalog_count: 0") {
		t.Fatalf("text report missing catalog_count:\n%s", text)
	}
}

func TestDogfood_PolicyEvidenceOff(t *testing.T) {
	// Default policy mode off → policy_mode=off, policy_source=off, policy_allow omitted.
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{})
	t.Cleanup(srv.Close)

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "stage",
		EmitDeptStreams: true,
		PolicyMode:      PolicyOff,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if rep.PolicyMode != "off" {
		t.Fatalf("PolicyMode: %q want off", rep.PolicyMode)
	}
	if rep.PolicySource != "off" {
		t.Fatalf("PolicySource: %q want off", rep.PolicySource)
	}
	if rep.PolicyAllow != nil {
		t.Fatalf("PolicyAllow: %v want nil when mode off", *rep.PolicyAllow)
	}
	pol, ok := dogfoodStep(rep, "policy")
	if !ok || pol.Status != StepSkip {
		t.Fatalf("policy step: ok=%v status=%s", ok, pol.Status)
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if parsed["policy_mode"] != "off" {
		t.Fatalf("json policy_mode: %v\n%s", parsed["policy_mode"], js)
	}
	if parsed["policy_source"] != "off" {
		t.Fatalf("json policy_source: %v\n%s", parsed["policy_source"], js)
	}
	if _, has := parsed["policy_allow"]; has {
		t.Fatalf("json policy_allow must be omitted when mode off:\n%s", js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "policy_mode: off") {
		t.Fatalf("text report missing policy_mode:\n%s", text)
	}
	if !strings.Contains(text, "policy_source: off") {
		t.Fatalf("text report missing policy_source:\n%s", text)
	}
	if strings.Contains(text, "policy_allow:") {
		t.Fatalf("text report must omit policy_allow when mode off:\n%s", text)
	}
}

func TestDogfood_PolicyEvidenceMeshAllow(t *testing.T) {
	// Policy mode advisory + mesh allow=true → PASS, policy_source=mesh, policy_allow=true.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case r.URL.Path == "/ready" || r.URL.Path == "/readyz":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ready"))
		case r.URL.Path == "/v1/policy/evaluate" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"allow": true, "reasons": []string{"dogfood ok"}})
		case r.URL.Path == "/v1/streams" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		default:
			// Soft-skip other optional planes.
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "stage",
		PolicyMode:      PolicyAdvisory,
		EmitDeptStreams: false,
		CatalogPlane:    false,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{
		SkipContext: true,
		SkipEmit:    true,
		SkipMemory:  true,
	})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if rep.PolicyMode != "advisory" {
		t.Fatalf("PolicyMode: %q want advisory", rep.PolicyMode)
	}
	if rep.PolicySource != "mesh" {
		t.Fatalf("PolicySource: %q want mesh", rep.PolicySource)
	}
	if rep.PolicyAllow == nil || !*rep.PolicyAllow {
		t.Fatalf("PolicyAllow: %v want true", rep.PolicyAllow)
	}
	pol, ok := dogfoodStep(rep, "policy")
	if !ok || pol.Status != StepPass {
		t.Fatalf("policy step: ok=%v status=%s detail=%s", ok, pol.Status, pol.Detail)
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if parsed["policy_mode"] != "advisory" {
		t.Fatalf("json policy_mode: %v\n%s", parsed["policy_mode"], js)
	}
	if parsed["policy_source"] != "mesh" {
		t.Fatalf("json policy_source: %v\n%s", parsed["policy_source"], js)
	}
	if parsed["policy_allow"] != true {
		t.Fatalf("json policy_allow: %v want true\n%s", parsed["policy_allow"], js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "policy_mode: advisory") || !strings.Contains(text, "policy_source: mesh") ||
		!strings.Contains(text, "policy_allow: true") {
		t.Fatalf("text report missing policy evidence:\n%s", text)
	}
}

func TestDogfood_PolicyEvidenceMeshDeny(t *testing.T) {
	// Policy mode enforce + mesh allow=false → still PASS step (decision evidence), policy_allow=false.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case r.URL.Path == "/ready" || r.URL.Path == "/readyz":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ready"))
		case r.URL.Path == "/v1/policy/evaluate" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"allow": false, "reasons": []string{"denied for dogfood"}})
		case r.URL.Path == "/v1/streams" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "stage",
		PolicyMode:      PolicyEnforce,
		EmitDeptStreams: false,
		CatalogPlane:    false,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{
		SkipContext: true,
		SkipEmit:    true,
		SkipMemory:  true,
	})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if rep.PolicyMode != "enforce" {
		t.Fatalf("PolicyMode: %q want enforce", rep.PolicyMode)
	}
	if rep.PolicySource != "mesh" {
		t.Fatalf("PolicySource: %q want mesh", rep.PolicySource)
	}
	if rep.PolicyAllow == nil || *rep.PolicyAllow {
		t.Fatalf("PolicyAllow: %v want false", rep.PolicyAllow)
	}
	pol, ok := dogfoodStep(rep, "policy")
	if !ok || pol.Status != StepPass {
		t.Fatalf("policy step: ok=%v status=%s detail=%s", ok, pol.Status, pol.Detail)
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if parsed["policy_mode"] != "enforce" {
		t.Fatalf("json policy_mode: %v\n%s", parsed["policy_mode"], js)
	}
	if parsed["policy_source"] != "mesh" {
		t.Fatalf("json policy_source: %v\n%s", parsed["policy_source"], js)
	}
	if parsed["policy_allow"] != false {
		t.Fatalf("json policy_allow: %v want false\n%s", parsed["policy_allow"], js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "policy_mode: enforce") || !strings.Contains(text, "policy_source: mesh") ||
		!strings.Contains(text, "policy_allow: false") {
		t.Fatalf("text report missing policy deny evidence:\n%s", text)
	}
}

func TestDogfood_StreamsEvidence(t *testing.T) {
	// ListStreams returns 2 streams → step PASS, streams_count=2, streams_names sample, JSON always emits.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case r.URL.Path == "/ready" || r.URL.Path == "/readyz":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ready"))
		case r.URL.Path == "/v1/streams" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"name": "MEMORY_INGEST", "subjects": []string{"*.memory.ingest.turn"}, "messages": 10},
				{"name": "dept", "subjects": []string{"dept.>"}, "messages": 3},
			})
		default:
			// Soft paths for other probes
			if strings.Contains(r.URL.Path, "catalog") {
				_ = json.NewEncoder(w).Encode(map[string]any{"products": []any{}})
				return
			}
			if strings.Contains(r.URL.Path, "MEMORY") || strings.Contains(r.URL.Path, "memory") {
				if strings.HasSuffix(r.URL.Path, "/publish") {
					_ = json.NewEncoder(w).Encode(map[string]any{"stream": "MEMORY_INGEST", "seq": 1})
					return
				}
				if strings.Contains(r.URL.Path, "retrieve") {
					_ = json.NewEncoder(w).Encode(map[string]any{"memories": []any{}})
					return
				}
			}
			if r.URL.Path == "/v1/streams/dept/publish" {
				w.WriteHeader(204)
				return
			}
			if r.URL.Path == "/v1/context/query" {
				_ = json.NewEncoder(w).Encode(map[string]string{"text": "ctx"})
				return
			}
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "stage",
		EmitDeptStreams: true,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{SkipMemory: true})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if rep.StreamsCount != 2 {
		t.Fatalf("StreamsCount: %d want 2", rep.StreamsCount)
	}
	if len(rep.StreamsNames) != 2 || rep.StreamsNames[0] != "MEMORY_INGEST" || rep.StreamsNames[1] != "dept" {
		t.Fatalf("StreamsNames: %v want [MEMORY_INGEST dept]", rep.StreamsNames)
	}
	var streamsOK bool
	for _, s := range rep.Steps {
		if s.Name == "streams" && s.Status == StepPass {
			streamsOK = true
			if !strings.Contains(s.Detail, "n=2") {
				t.Fatalf("streams detail: %s", s.Detail)
			}
			if !strings.Contains(s.Detail, "MEMORY_INGEST") || !strings.Contains(s.Detail, "dept") {
				t.Fatalf("streams names in detail: %s", s.Detail)
			}
		}
	}
	if !streamsOK {
		t.Fatal(FormatReport(rep))
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if n, ok := parsed["streams_count"].(float64); !ok || int(n) != 2 {
		t.Fatalf("json streams_count: %v want 2\n%s", parsed["streams_count"], js)
	}
	namesRaw, ok := parsed["streams_names"].([]any)
	if !ok {
		t.Fatalf("json streams_names not array: %T %v\n%s", parsed["streams_names"], parsed["streams_names"], js)
	}
	if len(namesRaw) != 2 || namesRaw[0] != "MEMORY_INGEST" || namesRaw[1] != "dept" {
		t.Fatalf("json streams_names: %v\n%s", namesRaw, js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "streams_count: 2") {
		t.Fatalf("text report missing streams_count:\n%s", text)
	}
	if !strings.Contains(text, "streams_names: MEMORY_INGEST,dept") {
		t.Fatalf("text report missing streams_names:\n%s", text)
	}
}

func TestDogfood_StreamsSoftFail(t *testing.T) {
	// ListStreams 500 → soft SKIP, streams_count=0, streams_names=[].
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case r.URL.Path == "/ready" || r.URL.Path == "/readyz":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ready"))
		case r.URL.Path == "/v1/streams" && r.Method == http.MethodGet:
			w.WriteHeader(500)
		default:
			if r.URL.Path == "/v1/context/query" {
				_ = json.NewEncoder(w).Encode(map[string]string{"text": "ctx"})
				return
			}
			if r.URL.Path == "/v1/streams/dept/publish" {
				w.WriteHeader(204)
				return
			}
			if strings.Contains(r.URL.Path, "catalog") {
				_ = json.NewEncoder(w).Encode(map[string]any{"products": []any{}})
				return
			}
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "stage",
		EmitDeptStreams: true,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{SkipMemory: true})
	if !rep.OK {
		t.Fatalf("soft streams fail should not fail report: %s\n%s", rep.Summary, FormatReport(rep))
	}
	if rep.StreamsCount != 0 {
		t.Fatalf("StreamsCount: %d want 0", rep.StreamsCount)
	}
	if len(rep.StreamsNames) != 0 {
		t.Fatalf("StreamsNames: %v want empty", rep.StreamsNames)
	}
	var found bool
	for _, s := range rep.Steps {
		if s.Name == "streams" {
			found = true
			if s.Status != StepSkip {
				t.Fatalf("streams status=%s detail=%s", s.Status, s.Detail)
			}
			if !strings.Contains(s.Detail, "soft-fail") {
				t.Fatalf("detail=%s", s.Detail)
			}
		}
	}
	if !found {
		t.Fatal(FormatReport(rep))
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if n, ok := parsed["streams_count"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json streams_count: %v want 0\n%s", parsed["streams_count"], js)
	}
	namesRaw, ok := parsed["streams_names"].([]any)
	if !ok || len(namesRaw) != 0 {
		t.Fatalf("json streams_names want []: %T %v\n%s", parsed["streams_names"], parsed["streams_names"], js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "streams_names: (none)") {
		t.Fatalf("text report missing streams_names (none):\n%s", text)
	}
}

func TestDogfood_SkipStreams(t *testing.T) {
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{})
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "stage", EmitDeptStreams: true}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{Strict: true, SkipStreams: true, SkipMemory: true})
	if !rep.OK {
		t.Fatalf("skip-streams should pass: %s\n%s", rep.Summary, FormatReport(rep))
	}
	if rep.StreamsCount != 0 {
		t.Fatalf("StreamsCount: %d want 0", rep.StreamsCount)
	}
	if len(rep.StreamsNames) != 0 {
		t.Fatalf("StreamsNames: %v want empty", rep.StreamsNames)
	}
	var found bool
	for _, s := range rep.Steps {
		if s.Name == "streams" && s.Status == StepSkip {
			found = true
			if !strings.Contains(s.Detail, "skip-streams") {
				t.Fatalf("detail=%s", s.Detail)
			}
		}
	}
	if !found {
		t.Fatal(FormatReport(rep))
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if n, ok := parsed["streams_count"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json streams_count: %v want 0\n%s", parsed["streams_count"], js)
	}
	namesRaw, ok := parsed["streams_names"].([]any)
	if !ok || len(namesRaw) != 0 {
		t.Fatalf("json streams_names want []: %T %v\n%s", parsed["streams_names"], parsed["streams_names"], js)
	}
}

func TestDogfood_ContextEvidence(t *testing.T) {
	// Minimal mesh mock: context plane returns text + lineage for top-level evidence (s296).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case r.URL.Path == "/ready" || r.URL.Path == "/readyz":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ready"))
		case r.URL.Path == "/v1/context/query" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"text": "stage-context: ops green",
				"lineage": []map[string]string{
					{"id": "dp-1", "subject": "dept.eng.events", "source": "kafka"},
					{"id": "dp-2", "product": "crm", "source": "portal"},
				},
			})
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "stage",
		ContextPlane: true, IncludeLineage: true,
		EmitDeptStreams: false,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{SkipMemory: true})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	wantText := FormatContextSnippet(ContextResult{
		Text: "stage-context: ops green",
		Lineage: []LineageRef{
			{ID: "dp-1", Subject: "dept.eng.events", Source: "kafka"},
			{ID: "dp-2", Product: "crm", Source: "portal"},
		},
	})
	if rep.ContextChars != len(wantText) || rep.ContextChars <= 0 {
		t.Fatalf("ContextChars: %d want %d (>0)", rep.ContextChars, len(wantText))
	}
	if rep.ContextLineageCount != 2 {
		t.Fatalf("ContextLineageCount: %d want 2", rep.ContextLineageCount)
	}
	var ctxOK bool
	for _, s := range rep.Steps {
		if s.Name == "context" && s.Status == StepPass {
			ctxOK = true
		}
	}
	if !ctxOK {
		t.Fatal(FormatReport(rep))
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if n, ok := parsed["context_chars"].(float64); !ok || int(n) != rep.ContextChars {
		t.Fatalf("json context_chars: %v want %d\n%s", parsed["context_chars"], rep.ContextChars, js)
	}
	if n, ok := parsed["context_lineage_count"].(float64); !ok || int(n) != 2 {
		t.Fatalf("json context_lineage_count: %v want 2\n%s", parsed["context_lineage_count"], js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, fmt.Sprintf("context_chars: %d", rep.ContextChars)) {
		t.Fatalf("text report missing context_chars:\n%s", text)
	}
	if !strings.Contains(text, "context_lineage_count: 2") {
		t.Fatalf("text report missing context_lineage_count:\n%s", text)
	}
}

func TestDogfood_ContextEvidenceSkipped(t *testing.T) {
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{})
	t.Cleanup(srv.Close)

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "stage",
		EmitDeptStreams: true,
		ContextPlane:    false,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if rep.ContextChars != 0 {
		t.Fatalf("ContextChars: %d want 0", rep.ContextChars)
	}
	if rep.ContextLineageCount != 0 {
		t.Fatalf("ContextLineageCount: %d want 0", rep.ContextLineageCount)
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if n, ok := parsed["context_chars"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json context_chars: %v want 0\n%s", parsed["context_chars"], js)
	}
	if n, ok := parsed["context_lineage_count"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json context_lineage_count: %v want 0\n%s", parsed["context_lineage_count"], js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "context_chars: 0") {
		t.Fatalf("text report missing context_chars 0:\n%s", text)
	}
	if !strings.Contains(text, "context_lineage_count: 0") {
		t.Fatalf("text report missing context_lineage_count 0:\n%s", text)
	}
}

func TestDogfood_UserAgentEvidence(t *testing.T) {
	prev := UserAgent()
	SetUserAgent("iomesh-tui/test-s290")
	t.Cleanup(func() { SetUserAgent(prev) })

	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{})
	t.Cleanup(srv.Close)

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "stage",
		EmitDeptStreams: true,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{})
	if rep.UserAgent != "iomesh-tui/test-s290" {
		t.Fatalf("report UserAgent: %q", rep.UserAgent)
	}
	js := FormatReportJSON(rep)
	if !strings.Contains(js, `"user_agent": "iomesh-tui/test-s290"`) &&
		!strings.Contains(js, `"user_agent":"iomesh-tui/test-s290"`) {
		t.Fatalf("FormatReportJSON missing user_agent:\n%s", js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "user_agent: iomesh-tui/test-s290") {
		t.Fatalf("FormatReport missing user_agent:\n%s", text)
	}
	sl := c.StatusLine()
	if !strings.Contains(sl, "ua=iomesh-tui/test-s290") {
		t.Fatalf("StatusLine missing ua=: %s", sl)
	}
}

func TestReady_OK(t *testing.T) {
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{})
	t.Cleanup(srv.Close)
	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	if err := c.Ready(context.Background()); err != nil {
		t.Fatal(err)
	}
}

// stepByName returns the first step with name, or zero Step.
func dogfoodStep(rep DogfoodReport, name string) (Step, bool) {
	for _, s := range rep.Steps {
		if s.Name == name {
			return s, true
		}
	}
	return Step{}, false
}

func TestDogfood_WaitReady_SucceedsAfterRetries(t *testing.T) {
	// Ready fails twice then OK within budget → wait_ready PASS; single-shot ready still runs.
	var readyHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case "/ready", "/readyz":
			n := readyHits.Add(1)
			if n < 3 {
				w.WriteHeader(503)
				return
			}
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ready"))
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "stage"}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{
		SkipContext:       true,
		SkipEmit:          true,
		SkipMemory:        true,
		WaitReady:         2 * time.Second,
		WaitReadyInterval: 10 * time.Millisecond,
	})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if rep.WaitReadyMS != 2000 {
		t.Fatalf("WaitReadyMS: %d want 2000", rep.WaitReadyMS)
	}
	wr, ok := dogfoodStep(rep, "wait_ready")
	if !ok || wr.Status != StepPass {
		t.Fatalf("wait_ready: ok=%v status=%s detail=%s", ok, wr.Status, wr.Detail)
	}
	if !strings.Contains(wr.Detail, "WaitReady OK") {
		t.Fatalf("wait_ready detail: %s", wr.Detail)
	}
	ready, ok := dogfoodStep(rep, "ready")
	if !ok || ready.Status != StepPass {
		t.Fatalf("ready after wait: ok=%v status=%s detail=%s", ok, ready.Status, ready.Detail)
	}
	if readyHits.Load() < 3 {
		t.Fatalf("expected ≥3 ready hits (wait retries + one-shot), got %d", readyHits.Load())
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if n, ok := parsed["wait_ready_ms"].(float64); !ok || int(n) != 2000 {
		t.Fatalf("json wait_ready_ms: %v want 2000\n%s", parsed["wait_ready_ms"], js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "wait_ready_ms: 2000") {
		t.Fatalf("text report missing wait_ready_ms:\n%s", text)
	}
}

func TestDogfood_WaitReady_TimeoutSoftSkip(t *testing.T) {
	// Always-fail ready within short budget → soft SKIP (not Strict).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case "/ready", "/readyz":
			w.WriteHeader(503)
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{
		Strict:            false,
		SkipContext:       true,
		SkipEmit:          true,
		SkipMemory:        true,
		WaitReady:         80 * time.Millisecond,
		WaitReadyInterval: 15 * time.Millisecond,
	})
	if !rep.OK {
		t.Fatalf("soft wait timeout should not FAIL report: %s\n%s", rep.Summary, FormatReport(rep))
	}
	if rep.WaitReadyMS != 80 {
		t.Fatalf("WaitReadyMS: %d want 80", rep.WaitReadyMS)
	}
	wr, ok := dogfoodStep(rep, "wait_ready")
	if !ok || wr.Status != StepSkip {
		t.Fatalf("wait_ready soft: ok=%v status=%s detail=%s", ok, wr.Status, wr.Detail)
	}
	if !strings.Contains(wr.Detail, "wait_ready soft-fail") {
		t.Fatalf("wait_ready detail: %s", wr.Detail)
	}
	// Single-shot ready also soft-fails (503).
	ready, ok := dogfoodStep(rep, "ready")
	if !ok || ready.Status != StepSkip {
		t.Fatalf("ready soft: ok=%v status=%s", ok, ready.Status)
	}
}

func TestDogfood_WaitReady_TimeoutStrictFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case "/ready", "/readyz":
			w.WriteHeader(503)
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{
		Strict:            true,
		SkipContext:       true,
		SkipEmit:          true,
		SkipMemory:        true,
		WaitReady:         80 * time.Millisecond,
		WaitReadyInterval: 15 * time.Millisecond,
	})
	if rep.OK {
		t.Fatalf("strict wait timeout should FAIL: %s\n%s", rep.Summary, FormatReport(rep))
	}
	wr, ok := dogfoodStep(rep, "wait_ready")
	if !ok || wr.Status != StepFail {
		t.Fatalf("wait_ready strict: ok=%v status=%s detail=%s", ok, wr.Status, wr.Detail)
	}
	if !strings.Contains(wr.Detail, "wait_ready:") {
		t.Fatalf("wait_ready detail: %s", wr.Detail)
	}
}

func TestDogfood_WaitReady_DefaultOff(t *testing.T) {
	// Zero WaitReady: no wait_ready step; wait_ready_ms always 0 in report.
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{})
	t.Cleanup(srv.Close)

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "stage",
		EmitDeptStreams: false,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{SkipMemory: true})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if rep.WaitReadyMS != 0 {
		t.Fatalf("WaitReadyMS: %d want 0", rep.WaitReadyMS)
	}
	if _, ok := dogfoodStep(rep, "wait_ready"); ok {
		t.Fatal("default dogfood must not include wait_ready step")
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if n, ok := parsed["wait_ready_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json wait_ready_ms: %v want 0\n%s", parsed["wait_ready_ms"], js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "wait_ready_ms: 0") {
		t.Fatalf("text report missing wait_ready_ms 0:\n%s", text)
	}
}

func TestDogfood_KV_UnsetSkip(t *testing.T) {
	// Empty KVBucket → kv SKIP "kv probe unset"; kv_key_count=0; kv_bucket omitted.
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{})
	t.Cleanup(srv.Close)

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "stage",
		EmitDeptStreams: false,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{SkipMemory: true, SkipStreams: true})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if rep.KVBucket != "" {
		t.Fatalf("KVBucket=%q want empty", rep.KVBucket)
	}
	if rep.KVKeyCount != 0 {
		t.Fatalf("KVKeyCount: %d want 0", rep.KVKeyCount)
	}
	kv, ok := dogfoodStep(rep, "kv")
	if !ok || kv.Status != StepSkip {
		t.Fatalf("kv step: ok=%v status=%s detail=%s", ok, kv.Status, kv.Detail)
	}
	if !strings.Contains(kv.Detail, "kv probe unset") {
		t.Fatalf("kv detail: %s", kv.Detail)
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if _, has := parsed["kv_bucket"]; has {
		t.Fatalf("json must omit kv_bucket when unset:\n%s", js)
	}
	if n, ok := parsed["kv_key_count"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json kv_key_count: %v want 0\n%s", parsed["kv_key_count"], js)
	}
	if ensured, ok := parsed["kv_ensured"].(bool); !ok || ensured {
		t.Fatalf("json kv_ensured: %v want false\n%s", parsed["kv_ensured"], js)
	}
	if rep.KVEnsured {
		t.Fatal("KVEnsured want false when unset")
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "kv_key_count: 0") {
		t.Fatalf("text report missing kv_key_count:\n%s", text)
	}
	if !strings.Contains(text, "kv_ensured: false") {
		t.Fatalf("text report missing kv_ensured:\n%s", text)
	}
	if strings.Contains(text, "kv_bucket:") {
		t.Fatalf("text report must omit kv_bucket when unset:\n%s", text)
	}
}

func TestDogfood_KV_ListKeysPass(t *testing.T) {
	// KVBucket set → soft list-keys PASS; top-level kv_bucket + kv_key_count.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case r.URL.Path == "/ready" || r.URL.Path == "/readyz":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ready"))
		case r.URL.Path == "/v1/kv/config" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"keys": []string{"app.json", "app.toml", "flags"},
			})
		default:
			if strings.Contains(r.URL.Path, "catalog") {
				_ = json.NewEncoder(w).Encode(map[string]any{"products": []any{}})
				return
			}
			if r.URL.Path == "/v1/streams" {
				_ = json.NewEncoder(w).Encode([]any{})
				return
			}
			if r.URL.Path == "/v1/context/query" {
				_ = json.NewEncoder(w).Encode(map[string]string{"text": "ctx"})
				return
			}
			if r.URL.Path == "/v1/streams/dept/publish" {
				w.WriteHeader(204)
				return
			}
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "stage",
		EmitDeptStreams: true,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{
		SkipMemory: true,
		KVBucket:   "config",
	})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if rep.KVBucket != "config" {
		t.Fatalf("KVBucket=%q want config", rep.KVBucket)
	}
	if rep.KVKeyCount != 3 {
		t.Fatalf("KVKeyCount: %d want 3", rep.KVKeyCount)
	}
	kv, ok := dogfoodStep(rep, "kv")
	if !ok || kv.Status != StepPass {
		t.Fatalf("kv step: ok=%v status=%s detail=%s", ok, kv.Status, kv.Detail)
	}
	if !strings.Contains(kv.Detail, "bucket=config") || !strings.Contains(kv.Detail, "n=3") {
		t.Fatalf("kv detail: %s", kv.Detail)
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if parsed["kv_bucket"] != "config" {
		t.Fatalf("json kv_bucket: %v\n%s", parsed["kv_bucket"], js)
	}
	if n, ok := parsed["kv_key_count"].(float64); !ok || int(n) != 3 {
		t.Fatalf("json kv_key_count: %v want 3\n%s", parsed["kv_key_count"], js)
	}
	if ensured, ok := parsed["kv_ensured"].(bool); !ok || ensured {
		t.Fatalf("json kv_ensured: %v want false (ensure not requested)\n%s", parsed["kv_ensured"], js)
	}
	if rep.KVEnsured {
		t.Fatal("KVEnsured want false without KVEnsure")
	}
	if !strings.Contains(kv.Detail, "ensure=skip") {
		t.Fatalf("kv detail want ensure=skip: %s", kv.Detail)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "kv_bucket: config") || !strings.Contains(text, "kv_key_count: 3") {
		t.Fatalf("text report missing kv evidence:\n%s", text)
	}
	if !strings.Contains(text, "kv_ensured: false") {
		t.Fatalf("text report missing kv_ensured:\n%s", text)
	}
}

func TestDogfood_KV_SoftFail(t *testing.T) {
	// ListKeys 500 → soft SKIP, kv_key_count=0, kv_bucket still set.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case r.URL.Path == "/ready" || r.URL.Path == "/readyz":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ready"))
		case strings.HasPrefix(r.URL.Path, "/v1/kv/") && r.Method == http.MethodGet:
			w.WriteHeader(500)
		default:
			if r.URL.Path == "/v1/context/query" {
				_ = json.NewEncoder(w).Encode(map[string]string{"text": "ctx"})
				return
			}
			if r.URL.Path == "/v1/streams/dept/publish" {
				w.WriteHeader(204)
				return
			}
			if strings.Contains(r.URL.Path, "catalog") {
				_ = json.NewEncoder(w).Encode(map[string]any{"products": []any{}})
				return
			}
			if r.URL.Path == "/v1/streams" {
				_ = json.NewEncoder(w).Encode([]any{})
				return
			}
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "stage",
		EmitDeptStreams: true,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{
		SkipMemory: true,
		KVBucket:   "config",
		Strict:     false,
	})
	if !rep.OK {
		t.Fatalf("soft fail should not fail report: %s\n%s", rep.Summary, FormatReport(rep))
	}
	if rep.KVBucket != "config" {
		t.Fatalf("KVBucket=%q", rep.KVBucket)
	}
	if rep.KVKeyCount != 0 {
		t.Fatalf("KVKeyCount: %d want 0", rep.KVKeyCount)
	}
	kv, ok := dogfoodStep(rep, "kv")
	if !ok || kv.Status != StepSkip {
		t.Fatalf("kv soft-fail: ok=%v status=%s detail=%s", ok, kv.Status, kv.Detail)
	}
	if !strings.Contains(kv.Detail, "kv soft-fail") {
		t.Fatalf("kv detail: %s", kv.Detail)
	}
}

func TestDogfood_KV_StrictFail(t *testing.T) {
	// ListKeys 500 + Strict → FAIL report.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case r.URL.Path == "/ready" || r.URL.Path == "/readyz":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ready"))
		case strings.HasPrefix(r.URL.Path, "/v1/kv/") && r.Method == http.MethodGet:
			w.WriteHeader(500)
		default:
			if r.URL.Path == "/v1/context/query" {
				_ = json.NewEncoder(w).Encode(map[string]string{"text": "ctx"})
				return
			}
			if r.URL.Path == "/v1/streams/dept/publish" {
				w.WriteHeader(204)
				return
			}
			if strings.Contains(r.URL.Path, "catalog") {
				_ = json.NewEncoder(w).Encode(map[string]any{"products": []any{}})
				return
			}
			if r.URL.Path == "/v1/streams" {
				_ = json.NewEncoder(w).Encode([]any{})
				return
			}
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "stage",
		EmitDeptStreams: true,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{
		SkipMemory: true,
		KVBucket:   "config",
		Strict:     true,
	})
	if rep.OK {
		t.Fatal("expected FAIL report under --strict")
	}
	kv, ok := dogfoodStep(rep, "kv")
	if !ok || kv.Status != StepFail {
		t.Fatalf("kv strict: ok=%v status=%s detail=%s", ok, kv.Status, kv.Detail)
	}
	if rep.KVKeyCount != 0 {
		t.Fatalf("KVKeyCount: %d want 0", rep.KVKeyCount)
	}
}

func TestDogfood_KV_EnsureOk(t *testing.T) {
	// KVEnsure + KVBucket → POST create then GET list; kv_ensured=true; detail ensure=ok.
	var sawCreate, sawList bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case r.URL.Path == "/ready" || r.URL.Path == "/readyz":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ready"))
		case r.URL.Path == "/v1/kv/config" && r.Method == http.MethodPost:
			sawCreate = true
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"name": "config"})
		case r.URL.Path == "/v1/kv/config" && r.Method == http.MethodGet:
			sawList = true
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []string{"a"}})
		default:
			if strings.Contains(r.URL.Path, "catalog") {
				_ = json.NewEncoder(w).Encode(map[string]any{"products": []any{}})
				return
			}
			if r.URL.Path == "/v1/streams" {
				_ = json.NewEncoder(w).Encode([]any{})
				return
			}
			if r.URL.Path == "/v1/context/query" {
				_ = json.NewEncoder(w).Encode(map[string]string{"text": "ctx"})
				return
			}
			if r.URL.Path == "/v1/streams/dept/publish" {
				w.WriteHeader(204)
				return
			}
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "stage",
		EmitDeptStreams: true,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{
		SkipMemory: true,
		KVBucket:   "config",
		KVEnsure:   true,
	})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if !sawCreate || !sawList {
		t.Fatalf("sawCreate=%v sawList=%v", sawCreate, sawList)
	}
	if !rep.KVEnsured {
		t.Fatal("KVEnsured want true")
	}
	if rep.KVKeyCount != 1 {
		t.Fatalf("KVKeyCount=%d", rep.KVKeyCount)
	}
	kv, ok := dogfoodStep(rep, "kv")
	if !ok || kv.Status != StepPass {
		t.Fatalf("kv step: ok=%v status=%s detail=%s", ok, kv.Status, kv.Detail)
	}
	if !strings.Contains(kv.Detail, "ensure=ok") {
		t.Fatalf("kv detail: %s", kv.Detail)
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if ensured, ok := parsed["kv_ensured"].(bool); !ok || !ensured {
		t.Fatalf("json kv_ensured: %v\n%s", parsed["kv_ensured"], js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "kv_ensured: true") {
		t.Fatalf("text missing kv_ensured true:\n%s", text)
	}
}

func TestDogfood_KV_Ensure409Idempotent(t *testing.T) {
	// Create 409 Conflict is success → kv_ensured=true.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case r.URL.Path == "/ready" || r.URL.Path == "/readyz":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ready"))
		case r.URL.Path == "/v1/kv/config" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusConflict)
		case r.URL.Path == "/v1/kv/config" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []string{}})
		default:
			if strings.Contains(r.URL.Path, "catalog") {
				_ = json.NewEncoder(w).Encode(map[string]any{"products": []any{}})
				return
			}
			if r.URL.Path == "/v1/streams" {
				_ = json.NewEncoder(w).Encode([]any{})
				return
			}
			if r.URL.Path == "/v1/context/query" {
				_ = json.NewEncoder(w).Encode(map[string]string{"text": "ctx"})
				return
			}
			if r.URL.Path == "/v1/streams/dept/publish" {
				w.WriteHeader(204)
				return
			}
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL, EmitDeptStreams: true}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{
		SkipMemory: true, KVBucket: "config", KVEnsure: true,
	})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if !rep.KVEnsured {
		t.Fatal("409 create should set KVEnsured")
	}
	kv, _ := dogfoodStep(rep, "kv")
	if kv.Status != StepPass || !strings.Contains(kv.Detail, "ensure=ok") {
		t.Fatalf("detail=%s status=%s", kv.Detail, kv.Status)
	}
}

func TestDogfood_KV_EnsureSoftFailStillLists(t *testing.T) {
	// Create 500 → ensure=soft-fail, kv_ensured=false, but list still runs and can PASS.
	var sawList bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case r.URL.Path == "/ready" || r.URL.Path == "/readyz":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ready"))
		case r.URL.Path == "/v1/kv/config" && r.Method == http.MethodPost:
			w.WriteHeader(500)
		case r.URL.Path == "/v1/kv/config" && r.Method == http.MethodGet:
			sawList = true
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []string{"k1", "k2"}})
		default:
			if strings.Contains(r.URL.Path, "catalog") {
				_ = json.NewEncoder(w).Encode(map[string]any{"products": []any{}})
				return
			}
			if r.URL.Path == "/v1/streams" {
				_ = json.NewEncoder(w).Encode([]any{})
				return
			}
			if r.URL.Path == "/v1/context/query" {
				_ = json.NewEncoder(w).Encode(map[string]string{"text": "ctx"})
				return
			}
			if r.URL.Path == "/v1/streams/dept/publish" {
				w.WriteHeader(204)
				return
			}
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL, EmitDeptStreams: true}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{
		SkipMemory: true, KVBucket: "config", KVEnsure: true, Strict: true,
	})
	if !rep.OK {
		// list succeeded; ensure soft-fail must not fail report even under strict
		t.Fatalf("ensure soft-fail should not fail report: %s\n%s", rep.Summary, FormatReport(rep))
	}
	if !sawList {
		t.Fatal("list should still run after ensure soft-fail")
	}
	if rep.KVEnsured {
		t.Fatal("KVEnsured want false on create error")
	}
	if rep.KVKeyCount != 2 {
		t.Fatalf("KVKeyCount=%d", rep.KVKeyCount)
	}
	kv, ok := dogfoodStep(rep, "kv")
	if !ok || kv.Status != StepPass {
		t.Fatalf("kv: ok=%v status=%s detail=%s", ok, kv.Status, kv.Detail)
	}
	if !strings.Contains(kv.Detail, "ensure=soft-fail") {
		t.Fatalf("detail: %s", kv.Detail)
	}
}

func TestDogfood_Pub_UnsetSkip(t *testing.T) {
	// PubSubject empty → pub step SKIP; pub_probed/pub_ok always false in JSON/text.
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{})
	t.Cleanup(srv.Close)

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "stage",
		EmitDeptStreams: false,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{SkipMemory: true, SkipStreams: true})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if rep.PubProbed || rep.PubOK {
		t.Fatalf("PubProbed=%v PubOK=%v want both false", rep.PubProbed, rep.PubOK)
	}
	pub, ok := dogfoodStep(rep, "pub")
	if !ok || pub.Status != StepSkip {
		t.Fatalf("pub step: ok=%v status=%s detail=%s", ok, pub.Status, pub.Detail)
	}
	if !strings.Contains(pub.Detail, "pub probe unset") {
		t.Fatalf("pub detail: %s", pub.Detail)
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if probed, ok := parsed["pub_probed"].(bool); !ok || probed {
		t.Fatalf("json pub_probed: %v want false\n%s", parsed["pub_probed"], js)
	}
	if pok, ok := parsed["pub_ok"].(bool); !ok || pok {
		t.Fatalf("json pub_ok: %v want false\n%s", parsed["pub_ok"], js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "pub_probed: false") || !strings.Contains(text, "pub_ok: false") {
		t.Fatalf("text report missing pub fields:\n%s", text)
	}
}

func TestDogfood_Pub_Pass(t *testing.T) {
	var gotSubject, gotPayload string
	var pubHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case r.URL.Path == "/ready" || r.URL.Path == "/readyz":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ready"))
		case r.URL.Path == "/v1/pub" && r.Method == http.MethodPost:
			pubHits++
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if s, ok := body["subject"].(string); ok {
				gotSubject = s
			}
			if p, ok := body["payload"].(string); ok {
				gotPayload = p
			}
			w.WriteHeader(204)
		default:
			if strings.Contains(r.URL.Path, "catalog") {
				_ = json.NewEncoder(w).Encode(map[string]any{"products": []any{}})
				return
			}
			if r.URL.Path == "/v1/streams" {
				_ = json.NewEncoder(w).Encode([]any{})
				return
			}
			if r.URL.Path == "/v1/context/query" {
				_ = json.NewEncoder(w).Encode(map[string]string{"text": "ctx"})
				return
			}
			if r.URL.Path == "/v1/streams/dept/publish" {
				w.WriteHeader(204)
				return
			}
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL, EmitDeptStreams: true}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{
		SkipMemory: true,
		PubSubject: "dept.agent.dogfood.ping",
	})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if pubHits != 1 {
		t.Fatalf("pub hits=%d", pubHits)
	}
	if gotSubject != "dept.agent.dogfood.ping" {
		t.Fatalf("subject=%q", gotSubject)
	}
	if gotPayload != `{"source":"iomesh-tui-dogfood"}` {
		t.Fatalf("payload=%q", gotPayload)
	}
	if !rep.PubProbed || !rep.PubOK {
		t.Fatalf("PubProbed=%v PubOK=%v", rep.PubProbed, rep.PubOK)
	}
	pub, ok := dogfoodStep(rep, "pub")
	if !ok || pub.Status != StepPass {
		t.Fatalf("pub: ok=%v status=%s detail=%s", ok, pub.Status, pub.Detail)
	}
	if !strings.Contains(pub.Detail, "subject=dept.agent.dogfood.ping") {
		t.Fatalf("detail: %s", pub.Detail)
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if probed, ok := parsed["pub_probed"].(bool); !ok || !probed {
		t.Fatalf("json pub_probed: %v\n%s", parsed["pub_probed"], js)
	}
	if pok, ok := parsed["pub_ok"].(bool); !ok || !pok {
		t.Fatalf("json pub_ok: %v\n%s", parsed["pub_ok"], js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "pub_probed: true") || !strings.Contains(text, "pub_ok: true") {
		t.Fatalf("text:\n%s", text)
	}
}

func TestDogfood_Pub_SoftFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case r.URL.Path == "/ready" || r.URL.Path == "/readyz":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ready"))
		case r.URL.Path == "/v1/pub":
			w.WriteHeader(500)
		default:
			if strings.Contains(r.URL.Path, "catalog") {
				_ = json.NewEncoder(w).Encode(map[string]any{"products": []any{}})
				return
			}
			if r.URL.Path == "/v1/streams" {
				_ = json.NewEncoder(w).Encode([]any{})
				return
			}
			if r.URL.Path == "/v1/context/query" {
				_ = json.NewEncoder(w).Encode(map[string]string{"text": "ctx"})
				return
			}
			if r.URL.Path == "/v1/streams/dept/publish" {
				w.WriteHeader(204)
				return
			}
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL, EmitDeptStreams: true}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{
		SkipMemory: true,
		PubSubject: "events.demo",
	})
	if !rep.OK {
		t.Fatalf("soft fail should still OK: %s\n%s", rep.Summary, FormatReport(rep))
	}
	if !rep.PubProbed || rep.PubOK {
		t.Fatalf("PubProbed=%v PubOK=%v", rep.PubProbed, rep.PubOK)
	}
	pub, ok := dogfoodStep(rep, "pub")
	if !ok || pub.Status != StepSkip {
		t.Fatalf("pub: ok=%v status=%s detail=%s", ok, pub.Status, pub.Detail)
	}
	if !strings.Contains(pub.Detail, "pub soft-fail") {
		t.Fatalf("detail: %s", pub.Detail)
	}
}

func TestDogfood_Pub_StrictFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case r.URL.Path == "/ready" || r.URL.Path == "/readyz":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ready"))
		case r.URL.Path == "/v1/pub":
			w.WriteHeader(503)
		default:
			if strings.Contains(r.URL.Path, "catalog") {
				_ = json.NewEncoder(w).Encode(map[string]any{"products": []any{}})
				return
			}
			if r.URL.Path == "/v1/streams" {
				_ = json.NewEncoder(w).Encode([]any{})
				return
			}
			if r.URL.Path == "/v1/context/query" {
				_ = json.NewEncoder(w).Encode(map[string]string{"text": "ctx"})
				return
			}
			if r.URL.Path == "/v1/streams/dept/publish" {
				w.WriteHeader(204)
				return
			}
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL, EmitDeptStreams: true}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{
		SkipMemory: true,
		PubSubject: "events.demo",
		Strict:     true,
	})
	if rep.OK {
		t.Fatal("strict pub fail should fail report")
	}
	if !rep.PubProbed || rep.PubOK {
		t.Fatalf("PubProbed=%v PubOK=%v", rep.PubProbed, rep.PubOK)
	}
	pub, ok := dogfoodStep(rep, "pub")
	if !ok || pub.Status != StepFail {
		t.Fatalf("pub: ok=%v status=%s detail=%s", ok, pub.Status, pub.Detail)
	}
}

func TestDogfood_Consumer_UnsetSkip(t *testing.T) {
	// ConsumerStream+Name empty → consumer SKIP; booleans always false in JSON/text.
	srv := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{})
	t.Cleanup(srv.Close)

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "stage",
		EmitDeptStreams: false,
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{SkipMemory: true, SkipStreams: true})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if rep.ConsumerProbed || rep.ConsumerOK || rep.ConsumerFetchOK {
		t.Fatalf("ConsumerProbed=%v ConsumerOK=%v ConsumerFetchOK=%v want all false",
			rep.ConsumerProbed, rep.ConsumerOK, rep.ConsumerFetchOK)
	}
	step, ok := dogfoodStep(rep, "consumer")
	if !ok || step.Status != StepSkip {
		t.Fatalf("consumer step: ok=%v status=%s detail=%s", ok, step.Status, step.Detail)
	}
	if !strings.Contains(step.Detail, "consumer probe unset") {
		t.Fatalf("consumer detail: %s", step.Detail)
	}
	// Partial args → needs stream and name
	rep2 := c.Dogfood(context.Background(), DogfoodOptions{
		SkipMemory: true, SkipStreams: true,
		ConsumerStream: "EVENTS",
	})
	step2, ok2 := dogfoodStep(rep2, "consumer")
	if !ok2 || step2.Status != StepSkip {
		t.Fatalf("partial: ok=%v status=%s detail=%s", ok2, step2.Status, step2.Detail)
	}
	if !strings.Contains(step2.Detail, "consumer probe needs stream and name") {
		t.Fatalf("partial detail: %s", step2.Detail)
	}
	if rep2.ConsumerProbed || rep2.ConsumerOK {
		t.Fatalf("partial probed=%v ok=%v", rep2.ConsumerProbed, rep2.ConsumerOK)
	}

	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	for _, key := range []string{"consumer_probed", "consumer_ok", "consumer_fetch_ok"} {
		if v, ok := parsed[key].(bool); !ok || v {
			t.Fatalf("json %s: %v want false\n%s", key, parsed[key], js)
		}
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "consumer_probed: false") ||
		!strings.Contains(text, "consumer_ok: false") ||
		!strings.Contains(text, "consumer_fetch_ok: false") {
		t.Fatalf("text report missing consumer fields:\n%s", text)
	}
}

func TestDogfood_Consumer_Create201OK(t *testing.T) {
	var createHits int
	var gotName, gotFilter string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case r.URL.Path == "/ready" || r.URL.Path == "/readyz":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ready"))
		case strings.HasSuffix(r.URL.Path, "/consumers") && r.Method == http.MethodPost:
			createHits++
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if n, ok := body["name"].(string); ok {
				gotName = n
			}
			if f, ok := body["filter_subject"].(string); ok {
				gotFilter = f
			}
			w.WriteHeader(201)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"stream": "EVENTS", "name": "worker-1",
				"filter_subject": "dept.events.>", "ack_floor": 0, "pending_count": 0,
			})
		default:
			if strings.Contains(r.URL.Path, "catalog") {
				_ = json.NewEncoder(w).Encode(map[string]any{"products": []any{}})
				return
			}
			if r.URL.Path == "/v1/streams" {
				_ = json.NewEncoder(w).Encode([]any{})
				return
			}
			if r.URL.Path == "/v1/context/query" {
				_ = json.NewEncoder(w).Encode(map[string]string{"text": "ctx"})
				return
			}
			if r.URL.Path == "/v1/streams/dept/publish" {
				w.WriteHeader(204)
				return
			}
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL, EmitDeptStreams: true}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{
		SkipMemory:     true,
		ConsumerStream: "EVENTS",
		ConsumerName:   "worker-1",
		ConsumerFilter: "dept.events.>",
	})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if createHits != 1 {
		t.Fatalf("create hits=%d", createHits)
	}
	if gotName != "worker-1" || gotFilter != "dept.events.>" {
		t.Fatalf("name=%q filter=%q", gotName, gotFilter)
	}
	if !rep.ConsumerProbed || !rep.ConsumerOK || rep.ConsumerFetchOK {
		t.Fatalf("probed=%v ok=%v fetch_ok=%v", rep.ConsumerProbed, rep.ConsumerOK, rep.ConsumerFetchOK)
	}
	step, ok := dogfoodStep(rep, "consumer")
	if !ok || step.Status != StepPass {
		t.Fatalf("consumer: ok=%v status=%s detail=%s", ok, step.Status, step.Detail)
	}
	if !strings.Contains(step.Detail, "stream=EVENTS") || !strings.Contains(step.Detail, "create=ok") {
		t.Fatalf("detail: %s", step.Detail)
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if v, ok := parsed["consumer_probed"].(bool); !ok || !v {
		t.Fatalf("json consumer_probed: %v\n%s", parsed["consumer_probed"], js)
	}
	if v, ok := parsed["consumer_ok"].(bool); !ok || !v {
		t.Fatalf("json consumer_ok: %v\n%s", parsed["consumer_ok"], js)
	}
	if v, ok := parsed["consumer_fetch_ok"].(bool); !ok || v {
		t.Fatalf("json consumer_fetch_ok: %v want false\n%s", parsed["consumer_fetch_ok"], js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "consumer_probed: true") || !strings.Contains(text, "consumer_ok: true") {
		t.Fatalf("text:\n%s", text)
	}
}

func TestDogfood_Consumer_Create409OK(t *testing.T) {
	var createHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case r.URL.Path == "/ready" || r.URL.Path == "/readyz":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ready"))
		case strings.HasSuffix(r.URL.Path, "/consumers") && r.Method == http.MethodPost:
			createHits++
			w.WriteHeader(409)
			_, _ = w.Write([]byte(`{"error":"already exists"}`))
		default:
			if strings.Contains(r.URL.Path, "catalog") {
				_ = json.NewEncoder(w).Encode(map[string]any{"products": []any{}})
				return
			}
			if r.URL.Path == "/v1/streams" {
				_ = json.NewEncoder(w).Encode([]any{})
				return
			}
			if r.URL.Path == "/v1/context/query" {
				_ = json.NewEncoder(w).Encode(map[string]string{"text": "ctx"})
				return
			}
			if r.URL.Path == "/v1/streams/dept/publish" {
				w.WriteHeader(204)
				return
			}
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL, EmitDeptStreams: false}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{
		SkipMemory: true, SkipStreams: true,
		ConsumerStream: "EVENTS",
		ConsumerName:   "worker-1",
	})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if createHits != 1 {
		t.Fatalf("create hits=%d", createHits)
	}
	if !rep.ConsumerProbed || !rep.ConsumerOK {
		t.Fatalf("probed=%v ok=%v (409 should count as success)", rep.ConsumerProbed, rep.ConsumerOK)
	}
	step, ok := dogfoodStep(rep, "consumer")
	if !ok || step.Status != StepPass {
		t.Fatalf("consumer: ok=%v status=%s detail=%s", ok, step.Status, step.Detail)
	}
}

func TestDogfood_Consumer_CreateSoftFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case r.URL.Path == "/ready" || r.URL.Path == "/readyz":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ready"))
		case strings.HasSuffix(r.URL.Path, "/consumers") && r.Method == http.MethodPost:
			w.WriteHeader(500)
		default:
			if strings.Contains(r.URL.Path, "catalog") {
				_ = json.NewEncoder(w).Encode(map[string]any{"products": []any{}})
				return
			}
			if r.URL.Path == "/v1/streams" {
				_ = json.NewEncoder(w).Encode([]any{})
				return
			}
			if r.URL.Path == "/v1/context/query" {
				_ = json.NewEncoder(w).Encode(map[string]string{"text": "ctx"})
				return
			}
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL, EmitDeptStreams: false}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{
		SkipMemory: true, SkipStreams: true,
		ConsumerStream: "EVENTS",
		ConsumerName:   "worker-1",
	})
	if !rep.OK {
		t.Fatalf("soft fail should still OK: %s\n%s", rep.Summary, FormatReport(rep))
	}
	if !rep.ConsumerProbed || rep.ConsumerOK || rep.ConsumerFetchOK {
		t.Fatalf("probed=%v ok=%v fetch_ok=%v", rep.ConsumerProbed, rep.ConsumerOK, rep.ConsumerFetchOK)
	}
	step, ok := dogfoodStep(rep, "consumer")
	if !ok || step.Status != StepSkip {
		t.Fatalf("consumer: ok=%v status=%s detail=%s", ok, step.Status, step.Detail)
	}
	if !strings.Contains(step.Detail, "consumer soft-fail") {
		t.Fatalf("detail: %s", step.Detail)
	}
}

func TestDogfood_Consumer_FetchEmptyOK(t *testing.T) {
	var createHits, fetchHits int
	var fetchMaxWait int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case r.URL.Path == "/ready" || r.URL.Path == "/readyz":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ready"))
		case strings.HasSuffix(r.URL.Path, "/fetch") && r.Method == http.MethodPost:
			fetchHits++
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if n, ok := body["max_wait_ms"].(float64); ok {
				fetchMaxWait = int(n)
			}
			w.WriteHeader(200)
			_ = json.NewEncoder(w).Encode(map[string]any{"messages": []any{}})
		case strings.HasSuffix(r.URL.Path, "/consumers") && r.Method == http.MethodPost:
			createHits++
			w.WriteHeader(201)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"stream": "EVENTS", "name": "worker-1",
			})
		default:
			if strings.Contains(r.URL.Path, "catalog") {
				_ = json.NewEncoder(w).Encode(map[string]any{"products": []any{}})
				return
			}
			if r.URL.Path == "/v1/streams" {
				_ = json.NewEncoder(w).Encode([]any{})
				return
			}
			if r.URL.Path == "/v1/context/query" {
				_ = json.NewEncoder(w).Encode(map[string]string{"text": "ctx"})
				return
			}
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL, EmitDeptStreams: false}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{
		SkipMemory: true, SkipStreams: true,
		ConsumerStream: "EVENTS",
		ConsumerName:   "worker-1",
		ConsumerFetch:  true,
	})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if createHits != 1 || fetchHits != 1 {
		t.Fatalf("create=%d fetch=%d", createHits, fetchHits)
	}
	if fetchMaxWait != 500 {
		t.Fatalf("max_wait_ms=%d want 500", fetchMaxWait)
	}
	if !rep.ConsumerProbed || !rep.ConsumerOK || !rep.ConsumerFetchOK {
		t.Fatalf("probed=%v ok=%v fetch_ok=%v", rep.ConsumerProbed, rep.ConsumerOK, rep.ConsumerFetchOK)
	}
	step, ok := dogfoodStep(rep, "consumer")
	if !ok || step.Status != StepPass {
		t.Fatalf("consumer: ok=%v status=%s detail=%s", ok, step.Status, step.Detail)
	}
	if !strings.Contains(step.Detail, "fetch=n=0") {
		t.Fatalf("detail: %s", step.Detail)
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if v, ok := parsed["consumer_fetch_ok"].(bool); !ok || !v {
		t.Fatalf("json consumer_fetch_ok: %v\n%s", parsed["consumer_fetch_ok"], js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "consumer_fetch_ok: true") {
		t.Fatalf("text:\n%s", text)
	}
}
