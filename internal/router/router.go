package router

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/security"
)

// MetricsSink receives optional call-level metrics (can be a no-op).
type MetricsSink interface {
	RecordLLMCall(meta CallMeta, usage Usage, err error)
}

// NopMetrics is a no-op MetricsSink.
type NopMetrics struct{}

func (NopMetrics) RecordLLMCall(CallMeta, Usage, error) {}

// Router selects models and executes OpenAI-compatible chat completions
// with cascading fallback on retryable errors.
type Router struct {
	mu           sync.RWMutex
	models       []ModelConfig
	clients      map[string]LLMClient
	logger       *slog.Logger
	defaultModel string
	override     string // session-level /model override
	metrics      MetricsSink
	maxAttempts  int
}

// Option configures a Router.
type Option func(*Router)

// WithLogger sets the structured logger.
func WithLogger(l *slog.Logger) Option {
	return func(r *Router) {
		if l != nil {
			r.logger = l
		}
	}
}

// WithMetrics sets a metrics sink.
func WithMetrics(m MetricsSink) Option {
	return func(r *Router) {
		if m != nil {
			r.metrics = m
		}
	}
}

// WithMaxAttempts overrides the fallback attempt budget (default 3).
func WithMaxAttempts(n int) Option {
	return func(r *Router) {
		if n > 0 {
			r.maxAttempts = n
		}
	}
}

// WithClientFactory replaces default HTTP clients (useful for tests).
func WithClientFactory(factory func(ModelConfig) LLMClient) Option {
	return func(r *Router) {
		if factory == nil {
			return
		}
		clients := make(map[string]LLMClient, len(r.models))
		for _, m := range r.models {
			clients[m.Name] = factory(m)
		}
		r.clients = clients
	}
}

// New creates a Router from model configs. Models are sorted by Priority ascending.
func New(models []ModelConfig, defaultModel string, opts ...Option) (*Router, error) {
	if len(models) == 0 {
		return nil, fmt.Errorf("router: at least one model is required")
	}
	copied := append([]ModelConfig(nil), models...)
	sort.SliceStable(copied, func(i, j int) bool {
		return copied[i].Priority < copied[j].Priority
	})

	if defaultModel == "" {
		defaultModel = copied[0].Name
	}
	if !hasModel(copied, defaultModel) {
		return nil, fmt.Errorf("router: default model %q not found in catalog", defaultModel)
	}

	clients := make(map[string]LLMClient, len(copied))
	for _, m := range copied {
		if m.Name == "" {
			return nil, fmt.Errorf("router: model with empty name")
		}
		if m.BaseURL == "" {
			return nil, fmt.Errorf("router: model %q missing base_url", m.Name)
		}
		if m.ModelID == "" {
			return nil, fmt.Errorf("router: model %q missing model_id", m.Name)
		}
		// Allow loopback so local OpenAI-compatible servers and tests work.
		// file:// and non-http schemes are always rejected.
		if err := security.ValidateHTTPURL(m.BaseURL, true); err != nil {
			return nil, fmt.Errorf("router: model %q base_url: %w", m.Name, err)
		}
		// Strip any accidental userinfo (user:pass@host) from base URL.
		if cleaned, err := stripURLUserinfo(m.BaseURL); err == nil {
			m.BaseURL = cleaned
		}
		// Never keep inline API keys in long-lived process dumps via Stringers;
		// keys stay on the struct for auth but are redacted in logs.
		clients[m.Name] = NewHTTPClient(m, nil)
	}

	r := &Router{
		models:       copied,
		clients:      clients,
		logger:       slog.Default(),
		defaultModel: defaultModel,
		metrics:      NopMetrics{},
		maxAttempts:  3,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r, nil
}

// Models returns a copy of the configured catalog (priority-sorted).
func (r *Router) Models() []ModelConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ModelConfig, len(r.models))
	copy(out, r.models)
	return out
}

