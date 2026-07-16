// Package iomesh integrates the agent harness with the I/O Mesh platform:
// governed operational context (context plane), dept.* event emission, and
// optional usage metering hooks for production agents.
//
// When disabled, all methods are no-ops so the agent runtime stays offline-first.
package iomesh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/router"
)

// Config is the runtime subset needed by the client.
type Config struct {
	Enabled         bool
	Endpoint        string
	Tenant          string
	APIKeyEnv       string
	EmitDeptStreams bool
	ContextPlane    bool
}

// Client talks to I/O Mesh control/data planes (OpenHTTP, fail-open).
type Client struct {
	cfg        Config
	httpClient *http.Client
	logger     *slog.Logger
}

// New builds a client. A nil or disabled config yields a safe no-op client.
func New(cfg Config, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		logger: logger,
	}
}

// Enabled reports whether platform integration is active.
func (c *Client) Enabled() bool {
	return c != nil && c.cfg.Enabled && c.cfg.Endpoint != ""
}

// ContextSnippet fetches governed operational context for injection into
// the agent system prompt (Knowledge / Operational Data Mesh). Fail-open:
// errors return empty string so the agent continues without mesh context.
func (c *Client) ContextSnippet(ctx context.Context, workspace, query string) string {
	if !c.Enabled() || !c.cfg.ContextPlane {
		return ""
	}
	url := strings.TrimRight(c.cfg.Endpoint, "/") + "/v1/context/query"
	payload := map[string]any{
		"tenant":    c.cfg.Tenant,
		"workspace": workspace,
		"query":     query,
		"limit":     20,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		c.logger.Debug("iomesh context: build request", "err", err)
		return ""
	}
	c.auth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Debug("iomesh context: request failed (fail-open)", "err", err)
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		c.logger.Debug("iomesh context: non-OK (fail-open)", "status", resp.StatusCode)
		return ""
	}
	var out struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return ""
	}
	return out.Text
}

// DeptEvent is a lightweight operational stream event (dept.* family).
type DeptEvent struct {
	Type      string         `json:"type"` // e.g. dept.agent.llm_call
	Timestamp time.Time      `json:"ts"`
	Tenant    string         `json:"tenant,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	Payload   map[string]any `json:"payload"`
}

// Emit publishes a dept.* event. Fail-open on error.
func (c *Client) Emit(ctx context.Context, ev DeptEvent) {
	if !c.Enabled() || !c.cfg.EmitDeptStreams {
		return
	}
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now().UTC()
	}
	if ev.Tenant == "" {
		ev.Tenant = c.cfg.Tenant
	}
	url := strings.TrimRight(c.cfg.Endpoint, "/") + "/v1/streams/dept"
	body, err := json.Marshal(ev)
	if err != nil {
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return
	}
	c.auth(req)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Debug("iomesh emit failed (fail-open)", "err", err, "type", ev.Type)
		return
	}
	_ = resp.Body.Close()
}

// RecordLLMCall implements router.MetricsSink for usage metering.
func (c *Client) RecordLLMCall(meta router.CallMeta, usage router.Usage, err error) {
	if !c.Enabled() {
		return
	}
	payload := map[string]any{
		"model":      meta.ModelName,
		"model_id":   meta.ModelID,
		"duration_ms": meta.Duration.Milliseconds(),
		"attempts":   meta.Attempts,
		"fallback":   meta.FallbackUsed,
		"est_usd":    meta.EstimatedUSD,
		"tokens": map[string]int{
			"prompt":     usage.PromptTokens,
			"completion": usage.CompletionTokens,
			"total":      usage.TotalTokens,
		},
	}
	if err != nil {
		payload["error"] = err.Error()
	}
	// Best-effort background emit with short timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.Emit(ctx, DeptEvent{
		Type:    "dept.agent.llm_call",
		Payload: payload,
	})
}

func (c *Client) auth(req *http.Request) {
	env := c.cfg.APIKeyEnv
	if env == "" {
		env = "IOMESH_API_KEY"
	}
	if key := os.Getenv(env); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	if c.cfg.Tenant != "" {
		req.Header.Set("X-IOMesh-Tenant", c.cfg.Tenant)
	}
}

// Health checks mesh reachability. Returns error if enabled and unhealthy.
func (c *Client) Health(ctx context.Context) error {
	if !c.Enabled() {
		return nil
	}
	url := strings.TrimRight(c.cfg.Endpoint, "/") + "/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("iomesh health: http %d", resp.StatusCode)
	}
	return nil
}
