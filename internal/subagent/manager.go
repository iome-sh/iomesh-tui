package subagent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// Runner executes one subagent turn (implemented by agent.Runtime via adapter).
type Runner interface {
	Run(ctx context.Context, systemPrompt, userPrompt string) (summary string, err error)
}

// RunnerFactory builds a Runner for a spawn.
type RunnerFactory func(ctx context.Context, sp SpawnParams) (Runner, error)

// SpawnParams is what the factory needs to construct a child runtime.
type SpawnParams struct {
	Spec       Spec
	Definition Definition
	// Effective tools/capability after merging type defaults + mode.
	Capability CapabilityMode
	// AllowWrite/AllowShell after capability filter.
	AllowWrite bool
	AllowShell bool
	AllowSpawn bool
	// Workspace root for the child (may be a worktree path).
	Workspace string
	// ParentYolo: mutating tools auto-approved when true.
	ParentYolo bool
	// Resume messages from a completed subagent (optional).
	ResumeMessages any
}

// Default concurrency / batch limits (open-source agent fan-out).
const (
	DefaultMaxConcurrent = 16
	DefaultMaxBatch      = 32
	DefaultMaxDepth      = 2
)

// Config tunes the manager.
type Config struct {
	Enabled       bool
	MaxConcurrent int
	MaxDepth      int
	// MaxBatch caps SpawnMany / spawn_subagents array size.
	MaxBatch int
	// Workspace is the default child cwd.
	Workspace string
	Yolo      bool
}

// SpawnManyOptions controls batch fan-out.
type SpawnManyOptions struct {
	// Wait blocks until every spawned child finishes (completed/failed/cancelled).
	Wait bool
}

// BatchResult is the response for parallel multi-spawn.
type BatchResult struct {
	Results       []Result `json:"results"`
	MaxConcurrent int      `json:"max_concurrent"`
	Spawned       int      `json:"spawned"`
	Completed     int      `json:"completed,omitempty"`
	Failed        int      `json:"failed,omitempty"`
	Running       int      `json:"running,omitempty"`
	Waited        bool     `json:"waited"`
}

// Manager orchestrates spawn, tracking, and retrieval.
type Manager struct {
	cfg     Config
	reg     *Registry
	factory RunnerFactory
	logger  *slog.Logger
	// parentDepth is 0 for the primary session manager.
	parentDepth int

	mu       sync.Mutex
	sem      chan struct{} // concurrency limiter
	worktree WorktreeBackend
}

// WorktreeBackend isolates a spawn (optional).
type WorktreeBackend interface {
	Create(ctx context.Context, parentRoot, id string) (path string, cleanup func() error, err error)
}

// NopWorktree rejects worktree isolation.
type NopWorktree struct{}

func (NopWorktree) Create(context.Context, string, string) (string, func() error, error) {
	return "", nil, fmt.Errorf("worktree isolation not implemented yet; use isolation=none")
}

// NewManager constructs a Manager. factory must not be nil when Enabled.
func NewManager(cfg Config, factory RunnerFactory, logger *slog.Logger) *Manager {
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = DefaultMaxConcurrent
	}
	if cfg.MaxDepth <= 0 {
		cfg.MaxDepth = DefaultMaxDepth
	}
	if cfg.MaxBatch <= 0 {
		cfg.MaxBatch = DefaultMaxBatch
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Manager{
		cfg:      cfg,
		reg:      NewRegistry(),
		factory:  factory,
		logger:   logger,
		sem:      make(chan struct{}, cfg.MaxConcurrent),
		worktree: NopWorktree{},
	}
}

// MaxConcurrent returns the configured parallel slot count.
func (m *Manager) MaxConcurrent() int {
	if m == nil {
		return 0
	}
	return m.cfg.MaxConcurrent
}

// MaxBatch returns the configured multi-spawn array cap.
func (m *Manager) MaxBatch() int {
	if m == nil {
		return 0
	}
	return m.cfg.MaxBatch
}

// SetWorktreeBackend replaces the isolation backend.
func (m *Manager) SetWorktreeBackend(b WorktreeBackend) {
	if b != nil {
		m.worktree = b
	}
}

// Enabled reports whether subagents are available.
func (m *Manager) Enabled() bool {
	return m != nil && m.cfg.Enabled
}

