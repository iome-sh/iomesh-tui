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
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/router"
	"github.com/iome-sh/iomesh-tui/internal/security"
)

// PolicyMode controls remote policy evaluation for agent tools.
// Empty / "off": disabled. "advisory": evaluate + log/emit, never block.
// "enforce": deny when the mesh returns allow=false (fail-open on transport errors).
type PolicyMode string

const (
	PolicyOff      PolicyMode = "off"
	PolicyAdvisory PolicyMode = "advisory"
	PolicyEnforce  PolicyMode = "enforce"
)

// Config is the runtime subset needed by the client.
type Config struct {
	Enabled         bool
	Endpoint        string
	Tenant          string
	APIKeyEnv       string
	EmitDeptStreams bool
	ContextPlane    bool
	// IncludeLineage asks the context plane for lineage refs (fail-open).
	IncludeLineage bool
	// PolicyMode: off | advisory | enforce (default off).
	PolicyMode PolicyMode
}

// Client talks to I/O Mesh control/data planes (OpenHTTP, fail-open).
type Client struct {
	cfg        Config
	httpClient *http.Client
	logger     *slog.Logger
	meter      *UsageMeter
}

// New builds a client. A nil or disabled config yields a safe no-op client.
// When Enabled and Endpoint is set, the endpoint must be a valid http(s) URL.
func New(cfg Config, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	// Validate endpoint early when integration is enabled (allow loopback for local mesh).
	if cfg.Enabled && cfg.Endpoint != "" {
		if err := security.ValidateHTTPURL(cfg.Endpoint, true); err != nil {
			logger.Warn("iomesh: invalid endpoint; disabling client", "err", err)
			cfg.Enabled = false
		}
	}
	mode := PolicyMode(strings.ToLower(strings.TrimSpace(string(cfg.PolicyMode))))
	switch mode {
	case PolicyAdvisory, PolicyEnforce:
		cfg.PolicyMode = mode
	default:
		cfg.PolicyMode = PolicyOff
	}
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			// Cap redirects to reduce open-redirect/SSRF chaining risk.
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("stopped after 5 redirects")
				}
				return security.ValidateHTTPURL(req.URL.String(), true)
			},
		},
		logger: logger,
		meter:  newUsageMeter(),
	}
}

// Enabled reports whether platform integration is active.
func (c *Client) Enabled() bool {
	return c != nil && c.cfg.Enabled && c.cfg.Endpoint != ""
}

// LineageRef is a governed data-product / stream lineage pointer from the context plane.
type LineageRef struct {
	ID        string `json:"id,omitempty"`
	Product   string `json:"product,omitempty"`
	Subject   string `json:"subject,omitempty"`
	Source    string `json:"source,omitempty"`
	Freshness string `json:"freshness,omitempty"`
}

// ContextResult is a fail-open context plane response (text + optional lineage).
type ContextResult struct {
	Text    string
	Lineage []LineageRef
}

