// Package mcp implements a minimal Model Context Protocol client over stdio
// or streamable HTTP/SSE JSON-RPC (initialize, tools/list, tools/call).
// Fail-open at manager level.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/security"
)

// ProtocolVersion negotiated with servers.
const ProtocolVersion = "2024-11-05"

// DefaultMaxOutputBytes caps tools/call text results.
const DefaultMaxOutputBytes = 20_000

// ServerConfig describes one MCP server (stdio and/or HTTP).
// Set Command for stdio, or URL for streamable HTTP. URL wins if both set.
type ServerConfig struct {
	Name    string            `toml:"name" json:"name"`
	Command string            `toml:"command" json:"command"`
	Args    []string          `toml:"args" json:"args"`
	Env     map[string]string `toml:"env" json:"env"`
	// Cwd is optional working directory for stdio processes (e.g. expanded PLUGIN_ROOT).
	// Empty leaves the process default (client process cwd).
	Cwd string `toml:"cwd" json:"cwd"`
	// URL enables streamable HTTP transport (POST JSON-RPC; JSON or SSE responses).
	URL string `toml:"url" json:"url"`
	// Headers are extra HTTP request headers (Authorization, etc.).
	Headers map[string]string `toml:"headers" json:"headers"`
	// AllowLoopback permits http://127.0.0.1 for local MCP servers (default true when unset via dial).
	AllowLoopback *bool `toml:"allow_loopback" json:"allow_loopback"`
	// Enabled defaults true when unset via manager.
	Enabled *bool `toml:"enabled" json:"enabled"`
	// Mutating: all tools from this server require approval (default true).
	Mutating *bool `toml:"mutating" json:"mutating"`
	// StartupTimeoutSec for initialize (default 30).
	StartupTimeoutSec int `toml:"startup_timeout_sec" json:"startup_timeout_sec"`
	// ToolTimeoutSec per tools/call (default 120).
	ToolTimeoutSec int `toml:"tool_timeout_sec" json:"tool_timeout_sec"`
	// OAuth optional bearer resolution for HTTP (see oauth.go).
	OAuth *OAuthConfig `toml:"oauth" json:"oauth"`
	// AccessTokenEnv if set injects Authorization: Bearer from that env var (simplest auth).
	AccessTokenEnv string `toml:"oauth_token_env" json:"oauth_token_env"`
}

func (c ServerConfig) isEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

func (c ServerConfig) isMutating() bool {
	if c.Mutating == nil {
		return true // fail-closed
	}
	return *c.Mutating
}

// Client is a connected MCP server (stdio or HTTP).
type Client struct {
	cfg    ServerConfig
	logger *slog.Logger

	// stdio transport
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	// optional owned process cancel
	cancel context.CancelFunc

	// http transport (streamable HTTP / SSE responses)
	httpClient *http.Client
	baseURL    string
	sessionID  string

	mu        sync.Mutex // pending map + tools + sessionID
	writeMu   sync.Mutex // serialize stdin writes (do not hold mu during Write — pipe deadlock)
	pending   map[int64]chan rpcResponse
	nextID    atomic.Int64
	tools     []Tool
	resources []Resource
	prompts   []Prompt
	closed    atomic.Bool

	maxOut int
	// oauthTok caches client-credentials access token (HTTP).
	oauthTok string
	oauthExp time.Time
	oauthMu  sync.Mutex
}

func (c *Client) isHTTP() bool { return c.httpClient != nil }

// DialStdio starts command and performs MCP initialize + tools/list.
func DialStdio(ctx context.Context, cfg ServerConfig, logger *slog.Logger) (*Client, error) {
	if cfg.Command == "" {
		return nil, fmt.Errorf("mcp: server %q: empty command", cfg.Name)
	}
	if logger == nil {
		logger = slog.Default()
	}
	startup := time.Duration(cfg.StartupTimeoutSec) * time.Second
	if startup <= 0 {
		startup = 30 * time.Second
	}

	runCtx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(runCtx, cfg.Command, cfg.Args...)
	if strings.TrimSpace(cfg.Cwd) != "" {
		cmd.Dir = cfg.Cwd
	}
	// Scrub secrets then apply explicit server env only.
	cmd.Env = security.ScrubEnv(nil)
	for k, v := range cfg.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	// Discard or log stderr lightly — do not block.
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("mcp: start %s: %w", cfg.Name, err)
	}

	c := &Client{
		cfg:     cfg,
		logger:  logger,
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		cancel:  cancel,
		pending: map[int64]chan rpcResponse{},
		maxOut:  DefaultMaxOutputBytes,
	}
	go c.readLoop()

	initCtx, initCancel := context.WithTimeout(ctx, startup)
	defer initCancel()
	if err := c.initialize(initCtx); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("mcp: initialize %s: %w", cfg.Name, err)
	}
	if err := c.refreshTools(initCtx); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("mcp: tools/list %s: %w", cfg.Name, err)
	}
	// Optional surfaces (fail-open).
	_ = c.refreshResources(initCtx)
	_ = c.refreshPrompts(initCtx)
	return c, nil
}

