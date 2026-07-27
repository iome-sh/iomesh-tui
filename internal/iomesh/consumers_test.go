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

// s675: Role + PullAllowSuffix ride Client.auth on all authenticated mesh requests (e.g. CreateConsumer).
func TestCreateConsumer_RoleAndPullAllowSuffixHeaders(t *testing.T) {
	var gotRole, gotSuffix, gotTenant string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRole = r.Header.Get("X-IOMesh-Role")
		gotSuffix = r.Header.Get("X-IOMesh-Pull-Allow-Suffix")
		gotTenant = r.Header.Get("X-IOMesh-Tenant")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"stream": "EVENTS", "name": "c"})
	}))
	defer srv.Close()

	c := New(Config{
		Enabled:         true,
		Endpoint:        srv.URL,
		Tenant:          "dept.research",
		Role:            "custom",
		PullAllowSuffix: "ops,memory",
	}, nil)
	if _, err := c.CreateConsumer(context.Background(), "EVENTS", "c", "dept.research.>"); err != nil {
		t.Fatal(err)
	}
	if gotTenant != "dept.research" {
		t.Fatalf("X-IOMesh-Tenant=%q", gotTenant)
	}
	if gotRole != "custom" {
		t.Fatalf("X-IOMesh-Role=%q", gotRole)
	}
	if gotSuffix != "ops,memory" {
		t.Fatalf("X-IOMesh-Pull-Allow-Suffix=%q", gotSuffix)
	}
}

