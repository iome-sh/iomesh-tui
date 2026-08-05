package iomesh

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// MemoryEnvelope is the async MEMORY_INGEST stream payload shape.
// Mirrors iomesh-client-sdk-go MemoryEnvelope (lean subset; no SDK dependency).
type MemoryEnvelope struct {
	Type       string `json:"type"` // memory_ingest
	SessionID  string `json:"session_id,omitempty"`
	Role       string `json:"role,omitempty"`
	Content    string `json:"content,omitempty"`
	EventTime  string `json:"event_time,omitempty"`  // RFC3339
	SessionSeq int    `json:"session_seq,omitempty"` // monotonic within session
}

// MemoryPubAck is a minimal publish acknowledgement from the broker.
type MemoryPubAck struct {
	Stream  string `json:"stream,omitempty"`
	Seq     uint64 `json:"seq,omitempty"`
	Subject string `json:"subject,omitempty"`
}

const (
	memoryEnvelopeIngest = "memory_ingest"
	memoryEnvelopeRecall = "memory_recall"
	streamMemoryIngest   = "MEMORY_INGEST"
	streamMemoryRPC      = "MEMORY_RPC"
)

// PublishMemoryIngest posts an async memory_ingest envelope to
// POST /v1/streams/MEMORY_INGEST/publish (subject = tenant+".memory.ingest.turn").
// Payload is base64-encoded JSON to match the public SDK Publish wire format.
// Returns an error on validation or transport failure (callers dual-write fail-open).
func (c *Client) PublishMemoryIngest(ctx context.Context, tenantID string, env MemoryEnvelope) (*MemoryPubAck, error) {
	if c == nil || !c.Enabled() {
		return nil, fmt.Errorf("iomesh: mesh client not enabled")
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		tenantID = strings.TrimSpace(c.cfg.Tenant)
	}
	if tenantID == "" {
		return nil, fmt.Errorf("iomesh: tenant_id required for memory ingest")
	}
	if strings.TrimSpace(env.Content) == "" {
		return nil, fmt.Errorf("iomesh: content required for memory ingest")
	}
	if strings.TrimSpace(env.Type) == "" {
		env.Type = memoryEnvelopeIngest
	}

	payload, err := json.Marshal(env)
	if err != nil {
		return nil, err
	}
	subject := tenantID + ".memory.ingest.turn"
	body := map[string]any{
		"subject": subject,
		"payload": base64.StdEncoding.EncodeToString(payload),
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := strings.TrimRight(c.cfg.Endpoint, "/") + "/v1/streams/" + streamMemoryIngest + "/publish"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	c.auth(req)
	req.Header.Set("Content-Type", "application/json")
	c.applyEntitlementHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Debug("iomesh memory ingest publish failed (fail-open)", "err", err)
		}
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("iomesh memory ingest: http %d", resp.StatusCode)
		if c.logger != nil {
			c.logger.Debug("iomesh memory ingest non-OK (fail-open)", "status", resp.StatusCode)
		}
		return nil, err
	}

	var ack MemoryPubAck
	if len(respBody) > 0 {
		_ = json.Unmarshal(respBody, &ack)
	}
	if ack.Stream == "" {
		ack.Stream = streamMemoryIngest
	}
	if ack.Subject == "" {
		ack.Subject = subject
	}
	return &ack, nil
}

