package iomesh

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestListStreams_OKAndUserAgent(t *testing.T) {
	prev := UserAgent()
	SetUserAgent("iomesh-tui/test-s298")
	t.Cleanup(func() { SetUserAgent(prev) })

	var gotUA, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotUA = r.Header.Get("User-Agent")
		if r.URL.Path != "/v1/streams" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"name":       "EVENTS",
				"subjects":   []string{"dept.events.>"},
				"messages":   3,
				"first_seq":  1,
				"last_seq":   3,
				"created_at": time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
			},
		})
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "acme"}, nil)
	streams, err := c.ListStreams(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("method=%q", gotMethod)
	}
	if gotUA != "iomesh-tui/test-s298" {
		t.Fatalf("User-Agent=%q", gotUA)
	}
	if len(streams) != 1 || streams[0].Name != "EVENTS" {
		t.Fatalf("streams=%+v", streams)
	}
	if streams[0].Messages != 3 || streams[0].LastSeq != 3 {
		t.Fatalf("stats=%+v", streams[0])
	}
	out := FormatStreams(streams)
	if !strings.Contains(out, "EVENTS") || !strings.Contains(out, "count=1") {
		t.Fatal(out)
	}
}

func TestListStreams_Envelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"streams": []map[string]any{
				{"name": "KV", "subjects": []string{"kv.>"}, "messages": 0, "first_seq": 0, "last_seq": 0},
			},
		})
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	streams, err := c.ListStreams(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 1 || streams[0].Name != "KV" {
		t.Fatalf("streams=%+v", streams)
	}
}

func TestGetStream_OK(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method != http.MethodGet || r.URL.Path != "/v1/streams/EVENTS" {
			http.NotFound(w, r)
			return
		}
		max := int64(1000)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":        "EVENTS",
			"subjects":    []string{"dept.events.>"},
			"retention":   "limits",
			"partitions":  1,
			"max_msgs":    max,
			"description": "ops events",
			"messages":    10,
			"first_seq":   1,
			"last_seq":    10,
			"created_at":  time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
		})
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	info, err := c.GetStream(context.Background(), "EVENTS")
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/streams/EVENTS" {
		t.Fatalf("path=%q", gotPath)
	}
	if info == nil || info.Name != "EVENTS" || info.LastSeq != 10 {
		t.Fatalf("info=%+v", info)
	}
	detail := FormatStreamDetail(*info)
	if !strings.Contains(detail, "ops events") || !strings.Contains(detail, "dept.events.>") {
		t.Fatal(detail)
	}
}

func TestGetStream_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"stream not found"}`))
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	_, err := c.GetStream(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "http 404") {
		t.Fatalf("err=%v", err)
	}
}

func TestListStreams_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	streams, err := c.ListStreams(context.Background())
	if err == nil {
		t.Fatal("expected error (not fail-open empty)")
	}
	if streams != nil {
		t.Fatalf("streams=%+v want nil on error", streams)
	}
	if !strings.Contains(err.Error(), "http 500") {
		t.Fatalf("err=%v", err)
	}
}

func TestStreams_DisabledClient(t *testing.T) {
	c := New(Config{Enabled: false}, nil)
	if _, err := c.ListStreams(context.Background()); err == nil || !strings.Contains(err.Error(), "mesh disabled") {
		t.Fatalf("ListStreams err=%v", err)
	}
	if _, err := c.GetStream(context.Background(), "EVENTS"); err == nil || !strings.Contains(err.Error(), "mesh disabled") {
		t.Fatalf("GetStream err=%v", err)
	}
}

func TestGetStream_EmptyName(t *testing.T) {
	c := New(Config{Enabled: true, Endpoint: "http://127.0.0.1:9"}, nil)
	_, err := c.GetStream(context.Background(), "  ")
	if err == nil || !strings.Contains(err.Error(), "stream name required") {
		t.Fatalf("err=%v", err)
	}
}
