package iomesh

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ConsumerInfo is durable pull-consumer metadata from POST /v1/streams/{stream}/consumers.
// Wire shape matches broker / iomesh-client-sdk-go ConsumerInfo.
// Lean TUI surface — no SDK dependency.
type ConsumerInfo struct {
	Stream        string `json:"stream"`
	Name          string `json:"name"`
	FilterSubject string `json:"filter_subject,omitempty"`
	AckFloor      uint64 `json:"ack_floor"`
	PendingCount  int    `json:"pending_count"`
}

// CreateConsumer registers a durable pull consumer via POST /v1/streams/{stream}/consumers.
// Body: name, optional filter_subject (max_deliver / ack_wait_sec omitted when zero).
// 201 decodes ConsumerInfo; 409 Conflict is treated as success (idempotent) returning
// &ConsumerInfo{Stream, Name} (name-only). Empty stream/name returns an error.
// Stream path segment is url.PathEscape'd. Mesh disabled → "mesh disabled".
// Mutating — CLI gates with create --yes.
func (c *Client) CreateConsumer(ctx context.Context, stream, name string, filter string) (*ConsumerInfo, error) {
	if c == nil || !c.Enabled() {
		return nil, fmt.Errorf("mesh disabled")
	}
	stream = strings.TrimSpace(stream)
	name = strings.TrimSpace(name)
	if stream == "" || name == "" {
		return nil, fmt.Errorf("iomesh consumer: stream and name required")
	}
	reqBody := struct {
		Name          string `json:"name"`
		FilterSubject string `json:"filter_subject,omitempty"`
		MaxDeliver    int    `json:"max_deliver,omitempty"`
		AckWaitSec    int    `json:"ack_wait_sec,omitempty"`
	}{
		Name:          name,
		FilterSubject: strings.TrimSpace(filter),
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	u := strings.TrimRight(c.cfg.Endpoint, "/") + "/v1/streams/" + url.PathEscape(stream) + "/consumers"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.auth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	// Idempotent: consumer already exists.
	if resp.StatusCode == http.StatusConflict {
		return &ConsumerInfo{Stream: stream, Name: name}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("iomesh consumer: http %d", resp.StatusCode)
	}
	var info ConsumerInfo
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &info); err != nil {
			return nil, err
		}
	}
	if info.Stream == "" {
		info.Stream = stream
	}
	if info.Name == "" {
		info.Name = name
	}
	return &info, nil
}

// ConsumerFetch pulls up to batch messages from a durable consumer via
// POST /v1/streams/{stream}/consumers/{name}/fetch.
// Body: {"batch", "max_wait_ms"}. maxWait <=0 defaults to 2s.
// Decodes base64 payload into StreamMessage.Payload (invalid base64 → raw bytes).
// Empty stream/name, batch<=0, non-2xx → error. Mesh disabled → "mesh disabled".
// Mutating long-poll — CLI gates with fetch --yes.
func (c *Client) ConsumerFetch(ctx context.Context, stream, name string, batch int, maxWait time.Duration) ([]StreamMessage, error) {
	if c == nil || !c.Enabled() {
		return nil, fmt.Errorf("mesh disabled")
	}
	stream = strings.TrimSpace(stream)
	name = strings.TrimSpace(name)
	if stream == "" || name == "" {
		return nil, fmt.Errorf("iomesh consumer: stream and name required")
	}
	if batch <= 0 {
		return nil, fmt.Errorf("iomesh consumer: batch must be > 0")
	}
	if maxWait <= 0 {
		maxWait = 2 * time.Second
	}
	reqBody := struct {
		Batch     int `json:"batch"`
		MaxWaitMS int `json:"max_wait_ms"`
	}{
		Batch:     batch,
		MaxWaitMS: int(maxWait / time.Millisecond),
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	u := strings.TrimRight(c.cfg.Endpoint, "/") + "/v1/streams/" + url.PathEscape(stream) +
		"/consumers/" + url.PathEscape(name) + "/fetch"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.auth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("iomesh consumer: http %d", resp.StatusCode)
	}
	return decodeConsumerFetch(raw, stream)
}

