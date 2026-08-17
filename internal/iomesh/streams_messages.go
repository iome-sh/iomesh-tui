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
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// StreamMessage is one persisted stream record from GET /v1/streams/{name}/messages.
// Wire payload is base64; Payload holds decoded bytes (JSON re-encodes as base64).
// Lean TUI surface — wire parity with broker consumer-fetch / replay message shape
// and public SDK ListStreamMessages intent (no SDK dependency).
type StreamMessage struct {
	Stream    string            `json:"stream,omitempty"`
	Seq       uint64            `json:"seq"`
	Subject   string            `json:"subject"`
	Partition int               `json:"partition,omitempty"`
	Payload   []byte            `json:"payload"`
	Headers   map[string]string `json:"headers,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// ListStreamMessagesOptions bounds a stream message list (query params).
// Zero values omit the corresponding query parameter (broker defaults apply).
type ListStreamMessagesOptions struct {
	FromSeq uint64 // from_seq
	ToSeq   uint64 // to_seq
	Limit   int    // limit
}

// ListStreamMessages returns stream messages via GET /v1/streams/{name}/messages.
// Query: from_seq, to_seq, limit (omitted when zero).
// Empty name returns an error. Non-2xx (including 403 replay gate, 404) returns error.
// When mesh is disabled / endpoint empty: returns (nil, error) with "mesh disabled".
// Decodes base64 payload into StreamMessage.Payload (invalid base64 → error).
func (c *Client) ListStreamMessages(ctx context.Context, name string, opts ListStreamMessagesOptions) ([]StreamMessage, error) {
	if c == nil || !c.Enabled() {
		return nil, fmt.Errorf("mesh disabled")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("iomesh streams: stream name required")
	}
	u, err := url.Parse(strings.TrimRight(c.cfg.Endpoint, "/") + "/v1/streams/" + url.PathEscape(name) + "/messages")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	if opts.FromSeq > 0 {
		q.Set("from_seq", strconv.FormatUint(opts.FromSeq, 10))
	}
	if opts.ToSeq > 0 {
		q.Set("to_seq", strconv.FormatUint(opts.ToSeq, 10))
	}
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	c.auth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, streamMessagesHTTPError(resp.StatusCode, body)
	}
	return decodeStreamMessages(body)
}

// streamMessagesHTTPError keeps the existing "http N" token (tests + scrapers) and
// appends a short body snippet so consume can map 403 replay_disabled without
// inventing a reason from status alone.
func streamMessagesHTTPError(status int, body []byte) error {
	snippet := strings.TrimSpace(string(body))
	if len(snippet) > 160 {
		snippet = snippet[:160]
	}
	if snippet == "" {
		return fmt.Errorf("iomesh streams: http %d", status)
	}
	return fmt.Errorf("iomesh streams: http %d: %s", status, snippet)
}

// IsReplayDisabled reports whether err is a broker 403 replay_disabled gate
// (GET /v1/streams/{name}/messages without tenant or AION_MEMORY_REPLAY_ENABLED).
func IsReplayDisabled(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "http 403") && strings.Contains(s, "replay_disabled")
}

type wireStreamMessage struct {
	Stream    string            `json:"stream"`
	Seq       uint64            `json:"seq"`
	Subject   string            `json:"subject"`
	Partition int               `json:"partition,omitempty"`
	Payload   string            `json:"payload"`
	Headers   map[string]string `json:"headers,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

func decodeStreamMessages(raw []byte) ([]StreamMessage, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return []StreamMessage{}, nil
	}
	var wires []wireStreamMessage
	if err := json.Unmarshal(raw, &wires); err != nil {
		var env struct {
			Messages []wireStreamMessage `json:"messages"`
		}
		if err2 := json.Unmarshal(raw, &env); err2 != nil {
			return nil, err
		}
		wires = env.Messages
	}
	if wires == nil {
		return []StreamMessage{}, nil
	}
	out := make([]StreamMessage, 0, len(wires))
	for _, w := range wires {
		var payload []byte
		if w.Payload != "" {
			decoded, err := base64.StdEncoding.DecodeString(w.Payload)
			if err != nil {
				return nil, fmt.Errorf("iomesh streams: decode payload seq %d: %w", w.Seq, err)
			}
			payload = decoded
		}
		out = append(out, StreamMessage{
			Stream:    w.Stream,
			Seq:       w.Seq,
			Subject:   w.Subject,
			Partition: w.Partition,
			Payload:   payload,
			Headers:   w.Headers,
			Timestamp: w.Timestamp,
		})
	}
	return out, nil
}