// PublishMemoryRecall posts an async memory_recall request to
// POST /v1/streams/MEMORY_RPC/publish (subject = tenant+".memory.retrieve.request").
// Mirrors public SDK RequestMemoryRecall (fire-and-forget; no sync hits in response).
// Optional sessionID correlates temporal recall with dual-write ingest (dogfood s247).
func (c *Client) PublishMemoryRecall(ctx context.Context, tenantID, query string, limit int, sessionID string) (*MemoryPubAck, error) {
	if c == nil || !c.Enabled() {
		return nil, fmt.Errorf("iomesh: mesh client not enabled")
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		tenantID = strings.TrimSpace(c.cfg.Tenant)
	}
	if tenantID == "" {
		return nil, fmt.Errorf("iomesh: tenant_id required for memory recall")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("iomesh: query required for memory recall")
	}

	bodyMap := map[string]any{
		"type":      memoryEnvelopeRecall,
		"tenant_id": tenantID,
		"query":     query,
	}
	if limit > 0 {
		bodyMap["limit"] = limit
	}
	if sid := strings.TrimSpace(sessionID); sid != "" {
		bodyMap["session_id"] = sid
	}
	payload, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, err
	}
	subject := tenantID + ".memory.retrieve.request"
	body := map[string]any{
		"subject": subject,
		"payload": base64.StdEncoding.EncodeToString(payload),
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := strings.TrimRight(c.cfg.Endpoint, "/") + "/v1/streams/" + streamMemoryRPC + "/publish"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	c.auth(req)
	req.Header.Set("Content-Type", "application/json")
	c.applyEntitlementHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Debug("iomesh memory recall publish failed (fail-open)", "err", err)
		}
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("iomesh memory recall: http %d", resp.StatusCode)
		if c.logger != nil {
			c.logger.Debug("iomesh memory recall non-OK (fail-open)", "status", resp.StatusCode)
		}
		return nil, err
	}

	var ack MemoryPubAck
	if len(respBody) > 0 {
		_ = json.Unmarshal(respBody, &ack)
	}
	if ack.Stream == "" {
		ack.Stream = streamMemoryRPC
	}
	if ack.Subject == "" {
		ack.Subject = subject
	}
	return &ack, nil
}

// MemoryHit is one hit from sync HTTP retrieve / related (platform memory sidecar).
type MemoryHit struct {
	ID          string  `json:"id"`
	Summary     string  `json:"summary"`
	Full        string  `json:"full"`
	Score       float64 `json:"score,omitempty"`
	Confidence  float64 `json:"confidence,omitempty"`
	Timestamp   string  `json:"timestamp,omitempty"`
	TurnID      string  `json:"turn_id,omitempty"`
	HopDistance int     `json:"hop_distance,omitempty"` // multi-hop related (s1135); 0 when absent
}

// MemoryRetrieveResult is the sync POST /v1/memory/retrieve (or /related) response.
type MemoryRetrieveResult struct {
	Memories []MemoryHit `json:"memories"`
	// Path is the successful API path (v1 or v5 fallback).
	Path string `json:"-"`
}

// MemoryRetrieveOptions are request fields for sync POST /v1|/v5/memory/retrieve.
// Temporal filters (Since/Until/SessionSeq) map to platform sidecar → kernel
// SearchMemoryWithOptions when non-empty. RFC3339 strings for Since/Until.
// Parity with iomesh-client-sdk-go MemoryRetrieveRequest (s1068).
type MemoryRetrieveOptions struct {
	Query      string
	Limit      int
	SessionID  string
	SessionSeq int    // query session order for temporal recall; omit when 0
	Since      string // RFC3339 inclusive lower bound
	Until      string // RFC3339 inclusive upper bound
}

// MemoryRelatedOptions are request fields for sync POST /v1|/v5/memory/related (s1135).
// Multi-hop lite associative recall over entity graph + entry entity tags.
// At least one of SeedEntity or Query is required. Not full graph RAG; not Memory GA.
// Parity with peer SDK RetrieveMemoryRelated / platform MCP memory_related.
type MemoryRelatedOptions struct {
	SeedEntity string // e.g. person:alice
	Query      string // optional seed query (derive entities from top hits)
	MaxHops    int    // BFS hops (default 2; platform typically clamps 1..4)
	Limit      int
	SessionID  string
}

// RetrieveMemory performs request/response hybrid recall against the memory sidecar HTTP API.
// Thin wrapper around RetrieveMemoryWithOptions (query/limit/session_id only).
// Tries POST /v1/memory/retrieve then /v5/memory/retrieve (same handler on the broker / platform).
// Base URL: cfg.MemoryEndpoint when set (stage warm sidecar), else mesh Endpoint.
// This is NOT MEMORY_RPC fire-and-forget — empty hits are a successful 200 with memories=[].
func (c *Client) RetrieveMemory(ctx context.Context, tenantID, query string, limit int, sessionID string) (*MemoryRetrieveResult, error) {
	return c.RetrieveMemoryWithOptions(ctx, tenantID, MemoryRetrieveOptions{
		Query:     query,
		Limit:     limit,
		SessionID: sessionID,
	})
}

