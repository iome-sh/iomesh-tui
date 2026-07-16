package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/router"
	"github.com/iome-sh/iomesh-tui/internal/workspace"
)

// ToolRegistry maps tool names to implementations and schemas.
type ToolRegistry struct {
	ws    *workspace.Workspace
	funcs map[string]toolFunc
	meta  map[string]toolMeta
}

type toolFunc func(ctx context.Context, args string) (string, error)

type toolMeta struct {
	mutating bool
	schema   router.Tool
}

// DefaultTools registers the scaffold toolset (read, list, grep, shell, edit).
func DefaultTools(ws *workspace.Workspace) ToolRegistry {
	reg := ToolRegistry{
		ws:    ws,
		funcs: map[string]toolFunc{},
		meta:  map[string]toolMeta{},
	}
	reg.register("read_file", false, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name:        "read_file",
			Description: "Read a UTF-8 text file relative to the workspace root.",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"},"offset":{"type":"integer"},"limit":{"type":"integer"}},"required":["path"]}`),
		},
	}, func(ctx context.Context, args string) (string, error) {
		var p struct {
			Path   string `json:"path"`
			Offset int    `json:"offset"`
			Limit  int    `json:"limit"`
		}
		if err := json.Unmarshal([]byte(args), &p); err != nil {
			return "", err
		}
		return ws.ReadFile(p.Path, p.Offset, p.Limit)
	})

	reg.register("list_dir", false, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name:        "list_dir",
			Description: "List files in a directory relative to the workspace root.",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}`),
		},
	}, func(ctx context.Context, args string) (string, error) {
		var p struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(args), &p); err != nil {
			return "", err
		}
		entries, err := ws.ListDir(p.Path)
		if err != nil {
			return "", err
		}
		return strings.Join(entries, "\n"), nil
	})

	reg.register("grep", false, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name:        "grep",
			Description: "Search workspace files for a regex pattern.",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"pattern":{"type":"string"},"path":{"type":"string"}},"required":["pattern"]}`),
		},
	}, func(ctx context.Context, args string) (string, error) {
		var p struct {
			Pattern string `json:"pattern"`
			Path    string `json:"path"`
		}
		if err := json.Unmarshal([]byte(args), &p); err != nil {
			return "", err
		}
		return ws.Grep(p.Pattern, p.Path)
	})

	reg.register("run_shell", true, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name:        "run_shell",
			Description: "Run a shell command in the workspace root (requires approval).",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"command":{"type":"string"}},"required":["command"]}`),
		},
	}, func(ctx context.Context, args string) (string, error) {
		var p struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal([]byte(args), &p); err != nil {
			return "", err
		}
		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, "bash", "-lc", p.Command)
		cmd.Dir = ws.Root()
		out, err := cmd.CombinedOutput()
		if err != nil {
			return string(out) + "\n" + err.Error(), nil
		}
		return string(out), nil
	})

	reg.register("write_file", true, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name:        "write_file",
			Description: "Write full contents to a file (requires approval).",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"},"content":{"type":"string"}},"required":["path","content"]}`),
		},
	}, func(ctx context.Context, args string) (string, error) {
		var p struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal([]byte(args), &p); err != nil {
			return "", err
		}
		if err := ws.WriteFile(p.Path, p.Content); err != nil {
			return "", err
		}
		return fmt.Sprintf("wrote %s (%d bytes)", p.Path, len(p.Content)), nil
	})

	return reg
}

func (r *ToolRegistry) register(name string, mutating bool, schema router.Tool, fn toolFunc) {
	r.funcs[name] = fn
	r.meta[name] = toolMeta{mutating: mutating, schema: schema}
}

// Schemas returns OpenAI tool definitions.
func (r ToolRegistry) Schemas() []router.Tool {
	out := make([]router.Tool, 0, len(r.meta))
	for _, m := range r.meta {
		out = append(out, m.schema)
	}
	return out
}

// IsMutating reports whether the tool mutates the system.
func (r ToolRegistry) IsMutating(name string) bool {
	m, ok := r.meta[name]
	return ok && m.mutating
}

// Execute runs a tool by name with a JSON args object.
func (r ToolRegistry) Execute(ctx context.Context, name, args string) (string, error) {
	fn, ok := r.funcs[name]
	if !ok {
		return "", fmt.Errorf("unknown tool %q", name)
	}
	return fn(ctx, args)
}
