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
	if c == nil || !c.Enabled() {
		return nil
	}
	interval := opts.Interval
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}

	var lastErr error
	for {
		if opts.RequireHealth {
			if err := c.Health(ctx); err != nil {
				lastErr = err
				if err := sleepWait(ctx, interval); err != nil {
					return wrapWaitReadyErr(lastErr, err)
				}
				continue
			}
		}
		if err := c.Ready(ctx); err != nil {
			lastErr = err
			if err := sleepWait(ctx, interval); err != nil {
				return wrapWaitReadyErr(lastErr, err)
			}
			continue
		}
		return nil
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
type MeshWaitEvidence struct {
	OK            bool
	ElapsedMS     int
	RequireHealth bool
	TimeoutMS     int
	IntervalMS    int
	Error         string // empty on success
}

// normalize clamps negative elapsed/timeout/interval to 0 and defaults empty Error on failure.
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
	if !e.OK && e.Error == "" {
		e.Error = "unknown error"
	}
	return e
}

// FormatMeshWaitResult renders operator preflight wait outcome as text.
// Always includes elapsed_ms, require_health, timeout_ms, and interval_ms for CI evidence.
func FormatMeshWaitResult(e MeshWaitEvidence) string {
	e = e.normalize()
	if e.OK {
		return fmt.Sprintf(
			"PASS mesh wait: ready\nelapsed_ms: %d\nrequire_health: %t\ntimeout_ms: %d\ninterval_ms: %d\n",
			e.ElapsedMS, e.RequireHealth, e.TimeoutMS, e.IntervalMS,
		)
	}
	return fmt.Sprintf(
		"FAIL mesh wait: %s\nelapsed_ms: %d\nrequire_health: %t\ntimeout_ms: %d\ninterval_ms: %d\n",
		e.Error, e.ElapsedMS, e.RequireHealth, e.TimeoutMS, e.IntervalMS,
	)
}

// FormatMeshWaitResultJSON renders wait outcome as compact JSON for scrapers.
// Always emits ok, elapsed_ms, require_health, timeout_ms, interval_ms; error only when ok is false.
func FormatMeshWaitResultJSON(e MeshWaitEvidence) string {
	e = e.normalize()
	type out struct {
		OK            bool   `json:"ok"`
		ElapsedMS     int    `json:"elapsed_ms"`
		RequireHealth bool   `json:"require_health"`
		TimeoutMS     int    `json:"timeout_ms"`
		IntervalMS    int    `json:"interval_ms"`
		Error         string `json:"error,omitempty"`
	}
	o := out{
		OK:            e.OK,
		ElapsedMS:     e.ElapsedMS,
		RequireHealth: e.RequireHealth,
		TimeoutMS:     e.TimeoutMS,
		IntervalMS:    e.IntervalMS,
	}
	if !e.OK {
		o.Error = e.Error
	}
	b, err := json.Marshal(o)
	if err != nil {
		return `{"ok":false,"elapsed_ms":0,"require_health":false,"timeout_ms":0,"interval_ms":0,"error":"mesh wait json marshal failed"}` + "\n"
	}
	return string(b) + "\n"
}
