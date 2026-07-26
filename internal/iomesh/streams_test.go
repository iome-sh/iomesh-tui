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
	// s699: list table always emits MAX_MSGS / MAX_AGE column headers for scrapers.
	// s702: TIER (retention_tier) column always present (empty when broker omits).
	if !strings.Contains(out, "MAX_MSGS") || !strings.Contains(out, "MAX_AGE") {
		t.Fatalf("want MAX_MSGS/MAX_AGE headers, got:\n%s", out)
	}
	if !strings.Contains(out, "TIER") {
		t.Fatalf("want TIER header, got:\n%s", out)
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
		Name:          "EVENTS",
		Description:   "ops events",
		Retention:     "limits",
		RetentionTier: "temp",
		Partitions:    1,
		MaxMsgs:       &max,
		MaxAgeSec:     &age,
		Messages:      10,
		FirstSeq:      1,
		LastSeq:       10,
		CreatedAt:     time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
		Subjects:      []string{"dept.events.>", "dept.ops.>"},
	})
	for _, want := range []string{
		"iomesh stream",
		"name:           EVENTS",
		"description:    ops events",
		"retention:      limits",
		"retention_tier: temp",
		"partitions:     1",
		"max_msgs:       1000",
		"max_age_sec:    3600",
		"messages:       10",
		"first_seq:      1",
		"last_seq:       10",
		"created_at:     2026-07-01T12:00:00Z",
		"subjects:",
		"dept.events.>",
		"dept.ops.>",
	} {
		if !strings.Contains(detail, want) {
			t.Fatalf("missing %q in:\n%s", want, detail)
		}
	}
}

// s699/s702: Empty / zero StreamInfo still always-emits every scraper key.
// max_msgs / max_age_sec print numeric 0 when *int64 nil (honest zero, not omit).
// retention_tier always-emits empty string when unset (decode real wire; never invent from max_age).
func TestFormatStreamDetail_EmptyAlwaysEmit(t *testing.T) {
	detail := FormatStreamDetail(StreamInfo{
		Name: "SPARSE",
	})
	for _, want := range []string{
		"iomesh stream",
		"name:           SPARSE",
		"description:    \n",
		"retention:      \n",
		"retention_tier: \n",
		"partitions:     0\n",
		"max_msgs:       0\n",
		"max_age_sec:    0\n",
		"messages:       0\n",
		"first_seq:      0\n",
		"last_seq:       0\n",
		"created_at:     \n",
		"subjects:\n",
		"  (none)\n",
	} {
		if !strings.Contains(detail, want) {
			t.Fatalf("missing %q in:\n%s", want, detail)
		}
	}
}

// s702: populated retention_tier prints as-is; max_age alone does not invent tier.
func TestFormatStreamDetail_RetentionTier(t *testing.T) {
	age := int64(604800) // 7d — must not invent "temp" from max_age alone
	emptyTier := FormatStreamDetail(StreamInfo{
		Name:      "AGED",
		MaxAgeSec: &age,
	})
	if !strings.Contains(emptyTier, "retention_tier: \n") {
		t.Fatalf("want empty retention_tier when unset, got:\n%s", emptyTier)
	}
	if strings.Contains(emptyTier, "retention_tier: temp") || strings.Contains(emptyTier, "retention_tier: hot") {
		t.Fatalf("must not invent tier from max_age alone, got:\n%s", emptyTier)
	}

	pop := FormatStreamDetail(StreamInfo{
		Name:          "TEMP",
		RetentionTier: "temp",
		MaxAgeSec:     &age,
	})
	if !strings.Contains(pop, "retention_tier: temp\n") {
		t.Fatalf("want populated retention_tier, got:\n%s", pop)
	}
}

// s699: explicit zero *int64 still prints 0 (same as nil → 0).
func TestFormatStreamDetail_ZeroPointerKnobs(t *testing.T) {
	zero := int64(0)
	detail := FormatStreamDetail(StreamInfo{
		Name:      "ZEROED",
		MaxMsgs:   &zero,
		MaxAgeSec: &zero,
	})
	for _, want := range []string{
		"max_msgs:       0\n",
		"max_age_sec:    0\n",
		"description:    \n",
		"retention:      \n",
		"retention_tier: \n",
		"partitions:     0\n",
	} {
		if !strings.Contains(detail, want) {
			t.Fatalf("missing %q in:\n%s", want, detail)
		}
	}
}

