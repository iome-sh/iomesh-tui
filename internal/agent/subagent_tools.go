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
			Description: "Spawn one child agent. Types: general-purpose, explore (no edits), plan (no edits). Prefer spawn_subagents for parallel fan-out. Use background=true to free the parent. isolation=worktree runs the child in a detached git worktree under .iomesh/worktrees/<id>.",
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
				"MAXIMUM PARALLEL fan-out: spawn up to max_concurrent=%d running (max_batch=%d tasks). Joins wait concurrently. For isolated parallel edits use default_isolation=worktree and optionally apply_after=true (requires wait=true) to merge all worktrees in parallel.",
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
          "isolation": {"type": "string", "enum": ["none", "worktree"]},
          "cwd": {"type": "string"}
        },
        "required": ["prompt"]
      }
    },
    "wait": {"type": "boolean", "description": "If true, wait for ALL tasks concurrently before returning"},
    "apply_after": {"type": "boolean", "description": "After wait, parallel-apply isolation worktrees into parent (requires wait=true; mutating)"},
    "remove_after_apply": {"type": "boolean", "description": "With apply_after, delete worktrees after successful apply"},
    "default_subagent_type": {"type": "string", "enum": ["general-purpose", "explore", "plan"]},
    "default_isolation": {"type": "string", "enum": ["none", "worktree"]}
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
				Isolation      string `json:"isolation"`
				CWD            string `json:"cwd"`
			} `json:"tasks"`
			Wait                bool   `json:"wait"`
			ApplyAfter          bool   `json:"apply_after"`
			RemoveAfterApply    bool   `json:"remove_after_apply"`
			DefaultSubagentType string `json:"default_subagent_type"`
			DefaultIsolation    string `json:"default_isolation"`
		}
		if err := json.Unmarshal([]byte(args), &p); err != nil {
			return "", err
		}
		// Note: apply_after performs parent writes; prefer explicit apply_worktrees when
		// approval UX is required. spawn_subagents remains non-mutating for the outer tool
		// classification; operators should use --yolo for unattended apply_after.
		defType := subagent.Type(p.DefaultSubagentType)
		if defType == "" {
			defType = subagent.TypeExplore
		}
		defIso := subagent.Isolation(p.DefaultIsolation)
		if defIso == "" {
			defIso = subagent.IsolationNone
		}
		specs := make([]subagent.Spec, 0, len(p.Tasks))
		for _, t := range p.Tasks {
			st := subagent.Type(t.SubagentType)
			if st == "" {
				st = defType
			}
			iso := subagent.Isolation(t.Isolation)
			if iso == "" {
				iso = defIso
			}
			specs = append(specs, subagent.Spec{
				Prompt:         t.Prompt,
				Description:    t.Description,
				SubagentType:   st,
				CapabilityMode: subagent.CapabilityMode(t.CapabilityMode),
				Isolation:      iso,
				CWD:            t.CWD,
				Background:     true,
			})
		}
		batch, err := mgr.SpawnMany(ctx, specs, subagent.SpawnManyOptions{
			Wait:             p.Wait,
			ApplyAfter:       p.ApplyAfter,
			RemoveAfterApply: p.RemoveAfterApply,
		})
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

	// Worktree apply/merge dance (mutating: copies into parent workspace).
	r.register("diff_worktree", false, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name:        "diff_worktree",
			Description: "Show git status/diff for an isolation worktree. Pass subagent id or worktree path from isolation=worktree results.",
			Parameters: json.RawMessage(`{
  "type": "object",
  "properties": {
    "id": {"type": "string", "description": "Subagent id or worktree path"}
  },
  "required": ["id"]
}`),
		},
	}, func(ctx context.Context, args string) (string, error) {
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal([]byte(args), &p); err != nil {
			return "", err
		}
		return mgr.DiffWorktree(ctx, p.ID)
	})

	r.register("list_worktrees", false, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name:        "list_worktrees",
			Description: "List isolation worktrees under .iomesh/worktrees (or configured worktree_base).",
			Parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
		},
	}, func(ctx context.Context, args string) (string, error) {
		list, err := mgr.ListWorktrees()
		if err != nil {
			return "", err
		}
		b, _ := json.MarshalIndent(list, "", "  ")
		return string(b), nil
	})

	r.register("apply_worktree", true, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name:        "apply_worktree",
			Description: "Merge file changes from an isolation worktree into the parent workspace (path-jailed copy). Use after a general-purpose subagent with isolation=worktree. Requires approval/--yolo. Optional remove=true deletes the worktree after apply.",
			Parameters: json.RawMessage(`{
  "type": "object",
  "properties": {
    "id": {"type": "string", "description": "Subagent id or worktree path"},
    "remove": {"type": "boolean", "description": "Remove worktree after successful apply"}
  },
  "required": ["id"]
}`),
		},
	}, func(ctx context.Context, args string) (string, error) {
		var p struct {
			ID     string `json:"id"`
			Remove bool   `json:"remove"`
		}
		if err := json.Unmarshal([]byte(args), &p); err != nil {
			return "", err
		}
		res, err := mgr.ApplyWorktree(ctx, p.ID, p.Remove)
		if err != nil {
			return "", err
		}
		b, _ := json.MarshalIndent(res, "", "  ")
		return string(b), nil
	})

	r.register("apply_worktrees", true, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name:        "apply_worktrees",
			Description: "MAXIMUM PARALLEL merge: apply many isolation worktrees into the parent concurrently (bounded by max_concurrent). Requires approval/--yolo. Prefer after spawn_subagents with default_isolation=worktree and wait=true.",
			Parameters: json.RawMessage(`{
  "type": "object",
  "properties": {
    "ids": {"type": "array", "items": {"type": "string"}, "description": "Subagent ids or worktree paths"},
    "remove": {"type": "boolean", "description": "Remove each worktree after successful apply"}
  },
  "required": ["ids"]
}`),
		},
	}, func(ctx context.Context, args string) (string, error) {
		var p struct {
			IDs    []string `json:"ids"`
			Remove bool     `json:"remove"`
		}
		if err := json.Unmarshal([]byte(args), &p); err != nil {
			return "", err
		}
		res, err := mgr.ApplyMany(ctx, p.IDs, p.Remove)
		if err != nil {
			return "", err
		}
		b, _ := json.MarshalIndent(res, "", "  ")
		return string(b), nil
	})

	r.register("remove_worktree", true, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name:        "remove_worktree",
			Description: "Delete an isolation git worktree without applying. Requires approval/--yolo.",
			Parameters: json.RawMessage(`{
  "type": "object",
  "properties": {
    "id": {"type": "string", "description": "Subagent id or worktree path"}
  },
  "required": ["id"]
}`),
		},
	}, func(ctx context.Context, args string) (string, error) {
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal([]byte(args), &p); err != nil {
			return "", err
		}
		if err := mgr.RemoveWorktree(ctx, p.ID); err != nil {
			return "", err
		}
		return `{"removed":true,"id":` + fmt.Sprintf("%q", p.ID) + `}`, nil
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
