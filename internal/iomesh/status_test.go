package iomesh

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestProbeStatus(t *testing.T) {
	st, msg := ProbeStatus(nil)
	if st != "ok" || msg != "" {
		t.Fatalf("nil err: status=%q msg=%q", st, msg)
	}
	st, msg = ProbeStatus(errors.New("boom"))
	if st != "err" || msg != "boom" {
		t.Fatalf("err: status=%q msg=%q", st, msg)
	}
}

func TestElapsedMS(t *testing.T) {
	if got := ElapsedMS(0); got != 0 {
		t.Fatalf("0: got %d", got)
	}
	if got := ElapsedMS(5 * time.Millisecond); got != 5 {
		t.Fatalf("5ms: got %d", got)
	}
	if got := ElapsedMS(-time.Second); got != 0 {
		t.Fatalf("negative: got %d want 0", got)
	}
}

func TestAggregateProbeResult(t *testing.T) {
	cases := []struct {
		health, ready, want string
	}{
		{"skipped", "skipped", "skipped"},
		{"ok", "ok", "ok"},
		{"err", "err", "err"},
		{"ok", "err", "err"},
		{"err", "ok", "err"},
		{"err", "skipped", "err"},
		{"skipped", "err", "err"},
		{"ok", "skipped", "partial"},
		{"skipped", "ok", "partial"},
	}
	for _, tc := range cases {
		got := AggregateProbeResult(tc.health, tc.ready)
		if got != tc.want {
			t.Fatalf("AggregateProbeResult(%q, %q)=%q want %q", tc.health, tc.ready, got, tc.want)
		}
	}
}

func TestMeshStatusExitCode(t *testing.T) {
	cases := []struct {
		strict bool
		result string
		want   int
	}{
		// fail-open default: always 0
		{false, "ok", 0},
		{false, "err", 0},
		{false, "skipped", 0},
		{false, "partial", 0},
		{false, "", 0},
		// strict: only hard err exits 1
		{true, "ok", 0},
		{true, "err", 1},
		{true, "skipped", 0}, // mesh disabled / probes skipped — not an error
		{true, "partial", 0}, // prefer 0: only hard err
		{true, "", 0},
	}
	for _, tc := range cases {
		got := MeshStatusExitCode(tc.strict, tc.result)
		if got != tc.want {
			t.Fatalf("MeshStatusExitCode(strict=%v, result=%q)=%d want %d", tc.strict, tc.result, got, tc.want)
		}
	}
}

func TestFormatMeshStatus_JSONAlwaysEmitsLatencies(t *testing.T) {
	// Disabled / skipped: health_ms, ready_ms, duration_ms must be present as 0;
	// result always present as skipped; strict always present (false when unset).
	s := MeshStatusSnapshot{
		Enabled:    false,
		Version:    "test",
		PolicyMode: "off",
		UserAgent:  "iomesh-tui/test",
		StatusLine: "mesh: disabled (offline-first)",
		Health:     "skipped",
		Ready:      "skipped",
		HealthMS:   0,
		ReadyMS:    0,
		DurationMS: 0,
		// Result intentionally empty — FormatMeshStatusJSON fills from health/ready.
		// Strict zero-value false — must still appear in JSON.
	}
	js := FormatMeshStatusJSON(s)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if n, ok := parsed["health_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("health_ms: %v want 0\n%s", parsed["health_ms"], js)
	}
	if n, ok := parsed["ready_ms"].(float64); !ok || int(n) != 0 {
		t.Fatalf("ready_ms: %v want 0\n%s", parsed["ready_ms"], js)
	}
	if n, ok := parsed["duration_ms"].(float64); !ok || int(n) < 0 {
		t.Fatalf("duration_ms: %v want always present and >= 0\n%s", parsed["duration_ms"], js)
	}
	if int(parsed["duration_ms"].(float64)) != 0 {
		t.Fatalf("duration_ms: %v want 0 when skipped\n%s", parsed["duration_ms"], js)
	}
	if parsed["health"] != "skipped" || parsed["ready"] != "skipped" {
		t.Fatalf("health/ready: %v / %v\n%s", parsed["health"], parsed["ready"], js)
	}
	if parsed["result"] != "skipped" {
		t.Fatalf("result: %v want skipped\n%s", parsed["result"], js)
	}
	if v, ok := parsed["strict"].(bool); !ok || v {
		t.Fatalf("strict: %v want false always-emitted\n%s", parsed["strict"], js)
	}
	if n, ok := parsed["exit_code"].(float64); !ok || int(n) != 0 {
		t.Fatalf("exit_code: %v want 0 always-emitted (strict=false)\n%s", parsed["exit_code"], js)
	}
}

