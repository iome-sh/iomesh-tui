package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/router"
)

// MemoryConfig configures optional Palace memory via MCP (aion-memory-mcp).
// See docs/architecture/memory-mcp.md Phase 0–1.
type MemoryConfig struct {
	// Enabled turns on memory integration when the MCP server is connected.
	Enabled bool
	// Server is the MCP server name (default aion-memory).
	Server string
	// Tenant is the Palace tenant namespace (e.g. dept.engineering).
	Tenant string
	// AutoRecall runs memory_retrieve before each LLM turn (fail-open).
	AutoRecall bool
	// AutoIngest runs memory_ingest_turn for user+assistant after a successful turn.
	// Ingest is opt-in; does not go through interactive approval (config is the consent).
	AutoIngest bool
	// RecallLimit caps retrieve hits (default 8).
	RecallLimit int
}

// DefaultMemoryConfig returns Phase 0 defaults (off until MCP + enabled).
func DefaultMemoryConfig() MemoryConfig {
	return MemoryConfig{
		Enabled:     false,
		Server:      "aion-memory",
		AutoRecall:  true, // when Enabled
		AutoIngest:  false,
		RecallLimit: 8,
	}
}

// ConfigureMemory sets memory integration options (call after AttachMCP).
func (rt *Runtime) ConfigureMemory(cfg MemoryConfig) {
	if rt == nil {
		return
	}
	if cfg.Server == "" {
		cfg.Server = "aion-memory"
	}
	if cfg.RecallLimit <= 0 {
		cfg.RecallLimit = 8
	}
	rt.memCfg = cfg
	if cfg.Enabled {
		rt.appendSystemNote("memory", fmt.Sprintf(
			"Palace memory via MCP server %q (tenant=%q auto_recall=%v auto_ingest=%v). Tools: mcp__%s__memory_*.",
			cfg.Server, cfg.Tenant, cfg.AutoRecall, cfg.AutoIngest, sanitizeMemoryServer(cfg.Server),
		))
	}
}

// MemoryStatus is a one-line operator summary for /memory.
func (rt *Runtime) MemoryStatus() string {
	if rt == nil {
		return "memory: no runtime"
	}
	if !rt.memCfg.Enabled {
		return "memory: disabled (set [memory] enabled=true + [[mcp.servers]] aion-memory)"
	}
	connected := false
	if rt.mcp != nil && rt.mcp.ClientByName(rt.memCfg.Server) != nil {
		connected = true
	}
	return fmt.Sprintf("memory: enabled server=%s connected=%v tenant=%q auto_recall=%v auto_ingest=%v last_hits=%d seq=%d",
		rt.memCfg.Server, connected, rt.memCfg.Tenant, rt.memCfg.AutoRecall, rt.memCfg.AutoIngest,
		rt.memLastHits, rt.memSessionSeq)
}

// memorySessionID returns a stable session key for Palace.
func (rt *Runtime) memorySessionID() string {
	if rt.sessionID != "" {
		return rt.sessionID
	}
	return "local"
}

