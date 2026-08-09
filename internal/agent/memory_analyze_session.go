package agent

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Analyze tick defaults (s1534 P6). Opt-in only — analyze_continuous default OFF.
const (
	defaultAnalyzeIntervalSec = 300
	minAnalyzeIntervalSec     = 30
	maxAnalyzeSummaryChars    = 2000
)

// AnalyzeTickConfig configures in-session analyze ticks (s1534 P6).
// Maps from [memory] analyze_* knobs. Opt-in only (analyze_continuous default OFF).
// Reuses MemoryStatusLine / MemoryOpsDigest — does not reimplement digest.
//
// Honesty: analyze tick ≠ invent Connected / Memory GA · dual_write OFF ·
// not Memory GA · catalog ≠ Connected · portal HITL.
type AnalyzeTickConfig struct {
	Enabled     bool
	IntervalSec int    // default 300; min floor 30
	Mode        string // "status" | "digest"
	Window      string // digest mode
	Horizon     string
	Limit       int
	Once        bool
}

// AnalyzeTickStatus is a residual-honest snapshot of in-session analyze state.
// Empty identity fields when idle; does not invent install green from Running.
type AnalyzeTickStatus struct {
	Running     bool
	Mode        string
	IntervalSec int
	LastAt      time.Time
	LastSummary string
	LastError   string
	TickCount   int
}

// DriftSnapshot residual-honest runtime state for /setup drift (no invent green).
// Does not claim Memory GA or Connected from presence of hooks alone.
type DriftSnapshot struct {
	MCPAttached    bool
	MCPServerCount int
	MemoryServerOK bool // ClientByName memory server present
	MemoryServer   string
	MeshEnabled    bool
	PullRunning    bool
	PullConsumer   string
	AnalyzeRunning bool
	DualWrite      bool // from rt.memory.DualWrite
	MemoryEnabled  bool
}

// StartAnalyzeTick starts residual-honest status/digest ticks in a background
// goroutine. Opt-in only. If a tick loop is already running, it is stopped and
// restarted. Mode "digest" requires memory hooks enabled; mode "status" can run
// with limited memory (uses MemoryStatusLine).
func (rt *Runtime) StartAnalyzeTick(cfg AnalyzeTickConfig) error {
	if rt == nil {
		return fmt.Errorf("analyze tick: no runtime")
	}
	cfg.Once = false
	return rt.startAnalyzeTick(cfg)
}

// StartAnalyzeTickOnce runs a single analyze tick then exits (dogfood / /setup analyze once).
func (rt *Runtime) StartAnalyzeTickOnce(cfg AnalyzeTickConfig) error {
	if rt == nil {
		return fmt.Errorf("analyze tick: no runtime")
	}
	cfg.Once = true
	return rt.startAnalyzeTick(cfg)
}

// StopAnalyzeTick cancels an in-flight analyze tick loop and waits for the
// goroutine to exit. No-op when not running.
func (rt *Runtime) StopAnalyzeTick() {
	if rt == nil {
		return
	}
	rt.analyzeMu.Lock()
	cancel := rt.analyzeCancel
	rt.analyzeCancel = nil
	rt.analyzeMu.Unlock()
	if cancel != nil {
		cancel()
	}
	rt.analyzeWG.Wait()
	rt.analyzeRunning.Store(false)
}

// AnalyzeTickStatus returns a residual-honest snapshot.
// When idle: Running=false and identity empty-honest (zero values).
func (rt *Runtime) AnalyzeTickStatus() AnalyzeTickStatus {
	if rt == nil {
		return AnalyzeTickStatus{}
	}
	rt.analyzeMu.Lock()
	defer rt.analyzeMu.Unlock()
	st := AnalyzeTickStatus{
		Running:     rt.analyzeRunning.Load(),
		Mode:        rt.analyzeCfg.Mode,
		IntervalSec: rt.analyzeCfg.IntervalSec,
		LastAt:      rt.analyzeLastAt,
		LastSummary: rt.analyzeLastSum,
		LastError:   rt.analyzeLastErr,
		TickCount:   rt.analyzeTicks,
	}
	// Idle empty-honest: never started / stopped with no residual identity.
	if !st.Running && st.Mode == "" && st.TickCount == 0 && st.LastSummary == "" && st.LastError == "" && st.LastAt.IsZero() {
		return AnalyzeTickStatus{Running: false}
	}
	return st
}

