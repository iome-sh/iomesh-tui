package iomesh

import (
	"encoding/json"
	"testing"
)

func TestMemoryOpsDigestReceipt_UnmarshalProvenanceAndTags(t *testing.T) {
	raw := `{
		"id": "m1",
		"event_time": "2026-09-10T06:46:00Z",
		"summary": "dept pull",
		"source_hint": "palace_timeline",
		"tags": ["source_hint:mesh", {"name": "origin:broker"}],
		"provenance": {"source_hint": "mesh"}
	}`
	var r MemoryOpsDigestReceipt
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatal(err)
	}
	if r.SourceHint != "palace_timeline" {
		t.Fatalf("source_hint=%q", r.SourceHint)
	}
	if r.Provenance.SourceHint != "mesh" {
		t.Fatalf("provenance.source_hint=%q", r.Provenance.SourceHint)
	}
	if len(r.Tags) != 2 || r.Tags[0] != "source_hint:mesh" || r.Tags[1] != "origin:broker" {
		t.Fatalf("tags=%v", r.Tags)
	}
}

func TestMemoryOpsDigestReceipt_UnmarshalProvenanceStringAndSourceAlias(t *testing.T) {
	raw := `{"id":"m2","source":"palace_timeline","provenance":"mesh","tags":["source_hint:mesh"]}`
	var r MemoryOpsDigestReceipt
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatal(err)
	}
	if r.SourceHint != "palace_timeline" {
		t.Fatalf("source alias=%q", r.SourceHint)
	}
	if r.Provenance.SourceHint != "mesh" {
		t.Fatalf("provenance string=%q", r.Provenance.SourceHint)
	}
	if len(r.Tags) != 1 || r.Tags[0] != "source_hint:mesh" {
		t.Fatalf("tags=%v", r.Tags)
	}
}

func TestMemoryOpsDigestReceipt_UnmarshalPlainSourceHint(t *testing.T) {
	raw := `{"id":"p1","event_time":"2026-08-04T10:00:00Z","summary":"deploy finished","source_hint":"palace_timeline"}`
	var r MemoryOpsDigestReceipt
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatal(err)
	}
	if r.SourceHint != "palace_timeline" || r.Provenance.SourceHint != "" || len(r.Tags) != 0 {
		t.Fatalf("plain receipt changed: %+v", r)
	}
}
