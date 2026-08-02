package agent

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/iome-sh/iomesh-tui/internal/iomesh"
	"github.com/iome-sh/iomesh-tui/internal/router"
)

// MemoryConfig controls optional Palace auto-recall / auto-ingest.
// Does not embed Palace — recall prefers lean HTTP RetrieveMemory when mesh is
// enabled, then MCP tools on a connected server. Optional DualWrite also
// publishes async MEMORY_INGEST envelopes to the mesh.
type MemoryConfig struct {
	Enabled    bool
	Server     string // MCP server name (default "memory")
	Tenant     string
	AutoRecall bool
	AutoIngest bool
	// DualWrite publishes memory_ingest to mesh MEMORY_INGEST when mesh client is enabled (fail-open).
	DualWrite       bool
	Limit           int // retrieve limit (default 8)
	MaxSnippetBytes int // cap injected context (default 6000)
	// SessionID overrides Runtime.sessionID when non-empty.
	SessionID string
	// RecallSince / RecallUntil optional RFC3339 time-window filters for auto-recall
	// and default /memory recall (s1068 temporal retrieve options). Empty = no bound.
	RecallSince string
	RecallUntil string
	// RecallSessionSeq optional session_seq lower-bound filter for temporal recall; 0 omits.
	RecallSessionSeq int
	// RecallCacheTTLMS short-TTL client-side sync RetrieveMemory reuse (s1069).
	// Default 3000; 0 disables. Fail-open process-local only — not product Memory GA.
	// Key includes tenant + session + query + limit + since/until.
	RecallCacheTTLMS int
}

// DefaultMemoryConfig returns fail-open off defaults.
// s768: dual_write default OFF (local-primary honesty — optional mesh audit only).
func DefaultMemoryConfig() MemoryConfig {
	return MemoryConfig{
		Enabled:         false,
		Server:          "memory",
		AutoRecall:      true,
		AutoIngest:      false,
		DualWrite:       false, // s768: dual_write default OFF (local-primary honesty)
		Limit:           8,
		MaxSnippetBytes:  6000,
		RecallCacheTTLMS: DefaultRecallCacheTTLMS,
	}
}

// AttachMemory configures optional memory hooks: sync HTTP retrieve and/or MCP
// server for recall/ingest, and/or DualWrite with mesh client for stream ingest.
// Auto-recall prefers sync HTTP RetrieveMemory when mesh is enabled, then MCP.
func (rt *Runtime) AttachMemory(cfg MemoryConfig) {
	if rt == nil {
		return
	}
	if cfg.Server == "" {
		cfg.Server = "memory"
	}
	if cfg.Limit <= 0 {
		cfg.Limit = 8
	}
	if cfg.MaxSnippetBytes <= 0 {
		cfg.MaxSnippetBytes = 6000
	}
	if cfg.RecallCacheTTLMS < 0 {
		cfg.RecallCacheTTLMS = DefaultRecallCacheTTLMS
	}
	rt.memory = cfg
	rt.initMemoryRecallCache()
	if cfg.Enabled {
		rt.appendSystemNote("memory", fmt.Sprintf(
			"Memory: server=%q tenant=%q auto_recall=%v auto_ingest=%v dual_write=%v recall_cache_ttl_ms=%d. Auto-recall prefers sync POST /v1/memory/retrieve when mesh enabled, else MCP mcp__%s__memory_*. Fail-open when unavailable.",
			cfg.Server, emptyDash(cfg.Tenant), cfg.AutoRecall, cfg.AutoIngest, cfg.DualWrite, cfg.RecallCacheTTLMS, cfg.Server,
		))
	}
}

// initMemoryRecallCache rebuilds the short-TTL sync retrieve cache from config (s1069).
func (rt *Runtime) initMemoryRecallCache() {
	if rt == nil {
		return
	}
	rt.memoryCache = newMemoryRecallCache(rt.memory.RecallCacheTTLMS)
}

// LastMemoryRetrieveMS returns latency of the most recent sync/MCP retrieve attempt in ms (s1069).
func (rt *Runtime) LastMemoryRetrieveMS() int {
	if rt == nil {
		return 0
	}
	return int(rt.lastMemoryRetrieveMS.Load())
}

