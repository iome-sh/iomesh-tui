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

func TestKVGet_OKAndUserAgent(t *testing.T) {
	prev := UserAgent()
	SetUserAgent("iomesh-tui/test-kv")
	t.Cleanup(func() { SetUserAgent(prev) })

	var gotUA, gotMethod, gotPath string
	payload := []byte(`{"hello":"world"}`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotUA = r.Header.Get("User-Agent")
		if r.Method != http.MethodGet || r.URL.Path != "/v1/kv/config/app.json" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"bucket":     "config",
			"key":        "app.json",
			"value":      base64.StdEncoding.EncodeToString(payload),
			"revision":   7,
			"created_at": time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
		})
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "acme"}, nil)
	entry, err := c.KVGet(context.Background(), "config", "app.json")
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("method=%q", gotMethod)
	}
	if gotPath != "/v1/kv/config/app.json" {
		t.Fatalf("path=%q", gotPath)
	}
	if gotUA != "iomesh-tui/test-kv" {
		t.Fatalf("User-Agent=%q", gotUA)
	}
	if entry == nil || entry.Bucket != "config" || entry.Key != "app.json" {
		t.Fatalf("entry=%+v", entry)
	}
	if entry.Revision != 7 {
		t.Fatalf("revision=%d", entry.Revision)
	}
	if string(entry.Value) != string(payload) {
		t.Fatalf("value=%q want %q", entry.Value, payload)
	}
	out := FormatKVEntry(*entry)
	if !strings.Contains(out, "app.json") || !strings.Contains(out, "hello") {
		t.Fatal(out)
	}
}

func TestKVGet_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	_, err := c.KVGet(context.Background(), "b", "missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "http 404") {
		t.Fatalf("err=%v", err)
	}
}

func TestKVGet_EmptyArgs(t *testing.T) {
	c := New(Config{Enabled: true, Endpoint: "http://127.0.0.1:9"}, nil)
	if _, err := c.KVGet(context.Background(), "", "k"); err == nil || !strings.Contains(err.Error(), "bucket required") {
		t.Fatalf("bucket err=%v", err)
	}
	if _, err := c.KVGet(context.Background(), "b", "  "); err == nil || !strings.Contains(err.Error(), "key required") {
		t.Fatalf("key err=%v", err)
	}
}

func TestKVListKeys_OKEnvelopeAndPrefix(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		if r.Method != http.MethodGet {
			http.Error(w, "method", 405)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []string{"app.json", "app.toml"},
		})
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	keys, err := c.KVListKeys(context.Background(), "config", "app")
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/kv/config" {
		t.Fatalf("path=%q", gotPath)
	}
	if gotQuery != "prefix=app" {
		t.Fatalf("query=%q", gotQuery)
	}
	if len(keys) != 2 || keys[0] != "app.json" {
		t.Fatalf("keys=%v", keys)
	}
	out := FormatKVKeys("config", keys)
	if !strings.Contains(out, "count=2") || !strings.Contains(out, "app.json") {
		t.Fatal(out)
	}
}

func TestKVListKeys_BareArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]string{"a", "b"})
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	keys, err := c.KVListKeys(context.Background(), "cfg", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 || keys[1] != "b" {
		t.Fatalf("keys=%v", keys)
	}
}

func TestKVListKeys_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	keys, err := c.KVListKeys(context.Background(), "b", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if keys != nil {
		t.Fatalf("keys=%v want nil on error", keys)
	}
	if !strings.Contains(err.Error(), "http 500") {
		t.Fatalf("err=%v", err)
	}
}

func TestKV_DisabledClient(t *testing.T) {
	c := New(Config{Enabled: false}, nil)
	if _, err := c.KVGet(context.Background(), "b", "k"); err == nil || !strings.Contains(err.Error(), "mesh disabled") {
		t.Fatalf("KVGet err=%v", err)
	}
	if _, err := c.KVListKeys(context.Background(), "b", ""); err == nil || !strings.Contains(err.Error(), "mesh disabled") {
		t.Fatalf("KVListKeys err=%v", err)
	}
}

func TestKVListKeys_EmptyBucket(t *testing.T) {
	c := New(Config{Enabled: true, Endpoint: "http://127.0.0.1:9"}, nil)
	if _, err := c.KVListKeys(context.Background(), "  ", ""); err == nil || !strings.Contains(err.Error(), "bucket required") {
		t.Fatalf("err=%v", err)
	}
}

func TestKVGet_PathEscape(t *testing.T) {
	// Server unescapes Path; use RequestURI / EscapedPath to verify client encoded.
	var gotEscaped string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotEscaped = r.URL.EscapedPath()
		if gotEscaped == "" {
			gotEscaped = r.URL.Path
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"bucket": "my bucket", "key": "a/b", "value": "", "revision": 1,
		})
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	entry, err := c.KVGet(context.Background(), "my bucket", "a/b")
	if err != nil {
		t.Fatal(err)
	}
	if entry == nil || entry.Key != "a/b" {
		t.Fatalf("entry=%+v", entry)
	}
	// PathEscape: space → %20, slash → %2F (EscapedPath preserves encoding).
	if gotEscaped != "/v1/kv/my%20bucket/a%2Fb" {
		t.Fatalf("escaped path=%q want /v1/kv/my%%20bucket/a%%2Fb", gotEscaped)
	}
}
