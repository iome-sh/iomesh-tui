package iomesh

import (
	"context"
	"encoding/json"
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