// StreamMessagePrint is a CLI-side print DTO for one nested stream message in
// scraper JSON (StreamMessagesPrint / ConsumerFetchPrint). Always emits stream
// ("" honest), seq, subject, partition (0 honest), payload (base64 []byte wire
// behaviour; nil → empty), headers (nil → {}), and timestamp as string
// ("" when zero; RFC3339 UTC when set — KVEntryPrint mold). Separate from wire
// StreamMessage so omitempty gaps never hide scraper keys.
//
// s723: nested always-emit residual after s720 outer envelope. Mold
// StreamInfoPrint s699/s702 / KVEntryPrint s714. Peer aion s722 residual.
// s759: completeness pin — docs + unit tests lock StreamMessagePrint (s723) with
// StreamMessagesPrint (s720) + StreamDeletePrint (s726) always-emit keys; does not
// invent new DTO fields or re-claim s720/s723/s726 product bodies. Peer aion
// s758 residual. item ≠ invent message success · dual_write OFF · offline unit ≠
// live APPLY · not full mesh RBAC GA · wire StreamMessage lean.
// Beta · offline unit ≠ live APPLY · empty/0/""/{} honest · dual_write default
// OFF · not full mesh RBAC GA · does not invent message success from fields alone.
type StreamMessagePrint struct {
	Stream    string            `json:"stream"`
	Seq       uint64            `json:"seq"`
	Subject   string            `json:"subject"`
	Partition int               `json:"partition"`
	Payload   []byte            `json:"payload"`
	Headers   map[string]string `json:"headers"`
	Timestamp string            `json:"timestamp"`
}

// NewStreamMessagePrint builds a nested message print DTO from wire StreamMessage.
// Nil payload → empty []byte; nil headers → empty map; zero Timestamp → "".
func NewStreamMessagePrint(m StreamMessage) StreamMessagePrint {
	p := StreamMessagePrint{
		Stream:    m.Stream,
		Seq:       m.Seq,
		Subject:   m.Subject,
		Partition: m.Partition,
		Payload:   m.Payload,
		Headers:   m.Headers,
	}
	if p.Payload == nil {
		p.Payload = []byte{}
	}
	if p.Headers == nil {
		p.Headers = map[string]string{}
	}
	if !m.Timestamp.IsZero() {
		p.Timestamp = m.Timestamp.UTC().Format(time.RFC3339)
	}
	return p
}

// streamMessagePrints maps wire messages to always-emit print DTOs.
// Nil msgs become []StreamMessagePrint{}.
func streamMessagePrints(msgs []StreamMessage) []StreamMessagePrint {
	if msgs == nil {
		return []StreamMessagePrint{}
	}
	out := make([]StreamMessagePrint, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, NewStreamMessagePrint(m))
	}
	return out
}

