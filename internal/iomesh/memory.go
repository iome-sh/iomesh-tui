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

// MemoryHit is one hit from sync HTTP retrieve (platform memory sidecar).
type MemoryHit struct {
	ID         string  `json:"id"`
	Summary    string  `json:"summary"`
	Full       string  `json:"full"`
	Score      float64 `json:"score,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
	Timestamp  string  `json:"timestamp,omitempty"`
	TurnID     string  `json:"turn_id,omitempty"`
}

// MemoryRetrieveResult is the sync POST /v1/memory/retrieve response.
type MemoryRetrieveResult struct {
	Memories []MemoryHit `json:"memories"`
	// Path is the successful API path (v1 or v5 fallback).
	Path string `json:"-"`
}

// RetrieveMemory performs request/response hybrid recall against the memory sidecar HTTP API.
// Tries POST /v1/memory/retrieve then /v5/memory/retrieve (same handler on the broker / platform).
// Base URL: cfg.MemoryEndpoint when set (stage warm sidecar), else mesh Endpoint.
// This is NOT MEMORY_RPC fire-and-forget — empty hits are a successful 200 with memories=[].
func (c *Client) RetrieveMemory(ctx context.Context, tenantID, query string, limit int, sessionID string) (*MemoryRetrieveResult, error) {
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
	query = strings.TrimSpace(query)
	sessionID = strings.TrimSpace(sessionID)
	if query == "" && sessionID == "" {
		return nil, fmt.Errorf("iomesh: query or session_id required for memory retrieve")
	}
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

// dogfoodSessionID returns a stable session id for memory dogfood probes.
func dogfoodSessionID(tenant string) string {
	tenant = strings.TrimSpace(tenant)
	if tenant == "" {
		return "mesh-dogfood"
	}
	return tenant + ".mesh-dogfood"
}