func TestFormatMeshStatus_JSONProbeLatencies(t *testing.T) {
	s := MeshStatusSnapshot{
		Enabled:    true,
		Endpoint:   "http://127.0.0.1:1",
		Version:    "0.1.0",
		PolicyMode: "off",
		UserAgent:  "iomesh-tui/0.1.0",
		StatusLine: "mesh: enabled · endpoint=http://127.0.0.1:1",
		Health:     "ok",
		HealthMS:   12,
		Ready:      "err",
		ReadyErr:   "iomesh ready: http 503",
		ReadyMS:    34,
		DurationMS: 50,
		Result:     "err",
		Strict:     true,
	}
	js := FormatMeshStatusJSON(s)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if n, ok := parsed["health_ms"].(float64); !ok || int(n) != 12 {
		t.Fatalf("health_ms: %v want 12\n%s", parsed["health_ms"], js)
	}
	if n, ok := parsed["ready_ms"].(float64); !ok || int(n) != 34 {
		t.Fatalf("ready_ms: %v want 34\n%s", parsed["ready_ms"], js)
	}
	if n, ok := parsed["duration_ms"].(float64); !ok || int(n) != 50 {
		t.Fatalf("duration_ms: %v want 50\n%s", parsed["duration_ms"], js)
	}
	if n, ok := parsed["duration_ms"].(float64); !ok || int(n) < 0 {
		t.Fatalf("duration_ms: %v want >= 0\n%s", parsed["duration_ms"], js)
	}
	if parsed["health"] != "ok" {
		t.Fatalf("health: %v", parsed["health"])
	}
	if parsed["ready"] != "err" {
		t.Fatalf("ready: %v", parsed["ready"])
	}
	if parsed["ready_err"] != "iomesh ready: http 503" {
		t.Fatalf("ready_err: %v", parsed["ready_err"])
	}
	// health_err omitted when empty
	if _, ok := parsed["health_err"]; ok {
		t.Fatalf("health_err should be omitted when empty: %v", parsed["health_err"])
	}
	if parsed["result"] != "err" {
		t.Fatalf("result: %v want err\n%s", parsed["result"], js)
	}
	if v, ok := parsed["strict"].(bool); !ok || !v {
		t.Fatalf("strict: %v want true\n%s", parsed["strict"], js)
	}
	if n, ok := parsed["exit_code"].(float64); !ok || int(n) != 1 {
		t.Fatalf("exit_code: %v want 1 (strict=true result=err)\n%s", parsed["exit_code"], js)
	}
}

func TestFormatMeshStatus_JSONAlwaysEmitsStrict(t *testing.T) {
	// strict is always present in JSON: false when zero-value, true when set.
	for _, want := range []bool{false, true} {
		s := MeshStatusSnapshot{
			Enabled:    true,
			Version:    "t",
			PolicyMode: "off",
			UserAgent:  "ua",
			StatusLine: "mesh: enabled",
			Health:     "ok",
			Ready:      "ok",
			Result:     "ok",
			Strict:     want,
		}
		js := FormatMeshStatusJSON(s)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(js), &parsed); err != nil {
			t.Fatalf("json strict=%v: %v\n%s", want, err, js)
		}
		got, ok := parsed["strict"].(bool)
		if !ok {
			t.Fatalf("strict missing or non-bool for want=%v: %v\n%s", want, parsed["strict"], js)
		}
		if got != want {
			t.Fatalf("strict: %v want %v\n%s", got, want, js)
		}
	}
}

func TestFormatMeshStatus_JSONAlwaysEmitsResult(t *testing.T) {
	// Every health/ready pair must yield a non-empty result in JSON.
	cases := []struct {
		health, ready, want string
	}{
		{"skipped", "skipped", "skipped"},
		{"ok", "ok", "ok"},
		{"err", "ok", "err"},
		{"ok", "err", "err"},
		{"ok", "skipped", "partial"},
		{"skipped", "ok", "partial"},
		{"err", "skipped", "err"},
		{"skipped", "err", "err"},
	}
	for _, tc := range cases {
		s := MeshStatusSnapshot{
			Enabled:    true,
			Version:    "t",
			PolicyMode: "off",
			UserAgent:  "ua",
			StatusLine: "mesh: enabled",
			Health:     tc.health,
			Ready:      tc.ready,
		}
		js := FormatMeshStatusJSON(s)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(js), &parsed); err != nil {
			t.Fatalf("json health=%s ready=%s: %v\n%s", tc.health, tc.ready, err, js)
		}
		got, ok := parsed["result"].(string)
		if !ok || got == "" {
			t.Fatalf("result missing or empty for health=%s ready=%s: %v\n%s", tc.health, tc.ready, parsed["result"], js)
		}
		if got != tc.want {
			t.Fatalf("result for health=%s ready=%s: %q want %q\n%s", tc.health, tc.ready, got, tc.want, js)
		}
	}
}