// StreamMessagesPrint is a CLI-side print DTO for mesh streams --messages --json.
// Always emits stream, from_seq / to_seq / limit (0 honest when unset), count,
// and messages (empty array when none; each element StreamMessagePrint nested
// always-emit, s723) so CI scrapers get a stable envelope rather than a bare
// []StreamMessage. Wire StreamMessage stays lean omitempty.
//
// s720 outer envelope + s723 nested message always-emit. Mold KVKeysPrint s714 /
// ConsumerFetchPrint s708. Peer aion s719/s722 residual.
// s759: completeness pin — docs + unit tests lock StreamMessagesPrint (s720) with
// StreamMessagePrint (s723) + StreamDeletePrint (s726) always-emit keys; does not
// invent new DTO fields or re-claim s720/s723/s726 product bodies. Peer aion
// s758 residual. envelope ≠ invent message success · dual_write OFF · offline
// unit ≠ live APPLY · not full mesh RBAC GA · wire StreamMessage lean.
// Beta · offline unit ≠ live APPLY · empty/0/""/{} honest · dual_write default
// OFF · not full mesh RBAC GA · does not invent message success from knobs alone.
type StreamMessagesPrint struct {
	Stream   string               `json:"stream"`
	FromSeq  uint64               `json:"from_seq"`
	ToSeq    uint64               `json:"to_seq"`
	Limit    int                  `json:"limit"`
	Count    int                  `json:"count"`
	Messages []StreamMessagePrint `json:"messages"`
}

// NewStreamMessagesPrint builds a messages print envelope. Nil msgs become
// []StreamMessagePrint{}; Count is always len(messages). Nested messages map
// via NewStreamMessagePrint (s723). Zero knobs emit as 0.
func NewStreamMessagesPrint(stream string, fromSeq, toSeq uint64, limit int, msgs []StreamMessage) StreamMessagesPrint {
	prints := streamMessagePrints(msgs)
	return StreamMessagesPrint{
		Stream:   stream,
		FromSeq:  fromSeq,
		ToSeq:    toSeq,
		Limit:    limit,
		Count:    len(prints),
		Messages: prints,
	}
}

// FormatStreamMessages renders a compact table for CLI operator inspection.
// Decoded payloads are shown as printable text when valid UTF-8; otherwise base64.
// Long payloads are truncated. Header includes count only (no query knobs);
// use FormatStreamMessagesPrint when from_seq/to_seq/limit should appear (s720).
func FormatStreamMessages(name string, msgs []StreamMessage) string {
	return formatStreamMessagesHeader(name, 0, 0, 0, streamMessagePrints(msgs), false)
}

// FormatStreamMessagesPrint is the operator text view for mesh streams --messages
// (s720). Always emits query knobs (from_seq / to_seq / limit; 0 honest) + count
// then the message table. Pure helper; no I/O. Does not invent success from knobs.
func FormatStreamMessagesPrint(p StreamMessagesPrint) string {
	return formatStreamMessagesHeader(p.Stream, p.FromSeq, p.ToSeq, p.Limit, p.Messages, true)
}

// FormatStreamMessagesJSON returns indented JSON for stage CI / scrapers.
// Always emits all StreamMessagesPrint fields without omitempty gaps.
// s741: Format*JSON helper completeness (DTO already always-emit s720/s723).
// Peer aion s740 residual. Mold FormatPubJSON / FormatCatalogJSON.
func FormatStreamMessagesJSON(p StreamMessagesPrint) string {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return `{"error":"stream messages json marshal failed"}` + "\n"
	}
	return string(b) + "\n"
}

func formatStreamMessagesHeader(name string, fromSeq, toSeq uint64, limit int, msgs []StreamMessagePrint, withKnobs bool) string {
	var b strings.Builder
	if withKnobs {
		fmt.Fprintf(&b, "iomesh stream messages name=%s count=%d from_seq=%d to_seq=%d limit=%d\n",
			name, len(msgs), fromSeq, toSeq, limit)
	} else {
		fmt.Fprintf(&b, "iomesh stream messages name=%s count=%d\n", name, len(msgs))
	}
	if len(msgs) == 0 {
		b.WriteString("(no messages)\n")
		return b.String()
	}
	fmt.Fprintf(&b, "%-8s %-36s %-20s %s\n", "SEQ", "SUBJECT", "TIME", "PREVIEW")
	for i, m := range msgs {
		if i >= 50 {
			fmt.Fprintf(&b, "… (%d more)\n", len(msgs)-50)
			break
		}
		fmt.Fprintf(&b, "%-8d %-36s %-20s %s\n",
			m.Seq,
			truncateRunes(m.Subject, 36),
			truncateRunes(m.Timestamp, 20),
			truncateRunes(FormatStreamPayloadPreview(m.Payload), 64),
		)
	}
	return b.String()
}

