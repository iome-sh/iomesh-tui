package iomesh

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/router"
)

func TestUsageMeter_RecordAndFormat(t *testing.T) {
	c := New(Config{}, nil)
	c.RecordLLMCall(router.CallMeta{
		ModelName: "deepseek-v4-flash", Duration: 10 * time.Millisecond, EstimatedUSD: 0.001,
	}, router.Usage{PromptTokens: 5, CompletionTokens: 7, TotalTokens: 12}, nil)
	c.RecordLLMCall(router.CallMeta{
		ModelName: "deepseek-v4-flash", Duration: 5 * time.Millisecond, EstimatedUSD: 0.002,
	}, router.Usage{TotalTokens: 3}, nil)
	c.RecordLLMCall(router.CallMeta{
		ModelName: "grok-4.5", Duration: time.Millisecond, EstimatedUSD: 0.01,
	}, router.Usage{TotalTokens: 100}, assertErr{})

	snap := c.Usage()
	if snap.Calls != 3 || snap.Errors != 1 || snap.Tokens != 115 {
		t.Fatalf("%+v", snap)
	}
	if len(snap.ByModel) != 2 {
		t.Fatalf("by_model=%+v", snap.ByModel)
	}
	out := FormatUsage(snap)
	if !strings.Contains(out, "deepseek-v4-flash") || !strings.Contains(out, "grok-4.5") {
		t.Fatal(out)
	}
	if !strings.Contains(out, "totals:") {
		t.Fatal(out)
	}
	js := FormatUsageJSON(snap)
	if !strings.Contains(js, `"calls": 3`) || !strings.Contains(js, `"by_model"`) {
		t.Fatal(js)
	}
	if !strings.Contains(js, "deepseek-v4-flash") {
		t.Fatal(js)
	}
}

func TestFormatUsageJSON_Empty(t *testing.T) {
	js := FormatUsageJSON(UsageSnapshot{})
	if !strings.Contains(js, `"by_model": []`) && !strings.Contains(js, `"by_model":[]`) {
		// indented form
		if !strings.Contains(js, "by_model") {
			t.Fatal(js)
		}
	}
	if !strings.HasSuffix(js, "\n") {
		t.Fatal("expected trailing newline")
	}
}

// s738: empty UsageSnapshot → UsagePrint always-emits all keys; started/as_of "";
// by_model []; no zero-time "0001-01-01".
func TestUsagePrint_JSONAlwaysEmitEmpty(t *testing.T) {
	t.Parallel()

	js := FormatUsageJSON(UsageSnapshot{})
	var obj map[string]any
	if err := json.Unmarshal([]byte(js), &obj); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, js)
	}
	for _, key := range []string{"started", "as_of", "calls", "errors", "tokens", "est_usd", "by_model"} {
		if _, ok := obj[key]; !ok {
			t.Fatalf("empty json missing key %q: %s", key, js)
		}
	}
	if obj["started"] != "" {
		t.Fatalf("started want \"\"; got %v\n%s", obj["started"], js)
	}
	if obj["as_of"] != "" {
		t.Fatalf("as_of want \"\"; got %v\n%s", obj["as_of"], js)
	}
	if obj["calls"].(float64) != 0 || obj["errors"].(float64) != 0 || obj["tokens"].(float64) != 0 {
		t.Fatalf("counters want 0: %s", js)
	}
	if obj["est_usd"].(float64) != 0 {
		t.Fatalf("est_usd want 0: %s", js)
	}
	by, ok := obj["by_model"].([]any)
	if !ok {
		t.Fatalf("by_model want array not null: %s", js)
	}
	if len(by) != 0 {
		t.Fatalf("by_model want []; got %v\n%s", by, js)
	}
	if strings.Contains(js, `"by_model": null`) {
		t.Fatalf("by_model must not be null: %s", js)
	}
	if strings.Contains(js, "0001-01-01") {
		t.Fatalf("zero time must not marshal as 0001-01-01: %s", js)
	}
	if !strings.HasSuffix(js, "\n") {
		t.Fatal("expected trailing newline")
	}

	// NewUsagePrint direct: nil ByModel → non-nil empty slice.
	p := NewUsagePrint(UsageSnapshot{})
	if p.ByModel == nil {
		t.Fatal("ByModel must be non-nil empty slice")
	}
	if p.Started != "" || p.AsOf != "" {
		t.Fatalf("empty times: %+v", p)
	}
}

