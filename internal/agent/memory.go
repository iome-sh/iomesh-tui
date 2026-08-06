package agent

import (
	"context"
	"encoding/json"
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
	// RelatedMaxHops default BFS hops for opt-in multi-hop related recall (s1135).
	// Default 2. Used by MemoryRelated / /memory related only — does NOT change
	// default auto-recall (still single-hop RetrieveMemory / memory_retrieve).
	// 0 disables the related path when callers gate on it; MemoryRelated still
	// defaults hops to 2 when MaxHops is unset for operator convenience.
	RelatedMaxHops int
}

// DefaultMemoryConfig returns fail-open off defaults.
// s768: dual_write default OFF (local-primary honesty — optional mesh audit only).
func DefaultMemoryConfig() MemoryConfig {
	return MemoryConfig{
		Enabled:          false,
		Server:           "memory",
		AutoRecall:       true,
		AutoIngest:       false,
		DualWrite:        false, // s768: dual_write default OFF (local-primary honesty)
		Limit:            8,
		MaxSnippetBytes:  6000,
		RecallCacheTTLMS: DefaultRecallCacheTTLMS,
		RelatedMaxHops:   2, // s1135 multi-hop lite opt-in default (not auto-recall)
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
	if cfg.RelatedMaxHops < 0 {
		cfg.RelatedMaxHops = 2
	}
	rt.memory = cfg
	rt.initMemoryRecallCache()
	if cfg.Enabled {
		rt.appendSystemNote("memory", fmt.Sprintf(
			"Memory: server=%q tenant=%q auto_recall=%v auto_ingest=%v dual_write=%v recall_cache_ttl_ms=%d related_max_hops=%d. Auto-recall prefers sync POST /v1/memory/retrieve when mesh enabled, else MCP mcp__%s__memory_* (single-hop; multi-hop related is opt-in via /memory related). Fail-open when unavailable.",
			cfg.Server, emptyDash(cfg.Tenant), cfg.AutoRecall, cfg.AutoIngest, cfg.DualWrite, cfg.RecallCacheTTLMS, cfg.RelatedMaxHops, cfg.Server,
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

// MemoryRelatedOpts overrides config for one opt-in multi-hop related call (s1135).
// Zero MaxHops falls back to MemoryConfig.RelatedMaxHops (default 2); when that is
// also 0, hops default to 2 for operator convenience. Zero Limit uses config Limit.
// PreferShorterHops: omit/nil = kernel default true (s1067/s1277); false = legacy seed-first.
// Multi-hop lite ≠ full graph RAG · not Memory GA · dual_write OFF · hop ranking path-aware lite.
type MemoryRelatedOpts struct {
	MaxHops           int
	Limit             int
	PreferShorterHops *bool // nil omits field (platform default true); false = legacy seed-first
}

// MemoryOpsDigestOpts overrides defaults for one opt-in ops digest export (s1200).
// Zero/empty fields use defaults: Window=day, Horizon=ops, Limit=20.
type MemoryOpsDigestOpts struct {
	Window  string // day|week
	Horizon string // ops|knowledge|analytical|all
	Limit   int
	AsOf    string // optional RFC3339
}

// MemoryOpsDigest exports an ops heartbeat digest pack (s1200).
// Prefers sync HTTP ExportOpsDigest (POST /v1|/v5/memory/ops_digest); falls back
// to MCP ops_digest_export when sync fails or mesh is unavailable.
// Returns human-readable text (patterns + receipts + honesty line).
// Does NOT run on default auto-recall (slash/CLI opt-in only).
// Honesty: ops GA-path · knowledge/analytical Beta · never invent GA · dual_write OFF ·
// not product Memory GA · not full graph RAG. Human owns irreversible decisions.
func (rt *Runtime) MemoryOpsDigest(ctx context.Context, opts ...MemoryOpsDigestOpts) (string, error) {
	if rt == nil || !rt.memory.Enabled {
		return "", fmt.Errorf("memory hooks disabled")
	}

	var call MemoryOpsDigestOpts
	if len(opts) > 0 {
		call = opts[0]
	}
	window := strings.ToLower(strings.TrimSpace(call.Window))
	if window == "" {
		window = "day"
	}
	horizon := strings.ToLower(strings.TrimSpace(call.Horizon))
	if horizon == "" {
		horizon = "ops"
	}
	limit := call.Limit
	if limit <= 0 {
		limit = 20
	}
	maxBytes := rt.memory.MaxSnippetBytes
	if maxBytes <= 0 {
		maxBytes = 6000
	}

	// Prefer sync ops digest against memory sidecar HTTP when mesh client is live.
	if rt.syncMemoryReady() {
		start := time.Now()
		res, err := rt.mesh.ExportOpsDigest(ctx, rt.memoryTenant(), iomesh.MemoryOpsDigestOptions{
			Window:  window,
			Horizon: horizon,
			Limit:   limit,
			AsOf:    strings.TrimSpace(call.AsOf),
		})
		latMS := int(time.Since(start).Milliseconds())
		rt.lastMemoryRetrieveMS.Store(int64(latMS))
		rt.lastMemoryRetrieveCacheHit.Store(false)
		if err == nil {
			return formatOpsDigest(res, maxBytes), nil
		}
		if rt.logger != nil {
			rt.logger.Debug("memory ops_digest sync failed; trying MCP fallback", "err", err, "ms", latMS)
		}
		// Fall through to MCP when sidecar path is missing (e.g. broker-only endpoint).
	}

	if !rt.mcpMemoryReady() {
		if rt.syncMemoryReady() {
			return "", fmt.Errorf("memory ops_digest sync failed and mcp server %q not connected", rt.memory.Server)
		}
		return "", fmt.Errorf("mcp server %q not connected (and mesh sync unavailable)", rt.memory.Server)
	}
	c := rt.mcp.ClientByName(rt.memory.Server)
	args := map[string]any{
		"window":  window,
		"horizon": horizon,
		"limit":   limit,
	}
	if t := rt.memoryTenant(); t != "" {
		args["tenant"] = t
	}
	if asOf := strings.TrimSpace(call.AsOf); asOf != "" {
		args["as_of"] = asOf
	}
	start := time.Now()
	out, err := c.CallTool(ctx, "ops_digest_export", args)
	latMS := int(time.Since(start).Milliseconds())
	rt.lastMemoryRetrieveMS.Store(int64(latMS))
	rt.lastMemoryRetrieveCacheHit.Store(false)
	if err != nil {
		return "", err
	}
	// MCP returns JSON text; try to re-format for operator readability, else pass through.
	if formatted := formatOpsDigestJSON(out, maxBytes); formatted != "" {
		return formatted, nil
	}
	return truncateBytes(out, maxBytes), nil
}

// formatOpsDigest turns a sync ExportOpsDigest result into a compact human-readable pack.
// Sections: header · patterns · receipts · honesty (residual-honest framing).
func formatOpsDigest(res *iomesh.MemoryOpsDigestResult, maxBytes int) string {
	if res == nil {
		return ""
	}
	var b strings.Builder
	window := res.Window
	if window == "" {
		window = "day"
	}
	horizon := res.Horizon
	if horizon == "" {
		horizon = "ops"
	}
	fmt.Fprintf(&b, "ops digest window=%s horizon=%s", window, horizon)
	if res.Since != "" || res.AsOf != "" {
		fmt.Fprintf(&b, " since=%s as_of=%s", emptyDash(res.Since), emptyDash(res.AsOf))
	}
	b.WriteByte('\n')

	if len(res.Patterns) == 0 {
		b.WriteString("patterns: (none)\n")
	} else {
		fmt.Fprintf(&b, "patterns (%d):\n", len(res.Patterns))
		for i, p := range res.Patterns {
			line := strings.TrimSpace(p.Summary)
			if line == "" {
				line = strings.TrimSpace(p.Subject)
			}
			if line == "" {
				line = strings.TrimSpace(p.Kind)
			}
			if line == "" {
				line = p.ID
			}
			prefix := ""
			if p.Score > 0 {
				prefix = fmt.Sprintf("[%.2f] ", p.Score)
			}
			kind := ""
			if k := strings.TrimSpace(p.Kind); k != "" {
				kind = k + " "
			}
			fmt.Fprintf(&b, "  %d. %s%s%s\n", i+1, prefix, kind, line)
		}
	}

	if len(res.Receipts) == 0 {
		b.WriteString("receipts: (none)\n")
	} else {
		fmt.Fprintf(&b, "receipts (%d):\n", len(res.Receipts))
		for i, r := range res.Receipts {
			sum := strings.TrimSpace(r.Summary)
			if sum == "" {
				sum = r.ID
			}
			when := strings.TrimSpace(r.EventTime)
			if when != "" {
				fmt.Fprintf(&b, "  %d. [%s] %s\n", i+1, when, sum)
			} else {
				fmt.Fprintf(&b, "  %d. %s\n", i+1, sum)
			}
		}
	}

	// Honesty line — residual framing pin (never invent GA).
	h := res.Honesty
	opsPulse := h.OpsPulse
	if opsPulse == "" {
		opsPulse = "ga_path"
	}
	know := h.Knowledge
	if know == "" {
		know = "beta"
	}
	anal := h.Analytical
	if anal == "" {
		anal = "beta"
	}
	dw := h.DualWriteDefault
	if dw == "" {
		dw = "off"
	}
	// never_invent_ga defaults true in residual framing when the field is absent from older payloads.
	neverInvent := h.NeverInventGA
	if !neverInvent && h.OpsPulse == "" && h.Knowledge == "" {
		neverInvent = true
	}
	fmt.Fprintf(&b, "honesty: ops=%s knowledge=%s analytical=%s never_invent_ga=%v dual_write=%s · not Memory GA · not full graph RAG",
		opsPulse, know, anal, neverInvent, dw)
	if note := strings.TrimSpace(h.Note); note != "" {
		fmt.Fprintf(&b, "\n  note: %s", note)
	}
	out := b.String()
	return truncateBytes(out, maxBytes)
}

// formatOpsDigestJSON attempts to parse MCP ops_digest_export JSON into the same
// human-readable layout as formatOpsDigest. Returns empty when parse fails.
func formatOpsDigestJSON(raw string, maxBytes int) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw[0] != '{' {
		return ""
	}
	var res iomesh.MemoryOpsDigestResult
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		return ""
	}
	return formatOpsDigest(&res, maxBytes)
}

// MemoryFactsAsOfOpts for opt-in bi-temporal lite validity listing (s1276 / aion Beta K4 lite).
// AsOf is required (RFC3339). Entity, Query, SessionID, Limit are optional.
// Does NOT run on default auto-recall (slash/CLI opt-in only).
//
// MCP-first: platform ships MCP tool memory_facts_as_of; there is no lean HTTP
// POST /v1|/v5/memory/facts_as_of route today — do not invent one.
// Honesty: bi-temporal lite · not full dual-clock Graphiti · not Memory GA · dual_write OFF.
type MemoryFactsAsOfOpts struct {
	AsOf      string // required RFC3339 validity instant
	Entity    string // optional entity filter
	Query     string // optional content substring filter
	SessionID string // optional; empty uses Runtime session
	Limit     int    // zero → config Limit (default 8)
}

// factsAsOfHonestyFooter is the residual-honest pin for facts-as-of output.
// Locked: bi-temporal lite · not full dual-clock Graphiti · not Memory GA · dual_write OFF.
const factsAsOfHonestyFooter = "honesty: bi-temporal lite · not full dual-clock Graphiti · not Memory GA · dual_write OFF"

// memoryFactsAsOfResult is the aion MCP memory_facts_as_of JSON wire shape.
type memoryFactsAsOfResult struct {
	AsOf  string             `json:"as_of"`
	Facts []iomesh.MemoryHit `json:"facts"`
}

// MemoryFactsAsOf lists palace entries valid at as_of (s1276 · aion Beta K4 lite).
// Prefers MCP tool memory_facts_as_of on the configured memory server (MCP-first;
// no lean HTTP facts_as_of route on platform today).
// Returns human-readable facts + honesty footer. Empty facts is honest empty
// (facts: (none)) — never invents memories. Offline / tool failure is residual-honest
// fail-open messaging (not silent empty-as-success).
// Opt-in only — does not run on auto-recall.
func (rt *Runtime) MemoryFactsAsOf(ctx context.Context, opts MemoryFactsAsOfOpts) (string, error) {
	if rt == nil || !rt.memory.Enabled {
		return "", fmt.Errorf("memory hooks disabled")
	}
	asOf := strings.TrimSpace(opts.AsOf)
	if asOf == "" {
		return "", fmt.Errorf("as_of required (RFC3339)")
	}
	// Soft-validate RFC3339 / RFC3339Nano; aion parseRFC3339Flexible is the authority at the tool.
	if _, err := time.Parse(time.RFC3339, asOf); err != nil {
		if _, err2 := time.Parse(time.RFC3339Nano, asOf); err2 != nil {
			return "", fmt.Errorf("as_of must be RFC3339: %w", err)
		}
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = rt.memory.Limit
	}
	if limit <= 0 {
		limit = 8
	}
	maxBytes := rt.memory.MaxSnippetBytes
	if maxBytes <= 0 {
		maxBytes = 6000
	}

	// MCP-first — no lean HTTP /memory/facts_as_of on platform (document, do not invent).
	if !rt.mcpMemoryReady() {
		return formatFactsAsOfOffline(rt.memory.Server, asOf), nil
	}
	c := rt.mcp.ClientByName(rt.memory.Server)
	args := map[string]any{
		"as_of": asOf,
		"limit": limit,
	}
	if t := rt.memoryTenant(); t != "" {
		args["tenant"] = t
	}
	if e := strings.TrimSpace(opts.Entity); e != "" {
		args["entity"] = e
	}
	if q := strings.TrimSpace(opts.Query); q != "" {
		args["query"] = q
	}
	sid := strings.TrimSpace(opts.SessionID)
	if sid == "" {
		sid = rt.memorySessionID()
	}
	if sid != "" {
		args["session_id"] = sid
	}
	start := time.Now()
	out, err := c.CallTool(ctx, "memory_facts_as_of", args)
	latMS := int(time.Since(start).Milliseconds())
	rt.lastMemoryRetrieveMS.Store(int64(latMS))
	rt.lastMemoryRetrieveCacheHit.Store(false)
	if err != nil {
		// Fail-open residual-honest call failure — do not invent facts.
		return formatFactsAsOfCallFailed(asOf, err), nil
	}
	if formatted := formatFactsAsOfJSON(out, maxBytes); formatted != "" {
		return formatted, nil
	}
	// Unknown payload — pass through with honesty footer (never invent structure).
	raw := strings.TrimSpace(out)
	if raw == "" {
		return formatFactsAsOf(asOf, nil, maxBytes), nil
	}
	return truncateBytes(raw+"\n"+factsAsOfHonestyFooter, maxBytes), nil
}

// formatFactsAsOfOffline is residual-honest fail-open when MCP memory server is unavailable.
// Explicitly not empty-facts success (empty ≠ invent memories).
func formatFactsAsOfOffline(server, asOf string) string {
	if server == "" {
		server = "memory"
	}
	return fmt.Sprintf(
		"facts-as-of as_of=%s\nstatus: unavailable · mcp server %q not connected · MCP-first (no lean HTTP /memory/facts_as_of)\n%s · fail-open (empty ≠ invent memories)",
		emptyDash(asOf), server, factsAsOfHonestyFooter,
	)
}

// formatFactsAsOfCallFailed is residual-honest fail-open when MCP tool call errors.
func formatFactsAsOfCallFailed(asOf string, err error) string {
	msg := "error"
	if err != nil {
		msg = err.Error()
	}
	return fmt.Sprintf(
		"facts-as-of as_of=%s\nstatus: unavailable · mcp call failed: %s\n%s · fail-open (empty ≠ invent memories)",
		emptyDash(asOf), msg, factsAsOfHonestyFooter,
	)
}

// formatFactsAsOf turns as_of + fact hits into a compact residual-honest listing.
// Empty facts → "facts: (none)" + honesty footer (never invent memories).
func formatFactsAsOf(asOf string, facts []iomesh.MemoryHit, maxBytes int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "facts-as-of as_of=%s\n", emptyDash(asOf))
	if len(facts) == 0 {
		b.WriteString("facts: (none)\n")
	} else {
		fmt.Fprintf(&b, "facts (%d):\n", len(facts))
		for i, f := range facts {
			text := strings.TrimSpace(f.Summary)
			if text == "" {
				text = strings.TrimSpace(f.Full)
			}
			if text == "" {
				text = strings.TrimSpace(f.ID)
			}
			if text == "" {
				text = "(empty)"
			}
			prefix := ""
			if f.Score > 0 {
				prefix = fmt.Sprintf("[%.2f] ", f.Score)
			}
			line := fmt.Sprintf("  %d. %s%s\n", i+1, prefix, text)
			if maxBytes > 0 && b.Len()+len(line)+len(factsAsOfHonestyFooter) > maxBytes {
				break
			}
			b.WriteString(line)
		}
	}
	b.WriteString(factsAsOfHonestyFooter)
	return truncateBytes(b.String(), maxBytes)
}

// formatFactsAsOfJSON parses MCP memory_facts_as_of JSON {as_of, facts:[...]} into
// the same human-readable layout as formatFactsAsOf. Returns empty when parse fails.
func formatFactsAsOfJSON(raw string, maxBytes int) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw[0] != '{' {
		return ""
	}
	var res memoryFactsAsOfResult
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		return ""
	}
	// Require as_of or facts key presence-ish: accept empty facts with as_of; if both
	// absent after unmarshal of unrelated {}, still format honestly with empty.
	asOf := strings.TrimSpace(res.AsOf)
	return formatFactsAsOf(asOf, res.Facts, maxBytes)
}