// LastMemoryRetrieveCacheHit reports whether the last sync retrieve used the TTL cache (s1069).
func (rt *Runtime) LastMemoryRetrieveCacheHit() bool {
	if rt == nil {
		return false
	}
	return rt.lastMemoryRetrieveCacheHit.Load()
}

// Memory returns the memory hook config.
func (rt *Runtime) Memory() MemoryConfig {
	if rt == nil {
		return DefaultMemoryConfig()
	}
	return rt.memory
}

func emptyDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func (rt *Runtime) memorySessionID() string {
	if rt.memory.SessionID != "" {
		return rt.memory.SessionID
	}
	return rt.sessionID
}

// nextSessionSeq returns a monotonic session_seq for dual-write envelopes.
func (rt *Runtime) nextSessionSeq() int {
	if rt == nil {
		return 0
	}
	return int(rt.sessionSeq.Add(1))
}

// memoryTenant resolves tenant for MCP tools and mesh dual-write.
func (rt *Runtime) memoryTenant() string {
	if rt == nil {
		return ""
	}
	if t := strings.TrimSpace(rt.memory.Tenant); t != "" {
		return t
	}
	if rt.mesh != nil {
		return strings.TrimSpace(rt.mesh.Tenant())
	}
	return ""
}

// dualWriteReady reports whether mesh dual-write can run.
func (rt *Runtime) dualWriteReady() bool {
	return rt != nil && rt.memory.Enabled && rt.memory.DualWrite &&
		rt.mesh != nil && rt.mesh.Enabled()
}

// mcpMemoryReady reports whether MCP memory tools are available.
func (rt *Runtime) mcpMemoryReady() bool {
	return rt != nil && rt.memory.Enabled && rt.mcp != nil &&
		rt.mcp.ClientByName(rt.memory.Server) != nil
}

// syncMemoryReady reports whether lean HTTP sync retrieve can run.
// True when mesh is enabled or a dedicated memory sidecar URL is configured.
// Broker-only mesh URLs fail-open to MCP on 404 unless MemoryEndpoint points at the sidecar.
func (rt *Runtime) syncMemoryReady() bool {
	return rt != nil && rt.memory.Enabled && rt.mesh != nil && rt.mesh.SyncMemoryReady()
}

// MemoryStatusLine is a short operator-facing status (slash /memory).
func (rt *Runtime) MemoryStatusLine() string {
	if rt == nil {
		return "memory: no runtime"
	}
	cfg := rt.memory
	if !cfg.Enabled {
		return "memory: hooks disabled (set [memory] enabled=true + MCP server or mesh for sync retrieve)"
	}
	connected := false
	if rt.mcp != nil {
		connected = rt.mcp.ClientByName(cfg.Server) != nil
	}
	return fmt.Sprintf("memory: enabled server=%q mcp=%v sync_http=%v tenant=%q auto_recall=%v auto_ingest=%v dual_write=%v session=%q",
		cfg.Server, connected, rt.syncMemoryReady(), emptyDash(cfg.Tenant), cfg.AutoRecall, cfg.AutoIngest, cfg.DualWrite, emptyDash(rt.memorySessionID()))
}

// MemoryRecallOpts overrides config temporal filters for one recall call.
// Empty/zero fields fall back to MemoryConfig.RecallSince/Until/SessionSeq.
type MemoryRecallOpts struct {
	Since      string
	Until      string
	SessionSeq int
	// SessionSeqSet is true when SessionSeq was explicitly provided (including 0 to clear).
	// When false, config RecallSessionSeq is used.
	SessionSeqSet bool
}

// MemoryRecall retrieves context for injection or /memory recall.
// Prefers sync HTTP RetrieveMemory when mesh is enabled (Phase 3+); falls back to MCP
// memory_retrieve when sync fails or mesh is unavailable.
// Uses MemoryConfig temporal filters (RecallSince/Until/SessionSeq) when set (s1068).
func (rt *Runtime) MemoryRecall(ctx context.Context, query string) (string, error) {
	return rt.MemoryRecallWithOpts(ctx, query, MemoryRecallOpts{})
}

