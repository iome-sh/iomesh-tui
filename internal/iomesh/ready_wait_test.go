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
	if !strings.Contains(pass, "result: ok") {
		t.Fatalf("pass missing result ok:\n%s", pass)
	}
	if !strings.Contains(pass, "exit_code: 0") {
		t.Fatalf("pass missing exit_code 0:\n%s", pass)
	}
	// version always emitted (empty when unset).
	if !strings.Contains(pass, "version: ") {
		t.Fatalf("pass missing version key:\n%s", pass)
	}
	// user_agent always emitted (empty when unset on evidence).
	if !strings.Contains(pass, "user_agent: ") {
		t.Fatalf("pass missing user_agent key:\n%s", pass)
	}
	// identity always emitted (empty when unset on evidence).
	for _, key := range []string{"endpoint: ", "tenant: ", "org: ", "workspace: "} {
		if !strings.Contains(pass, key) {
			t.Fatalf("pass missing identity key %q:\n%s", key, pass)
		}
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
	if !strings.Contains(fail, "result: err") {
		t.Fatalf("fail missing result err:\n%s", fail)
	}
	if !strings.Contains(fail, "exit_code: 1") {
		t.Fatalf("fail missing exit_code 1:\n%s", fail)
	}
	if !strings.Contains(fail, "version: ") {
		t.Fatalf("fail missing version key:\n%s", fail)
	}
	if !strings.Contains(fail, "user_agent: ") {
		t.Fatalf("fail missing user_agent key:\n%s", fail)
	}
	for _, key := range []string{"endpoint: ", "tenant: ", "org: ", "workspace: "} {
		if !strings.Contains(fail, key) {
			t.Fatalf("fail missing identity key %q:\n%s", key, fail)
		}
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
	if !strings.Contains(clamped, "result: err") {
		t.Fatalf("clamped fail missing result err:\n%s", clamped)
	}
	if !strings.Contains(clamped, "exit_code: 1") {
		t.Fatalf("clamped fail missing exit_code 1:\n%s", clamped)
	}
	if !strings.Contains(clamped, "version: ") {
		t.Fatalf("clamped missing version key:\n%s", clamped)
	}
	if !strings.Contains(clamped, "user_agent: ") {
		t.Fatalf("clamped missing user_agent key:\n%s", clamped)
	}

	// Stale ExitCode / Result are re-derived from OK in normalize.
	stale := FormatMeshWaitResult(MeshWaitEvidence{OK: true, ExitCode: 99, Result: "err", Attempts: 1})
	if !strings.Contains(stale, "exit_code: 0") {
		t.Fatalf("stale ExitCode must re-derive to 0:\n%s", stale)
	}
	if !strings.Contains(stale, "result: ok") {
		t.Fatalf("stale Result must re-derive to ok:\n%s", stale)
	}

	// Explicit Version is rendered; ProductVersion is wired at CLI layer.
	withVer := FormatMeshWaitResult(MeshWaitEvidence{
		OK: true, Attempts: 1, Version: "1.2.3-wait",
	})
	if !strings.Contains(withVer, "version: 1.2.3-wait") {
		t.Fatalf("text version should match field:\n%s", withVer)
	}

	// Explicit UserAgent is rendered; UserAgent() is wired at CLI layer.
	withUA := FormatMeshWaitResult(MeshWaitEvidence{
		OK: true, Attempts: 1, UserAgent: "iomesh-tui/test-wait-ua",
	})
	if !strings.Contains(withUA, "user_agent: iomesh-tui/test-wait-ua") {
		t.Fatalf("text user_agent should match field:\n%s", withUA)
	}

	// Explicit identity is rendered; CLI wires from config (empty honest when unset).
	withID := FormatMeshWaitResult(MeshWaitEvidence{
		OK: true, Attempts: 1,
		Endpoint: "http://mesh.example", Tenant: "t1", Org: "o1", Workspace: "w1",
	})
	for _, want := range []string{
		"endpoint: http://mesh.example",
		"tenant: t1",
		"org: o1",
		"workspace: w1",
	} {
		if !strings.Contains(withID, want) {
			t.Fatalf("text identity missing %q:\n%s", want, withID)
		}
	}
	// Order: endpoint → tenant → org → workspace
	epIdx := strings.Index(withID, "endpoint:")
	tnIdx := strings.Index(withID, "tenant:")
	orgIdx := strings.Index(withID, "org:")
	wsIdx := strings.Index(withID, "workspace:")
	if !(epIdx < tnIdx && tnIdx < orgIdx && orgIdx < wsIdx) {
		t.Fatalf("identity order want endpoint < tenant < org < workspace:\n%s", withID)
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
	if okObj["result"] != "ok" {
		t.Fatalf("result: %v want ok (OK)\n%s", okObj["result"], js)
	}
	if n, _ := okObj["exit_code"].(float64); int(n) != 0 {
		t.Fatalf("exit_code: %v want 0 (OK)\n%s", okObj["exit_code"], js)
	}
	// version always present; empty string when unset on evidence.
	v, hasVer := okObj["version"]
	if !hasVer {
		t.Fatalf("success JSON missing version:\n%s", js)
	}
	if v != "" {
		t.Fatalf("unset Version should emit empty string, got %q\n%s", v, js)
	}
	// user_agent always present; empty string when unset on evidence.
	ua, hasUA := okObj["user_agent"]
	if !hasUA {
		t.Fatalf("success JSON missing user_agent:\n%s", js)
	}
	if ua != "" {
		t.Fatalf("unset UserAgent should emit empty string, got %q\n%s", ua, js)
	}
	// identity always present; empty string when unset on evidence.
	for _, key := range []string{"endpoint", "tenant", "org", "workspace"} {
		v, has := okObj[key]
		if !has {
			t.Fatalf("success JSON missing %s:\n%s", key, js)
		}
		if v != "" {
			t.Fatalf("unset %s should emit empty string, got %q\n%s", key, v, js)
		}
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
	if failObj["result"] != "err" {
		t.Fatalf("result: %v want err (FAIL)\n%s", failObj["result"], jsFail)
	}
	if n, _ := failObj["exit_code"].(float64); int(n) != 1 {
		t.Fatalf("exit_code: %v want 1 (FAIL)\n%s", failObj["exit_code"], jsFail)
	}
	if failObj["error"] != "wait ready: deadline exceeded" {
		t.Fatalf("error: %v\n%s", failObj["error"], jsFail)
	}
	if _, has := failObj["version"]; !has {
		t.Fatalf("fail JSON missing version:\n%s", jsFail)
	}
	if _, has := failObj["user_agent"]; !has {
		t.Fatalf("fail JSON missing user_agent:\n%s", jsFail)
	}

	// Always-emit shape: ok + elapsed_ms + require_health + timeout_ms + interval_ms + attempts + result + exit_code + version + user_agent + identity on both paths.
	for _, s := range []string{js, jsFail} {
		var m map[string]any
		if err := json.Unmarshal([]byte(s), &m); err != nil {
			t.Fatalf("unmarshal: %v\n%s", err, s)
		}
		for _, key := range []string{"ok", "elapsed_ms", "require_health", "timeout_ms", "interval_ms", "attempts", "result", "exit_code", "version", "user_agent", "endpoint", "tenant", "org", "workspace"} {
			if _, ok := m[key]; !ok {
				t.Fatalf("missing %s:\n%s", key, s)
			}
		}
	}

	// Zero attempts always emit (disabled path); result=ok + exit_code 0 when OK.
	jsZero := FormatMeshWaitResultJSON(MeshWaitEvidence{OK: true, Attempts: 0})
	var zeroObj map[string]any
	if err := json.Unmarshal([]byte(jsZero), &zeroObj); err != nil {
		t.Fatalf("json zero: %v\n%s", err, jsZero)
	}
	if n, _ := zeroObj["attempts"].(float64); int(n) != 0 {
		t.Fatalf("zero attempts: %v want 0\n%s", zeroObj["attempts"], jsZero)
	}
	if zeroObj["result"] != "ok" {
		t.Fatalf("zero path result: %v want ok\n%s", zeroObj["result"], jsZero)
	}
	if n, _ := zeroObj["exit_code"].(float64); int(n) != 0 {
		t.Fatalf("zero path exit_code: %v want 0\n%s", zeroObj["exit_code"], jsZero)
	}
	if _, has := zeroObj["version"]; !has {
		t.Fatalf("zero path missing version:\n%s", jsZero)
	}
	if _, has := zeroObj["user_agent"]; !has {
		t.Fatalf("zero path missing user_agent:\n%s", jsZero)
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

	// Stale ExitCode / Result re-derived from OK.
	jsStale := FormatMeshWaitResultJSON(MeshWaitEvidence{OK: false, ExitCode: 0, Result: "ok", Error: "x"})
	var staleObj map[string]any
	if err := json.Unmarshal([]byte(jsStale), &staleObj); err != nil {
		t.Fatalf("json stale: %v\n%s", err, jsStale)
	}
	if n, _ := staleObj["exit_code"].(float64); int(n) != 1 {
		t.Fatalf("stale ExitCode must re-derive to 1: %v\n%s", staleObj["exit_code"], jsStale)
	}
	if staleObj["result"] != "err" {
		t.Fatalf("stale Result must re-derive to err: %v\n%s", staleObj["result"], jsStale)
	}

	// Explicit Version field value always emitted (CLI sets from ProductVersion).
	jsVer := FormatMeshWaitResultJSON(MeshWaitEvidence{
		OK: true, Attempts: 1, Version: "9.9.9-wait",
	})
	var verObj map[string]any
	if err := json.Unmarshal([]byte(jsVer), &verObj); err != nil {
		t.Fatalf("json ver: %v\n%s", err, jsVer)
	}
	if verObj["version"] != "9.9.9-wait" {
		t.Fatalf("version: %v want 9.9.9-wait\n%s", verObj["version"], jsVer)
	}

	// Explicit UserAgent field value always emitted (CLI sets from UserAgent()).
	jsUA := FormatMeshWaitResultJSON(MeshWaitEvidence{
		OK: true, Attempts: 1, UserAgent: "iomesh-tui/json-wait-ua",
	})
	var uaObj map[string]any
	if err := json.Unmarshal([]byte(jsUA), &uaObj); err != nil {
		t.Fatalf("json ua: %v\n%s", err, jsUA)
	}
	if uaObj["user_agent"] != "iomesh-tui/json-wait-ua" {
		t.Fatalf("user_agent: %v want iomesh-tui/json-wait-ua\n%s", uaObj["user_agent"], jsUA)
	}

	// Explicit identity field values always emitted (CLI sets from config).
	jsID := FormatMeshWaitResultJSON(MeshWaitEvidence{
		OK: true, Attempts: 1,
		Endpoint: "http://mesh.example", Tenant: "t1", Org: "o1", Workspace: "w1",
	})
	var idObj map[string]any
	if err := json.Unmarshal([]byte(jsID), &idObj); err != nil {
		t.Fatalf("json id: %v\n%s", err, jsID)
	}
	if idObj["endpoint"] != "http://mesh.example" || idObj["tenant"] != "t1" || idObj["org"] != "o1" || idObj["workspace"] != "w1" {
		t.Fatalf("identity: endpoint=%v tenant=%v org=%v workspace=%v\n%s",
			idObj["endpoint"], idObj["tenant"], idObj["org"], idObj["workspace"], jsID)
	}
}

func TestFormatMeshWaitResult_ProductVersion(t *testing.T) {
	// CLI wires ProductVersion into evidence; formatters pass Version through.
	// Simulate cmdMeshWait: Version: ProductVersion() after SetProductVersion.
	prev := ProductVersion()
	t.Cleanup(func() { productVersion = prev })
	SetProductVersion("0.52.0-product")

	v := ProductVersion()
	if v != "0.52.0-product" {
		t.Fatalf("ProductVersion: %q", v)
	}
	text := FormatMeshWaitResult(MeshWaitEvidence{OK: true, Attempts: 1, Version: v})
	if !strings.Contains(text, "version: 0.52.0-product") {
		t.Fatalf("text ProductVersion:\n%s", text)
	}
	js := FormatMeshWaitResultJSON(MeshWaitEvidence{OK: false, Error: "x", Version: v})
	var m map[string]any
	if err := json.Unmarshal([]byte(js), &m); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if m["version"] != "0.52.0-product" {
		t.Fatalf("json ProductVersion: %v\n%s", m["version"], js)
	}
}

func TestFormatMeshWaitResult_UserAgent(t *testing.T) {
	// CLI wires UserAgent() into evidence; formatters pass UserAgent through.
	// Simulate cmdMeshWait: UserAgent: iomesh.UserAgent() after SetUserAgent.
	prev := UserAgent()
	t.Cleanup(func() { SetUserAgent(prev) })
	SetUserAgent("iomesh-tui/test-mesh-wait-ua")

	ua := UserAgent()
	if ua != "iomesh-tui/test-mesh-wait-ua" {
		t.Fatalf("UserAgent: %q", ua)
	}
	text := FormatMeshWaitResult(MeshWaitEvidence{OK: true, Attempts: 1, UserAgent: ua})
	if !strings.Contains(text, "user_agent: iomesh-tui/test-mesh-wait-ua") {
		t.Fatalf("text UserAgent:\n%s", text)
	}
	js := FormatMeshWaitResultJSON(MeshWaitEvidence{OK: false, Error: "x", UserAgent: ua})
	var m map[string]any
	if err := json.Unmarshal([]byte(js), &m); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if m["user_agent"] != "iomesh-tui/test-mesh-wait-ua" {
		t.Fatalf("json UserAgent: %v\n%s", m["user_agent"], js)
	}
	// Always-emit: key present even when field empty.
	empty := FormatMeshWaitResultJSON(MeshWaitEvidence{OK: true, Attempts: 0})
	var emptyObj map[string]any
	if err := json.Unmarshal([]byte(empty), &emptyObj); err != nil {
		t.Fatalf("json empty: %v\n%s", err, empty)
	}
	if _, has := emptyObj["user_agent"]; !has {
		t.Fatalf("always-emit user_agent missing:\n%s", empty)
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

func TestMeshWaitResult(t *testing.T) {
	if got := MeshWaitResult(MeshWaitEvidence{OK: true}); got != "ok" {
		t.Fatalf("OK: MeshWaitResult=%q want ok", got)
	}
	if got := MeshWaitResult(MeshWaitEvidence{OK: false}); got != "err" {
		t.Fatalf("FAIL: MeshWaitResult=%q want err", got)
	}
	// Field Result is ignored; derived from OK only.
	if got := MeshWaitResult(MeshWaitEvidence{OK: true, Result: "err"}); got != "ok" {
		t.Fatalf("OK ignores Result field: %q want ok", got)
	}
	if got := MeshWaitResult(MeshWaitEvidence{OK: false, Result: "ok"}); got != "err" {
		t.Fatalf("FAIL ignores Result field: %q want err", got)
	}
}

func TestFormatMeshWaitResult_AlwaysEmitsResult(t *testing.T) {
	// Text + JSON always emit result=ok|err derived from OK (ok path + err path).
	okText := FormatMeshWaitResult(MeshWaitEvidence{OK: true, Attempts: 1})
	if !strings.Contains(okText, "result: ok") {
		t.Fatalf("ok text missing result ok:\n%s", okText)
	}
	errText := FormatMeshWaitResult(MeshWaitEvidence{OK: false, Error: "timeout", Attempts: 2})
	if !strings.Contains(errText, "result: err") {
		t.Fatalf("err text missing result err:\n%s", errText)
	}

	okJS := FormatMeshWaitResultJSON(MeshWaitEvidence{OK: true, Attempts: 1})
	var okObj map[string]any
	if err := json.Unmarshal([]byte(okJS), &okObj); err != nil {
		t.Fatalf("ok json: %v\n%s", err, okJS)
	}
	if okObj["result"] != "ok" {
		t.Fatalf("ok json result: %v want ok\n%s", okObj["result"], okJS)
	}

	errJS := FormatMeshWaitResultJSON(MeshWaitEvidence{OK: false, Error: "timeout", Attempts: 2})
	var errObj map[string]any
	if err := json.Unmarshal([]byte(errJS), &errObj); err != nil {
		t.Fatalf("err json: %v\n%s", err, errJS)
	}
	if errObj["result"] != "err" {
		t.Fatalf("err json result: %v want err\n%s", errObj["result"], errJS)
	}
}

func TestFormatMeshWaitResult_AlwaysEmitsIdentity(t *testing.T) {
	// Text + JSON always emit endpoint/tenant/org/workspace (empty when unset; no invent readiness).
	emptyText := FormatMeshWaitResult(MeshWaitEvidence{OK: true, Attempts: 0})
	for _, want := range []string{"endpoint: ", "tenant: ", "org: ", "workspace: "} {
		if !strings.Contains(emptyText, want) {
			t.Fatalf("empty text missing %q:\n%s", want, emptyText)
		}
	}
	// Empty identity lines present (value after colon may be blank).
	for _, line := range []string{"endpoint: \n", "tenant: \n", "org: \n", "workspace: \n"} {
		if !strings.Contains(emptyText, line) {
			t.Fatalf("empty identity line want %q:\n%s", line, emptyText)
		}
	}

	emptyJS := FormatMeshWaitResultJSON(MeshWaitEvidence{OK: false, Error: "x"})
	var m map[string]any
	if err := json.Unmarshal([]byte(emptyJS), &m); err != nil {
		t.Fatalf("json: %v\n%s", err, emptyJS)
	}
	for _, key := range []string{"endpoint", "tenant", "org", "workspace"} {
		v, has := m[key]
		if !has {
			t.Fatalf("always-emit %s missing:\n%s", key, emptyJS)
		}
		if v != "" {
			t.Fatalf("unset %s should be empty string, got %q\n%s", key, v, emptyJS)
		}
	}

	// Populated identity passes through; does not invent readiness success.
	pop := FormatMeshWaitResultJSON(MeshWaitEvidence{
		OK: false, Error: "wait ready: timeout", Attempts: 2,
		Endpoint: "http://127.0.0.1:1", Tenant: "dept.x", Org: "org_a", Workspace: "ws_y",
	})
	var p map[string]any
	if err := json.Unmarshal([]byte(pop), &p); err != nil {
		t.Fatalf("json pop: %v\n%s", err, pop)
	}
	if p["ok"] != false || p["result"] != "err" {
		t.Fatalf("identity must not invent success: ok=%v result=%v\n%s", p["ok"], p["result"], pop)
	}
	if p["endpoint"] != "http://127.0.0.1:1" || p["tenant"] != "dept.x" || p["org"] != "org_a" || p["workspace"] != "ws_y" {
		t.Fatalf("populated identity: %v\n%s", p, pop)
	}
}