func TestCreateConsumer_OmitsRoleAndSuffixWhenEmpty(t *testing.T) {
	var gotRole, gotSuffix string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Header.Get is case-insensitive; empty means fail-open omit (not set / blank).
		gotRole = r.Header.Get("X-IOMesh-Role")
		gotSuffix = r.Header.Get("X-IOMesh-Pull-Allow-Suffix")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"stream": "S", "name": "c"})
	}))
	defer srv.Close()

	// Whitespace-only is fail-open omit (TrimSpace).
	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "t", Role: "  ", PullAllowSuffix: "\t"}, nil)
	if _, err := c.CreateConsumer(context.Background(), "S", "c", ""); err != nil {
		t.Fatal(err)
	}
	if gotRole != "" || gotSuffix != "" {
		t.Fatalf("expected no role/suffix headers when empty; role=%q suffix=%q", gotRole, gotSuffix)
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

func TestFormatConsumerInfo_EmptyFilterAlwaysEmit(t *testing.T) {
	out := FormatConsumerInfo(ConsumerInfo{Stream: "S", Name: "c", AckFloor: 1, PendingCount: 0})
	// Always emit filter_subject (blank when unset) for scrapers.
	if !strings.Contains(out, "filter_subject:  \n") {
		t.Fatalf("want blank filter_subject always emitted, got:\n%q", out)
	}
	if !strings.Contains(out, "stream:          S") || !strings.Contains(out, "name:            c") {
		t.Fatal(out)
	}
	// s696: FormatConsumerInfo always-emits blank pull identity when no auth passed.
	if !strings.Contains(out, "pull_role:       \n") {
		t.Fatalf("want blank pull_role always emitted, got:\n%q", out)
	}
	if !strings.Contains(out, "pull_allow_suffix: \n") {
		t.Fatalf("want blank pull_allow_suffix always emitted, got:\n%q", out)
	}
	// Set filter still prints value.
	out2 := FormatConsumerInfo(ConsumerInfo{
		Stream: "S", Name: "c", AckFloor: 1, PendingCount: 0, FilterSubject: "dept.events.>",
	})
	if !strings.Contains(out2, "filter_subject:  dept.events.>\n") {
		t.Fatalf("want set filter_subject, got:\n%s", out2)
	}
}

// s696: FormatConsumerInfoWithAuth always-emits pull_role / pull_allow_suffix
// (empty when unset; populated when set) next to filter_subject for CI scrapers.
func TestFormatConsumerInfoWithAuth_AlwaysEmitPullIdentity(t *testing.T) {
	t.Parallel()

	// Empty auth identity: blank lines always present.
	empty := FormatConsumerInfoWithAuth(ConsumerInfo{
		Stream: "EVENTS", Name: "c", AckFloor: 0, PendingCount: 0, FilterSubject: "",
	}, "", "")
	for _, want := range []string{
		"filter_subject:  \n",
		"pull_role:       \n",
		"pull_allow_suffix: \n",
	} {
		if !strings.Contains(empty, want) {
			t.Fatalf("empty auth missing %q in:\n%q", want, empty)
		}
	}
	// Order: filter_subject → pull_role → pull_allow_suffix
	fsIdx := strings.Index(empty, "filter_subject:")
	prIdx := strings.Index(empty, "pull_role:")
	psIdx := strings.Index(empty, "pull_allow_suffix:")
	if fsIdx < 0 || prIdx < 0 || psIdx < 0 || !(fsIdx < prIdx && prIdx < psIdx) {
		t.Fatalf("identity order want filter_subject < pull_role < pull_allow_suffix:\n%s", empty)
	}

	// Populated values.
	pop := FormatConsumerInfoWithAuth(ConsumerInfo{
		Stream: "EVENTS", Name: "agent-1", AckFloor: 1, PendingCount: 2,
		FilterSubject: "acme.events.>",
	}, "agent", "ops")
	for _, want := range []string{
		"filter_subject:  acme.events.>\n",
		"pull_role:       agent\n",
		"pull_allow_suffix: ops\n",
	} {
		if !strings.Contains(pop, want) {
			t.Fatalf("populated missing %q in:\n%s", want, pop)
		}
	}

	// custom + multi-suffix.
	custom := FormatConsumerInfoWithAuth(ConsumerInfo{
		Stream: "S", Name: "c", FilterSubject: "t.ops.>",
	}, "custom", "ops,memory")
	if !strings.Contains(custom, "pull_role:       custom\n") ||
		!strings.Contains(custom, "pull_allow_suffix: ops,memory\n") {
		t.Fatalf("custom role/suffix:\n%s", custom)
	}

	// FormatConsumerInfo delegates empty auth (same always-emit contract).
	base := FormatConsumerInfo(ConsumerInfo{Stream: "S", Name: "c"})
	withEmpty := FormatConsumerInfoWithAuth(ConsumerInfo{Stream: "S", Name: "c"}, "", "")
	if base != withEmpty {
		t.Fatalf("FormatConsumerInfo should equal WithAuth empty; base:\n%q\nwith:\n%q", base, withEmpty)
	}
}

// s696: ConsumerInfoPrint JSON always-emits pull_role / pull_allow_suffix
// (and filter_subject) without omitempty gaps; does not pollute wire ConsumerInfo.
func TestConsumerInfoPrint_JSONAlwaysEmitPullIdentity(t *testing.T) {
	t.Parallel()

	// Empty auth.
	emptyDTO := NewConsumerInfoPrint(ConsumerInfo{
		Stream: "EVENTS", Name: "c", AckFloor: 0, PendingCount: 0,
	}, "", "")
	emptyJS, err := json.Marshal(emptyDTO)
	if err != nil {
		t.Fatal(err)
	}
	var emptyObj map[string]any
	if err := json.Unmarshal(emptyJS, &emptyObj); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"stream", "name", "filter_subject", "ack_floor", "pending_count", "pull_role", "pull_allow_suffix"} {
		if _, ok := emptyObj[key]; !ok {
			t.Fatalf("empty json missing key %q: %s", key, emptyJS)
		}
	}
	if emptyObj["pull_role"] != "" || emptyObj["pull_allow_suffix"] != "" || emptyObj["filter_subject"] != "" {
		t.Fatalf("empty identity want \"\"; got pull_role=%v pull_allow_suffix=%v filter_subject=%v\n%s",
			emptyObj["pull_role"], emptyObj["pull_allow_suffix"], emptyObj["filter_subject"], emptyJS)
	}

	// Populated.
	popDTO := NewConsumerInfoPrint(ConsumerInfo{
		Stream: "EVENTS", Name: "agent-1", FilterSubject: "acme.events.>",
		AckFloor: 3, PendingCount: 1,
	}, "memory", "ops")
	popJS, err := json.Marshal(popDTO)
	if err != nil {
		t.Fatal(err)
	}
	var popObj map[string]any
	if err := json.Unmarshal(popJS, &popObj); err != nil {
		t.Fatal(err)
	}
	if popObj["pull_role"] != "memory" || popObj["pull_allow_suffix"] != "ops" {
		t.Fatalf("populated pull identity: %v %v\n%s", popObj["pull_role"], popObj["pull_allow_suffix"], popJS)
	}
	if popObj["filter_subject"] != "acme.events.>" || popObj["name"] != "agent-1" {
		t.Fatalf("populated consumer fields: %s", popJS)
	}

	// Wire ConsumerInfo still omits empty filter_subject (omitempty) — print DTO is separate.
	wire, err := json.Marshal(ConsumerInfo{Stream: "S", Name: "c"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(wire), "pull_role") || strings.Contains(string(wire), "filter_subject") {
		t.Fatalf("wire ConsumerInfo should not carry pull_role / empty filter_subject: %s", wire)
	}
}

// s741: FormatConsumerInfoJSON keys present + trailing newline (DTO already always-emit s696).
func TestFormatConsumerInfoJSON_KeysAndNewline(t *testing.T) {
	t.Parallel()

	js := FormatConsumerInfoJSON(NewConsumerInfoPrint(ConsumerInfo{
		Stream: "EVENTS", Name: "c",
	}, "", ""))
	if !strings.HasSuffix(js, "\n") {
		t.Fatal("expected trailing newline")
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(js), &obj); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, js)
	}
	for _, key := range []string{"stream", "name", "filter_subject", "ack_floor", "pending_count", "pull_role", "pull_allow_suffix"} {
		if _, ok := obj[key]; !ok {
			t.Fatalf("missing key %q: %s", key, js)
		}
	}
	if obj["stream"] != "EVENTS" || obj["name"] != "c" {
		t.Fatalf("identity: %s", js)
	}
	if obj["pull_role"] != "" || obj["pull_allow_suffix"] != "" || obj["filter_subject"] != "" {
		t.Fatalf("empty identity want \"\": %s", js)
	}
}