func decodeConsumerFetch(raw []byte, defaultStream string) ([]StreamMessage, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return []StreamMessage{}, nil
	}
	var env struct {
		Messages []wireStreamMessage `json:"messages"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Messages == nil {
		return []StreamMessage{}, nil
	}
	out := make([]StreamMessage, 0, len(env.Messages))
	for _, m := range env.Messages {
		payload, err := base64.StdEncoding.DecodeString(m.Payload)
		if err != nil {
			// Broker may return raw string payloads (not base64).
			payload = []byte(m.Payload)
		}
		stream := m.Stream
		if stream == "" {
			stream = defaultStream
		}
		out = append(out, StreamMessage{
			Stream:    stream,
			Seq:       m.Seq,
			Subject:   m.Subject,
			Partition: m.Partition,
			Payload:   payload,
			Headers:   m.Headers,
			Timestamp: m.Timestamp,
		})
	}
	return out, nil
}

// ConsumerAck acknowledges message sequences via
// POST /v1/streams/{stream}/consumers/{name}/ack body {"seqs":[...]}.
// Stream and name path segments are url.PathEscape'd.
// Empty stream/name/seqs → error. Non-2xx → explicit error.
// On 2xx, optionally decodes ack_floor from the response body (0 if absent/empty).
// Mesh disabled → "mesh disabled". Mutating — CLI gates with ack --yes.
func (c *Client) ConsumerAck(ctx context.Context, stream, name string, seqs ...uint64) (ackFloor uint64, err error) {
	return c.consumerAckNack(ctx, stream, name, "ack", seqs)
}

// ConsumerNack negatively-acknowledges message sequences via
// POST /v1/streams/{stream}/consumers/{name}/nack body {"seqs":[...]}.
// Stream and name path segments are url.PathEscape'd.
// Empty stream/name/seqs → error. Non-2xx → explicit error.
// On 2xx, optionally decodes ack_floor from the response body (0 if absent/empty).
// Mesh disabled → "mesh disabled". Mutating — CLI gates with nack --yes.
func (c *Client) ConsumerNack(ctx context.Context, stream, name string, seqs ...uint64) (ackFloor uint64, err error) {
	return c.consumerAckNack(ctx, stream, name, "nack", seqs)
}

func (c *Client) consumerAckNack(ctx context.Context, stream, name, op string, seqs []uint64) (uint64, error) {
	if c == nil || !c.Enabled() {
		return 0, fmt.Errorf("mesh disabled")
	}
	stream = strings.TrimSpace(stream)
	name = strings.TrimSpace(name)
	if stream == "" || name == "" {
		return 0, fmt.Errorf("iomesh consumer: stream and name required")
	}
	if len(seqs) == 0 {
		return 0, fmt.Errorf("iomesh consumer: seqs required")
	}
	reqBody := struct {
		Seqs []uint64 `json:"seqs"`
	}{Seqs: seqs}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return 0, err
	}
	u := strings.TrimRight(c.cfg.Endpoint, "/") + "/v1/streams/" + url.PathEscape(stream) +
		"/consumers/" + url.PathEscape(name) + "/" + op
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.auth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("iomesh consumer: http %d", resp.StatusCode)
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return 0, nil
	}
	var out struct {
		AckFloor uint64 `json:"ack_floor"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		// 2xx with non-JSON body: treat as success with floor 0.
		return 0, nil
	}
	return out.AckFloor, nil
}

// DeleteConsumer removes a durable pull consumer via
// DELETE /v1/streams/{stream}/consumers/{name}.
// Stream and name path segments are url.PathEscape'd.
// Empty stream/name → error. 2xx (including 204 No Content) is success; non-2xx → error.
// Mesh disabled → "mesh disabled". Mutating — CLI gates with delete --yes.
func (c *Client) DeleteConsumer(ctx context.Context, stream, name string) error {
	if c == nil || !c.Enabled() {
		return fmt.Errorf("mesh disabled")
	}
	stream = strings.TrimSpace(stream)
	name = strings.TrimSpace(name)
	if stream == "" || name == "" {
		return fmt.Errorf("iomesh consumer: stream and name required")
	}
	u := strings.TrimRight(c.cfg.Endpoint, "/") + "/v1/streams/" + url.PathEscape(stream) +
		"/consumers/" + url.PathEscape(name)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	c.auth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("iomesh consumer: http %d", resp.StatusCode)
	}
	return nil
}

// FormatConsumerInfo is a multi-line view for one durable consumer (CLI).
// Pure helper with no network I/O. filter_subject is omitted when empty.
func FormatConsumerInfo(info ConsumerInfo) string {
	var b strings.Builder
	b.WriteString("iomesh consumer\n")
	fmt.Fprintf(&b, "stream:          %s\n", info.Stream)
	fmt.Fprintf(&b, "name:            %s\n", info.Name)
	fmt.Fprintf(&b, "ack_floor:       %d\n", info.AckFloor)
	fmt.Fprintf(&b, "pending_count:   %d\n", info.PendingCount)
	if info.FilterSubject != "" {
		fmt.Fprintf(&b, "filter_subject:  %s\n", info.FilterSubject)
	}
	return b.String()
}
