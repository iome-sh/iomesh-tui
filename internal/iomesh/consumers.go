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

// ResolveMeshPullAuth resolves federated role + pull-allow-suffix for mesh client
// Config (s675/s681/s684). Flag values override config ([memory].pull_role /
// pull_allow_suffix). Whitespace-only is treated as empty. Fail-open empty →
// Client.auth omits X-IOMesh-Role / X-IOMesh-Pull-Allow-Suffix. Pure: no I/O.
//
// Beta federated ACL — not full mesh RBAC GA. Peer aion continuum (s680/s683).
func ResolveMeshPullAuth(roleFlag, suffixFlag, configRole, configSuffix string) (role, allowSuffix string) {
	role = strings.TrimSpace(roleFlag)
	if role == "" {
		role = strings.TrimSpace(configRole)
	}
	allowSuffix = strings.TrimSpace(suffixFlag)
	if allowSuffix == "" {
		allowSuffix = strings.TrimSpace(configSuffix)
	}
	return role, allowSuffix
}

// ResolveConsumerCreateAuthAndFilter resolves role, pull-allow-suffix, and effective
// filter_subject for mesh consumer create (s681). Flag values override config
// ([memory].pull_role / pull_allow_suffix). Empty filter uses
// DefaultMemoryPullFilterForRole (same role-aware defaults as memory pull s678).
// Tenant should be the IOMesh tenant (mesh command pattern). Pure: no I/O.
//
// Beta federated ACL headers + defaults — fail-open when role/suffix empty
// (headers omitted); not full mesh RBAC GA. Peer aion s680 continuum.
func ResolveConsumerCreateAuthAndFilter(explicitFilter, tenant, roleFlag, suffixFlag, configRole, configSuffix string) (filter, role, allowSuffix string) {
	role, allowSuffix = ResolveMeshPullAuth(roleFlag, suffixFlag, configRole, configSuffix)
	filter = DefaultMemoryPullFilterForRole(explicitFilter, tenant, role, allowSuffix)
	return filter, role, allowSuffix
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

// ConsumerInfoPrint is a CLI-side print DTO for consumer create JSON output.
// Always emits filter_subject, pull_role, and pull_allow_suffix as strings
// (empty when unset) so CI scrapers can key stable identity without omitempty
// gaps. Separate from wire ConsumerInfo so broker decode stays lean.
//
// s696: peer status/wait/dogfood pull identity continuum. Beta federated ACL;
// fail-open empty; dual_write default OFF; not full mesh RBAC GA.
type ConsumerInfoPrint struct {
	Stream          string `json:"stream"`
	Name            string `json:"name"`
	FilterSubject   string `json:"filter_subject"`
	AckFloor        uint64 `json:"ack_floor"`
	PendingCount    int    `json:"pending_count"`
	PullRole        string `json:"pull_role"`
	PullAllowSuffix string `json:"pull_allow_suffix"`
}

// NewConsumerInfoPrint builds a print DTO from broker ConsumerInfo plus
// resolved pull auth identity (role / allow-suffix from s681/s684). Empty
// role/suffix are always emitted as "" (not omitted).
func NewConsumerInfoPrint(info ConsumerInfo, role, allowSuffix string) ConsumerInfoPrint {
	return ConsumerInfoPrint{
		Stream:          info.Stream,
		Name:            info.Name,
		FilterSubject:   info.FilterSubject,
		AckFloor:        info.AckFloor,
		PendingCount:    info.PendingCount,
		PullRole:        role,
		PullAllowSuffix: allowSuffix,
	}
}

// FormatConsumerInfo is a multi-line view for one durable consumer (CLI).
// Pure helper with no network I/O. Always emits filter_subject, pull_role, and
// pull_allow_suffix (empty string when unset). Delegates to
// FormatConsumerInfoWithAuth with empty auth identity.
func FormatConsumerInfo(info ConsumerInfo) string {
	return FormatConsumerInfoWithAuth(info, "", "")
}

// FormatConsumerInfoWithAuth is a multi-line operator view for one durable
// consumer including resolved pull auth identity (s696). Always emits:
//   - filter_subject: (empty when unset)
//   - pull_role: (empty when unset)
//   - pull_allow_suffix: (empty when unset)
//
// Pure helper with no network I/O. Use from mesh consumer create with role/
// suffix resolved via ResolveConsumerCreateAuthAndFilter (s681) so scrapers
// always see pull identity next to filter_subject. Beta federated ACL;
// fail-open empty; dual_write default OFF; not full mesh RBAC GA; peer aion
// s695 sales claim continuum.
func FormatConsumerInfoWithAuth(info ConsumerInfo, role, allowSuffix string) string {
	var b strings.Builder
	b.WriteString("iomesh consumer\n")
	fmt.Fprintf(&b, "stream:          %s\n", info.Stream)
	fmt.Fprintf(&b, "name:            %s\n", info.Name)
	fmt.Fprintf(&b, "ack_floor:       %d\n", info.AckFloor)
	fmt.Fprintf(&b, "pending_count:   %d\n", info.PendingCount)
	fmt.Fprintf(&b, "filter_subject:  %s\n", info.FilterSubject)
	fmt.Fprintf(&b, "pull_role:       %s\n", role)
	fmt.Fprintf(&b, "pull_allow_suffix: %s\n", allowSuffix)
	return b.String()
}

// ConsumerFetchPrint is a CLI-side print DTO for mesh consumer fetch JSON.
// Always emits stream/name (consumer), pull_role / pull_allow_suffix (empty
// string honest when unset), batch / max_wait_ms knobs, count, and messages
// so CI scrapers can key stable pull identity without omitempty gaps. Wire
// []StreamMessage stays lean (no auth fields).
//
// s708: peer create FormatConsumerInfo s696 + memory-pull s705 continuum;
// peer aion s707 gate completeness. Beta · offline unit ≠ live APPLY · empty
// role honest · dual_write OFF · not full mesh RBAC GA · does not invent
// fetch success from identity fields alone.
type ConsumerFetchPrint struct {
	Stream          string          `json:"stream"`
	Name            string          `json:"name"`
	PullRole        string          `json:"pull_role"`
	PullAllowSuffix string          `json:"pull_allow_suffix"`
	Batch           int             `json:"batch"`
	MaxWaitMS       int             `json:"max_wait_ms"`
	Count           int             `json:"count"`
	Messages        []StreamMessage `json:"messages"`
}

// NewConsumerFetchPrint builds a fetch print DTO from resolved fetch identity
// (s684 role/suffix), knobs, and wire messages. Nil msgs becomes empty slice;
// Count is always len(messages). Empty role/suffix always emit as "".
func NewConsumerFetchPrint(stream, name, role, allowSuffix string, batch, maxWaitMS int, msgs []StreamMessage) ConsumerFetchPrint {
	if msgs == nil {
		msgs = []StreamMessage{}
	}
	return ConsumerFetchPrint{
		Stream:          stream,
		Name:            name,
		PullRole:        role,
		PullAllowSuffix: allowSuffix,
		Batch:           batch,
		MaxWaitMS:       maxWaitMS,
		Count:           len(msgs),
		Messages:        msgs,
	}
}

// FormatConsumerFetch is a multi-line operator view for mesh consumer fetch
// (s708). Always emits pull identity + knobs + count, then the message table
// (FormatStreamMessages). Pure helper; no I/O. Does not invent success from
// identity — messages/count come from the fetch result only.
func FormatConsumerFetch(p ConsumerFetchPrint) string {
	var b strings.Builder
	b.WriteString("iomesh consumer fetch\n")
	fmt.Fprintf(&b, "stream:            %s\n", p.Stream)
	fmt.Fprintf(&b, "name:              %s\n", p.Name)
	fmt.Fprintf(&b, "pull_role:         %s\n", p.PullRole)
	fmt.Fprintf(&b, "pull_allow_suffix: %s\n", p.PullAllowSuffix)
	fmt.Fprintf(&b, "batch:             %d\n", p.Batch)
	fmt.Fprintf(&b, "max_wait_ms:       %d\n", p.MaxWaitMS)
	fmt.Fprintf(&b, "count:             %d\n", p.Count)
	label := p.Stream + "/" + p.Name
	b.WriteString(FormatStreamMessages(label, p.Messages))
	return b.String()
}

// FormatConsumerFetchJSON returns indented JSON for stage CI / scrapers.
// Always emits all ConsumerFetchPrint fields without omitempty gaps.
func FormatConsumerFetchJSON(p ConsumerFetchPrint) string {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return `{"error":"consumer fetch json marshal failed"}` + "\n"
	}
	return string(b) + "\n"
}