// s708: ConsumerFetchPrint JSON always-emits pull identity + knobs + count
// without omitempty gaps; s723 nested StreamMessagePrint always-emit; wire lean.
func TestConsumerFetchPrint_JSONAlwaysEmitPullIdentity(t *testing.T) {
	t.Parallel()

	// Empty auth + nil messages → empty slice, count 0, blank identity.
	emptyDTO := NewConsumerFetchPrint("EVENTS", "worker-1", "", "", 1, 2000, nil)
	emptyJS, err := json.Marshal(emptyDTO)
	if err != nil {
		t.Fatal(err)
	}
	var emptyObj map[string]any
	if err := json.Unmarshal(emptyJS, &emptyObj); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"stream", "name", "pull_role", "pull_allow_suffix",
		"batch", "max_wait_ms", "count", "messages",
	} {
		if _, ok := emptyObj[key]; !ok {
			t.Fatalf("empty json missing key %q: %s", key, emptyJS)
		}
	}
	if emptyObj["pull_role"] != "" || emptyObj["pull_allow_suffix"] != "" {
		t.Fatalf("empty identity want \"\"; got pull_role=%v pull_allow_suffix=%v\n%s",
			emptyObj["pull_role"], emptyObj["pull_allow_suffix"], emptyJS)
	}
	if emptyObj["stream"] != "EVENTS" || emptyObj["name"] != "worker-1" {
		t.Fatalf("identity stream/name: %s", emptyJS)
	}
	if emptyObj["batch"] != float64(1) || emptyObj["max_wait_ms"] != float64(2000) {
		t.Fatalf("knobs: %s", emptyJS)
	}
	if emptyObj["count"] != float64(0) {
		t.Fatalf("count want 0: %s", emptyJS)
	}
	msgs, ok := emptyObj["messages"].([]any)
	if !ok || len(msgs) != 0 {
		t.Fatalf("messages want empty array: %s", emptyJS)
	}

	// Populated role/suffix + one sparse message → nested always-emit keys (s723).
	popMsgs := []StreamMessage{
		{Seq: 7, Subject: "dept.events.hello", Payload: []byte("hi")},
	}
	popDTO := NewConsumerFetchPrint("EVENTS", "agent-1", "agent", "ops", 5, 2000, popMsgs)
	popJS, err := json.Marshal(popDTO)
	if err != nil {
		t.Fatal(err)
	}
	var popObj map[string]any
	if err := json.Unmarshal(popJS, &popObj); err != nil {
		t.Fatal(err)
	}
	if popObj["pull_role"] != "agent" || popObj["pull_allow_suffix"] != "ops" {
		t.Fatalf("populated pull identity: %v %v\n%s", popObj["pull_role"], popObj["pull_allow_suffix"], popJS)
	}
	if popObj["count"] != float64(1) || popObj["batch"] != float64(5) {
		t.Fatalf("count/batch: %s", popJS)
	}
	popList, ok := popObj["messages"].([]any)
	if !ok || len(popList) != 1 {
		t.Fatalf("messages: %s", popJS)
	}
	nested, ok := popList[0].(map[string]any)
	if !ok {
		t.Fatalf("nested message not object: %s", popJS)
	}
	for _, key := range []string{"stream", "seq", "subject", "partition", "payload", "headers", "timestamp"} {
		if _, ok := nested[key]; !ok {
			t.Fatalf("nested missing always-emit key %q: %s", key, popJS)
		}
	}
	if nested["stream"] != "" || nested["partition"] != float64(0) || nested["timestamp"] != "" {
		t.Fatalf("nested empty/0 honest: %s", popJS)
	}
	// custom + multi-suffix still always present as strings.
	customDTO := NewConsumerFetchPrint("S", "c", "custom", "ops,memory", 1, 500, []StreamMessage{})
	customJS, err := json.Marshal(customDTO)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(customJS), `"pull_role":"custom"`) ||
		!strings.Contains(string(customJS), `"pull_allow_suffix":"ops,memory"`) {
		t.Fatalf("custom role/suffix: %s", customJS)
	}

	// Wire StreamMessage slice JSON still has no pull_role (print DTO is separate).
	wire, err := json.Marshal([]StreamMessage{{Seq: 1, Subject: "s"}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(wire), "pull_role") {
		t.Fatalf("wire []StreamMessage should not carry pull_role: %s", wire)
	}
}

