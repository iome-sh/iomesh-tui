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
		ContextPlane: true, CatalogPlane: true, EmitDeptStreams: true,
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
	// Top-level wait_ready_elapsed_ms / health_ms / ready_ms / context_ms /
	// streams_ms / catalog_ms / emit_ms / llm_meter_ms / pub_ms / policy_ms /
	// consumer_ms / kv_ms / kv_ensure_ms / kv_list_ms / memory_*_ms / duration_ms always
	// present (>= 0; often 0 on fast mock; wait_ready_elapsed_ms 0 when
	// preflight off; kv_ensure_ms / kv_list_ms 0 when ensure off / probe unset).
	if rep.WaitReadyElapsedMS < 0 {
		t.Fatalf("WaitReadyElapsedMS: %d want >= 0", rep.WaitReadyElapsedMS)
	}
	if rep.HealthMS < 0 {
		t.Fatalf("HealthMS: %d want >= 0", rep.HealthMS)
	}
	if rep.ReadyMS < 0 {
		t.Fatalf("ReadyMS: %d want >= 0", rep.ReadyMS)
	}
	if rep.ContextMS < 0 {
		t.Fatalf("ContextMS: %d want >= 0", rep.ContextMS)
	}
	if rep.StreamsMS < 0 {
		t.Fatalf("StreamsMS: %d want >= 0", rep.StreamsMS)
	}
	if rep.CatalogMS < 0 {
		t.Fatalf("CatalogMS: %d want >= 0", rep.CatalogMS)
	}
	if rep.EmitMS < 0 {
		t.Fatalf("EmitMS: %d want >= 0", rep.EmitMS)
	}
	if rep.LLMMeterMS < 0 {
		t.Fatalf("LLMMeterMS: %d want >= 0", rep.LLMMeterMS)
	}
	if rep.PubMS < 0 {
		t.Fatalf("PubMS: %d want >= 0", rep.PubMS)
	}
	if rep.PolicyMS < 0 {
		t.Fatalf("PolicyMS: %d want >= 0", rep.PolicyMS)
	}
	if rep.ConsumerMS < 0 {
		t.Fatalf("ConsumerMS: %d want >= 0", rep.ConsumerMS)
	}
	if rep.KVMS < 0 {
		t.Fatalf("KVMS: %d want >= 0", rep.KVMS)
	}
	if rep.KVEnsureMS < 0 {
		t.Fatalf("KVEnsureMS: %d want >= 0", rep.KVEnsureMS)
	}
	if rep.KVEnsureMS != 0 {
		// Default full-pass has no --kv-ensure / bucket → ensure not attempted.
		t.Fatalf("KVEnsureMS: %d want 0 when ensure off", rep.KVEnsureMS)
	}
	if rep.KVListMS < 0 {
		t.Fatalf("KVListMS: %d want >= 0", rep.KVListMS)
	}
	if rep.KVListMS != 0 {
		// Default full-pass has no --kv-bucket → list not run.
		t.Fatalf("KVListMS: %d want 0 when kv probe unset", rep.KVListMS)
	}
	if rep.MemoryIngestMS < 0 {
		t.Fatalf("MemoryIngestMS: %d want >= 0", rep.MemoryIngestMS)
	}
	if rep.MemoryRecallMS < 0 {
		t.Fatalf("MemoryRecallMS: %d want >= 0", rep.MemoryRecallMS)
	}
	if rep.MemoryRetrieveMS < 0 {
		t.Fatalf("MemoryRetrieveMS: %d want >= 0", rep.MemoryRetrieveMS)
	}
	if rep.DurationMS < 0 {
		t.Fatalf("DurationMS: %d want >= 0", rep.DurationMS)
	}
	// Top-level steps_pass / steps_fail / steps_skip always present for CI.
	// Full-pass mock: at least one PASS, zero FAIL; skip count may be > 0
	// (e.g. wait_ready off, pub/kv/consumer unset, policy mode off).
	if rep.StepsPass <= 0 {
		t.Fatalf("StepsPass: %d want > 0\n%s", rep.StepsPass, FormatReport(rep))
	}
	if rep.StepsFail != 0 {
		t.Fatalf("StepsFail: %d want 0\n%s", rep.StepsFail, FormatReport(rep))
	}
	if rep.StepsSkip < 0 {
		t.Fatalf("StepsSkip: %d want >= 0", rep.StepsSkip)
	}
	// Counts must match step statuses.
	var wantPass, wantFail, wantSkip int
	for _, s := range rep.Steps {
		switch s.Status {
		case StepPass:
			wantPass++
		case StepFail:
			wantFail++
		case StepSkip:
			wantSkip++
		}
	}
	if rep.StepsPass != wantPass || rep.StepsFail != wantFail || rep.StepsSkip != wantSkip {
		t.Fatalf("step counts mismatch: report pass=%d fail=%d skip=%d counted pass=%d fail=%d skip=%d",
			rep.StepsPass, rep.StepsFail, rep.StepsSkip, wantPass, wantFail, wantSkip)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if n, ok := parsed["steps_pass"].(float64); !ok || int(n) != rep.StepsPass {
		t.Fatalf("json steps_pass: %v want %d\n%s", parsed["steps_pass"], rep.StepsPass, js)
	}
	if n, ok := parsed["steps_fail"].(float64); !ok || int(n) != rep.StepsFail {
		t.Fatalf("json steps_fail: %v want %d\n%s", parsed["steps_fail"], rep.StepsFail, js)
	}
	if n, ok := parsed["steps_skip"].(float64); !ok || int(n) != rep.StepsSkip {
		t.Fatalf("json steps_skip: %v want %d\n%s", parsed["steps_skip"], rep.StepsSkip, js)
	}
	if _, ok := parsed["wait_ready_elapsed_ms"].(float64); !ok {
		t.Fatalf("json wait_ready_elapsed_ms missing or wrong type: %v\n%s", parsed["wait_ready_elapsed_ms"], js)
	}
	if n, ok := parsed["wait_ready_interval_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json wait_ready_interval_ms: %v want 0 when wait off\n%s", parsed["wait_ready_interval_ms"], js)
	}
	if v, ok := parsed["wait_require_health"].(bool); !ok || v {
		t.Fatalf("json wait_require_health: %v want false when unset\n%s", parsed["wait_require_health"], js)
	}
	if s, ok := parsed["wait_ready_result"].(string); !ok || s != "off" {
		t.Fatalf("json wait_ready_result: %v want off when wait off\n%s", parsed["wait_ready_result"], js)
	}
	if n, ok := parsed["wait_ready_attempts"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json wait_ready_attempts: %v want 0 when wait off\n%s", parsed["wait_ready_attempts"], js)
	}
	if _, ok := parsed["health_ms"].(float64); !ok {
		t.Fatalf("json health_ms missing or wrong type: %v\n%s", parsed["health_ms"], js)
	}
	if _, ok := parsed["ready_ms"].(float64); !ok {
		t.Fatalf("json ready_ms missing or wrong type: %v\n%s", parsed["ready_ms"], js)
	}
	if _, ok := parsed["context_ms"].(float64); !ok {
		t.Fatalf("json context_ms missing or wrong type: %v\n%s", parsed["context_ms"], js)
	}
	if _, ok := parsed["streams_ms"].(float64); !ok {
		t.Fatalf("json streams_ms missing or wrong type: %v\n%s", parsed["streams_ms"], js)
	}
	if _, ok := parsed["catalog_ms"].(float64); !ok {
		t.Fatalf("json catalog_ms missing or wrong type: %v\n%s", parsed["catalog_ms"], js)
	}
	if _, ok := parsed["emit_ms"].(float64); !ok {
		t.Fatalf("json emit_ms missing or wrong type: %v\n%s", parsed["emit_ms"], js)
	}
	if _, ok := parsed["llm_meter_ms"].(float64); !ok {
		t.Fatalf("json llm_meter_ms missing or wrong type: %v\n%s", parsed["llm_meter_ms"], js)
	}
	if _, ok := parsed["pub_ms"].(float64); !ok {
		t.Fatalf("json pub_ms missing or wrong type: %v\n%s", parsed["pub_ms"], js)
	}
	if _, ok := parsed["policy_ms"].(float64); !ok {
		t.Fatalf("json policy_ms missing or wrong type: %v\n%s", parsed["policy_ms"], js)
	}
	if _, ok := parsed["consumer_ms"].(float64); !ok {
		t.Fatalf("json consumer_ms missing or wrong type: %v\n%s", parsed["consumer_ms"], js)
	}
	if _, ok := parsed["kv_ms"].(float64); !ok {
		t.Fatalf("json kv_ms missing or wrong type: %v\n%s", parsed["kv_ms"], js)
	}
	if n, ok := parsed["kv_ensure_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json kv_ensure_ms: %v want 0 when ensure off\n%s", parsed["kv_ensure_ms"], js)
	}
	if n, ok := parsed["kv_list_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json kv_list_ms: %v want 0 when kv probe unset\n%s", parsed["kv_list_ms"], js)
	}
	if _, ok := parsed["memory_ingest_ms"].(float64); !ok {
		t.Fatalf("json memory_ingest_ms missing or wrong type: %v\n%s", parsed["memory_ingest_ms"], js)
	}
	if _, ok := parsed["memory_recall_ms"].(float64); !ok {
		t.Fatalf("json memory_recall_ms missing or wrong type: %v\n%s", parsed["memory_recall_ms"], js)
	}
	if _, ok := parsed["memory_retrieve_ms"].(float64); !ok {
		t.Fatalf("json memory_retrieve_ms missing or wrong type: %v\n%s", parsed["memory_retrieve_ms"], js)
	}
	if _, ok := parsed["duration_ms"].(float64); !ok {
		t.Fatalf("json duration_ms missing or wrong type: %v\n%s", parsed["duration_ms"], js)
	}
	if _, ok := parsed["version"]; !ok {
		t.Fatalf("json version missing\n%s", js)
	}
	if !strings.Contains(out, "wait_ready_elapsed_ms:") {
		t.Fatalf("text report missing wait_ready_elapsed_ms:\n%s", out)
	}
	if !strings.Contains(out, "wait_ready_interval_ms: 0") {
		t.Fatalf("text report missing wait_ready_interval_ms 0:\n%s", out)
	}
	if !strings.Contains(out, "wait_require_health: false") {
		t.Fatalf("text report missing wait_require_health false:\n%s", out)
	}
	if !strings.Contains(out, "wait_ready_result: off") {
		t.Fatalf("text report missing wait_ready_result off:\n%s", out)
	}
	if !strings.Contains(out, "wait_ready_attempts: 0") {
		t.Fatalf("text report missing wait_ready_attempts 0:\n%s", out)
	}
	if !strings.Contains(out, "health_ms:") || !strings.Contains(out, "ready_ms:") {
		t.Fatalf("text report missing health_ms/ready_ms:\n%s", out)
	}
	if !strings.Contains(out, "context_ms:") || !strings.Contains(out, "streams_ms:") || !strings.Contains(out, "catalog_ms:") {
		t.Fatalf("text report missing context_ms/streams_ms/catalog_ms:\n%s", out)
	}
	if !strings.Contains(out, "emit_ms:") || !strings.Contains(out, "policy_ms:") || !strings.Contains(out, "duration_ms:") {
		t.Fatalf("text report missing emit_ms/policy_ms/duration_ms:\n%s", out)
	}
	if !strings.Contains(out, "llm_meter_ms:") || !strings.Contains(out, "pub_ms:") {
		t.Fatalf("text report missing llm_meter_ms/pub_ms:\n%s", out)
	}
	if !strings.Contains(out, "consumer_ms:") || !strings.Contains(out, "kv_ms:") {
		t.Fatalf("text report missing consumer_ms/kv_ms:\n%s", out)
	}
	if !strings.Contains(out, "kv_ensure_ms: 0") {
		t.Fatalf("text report missing kv_ensure_ms: 0:\n%s", out)
	}
	if !strings.Contains(out, "kv_list_ms: 0") {
		t.Fatalf("text report missing kv_list_ms: 0:\n%s", out)
	}
	if !strings.Contains(out, "memory_ingest_ms:") || !strings.Contains(out, "memory_recall_ms:") || !strings.Contains(out, "memory_retrieve_ms:") {
		t.Fatalf("text report missing memory_ingest_ms/memory_recall_ms/memory_retrieve_ms:\n%s", out)
	}
	if !strings.Contains(out, fmt.Sprintf("steps_pass: %d", rep.StepsPass)) {
		t.Fatalf("text report missing steps_pass:\n%s", out)
	}
	if !strings.Contains(out, "steps_fail: 0") {
		t.Fatalf("text report missing steps_fail: 0:\n%s", out)
	}
	if !strings.Contains(out, fmt.Sprintf("steps_skip: %d", rep.StepsSkip)) {
		t.Fatalf("text report missing steps_skip:\n%s", out)
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
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if got, ok := parsed["memory_endpoint"].(string); !ok || got != wantBase {
		t.Fatalf("json memory_endpoint: %v want %q\n%s", parsed["memory_endpoint"], wantBase, js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "memory_endpoint: "+wantBase) {
		t.Fatalf("text missing memory_endpoint value:\n%s", text)
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
	// Fail path: health_err always non-empty (underlying probe error).
	if rep.HealthErr == "" {
		t.Fatalf("HealthErr empty on health FAIL; want non-empty detail")
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	he, ok := parsed["health_err"].(string)
	if !ok || he == "" {
		t.Fatalf("json health_err: %v want non-empty string\n%s", parsed["health_err"], js)
	}
	if he != rep.HealthErr {
		t.Fatalf("json health_err %q != report %q", he, rep.HealthErr)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "health_err: "+rep.HealthErr) {
		t.Fatalf("text missing health_err detail:\n%s", text)
	}
}

// TestDogfood_ExitCodeEvidence: exit_code always emitted; 0 when OK, 1 when not.
func TestDogfood_ExitCodeEvidence(t *testing.T) {
	// 1) Mesh disabled → OK=true → exit_code 0
	cOff := New(Config{Enabled: false}, nil)
	repOff := cOff.Dogfood(context.Background(), DogfoodOptions{})
	if !repOff.OK {
		t.Fatalf("disabled want OK: %+v", repOff)
	}
	if repOff.ExitCode != 0 {
		t.Fatalf("disabled ExitCode: %d want 0", repOff.ExitCode)
	}
	jsOff := FormatReportJSON(repOff)
	var parsedOff map[string]any
	if err := json.Unmarshal([]byte(jsOff), &parsedOff); err != nil {
		t.Fatalf("json disabled: %v\n%s", err, jsOff)
	}
	if n, ok := parsedOff["exit_code"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json disabled exit_code: %v want 0\n%s", parsedOff["exit_code"], jsOff)
	}
	if _, ok := parsedOff["exit_code"]; !ok {
		t.Fatalf("json disabled exit_code missing\n%s", jsOff)
	}
	textOff := FormatReport(repOff)
	if !strings.Contains(textOff, "exit_code: 0") {
		t.Fatalf("text disabled missing exit_code 0:\n%s", textOff)
	}
	// exit_code appears after strict in text report
	strictIdx := strings.Index(textOff, "strict:")
	exitIdx := strings.Index(textOff, "exit_code:")
	if strictIdx < 0 || exitIdx < 0 || exitIdx < strictIdx {
		t.Fatalf("exit_code must appear after strict:\n%s", textOff)
	}

	// 2) Health fail → OK=false → exit_code 1
	srvFail := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{failHealth: true})
	t.Cleanup(srvFail.Close)
	cFail := New(Config{Enabled: true, Endpoint: srvFail.URL, ContextPlane: true, EmitDeptStreams: true}, nil)
	repFail := cFail.Dogfood(context.Background(), DogfoodOptions{})
	if repFail.OK {
		t.Fatal("health fail want !OK")
	}
	if repFail.ExitCode != 1 {
		t.Fatalf("fail ExitCode: %d want 1", repFail.ExitCode)
	}
	jsFail := FormatReportJSON(repFail)
	var parsedFail map[string]any
	if err := json.Unmarshal([]byte(jsFail), &parsedFail); err != nil {
		t.Fatalf("json fail: %v\n%s", err, jsFail)
	}
	if n, ok := parsedFail["exit_code"].(float64); !ok || int(n) != 1 {
		t.Fatalf("json fail exit_code: %v want 1\n%s", parsedFail["exit_code"], jsFail)
	}
	// Compose: ok/strict/result still present
	if v, ok := parsedFail["ok"].(bool); !ok || v {
		t.Fatalf("json fail ok: %v want false\n%s", parsedFail["ok"], jsFail)
	}
	if _, ok := parsedFail["strict"]; !ok {
		t.Fatalf("json fail strict missing\n%s", jsFail)
	}
	if s, ok := parsedFail["result"].(string); !ok || s != "FAIL" {
		t.Fatalf("json fail result: %v want FAIL\n%s", parsedFail["result"], jsFail)
	}
	textFail := FormatReport(repFail)
	if !strings.Contains(textFail, "exit_code: 1") {
		t.Fatalf("text fail missing exit_code 1:\n%s", textFail)
	}
	if !strings.Contains(textFail, "RESULT=FAIL") {
		t.Fatalf("text fail missing RESULT=FAIL:\n%s", textFail)
	}

	// 3) Full PASS → OK=true → exit_code 0
	srvPass := mockMeshServer(t, struct {
		failHealth bool
		noReady    bool
		emptyCtx   bool
		failEmit   bool
		failMemory bool
		noMemory   bool
		failRecall bool
		noRecall   bool
	}{})
	t.Cleanup(srvPass.Close)
	cPass := New(Config{
		Enabled: true, Endpoint: srvPass.URL, Tenant: "stage",
		ContextPlane: true, CatalogPlane: true, EmitDeptStreams: true,
	}, nil)
	repPass := cPass.Dogfood(context.Background(), DogfoodOptions{Strict: true, Workspace: "/ws"})
	if !repPass.OK {
		t.Fatalf("pass want OK: %s\n%s", repPass.Summary, FormatReport(repPass))
	}
	if repPass.ExitCode != 0 {
		t.Fatalf("pass ExitCode: %d want 0", repPass.ExitCode)
	}
	jsPass := FormatReportJSON(repPass)
	var parsedPass map[string]any
	if err := json.Unmarshal([]byte(jsPass), &parsedPass); err != nil {
		t.Fatalf("json pass: %v\n%s", err, jsPass)
	}
	if n, ok := parsedPass["exit_code"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json pass exit_code: %v want 0\n%s", parsedPass["exit_code"], jsPass)
	}
	if v, ok := parsedPass["ok"].(bool); !ok || !v {
		t.Fatalf("json pass ok: %v want true\n%s", parsedPass["ok"], jsPass)
	}
	if s, ok := parsedPass["result"].(string); !ok || s != "PASS" {
		t.Fatalf("json pass result: %v want PASS\n%s", parsedPass["result"], jsPass)
	}
	textPass := FormatReport(repPass)
	if !strings.Contains(textPass, "exit_code: 0") {
		t.Fatalf("text pass missing exit_code 0:\n%s", textPass)
	}

	// dogfoodExitCode helper parity
	if dogfoodExitCode(true) != 0 || dogfoodExitCode(false) != 1 {
		t.Fatalf("dogfoodExitCode(true)=%d dogfoodExitCode(false)=%d", dogfoodExitCode(true), dogfoodExitCode(false))
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
	// wait_ready/health/ready/context/streams/catalog/emit/llm_meter/pub/policy/memory steps
	// absent → top-level latencies always 0.
	// duration_ms still present and >= 0 (wall clock of disabled early return).
	if rep.WaitReadyElapsedMS != 0 {
		t.Fatalf("disabled WaitReadyElapsedMS: %d want 0", rep.WaitReadyElapsedMS)
	}
	if rep.WaitReadyIntervalMS != 0 {
		t.Fatalf("disabled WaitReadyIntervalMS: %d want 0", rep.WaitReadyIntervalMS)
	}
	if rep.WaitRequireHealth {
		t.Fatalf("disabled WaitRequireHealth: true want false")
	}
	if rep.WaitReadyResult != "off" {
		t.Fatalf("disabled WaitReadyResult: %q want off", rep.WaitReadyResult)
	}
	if rep.WaitReadyAttempts != 0 {
		t.Fatalf("disabled WaitReadyAttempts: %d want 0", rep.WaitReadyAttempts)
	}
	if rep.HealthMS != 0 {
		t.Fatalf("disabled HealthMS: %d want 0", rep.HealthMS)
	}
	if rep.ReadyMS != 0 {
		t.Fatalf("disabled ReadyMS: %d want 0", rep.ReadyMS)
	}
	if rep.ContextMS != 0 {
		t.Fatalf("disabled ContextMS: %d want 0", rep.ContextMS)
	}
	if rep.StreamsMS != 0 {
		t.Fatalf("disabled StreamsMS: %d want 0", rep.StreamsMS)
	}
	if rep.CatalogMS != 0 {
		t.Fatalf("disabled CatalogMS: %d want 0", rep.CatalogMS)
	}
	if rep.EmitMS != 0 {
		t.Fatalf("disabled EmitMS: %d want 0", rep.EmitMS)
	}
	if rep.LLMMeterMS != 0 {
		t.Fatalf("disabled LLMMeterMS: %d want 0", rep.LLMMeterMS)
	}
	if rep.PubMS != 0 {
		t.Fatalf("disabled PubMS: %d want 0", rep.PubMS)
	}
	if rep.PolicyMS != 0 {
		t.Fatalf("disabled PolicyMS: %d want 0", rep.PolicyMS)
	}
	if rep.ConsumerMS != 0 {
		t.Fatalf("disabled ConsumerMS: %d want 0", rep.ConsumerMS)
	}
	if rep.KVMS != 0 {
		t.Fatalf("disabled KVMS: %d want 0", rep.KVMS)
	}
	if rep.KVEnsureMS != 0 {
		t.Fatalf("disabled KVEnsureMS: %d want 0", rep.KVEnsureMS)
	}
	if rep.KVListMS != 0 {
		t.Fatalf("disabled KVListMS: %d want 0", rep.KVListMS)
	}
	if rep.MemoryIngestMS != 0 {
		t.Fatalf("disabled MemoryIngestMS: %d want 0", rep.MemoryIngestMS)
	}
	if rep.MemoryRecallMS != 0 {
		t.Fatalf("disabled MemoryRecallMS: %d want 0", rep.MemoryRecallMS)
	}
	if rep.MemoryRetrieveMS != 0 {
		t.Fatalf("disabled MemoryRetrieveMS: %d want 0", rep.MemoryRetrieveMS)
	}
	if rep.DurationMS < 0 {
		t.Fatalf("disabled DurationMS: %d want >= 0", rep.DurationMS)
	}
	// Mesh disabled: single SKIP enabled step → steps_skip >= 1, fail=0, pass=0.
	if rep.StepsPass != 0 {
		t.Fatalf("disabled StepsPass: %d want 0", rep.StepsPass)
	}
	if rep.StepsFail != 0 {
		t.Fatalf("disabled StepsFail: %d want 0", rep.StepsFail)
	}
	if rep.StepsSkip < 1 {
		t.Fatalf("disabled StepsSkip: %d want >= 1", rep.StepsSkip)
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if n, ok := parsed["steps_pass"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json steps_pass: %v want 0\n%s", parsed["steps_pass"], js)
	}
	if n, ok := parsed["steps_fail"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json steps_fail: %v want 0\n%s", parsed["steps_fail"], js)
	}
	if n, ok := parsed["steps_skip"].(float64); !ok || int(n) < 1 {
		t.Fatalf("json steps_skip: %v want >= 1\n%s", parsed["steps_skip"], js)
	}
	if n, ok := parsed["wait_ready_elapsed_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json wait_ready_elapsed_ms: %v want 0\n%s", parsed["wait_ready_elapsed_ms"], js)
	}
	if n, ok := parsed["wait_ready_interval_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json wait_ready_interval_ms: %v want 0\n%s", parsed["wait_ready_interval_ms"], js)
	}
	if v, ok := parsed["wait_require_health"].(bool); !ok || v {
		t.Fatalf("json wait_require_health: %v want false\n%s", parsed["wait_require_health"], js)
	}
	if s, ok := parsed["wait_ready_result"].(string); !ok || s != "off" {
		t.Fatalf("json wait_ready_result: %v want off\n%s", parsed["wait_ready_result"], js)
	}
	if n, ok := parsed["wait_ready_attempts"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json wait_ready_attempts: %v want 0\n%s", parsed["wait_ready_attempts"], js)
	}
	if n, ok := parsed["health_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json health_ms: %v want 0\n%s", parsed["health_ms"], js)
	}
	if n, ok := parsed["ready_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json ready_ms: %v want 0\n%s", parsed["ready_ms"], js)
	}
	if n, ok := parsed["context_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json context_ms: %v want 0\n%s", parsed["context_ms"], js)
	}
	if n, ok := parsed["streams_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json streams_ms: %v want 0\n%s", parsed["streams_ms"], js)
	}
	if n, ok := parsed["catalog_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json catalog_ms: %v want 0\n%s", parsed["catalog_ms"], js)
	}
	if n, ok := parsed["emit_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json emit_ms: %v want 0\n%s", parsed["emit_ms"], js)
	}
	if n, ok := parsed["llm_meter_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json llm_meter_ms: %v want 0\n%s", parsed["llm_meter_ms"], js)
	}
	if n, ok := parsed["pub_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json pub_ms: %v want 0\n%s", parsed["pub_ms"], js)
	}
	if n, ok := parsed["policy_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json policy_ms: %v want 0\n%s", parsed["policy_ms"], js)
	}
	if n, ok := parsed["consumer_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json consumer_ms: %v want 0\n%s", parsed["consumer_ms"], js)
	}
	if n, ok := parsed["kv_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json kv_ms: %v want 0\n%s", parsed["kv_ms"], js)
	}
	if n, ok := parsed["kv_ensure_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json kv_ensure_ms: %v want 0\n%s", parsed["kv_ensure_ms"], js)
	}
	if n, ok := parsed["kv_list_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json kv_list_ms: %v want 0\n%s", parsed["kv_list_ms"], js)
	}
	if n, ok := parsed["memory_ingest_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json memory_ingest_ms: %v want 0\n%s", parsed["memory_ingest_ms"], js)
	}
	if n, ok := parsed["memory_recall_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json memory_recall_ms: %v want 0\n%s", parsed["memory_recall_ms"], js)
	}
	if n, ok := parsed["memory_retrieve_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json memory_retrieve_ms: %v want 0\n%s", parsed["memory_retrieve_ms"], js)
	}
	if n, ok := parsed["duration_ms"].(float64); !ok || int(n) < 0 {
		t.Fatalf("json duration_ms: %v want >= 0\n%s", parsed["duration_ms"], js)
	}
	if _, ok := parsed["version"]; !ok {
		t.Fatalf("json version missing when disabled\n%s", js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "wait_ready_elapsed_ms: 0") {
		t.Fatalf("text report missing wait_ready_elapsed_ms 0:\n%s", text)
	}
	if !strings.Contains(text, "wait_ready_interval_ms: 0") {
		t.Fatalf("text report missing wait_ready_interval_ms 0:\n%s", text)
	}
	if !strings.Contains(text, "wait_require_health: false") {
		t.Fatalf("text report missing wait_require_health false:\n%s", text)
	}
	if !strings.Contains(text, "wait_ready_result: off") {
		t.Fatalf("text report missing wait_ready_result off:\n%s", text)
	}
	if !strings.Contains(text, "wait_ready_attempts: 0") {
		t.Fatalf("text report missing wait_ready_attempts 0:\n%s", text)
	}
	if !strings.Contains(text, "health_ms: 0") || !strings.Contains(text, "ready_ms: 0") {
		t.Fatalf("text report missing health_ms/ready_ms 0:\n%s", text)
	}
	if !strings.Contains(text, "context_ms: 0") || !strings.Contains(text, "streams_ms: 0") || !strings.Contains(text, "catalog_ms: 0") {
		t.Fatalf("text report missing context_ms/streams_ms/catalog_ms 0:\n%s", text)
	}
	if !strings.Contains(text, "emit_ms: 0") || !strings.Contains(text, "policy_ms: 0") {
		t.Fatalf("text report missing emit_ms/policy_ms 0:\n%s", text)
	}
	if !strings.Contains(text, "llm_meter_ms: 0") || !strings.Contains(text, "pub_ms: 0") {
		t.Fatalf("text report missing llm_meter_ms/pub_ms 0:\n%s", text)
	}
	if !strings.Contains(text, "consumer_ms: 0") || !strings.Contains(text, "kv_ms: 0") {
		t.Fatalf("text report missing consumer_ms/kv_ms 0:\n%s", text)
	}
	if !strings.Contains(text, "kv_ensure_ms: 0") {
		t.Fatalf("text report missing kv_ensure_ms 0:\n%s", text)
	}
	if !strings.Contains(text, "kv_list_ms: 0") {
		t.Fatalf("text report missing kv_list_ms 0:\n%s", text)
	}
	if !strings.Contains(text, "memory_ingest_ms: 0") || !strings.Contains(text, "memory_recall_ms: 0") || !strings.Contains(text, "memory_retrieve_ms: 0") {
		t.Fatalf("text report missing memory_*_ms 0:\n%s", text)
	}
	if !strings.Contains(text, "duration_ms:") {
		t.Fatalf("text report missing duration_ms:\n%s", text)
	}
	if !strings.Contains(text, "steps_pass: 0") || !strings.Contains(text, "steps_fail: 0") {
		t.Fatalf("text report missing steps_pass/fail 0:\n%s", text)
	}
	if !strings.Contains(text, "steps_skip: 1") {
		t.Fatalf("text report missing steps_skip: 1:\n%s", text)
	}
}

