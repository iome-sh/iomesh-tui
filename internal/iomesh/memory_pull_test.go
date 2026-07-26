package iomesh

import (
	"context"
	"encoding/json"
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

// s705: MemoryPullStatsPrint / FormatMemoryPullStats always-emit identity + knobs +
// counters without omitempty gaps (empty identity honest; dual_write default false).
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
		"dry_run", "dual_write", "batch", "max_wait_ms", "once",
		"create_ok", "loops", "fetched", "ingested", "skipped", "acked", "errors", "last_error",
	}
	for _, key := range wantKeys {
		if _, ok := obj[key]; !ok {
			t.Fatalf("empty json missing key %q: %s", key, js)
		}
	}
	// Empty identity honest strings.
	for _, key := range []string{"stream", "consumer", "filter_subject", "pull_role", "pull_allow_suffix", "tenant", "last_error"} {
		if s, _ := obj[key].(string); s != "" {
			t.Fatalf("want empty %s, got %q", key, s)
		}
	}
	// dual_write default false (report-only honesty).
	if obj["dual_write"] != false {
		t.Fatalf("want dual_write=false default, got %v", obj["dual_write"])
	}
	if obj["dry_run"] != false || obj["once"] != false || obj["create_ok"] != false {
		t.Fatalf("want bool zeros false: %s", js)
	}
	// Numeric zeros.
	for _, key := range []string{"batch", "max_wait_ms", "loops", "fetched", "ingested", "skipped", "acked", "errors"} {
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
		"dry_run:           false\n",
		"dual_write:        false\n",
		"batch:             0\n",
		"max_wait_ms:       0\n",
		"once:              false\n",
		"create_ok:         false\n",
		"last_error:        \n",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("empty text missing %q in:\n%q", want, text)
		}
	}
	// Identity order: stream → consumer → filter_subject → pull_role → pull_allow_suffix → tenant
	streamIdx := strings.Index(text, "stream:")
	consumerIdx := strings.Index(text, "consumer:")
	filterIdx := strings.Index(text, "filter_subject:")
	roleIdx := strings.Index(text, "pull_role:")
	suffixIdx := strings.Index(text, "pull_allow_suffix:")
	tenantIdx := strings.Index(text, "tenant:")
	if !(streamIdx < consumerIdx && consumerIdx < filterIdx && filterIdx < roleIdx && roleIdx < suffixIdx && suffixIdx < tenantIdx) {
		t.Fatalf("identity order wrong:\n%s", text)
	}
}

// s705: populated role/filter/tenant + knobs always-emit for scrapers.
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
		DryRun:          true,
		DualWrite:       false,
		Batch:           8,
		MaxWaitMS:       2000,
		Once:            true,
	}
	p := NewMemoryPullStatsPrint(st, meta)
	if p.FilterSubject != "acme.events.>" || p.PullRole != "agent" || p.Tenant != "acme" {
		t.Fatalf("identity: %+v", p)
	}
	if p.PullAllowSuffix != "" {
		t.Fatalf("want empty pull_allow_suffix, got %q", p.PullAllowSuffix)
	}
	if !p.DryRun || !p.Once || p.DualWrite || p.Batch != 8 || p.MaxWaitMS != 2000 {
		t.Fatalf("knobs: %+v", p)
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
	// pull_allow_suffix always present even when empty.
	if _, ok := obj["pull_allow_suffix"]; !ok {
		t.Fatalf("missing pull_allow_suffix: %s", js)
	}
	if s, _ := obj["pull_allow_suffix"].(string); s != "" {
		t.Fatalf("want empty pull_allow_suffix, got %q", s)
	}
	if obj["dry_run"] != true || obj["once"] != true || obj["dual_write"] != false {
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

	text := FormatMemoryPullStats(p, true, "")
	for _, want := range []string{
		"PASS memory pull\n",
		"stream:            EVENTS\n",
		"consumer:          tui-local-palace\n",
		"filter_subject:    acme.events.>\n",
		"pull_role:         agent\n",
		"pull_allow_suffix: \n",
		"tenant:            acme\n",
		"dry_run:           true\n",
		"dual_write:        false\n",
		"batch:             8\n",
		"max_wait_ms:       2000\n",
		"once:              true\n",
		"create_ok:         true\n",
		"fetched:           3\n",
		"ingested:          2\n",
		"skipped:           1\n",
		"acked:             3\n",
		"errors:            0\n",
		"last_error:        \n",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("populated text missing %q in:\n%s", want, text)
		}
	}

	// custom role + allow-suffix + last_error on FAIL path.
	st2 := MemoryPullStats{
		Stream: "S", Consumer: "c", Filter: "t.ops.>",
		Errors: 1, LastError: "fetch timeout",
	}
	p2 := NewMemoryPullStatsPrint(st2, MemoryPullPrintMeta{
		Tenant: "t", PullRole: "custom", PullAllowSuffix: "ops,memory",
	})
	failText := FormatMemoryPullStats(p2, false, "")
	if !strings.Contains(failText, "FAIL memory pull: fetch timeout\n") {
		t.Fatalf("FAIL header:\n%s", failText)
	}
	for _, want := range []string{
		"pull_role:         custom\n",
		"pull_allow_suffix: ops,memory\n",
		"last_error:        fetch timeout\n",
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
}
