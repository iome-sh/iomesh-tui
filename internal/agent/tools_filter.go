package agent

import (
	"github.com/iome-sh/iomesh-tui/internal/router"
)

// FilterTools returns a registry view limited to allowed tool names.
func (r ToolRegistry) FilterTools(allow []string) ToolRegistry {
	set := map[string]bool{}
	for _, n := range allow {
		set[n] = true
	}
	out := ToolRegistry{
		ws:    r.ws,
		funcs: map[string]toolFunc{},
		meta:  map[string]toolMeta{},
	}
	for name, fn := range r.funcs {
		if !set[name] {
			continue
		}
		out.funcs[name] = fn
		out.meta[name] = r.meta[name]
	}
	return out
}

// ToolNames returns registered tool names.
func (r ToolRegistry) ToolNames() []string {
	out := make([]string, 0, len(r.meta))
	for n := range r.meta {
		out = append(out, n)
	}
	return out
}

// CloneSchemas is a convenience for tests.
func CloneSchemas(in []router.Tool) []router.Tool {
	out := make([]router.Tool, len(in))
	copy(out, in)
	return out
}
