package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/iome-sh/iomesh-tui/internal/agent"
	"github.com/iome-sh/iomesh-tui/internal/config"
	"github.com/iome-sh/iomesh-tui/internal/iomesh"
	"github.com/iome-sh/iomesh-tui/internal/mcp"
	"github.com/iome-sh/iomesh-tui/internal/router"
	"github.com/iome-sh/iomesh-tui/internal/session"
	"github.com/iome-sh/iomesh-tui/internal/skills"
)

// Options configure the ACP stdio server.
type Options struct {
	ConfigPath string
	Workspace  string
	Model      string
	Yolo       bool
	Version    string
	Logger     *slog.Logger
}

// Server is a newline-delimited JSON-RPC ACP server.
type Server struct {
	opts   Options
	logger *slog.Logger
	cfg    *config.Config

	mu       sync.Mutex
	sessions map[string]*agent.Runtime
	stores   map[string]*session.Store
	seq      atomic.Uint64
	toolSeq  atomic.Uint64

	outMu sync.Mutex
	out   io.Writer
}

// New constructs a server (does not start the loop).
func New(opts Options) *Server {
	if opts.Logger == nil {
		opts.Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	if opts.Version == "" {
		opts.Version = "0.1.0-dev"
	}
	return &Server{
		opts:     opts,
		logger:   opts.Logger,
		sessions: make(map[string]*agent.Runtime),
		stores:   make(map[string]*session.Store),
	}
}

// Run reads JSON-RPC lines from in and writes responses/notifications to out.
// out must be stdout for real clients; in is stdin.
func (s *Server) Run(ctx context.Context, in io.Reader, out io.Writer) error {
	s.out = out
	cfg, err := loadConfig(s.opts.ConfigPath)
	if err != nil {
		return err
	}
	s.cfg = cfg
	if s.opts.Yolo {
		s.cfg.Agent.Yolo = true
	}
	if s.opts.Workspace != "" {
		s.cfg.Agent.Workspace = s.opts.Workspace
	}

	sc := bufio.NewScanner(in)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 16*1024*1024)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if !sc.Scan() {
			if err := sc.Err(); err != nil {
				return err
			}
			return nil // EOF
		}
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if err := s.handleLine(ctx, line); err != nil {
			s.logger.Error("acp handle", "err", err)
		}
	}
}

func (s *Server) handleLine(ctx context.Context, line string) error {
	var req request
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		return s.write(response{
			JSONRPC: "2.0",
			Error:   &rpcError{Code: codeParseError, Message: "parse error"},
		})
	}
	if req.JSONRPC != "" && req.JSONRPC != "2.0" {
		return s.replyError(req.ID, codeInvalidRequest, "jsonrpc must be 2.0")
	}
	// Notifications have no id.
	isNotif := len(req.ID) == 0 || string(req.ID) == "null"

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "initialized", "notifications/initialized":
		return nil // client ack
	case "session/new":
		return s.handleSessionNew(req)
	case "session/load":
		return s.handleSessionLoad(req)
	case "session/prompt":
		return s.handleSessionPrompt(ctx, req)
	case "session/cancel":
		// Best-effort: no per-session cancel token yet.
		if !isNotif {
			return s.replyResult(req.ID, map[string]any{"ok": true})
		}
		return nil
	case "shutdown":
		if !isNotif {
			_ = s.replyResult(req.ID, map[string]any{})
		}
		return io.EOF
	case "exit":
		return io.EOF
	default:
		if isNotif {
			return nil
		}
		return s.replyError(req.ID, codeMethodNotFound, "method not found: "+req.Method)
	}
}

func (s *Server) handleInitialize(req request) error {
	var p initializeParams
	_ = json.Unmarshal(req.Params, &p)
	ver := ProtocolVersion
	if p.ProtocolVersion != "" {
		// Accept client version string; report ours.
		ver = ProtocolVersion
	}
	return s.replyResult(req.ID, initializeResult{
		ProtocolVersion: ver,
		ServerInfo: serverInfo{
			Name:    "iomesh",
			Version: s.opts.Version,
		},
		Capabilities: map[string]any{
			"loadSession": true,
			"promptCapabilities": map[string]any{
				"image":           false,
				"audio":           false,
				"embeddedContext": true,
			},
		},
	})
}

