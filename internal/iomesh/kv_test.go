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
	// created_at always-emitted when set (RFC3339 UTC).
	if !strings.Contains(out, "created_at: 2026-07-01T12:00:00Z") {
		t.Fatalf("want created_at RFC3339 present, got:\n%s", out)
	}
}

func TestFormatKVEntry_CreatedAtAlwaysEmit(t *testing.T) {
	// Zero CreatedAt always emits blank created_at: line (not omitted).
	out := FormatKVEntry(KVEntry{
		Bucket:   "cfg",
		Key:      "k",
		Revision: 1,
		Value:    []byte("v"),
	})
	// "created_at: %s\n" with empty created → "created_at: \n" (space after colon).
	if !strings.Contains(out, "created_at: \n") {
		t.Fatalf("want blank created_at line always emitted, got:\n%q", out)
	}
	// Set CreatedAt → RFC3339 value.
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	out2 := FormatKVEntry(KVEntry{
		Bucket:    "cfg",
		Key:       "k",
		Revision:  2,
		CreatedAt: ts,
		Value:     []byte("v"),
	})
	if !strings.Contains(out2, "created_at: 2026-01-02T03:04:05Z") {
		t.Fatalf("want RFC3339 created_at, got:\n%s", out2)
	}
}

// s714: KVBucketInfoPrint JSON always-emits name/history/max_bytes/ttl_seconds (0 when nil).
func TestKVBucketInfoPrint_AlwaysEmit(t *testing.T) {
	// Empty / nil knobs
	emptyJS, err := json.Marshal(NewKVBucketInfoPrint(KVBucketInfo{Name: "cfg"}))
	if err != nil {
		t.Fatal(err)
	}
	var emptyObj map[string]any
	if err := json.Unmarshal(emptyJS, &emptyObj); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"name", "history", "max_bytes", "ttl_seconds"} {
		if _, ok := emptyObj[key]; !ok {
			t.Fatalf("missing always-emit key %q in %s", key, emptyJS)
		}
	}
	if emptyObj["name"] != "cfg" {
		t.Fatalf("name: %s", emptyJS)
	}
	if emptyObj["history"].(float64) != 0 || emptyObj["max_bytes"].(float64) != 0 || emptyObj["ttl_seconds"].(float64) != 0 {
		t.Fatalf("numeric knobs want 0; got %s", emptyJS)
	}

	// Populated knobs
	var maxBytes int64 = 1024
	var ttl int64 = 3600
	pop := NewKVBucketInfoPrint(KVBucketInfo{
		Name:       "config",
		History:    5,
		MaxBytes:   &maxBytes,
		TTLSeconds: &ttl,
	})
	popJS, err := json.Marshal(pop)
	if err != nil {
		t.Fatal(err)
	}
	var popObj map[string]any
	if err := json.Unmarshal(popJS, &popObj); err != nil {
		t.Fatal(err)
	}
	if popObj["name"] != "config" || popObj["history"].(float64) != 5 ||
		popObj["max_bytes"].(float64) != 1024 || popObj["ttl_seconds"].(float64) != 3600 {
		t.Fatalf("populated: %s", popJS)
	}

	// Wire KVBucketInfo still omitempty on empty optional knobs
	wire, err := json.Marshal(KVBucketInfo{Name: "cfg"})
	if err != nil {
		t.Fatal(err)
	}
	wireS := string(wire)
	if strings.Contains(wireS, "history") || strings.Contains(wireS, "max_bytes") || strings.Contains(wireS, "ttl_seconds") {
		t.Fatalf("wire KVBucketInfo should omitempty empty optionals: %s", wireS)
	}
}