// DriftSnapshot returns residual-honest runtime state for /setup drift surfaces.
// No invent green: MCPAttached is manager present, MemoryServerOK is ClientByName
// only (package wire ≠ Connected claim), DualWrite mirrors config not product GA.
func (rt *Runtime) DriftSnapshot() DriftSnapshot {
	if rt == nil {
		return DriftSnapshot{}
	}
	snap := DriftSnapshot{
		MemoryEnabled: rt.memory.Enabled,
		DualWrite:     rt.memory.DualWrite,
		MemoryServer:  strings.TrimSpace(rt.memory.Server),
	}
	if snap.MemoryServer == "" {
		snap.MemoryServer = "memory"
	}
	rt.mu.Lock()
	mgr := rt.mcp
	rt.mu.Unlock()
	if mgr != nil {
		snap.MCPAttached = true
		snap.MCPServerCount = mgr.Len()
		snap.MemoryServerOK = mgr.ClientByName(snap.MemoryServer) != nil
	}
	if rt.mesh != nil && rt.mesh.Enabled() {
		snap.MeshEnabled = true
	}
	ps := rt.ContinuousMemoryPullStatus()
	snap.PullRunning = ps.Running
	snap.PullConsumer = ps.Consumer
	snap.AnalyzeRunning = rt.AnalyzeTickStatus().Running
	return snap
}

func (rt *Runtime) startAnalyzeTick(cfg AnalyzeTickConfig) error {
	mode := strings.ToLower(strings.TrimSpace(cfg.Mode))
	if mode == "" {
		mode = "status"
	}
	if mode != "status" && mode != "digest" {
		return fmt.Errorf("analyze tick: mode must be \"status\" or \"digest\" (got %q); analyze ≠ invent Connected", mode)
	}
	// Digest mode needs memory hooks enabled-ish (MemoryOpsDigest fails closed otherwise).
	if mode == "digest" && !rt.memory.Enabled {
		return fmt.Errorf("analyze tick: digest mode requires [memory] enabled (status mode still works with limited memory; analyze ≠ invent Memory GA)")
	}

	interval := cfg.IntervalSec
	if interval <= 0 {
		interval = defaultAnalyzeIntervalSec
	}
	if interval < minAnalyzeIntervalSec {
		interval = minAnalyzeIntervalSec
	}

	// Prefer stop+restart so re-Start is safe.
	rt.StopAnalyzeTick()

	resolved := AnalyzeTickConfig{
		Enabled:     true,
		IntervalSec: interval,
		Mode:        mode,
		Window:      strings.TrimSpace(cfg.Window),
		Horizon:     strings.TrimSpace(cfg.Horizon),
		Limit:       cfg.Limit,
		Once:        cfg.Once,
	}

	ctx, cancel := context.WithCancel(context.Background())

	rt.analyzeMu.Lock()
	rt.analyzeCancel = cancel
	rt.analyzeCfg = resolved
	rt.analyzeLastErr = ""
	// Preserve LastAt/LastSummary/TickCount across restart so operators see history;
	// do not invent green Connected from residual counters.
	rt.analyzeRunning.Store(true)
	rt.analyzeWG.Add(1)
	rt.analyzeMu.Unlock()

	go func() {
		defer rt.analyzeWG.Done()
		defer rt.analyzeRunning.Store(false)

		// Immediate first tick (continuous and once).
		rt.runAnalyzeTickOnce(ctx, resolved)
		if resolved.Once {
			return
		}

		ticker := time.NewTicker(time.Duration(resolved.IntervalSec) * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				rt.runAnalyzeTickOnce(ctx, resolved)
			}
		}
	}()

	return nil
}

// runAnalyzeTickOnce executes one status and/or digest tick. Fail-open: errors
// record LastError and continue (do not kill the loop).
func (rt *Runtime) runAnalyzeTickOnce(ctx context.Context, cfg AnalyzeTickConfig) {
	if ctx.Err() != nil {
		return
	}
	var summary string
	var tickErr error

	switch cfg.Mode {
	case "digest":
		opts := MemoryOpsDigestOpts{
			Window:  cfg.Window,
			Horizon: cfg.Horizon,
			Limit:   cfg.Limit,
		}
		out, err := rt.MemoryOpsDigest(ctx, opts)
		if err != nil {
			tickErr = err
			// Residual-honest fallback line so operators still see status context.
			summary = rt.MemoryStatusLine() + "\nops digest: " + err.Error()
		} else {
			summary = out
		}
	default: // status
		summary = rt.MemoryStatusLine()
	}

	summary = truncateRunes(summary, maxAnalyzeSummaryChars)

	rt.analyzeMu.Lock()
	rt.analyzeTicks++
	rt.analyzeLastAt = time.Now()
	rt.analyzeLastSum = summary
	if tickErr != nil {
		rt.analyzeLastErr = tickErr.Error()
	} else {
		rt.analyzeLastErr = ""
	}
	rt.analyzeMu.Unlock()

	if tickErr != nil && rt.logger != nil {
		rt.logger.Debug("analyze tick fail-open", "mode", cfg.Mode, "err", tickErr)
	}
}