// s708: FormatConsumerFetch always-emits pull identity + knobs + count (empty or populated).
func TestFormatConsumerFetch_AlwaysEmitPullIdentity(t *testing.T) {
	t.Parallel()

	empty := FormatConsumerFetch(NewConsumerFetchPrint("EVENTS", "c", "", "", 1, 2000, nil))
	for _, want := range []string{
		"iomesh consumer fetch\n",
		"stream:            EVENTS\n",
		"name:              c\n",
		"pull_role:         \n",
		"pull_allow_suffix: \n",
		"batch:             1\n",
		"max_wait_ms:       2000\n",
		"count:             0\n",
		"name=EVENTS/c count=0",
	} {
		if !strings.Contains(empty, want) {
			t.Fatalf("empty missing %q in:\n%q", want, empty)
		}
	}
	// Order: stream → name → pull_role → pull_allow_suffix
	sIdx := strings.Index(empty, "stream:")
	nIdx := strings.Index(empty, "name:")
	prIdx := strings.Index(empty, "pull_role:")
	psIdx := strings.Index(empty, "pull_allow_suffix:")
	if sIdx < 0 || nIdx < 0 || prIdx < 0 || psIdx < 0 || !(sIdx < nIdx && nIdx < prIdx && prIdx < psIdx) {
		t.Fatalf("identity order want stream < name < pull_role < pull_allow_suffix:\n%s", empty)
	}

	pop := FormatConsumerFetch(NewConsumerFetchPrint(
		"EVENTS", "agent-1", "memory", "ops",
		5, 2000,
		[]StreamMessage{{Seq: 3, Subject: "dept.events.x", Payload: []byte("body")}},
	))
	for _, want := range []string{
		"pull_role:         memory\n",
		"pull_allow_suffix: ops\n",
		"batch:             5\n",
		"count:             1\n",
		"dept.events.x",
	} {
		if !strings.Contains(pop, want) {
			t.Fatalf("populated missing %q in:\n%s", want, pop)
		}
	}
}

// s708: ConsumerDeletePrint JSON always-emits {ok,stream,name,pull_role,pull_allow_suffix}.
func TestConsumerDeletePrint_JSONAlwaysEmitPullIdentity(t *testing.T) {
	t.Parallel()

	emptyDTO := NewConsumerDeletePrint("EVENTS", "worker-1", "", "")
	emptyJS, err := json.Marshal(emptyDTO)
	if err != nil {
		t.Fatal(err)
	}
	var emptyObj map[string]any
	if err := json.Unmarshal(emptyJS, &emptyObj); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"ok", "stream", "name", "pull_role", "pull_allow_suffix"} {
		if _, ok := emptyObj[key]; !ok {
			t.Fatalf("empty json missing key %q: %s", key, emptyJS)
		}
	}
	if emptyObj["ok"] != true {
		t.Fatalf("ok want true: %s", emptyJS)
	}
	if emptyObj["pull_role"] != "" || emptyObj["pull_allow_suffix"] != "" {
		t.Fatalf("empty identity want \"\"; got %v %v\n%s",
			emptyObj["pull_role"], emptyObj["pull_allow_suffix"], emptyJS)
	}
	if emptyObj["stream"] != "EVENTS" || emptyObj["name"] != "worker-1" {
		t.Fatalf("stream/name: %s", emptyJS)
	}

	popDTO := NewConsumerDeletePrint("EVENTS", "agent-1", "agent", "ops")
	popJS, err := json.Marshal(popDTO)
	if err != nil {
		t.Fatal(err)
	}
	var popObj map[string]any
	if err := json.Unmarshal(popJS, &popObj); err != nil {
		t.Fatal(err)
	}
	if popObj["pull_role"] != "agent" || popObj["pull_allow_suffix"] != "ops" || popObj["ok"] != true {
		t.Fatalf("populated: %s", popJS)
	}

	// Format helpers round-trip keys for scrapers.
	formatted := FormatConsumerDeleteJSON(popDTO)
	if !strings.Contains(formatted, `"pull_role": "agent"`) ||
		!strings.Contains(formatted, `"pull_allow_suffix": "ops"`) {
		t.Fatalf("FormatConsumerDeleteJSON: %s", formatted)
	}
}

// s708: FormatConsumerDelete always-emits pull identity (empty or populated).
func TestFormatConsumerDelete_AlwaysEmitPullIdentity(t *testing.T) {
	t.Parallel()

	empty := FormatConsumerDelete(NewConsumerDeletePrint("EVENTS", "c", "", ""))
	for _, want := range []string{
		"PASS mesh consumer delete\n",
		"stream:            EVENTS\n",
		"name:              c\n",
		"pull_role:         \n",
		"pull_allow_suffix: \n",
	} {
		if !strings.Contains(empty, want) {
			t.Fatalf("empty missing %q in:\n%q", want, empty)
		}
	}

	pop := FormatConsumerDelete(NewConsumerDeletePrint("S", "mem-1", "custom", "ops,memory"))
	if !strings.Contains(pop, "pull_role:         custom\n") ||
		!strings.Contains(pop, "pull_allow_suffix: ops,memory\n") {
		t.Fatalf("populated:\n%s", pop)
	}
}