// NewClientForTest wires pipes without spawning a process (unit tests).
func NewClientForTest(cfg ServerConfig, stdin io.WriteCloser, stdout io.ReadCloser, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	c := &Client{
		cfg:     cfg,
		logger:  logger,
		stdin:   stdin,
		stdout:  stdout,
		pending: map[int64]chan rpcResponse{},
		maxOut:  DefaultMaxOutputBytes,
	}
	go c.readLoop()
	return c
}

// InitForTest runs initialize + tools/list (for cross-package tests).
func (c *Client) InitForTest(ctx context.Context) error {
	if err := c.initialize(ctx); err != nil {
		return err
	}
	if err := c.refreshTools(ctx); err != nil {
		return err
	}
	// Soft: resources/prompts optional on mocks.
	_ = c.refreshResources(ctx)
	_ = c.refreshPrompts(ctx)
	return nil
}

func (c *Client) Name() string { return c.cfg.Name }
func (c *Client) Mutating() bool {
	return c.cfg.isMutating()
}
func (c *Client) Tools() []Tool {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Tool, len(c.tools))
	copy(out, c.tools)
	return out
}

// Resources returns cached resources/list (may be empty if unsupported).
func (c *Client) Resources() []Resource {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Resource, len(c.resources))
	copy(out, c.resources)
	return out
}

// Prompts returns cached prompts/list (may be empty if unsupported).
func (c *Client) Prompts() []Prompt {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Prompt, len(c.prompts))
	copy(out, c.prompts)
	return out
}

func (c *Client) initialize(ctx context.Context) error {
	params := map[string]any{
		"protocolVersion": ProtocolVersion,
		"capabilities": map[string]any{
			"roots":    map[string]any{"listChanged": false},
			"sampling": map[string]any{},
		},
		"clientInfo": map[string]string{
			"name":    "iomesh-tui",
			"version": "0.1.0",
		},
	}
	var res initializeResult
	if err := c.call(ctx, "initialize", params, &res); err != nil {
		return err
	}
	// notifications/initialized (no response)
	_ = c.notify("notifications/initialized", map[string]any{})
	return nil
}

func (c *Client) refreshTools(ctx context.Context) error {
	var res toolsListResult
	if err := c.call(ctx, "tools/list", map[string]any{}, &res); err != nil {
		return err
	}
	c.mu.Lock()
	c.tools = res.Tools
	c.mu.Unlock()
	return nil
}

func (c *Client) refreshResources(ctx context.Context) error {
	var res resourcesListResult
	if err := c.call(ctx, "resources/list", map[string]any{}, &res); err != nil {
		// Method not supported — soft empty.
		c.logger.Debug("mcp resources/list unavailable", "server", c.cfg.Name, "err", err)
		return err
	}
	c.mu.Lock()
	c.resources = res.Resources
	c.mu.Unlock()
	return nil
}

func (c *Client) refreshPrompts(ctx context.Context) error {
	var res promptsListResult
	if err := c.call(ctx, "prompts/list", map[string]any{}, &res); err != nil {
		c.logger.Debug("mcp prompts/list unavailable", "server", c.cfg.Name, "err", err)
		return err
	}
	c.mu.Lock()
	c.prompts = res.Prompts
	c.mu.Unlock()
	return nil
}

// ListResources returns resources (refreshes when force or cache empty).
func (c *Client) ListResources(ctx context.Context, force bool) ([]Resource, error) {
	c.mu.Lock()
	empty := len(c.resources) == 0
	c.mu.Unlock()
	if force || empty {
		if err := c.refreshResources(ctx); err != nil && force {
			return nil, err
		}
	}
	return c.Resources(), nil
}

// ReadResource calls resources/read and returns concatenated text.
func (c *Client) ReadResource(ctx context.Context, uri string) (string, error) {
	if uri == "" {
		return "", fmt.Errorf("mcp: empty resource uri")
	}
	var res resourcesReadResult
	if err := c.call(ctx, "resources/read", resourcesReadParams{URI: uri}, &res); err != nil {
		return "", err
	}
	var b strings.Builder
	for _, p := range res.Contents {
		if p.Text != "" {
			b.WriteString(p.Text)
		} else if p.Blob != "" {
			// Avoid dumping large blobs; note presence.
			fmt.Fprintf(&b, "[blob mime=%s bytes≈%d]", p.MIME, len(p.Blob))
		}
	}
	text := security.Redact(b.String())
	return truncate(text, c.maxOut), nil
}

// ListPrompts returns prompts (refreshes when force or cache empty).
func (c *Client) ListPrompts(ctx context.Context, force bool) ([]Prompt, error) {
	c.mu.Lock()
	empty := len(c.prompts) == 0
	c.mu.Unlock()
	if force || empty {
		if err := c.refreshPrompts(ctx); err != nil && force {
			return nil, err
		}
	}
	return c.Prompts(), nil
}