// s714: KVEntryPrint JSON always-emits bucket/key/value/revision/created_at.
func TestKVEntryPrint_AlwaysEmit(t *testing.T) {
	// Empty / zero created_at + nil value
	emptyJS, err := json.Marshal(NewKVEntryPrint(KVEntry{
		Bucket: "cfg", Key: "k", Revision: 0,
	}))
	if err != nil {
		t.Fatal(err)
	}
	var emptyObj map[string]any
	if err := json.Unmarshal(emptyJS, &emptyObj); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"bucket", "key", "value", "revision", "created_at"} {
		if _, ok := emptyObj[key]; !ok {
			t.Fatalf("missing always-emit key %q in %s", key, emptyJS)
		}
	}
	if emptyObj["bucket"] != "cfg" || emptyObj["key"] != "k" {
		t.Fatalf("identity: %s", emptyJS)
	}
	if emptyObj["created_at"] != "" {
		t.Fatalf("created_at want \"\"; got %v", emptyObj["created_at"])
	}
	if emptyObj["revision"].(float64) != 0 {
		t.Fatalf("revision want 0; got %v", emptyObj["revision"])
	}
	// nil value → empty []byte → JSON ""
	if emptyObj["value"] != "" {
		t.Fatalf("value want \"\"; got %v", emptyObj["value"])
	}

	// Populated
	payload := []byte(`{"hello":"world"}`)
	pop := NewKVEntryPrint(KVEntry{
		Bucket:    "config",
		Key:       "app.json",
		Value:     payload,
		Revision:  7,
		CreatedAt: time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
	})
	popJS, err := json.Marshal(pop)
	if err != nil {
		t.Fatal(err)
	}
	var popObj map[string]any
	if err := json.Unmarshal(popJS, &popObj); err != nil {
		t.Fatal(err)
	}
	if popObj["bucket"] != "config" || popObj["key"] != "app.json" || popObj["revision"].(float64) != 7 {
		t.Fatalf("populated identity: %s", popJS)
	}
	if popObj["created_at"] != "2026-07-01T12:00:00Z" {
		t.Fatalf("created_at: %s", popJS)
	}
	wantB64 := base64.StdEncoding.EncodeToString(payload)
	if popObj["value"] != wantB64 {
		t.Fatalf("value b64 want %q got %v", wantB64, popObj["value"])
	}

	// Wire KVEntry: zero time.Time marshals as zero RFC3339 but value nil → null;
	// print DTO is the scraper surface (always-emit created_at as "" when zero).
	// Confirm print empty path is stable and wire type remains distinct (lean decode).
	if NewKVEntryPrint(KVEntry{}).CreatedAt != "" {
		t.Fatal("zero CreatedAt must print as empty string")
	}
}

// s714: KVKeysPrint list envelope always-emits bucket/prefix/count/keys.
func TestKVKeysPrint_AlwaysEmit(t *testing.T) {
	// Empty keys + empty prefix
	emptyJS, err := json.Marshal(NewKVKeysPrint("cfg", "", nil))
	if err != nil {
		t.Fatal(err)
	}
	var emptyObj map[string]any
	if err := json.Unmarshal(emptyJS, &emptyObj); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"bucket", "prefix", "count", "keys"} {
		if _, ok := emptyObj[key]; !ok {
			t.Fatalf("missing always-emit key %q in %s", key, emptyJS)
		}
	}
	if emptyObj["bucket"] != "cfg" || emptyObj["prefix"] != "" {
		t.Fatalf("identity: %s", emptyJS)
	}
	if emptyObj["count"].(float64) != 0 {
		t.Fatalf("count want 0; got %v", emptyObj["count"])
	}
	keys, ok := emptyObj["keys"].([]any)
	if !ok || len(keys) != 0 {
		t.Fatalf("keys want []; got %v", emptyObj["keys"])
	}

	// Populated
	popJS, err := json.Marshal(NewKVKeysPrint("config", "app", []string{"app.json", "app.toml"}))
	if err != nil {
		t.Fatal(err)
	}
	var popObj map[string]any
	if err := json.Unmarshal(popJS, &popObj); err != nil {
		t.Fatal(err)
	}
	if popObj["bucket"] != "config" || popObj["prefix"] != "app" || popObj["count"].(float64) != 2 {
		t.Fatalf("populated: %s", popJS)
	}
	popKeys, ok := popObj["keys"].([]any)
	if !ok || len(popKeys) != 2 || popKeys[0] != "app.json" {
		t.Fatalf("keys: %s", popJS)
	}
}

