package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/iomesh"
)

// ContinuousPullConfig configures in-session continuous memory pull (s1530 P5).
// Maps from [memory] pull_* knobs. Opt-in only (pull_continuous default OFF).
// Reuses iomesh.Client.RunMemoryPull — does not reimplement the pull loop.
//
// Honesty: pull running ≠ invent install green / Ops Pack GA · dual_write OFF ·
// not Memory GA · catalog ≠ Connected · portal HITL.
type ContinuousPullConfig struct {
	Enabled   bool // pull_continuous
	Stream    string
	Consumer  string
	Filter    string
	Batch     int
	MaxWaitMS int
	// DryRun maps + logs only; no MCP local ingest.
	DryRun bool
	// Once runs a single fetch cycle (MaxLoops=1); continuous uses MaxLoops=0.
	Once bool
	// Server is the MCP memory server name (default "memory" / [memory].server).
	Server string
	// Tenant optional MCP ingest tenant (falls back to Runtime memory/mesh tenant).
	Tenant string
}

// ContinuousPullStatus is a residual-honest snapshot of in-session pull state.
// Empty identity fields when idle; does not invent install green from Running.
type ContinuousPullStatus struct {
	Running   bool
	Stream    string
	Consumer  string
	Filter    string
	Stats     iomesh.MemoryPullStats
	LastError string
}

// StartContinuousMemoryPull starts mesh durable consumer → local MCP palace pull
// in a background goroutine (MaxLoops=0 continuous). Opt-in only.
//
// If a pull is already running, it is stopped and restarted (ReplaceMCP-friendly).
// Requires mesh enabled + non-empty Consumer. Non-dry-run requires a connected
// MCP memory client at start (LocalIngest re-reads MCP each call so /setup reload
// can hot-swap without restarting pull).
func (rt *Runtime) StartContinuousMemoryPull(cfg ContinuousPullConfig) error {
	if rt == nil {
		return fmt.Errorf("continuous memory pull: no runtime")
	}
	cfg.Once = false
	return rt.startContinuousMemoryPull(cfg)
}

// StartContinuousMemoryPullOnce is StartContinuousMemoryPull with MaxLoops=1
// (single fetch cycle; useful for dogfood / /setup pull once).
func (rt *Runtime) StartContinuousMemoryPullOnce(cfg ContinuousPullConfig) error {
	if rt == nil {
		return fmt.Errorf("continuous memory pull: no runtime")
	}
	cfg.Once = true
	return rt.startContinuousMemoryPull(cfg)
}

// StopContinuousMemoryPull cancels an in-flight continuous pull and waits for
// the goroutine to exit. No-op when not running.
func (rt *Runtime) StopContinuousMemoryPull() {
	if rt == nil {
		return
	}
	rt.pullMu.Lock()
	cancel := rt.pullCancel
	rt.pullCancel = nil
	rt.pullMu.Unlock()
	if cancel != nil {
		cancel()
	}
	rt.pullWG.Wait()
	rt.pullRunning.Store(false)
}

// ContinuousMemoryPullStatus returns a residual-honest snapshot.
// When idle: Running=false and identity/stats empty-honest (zero values).
func (rt *Runtime) ContinuousMemoryPullStatus() ContinuousPullStatus {
	if rt == nil {
		return ContinuousPullStatus{}
	}
	rt.pullMu.Lock()
	defer rt.pullMu.Unlock()
	st := ContinuousPullStatus{
		Running:   rt.pullRunning.Load(),
		Stream:    rt.pullCfg.Stream,
		Consumer:  rt.pullCfg.Consumer,
		Filter:    rt.pullStats.Filter,
		Stats:     rt.pullStats,
		LastError: rt.pullLastErr,
	}
	if st.Filter == "" {
		st.Filter = rt.pullCfg.Filter
	}
	// Idle empty-honest: clear identity when never started / stopped with no cfg.
	if !st.Running && st.Consumer == "" && st.Stream == "" {
		return ContinuousPullStatus{Running: false}
	}
	return st
}

