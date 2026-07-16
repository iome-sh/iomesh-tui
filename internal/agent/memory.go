package agent

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/iome-sh/iomesh-tui/internal/router"
)

// MemoryConfig controls optional Palace MCP auto-recall / auto-ingest.
// Does not embed Palace — only calls MCP tools on a connected server.
type MemoryConfig struct {
	Enabled         bool
	Server          string // MCP server name (default "memory")
	Tenant          string
	AutoRecall      bool
	AutoIngest      bool
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
		Limit:           8,
		MaxSnippetBytes: 6000,
	}
}

// AttachMemory configures optional memory hooks (requires MCP manager with server).
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
			"Memory Palace MCP: server=%q tenant=%q auto_recall=%v auto_ingest=%v. Use mcp__%s__memory_* tools or /memory slash. Fail-open when server unavailable.",
			cfg.Server, emptyDash(cfg.Tenant), cfg.AutoRecall, cfg.AutoIngest, cfg.Server,
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
	return fmt.Sprintf("memory: enabled server=%q connected=%v tenant=%q auto_recall=%v auto_ingest=%v session=%q",
		cfg.Server, connected, emptyDash(cfg.Tenant), cfg.AutoRecall, cfg.AutoIngest, emptyDash(rt.memorySessionID()))
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
	if rt.memory.Tenant != "" {
		args["tenant"] = rt.memory.Tenant
	}
	if sid := rt.memorySessionID(); sid != "" {
		args["session_id"] = sid
	}
	return c.CallTool(ctx, "memory_retrieve", args)
}

// MemoryIngestTurn calls memory_ingest_turn (mutating).
func (rt *Runtime) MemoryIngestTurn(ctx context.Context, role, content string) (string, error) {
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
	content = strings.TrimSpace(content)
	if content == "" {
		return "", fmt.Errorf("empty content")
	}
	args := map[string]any{
		"role":       role,
		"content":    content,
		"event_time": time.Now().UTC().Format(time.RFC3339),
	}
	if rt.memory.Tenant != "" {
		args["tenant"] = rt.memory.Tenant
	}
	if sid := rt.memorySessionID(); sid != "" {
		args["session_id"] = sid
	}
	return c.CallTool(ctx, "memory_ingest_turn", args)
}

// maybeInjectMemoryRecall appends a system message when auto_recall is on (fail-open).
func (rt *Runtime) maybeInjectMemoryRecall(ctx context.Context, userText string, onEvent func(Event)) {
	if rt == nil || !rt.memory.Enabled || !rt.memory.AutoRecall {
		return
	}
	if rt.mcp == nil || rt.mcp.ClientByName(rt.memory.Server) == nil {
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
func (rt *Runtime) maybeAutoIngest(ctx context.Context, userText, assistantText string, onEvent func(Event)) {
	if rt == nil || !rt.memory.Enabled || !rt.memory.AutoIngest {
		return
	}
	if rt.mcp == nil || rt.mcp.ClientByName(rt.memory.Server) == nil {
		return
	}
	if strings.TrimSpace(userText) != "" {
		if _, err := rt.MemoryIngestTurn(ctx, "user", userText); err != nil {
			if rt.logger != nil {
				rt.logger.Debug("memory auto_ingest user", "err", err)
			}
			onEvent(Event{Type: EventMemoryIngest, Text: "auto_ingest user failed: " + err.Error()})
		} else {
			onEvent(Event{Type: EventMemoryIngest, Text: "auto_ingest user turn"})
		}
	}
	if strings.TrimSpace(assistantText) != "" {
		if _, err := rt.MemoryIngestTurn(ctx, "assistant", assistantText); err != nil {
			if rt.logger != nil {
				rt.logger.Debug("memory auto_ingest assistant", "err", err)
			}
			onEvent(Event{Type: EventMemoryIngest, Text: "auto_ingest assistant failed: " + err.Error()})
		} else {
			onEvent(Event{Type: EventMemoryIngest, Text: "auto_ingest assistant turn"})
		}
	}
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