// MemorySupersedeOpts for opt-in HITL A3 lite entity supersession (s1282 / aion s640).
// Entity is required. AsOf is optional RFC3339 (empty → server default now).
// Confirm must be true to call MCP — accidental mutation is refused residual-honestly.
//
// MCP-first: platform ships MCP tool memory_supersede_entity; do not invent lean HTTP supersede.
// Honesty: A3 lite · closes valid_until · not NLP contradiction · not full dual-clock Graphiti ·
// not Memory GA · dual_write OFF · mutating (valid_until close).
type MemorySupersedeOpts struct {
	Entity  string
	AsOf    string // optional RFC3339; empty = server default now
	Confirm bool   // must be true to call MCP
}

// supersedeHonestyFooter is the residual-honest pin for supersede output (s1282).
// Locked: A3 lite · not NLP contradiction · not full dual-clock Graphiti · not Memory GA ·
// dual_write OFF · mutating (valid_until close).
const supersedeHonestyFooter = "honesty: A3 lite supersede · not NLP contradiction · not full dual-clock Graphiti · not Memory GA · dual_write OFF · mutating (valid_until close)"

// memorySupersedeResult is the aion MCP memory_supersede_entity JSON wire shape (s640).
// Output: { entity, as_of, superseded_count }.
type memorySupersedeResult struct {
	Entity          string `json:"entity"`
	AsOf            string `json:"as_of"`
	SupersededCount int    `json:"superseded_count"`
}

// MemorySupersede closes open validity windows for entity tags (s1282 · aion A3 lite / s640).
// Prefers MCP tool memory_supersede_entity on the configured memory server (MCP-first;
// no lean HTTP supersede invent). Mutating: sets valid_until=as_of for open facts tagged
// with the entity — NOT NLP contradiction detection; NOT full dual-clock Graphiti; NOT Memory GA.
//
// HITL gate: Confirm must be true or the call is refused residual-honestly (string, not error)
// without invoking MCP. Offline / tool failure is residual-honest fail-open messaging
// (never invents superseded_count as success). Opt-in only — never auto-mutate.
func (rt *Runtime) MemorySupersede(ctx context.Context, opts MemorySupersedeOpts) (string, error) {
	if rt == nil || !rt.memory.Enabled {
		return "", fmt.Errorf("memory hooks disabled")
	}
	entity := strings.TrimSpace(opts.Entity)
	if entity == "" {
		return "", fmt.Errorf("entity required")
	}
	asOf := strings.TrimSpace(opts.AsOf)
	if asOf != "" {
		// Soft-validate RFC3339 / RFC3339Nano; aion is the authority at the tool.
		if _, err := time.Parse(time.RFC3339, asOf); err != nil {
			if _, err2 := time.Parse(time.RFC3339Nano, asOf); err2 != nil {
				return "", fmt.Errorf("as_of must be RFC3339: %w", err)
			}
		}
	}

	// HITL residual-honest refusal — do NOT call MCP without explicit confirm.
	if !opts.Confirm {
		return formatSupersedeRefuseConfirm(entity, asOf), nil
	}

	// MCP-first — no lean HTTP supersede invent on platform (document, do not invent).
	if !rt.mcpMemoryReady() {
		return formatSupersedeOffline(rt.memory.Server, entity, asOf), nil
	}
	c := rt.mcp.ClientByName(rt.memory.Server)
	args := map[string]any{
		"entity": entity,
	}
	if t := rt.memoryTenant(); t != "" {
		args["tenant"] = t
	}
	if asOf != "" {
		args["as_of"] = asOf
	}
	start := time.Now()
	out, err := c.CallTool(ctx, "memory_supersede_entity", args)
	latMS := int(time.Since(start).Milliseconds())
	rt.lastMemoryRetrieveMS.Store(int64(latMS))
	rt.lastMemoryRetrieveCacheHit.Store(false)
	if err != nil {
		// Fail-open residual-honest call failure — do not invent superseded_count.
		return formatSupersedeCallFailed(entity, asOf, err), nil
	}
	if formatted := formatSupersedeJSON(out); formatted != "" {
		return formatted, nil
	}
	// Unknown payload — pass through with honesty footer (never invent count).
	raw := strings.TrimSpace(out)
	if raw == "" {
		return formatSupersedeEmptyPayload(entity, asOf), nil
	}
	return raw + "\n" + supersedeHonestyFooter, nil
}