func TestDogfood_VersionEvidence(t *testing.T) {
	// opts.Version is always copied (trimmed) into report + JSON/text.
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
	rep := c.Dogfood(context.Background(), DogfoodOptions{
		SkipMemory: true,
		Version:    "  1.2.3-test  ",
	})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if rep.Version != "1.2.3-test" {
		t.Fatalf("Version: %q want 1.2.3-test", rep.Version)
	}
	js := FormatReportJSON(rep)
	if !strings.Contains(js, `"version": "1.2.3-test"`) &&
		!strings.Contains(js, `"version":"1.2.3-test"`) {
		t.Fatalf("FormatReportJSON missing version:\n%s", js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "version:  1.2.3-test") && !strings.Contains(text, "version: 1.2.3-test") {
		t.Fatalf("FormatReport missing version:\n%s", text)
	}

	// Empty opts.Version + empty ProductVersion → always emit empty string (not omitted).
	prevPV := ProductVersion()
	productVersion = ""
	t.Cleanup(func() { productVersion = prevPV })

	rep2 := c.Dogfood(context.Background(), DogfoodOptions{SkipMemory: true})
	if rep2.Version != "" {
		t.Fatalf("empty Version want \"\", got %q", rep2.Version)
	}
	js2 := FormatReportJSON(rep2)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js2), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js2)
	}
	if v, ok := parsed["version"].(string); !ok || v != "" {
		t.Fatalf("json version when unset: %v want \"\"\n%s", parsed["version"], js2)
	}

	// Empty opts.Version falls back to ProductVersion when set.
	SetProductVersion("  9.9.9-product  ")
	rep3 := c.Dogfood(context.Background(), DogfoodOptions{SkipMemory: true})
	if rep3.Version != "9.9.9-product" {
		t.Fatalf("ProductVersion default: %q want 9.9.9-product", rep3.Version)
	}
	// Explicit opts.Version still wins over ProductVersion.
	rep4 := c.Dogfood(context.Background(), DogfoodOptions{SkipMemory: true, Version: "opts-wins"})
	if rep4.Version != "opts-wins" {
		t.Fatalf("opts.Version should win: %q", rep4.Version)
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
	// Structured JSON fields for stage CI / mesh gates (s237).
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
	// Always-emit: top-level org/workspace keys present as empty strings when unset
	// (step detail still omits org=/workspace= tokens above).
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	for _, key := range []string{"org", "workspace"} {
		v, ok := parsed[key]
		if !ok {
			t.Fatalf("always-emit %s key missing:\n%s", key, js)
		}
		if s, ok := v.(string); !ok || s != "" {
			t.Fatalf("unset %s should emit empty string, got %v\n%s", key, v, js)
		}
	}
}