// Registry exposes the underlying store (for TUI / diagnostics).
func (m *Manager) Registry() *Registry {
	if m == nil {
		return nil
	}
	return m.reg
}

// Spawn starts a subagent per Spec. Synchronous unless Spec.Background.
func (m *Manager) Spawn(ctx context.Context, spec Spec) (Result, error) {
	if m == nil || !m.cfg.Enabled {
		return Result{}, fmt.Errorf("subagents disabled")
	}
	if strings.TrimSpace(spec.Prompt) == "" {
		return Result{}, fmt.Errorf("prompt is required")
	}
	if spec.SubagentType == "" {
		spec.SubagentType = TypeGeneralPurpose
	}
	if spec.Isolation == "" {
		spec.Isolation = IsolationNone
	}
	if spec.Depth == 0 && m.parentDepth > 0 {
		spec.Depth = m.parentDepth
	}
	if spec.Depth >= m.cfg.MaxDepth {
		return Result{}, fmt.Errorf("max subagent depth %d exceeded", m.cfg.MaxDepth)
	}

	defs := Builtins()
	def, ok := defs[spec.SubagentType]
	if !ok {
		return Result{}, fmt.Errorf("unknown subagent_type %q (want general-purpose|explore|plan)", spec.SubagentType)
	}

	capMode := spec.CapabilityMode
	if capMode == "" {
		capMode = def.DefaultCapability
	}
	allowWrite, allowShell := resolveCapabilities(def, capMode)

	id := m.reg.AllocID()
	if spec.Description == "" {
		spec.Description = string(spec.SubagentType)
	}

	rec := &Record{
		ID:        id,
		Spec:      spec,
		Status:    StatusPending,
		StartedAt: time.Now().UTC(),
	}
	m.reg.Put(rec)

	// resume_from: copy summary context note (full transcript resume in later PR).
	var resumeNote string
	if spec.ResumeFrom != "" {
		prev, ok := m.reg.Get(spec.ResumeFrom)
		if !ok {
			_ = m.reg.Update(id, func(r *Record) {
				r.Status = StatusFailed
				r.Error = fmt.Sprintf("resume_from %q not found", spec.ResumeFrom)
				r.FinishedAt = time.Now().UTC()
			})
			return Result{}, fmt.Errorf("resume_from %q not found", spec.ResumeFrom)
		}
		if prev.Status != StatusCompleted {
			return Result{}, fmt.Errorf("resume_from %q status=%s (need completed)", spec.ResumeFrom, prev.Status)
		}
		resumeNote = prev.Summary
		_ = m.reg.Update(id, func(r *Record) { r.MessagesCopied = 1 })
	}

	workspace := m.cfg.Workspace
	if spec.CWD != "" {
		workspace = spec.CWD
	}
	var cleanup func() error
	if spec.Isolation == IsolationWorktree {
		path, c, err := m.worktree.Create(ctx, workspace, id)
		if err != nil {
			_ = m.reg.Update(id, func(r *Record) {
				r.Status = StatusFailed
				r.Error = err.Error()
				r.FinishedAt = time.Now().UTC()
			})
			return Result{}, err
		}
		workspace = path
		cleanup = c
		_ = m.reg.Update(id, func(r *Record) { r.WorktreePath = path })
	}

	params := SpawnParams{
		Spec:       spec,
		Definition: def,
		Capability: capMode,
		AllowWrite: allowWrite,
		AllowShell: allowShell,
		AllowSpawn: def.AllowSpawn && spec.Depth+1 < m.cfg.MaxDepth,
		Workspace:  workspace,
		ParentYolo: m.cfg.Yolo,
	}

	run := func() Result {
		// Concurrency limit.
		select {
		case m.sem <- struct{}{}:
			defer func() { <-m.sem }()
		case <-ctx.Done():
			_ = m.reg.Update(id, func(r *Record) {
				r.Status = StatusCancelled
				r.Error = ctx.Err().Error()
				r.FinishedAt = time.Now().UTC()
			})
			return m.resultOf(id)
		}

		_ = m.reg.Update(id, func(r *Record) { r.Status = StatusRunning })
		m.logger.Info("subagent start",
			"id", id,
			"type", spec.SubagentType,
			"desc", spec.Description,
			"background", spec.Background,
			"capability", capMode,
		)

		runner, err := m.factory(ctx, params)
		if err != nil {
			_ = m.reg.Update(id, func(r *Record) {
				r.Status = StatusFailed
				r.Error = err.Error()
				r.FinishedAt = time.Now().UTC()
			})
			if cleanup != nil {
				_ = cleanup()
			}
			return m.resultOf(id)
		}

		userPrompt := spec.Prompt
		if resumeNote != "" {
			userPrompt = "Prior subagent summary (resume_from):\n" + resumeNote + "\n\n---\n\nNew task:\n" + spec.Prompt
		}

		summary, err := runner.Run(ctx, def.SystemPrompt, userPrompt)
		finished := time.Now().UTC()
		if err != nil {
			_ = m.reg.Update(id, func(r *Record) {
				r.Status = StatusFailed
				r.Error = err.Error()
				r.Summary = summary
				r.FinishedAt = finished
			})
			m.logger.Warn("subagent failed", "id", id, "err", err)
		} else {
			_ = m.reg.Update(id, func(r *Record) {
				r.Status = StatusCompleted
				r.Summary = summary
				r.FinishedAt = finished
			})
			m.logger.Info("subagent completed", "id", id, "chars", len(summary))
		}
		// Keep worktree for inspection; cleanup is optional later.
		return m.resultOf(id)
	}

	if spec.Background {
		go func() {
			// Detached from parent cancel so background work can finish;
			// still bound by a generous timeout.
			bg, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()
			_ = runWithContext(bg, run)
		}()
		// Return pending/running immediately.
		time.Sleep(5 * time.Millisecond) // let status flip to running when possible
		return m.resultOf(id), nil
	}

	return run(), nil
}