func TestFormatMeshStatus_JSONAlwaysEmitsExitCode(t *testing.T) {
	// exit_code always present: 0 when strict=false; 1 only when strict && result=err.
	cases := []struct {
		name     string
		strict   bool
		result   string
		health   string
		ready    string
		wantExit int
	}{
		{"strict_false_ok", false, "ok", "ok", "ok", 0},
		{"strict_false_err", false, "err", "err", "err", 0},
		{"strict_false_skipped", false, "skipped", "skipped", "skipped", 0},
		{"strict_true_ok", true, "ok", "ok", "ok", 0},
		{"strict_true_err", true, "err", "err", "err", 1},
		{"strict_true_skipped", true, "skipped", "skipped", "skipped", 0},
		{"strict_true_partial", true, "partial", "ok", "skipped", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := MeshStatusSnapshot{
				Enabled:    true,
				Version:    "t",
				PolicyMode: "off",
				UserAgent:  "ua",
				StatusLine: "mesh: enabled",
				Health:     tc.health,
				Ready:      tc.ready,
				Result:     tc.result,
				Strict:     tc.strict,
			}
			js := FormatMeshStatusJSON(s)
			var parsed map[string]any
			if err := json.Unmarshal([]byte(js), &parsed); err != nil {
				t.Fatalf("json: %v\n%s", err, js)
			}
			n, ok := parsed["exit_code"].(float64)
			if !ok {
				t.Fatalf("exit_code missing or non-number: %v\n%s", parsed["exit_code"], js)
			}
			if int(n) != tc.wantExit {
				t.Fatalf("exit_code: %v want %d (strict=%v result=%q)\n%s", parsed["exit_code"], tc.wantExit, tc.strict, tc.result, js)
			}
			if v, ok := parsed["strict"].(bool); !ok || v != tc.strict {
				t.Fatalf("strict: %v want %v\n%s", parsed["strict"], tc.strict, js)
			}
			if got, _ := parsed["result"].(string); got != tc.result {
				t.Fatalf("result: %v want %q\n%s", parsed["result"], tc.result, js)
			}
		})
	}
}

func TestFormatMeshStatus_Text(t *testing.T) {
	s := MeshStatusSnapshot{
		Enabled:        true,
		Endpoint:       "http://mesh.example",
		Tenant:         "t1",
		Org:            "o1",
		Workspace:      "w1",
		Version:        "dev",
		PolicyMode:     "advisory",
		ContextPlane:   true,
		CatalogPlane:   false,
		IncludeLineage: true,
		EmitDept:       true,
		UserAgent:      "iomesh-tui/dev",
		StatusLine:     "mesh: enabled · endpoint=http://mesh.example",
		Health:         "ok",
		HealthMS:       7,
		Ready:          "err",
		ReadyErr:       "timeout",
		ReadyMS:        9,
		DurationMS:     16,
		Result:         "err",
	}
	out := FormatMeshStatus(s)
	for _, want := range []string{
		"iomesh mesh status",
		"status_line: mesh: enabled",
		"version:     dev",
		"endpoint:    http://mesh.example",
		"tenant:      t1",
		"org:         o1",
		"workspace:   w1",
		"policy_mode: advisory",
		"context_plane: true",
		"catalog_plane: false",
		"include_lineage: true",
		"emit_dept:   true",
		"user_agent:  iomesh-tui/dev",
		"health:      ok",
		"health_ms:   7",
		"ready:       err (timeout)",
		"ready_ms:    9",
		"duration_ms: 16",
		"result:      err",
		"strict:      false",
		"exit_code:   0",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("text missing %q:\n%s", want, out)
		}
	}
}

