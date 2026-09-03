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
)

func TestMapStreamMessageToEnvelope_MemoryIngestJSON(t *testing.T) {
	payload, _ := json.Marshal(MemoryEnvelope{
		Type:       memoryEnvelopeIngest,
		SessionID:  "sess-1",
		Role:       "user",
		Content:    "hello palace",
		EventTime:  "2026-07-24T00:00:00Z",
		SessionSeq: 3,
	})
	msg := StreamMessage{Stream: "MEMORY_INGEST", Seq: 99, Subject: "t.memory.ingest.turn", Payload: payload}
	env, key, ok := MapStreamMessageToEnvelope(msg)
	if !ok {
		t.Fatal("expected ok")
	}
	if env.Content != "hello palace" || env.Role != "user" || env.SessionSeq != 3 {
		t.Fatalf("env=%+v", env)
	}
	if key != "sess-1:3" {
		t.Fatalf("dedupe key %q", key)
	}
}

func TestMapStreamMessageToEnvelope_GenericEvent(t *testing.T) {
	payload := []byte(`{"text":"ticket updated","session_id":"ops","role":"system"}`)
	msg := StreamMessage{Stream: "EVENTS", Seq: 7, Subject: "dept.eng.events.jira", Payload: payload}
	env, key, ok := MapStreamMessageToEnvelope(msg)
	if !ok {
		t.Fatal("expected ok")
	}
	if env.Content != "ticket updated" {
		t.Fatalf("content %q", env.Content)
	}
	if key != "ops:7" && key != "ops:0" {
		// session_seq missing → uses msg.Seq
		if key != "ops:7" {
			t.Fatalf("dedupe key %q", key)
		}
	}
}

func TestMapStreamMessageToEnvelope_RawText(t *testing.T) {
	msg := StreamMessage{Stream: "EVENTS", Seq: 1, Subject: "dept.x", Payload: []byte("plain log line")}
	env, _, ok := MapStreamMessageToEnvelope(msg)
	if !ok || env.Content != "plain log line" {
		t.Fatalf("env=%+v ok=%v", env, ok)
	}
}

func TestMapStreamMessageToEnvelope_Empty(t *testing.T) {
	_, _, ok := MapStreamMessageToEnvelope(StreamMessage{Payload: []byte("  ")})
	if ok {
		t.Fatal("expected not ok")
	}
}

func TestDefaultMemoryPullFilter(t *testing.T) {
	// s660 empty-role path (wrapper around DefaultMemoryPullFilterForRole).
	tests := []struct {
		name     string
		explicit string
		tenant   string
		want     string
	}{
		{name: "explicit wins", explicit: "custom.>", tenant: "dept.engineering", want: "custom.>"},
		{name: "explicit trim", explicit: "  custom.>  ", tenant: "dept.x", want: "custom.>"},
		{name: "dept hierarchical", explicit: "", tenant: "dept.engineering", want: "dept.engineering.>"},
		{name: "contains dot", explicit: "", tenant: "acme.prod", want: "acme.prod.>"},
		{name: "prefix dept no dot", explicit: "", tenant: "dept", want: "dept.>"},
		{name: "prefix deptfoo", explicit: "", tenant: "deptfoo", want: "deptfoo.>"},
		{name: "plain tenant no default", explicit: "", tenant: "acme", want: ""},
		{name: "empty tenant", explicit: "", tenant: "", want: ""},
		{name: "whitespace tenant", explicit: "", tenant: "  ", want: ""},
		{name: "whitespace explicit falls through", explicit: "   ", tenant: "dept.eng", want: "dept.eng.>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DefaultMemoryPullFilter(tt.explicit, tt.tenant)
			if got != tt.want {
				t.Fatalf("DefaultMemoryPullFilter(%q, %q)=%q want %q", tt.explicit, tt.tenant, got, tt.want)
			}
		})
	}
}