func (s *Server) handleSessionNew(req request) error {
	var p sessionNewParams
	if err := json.Unmarshal(req.Params, &p); err != nil && len(req.Params) > 0 {
		return s.replyError(req.ID, codeInvalidParams, "invalid session/new params")
	}
	cwd := p.Cwd
	if cwd == "" {
		cwd = s.cfg.Agent.Workspace
	}
	if cwd == "" {
		wd, err := osGetwd()
		if err != nil {
			return s.replyError(req.ID, codeInternalError, err.Error())
		}
		cwd = wd
	}
	rt, store, err := s.newRuntime(cwd)
	if err != nil {
		return s.replyError(req.ID, codeInternalError, err.Error())
	}
	id := fmt.Sprintf("acp-%d", s.seq.Add(1))
	s.mu.Lock()
	s.sessions[id] = rt
	s.stores[id] = store
	s.mu.Unlock()
	return s.replyResult(req.ID, sessionNewResult{SessionID: id})
}

func (s *Server) handleSessionLoad(req request) error {
	var p sessionLoadParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return s.replyError(req.ID, codeInvalidParams, "invalid session/load params")
	}
	if p.SessionID == "" {
		return s.replyError(req.ID, codeInvalidParams, "sessionId required")
	}
	cwd := p.Cwd
	if cwd == "" {
		cwd = s.cfg.Agent.Workspace
	}
	if cwd == "" {
		wd, _ := osGetwd()
		cwd = wd
	}
	rt, store, err := s.newRuntime(cwd)
	if err != nil {
		return s.replyError(req.ID, codeInternalError, err.Error())
	}
	snap, err := store.Load(p.SessionID)
	if err != nil {
		// Also try as ACP id already in memory — fail clearly.
		return s.replyError(req.ID, codeInternalError, "load session: "+err.Error())
	}
	if err := rt.LoadSession(snap); err != nil {
		return s.replyError(req.ID, codeInternalError, err.Error())
	}
	// Use stable ACP session id (may differ from disk id).
	id := p.SessionID
	s.mu.Lock()
	s.sessions[id] = rt
	s.stores[id] = store
	s.mu.Unlock()
	return s.replyResult(req.ID, sessionNewResult{SessionID: id})
}

func (s *Server) handleSessionPrompt(ctx context.Context, req request) error {
	var p sessionPromptParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return s.replyError(req.ID, codeInvalidParams, "invalid session/prompt params")
	}
	if p.SessionID == "" {
		return s.replyError(req.ID, codeInvalidParams, "sessionId required")
	}
	text := joinPrompt(p.Prompt)
	if strings.TrimSpace(text) == "" {
		return s.replyError(req.ID, codeInvalidParams, "empty prompt")
	}

	s.mu.Lock()
	rt := s.sessions[p.SessionID]
	store := s.stores[p.SessionID]
	s.mu.Unlock()
	if rt == nil {
		return s.replyError(req.ID, codeInvalidParams, "unknown sessionId")
	}

	// Stream agent events as session/update notifications (subagent tools included).
	_, err := rt.RunTurn(ctx, text, func(ev agent.Event) {
		s.emitAgentEvent(p.SessionID, ev)
	})
	if err != nil {
		_ = s.notifyUpdate(p.SessionID, map[string]any{
			"sessionUpdate": "tool_call_update",
			"title":         "error",
			"status":        "failed",
			"content":       []any{map[string]string{"type": "text", "text": err.Error()}},
		})
		return s.replyResult(req.ID, sessionPromptResult{StopReason: "cancelled"})
	}
	if store != nil {
		rt.AutoSaveAfterTurn(store)
	}
	return s.replyResult(req.ID, sessionPromptResult{StopReason: "end_turn"})
}

