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
//
// RetentionTier is the broker product-facing class (hot|temp|extended|archive) when present
// on the wire (aion s701 / s654). omitempty keeps the wire lean; do not invent tier from
// max_age alone — empty means the broker omitted it. Beta · offline unit ≠ live APPLY.
type StreamInfo struct {
	Name          string    `json:"name"`
	Subjects      []string  `json:"subjects"`
	Retention     string    `json:"retention,omitempty"`
	RetentionTier string    `json:"retention_tier,omitempty"`
	Partitions    int       `json:"partitions,omitempty"`
	MaxMsgs       *int64    `json:"max_msgs,omitempty"`
	MaxAgeSec     *int64    `json:"max_age_sec,omitempty"`
	Description   string    `json:"description,omitempty"`
	Messages      uint64    `json:"messages"`
	FirstSeq      uint64    `json:"first_seq"`
	LastSeq       uint64    `json:"last_seq"`
	CreatedAt     time.Time `json:"created_at"`
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

// Console-matching create defaults (Temp 7d limits). Never send retention_tier
// on the create wire — broker classifies; do not invent unpaid 403.
const (
	defaultCreateRetention          = "limits"
	defaultCreateMaxAgeSec          = int64(604800)
	defaultCreateMaxMsgs            = int64(1_000_000)
	defaultOperationalEventsName    = "OPERATIONAL_EVENTS"
	defaultOperationalEventsSubject = "dept.engineering.events.github"
)

// StreamCreateConfig is the operator-facing create input. Retention knobs are
// filled by CreateStream (console defaults). No RetentionTier field — omitted
// on the wire.
type StreamCreateConfig struct {
	Name        string
	Subjects    []string
	Description string
}

// DefaultOperationalEventsCreate returns console defaults for OPERATIONAL_EVENTS.
// Subject: empty tenant → dept.engineering.events.github; tenant already
// starting with "dept." → tenant+".events.github"; else "dept."+tenant+".events.github".
func DefaultOperationalEventsCreate(tenant string) StreamCreateConfig {
	tenant = strings.TrimSpace(tenant)
	subject := defaultOperationalEventsSubject
	if tenant != "" {
		if strings.HasPrefix(tenant, "dept.") {
			subject = tenant + ".events.github"
		} else {
			subject = "dept." + tenant + ".events.github"
		}
	}
	return StreamCreateConfig{
		Name:     defaultOperationalEventsName,
		Subjects: []string{subject},
	}
}

// CreateStream registers a broker stream via POST /v1/streams.
// Body: name, subjects, retention=limits, max_age_sec=604800, max_msgs=1000000,
// optional description. retention_tier is never sent.
// 201 decodes StreamInfo. 409 Conflict is success (idempotent): GetStream, or
// &StreamInfo{Name} if get fails. Empty name / no subjects / other non-2xx → error.
// Mesh disabled → "mesh disabled". Mutating — CLI gates with --create --yes.
// Create ≠ PULSE (listed stream with 0 messages is still empty_stream).
func (c *Client) CreateStream(ctx context.Context, cfg StreamCreateConfig) (*StreamInfo, error) {
	if c == nil || !c.Enabled() {
		return nil, fmt.Errorf("mesh disabled")
	}
	name := strings.TrimSpace(cfg.Name)
	if name == "" {
		return nil, fmt.Errorf("iomesh streams: stream name required")
	}
	subjects := make([]string, 0, len(cfg.Subjects))
	for _, s := range cfg.Subjects {
		if s = strings.TrimSpace(s); s != "" {
			subjects = append(subjects, s)
		}
	}
	if len(subjects) == 0 {
		return nil, fmt.Errorf("iomesh streams: subject required")
	}
	reqBody := struct {
		Name        string   `json:"name"`
		Subjects    []string `json:"subjects"`
		Retention   string   `json:"retention"`
		MaxAgeSec   int64    `json:"max_age_sec"`
		MaxMsgs     int64    `json:"max_msgs"`
		Description string   `json:"description,omitempty"`
	}{
		Name:        name,
		Subjects:    subjects,
		Retention:   defaultCreateRetention,
		MaxAgeSec:   defaultCreateMaxAgeSec,
		MaxMsgs:     defaultCreateMaxMsgs,
		Description: strings.TrimSpace(cfg.Description),
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	u := strings.TrimRight(c.cfg.Endpoint, "/") + "/v1/streams"
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
	if resp.StatusCode == http.StatusConflict {
		if info, getErr := c.GetStream(ctx, name); getErr == nil && info != nil {
			return info, nil
		}
		return &StreamInfo{Name: name}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("iomesh streams: http %d", resp.StatusCode)
	}
	var info StreamInfo
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &info); err != nil {
			return nil, err
		}
	}
	if info.Name == "" {
		info.Name = name
	}
	return &info, nil
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
// MAX_MSGS / MAX_AGE numeric columns (0 when *int64 nil) for CI scrapers (s699),
// and TIER (retention_tier; empty when broker omits — never invent from max_age) (s702).
func FormatStreams(streams []StreamInfo) string {
	var b strings.Builder
	fmt.Fprintf(&b, "iomesh streams count=%d\n", len(streams))
	if len(streams) == 0 {
		b.WriteString("(no streams)\n")
		return b.String()
	}
	fmt.Fprintf(&b, "%-20s %8s %8s %8s %5s %8s %8s %-10s %-8s %s\n",
		"NAME", "MSGS", "FIRST", "LAST", "PART", "MAX_MSGS", "MAX_AGE", "RETENTION", "TIER", "SUBJECTS")
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
		fmt.Fprintf(&b, "%-20s %8d %8d %8d %5d %8d %8d %-10s %-8s %s\n",
			truncateRunes(s.Name, 20),
			s.Messages, s.FirstSeq, s.LastSeq, s.Partitions,
			maxMsgs, maxAge,
			truncateRunes(s.Retention, 10),
			truncateRunes(s.RetentionTier, 8),
			truncateRunes(subj, 40),
		)
	}
	return b.String()
}

