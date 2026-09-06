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
	// CatalogPlane enables GET catalog data-product discovery (fail-open).
	CatalogPlane bool
	// InjectCatalog adds a short catalog snippet into agent turns when true.
	InjectCatalog bool
	// OrgID is optional PlanGate / entitlements org (X-IOMesh-Org on dept emit + memory).
	OrgID string
	// WorkspaceID is optional workspace scope (X-IOMesh-Workspace on dept emit + memory).
	WorkspaceID string
	// DualWrite mirrors agent [memory].dual_write / IOMESH_MEMORY_DUAL_WRITE for dogfood
	// JSON evidence (does not gate the memory_ingest probe; default false).
	DualWrite bool
	// MemoryEndpoint is optional base URL for sync POST /v1/memory/retrieve (memory sidecar).
	// When set, RetrieveMemory prefers this over Endpoint (stage warm plane vs broker-only mesh).
	// Env: IOMESH_MEMORY_ENDPOINT / MEMORY_SIDECAR_URL · config [memory].endpoint
	MemoryEndpoint string
	// Role is optional federated mesh role for X-IOMesh-Role (operator|admin|agent|auditor|viewer|memory|custom).
	// Fail-open: empty omits the header (local/dev honesty; not full IdP RBAC). s675/s687 / mesh s671/s686 peer.
	// role=memory → default filter tenant.memory.> via DefaultMemoryPullFilterForRole.
	Role string
	// PullAllowSuffix is optional comma-separated literal tokens for role=custom
	// (X-IOMesh-Pull-Allow-Suffix). Fail-open: empty omits the header. s675.
	PullAllowSuffix string
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
	// Optional memory sidecar base (stage warm plane); clear if invalid.
	if ep := strings.TrimSpace(cfg.MemoryEndpoint); ep != "" {
		if err := security.ValidateHTTPURL(ep, true); err != nil {
			logger.Warn("iomesh: invalid memory_endpoint; clearing", "err", err)
			cfg.MemoryEndpoint = ""
		} else {
			cfg.MemoryEndpoint = strings.TrimRight(ep, "/")
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

// applyEntitlementHeaders sets X-IOMesh-Org / X-IOMesh-Workspace for PlanGate / multi-tenant metering.
// Parity with platform metering Org / Workspace headers (memory dual-write + dept emit).
func (c *Client) applyEntitlementHeaders(req *http.Request) {
	if c == nil || req == nil {
		return
	}
	if org := strings.TrimSpace(c.cfg.OrgID); org != "" {
		req.Header.Set("X-IOMesh-Org", org)
	}
	if ws := strings.TrimSpace(c.cfg.WorkspaceID); ws != "" {
		req.Header.Set("X-IOMesh-Workspace", ws)
	}
}

// MemoryEndpointConfigured reports whether a dedicated memory sidecar base URL is set.
func (c *Client) MemoryEndpointConfigured() bool {
	return c != nil && strings.TrimSpace(c.cfg.MemoryEndpoint) != ""
}

// MemoryBaseURL returns the HTTP base for sync memory retrieve (sidecar if set, else mesh endpoint).
func (c *Client) MemoryBaseURL() string {
	if c == nil {
		return ""
	}
	if ep := strings.TrimSpace(c.cfg.MemoryEndpoint); ep != "" {
		return strings.TrimRight(ep, "/")
	}
	return strings.TrimRight(strings.TrimSpace(c.cfg.Endpoint), "/")
}

// SyncMemoryReady reports whether lean HTTP sync retrieve can be attempted.
// True when mesh is enabled or a dedicated memory sidecar endpoint is configured.
func (c *Client) SyncMemoryReady() bool {
	return c != nil && (c.Enabled() || c.MemoryEndpointConfigured()) && c.MemoryBaseURL() != ""
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
	if err := c.EmitErr(ctx, ev); err != nil && c.logger != nil {
		c.logger.Debug("iomesh emit failed (fail-open)", "err", err, "type", ev.Type)
	}
}

// RecordLLMCall implements router.MetricsSink for usage metering.
// Always updates the local process meter; when mesh is enabled and emit_dept_streams is on,
// publishes dept.agent.llm_call for platform remote metering dashboards (multi-tenant via org/workspace headers).
func (c *Client) RecordLLMCall(meta router.CallMeta, usage router.Usage, err error) {
	if c == nil {
		return
	}
	if c.meter != nil {
		c.meter.Record(meta, usage, err)
	}
	if !c.Enabled() || !c.cfg.EmitDeptStreams {
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
	if t := strings.TrimSpace(c.cfg.Tenant); t != "" {
		payload["tenant"] = t
	}
	if org := strings.TrimSpace(c.cfg.OrgID); org != "" {
		payload["org"] = org
	}
	if ws := strings.TrimSpace(c.cfg.WorkspaceID); ws != "" {
		payload["workspace"] = ws
	}
	if err != nil {
		payload["error"] = security.Redact(err.Error())
	}
	// Best-effort emit with short timeout (same path as EmitErr: org/workspace headers).
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = c.EmitErr(ctx, DeptEvent{
		Type:    "dept.agent.llm_call",
		Tenant:  c.cfg.Tenant,
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

// CatalogEnabled reports whether catalog discovery is configured.
func (c *Client) CatalogEnabled() bool {
	return c != nil && c.Enabled() && c.cfg.CatalogPlane
}

// InjectCatalog reports whether agent turns should inject a catalog snippet.
func (c *Client) InjectCatalog() bool {
	return c != nil && c.CatalogEnabled() && c.cfg.InjectCatalog
}

// Endpoint returns the configured mesh endpoint (may be empty).
func (c *Client) Endpoint() string {
	if c == nil {
		return ""
	}
	return c.cfg.Endpoint
}

// Tenant returns the configured tenant (may be empty).
func (c *Client) Tenant() string {
	if c == nil {
		return ""
	}
	return c.cfg.Tenant
}

// OrgID returns the configured org id for PlanGate / multi-tenant headers (may be empty).
// Used by agent IntegrationsStatus for residual-honest list_org_connector_installs (s1271).
// Empty means skip the MCP call with residual note — never invent empty-as-none installs.
func (c *Client) OrgID() string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.cfg.OrgID)
}

// StatusLine is a one-line operator summary for TUI /mesh.
func (c *Client) StatusLine() string {
	if c == nil || !c.Enabled() {
		line := "mesh: disabled (offline-first)"
		if v := ProductVersion(); v != "" {
			line += " · version=" + v
		}
		return line
	}
	parts := []string{"mesh: enabled", "endpoint=" + c.cfg.Endpoint}
	if c.cfg.Tenant != "" {
		parts = append(parts, "tenant="+c.cfg.Tenant)
	}
	parts = append(parts,
		fmt.Sprintf("context=%v", c.cfg.ContextPlane),
		fmt.Sprintf("lineage=%v", c.cfg.IncludeLineage),
		fmt.Sprintf("catalog=%v", c.cfg.CatalogPlane),
		fmt.Sprintf("policy=%s", c.cfg.PolicyMode),
		fmt.Sprintf("emit=%v", c.cfg.EmitDeptStreams),
		"ua="+UserAgent(),
	)
	if v := ProductVersion(); v != "" {
		parts = append(parts, "version="+v)
	}
	return strings.Join(parts, " · ")
}

// userAgent is the outbound User-Agent for mesh HTTP (operator supportability).
// Set from main via SetUserAgent("iomesh-tui/"+version); default is "iomesh-tui".
var userAgent = "iomesh-tui"

// SetUserAgent sets the package-level User-Agent used by all Clients (empty keeps current).
func SetUserAgent(ua string) {
	if s := strings.TrimSpace(ua); s != "" {
		userAgent = s
	}
}

// UserAgent returns the current package User-Agent string.
func UserAgent() string { return userAgent }

// productVersion is the package product/binary version for StatusLine and dogfood
// evidence. Set from main via SetProductVersion(version); default is empty.
var productVersion string

// SetProductVersion sets the package-level product version (empty keeps current).
func SetProductVersion(v string) {
	if s := strings.TrimSpace(v); s != "" {
		productVersion = s
	}
}

// ProductVersion returns the current package product version string (may be empty).
func ProductVersion() string { return productVersion }

func (c *Client) auth(req *http.Request) {
	if req == nil {
		return
	}
	req.Header.Set("User-Agent", userAgent)
	// Console-minted secret lives in IOMESH_TOKEN; IOMESH_API_KEY remains a fallback.
	env := strings.TrimSpace(c.cfg.APIKeyEnv)
	if env == "" {
		env = "IOMESH_TOKEN"
	}
	key := os.Getenv(env)
	if key == "" && env != "IOMESH_TOKEN" {
		key = os.Getenv("IOMESH_TOKEN")
	}
	if key == "" {
		key = os.Getenv("IOMESH_API_KEY")
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	if c.cfg.Tenant != "" {
		req.Header.Set("X-IOMesh-Tenant", c.cfg.Tenant)
	}
	// Federated pull ACL headers (s675 / mesh M4+ roles + s671 custom suffix). Fail-open: empty → omit.
	if role := strings.TrimSpace(c.cfg.Role); role != "" {
		req.Header.Set("X-IOMesh-Role", role)
	}
	if suffix := strings.TrimSpace(c.cfg.PullAllowSuffix); suffix != "" {
		req.Header.Set("X-IOMesh-Pull-Allow-Suffix", suffix)
	}
	// Org/Workspace on every request (ListStreams / ListStreamMessages included).
	c.applyEntitlementHeaders(req)
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
	c.auth(req)
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