// formatSupersedeRefuseConfirm is residual-honest HITL refusal when Confirm is false.
// Does not call MCP; does not invent supersede success.
func formatSupersedeRefuseConfirm(entity, asOf string) string {
	return fmt.Sprintf(
		"supersede entity=%s as_of=%s\nstatus: refused · HITL confirmation required\n  mutating A3 lite closes open valid_until windows — pass --i-confirm to proceed\n  (not accidental; not NLP contradiction)\n%s",
		emptyDash(entity), emptyDash(asOf), supersedeHonestyFooter,
	)
}

// formatSupersedeOffline is residual-honest fail-open when MCP memory server is unavailable.
// Explicitly not inventing superseded_count success.
func formatSupersedeOffline(server, entity, asOf string) string {
	if server == "" {
		server = "memory"
	}
	return fmt.Sprintf(
		"supersede entity=%s as_of=%s\nstatus: unavailable · mcp server %q not connected · MCP-first (no lean HTTP supersede invent)\n%s · fail-open (not inventing superseded_count)",
		emptyDash(entity), emptyDash(asOf), server, supersedeHonestyFooter,
	)
}

// formatSupersedeCallFailed is residual-honest fail-open when MCP tool call errors.
func formatSupersedeCallFailed(entity, asOf string, err error) string {
	msg := "error"
	if err != nil {
		msg = err.Error()
	}
	return fmt.Sprintf(
		"supersede entity=%s as_of=%s\nstatus: unavailable · mcp call failed: %s\n%s · fail-open (not inventing superseded_count)",
		emptyDash(entity), emptyDash(asOf), msg, supersedeHonestyFooter,
	)
}

// formatSupersedeEmptyPayload is residual-honest when MCP returns empty body after call.
// Never invents a superseded_count.
func formatSupersedeEmptyPayload(entity, asOf string) string {
	return fmt.Sprintf(
		"supersede entity=%s as_of=%s\nstatus: ok · empty payload (not inventing superseded_count)\n%s",
		emptyDash(entity), emptyDash(asOf), supersedeHonestyFooter,
	)
}

// formatSupersede turns entity + as_of + count into a compact residual-honest result.
// Call only when MCP returned a parseable count — never invent success counts offline.
func formatSupersede(entity, asOf string, count int) string {
	return fmt.Sprintf(
		"supersede entity=%s as_of=%s\nsuperseded_count: %d\n%s",
		emptyDash(entity), emptyDash(asOf), count, supersedeHonestyFooter,
	)
}

// formatSupersedeJSON parses MCP memory_supersede_entity JSON
// {entity, as_of, superseded_count} into the same human-readable layout as formatSupersede.
// Returns empty when parse fails (caller may pass through raw).
func formatSupersedeJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw[0] != '{' {
		return ""
	}
	var res memorySupersedeResult
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		return ""
	}
	// Accept zero count (honest empty supersede) when JSON parsed — that is real wire, not invent.
	entity := strings.TrimSpace(res.Entity)
	asOf := strings.TrimSpace(res.AsOf)
	return formatSupersede(entity, asOf, res.SupersededCount)
}

// MemoryPatternsOpts for opt-in ops-pulse pattern listing (s1287 / aion s138 T2 · s789 Beta).
// Limit optional (zero → config Limit, default 8). Does NOT run on default auto-recall.
//
// MCP-first: platform ships MCP tool memory_patterns_list; do not invent lean HTTP patterns routes.
// Honesty: ops pulse Beta · suggestive only · not medical diagnosis · not OTel host metrics ·
// not invent GA window engine · dual_write OFF · not Memory GA · book-demo OFF.
type MemoryPatternsOpts struct {
	Limit int
}

// MemoryAnomaliesOpts for opt-in ops-pulse anomaly listing (s1287 / aion s138 T2 · s789 Beta).
// Limit optional (zero → config Limit, default 8). Does NOT run on default auto-recall.
//
// MCP-first: platform ships MCP tool memory_anomalies_list; do not invent lean HTTP anomalies routes.
// Honesty: ops pulse Beta · suggestive only · not medical diagnosis · not OTel host metrics ·
// not invent GA window engine · dual_write OFF · not Memory GA · book-demo OFF.
type MemoryAnomaliesOpts struct {
	Limit int
}

// pulseHonestyFooter is the residual-honest pin for patterns/anomalies ops-pulse output (s1287).
// Locked: ops pulse Beta · suggestive only · not medical diagnosis · not OTel host metrics ·
// not invent GA window engine · dual_write OFF · not Memory GA.
// (s138 T2 · s789 Beta framing; offline analysis; empty ≠ invent patterns/anomalies.)
const pulseHonestyFooter = "honesty: ops pulse Beta · suggestive only · not medical diagnosis · not OTel host metrics · not invent GA window engine · dual_write OFF · not Memory GA"

// pulseSignal is a defensive wire shape for aion PatternSignal / AnomalySignal.
// Typical fields: subject, kind, count, score, summary/note, window — all optional for residual-honest parse.
type pulseSignal struct {
	ID      string  `json:"id"`
	Kind    string  `json:"kind"`
	Subject string  `json:"subject"`
	Count   int     `json:"count"`
	Score   float64 `json:"score"`
	Summary string  `json:"summary"`
	Note    string  `json:"note"` // alternate text field some emitters may use
	Window  string  `json:"window"`
}

type memoryPatternsResult struct {
	Patterns []pulseSignal `json:"patterns"`
}

type memoryAnomaliesResult struct {
	Anomalies []pulseSignal `json:"anomalies"`
}

// MemoryPatterns lists recurring subject/keyphrase ops-pulse signals (s1287 · aion s138 T2).
// Prefers MCP tool memory_patterns_list on the configured memory server (MCP-first;
// no lean HTTP patterns invent). Returns human-readable lines + honesty footer.
// Empty list is honest empty (patterns: (none)) — never invents signals.
// Offline / tool failure is residual-honest fail-open messaging. Opt-in only — not auto-recall.
// Suggestive ops pulse only: not medical diagnosis · not OTel host metrics · Beta offline analysis.
func (rt *Runtime) MemoryPatterns(ctx context.Context, opts MemoryPatternsOpts) (string, error) {
	if rt == nil || !rt.memory.Enabled {
		return "", fmt.Errorf("memory hooks disabled")
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = rt.memory.Limit
	}
	if limit <= 0 {
		limit = 8
	}
	maxBytes := rt.memory.MaxSnippetBytes
	if maxBytes <= 0 {
		maxBytes = 6000
	}

	// MCP-first — no lean HTTP /memory/patterns on platform (document, do not invent).
	if !rt.mcpMemoryReady() {
		return formatPatternsOffline(rt.memory.Server), nil
	}
	c := rt.mcp.ClientByName(rt.memory.Server)
	args := map[string]any{
		"limit": limit,
	}
	if t := rt.memoryTenant(); t != "" {
		args["tenant"] = t
	}
	start := time.Now()
	out, err := c.CallTool(ctx, "memory_patterns_list", args)
	latMS := int(time.Since(start).Milliseconds())
	rt.lastMemoryRetrieveMS.Store(int64(latMS))
	rt.lastMemoryRetrieveCacheHit.Store(false)
	if err != nil {
		// Fail-open residual-honest call failure — do not invent patterns.
		return formatPatternsCallFailed(err), nil
	}
	if formatted := formatPatternsJSON(out, maxBytes); formatted != "" {
		return formatted, nil
	}
	// Unknown payload — pass through with honesty footer (never invent structure).
	raw := strings.TrimSpace(out)
	if raw == "" {
		return formatPatterns(nil, maxBytes), nil
	}
	return truncateBytes(raw+"\n"+pulseHonestyFooter, maxBytes), nil
}

// MemoryAnomalies lists rate/burst ops-pulse signals (s1287 · aion s138 T2).
// Prefers MCP tool memory_anomalies_list on the configured memory server (MCP-first;
// no lean HTTP anomalies invent). Returns human-readable lines + honesty footer.
// Empty list is honest empty (anomalies: (none)) — never invents signals.
// Offline / tool failure is residual-honest fail-open messaging. Opt-in only — not auto-recall.
// Suggestive ops pulse only: not medical diagnosis · not OTel host metrics · Beta offline analysis.
func (rt *Runtime) MemoryAnomalies(ctx context.Context, opts MemoryAnomaliesOpts) (string, error) {
	if rt == nil || !rt.memory.Enabled {
		return "", fmt.Errorf("memory hooks disabled")
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = rt.memory.Limit
	}
	if limit <= 0 {
		limit = 8
	}
	maxBytes := rt.memory.MaxSnippetBytes
	if maxBytes <= 0 {
		maxBytes = 6000
	}

	// MCP-first — no lean HTTP /memory/anomalies on platform (document, do not invent).
	if !rt.mcpMemoryReady() {
		return formatAnomaliesOffline(rt.memory.Server), nil
	}
	c := rt.mcp.ClientByName(rt.memory.Server)
	args := map[string]any{
		"limit": limit,
	}
	if t := rt.memoryTenant(); t != "" {
		args["tenant"] = t
	}
	start := time.Now()
	out, err := c.CallTool(ctx, "memory_anomalies_list", args)
	latMS := int(time.Since(start).Milliseconds())
	rt.lastMemoryRetrieveMS.Store(int64(latMS))
	rt.lastMemoryRetrieveCacheHit.Store(false)
	if err != nil {
		// Fail-open residual-honest call failure — do not invent anomalies.
		return formatAnomaliesCallFailed(err), nil
	}
	if formatted := formatAnomaliesJSON(out, maxBytes); formatted != "" {
		return formatted, nil
	}
	// Unknown payload — pass through with honesty footer (never invent structure).
	raw := strings.TrimSpace(out)
	if raw == "" {
		return formatAnomalies(nil, maxBytes), nil
	}
	return truncateBytes(raw+"\n"+pulseHonestyFooter, maxBytes), nil
}

