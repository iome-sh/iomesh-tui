package iomesh

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// MeshStatusSnapshot is the operator one-shot mesh status payload for
// `iomesh mesh status` (JSON and text). Probe fields are fail-open: health/ready
// are "ok", "err", or "skipped"; latencies are always emitted (0 when skipped).
// Result is always emitted (ok|err|skipped|partial) as the aggregate of health+ready.
type MeshStatusSnapshot struct {
	Enabled        bool   `json:"enabled"`
	Endpoint       string `json:"endpoint,omitempty"`
	Tenant         string `json:"tenant,omitempty"`
	Org            string `json:"org,omitempty"`
	Workspace      string `json:"workspace,omitempty"`
	Version        string `json:"version"` // binary version (main.version)
	PolicyMode     string `json:"policy_mode"`
	ContextPlane   bool   `json:"context_plane"`
	CatalogPlane   bool   `json:"catalog_plane"`
	IncludeLineage bool   `json:"include_lineage"`
	EmitDept       bool   `json:"emit_dept"`
	UserAgent      string `json:"user_agent"`
	StatusLine     string `json:"status_line"`
	Health         string `json:"health"` // ok|err|skipped
	HealthErr      string `json:"health_err,omitempty"`
	// HealthMS is Health probe latency in milliseconds (always emitted; 0 when skipped/disabled).
	HealthMS int    `json:"health_ms"`
	Ready    string `json:"ready"` // ok|err|skipped
	ReadyErr string `json:"ready_err,omitempty"`
	// ReadyMS is Ready probe latency in milliseconds (always emitted; 0 when skipped/disabled).
	ReadyMS int `json:"ready_ms"`
	// DurationMS is wall-clock for the Health+Ready probe path in milliseconds
	// (always emitted; >=0; ~0 when mesh disabled / probes skipped).
	DurationMS int `json:"duration_ms"`
	// Result aggregates health+ready: ok|err|skipped|partial (always emitted).
	Result string `json:"result"`
	// Strict is the --strict exit-gate flag (always emitted; false when unset).
	Strict bool `json:"strict"`
}

// AggregateProbeResult returns the aggregate mesh status result from health and
// ready probe statuses (each ok|err|skipped):
//   - both skipped → skipped
//   - both ok → ok
//   - either err → err
//   - one ok and one skipped → partial
func AggregateProbeResult(health, ready string) string {
	if health == "err" || ready == "err" {
		return "err"
	}
	if health == "skipped" && ready == "skipped" {
		return "skipped"
	}
	if health == "ok" && ready == "ok" {
		return "ok"
	}
	// one ok + one skipped (or other non-err mix) → partial
	return "partial"
}

// MeshStatusExitCode returns the process exit code for `iomesh mesh status`.
// Default (strict=false) is fail-open: always 0 after a successful print.
// With --strict, only aggregate result "err" exits 1; ok / skipped / partial stay 0
// (mesh disabled → skipped is not an error).
func MeshStatusExitCode(strict bool, result string) int {
	if !strict {
		return 0
	}
	if result == "err" {
		return 1
	}
	return 0
}

// ProbeStatus returns "ok" or "err" and an optional error message for fail-open display.
func ProbeStatus(err error) (status, errMsg string) {
	if err != nil {
		return "err", err.Error()
	}
	return "ok", ""
}

// ElapsedMS converts a duration to non-negative milliseconds for status/dogfood evidence.
func ElapsedMS(d time.Duration) int {
	ms := int(d.Milliseconds())
	if ms < 0 {
		return 0
	}
	return ms
}

// FormatMeshStatusJSON returns indented JSON for stage CI / operator scrapers.
// Result is filled from health/ready when empty so scrapers always see result.
func FormatMeshStatusJSON(s MeshStatusSnapshot) string {
	if s.Result == "" {
		s.Result = AggregateProbeResult(s.Health, s.Ready)
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return `{"error":"mesh status json marshal failed"}` + "\n"
	}
	return string(b) + "\n"
}

// FormatMeshStatus renders a human-readable operator snapshot.
// Result is filled from health/ready when empty so operators always see result.
func FormatMeshStatus(s MeshStatusSnapshot) string {
	if s.Result == "" {
		s.Result = AggregateProbeResult(s.Health, s.Ready)
	}
	var b strings.Builder
	b.WriteString("iomesh mesh status\n")
	fmt.Fprintf(&b, "  status_line: %s\n", s.StatusLine)
	fmt.Fprintf(&b, "  version:     %s\n", s.Version)
	fmt.Fprintf(&b, "  endpoint:    %s\n", s.Endpoint)
	if s.Tenant != "" {
		fmt.Fprintf(&b, "  tenant:      %s\n", s.Tenant)
	}
	if s.Org != "" {
		fmt.Fprintf(&b, "  org:         %s\n", s.Org)
	}
	if s.Workspace != "" {
		fmt.Fprintf(&b, "  workspace:   %s\n", s.Workspace)
	}
	fmt.Fprintf(&b, "  policy_mode: %s\n", s.PolicyMode)
	fmt.Fprintf(&b, "  context_plane: %v\n", s.ContextPlane)
	fmt.Fprintf(&b, "  catalog_plane: %v\n", s.CatalogPlane)
	fmt.Fprintf(&b, "  include_lineage: %v\n", s.IncludeLineage)
	fmt.Fprintf(&b, "  emit_dept:   %v\n", s.EmitDept)
	fmt.Fprintf(&b, "  user_agent:  %s\n", s.UserAgent)
	if s.HealthErr != "" {
		fmt.Fprintf(&b, "  health:      %s (%s)\n", s.Health, s.HealthErr)
	} else {
		fmt.Fprintf(&b, "  health:      %s\n", s.Health)
	}
	fmt.Fprintf(&b, "  health_ms:   %d\n", s.HealthMS)
	if s.ReadyErr != "" {
		fmt.Fprintf(&b, "  ready:       %s (%s)\n", s.Ready, s.ReadyErr)
	} else {
		fmt.Fprintf(&b, "  ready:       %s\n", s.Ready)
	}
	fmt.Fprintf(&b, "  ready_ms:    %d\n", s.ReadyMS)
	fmt.Fprintf(&b, "  duration_ms: %d\n", s.DurationMS)
	fmt.Fprintf(&b, "  result:      %s\n", s.Result)
	fmt.Fprintf(&b, "  strict:      %t\n", s.Strict)
	return b.String()
}