// s741: FormatKVBucketInfoJSON / FormatKVEntryJSON / FormatKVKeysJSON keys + trailing newline
// (DTOs already always-emit s714; helper completeness only — no new fields).
func TestFormatKVPrintJSON_KeysAndNewline(t *testing.T) {
	t.Parallel()

	bucketJS := FormatKVBucketInfoJSON(NewKVBucketInfoPrint(KVBucketInfo{Name: "cfg"}))
	if !strings.HasSuffix(bucketJS, "\n") {
		t.Fatal("bucket: expected trailing newline")
	}
	var bucketObj map[string]any
	if err := json.Unmarshal([]byte(bucketJS), &bucketObj); err != nil {
		t.Fatalf("bucket unmarshal: %v\n%s", err, bucketJS)
	}
	for _, key := range []string{"name", "history", "max_bytes", "ttl_seconds"} {
		if _, ok := bucketObj[key]; !ok {
			t.Fatalf("bucket missing key %q: %s", key, bucketJS)
		}
	}

	entryJS := FormatKVEntryJSON(NewKVEntryPrint(KVEntry{Bucket: "cfg", Key: "k"}))
	if !strings.HasSuffix(entryJS, "\n") {
		t.Fatal("entry: expected trailing newline")
	}
	var entryObj map[string]any
	if err := json.Unmarshal([]byte(entryJS), &entryObj); err != nil {
		t.Fatalf("entry unmarshal: %v\n%s", err, entryJS)
	}
	for _, key := range []string{"bucket", "key", "value", "revision", "created_at"} {
		if _, ok := entryObj[key]; !ok {
			t.Fatalf("entry missing key %q: %s", key, entryJS)
		}
	}
	if entryObj["created_at"] != "" {
		t.Fatalf("created_at want \"\"; got %v", entryObj["created_at"])
	}

	keysJS := FormatKVKeysJSON(NewKVKeysPrint("cfg", "", nil))
	if !strings.HasSuffix(keysJS, "\n") {
		t.Fatal("keys: expected trailing newline")
	}
	var keysObj map[string]any
	if err := json.Unmarshal([]byte(keysJS), &keysObj); err != nil {
		t.Fatalf("keys unmarshal: %v\n%s", err, keysJS)
	}
	for _, key := range []string{"bucket", "prefix", "count", "keys"} {
		if _, ok := keysObj[key]; !ok {
			t.Fatalf("keys missing key %q: %s", key, keysJS)
		}
	}
	keys, ok := keysObj["keys"].([]any)
	if !ok || len(keys) != 0 {
		t.Fatalf("keys want [] not null: %s", keysJS)
	}
	if strings.Contains(keysJS, `"keys": null`) {
		t.Fatalf("keys must not be null: %s", keysJS)
	}
}