func sanitizeMemoryServer(name string) string {
	// Match mcp.sanitize roughly for display (qualified tools use mcp sanitize).
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

// memoryCall invokes a tool on the configured memory MCP server (fail-open empty).
func (rt *Runtime) memoryCall(ctx context.Context, tool string, args map[string]any) (string, error) {
	if rt == nil || rt.mcp == nil || !rt.memCfg.Enabled {
		return "", fmt.Errorf("memory mcp disabled")
	}
	server := rt.memCfg.Server
	if server == "" {
		server = "aion-memory"
	}
	c := rt.mcp.ClientByName(server)
	if c == nil {
		return "", fmt.Errorf("memory mcp server %q not connected", server)
	}
	return c.CallTool(ctx, tool, args)
}

// injectMemoryRecall fail-open retrieves and appends a system message.
func (rt *Runtime) injectMemoryRecall(ctx context.Context, query string, onEvent func(Event)) {
	if rt == nil || !rt.memCfg.Enabled || !rt.memCfg.AutoRecall {
		return
	}
	if rt.mcp == nil || rt.mcp.ClientByName(rt.memCfg.Server) == nil {
		return
	}
	limit := rt.memCfg.RecallLimit
	if limit <= 0 {
		limit = 8
	}
	args := map[string]any{
		"query":      query,
		"limit":      limit,
		"session_id": rt.memorySessionID(),
	}
	if rt.memCfg.Tenant != "" {
		args["tenant"] = rt.memCfg.Tenant
	}
	raw, err := rt.memoryCall(ctx, "memory_retrieve", args)
	if err != nil {
		if rt.logger != nil {
			rt.logger.Debug("memory recall failed (fail-open)", "err", err)
		}
		return
	}
	snippet, hits := FormatMemoryRecallSnippet(raw, limit)
	rt.memLastHits = hits
	if snippet == "" {
		return
	}
	rt.messages = append(rt.messages, router.Message{
		Role:    "system",
		Content: "<iomesh-memory>\n" + snippet + "\n</iomesh-memory>",
	})
	onEvent(Event{Type: EventMemory, Text: fmt.Sprintf("injected Palace memory (%d hits)", hits)})
}

// autoIngestTurns fail-open writes user + assistant turns to Palace.
func (rt *Runtime) autoIngestTurns(ctx context.Context, userText, assistantText string, onEvent func(Event)) {
	if rt == nil || !rt.memCfg.Enabled || !rt.memCfg.AutoIngest {
		return
	}
	if rt.mcp == nil || rt.mcp.ClientByName(rt.memCfg.Server) == nil {
		return
	}
	userText = strings.TrimSpace(userText)
	assistantText = strings.TrimSpace(assistantText)
	if userText == "" && assistantText == "" {
		return
	}
	eventTime := time.Now().UTC().Format(time.RFC3339)
	sessionID := rt.memorySessionID()
	n := 0
	if userText != "" {
		if err := rt.ingestOne(ctx, "user", userText, sessionID, eventTime); err != nil {
			if rt.logger != nil {
				rt.logger.Debug("memory ingest user failed", "err", err)
			}
		} else {
			n++
		}
	}
	if assistantText != "" {
		if err := rt.ingestOne(ctx, "assistant", assistantText, sessionID, eventTime); err != nil {
			if rt.logger != nil {
				rt.logger.Debug("memory ingest assistant failed", "err", err)
			}
		} else {
			n++
		}
	}
	if n > 0 {
		onEvent(Event{Type: EventMemory, Text: fmt.Sprintf("ingested %d turn(s) to Palace", n)})
	}
}

func (rt *Runtime) ingestOne(ctx context.Context, role, content, sessionID, eventTime string) error {
	rt.memSessionSeq++
	args := map[string]any{
		"session_id":  sessionID,
		"role":        role,
		"content":     content,
		"event_time":  eventTime,
		"session_seq": rt.memSessionSeq,
	}
	if rt.memCfg.Tenant != "" {
		args["tenant"] = rt.memCfg.Tenant
	}
	_, err := rt.memoryCall(ctx, "memory_ingest_turn", args)
	return err
}

// FormatMemoryRecallSnippet turns MCP memory_retrieve JSON/text into a prompt block.
// Returns snippet and approximate hit count.
func FormatMemoryRecallSnippet(raw string, max int) (string, int) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", 0
	}
	if max <= 0 {
		max = 8
	}
	// Prefer JSON {"memories":[...]} from aion-memory-mcp.
	var payload struct {
		Memories []struct {
			ID      string  `json:"id"`
			Content string  `json:"content"`
			Summary string  `json:"summary"`
			Text    string  `json:"text"`
			Score   float64 `json:"score"`
			Tier    int     `json:"tier"`
		} `json:"memories"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err == nil && len(payload.Memories) > 0 {
		var b strings.Builder
		b.WriteString("Retrieved Palace memories (temporal hybrid recall):\n")
		n := 0
		for _, m := range payload.Memories {
			if n >= max {
				fmt.Fprintf(&b, "… +%d more\n", len(payload.Memories)-max)
				break
			}
			text := firstNonEmptyMem(m.Content, m.Summary, m.Text)
			if text == "" {
				continue
			}
			id := m.ID
			if id == "" {
				id = fmt.Sprintf("#%d", n+1)
			}
			fmt.Fprintf(&b, "- [%s]", id)
			if m.Tier > 0 {
				fmt.Fprintf(&b, " tier=%d", m.Tier)
			}
			if m.Score > 0 {
				fmt.Fprintf(&b, " score=%.3f", m.Score)
			}
			b.WriteString(" ")
			b.WriteString(truncateMem(text, 400))
			b.WriteByte('\n')
			n++
		}
		if n == 0 {
			return "", 0
		}
		return strings.TrimSpace(b.String()), n
	}
	// Fallback: raw text (truncated).
	return "Retrieved Palace memories:\n" + truncateMem(raw, 2000), 1
}

func firstNonEmptyMem(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func truncateMem(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}