// MemoryRecallWithOpts is MemoryRecall with optional per-call temporal overrides
// (slash /memory recall --since/--until/--session-seq).
func (rt *Runtime) MemoryRecallWithOpts(ctx context.Context, query string, opts MemoryRecallOpts) (string, error) {
	if rt == nil || !rt.memory.Enabled {
		return "", fmt.Errorf("memory hooks disabled")
	}
	q := strings.TrimSpace(query)
	if q == "" {
		q = "*"
	}
	limit := rt.memory.Limit
	if limit <= 0 {
		limit = 8
	}
	maxBytes := rt.memory.MaxSnippetBytes
	if maxBytes <= 0 {
		maxBytes = 6000
	}

	since := strings.TrimSpace(opts.Since)
	if since == "" {
		since = strings.TrimSpace(rt.memory.RecallSince)
	}
	until := strings.TrimSpace(opts.Until)
	if until == "" {
		until = strings.TrimSpace(rt.memory.RecallUntil)
	}
	sessionSeq := rt.memory.RecallSessionSeq
	if opts.SessionSeqSet {
		sessionSeq = opts.SessionSeq
	}

	// Prefer sync request/response against memory sidecar HTTP when mesh client is live.
	if rt.syncMemoryReady() {
		key := memoryRecallCacheKey{
			Tenant:  rt.memoryTenant(),
			Session: rt.memorySessionID(),
			Query:   q,
			Limit:   limit,
			Since:   since,
			Until:   until,
		}
		if hits, latMS, ok := rt.memoryCache.get(key); ok {
			rt.lastMemoryRetrieveMS.Store(int64(latMS))
			rt.lastMemoryRetrieveCacheHit.Store(true)
			if rt.logger != nil {
				rt.logger.Debug("memory sync retrieve cache hit", "tenant", key.Tenant, "query", q, "orig_ms", latMS)
			}
			return formatMemoryHits(hits, maxBytes), nil
		}

		start := time.Now()
		res, err := rt.mesh.RetrieveMemoryWithOptions(ctx, rt.memoryTenant(), iomesh.MemoryRetrieveOptions{
			Query:      q,
			Limit:      limit,
			SessionID:  rt.memorySessionID(),
			SessionSeq: sessionSeq,
			Since:      since,
			Until:      until,
		})
		latMS := int(time.Since(start).Milliseconds())
		if err == nil {
			rt.lastMemoryRetrieveMS.Store(int64(latMS))
			rt.lastMemoryRetrieveCacheHit.Store(false)
			hits := res.Memories
			if hits == nil {
				hits = []iomesh.MemoryHit{}
			}
			rt.memoryCache.put(key, hits, latMS)
			return formatMemoryHits(hits, maxBytes), nil
		}
		rt.lastMemoryRetrieveMS.Store(int64(latMS))
		rt.lastMemoryRetrieveCacheHit.Store(false)
		if rt.logger != nil {
			rt.logger.Debug("memory sync retrieve failed; trying MCP fallback", "err", err, "ms", latMS)
		}
		// Fall through to MCP when sidecar path is missing (e.g. broker-only endpoint).
	}

	if !rt.mcpMemoryReady() {
		if rt.syncMemoryReady() {
			return "", fmt.Errorf("memory sync retrieve failed and mcp server %q not connected", rt.memory.Server)
		}
		return "", fmt.Errorf("mcp server %q not connected (and mesh sync unavailable)", rt.memory.Server)
	}
	c := rt.mcp.ClientByName(rt.memory.Server)
	args := map[string]any{
		"query": q,
		"limit": limit,
	}
	if t := rt.memoryTenant(); t != "" {
		args["tenant"] = t
	}
	if sid := rt.memorySessionID(); sid != "" {
		args["session_id"] = sid
	}
	if sessionSeq != 0 {
		args["session_seq"] = sessionSeq
	}
	if since != "" {
		args["since"] = since
	}
	if until != "" {
		args["until"] = until
	}
	start := time.Now()
	out, err := c.CallTool(ctx, "memory_retrieve", args)
	latMS := int(time.Since(start).Milliseconds())
	rt.lastMemoryRetrieveMS.Store(int64(latMS))
	rt.lastMemoryRetrieveCacheHit.Store(false)
	if err != nil {
		return "", err
	}
	return truncateBytes(out, maxBytes), nil
}

