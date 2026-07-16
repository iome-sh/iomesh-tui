// Package subagent implements child-session orchestration for the coding agent:
// independent context windows, typed agent roles (explore/plan/general-purpose),
// capability modes, background completion, and optional isolation.
package subagent

import (
	"time"
)

// Type selects the built-in agent role for a child session.
type Type string

const (
	TypeGeneralPurpose Type = "general-purpose"
	TypeExplore        Type = "explore"
	TypePlan           Type = "plan"
)

// CapabilityMode is a coarse filter on the child toolset.
type CapabilityMode string

const (
	CapabilityReadOnly  CapabilityMode = "read-only"
	CapabilityReadWrite CapabilityMode = "read-write"
	CapabilityExecute   CapabilityMode = "execute"
	CapabilityAll       CapabilityMode = "all"
)

// Isolation controls whether the child shares the parent workspace.
type Isolation string

const (
	IsolationNone     Isolation = "none"
	IsolationWorktree Isolation = "worktree"
)

// Status is the lifecycle state of a subagent.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

// Spec is the spawn request from the parent agent (or tests).
type Spec struct {
	Prompt         string         `json:"prompt"`
	Description    string         `json:"description"`
	SubagentType   Type           `json:"subagent_type"`
	Background     bool           `json:"background"`
	CapabilityMode CapabilityMode `json:"capability_mode"`
	Isolation      Isolation      `json:"isolation"`
	ResumeFrom     string         `json:"resume_from"`
	CWD            string         `json:"cwd"`
	// ModelOverride pins a logical router model for the child (optional).
	ModelOverride string `json:"model_override"`
	// MaxDepth is set by the manager; children cannot spawn beyond limit.
	MaxDepth int `json:"-"`
	// Depth is the current nesting level (0 = parent).
	Depth int `json:"-"`
}

// Record is a tracked subagent instance.
type Record struct {
	ID             string
	Spec           Spec
	Status         Status
	StartedAt      time.Time
	FinishedAt     time.Time
	Summary        string
	Error          string
	WorktreePath   string
	ParentSession  string
	MessagesCopied int // for resume_from
}

// Result is returned to the parent tool call.
type Result struct {
	ID           string `json:"id"`
	Status       Status `json:"status"`
	Summary      string `json:"summary,omitempty"`
	Error        string `json:"error,omitempty"`
	WorktreePath string `json:"worktree_path,omitempty"`
	Description  string `json:"description,omitempty"`
	SubagentType Type   `json:"subagent_type,omitempty"`
	DurationMS   int64  `json:"duration_ms,omitempty"`
}

// Definition describes a built-in agent type.
type Definition struct {
	Type         Type
	Description  string
	SystemPrompt string
	// DefaultCapability applies when Spec.CapabilityMode is empty.
	DefaultCapability CapabilityMode
	// AllowShell when true includes run_shell (subject to capability filter).
	AllowShell bool
	// AllowWrite when true includes write_file (subject to capability filter).
	AllowWrite bool
	// AllowSpawn when true allows nested spawn_subagent (usually false).
	AllowSpawn bool
}

// Builtins returns the catalog of built-in agent types.
func Builtins() map[Type]Definition {
	return map[Type]Definition{
		TypeGeneralPurpose: {
			Type:              TypeGeneralPurpose,
			Description:       "Full-capability agent for any delegated task.",
			SystemPrompt:      generalPurposePrompt,
			DefaultCapability: CapabilityAll,
			AllowShell:        true,
			AllowWrite:        true,
			AllowSpawn:        false,
		},
		TypeExplore: {
			Type:              TypeExplore,
			Description:       "Research agent: read, search, shell — no file edits.",
			SystemPrompt:      explorePrompt,
			DefaultCapability: CapabilityExecute, // read + shell, no write
			AllowShell:        true,
			AllowWrite:        false,
			AllowSpawn:        false,
		},
		TypePlan: {
			Type:              TypePlan,
			Description:       "Planning agent: explore and produce an implementation plan — no edits.",
			SystemPrompt:      planPrompt,
			DefaultCapability: CapabilityExecute,
			AllowShell:        true,
			AllowWrite:        false,
			AllowSpawn:        false,
		},
	}
}

const generalPurposePrompt = `You are a general-purpose subagent of I/O Mesh TUI.
Complete the delegated task thoroughly. Use tools as needed. When finished, reply with a concise summary of findings and any file paths touched.
Do not ask the user questions — decide and act within the workspace.`

const explorePrompt = `You are an explore subagent. Investigate the codebase with read/search/shell tools only.
Do NOT edit files. Return a structured summary:
1) What you searched
2) Key files and findings (with paths)
3) Open questions / risks
Be thorough but prefer evidence over speculation.`

const planPrompt = `You are a plan-mode subagent. Explore the codebase (read-only + shell) and produce a structured implementation plan.
Do NOT edit files. Output:
## Goal
## Current state
## Approach
## Steps (ordered)
## Risks
## Files likely touched
Keep the plan actionable for a follow-on implementer.`