func TestFormatReport_AlwaysEmitsProbeErrs(t *testing.T) {
	// health_err / ready_err always present as strings in JSON and text, including empty.
	// Peers mesh status probe-err always-emit continuum.
	t.Run("empty", func(t *testing.T) {
		empty := DogfoodReport{Summary: "PASS", OK: true}
		js := FormatReportJSON(empty)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(js), &parsed); err != nil {
			t.Fatalf("json: %v\n%s", err, js)
		}
		for _, key := range []string{"health_err", "ready_err"} {
			v, ok := parsed[key]
			if !ok {
				t.Fatalf("always-emit %s key missing:\n%s", key, js)
			}
			str, ok := v.(string)
			if !ok {
				t.Fatalf("%s: %v want string\n%s", key, v, js)
			}
			if str != "" {
				t.Fatalf("%s: %q want empty string\n%s", key, str, js)
			}
		}
		text := FormatReport(empty)
		for _, want := range []string{
			"health_err: ",
			"ready_err: ",
		} {
			if !strings.Contains(text, want) {
				t.Fatalf("text always-emit probe-err line missing %q:\n%s", want, text)
			}
		}
		// Order: health_ms → health_err → ready_ms → ready_err
		// Prefix with newline+indent so "ready_ms" is not confused with "wait_ready_ms".
		hmsIdx := strings.Index(text, "\n  health_ms:")
		heIdx := strings.Index(text, "\n  health_err:")
		rmsIdx := strings.Index(text, "\n  ready_ms:")
		reIdx := strings.Index(text, "\n  ready_err:")
		if hmsIdx < 0 || heIdx < 0 || rmsIdx < 0 || reIdx < 0 {
			t.Fatalf("missing probe latency/err keys:\n%s", text)
		}
		if !(hmsIdx < heIdx && heIdx < rmsIdx && rmsIdx < reIdx) {
			t.Fatalf("probe order want health_ms < health_err < ready_ms < ready_err:\n%s", text)
		}
	})
	t.Run("populated", func(t *testing.T) {
		rep := DogfoodReport{
			HealthErr: "connection refused",
			ReadyErr:  "iomesh ready: http 503",
			Summary:   "FAIL",
			OK:        false,
		}
		js := FormatReportJSON(rep)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(js), &parsed); err != nil {
			t.Fatalf("json: %v\n%s", err, js)
		}
		if got, _ := parsed["health_err"].(string); got != "connection refused" {
			t.Fatalf("health_err: %v want connection refused\n%s", parsed["health_err"], js)
		}
		if got, _ := parsed["ready_err"].(string); got != "iomesh ready: http 503" {
			t.Fatalf("ready_err: %v want iomesh ready: http 503\n%s", parsed["ready_err"], js)
		}
		text := FormatReport(rep)
		for _, line := range []string{
			"health_err: connection refused",
			"ready_err: iomesh ready: http 503",
		} {
			if !strings.Contains(text, line) {
				t.Fatalf("text missing %q:\n%s", line, text)
			}
		}
	})
	t.Run("mesh_disabled_empty", func(t *testing.T) {
		// Mesh-disabled early return: both empty string (always emit keys).
		rep := (*Client)(nil).Dogfood(context.Background(), DogfoodOptions{})
		if rep.HealthErr != "" || rep.ReadyErr != "" {
			t.Fatalf("disabled HealthErr/ReadyErr: %q/%q want empty", rep.HealthErr, rep.ReadyErr)
		}
		js := FormatReportJSON(rep)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(js), &parsed); err != nil {
			t.Fatalf("json: %v\n%s", err, js)
		}
		for _, key := range []string{"health_err", "ready_err"} {
			v, ok := parsed[key]
			if !ok {
				t.Fatalf("disabled always-emit %s key missing:\n%s", key, js)
			}
			str, ok := v.(string)
			if !ok || str != "" {
				t.Fatalf("disabled %s: %v want empty string\n%s", key, v, js)
			}
		}
		text := FormatReport(rep)
		if !strings.Contains(text, "health_err: ") || !strings.Contains(text, "ready_err: ") {
			t.Fatalf("disabled text missing health_err/ready_err lines:\n%s", text)
		}
	})
	t.Run("pass_empty_ok", func(t *testing.T) {
		// Full health/ready PASS → empty err strings (honest: no invented failure).
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
		c := New(Config{Enabled: true, Endpoint: srv.URL, EmitDeptStreams: true}, nil)
		rep := c.Dogfood(context.Background(), DogfoodOptions{SkipContext: true, SkipMemory: true, SkipStreams: true})
		if rep.HealthErr != "" {
			t.Fatalf("pass HealthErr: %q want empty", rep.HealthErr)
		}
		if rep.ReadyErr != "" {
			t.Fatalf("pass ReadyErr: %q want empty", rep.ReadyErr)
		}
		js := FormatReportJSON(rep)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(js), &parsed); err != nil {
			t.Fatalf("json: %v\n%s", err, js)
		}
		for _, key := range []string{"health_err", "ready_err"} {
			str, ok := parsed[key].(string)
			if !ok {
				t.Fatalf("%s missing or non-string: %v\n%s", key, parsed[key], js)
			}
			if str != "" {
				t.Fatalf("%s: %q want empty on PASS\n%s", key, str, js)
			}
		}
	})
	t.Run("ready_soft_skip_embeds_err", func(t *testing.T) {
		// Optional 404 ready → soft SKIP with underlying err captured in ready_err.
		srv := mockMeshServer(t, struct {
			failHealth bool
			noReady    bool
			emptyCtx   bool
			failEmit   bool
			failMemory bool
			noMemory   bool
			failRecall bool
			noRecall   bool
		}{noReady: true})
		t.Cleanup(srv.Close)
		c := New(Config{Enabled: true, Endpoint: srv.URL, EmitDeptStreams: true}, nil)
		rep := c.Dogfood(context.Background(), DogfoodOptions{SkipContext: true, SkipMemory: true, SkipStreams: true})
		if rep.HealthErr != "" {
			t.Fatalf("HealthErr on ready-only soft skip: %q want empty", rep.HealthErr)
		}
		if rep.ReadyErr == "" {
			t.Fatalf("ReadyErr empty on ready soft SKIP with embedded err; want non-empty")
		}
		// Underlying Ready() error typically mentions http 404.
		if !strings.Contains(rep.ReadyErr, "404") && !strings.Contains(strings.ToLower(rep.ReadyErr), "ready") {
			t.Fatalf("ReadyErr unexpected: %q", rep.ReadyErr)
		}
		js := FormatReportJSON(rep)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(js), &parsed); err != nil {
			t.Fatalf("json: %v\n%s", err, js)
		}
		re, ok := parsed["ready_err"].(string)
		if !ok || re == "" {
			t.Fatalf("json ready_err: %v want non-empty\n%s", parsed["ready_err"], js)
		}
		if re != rep.ReadyErr {
			t.Fatalf("json ready_err %q != report %q", re, rep.ReadyErr)
		}
		he, ok := parsed["health_err"].(string)
		if !ok || he != "" {
			t.Fatalf("json health_err: %v want empty string\n%s", parsed["health_err"], js)
		}
		text := FormatReport(rep)
		if !strings.Contains(text, "ready_err: "+rep.ReadyErr) {
			t.Fatalf("text missing ready_err detail:\n%s", text)
		}
	})
}