// s729: KVPutPrint JSON always-emits {ok,bucket,key,revision} (no pull_role / value invent).
func TestKVPutPrint_JSONAlwaysEmitKeys(t *testing.T) {
	t.Parallel()

	// Empty bucket/key + revision 0 honest (still always-emit keys).
	emptyDTO := NewKVPutPrint("", "", 0)
	emptyJS, err := json.Marshal(emptyDTO)
	if err != nil {
		t.Fatal(err)
	}
	var emptyObj map[string]any
	if err := json.Unmarshal(emptyJS, &emptyObj); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"ok", "bucket", "key", "revision"} {
		if _, ok := emptyObj[key]; !ok {
			t.Fatalf("empty json missing key %q: %s", key, emptyJS)
		}
	}
	if emptyObj["ok"] != true {
		t.Fatalf("ok want true: %s", emptyJS)
	}
	if emptyObj["bucket"] != "" || emptyObj["key"] != "" {
		t.Fatalf("empty identity want \"\"; got bucket=%v key=%v\n%s", emptyObj["bucket"], emptyObj["key"], emptyJS)
	}
	if emptyObj["revision"].(float64) != 0 {
		t.Fatalf("revision want 0; got %v\n%s", emptyObj["revision"], emptyJS)
	}
	// Do not invent pull_role or value echo on put JSON.
	if _, ok := emptyObj["pull_role"]; ok {
		t.Fatalf("must not invent pull_role on KVPutPrint: %s", emptyJS)
	}
	if _, ok := emptyObj["value"]; ok {
		t.Fatalf("must not invent value echo on KVPutPrint: %s", emptyJS)
	}

	popDTO := NewKVPutPrint("config", "app.json", 7)
	popJS, err := json.Marshal(popDTO)
	if err != nil {
		t.Fatal(err)
	}
	var popObj map[string]any
	if err := json.Unmarshal(popJS, &popObj); err != nil {
		t.Fatal(err)
	}
	if popObj["ok"] != true || popObj["bucket"] != "config" ||
		popObj["key"] != "app.json" || popObj["revision"].(float64) != 7 {
		t.Fatalf("populated: %s", popJS)
	}

	// Format helpers round-trip keys for scrapers.
	formatted := FormatKVPutJSON(popDTO)
	if !strings.Contains(formatted, `"ok": true`) ||
		!strings.Contains(formatted, `"bucket": "config"`) ||
		!strings.Contains(formatted, `"key": "app.json"`) ||
		!strings.Contains(formatted, `"revision": 7`) {
		t.Fatalf("FormatKVPutJSON: %s", formatted)
	}
	if strings.Contains(formatted, "pull_role") || strings.Contains(formatted, `"value"`) {
		t.Fatalf("FormatKVPutJSON must not invent pull_role/value: %s", formatted)
	}
}

// s729: FormatKVPut always-emits bucket/key/revision (empty or populated).
func TestFormatKVPut_AlwaysEmit(t *testing.T) {
	t.Parallel()

	empty := FormatKVPut(NewKVPutPrint("", "", 0))
	for _, want := range []string{
		"PASS mesh kv put\n",
		"bucket:   \n",
		"key:      \n",
		"revision: 0\n",
	} {
		if !strings.Contains(empty, want) {
			t.Fatalf("empty missing %q in:\n%q", want, empty)
		}
	}

	pop := FormatKVPut(NewKVPutPrint("config", "app.json", 7))
	if !strings.Contains(pop, "PASS mesh kv put\n") ||
		!strings.Contains(pop, "bucket:   config\n") ||
		!strings.Contains(pop, "key:      app.json\n") ||
		!strings.Contains(pop, "revision: 7\n") {
		t.Fatalf("populated:\n%s", pop)
	}
	if strings.Contains(pop, "pull_role") {
		t.Fatalf("text must not invent pull_role:\n%s", pop)
	}
}

// s729: KVDeletePrint JSON always-emits {ok,bucket,key} (no pull_role invent).
func TestKVDeletePrint_JSONAlwaysEmitKeys(t *testing.T) {
	t.Parallel()

	// Empty bucket/key honest (still always-emit keys).
	emptyDTO := NewKVDeletePrint("", "")
	emptyJS, err := json.Marshal(emptyDTO)
	if err != nil {
		t.Fatal(err)
	}
	var emptyObj map[string]any
	if err := json.Unmarshal(emptyJS, &emptyObj); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"ok", "bucket", "key"} {
		if _, ok := emptyObj[key]; !ok {
			t.Fatalf("empty json missing key %q: %s", key, emptyJS)
		}
	}
	if emptyObj["ok"] != true {
		t.Fatalf("ok want true: %s", emptyJS)
	}
	if emptyObj["bucket"] != "" || emptyObj["key"] != "" {
		t.Fatalf("empty identity want \"\"; got bucket=%v key=%v\n%s", emptyObj["bucket"], emptyObj["key"], emptyJS)
	}
	// Do not invent pull_role on kv delete.
	if _, ok := emptyObj["pull_role"]; ok {
		t.Fatalf("must not invent pull_role on KVDeletePrint: %s", emptyJS)
	}

	popDTO := NewKVDeletePrint("config", "tmp.key")
	popJS, err := json.Marshal(popDTO)
	if err != nil {
		t.Fatal(err)
	}
	var popObj map[string]any
	if err := json.Unmarshal(popJS, &popObj); err != nil {
		t.Fatal(err)
	}
	if popObj["ok"] != true || popObj["bucket"] != "config" || popObj["key"] != "tmp.key" {
		t.Fatalf("populated: %s", popJS)
	}

	// Format helpers round-trip keys for scrapers.
	formatted := FormatKVDeleteJSON(popDTO)
	if !strings.Contains(formatted, `"ok": true`) ||
		!strings.Contains(formatted, `"bucket": "config"`) ||
		!strings.Contains(formatted, `"key": "tmp.key"`) {
		t.Fatalf("FormatKVDeleteJSON: %s", formatted)
	}
	if strings.Contains(formatted, "pull_role") {
		t.Fatalf("FormatKVDeleteJSON must not invent pull_role: %s", formatted)
	}
}