func (rt *Runtime) startContinuousMemoryPull(cfg ContinuousPullConfig) error {
	consumer := strings.TrimSpace(cfg.Consumer)
	if consumer == "" {
		return fmt.Errorf("continuous memory pull: pull_consumer required (set [memory].pull_consumer)")
	}
	if rt.mesh == nil || !rt.mesh.Enabled() {
		return fmt.Errorf("continuous memory pull: mesh disabled (set IOMESH_ENDPOINT / [iomesh]; pull running ≠ invent install green)")
	}

	stream := strings.TrimSpace(cfg.Stream)
	if stream == "" {
		stream = "EVENTS"
	}
	server := strings.TrimSpace(cfg.Server)
	if server == "" {
		server = strings.TrimSpace(rt.memory.Server)
	}
	if server == "" {
		server = "memory"
	}
	tenant := strings.TrimSpace(cfg.Tenant)
	if tenant == "" {
		tenant = rt.memoryTenant()
	}
	batch := cfg.Batch
	if batch <= 0 {
		batch = 8
	}
	maxWaitMS := cfg.MaxWaitMS
	if maxWaitMS <= 0 {
		maxWaitMS = 2000
	}
	filter := strings.TrimSpace(cfg.Filter)

	// Fail-closed preflight for mutating path: MCP memory client must be present now.
	// LocalIngest still re-reads manager each call so /setup reload can hot-swap.
	if !cfg.DryRun {
		rt.mu.Lock()
		mgr := rt.mcp
		rt.mu.Unlock()
		if mgr == nil || mgr.ClientByName(server) == nil {
			return fmt.Errorf("continuous memory pull: MCP memory server %q not connected (use --dry-run or attach MCP; package wire ≠ Connected)", server)
		}
	}

	// Prefer stop+restart so ReplaceMCP / re-Start is safe.
	rt.StopContinuousMemoryPull()

	resolved := ContinuousPullConfig{
		Enabled:   true,
		Stream:    stream,
		Consumer:  consumer,
		Filter:    filter,
		Batch:     batch,
		MaxWaitMS: maxWaitMS,
		DryRun:    cfg.DryRun,
		Once:      cfg.Once,
		Server:    server,
		Tenant:    tenant,
	}

	ctx, cancel := context.WithCancel(context.Background())

	rt.pullMu.Lock()
	rt.pullCancel = cancel
	rt.pullCfg = resolved
	rt.pullLastErr = ""
	rt.pullStats = iomesh.MemoryPullStats{
		Stream:   stream,
		Consumer: consumer,
		Filter:   filter,
	}
	rt.pullRunning.Store(true)
	rt.pullWG.Add(1)
	rt.pullMu.Unlock()

	// Capture server/tenant for LocalIngest closure (re-read MCP each call).
	ingestServer := server
	ingestTenant := tenant

	opt := iomesh.MemoryPullOptions{
		Stream:  stream,
		Name:    consumer,
		Filter:  filter,
		Batch:   batch,
		MaxWait: time.Duration(maxWaitMS) * time.Millisecond,
		Ack:     true,
		DryRun:  cfg.DryRun,
		OnMessage: func(_ iomesh.StreamMessage, _ iomesh.MemoryEnvelope, skipped bool, err error) {
			rt.pullMu.Lock()
			defer rt.pullMu.Unlock()
			rt.pullStats.Fetched++
			if skipped {
				rt.pullStats.Skipped++
				return
			}
			if err != nil {
				rt.pullStats.Errors++
				rt.pullStats.LastError = err.Error()
				rt.pullLastErr = err.Error()
				return
			}
			rt.pullStats.Ingested++
		},
	}
	if cfg.Once {
		opt.MaxLoops = 1
	}
	if !cfg.DryRun {
		opt.LocalIngest = func(cctx context.Context, env iomesh.MemoryEnvelope) error {
			// Re-read current MCP manager each call so /setup reload works without restarting pull.
			rt.mu.Lock()
			mgr := rt.mcp
			rt.mu.Unlock()
			if mgr == nil {
				return fmt.Errorf("MCP manager detached (reload / package wire ≠ Connected)")
			}
			cl := mgr.ClientByName(ingestServer)
			if cl == nil {
				return fmt.Errorf("MCP server %q not connected", ingestServer)
			}
			args := map[string]any{
				"role":    env.Role,
				"content": env.Content,
			}
			if env.EventTime != "" {
				args["event_time"] = env.EventTime
			}
			if env.SessionID != "" {
				args["session_id"] = env.SessionID
			}
			if ingestTenant != "" {
				args["tenant"] = ingestTenant
			}
			_, err := cl.CallTool(cctx, "memory_ingest_turn", args)
			return err
		}
	}

	mesh := rt.mesh
	go func() {
		defer rt.pullWG.Done()
		st, err := mesh.RunMemoryPull(ctx, opt)
		rt.pullMu.Lock()
		// Preserve effective filter from RunMemoryPull when available.
		if st.Filter != "" {
			rt.pullCfg.Filter = st.Filter
		}
		rt.pullStats = st
		if err != nil && ctx.Err() == nil {
			rt.pullLastErr = err.Error()
			if st.LastError == "" {
				rt.pullStats.LastError = err.Error()
			}
		} else if err != nil && ctx.Err() != nil {
			// Cancel is expected stop — keep last stats; leave LastError only if prior soft errors.
		}
		// Clear cancel ownership if we are still the active pull.
		if rt.pullCancel != nil {
			// Do not nil cancel here if Stop already nil'd it; just mark not running.
		}
		rt.pullRunning.Store(false)
		rt.pullMu.Unlock()
	}()

	return nil
}