func TestFormatReport_AlwaysEmitsIdentity(t *testing.T) {
	// tenant/org/workspace always present as strings in JSON and text, including empty.
	// endpoint already always printed. Peers mesh status identity always-emit.
	t.Run("empty", func(t *testing.T) {
		empty := DogfoodReport{Summary: "PASS", OK: true}
		js := FormatReportJSON(empty)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(js), &parsed); err != nil {
			t.Fatalf("json: %v\n%s", err, js)
		}
		for _, key := range []string{"endpoint", "tenant", "org", "workspace"} {
			v, ok := parsed[key]
			if !ok {
				t.Fatalf("always-emit %s key missing:\n%s", key, js)
			}
			str, ok := v.(string)
			if !ok {
				t.Fatalf("%s: %v want string\n%s", key, v, js)
			}
			if str != "" {
				t.Fatalf("%s: %q want empty string\n%s", key, str, js)
			}
		}
		text := FormatReport(empty)
		for _, want := range []string{
			"endpoint: ",
			"tenant:   ",
			"org:      ",
			"workspace: ",
		} {
			if !strings.Contains(text, want) {
				t.Fatalf("text always-emit identity line missing %q:\n%s", want, text)
			}
		}
		// Order: endpoint → tenant → org → workspace
		epIdx := strings.Index(text, "endpoint:")
		tnIdx := strings.Index(text, "tenant:")
		orgIdx := strings.Index(text, "org:")
		wsIdx := strings.Index(text, "workspace:")
		if epIdx < 0 || tnIdx < 0 || orgIdx < 0 || wsIdx < 0 {
			t.Fatalf("missing identity keys:\n%s", text)
		}
		if !(epIdx < tnIdx && tnIdx < orgIdx && orgIdx < wsIdx) {
			t.Fatalf("identity order want endpoint < tenant < org < workspace:\n%s", text)
		}
	})
	t.Run("populated", func(t *testing.T) {
		rep := DogfoodReport{
			Endpoint:  "http://mesh.example",
			Tenant:    "t1",
			Org:       "o1",
			Workspace: "w1",
			Summary:   "PASS",
			OK:        true,
		}
		js := FormatReportJSON(rep)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(js), &parsed); err != nil {
			t.Fatalf("json: %v\n%s", err, js)
		}
		want := map[string]string{
			"endpoint":  "http://mesh.example",
			"tenant":    "t1",
			"org":       "o1",
			"workspace": "w1",
		}
		for key, w := range want {
			got, ok := parsed[key].(string)
			if !ok || got != w {
				t.Fatalf("%s: %v want %q\n%s", key, parsed[key], w, js)
			}
		}
		text := FormatReport(rep)
		for _, line := range []string{
			"endpoint: http://mesh.example",
			"tenant:   t1",
			"org:      o1",
			"workspace: w1",
		} {
			if !strings.Contains(text, line) {
				t.Fatalf("text missing %q:\n%s", line, text)
			}
		}
	})
}

func TestFormatReport_AlwaysEmitsKVConsumerIdentity(t *testing.T) {
	// kv_bucket / consumer_stream / consumer_name / consumer_filter / pull_role /
	// pull_allow_suffix always present as strings in JSON and text, including empty.
	// Empty identity does not invent probe success (bools/counts stay zero-value).
	// Peers identity always-emit continuum (s687 pull_role/pull_allow_suffix).
	keys := []string{"kv_bucket", "consumer_stream", "consumer_name", "consumer_filter", "pull_role", "pull_allow_suffix"}
	t.Run("empty", func(t *testing.T) {
		empty := DogfoodReport{Summary: "PASS", OK: true}
		if empty.KVBucket != "" || empty.ConsumerStream != "" || empty.ConsumerName != "" || empty.ConsumerFilter != "" ||
			empty.PullRole != "" || empty.PullAllowSuffix != "" {
			t.Fatalf("zero-value identity: kv=%q stream=%q name=%q filter=%q role=%q suffix=%q",
				empty.KVBucket, empty.ConsumerStream, empty.ConsumerName, empty.ConsumerFilter,
				empty.PullRole, empty.PullAllowSuffix)
		}
		// Empty identity must compose with false probe flags / zero counts.
		if empty.KVKeyCount != 0 || empty.KVEnsured || empty.ConsumerProbed || empty.ConsumerOK {
			t.Fatalf("zero-value must not invent success: count=%d ensured=%v probed=%v ok=%v",
				empty.KVKeyCount, empty.KVEnsured, empty.ConsumerProbed, empty.ConsumerOK)
		}
		js := FormatReportJSON(empty)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(js), &parsed); err != nil {
			t.Fatalf("json: %v\n%s", err, js)
		}
		for _, key := range keys {
			v, ok := parsed[key]
			if !ok {
				t.Fatalf("always-emit %s key missing:\n%s", key, js)
			}
			str, ok := v.(string)
			if !ok {
				t.Fatalf("%s: %v want string\n%s", key, v, js)
			}
			if str != "" {
				t.Fatalf("%s: %q want empty string\n%s", key, str, js)
			}
		}
		// Bools still always-emit false (unchanged).
		for _, key := range []string{"kv_ensured", "consumer_probed", "consumer_ok"} {
			if v, ok := parsed[key].(bool); !ok || v {
				t.Fatalf("json %s: %v want false\n%s", key, parsed[key], js)
			}
		}
		text := FormatReport(empty)
		for _, want := range []string{
			"kv_bucket: ",
			"consumer_stream: ",
			"consumer_name: ",
			"consumer_filter: ",
			"pull_role: ",
			"pull_allow_suffix: ",
		} {
			if !strings.Contains(text, want) {
				t.Fatalf("text always-emit identity line missing %q:\n%s", want, text)
			}
		}
		// Order: kv_bucket → consumer_stream → consumer_name → consumer_filter → pull_role → pull_allow_suffix
		kvIdx := strings.Index(text, "kv_bucket:")
		csIdx := strings.Index(text, "consumer_stream:")
		cnIdx := strings.Index(text, "consumer_name:")
		cfIdx := strings.Index(text, "consumer_filter:")
		prIdx := strings.Index(text, "pull_role:")
		psIdx := strings.Index(text, "pull_allow_suffix:")
		if kvIdx < 0 || csIdx < 0 || cnIdx < 0 || cfIdx < 0 || prIdx < 0 || psIdx < 0 {
			t.Fatalf("missing identity keys:\n%s", text)
		}
		if !(kvIdx < csIdx && csIdx < cnIdx && cnIdx < cfIdx && cfIdx < prIdx && prIdx < psIdx) {
			t.Fatalf("identity order want kv_bucket < consumer_stream < consumer_name < consumer_filter < pull_role < pull_allow_suffix:\n%s", text)
		}
	})
	t.Run("populated", func(t *testing.T) {
		rep := DogfoodReport{
			KVBucket:        "config",
			ConsumerStream:  "EVENTS",
			ConsumerName:    "worker-1",
			ConsumerFilter:  "dept.events.>",
			PullRole:        "memory",
			PullAllowSuffix: "ops",
			// Values pass through; probe flags independent (compose, don't invent).
			KVKeyCount:     3,
			ConsumerProbed: true,
			ConsumerOK:     true,
			Summary:        "PASS",
			OK:             true,
		}
		js := FormatReportJSON(rep)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(js), &parsed); err != nil {
			t.Fatalf("json: %v\n%s", err, js)
		}
		want := map[string]string{
			"kv_bucket":         "config",
			"consumer_stream":   "EVENTS",
			"consumer_name":     "worker-1",
			"consumer_filter":   "dept.events.>",
			"pull_role":         "memory",
			"pull_allow_suffix": "ops",
		}
		for key, w := range want {
			got, ok := parsed[key].(string)
			if !ok || got != w {
				t.Fatalf("%s: %v want %q\n%s", key, parsed[key], w, js)
			}
		}
		text := FormatReport(rep)
		for _, line := range []string{
			"kv_bucket: config",
			"consumer_stream: EVENTS",
			"consumer_name: worker-1",
			"consumer_filter: dept.events.>",
			"pull_role: memory",
			"pull_allow_suffix: ops",
		} {
			if !strings.Contains(text, line) {
				t.Fatalf("text missing %q:\n%s", line, text)
			}
		}
	})
}

func TestFormatReport_AlwaysEmitsCatalogPolicySource(t *testing.T) {
	// catalog_source / policy_source always present as strings in JSON and text, including empty.
	// Peers identity / memory_endpoint / health_err always-emit continuum.
	t.Run("empty", func(t *testing.T) {
		empty := DogfoodReport{Summary: "PASS", OK: true}
		if empty.CatalogSource != "" || empty.PolicySource != "" {
			t.Fatalf("zero-value CatalogSource/PolicySource: %q/%q want empty", empty.CatalogSource, empty.PolicySource)
		}
		js := FormatReportJSON(empty)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(js), &parsed); err != nil {
			t.Fatalf("json: %v\n%s", err, js)
		}
		for _, key := range []string{"catalog_source", "policy_source"} {
			v, ok := parsed[key]
			if !ok {
				t.Fatalf("always-emit %s key missing:\n%s", key, js)
			}
			str, ok := v.(string)
			if !ok {
				t.Fatalf("%s: %v want string\n%s", key, v, js)
			}
			if str != "" {
				t.Fatalf("%s: %q want empty string\n%s", key, str, js)
			}
			if !strings.Contains(js, `"`+key+`"`) {
				t.Fatalf("json missing %s key:\n%s", key, js)
			}
		}
		text := FormatReport(empty)
		for _, want := range []string{
			"catalog_source: ",
			"policy_source: ",
		} {
			if !strings.Contains(text, want) {
				t.Fatalf("text always-emit source line missing %q:\n%s", want, text)
			}
		}
		// Order: dual_write → catalog_source → catalog_count; policy_mode → policy_source
		dwIdx := strings.Index(text, "dual_write:")
		csIdx := strings.Index(text, "catalog_source:")
		ccIdx := strings.Index(text, "catalog_count:")
		pmIdx := strings.Index(text, "policy_mode:")
		psIdx := strings.Index(text, "policy_source:")
		if dwIdx < 0 || csIdx < 0 || ccIdx < 0 || pmIdx < 0 || psIdx < 0 {
			t.Fatalf("missing source order keys:\n%s", text)
		}
		if !(dwIdx < csIdx && csIdx < ccIdx) {
			t.Fatalf("order want dual_write < catalog_source < catalog_count:\n%s", text)
		}
		if !(pmIdx < psIdx) {
			t.Fatalf("order want policy_mode < policy_source:\n%s", text)
		}
	})
	t.Run("populated", func(t *testing.T) {
		rep := DogfoodReport{
			CatalogSource: "mesh",
			PolicySource:  "mesh",
			Summary:       "PASS",
			OK:            true,
		}
		js := FormatReportJSON(rep)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(js), &parsed); err != nil {
			t.Fatalf("json: %v\n%s", err, js)
		}
		if got, _ := parsed["catalog_source"].(string); got != "mesh" {
			t.Fatalf("catalog_source: %v want mesh\n%s", parsed["catalog_source"], js)
		}
		if got, _ := parsed["policy_source"].(string); got != "mesh" {
			t.Fatalf("policy_source: %v want mesh\n%s", parsed["policy_source"], js)
		}
		text := FormatReport(rep)
		for _, line := range []string{
			"catalog_source: mesh",
			"policy_source: mesh",
		} {
			if !strings.Contains(text, line) {
				t.Fatalf("text missing %q:\n%s", line, text)
			}
		}
	})
	t.Run("mesh_disabled_empty", func(t *testing.T) {
		// Mesh-disabled early return: both empty string (always emit keys).
		c := New(Config{Enabled: false}, nil)
		rep := c.Dogfood(context.Background(), DogfoodOptions{})
		if rep.CatalogSource != "" || rep.PolicySource != "" {
			t.Fatalf("disabled CatalogSource/PolicySource: %q/%q want empty", rep.CatalogSource, rep.PolicySource)
		}
		js := FormatReportJSON(rep)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(js), &parsed); err != nil {
			t.Fatalf("json: %v\n%s", err, js)
		}
		for _, key := range []string{"catalog_source", "policy_source"} {
			v, ok := parsed[key]
			if !ok {
				t.Fatalf("disabled always-emit %s key missing:\n%s", key, js)
			}
			str, ok := v.(string)
			if !ok || str != "" {
				t.Fatalf("disabled %s: %v want empty string\n%s", key, v, js)
			}
		}
		text := FormatReport(rep)
		if !strings.Contains(text, "catalog_source: ") || !strings.Contains(text, "policy_source: ") {
			t.Fatalf("disabled text missing catalog_source/policy_source lines:\n%s", text)
		}
	})
}