// s699/s702: StreamInfoPrint JSON always-emits retention knobs + retention_tier (empty/0 when unset).
func TestStreamInfoPrint_AlwaysEmit(t *testing.T) {
	// Empty / nil knobs — retention_tier always present as ""
	emptyJS, err := json.Marshal(NewStreamInfoPrint(StreamInfo{Name: "SPARSE"}))
	if err != nil {
		t.Fatal(err)
	}
	var emptyObj map[string]any
	if err := json.Unmarshal(emptyJS, &emptyObj); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"name", "description", "retention", "retention_tier", "partitions", "max_msgs", "max_age_sec",
		"messages", "first_seq", "last_seq", "created_at", "subjects",
	} {
		if _, ok := emptyObj[key]; !ok {
			t.Fatalf("missing always-emit key %q in %s", key, emptyJS)
		}
	}
	if emptyObj["description"] != "" || emptyObj["retention"] != "" || emptyObj["retention_tier"] != "" || emptyObj["created_at"] != "" {
		t.Fatalf("empty strings want \"\"; got %v", emptyObj)
	}
	if emptyObj["partitions"].(float64) != 0 || emptyObj["max_msgs"].(float64) != 0 || emptyObj["max_age_sec"].(float64) != 0 {
		t.Fatalf("numeric knobs want 0; got partitions=%v max_msgs=%v max_age_sec=%v",
			emptyObj["partitions"], emptyObj["max_msgs"], emptyObj["max_age_sec"])
	}
	subs, ok := emptyObj["subjects"].([]any)
	if !ok || len(subs) != 0 {
		t.Fatalf("subjects want []; got %v", emptyObj["subjects"])
	}

	// Populated knobs including retention_tier from wire (not invented from max_age)
	max := int64(1000)
	age := int64(3600)
	pop := NewStreamInfoPrint(StreamInfo{
		Name:          "EVENTS",
		Description:   "ops events",
		Retention:     "limits",
		RetentionTier: "hot",
		Partitions:    2,
		MaxMsgs:       &max,
		MaxAgeSec:     &age,
		Messages:      10,
		FirstSeq:      1,
		LastSeq:       10,
		CreatedAt:     time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
		Subjects:      []string{"dept.events.>"},
	})
	popJS, err := json.Marshal(pop)
	if err != nil {
		t.Fatal(err)
	}
	var popObj map[string]any
	if err := json.Unmarshal(popJS, &popObj); err != nil {
		t.Fatal(err)
	}
	if popObj["description"] != "ops events" || popObj["retention"] != "limits" || popObj["retention_tier"] != "hot" {
		t.Fatalf("populated strings: %s", popJS)
	}
	if popObj["partitions"].(float64) != 2 || popObj["max_msgs"].(float64) != 1000 || popObj["max_age_sec"].(float64) != 3600 {
		t.Fatalf("populated nums: %s", popJS)
	}
	if popObj["created_at"] != "2026-07-01T12:00:00Z" {
		t.Fatalf("created_at: %s", popJS)
	}

	// Wire StreamInfo still omitempty on empty optional knobs (including retention_tier)
	wire, err := json.Marshal(StreamInfo{Name: "SPARSE"})
	if err != nil {
		t.Fatal(err)
	}
	wireS := string(wire)
	if strings.Contains(wireS, "retention") || strings.Contains(wireS, "max_msgs") || strings.Contains(wireS, "description") {
		t.Fatalf("wire StreamInfo should omitempty empty optionals: %s", wireS)
	}

	// Wire decodes retention_tier when broker sends it; print maps as-is
	wirePop, err := json.Marshal(StreamInfo{Name: "T", RetentionTier: "archive"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(wirePop), `"retention_tier":"archive"`) {
		t.Fatalf("wire should include set retention_tier: %s", wirePop)
	}
	var decoded StreamInfo
	if err := json.Unmarshal([]byte(`{"name":"T","retention_tier":"extended","max_age_sec":2592000}`), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.RetentionTier != "extended" {
		t.Fatalf("decode retention_tier: %+v", decoded)
	}
	// max_age alone must not fill tier on print
	ageOnly := NewStreamInfoPrint(StreamInfo{Name: "A", MaxAgeSec: &age})
	if ageOnly.RetentionTier != "" {
		t.Fatalf("must not invent tier from max_age: %+v", ageOnly)
	}
}

// s699/s702: FormatStreams always emits MAX_MSGS / MAX_AGE / RETENTION / TIER.
func TestFormatStreams_AlwaysEmitRetentionColumns(t *testing.T) {
	max := int64(5000)
	age := int64(604800)
	out := FormatStreams([]StreamInfo{
		{Name: "TEMP", Messages: 1, FirstSeq: 1, LastSeq: 1, Partitions: 1, Retention: "limits", RetentionTier: "temp"},
		{Name: "CAPPED", MaxMsgs: &max, MaxAgeSec: &age, Messages: 2, FirstSeq: 1, LastSeq: 2, Retention: "limits"},
	})
	for _, want := range []string{"MAX_MSGS", "MAX_AGE", "RETENTION", "TIER", "TEMP", "CAPPED", "5000", "604800", "temp"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	// Nil knobs on TEMP still print numeric 0 columns (headers present + row values).
	if !strings.Contains(out, "count=2") {
		t.Fatalf("count:\n%s", out)
	}
	empty := FormatStreams(nil)
	if !strings.Contains(empty, "count=0") {
		t.Fatalf("empty list:\n%s", empty)
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

// s726: StreamDeletePrint JSON always-emits {ok,name} (no pull_role invent).
func TestStreamDeletePrint_JSONAlwaysEmitKeys(t *testing.T) {
	t.Parallel()

	// Empty name honest (still always-emit keys).
	emptyDTO := NewStreamDeletePrint("")
	emptyJS, err := json.Marshal(emptyDTO)
	if err != nil {
		t.Fatal(err)
	}
	var emptyObj map[string]any
	if err := json.Unmarshal(emptyJS, &emptyObj); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"ok", "name"} {
		if _, ok := emptyObj[key]; !ok {
			t.Fatalf("empty json missing key %q: %s", key, emptyJS)
		}
	}
	if emptyObj["ok"] != true {
		t.Fatalf("ok want true: %s", emptyJS)
	}
	if emptyObj["name"] != "" {
		t.Fatalf("empty name want \"\"; got %v\n%s", emptyObj["name"], emptyJS)
	}
	// Do not invent pull_role on stream delete.
	if _, ok := emptyObj["pull_role"]; ok {
		t.Fatalf("must not invent pull_role on StreamDeletePrint: %s", emptyJS)
	}

	popDTO := NewStreamDeletePrint("TEMP")
	popJS, err := json.Marshal(popDTO)
	if err != nil {
		t.Fatal(err)
	}
	var popObj map[string]any
	if err := json.Unmarshal(popJS, &popObj); err != nil {
		t.Fatal(err)
	}
	if popObj["ok"] != true || popObj["name"] != "TEMP" {
		t.Fatalf("populated: %s", popJS)
	}

	// Format helpers round-trip keys for scrapers.
	formatted := FormatStreamDeleteJSON(popDTO)
	if !strings.Contains(formatted, `"ok": true`) ||
		!strings.Contains(formatted, `"name": "TEMP"`) {
		t.Fatalf("FormatStreamDeleteJSON: %s", formatted)
	}
	if strings.Contains(formatted, "pull_role") {
		t.Fatalf("FormatStreamDeleteJSON must not invent pull_role: %s", formatted)
	}
}

// s726: FormatStreamDelete always-emits name (empty or populated).
func TestFormatStreamDelete_AlwaysEmitName(t *testing.T) {
	t.Parallel()

	empty := FormatStreamDelete(NewStreamDeletePrint(""))
	for _, want := range []string{
		"PASS mesh streams delete\n",
		"name: \n",
	} {
		if !strings.Contains(empty, want) {
			t.Fatalf("empty missing %q in:\n%q", want, empty)
		}
	}

	pop := FormatStreamDelete(NewStreamDeletePrint("TEMP"))
	if !strings.Contains(pop, "PASS mesh streams delete\n") ||
		!strings.Contains(pop, "name: TEMP\n") {
		t.Fatalf("populated:\n%s", pop)
	}
	if strings.Contains(pop, "pull_role") {
		t.Fatalf("text must not invent pull_role:\n%s", pop)
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

// s723: StreamMessagePrint nested always-emits scraper keys without omitempty gaps;
// wire StreamMessage stays lean (stream/partition/headers omitempty).
func TestStreamMessagePrint_AlwaysEmit(t *testing.T) {
	t.Parallel()

	// Empty / zero fields: stream "", partition 0, headers {}, timestamp "", payload "".
	emptyJS, err := json.Marshal(NewStreamMessagePrint(StreamMessage{Seq: 0, Subject: ""}))
	if err != nil {
		t.Fatal(err)
	}
	var emptyObj map[string]any
	if err := json.Unmarshal(emptyJS, &emptyObj); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"stream", "seq", "subject", "partition", "payload", "headers", "timestamp"} {
		if _, ok := emptyObj[key]; !ok {
			t.Fatalf("empty json missing key %q: %s", key, emptyJS)
		}
	}
	if emptyObj["stream"] != "" {
		t.Fatalf("stream want \"\"; got %v", emptyObj["stream"])
	}
	if emptyObj["partition"] != float64(0) {
		t.Fatalf("partition want 0; got %v", emptyObj["partition"])
	}
	if emptyObj["timestamp"] != "" {
		t.Fatalf("timestamp want \"\"; got %v", emptyObj["timestamp"])
	}
	if emptyObj["payload"] != "" {
		t.Fatalf("payload want \"\"; got %v", emptyObj["payload"])
	}
	headers, ok := emptyObj["headers"].(map[string]any)
	if !ok || len(headers) != 0 {
		t.Fatalf("headers want empty object: %s", emptyJS)
	}
	// Confirm not JSON null for headers/payload.
	if strings.Contains(string(emptyJS), `"headers":null`) || strings.Contains(string(emptyJS), `"payload":null`) {
		t.Fatalf("headers/payload must not be null: %s", emptyJS)
	}

	// Populated nested message.
	ts := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	pop := NewStreamMessagePrint(StreamMessage{
		Stream:    "EVENTS",
		Seq:       7,
		Subject:   "dept.events.hello",
		Partition: 2,
		Payload:   []byte("hi"),
		Headers:   map[string]string{"x-trace": "abc"},
		Timestamp: ts,
	})
	popJS, err := json.Marshal(pop)
	if err != nil {
		t.Fatal(err)
	}
	var popObj map[string]any
	if err := json.Unmarshal(popJS, &popObj); err != nil {
		t.Fatal(err)
	}
	if popObj["stream"] != "EVENTS" || popObj["seq"] != float64(7) || popObj["subject"] != "dept.events.hello" {
		t.Fatalf("identity: %s", popJS)
	}
	if popObj["partition"] != float64(2) {
		t.Fatalf("partition: %s", popJS)
	}
	if popObj["timestamp"] != "2026-07-01T12:00:00Z" {
		t.Fatalf("timestamp: %s", popJS)
	}
	wantB64 := base64.StdEncoding.EncodeToString([]byte("hi"))
	if popObj["payload"] != wantB64 {
		t.Fatalf("payload b64 want %q got %v", wantB64, popObj["payload"])
	}
	popHeaders, ok := popObj["headers"].(map[string]any)
	if !ok || popHeaders["x-trace"] != "abc" {
		t.Fatalf("headers: %s", popJS)
	}

	// Wire StreamMessage omits empty stream/partition/headers (lean omitempty).
	wire, err := json.Marshal(StreamMessage{Seq: 1, Subject: "s"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(wire), `"stream"`) || strings.Contains(string(wire), `"partition"`) ||
		strings.Contains(string(wire), `"headers"`) {
		t.Fatalf("wire StreamMessage should omit empty stream/partition/headers: %s", wire)
	}
	if NewStreamMessagePrint(StreamMessage{}).Timestamp != "" {
		t.Fatal("zero Timestamp must print as empty string")
	}
}

// s720: StreamMessagesPrint envelope always-emits stream + knobs + count + messages
// without omitempty gaps; s723 nested StreamMessagePrint always-emit; wire stays lean.
func TestStreamMessagesPrint_AlwaysEmit(t *testing.T) {
	t.Parallel()

	// Empty/nil messages + zero knobs → empty slice, count 0, knobs 0 honest.
	emptyDTO := NewStreamMessagesPrint("EVENTS", 0, 0, 0, nil)
	emptyJS, err := json.Marshal(emptyDTO)
	if err != nil {
		t.Fatal(err)
	}
	var emptyObj map[string]any
	if err := json.Unmarshal(emptyJS, &emptyObj); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"stream", "from_seq", "to_seq", "limit", "count", "messages"} {
		if _, ok := emptyObj[key]; !ok {
			t.Fatalf("empty json missing key %q: %s", key, emptyJS)
		}
	}
	if emptyObj["stream"] != "EVENTS" {
		t.Fatalf("stream: %s", emptyJS)
	}
	if emptyObj["from_seq"] != float64(0) || emptyObj["to_seq"] != float64(0) || emptyObj["limit"] != float64(0) {
		t.Fatalf("zero knobs honest: %s", emptyJS)
	}
	if emptyObj["count"] != float64(0) {
		t.Fatalf("count want 0: %s", emptyJS)
	}
	msgs, ok := emptyObj["messages"].([]any)
	if !ok || len(msgs) != 0 {
		t.Fatalf("messages want empty array: %s", emptyJS)
	}

	// Populated knobs + one sparse message → nested always-emit keys present.
	popMsgs := []StreamMessage{
		{Seq: 7, Subject: "dept.events.hello", Payload: []byte("hi")},
	}
	popDTO := NewStreamMessagesPrint("EVENTS", 1, 100, 20, popMsgs)
	popJS, err := json.Marshal(popDTO)
	if err != nil {
		t.Fatal(err)
	}
	var popObj map[string]any
	if err := json.Unmarshal(popJS, &popObj); err != nil {
		t.Fatal(err)
	}
	if popObj["stream"] != "EVENTS" {
		t.Fatalf("stream: %s", popJS)
	}
	if popObj["from_seq"] != float64(1) || popObj["to_seq"] != float64(100) || popObj["limit"] != float64(20) {
		t.Fatalf("knobs: %s", popJS)
	}
	if popObj["count"] != float64(1) {
		t.Fatalf("count: %s", popJS)
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
	nestedHeaders, ok := nested["headers"].(map[string]any)
	if !ok || len(nestedHeaders) != 0 {
		t.Fatalf("nested headers want {}: %s", popJS)
	}

	// Wire StreamMessage JSON has no envelope knobs (print DTO is separate).
	wire, err := json.Marshal(StreamMessage{Seq: 1, Subject: "s"})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"from_seq", "to_seq", "limit", "count"} {
		if strings.Contains(string(wire), key) {
			t.Fatalf("wire StreamMessage should not carry %q: %s", key, wire)
		}
	}

	// Text path with knobs (FormatStreamMessagesPrint).
	emptyText := FormatStreamMessagesPrint(emptyDTO)
	for _, want := range []string{
		"iomesh stream messages name=EVENTS count=0 from_seq=0 to_seq=0 limit=0\n",
		"(no messages)\n",
	} {
		if !strings.Contains(emptyText, want) {
			t.Fatalf("empty text missing %q:\n%s", want, emptyText)
		}
	}
	popText := FormatStreamMessagesPrint(popDTO)
	if !strings.Contains(popText, "count=1 from_seq=1 to_seq=100 limit=20") ||
		!strings.Contains(popText, "dept.events.hello") {
		t.Fatalf("populated text:\n%s", popText)
	}
	// Legacy FormatStreamMessages (no knobs) still used by consumer fetch.
	legacy := FormatStreamMessages("EVENTS", nil)
	if strings.Contains(legacy, "from_seq=") || !strings.Contains(legacy, "count=0") {
		t.Fatalf("legacy FormatStreamMessages: %s", legacy)
	}
}