// s711: ConsumerAckPrint JSON always-emits {ok,op,stream,name,pull_role,pull_allow_suffix,seqs,ack_floor,count}.
func TestConsumerAckPrint_JSONAlwaysEmitPullIdentity(t *testing.T) {
	t.Parallel()

	// Empty auth + single seq → blank identity, count 1.
	emptyDTO := NewConsumerAckPrint("ack", "EVENTS", "worker-1", "", "", []uint64{1}, 0)
	emptyJS, err := json.Marshal(emptyDTO)
	if err != nil {
		t.Fatal(err)
	}
	var emptyObj map[string]any
	if err := json.Unmarshal(emptyJS, &emptyObj); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"ok", "op", "stream", "name", "pull_role", "pull_allow_suffix",
		"seqs", "ack_floor", "count",
	} {
		if _, ok := emptyObj[key]; !ok {
			t.Fatalf("empty json missing key %q: %s", key, emptyJS)
		}
	}
	if emptyObj["ok"] != true {
		t.Fatalf("ok want true: %s", emptyJS)
	}
	if emptyObj["op"] != "ack" {
		t.Fatalf("op want ack: %s", emptyJS)
	}
	if emptyObj["pull_role"] != "" || emptyObj["pull_allow_suffix"] != "" {
		t.Fatalf("empty identity want \"\"; got pull_role=%v pull_allow_suffix=%v\n%s",
			emptyObj["pull_role"], emptyObj["pull_allow_suffix"], emptyJS)
	}
	if emptyObj["stream"] != "EVENTS" || emptyObj["name"] != "worker-1" {
		t.Fatalf("stream/name: %s", emptyJS)
	}
	if emptyObj["ack_floor"] != float64(0) || emptyObj["count"] != float64(1) {
		t.Fatalf("ack_floor/count: %s", emptyJS)
	}
	seqs, ok := emptyObj["seqs"].([]any)
	if !ok || len(seqs) != 1 || seqs[0] != float64(1) {
		t.Fatalf("seqs want [1]: %s", emptyJS)
	}

	// Populated role/suffix + multi seq + floor + nack op.
	popDTO := NewConsumerAckPrint("nack", "EVENTS", "agent-1", "agent", "ops", []uint64{2, 3}, 3)
	popJS, err := json.Marshal(popDTO)
	if err != nil {
		t.Fatal(err)
	}
	var popObj map[string]any
	if err := json.Unmarshal(popJS, &popObj); err != nil {
		t.Fatal(err)
	}
	if popObj["pull_role"] != "agent" || popObj["pull_allow_suffix"] != "ops" || popObj["ok"] != true {
		t.Fatalf("populated: %s", popJS)
	}
	if popObj["op"] != "nack" || popObj["ack_floor"] != float64(3) || popObj["count"] != float64(2) {
		t.Fatalf("op/floor/count: %s", popJS)
	}

	// custom + multi-suffix still always present as strings.
	customDTO := NewConsumerAckPrint("ack", "S", "c", "custom", "ops,memory", []uint64{9}, 9)
	customJS, err := json.Marshal(customDTO)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(customJS), `"pull_role":"custom"`) ||
		!strings.Contains(string(customJS), `"pull_allow_suffix":"ops,memory"`) {
		t.Fatalf("custom role/suffix: %s", customJS)
	}

	// Nil seqs → empty array, count 0 (honest empty; CLI requires seqs before call).
	nilDTO := NewConsumerAckPrint("ack", "S", "c", "", "", nil, 0)
	nilJS, err := json.Marshal(nilDTO)
	if err != nil {
		t.Fatal(err)
	}
	var nilObj map[string]any
	if err := json.Unmarshal(nilJS, &nilObj); err != nil {
		t.Fatal(err)
	}
	nilSeqs, ok := nilObj["seqs"].([]any)
	if !ok || len(nilSeqs) != 0 {
		t.Fatalf("nil seqs want empty array: %s", nilJS)
	}
	if nilObj["count"] != float64(0) {
		t.Fatalf("nil count want 0: %s", nilJS)
	}

	// Format helpers round-trip keys for scrapers.
	formatted := FormatConsumerAckJSON(popDTO)
	if !strings.Contains(formatted, `"pull_role": "agent"`) ||
		!strings.Contains(formatted, `"pull_allow_suffix": "ops"`) ||
		!strings.Contains(formatted, `"op": "nack"`) {
		t.Fatalf("FormatConsumerAckJSON: %s", formatted)
	}
}