func runWithContext(ctx context.Context, run func() Result) Result {
	done := make(chan Result, 1)
	go func() { done <- run() }()
	select {
	case r := <-done:
		return r
	case <-ctx.Done():
		return Result{Status: StatusCancelled, Error: ctx.Err().Error()}
	}
}

// Get returns the current result for a subagent id.
func (m *Manager) Get(id string) (Result, error) {
	if m == nil {
		return Result{}, fmt.Errorf("subagents disabled")
	}
	rec, ok := m.reg.Get(id)
	if !ok {
		return Result{}, fmt.Errorf("subagent %q not found", id)
	}
	return recordToResult(rec), nil
}

// Wait blocks until the subagent finishes or ctx is done.
func (m *Manager) Wait(ctx context.Context, id string) (Result, error) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		res, err := m.Get(id)
		if err != nil {
			return Result{}, err
		}
		switch res.Status {
		case StatusCompleted, StatusFailed, StatusCancelled:
			return res, nil
		}
		select {
		case <-ctx.Done():
			return res, ctx.Err()
		case <-ticker.C:
		}
	}
}

// SpawnMany starts many subagents in parallel (always background-started so the
// semaphore fans out up to MaxConcurrent). When opts.Wait is true, blocks until
// every child reaches a terminal status.
func (m *Manager) SpawnMany(ctx context.Context, specs []Spec, opts SpawnManyOptions) (BatchResult, error) {
	if m == nil || !m.cfg.Enabled {
		return BatchResult{}, fmt.Errorf("subagents disabled")
	}
	if len(specs) == 0 {
		return BatchResult{}, fmt.Errorf("spawn_subagents: empty tasks array")
	}
	if len(specs) > m.cfg.MaxBatch {
		return BatchResult{}, fmt.Errorf("spawn_subagents: %d tasks exceeds max_batch %d", len(specs), m.cfg.MaxBatch)
	}

	out := BatchResult{
		Results:       make([]Result, 0, len(specs)),
		MaxConcurrent: m.cfg.MaxConcurrent,
		Waited:        opts.Wait,
	}

	// Fan-out: always background so N tasks start without serializing on Run().
	for i, s := range specs {
		s.Background = true
		if s.Description == "" {
			s.Description = fmt.Sprintf("parallel-%d", i+1)
		}
		res, err := m.Spawn(ctx, s)
		if err != nil {
			// Record failure placeholder so callers see partial batch.
			out.Results = append(out.Results, Result{
				Status:      StatusFailed,
				Error:       err.Error(),
				Description: s.Description,
			})
			out.Failed++
			continue
		}
		out.Results = append(out.Results, res)
		out.Spawned++
	}

	if opts.Wait {
		waited := make([]Result, 0, len(out.Results))
		for _, r := range out.Results {
			if r.ID == "" {
				waited = append(waited, r)
				continue
			}
			final, err := m.Wait(ctx, r.ID)
			if err != nil && final.ID == "" {
				final = r
				final.Error = err.Error()
				final.Status = StatusCancelled
			}
			waited = append(waited, final)
		}
		out.Results = waited
	}

	out.Completed, out.Failed, out.Running = 0, 0, 0
	for _, r := range out.Results {
		switch r.Status {
		case StatusCompleted:
			out.Completed++
		case StatusFailed, StatusCancelled:
			out.Failed++
		default:
			out.Running++
		}
	}
	m.logger.Info("subagent batch",
		"spawned", out.Spawned,
		"completed", out.Completed,
		"failed", out.Failed,
		"running", out.Running,
		"max_concurrent", out.MaxConcurrent,
		"waited", out.Waited,
	)
	return out, nil
}