// StreamInfoPrint is a CLI-side print DTO for mesh stream get/detail JSON.
// Always emits retention knobs for CI scrapers without omitempty gaps:
// description, retention (empty string when unset), retention_tier (empty when
// broker omits — never invent from max_age), partitions (0 when unset),
// max_msgs / max_age_sec (0 when wire *int64 nil), created_at ("" when zero),
// subjects ([] when empty). Separate from wire StreamInfo so broker decode stays
// lean (omitempty intact on the wire type).
//
// s699 retention knobs + s702 retention_tier always-emit. Peer FormatStreamDetail
// text + aion s701 mesh-stream-retention residual. Beta · offline unit ≠ live APPLY
// · does not invent freemium unlimited retain · dual_write default OFF.
type StreamInfoPrint struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Retention     string   `json:"retention"`
	RetentionTier string   `json:"retention_tier"`
	Partitions    int      `json:"partitions"`
	MaxMsgs       int64    `json:"max_msgs"`
	MaxAgeSec     int64    `json:"max_age_sec"`
	Messages      uint64   `json:"messages"`
	FirstSeq      uint64   `json:"first_seq"`
	LastSeq       uint64   `json:"last_seq"`
	CreatedAt     string   `json:"created_at"`
	Subjects      []string `json:"subjects"`
}

// NewStreamInfoPrint builds a print DTO from wire StreamInfo. Nil *int64 knobs
// become 0; empty strings/slices stay empty (never omitted on marshal).
// RetentionTier maps s.RetentionTier as-is (empty when broker omit).
func NewStreamInfoPrint(s StreamInfo) StreamInfoPrint {
	p := StreamInfoPrint{
		Name:          s.Name,
		Description:   s.Description,
		Retention:     s.Retention,
		RetentionTier: s.RetentionTier,
		Partitions:    s.Partitions,
		Messages:      s.Messages,
		FirstSeq:      s.FirstSeq,
		LastSeq:       s.LastSeq,
		Subjects:      s.Subjects,
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

// FormatStreamInfoJSON returns indented JSON for stage CI / scrapers.
// Always emits all StreamInfoPrint fields without omitempty gaps.
// s741: Format*JSON helper completeness (DTO already always-emit s699/s702).
// s750: completeness pin — docs + unit tests lock the full s741 helper set
// (this helper + FormatStreamMessagesJSON / FormatStreamInfoListJSON /
// FormatConsumerInfoJSON / FormatKVBucketInfoJSON / FormatKVEntryJSON /
// FormatKVKeysJSON); does not invent new DTO fields or re-claim s741 product
// body. Peer aion s749 residual. CLI prefer Format*JSON · dual_write OFF ·
// offline unit ≠ live APPLY · not full mesh RBAC GA.
// Peer aion s740 residual. Mold FormatPubJSON / FormatStreamDeleteJSON.
func FormatStreamInfoJSON(p StreamInfoPrint) string {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return `{"error":"stream info json marshal failed"}` + "\n"
	}
	return string(b) + "\n"
}

// FormatStreamInfoListJSON returns indented JSON for mesh streams list --json.
// Nil list becomes empty array (never null). Always emits each StreamInfoPrint
// element without omitempty gaps.
// s741: Format*JSON helper completeness (list print DTO already always-emit s702).
// Peer aion s740 residual. Mold FormatStreamInfoJSON / FormatPubJSON.
func FormatStreamInfoListJSON(list []StreamInfoPrint) string {
	if list == nil {
		list = []StreamInfoPrint{}
	}
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return `{"error":"stream info list json marshal failed"}` + "\n"
	}
	return string(b) + "\n"
}

