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
	streamMemoryIngest   = "MEMORY_INGEST"
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