// RetrieveMemoryWithOptions is sync hybrid recall with optional temporal filters
// (since/until/session_seq). Non-empty temporal fields are included in the JSON body
// so the platform sidecar can narrow hits (efficiency + time-windowed auto-recall).
func (c *Client) RetrieveMemoryWithOptions(ctx context.Context, tenantID string, opts MemoryRetrieveOptions) (*MemoryRetrieveResult, error) {
	if c == nil || !c.SyncMemoryReady() {
		return nil, fmt.Errorf("iomesh: sync memory not configured (mesh endpoint or memory sidecar)")
	}
	base := c.MemoryBaseURL()
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		tenantID = strings.TrimSpace(c.cfg.Tenant)
	}
	if tenantID == "" {
		return nil, fmt.Errorf("iomesh: tenant_id required for memory retrieve")
	}
	query := strings.TrimSpace(opts.Query)
	sessionID := strings.TrimSpace(opts.SessionID)
	if query == "" && sessionID == "" {
		return nil, fmt.Errorf("iomesh: query or session_id required for memory retrieve")
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}
	bodyMap := map[string]any{
		"tenant_id": tenantID,
		"type":      memoryEnvelopeRecall,
		"query":     query,
		"limit":     limit,
	}
	if sessionID != "" {
		bodyMap["session_id"] = sessionID
	}
	if opts.SessionSeq != 0 {
		bodyMap["session_seq"] = opts.SessionSeq
	}
	if since := strings.TrimSpace(opts.Since); since != "" {
		bodyMap["since"] = since
	}
	if until := strings.TrimSpace(opts.Until); until != "" {
		bodyMap["until"] = until
	}
	raw, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for _, path := range []string{"/v1/memory/retrieve", "/v5/memory/retrieve"} {
		url := base + path
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
		if err != nil {
			lastErr = err
			continue
		}
		c.auth(req)
		req.Header.Set("Content-Type", "application/json")
		c.applyEntitlementHeaders(req)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			if c.logger != nil {
				c.logger.Debug("iomesh memory retrieve failed (fail-open)", "err", err, "path", path)
			}
			lastErr = err
			continue
		}
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			lastErr = fmt.Errorf("iomesh memory retrieve: http 404 path=%s", path)
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("iomesh memory retrieve: http %d path=%s", resp.StatusCode, path)
			if c.logger != nil {
				c.logger.Debug("iomesh memory retrieve non-OK (fail-open)", "status", resp.StatusCode, "path", path)
			}
			// 400 on v1 is unlikely to succeed on v5; still try once.
			continue
		}
		var out MemoryRetrieveResult
		if len(respBody) > 0 {
			if err := json.Unmarshal(respBody, &out); err != nil {
				lastErr = fmt.Errorf("iomesh memory retrieve decode: %w", err)
				continue
			}
		}
		if out.Memories == nil {
			out.Memories = []MemoryHit{}
		}
		out.Path = path
		return &out, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("iomesh memory retrieve: no path succeeded")
	}
	return nil, lastErr
}

// RetrieveMemoryRelated is sync multi-hop lite related recall (s1135).
// Tries POST /v1/memory/related then /v5/memory/related (same handler on the broker / platform).
// Base URL: cfg.MemoryEndpoint when set (stage warm sidecar), else mesh Endpoint.
// Empty hits are a successful 200 with memories=[]. Optional hop_distance on each hit.
// Fail-open callers treat transport/404 as fallback to MCP memory_related.
// Not full graph RAG; not product Memory GA; dual_write independent/OFF by default.
func (c *Client) RetrieveMemoryRelated(ctx context.Context, tenantID string, opts MemoryRelatedOptions) (*MemoryRetrieveResult, error) {
	if c == nil || !c.SyncMemoryReady() {
		return nil, fmt.Errorf("iomesh: sync memory not configured (mesh endpoint or memory sidecar)")
	}
	base := c.MemoryBaseURL()
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		tenantID = strings.TrimSpace(c.cfg.Tenant)
	}
	if tenantID == "" {
		return nil, fmt.Errorf("iomesh: tenant_id required for memory related")
	}
	seedEntity := strings.TrimSpace(opts.SeedEntity)
	query := strings.TrimSpace(opts.Query)
	if seedEntity == "" && query == "" {
		return nil, fmt.Errorf("iomesh: seed_entity or query required for memory related")
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}
	maxHops := opts.MaxHops
	if maxHops <= 0 {
		maxHops = 2
	}
	bodyMap := map[string]any{
		"tenant_id": tenantID,
		"type":      "memory_related",
		"limit":     limit,
		"max_hops":  maxHops,
	}
	if seedEntity != "" {
		bodyMap["seed_entity"] = seedEntity
	}
	if query != "" {
		bodyMap["query"] = query
	}
	if sid := strings.TrimSpace(opts.SessionID); sid != "" {
		bodyMap["session_id"] = sid
	}
	raw, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for _, path := range []string{"/v1/memory/related", "/v5/memory/related"} {
		url := base + path
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
		if err != nil {
			lastErr = err
			continue
		}
		c.auth(req)
		req.Header.Set("Content-Type", "application/json")
		c.applyEntitlementHeaders(req)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			if c.logger != nil {
				c.logger.Debug("iomesh memory related failed (fail-open)", "err", err, "path", path)
			}
			lastErr = err
			continue
		}
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			lastErr = fmt.Errorf("iomesh memory related: http 404 path=%s", path)
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("iomesh memory related: http %d path=%s", resp.StatusCode, path)
			if c.logger != nil {
				c.logger.Debug("iomesh memory related non-OK (fail-open)", "status", resp.StatusCode, "path", path)
			}
			continue
		}
		var out MemoryRetrieveResult
		if len(respBody) > 0 {
			if err := json.Unmarshal(respBody, &out); err != nil {
				lastErr = fmt.Errorf("iomesh memory related decode: %w", err)
				continue
			}
		}
		if out.Memories == nil {
			out.Memories = []MemoryHit{}
		}
		out.Path = path
		return &out, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("iomesh memory related: no path succeeded")
	}
	return nil, lastErr
}