func (s *Server) emitAgentEvent(sessionID string, ev agent.Event) {
	switch ev.Type {
	case agent.EventTextDelta:
		_ = s.notifyUpdate(sessionID, agentMessageChunk{
			SessionUpdate: "agent_message_chunk",
			Content:       chunk{Type: "text", Text: ev.Text},
		})
	case agent.EventThinkingDelta:
		_ = s.notifyUpdate(sessionID, agentThoughtChunk{
			SessionUpdate: "agent_thought_chunk",
			Content:       chunk{Type: "text", Text: ev.Text},
		})
	case agent.EventModelSelected:
		_ = s.notifyUpdate(sessionID, modelSelectedUpdate{
			SessionUpdate: "model_selected",
			Model:         ev.Model,
		})
	case agent.EventToolStart:
		id := fmt.Sprintf("tool-%d", s.toolSeq.Add(1))
		kind := toolKind(ev.Tool)
		_ = s.notifyUpdate(sessionID, toolCallUpdate{
			SessionUpdate: "tool_call",
			ToolCallID:    id,
			Title:         ev.Tool,
			Kind:          kind,
			Status:        "in_progress",
			RawInput:      ev.Text,
		})
		// Stash last id on a side map would need session state; emit update with same title for end.
		s.mu.Lock()
		// reuse tool id via title map on server — simple map tool name+seq is enough for stream UX
		s.mu.Unlock()
		// Store on event by encoding id in concurrent map
		s.rememberToolID(sessionID, ev.Tool, id)
	case agent.EventToolEnd, agent.EventToolError, agent.EventToolDenied:
		id := s.takeToolID(sessionID, ev.Tool)
		status := "completed"
		if ev.Type != agent.EventToolEnd {
			status = "failed"
		}
		_ = s.notifyUpdate(sessionID, toolCallUpdate{
			SessionUpdate: "tool_call_update",
			ToolCallID:    id,
			Title:         ev.Tool,
			Kind:          toolKind(ev.Tool),
			Status:        status,
			Content:       []any{map[string]string{"type": "text", "text": truncate(ev.Text, 2000)}},
		})
	case agent.EventLLMDone:
		_ = s.notifyUpdate(sessionID, map[string]any{
			"sessionUpdate": "usage",
			"model":         ev.Model,
			"tokens":        ev.Tokens,
			"est_usd":       ev.CostUSD,
			"duration_ms":   ev.Duration.Milliseconds(),
		})
	case agent.EventMeshContext:
		_ = s.notifyUpdate(sessionID, map[string]any{
			"sessionUpdate": "agent_thought_chunk",
			"content":       map[string]string{"type": "text", "text": "[iomesh] " + ev.Text},
		})
	case agent.EventMemoryRecall, agent.EventMemoryIngest:
		_ = s.notifyUpdate(sessionID, map[string]any{
			"sessionUpdate": "agent_thought_chunk",
			"content":       map[string]string{"type": "text", "text": "[memory] " + ev.Text},
		})
	}
}

func toolKind(name string) string {
	switch name {
	case "read_file", "list_dir", "grep", "diff_worktree", "list_worktrees":
		return "read"
	case "write_file", "apply_worktree", "apply_worktrees":
		return "edit"
	case "run_shell", "remove_worktree":
		return "execute"
	case "spawn_subagent", "spawn_subagents", "get_subagent_output", "wait_subagents":
		return "other" // subagent orchestration — clients can special-case title
	default:
		return "other"
	}
}

// toolIDs maps session|toolName -> last tool call id (best-effort pairing).
var toolIDs sync.Map

func (s *Server) rememberToolID(sessionID, tool, id string) {
	toolIDs.Store(sessionID+"\x00"+tool, id)
}

func (s *Server) takeToolID(sessionID, tool string) string {
	key := sessionID + "\x00" + tool
	if v, ok := toolIDs.LoadAndDelete(key); ok {
		return v.(string)
	}
	return fmt.Sprintf("tool-%d", s.toolSeq.Add(1))
}

