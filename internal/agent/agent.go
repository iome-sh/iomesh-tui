// Package agent implements the coding-agent runtime loop: prompt → LLM
// (via router) → tool calls → workspace effects, with hooks for TUI,
// headless, and ACP front-ends.
package agent

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/iomesh"
	"github.com/iome-sh/iomesh-tui/internal/mcp"
	"github.com/iome-sh/iomesh-tui/internal/router"
	"github.com/iome-sh/iomesh-tui/internal/skills"
	"github.com/iome-sh/iomesh-tui/internal/subagent"
	"github.com/iome-sh/iomesh-tui/internal/workspace"
)

// Mode selects the front-end integration style.
type Mode string

const (
	ModeTUI      Mode = "tui"
	ModeHeadless Mode = "headless"
	ModeACP      Mode = "acp"
)

// Config configures a Runtime.
type Config struct {
	Mode      Mode
	Workspace string
	Yolo      bool // auto-approve tools
	// Task hints for the router (overridable per turn).
	DefaultTaskType   router.TaskType
	DefaultComplexity router.Complexity
	// SubagentsEnabled registers spawn_subagent tools (default true when unset via New defaults).
	SubagentsEnabled bool
	// MaxSubagentDepth limits nesting (default 2).
	MaxSubagentDepth int
	// MaxSubagentConcurrent limits parallel children (default 16).
	MaxSubagentConcurrent int
	// MaxSubagentBatch caps spawn_subagents array size (default 32).
	MaxSubagentBatch int
	// WorktreeBase relative path under workspace for git worktrees.
	WorktreeBase string
	// WorktreeAutoRemove deletes successful isolation worktrees.
	WorktreeAutoRemove bool
}

// Runtime is the agent loop shared by TUI / headless / ACP.
type Runtime struct {
	cfg       Config
	router    *router.Router
	mesh      *iomesh.Client
	ws        *workspace.Workspace
	logger    *slog.Logger
	tools     ToolRegistry
	subagents *subagent.Manager
	skills    *skills.Catalog
	mcp       *mcp.Manager
	memory    MemoryConfig

	// Session transcript and persistence hooks.
	messages  []router.Message
	sessionID string
	autoSave  bool
	// sessionSeq is a monotonic counter for dual-write memory_ingest envelopes.
	sessionSeq atomic.Int64

	// s1069 short-TTL sync retrieve cache + last latency (fail-open, not Memory GA).
	memoryCache                *memoryRecallCache
	lastMemoryRetrieveMS       atomic.Int64
	lastMemoryRetrieveCacheHit atomic.Bool

	// Permission / approval for mutating tools (subagent apply, shell, write, …).
	mu           sync.Mutex
	approver     Approver
	sessionAllow map[string]bool
}