// MemoryOpsDigestOptions are request fields for sync POST /v1|/v5/memory/ops_digest (s1200).
// Parity with aion s1198 HTTP / MCP ops_digest_export (s1197) and peer SDK ExportOpsDigest (s1199).
// Window defaults to day; Horizon defaults to ops when empty.
// Honesty: ops GA-path framing · knowledge/analytical Beta · never invent GA ·
// dual_write OFF · book-demo OFF · not product Memory GA · not full graph RAG.
type MemoryOpsDigestOptions struct {
	Window  string // day|week (default day)
	Horizon string // ops|knowledge|analytical|all (default ops)
	Limit   int    // max patterns and receipts each (platform default 20, max 50)
	AsOf    string // optional RFC3339 upper bound
}

// MemoryOpsDigestHonesty is residual-honest framing on the ops digest export.
type MemoryOpsDigestHonesty struct {
	OpsPulse         string `json:"ops_pulse"`
	Knowledge        string `json:"knowledge"`
	Analytical       string `json:"analytical"`
	NeverInventGA    bool   `json:"never_invent_ga"`
	DualWriteDefault string `json:"dual_write_default"`
	BookDemo         string `json:"book_demo"`
	Note             string `json:"note,omitempty"`
}

// MemoryOpsDigestPattern is one pattern signal in an ops digest (aion PatternSignal wire shape).
type MemoryOpsDigestPattern struct {
	ID        string  `json:"id,omitempty"`
	Kind      string  `json:"kind,omitempty"`
	Subject   string  `json:"subject,omitempty"`
	Count     int     `json:"count,omitempty"`
	Window    string  `json:"window,omitempty"`
	Score     float64 `json:"score,omitempty"`
	Summary   string  `json:"summary,omitempty"`
	FirstSeen string  `json:"first_seen,omitempty"`
	LastSeen  string  `json:"last_seen,omitempty"`
}

// MemoryOpsDigestReceipt is one timeline receipt in an ops digest pack.
type MemoryOpsDigestReceipt struct {
	ID         string `json:"id,omitempty"`
	EventTime  string `json:"event_time,omitempty"`
	Summary    string `json:"summary,omitempty"`
	SourceHint string `json:"source_hint,omitempty"`
}

// MemoryOpsDigestDecisionStub is a human-owned decision scaffold (not auto-apply).
type MemoryOpsDigestDecisionStub struct {
	Pattern                string   `json:"pattern,omitempty"`
	ReceiptsRef            []string `json:"receipts_ref,omitempty"`
	ProductOrGTMHypothesis string   `json:"product_or_gtm_hypothesis,omitempty"`
}