// s729: FormatKVDelete always-emits bucket/key (empty or populated).
func TestFormatKVDelete_AlwaysEmit(t *testing.T) {
	t.Parallel()

	empty := FormatKVDelete(NewKVDeletePrint("", ""))
	for _, want := range []string{
		"PASS mesh kv delete\n",
		"bucket: \n",
		"key:    \n",
	} {
		if !strings.Contains(empty, want) {
			t.Fatalf("empty missing %q in:\n%q", want, empty)
		}
	}

	pop := FormatKVDelete(NewKVDeletePrint("config", "tmp.key"))
	if !strings.Contains(pop, "PASS mesh kv delete\n") ||
		!strings.Contains(pop, "bucket: config\n") ||
		!strings.Contains(pop, "key:    tmp.key\n") {
		t.Fatalf("populated:\n%s", pop)
	}
	if strings.Contains(pop, "pull_role") {
		t.Fatalf("text must not invent pull_role:\n%s", pop)
	}
}

func TestFormatKVBucketInfo_AlwaysEmitOptionalKnobs(t *testing.T) {
	// Nil optional *int64 knobs: always emit blank max_bytes / ttl_seconds (do not invent 0).
	out := FormatKVBucketInfo(KVBucketInfo{Name: "cfg", History: 0})
	if !strings.Contains(out, "name:       cfg\n") {
		t.Fatalf("name missing:\n%s", out)
	}
	if !strings.Contains(out, "history:    0\n") {
		t.Fatalf("history always-emit missing:\n%s", out)
	}
	if !strings.Contains(out, "max_bytes:  \n") {
		t.Fatalf("want blank max_bytes line when nil, got:\n%q", out)
	}
	if !strings.Contains(out, "ttl_seconds: \n") {
		t.Fatalf("want blank ttl_seconds line when nil, got:\n%q", out)
	}
	// Must not invent zero values for nil pointers.
	if strings.Contains(out, "max_bytes:  0\n") {
		t.Fatalf("invented 0 for nil max_bytes:\n%s", out)
	}
	if strings.Contains(out, "ttl_seconds: 0\n") {
		t.Fatalf("invented 0 for nil ttl_seconds:\n%s", out)
	}

	// When set, show values.
	var maxBytes int64 = 1024
	var ttl int64 = 3600
	out2 := FormatKVBucketInfo(KVBucketInfo{
		Name:       "cfg",
		History:    5,
		MaxBytes:   &maxBytes,
		TTLSeconds: &ttl,
	})
	if !strings.Contains(out2, "history:    5\n") {
		t.Fatalf("history value missing:\n%s", out2)
	}
	if !strings.Contains(out2, "max_bytes:  1024\n") {
		t.Fatalf("max_bytes value missing:\n%s", out2)
	}
	if !strings.Contains(out2, "ttl_seconds: 3600\n") {
		t.Fatalf("ttl_seconds value missing:\n%s", out2)
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
	if _, err := c.KVPut(context.Background(), "b", "k", []byte("v")); err == nil || !strings.Contains(err.Error(), "mesh disabled") {
		t.Fatalf("KVPut err=%v", err)
	}
	if err := c.KVDelete(context.Background(), "b", "k"); err == nil || !strings.Contains(err.Error(), "mesh disabled") {
		t.Fatalf("KVDelete err=%v", err)
	}
	if _, err := c.KVCreateBucket(context.Background(), "b"); err == nil || !strings.Contains(err.Error(), "mesh disabled") {
		t.Fatalf("KVCreateBucket err=%v", err)
	}
}

func TestKVCreateBucket_201Body(t *testing.T) {
	prev := UserAgent()
	SetUserAgent("iomesh-tui/test-kv-create")
	t.Cleanup(func() { SetUserAgent(prev) })

	var maxBytes int64 = 1024
	var ttl int64 = 3600
	var gotMethod, gotPath, gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotUA = r.Header.Get("User-Agent")
		if r.Method != http.MethodPost || r.URL.Path != "/v1/kv/config" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":        "config",
			"max_bytes":   maxBytes,
			"history":     5,
			"ttl_seconds": ttl,
		})
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "acme"}, nil)
	info, err := c.KVCreateBucket(context.Background(), "config")
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method=%q", gotMethod)
	}
	if gotPath != "/v1/kv/config" {
		t.Fatalf("path=%q", gotPath)
	}
	if gotUA != "iomesh-tui/test-kv-create" {
		t.Fatalf("User-Agent=%q", gotUA)
	}
	if info == nil || info.Name != "config" {
		t.Fatalf("info=%+v", info)
	}
	if info.History != 5 {
		t.Fatalf("history=%d", info.History)
	}
	if info.MaxBytes == nil || *info.MaxBytes != maxBytes {
		t.Fatalf("max_bytes=%v", info.MaxBytes)
	}
	if info.TTLSeconds == nil || *info.TTLSeconds != ttl {
		t.Fatalf("ttl_seconds=%v", info.TTLSeconds)
	}
	out := FormatKVBucketInfo(*info)
	if !strings.Contains(out, "config") || !strings.Contains(out, "history") || !strings.Contains(out, "1024") {
		t.Fatal(out)
	}
}