// DefaultModel returns the configured default logical name.
func (r *Router) DefaultModel() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.defaultModel
}

// SetOverride pins model selection to name until cleared (empty string clears).
// Used by CLI -m and TUI /model.
func (r *Router) SetOverride(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if name == "" {
		r.override = ""
		return nil
	}
	if _, ok := r.clients[name]; !ok {
		return fmt.Errorf("unknown model %q", name)
	}
	r.override = name
	return nil
}

// Override returns the current session override, if any.
func (r *Router) Override() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.override
}

// Model returns the config for a logical name.
func (r *Router) Model(name string) (ModelConfig, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, m := range r.models {
		if m.Name == name {
			return m, true
		}
	}
	return ModelConfig{}, false
}

// SelectModel returns the best logical model name for the given params.
func (r *Router) SelectModel(p SelectParams) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if p.Override != "" {
		if _, ok := r.clients[p.Override]; ok {
			return p.Override
		}
	}
	if r.override != "" {
		return r.override
	}

	complexity := p.Complexity
	if complexity == ComplexityRoutine {
		// Infer complexity from task type when caller left it at zero.
		switch p.TaskType {
		case TaskPlan, TaskReview:
			complexity = ComplexityPlan
		case TaskEdit:
			// Multi-file edits are mid-tier; still prefer value models first.
			complexity = ComplexityRoutine
		}
	}

	candidates := r.eligible(p.EstimatedTokens)
	if len(candidates) == 0 {
		return r.defaultModel
	}

	// Capability filter (soft): prefer matches, do not hard-fail.
	if len(p.PreferCapabilities) > 0 {
		if filtered := filterByAnyCapability(candidates, p.PreferCapabilities); len(filtered) > 0 {
			candidates = filtered
		}
	}

	switch complexity {
	case ComplexityHighStakes:
		// Prefer premium / higher cost_tier coding models.
		if m := firstWithCapability(candidates, "premium"); m != "" {
			return m
		}
		if m := lastByCostTier(candidates); m != "" {
			return m
		}
	case ComplexityPlan:
		// Prefer stronger coding models; still cost-aware (Pro over Flash if tagged).
		if m := firstWithAllCapabilities(candidates, "coding"); m != "" {
			// Among coding-capable, pick mid tier when available (not cheapest flash-only).
			coding := filterByAnyCapability(candidates, []string{"coding"})
			if mid := pickByComplexity(coding, ComplexityPlan); mid != "" {
				return mid
			}
			return m
		}
	default: // routine / subagent
		if m := firstWithCapability(candidates, "fast"); m != "" {
			return m
		}
		if m := firstByCostTier(candidates); m != "" {
			return m
		}
	}

	return candidates[0].Name
}

// ExecuteWithFallback runs a non-streaming completion with automatic fallback.
func (r *Router) ExecuteWithFallback(ctx context.Context, req ChatRequest, p SelectParams) (ChatResponse, CallMeta, error) {
	return r.execute(ctx, req, p, false, nil)
}

// ExecuteStreamWithFallback runs a streaming completion with automatic fallback.
func (r *Router) ExecuteStreamWithFallback(ctx context.Context, req ChatRequest, p SelectParams, onDelta func(StreamDelta) error) (ChatResponse, CallMeta, error) {
	return r.execute(ctx, req, p, true, onDelta)
}

