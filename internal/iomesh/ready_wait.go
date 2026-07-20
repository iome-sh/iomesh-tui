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

// FormatMeshWaitResult renders operator preflight wait outcome as text.
// Always includes elapsed_ms for CI evidence (success and failure).
func FormatMeshWaitResult(ok bool, elapsedMS int, errMsg string) string {
	if elapsedMS < 0 {
		elapsedMS = 0
	}
	if ok {
		return fmt.Sprintf("PASS mesh wait: ready\nelapsed_ms: %d\n", elapsedMS)
	}
	if errMsg == "" {
		errMsg = "unknown error"
	}
	return fmt.Sprintf("FAIL mesh wait: %s\nelapsed_ms: %d\n", errMsg, elapsedMS)
}

// FormatMeshWaitResultJSON renders wait outcome as compact JSON for scrapers.
// Always emits ok and elapsed_ms; error only when ok is false.
func FormatMeshWaitResultJSON(ok bool, elapsedMS int, errMsg string) string {
	if elapsedMS < 0 {
		elapsedMS = 0
	}
	type out struct {
		OK        bool   `json:"ok"`
		ElapsedMS int    `json:"elapsed_ms"`
		Error     string `json:"error,omitempty"`
	}
	o := out{OK: ok, ElapsedMS: elapsedMS}
	if !ok {
		if errMsg == "" {
			errMsg = "unknown error"
		}
		o.Error = errMsg
	}
	b, err := json.Marshal(o)
	if err != nil {
		return `{"ok":false,"elapsed_ms":0,"error":"mesh wait json marshal failed"}` + "\n"
	}
	return string(b) + "\n"
}
