package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/iome-sh/iomesh-tui/internal/mcp"
	"github.com/iome-sh/iomesh-tui/internal/router"
)

// RegisterMCPTools exposes connected MCP server tools as agent tools
// named mcp__<server>__<tool>. Mutating flag comes from server config (default true).
func (r *ToolRegistry) RegisterMCPTools(mgr *mcp.Manager) {
	if mgr == nil || mgr.Len() == 0 {
		return
	}
	for _, b := range mgr.Bindings() {
		b := b
		// Find description/schema from client tool list.
		var desc string
		var schema json.RawMessage = json.RawMessage(`{"type":"object","properties":{}}`)
		for _, t := range b.Client.Tools() {
			if t.Name == b.Tool {
				desc = t.Description
				if len(t.InputSchema) > 0 {
					schema = t.InputSchema
				}
				break
			}
		}
		if desc == "" {
			desc = fmt.Sprintf("MCP tool %s from server %s", b.Tool, b.Server)
		} else {
			desc = fmt.Sprintf("[MCP:%s] %s", b.Server, desc)
		}
		name := b.Qualified
		r.register(name, b.Mutating, router.Tool{
			Type: "function",
			Function: router.ToolFunction{
				Name:        name,
				Description: desc,
				Parameters:  schema,
			},
		}, func(ctx context.Context, args string) (string, error) {
			var raw map[string]any
			if args != "" && args != "{}" {
				if err := json.Unmarshal([]byte(args), &raw); err != nil {
					return "", fmt.Errorf("mcp args: %w", err)
				}
			}
			return mgr.Call(ctx, name, raw)
		})
	}
}
