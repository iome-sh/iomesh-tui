// Package acp implements a minimal Agent Client Protocol (ACP) server over
// stdio JSON-RPC for IDE and automation clients.
//
// Wire format: newline-delimited JSON-RPC 2.0 (one object per line on stdin/stdout).
// Logs must go to stderr only so stdout stays a clean protocol stream.
package acp

import "encoding/json"

// JSON-RPC 2.0 envelopes.
type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Standard JSON-RPC error codes.
const (
	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternalError  = -32603
)

// --- initialize ---

type initializeParams struct {
	ProtocolVersion string         `json:"protocolVersion"`
	ClientInfo      *clientInfo    `json:"clientInfo,omitempty"`
	Capabilities    map[string]any `json:"capabilities,omitempty"`
}

type clientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type initializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	ServerInfo      serverInfo     `json:"serverInfo"`
	Capabilities    map[string]any `json:"capabilities"`
}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// --- session/new ---

type sessionNewParams struct {
	Cwd        string         `json:"cwd"`
	MCPServers []any          `json:"mcpServers,omitempty"`
	Metadata   map[string]any `json:"_meta,omitempty"`
}

type sessionNewResult struct {
	SessionID string `json:"sessionId"`
}

// --- session/load ---

type sessionLoadParams struct {
	SessionID string `json:"sessionId"`
	Cwd       string `json:"cwd,omitempty"`
}

// --- session/prompt ---

type sessionPromptParams struct {
	SessionID string          `json:"sessionId"`
	Prompt    []promptContent `json:"prompt"`
}

type promptContent struct {
	Type string `json:"type"` // "text"
	Text string `json:"text,omitempty"`
}

type sessionPromptResult struct {
	StopReason string `json:"stopReason"` // "end_turn" | "cancelled" | "max_tokens"
}

// --- session/update notifications ---

type sessionUpdateParams struct {
	SessionID     string `json:"sessionId"`
	SessionUpdate any    `json:"sessionUpdate"` // typed update objects
	// Also flatten common fields for simple clients:
	UpdateType string `json:"updateType,omitempty"`
}

// Update payloads (sessionUpdate shapes).
type agentMessageChunk struct {
	SessionUpdate string `json:"sessionUpdate"` // agent_message_chunk
	Content       chunk  `json:"content"`
}

type agentThoughtChunk struct {
	SessionUpdate string `json:"sessionUpdate"` // agent_thought_chunk
	Content       chunk  `json:"content"`
}

type chunk struct {
	Type string `json:"type"` // text
	Text string `json:"text"`
}

type toolCallUpdate struct {
	SessionUpdate string `json:"sessionUpdate"` // tool_call | tool_call_update
	ToolCallID    string `json:"toolCallId"`
	Title         string `json:"title,omitempty"`
	Kind          string `json:"kind,omitempty"` // read | edit | execute | search | other
	Status        string `json:"status"`         // pending | in_progress | completed | failed
	RawInput      any    `json:"rawInput,omitempty"`
	Content       []any  `json:"content,omitempty"`
}

type modelSelectedUpdate struct {
	SessionUpdate string `json:"sessionUpdate"` // model_selected (iomesh extension)
	Model         string `json:"model"`
}

// ProtocolVersion supported by this server.
const ProtocolVersion = "2024-11-05"