// GetPrompt calls prompts/get and returns a readable message dump.
func (c *Client) GetPrompt(ctx context.Context, name string, arguments map[string]string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("mcp: empty prompt name")
	}
	var res promptsGetResult
	if err := c.call(ctx, "prompts/get", promptsGetParams{Name: name, Arguments: arguments}, &res); err != nil {
		return "", err
	}
	var b strings.Builder
	if res.Description != "" {
		fmt.Fprintf(&b, "description: %s\n\n", res.Description)
	}
	for _, msg := range res.Messages {
		fmt.Fprintf(&b, "[%s] %s\n", msg.Role, formatPromptContent(msg.Content))
	}
	text := security.Redact(b.String())
	return truncate(text, c.maxOut), nil
}

func formatPromptContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	// content may be a string
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var p contentPart
	if err := json.Unmarshal(raw, &p); err == nil && p.Text != "" {
		return p.Text
	}
	// array of parts
	var parts []contentPart
	if err := json.Unmarshal(raw, &parts); err == nil {
		var b strings.Builder
		for _, part := range parts {
			b.WriteString(part.Text)
		}
		return b.String()
	}
	return string(raw)
}

// CallTool invokes tools/call and returns concatenated text content.
func (c *Client) CallTool(ctx context.Context, name string, arguments map[string]any) (string, error) {
	if c.closed.Load() {
		return "", fmt.Errorf("mcp: client closed")
	}
	timeout := time.Duration(c.cfg.ToolTimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var res callToolResult
	if err := c.call(ctx, "tools/call", callToolParams{Name: name, Arguments: arguments}, &res); err != nil {
		return "", err
	}
	var b strings.Builder
	for _, p := range res.Content {
		if p.Type == "text" || p.Type == "" {
			b.WriteString(p.Text)
		}
	}
	text := b.String()
	if res.IsError {
		return "", fmt.Errorf("mcp tool error: %s", security.Redact(truncate(text, c.maxOut)))
	}
	text = security.Redact(text)
	return truncate(text, c.maxOut), nil
}

func (c *Client) call(ctx context.Context, method string, params any, result any) error {
	if c.isHTTP() {
		return c.callHTTP(ctx, method, params, result)
	}
	id := c.nextID.Add(1)
	rawParams, err := json.Marshal(params)
	if err != nil {
		return err
	}
	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  rawParams,
	}
	ch := make(chan rpcResponse, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	line, err := json.Marshal(req)
	if err != nil {
		return err
	}
	line = append(line, '\n')
	c.writeMu.Lock()
	_, err = c.stdin.Write(line)
	c.writeMu.Unlock()
	if err != nil {
		return fmt.Errorf("mcp write: %w", err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case resp := <-ch:
		if resp.Error != nil {
			return fmt.Errorf("mcp rpc: %s", resp.Error.Message)
		}
		if result != nil && len(resp.Result) > 0 {
			if err := json.Unmarshal(resp.Result, result); err != nil {
				return fmt.Errorf("mcp decode: %w", err)
			}
		}
		return nil
	}
}

func (c *Client) notify(method string, params any) error {
	if c.isHTTP() {
		return c.notifyHTTP(context.Background(), method, params)
	}
	rawParams, err := json.Marshal(params)
	if err != nil {
		return err
	}
	req := rpcRequest{JSONRPC: "2.0", Method: method, Params: rawParams}
	line, err := json.Marshal(req)
	if err != nil {
		return err
	}
	line = append(line, '\n')
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_, err = c.stdin.Write(line)
	return err
}

func (c *Client) readLoop() {
	sc := bufio.NewScanner(c.stdout)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 4*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(bytesTrimSpace(line)) == 0 {
			continue
		}
		var resp rpcResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			c.logger.Debug("mcp bad line", "server", c.cfg.Name, "err", err)
			continue
		}
		// Response with id
		if resp.ID != nil {
			id := coerceID(resp.ID)
			c.mu.Lock()
			ch := c.pending[id]
			c.mu.Unlock()
			if ch != nil {
				select {
				case ch <- resp:
				default:
				}
			}
		}
	}
}

func (c *Client) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}
	if c.isHTTP() {
		// Best-effort session teardown (streamable HTTP optional DELETE).
		c.mu.Lock()
		sid := c.sessionID
		url := c.baseURL
		c.mu.Unlock()
		if sid != "" && url != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
			if err == nil {
				req.Header.Set("Mcp-Session-Id", sid)
				for k, v := range c.cfg.Headers {
					req.Header.Set(k, v)
				}
				resp, err := c.httpClient.Do(req)
				if err == nil {
					_ = resp.Body.Close()
				}
			}
			cancel()
		}
		return nil
	}
	if c.stdin != nil {
		_ = c.stdin.Close()
	}
	if c.cancel != nil {
		c.cancel()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
		_, _ = c.cmd.Process.Wait()
	}
	return nil
}

func coerceID(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int64:
		return t
	case int:
		return int64(t)
	case json.Number:
		i, _ := t.Int64()
		return i
	default:
		return 0
	}
}

func bytesTrimSpace(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}

func truncate(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n] + "…[truncated]"
}

// ToolQualifiedName builds the agent-facing tool id.
func ToolQualifiedName(server, tool string) string {
	return "mcp__" + sanitize(server) + "__" + sanitize(tool)
}

func sanitize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}
