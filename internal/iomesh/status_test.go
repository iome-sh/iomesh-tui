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

func TestFormatMeshStatus_JSONAlwaysEmitsLatencies(t *testing.T) {
	// Disabled / skipped: health_ms, ready_ms, duration_ms must be present as 0.
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
}
