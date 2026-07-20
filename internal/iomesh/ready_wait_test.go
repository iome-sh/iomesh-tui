package iomesh

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestWaitReady_SucceedsAfterNFailures(t *testing.T) {
	var readyHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
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

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.WaitReady(ctx, WaitReadyOptions{Interval: 10 * time.Millisecond}); err != nil {
		t.Fatalf("WaitReady: %v", err)
	}
	if readyHits.Load() < 3 {
		t.Fatalf("expected ≥3 ready hits, got %d", readyHits.Load())
	}
}

func TestWaitReady_TimeoutOnAlwaysFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	err := c.WaitReady(ctx, WaitReadyOptions{Interval: 15 * time.Millisecond})
	if err == nil {
		t.Fatal("expected error on timeout")
	}
	if !strings.Contains(err.Error(), "wait ready") {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}

func TestWaitReady_Unreachable(t *testing.T) {
	// Closed listener: connection refused.
	c := New(Config{Enabled: true, Endpoint: "http://127.0.0.1:1"}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := c.WaitReady(ctx, WaitReadyOptions{Interval: 20 * time.Millisecond})
	if err == nil {
		t.Fatal("expected error for unreachable endpoint")
	}
}

func TestWaitReady_DisabledNil(t *testing.T) {
	ctx := context.Background()
	var nilClient *Client
	if err := nilClient.WaitReady(ctx, WaitReadyOptions{}); err != nil {
		t.Fatalf("nil client: %v", err)
	}
	c := New(Config{Enabled: false, Endpoint: ""}, nil)
	if err := c.WaitReady(ctx, WaitReadyOptions{}); err != nil {
		t.Fatalf("disabled: %v", err)
	}
}

func TestWaitReady_RequireHealth(t *testing.T) {
	var healthHits, readyHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			n := healthHits.Add(1)
			if n < 2 {
				w.WriteHeader(503)
				return
			}
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case "/ready", "/readyz":
			readyHits.Add(1)
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ready"))
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.WaitReady(ctx, WaitReadyOptions{
		Interval:      10 * time.Millisecond,
		RequireHealth: true,
	}); err != nil {
		t.Fatalf("WaitReady: %v", err)
	}
	if healthHits.Load() < 2 {
		t.Fatalf("expected health retries, got %d", healthHits.Load())
	}
	if readyHits.Load() < 1 {
		t.Fatal("expected ready after health ok")
	}
}

func TestFormatMeshWaitResult_Text(t *testing.T) {
	pass := FormatMeshWaitResult(true, 42, "", false)
	if !strings.Contains(pass, "PASS mesh wait: ready") {
		t.Fatalf("pass missing PASS line:\n%s", pass)
	}
	if !strings.Contains(pass, "elapsed_ms: 42") {
		t.Fatalf("pass missing elapsed_ms:\n%s", pass)
	}
	if !strings.Contains(pass, "require_health: false") {
		t.Fatalf("pass missing require_health:\n%s", pass)
	}
	if strings.Contains(pass, "FAIL") {
		t.Fatalf("pass should not contain FAIL:\n%s", pass)
	}

	fail := FormatMeshWaitResult(false, 1500, "wait ready: context deadline exceeded", true)
	if !strings.Contains(fail, "FAIL mesh wait: wait ready: context deadline exceeded") {
		t.Fatalf("fail missing FAIL line:\n%s", fail)
	}
	if !strings.Contains(fail, "elapsed_ms: 1500") {
		t.Fatalf("fail missing elapsed_ms:\n%s", fail)
	}
	if !strings.Contains(fail, "require_health: true") {
		t.Fatalf("fail missing require_health:\n%s", fail)
	}

	// Negative elapsed clamps to 0; empty errMsg gets a default.
	clamped := FormatMeshWaitResult(false, -1, "", false)
	if !strings.Contains(clamped, "elapsed_ms: 0") {
		t.Fatalf("negative elapsed should clamp to 0:\n%s", clamped)
	}
	if !strings.Contains(clamped, "unknown error") {
		t.Fatalf("empty errMsg should default:\n%s", clamped)
	}
	if !strings.Contains(clamped, "require_health: false") {
		t.Fatalf("clamped missing require_health:\n%s", clamped)
	}
}

func TestFormatMeshWaitResultJSON(t *testing.T) {
	js := FormatMeshWaitResultJSON(true, 123, "ignored on success", true)
	var okObj map[string]any
	if err := json.Unmarshal([]byte(js), &okObj); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if okObj["ok"] != true {
		t.Fatalf("ok: %v want true\n%s", okObj["ok"], js)
	}
	if n, _ := okObj["elapsed_ms"].(float64); int(n) != 123 {
		t.Fatalf("elapsed_ms: %v want 123\n%s", okObj["elapsed_ms"], js)
	}
	if okObj["require_health"] != true {
		t.Fatalf("require_health: %v want true\n%s", okObj["require_health"], js)
	}
	if _, has := okObj["error"]; has {
		t.Fatalf("success JSON should omit error:\n%s", js)
	}

	jsFail := FormatMeshWaitResultJSON(false, 456, "wait ready: deadline exceeded", false)
	var failObj map[string]any
	if err := json.Unmarshal([]byte(jsFail), &failObj); err != nil {
		t.Fatalf("json fail: %v\n%s", err, jsFail)
	}
	if failObj["ok"] != false {
		t.Fatalf("ok: %v want false\n%s", failObj["ok"], jsFail)
	}
	if n, _ := failObj["elapsed_ms"].(float64); int(n) != 456 {
		t.Fatalf("elapsed_ms: %v want 456\n%s", failObj["elapsed_ms"], jsFail)
	}
	if failObj["require_health"] != false {
		t.Fatalf("require_health: %v want false\n%s", failObj["require_health"], jsFail)
	}
	if failObj["error"] != "wait ready: deadline exceeded" {
		t.Fatalf("error: %v\n%s", failObj["error"], jsFail)
	}

	// Always-emit shape: ok + elapsed_ms + require_health present on both paths.
	for _, s := range []string{js, jsFail} {
		var m map[string]any
		if err := json.Unmarshal([]byte(s), &m); err != nil {
			t.Fatalf("unmarshal: %v\n%s", err, s)
		}
		if _, ok := m["ok"]; !ok {
			t.Fatalf("missing ok:\n%s", s)
		}
		if _, ok := m["elapsed_ms"]; !ok {
			t.Fatalf("missing elapsed_ms:\n%s", s)
		}
		if _, ok := m["require_health"]; !ok {
			t.Fatalf("missing require_health:\n%s", s)
		}
	}
}
