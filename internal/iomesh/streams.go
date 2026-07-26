package iomesh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// StreamInfo is broker stream metadata from GET /v1/streams and GET /v1/streams/{name}.
// Wire shape matches broker / iomesh-client-sdk-go StreamInfo (name, subjects, stats, retention knobs).
// Lean TUI surface — no SDK dependency.
type StreamInfo struct {
	Name        string    `json:"name"`
	Subjects    []string  `json:"subjects"`
	Retention   string    `json:"retention,omitempty"`
	Partitions  int       `json:"partitions,omitempty"`
	MaxMsgs     *int64    `json:"max_msgs,omitempty"`
	MaxAgeSec   *int64    `json:"max_age_sec,omitempty"`
	Description string    `json:"description,omitempty"`
	Messages    uint64    `json:"messages"`
	FirstSeq    uint64    `json:"first_seq"`
	LastSeq     uint64    `json:"last_seq"`
	CreatedAt   time.Time `json:"created_at"`
}

// ListStreams returns all streams via GET /v1/streams.
// Unlike fail-open helpers (catalog/context/policy), this is explicit discovery:
// non-2xx and transport errors return error (not an empty list).
// Accepts a JSON array body, or optionally an envelope {"streams":[...]}.
// When mesh is disabled / endpoint empty: returns (nil, error) with "mesh disabled".
func (c *Client) ListStreams(ctx context.Context) ([]StreamInfo, error) {
	if c == nil || !c.Enabled() {
		return nil, fmt.Errorf("mesh disabled")
	}
	u := strings.TrimRight(c.cfg.Endpoint, "/") + "/v1/streams"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
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
	return decodeStreamsList(body)
}

