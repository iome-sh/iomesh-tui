package iomesh

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/router"
)

func TestDisabledIsNoop(t *testing.T) {
	c := New(Config{Enabled: false}, nil)
	if c.Enabled() {
		t.Fatal()
	}
	if snip := c.ContextSnippet(context.Background(), "/ws", "q"); snip != "" {
		t.Fatal(snip)
	}
	c.Emit(context.Background(), DeptEvent{Type: "dept.test"})
	if err := c.Health(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidEndpointDisables(t *testing.T) {
	c := New(Config{Enabled: true, Endpoint: "file:///etc/passwd"}, nil)
	if c.Enabled() {
		t.Fatal("file URL must disable client")
	}
}

func TestContextSnippet_SuccessAndFailOpen(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/context/query" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "mesh-ctx"})
	}))
	defer srv.Close()

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, ContextPlane: true, Tenant: "t",
	}, nil)
	if !c.Enabled() {
		t.Fatal("should enable")
	}
	if got := c.ContextSnippet(context.Background(), "/ws", "q"); got != "mesh-ctx" {
		t.Fatalf("got %q", got)
	}

	// Fail-open on 500
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer bad.Close()
	c2 := New(Config{Enabled: true, Endpoint: bad.URL, ContextPlane: true}, nil)
	if got := c2.ContextSnippet(context.Background(), "/ws", "q"); got != "" {
		t.Fatalf("fail-open got %q", got)
	}
}

func TestEmitAndRecordLLMCall(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(204)
	}))
	defer srv.Close()

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, EmitDeptStreams: true, Tenant: "acme",
	}, nil)
	c.RecordLLMCall(router.CallMeta{
		ModelName: "deepseek-v4-flash", ModelID: "deepseek-v4-flash",
		Duration: time.Millisecond, EstimatedUSD: 0.001,
	}, router.Usage{TotalTokens: 10}, nil)

	// Allow async-ish emit to complete (RecordLLMCall is sync with timeout).
	if !strings.Contains(gotBody, "dept.agent.llm_call") {
		t.Fatalf("body=%q", gotBody)
	}
	if strings.Contains(gotBody, "Bearer ") {
		t.Fatal("must not log bearer in payload")
	}
}

func TestHealth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(200)
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()
	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	if err := c.Health(context.Background()); err != nil {
		t.Fatal(err)
	}
}