// StreamDeletePrint is a CLI-side print DTO for mesh streams --delete success.
// Always emits ok / name (empty string honest when unset) so scrapers see a
// stable envelope without omitempty gaps. No pull_role — stream delete is not
// consumer pull-auth; do not invent identity fields. Wire DeleteStream stays
// lean (error return only).
//
// s726: mold ConsumerDeletePrint s708; peer aion s725 residual.
// s759: completeness pin — docs + unit tests lock StreamDeletePrint (s726) with
// StreamMessagesPrint (s720) + StreamMessagePrint (s723) always-emit keys; does
// not invent new DTO fields or re-claim s720/s723/s726 product bodies. Peer aion
// s758 residual. DTO ≠ invent stream gone · dual_write OFF · offline unit ≠ live
// APPLY · not full mesh RBAC GA.
// Beta · offline unit ≠ live APPLY · empty name honest · dual_write OFF · not
// full mesh RBAC GA · does not invent delete success when HTTP failed (call only
// after DeleteStream returns nil).
type StreamDeletePrint struct {
	OK   bool   `json:"ok"`
	Name string `json:"name"`
}

// NewStreamDeletePrint builds a delete success print DTO. OK is always true
// (call only after DeleteStream returns nil).
func NewStreamDeletePrint(name string) StreamDeletePrint {
	return StreamDeletePrint{
		OK:   true,
		Name: name,
	}
}

// FormatStreamDelete is a multi-line operator view for mesh streams delete
// success (s726). Always emits name (empty when unset). Pure helper; no I/O.
func FormatStreamDelete(p StreamDeletePrint) string {
	var b strings.Builder
	b.WriteString("PASS mesh streams delete\n")
	fmt.Fprintf(&b, "name: %s\n", p.Name)
	return b.String()
}

// FormatStreamDeleteJSON returns indented JSON for stage CI / scrapers.
// Always emits all StreamDeletePrint fields without omitempty gaps.
func FormatStreamDeleteJSON(p StreamDeletePrint) string {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return `{"error":"stream delete json marshal failed"}` + "\n"
	}
	return string(b) + "\n"
}

// FormatStreamDetail is a multi-line view for one stream (CLI).
// Pure helper with no network I/O.
//
// s699 always-emit retention knobs + s702 retention_tier for CI scrapers (same
// discipline as filter_subject / pull_role): description, retention, retention_tier
// (empty when unset — never invent from max_age alone), partitions (0 when unset),
// max_msgs / max_age_sec (numeric; 0 when *int64 nil), created_at (blank when zero),
// subjects ("  (none)" when empty). Beta · offline unit ≠ live APPLY · peer aion
// s701 mesh-stream-retention residual · dual_write default OFF.
func FormatStreamDetail(s StreamInfo) string {
	var b strings.Builder
	b.WriteString("iomesh stream\n")
	fmt.Fprintf(&b, "name:           %s\n", s.Name)
	fmt.Fprintf(&b, "description:    %s\n", s.Description)
	fmt.Fprintf(&b, "retention:      %s\n", s.Retention)
	fmt.Fprintf(&b, "retention_tier: %s\n", s.RetentionTier)
	fmt.Fprintf(&b, "partitions:     %d\n", s.Partitions)
	maxMsgs := int64(0)
	if s.MaxMsgs != nil {
		maxMsgs = *s.MaxMsgs
	}
	fmt.Fprintf(&b, "max_msgs:       %d\n", maxMsgs)
	maxAge := int64(0)
	if s.MaxAgeSec != nil {
		maxAge = *s.MaxAgeSec
	}
	fmt.Fprintf(&b, "max_age_sec:    %d\n", maxAge)
	fmt.Fprintf(&b, "messages:       %d\n", s.Messages)
	fmt.Fprintf(&b, "first_seq:      %d\n", s.FirstSeq)
	fmt.Fprintf(&b, "last_seq:       %d\n", s.LastSeq)
	created := ""
	if !s.CreatedAt.IsZero() {
		created = s.CreatedAt.UTC().Format(time.RFC3339)
	}
	fmt.Fprintf(&b, "created_at:     %s\n", created)
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