// formatPatternsOffline is residual-honest fail-open when MCP memory server is unavailable.
// Explicitly not empty-patterns success (empty ≠ invent patterns).
func formatPatternsOffline(server string) string {
	if server == "" {
		server = "memory"
	}
	return fmt.Sprintf(
		"patterns\nstatus: unavailable · mcp server %q not connected · MCP-first (no lean HTTP patterns invent)\n%s · fail-open (empty ≠ invent patterns)",
		server, pulseHonestyFooter,
	)
}

// formatAnomaliesOffline is residual-honest fail-open when MCP memory server is unavailable.
// Explicitly not empty-anomalies success (empty ≠ invent anomalies).
func formatAnomaliesOffline(server string) string {
	if server == "" {
		server = "memory"
	}
	return fmt.Sprintf(
		"anomalies\nstatus: unavailable · mcp server %q not connected · MCP-first (no lean HTTP anomalies invent)\n%s · fail-open (empty ≠ invent anomalies)",
		server, pulseHonestyFooter,
	)
}

// formatPatternsCallFailed is residual-honest fail-open when MCP tool call errors.
func formatPatternsCallFailed(err error) string {
	msg := "error"
	if err != nil {
		msg = err.Error()
	}
	return fmt.Sprintf(
		"patterns\nstatus: unavailable · mcp call failed: %s\n%s · fail-open (empty ≠ invent patterns)",
		msg, pulseHonestyFooter,
	)
}

// formatAnomaliesCallFailed is residual-honest fail-open when MCP tool call errors.
func formatAnomaliesCallFailed(err error) string {
	msg := "error"
	if err != nil {
		msg = err.Error()
	}
	return fmt.Sprintf(
		"anomalies\nstatus: unavailable · mcp call failed: %s\n%s · fail-open (empty ≠ invent anomalies)",
		msg, pulseHonestyFooter,
	)
}

// formatPulseSignalLine builds one compact human line from a defensive pulse signal.
// Prefers subject/kind/score/count/summary; falls back to id or "(empty)".
func formatPulseSignalLine(i int, s pulseSignal) string {
	text := strings.TrimSpace(s.Summary)
	if text == "" {
		text = strings.TrimSpace(s.Note)
	}
	var parts []string
	if subj := strings.TrimSpace(s.Subject); subj != "" {
		parts = append(parts, "subject="+subj)
	}
	if kind := strings.TrimSpace(s.Kind); kind != "" {
		parts = append(parts, "kind="+kind)
	}
	if s.Count > 0 {
		parts = append(parts, fmt.Sprintf("count=%d", s.Count))
	}
	if s.Score > 0 {
		parts = append(parts, fmt.Sprintf("score=%.2f", s.Score))
	}
	if win := strings.TrimSpace(s.Window); win != "" {
		parts = append(parts, "window="+win)
	}
	body := strings.Join(parts, " ")
	if body == "" {
		if id := strings.TrimSpace(s.ID); id != "" {
			body = id
		} else if text != "" {
			body = text
			text = ""
		} else {
			body = "(empty)"
		}
	}
	if text != "" {
		return fmt.Sprintf("  %d. %s — %s\n", i+1, body, text)
	}
	return fmt.Sprintf("  %d. %s\n", i+1, body)
}

// formatPatterns turns pattern signals into a compact residual-honest listing.
// Empty → "patterns: (none)" + honesty footer (never invent signals).
func formatPatterns(patterns []pulseSignal, maxBytes int) string {
	var b strings.Builder
	b.WriteString("patterns\n")
	if len(patterns) == 0 {
		b.WriteString("patterns: (none)\n")
	} else {
		fmt.Fprintf(&b, "patterns (%d):\n", len(patterns))
		for i, p := range patterns {
			line := formatPulseSignalLine(i, p)
			if maxBytes > 0 && b.Len()+len(line)+len(pulseHonestyFooter) > maxBytes {
				break
			}
			b.WriteString(line)
		}
	}
	b.WriteString(pulseHonestyFooter)
	return truncateBytes(b.String(), maxBytes)
}

// formatAnomalies turns anomaly signals into a compact residual-honest listing.
// Empty → "anomalies: (none)" + honesty footer (never invent signals).
func formatAnomalies(anomalies []pulseSignal, maxBytes int) string {
	var b strings.Builder
	b.WriteString("anomalies\n")
	if len(anomalies) == 0 {
		b.WriteString("anomalies: (none)\n")
	} else {
		fmt.Fprintf(&b, "anomalies (%d):\n", len(anomalies))
		for i, a := range anomalies {
			line := formatPulseSignalLine(i, a)
			if maxBytes > 0 && b.Len()+len(line)+len(pulseHonestyFooter) > maxBytes {
				break
			}
			b.WriteString(line)
		}
	}
	b.WriteString(pulseHonestyFooter)
	return truncateBytes(b.String(), maxBytes)
}

// formatPatternsJSON parses MCP memory_patterns_list JSON {patterns:[...]} into
// the same human-readable layout as formatPatterns. Returns empty when parse fails.
// Defensive: accepts partial signal objects (subject/kind/score optional).
func formatPatternsJSON(raw string, maxBytes int) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw[0] != '{' {
		return ""
	}
	var res memoryPatternsResult
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		return ""
	}
	// Require patterns key presence-ish: empty array is honest empty; unrelated {} also formats empty.
	return formatPatterns(res.Patterns, maxBytes)
}

// formatAnomaliesJSON parses MCP memory_anomalies_list JSON {anomalies:[...]} into
// the same human-readable layout as formatAnomalies. Returns empty when parse fails.
// Defensive: accepts partial signal objects (subject/kind/score optional).
func formatAnomaliesJSON(raw string, maxBytes int) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw[0] != '{' {
		return ""
	}
	var res memoryAnomaliesResult
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		return ""
	}
	return formatAnomalies(res.Anomalies, maxBytes)
}

// MemoryTimelineOpts for opt-in temporal timeline listing (s1296 / aion memory_timeline).
// Since, Until, Query, SessionID, Limit optional. Does NOT run on default auto-recall.
//
// MCP-first: platform ships MCP tool memory_timeline; there is no lean HTTP
// POST /v1|/v5/memory/timeline route today — do not invent one.
// Honesty: temporal timeline · filters before limit · not Memory GA · dual_write OFF.
// Mutating compact: use MemoryTriggerCompact HITL (s1311) — not auto from timeline.
type MemoryTimelineOpts struct {
	Since     string // optional RFC3339 inclusive lower bound
	Until     string // optional RFC3339 inclusive upper bound
	Query     string // optional content substring filter
	SessionID string // optional; empty uses Runtime session when set
	Limit     int    // zero → config Limit (default 8)
}

// MemoryCompactStatusOpts for opt-in Palace tier counts / last compaction (s1296).
// Tenant is taken from Runtime memory config — no caller override today.
// Read-only; does NOT run on default auto-recall.
//
// MCP-first: platform ships MCP tool memory_compact_status; do not invent lean HTTP.
// Honesty: Palace tier counts residual · not Memory GA · not auto-compact product · dual_write OFF.
// Mutating compact: use MemoryTriggerCompact HITL (s1311) — not auto from compact-status.
type MemoryCompactStatusOpts struct{}

// timelineHonestyFooter is the residual-honest pin for timeline output (s1296).
// Locked: temporal timeline · filters before limit · not Memory GA · dual_write OFF · MCP-first.
const timelineHonestyFooter = "honesty: temporal timeline · filters before limit · not Memory GA · dual_write OFF · MCP-first (no lean HTTP timeline invent)"

// compactStatusHonestyFooter is the residual-honest pin for compact-status output (s1296).
// Locked: Palace tier counts residual · not Memory GA · not auto-compact product · dual_write OFF.
const compactStatusHonestyFooter = "honesty: Palace tier counts residual · not Memory GA · not auto-compact product · dual_write OFF · MCP-first (no lean HTTP invent)"

// timelineEntry is a defensive wire shape for aion memory_timeline entries (memoryHit-like).
// Accepts id/summary/full/score + timestamp or event_time when present.
type timelineEntry struct {
	ID        string  `json:"id"`
	Summary   string  `json:"summary"`
	Full      string  `json:"full"`
	Score     float64 `json:"score"`
	Timestamp string  `json:"timestamp"`
	EventTime string  `json:"event_time"`
}

type memoryTimelineResult struct {
	Entries []timelineEntry `json:"entries"`
}

// memoryCompactStatusResult is the aion MCP memory_compact_status JSON wire shape.
// stats may be PascalCase (palace.MemoryStats has no json tags) or snake_case —
// parse defensively via raw map + helper fields.
type memoryCompactStatusResult struct {
	Stats          json.RawMessage `json:"stats"`
	LastCompaction string          `json:"last_compaction"`
	// Flat alternate keys some emitters may use (resource URI shape).
	WorkingCount    *int `json:"working_count"`
	ContextualCount *int `json:"contextual_count"`
	ArchivalCount   *int `json:"archival_count"`
	SemanticCount   *int `json:"semantic_count"`
	TotalEntries    *int `json:"total_entries"`
}