func (s *Server) newRuntime(cwd string) (*agent.Runtime, *session.Store, error) {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return nil, nil, err
	}
	cfg := *s.cfg
	cfg.Agent.Workspace = abs

	var metrics router.MetricsSink = router.NopMetrics{}
	mesh := iomesh.New(iomesh.Config{
		Enabled:         cfg.IOMesh.Enabled,
		Endpoint:        cfg.IOMesh.Endpoint,
		Tenant:          cfg.IOMesh.Tenant,
		APIKeyEnv:       cfg.IOMesh.APIKeyEnv,
		OrgID:           cfg.IOMesh.Org,
		WorkspaceID:     cfg.IOMesh.Workspace,
		DualWrite:       cfg.Memory.DualWrite,
		MemoryEndpoint:  cfg.Memory.Endpoint,
		EmitDeptStreams: cfg.IOMesh.EmitDeptStreams,
		ContextPlane:    cfg.IOMesh.ContextPlane,
		IncludeLineage:  cfg.IOMesh.IncludeLineage,
		PolicyMode:      iomesh.PolicyMode(cfg.IOMesh.PolicyMode),
		CatalogPlane:    cfg.IOMesh.CatalogPlane,
		InjectCatalog:   cfg.IOMesh.InjectCatalog,
	}, s.logger)
	if mesh.Enabled() {
		metrics = mesh
	}
	rtr, err := cfg.NewRouter(router.WithLogger(s.logger), router.WithMetrics(metrics))
	if err != nil {
		return nil, nil, err
	}
	if s.opts.Model != "" {
		if err := rtr.SetOverride(s.opts.Model); err != nil {
			return nil, nil, err
		}
	}
	subEnabled := cfg.Subagents.Enabled && cfg.Features.Subagents
	rt, err := agent.New(agent.Config{
		Mode:                  agent.ModeACP,
		Workspace:             abs,
		Yolo:                  cfg.Agent.Yolo,
		SubagentsEnabled:      subEnabled,
		MaxSubagentDepth:      cfg.Subagents.MaxDepth,
		MaxSubagentConcurrent: cfg.Subagents.MaxConcurrent,
		MaxSubagentBatch:      cfg.Subagents.MaxBatch,
		WorktreeBase:          cfg.Subagents.WorktreeBase,
		WorktreeAutoRemove:    cfg.Subagents.WorktreeAutoRemove,
	}, rtr, mesh, s.logger)
	if err != nil {
		return nil, nil, err
	}
	rt.AttachMeshTools()
	if cfg.Skills.Enabled && cfg.Features.Skills {
		dirs := skills.DefaultDirs(abs)
		dirs = append(dirs, cfg.Skills.Dirs...)
		// Builtin always merged (s1251 connector-integrations-setup).
		if cat, err := skills.LoadWithBuiltin(dirs...); err == nil && cat.Len() > 0 {
			rt.AttachSkills(cat)
		}
	}
	if cfg.MCP.Enabled && cfg.Features.MCP && len(cfg.MCP.Servers) > 0 {
		var servers []mcp.ServerConfig
		for _, s := range cfg.MCP.Servers {
			sc := mcp.ServerConfig{
				Name: s.Name, Command: s.Command, Args: s.Args, Env: s.Env,
				URL: s.URL, Headers: s.Headers, AllowLoopback: s.AllowLoopback,
				Enabled: s.Enabled, Mutating: s.Mutating,
				StartupTimeoutSec: s.StartupTimeoutSec, ToolTimeoutSec: s.ToolTimeoutSec,
				AccessTokenEnv: s.OAuthTokenEnv,
			}
			if s.OAuth != nil {
				sc.OAuth = &mcp.OAuthConfig{
					TokenURL: s.OAuth.TokenURL, ClientID: s.OAuth.ClientID,
					ClientSecretEnv: s.OAuth.ClientSecretEnv, Scopes: s.OAuth.Scopes,
					AccessTokenEnv: s.OAuth.AccessTokenEnv, AllowLoopback: s.OAuth.AllowLoopback,
				}
			}
			servers = append(servers, sc)
		}
		mgr := mcp.NewManager(context.Background(), servers, s.logger)
		rt.AttachMCP(mgr)
	}
	if cfg.Memory.Enabled {
		rt.AttachMemory(agent.MemoryConfig{
			Enabled:          true,
			Server:           cfg.Memory.Server,
			Tenant:           cfg.Memory.Tenant,
			AutoRecall:       cfg.Memory.AutoRecall,
			AutoIngest:       cfg.Memory.AutoIngest,
			DualWrite:        cfg.Memory.DualWrite,
			Limit:            cfg.Memory.Limit,
			MaxSnippetBytes:  cfg.Memory.MaxSnippetBytes,
			RecallSince:      cfg.Memory.RecallSince,
			RecallUntil:      cfg.Memory.RecallUntil,
			RecallSessionSeq: cfg.Memory.RecallSessionSeq,
			RecallCacheTTLMS: cfg.Memory.RecallCacheTTLMS,
		})
	}
	store, err := session.Open(abs)
	if err != nil {
		return nil, nil, err
	}
	rt.EnableAutoSave(true)
	return rt, store, nil
}

func (s *Server) notifyUpdate(sessionID string, update any) error {
	return s.write(notification{
		JSONRPC: "2.0",
		Method:  "session/update",
		Params: sessionUpdateParams{
			SessionID:     sessionID,
			SessionUpdate: update,
		},
	})
}

func (s *Server) replyResult(id json.RawMessage, result any) error {
	return s.write(response{JSONRPC: "2.0", ID: id, Result: result})
}

func (s *Server) replyError(id json.RawMessage, code int, msg string) error {
	return s.write(response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: msg},
	})
}

func (s *Server) write(v any) error {
	s.outMu.Lock()
	defer s.outMu.Unlock()
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = s.out.Write(append(data, '\n'))
	return err
}

func joinPrompt(parts []promptContent) string {
	var b strings.Builder
	for _, p := range parts {
		if p.Type == "" || p.Type == "text" {
			b.WriteString(p.Text)
		}
	}
	return b.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func loadConfig(path string) (*config.Config, error) {
	if path != "" {
		return config.Load(path)
	}
	return config.LoadUser()
}

// osGetwd indirection for tests.
var osGetwd = os.Getwd
