package iomesh

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCreateConsumer_201FullInfo(t *testing.T) {
	prev := UserAgent()
	SetUserAgent("iomesh-tui/test-consumer")
	t.Cleanup(func() { SetUserAgent(prev) })

	var gotMethod, gotPath, gotUA string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotUA = r.Header.Get("User-Agent")
		if r.Method != http.MethodPost || r.URL.Path != "/v1/streams/EVENTS/consumers" {
			http.NotFound(w, r)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"stream":         "EVENTS",
			"name":           "worker-1",
			"ack_floor":      42,
			"pending_count":  3,
			"filter_subject": "dept.events.>",
		})
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "acme"}, nil)
	info, err := c.CreateConsumer(context.Background(), "EVENTS", "worker-1", "dept.events.>")
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method=%q", gotMethod)
	}
	if gotPath != "/v1/streams/EVENTS/consumers" {
		t.Fatalf("path=%q", gotPath)
	}
	if gotUA != "iomesh-tui/test-consumer" {
		t.Fatalf("User-Agent=%q", gotUA)
	}
	if gotBody["name"] != "worker-1" {
		t.Fatalf("body name=%v", gotBody["name"])
	}
	if gotBody["filter_subject"] != "dept.events.>" {
		t.Fatalf("body filter=%v", gotBody["filter_subject"])
	}
	if info == nil || info.Stream != "EVENTS" || info.Name != "worker-1" {
		t.Fatalf("info=%+v", info)
	}
	if info.AckFloor != 42 || info.PendingCount != 3 || info.FilterSubject != "dept.events.>" {
		t.Fatalf("info fields=%+v", info)
	}
	out := FormatConsumerInfo(*info)
	if !strings.Contains(out, "worker-1") || !strings.Contains(out, "dept.events.>") || !strings.Contains(out, "ack_floor") {
		t.Fatal(out)
	}
}

func TestCreateConsumer_409NameOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", 405)
			return
		}
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"consumer already exists"}`))
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	info, err := c.CreateConsumer(context.Background(), "EVENTS", "worker-1", "")
	if err != nil {
		t.Fatal(err)
	}
	if info == nil || info.Stream != "EVENTS" || info.Name != "worker-1" {
		t.Fatalf("info=%+v want Stream/Name only", info)
	}
	if info.AckFloor != 0 || info.PendingCount != 0 || info.FilterSubject != "" {
		t.Fatalf("expected name-only on 409, got %+v", info)
	}
}

func TestCreateConsumer_EmptyArgsAndDisabled(t *testing.T) {
	c := New(Config{Enabled: true, Endpoint: "http://127.0.0.1:9"}, nil)
	if _, err := c.CreateConsumer(context.Background(), "", "c", ""); err == nil || !strings.Contains(err.Error(), "stream and name required") {
		t.Fatalf("empty stream err=%v", err)
	}
	if _, err := c.CreateConsumer(context.Background(), "S", "  ", ""); err == nil || !strings.Contains(err.Error(), "stream and name required") {
		t.Fatalf("blank name err=%v", err)
	}
	off := New(Config{Enabled: false}, nil)
	if _, err := off.CreateConsumer(context.Background(), "S", "c", ""); err == nil || !strings.Contains(err.Error(), "mesh disabled") {
		t.Fatalf("disabled err=%v", err)
	}
}

func TestCreateConsumer_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	_, err := c.CreateConsumer(context.Background(), "S", "c", "")
	if err == nil || !strings.Contains(err.Error(), "http 500") {
		t.Fatalf("err=%v", err)
	}
}

func TestCreateConsumer_PathEscape(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"stream": "a/b",
			"name":   "c",
		})
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	info, err := c.CreateConsumer(context.Background(), "a/b", "c", "")
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/streams/a%2Fb/consumers" {
		t.Fatalf("path=%q want escaped stream", gotPath)
	}
	if info.Stream != "a/b" || info.Name != "c" {
		t.Fatalf("info=%+v", info)
	}
}

func TestFormatConsumerInfo_OmitsEmptyFilter(t *testing.T) {
	out := FormatConsumerInfo(ConsumerInfo{Stream: "S", Name: "c", AckFloor: 1, PendingCount: 0})
	if strings.Contains(out, "filter_subject") {
		t.Fatalf("empty filter should be omitted: %s", out)
	}
	if !strings.Contains(out, "stream:          S") || !strings.Contains(out, "name:            c") {
		t.Fatal(out)
	}
}

func TestConsumerFetch_OK(t *testing.T) {
	payload := []byte(`{"ok":true}`)
	var gotBody map[string]any
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method != http.MethodPost || r.URL.Path != "/v1/streams/EVENTS/consumers/worker-1/fetch" {
			http.NotFound(w, r)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"messages": []map[string]any{
				{
					"stream":  "EVENTS",
					"seq":     7,
					"subject": "dept.events.ping",
					"payload": base64.StdEncoding.EncodeToString(payload),
				},
			},
		})
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	msgs, err := c.ConsumerFetch(context.Background(), "EVENTS", "worker-1", 1, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/streams/EVENTS/consumers/worker-1/fetch" {
		t.Fatalf("path=%q", gotPath)
	}
	if gotBody["batch"] != float64(1) {
		t.Fatalf("batch=%v", gotBody["batch"])
	}
	if gotBody["max_wait_ms"] != float64(2000) {
		t.Fatalf("max_wait_ms=%v", gotBody["max_wait_ms"])
	}
	if len(msgs) != 1 || msgs[0].Seq != 7 || string(msgs[0].Payload) != string(payload) {
		t.Fatalf("msgs=%+v", msgs)
	}
}

func TestConsumerFetch_Validation(t *testing.T) {
	c := New(Config{Enabled: true, Endpoint: "http://127.0.0.1:9"}, nil)
	if _, err := c.ConsumerFetch(context.Background(), "S", "c", 0, 0); err == nil || !strings.Contains(err.Error(), "batch") {
		t.Fatalf("batch err=%v", err)
	}
	off := New(Config{Enabled: false}, nil)
	if _, err := off.ConsumerFetch(context.Background(), "S", "c", 1, 0); err == nil || !strings.Contains(err.Error(), "mesh disabled") {
		t.Fatalf("disabled err=%v", err)
	}
}