// s711: FormatConsumerAck always-emits pull identity + op/seqs/ack_floor/count (empty or populated).
func TestFormatConsumerAck_AlwaysEmitPullIdentity(t *testing.T) {
	t.Parallel()

	empty := FormatConsumerAck(NewConsumerAckPrint("ack", "EVENTS", "c", "", "", []uint64{1}, 0))
	for _, want := range []string{
		"PASS mesh consumer ack\n",
		"op:                ack\n",
		"stream:            EVENTS\n",
		"name:              c\n",
		"pull_role:         \n",
		"pull_allow_suffix: \n",
		"seqs:              1\n",
		"ack_floor:         0\n",
		"count:             1\n",
	} {
		if !strings.Contains(empty, want) {
			t.Fatalf("empty missing %q in:\n%q", want, empty)
		}
	}
	// Order: op → stream → name → pull_role → pull_allow_suffix
	opIdx := strings.Index(empty, "op:")
	sIdx := strings.Index(empty, "stream:")
	nIdx := strings.Index(empty, "name:")
	prIdx := strings.Index(empty, "pull_role:")
	psIdx := strings.Index(empty, "pull_allow_suffix:")
	if opIdx < 0 || sIdx < 0 || nIdx < 0 || prIdx < 0 || psIdx < 0 ||
		!(opIdx < sIdx && sIdx < nIdx && nIdx < prIdx && prIdx < psIdx) {
		t.Fatalf("identity order want op < stream < name < pull_role < pull_allow_suffix:\n%s", empty)
	}

	pop := FormatConsumerAck(NewConsumerAckPrint(
		"nack", "EVENTS", "agent-1", "memory", "ops",
		[]uint64{2, 5}, 5,
	))
	for _, want := range []string{
		"PASS mesh consumer nack\n",
		"op:                nack\n",
		"pull_role:         memory\n",
		"pull_allow_suffix: ops\n",
		"seqs:              2,5\n",
		"ack_floor:         5\n",
		"count:             2\n",
	} {
		if !strings.Contains(pop, want) {
			t.Fatalf("populated missing %q in:\n%s", want, pop)
		}
	}

	custom := FormatConsumerAck(NewConsumerAckPrint("ack", "S", "mem-1", "custom", "ops,memory", []uint64{7}, 7))
	if !strings.Contains(custom, "pull_role:         custom\n") ||
		!strings.Contains(custom, "pull_allow_suffix: ops,memory\n") {
		t.Fatalf("custom:\n%s", custom)
	}
}

// s684: ResolveMeshPullAuth is the pure flag>config path for fetch/ack/nack/delete (and create).
func TestResolveMeshPullAuth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		roleFlag     string
		suffixFlag   string
		configRole   string
		configSuffix string
		wantRole     string
		wantSuffix   string
	}{
		{name: "flag wins", roleFlag: "agent", suffixFlag: "ops", configRole: "admin", configSuffix: "memory", wantRole: "agent", wantSuffix: "ops"},
		{name: "config when flags empty", configRole: "viewer", configSuffix: "a,b", wantRole: "viewer", wantSuffix: "a,b"},
		{name: "whitespace flags fall back to config", roleFlag: "  ", suffixFlag: "\t", configRole: "auditor", configSuffix: "x", wantRole: "auditor", wantSuffix: "x"},
		{name: "empty fail-open"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotR, gotS := ResolveMeshPullAuth(tt.roleFlag, tt.suffixFlag, tt.configRole, tt.configSuffix)
			if gotR != tt.wantRole || gotS != tt.wantSuffix {
				t.Fatalf("got role=%q suffix=%q want role=%q suffix=%q", gotR, gotS, tt.wantRole, tt.wantSuffix)
			}
		})
	}
}

// s681: mesh consumer create resolves role/suffix (flag > config) and role-aware empty filter
// via DefaultMemoryPullFilterForRole (same pure path as memory pull s678).
func TestResolveConsumerCreateAuthAndFilter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		explicit        string
		tenant          string
		roleFlag        string
		suffixFlag      string
		configRole      string
		configSuffix    string
		wantFilter      string
		wantRole        string
		wantAllowSuffix string
	}{
		{
			name:   "agent empty filter → tenant.events.>",
			tenant: "acme", roleFlag: "agent",
			wantFilter: "acme.events.>", wantRole: "agent",
		},
		{
			name:     "flag role overrides config; explicit filter wins",
			explicit: "dept.ops.>", tenant: "acme",
			roleFlag: "viewer", configRole: "admin",
			wantFilter: "dept.ops.>", wantRole: "viewer",
		},
		{
			name:   "config role/suffix when flags empty",
			tenant: "dept.research", configRole: "custom", configSuffix: "memory",
			wantFilter: "dept.research.memory.>", wantRole: "custom", wantAllowSuffix: "memory",
		},
		{
			name:   "flag suffix overrides config",
			tenant: "t", roleFlag: "custom", suffixFlag: "ops", configSuffix: "memory",
			wantFilter: "t.ops.>", wantRole: "custom", wantAllowSuffix: "ops",
		},
		{
			name:       "empty role fail-open; hierarchical tenant s660 default",
			tenant:     "dept.research",
			wantFilter: "dept.research.>",
		},
		{
			name:   "whitespace flags fall back to config",
			tenant: "acme", roleFlag: "  ", configRole: "auditor",
			wantFilter: "acme.audit.>", wantRole: "auditor",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotF, gotR, gotS := ResolveConsumerCreateAuthAndFilter(
				tt.explicit, tt.tenant, tt.roleFlag, tt.suffixFlag, tt.configRole, tt.configSuffix,
			)
			if gotF != tt.wantFilter || gotR != tt.wantRole || gotS != tt.wantAllowSuffix {
				t.Fatalf("got filter=%q role=%q suffix=%q want filter=%q role=%q suffix=%q",
					gotF, gotR, gotS, tt.wantFilter, tt.wantRole, tt.wantAllowSuffix)
			}
		})
	}
}