func (r *Router) execute(ctx context.Context, req ChatRequest, p SelectParams, stream bool, onDelta func(StreamDelta) error) (ChatResponse, CallMeta, error) {
	modelName := r.SelectModel(p)
	chain := r.fallbackChain(modelName)
	if len(chain) == 0 {
		return ChatResponse{}, CallMeta{}, fmt.Errorf("no models available")
	}

	var lastErr error
	meta := CallMeta{FallbackUsed: false}

	for i, name := range chain {
		if i >= r.maxAttempts {
			break
		}
		client, ok := r.client(name)
		if !ok {
			r.logger.Warn("model client not found", "model", name)
			continue
		}
		cfg, _ := r.Model(name)
		reqCopy := req
		reqCopy.Model = cfg.ModelID

		start := time.Now()
		var (
			resp ChatResponse
			err  error
		)
		if stream {
			resp, err = client.ChatCompletionStream(ctx, reqCopy, onDelta)
		} else {
			resp, err = client.ChatCompletion(ctx, reqCopy)
		}
		dur := time.Since(start)
		meta.Attempts = i + 1
		meta.ModelName = name
		meta.ModelID = cfg.ModelID
		meta.Duration = dur
		meta.FallbackUsed = i > 0

		if err == nil {
			meta.EstimatedUSD = r.EstimateCost(name, resp.Usage).USD
			r.logger.Info("llm call succeeded",
				"model", name,
				"model_id", cfg.ModelID,
				"duration_ms", dur.Milliseconds(),
				"tokens", resp.Usage.TotalTokens,
				"est_usd", meta.EstimatedUSD,
				"fallback", meta.FallbackUsed,
				"task", string(p.TaskType),
				"complexity", p.Complexity.String(),
			)
			r.metrics.RecordLLMCall(meta, resp.Usage, nil)
			return resp, meta, nil
		}

		lastErr = err
		r.logger.Warn("llm call failed, considering fallback",
			"model", name,
			"error", security.Redact(err.Error()),
			"attempt", i+1,
			"retryable", isRetryable(err),
		)
		r.metrics.RecordLLMCall(meta, Usage{}, err)

		if !isRetryable(err) && !isRateLimit(err) {
			// Auth / 4xx: still try next model in chain (different provider may work).
			// Continue unless it is a context cancellation.
			if ctx.Err() != nil {
				return ChatResponse{}, meta, ctx.Err()
			}
		}
		if ctx.Err() != nil {
			return ChatResponse{}, meta, ctx.Err()
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("all fallback models exhausted")
	}
	return ChatResponse{}, meta, fmt.Errorf("all fallback models exhausted after %d attempts: %w", meta.Attempts, lastErr)
}

// EstimateCost projects USD cost for a usage blob on model name.
func (r *Router) EstimateCost(modelName string, usage Usage) CostEstimate {
	cfg, ok := r.Model(modelName)
	if !ok {
		return CostEstimate{ModelName: modelName}
	}
	return estimate(cfg, usage.PromptTokens, usage.CompletionTokens, cachedTokens(usage))
}

// EstimateCostTokens projects USD for expected token counts.
func (r *Router) EstimateCostTokens(modelName string, inputTokens, outputTokens, cachedInput int) CostEstimate {
	cfg, ok := r.Model(modelName)
	if !ok {
		return CostEstimate{ModelName: modelName, InputTokens: inputTokens, OutputTokens: outputTokens, CachedInputTokens: cachedInput}
	}
	return estimate(cfg, inputTokens, outputTokens, cachedInput)
}

func estimate(cfg ModelConfig, input, output, cached int) CostEstimate {
	est := CostEstimate{
		ModelName:         cfg.Name,
		InputTokens:       input,
		OutputTokens:      output,
		CachedInputTokens: cached,
	}
	inRate := cfg.InputCostPerM
	outRate := cfg.OutputCostPerM
	cacheRate := cfg.CacheHitCostPerM
	if inRate == 0 && outRate == 0 && cfg.CostTier > 0 {
		// Relative placeholder when absolute rates unset: $0.10 * cost_tier per 1M blended.
		blended := 0.10 * cfg.CostTier
		est.USD = float64(input+output) / 1_000_000 * blended
		return est
	}
	billableInput := input - cached
	if billableInput < 0 {
		billableInput = 0
	}
	est.USD = float64(billableInput)/1_000_000*inRate +
		float64(cached)/1_000_000*cacheRate +
		float64(output)/1_000_000*outRate
	return est
}

func cachedTokens(u Usage) int {
	if u.PromptTokensDetails != nil {
		return u.PromptTokensDetails.CachedTokens
	}
	return 0
}

func (r *Router) client(name string) (LLMClient, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.clients[name]
	return c, ok
}

// eligible returns models that can fit estimatedTokens (0 = no filter).
func (r *Router) eligible(estimatedTokens int) []ModelConfig {
	out := make([]ModelConfig, 0, len(r.models))
	for _, m := range r.models {
		if estimatedTokens > 0 && m.MaxContext > 0 && m.MaxContext < estimatedTokens {
			continue
		}
		out = append(out, m)
	}
	return out
}

// fallbackChain starts at preferred and continues through remaining models
// by priority order (cheapest/highest priority first after the preferred).
func (r *Router) fallbackChain(preferred string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var chain []string
	seen := map[string]bool{}
	if preferred != "" {
		if _, ok := r.clients[preferred]; ok {
			chain = append(chain, preferred)
			seen[preferred] = true
		}
	}
	for _, m := range r.models {
		if seen[m.Name] {
			continue
		}
		chain = append(chain, m.Name)
		seen[m.Name] = true
	}
	return chain
}

func hasModel(models []ModelConfig, name string) bool {
	for _, m := range models {
		if m.Name == name {
			return true
		}
	}
	return false
}

func containsFold(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}

func filterByAnyCapability(models []ModelConfig, caps []string) []ModelConfig {
	var out []ModelConfig
	for _, m := range models {
		for _, c := range caps {
			if containsFold(m.Capabilities, c) {
				out = append(out, m)
				break
			}
		}
	}
	return out
}

func firstWithCapability(models []ModelConfig, cap string) string {
	for _, m := range models {
		if containsFold(m.Capabilities, cap) {
			return m.Name
		}
	}
	return ""
}

func firstWithAllCapabilities(models []ModelConfig, caps ...string) string {
	for _, m := range models {
		ok := true
		for _, c := range caps {
			if !containsFold(m.Capabilities, c) {
				ok = false
				break
			}
		}
		if ok {
			return m.Name
		}
	}
	return ""
}

func firstByCostTier(models []ModelConfig) string {
	if len(models) == 0 {
		return ""
	}
	best := models[0]
	for _, m := range models[1:] {
		if m.CostTier > 0 && (best.CostTier == 0 || m.CostTier < best.CostTier) {
			best = m
		}
	}
	return best.Name
}

func lastByCostTier(models []ModelConfig) string {
	if len(models) == 0 {
		return ""
	}
	best := models[0]
	for _, m := range models[1:] {
		if m.CostTier >= best.CostTier {
			best = m
		}
	}
	return best.Name
}

// pickByComplexity chooses among candidates: routine → lowest cost_tier,
// plan → median-ish (prefer "step-up" over pure flash), high → highest.
func pickByComplexity(models []ModelConfig, c Complexity) string {
	if len(models) == 0 {
		return ""
	}
	sorted := append([]ModelConfig(nil), models...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].CostTier < sorted[j].CostTier
	})
	switch c {
	case ComplexityHighStakes:
		return sorted[len(sorted)-1].Name
	case ComplexityPlan:
		// Prefer second-cheapest when available (Flash → Pro step-up).
		if len(sorted) >= 2 {
			return sorted[1].Name
		}
		return sorted[0].Name
	default:
		return sorted[0].Name
	}
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if api, ok := err.(*APIError); ok {
		return api.Retryable
	}
	// Network failures are retryable on another model.
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "connection") ||
		strings.Contains(msg, "temporary") ||
		strings.Contains(msg, "eof")
}

func isRateLimit(err error) bool {
	if api, ok := err.(*APIError); ok {
		return api.RateLimit
	}
	return strings.Contains(strings.ToLower(err.Error()), "rate limit")
}

func stripURLUserinfo(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return raw, err
	}
	u.User = nil
	return u.String(), nil
}