// MemoryTimeline lists palace entries ordered by event time (s1296 · aion memory_timeline).
// Prefers MCP tool memory_timeline on the configured memory server (MCP-first;
// no lean HTTP timeline route on platform today). Returns human-readable entries + honesty footer.
// Empty list is honest empty (entries: (none)) — never invents memories.
// Offline / tool failure is residual-honest fail-open messaging. Opt-in only — not auto-recall.
// Filters (since/until/query) are applied before limit on the server side.
func (rt *Runtime) MemoryTimeline(ctx context.Context, opts MemoryTimelineOpts) (string, error) {
	if rt == nil || !rt.memory.Enabled {
		return "", fmt.Errorf("memory hooks disabled")
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = rt.memory.Limit
	}
	if limit <= 0 {
		limit = 8
	}
	maxBytes := rt.memory.MaxSnippetBytes
	if maxBytes <= 0 {
		maxBytes = 6000
	}

	// MCP-first — no lean HTTP /memory/timeline on platform (document, do not invent).
	if !rt.mcpMemoryReady() {
		return formatTimelineOffline(rt.memory.Server), nil
	}
	c := rt.mcp.ClientByName(rt.memory.Server)
	args := map[string]any{
		"limit": limit,
	}
	if t := rt.memoryTenant(); t != "" {
		args["tenant"] = t
	}
	if s := strings.TrimSpace(opts.Since); s != "" {
		args["since"] = s
	}
	if u := strings.TrimSpace(opts.Until); u != "" {
		args["until"] = u
	}
	if q := strings.TrimSpace(opts.Query); q != "" {
		args["query"] = q
	}
	sid := strings.TrimSpace(opts.SessionID)
	if sid == "" {
		sid = rt.memorySessionID()
	}
	if sid != "" {
		// Pass-through when set; aion timeline may ignore unknown session_id today.
		args["session_id"] = sid
	}
	start := time.Now()
	out, err := c.CallTool(ctx, "memory_timeline", args)
	latMS := int(time.Since(start).Milliseconds())
	rt.lastMemoryRetrieveMS.Store(int64(latMS))
	rt.lastMemoryRetrieveCacheHit.Store(false)
	if err != nil {
		// Fail-open residual-honest call failure — do not invent entries.
		return formatTimelineCallFailed(err), nil
	}
	if formatted := formatTimelineJSON(out, maxBytes); formatted != "" {
		return formatted, nil
	}
	// Unknown payload — pass through with honesty footer (never invent structure).
	raw := strings.TrimSpace(out)
	if raw == "" {
		return formatTimeline(nil, maxBytes), nil
	}
	return truncateBytes(raw+"\n"+timelineHonestyFooter, maxBytes), nil
}

// MemoryCompactStatus returns Palace tier counts + last_compaction (s1296 · aion memory_compact_status).
// Prefers MCP tool memory_compact_status on the configured memory server (MCP-first;
// no lean HTTP invent). Read-only residual — not auto-compact product · not Memory GA.
// Offline / tool failure is residual-honest fail-open messaging. Opt-in only — not auto-recall.
// Does NOT call memory_trigger_compact (mutating RecMem advisory; use MemoryTriggerCompact HITL s1311).
func (rt *Runtime) MemoryCompactStatus(ctx context.Context, _ MemoryCompactStatusOpts) (string, error) {
	if rt == nil || !rt.memory.Enabled {
		return "", fmt.Errorf("memory hooks disabled")
	}
	maxBytes := rt.memory.MaxSnippetBytes
	if maxBytes <= 0 {
		maxBytes = 6000
	}

	// MCP-first — no lean HTTP /memory/compact_status invent.
	if !rt.mcpMemoryReady() {
		return formatCompactStatusOffline(rt.memory.Server), nil
	}
	c := rt.mcp.ClientByName(rt.memory.Server)
	args := map[string]any{}
	if t := rt.memoryTenant(); t != "" {
		args["tenant"] = t
	}
	start := time.Now()
	out, err := c.CallTool(ctx, "memory_compact_status", args)
	latMS := int(time.Since(start).Milliseconds())
	rt.lastMemoryRetrieveMS.Store(int64(latMS))
	rt.lastMemoryRetrieveCacheHit.Store(false)
	if err != nil {
		// Fail-open residual-honest call failure — do not invent tier counts.
		return formatCompactStatusCallFailed(err), nil
	}
	if formatted := formatCompactStatusJSON(out, maxBytes); formatted != "" {
		return formatted, nil
	}
	// Unknown payload — pass through with honesty footer (never invent structure).
	raw := strings.TrimSpace(out)
	if raw == "" {
		return formatCompactStatus(compactStatusView{}, maxBytes), nil
	}
	return truncateBytes(raw+"\n"+compactStatusHonestyFooter, maxBytes), nil
}

// formatTimelineOffline is residual-honest fail-open when MCP memory server is unavailable.
// Explicitly not empty-entries success (empty ≠ invent memories).
func formatTimelineOffline(server string) string {
	if server == "" {
		server = "memory"
	}
	return fmt.Sprintf(
		"timeline\nstatus: unavailable · mcp server %q not connected · MCP-first (no lean HTTP timeline invent)\n%s · fail-open (empty ≠ invent memories)",
		server, timelineHonestyFooter,
	)
}

// formatTimelineCallFailed is residual-honest fail-open when MCP tool call errors.
func formatTimelineCallFailed(err error) string {
	msg := "error"
	if err != nil {
		msg = err.Error()
	}
	return fmt.Sprintf(
		"timeline\nstatus: unavailable · mcp call failed: %s\n%s · fail-open (empty ≠ invent memories)",
		msg, timelineHonestyFooter,
	)
}

// formatTimeline turns timeline entries into a compact residual-honest listing.
// Empty → "entries: (none)" + honesty footer (never invent memories).
// Lines prefer summary/full/id; event_time or timestamp when present.
func formatTimeline(entries []timelineEntry, maxBytes int) string {
	var b strings.Builder
	b.WriteString("timeline\n")
	if len(entries) == 0 {
		b.WriteString("entries: (none)\n")
	} else {
		fmt.Fprintf(&b, "entries (%d):\n", len(entries))
		for i, e := range entries {
			text := strings.TrimSpace(e.Summary)
			if text == "" {
				text = strings.TrimSpace(e.Full)
			}
			if text == "" {
				text = strings.TrimSpace(e.ID)
			}
			if text == "" {
				text = "(empty)"
			}
			when := strings.TrimSpace(e.EventTime)
			if when == "" {
				when = strings.TrimSpace(e.Timestamp)
			}
			prefix := ""
			if e.Score > 0 {
				prefix = fmt.Sprintf("[%.2f] ", e.Score)
			}
			var line string
			if when != "" {
				line = fmt.Sprintf("  %d. %s[%s] %s\n", i+1, prefix, when, text)
			} else {
				line = fmt.Sprintf("  %d. %s%s\n", i+1, prefix, text)
			}
			if maxBytes > 0 && b.Len()+len(line)+len(timelineHonestyFooter) > maxBytes {
				break
			}
			b.WriteString(line)
		}
	}
	b.WriteString(timelineHonestyFooter)
	return truncateBytes(b.String(), maxBytes)
}

// formatTimelineJSON parses MCP memory_timeline JSON {entries:[...]} into
// the same human-readable layout as formatTimeline. Returns empty when parse fails.
func formatTimelineJSON(raw string, maxBytes int) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw[0] != '{' {
		return ""
	}
	var res memoryTimelineResult
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		return ""
	}
	// Empty array is honest empty; unrelated {} also formats empty entries.
	return formatTimeline(res.Entries, maxBytes)
}

// compactStatusView is a residual-honest view of tier counts + last compaction.
type compactStatusView struct {
	WorkingCount    int
	ContextualCount int
	ArchivalCount   int
	SemanticCount   int
	TotalEntries    int
	LastCompaction  string
	HasWorking      bool
	HasContextual   bool
	HasArchival     bool
	HasSemantic     bool
	HasTotal        bool
}

// formatCompactStatusOffline is residual-honest fail-open when MCP memory server is unavailable.
// Explicitly not inventing green compaction / zero tier success.
func formatCompactStatusOffline(server string) string {
	if server == "" {
		server = "memory"
	}
	return fmt.Sprintf(
		"compact-status\nstatus: unavailable · mcp server %q not connected · MCP-first (no lean HTTP invent)\n%s · fail-open (empty ≠ invent compaction green)",
		server, compactStatusHonestyFooter,
	)
}

// formatCompactStatusCallFailed is residual-honest fail-open when MCP tool call errors.
func formatCompactStatusCallFailed(err error) string {
	msg := "error"
	if err != nil {
		msg = err.Error()
	}
	return fmt.Sprintf(
		"compact-status\nstatus: unavailable · mcp call failed: %s\n%s · fail-open (empty ≠ invent compaction green)",
		msg, compactStatusHonestyFooter,
	)
}

// formatCompactStatus turns tier counts + last_compaction into residual-honest lines.
func formatCompactStatus(v compactStatusView, maxBytes int) string {
	var b strings.Builder
	b.WriteString("compact-status\n")
	// Always emit tier lines when any count field was present; otherwise residual unknown.
	if v.HasWorking || v.HasContextual || v.HasArchival || v.HasSemantic || v.HasTotal {
		fmt.Fprintf(&b, "tiers: working=%d contextual=%d archival=%d semantic=%d",
			v.WorkingCount, v.ContextualCount, v.ArchivalCount, v.SemanticCount)
		if v.HasTotal {
			fmt.Fprintf(&b, " total=%d", v.TotalEntries)
		}
		b.WriteString("\n")
	} else {
		b.WriteString("tiers: (unavailable from wire)\n")
	}
	last := strings.TrimSpace(v.LastCompaction)
	// Zero-value last compaction is honest residual, not invent green.
	if last == "" || last == "0001-01-01T00:00:00Z" {
		b.WriteString("last_compaction: (none)\n")
	} else {
		fmt.Fprintf(&b, "last_compaction: %s\n", last)
	}
	b.WriteString(compactStatusHonestyFooter)
	return truncateBytes(b.String(), maxBytes)
}

