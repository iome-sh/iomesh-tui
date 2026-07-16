package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iome-sh/iomesh-tui/internal/router"
	"github.com/iome-sh/iomesh-tui/internal/subagent"
)

// RegisterSubagentTools adds spawn / batch-parallel / wait tools when mgr is enabled.
func (r *ToolRegistry) RegisterSubagentTools(mgr *subagent.Manager) {
	if mgr == nil || !mgr.Enabled() {
		return
	}
	r.register("spawn_subagent", false, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name:        "spawn_subagent",
			Description: "Spawn one child agent. Types: general-purpose, explore (no edits), plan (no edits). Prefer spawn_subagents for parallel fan-out. Use background=true to free the parent immediately.",
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

	// Maximum parallel fan-out: one tool call → many concurrent children.
	r.register("spawn_subagents", false, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name: "spawn_subagents",
			Description: fmt.Sprintf(
				"Spawn MANY child agents in parallel (up to max_concurrent=%d running, max_batch=%d tasks). Prefer this over serial spawn_subagent when tasks are independent. Set wait=true to block until all finish; wait=false returns ids immediately for wait_subagents.",
				mgr.MaxConcurrent(), mgr.MaxBatch(),
			),
			Parameters: json.RawMessage(`{
  "type": "object",
  "properties": {
    "tasks": {
      "type": "array",
      "description": "Independent subagent tasks to run in parallel",
      "items": {
        "type": "object",
        "properties": {
          "prompt": {"type": "string"},
          "description": {"type": "string"},
          "subagent_type": {"type": "string", "enum": ["general-purpose", "explore", "plan"]},
          "capability_mode": {"type": "string", "enum": ["read-only", "read-write", "execute", "all"]},
          "cwd": {"type": "string"}
        },
        "required": ["prompt"]
      }
    },
    "wait": {"type": "boolean", "description": "If true, wait for all tasks to finish before returning summaries"},
    "default_subagent_type": {"type": "string", "enum": ["general-purpose", "explore", "plan"]}
  },
  "required": ["tasks"]
}`),
		},
	}, func(ctx context.Context, args string) (string, error) {
		var p struct {
			Tasks []struct {
				Prompt         string `json:"prompt"`
				Description    string `json:"description"`
				SubagentType   string `json:"subagent_type"`
				CapabilityMode string `json:"capability_mode"`
				CWD            string `json:"cwd"`
			} `json:"tasks"`
			Wait                bool   `json:"wait"`
			DefaultSubagentType string `json:"default_subagent_type"`
		}
		if err := json.Unmarshal([]byte(args), &p); err != nil {
			return "", err
		}
		defType := subagent.Type(p.DefaultSubagentType)
		if defType == "" {
			defType = subagent.TypeExplore
		}
		specs := make([]subagent.Spec, 0, len(p.Tasks))
		for _, t := range p.Tasks {
			st := subagent.Type(t.SubagentType)
			if st == "" {
				st = defType
			}
			specs = append(specs, subagent.Spec{
				Prompt:         t.Prompt,
				Description:    t.Description,
				SubagentType:   st,
				CapabilityMode: subagent.CapabilityMode(t.CapabilityMode),
				CWD:            t.CWD,
				Background:     true,
			})
		}
		batch, err := mgr.SpawnMany(ctx, specs, subagent.SpawnManyOptions{Wait: p.Wait})
		if err != nil {
			return "", err
		}
		b, _ := json.MarshalIndent(batch, "", "  ")
		return string(b), nil
	})

	r.register("get_subagent_output", false, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name:        "get_subagent_output",
			Description: "Get status/summary for one subagent id. Optionally wait until completion.",
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

	r.register("wait_subagents", false, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name:        "wait_subagents",
			Description: "Wait for multiple subagent ids (from spawn_subagents with wait=false). Returns final summaries in order.",
			Parameters: json.RawMessage(`{
  "type": "object",
  "properties": {
    "ids": {"type": "array", "items": {"type": "string"}, "description": "Subagent ids to wait on"}
  },
  "required": ["ids"]
}`),
		},
	}, func(ctx context.Context, args string) (string, error) {
		var p struct {
			IDs []string `json:"ids"`
		}
		if err := json.Unmarshal([]byte(args), &p); err != nil {
			return "", err
		}
		res, err := mgr.WaitAll(ctx, p.IDs)
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