// MemoryOpsDigestResult is the ops digest export from POST /v1|/v5/memory/ops_digest.
type MemoryOpsDigestResult struct {
	Window       string                      `json:"window"`
	Horizon      string                      `json:"horizon"`
	AsOf         string                      `json:"as_of"`
	Since        string                      `json:"since,omitempty"`
	Honesty      MemoryOpsDigestHonesty      `json:"honesty"`
	Patterns     []MemoryOpsDigestPattern    `json:"patterns"`
	Receipts     []MemoryOpsDigestReceipt    `json:"receipts"`
	DecisionStub MemoryOpsDigestDecisionStub `json:"decision_stub"`
	// Path is the successful API path (v1 or v5 fallback).
	Path string `json:"-"`
}

// ExportOpsDigest is sync ops heartbeat digest export (s1200).
// Tries POST /v1/memory/ops_digest then /v5/memory/ops_digest (same handler on the sidecar / platform).
// Base URL: cfg.MemoryEndpoint when set (stage warm sidecar), else mesh Endpoint.
// Empty patterns/receipts are a successful 200 with [].
// Fail-open callers treat transport/404 as fallback to MCP ops_digest_export.
// Honesty: ops GA-path · knowledge/analytical Beta · never invent GA · dual_write OFF ·
// not product Memory GA · not full graph RAG. Human owns irreversible decisions.
func (c *Client) ExportOpsDigest(ctx context.Context, tenantID string, opts MemoryOpsDigestOptions) (*MemoryOpsDigestResult, error) {
	if c == nil || !c.SyncMemoryReady() {
		return nil, fmt.Errorf("iomesh: sync memory not configured (mesh endpoint or memory sidecar)")
	}
	base := c.MemoryBaseURL()
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		tenantID = strings.TrimSpace(c.cfg.Tenant)
	}
	if tenantID == "" {
		return nil, fmt.Errorf("iomesh: tenant_id required for memory ops_digest")
	}
	window := strings.ToLower(strings.TrimSpace(opts.Window))
	if window == "" {
		window = "day"
	}
	if window != "day" && window != "week" {
		return nil, fmt.Errorf("iomesh: window must be day or week")
	}
	horizon := strings.ToLower(strings.TrimSpace(opts.Horizon))
	if horizon == "" {
		horizon = "ops"
	}
	switch horizon {
	case "ops", "knowledge", "analytical", "all":
	default:
		return nil, fmt.Errorf("iomesh: horizon must be ops|knowledge|analytical|all")
	}
	bodyMap := map[string]any{
		"tenant_id": tenantID,
		"window":    window,
		"horizon":   horizon,
	}
	if opts.Limit > 0 {
		bodyMap["limit"] = opts.Limit
	}
	if asOf := strings.TrimSpace(opts.AsOf); asOf != "" {
		bodyMap["as_of"] = asOf
	}
	raw, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for _, path := range []string{"/v1/memory/ops_digest", "/v5/memory/ops_digest"} {
		url := base + path
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
		if err != nil {
			lastErr = err
			continue
		}
		c.auth(req)
		req.Header.Set("Content-Type", "application/json")
		c.applyEntitlementHeaders(req)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			if c.logger != nil {
				c.logger.Debug("iomesh memory ops_digest failed (fail-open)", "err", err, "path", path)
			}
			lastErr = err
			continue
		}
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			lastErr = fmt.Errorf("iomesh memory ops_digest: http 404 path=%s", path)
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("iomesh memory ops_digest: http %d path=%s", resp.StatusCode, path)
			if c.logger != nil {
				c.logger.Debug("iomesh memory ops_digest non-OK (fail-open)", "status", resp.StatusCode, "path", path)
			}
			continue
		}
		var out MemoryOpsDigestResult
		if len(respBody) > 0 {
			if err := json.Unmarshal(respBody, &out); err != nil {
				lastErr = fmt.Errorf("iomesh memory ops_digest decode: %w", err)
				continue
			}
		}
		if out.Patterns == nil {
			out.Patterns = []MemoryOpsDigestPattern{}
		}
		if out.Receipts == nil {
			out.Receipts = []MemoryOpsDigestReceipt{}
		}
		out.Path = path
		return &out, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("iomesh memory ops_digest: no path succeeded")
	}
	return nil, lastErr
}

// dogfoodSessionID returns a stable session id for memory dogfood probes.
func dogfoodSessionID(tenant string) string {
	tenant = strings.TrimSpace(tenant)
	if tenant == "" {
		return "mesh-dogfood"
	}
	return tenant + ".mesh-dogfood"
}