// formatCompactStatusJSON parses MCP memory_compact_status JSON into human layout.
// Defensive: accepts nested stats (PascalCase MemoryStats or snake_case) + top-level last_compaction.
// Returns empty when parse fails (caller may pass through raw).
func formatCompactStatusJSON(raw string, maxBytes int) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw[0] != '{' {
		return ""
	}
	var res memoryCompactStatusResult
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		return ""
	}
	v := compactStatusView{}
	// Prefer nested stats object when present.
	if len(res.Stats) > 0 && string(res.Stats) != "null" {
		// Try as generic map for PascalCase + snake_case keys.
		var m map[string]any
		if err := json.Unmarshal(res.Stats, &m); err == nil {
			v.WorkingCount, v.HasWorking = compactStatusInt(m, "WorkingCount", "working_count")
			v.ContextualCount, v.HasContextual = compactStatusInt(m, "ContextualCount", "contextual_count")
			v.ArchivalCount, v.HasArchival = compactStatusInt(m, "ArchivalCount", "archival_count")
			v.SemanticCount, v.HasSemantic = compactStatusInt(m, "SemanticCount", "semantic_count")
			v.TotalEntries, v.HasTotal = compactStatusInt(m, "TotalEntries", "total_entries")
			if lc, ok := compactStatusString(m, "LastCompaction", "last_compaction"); ok {
				v.LastCompaction = lc
			}
		}
	}
	// Flat top-level snake_case fallbacks (resource URI shape / partial payloads).
	if res.WorkingCount != nil {
		v.WorkingCount = *res.WorkingCount
		v.HasWorking = true
	}
	if res.ContextualCount != nil {
		v.ContextualCount = *res.ContextualCount
		v.HasContextual = true
	}
	if res.ArchivalCount != nil {
		v.ArchivalCount = *res.ArchivalCount
		v.HasArchival = true
	}
	if res.SemanticCount != nil {
		v.SemanticCount = *res.SemanticCount
		v.HasSemantic = true
	}
	if res.TotalEntries != nil {
		v.TotalEntries = *res.TotalEntries
		v.HasTotal = true
	}
	if lc := strings.TrimSpace(res.LastCompaction); lc != "" {
		v.LastCompaction = lc
	}
	// Require at least some signal that this is compact-status shaped; empty {}
	// still formats residual (tiers unavailable) so callers get honesty footer.
	return formatCompactStatus(v, maxBytes)
}

// compactStatusInt extracts an int from a defensive stats map under alternate keys.
func compactStatusInt(m map[string]any, keys ...string) (int, bool) {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		switch n := v.(type) {
		case float64:
			return int(n), true
		case int:
			return n, true
		case int64:
			return int(n), true
		case json.Number:
			i, err := n.Int64()
			if err == nil {
				return int(i), true
			}
		}
	}
	return 0, false
}

// compactStatusString extracts a string (or RFC3339 time) under alternate keys.
func compactStatusString(m map[string]any, keys ...string) (string, bool) {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		switch s := v.(type) {
		case string:
			s = strings.TrimSpace(s)
			if s != "" {
				return s, true
			}
		}
	}
	return "", false
}

// MemoryTriggerCompactOpts for opt-in HITL RecMem compaction advisory (s1311 / aion memory_trigger_compact).
// Confirm must be true to call MCP — accidental mutation is refused residual-honestly.
//
// MCP-first: platform ships MCP tool memory_trigger_compact (publishes memory.compact.trigger
// advisory for RecMem worker). Do not invent lean HTTP trigger. Honesty: RecMem advisory ·
// not invent compaction green · dual_write OFF · not Memory GA · mutating HITL.
type MemoryTriggerCompactOpts struct {
	Confirm bool // must be true to call MCP
}

// triggerCompactHonestyFooter is the residual-honest pin for trigger-compact output (s1311).
// Locked: RecMem advisory · not invent compaction green · dual_write OFF · not Memory GA · mutating HITL.
const triggerCompactHonestyFooter = "honesty: RecMem advisory · not invent compaction green · dual_write OFF · not Memory GA · mutating HITL · MCP-first"

// memoryTriggerCompactResult is the aion MCP memory_trigger_compact JSON wire shape.
// Output: { triggered, cluster_size }.
type memoryTriggerCompactResult struct {
	Triggered   bool `json:"triggered"`
	ClusterSize int  `json:"cluster_size"`
}

// MemoryTriggerCompact publishes a RecMem compaction advisory (s1311 · aion memory_trigger_compact).
// Prefers MCP tool memory_trigger_compact on the configured memory server (MCP-first;
// no lean HTTP invent). Mutating advisory for RecMem worker — NOT auto-compact product green;
// NOT Memory GA; dual_write OFF.
//
// HITL gate: Confirm must be true or the call is refused residual-honestly (string, not error)
// without invoking MCP. Offline / tool failure is residual-honest fail-open messaging
// (never invents triggered/cluster_size success). Opt-in only — never auto-trigger.
func (rt *Runtime) MemoryTriggerCompact(ctx context.Context, opts MemoryTriggerCompactOpts) (string, error) {
	if rt == nil || !rt.memory.Enabled {
		return "", fmt.Errorf("memory hooks disabled")
	}

	// HITL residual-honest refusal — do NOT call MCP without explicit confirm.
	if !opts.Confirm {
		return formatTriggerCompactRefuseConfirm(), nil
	}

	// MCP-first — no lean HTTP trigger invent on platform (document, do not invent).
	if !rt.mcpMemoryReady() {
		return formatTriggerCompactOffline(rt.memory.Server), nil
	}
	c := rt.mcp.ClientByName(rt.memory.Server)
	args := map[string]any{}
	if t := rt.memoryTenant(); t != "" {
		args["tenant"] = t
	}
	start := time.Now()
	out, err := c.CallTool(ctx, "memory_trigger_compact", args)
	latMS := int(time.Since(start).Milliseconds())
	rt.lastMemoryRetrieveMS.Store(int64(latMS))
	rt.lastMemoryRetrieveCacheHit.Store(false)
	if err != nil {
		// Fail-open residual-honest call failure — do not invent triggered success.
		return formatTriggerCompactCallFailed(err), nil
	}
	if formatted := formatTriggerCompactJSON(out); formatted != "" {
		return formatted, nil
	}
	// Unknown payload — pass through with honesty footer (never invent triggered green).
	raw := strings.TrimSpace(out)
	if raw == "" {
		return formatTriggerCompactEmptyPayload(), nil
	}
	return raw + "\n" + triggerCompactHonestyFooter, nil
}

// formatTriggerCompactRefuseConfirm is residual-honest HITL refusal when Confirm is false.
// Does not call MCP; does not invent trigger success.
func formatTriggerCompactRefuseConfirm() string {
	return fmt.Sprintf(
		"trigger-compact\nstatus: refused · HITL confirmation required\n  mutating RecMem compaction advisory — pass --i-confirm to proceed\n  (not accidental; not invent compaction green)\n%s",
		triggerCompactHonestyFooter,
	)
}

// formatTriggerCompactOffline is residual-honest fail-open when MCP memory server is unavailable.
// Explicitly not inventing triggered / cluster_size success.
func formatTriggerCompactOffline(server string) string {
	if server == "" {
		server = "memory"
	}
	return fmt.Sprintf(
		"trigger-compact\nstatus: unavailable · mcp server %q not connected · MCP-first (no lean HTTP invent)\n%s · fail-open (not inventing triggered/cluster_size)",
		server, triggerCompactHonestyFooter,
	)
}

// formatTriggerCompactCallFailed is residual-honest fail-open when MCP tool call errors.
func formatTriggerCompactCallFailed(err error) string {
	msg := "error"
	if err != nil {
		msg = err.Error()
	}
	return fmt.Sprintf(
		"trigger-compact\nstatus: unavailable · mcp call failed: %s\n%s · fail-open (not inventing triggered/cluster_size)",
		msg, triggerCompactHonestyFooter,
	)
}

// formatTriggerCompactEmptyPayload is residual-honest when MCP returns empty body after call.
// Never invents triggered green.
func formatTriggerCompactEmptyPayload() string {
	return fmt.Sprintf(
		"trigger-compact\nstatus: ok · empty payload (not inventing triggered/cluster_size)\n%s",
		triggerCompactHonestyFooter,
	)
}

// formatTriggerCompact turns triggered + cluster_size into residual-honest result.
// Call only when MCP returned a parseable payload — never invent success offline.
func formatTriggerCompact(triggered bool, clusterSize int) string {
	return fmt.Sprintf(
		"trigger-compact\ntriggered: %v\ncluster_size: %d\n%s",
		triggered, clusterSize, triggerCompactHonestyFooter,
	)
}

// formatTriggerCompactJSON parses MCP memory_trigger_compact JSON
// {triggered, cluster_size} into the same human-readable layout as formatTriggerCompact.
// Returns empty when parse fails (caller may pass through raw).
func formatTriggerCompactJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw[0] != '{' {
		return ""
	}
	var res memoryTriggerCompactResult
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		return ""
	}
	// Accept triggered=false + zero cluster (honest wire) when JSON parsed — that is real wire, not invent.
	// Require at least one known key so unrelated {} does not silently format as false/0 success.
	var probe map[string]any
	if err := json.Unmarshal([]byte(raw), &probe); err != nil {
		return ""
	}
	if _, okT := probe["triggered"]; !okT {
		if _, okC := probe["cluster_size"]; !okC {
			return ""
		}
	}
	return formatTriggerCompact(res.Triggered, res.ClusterSize)
}

// advancedMemoryTools is the residual-honest inventory of advanced MCP tools for
// MemoryAdvancedStatus (s1311). Order is operator-facing slash surface order.
var advancedMemoryTools = []struct {
	Tool  string
	Slash string // human slash surface label
	Note  string // residual note (HITL / read-only / etc.)
}{
	{"memory_related", "related", "multi-hop lite"},
	{"memory_facts_as_of", "facts-as-of", "K4 lite"},
	{"memory_supersede_entity", "supersede", "HITL mutating"},
	{"memory_timeline", "timeline", "read-only"},
	{"memory_compact_status", "compact-status", "read-only"},
	{"memory_search_semantic", "semantic", "tier-4 residual"},
	{"memory_ingest_event", "ingest-event", "s138 T1 · not turn"},
	{"memory_patterns_list", "patterns", "ops pulse Beta"},
	{"memory_anomalies_list", "anomalies", "ops pulse Beta"},
	{"ops_digest_export", "digest", "ops GA-path framing"},
	{"memory_trigger_compact", "trigger-compact", "HITL mutating RecMem advisory"},
}

// advancedStatusHonestyFooter is the residual-honest pin for MemoryAdvancedStatus (s1311).
const advancedStatusHonestyFooter = "honesty: advanced MCP inventory residual · dual_write OFF · not Memory GA · presence ≠ product green · trigger-compact requires HITL · fail-open"