// FormatStreamPayloadPreview peels stacked base64 (rqlite persist) and unwraps
// observation.payload to event_type · repository (or title/summary).
// Text table only — JSON --messages still emits raw payload bytes.
// Pretty preview ≠ Connected / PULSE / live APPLY.
func FormatStreamPayloadPreview(payload []byte) string {
	raw := peelJSONPayload(payload)
	if title := unwrapObservationTitle(raw); title != "" {
		return title
	}
	if len(raw) > 0 && utf8.Valid(raw) && !strings.HasPrefix(strings.TrimSpace(string(raw)), "eyJ") {
		return string(raw)
	}
	if len(payload) == 0 {
		return ""
	}
	if utf8.Valid(payload) && !strings.HasPrefix(strings.TrimSpace(string(payload)), "eyJ") {
		return string(payload)
	}
	return base64.StdEncoding.EncodeToString(payload)
}

func peelJSONPayload(payload []byte) []byte {
	raw := []byte(strings.TrimSpace(string(payload)))
	if len(raw) == 0 {
		return nil
	}
	for i := 0; i < 4; i++ {
		if looksLikeJSONObject(raw) {
			return raw
		}
		next, err := decodeStackedBase64(raw)
		if err != nil || len(next) == 0 {
			return nil
		}
		raw = next
	}
	if looksLikeJSONObject(raw) {
		return raw
	}
	return nil
}

func decodeStackedBase64(raw []byte) ([]byte, error) {
	s := strings.TrimSpace(string(raw))
	if decoded, err := base64.StdEncoding.DecodeString(s); err == nil && len(decoded) > 0 {
		return decoded, nil
	}
	return base64.URLEncoding.DecodeString(s)
}

func looksLikeJSONObject(raw []byte) bool {
	s := strings.TrimSpace(string(raw))
	return strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[")
}

type streamEnvelopeHint struct {
	Payload json.RawMessage `json:"payload"`
	Title   string          `json:"title"`
	Summary string          `json:"summary"`
	Message string          `json:"message"`
	Text    string          `json:"text"`
}

type streamInnerHint struct {
	Title      string          `json:"title"`
	Summary    string          `json:"summary"`
	Message    string          `json:"message"`
	Text       string          `json:"text"`
	EventType  string          `json:"event_type"`
	Repository json.RawMessage `json:"repository"`
}

func unwrapObservationTitle(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var env streamEnvelopeHint
	innerRaw := raw
	if json.Unmarshal(raw, &env) == nil && len(env.Payload) > 0 && looksLikeJSONObject(env.Payload) {
		innerRaw = env.Payload
	}
	var inner streamInnerHint
	if len(innerRaw) > 0 {
		_ = json.Unmarshal(innerRaw, &inner)
	}
	repo := repositoryFullName(inner.Repository)
	event := strings.TrimSpace(inner.EventType)
	joined := strings.TrimSpace(strings.Trim(event+" · "+repo, " ·"))
	for _, cand := range []string{
		strings.TrimSpace(inner.Title),
		strings.TrimSpace(inner.Summary),
		strings.TrimSpace(inner.Message),
		strings.TrimSpace(inner.Text),
		joined,
		strings.TrimSpace(env.Title),
		strings.TrimSpace(env.Summary),
	} {
		if cand != "" && !strings.HasPrefix(cand, "eyJ") {
			return cand
		}
	}
	return ""
}

func repositoryFullName(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return strings.TrimSpace(s)
	}
	var obj struct {
		FullName string `json:"full_name"`
		Name     string `json:"name"`
	}
	if json.Unmarshal(raw, &obj) != nil {
		return ""
	}
	if t := strings.TrimSpace(obj.FullName); t != "" {
		return t
	}
	return strings.TrimSpace(obj.Name)
}