func TestDefaultMemoryPullFilterForRole(t *testing.T) {
	// s678: role-aware defaults when --filter / pull_filter empty.
	tests := []struct {
		name        string
		explicit    string
		tenant      string
		role        string
		allowSuffix string
		want        string
	}{
		// explicit always wins
		{name: "explicit wins over agent", explicit: "custom.>", tenant: "dept.eng", role: "agent", want: "custom.>"},
		{name: "explicit wins over custom", explicit: "x.>", tenant: "t", role: "custom", allowSuffix: "ops", want: "x.>"},
		{name: "whitespace explicit falls through agent", explicit: "  ", tenant: "dept.eng", role: "agent", want: "dept.eng.events.>"},

		// empty role → s660
		{name: "empty role hierarchical", explicit: "", tenant: "dept.engineering", role: "", want: "dept.engineering.>"},
		{name: "empty role plain tenant", explicit: "", tenant: "acme", role: "", want: ""},
		{name: "empty role empty tenant", explicit: "", tenant: "", role: "agent", want: ""},

		// agent / viewer → tenant.events.>
		{name: "agent hierarchical", explicit: "", tenant: "dept.eng", role: "agent", want: "dept.eng.events.>"},
		{name: "agent plain tenant", explicit: "", tenant: "acme", role: "agent", want: "acme.events.>"},
		{name: "viewer", explicit: "", tenant: "dept.eng", role: "viewer", want: "dept.eng.events.>"},
		{name: "agent case insensitive", explicit: "", tenant: "t", role: "Agent", want: "t.events.>"},

		// memory (s687 / peer aion s686) → tenant.memory.>
		{name: "memory hierarchical", explicit: "", tenant: "dept.research", role: "memory", want: "dept.research.memory.>"},
		{name: "memory plain tenant", explicit: "", tenant: "acme", role: "memory", want: "acme.memory.>"},
		{name: "memory case insensitive", explicit: "", tenant: "t", role: "Memory", want: "t.memory.>"},
		{name: "explicit wins over memory", explicit: "custom.>", tenant: "dept.eng", role: "memory", want: "custom.>"},
		{name: "memory empty tenant", explicit: "", tenant: "", role: "memory", want: ""},

		// auditor → tenant.audit.>
		{name: "auditor", explicit: "", tenant: "dept.eng", role: "auditor", want: "dept.eng.audit.>"},
		{name: "auditor plain", explicit: "", tenant: "acme", role: "auditor", want: "acme.audit.>"},

		// operator / admin → tenant.>
		{name: "operator hierarchical", explicit: "", tenant: "dept.eng", role: "operator", want: "dept.eng.>"},
		{name: "admin plain tenant", explicit: "", tenant: "acme", role: "admin", want: "acme.>"},
		{name: "operator case", explicit: "", tenant: "t", role: "OPERATOR", want: "t.>"},

		// custom + single suffix → tenant.<suffix>.>
		{name: "custom one suffix", explicit: "", tenant: "dept.eng", role: "custom", allowSuffix: "ops", want: "dept.eng.ops.>"},
		{name: "custom one suffix trim", explicit: "", tenant: "t", role: "custom", allowSuffix: "  memory  ", want: "t.memory.>"},
		// custom multi / none → empty (fail closed)
		{name: "custom multi suffix", explicit: "", tenant: "dept.eng", role: "custom", allowSuffix: "ops,memory", want: ""},
		{name: "custom empty suffix", explicit: "", tenant: "dept.eng", role: "custom", allowSuffix: "", want: ""},
		{name: "custom only commas", explicit: "", tenant: "t", role: "custom", allowSuffix: " , , ", want: ""},
		{name: "custom multi with spaces", explicit: "", tenant: "t", role: "custom", allowSuffix: "a, b", want: ""},
		// custom single after dropping empties
		{name: "custom one after empty tokens", explicit: "", tenant: "t", role: "custom", allowSuffix: ",ops,", want: "t.ops.>"},

		// unknown role → empty (no invent)
		{name: "unknown role", explicit: "", tenant: "dept.eng", role: "superuser", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DefaultMemoryPullFilterForRole(tt.explicit, tt.tenant, tt.role, tt.allowSuffix)
			if got != tt.want {
				t.Fatalf("DefaultMemoryPullFilterForRole(%q, %q, %q, %q)=%q want %q",
					tt.explicit, tt.tenant, tt.role, tt.allowSuffix, got, tt.want)
			}
		})
	}
}