// GetStream returns one stream via GET /v1/streams/{name}.
// Empty name returns an error. Non-2xx (including 404) returns an error.
// When mesh is disabled / endpoint empty: returns (nil, error) with "mesh disabled".
func (c *Client) GetStream(ctx context.Context, name string) (*StreamInfo, error) {
	if c == nil || !c.Enabled() {
		return nil, fmt.Errorf("mesh disabled")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("iomesh streams: stream name required")
	}
	u := strings.TrimRight(c.cfg.Endpoint, "/") + "/v1/streams/" + url.PathEscape(name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
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
	var info StreamInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// DeleteStream deletes a broker stream via DELETE /v1/streams/{name}.
// Empty name returns an error. 2xx (including 204 No Content) is success; non-2xx returns error.
// When mesh is disabled / endpoint empty: returns error with "mesh disabled".
// Destructive — CLI gates with --delete --name --yes (s302).
func (c *Client) DeleteStream(ctx context.Context, name string) error {
	if c == nil || !c.Enabled() {
		return fmt.Errorf("mesh disabled")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("iomesh streams: stream name required")
	}
	u := strings.TrimRight(c.cfg.Endpoint, "/") + "/v1/streams/" + url.PathEscape(name)
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
		return fmt.Errorf("iomesh streams: http %d", resp.StatusCode)
	}
	return nil
}

func decodeStreamsList(raw []byte) ([]StreamInfo, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return []StreamInfo{}, nil
	}
	var streams []StreamInfo
	if err := json.Unmarshal(raw, &streams); err == nil {
		if streams == nil {
			streams = []StreamInfo{}
		}
		return streams, nil
	}
	var env struct {
		Streams []StreamInfo `json:"streams"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Streams == nil {
		env.Streams = []StreamInfo{}
	}
	return env.Streams, nil
}

// FormatStreams renders a compact table for CLI operator discovery.
// Always prints PART (0 when unset) and RETENTION (empty when unset), plus
// MAX_MSGS / MAX_AGE numeric columns (0 when *int64 nil) for CI scrapers (s699).
// Does not invent retention_tier (not on StreamInfo wire).
func FormatStreams(streams []StreamInfo) string {
	var b strings.Builder
	fmt.Fprintf(&b, "iomesh streams count=%d\n", len(streams))
	if len(streams) == 0 {
		b.WriteString("(no streams)\n")
		return b.String()
	}
	fmt.Fprintf(&b, "%-20s %8s %8s %8s %5s %8s %8s %-10s %s\n",
		"NAME", "MSGS", "FIRST", "LAST", "PART", "MAX_MSGS", "MAX_AGE", "RETENTION", "SUBJECTS")
	for i, s := range streams {
		if i >= 50 {
			fmt.Fprintf(&b, "… (%d more)\n", len(streams)-50)
			break
		}
		subj := strings.Join(s.Subjects, ",")
		maxMsgs := int64(0)
		if s.MaxMsgs != nil {
			maxMsgs = *s.MaxMsgs
		}
		maxAge := int64(0)
		if s.MaxAgeSec != nil {
			maxAge = *s.MaxAgeSec
		}
		fmt.Fprintf(&b, "%-20s %8d %8d %8d %5d %8d %8d %-10s %s\n",
			truncateRunes(s.Name, 20),
			s.Messages, s.FirstSeq, s.LastSeq, s.Partitions,
			maxMsgs, maxAge,
			truncateRunes(s.Retention, 10),
			truncateRunes(subj, 40),
		)
	}
	return b.String()
}

// StreamInfoPrint is a CLI-side print DTO for mesh stream get/detail JSON.
// Always emits retention knobs for CI scrapers without omitempty gaps:
// description, retention (empty string when unset), partitions (0 when unset),
// max_msgs / max_age_sec (0 when wire *int64 nil), created_at ("" when zero),
// subjects ([] when empty). Separate from wire StreamInfo so broker decode stays
// lean (omitempty intact on the wire type).
//
// s699: peer FormatStreamDetail text always-emit + aion s698 cost-max residual
// suite. Does not invent retention_tier (not on StreamInfo wire). Beta · offline
// unit ≠ live APPLY.
type StreamInfoPrint struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Retention   string   `json:"retention"`
	Partitions  int      `json:"partitions"`
	MaxMsgs     int64    `json:"max_msgs"`
	MaxAgeSec   int64    `json:"max_age_sec"`
	Messages    uint64   `json:"messages"`
	FirstSeq    uint64   `json:"first_seq"`
	LastSeq     uint64   `json:"last_seq"`
	CreatedAt   string   `json:"created_at"`
	Subjects    []string `json:"subjects"`
}

// NewStreamInfoPrint builds a print DTO from wire StreamInfo. Nil *int64 knobs
// become 0; empty strings/slices stay empty (never omitted on marshal).
func NewStreamInfoPrint(s StreamInfo) StreamInfoPrint {
	p := StreamInfoPrint{
		Name:        s.Name,
		Description: s.Description,
		Retention:   s.Retention,
		Partitions:  s.Partitions,
		Messages:    s.Messages,
		FirstSeq:    s.FirstSeq,
		LastSeq:     s.LastSeq,
		Subjects:    s.Subjects,
	}
	if p.Subjects == nil {
		p.Subjects = []string{}
	}
	if s.MaxMsgs != nil {
		p.MaxMsgs = *s.MaxMsgs
	}
	if s.MaxAgeSec != nil {
		p.MaxAgeSec = *s.MaxAgeSec
	}
	if !s.CreatedAt.IsZero() {
		p.CreatedAt = s.CreatedAt.UTC().Format(time.RFC3339)
	}
	return p
}

// FormatStreamDetail is a multi-line view for one stream (CLI).
// Pure helper with no network I/O.
//
// s699 always-emit retention knobs for CI scrapers (same discipline as
// filter_subject / pull_role): description, retention (empty when unset),
// partitions (0 when unset), max_msgs / max_age_sec (numeric; 0 when *int64
// nil), created_at (blank when zero), subjects ("  (none)" when empty).
// Does not invent retention_tier (not on StreamInfo wire). Beta · offline unit
// ≠ live APPLY · peer aion s698 cost-max residual suite.
func FormatStreamDetail(s StreamInfo) string {
	var b strings.Builder
	b.WriteString("iomesh stream\n")
	fmt.Fprintf(&b, "name:        %s\n", s.Name)
	fmt.Fprintf(&b, "description: %s\n", s.Description)
	fmt.Fprintf(&b, "retention:   %s\n", s.Retention)
	fmt.Fprintf(&b, "partitions:  %d\n", s.Partitions)
	maxMsgs := int64(0)
	if s.MaxMsgs != nil {
		maxMsgs = *s.MaxMsgs
	}
	fmt.Fprintf(&b, "max_msgs:    %d\n", maxMsgs)
	maxAge := int64(0)
	if s.MaxAgeSec != nil {
		maxAge = *s.MaxAgeSec
	}
	fmt.Fprintf(&b, "max_age_sec: %d\n", maxAge)
	fmt.Fprintf(&b, "messages:    %d\n", s.Messages)
	fmt.Fprintf(&b, "first_seq:   %d\n", s.FirstSeq)
	fmt.Fprintf(&b, "last_seq:    %d\n", s.LastSeq)
	created := ""
	if !s.CreatedAt.IsZero() {
		created = s.CreatedAt.UTC().Format(time.RFC3339)
	}
	fmt.Fprintf(&b, "created_at:  %s\n", created)
	b.WriteString("subjects:\n")
	if len(s.Subjects) == 0 {
		b.WriteString("  (none)\n")
	} else {
		for i, sub := range s.Subjects {
			if i >= 24 {
				fmt.Fprintf(&b, "  … +%d more\n", len(s.Subjects)-24)
				break
			}
			fmt.Fprintf(&b, "  - %s\n", sub)
		}
	}
	return b.String()
}