// WaitAll waits for every listed id (order preserved). Missing ids become Failed results.
func (m *Manager) WaitAll(ctx context.Context, ids []string) ([]Result, error) {
	if m == nil || !m.cfg.Enabled {
		return nil, fmt.Errorf("subagents disabled")
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("wait_subagents: empty ids")
	}
	out := make([]Result, 0, len(ids))
	for _, id := range ids {
		res, err := m.Wait(ctx, id)
		if err != nil {
			out = append(out, Result{ID: id, Status: StatusFailed, Error: err.Error()})
			continue
		}
		out = append(out, res)
	}
	return out, nil
}

// GetMany returns current status for each id without blocking.
func (m *Manager) GetMany(ids []string) []Result {
	out := make([]Result, 0, len(ids))
	for _, id := range ids {
		res, err := m.Get(id)
		if err != nil {
			out = append(out, Result{ID: id, Status: StatusFailed, Error: err.Error()})
			continue
		}
		out = append(out, res)
	}
	return out
}

func (m *Manager) resultOf(id string) Result {
	rec, ok := m.reg.Get(id)
	if !ok {
		return Result{ID: id, Status: StatusFailed, Error: "missing record"}
	}
	return recordToResult(rec)
}

func recordToResult(rec Record) Result {
	var dur int64
	if !rec.FinishedAt.IsZero() {
		dur = rec.FinishedAt.Sub(rec.StartedAt).Milliseconds()
	} else if !rec.StartedAt.IsZero() {
		dur = time.Since(rec.StartedAt).Milliseconds()
	}
	return Result{
		ID:           rec.ID,
		Status:       rec.Status,
		Summary:      rec.Summary,
		Error:        rec.Error,
		WorktreePath: rec.WorktreePath,
		Description:  rec.Spec.Description,
		SubagentType: rec.Spec.SubagentType,
		DurationMS:   dur,
	}
}

// resolveCapabilities merges type defaults with explicit mode.
func resolveCapabilities(def Definition, mode CapabilityMode) (allowWrite, allowShell bool) {
	allowWrite = def.AllowWrite
	allowShell = def.AllowShell
	switch mode {
	case CapabilityReadOnly:
		return false, false
	case CapabilityReadWrite:
		return def.AllowWrite, false
	case CapabilityExecute:
		return false, def.AllowShell
	case CapabilityAll:
		return def.AllowWrite, def.AllowShell
	default:
		return allowWrite, allowShell
	}
}

// EffectiveTools lists tool names permitted for a spawn.
func EffectiveTools(allowWrite, allowShell, allowSpawn bool) []string {
	tools := []string{"read_file", "list_dir", "grep"}
	if allowShell {
		tools = append(tools, "run_shell")
	}
	if allowWrite {
		tools = append(tools, "write_file")
	}
	if allowSpawn {
		tools = append(tools,
			"spawn_subagent", "spawn_subagents",
			"get_subagent_output", "wait_subagents",
		)
	}
	return tools
}
