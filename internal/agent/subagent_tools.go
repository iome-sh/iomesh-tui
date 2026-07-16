package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iome-sh/iomesh-tui/internal/router"
	"github.com/iome-sh/iomesh-tui/internal/subagent"
)

// RegisterSubagentTools adds spawn_subagent and get_subagent_output when mgr is enabled.
func (r *ToolRegistry) RegisterSubagentTools(mgr *subagent.Manager) {
	if mgr == nil || !mgr.Enabled() {
		return
	}
	r.register("spawn_subagent", false, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name:        "spawn_subagent",
			Description: "Spawn a child agent with its own context. Types: general-purpose, explore (read/search/shell, no edits), plan (implementation plan, no edits). Use background=true for parallel work; poll with get_subagent_output.",
			Parameters: json.RawMessage(`{
  "type": "object",
  "properties": {
    "prompt": {"type": "string", "description": "Full task prompt for the child"},
    "description": {"type": "string", "description": "Short 3-5 word label"},
    "subagent_type": {"type": "string", "enum": ["general-purpose", "explore", "plan"]},
    "background": {"type": "boolean"},
    "capability_mode": {"type": "string", "enum": ["read-only", "read-write", "execute", "all"]},
    "isolation": {"type": "string", "enum": ["none", "worktree"]},
    "resume_from": {"type": "string", "description": "Completed subagent id to continue from"},
    "cwd": {"type": "string"}
  },
  "required": ["prompt"]
}`),
		},
	}, func(ctx context.Context, args string) (string, error) {
		var p struct {
			Prompt         string `json:"prompt"`
			Description    string `json:"description"`
			SubagentType   string `json:"subagent_type"`
			Background     bool   `json:"background"`
			CapabilityMode string `json:"capability_mode"`
			Isolation      string `json:"isolation"`
			ResumeFrom     string `json:"resume_from"`
			CWD            string `json:"cwd"`
		}
		if err := json.Unmarshal([]byte(args), &p); err != nil {
			return "", err
		}
		res, err := mgr.Spawn(ctx, subagent.Spec{
			Prompt:         p.Prompt,
			Description:    p.Description,
			SubagentType:   subagent.Type(p.SubagentType),
			Background:     p.Background,
			CapabilityMode: subagent.CapabilityMode(p.CapabilityMode),
			Isolation:      subagent.Isolation(p.Isolation),
			ResumeFrom:     p.ResumeFrom,
			CWD:            p.CWD,
		})
		if err != nil {
			return "", err
		}
		b, _ := json.MarshalIndent(res, "", "  ")
		return string(b), nil
	})

	r.register("get_subagent_output", false, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name:        "get_subagent_output",
			Description: "Get status/summary for a subagent id from spawn_subagent (background). Optionally wait until completion.",
			Parameters: json.RawMessage(`{
  "type": "object",
  "properties": {
    "id": {"type": "string"},
    "wait": {"type": "boolean", "description": "Block until completed/failed"}
  },
  "required": ["id"]
}`),
		},
	}, func(ctx context.Context, args string) (string, error) {
		var p struct {
			ID   string `json:"id"`
			Wait bool   `json:"wait"`
		}
		if err := json.Unmarshal([]byte(args), &p); err != nil {
			return "", err
		}
		var (
			res subagent.Result
			err error
		)
		if p.Wait {
			res, err = mgr.Wait(ctx, p.ID)
		} else {
			res, err = mgr.Get(p.ID)
		}
		if err != nil {
			return "", err
		}
		b, _ := json.MarshalIndent(res, "", "  ")
		return string(b), nil
	})
}

// childRunner adapts Runtime for subagent.Manager.
type childRunner struct {
	rt *Runtime
}

func (c childRunner) Run(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	// Replace system message with type-specific prompt.
	if len(c.rt.messages) > 0 && c.rt.messages[0].Role == "system" {
		c.rt.messages[0].Content = systemPrompt + "\n\nWorkspace: " + c.rt.ws.Root()
	} else {
		c.rt.messages = append([]router.Message{{Role: "system", Content: systemPrompt}}, c.rt.messages...)
	}
	// Force routine routing for cost control on children.
	c.rt.cfg.DefaultTaskType = router.TaskSubagent
	c.rt.cfg.DefaultComplexity = router.ComplexityRoutine

	text, err := c.rt.RunTurn(ctx, userPrompt, nil)
	if err != nil {
		return text, err
	}
	// Prefer last assistant message; fall back to accumulated text.
	summary := strings.TrimSpace(text)
	if summary == "" {
		for i := len(c.rt.messages) - 1; i >= 0; i-- {
			if c.rt.messages[i].Role == "assistant" && c.rt.messages[i].Content != "" {
				summary = c.rt.messages[i].Content
				break
			}
		}
	}
	if summary == "" {
		summary = "(subagent finished with empty summary)"
	}
	return summary, nil
}

// newChildFactory returns a RunnerFactory bound to parent router/mesh/logger.
func (rt *Runtime) newChildFactory() subagent.RunnerFactory {
	return func(ctx context.Context, sp subagent.SpawnParams) (subagent.Runner, error) {
		childCfg := Config{
			Mode:              ModeHeadless,
			Workspace:         sp.Workspace,
			Yolo:              sp.ParentYolo,
			DefaultTaskType:   router.TaskSubagent,
			DefaultComplexity: router.ComplexityRoutine,
			// Prevent nested manager recursion unless AllowSpawn.
			SubagentsEnabled: sp.AllowSpawn,
			MaxSubagentDepth: 0, // children use parent manager depth checks
		}
		child, err := New(childCfg, rt.router, rt.mesh, rt.logger)
		if err != nil {
			return nil, err
		}
		// Rebuild tools for capability filter (no nested subagents unless allowed).
		tools := DefaultTools(child.ws)
		allow := subagent.EffectiveTools(sp.AllowWrite, sp.AllowShell, sp.AllowSpawn)
		child.tools = tools.FilterTools(allow)
		if sp.AllowSpawn && rt.subagents != nil {
			// Nested spawns share parent manager with elevated depth — not wired in v1;
			// AllowSpawn is false for builtins.
			child.tools.RegisterSubagentTools(rt.subagents)
		}
		if sp.Spec.ModelOverride != "" {
			if err := child.router.SetOverride(sp.Spec.ModelOverride); err != nil {
				return nil, fmt.Errorf("child model: %w", err)
			}
		}
		return childRunner{rt: child}, nil
	}
}
