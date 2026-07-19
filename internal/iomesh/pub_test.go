package iomesh

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPub_Disabled(t *testing.T) {
	c := New(Config{}, nil)
	if err := c.Pub(context.Background(), "s.events", []byte("x"), nil); err == nil || !strings.Contains(err.Error(), "mesh disabled") {
		t.Fatalf("Pub err=%v", err)
	}
	c2 := New(Config{Enabled: true, Endpoint: ""}, nil)
	if err := c2.Pub(context.Background(), "s.events", []byte("x"), nil); err == nil || !strings.Contains(err.Error(), "mesh disabled") {
		t.Fatalf("empty endpoint Pub err=%v", err)
	}
}

func TestPub_EmptySubject(t *testing.T) {
	c := New(Config{Enabled: true, Endpoint: "http://127.0.0.1:9"}, nil)
	if err := c.Pub(context.Background(), "", []byte("x"), nil); err == nil || !strings.Contains(err.Error(), "subject required") {
		t.Fatalf("empty err=%v", err)
	}
	if err := c.Pub(context.Background(), "  ", []byte("x"), nil); err == nil || !strings.Contains(err.Error(), "subject required") {
		t.Fatalf("blank err=%v", err)
	}
}

func TestPub_WireBodyAndHeaders(t *testing.T) {
	prev := UserAgent()
	SetUserAgent("iomesh-tui/test-pub")
	t.Cleanup(func() { SetUserAgent(prev) })

	var gotMethod, gotPath, gotUA, gotTenant, gotOrg, gotWS, gotCT string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotUA = r.Header.Get("User-Agent")
		gotTenant = r.Header.Get("X-IOMesh-Tenant")
		gotOrg = r.Header.Get("X-IOMesh-Org")
		gotWS = r.Header.Get("X-IOMesh-Workspace")
		gotCT = r.Header.Get("Content-Type")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "acme",
		OrgID: "org-1", WorkspaceID: "ws-9",
	}, nil)
	err := c.Pub(context.Background(), "dept.agent.ping", []byte(`{"ok":true}`), map[string]string{
		"x-trace": "t1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method=%q", gotMethod)
	}
	if gotPath != "/v1/pub" {
		t.Fatalf("path=%q", gotPath)
	}
	if gotUA != "iomesh-tui/test-pub" {
		t.Fatalf("User-Agent=%q", gotUA)
	}
	if gotTenant != "acme" {
		t.Fatalf("tenant=%q", gotTenant)
	}
	if gotOrg != "org-1" {
		t.Fatalf("org=%q", gotOrg)
	}
	if gotWS != "ws-9" {
		t.Fatalf("workspace=%q", gotWS)
	}
	if !strings.Contains(gotCT, "application/json") {
		t.Fatalf("Content-Type=%q", gotCT)
	}
	if gotBody["subject"] != "dept.agent.ping" {
		t.Fatalf("subject=%v", gotBody["subject"])
	}
	// SDK wire: payload is raw string, not base64.
	if gotBody["payload"] != `{"ok":true}` {
		t.Fatalf("payload=%v want raw string", gotBody["payload"])
	}
	hdrs, ok := gotBody["headers"].(map[string]any)
	if !ok || hdrs["x-trace"] != "t1" {
		t.Fatalf("headers=%v", gotBody["headers"])
	}
}

func TestPub_NilPayloadEmptyString(t *testing.T) {
	var gotPayload any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotPayload = body["payload"]
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	if err := c.Pub(context.Background(), "events.demo", nil, nil); err != nil {
		t.Fatal(err)
	}
	if gotPayload != "" {
		t.Fatalf("nil payload wire=%v want empty string", gotPayload)
	}
}

func TestPub_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("boom"))
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	err := c.Pub(context.Background(), "events.demo", []byte("x"), nil)
	if err == nil || !strings.Contains(err.Error(), "http 500") {
		t.Fatalf("err=%v", err)
	}
}

func TestPub_OmitsHeadersWhenNil(t *testing.T) {
	var raw string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		raw = string(b)
		w.WriteHeader(204)
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	if err := c.Pub(context.Background(), "s", []byte("p"), nil); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(raw, "headers") {
		t.Fatalf("nil headers should be omitempty: %s", raw)
	}
}