// MemoryAdvancedStatus is the residual-honest advanced MCP tool inventory pulse (s1311).
// Probes MCP tool presence (same discovery as mcpToolPresence; does not invent) when
// the memory server path is connected; fail-open offline/missing. Lists key advanced
// surfaces: related, facts-as-of, supersede, timeline, compact-status, semantic,
// ingest-event, patterns, anomalies, ops digest, trigger-compact.
//
// Always includes dual_write OFF + not Memory GA + integrations one-liner pointer.
// Does not call tools (presence probe only) — no invent of tool results.
func (rt *Runtime) MemoryAdvancedStatus(ctx context.Context) (string, error) {
	_ = ctx // reserved for future optional live probes; presence-only today
	var b strings.Builder
	b.WriteString("memory advanced status (s1311 residual-honest inventory pulse)\n")

	// 1) Memory hooks + MCP memory server path
	if rt == nil || !rt.memory.Enabled {
		b.WriteString("memory: hooks disabled (set [memory] enabled=true + MCP server)\n")
	} else {
		server := rt.memory.Server
		if server == "" {
			server = "memory"
		}
		connected := rt.mcpMemoryReady()
		fmt.Fprintf(&b, "memory: enabled server=%q mcp=%v dual_write=%v\n",
			server, connected, rt.memory.DualWrite)
	}

	// 2) MCP path (manager-level)
	pathState, nServers := rt.mcpPathState()
	switch pathState {
	case "available":
		fmt.Fprintf(&b, "MCP path: available (%d server(s))\n", nServers)
	case "empty":
		b.WriteString("MCP path: connected-empty (manager present, 0 servers) · fail-open\n")
	default:
		b.WriteString("MCP path: offline (no MCP manager/clients) · fail-open\n")
	}

	// 3) Advanced tool inventory (present | missing | offline)
	b.WriteString("advanced tools:\n")
	for _, t := range advancedMemoryTools {
		st := rt.mcpToolPresence(t.Tool)
		fmt.Fprintf(&b, "  %-32s %s", t.Tool+":", st)
		if t.Note != "" {
			fmt.Fprintf(&b, " · %s", t.Note)
		}
		if t.Slash != "" {
			fmt.Fprintf(&b, " · /memory %s", t.Slash)
		}
		b.WriteString("\n")
	}

	// 4) Residual pins always
	b.WriteString("dual_write: OFF (default local-primary honesty)\n")
	if rt != nil && rt.memory.Enabled && rt.memory.DualWrite {
		// Config override — still residual; do not invent GA.
		b.WriteString("  note: config dual_write=true is optional mesh audit only · not Memory GA\n")
	}
	b.WriteString("not Memory GA · presence ≠ Connected / product green\n")
	b.WriteString("integrations: see /integrations status for connector path\n")
	b.WriteString(advancedStatusHonestyFooter)
	return strings.TrimSpace(b.String()), nil
}

// MemorySemanticOpts for opt-in tier-4 semantic facts search (s1301 / aion memory_search_semantic).
// Query required. Limit optional (zero → config Limit). Does NOT run on default auto-recall.
//
// MCP-first: platform ships MCP tool memory_search_semantic; there is no lean HTTP
// semantic-search invent. Honesty: tier-4 semantic facts residual · not Memory GA · dual_write OFF.
// Empty facts ≠ invent memories.
type MemorySemanticOpts struct {
	Query string
	Limit int
}

// MemoryIngestEventOpts for opt-in ops/telemetry event ingest (s1301 / aion memory_ingest_event).
// Subject + Content required. Optional EventTime, SessionID, SessionSeq, Severity, SourceStream.
// This is **not** a conversation turn (use MemoryIngestTurn / /memory ingest for turns).
//
// MCP-first: platform ships MCP tool memory_ingest_event (s138 T1 temporal event telemetry).
// Honesty: s138 T1 temporal event telemetry · not conversation turn · not Memory GA · dual_write OFF.
// Never invent memory_id when offline / call failed.
type MemoryIngestEventOpts struct {
	Subject      string
	Content      string
	EventTime    string
	SessionID    string
	Severity     string
	SourceStream string
	SessionSeq   int
}

// semanticHonestyFooter is the residual-honest pin for semantic search output (s1301).
// Locked: tier-4 semantic facts residual · not Memory GA · dual_write OFF · MCP-first.
const semanticHonestyFooter = "honesty: tier-4 semantic facts residual · not Memory GA · dual_write OFF · MCP-first (no lean HTTP invent) · empty ≠ invent"

// ingestEventHonestyFooter is the residual-honest pin for ingest-event output (s1301).
// Locked: s138 T1 temporal event telemetry · not conversation turn · not Memory GA · dual_write OFF · MCP-first.
const ingestEventHonestyFooter = "honesty: s138 T1 temporal event telemetry · not conversation turn · not Memory GA · dual_write OFF · MCP-first"

// semanticFact is a defensive wire shape for aion memory_search_semantic facts.
// Accepts id/summary/full/score when present (memoryHit-like).
type semanticFact struct {
	ID      string  `json:"id"`
	Summary string  `json:"summary"`
	Full    string  `json:"full"`
	Score   float64 `json:"score"`
}

type memorySemanticResult struct {
	Facts []semanticFact `json:"facts"`
}

// memoryIngestEventResult is the aion MCP memory_ingest_event JSON wire shape.
// Only fields present on the wire are shown — never invent memory_id.
type memoryIngestEventResult struct {
	MemoryID  string `json:"memory_id"`
	Tier      *int   `json:"tier"`
	EventTime string `json:"event_time"`
	Audited   *bool  `json:"audited"`
}

// MemorySearchSemantic lists tier-4 semantic facts for a query (s1301 · aion memory_search_semantic).
// Prefers MCP tool memory_search_semantic on the configured memory server (MCP-first;
// no lean HTTP invent). Returns human-readable facts + honesty footer.
// Empty list is honest empty (facts: (none)) — never invents memories.
// Offline / tool failure is residual-honest fail-open messaging. Opt-in only — not auto-recall.
func (rt *Runtime) MemorySearchSemantic(ctx context.Context, opts MemorySemanticOpts) (string, error) {
	if rt == nil || !rt.memory.Enabled {
		return "", fmt.Errorf("memory hooks disabled")
	}
	query := strings.TrimSpace(opts.Query)
	if query == "" {
		return "", fmt.Errorf("query required for memory semantic search")
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = rt.memory.Limit
	}
	if limit <= 0 {
		limit = 8
	}
	maxBytes := rt.memory.MaxSnippetBytes
	if maxBytes <= 0 {
		maxBytes = 6000
	}

	// MCP-first — no lean HTTP /memory/search_semantic invent.
	if !rt.mcpMemoryReady() {
		return formatSemanticOffline(rt.memory.Server, query), nil
	}
	c := rt.mcp.ClientByName(rt.memory.Server)
	args := map[string]any{
		"query": query,
		"limit": limit,
	}
	if t := rt.memoryTenant(); t != "" {
		args["tenant"] = t
	}
	start := time.Now()
	out, err := c.CallTool(ctx, "memory_search_semantic", args)
	latMS := int(time.Since(start).Milliseconds())
	rt.lastMemoryRetrieveMS.Store(int64(latMS))
	rt.lastMemoryRetrieveCacheHit.Store(false)
	if err != nil {
		// Fail-open residual-honest call failure — do not invent facts.
		return formatSemanticCallFailed(query, err), nil
	}
	if formatted := formatSemanticJSON(out, query, maxBytes); formatted != "" {
		return formatted, nil
	}
	// Unknown payload — pass through with honesty footer (never invent structure).
	raw := strings.TrimSpace(out)
	if raw == "" {
		return formatSemantic(query, nil, maxBytes), nil
	}
	return truncateBytes(raw+"\n"+semanticHonestyFooter, maxBytes), nil
}

// MemoryIngestEvent ingests an ops/telemetry event (s1301 · aion memory_ingest_event · s138 T1).
// Prefers MCP tool memory_ingest_event on the configured memory server (MCP-first).
// Subject + Content required. This is **not** a conversation turn — use MemoryIngestTurn for turns.
// dual_write is intentionally not invoked (MCP-first residual; dual_write OFF for this surface).
// Offline / tool failure is residual-honest fail-open messaging — never invent memory_id.
// Opt-in only — not auto-ingest.
func (rt *Runtime) MemoryIngestEvent(ctx context.Context, opts MemoryIngestEventOpts) (string, error) {
	if rt == nil || !rt.memory.Enabled {
		return "", fmt.Errorf("memory hooks disabled")
	}
	subject := strings.TrimSpace(opts.Subject)
	content := strings.TrimSpace(opts.Content)
	if subject == "" {
		return "", fmt.Errorf("subject required for memory ingest-event")
	}
	if content == "" {
		return "", fmt.Errorf("content required for memory ingest-event")
	}
	maxBytes := rt.memory.MaxSnippetBytes
	if maxBytes <= 0 {
		maxBytes = 6000
	}

	// MCP-first — no lean HTTP invent; dual_write OFF for this surface.
	if !rt.mcpMemoryReady() {
		return formatIngestEventOffline(rt.memory.Server, subject), nil
	}
	c := rt.mcp.ClientByName(rt.memory.Server)
	args := map[string]any{
		"subject": subject,
		"content": content,
	}
	if t := rt.memoryTenant(); t != "" {
		args["tenant"] = t
	}
	if et := strings.TrimSpace(opts.EventTime); et != "" {
		args["event_time"] = et
	}
	sid := strings.TrimSpace(opts.SessionID)
	if sid == "" {
		sid = rt.memorySessionID()
	}
	if sid != "" {
		args["session_id"] = sid
	}
	if opts.SessionSeq != 0 {
		args["session_seq"] = opts.SessionSeq
	}
	if sev := strings.TrimSpace(opts.Severity); sev != "" {
		args["severity"] = sev
	}
	if ss := strings.TrimSpace(opts.SourceStream); ss != "" {
		args["source_stream"] = ss
	}
	start := time.Now()
	out, err := c.CallTool(ctx, "memory_ingest_event", args)
	latMS := int(time.Since(start).Milliseconds())
	rt.lastMemoryRetrieveMS.Store(int64(latMS))
	rt.lastMemoryRetrieveCacheHit.Store(false)
	if err != nil {
		// Fail-open residual-honest call failure — do not invent memory_id.
		return formatIngestEventCallFailed(subject, err), nil
	}
	if formatted := formatIngestEventJSON(out, subject, maxBytes); formatted != "" {
		return formatted, nil
	}
	// Unknown payload — pass through with honesty footer (never invent memory_id).
	raw := strings.TrimSpace(out)
	if raw == "" {
		return formatIngestEvent(subject, memoryIngestEventResult{}, maxBytes), nil
	}
	return truncateBytes(raw+"\n"+ingestEventHonestyFooter, maxBytes), nil
}