// s738: populated snapshot → started/as_of RFC3339; by_model rows always-emit
// all ModelUsagePrint keys.
func TestUsagePrint_JSONAlwaysEmitPopulated(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 7, 27, 10, 0, 0, 0, time.UTC)
	asOf := time.Date(2026, 7, 27, 10, 5, 0, 0, time.UTC)
	snap := UsageSnapshot{
		Started: started,
		AsOf:    asOf,
		Calls:   2,
		Errors:  1,
		Tokens:  42,
		EstUSD:  0.0125,
		ByModel: []ModelUsage{
			{
				Model:            "deepseek-v4-flash",
				Calls:            2,
				Errors:           1,
				PromptTokens:     10,
				CompletionTokens: 32,
				TotalTokens:      42,
				EstUSD:           0.0125,
				DurationMS:       15,
			},
		},
	}
	js := FormatUsageJSON(snap)
	var obj map[string]any
	if err := json.Unmarshal([]byte(js), &obj); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, js)
	}
	if obj["started"] != "2026-07-27T10:00:00Z" {
		t.Fatalf("started RFC3339: %v\n%s", obj["started"], js)
	}
	if obj["as_of"] != "2026-07-27T10:05:00Z" {
		t.Fatalf("as_of RFC3339: %v\n%s", obj["as_of"], js)
	}
	if obj["calls"].(float64) != 2 || obj["errors"].(float64) != 1 || obj["tokens"].(float64) != 42 {
		t.Fatalf("totals: %s", js)
	}
	if strings.Contains(js, "0001-01-01") {
		t.Fatalf("must not contain zero time: %s", js)
	}

	rows, ok := obj["by_model"].([]any)
	if !ok || len(rows) != 1 {
		t.Fatalf("by_model: %s", js)
	}
	row, ok := rows[0].(map[string]any)
	if !ok {
		t.Fatalf("row type: %s", js)
	}
	for _, key := range []string{
		"model", "calls", "errors", "prompt_tokens", "completion_tokens",
		"total_tokens", "est_usd", "duration_ms",
	} {
		if _, ok := row[key]; !ok {
			t.Fatalf("by_model[0] missing key %q: %s", key, js)
		}
	}
	if row["model"] != "deepseek-v4-flash" {
		t.Fatalf("model: %v", row["model"])
	}
	if row["calls"].(float64) != 2 || row["errors"].(float64) != 1 {
		t.Fatalf("row counters: %v", row)
	}
	if row["prompt_tokens"].(float64) != 10 || row["completion_tokens"].(float64) != 32 ||
		row["total_tokens"].(float64) != 42 {
		t.Fatalf("row tokens: %v", row)
	}
	if row["duration_ms"].(float64) != 15 {
		t.Fatalf("duration_ms: %v", row)
	}
	if !strings.Contains(js, `"est_usd"`) {
		t.Fatalf("est_usd always-emit: %s", js)
	}

	// Sparse row: all ModelUsagePrint keys present with empty/0 honest.
	sparse := NewUsagePrint(UsageSnapshot{
		ByModel: []ModelUsage{{Model: ""}},
	})
	sparseJS, err := json.Marshal(sparse)
	if err != nil {
		t.Fatal(err)
	}
	var sparseObj map[string]any
	if err := json.Unmarshal(sparseJS, &sparseObj); err != nil {
		t.Fatal(err)
	}
	srows := sparseObj["by_model"].([]any)
	srow := srows[0].(map[string]any)
	for _, key := range []string{
		"model", "calls", "errors", "prompt_tokens", "completion_tokens",
		"total_tokens", "est_usd", "duration_ms",
	} {
		if _, ok := srow[key]; !ok {
			t.Fatalf("sparse by_model[0] missing key %q: %s", key, sparseJS)
		}
	}
	if srow["model"] != "" {
		t.Fatalf("sparse model want \"\"; got %v", srow["model"])
	}
}