// New constructs a Runtime. mesh may be nil.
func New(cfg Config, r *router.Router, mesh *iomesh.Client, logger *slog.Logger) (*Runtime, error) {
	if r == nil {
		return nil, fmt.Errorf("agent: router is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	ws, err := workspace.Open(cfg.Workspace)
	if err != nil {
		return nil, err
	}
	rt := &Runtime{
		cfg:    cfg,
		router: r,
		mesh:   mesh,
		ws:     ws,
		logger: logger,
		tools:  DefaultTools(ws),
		messages: []router.Message{
			{Role: "system", Content: defaultSystemPrompt(ws.Root())},
		},
	}
	if cfg.SubagentsEnabled {
		maxDepth := cfg.MaxSubagentDepth
		if maxDepth <= 0 {
			maxDepth = subagent.DefaultMaxDepth
		}
		maxConc := cfg.MaxSubagentConcurrent
		if maxConc <= 0 {
			maxConc = subagent.DefaultMaxConcurrent
		}
		maxBatch := cfg.MaxSubagentBatch
		if maxBatch <= 0 {
			maxBatch = subagent.DefaultMaxBatch
		}
		// Factory closes over rt so children share router/mesh/logger.
		rt.subagents = subagent.NewManager(subagent.Config{
			Enabled:            true,
			MaxConcurrent:      maxConc,
			MaxDepth:           maxDepth,
			MaxBatch:           maxBatch,
			Workspace:          ws.Root(),
			Yolo:               cfg.Yolo,
			WorktreeAutoRemove: cfg.WorktreeAutoRemove,
		}, rt.newChildFactory(), logger)
		// Prefer real git worktrees when available; Nop otherwise.
		if gw, ok := subagent.LookupGit().(*subagent.GitWorktree); ok {
			if cfg.WorktreeBase != "" {
				gw.BaseDir = cfg.WorktreeBase
			}
			rt.subagents.SetWorktreeBackend(gw)
		}
		rt.tools.RegisterSubagentTools(rt.subagents)
	}
	return rt, nil
}

// Subagents returns the manager (may be nil if disabled).
func (rt *Runtime) Subagents() *subagent.Manager { return rt.subagents }

// Router exposes the LLM router (for /model and cost estimates).
func (rt *Runtime) Router() *router.Router { return rt.router }

// Workspace returns the bound workspace.
func (rt *Runtime) Workspace() *workspace.Workspace { return rt.ws }

// Skills returns the loaded skill catalog (may be nil).
func (rt *Runtime) Skills() *skills.Catalog { return rt.skills }

// MCP returns the MCP manager (may be nil).
func (rt *Runtime) MCP() *mcp.Manager { return rt.mcp }

// Mesh returns the I/O Mesh client (may be nil or disabled).
func (rt *Runtime) Mesh() *iomesh.Client { return rt.mesh }

// MeshUsage returns the process-local LLM usage snapshot (empty if no mesh client).
func (rt *Runtime) MeshUsage() iomesh.UsageSnapshot {
	if rt == nil || rt.mesh == nil {
		return iomesh.UsageSnapshot{}
	}
	return rt.mesh.Usage()
}

// AttachMeshTools registers list_mesh_catalog / mesh_status when catalog plane is enabled.
func (rt *Runtime) AttachMeshTools() {
	if rt == nil || rt.mesh == nil || !rt.mesh.CatalogEnabled() {
		return
	}
	rt.tools.RegisterMeshTools(rt.mesh)
	rt.appendSystemNote("iomesh-tools", "I/O Mesh tools: list_mesh_catalog, get_mesh_catalog_product, mesh_status. Catalog tries broker then portal federation. Fail-open when unavailable.")
}

// AttachSkills registers list/read skill tools and appends a catalog block to the system prompt.
func (rt *Runtime) AttachSkills(cat *skills.Catalog) {
	if rt == nil || cat == nil || cat.Len() == 0 {
		return
	}
	rt.skills = cat
	rt.tools.RegisterSkillsTools(cat)
	if block := cat.PromptBlock(); block != "" {
		rt.appendSystemNote("skills", block)
	}
}

// AttachMCP registers mcp__* tools from connected servers.
// Also injects residual-honest integrations guidance (s1251) so the agent uses
// list → plan → portal HITL without inventing install green, and advanced
// memory guidance (s1291) so multi-hop / HITL supersede / ops pulse stay opt-in.
func (rt *Runtime) AttachMCP(mgr *mcp.Manager) {
	if rt == nil || mgr == nil || mgr.Len() == 0 {
		return
	}
	rt.mcp = mgr
	rt.tools.RegisterMCPTools(mgr)
	n := 0
	for range mgr.Bindings() {
		n++
	}
	rt.appendSystemNote("mcp", fmt.Sprintf("MCP: %d server(s), %d tool(s) available as mcp__<server>__<tool> (mutating tools require approval).", mgr.Len(), n))
	// s1251: residual-honest connector integrations workflow for the agent.
	rt.appendSystemNote("integrations", IntegrationsAgentGuidanceNote())
	// s1291: residual-honest advanced memory agent path (opt-in surfaces only).
	rt.appendSystemNote("memory-advanced", MemoryAdvancedAgentGuidanceNote())
}

// Close releases MCP subprocesses and other runtime resources.
func (rt *Runtime) Close() error {
	if rt == nil {
		return nil
	}
	if rt.mcp != nil {
		return rt.mcp.Close()
	}
	return nil
}

func (rt *Runtime) appendSystemNote(tag, body string) {
	if len(rt.messages) == 0 {
		return
	}
	// Prefer first system message.
	if rt.messages[0].Role == "system" {
		rt.messages[0].Content += "\n\n<" + tag + ">\n" + body + "\n</" + tag + ">"
		return
	}
	rt.messages = append([]router.Message{{
		Role: "system", Content: "<" + tag + ">\n" + body + "\n</" + tag + ">",
	}}, rt.messages...)
}

// Messages returns a copy of the transcript.
func (rt *Runtime) Messages() []router.Message {
	out := make([]router.Message, len(rt.messages))
	copy(out, rt.messages)
	return out
}

// RunTurn executes one user turn: optional mesh context, LLM call, tool loop.
// onEvent receives streaming text and tool lifecycle events for the UI.
func (rt *Runtime) RunTurn(ctx context.Context, userText string, onEvent func(Event)) (string, error) {
	if onEvent == nil {
		onEvent = func(Event) {}
	}

	// Optional I/O Mesh context plane + catalog composition (fail-open).
	if rt.mesh != nil {
		if snippet := rt.mesh.ContextSnippet(ctx, rt.ws.Root(), userText); snippet != "" {
			rt.messages = append(rt.messages, router.Message{
				Role:    "system",
				Content: "<iomesh-context>\n" + snippet + "\n</iomesh-context>",
			})
			onEvent(Event{Type: EventMeshContext, Text: "injected I/O Mesh context"})
		}
		if rt.mesh.InjectCatalog() {
			cat := rt.mesh.ListCatalog(ctx, userText)
			if block := iomesh.CatalogSnippet(cat, 12); block != "" {
				rt.messages = append(rt.messages, router.Message{
					Role:    "system",
					Content: "<iomesh-catalog>\n" + block + "\n</iomesh-catalog>",
				})
				onEvent(Event{Type: EventMeshContext, Text: fmt.Sprintf("injected I/O Mesh catalog (%d products, source=%s)", len(cat.Products), cat.Source)})
			}
		}
	}

	// Optional Memory Palace auto-recall: sync HTTP then MCP (fail-open).
	rt.maybeInjectMemoryRecall(ctx, userText, onEvent)

	rt.messages = append(rt.messages, router.Message{Role: "user", Content: userText})

	complexity := rt.cfg.DefaultComplexity
	taskType := rt.cfg.DefaultTaskType
	if taskType == "" {
		taskType = router.TaskRoutine
	}
	// Lightweight heuristic: planning keywords escalate complexity.
	lower := strings.ToLower(userText)
	if strings.Contains(lower, "plan") || strings.Contains(lower, "design") || strings.Contains(lower, "architect") {
		complexity = router.ComplexityPlan
		taskType = router.TaskPlan
	}
	if strings.Contains(lower, "production") || strings.Contains(lower, "migrate") || strings.Contains(lower, "security") {
		complexity = router.ComplexityHighStakes
	}

	tools := rt.tools.Schemas()
	estTokens := estimateTokens(rt.messages)

	var final strings.Builder
	// Tool loop (bounded).
	const maxToolRounds = 16
	for round := 0; round < maxToolRounds; round++ {
		req := router.ChatRequest{
			Messages:  rt.messages,
			Tools:     tools,
			MaxTokens: 8192,
		}
		params := router.SelectParams{
			TaskType:        taskType,
			EstimatedTokens: estTokens,
			Complexity:      complexity,
		}

		selected := rt.router.SelectModel(params)
		onEvent(Event{
			Type:  EventModelSelected,
			Model: selected,
		})

		resp, meta, err := rt.router.ExecuteStreamWithFallback(ctx, req, params, func(d router.StreamDelta) error {
			if d.Content != "" {
				onEvent(Event{Type: EventTextDelta, Text: d.Content, Model: selected})
			}
			if d.ReasoningContent != "" {
				onEvent(Event{Type: EventThinkingDelta, Text: d.ReasoningContent})
			}
			return nil
		})
		// Fall back to non-stream if stream failed, unsupported, or returned an empty
		// assistant turn (common when a server returns plain JSON without SSE framing).
		needNonStream := err != nil || len(resp.Choices) == 0
		if !needNonStream {
			m := resp.Choices[0].Message
			if m.Content == "" && len(m.ToolCalls) == 0 {
				needNonStream = true
			}
		}
		if needNonStream {
			resp, meta, err = rt.router.ExecuteWithFallback(ctx, req, params)
			if err != nil {
				return final.String(), err
			}
			if len(resp.Choices) > 0 && resp.Choices[0].Message.Content != "" {
				onEvent(Event{Type: EventTextDelta, Text: resp.Choices[0].Message.Content, Model: meta.ModelName})
			}
		}

		onEvent(Event{
			Type:     EventLLMDone,
			Model:    meta.ModelName,
			CostUSD:  meta.EstimatedUSD,
			Duration: meta.Duration,
			Tokens:   resp.Usage.TotalTokens,
		})

		if len(resp.Choices) == 0 {
			return final.String(), fmt.Errorf("empty llm response")
		}
		msg := resp.Choices[0].Message
		rt.messages = append(rt.messages, msg)
		final.WriteString(msg.Content)

		if len(msg.ToolCalls) == 0 {
			// Opt-in Palace auto-ingest after final assistant answer (fail-open).
			rt.maybeAutoIngest(ctx, userText, final.String(), onEvent)
			return final.String(), nil
		}

		for _, tc := range msg.ToolCalls {
			onEvent(Event{Type: EventToolStart, Tool: tc.Function.Name, Text: tc.Function.Arguments})
			// Optional remote mesh policy (Rego/OPA via broker). Fail-open unless enforce+deny.
			if rt.mesh != nil && rt.mesh.PolicyEnabled() {
				dec := rt.mesh.EvaluatePolicy(ctx, iomesh.PolicyInput{
					Action: "tool." + tc.Function.Name,
					Tool:   tc.Function.Name,
					Attributes: map[string]any{
						"mutating": rt.tools.IsMutating(tc.Function.Name),
						"args":     truncate(tc.Function.Arguments, 500),
					},
				})
				onEvent(Event{Type: EventMeshPolicy, Tool: tc.Function.Name, Text: dec.Summary()})
				if dec.ShouldBlockTool() {
					msg := "tool denied by I/O Mesh policy (" + dec.Summary() + ")"
					onEvent(Event{Type: EventToolDenied, Tool: tc.Function.Name, Text: msg})
					rt.messages = append(rt.messages, router.Message{
						Role:       "tool",
						ToolCallID: tc.ID,
						Content:    msg,
					})
					continue
				}
			}
			if rt.tools.IsMutating(tc.Function.Name) {
				switch rt.decideApproval(ctx, tc.Function.Name, tc.Function.Arguments) {
				case ApprovalDeny:
					msg := "tool denied: user approval required (use --yolo, or approve interactively in TUI)"
					onEvent(Event{Type: EventToolDenied, Tool: tc.Function.Name, Text: msg})
					rt.messages = append(rt.messages, router.Message{
						Role:       "tool",
						ToolCallID: tc.ID,
						Content:    msg,
					})
					continue
				case ApprovalOnce, ApprovalAlways:
					// proceed
				}
			}
			result, err := rt.tools.Execute(ctx, tc.Function.Name, tc.Function.Arguments)
			if err != nil {
				result = fmt.Sprintf("error: %v", err)
				onEvent(Event{Type: EventToolError, Tool: tc.Function.Name, Text: result})
			} else {
				onEvent(Event{Type: EventToolEnd, Tool: tc.Function.Name, Text: truncate(result, 500)})
			}
			rt.messages = append(rt.messages, router.Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    result,
			})
		}
		// After tools, continue loop for model follow-up.
		estTokens = estimateTokens(rt.messages)
	}
	return final.String(), fmt.Errorf("tool loop exceeded %d rounds", maxToolRounds)
}

// RunHeadless prints a single-prompt turn to w (CI / scripting).
func (rt *Runtime) RunHeadless(ctx context.Context, prompt string, w io.Writer) error {
	text, err := rt.RunTurn(ctx, prompt, func(ev Event) {
		switch ev.Type {
		case EventTextDelta:
			_, _ = io.WriteString(w, ev.Text)
		case EventToolStart:
			_, _ = fmt.Fprintf(w, "\n[tool:%s]\n", ev.Tool)
		case EventLLMDone:
			_, _ = fmt.Fprintf(w, "\n<!-- model=%s tokens=%d est_usd=%.6f duration=%s -->\n",
				ev.Model, ev.Tokens, ev.CostUSD, ev.Duration.Round(time.Millisecond))
		}
	})
	if err != nil {
		return err
	}
	if !strings.HasSuffix(text, "\n") {
		_, _ = io.WriteString(w, "\n")
	}
	return nil
}

func defaultSystemPrompt(root string) string {
	return fmt.Sprintf(`You are I/O Mesh TUI (iomesh), a coding agent harness rewritten in Go for tight platform integration.
You operate in workspace: %s

Capabilities: read/search files, propose edits, run shell commands (when approved), spawn subagents (explore/plan/general-purpose), optional skills (list_skills/read_skill), optional MCP tools (mcp__server__tool), and reason over repository-scale context.
Prefer precise, minimal diffs. Use tools instead of inventing file contents.
Delegate exploration and planning to subagents when it preserves your context:
- explore: codebase investigation (no edits)
- plan: structured implementation plan (no edits)
- general-purpose: full capability delegated work
MAXIMUM PARALLELISM: for independent tasks always use spawn_subagents (never serial spawn_subagent loops). wait=true joins concurrently. Prefer default_subagent_type=explore for research fan-out. For parallel isolated edits: default_isolation=worktree + wait=true, then apply_worktrees (or apply_after=true). Cap is max_concurrent running children.
ISOLATED EDITS: isolation=worktree → diff_worktree → apply_worktree / apply_worktrees (approval/--yolo).
SKILLS: when a <skills> catalog is present, use list_skills/read_skill before inventing process.
MCP: tools prefixed mcp__ require approval when the server is mutating (default).
When I/O Mesh context is provided inside <iomesh-context>, treat it as governed operational truth for production systems.
When <memory-context> is present, treat it as recalled Memory Palace turns/facts (temporal when timestamps/session_id apply); do not invent memories not present there.`, root)
}

func estimateTokens(msgs []router.Message) int {
	// Rough 4 chars/token heuristic for routing only.
	n := 0
	for _, m := range msgs {
		n += len(m.Content) / 4
		for _, tc := range m.ToolCalls {
			n += len(tc.Function.Arguments) / 4
		}
	}
	return n + 1024 // tools + reply budget
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