// formatMemoryHits turns sync RetrieveMemory hits into a compact recall snippet.
// When maxBytes > 0, stops once the budget is reached without formatting remaining hits (s1069).
func formatMemoryHits(hits []iomesh.MemoryHit, maxBytes int) string {
	if len(hits) == 0 {
		return ""
	}
	var b strings.Builder
	n := 0
	for _, h := range hits {
		text := strings.TrimSpace(h.Summary)
		if text == "" {
			text = strings.TrimSpace(h.Full)
		}
		if text == "" {
			continue
		}
		var piece string
		if h.Score > 0 {
			piece = fmt.Sprintf("[%.2f] %s", h.Score, text)
		} else {
			piece = text
		}
		sep := ""
		if n > 0 {
			sep = "\n---\n"
		}
		if maxBytes > 0 && b.Len()+len(sep)+len(piece) > maxBytes {
			if b.Len() == 0 {
				return truncateBytes(piece, maxBytes)
			}
			break
		}
		b.WriteString(sep)
		b.WriteString(piece)
		n++
	}
	return b.String()
}

// MemoryIngestTurn persists a turn via MCP (when connected) and/or dual-write
// MEMORY_INGEST (when DualWrite + mesh enabled). Dual-write is independent and fail-open
// relative to MCP; at least one path must succeed for a nil error.
func (rt *Runtime) MemoryIngestTurn(ctx context.Context, role, content string) (string, error) {
	if rt == nil || !rt.memory.Enabled {
		return "", fmt.Errorf("memory hooks disabled")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return "", fmt.Errorf("empty content")
	}

	mcpReady := rt.mcpMemoryReady()
	dualReady := rt.dualWriteReady()
	if !mcpReady && !dualReady {
		return "", fmt.Errorf("mcp server %q not connected (and dual_write unavailable)", rt.memory.Server)
	}

	eventTime := time.Now().UTC().Format(time.RFC3339)
	var parts []string
	ok := false

	if mcpReady {
		c := rt.mcp.ClientByName(rt.memory.Server)
		args := map[string]any{
			"role":       role,
			"content":    content,
			"event_time": eventTime,
		}
		if t := rt.memoryTenant(); t != "" {
			args["tenant"] = t
		}
		if sid := rt.memorySessionID(); sid != "" {
			args["session_id"] = sid
		}
		out, err := c.CallTool(ctx, "memory_ingest_turn", args)
		if err != nil {
			if rt.logger != nil {
				rt.logger.Debug("memory MCP ingest", "err", err)
			}
			parts = append(parts, "mcp failed: "+err.Error())
		} else {
			ok = true
			if s := strings.TrimSpace(out); s != "" {
				parts = append(parts, s)
			} else {
				parts = append(parts, "mcp ingest ok")
			}
		}
	}

	// Dual-write independently (even when MCP failed or is absent).
	if dualReady {
		if err := rt.publishMemoryDualWrite(ctx, role, content, eventTime); err != nil {
			if rt.logger != nil {
				rt.logger.Debug("memory dual_write", "err", err)
			}
			parts = append(parts, "dual_write failed: "+err.Error())
		} else {
			ok = true
			parts = append(parts, "dual_write MEMORY_INGEST ok")
		}
	}

	msg := strings.Join(parts, "; ")
	if ok {
		return msg, nil
	}
	if msg == "" {
		msg = "memory ingest failed"
	}
	return msg, fmt.Errorf("%s", msg)
}

// publishMemoryDualWrite emits one MEMORY_INGEST envelope (caller fail-open).
func (rt *Runtime) publishMemoryDualWrite(ctx context.Context, role, content, eventTime string) error {
	tenant := rt.memoryTenant()
	if tenant == "" {
		return fmt.Errorf("tenant required for dual_write")
	}
	seq := rt.nextSessionSeq()
	env := iomesh.MemoryEnvelope{
		Type:       "memory_ingest",
		SessionID:  rt.memorySessionID(),
		Role:       role,
		Content:    content,
		EventTime:  eventTime,
		SessionSeq: seq,
	}
	_, err := rt.mesh.PublishMemoryIngest(ctx, tenant, env)
	return err
}

