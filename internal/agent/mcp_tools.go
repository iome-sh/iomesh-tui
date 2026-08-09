package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iome-sh/iomesh-tui/internal/mcp"
	"github.com/iome-sh/iomesh-tui/internal/router"
)

// mcpMetaToolNames are registry names for MCP resource/prompt meta tools (not mcp__*).
var mcpMetaToolNames = []string{
	"list_mcp_resources",
	"read_mcp_resource",
	"list_mcp_prompts",
	"get_mcp_prompt",
}

// UnregisterMCPTools removes all mcp__* bindings and MCP meta tools from the registry.
// Used by Runtime.ReplaceMCP before re-attach (s1526 P4 hot reload).
func (r *ToolRegistry) UnregisterMCPTools() {
	if r == nil || r.funcs == nil {
		return
	}
	for name := range r.funcs {
		if strings.HasPrefix(name, "mcp__") {
			delete(r.funcs, name)
			delete(r.meta, name)
		}
	}
	for _, name := range mcpMetaToolNames {
		delete(r.funcs, name)
		delete(r.meta, name)
	}
}

// RegisterMCPTools exposes connected MCP server tools as agent tools
// named mcp__<server>__<tool>. Mutating flag comes from server config (default true).
// Also registers read-only meta tools for resources and prompts.
func (r *ToolRegistry) RegisterMCPTools(mgr *mcp.Manager) {
	if mgr == nil || mgr.Len() == 0 {
		return
	}
	// Meta tools (read-only) for MCP resources / prompts.
	r.register("list_mcp_resources", false, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name:        "list_mcp_resources",
			Description: "List MCP resources (uri) from connected servers. Optional server filter.",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"server":{"type":"string"},"refresh":{"type":"boolean"}}}`),
		},
	}, func(ctx context.Context, args string) (string, error) {
		var p struct {
			Server  string `json:"server"`
			Refresh bool   `json:"refresh"`
		}
		_ = json.Unmarshal([]byte(args), &p)
		return mgr.ListAllResources(ctx, p.Server, p.Refresh)
	})
	r.register("read_mcp_resource", false, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name:        "read_mcp_resource",
			Description: "Read an MCP resource by uri (optional server when multiple MCP servers).",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"uri":{"type":"string"},"server":{"type":"string"}},"required":["uri"]}`),
		},
	}, func(ctx context.Context, args string) (string, error) {
		var p struct {
			URI    string `json:"uri"`
			Server string `json:"server"`
		}
		if err := json.Unmarshal([]byte(args), &p); err != nil {
			return "", err
		}
		return mgr.ReadResourceOn(ctx, p.Server, p.URI)
	})
	r.register("list_mcp_prompts", false, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name:        "list_mcp_prompts",
			Description: "List MCP prompt templates from connected servers.",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"server":{"type":"string"},"refresh":{"type":"boolean"}}}`),
		},
	}, func(ctx context.Context, args string) (string, error) {
		var p struct {
			Server  string `json:"server"`
			Refresh bool   `json:"refresh"`
		}
		_ = json.Unmarshal([]byte(args), &p)
		return mgr.ListAllPrompts(ctx, p.Server, p.Refresh)
	})
	r.register("get_mcp_prompt", false, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name:        "get_mcp_prompt",
			Description: "Fetch an MCP prompt by name (optional arguments map and server).",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"},"server":{"type":"string"},"arguments":{"type":"object","additionalProperties":{"type":"string"}}},"required":["name"]}`),
		},
	}, func(ctx context.Context, args string) (string, error) {
		var p struct {
			Name      string            `json:"name"`
			Server    string            `json:"server"`
			Arguments map[string]string `json:"arguments"`
		}
		if err := json.Unmarshal([]byte(args), &p); err != nil {
			return "", err
		}
		return mgr.GetPromptOn(ctx, p.Server, p.Name, p.Arguments)
	})

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
