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

func TestWaitReadyAttempts_SucceedsAttemptsGte1(t *testing.T) {
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
	attempts, err := c.WaitReadyAttempts(ctx, WaitReadyOptions{Interval: 10 * time.Millisecond})
	if err != nil {
		t.Fatalf("WaitReadyAttempts: %v", err)
	}
	if attempts < 1 {
		t.Fatalf("success attempts: %d want >= 1", attempts)
	}
	if int(readyHits.Load()) != attempts {
		t.Fatalf("attempts %d != ready hits %d", attempts, readyHits.Load())
	}
	if attempts < 3 {
		t.Fatalf("expected >=3 attempts after 2 failures, got %d", attempts)
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

func TestWaitReadyAttempts_DisabledNil(t *testing.T) {
	ctx := context.Background()
	var nilClient *Client
	attempts, err := nilClient.WaitReadyAttempts(ctx, WaitReadyOptions{})
	if err != nil {
		t.Fatalf("nil client: %v", err)
	}
	if attempts != 0 {
		t.Fatalf("nil client attempts: %d want 0", attempts)
	}
	c := New(Config{Enabled: false, Endpoint: ""}, nil)
	attempts, err = c.WaitReadyAttempts(ctx, WaitReadyOptions{})
	if err != nil {
		t.Fatalf("disabled: %v", err)
	}
	if attempts != 0 {
		t.Fatalf("disabled attempts: %d want 0", attempts)
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
	pass := FormatMeshWaitResult(MeshWaitEvidence{
		OK: true, ElapsedMS: 42, RequireHealth: false, TimeoutMS: 30000, IntervalMS: 500, Attempts: 3,
	})
	if !strings.Contains(pass, "PASS mesh wait: ready") {
		t.Fatalf("pass missing PASS line:\n%s", pass)
	}
	if !strings.Contains(pass, "elapsed_ms: 42") {
		t.Fatalf("pass missing elapsed_ms:\n%s", pass)
	}
	if !strings.Contains(pass, "require_health: false") {
		t.Fatalf("pass missing require_health:\n%s", pass)
	}
	if !strings.Contains(pass, "timeout_ms: 30000") {
		t.Fatalf("pass missing timeout_ms:\n%s", pass)
	}
	if !strings.Contains(pass, "interval_ms: 500") {
		t.Fatalf("pass missing interval_ms:\n%s", pass)
	}
	if !strings.Contains(pass, "attempts: 3") {
		t.Fatalf("pass missing attempts:\n%s", pass)
	}
	if !strings.Contains(pass, "exit_code: 0") {
		t.Fatalf("pass missing exit_code 0:\n%s", pass)
	}
	if strings.Contains(pass, "FAIL") {
		t.Fatalf("pass should not contain FAIL:\n%s", pass)
	}

	fail := FormatMeshWaitResult(MeshWaitEvidence{
		OK: false, ElapsedMS: 1500, RequireHealth: true,
		TimeoutMS: 1500, IntervalMS: 250, Attempts: 7,
		Error: "wait ready: context deadline exceeded",
	})
	if !strings.Contains(fail, "FAIL mesh wait: wait ready: context deadline exceeded") {
		t.Fatalf("fail missing FAIL line:\n%s", fail)
	}
	if !strings.Contains(fail, "elapsed_ms: 1500") {
		t.Fatalf("fail missing elapsed_ms:\n%s", fail)
	}
	if !strings.Contains(fail, "require_health: true") {
		t.Fatalf("fail missing require_health:\n%s", fail)
	}
	if !strings.Contains(fail, "timeout_ms: 1500") {
		t.Fatalf("fail missing timeout_ms:\n%s", fail)
	}
	if !strings.Contains(fail, "interval_ms: 250") {
		t.Fatalf("fail missing interval_ms:\n%s", fail)
	}
	if !strings.Contains(fail, "attempts: 7") {
		t.Fatalf("fail missing attempts:\n%s", fail)
	}
	if !strings.Contains(fail, "exit_code: 1") {
		t.Fatalf("fail missing exit_code 1:\n%s", fail)
	}

	// Negative elapsed/timeout/interval/attempts clamp to 0; empty Error gets a default.
	clamped := FormatMeshWaitResult(MeshWaitEvidence{
		OK: false, ElapsedMS: -1, TimeoutMS: -5, IntervalMS: -10, Attempts: -3,
	})
	if !strings.Contains(clamped, "elapsed_ms: 0") {
		t.Fatalf("negative elapsed should clamp to 0:\n%s", clamped)
	}
	if !strings.Contains(clamped, "timeout_ms: 0") {
		t.Fatalf("negative timeout should clamp to 0:\n%s", clamped)
	}
	if !strings.Contains(clamped, "interval_ms: 0") {
		t.Fatalf("negative interval should clamp to 0:\n%s", clamped)
	}
	if !strings.Contains(clamped, "attempts: 0") {
		t.Fatalf("negative attempts should clamp to 0:\n%s", clamped)
	}
	if !strings.Contains(clamped, "unknown error") {
		t.Fatalf("empty Error should default:\n%s", clamped)
	}
	if !strings.Contains(clamped, "require_health: false") {
		t.Fatalf("clamped missing require_health:\n%s", clamped)
	}
	if !strings.Contains(clamped, "exit_code: 1") {
		t.Fatalf("clamped fail missing exit_code 1:\n%s", clamped)
	}

	// Stale ExitCode is re-derived from OK in normalize.
	stale := FormatMeshWaitResult(MeshWaitEvidence{OK: true, ExitCode: 99, Attempts: 1})
	if !strings.Contains(stale, "exit_code: 0") {
		t.Fatalf("stale ExitCode must re-derive to 0:\n%s", stale)
	}
}

func TestFormatMeshWaitResultJSON(t *testing.T) {
	js := FormatMeshWaitResultJSON(MeshWaitEvidence{
		OK: true, ElapsedMS: 123, RequireHealth: true,
		TimeoutMS: 30000, IntervalMS: 500, Attempts: 2,
		Error: "ignored on success",
	})
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
	if n, _ := okObj["timeout_ms"].(float64); int(n) != 30000 {
		t.Fatalf("timeout_ms: %v want 30000\n%s", okObj["timeout_ms"], js)
	}
	if n, _ := okObj["interval_ms"].(float64); int(n) != 500 {
		t.Fatalf("interval_ms: %v want 500\n%s", okObj["interval_ms"], js)
	}
	if n, _ := okObj["attempts"].(float64); int(n) != 2 {
		t.Fatalf("attempts: %v want 2\n%s", okObj["attempts"], js)
	}
	if n, _ := okObj["exit_code"].(float64); int(n) != 0 {
		t.Fatalf("exit_code: %v want 0 (OK)\n%s", okObj["exit_code"], js)
	}
	if _, has := okObj["error"]; has {
		t.Fatalf("success JSON should omit error:\n%s", js)
	}

	jsFail := FormatMeshWaitResultJSON(MeshWaitEvidence{
		OK: false, ElapsedMS: 456, RequireHealth: false,
		TimeoutMS: 500, IntervalMS: 100, Attempts: 5,
		Error: "wait ready: deadline exceeded",
	})
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
	if n, _ := failObj["timeout_ms"].(float64); int(n) != 500 {
		t.Fatalf("timeout_ms: %v want 500\n%s", failObj["timeout_ms"], jsFail)
	}
	if n, _ := failObj["interval_ms"].(float64); int(n) != 100 {
		t.Fatalf("interval_ms: %v want 100\n%s", failObj["interval_ms"], jsFail)
	}
	if n, _ := failObj["attempts"].(float64); int(n) != 5 {
		t.Fatalf("attempts: %v want 5\n%s", failObj["attempts"], jsFail)
	}
	if n, _ := failObj["exit_code"].(float64); int(n) != 1 {
		t.Fatalf("exit_code: %v want 1 (FAIL)\n%s", failObj["exit_code"], jsFail)
	}
	if failObj["error"] != "wait ready: deadline exceeded" {
		t.Fatalf("error: %v\n%s", failObj["error"], jsFail)
	}

	// Always-emit shape: ok + elapsed_ms + require_health + timeout_ms + interval_ms + attempts + exit_code on both paths.
	for _, s := range []string{js, jsFail} {
		var m map[string]any
		if err := json.Unmarshal([]byte(s), &m); err != nil {
			t.Fatalf("unmarshal: %v\n%s", err, s)
		}
		for _, key := range []string{"ok", "elapsed_ms", "require_health", "timeout_ms", "interval_ms", "attempts", "exit_code"} {
			if _, ok := m[key]; !ok {
				t.Fatalf("missing %s:\n%s", key, s)
			}
		}
	}

	// Zero attempts always emit (disabled path); exit_code 0 when OK.
	jsZero := FormatMeshWaitResultJSON(MeshWaitEvidence{OK: true, Attempts: 0})
	var zeroObj map[string]any
	if err := json.Unmarshal([]byte(jsZero), &zeroObj); err != nil {
		t.Fatalf("json zero: %v\n%s", err, jsZero)
	}
	if n, _ := zeroObj["attempts"].(float64); int(n) != 0 {
		t.Fatalf("zero attempts: %v want 0\n%s", zeroObj["attempts"], jsZero)
	}
	if n, _ := zeroObj["exit_code"].(float64); int(n) != 0 {
		t.Fatalf("zero path exit_code: %v want 0\n%s", zeroObj["exit_code"], jsZero)
	}

	// Negative attempts clamp to 0 in JSON.
	jsNeg := FormatMeshWaitResultJSON(MeshWaitEvidence{OK: true, Attempts: -9})
	var negObj map[string]any
	if err := json.Unmarshal([]byte(jsNeg), &negObj); err != nil {
		t.Fatalf("json neg: %v\n%s", err, jsNeg)
	}
	if n, _ := negObj["attempts"].(float64); int(n) != 0 {
		t.Fatalf("negative attempts clamp: %v want 0\n%s", negObj["attempts"], jsNeg)
	}

	// Stale ExitCode re-derived from OK.
	jsStale := FormatMeshWaitResultJSON(MeshWaitEvidence{OK: false, ExitCode: 0, Error: "x"})
	var staleObj map[string]any
	if err := json.Unmarshal([]byte(jsStale), &staleObj); err != nil {
		t.Fatalf("json stale: %v\n%s", err, jsStale)
	}
	if n, _ := staleObj["exit_code"].(float64); int(n) != 1 {
		t.Fatalf("stale ExitCode must re-derive to 1: %v\n%s", staleObj["exit_code"], jsStale)
	}
}

func TestMeshWaitExitCode(t *testing.T) {
	if got := MeshWaitExitCode(MeshWaitEvidence{OK: true}); got != 0 {
		t.Fatalf("OK: MeshWaitExitCode=%d want 0", got)
	}
	if got := MeshWaitExitCode(MeshWaitEvidence{OK: false}); got != 1 {
		t.Fatalf("FAIL: MeshWaitExitCode=%d want 1", got)
	}
	// Field ExitCode is ignored; derived from OK only.
	if got := MeshWaitExitCode(MeshWaitEvidence{OK: true, ExitCode: 1}); got != 0 {
		t.Fatalf("OK ignores ExitCode field: %d want 0", got)
	}
	if got := MeshWaitExitCode(MeshWaitEvidence{OK: false, ExitCode: 0}); got != 1 {
		t.Fatalf("FAIL ignores ExitCode field: %d want 1", got)
	}
}