// QueryContext fetches governed operational context. Fail-open: errors yield empty result.
func (c *Client) QueryContext(ctx context.Context, workspace, query string) ContextResult {
	var empty ContextResult
	if !c.Enabled() || !c.cfg.ContextPlane {
		return empty
	}
	url := strings.TrimRight(c.cfg.Endpoint, "/") + "/v1/context/query"
	payload := map[string]any{
		"tenant":    c.cfg.Tenant,
		"workspace": workspace,
		"query":     query,
		"limit":     20,
	}
	if c.cfg.IncludeLineage {
		payload["include_lineage"] = true
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		c.logger.Debug("iomesh context: build request", "err", err)
		return empty
	}
	c.auth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Debug("iomesh context: request failed (fail-open)", "err", err)
		return empty
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		c.logger.Debug("iomesh context: non-OK (fail-open)", "status", resp.StatusCode)
		return empty
	}
	var out struct {
		Text    string       `json:"text"`
		Lineage []LineageRef `json:"lineage"`
		// Alternate shapes used by some brokers.
		Items []struct {
			Text    string       `json:"text"`
			Lineage []LineageRef `json:"lineage"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return empty
	}
	res := ContextResult{Text: strings.TrimSpace(out.Text), Lineage: out.Lineage}
	if res.Text == "" && len(out.Items) > 0 {
		var parts []string
		for _, it := range out.Items {
			if t := strings.TrimSpace(it.Text); t != "" {
				parts = append(parts, t)
			}
			res.Lineage = append(res.Lineage, it.Lineage...)
		}
		res.Text = strings.Join(parts, "\n")
	}
	return res
}

// ContextSnippet fetches governed operational context for injection into
// the agent system prompt (Knowledge / Operational Data Mesh). Fail-open:
// errors return empty string so the agent continues without mesh context.
// When lineage is present it is appended as a compact block for the model.
func (c *Client) ContextSnippet(ctx context.Context, workspace, query string) string {
	res := c.QueryContext(ctx, workspace, query)
	return FormatContextSnippet(res)
}

// FormatContextSnippet merges text + lineage for prompt injection.
func FormatContextSnippet(res ContextResult) string {
	var b strings.Builder
	if t := strings.TrimSpace(res.Text); t != "" {
		b.WriteString(t)
	}
	if len(res.Lineage) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("<iomesh-lineage>\n")
		for i, ref := range res.Lineage {
			if i >= 12 {
				b.WriteString("…\n")
				break
			}
			id := firstNonEmpty(ref.ID, ref.Product)
			parts := make([]string, 0, 4)
			if id != "" {
				parts = append(parts, id)
			}
			if ref.Subject != "" {
				parts = append(parts, "subject="+ref.Subject)
			}
			if ref.Source != "" {
				parts = append(parts, "source="+ref.Source)
			}
			if ref.Freshness != "" {
				parts = append(parts, "freshness="+ref.Freshness)
			}
			if len(parts) == 0 {
				continue
			}
			b.WriteString("- ")
			b.WriteString(strings.Join(parts, " · "))
			b.WriteByte('\n')
		}
		b.WriteString("</iomesh-lineage>")
	}
	return strings.TrimSpace(b.String())
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
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
// Always updates the local process meter; emits dept.* only when mesh is enabled.
func (c *Client) RecordLLMCall(meta router.CallMeta, usage router.Usage, err error) {
	if c == nil {
		return
	}
	if c.meter != nil {
		c.meter.Record(meta, usage, err)
	}
	if !c.Enabled() {
		return
	}
	payload := map[string]any{
		"model":       meta.ModelName,
		"model_id":    meta.ModelID,
		"duration_ms": meta.Duration.Milliseconds(),
		"attempts":    meta.Attempts,
		"fallback":    meta.FallbackUsed,
		"est_usd":     meta.EstimatedUSD,
		"tokens": map[string]int{
			"prompt":     usage.PromptTokens,
			"completion": usage.CompletionTokens,
			"total":      usage.TotalTokens,
		},
	}
	if err != nil {
		payload["error"] = security.Redact(err.Error())
	}
	// Best-effort background emit with short timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.Emit(ctx, DeptEvent{
		Type:    "dept.agent.llm_call",
		Payload: payload,
	})
}

// Usage returns a snapshot of local LLM metering for this process (not a remote dashboard).
func (c *Client) Usage() UsageSnapshot {
	if c == nil || c.meter == nil {
		return UsageSnapshot{}
	}
	return c.meter.Snapshot()
}

// PolicyEnabled reports whether remote policy evaluation is configured.
func (c *Client) PolicyEnabled() bool {
	return c != nil && c.Enabled() && (c.cfg.PolicyMode == PolicyAdvisory || c.cfg.PolicyMode == PolicyEnforce)
}

// PolicyMode returns the configured policy mode.
func (c *Client) PolicyMode() PolicyMode {
	if c == nil {
		return PolicyOff
	}
	return c.cfg.PolicyMode
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
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("iomesh health: http %d", resp.StatusCode)
	}
	return nil
}