func TestFormatMeshStatus_TextDisabledZeros(t *testing.T) {
	s := MeshStatusSnapshot{
		Enabled:    false,
		Version:    "x",
		PolicyMode: "off",
		UserAgent:  "ua",
		StatusLine: "mesh: disabled (offline-first)",
		Health:     "skipped",
		Ready:      "skipped",
		HealthMS:   0,
		ReadyMS:    0,
		DurationMS: 0,
	}
	out := FormatMeshStatus(s)
	if !strings.Contains(out, "mesh: disabled (offline-first)") {
		t.Fatalf("missing offline-first status_line:\n%s", out)
	}
	if !strings.Contains(out, "health:      skipped") || !strings.Contains(out, "ready:       skipped") {
		t.Fatalf("missing skipped probes:\n%s", out)
	}
	if !strings.Contains(out, "health_ms:   0") || !strings.Contains(out, "ready_ms:    0") {
		t.Fatalf("missing zero latencies:\n%s", out)
	}
	if !strings.Contains(out, "duration_ms: 0") {
		t.Fatalf("missing zero duration_ms:\n%s", out)
	}
	if !strings.Contains(out, "result:      skipped") {
		t.Fatalf("missing result skipped:\n%s", out)
	}
	if !strings.Contains(out, "strict:      false") {
		t.Fatalf("missing strict false:\n%s", out)
	}
	if !strings.Contains(out, "exit_code:   0") {
		t.Fatalf("missing exit_code 0:\n%s", out)
	}
}

func TestFormatMeshStatus_TextAlwaysEmitsStrict(t *testing.T) {
	// strict always appears in text for both false and true.
	falseSnap := MeshStatusSnapshot{
		Enabled: true, Version: "v", PolicyMode: "off", UserAgent: "ua",
		StatusLine: "mesh: enabled", Health: "ok", Ready: "ok", Strict: false,
	}
	out := FormatMeshStatus(falseSnap)
	if !strings.Contains(out, "strict:      false") {
		t.Fatalf("want strict false:\n%s", out)
	}
	if !strings.Contains(out, "exit_code:   0") {
		t.Fatalf("want exit_code 0 with strict false:\n%s", out)
	}
	trueSnap := MeshStatusSnapshot{
		Enabled: true, Version: "v", PolicyMode: "off", UserAgent: "ua",
		StatusLine: "mesh: enabled", Health: "err", Ready: "err", Result: "err", Strict: true,
	}
	out = FormatMeshStatus(trueSnap)
	if !strings.Contains(out, "strict:      true") {
		t.Fatalf("want strict true:\n%s", out)
	}
	if !strings.Contains(out, "result:      err") {
		t.Fatalf("want result err with strict true:\n%s", out)
	}
	if !strings.Contains(out, "exit_code:   1") {
		t.Fatalf("want exit_code 1 with strict true result err:\n%s", out)
	}
}

func TestFormatMeshStatus_TextAlwaysEmitsExitCode(t *testing.T) {
	// exit_code always appears: strict=false → 0; strict=true result=err → 1; strict=true result=ok → 0.
	cases := []struct {
		name     string
		strict   bool
		result   string
		health   string
		ready    string
		wantLine string
	}{
		{"strict_false", false, "err", "err", "err", "exit_code:   0"},
		{"strict_true_err", true, "err", "err", "err", "exit_code:   1"},
		{"strict_true_ok", true, "ok", "ok", "ok", "exit_code:   0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := MeshStatusSnapshot{
				Enabled: true, Version: "v", PolicyMode: "off", UserAgent: "ua",
				StatusLine: "mesh: enabled", Health: tc.health, Ready: tc.ready,
				Result: tc.result, Strict: tc.strict,
			}
			out := FormatMeshStatus(s)
			if !strings.Contains(out, tc.wantLine) {
				t.Fatalf("want %q:\n%s", tc.wantLine, out)
			}
			// exit_code always follows the strict line
			strictIdx := strings.Index(out, "strict:")
			exitIdx := strings.Index(out, "exit_code:")
			if strictIdx < 0 || exitIdx < 0 || exitIdx < strictIdx {
				t.Fatalf("exit_code must appear after strict:\n%s", out)
			}
		})
	}
}

func TestFormatMeshStatus_TextPartialAndOK(t *testing.T) {
	okSnap := MeshStatusSnapshot{
		Enabled: true, Version: "v", PolicyMode: "off", UserAgent: "ua",
		StatusLine: "mesh: enabled", Health: "ok", Ready: "ok",
	}
	out := FormatMeshStatus(okSnap)
	if !strings.Contains(out, "result:      ok") {
		t.Fatalf("want result ok:\n%s", out)
	}
	partial := MeshStatusSnapshot{
		Enabled: true, Version: "v", PolicyMode: "off", UserAgent: "ua",
		StatusLine: "mesh: enabled", Health: "ok", Ready: "skipped",
	}
	out = FormatMeshStatus(partial)
	if !strings.Contains(out, "result:      partial") {
		t.Fatalf("want result partial:\n%s", out)
	}
}
