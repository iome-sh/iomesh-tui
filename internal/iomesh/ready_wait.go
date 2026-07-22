package iomesh

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// WaitReadyOptions controls WaitReady.
type WaitReadyOptions struct {
	// Interval between attempts (default 500ms). Zero → 500ms.
	Interval time.Duration
	// Also require Health OK each attempt before Ready (default false).
	RequireHealth bool
}

// WaitReady polls Ready (and optional Health) until success or ctx done.
// Respects ctx deadline/cancel. Returns last error if deadline exceeded.
// Disabled/nil client returns nil (offline-first).
func (c *Client) WaitReady(ctx context.Context, opts WaitReadyOptions) error {
	_, err := c.WaitReadyAttempts(ctx, opts)
	return err
}

// WaitReadyAttempts is WaitReady plus the number of probe attempt cycles.
// Each loop iteration that tries Health-if-required + Ready counts as one attempt
// (incremented at the start of the cycle, before probes).
// On success attempts is >= 1. On timeout attempts is the count so far (>= 0).
// Nil/disabled client returns (0, nil).
func (c *Client) WaitReadyAttempts(ctx context.Context, opts WaitReadyOptions) (attempts int, err error) {
	if c == nil || !c.Enabled() {
		return 0, nil
	}
	interval := opts.Interval
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}

	var lastErr error
	for {
		attempts++
		if opts.RequireHealth {
			if err := c.Health(ctx); err != nil {
				lastErr = err
				if err := sleepWait(ctx, interval); err != nil {
					return attempts, wrapWaitReadyErr(lastErr, err)
				}
				continue
			}
		}
		if err := c.Ready(ctx); err != nil {
			lastErr = err
			if err := sleepWait(ctx, interval); err != nil {
				return attempts, wrapWaitReadyErr(lastErr, err)
			}
			continue
		}
		return attempts, nil
	}
}

