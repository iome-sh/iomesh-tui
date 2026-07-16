package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Binding maps an agent-facing tool name to a server tool.
type Binding struct {
	Qualified string
	Server    string
	Tool      string // original MCP tool name
	Mutating  bool
	Client    *Client
}

// Manager owns multiple MCP server clients (fail-open connect).
type Manager struct {
	logger   *slog.Logger
	mu       sync.Mutex
	clients  map[string]*Client
	bindings map[string]Binding // qualified → binding
}

// NewManager connects enabled servers. Individual failures are logged and skipped.
func NewManager(ctx context.Context, servers []ServerConfig, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	m := &Manager{
		logger:   logger,
		clients:  map[string]*Client{},
		bindings: map[string]Binding{},
	}
	for _, cfg := range servers {
		if cfg.Name == "" || !cfg.isEnabled() {
			continue
		}
		var (
			c   *Client
			err error
		)
		switch {
		case cfg.URL != "":
			c, err = DialHTTP(ctx, cfg, logger)
		case cfg.Command != "":
			c, err = DialStdio(ctx, cfg, logger)
		default:
			logger.Warn("mcp server skipped: need command or url", "name", cfg.Name)
			continue
		}
		if err != nil {
			logger.Warn("mcp server connect failed", "name", cfg.Name, "err", err)
			continue
		}
		m.attachLocked(c)
		transport := "stdio"
		if c.isHTTP() {
			transport = "http"
		}
		logger.Info("mcp server connected", "name", cfg.Name, "transport", transport, "tools", len(c.Tools()))
	}
	return m
}

// NewManagerEmpty is a connected-empty manager for tests / disabled MCP.
func NewManagerEmpty(logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	return &Manager{logger: logger, clients: map[string]*Client{}, bindings: map[string]Binding{}}
}

// Attach registers an already-connected client (tests).
func (m *Manager) Attach(c *Client) {
	if m == nil || c == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attachLocked(c)
}

func (m *Manager) attachLocked(c *Client) {
	m.clients[c.Name()] = c
	for _, t := range c.Tools() {
		q := ToolQualifiedName(c.Name(), t.Name)
		m.bindings[q] = Binding{
			Qualified: q,
			Server:    c.Name(),
			Tool:      t.Name,
			Mutating:  c.Mutating(),
			Client:    c,
		}
	}
}

// Bindings returns all agent-facing tool bindings.
func (m *Manager) Bindings() []Binding {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Binding, 0, len(m.bindings))
	for _, b := range m.bindings {
		out = append(out, b)
	}
	return out
}

// Clients returns connected clients.
func (m *Manager) Clients() []*Client {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Client, 0, len(m.clients))
	for _, c := range m.clients {
		out = append(out, c)
	}
	return out
}

// Len returns connected server count.
func (m *Manager) Len() int {
	if m == nil {
		return 0
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.clients)
}

// Call routes a qualified tool name mcp__server__tool.
func (m *Manager) Call(ctx context.Context, qualified string, args map[string]any) (string, error) {
	m.mu.Lock()
	b, ok := m.bindings[qualified]
	m.mu.Unlock()
	if !ok {
		// Fallback parse for dynamic names.
		server, tool, ok2 := SplitQualified(qualified)
		if !ok2 {
			return "", fmt.Errorf("mcp: unknown tool %q", qualified)
		}
		m.mu.Lock()
		var c *Client
		for name, cl := range m.clients {
			if sanitize(name) == server || name == server {
				c = cl
				break
			}
		}
		m.mu.Unlock()
		if c == nil {
			return "", fmt.Errorf("mcp: server %q not connected", server)
		}
		return c.CallTool(ctx, tool, args)
	}
	return b.Client.CallTool(ctx, b.Tool, args)
}

// Close shuts down all clients.
func (m *Manager) Close() error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, c := range m.clients {
		_ = c.Close()
		delete(m.clients, name)
	}
	return nil
}

// SplitQualified parses mcp__server__tool.
func SplitQualified(name string) (server, tool string, ok bool) {
	if len(name) < 7 || name[:5] != "mcp__" {
		return "", "", false
	}
	rest := name[5:]
	i := 0
	for i < len(rest) {
		if i+1 < len(rest) && rest[i] == '_' && rest[i+1] == '_' {
			server = rest[:i]
			tool = rest[i+2:]
			if server != "" && tool != "" {
				return server, tool, true
			}
			return "", "", false
		}
		i++
	}
	return "", "", false
}