func TestFormatReport_AlwaysEmitsMemoryEndpoint(t *testing.T) {
	// memory_endpoint always present as string in JSON and text, including empty.
	// Peers identity always-emit mold (endpoint/tenant); empty-honest when unset —
	// does not invent memory plane readiness.
	t.Run("empty", func(t *testing.T) {
		empty := DogfoodReport{Summary: "PASS", OK: true}
		if empty.MemoryEndpoint != "" {
			t.Fatalf("zero-value MemoryEndpoint: %q want empty", empty.MemoryEndpoint)
		}
		js := FormatReportJSON(empty)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(js), &parsed); err != nil {
			t.Fatalf("json: %v\n%s", err, js)
		}
		v, ok := parsed["memory_endpoint"]
		if !ok {
			t.Fatalf("always-emit memory_endpoint key missing:\n%s", js)
		}
		str, ok := v.(string)
		if !ok {
			t.Fatalf("memory_endpoint: %v want string\n%s", v, js)
		}
		if str != "" {
			t.Fatalf("memory_endpoint: %q want empty string\n%s", str, js)
		}
		// Ensure key is present even when empty (no omitempty gap).
		if !strings.Contains(js, `"memory_endpoint"`) {
			t.Fatalf("json missing memory_endpoint key:\n%s", js)
		}
		text := FormatReport(empty)
		// Format: "  memory_endpoint: %s\n" → empty value still prints the line.
		if !strings.Contains(text, "memory_endpoint:") {
			t.Fatalf("text always-emit memory_endpoint line missing:\n%s", text)
		}
		if !strings.Contains(text, "memory_endpoint: ") {
			t.Fatalf("text memory_endpoint line not empty-honest:\n%s", text)
		}
		// Order: workspace → memory_endpoint → dual_write
		wsIdx := strings.Index(text, "workspace:")
		meIdx := strings.Index(text, "memory_endpoint:")
		dwIdx := strings.Index(text, "dual_write:")
		if wsIdx < 0 || meIdx < 0 || dwIdx < 0 {
			t.Fatalf("missing order keys:\n%s", text)
		}
		if !(wsIdx < meIdx && meIdx < dwIdx) {
			t.Fatalf("order want workspace < memory_endpoint < dual_write:\n%s", text)
		}
	})
	t.Run("populated", func(t *testing.T) {
		want := "http://memory.sidecar:8080"
		rep := DogfoodReport{
			Endpoint:       "http://mesh.example",
			MemoryEndpoint: want,
			Summary:        "PASS",
			OK:             true,
		}
		js := FormatReportJSON(rep)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(js), &parsed); err != nil {
			t.Fatalf("json: %v\n%s", err, js)
		}
		got, ok := parsed["memory_endpoint"].(string)
		if !ok || got != want {
			t.Fatalf("memory_endpoint: %v want %q\n%s", parsed["memory_endpoint"], want, js)
		}
		text := FormatReport(rep)
		if !strings.Contains(text, "memory_endpoint: "+want) {
			t.Fatalf("text missing memory_endpoint value:\n%s", text)
		}
	})
	t.Run("mesh_disabled_empty", func(t *testing.T) {
		// Mesh-disabled early return: memory_endpoint still always emitted as "".
		c := New(Config{Enabled: false}, nil)
		rep := c.Dogfood(context.Background(), DogfoodOptions{})
		if rep.MemoryEndpoint != "" {
			t.Fatalf("disabled MemoryEndpoint: %q want empty", rep.MemoryEndpoint)
		}
		js := FormatReportJSON(rep)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(js), &parsed); err != nil {
			t.Fatalf("json: %v\n%s", err, js)
		}
		got, ok := parsed["memory_endpoint"].(string)
		if !ok {
			t.Fatalf("disabled memory_endpoint: %v want empty string\n%s", parsed["memory_endpoint"], js)
		}
		if got != "" {
			t.Fatalf("disabled memory_endpoint: %q want empty\n%s", got, js)
		}
		text := FormatReport(rep)
		if !strings.Contains(text, "memory_endpoint:") {
			t.Fatalf("disabled text missing memory_endpoint line:\n%s", text)
		}
	})
}

