package iomesh

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
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
		case r.URL.Path == "/v1/streams/dept" && r.Method == http.MethodPost:
			emits.Add(1)
			if opts.failEmit {
				w.WriteHeader(500)
				return
			}
			b, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(b), "dept.agent.dogfood") {
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
	var memOK, recallOK bool
	for _, s := range rep.Steps {
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
	}
	if !memOK || !recallOK {
		t.Fatal(FormatReport(rep))
	}
	js := FormatReportJSON(rep)
	if !strings.Contains(js, `"result": "PASS"`) || !strings.Contains(js, `"ok": true`) {
		t.Fatal(js)
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
	var foundIngest, foundRecall bool
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
	}
	if !foundIngest || !foundRecall {
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