// s681 end-to-end pure path: empty filter + role=agent feeds CreateConsumer body via resolved filter.
func TestCreateConsumer_ResolvedAgentDefaultFilter(t *testing.T) {
	var gotBody map[string]any
	var gotRole string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRole = r.Header.Get("X-IOMesh-Role")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"stream": "EVENTS", "name": "agent-1", "filter_subject": gotBody["filter_subject"]})
	}))
	defer srv.Close()

	filter, role, suffix := ResolveConsumerCreateAuthAndFilter("", "acme", "agent", "", "", "")
	if filter != "acme.events.>" || role != "agent" || suffix != "" {
		t.Fatalf("resolve filter=%q role=%q suffix=%q", filter, role, suffix)
	}
	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "acme", Role: role, PullAllowSuffix: suffix}, nil)
	info, err := c.CreateConsumer(context.Background(), "EVENTS", "agent-1", filter)
	if err != nil {
		t.Fatal(err)
	}
	if gotRole != "agent" {
		t.Fatalf("X-IOMesh-Role=%q", gotRole)
	}
	if gotBody["filter_subject"] != "acme.events.>" {
		t.Fatalf("body filter_subject=%v", gotBody["filter_subject"])
	}
	if info == nil || info.Name != "agent-1" {
		t.Fatalf("info=%+v", info)
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

// s684: Role + PullAllowSuffix ride Client.auth on ConsumerFetch (federated ACL on broker fetch path).
func TestConsumerFetch_RoleAndPullAllowSuffixHeaders(t *testing.T) {
	var gotRole, gotSuffix, gotTenant string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRole = r.Header.Get("X-IOMesh-Role")
		gotSuffix = r.Header.Get("X-IOMesh-Pull-Allow-Suffix")
		gotTenant = r.Header.Get("X-IOMesh-Tenant")
		_ = json.NewEncoder(w).Encode(map[string]any{"messages": []any{}})
	}))
	defer srv.Close()

	role, suffix := ResolveMeshPullAuth("custom", "ops,memory", "agent", "ignored")
	c := New(Config{
		Enabled:         true,
		Endpoint:        srv.URL,
		Tenant:          "dept.research",
		Role:            role,
		PullAllowSuffix: suffix,
	}, nil)
	if _, err := c.ConsumerFetch(context.Background(), "EVENTS", "c", 1, time.Second); err != nil {
		t.Fatal(err)
	}
	if gotTenant != "dept.research" {
		t.Fatalf("X-IOMesh-Tenant=%q", gotTenant)
	}
	if gotRole != "custom" {
		t.Fatalf("X-IOMesh-Role=%q", gotRole)
	}
	if gotSuffix != "ops,memory" {
		t.Fatalf("X-IOMesh-Pull-Allow-Suffix=%q", gotSuffix)
	}
}

func TestConsumerFetch_OmitsRoleAndSuffixWhenEmpty(t *testing.T) {
	var gotRole, gotSuffix string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRole = r.Header.Get("X-IOMesh-Role")
		gotSuffix = r.Header.Get("X-IOMesh-Pull-Allow-Suffix")
		_ = json.NewEncoder(w).Encode(map[string]any{"messages": []any{}})
	}))
	defer srv.Close()

	role, suffix := ResolveMeshPullAuth("  ", "\t", "", "")
	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "t", Role: role, PullAllowSuffix: suffix}, nil)
	if _, err := c.ConsumerFetch(context.Background(), "S", "c", 1, time.Second); err != nil {
		t.Fatal(err)
	}
	if gotRole != "" || gotSuffix != "" {
		t.Fatalf("expected no role/suffix headers when empty; role=%q suffix=%q", gotRole, gotSuffix)
	}
}

// s684 defense-in-depth: ack also carries Role/PullAllowSuffix via Client.auth.
func TestConsumerAck_RoleAndPullAllowSuffixHeaders(t *testing.T) {
	var gotRole, gotSuffix string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRole = r.Header.Get("X-IOMesh-Role")
		gotSuffix = r.Header.Get("X-IOMesh-Pull-Allow-Suffix")
		_ = json.NewEncoder(w).Encode(map[string]any{"ack_floor": 1})
	}))
	defer srv.Close()

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "t",
		Role: "agent", PullAllowSuffix: "memory",
	}, nil)
	if _, err := c.ConsumerAck(context.Background(), "S", "c", 1); err != nil {
		t.Fatal(err)
	}
	if gotRole != "agent" {
		t.Fatalf("X-IOMesh-Role=%q", gotRole)
	}
	if gotSuffix != "memory" {
		t.Fatalf("X-IOMesh-Pull-Allow-Suffix=%q", gotSuffix)
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

func TestConsumerAck_OKBodyPath(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if r.Method != http.MethodPost || r.URL.Path != "/v1/streams/EVENTS/consumers/worker-1/ack" {
			http.NotFound(w, r)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{"ack_floor": 9})
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	floor, err := c.ConsumerAck(context.Background(), "EVENTS", "worker-1", 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method=%q", gotMethod)
	}
	if gotPath != "/v1/streams/EVENTS/consumers/worker-1/ack" {
		t.Fatalf("path=%q", gotPath)
	}
	seqs, ok := gotBody["seqs"].([]any)
	if !ok || len(seqs) != 2 || seqs[0] != float64(1) || seqs[1] != float64(2) {
		t.Fatalf("body seqs=%v", gotBody["seqs"])
	}
	if floor != 9 {
		t.Fatalf("ack_floor=%d want 9", floor)
	}
}

func TestConsumerNack_Path(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	floor, err := c.ConsumerNack(context.Background(), "EVENTS", "worker-1", 3)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/streams/EVENTS/consumers/worker-1/nack" {
		t.Fatalf("path=%q", gotPath)
	}
	seqs, ok := gotBody["seqs"].([]any)
	if !ok || len(seqs) != 1 || seqs[0] != float64(3) {
		t.Fatalf("body seqs=%v", gotBody["seqs"])
	}
	if floor != 0 {
		t.Fatalf("empty body floor=%d want 0", floor)
	}
}