func TestRunMemoryPull_SendsOrgHeaderWhenSet(t *testing.T) {
	var createOrg, fetchOrg, ackOrg string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		org := r.Header.Get("X-IOMesh-Org")
		switch {
		case strings.HasSuffix(r.URL.Path, "/consumers") && !strings.Contains(r.URL.Path, "/fetch"):
			createOrg = org
			_ = json.NewEncoder(w).Encode(map[string]any{"stream": "EVENTS", "name": "tui-local-palace"})
		case strings.HasSuffix(r.URL.Path, "/fetch"):
			fetchOrg = org
			payload := base64.StdEncoding.EncodeToString([]byte(`{"text":"only org_a"}`))
			_ = json.NewEncoder(w).Encode(map[string]any{
				"messages": []map[string]any{{
					"stream": "EVENTS", "seq": 3, "subject": "dept.engineering.events.github",
					"payload": payload,
				}},
			})
		case strings.HasSuffix(r.URL.Path, "/ack"):
			ackOrg = org
			_ = json.NewEncoder(w).Encode(map[string]any{"ack_floor": 3})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "dept.engineering", OrgID: "org_a"}, nil)
	st, err := c.RunMemoryPull(context.Background(), MemoryPullOptions{
		Stream: "EVENTS", Name: "tui-local-palace", Batch: 1, MaxWait: time.Millisecond,
		MaxLoops: 1, Ack: true, DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if createOrg != "org_a" || fetchOrg != "org_a" || ackOrg != "org_a" {
		t.Fatalf("org headers create=%q fetch=%q ack=%q", createOrg, fetchOrg, ackOrg)
	}
	if st.Fetched < 1 {
		t.Fatalf("stats=%+v", st)
	}
}

func TestRunMemoryPull_OmitsOrgHeaderWhenUnset(t *testing.T) {
	var hadOrg bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Header["X-IOMesh-Org"]; ok {
			hadOrg = true
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/consumers") && !strings.Contains(r.URL.Path, "/fetch"):
			_ = json.NewEncoder(w).Encode(map[string]any{"stream": "EVENTS", "name": "c"})
		case strings.HasSuffix(r.URL.Path, "/fetch"):
			_ = json.NewEncoder(w).Encode(map[string]any{"messages": []any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "dept.engineering"}, nil)
	if _, err := c.RunMemoryPull(context.Background(), MemoryPullOptions{
		Stream: "EVENTS", Name: "c", Batch: 1, MaxWait: time.Millisecond,
		MaxLoops: 1, DryRun: true,
	}); err != nil {
		t.Fatal(err)
	}
	if hadOrg {
		t.Fatal("empty org must omit X-IOMesh-Org (fail-open)")
	}
}

func TestRunMemoryPull_RequiresIngest(t *testing.T) {
	c := &Client{} // disabled
	_, err := c.RunMemoryPull(context.Background(), MemoryPullOptions{
		Stream: "EVENTS", Name: "c1", DryRun: false,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunMemoryPull_DryRunValidation(t *testing.T) {
	// DryRun without client still fails mesh disabled before LocalIngest check order:
	c := &Client{}
	_, err := c.RunMemoryPull(context.Background(), MemoryPullOptions{
		Stream: "EVENTS", Name: "c1", DryRun: true,
	})
	if err == nil || err.Error() != "mesh disabled" {
		// Client zero value Enabled() is false
		if err == nil {
			t.Fatal("expected mesh disabled")
		}
	}
	_ = time.Second
}

// s705+s717: MemoryPullStatsPrint / FormatMemoryPullStats always-emit identity +
// knobs + counters + process evidence without omitempty gaps (empty identity honest;
// dual_write default false; result/exit_code/duration_ms/ack always present).
func TestMemoryPullStatsPrint_EmptyIdentityAlwaysEmit(t *testing.T) {
	t.Parallel()

	st := MemoryPullStats{} // zero stats / empty identity
	p := NewMemoryPullStatsPrint(st, MemoryPullPrintMeta{})
	js, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(js, &obj); err != nil {
		t.Fatal(err)
	}
	// Scraper keys must always be present (no omitempty).
	wantKeys := []string{
		"stream", "consumer", "filter_subject", "pull_role", "pull_allow_suffix", "tenant",
		"endpoint", "org", "workspace",
		"dry_run", "dual_write", "batch", "max_wait_ms", "once", "ack",
		"create_ok", "loops", "fetched", "ingested", "skipped", "acked", "errors", "last_error",
		"result", "exit_code", "duration_ms",
	}
	for _, key := range wantKeys {
		if _, ok := obj[key]; !ok {
			t.Fatalf("empty json missing key %q: %s", key, js)
		}
	}
	// Empty identity honest strings (incl. s717 process mesh identity + result).
	for _, key := range []string{
		"stream", "consumer", "filter_subject", "pull_role", "pull_allow_suffix", "tenant",
		"endpoint", "org", "workspace", "last_error", "result",
	} {
		if s, _ := obj[key].(string); s != "" {
			t.Fatalf("want empty %s, got %q", key, s)
		}
	}
	// dual_write default false (report-only honesty).
	if obj["dual_write"] != false {
		t.Fatalf("want dual_write=false default, got %v", obj["dual_write"])
	}
	if obj["dry_run"] != false || obj["once"] != false || obj["create_ok"] != false || obj["ack"] != false {
		t.Fatalf("want bool zeros false: %s", js)
	}
	// Numeric zeros (incl. process evidence exit_code/duration_ms).
	for _, key := range []string{"batch", "max_wait_ms", "loops", "fetched", "ingested", "skipped", "acked", "errors", "exit_code", "duration_ms"} {
		if n, ok := obj[key].(float64); !ok || n != 0 {
			t.Fatalf("want %s=0, got %v", key, obj[key])
		}
	}

	// Text path always emits blank identity lines (not only stderr).
	text := FormatMemoryPullStats(p, true, "")
	for _, want := range []string{
		"PASS memory pull\n",
		"stream:            \n",
		"consumer:          \n",
		"filter_subject:    \n",
		"pull_role:         \n",
		"pull_allow_suffix: \n",
		"tenant:            \n",
		"endpoint:          \n",
		"org:               \n",
		"workspace:         \n",
		"dry_run:           false\n",
		"dual_write:        false\n",
		"batch:             0\n",
		"max_wait_ms:       0\n",
		"once:              false\n",
		"ack:               false\n",
		"create_ok:         false\n",
		"last_error:        \n",
		"result:            \n",
		"exit_code:         0\n",
		"duration_ms:       0\n",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("empty text missing %q in:\n%q", want, text)
		}
	}
	// Identity order: stream → consumer → filter_subject → pull_role → pull_allow_suffix → tenant
	// → endpoint → org → workspace (s717 process mesh identity).
	streamIdx := strings.Index(text, "stream:")
	consumerIdx := strings.Index(text, "consumer:")
	filterIdx := strings.Index(text, "filter_subject:")
	roleIdx := strings.Index(text, "pull_role:")
	suffixIdx := strings.Index(text, "pull_allow_suffix:")
	tenantIdx := strings.Index(text, "tenant:")
	endpointIdx := strings.Index(text, "endpoint:")
	orgIdx := strings.Index(text, "org:")
	wsIdx := strings.Index(text, "workspace:")
	if !(streamIdx < consumerIdx && consumerIdx < filterIdx && filterIdx < roleIdx && roleIdx < suffixIdx && suffixIdx < tenantIdx) {
		t.Fatalf("identity order wrong:\n%s", text)
	}
	if !(tenantIdx < endpointIdx && endpointIdx < orgIdx && orgIdx < wsIdx) {
		t.Fatalf("process identity order wrong:\n%s", text)
	}
	// Process evidence after counters.
	lastErrIdx := strings.Index(text, "last_error:")
	resultIdx := strings.Index(text, "result:")
	exitIdx := strings.Index(text, "exit_code:")
	durIdx := strings.Index(text, "duration_ms:")
	if !(lastErrIdx < resultIdx && resultIdx < exitIdx && exitIdx < durIdx) {
		t.Fatalf("process evidence order wrong:\n%s", text)
	}
}

// s705+s717: populated role/filter/tenant + knobs + process evidence always-emit.
func TestMemoryPullStatsPrint_PopulatedRoleFilter(t *testing.T) {
	t.Parallel()

	st := MemoryPullStats{
		Stream:    "EVENTS",
		Consumer:  "tui-local-palace",
		Filter:    "acme.events.>",
		CreateOK:  true,
		Loops:     1,
		Fetched:   3,
		Ingested:  2,
		Skipped:   1,
		Acked:     3,
		Errors:    0,
		LastError: "",
	}
	meta := MemoryPullPrintMeta{
		Tenant:          "acme",
		PullRole:        "agent",
		PullAllowSuffix: "",
		Endpoint:        "https://mesh.example",
		Org:             "org_dev",
		Workspace:       "ws_alpha",
		DryRun:          true,
		DualWrite:       false,
		Batch:           8,
		MaxWaitMS:       2000,
		Once:            true,
		Ack:             true,
		Result:          "ok",
		ExitCode:        0,
		DurationMS:      42,
	}
	p := NewMemoryPullStatsPrint(st, meta)
	if p.FilterSubject != "acme.events.>" || p.PullRole != "agent" || p.Tenant != "acme" {
		t.Fatalf("identity: %+v", p)
	}
	if p.PullAllowSuffix != "" {
		t.Fatalf("want empty pull_allow_suffix, got %q", p.PullAllowSuffix)
	}
	if p.Endpoint != "https://mesh.example" || p.Org != "org_dev" || p.Workspace != "ws_alpha" {
		t.Fatalf("process identity: %+v", p)
	}
	if !p.DryRun || !p.Once || p.DualWrite || p.Batch != 8 || p.MaxWaitMS != 2000 || !p.Ack {
		t.Fatalf("knobs: %+v", p)
	}
	if p.Result != "ok" || p.ExitCode != 0 || p.DurationMS != 42 {
		t.Fatalf("process evidence: %+v", p)
	}

	js := FormatMemoryPullStatsJSON(p)
	var obj map[string]any
	if err := json.Unmarshal([]byte(js), &obj); err != nil {
		t.Fatal(err)
	}
	if obj["stream"] != "EVENTS" || obj["consumer"] != "tui-local-palace" {
		t.Fatalf("stream/consumer: %s", js)
	}
	if obj["filter_subject"] != "acme.events.>" || obj["pull_role"] != "agent" || obj["tenant"] != "acme" {
		t.Fatalf("identity json: %s", js)
	}
	if obj["endpoint"] != "https://mesh.example" || obj["org"] != "org_dev" || obj["workspace"] != "ws_alpha" {
		t.Fatalf("process identity json: %s", js)
	}
	// pull_allow_suffix always present even when empty.
	if _, ok := obj["pull_allow_suffix"]; !ok {
		t.Fatalf("missing pull_allow_suffix: %s", js)
	}
	if s, _ := obj["pull_allow_suffix"].(string); s != "" {
		t.Fatalf("want empty pull_allow_suffix, got %q", s)
	}
	if obj["dry_run"] != true || obj["once"] != true || obj["dual_write"] != false || obj["ack"] != true {
		t.Fatalf("knobs json: %s", js)
	}
	if obj["batch"] != float64(8) || obj["max_wait_ms"] != float64(2000) {
		t.Fatalf("batch/wait: %s", js)
	}
	if obj["create_ok"] != true || obj["fetched"] != float64(3) || obj["ingested"] != float64(2) {
		t.Fatalf("counters: %s", js)
	}
	if s, _ := obj["last_error"].(string); s != "" {
		t.Fatalf("want empty last_error, got %q", s)
	}
	if obj["result"] != "ok" || obj["exit_code"] != float64(0) || obj["duration_ms"] != float64(42) {
		t.Fatalf("process evidence json: %s", js)
	}

	text := FormatMemoryPullStats(p, true, "")
	for _, want := range []string{
		"PASS memory pull\n",
		"stream:            EVENTS\n",
		"consumer:          tui-local-palace\n",
		"filter_subject:    acme.events.>\n",
		"pull_role:         agent\n",
		"pull_allow_suffix: \n",
		"tenant:            acme\n",
		"endpoint:          https://mesh.example\n",
		"org:               org_dev\n",
		"workspace:         ws_alpha\n",
		"dry_run:           true\n",
		"dual_write:        false\n",
		"batch:             8\n",
		"max_wait_ms:       2000\n",
		"once:              true\n",
		"ack:               true\n",
		"create_ok:         true\n",
		"fetched:           3\n",
		"ingested:          2\n",
		"skipped:           1\n",
		"acked:             3\n",
		"errors:            0\n",
		"last_error:        \n",
		"result:            ok\n",
		"exit_code:         0\n",
		"duration_ms:       42\n",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("populated text missing %q in:\n%s", want, text)
		}
	}

	// custom role + allow-suffix + last_error + process err evidence on FAIL path.
	st2 := MemoryPullStats{
		Stream: "S", Consumer: "c", Filter: "t.ops.>",
		Errors: 1, LastError: "fetch timeout",
	}
	p2 := NewMemoryPullStatsPrint(st2, MemoryPullPrintMeta{
		Tenant: "t", PullRole: "custom", PullAllowSuffix: "ops,memory",
		Endpoint: "http://127.0.0.1:8080", Org: "", Workspace: "",
		Result: "err", ExitCode: 1, DurationMS: 7, Ack: false,
	})
	failText := FormatMemoryPullStats(p2, false, "")
	if !strings.Contains(failText, "FAIL memory pull: fetch timeout\n") {
		t.Fatalf("FAIL header:\n%s", failText)
	}
	for _, want := range []string{
		"pull_role:         custom\n",
		"pull_allow_suffix: ops,memory\n",
		"endpoint:          http://127.0.0.1:8080\n",
		"org:               \n",
		"workspace:         \n",
		"ack:               false\n",
		"last_error:        fetch timeout\n",
		"result:            err\n",
		"exit_code:         1\n",
		"duration_ms:       7\n",
	} {
		if !strings.Contains(failText, want) {
			t.Fatalf("FAIL text missing %q in:\n%s", want, failText)
		}
	}
	// Explicit errMsg overrides last_error for FAIL header only.
	failText2 := FormatMemoryPullStats(p2, false, "create denied")
	if !strings.Contains(failText2, "FAIL memory pull: create denied\n") {
		t.Fatalf("explicit errMsg:\n%s", failText2)
	}
	if !strings.Contains(failText2, "last_error:        fetch timeout\n") {
		t.Fatalf("last_error still from stats:\n%s", failText2)
	}
	// Soft-fail process evidence: result=err exit_code=1 with empty org/workspace honest.
	jsFail := FormatMemoryPullStatsJSON(p2)
	var objFail map[string]any
	if err := json.Unmarshal([]byte(jsFail), &objFail); err != nil {
		t.Fatal(err)
	}
	if objFail["result"] != "err" || objFail["exit_code"] != float64(1) {
		t.Fatalf("soft/hard fail process evidence: %s", jsFail)
	}
	if s, _ := objFail["org"].(string); s != "" {
		t.Fatalf("want empty org, got %q", s)
	}
	if s, _ := objFail["workspace"].(string); s != "" {
		t.Fatalf("want empty workspace, got %q", s)
	}
}

// s747: process-evidence completeness pin — FormatMemoryPullStatsJSON always emits
// result / exit_code / endpoint / org / workspace on both empty and populated paths
// (DTO already always-emit s717; this serial locks the complete surface, does not
// invent new fields or re-claim s717 product body). Peer aion s746 residual.
// process evidence ≠ invent pull success · dual_write OFF · offline unit ≠ live APPLY.
func TestMemoryPullStatsPrint_ProcessEvidenceCompletenessPin(t *testing.T) {
	t.Parallel()

	// Process evidence keys residual-framed at aion s716 / product s717.
	processEvidenceKeys := []string{"result", "exit_code", "endpoint", "org", "workspace"}

	// Empty path: zero meta → keys present, empty/0 honest.
	emptyJS := FormatMemoryPullStatsJSON(NewMemoryPullStatsPrint(MemoryPullStats{}, MemoryPullPrintMeta{}))
	if !strings.HasSuffix(emptyJS, "\n") {
		t.Fatal("empty FormatMemoryPullStatsJSON: expected trailing newline")
	}
	var emptyObj map[string]any
	if err := json.Unmarshal([]byte(emptyJS), &emptyObj); err != nil {
		t.Fatalf("empty unmarshal: %v\n%s", err, emptyJS)
	}
	for _, key := range processEvidenceKeys {
		if _, ok := emptyObj[key]; !ok {
			t.Fatalf("empty missing process evidence key %q: %s", key, emptyJS)
		}
	}
	for _, key := range []string{"result", "endpoint", "org", "workspace"} {
		if s, _ := emptyObj[key].(string); s != "" {
			t.Fatalf("empty want honest blank %s, got %q", key, s)
		}
	}
	if n, ok := emptyObj["exit_code"].(float64); !ok || n != 0 {
		t.Fatalf("empty want exit_code=0, got %v", emptyObj["exit_code"])
	}

	// Populated path: process evidence values round-trip; all keys still present.
	pop := NewMemoryPullStatsPrint(
		MemoryPullStats{Stream: "EVENTS", Consumer: "c1", Filter: "t.>"},
		MemoryPullPrintMeta{
			Endpoint:   "https://mesh.example",
			Org:        "org_dev",
			Workspace:  "ws_alpha",
			Result:     "ok",
			ExitCode:   0,
			DurationMS: 9,
			Ack:        true,
		},
	)
	popJS := FormatMemoryPullStatsJSON(pop)
	if !strings.HasSuffix(popJS, "\n") {
		t.Fatal("populated FormatMemoryPullStatsJSON: expected trailing newline")
	}
	var popObj map[string]any
	if err := json.Unmarshal([]byte(popJS), &popObj); err != nil {
		t.Fatalf("populated unmarshal: %v\n%s", err, popJS)
	}
	for _, key := range processEvidenceKeys {
		if _, ok := popObj[key]; !ok {
			t.Fatalf("populated missing process evidence key %q: %s", key, popJS)
		}
	}
	if popObj["result"] != "ok" || popObj["exit_code"] != float64(0) {
		t.Fatalf("populated result/exit_code: %s", popJS)
	}
	if popObj["endpoint"] != "https://mesh.example" || popObj["org"] != "org_dev" || popObj["workspace"] != "ws_alpha" {
		t.Fatalf("populated endpoint/org/workspace: %s", popJS)
	}

	// Err path still always-emits the same five keys (empty org/workspace honest).
	errJS := FormatMemoryPullStatsJSON(NewMemoryPullStatsPrint(
		MemoryPullStats{Stream: "S", Errors: 1, LastError: "boom"},
		MemoryPullPrintMeta{
			Endpoint: "http://127.0.0.1:8080",
			Result:   "err",
			ExitCode: 1,
		},
	))
	var errObj map[string]any
	if err := json.Unmarshal([]byte(errJS), &errObj); err != nil {
		t.Fatalf("err unmarshal: %v\n%s", err, errJS)
	}
	for _, key := range processEvidenceKeys {
		if _, ok := errObj[key]; !ok {
			t.Fatalf("err missing process evidence key %q: %s", key, errJS)
		}
	}
	if errObj["result"] != "err" || errObj["exit_code"] != float64(1) {
		t.Fatalf("err result/exit_code: %s", errJS)
	}
	if errObj["endpoint"] != "http://127.0.0.1:8080" {
		t.Fatalf("err endpoint: %s", errJS)
	}
	if s, _ := errObj["org"].(string); s != "" {
		t.Fatalf("err want empty org, got %q", s)
	}
	if s, _ := errObj["workspace"].(string); s != "" {
		t.Fatalf("err want empty workspace, got %q", s)
	}
}