func sleepWait(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func wrapWaitReadyErr(lastErr, ctxErr error) error {
	if lastErr != nil {
		return fmt.Errorf("wait ready: %w", lastErr)
	}
	if ctxErr != nil {
		return fmt.Errorf("wait ready: %w", ctxErr)
	}
	return fmt.Errorf("wait ready: deadline exceeded")
}

// MeshWaitEvidence is the always-emit operator preflight wait result shape.
// TimeoutMS and IntervalMS are the configured WaitReady budget and poll interval.
// Attempts is the number of WaitReady probe attempt cycles.
// Result is always-emitted ok|err derived from OK (peer to mesh status result continuum;
// wait only uses ok|err — does not invent readiness schema beyond OK/waitErr).
// ExitCode is the process exit code for this wait (0 when OK, 1 when not OK).
// Version is the package product/binary version via ProductVersion() (empty when unset).
// UserAgent is the package mesh HTTP User-Agent via UserAgent() (default "iomesh-tui").
// Identity fields (Endpoint, Tenant, Org, Workspace) are always emitted as strings
// (empty when unset) so CI scrapers can key on stable identity without omitempty gaps;
// peer to mesh status identity continuum. Does not invent readiness from identity.
// Error is always emitted (empty string when OK) so CI scrapers can key without omitempty gaps.
type MeshWaitEvidence struct {
	OK            bool
	ElapsedMS     int
	RequireHealth bool
	TimeoutMS     int
	IntervalMS    int
	Attempts      int
	Result        string // always emit: "ok" | "err" derived from OK
	ExitCode      int
	Version       string // always emit; empty when ProductVersion unset
	UserAgent     string // always emit; package mesh HTTP UA (default "iomesh-tui")
	Endpoint      string // always emit; empty when unset
	Tenant        string // always emit; empty when unset
	Org           string // always emit; empty when unset
	Workspace     string // always emit; empty when unset
	Error         string // always emit; empty string when OK (or unset on success)
}

// MeshWaitResult returns the always-emitted mesh wait result token:
// "ok" when OK, "err" when not OK. Derived from OK / waitErr only (no invent readiness).
func MeshWaitResult(e MeshWaitEvidence) string {
	if e.OK {
		return "ok"
	}
	return "err"
}

// MeshWaitExitCode returns the process exit code for mesh wait evidence:
// 0 when OK, 1 when not OK. Matches cmdMeshWait process exit.
func MeshWaitExitCode(e MeshWaitEvidence) int {
	if e.OK {
		return 0
	}
	return 1
}

// normalize clamps negative elapsed/timeout/interval/attempts to 0, defaults empty
// Error on failure (and clears Error when OK so scrapers trust empty error with ok),
// and always re-derives Result + ExitCode from OK so scrapers trust the pair.
func (e MeshWaitEvidence) normalize() MeshWaitEvidence {
	if e.ElapsedMS < 0 {
		e.ElapsedMS = 0
	}
	if e.TimeoutMS < 0 {
		e.TimeoutMS = 0
	}
	if e.IntervalMS < 0 {
		e.IntervalMS = 0
	}
	if e.Attempts < 0 {
		e.Attempts = 0
	}
	if e.OK {
		e.Error = ""
	} else if e.Error == "" {
		e.Error = "unknown error"
	}
	// Always re-derive from OK so scrapers trust result + exit_code with the ok flag.
	e.Result = MeshWaitResult(e)
	if e.OK {
		e.ExitCode = 0
	} else {
		e.ExitCode = 1
	}
	return e
}

// FormatMeshWaitResult renders operator preflight wait outcome as text.
// Always includes elapsed_ms, require_health, timeout_ms, interval_ms, attempts, result,
// exit_code, version, user_agent, identity fields endpoint/tenant/org/workspace, and error
// (empty string when OK / unset) for CI evidence.
func FormatMeshWaitResult(e MeshWaitEvidence) string {
	e = e.normalize()
	if e.OK {
		return fmt.Sprintf(
			"PASS mesh wait: ready\nelapsed_ms: %d\nrequire_health: %t\ntimeout_ms: %d\ninterval_ms: %d\nattempts: %d\nresult: %s\nexit_code: %d\nversion: %s\nuser_agent: %s\nendpoint: %s\ntenant: %s\norg: %s\nworkspace: %s\nerror: %s\n",
			e.ElapsedMS, e.RequireHealth, e.TimeoutMS, e.IntervalMS, e.Attempts, e.Result, e.ExitCode, e.Version, e.UserAgent,
			e.Endpoint, e.Tenant, e.Org, e.Workspace, e.Error,
		)
	}
	return fmt.Sprintf(
		"FAIL mesh wait: %s\nelapsed_ms: %d\nrequire_health: %t\ntimeout_ms: %d\ninterval_ms: %d\nattempts: %d\nresult: %s\nexit_code: %d\nversion: %s\nuser_agent: %s\nendpoint: %s\ntenant: %s\norg: %s\nworkspace: %s\nerror: %s\n",
		e.Error, e.ElapsedMS, e.RequireHealth, e.TimeoutMS, e.IntervalMS, e.Attempts, e.Result, e.ExitCode, e.Version, e.UserAgent,
		e.Endpoint, e.Tenant, e.Org, e.Workspace, e.Error,
	)
}

// FormatMeshWaitResultJSON renders wait outcome as compact JSON for scrapers.
// Always emits ok, elapsed_ms, require_health, timeout_ms, interval_ms, attempts, result,
// exit_code, version, user_agent, endpoint, tenant, org, workspace, and error (empty
// string when OK / unset) so CI scrapers can key on stable fields without omitempty gaps.
func FormatMeshWaitResultJSON(e MeshWaitEvidence) string {
	e = e.normalize()
	type out struct {
		OK            bool   `json:"ok"`
		ElapsedMS     int    `json:"elapsed_ms"`
		RequireHealth bool   `json:"require_health"`
		TimeoutMS     int    `json:"timeout_ms"`
		IntervalMS    int    `json:"interval_ms"`
		Attempts      int    `json:"attempts"`
		Result        string `json:"result"`
		ExitCode      int    `json:"exit_code"`
		Version       string `json:"version"`
		UserAgent     string `json:"user_agent"`
		Endpoint      string `json:"endpoint"`
		Tenant        string `json:"tenant"`
		Org           string `json:"org"`
		Workspace     string `json:"workspace"`
		Error         string `json:"error"`
	}
	o := out{
		OK:            e.OK,
		ElapsedMS:     e.ElapsedMS,
		RequireHealth: e.RequireHealth,
		TimeoutMS:     e.TimeoutMS,
		IntervalMS:    e.IntervalMS,
		Attempts:      e.Attempts,
		Result:        e.Result,
		ExitCode:      e.ExitCode,
		Version:       e.Version,
		UserAgent:     e.UserAgent,
		Endpoint:      e.Endpoint,
		Tenant:        e.Tenant,
		Org:           e.Org,
		Workspace:     e.Workspace,
		Error:         e.Error,
	}
	b, err := json.Marshal(o)
	if err != nil {
		return `{"ok":false,"elapsed_ms":0,"require_health":false,"timeout_ms":0,"interval_ms":0,"attempts":0,"result":"err","exit_code":1,"version":"","user_agent":"","endpoint":"","tenant":"","org":"","workspace":"","error":"mesh wait json marshal failed"}` + "\n"
	}
	return string(b) + "\n"
}