// maybeInjectMemoryRecall appends a system message when auto_recall is on (fail-open).
// Uses sync HTTP retrieve when mesh is enabled, else MCP (see MemoryRecall).
func (rt *Runtime) maybeInjectMemoryRecall(ctx context.Context, userText string, onEvent func(Event)) {
	if rt == nil || !rt.memory.Enabled || !rt.memory.AutoRecall {
		return
	}
	if !rt.syncMemoryReady() && !rt.mcpMemoryReady() {
		return
	}
	out, err := rt.MemoryRecall(ctx, userText)
	if err != nil || strings.TrimSpace(out) == "" {
		if err != nil && rt.logger != nil {
			rt.logger.Debug("memory auto_recall skipped", "err", err)
		}
		return
	}
	snippet := truncateBytes(out, rt.memory.MaxSnippetBytes)
	if snippet == "" {
		return
	}
	rt.messages = append(rt.messages, routerMessageSystem("<memory-context>\n"+snippet+"\n</memory-context>"))
	ms := rt.LastMemoryRetrieveMS()
	if ms < 0 {
		ms = 0
	}
	latNote := fmt.Sprintf("%dms", ms)
	if rt.LastMemoryRetrieveCacheHit() {
		latNote = fmt.Sprintf("%dms cache", ms)
	}
	onEvent(Event{
		Type:     EventMemoryRecall,
		Text:     fmt.Sprintf("injected memory recall (%d bytes, %s)", len(snippet), latNote),
		Duration: time.Duration(ms) * time.Millisecond,
	})
}

// maybeAutoIngest writes user + assistant turns after a successful final answer (fail-open).
// MCP ingest runs first when connected; dual-write runs independently when DualWrite is set.
func (rt *Runtime) maybeAutoIngest(ctx context.Context, userText, assistantText string, onEvent func(Event)) {
	if rt == nil || !rt.memory.Enabled || !rt.memory.AutoIngest {
		return
	}
	// Need at least one path: MCP or dual-write.
	if !rt.mcpMemoryReady() && !rt.dualWriteReady() {
		return
	}
	if strings.TrimSpace(userText) != "" {
		rt.autoIngestOne(ctx, "user", userText, onEvent)
	}
	if strings.TrimSpace(assistantText) != "" {
		rt.autoIngestOne(ctx, "assistant", assistantText, onEvent)
	}
}

func (rt *Runtime) autoIngestOne(ctx context.Context, role, content string, onEvent func(Event)) {
	eventTime := time.Now().UTC().Format(time.RFC3339)
	anyOK := false

	// MCP path first (existing).
	if rt.mcpMemoryReady() {
		c := rt.mcp.ClientByName(rt.memory.Server)
		args := map[string]any{
			"role":       role,
			"content":    content,
			"event_time": eventTime,
		}
		if t := rt.memoryTenant(); t != "" {
			args["tenant"] = t
		}
		if sid := rt.memorySessionID(); sid != "" {
			args["session_id"] = sid
		}
		if _, err := c.CallTool(ctx, "memory_ingest_turn", args); err != nil {
			if rt.logger != nil {
				rt.logger.Debug("memory auto_ingest "+role, "err", err)
			}
			onEvent(Event{Type: EventMemoryIngest, Text: "auto_ingest " + role + " failed: " + err.Error()})
		} else {
			anyOK = true
			onEvent(Event{Type: EventMemoryIngest, Text: "auto_ingest " + role + " turn"})
		}
	}

	// Dual-write independently (fail-open), even when MCP is absent or failed.
	if rt.dualWriteReady() {
		if err := rt.publishMemoryDualWrite(ctx, role, content, eventTime); err != nil {
			if rt.logger != nil {
				rt.logger.Debug("memory dual_write "+role, "err", err)
			}
			onEvent(Event{Type: EventMemoryIngest, Text: "dual_write " + role + " failed: " + err.Error()})
		} else {
			anyOK = true
			onEvent(Event{Type: EventMemoryIngest, Text: "dual_write " + role + " MEMORY_INGEST"})
		}
	}

	_ = anyOK
}

func routerMessageSystem(content string) router.Message {
	return router.Message{Role: "system", Content: content}
}

func truncateBytes(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	// Avoid splitting mid-rune.
	if max < 4 {
		max = 4
	}
	cut := max - 3
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	if cut <= 0 {
		return s[:max]
	}
	return s[:cut] + "..."
}
