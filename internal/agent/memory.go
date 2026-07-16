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

// MemoryConfig controls optional Palace MCP auto-recall / auto-ingest.
// Does not embed Palace — only calls MCP tools on a connected server.
// Optional DualWrite also publishes async MEMORY_INGEST envelopes to the mesh.
type MemoryConfig struct {
	Enabled         bool
	Server          string // MCP server name (default "memory")
	Tenant          string
	AutoRecall      bool
	AutoIngest      bool
	// DualWrite publishes memory_ingest to mesh MEMORY_INGEST when mesh client is enabled (fail-open).
	DualWrite       bool
	Limit           int // retrieve limit (default 8)
	MaxSnippetBytes int // cap injected context (default 6000)
	// SessionID overrides Runtime.sessionID when non-empty.
	SessionID string
}

// DefaultMemoryConfig returns fail-open off defaults.
func DefaultMemoryConfig() MemoryConfig {
	return MemoryConfig{
		Enabled:         false,
		Server:          "memory",
		AutoRecall:      true,
		AutoIngest:      false,
		DualWrite:       false,
		Limit:           8,
		MaxSnippetBytes: 6000,
	}
}

// AttachMemory configures optional memory hooks (requires MCP manager with server,
// and/or DualWrite with mesh client for stream ingest).
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
	rt.memory = cfg
	if cfg.Enabled {
		rt.appendSystemNote("memory", fmt.Sprintf(
			"Memory Palace MCP: server=%q tenant=%q auto_recall=%v auto_ingest=%v dual_write=%v. Use mcp__%s__memory_* tools or /memory slash. Fail-open when server unavailable.",
			cfg.Server, emptyDash(cfg.Tenant), cfg.AutoRecall, cfg.AutoIngest, cfg.DualWrite, cfg.Server,
		))
	}
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

// MemoryStatusLine is a short operator-facing status (slash /memory).
func (rt *Runtime) MemoryStatusLine() string {
	if rt == nil {
		return "memory: no runtime"
	}
	cfg := rt.memory
	if !cfg.Enabled {
		return "memory: hooks disabled (set [memory] enabled=true + MCP server)"
	}
	connected := false
	if rt.mcp != nil {
		connected = rt.mcp.ClientByName(cfg.Server) != nil
	}
	return fmt.Sprintf("memory: enabled server=%q connected=%v tenant=%q auto_recall=%v auto_ingest=%v dual_write=%v session=%q",
		cfg.Server, connected, emptyDash(cfg.Tenant), cfg.AutoRecall, cfg.AutoIngest, cfg.DualWrite, emptyDash(rt.memorySessionID()))
}

// MemoryRecall calls memory_retrieve on the configured MCP server (fail-open → empty).
func (rt *Runtime) MemoryRecall(ctx context.Context, query string) (string, error) {
	if rt == nil || !rt.memory.Enabled {
		return "", fmt.Errorf("memory hooks disabled")
	}
	if rt.mcp == nil {
		return "", fmt.Errorf("mcp not attached")
	}
	c := rt.mcp.ClientByName(rt.memory.Server)
	if c == nil {
		return "", fmt.Errorf("mcp server %q not connected", rt.memory.Server)
	}
	q := strings.TrimSpace(query)
	if q == "" {
		q = "*"
	}
	limit := rt.memory.Limit
	if limit <= 0 {
		limit = 8
	}
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
	return c.CallTool(ctx, "memory_retrieve", args)
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
func (rt *Runtime) maybeInjectMemoryRecall(ctx context.Context, userText string, onEvent func(Event)) {
	if rt == nil || !rt.memory.Enabled || !rt.memory.AutoRecall {
		return
	}
	if !rt.mcpMemoryReady() {
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
	onEvent(Event{Type: EventMemoryRecall, Text: fmt.Sprintf("injected memory recall (%d bytes)", len(snippet))})
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
