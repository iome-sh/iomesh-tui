package iomesh

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEvaluatePolicy_OffAndFailOpen(t *testing.T) {
	c := New(Config{Enabled: true, Endpoint: "http://127.0.0.1:1", PolicyMode: PolicyOff}, nil)
	dec := c.EvaluatePolicy(context.Background(), PolicyInput{Tool: "run_shell"})
	if !dec.Allow || dec.Source != "off" {
		t.Fatalf("%+v", dec)
	}

	// Unreachable → fail-open allow
	c2 := New(Config{Enabled: true, Endpoint: "http://127.0.0.1:1", PolicyMode: PolicyEnforce}, nil)
	dec2 := c2.EvaluatePolicy(context.Background(), PolicyInput{Action: "tool.run_shell", Tool: "run_shell"})
	if !dec2.Allow || dec2.Source != "fail-open" {
		t.Fatalf("%+v", dec2)
	}
	if dec2.ShouldBlockTool() {
		t.Fatal("fail-open must not block")
	}
}

func TestEvaluatePolicy_EnforceDeny(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/policy/evaluate" {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["tool"] != "run_shell" {
			t.Errorf("tool=%v", body["tool"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"allow":   false,
			"reasons": []string{"rego: shell blocked"},
		})
	}))
	defer srv.Close()

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, PolicyMode: PolicyEnforce, Tenant: "t",
		EmitDeptStreams: true,
	}, nil)
	dec := c.EvaluatePolicy(context.Background(), PolicyInput{Tool: "run_shell"})
	if dec.Allow || !dec.ShouldBlockTool() || dec.Source != "mesh" {
		t.Fatalf("%+v", dec)
	}
	if !strings.Contains(dec.Summary(), "deny") {
		t.Fatalf("summary=%s", dec.Summary())
	}
}

func TestEvaluatePolicy_AdvisoryNeverBlocks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"allow": false, "reason": "nope"})
	}))
	defer srv.Close()
	c := New(Config{Enabled: true, Endpoint: srv.URL, PolicyMode: PolicyAdvisory}, nil)
	dec := c.EvaluatePolicy(context.Background(), PolicyInput{Tool: "write_file"})
	if dec.Allow {
		t.Fatal("mesh said deny")
	}
	if dec.ShouldBlockTool() {
		t.Fatal("advisory must not ShouldBlockTool")
	}
}

func TestEvaluatePolicy_404Unavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()
	c := New(Config{Enabled: true, Endpoint: srv.URL, PolicyMode: PolicyEnforce}, nil)
	dec := c.EvaluatePolicy(context.Background(), PolicyInput{Tool: "x"})
	if !dec.Allow || dec.Source != "unavailable" || dec.ShouldBlockTool() {
		t.Fatalf("%+v", dec)
	}
}