// ConsumerDeletePrint is a CLI-side print DTO for mesh consumer delete JSON.
// Always emits ok / stream / name / pull_role / pull_allow_suffix (empty string
// honest when unset) so scrapers see pull identity on delete success without
// omitempty gaps.
//
// s708: peer create FormatConsumerInfo s696 + fetch identity continuum; peer
// aion s707. Beta · offline unit ≠ live APPLY · empty role honest · dual_write
// OFF · not full mesh RBAC GA · does not invent delete success from identity.
type ConsumerDeletePrint struct {
	OK              bool   `json:"ok"`
	Stream          string `json:"stream"`
	Name            string `json:"name"`
	PullRole        string `json:"pull_role"`
	PullAllowSuffix string `json:"pull_allow_suffix"`
}

// NewConsumerDeletePrint builds a delete success print DTO with always-emit
// pull identity from resolved s684 auth. OK is always true (call only after
// DeleteConsumer returns nil).
func NewConsumerDeletePrint(stream, name, role, allowSuffix string) ConsumerDeletePrint {
	return ConsumerDeletePrint{
		OK:              true,
		Stream:          stream,
		Name:            name,
		PullRole:        role,
		PullAllowSuffix: allowSuffix,
	}
}

// FormatConsumerDelete is a multi-line operator view for mesh consumer delete
// success (s708). Always emits pull_role / pull_allow_suffix (empty when unset)
// next to stream/name. Pure helper; no I/O.
func FormatConsumerDelete(p ConsumerDeletePrint) string {
	var b strings.Builder
	b.WriteString("PASS mesh consumer delete\n")
	fmt.Fprintf(&b, "stream:            %s\n", p.Stream)
	fmt.Fprintf(&b, "name:              %s\n", p.Name)
	fmt.Fprintf(&b, "pull_role:         %s\n", p.PullRole)
	fmt.Fprintf(&b, "pull_allow_suffix: %s\n", p.PullAllowSuffix)
	return b.String()
}

// FormatConsumerDeleteJSON returns indented JSON for stage CI / scrapers.
// Always emits all ConsumerDeletePrint fields without omitempty gaps.
func FormatConsumerDeleteJSON(p ConsumerDeletePrint) string {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return `{"error":"consumer delete json marshal failed"}` + "\n"
	}
	return string(b) + "\n"
}
