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
		return nil, fmt.Errorf("iomesh streams: http %d", resp.StatusCode)
	}
	return decodeStreamMessages(body)
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

// FormatStreamMessages renders a compact table for CLI operator inspection.
// Decoded payloads are shown as printable text when valid UTF-8; otherwise base64.
// Long payloads are truncated.
func FormatStreamMessages(name string, msgs []StreamMessage) string {
	var b strings.Builder
	fmt.Fprintf(&b, "iomesh stream messages name=%s count=%d\n", name, len(msgs))
	if len(msgs) == 0 {
		b.WriteString("(no messages)\n")
		return b.String()
	}
	fmt.Fprintf(&b, "%-8s %-28s %-20s %s\n", "SEQ", "SUBJECT", "TIME", "PAYLOAD")
	for i, m := range msgs {
		if i >= 50 {
			fmt.Fprintf(&b, "… (%d more)\n", len(msgs)-50)
			break
		}
		ts := ""
		if !m.Timestamp.IsZero() {
			ts = m.Timestamp.UTC().Format(time.RFC3339)
		}
		fmt.Fprintf(&b, "%-8d %-28s %-20s %s\n",
			m.Seq,
			truncateRunes(m.Subject, 28),
			truncateRunes(ts, 20),
			truncateRunes(formatPayloadPreview(m.Payload), 64),
		)
	}
	return b.String()
}

func formatPayloadPreview(payload []byte) string {
	if len(payload) == 0 {
		return ""
	}
	if utf8.Valid(payload) {
		return string(payload)
	}
	return base64.StdEncoding.EncodeToString(payload)
}
