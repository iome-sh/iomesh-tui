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

func TestFormatStreamDetail_Fields(t *testing.T) {
	max := int64(1000)
	age := int64(3600)
	detail := FormatStreamDetail(StreamInfo{
		Name:        "EVENTS",
		Description: "ops events",
		Retention:   "limits",
		Partitions:  1,
		MaxMsgs:     &max,
		MaxAgeSec:   &age,
		Messages:    10,
		FirstSeq:    1,
		LastSeq:     10,
		CreatedAt:   time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
		Subjects:    []string{"dept.events.>", "dept.ops.>"},
	})
	for _, want := range []string{
		"iomesh stream",
		"name:        EVENTS",
		"description: ops events",
		"retention:   limits",
		"partitions:  1",
		"max_msgs:    1000",
		"max_age_sec: 3600",
		"messages:    10",
		"first_seq:   1",
		"last_seq:    10",
		"created_at:  2026-07-01T12:00:00Z",
		"subjects:",
		"dept.events.>",
		"dept.ops.>",
	} {
		if !strings.Contains(detail, want) {
			t.Fatalf("missing %q in:\n%s", want, detail)
		}
	}
}

// Empty / zero StreamInfo still always-emits every scraper key (honest blanks).
func TestFormatStreamDetail_EmptyAlwaysEmit(t *testing.T) {
	detail := FormatStreamDetail(StreamInfo{
		Name: "SPARSE",
	})
	for _, want := range []string{
		"iomesh stream",
		"name:        SPARSE",
		"description: \n",
		"retention:   \n",
		"partitions:  0\n",
		"max_msgs:    \n",
		"max_age_sec: \n",
		"messages:    0\n",
		"first_seq:   0\n",
		"last_seq:    0\n",
		"created_at:  \n",
		"subjects:\n",
		"  (none)\n",
	} {
		if !strings.Contains(detail, want) {
			t.Fatalf("missing %q in:\n%s", want, detail)
		}
	}
	// nil *int64 must not invent 0; blank value only
	if strings.Contains(detail, "max_msgs:    0") {
		t.Fatalf("max_msgs should be blank when nil, got:\n%s", detail)
	}
	if strings.Contains(detail, "max_age_sec: 0") {
		t.Fatalf("max_age_sec should be blank when nil, got:\n%s", detail)
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
	if err := c.DeleteStream(context.Background(), "EVENTS"); err == nil || !strings.Contains(err.Error(), "mesh disabled") {
		t.Fatalf("DeleteStream err=%v", err)
	}
}

func TestGetStream_EmptyName(t *testing.T) {
	c := New(Config{Enabled: true, Endpoint: "http://127.0.0.1:9"}, nil)
	_, err := c.GetStream(context.Background(), "  ")
	if err == nil || !strings.Contains(err.Error(), "stream name required") {
		t.Fatalf("err=%v", err)
	}
}

func TestDeleteStream_204(t *testing.T) {
	var gotMethod, gotPath, gotUA string
	prev := UserAgent()
	SetUserAgent("iomesh-tui/test-s302")
	t.Cleanup(func() { SetUserAgent(prev) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotUA = r.Header.Get("User-Agent")
		if r.Method != http.MethodDelete || r.URL.Path != "/v1/streams/TEMP" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	if err := c.DeleteStream(context.Background(), "TEMP"); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodDelete {
		t.Fatalf("method=%q", gotMethod)
	}
	if gotPath != "/v1/streams/TEMP" {
		t.Fatalf("path=%q", gotPath)
	}
	if gotUA != "iomesh-tui/test-s302" {
		t.Fatalf("User-Agent=%q", gotUA)
	}
}

func TestDeleteStream_EmptyName(t *testing.T) {
	c := New(Config{Enabled: true, Endpoint: "http://127.0.0.1:9"}, nil)
	err := c.DeleteStream(context.Background(), "  ")
	if err == nil || !strings.Contains(err.Error(), "stream name required") {
		t.Fatalf("err=%v", err)
	}
}

func TestDeleteStream_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"stream not found"}`))
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	err := c.DeleteStream(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "http 404") {
		t.Fatalf("err=%v", err)
	}
}

func TestListStreamMessages_OKAndQuery(t *testing.T) {
	prev := UserAgent()
	SetUserAgent("iomesh-tui/test-messages")
	t.Cleanup(func() { SetUserAgent(prev) })

	payloadB64 := base64.StdEncoding.EncodeToString([]byte(`{"event":"hello"}`))
	var gotPath, gotQuery, gotMethod, gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotUA = r.Header.Get("User-Agent")
		if r.Method != http.MethodGet || r.URL.Path != "/v1/streams/EVENTS/messages" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"messages": []map[string]any{
				{
					"stream":    "EVENTS",
					"seq":       1,
					"subject":   "dept.events.hello",
					"payload":   payloadB64,
					"timestamp": time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
				},
				{
					"stream":  "EVENTS",
					"seq":     2,
					"subject": "dept.events.world",
					"payload": base64.StdEncoding.EncodeToString([]byte("plain-text")),
				},
			},
		})
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "acme"}, nil)
	msgs, err := c.ListStreamMessages(context.Background(), "EVENTS", ListStreamMessagesOptions{
		FromSeq: 1,
		ToSeq:   10,
		Limit:   20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("method=%q", gotMethod)
	}
	if gotPath != "/v1/streams/EVENTS/messages" {
		t.Fatalf("path=%q", gotPath)
	}
	if !strings.Contains(gotQuery, "from_seq=1") || !strings.Contains(gotQuery, "to_seq=10") || !strings.Contains(gotQuery, "limit=20") {
		t.Fatalf("query=%q", gotQuery)
	}
	if gotUA != "iomesh-tui/test-messages" {
		t.Fatalf("User-Agent=%q", gotUA)
	}
	if len(msgs) != 2 || msgs[0].Seq != 1 || string(msgs[0].Payload) != `{"event":"hello"}` {
		t.Fatalf("msgs=%+v", msgs)
	}
	if string(msgs[1].Payload) != "plain-text" {
		t.Fatalf("payload[1]=%q", msgs[1].Payload)
	}
	out := FormatStreamMessages("EVENTS", msgs)
	if !strings.Contains(out, "count=2") || !strings.Contains(out, "dept.events.hello") || !strings.Contains(out, "plain-text") {
		t.Fatal(out)
	}
}

func TestListStreamMessages_403(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"code":"replay_disabled","message":"stream replay requires tenant or flag"}}`))
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	msgs, err := c.ListStreamMessages(context.Background(), "EVENTS", ListStreamMessagesOptions{Limit: 5})
	if err == nil {
		t.Fatal("expected error")
	}
	if msgs != nil {
		t.Fatalf("msgs=%+v want nil on error", msgs)
	}
	if !strings.Contains(err.Error(), "http 403") {
		t.Fatalf("err=%v", err)
	}
}

func TestListStreamMessages_EmptyName(t *testing.T) {
	c := New(Config{Enabled: true, Endpoint: "http://127.0.0.1:9"}, nil)
	_, err := c.ListStreamMessages(context.Background(), "  ", ListStreamMessagesOptions{})
	if err == nil || !strings.Contains(err.Error(), "stream name required") {
		t.Fatalf("err=%v", err)
	}
}

func TestListStreamMessages_DisabledClient(t *testing.T) {
	c := New(Config{Enabled: false}, nil)
	if _, err := c.ListStreamMessages(context.Background(), "EVENTS", ListStreamMessagesOptions{}); err == nil || !strings.Contains(err.Error(), "mesh disabled") {
		t.Fatalf("ListStreamMessages err=%v", err)
	}
}

func TestListStreamMessages_BareArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"seq": 9, "subject": "x.y", "payload": base64.StdEncoding.EncodeToString([]byte("z"))},
		})
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	msgs, err := c.ListStreamMessages(context.Background(), "S", ListStreamMessagesOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0].Seq != 9 || string(msgs[0].Payload) != "z" {
		t.Fatalf("msgs=%+v", msgs)
	}
}