// s756: mutate/print JSON completeness pin — locks UsagePrint (s738) always-emit
// keys via FormatUsageJSON / NewUsagePrint (DTO surface already always-emit; this
// serial docs+tests pin completeness, does not invent new fields or re-claim
// s738 product body). Peer aion s755 residual. Zero-time "" never 0001-01-01 ·
// by_model [] not null · local process meter ≠ remote dashboard · DTO ≠ invent
// usage success · dual_write OFF · offline unit ≠ live APPLY · not full mesh RBAC GA.
func TestUsagePrint_JSONCompletenessPin(t *testing.T) {
	t.Parallel()

	usageKeys := []string{"started", "as_of", "calls", "errors", "tokens", "est_usd", "by_model"}

	// Empty surface: all keys present; started/as_of ""; by_model [] not null.
	emptyJS := FormatUsageJSON(UsageSnapshot{})
	if !strings.HasSuffix(emptyJS, "\n") {
		t.Fatal("FormatUsageJSON: expected trailing newline")
	}
	var emptyObj map[string]any
	if err := json.Unmarshal([]byte(emptyJS), &emptyObj); err != nil {
		t.Fatalf("empty unmarshal: %v\n%s", err, emptyJS)
	}
	for _, key := range usageKeys {
		if _, ok := emptyObj[key]; !ok {
			t.Fatalf("empty missing UsagePrint key %q: %s", key, emptyJS)
		}
	}
	if emptyObj["started"] != "" || emptyObj["as_of"] != "" {
		t.Fatalf("zero time must be \"\" not zero-value date: %s", emptyJS)
	}
	if strings.Contains(emptyJS, "0001-01-01") {
		t.Fatalf("must not emit 0001-01-01 zero time: %s", emptyJS)
	}
	byModel, ok := emptyObj["by_model"].([]any)
	if !ok {
		t.Fatalf("by_model want array not null: %s", emptyJS)
	}
	if len(byModel) != 0 {
		t.Fatalf("by_model want []; got %v\n%s", byModel, emptyJS)
	}
	if strings.Contains(emptyJS, `"by_model": null`) || strings.Contains(emptyJS, `"by_model":null`) {
		t.Fatalf("by_model must not be null: %s", emptyJS)
	}

	// Populated path still always-emits the same key set (no invent fields).
	pop := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	popJS := FormatUsageJSON(UsageSnapshot{
		Started: pop,
		AsOf:    pop,
		Calls:   2,
		Errors:  1,
		Tokens:  42,
		EstUSD:  0.01,
		ByModel: []ModelUsage{{Model: "grok-4.5", Calls: 2, TotalTokens: 42}},
	})
	var popObj map[string]any
	if err := json.Unmarshal([]byte(popJS), &popObj); err != nil {
		t.Fatalf("populated unmarshal: %v\n%s", err, popJS)
	}
	for _, key := range usageKeys {
		if _, ok := popObj[key]; !ok {
			t.Fatalf("populated missing UsagePrint key %q: %s", key, popJS)
		}
	}
	if strings.Contains(popJS, "0001-01-01") {
		t.Fatalf("populated must not emit 0001-01-01: %s", popJS)
	}
	rows, ok := popObj["by_model"].([]any)
	if !ok || len(rows) != 1 {
		t.Fatalf("populated by_model: %s", popJS)
	}
}

type assertErr struct{}

func (assertErr) Error() string { return "boom" }