func TestConsumerAckNack_EmptySeqsAndDisabled(t *testing.T) {
	c := New(Config{Enabled: true, Endpoint: "http://127.0.0.1:9"}, nil)
	if _, err := c.ConsumerAck(context.Background(), "S", "c"); err == nil || !strings.Contains(err.Error(), "seqs required") {
		t.Fatalf("empty seqs ack err=%v", err)
	}
	if _, err := c.ConsumerNack(context.Background(), "S", "c"); err == nil || !strings.Contains(err.Error(), "seqs required") {
		t.Fatalf("empty seqs nack err=%v", err)
	}
	if _, err := c.ConsumerAck(context.Background(), "", "c", 1); err == nil || !strings.Contains(err.Error(), "stream and name required") {
		t.Fatalf("empty stream err=%v", err)
	}
	off := New(Config{Enabled: false}, nil)
	if _, err := off.ConsumerAck(context.Background(), "S", "c", 1); err == nil || !strings.Contains(err.Error(), "mesh disabled") {
		t.Fatalf("disabled ack err=%v", err)
	}
	if _, err := off.ConsumerNack(context.Background(), "S", "c", 1); err == nil || !strings.Contains(err.Error(), "mesh disabled") {
		t.Fatalf("disabled nack err=%v", err)
	}
}

func TestConsumerAck_HTTPErrorAndPathEscape(t *testing.T) {
	srvErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srvErr.Close()
	cErr := New(Config{Enabled: true, Endpoint: srvErr.URL}, nil)
	if _, err := cErr.ConsumerAck(context.Background(), "S", "c", 1); err == nil || !strings.Contains(err.Error(), "http 500") {
		t.Fatalf("http err=%v", err)
	}

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	if _, err := c.ConsumerAck(context.Background(), "a/b", "c/d", 1); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/streams/a%2Fb/consumers/c%2Fd/ack" {
		t.Fatalf("path=%q want escaped stream+name", gotPath)
	}
}

func TestDeleteConsumer_204(t *testing.T) {
	prev := UserAgent()
	SetUserAgent("iomesh-tui/test-consumer-delete")
	t.Cleanup(func() { SetUserAgent(prev) })

	var gotMethod, gotPath, gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotUA = r.Header.Get("User-Agent")
		if r.Method != http.MethodDelete || r.URL.Path != "/v1/streams/EVENTS/consumers/worker-1" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "acme"}, nil)
	if err := c.DeleteConsumer(context.Background(), "EVENTS", "worker-1"); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodDelete {
		t.Fatalf("method=%q", gotMethod)
	}
	if gotPath != "/v1/streams/EVENTS/consumers/worker-1" {
		t.Fatalf("path=%q", gotPath)
	}
	if gotUA != "iomesh-tui/test-consumer-delete" {
		t.Fatalf("User-Agent=%q", gotUA)
	}
}

func TestDeleteConsumer_EmptyArgsAndDisabled(t *testing.T) {
	c := New(Config{Enabled: true, Endpoint: "http://127.0.0.1:9"}, nil)
	if err := c.DeleteConsumer(context.Background(), "", "c"); err == nil || !strings.Contains(err.Error(), "stream and name required") {
		t.Fatalf("empty stream err=%v", err)
	}
	if err := c.DeleteConsumer(context.Background(), "S", "  "); err == nil || !strings.Contains(err.Error(), "stream and name required") {
		t.Fatalf("blank name err=%v", err)
	}
	off := New(Config{Enabled: false}, nil)
	if err := off.DeleteConsumer(context.Background(), "S", "c"); err == nil || !strings.Contains(err.Error(), "mesh disabled") {
		t.Fatalf("disabled err=%v", err)
	}
}

func TestDeleteConsumer_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	err := c.DeleteConsumer(context.Background(), "S", "c")
	if err == nil || !strings.Contains(err.Error(), "http 500") {
		t.Fatalf("err=%v", err)
	}
}

func TestDeleteConsumer_PathEscape(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	if err := c.DeleteConsumer(context.Background(), "a/b", "c/d"); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/streams/a%2Fb/consumers/c%2Fd" {
		t.Fatalf("path=%q want escaped stream+name", gotPath)
	}
}
