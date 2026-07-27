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

// s732: PubPrint JSON always-emits {ok,subject,bytes} (no pull_role / payload invent).
func TestPubPrint_JSONAlwaysEmitKeys(t *testing.T) {
	t.Parallel()

	// Empty subject + 0 bytes honest (still always-emit keys).
	emptyDTO := NewPubPrint("", 0)
	emptyJS, err := json.Marshal(emptyDTO)
	if err != nil {
		t.Fatal(err)
	}
	var emptyObj map[string]any
	if err := json.Unmarshal(emptyJS, &emptyObj); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"ok", "subject", "bytes"} {
		if _, ok := emptyObj[key]; !ok {
			t.Fatalf("empty json missing key %q: %s", key, emptyJS)
		}
	}
	if emptyObj["ok"] != true {
		t.Fatalf("ok want true: %s", emptyJS)
	}
	if emptyObj["subject"] != "" {
		t.Fatalf("empty subject want \"\"; got %v\n%s", emptyObj["subject"], emptyJS)
	}
	if emptyObj["bytes"].(float64) != 0 {
		t.Fatalf("bytes want 0; got %v\n%s", emptyObj["bytes"], emptyJS)
	}
	// Do not invent pull_role or payload echo on pub JSON.
	if _, ok := emptyObj["pull_role"]; ok {
		t.Fatalf("must not invent pull_role on PubPrint: %s", emptyJS)
	}
	if _, ok := emptyObj["payload"]; ok {
		t.Fatalf("must not invent payload echo on PubPrint: %s", emptyJS)
	}

	popDTO := NewPubPrint("dept.agent.ping", 11)
	popJS, err := json.Marshal(popDTO)
	if err != nil {
		t.Fatal(err)
	}
	var popObj map[string]any
	if err := json.Unmarshal(popJS, &popObj); err != nil {
		t.Fatal(err)
	}
	if popObj["ok"] != true || popObj["subject"] != "dept.agent.ping" ||
		popObj["bytes"].(float64) != 11 {
		t.Fatalf("populated: %s", popJS)
	}

	// Format helpers round-trip keys for scrapers.
	formatted := FormatPubJSON(popDTO)
	if !strings.Contains(formatted, `"ok": true`) ||
		!strings.Contains(formatted, `"subject": "dept.agent.ping"`) ||
		!strings.Contains(formatted, `"bytes": 11`) {
		t.Fatalf("FormatPubJSON: %s", formatted)
	}
	if strings.Contains(formatted, "pull_role") || strings.Contains(formatted, `"payload"`) {
		t.Fatalf("FormatPubJSON must not invent pull_role/payload: %s", formatted)
	}
}

// s732: FormatPub always-emits subject/bytes (empty or populated).
func TestFormatPub_AlwaysEmit(t *testing.T) {
	t.Parallel()

	empty := FormatPub(NewPubPrint("", 0))
	for _, want := range []string{
		"PASS mesh pub\n",
		"subject: \n",
		"bytes:   0\n",
	} {
		if !strings.Contains(empty, want) {
			t.Fatalf("empty missing %q in:\n%q", want, empty)
		}
	}

	pop := FormatPub(NewPubPrint("dept.agent.ping", 11))
	if !strings.Contains(pop, "PASS mesh pub\n") ||
		!strings.Contains(pop, "subject: dept.agent.ping\n") ||
		!strings.Contains(pop, "bytes:   11\n") {
		t.Fatalf("populated:\n%s", pop)
	}
	if strings.Contains(pop, "pull_role") {
		t.Fatalf("text must not invent pull_role:\n%s", pop)
	}
}