// formatSemanticOffline is residual-honest fail-open when MCP memory server is unavailable.
// Explicitly not empty-facts success (empty ≠ invent memories).
func formatSemanticOffline(server, query string) string {
	if server == "" {
		server = "memory"
	}
	return fmt.Sprintf(
		"semantic query=%s\nstatus: unavailable · mcp server %q not connected · MCP-first (no lean HTTP invent)\n%s · fail-open (empty ≠ invent memories)",
		emptyDash(query), server, semanticHonestyFooter,
	)
}

// formatSemanticCallFailed is residual-honest fail-open when MCP tool call errors.
func formatSemanticCallFailed(query string, err error) string {
	msg := "error"
	if err != nil {
		msg = err.Error()
	}
	return fmt.Sprintf(
		"semantic query=%s\nstatus: unavailable · mcp call failed: %s\n%s · fail-open (empty ≠ invent memories)",
		emptyDash(query), msg, semanticHonestyFooter,
	)
}

// formatSemantic turns semantic facts into a compact residual-honest listing.
// Empty → "facts: (none)" + honesty footer (never invent memories).
// Lines prefer id + summary/full truncated.
func formatSemantic(query string, facts []semanticFact, maxBytes int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "semantic query=%s\n", emptyDash(query))
	if len(facts) == 0 {
		b.WriteString("facts: (none)\n")
	} else {
		fmt.Fprintf(&b, "facts (%d):\n", len(facts))
		for i, f := range facts {
			id := strings.TrimSpace(f.ID)
			text := strings.TrimSpace(f.Summary)
			if text == "" {
				text = strings.TrimSpace(f.Full)
			}
			if text == "" && id != "" {
				text = id
			}
			if text == "" {
				text = "(empty)"
			}
			// Truncate long full/summary for display residual.
			const maxFactText = 240
			if utf8.RuneCountInString(text) > maxFactText {
				runes := []rune(text)
				text = string(runes[:maxFactText]) + "…"
			}
			prefix := ""
			if f.Score > 0 {
				prefix = fmt.Sprintf("[%.2f] ", f.Score)
			}
			var line string
			if id != "" && id != text {
				line = fmt.Sprintf("  %d. %s%s · %s\n", i+1, prefix, id, text)
			} else {
				line = fmt.Sprintf("  %d. %s%s\n", i+1, prefix, text)
			}
			if maxBytes > 0 && b.Len()+len(line)+len(semanticHonestyFooter) > maxBytes {
				break
			}
			b.WriteString(line)
		}
	}
	b.WriteString(semanticHonestyFooter)
	return truncateBytes(b.String(), maxBytes)
}

// formatSemanticJSON parses MCP memory_search_semantic JSON {facts:[...]} into
// the same human-readable layout as formatSemantic. Returns empty when parse fails.
func formatSemanticJSON(raw, query string, maxBytes int) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw[0] != '{' {
		return ""
	}
	var res memorySemanticResult
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		return ""
	}
	// Empty array is honest empty; unrelated {} also formats empty facts.
	return formatSemantic(query, res.Facts, maxBytes)
}

// formatIngestEventOffline is residual-honest fail-open when MCP memory server is unavailable.
// Explicitly never invents memory_id.
func formatIngestEventOffline(server, subject string) string {
	if server == "" {
		server = "memory"
	}
	return fmt.Sprintf(
		"ingest-event subject=%s\nstatus: unavailable · mcp server %q not connected · MCP-first (no lean HTTP invent)\n%s · fail-open (never invent memory_id)",
		emptyDash(subject), server, ingestEventHonestyFooter,
	)
}

// formatIngestEventCallFailed is residual-honest fail-open when MCP tool call errors.
func formatIngestEventCallFailed(subject string, err error) string {
	msg := "error"
	if err != nil {
		msg = err.Error()
	}
	return fmt.Sprintf(
		"ingest-event subject=%s\nstatus: unavailable · mcp call failed: %s\n%s · fail-open (never invent memory_id)",
		emptyDash(subject), msg, ingestEventHonestyFooter,
	)
}

// formatIngestEvent turns wire fields into residual-honest lines.
// Only emits memory_id / tier / event_time / audited when present on wire — never invents ids.
func formatIngestEvent(subject string, res memoryIngestEventResult, maxBytes int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "ingest-event subject=%s\n", emptyDash(subject))
	id := strings.TrimSpace(res.MemoryID)
	if id != "" {
		fmt.Fprintf(&b, "memory_id: %s\n", id)
	} else {
		b.WriteString("memory_id: (none from wire)\n")
	}
	if res.Tier != nil {
		fmt.Fprintf(&b, "tier: %d\n", *res.Tier)
	} else {
		b.WriteString("tier: (unavailable from wire)\n")
	}
	et := strings.TrimSpace(res.EventTime)
	if et != "" {
		fmt.Fprintf(&b, "event_time: %s\n", et)
	} else {
		b.WriteString("event_time: (none from wire)\n")
	}
	if res.Audited != nil {
		fmt.Fprintf(&b, "audited: %t\n", *res.Audited)
	} else {
		b.WriteString("audited: (unavailable from wire)\n")
	}
	b.WriteString(ingestEventHonestyFooter)
	return truncateBytes(b.String(), maxBytes)
}

// formatIngestEventJSON parses MCP memory_ingest_event JSON into human layout.
// Returns empty when parse fails (caller may pass through raw).
func formatIngestEventJSON(raw, subject string, maxBytes int) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw[0] != '{' {
		return ""
	}
	var res memoryIngestEventResult
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		return ""
	}
	// Empty {} still formats residual (memory_id none) so callers get honesty footer.
	return formatIngestEvent(subject, res, maxBytes)
}

// MemoryRelated performs opt-in multi-hop lite related recall (s1135).
// Prefers sync HTTP RetrieveMemoryRelated (POST /v1|/v5/memory/related); falls back
// to MCP memory_related when sync fails or mesh is unavailable.
// At least one of seedEntity or query is required.
// Does NOT run on default auto-recall (honesty: multi-hop lite is slash/CLI opt-in).
// Not full graph RAG; not product Memory GA; dual_write OFF by default.
// Hop ranking path-aware lite: PreferShorterHops nil = kernel default true (s1067/s1277/s1281).
func (rt *Runtime) MemoryRelated(ctx context.Context, seedEntity, query string, opts ...MemoryRelatedOpts) (string, error) {
	if rt == nil || !rt.memory.Enabled {
		return "", fmt.Errorf("memory hooks disabled")
	}
	seedEntity = strings.TrimSpace(seedEntity)
	query = strings.TrimSpace(query)
	if seedEntity == "" && query == "" {
		return "", fmt.Errorf("seed_entity or query required for memory related")
	}

	var call MemoryRelatedOpts
	if len(opts) > 0 {
		call = opts[0]
	}
	maxHops := call.MaxHops
	if maxHops <= 0 {
		maxHops = rt.memory.RelatedMaxHops
	}
	if maxHops <= 0 {
		maxHops = 2
	}
	limit := call.Limit
	if limit <= 0 {
		limit = rt.memory.Limit
	}
	if limit <= 0 {
		limit = 8
	}
	maxBytes := rt.memory.MaxSnippetBytes
	if maxBytes <= 0 {
		maxBytes = 6000
	}

	// Prefer sync multi-hop related against memory sidecar HTTP when mesh client is live.
	if rt.syncMemoryReady() {
		start := time.Now()
		res, err := rt.mesh.RetrieveMemoryRelated(ctx, rt.memoryTenant(), iomesh.MemoryRelatedOptions{
			SeedEntity:        seedEntity,
			Query:             query,
			MaxHops:           maxHops,
			Limit:             limit,
			SessionID:         rt.memorySessionID(),
			PreferShorterHops: call.PreferShorterHops,
		})
		latMS := int(time.Since(start).Milliseconds())
		rt.lastMemoryRetrieveMS.Store(int64(latMS))
		rt.lastMemoryRetrieveCacheHit.Store(false)
		if err == nil {
			hits := res.Memories
			if hits == nil {
				hits = []iomesh.MemoryHit{}
			}
			return formatMemoryHits(hits, maxBytes), nil
		}
		if rt.logger != nil {
			rt.logger.Debug("memory related sync failed; trying MCP fallback", "err", err, "ms", latMS)
		}
		// Fall through to MCP when sidecar path is missing (e.g. broker-only endpoint).
	}

	if !rt.mcpMemoryReady() {
		if rt.syncMemoryReady() {
			return "", fmt.Errorf("memory related sync failed and mcp server %q not connected", rt.memory.Server)
		}
		return "", fmt.Errorf("mcp server %q not connected (and mesh sync unavailable)", rt.memory.Server)
	}
	c := rt.mcp.ClientByName(rt.memory.Server)
	args := map[string]any{
		"max_hops": maxHops,
		"limit":    limit,
	}
	if seedEntity != "" {
		args["seed_entity"] = seedEntity
	}
	if query != "" {
		args["query"] = query
	}
	if t := rt.memoryTenant(); t != "" {
		args["tenant"] = t
	}
	if sid := rt.memorySessionID(); sid != "" {
		args["session_id"] = sid
	}
	// s1281: omit prefer_shorter_hops when nil (platform default true); send only when set.
	if call.PreferShorterHops != nil {
		args["prefer_shorter_hops"] = *call.PreferShorterHops
	}
	start := time.Now()
	out, err := c.CallTool(ctx, "memory_related", args)
	latMS := int(time.Since(start).Milliseconds())
	rt.lastMemoryRetrieveMS.Store(int64(latMS))
	rt.lastMemoryRetrieveCacheHit.Store(false)
	if err != nil {
		return "", err
	}
	return truncateBytes(out, maxBytes), nil
}

// formatMemoryHits turns sync RetrieveMemory / related hits into a compact recall snippet.
// When maxBytes > 0, stops once the budget is reached without formatting remaining hits (s1069).
// Optional hop_distance is shown as hop=N when non-zero (s1135 multi-hop related).
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
		switch {
		case h.HopDistance > 0 && h.Score > 0:
			piece = fmt.Sprintf("[hop=%d] [%.2f] %s", h.HopDistance, h.Score, text)
		case h.HopDistance > 0:
			piece = fmt.Sprintf("[hop=%d] %s", h.HopDistance, text)
		case h.Score > 0:
			piece = fmt.Sprintf("[%.2f] %s", h.Score, text)
		default:
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