func TestFormatReportJSON_AlwaysEmitsStepDetailLatency(t *testing.T) {
	// Per-step detail + latency + latency_ms always present in JSON
	// (empty string / 0 when unset / zero) so CI scrapers can key on stable
	// step fields without omitempty gaps. Text report already prints steps
	// with empty detail (no change); duration still shown in parens when timed.
	t.Run("empty_detail_zero_latency", func(t *testing.T) {
		rep := DogfoodReport{
			Summary: "PASS (pass=1)",
			OK:      true,
			Steps: []Step{
				{Name: "enabled", Status: StepPass}, // Detail="", Latency=0, LatencyMS=0
			},
		}
		js := FormatReportJSON(rep)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(js), &parsed); err != nil {
			t.Fatalf("json: %v\n%s", err, js)
		}
		steps, ok := parsed["steps"].([]any)
		if !ok || len(steps) != 1 {
			t.Fatalf("steps: %v want len 1\n%s", parsed["steps"], js)
		}
		step, ok := steps[0].(map[string]any)
		if !ok {
			t.Fatalf("step[0] type: %T\n%s", steps[0], js)
		}
		for _, key := range []string{"detail", "latency"} {
			v, ok := step[key]
			if !ok {
				t.Fatalf("always-emit step %s key missing:\n%s", key, js)
			}
			str, ok := v.(string)
			if !ok {
				t.Fatalf("step %s: %v (%T) want string\n%s", key, v, v, js)
			}
			if str != "" {
				t.Fatalf("step %s: %q want empty string when unset/zero\n%s", key, str, js)
			}
		}
		lms, ok := step["latency_ms"]
		if !ok {
			t.Fatalf("always-emit step latency_ms key missing:\n%s", js)
		}
		// JSON numbers unmarshal as float64 into map[string]any
		n, ok := lms.(float64)
		if !ok {
			t.Fatalf("step latency_ms: %v (%T) want number\n%s", lms, lms, js)
		}
		if n != 0 {
			t.Fatalf("step latency_ms: %v want 0 when zero/not timed\n%s", n, js)
		}
		// Text still prints step with empty detail OK.
		text := FormatReport(rep)
		if !strings.Contains(text, "enabled") || !strings.Contains(text, "PASS") {
			t.Fatalf("text missing enabled PASS step:\n%s", text)
		}
	})
	t.Run("populated", func(t *testing.T) {
		rep := DogfoodReport{
			Summary: "PASS (pass=1)",
			OK:      true,
			Steps: []Step{
				{
					Name:      "health",
					Status:    StepPass,
					Detail:    "ok",
					Latency:   12 * time.Millisecond,
					LatencyMS: 12,
				},
			},
		}
		js := FormatReportJSON(rep)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(js), &parsed); err != nil {
			t.Fatalf("json: %v\n%s", err, js)
		}
		steps := parsed["steps"].([]any)
		step := steps[0].(map[string]any)
		if got, _ := step["detail"].(string); got != "ok" {
			t.Fatalf("detail: %v want ok\n%s", step["detail"], js)
		}
		lat, ok := step["latency"].(string)
		if !ok || lat == "" {
			t.Fatalf("latency: %v want non-empty duration string\n%s", step["latency"], js)
		}
		// Round(ms) → "12ms"
		if lat != "12ms" {
			t.Fatalf("latency: %q want 12ms\n%s", lat, js)
		}
		lms, ok := step["latency_ms"].(float64)
		if !ok {
			t.Fatalf("latency_ms: %v (%T) want number\n%s", step["latency_ms"], step["latency_ms"], js)
		}
		if lms != 12 {
			t.Fatalf("latency_ms: %v want 12\n%s", lms, js)
		}
	})
	t.Run("populated_derive_ms_from_latency", func(t *testing.T) {
		// Hand-built Step with only Latency set still emits latency_ms from Duration.
		rep := DogfoodReport{
			Summary: "PASS (pass=1)",
			OK:      true,
			Steps: []Step{
				{
					Name:    "health",
					Status:  StepPass,
					Detail:  "ok",
					Latency: 25 * time.Millisecond,
					// LatencyMS left 0 — FormatReportJSON derives from Latency
				},
			},
		}
		js := FormatReportJSON(rep)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(js), &parsed); err != nil {
			t.Fatalf("json: %v\n%s", err, js)
		}
		step := parsed["steps"].([]any)[0].(map[string]any)
		lms, ok := step["latency_ms"].(float64)
		if !ok {
			t.Fatalf("latency_ms: %v (%T) want number\n%s", step["latency_ms"], step["latency_ms"], js)
		}
		if lms != 25 {
			t.Fatalf("latency_ms: %v want 25 (derived from Latency)\n%s", lms, js)
		}
		if lat, _ := step["latency"].(string); lat != "25ms" {
			t.Fatalf("latency: %q want 25ms\n%s", lat, js)
		}
	})
	t.Run("mesh_disabled_steps", func(t *testing.T) {
		// Mesh-disabled early return still emits steps with detail+latency+latency_ms keys.
		c := New(Config{Enabled: false}, nil)
		rep := c.Dogfood(context.Background(), DogfoodOptions{})
		if len(rep.Steps) == 0 {
			t.Fatal("expected at least one step when mesh disabled")
		}
		js := FormatReportJSON(rep)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(js), &parsed); err != nil {
			t.Fatalf("json: %v\n%s", err, js)
		}
		steps, ok := parsed["steps"].([]any)
		if !ok || len(steps) == 0 {
			t.Fatalf("steps missing/empty:\n%s", js)
		}
		for i, raw := range steps {
			step, ok := raw.(map[string]any)
			if !ok {
				t.Fatalf("step[%d] type: %T", i, raw)
			}
			for _, key := range []string{"name", "status", "detail", "latency", "latency_ms"} {
				if _, ok := step[key]; !ok {
					t.Fatalf("step[%d] missing always-emit key %s:\n%s", i, key, js)
				}
			}
			if _, ok := step["detail"].(string); !ok {
				t.Fatalf("step[%d] detail not string: %v\n%s", i, step["detail"], js)
			}
			if _, ok := step["latency"].(string); !ok {
				t.Fatalf("step[%d] latency not string: %v\n%s", i, step["latency"], js)
			}
			if n, ok := step["latency_ms"].(float64); !ok {
				t.Fatalf("step[%d] latency_ms not number: %v\n%s", i, step["latency_ms"], js)
			} else if n < 0 {
				t.Fatalf("step[%d] latency_ms: %v want >= 0\n%s", i, n, js)
			}
		}
	})
	t.Run("step_timed_sets_latency_ms", func(t *testing.T) {
		// Live probe path: stepTimed populates both Latency and LatencyMS; JSON emits both.
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
		rep := c.Dogfood(context.Background(), DogfoodOptions{
			SkipContext: true,
			SkipEmit:    true,
			SkipMemory:  true,
			SkipStreams: true,
		})
		js := FormatReportJSON(rep)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(js), &parsed); err != nil {
			t.Fatalf("json: %v\n%s", err, js)
		}
		steps, ok := parsed["steps"].([]any)
		if !ok || len(steps) == 0 {
			t.Fatalf("steps missing/empty:\n%s", js)
		}
		var sawTimed bool
		for i, raw := range steps {
			step := raw.(map[string]any)
			if _, ok := step["latency_ms"]; !ok {
				t.Fatalf("step[%d] missing latency_ms:\n%s", i, js)
			}
			n, ok := step["latency_ms"].(float64)
			if !ok {
				t.Fatalf("step[%d] latency_ms not number: %v\n%s", i, step["latency_ms"], js)
			}
			if n < 0 {
				t.Fatalf("step[%d] latency_ms: %v want >= 0\n%s", i, n, js)
			}
			// Timed steps (health/ready via stepTimed) should have Latency set on report.
			name, _ := step["name"].(string)
			if name == "health" || name == "ready" {
				if n <= 0 {
					// Allow 0 on extremely fast local mock; still require key + number type.
					// Prefer >0 when wall clock records any ms.
				}
				// In-memory Step should have LatencyMS set when Latency is set.
				for _, s := range rep.Steps {
					if s.Name == name && s.Latency > 0 {
						if s.LatencyMS < 0 {
							t.Fatalf("in-memory step %s LatencyMS %d want >= 0", name, s.LatencyMS)
						}
						// LatencyMS should match Milliseconds() of Latency (clamped).
						want := int(s.Latency.Milliseconds())
						if want < 0 {
							want = 0
						}
						if s.LatencyMS != want {
							t.Fatalf("step %s LatencyMS=%d want %d (from Latency)", name, s.LatencyMS, want)
						}
						sawTimed = true
					}
				}
			}
		}
		if !sawTimed {
			// health/ready always run when mesh enabled; at least one should be timed.
			for _, s := range rep.Steps {
				if (s.Name == "health" || s.Name == "ready") && s.Latency > 0 {
					sawTimed = true
					break
				}
			}
			if !sawTimed {
				t.Fatalf("expected health/ready steps with Latency set from stepTimed; steps=%+v", rep.Steps)
			}
		}
	})
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
	// Always emit dual_write key (identity tenant/org/workspace always-emit peers).
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
	// Default policy mode off → policy_mode=off, policy_source=off, policy_allow="" (always-emit).
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
	if rep.PolicyAllow != "" {
		t.Fatalf("PolicyAllow: %q want empty when mode off", rep.PolicyAllow)
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
	if _, has := parsed["policy_allow"]; !has {
		t.Fatalf("json policy_allow must always be present when mode off:\n%s", js)
	}
	if got, _ := parsed["policy_allow"].(string); got != "" {
		t.Fatalf("json policy_allow: %v want empty string when mode off\n%s", parsed["policy_allow"], js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "policy_mode: off") {
		t.Fatalf("text report missing policy_mode:\n%s", text)
	}
	if !strings.Contains(text, "policy_source: off") {
		t.Fatalf("text report missing policy_source:\n%s", text)
	}
	if !strings.Contains(text, "policy_allow: ") {
		t.Fatalf("text report must always print policy_allow when mode off:\n%s", text)
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
	if rep.PolicyAllow != "true" {
		t.Fatalf("PolicyAllow: %q want true", rep.PolicyAllow)
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
	if parsed["policy_allow"] != "true" {
		t.Fatalf("json policy_allow: %v want \"true\"\n%s", parsed["policy_allow"], js)
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
	if rep.PolicyAllow != "false" {
		t.Fatalf("PolicyAllow: %q want false", rep.PolicyAllow)
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
	if parsed["policy_allow"] != "false" {
		t.Fatalf("json policy_allow: %v want \"false\"\n%s", parsed["policy_allow"], js)
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
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if ua, ok := parsed["user_agent"].(string); !ok || ua != "iomesh-tui/test-s290" {
		t.Fatalf("json user_agent: %v want iomesh-tui/test-s290\n%s", parsed["user_agent"], js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "user_agent: iomesh-tui/test-s290") {
		t.Fatalf("FormatReport missing user_agent:\n%s", text)
	}
	sl := c.StatusLine()
	if !strings.Contains(sl, "ua=iomesh-tui/test-s290") {
		t.Fatalf("StatusLine missing ua=: %s", sl)
	}

	// Mesh-disabled path still sets UserAgent early via UserAgent().
	repOff := (*Client)(nil).Dogfood(context.Background(), DogfoodOptions{})
	if repOff.UserAgent != "iomesh-tui/test-s290" {
		t.Fatalf("mesh-disabled UserAgent: %q", repOff.UserAgent)
	}
	jsOff := FormatReportJSON(repOff)
	var parsedOff map[string]any
	if err := json.Unmarshal([]byte(jsOff), &parsedOff); err != nil {
		t.Fatalf("json disabled: %v\n%s", err, jsOff)
	}
	if ua, ok := parsedOff["user_agent"].(string); !ok || ua != "iomesh-tui/test-s290" {
		t.Fatalf("disabled json user_agent: %v\n%s", parsedOff["user_agent"], jsOff)
	}
	if !strings.Contains(FormatReport(repOff), "user_agent: iomesh-tui/test-s290") {
		t.Fatalf("disabled text missing user_agent:\n%s", FormatReport(repOff))
	}

	// Always-emit: empty UserAgent still present in JSON (key not omitted) and text line.
	empty := DogfoodReport{UserAgent: "", Summary: "PASS", OK: true}
	jsEmpty := FormatReportJSON(empty)
	var parsedEmpty map[string]any
	if err := json.Unmarshal([]byte(jsEmpty), &parsedEmpty); err != nil {
		t.Fatalf("json empty: %v\n%s", err, jsEmpty)
	}
	uaEmpty, hasUA := parsedEmpty["user_agent"]
	if !hasUA {
		t.Fatalf("always-emit user_agent key missing:\n%s", jsEmpty)
	}
	if s, ok := uaEmpty.(string); !ok || s != "" {
		t.Fatalf("unset UserAgent should emit empty string, got %v\n%s", uaEmpty, jsEmpty)
	}
	textEmpty := FormatReport(empty)
	if !strings.Contains(textEmpty, "user_agent: ") {
		t.Fatalf("text always-emit user_agent line missing:\n%s", textEmpty)
	}
}

func TestStatusLine_ProductVersion(t *testing.T) {
	prev := ProductVersion()
	SetProductVersion("1.2.3-status")
	t.Cleanup(func() {
		// SetProductVersion ignores empty; re-apply previous non-empty if any.
		if prev != "" {
			SetProductVersion(prev)
		} else {
			// leave 1.2.3-status if no previous — force clear via package var not exported.
			// Other tests set their own ProductVersion with cleanup when needed.
			productVersion = prev
		}
	})

	c := New(Config{
		Enabled: true, Endpoint: "http://mesh.example", Tenant: "t1",
	}, nil)
	sl := c.StatusLine()
	if !strings.Contains(sl, "version=1.2.3-status") {
		t.Fatalf("StatusLine missing version=: %s", sl)
	}
	if !strings.Contains(sl, "ua=") {
		t.Fatalf("StatusLine missing ua=: %s", sl)
	}

	// Disabled mesh still surfaces product version for slash /mesh.
	off := New(Config{Enabled: false}, nil)
	slOff := off.StatusLine()
	if !strings.Contains(slOff, "mesh: disabled (offline-first)") {
		t.Fatalf("disabled StatusLine base: %s", slOff)
	}
	if !strings.Contains(slOff, "version=1.2.3-status") {
		t.Fatalf("disabled StatusLine missing version=: %s", slOff)
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
	if rep.WaitReadyElapsedMS <= 0 {
		t.Fatalf("WaitReadyElapsedMS: %d want > 0 when wait_ready ran", rep.WaitReadyElapsedMS)
	}
	if rep.WaitReadyIntervalMS != 10 {
		t.Fatalf("WaitReadyIntervalMS: %d want 10", rep.WaitReadyIntervalMS)
	}
	if rep.WaitRequireHealth {
		t.Fatalf("WaitRequireHealth: true want false")
	}
	if rep.WaitReadyResult != "ok" {
		t.Fatalf("WaitReadyResult: %q want ok", rep.WaitReadyResult)
	}
	if rep.WaitReadyAttempts < 1 {
		t.Fatalf("WaitReadyAttempts: %d want >= 1 on success", rep.WaitReadyAttempts)
	}
	wr, ok := dogfoodStep(rep, "wait_ready")
	if !ok || wr.Status != StepPass {
		t.Fatalf("wait_ready: ok=%v status=%s detail=%s", ok, wr.Status, wr.Detail)
	}
	if !strings.Contains(wr.Detail, "WaitReady OK") {
		t.Fatalf("wait_ready detail: %s", wr.Detail)
	}
	if !strings.Contains(wr.Detail, "attempts=") {
		t.Fatalf("wait_ready detail missing attempts=: %s", wr.Detail)
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
	if n, ok := parsed["wait_ready_elapsed_ms"].(float64); !ok || int(n) <= 0 {
		t.Fatalf("json wait_ready_elapsed_ms: %v want > 0\n%s", parsed["wait_ready_elapsed_ms"], js)
	}
	if n, ok := parsed["wait_ready_interval_ms"].(float64); !ok || int(n) != 10 {
		t.Fatalf("json wait_ready_interval_ms: %v want 10\n%s", parsed["wait_ready_interval_ms"], js)
	}
	if v, ok := parsed["wait_require_health"].(bool); !ok || v {
		t.Fatalf("json wait_require_health: %v want false\n%s", parsed["wait_require_health"], js)
	}
	if s, ok := parsed["wait_ready_result"].(string); !ok || s != "ok" {
		t.Fatalf("json wait_ready_result: %v want ok\n%s", parsed["wait_ready_result"], js)
	}
	if n, ok := parsed["wait_ready_attempts"].(float64); !ok || int(n) < 1 {
		t.Fatalf("json wait_ready_attempts: %v want >= 1\n%s", parsed["wait_ready_attempts"], js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "wait_ready_ms: 2000") {
		t.Fatalf("text report missing wait_ready_ms:\n%s", text)
	}
	if !strings.Contains(text, "wait_ready_elapsed_ms:") {
		t.Fatalf("text report missing wait_ready_elapsed_ms:\n%s", text)
	}
	if !strings.Contains(text, "wait_ready_interval_ms: 10") {
		t.Fatalf("text report missing wait_ready_interval_ms 10:\n%s", text)
	}
	if !strings.Contains(text, "wait_require_health: false") {
		t.Fatalf("text report missing wait_require_health false:\n%s", text)
	}
	if !strings.Contains(text, "wait_ready_result: ok") {
		t.Fatalf("text report missing wait_ready_result ok:\n%s", text)
	}
	if !strings.Contains(text, "wait_ready_attempts:") {
		t.Fatalf("text report missing wait_ready_attempts:\n%s", text)
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
	if rep.WaitReadyElapsedMS <= 0 {
		t.Fatalf("WaitReadyElapsedMS: %d want > 0 when wait_ready ran (soft timeout)", rep.WaitReadyElapsedMS)
	}
	if rep.WaitReadyResult != "skip" {
		t.Fatalf("WaitReadyResult: %q want skip", rep.WaitReadyResult)
	}
	if rep.WaitReadyAttempts < 1 {
		t.Fatalf("WaitReadyAttempts: %d want >= 1 on soft timeout", rep.WaitReadyAttempts)
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
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if s, ok := parsed["wait_ready_result"].(string); !ok || s != "skip" {
		t.Fatalf("json wait_ready_result: %v want skip\n%s", parsed["wait_ready_result"], js)
	}
	if n, ok := parsed["wait_ready_attempts"].(float64); !ok || int(n) < 1 {
		t.Fatalf("json wait_ready_attempts: %v want >= 1 on soft timeout\n%s", parsed["wait_ready_attempts"], js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "wait_ready_result: skip") {
		t.Fatalf("text report missing wait_ready_result skip:\n%s", text)
	}
	if !strings.Contains(text, "wait_ready_attempts:") {
		t.Fatalf("text report missing wait_ready_attempts:\n%s", text)
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
	if rep.WaitReadyResult != "err" {
		t.Fatalf("WaitReadyResult: %q want err", rep.WaitReadyResult)
	}
	wr, ok := dogfoodStep(rep, "wait_ready")
	if !ok || wr.Status != StepFail {
		t.Fatalf("wait_ready strict: ok=%v status=%s detail=%s", ok, wr.Status, wr.Detail)
	}
	if !strings.Contains(wr.Detail, "wait_ready:") {
		t.Fatalf("wait_ready detail: %s", wr.Detail)
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if s, ok := parsed["wait_ready_result"].(string); !ok || s != "err" {
		t.Fatalf("json wait_ready_result: %v want err\n%s", parsed["wait_ready_result"], js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "wait_ready_result: err") {
		t.Fatalf("text report missing wait_ready_result err:\n%s", text)
	}
}

func TestDogfood_WaitReady_DefaultOff(t *testing.T) {
	// Zero WaitReady: no wait_ready step; wait knobs always emitted (interval 0, require_health false).
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
	if rep.WaitReadyElapsedMS != 0 {
		t.Fatalf("WaitReadyElapsedMS: %d want 0 when wait_ready off", rep.WaitReadyElapsedMS)
	}
	if rep.WaitReadyIntervalMS != 0 {
		t.Fatalf("WaitReadyIntervalMS: %d want 0 when wait_ready off", rep.WaitReadyIntervalMS)
	}
	if rep.WaitRequireHealth {
		t.Fatalf("WaitRequireHealth: true want false when unset")
	}
	if rep.WaitReadyResult != "off" {
		t.Fatalf("WaitReadyResult: %q want off", rep.WaitReadyResult)
	}
	if rep.WaitReadyAttempts != 0 {
		t.Fatalf("WaitReadyAttempts: %d want 0 when wait off", rep.WaitReadyAttempts)
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
	if n, ok := parsed["wait_ready_elapsed_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json wait_ready_elapsed_ms: %v want 0\n%s", parsed["wait_ready_elapsed_ms"], js)
	}
	if n, ok := parsed["wait_ready_interval_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json wait_ready_interval_ms: %v want 0\n%s", parsed["wait_ready_interval_ms"], js)
	}
	if v, ok := parsed["wait_require_health"].(bool); !ok || v {
		t.Fatalf("json wait_require_health: %v want false\n%s", parsed["wait_require_health"], js)
	}
	if s, ok := parsed["wait_ready_result"].(string); !ok || s != "off" {
		t.Fatalf("json wait_ready_result: %v want off\n%s", parsed["wait_ready_result"], js)
	}
	if n, ok := parsed["wait_ready_attempts"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json wait_ready_attempts: %v want 0 when wait off\n%s", parsed["wait_ready_attempts"], js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "wait_ready_ms: 0") {
		t.Fatalf("text report missing wait_ready_ms 0:\n%s", text)
	}
	if !strings.Contains(text, "wait_ready_elapsed_ms: 0") {
		t.Fatalf("text report missing wait_ready_elapsed_ms 0:\n%s", text)
	}
	if !strings.Contains(text, "wait_ready_interval_ms: 0") {
		t.Fatalf("text report missing wait_ready_interval_ms 0:\n%s", text)
	}
	if !strings.Contains(text, "wait_require_health: false") {
		t.Fatalf("text report missing wait_require_health false:\n%s", text)
	}
	if !strings.Contains(text, "wait_ready_result: off") {
		t.Fatalf("text report missing wait_ready_result off:\n%s", text)
	}
	if !strings.Contains(text, "wait_ready_attempts: 0") {
		t.Fatalf("text report missing wait_ready_attempts 0:\n%s", text)
	}
}

func TestDogfood_WaitReady_DefaultIntervalAndRequireHealth(t *testing.T) {
	// WaitReady>0 with Interval=0 → effective interval_ms=500; WaitRequireHealth always emitted as configured.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case "/ready", "/readyz":
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
		WaitReady:         500 * time.Millisecond,
		WaitReadyInterval: 0, // effective default 500ms
		WaitRequireHealth: true,
	})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if rep.WaitReadyMS != 500 {
		t.Fatalf("WaitReadyMS: %d want 500", rep.WaitReadyMS)
	}
	if rep.WaitReadyIntervalMS != 500 {
		t.Fatalf("WaitReadyIntervalMS: %d want 500 (effective default)", rep.WaitReadyIntervalMS)
	}
	if !rep.WaitRequireHealth {
		t.Fatalf("WaitRequireHealth: false want true")
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if n, ok := parsed["wait_ready_interval_ms"].(float64); !ok || int(n) != 500 {
		t.Fatalf("json wait_ready_interval_ms: %v want 500\n%s", parsed["wait_ready_interval_ms"], js)
	}
	if v, ok := parsed["wait_require_health"].(bool); !ok || !v {
		t.Fatalf("json wait_require_health: %v want true\n%s", parsed["wait_require_health"], js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "wait_ready_interval_ms: 500") {
		t.Fatalf("text report missing wait_ready_interval_ms 500:\n%s", text)
	}
	if !strings.Contains(text, "wait_require_health: true") {
		t.Fatalf("text report missing wait_require_health true:\n%s", text)
	}

	// WaitRequireHealth configured true even when WaitReady off (configured knobs always emit).
	repOff := c.Dogfood(context.Background(), DogfoodOptions{
		SkipContext:       true,
		SkipEmit:          true,
		SkipMemory:        true,
		WaitRequireHealth: true,
	})
	if repOff.WaitReadyMS != 0 || repOff.WaitReadyIntervalMS != 0 {
		t.Fatalf("wait off knobs: ms=%d interval=%d want 0/0", repOff.WaitReadyMS, repOff.WaitReadyIntervalMS)
	}
	if !repOff.WaitRequireHealth {
		t.Fatalf("WaitRequireHealth configured true must emit even when wait off")
	}
	jsOff := FormatReportJSON(repOff)
	var parsedOff map[string]any
	if err := json.Unmarshal([]byte(jsOff), &parsedOff); err != nil {
		t.Fatalf("json off: %v\n%s", err, jsOff)
	}
	if v, ok := parsedOff["wait_require_health"].(bool); !ok || !v {
		t.Fatalf("json wait_require_health when wait off: %v want true\n%s", parsedOff["wait_require_health"], jsOff)
	}
	if n, ok := parsedOff["wait_ready_interval_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json wait_ready_interval_ms when wait off: %v want 0\n%s", parsedOff["wait_ready_interval_ms"], jsOff)
	}
}

func TestDogfood_KV_UnsetSkip(t *testing.T) {
	// Empty KVBucket → kv SKIP "kv probe unset"; kv_key_count=0; kv_bucket always-emit empty.
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
	// Always-emit empty string when unset (CI scrapers; does not invent probe success).
	v, has := parsed["kv_bucket"]
	if !has {
		t.Fatalf("json must always emit kv_bucket when unset:\n%s", js)
	}
	if str, ok := v.(string); !ok || str != "" {
		t.Fatalf("json kv_bucket: %v want empty string\n%s", v, js)
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
	if rep.KVEnsureMS != 0 {
		t.Fatalf("KVEnsureMS: %d want 0 when probe unset", rep.KVEnsureMS)
	}
	if n, ok := parsed["kv_ensure_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json kv_ensure_ms: %v want 0\n%s", parsed["kv_ensure_ms"], js)
	}
	if rep.KVListMS != 0 {
		t.Fatalf("KVListMS: %d want 0 when probe unset", rep.KVListMS)
	}
	if n, ok := parsed["kv_list_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json kv_list_ms: %v want 0\n%s", parsed["kv_list_ms"], js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "kv_key_count: 0") {
		t.Fatalf("text report missing kv_key_count:\n%s", text)
	}
	if !strings.Contains(text, "kv_ensured: false") {
		t.Fatalf("text report missing kv_ensured:\n%s", text)
	}
	if !strings.Contains(text, "kv_ensure_ms: 0") {
		t.Fatalf("text report missing kv_ensure_ms: 0:\n%s", text)
	}
	if !strings.Contains(text, "kv_list_ms: 0") {
		t.Fatalf("text report missing kv_list_ms: 0:\n%s", text)
	}
	if !strings.Contains(text, "kv_bucket: ") {
		t.Fatalf("text report must always emit kv_bucket when unset:\n%s", text)
	}
	// Empty identity must not invent probe success: step is SKIP, counts zero, flags false.
	if rep.KVKeyCount != 0 || rep.KVEnsured {
		t.Fatalf("empty kv identity must not invent success: count=%d ensured=%v", rep.KVKeyCount, rep.KVEnsured)
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
	if rep.KVEnsureMS != 0 {
		t.Fatalf("KVEnsureMS: %d want 0 when ensure off", rep.KVEnsureMS)
	}
	if n, ok := parsed["kv_ensure_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("json kv_ensure_ms: %v want 0 when ensure off\n%s", parsed["kv_ensure_ms"], js)
	}
	if rep.KVListMS < 0 {
		t.Fatalf("KVListMS: %d want >= 0 with mock list", rep.KVListMS)
	}
	if n, ok := parsed["kv_list_ms"].(float64); !ok || int(n) < 0 {
		t.Fatalf("json kv_list_ms: %v want >= 0 with mock list\n%s", parsed["kv_list_ms"], js)
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
	if !strings.Contains(text, "kv_ensure_ms: 0") {
		t.Fatalf("text report missing kv_ensure_ms: 0:\n%s", text)
	}
	if !strings.Contains(text, "kv_list_ms:") {
		t.Fatalf("text report missing kv_list_ms:\n%s", text)
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
	if rep.KVEnsureMS < 0 {
		t.Fatalf("KVEnsureMS: %d want >= 0 when ensure on", rep.KVEnsureMS)
	}
	if rep.KVMS < 0 {
		t.Fatalf("KVMS: %d want >= 0", rep.KVMS)
	}
	if rep.KVListMS < 0 {
		t.Fatalf("KVListMS: %d want >= 0 when list runs", rep.KVListMS)
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
	if n, ok := parsed["kv_ensure_ms"].(float64); !ok || int(n) < 0 {
		t.Fatalf("json kv_ensure_ms: %v want >= 0\n%s", parsed["kv_ensure_ms"], js)
	}
	if n, ok := parsed["kv_list_ms"].(float64); !ok || int(n) < 0 {
		t.Fatalf("json kv_list_ms: %v want >= 0\n%s", parsed["kv_list_ms"], js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "kv_ensured: true") {
		t.Fatalf("text missing kv_ensured true:\n%s", text)
	}
	if !strings.Contains(text, "kv_ensure_ms:") {
		t.Fatalf("text missing kv_ensure_ms:\n%s", text)
	}
	if !strings.Contains(text, "kv_list_ms:") {
		t.Fatalf("text missing kv_list_ms:\n%s", text)
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
	// Ensure was attempted (even on soft-fail) → latency always >= 0.
	if rep.KVEnsureMS < 0 {
		t.Fatalf("KVEnsureMS: %d want >= 0 on ensure soft-fail", rep.KVEnsureMS)
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
	if rep.ConsumerProbed || rep.ConsumerOK || rep.ConsumerFetchOK ||
		rep.ConsumerDeleteProbed || rep.ConsumerDeleteOK {
		t.Fatalf("ConsumerProbed=%v ConsumerOK=%v ConsumerFetchOK=%v delete_probed=%v delete_ok=%v want all false",
			rep.ConsumerProbed, rep.ConsumerOK, rep.ConsumerFetchOK,
			rep.ConsumerDeleteProbed, rep.ConsumerDeleteOK)
	}
	if rep.ConsumerStream != "" || rep.ConsumerName != "" || rep.ConsumerFilter != "" {
		t.Fatalf("identity unset: stream=%q name=%q filter=%q",
			rep.ConsumerStream, rep.ConsumerName, rep.ConsumerFilter)
	}
	step, ok := dogfoodStep(rep, "consumer")
	if !ok || step.Status != StepSkip {
		t.Fatalf("consumer step: ok=%v status=%s detail=%s", ok, step.Status, step.Detail)
	}
	if !strings.Contains(step.Detail, "consumer probe unset") {
		t.Fatalf("consumer detail: %s", step.Detail)
	}
	// Partial args → needs stream and name; no identity fields
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
	if rep2.ConsumerProbed || rep2.ConsumerOK || rep2.ConsumerDeleteProbed || rep2.ConsumerDeleteOK {
		t.Fatalf("partial probed=%v ok=%v delete_probed=%v delete_ok=%v",
			rep2.ConsumerProbed, rep2.ConsumerOK, rep2.ConsumerDeleteProbed, rep2.ConsumerDeleteOK)
	}
	if rep2.ConsumerStream != "" || rep2.ConsumerName != "" {
		t.Fatalf("partial should not set identity: stream=%q name=%q", rep2.ConsumerStream, rep2.ConsumerName)
	}

	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	for _, key := range []string{
		"consumer_probed", "consumer_ok", "consumer_fetch_ok",
		"consumer_delete_probed", "consumer_delete_ok",
	} {
		if v, ok := parsed[key].(bool); !ok || v {
			t.Fatalf("json %s: %v want false\n%s", key, parsed[key], js)
		}
	}
	// Always-emit empty strings when unset (CI scrapers; does not invent probe success).
	for _, key := range []string{"consumer_stream", "consumer_name", "consumer_filter", "pull_role", "pull_allow_suffix"} {
		v, ok := parsed[key]
		if !ok {
			t.Fatalf("json %s must always emit when unset\n%s", key, js)
		}
		str, ok := v.(string)
		if !ok || str != "" {
			t.Fatalf("json %s: %v want empty string\n%s", key, v, js)
		}
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "consumer_probed: false") ||
		!strings.Contains(text, "consumer_ok: false") ||
		!strings.Contains(text, "consumer_fetch_ok: false") ||
		!strings.Contains(text, "consumer_delete_probed: false") ||
		!strings.Contains(text, "consumer_delete_ok: false") {
		t.Fatalf("text report missing consumer fields:\n%s", text)
	}
	if !strings.Contains(text, "consumer_stream: ") ||
		!strings.Contains(text, "consumer_name: ") ||
		!strings.Contains(text, "consumer_filter: ") ||
		!strings.Contains(text, "pull_role: ") ||
		!strings.Contains(text, "pull_allow_suffix: ") {
		t.Fatalf("text must always emit consumer identity when unset:\n%s", text)
	}
	// Empty identity must not invent probe success.
	if rep.ConsumerProbed || rep.ConsumerOK {
		t.Fatalf("empty consumer identity must not invent success: probed=%v ok=%v",
			rep.ConsumerProbed, rep.ConsumerOK)
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
	if rep.ConsumerStream != "EVENTS" || rep.ConsumerName != "worker-1" || rep.ConsumerFilter != "dept.events.>" {
		t.Fatalf("identity stream=%q name=%q filter=%q", rep.ConsumerStream, rep.ConsumerName, rep.ConsumerFilter)
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
	if parsed["consumer_stream"] != "EVENTS" || parsed["consumer_name"] != "worker-1" ||
		parsed["consumer_filter"] != "dept.events.>" {
		t.Fatalf("json identity: stream=%v name=%v filter=%v\n%s",
			parsed["consumer_stream"], parsed["consumer_name"], parsed["consumer_filter"], js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "consumer_probed: true") || !strings.Contains(text, "consumer_ok: true") {
		t.Fatalf("text:\n%s", text)
	}
	if !strings.Contains(text, "consumer_stream: EVENTS") ||
		!strings.Contains(text, "consumer_name: worker-1") ||
		!strings.Contains(text, "consumer_filter: dept.events.>") {
		t.Fatalf("text missing identity:\n%s", text)
	}
}

// s687: dogfood always-emits pull_role / pull_allow_suffix from Client Config;
// empty consumer filter + role=memory → tenant.memory.> (peer mesh s686).
func TestDogfood_PullRoleIdentityAndMemoryDefaultFilter(t *testing.T) {
	var gotRole, gotSuffix, gotFilter string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case r.URL.Path == "/ready" || r.URL.Path == "/readyz":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ready"))
		case strings.HasSuffix(r.URL.Path, "/consumers") && r.Method == http.MethodPost:
			gotRole = r.Header.Get("X-IOMesh-Role")
			gotSuffix = r.Header.Get("X-IOMesh-Pull-Allow-Suffix")
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if f, ok := body["filter_subject"].(string); ok {
				gotFilter = f
			}
			w.WriteHeader(201)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"stream": "EVENTS", "name": "mem-1",
				"filter_subject": "dept.research.memory.>", "ack_floor": 0, "pending_count": 0,
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
			if strings.Contains(r.URL.Path, "/publish") {
				w.WriteHeader(204)
				return
			}
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "dept.research",
		EmitDeptStreams: true,
		Role:            "memory",
		PullAllowSuffix: "ops",
	}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{
		SkipMemory:     true,
		ConsumerStream: "EVENTS",
		ConsumerName:   "mem-1",
		// empty filter → role-aware default tenant.memory.>
	})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if gotRole != "memory" {
		t.Fatalf("X-IOMesh-Role=%q want memory", gotRole)
	}
	if gotSuffix != "ops" {
		t.Fatalf("X-IOMesh-Pull-Allow-Suffix=%q want ops", gotSuffix)
	}
	if gotFilter != "dept.research.memory.>" {
		t.Fatalf("filter_subject=%q want dept.research.memory.>", gotFilter)
	}
	if rep.PullRole != "memory" || rep.PullAllowSuffix != "ops" {
		t.Fatalf("report identity role=%q suffix=%q", rep.PullRole, rep.PullAllowSuffix)
	}
	if rep.ConsumerFilter != "dept.research.memory.>" {
		t.Fatalf("report consumer_filter=%q", rep.ConsumerFilter)
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if parsed["pull_role"] != "memory" || parsed["pull_allow_suffix"] != "ops" {
		t.Fatalf("json pull identity: role=%v suffix=%v\n%s", parsed["pull_role"], parsed["pull_allow_suffix"], js)
	}
	if parsed["consumer_filter"] != "dept.research.memory.>" {
		t.Fatalf("json consumer_filter: %v\n%s", parsed["consumer_filter"], js)
	}
	text := FormatReport(rep)
	for _, line := range []string{
		"pull_role: memory",
		"pull_allow_suffix: ops",
		"consumer_filter: dept.research.memory.>",
	} {
		if !strings.Contains(text, line) {
			t.Fatalf("text missing %q:\n%s", line, text)
		}
	}

	// Disabled mesh still always-emits empty pull identity (no invent).
	disabled := New(Config{Enabled: false}, nil)
	repOff := disabled.Dogfood(context.Background(), DogfoodOptions{SkipMemory: true})
	if repOff.PullRole != "" || repOff.PullAllowSuffix != "" {
		t.Fatalf("disabled: role=%q suffix=%q want empty", repOff.PullRole, repOff.PullAllowSuffix)
	}
	jsOff := FormatReportJSON(repOff)
	var parsedOff map[string]any
	if err := json.Unmarshal([]byte(jsOff), &parsedOff); err != nil {
		t.Fatalf("json off: %v\n%s", err, jsOff)
	}
	for _, key := range []string{"pull_role", "pull_allow_suffix"} {
		v, ok := parsedOff[key]
		if !ok {
			t.Fatalf("disabled always-emit %s missing\n%s", key, jsOff)
		}
		if s, ok := v.(string); !ok || s != "" {
			t.Fatalf("disabled %s: %v want empty string\n%s", key, v, jsOff)
		}
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
	// Identity set when stream+name provided even if create fails.
	if rep.ConsumerStream != "EVENTS" || rep.ConsumerName != "worker-1" {
		t.Fatalf("identity on soft-fail: stream=%q name=%q", rep.ConsumerStream, rep.ConsumerName)
	}
	step, ok := dogfoodStep(rep, "consumer")
	if !ok || step.Status != StepSkip {
		t.Fatalf("consumer: ok=%v status=%s detail=%s", ok, step.Status, step.Detail)
	}
	if !strings.Contains(step.Detail, "consumer soft-fail") {
		t.Fatalf("detail: %s", step.Detail)
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if parsed["consumer_stream"] != "EVENTS" || parsed["consumer_name"] != "worker-1" {
		t.Fatalf("json identity on soft-fail: %v %v\n%s", parsed["consumer_stream"], parsed["consumer_name"], js)
	}
	// Always-emit empty filter when not configured (stream+name set, filter unset).
	if v, ok := parsed["consumer_filter"]; !ok {
		t.Fatalf("json consumer_filter must always emit when empty\n%s", js)
	} else if str, ok := v.(string); !ok || str != "" {
		t.Fatalf("json consumer_filter: %v want empty string\n%s", v, js)
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

func TestDogfood_Consumer_DeleteAfterCreate(t *testing.T) {
	var createHits, deleteHits int
	var deletePath string
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
			w.WriteHeader(201)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"stream": "EVENTS", "name": "worker-1",
			})
		case strings.Contains(r.URL.Path, "/consumers/") && r.Method == http.MethodDelete:
			deleteHits++
			deletePath = r.URL.Path
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
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL, EmitDeptStreams: false}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{
		SkipMemory: true, SkipStreams: true,
		ConsumerStream: "EVENTS",
		ConsumerName:   "worker-1",
		ConsumerDelete: true,
	})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if createHits != 1 || deleteHits != 1 {
		t.Fatalf("create=%d delete=%d", createHits, deleteHits)
	}
	if !strings.Contains(deletePath, "/v1/streams/EVENTS/consumers/worker-1") {
		t.Fatalf("delete path: %s", deletePath)
	}
	if !rep.ConsumerProbed || !rep.ConsumerOK {
		t.Fatalf("probed=%v ok=%v", rep.ConsumerProbed, rep.ConsumerOK)
	}
	if !rep.ConsumerDeleteProbed || !rep.ConsumerDeleteOK {
		t.Fatalf("delete_probed=%v delete_ok=%v", rep.ConsumerDeleteProbed, rep.ConsumerDeleteOK)
	}
	if rep.ConsumerFetchOK {
		t.Fatalf("fetch_ok should be false when fetch not requested")
	}
	step, ok := dogfoodStep(rep, "consumer")
	if !ok || step.Status != StepPass {
		t.Fatalf("consumer: ok=%v status=%s detail=%s", ok, step.Status, step.Detail)
	}
	if !strings.Contains(step.Detail, "create=ok") || !strings.Contains(step.Detail, "delete=ok") {
		t.Fatalf("detail: %s", step.Detail)
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if v, ok := parsed["consumer_delete_probed"].(bool); !ok || !v {
		t.Fatalf("json consumer_delete_probed: %v\n%s", parsed["consumer_delete_probed"], js)
	}
	if v, ok := parsed["consumer_delete_ok"].(bool); !ok || !v {
		t.Fatalf("json consumer_delete_ok: %v\n%s", parsed["consumer_delete_ok"], js)
	}
	text := FormatReport(rep)
	if !strings.Contains(text, "consumer_delete_probed: true") ||
		!strings.Contains(text, "consumer_delete_ok: true") {
		t.Fatalf("text:\n%s", text)
	}
}

func TestDogfood_Consumer_DeleteFlagOff(t *testing.T) {
	var createHits, deleteHits int
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
			w.WriteHeader(201)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"stream": "EVENTS", "name": "worker-1",
			})
		case strings.Contains(r.URL.Path, "/consumers/") && r.Method == http.MethodDelete:
			deleteHits++
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
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL, EmitDeptStreams: false}, nil)
	// Create path without ConsumerDelete → delete flags stay false, no DELETE call.
	rep := c.Dogfood(context.Background(), DogfoodOptions{
		SkipMemory: true, SkipStreams: true,
		ConsumerStream: "EVENTS",
		ConsumerName:   "worker-1",
	})
	if !rep.OK {
		t.Fatalf("%s\n%s", rep.Summary, FormatReport(rep))
	}
	if createHits != 1 || deleteHits != 0 {
		t.Fatalf("create=%d delete=%d want create=1 delete=0", createHits, deleteHits)
	}
	if rep.ConsumerDeleteProbed || rep.ConsumerDeleteOK {
		t.Fatalf("delete_probed=%v delete_ok=%v want false", rep.ConsumerDeleteProbed, rep.ConsumerDeleteOK)
	}
	js := FormatReportJSON(rep)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if v, ok := parsed["consumer_delete_probed"].(bool); !ok || v {
		t.Fatalf("json consumer_delete_probed: %v want false\n%s", parsed["consumer_delete_probed"], js)
	}
	if v, ok := parsed["consumer_delete_ok"].(bool); !ok || v {
		t.Fatalf("json consumer_delete_ok: %v want false\n%s", parsed["consumer_delete_ok"], js)
	}
}

func TestDogfood_Consumer_DeleteSoftFail(t *testing.T) {
	var createHits, deleteHits int
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
			w.WriteHeader(201)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"stream": "EVENTS", "name": "worker-1",
			})
		case strings.Contains(r.URL.Path, "/consumers/") && r.Method == http.MethodDelete:
			deleteHits++
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
		ConsumerDelete: true,
	})
	if !rep.OK {
		t.Fatalf("soft delete fail should still OK: %s\n%s", rep.Summary, FormatReport(rep))
	}
	if createHits != 1 || deleteHits != 1 {
		t.Fatalf("create=%d delete=%d", createHits, deleteHits)
	}
	if !rep.ConsumerOK || !rep.ConsumerDeleteProbed || rep.ConsumerDeleteOK {
		t.Fatalf("ok=%v delete_probed=%v delete_ok=%v",
			rep.ConsumerOK, rep.ConsumerDeleteProbed, rep.ConsumerDeleteOK)
	}
	step, ok := dogfoodStep(rep, "consumer")
	if !ok || step.Status != StepSkip {
		t.Fatalf("consumer: ok=%v status=%s detail=%s", ok, step.Status, step.Detail)
	}
	if !strings.Contains(step.Detail, "delete soft-fail") {
		t.Fatalf("detail: %s", step.Detail)
	}

	// Strict → FAIL on delete error
	rep2 := c.Dogfood(context.Background(), DogfoodOptions{
		Strict: true, SkipMemory: true, SkipStreams: true,
		ConsumerStream: "EVENTS",
		ConsumerName:   "worker-1",
		ConsumerDelete: true,
	})
	if rep2.OK {
		t.Fatalf("strict delete fail should not OK:\n%s", FormatReport(rep2))
	}
	if !rep2.ConsumerDeleteProbed || rep2.ConsumerDeleteOK {
		t.Fatalf("strict delete_probed=%v delete_ok=%v", rep2.ConsumerDeleteProbed, rep2.ConsumerDeleteOK)
	}
	step2, ok2 := dogfoodStep(rep2, "consumer")
	if !ok2 || step2.Status != StepFail {
		t.Fatalf("strict consumer: ok=%v status=%s detail=%s", ok2, step2.Status, step2.Detail)
	}
}

func TestDogfood_Consumer_DeleteNotOnCreateFail(t *testing.T) {
	var deleteHits int
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
		case strings.Contains(r.URL.Path, "/consumers/") && r.Method == http.MethodDelete:
			deleteHits++
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
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL, EmitDeptStreams: false}, nil)
	rep := c.Dogfood(context.Background(), DogfoodOptions{
		SkipMemory: true, SkipStreams: true,
		ConsumerStream: "EVENTS",
		ConsumerName:   "worker-1",
		ConsumerDelete: true,
	})
	if !rep.OK {
		t.Fatalf("create soft-fail should still OK: %s\n%s", rep.Summary, FormatReport(rep))
	}
	if deleteHits != 0 {
		t.Fatalf("delete should not run when create fails: hits=%d", deleteHits)
	}
	if rep.ConsumerDeleteProbed || rep.ConsumerDeleteOK {
		t.Fatalf("delete_probed=%v delete_ok=%v want false when create fails",
			rep.ConsumerDeleteProbed, rep.ConsumerDeleteOK)
	}
}