func TestKVCreateBucket_409Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", 405)
			return
		}
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"bucket exists"}`))
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	info, err := c.KVCreateBucket(context.Background(), "existing")
	if err != nil {
		t.Fatal(err)
	}
	if info == nil || info.Name != "existing" {
		t.Fatalf("info=%+v want Name=existing", info)
	}
}

func TestKVCreateBucket_EmptyName(t *testing.T) {
	c := New(Config{Enabled: true, Endpoint: "http://127.0.0.1:9"}, nil)
	if _, err := c.KVCreateBucket(context.Background(), ""); err == nil || !strings.Contains(err.Error(), "bucket required") {
		t.Fatalf("empty err=%v", err)
	}
	if _, err := c.KVCreateBucket(context.Background(), "  "); err == nil || !strings.Contains(err.Error(), "bucket required") {
		t.Fatalf("blank err=%v", err)
	}
}

func TestKVPut_OKAndUserAgent(t *testing.T) {
	prev := UserAgent()
	SetUserAgent("iomesh-tui/test-kv-put")
	t.Cleanup(func() { SetUserAgent(prev) })

	payload := []byte(`{"hello":"put"}`)
	var gotMethod, gotPath, gotUA, gotCT string
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotUA = r.Header.Get("User-Agent")
		gotCT = r.Header.Get("Content-Type")
		if r.Method != http.MethodPut || r.URL.Path != "/v1/kv/config/app.json" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"bucket": "config", "key": "app.json", "revision": 42,
		})
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL, Tenant: "acme"}, nil)
	rev, err := c.KVPut(context.Background(), "config", "app.json", payload)
	if err != nil {
		t.Fatal(err)
	}
	if rev != 42 {
		t.Fatalf("revision=%d want 42", rev)
	}
	if gotMethod != http.MethodPut {
		t.Fatalf("method=%q", gotMethod)
	}
	if gotPath != "/v1/kv/config/app.json" {
		t.Fatalf("path=%q", gotPath)
	}
	if gotUA != "iomesh-tui/test-kv-put" {
		t.Fatalf("User-Agent=%q", gotUA)
	}
	if !strings.Contains(gotCT, "application/json") {
		t.Fatalf("Content-Type=%q", gotCT)
	}
	wantB64 := base64.StdEncoding.EncodeToString(payload)
	if gotBody["value"] != wantB64 {
		t.Fatalf("body value=%q want %q", gotBody["value"], wantB64)
	}
}

func TestKVPut_EmptyArgs(t *testing.T) {
	c := New(Config{Enabled: true, Endpoint: "http://127.0.0.1:9"}, nil)
	if _, err := c.KVPut(context.Background(), "", "k", []byte("v")); err == nil || !strings.Contains(err.Error(), "bucket required") {
		t.Fatalf("bucket err=%v", err)
	}
	if _, err := c.KVPut(context.Background(), "b", "  ", []byte("v")); err == nil || !strings.Contains(err.Error(), "key required") {
		t.Fatalf("key err=%v", err)
	}
}

func TestKVPut_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	rev, err := c.KVPut(context.Background(), "b", "k", []byte("v"))
	if err == nil {
		t.Fatal("expected error")
	}
	if rev != 0 {
		t.Fatalf("rev=%d want 0 on error", rev)
	}
	if !strings.Contains(err.Error(), "http 403") {
		t.Fatalf("err=%v", err)
	}
}

func TestKVPut_PathEscape(t *testing.T) {
	var gotEscaped string
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotEscaped = r.URL.EscapedPath()
		if gotEscaped == "" {
			gotEscaped = r.URL.Path
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{"revision": 3})
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	rev, err := c.KVPut(context.Background(), "my bucket", "a/b", []byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	if rev != 3 {
		t.Fatalf("revision=%d", rev)
	}
	if gotEscaped != "/v1/kv/my%20bucket/a%2Fb" {
		t.Fatalf("escaped path=%q want /v1/kv/my%%20bucket/a%%2Fb", gotEscaped)
	}
}

func TestKVDelete_204(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if r.Method != http.MethodDelete || r.URL.Path != "/v1/kv/config/tmp" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	if err := c.KVDelete(context.Background(), "config", "tmp"); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodDelete {
		t.Fatalf("method=%q", gotMethod)
	}
	if gotPath != "/v1/kv/config/tmp" {
		t.Fatalf("path=%q", gotPath)
	}
}

func TestKVDelete_EmptyArgs(t *testing.T) {
	c := New(Config{Enabled: true, Endpoint: "http://127.0.0.1:9"}, nil)
	if err := c.KVDelete(context.Background(), "", "k"); err == nil || !strings.Contains(err.Error(), "bucket required") {
		t.Fatalf("bucket err=%v", err)
	}
	if err := c.KVDelete(context.Background(), "b", "  "); err == nil || !strings.Contains(err.Error(), "key required") {
		t.Fatalf("key err=%v", err)
	}
}

func TestKVDelete_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	err := c.KVDelete(context.Background(), "b", "missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "http 404") {
		t.Fatalf("err=%v", err)
	}
}

func TestKVDelete_PathEscape(t *testing.T) {
	var gotEscaped string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotEscaped = r.URL.EscapedPath()
		if gotEscaped == "" {
			gotEscaped = r.URL.Path
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	if err := c.KVDelete(context.Background(), "my bucket", "a/b"); err != nil {
		t.Fatal(err)
	}
	if gotEscaped != "/v1/kv/my%20bucket/a%2Fb" {
		t.Fatalf("escaped path=%q want /v1/kv/my%%20bucket/a%%2Fb", gotEscaped)
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
