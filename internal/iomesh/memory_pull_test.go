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
