package tui

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/iome-sh/iomesh-tui/internal/iomesh"
)

// Consume reasons — residual-honest, mapped from broker /v1 only (not /v52).
const (
	consumeReasonMissing           = "consume missing"
	consumeReasonBrokerUnavailable = "broker_unavailable"
	consumeReasonNoStreams         = "no_streams"
	consumeReasonEmptyStream       = "empty_stream"
	consumeReasonReplayDisabled    = "replay_disabled"
	consumeReasonConsumed          = "consumed"
)

const (
	dashboardConsumeStreamCap = 4
	dashboardConsumeMsgLimit  = 20
	dashboardConsumeTimeout   = 8 * time.Second
)

func meshClient(rt runtimeAdapter) *iomesh.Client {
	if rt.rt == nil {
		return nil
	}
	return rt.rt.Mesh()
}

func (d *dashboardState) applyConsume(events []HeartbeatEvent, names []string, reason string) {
	if d == nil || d.Preview {
		return
	}
	d.Events = events
	d.StreamNames = names
	if strings.TrimSpace(reason) == "" {
		reason = consumeReasonMissing
	}
	d.ConsumeReason = reason
}

// probeDashboardIfAttached runs the broker consume probe once (fail-open).
// Never invents PULSE from eval seed or from a stream list alone.
func probeDashboardIfAttached(d *dashboardState, rt runtimeAdapter) {
	if d == nil || d.Preview {
		return
	}
	c := meshClient(rt)
	if c == nil || !c.Enabled() {
		if strings.TrimSpace(d.ConsumeReason) == "" {
			d.ConsumeReason = consumeReasonMissing
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), dashboardConsumeTimeout)
	defer cancel()
	events, names, reason := probeDashboardConsume(ctx, c)
	d.applyConsume(events, names, reason)
}

func probeDashboardConsume(ctx context.Context, c *iomesh.Client) (events []HeartbeatEvent, names []string, reason string) {
	if c == nil || !c.Enabled() {
		return nil, nil, consumeReasonMissing
	}
	if ctx == nil {
		ctx = context.Background()
	}
	streams, err := c.ListStreams(ctx)
	if err != nil {
		return nil, nil, consumeReasonBrokerUnavailable
	}
	for _, s := range streams {
		if n := strings.TrimSpace(s.Name); n != "" {
			names = append(names, n)
		}
	}
	if len(names) == 0 {
		return nil, names, consumeReasonNoStreams
	}

	capN := dashboardConsumeStreamCap
	if len(names) < capN {
		capN = len(names)
	}
	var sawReplay, sawOther, sawOK bool
	for _, name := range names[:capN] {
		msgs, err := c.ListStreamMessages(ctx, name, iomesh.ListStreamMessagesOptions{Limit: dashboardConsumeMsgLimit})
		if err != nil {
			if isConsumeReplayDisabled(err) {
				sawReplay = true
				continue
			}
			sawOther = true
			continue
		}
		sawOK = true
		for _, m := range msgs {
			events = append(events, heartbeatFromStreamMessage(m))
			if len(events) >= dashboardMaxEvents {
				break
			}
		}
		if len(events) >= dashboardMaxEvents {
			break
		}
	}
	if len(events) > 0 {
		return events, names, consumeReasonConsumed
	}
	if !sawOK && sawReplay && !sawOther {
		return nil, names, consumeReasonReplayDisabled
	}
	if !sawOK && sawOther && !sawReplay {
		return nil, names, consumeReasonBrokerUnavailable
	}
	if !sawOK && sawReplay {
		return nil, names, consumeReasonReplayDisabled
	}
	return nil, names, consumeReasonEmptyStream
}

func isConsumeReplayDisabled(err error) bool {
	if err == nil {
		return false
	}
	if iomesh.IsReplayDisabled(err) {
		return true
	}
	// Broker replay gate is 403; treat messages 403 as replay_disabled.
	return strings.Contains(err.Error(), "http 403")
}

// heartbeatFromStreamMessage maps a broker row conservatively.
// Time from timestamp, Dept from subject dept.X token, Title from peeled
// observation.payload (event_type · repository, or title/summary) then subject.
// Kind from subject tokens (same as CP). Does not invent P2 checkout titles.
func heartbeatFromStreamMessage(m iomesh.StreamMessage) HeartbeatEvent {
	t := ""
	if !m.Timestamp.IsZero() {
		t = m.Timestamp.UTC().Format("15:04:05")
	}
	dept := deptFromSubject(m.Subject)
	raw := decodeConsumePayload(m.Payload)
	title, detail := consumeTitleDetail(raw)
	if title == "" {
		title = strings.TrimSpace(m.Subject)
	}
	if title == "" {
		title = shortPayloadText(m.Payload)
	}
	if detail == "" || detail == title {
		detail = shortPayloadText(m.Payload)
	}
	if detail == "" || detail == title {
		detail = strings.TrimSpace(m.Stream)
	}
	return HeartbeatEvent{
		T:      t,
		Dept:   dept,
		Kind:   kindFromSubject(m.Subject),
		Title:  title,
		Detail: detail,
	}
}

// kindFromSubject uses the same subject tokens as CP.
// GitHub stays ops. Knowledge/analytics classification ≠ GA.
func kindFromSubject(subject string) HeartbeatKind {
	s := strings.ToLower(strings.TrimSpace(subject))
	if s == "" || strings.Contains(s, "github") {
		return kindOps
	}
	switch {
	case containsSubjectToken(s, "events.docs", "events.documents", "notion", "confluence", "sharepoint", "google_drive", ".docs."):
		return kindKnowledge
	case containsSubjectToken(s, "metric.", ".cdc", "embedding", "warehouse", ".dbt"):
		return kindAnalytics
	default:
		return kindOps
	}
}

func containsSubjectToken(s string, tokens ...string) bool {
	for _, tok := range tokens {
		if strings.Contains(s, tok) {
			return true
		}
	}
	return false
}

func deptFromSubject(subject string) string {
	parts := strings.Split(strings.TrimSpace(subject), ".")
	if len(parts) >= 2 && strings.EqualFold(parts[0], "dept") {
		return parts[1]
	}
	return ""
}

// decodeConsumePayload peels stacked standard or URL-safe base64 (max 4)
// until the bytes look like a JSON object/array. ListStreamMessages already
// decodes the wire once; persist can leave another layer (eyJ…).
// Never treat leftover eyJ as text.
func decodeConsumePayload(payload []byte) []byte {
	raw := []byte(strings.TrimSpace(string(payload)))
	if len(raw) == 0 {
		return nil
	}
	for i := 0; i < 4; i++ {
		if looksLikeJSONObject(raw) {
			return raw
		}
		next, err := decodeConsumeBase64(raw)
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

func decodeConsumeBase64(raw []byte) ([]byte, error) {
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

type consumeEnvelopeHint struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
	Title   string          `json:"title"`
	Summary string          `json:"summary"`
	Message string          `json:"message"`
	Text    string          `json:"text"`
}

type consumeInnerHint struct {
	Title      string          `json:"title"`
	Summary    string          `json:"summary"`
	Message    string          `json:"message"`
	Text       string          `json:"text"`
	EventType  string          `json:"event_type"`
	Repository json.RawMessage `json:"repository"`
}

// consumeTitleDetail unwraps observation.payload for event_type + repository
// (or title/summary). Pretty title ≠ Connected / live APPLY / P2 checkout.
func consumeTitleDetail(raw []byte) (title, detail string) {
	if len(raw) == 0 {
		return "", ""
	}
	var env consumeEnvelopeHint
	innerRaw := raw
	if json.Unmarshal(raw, &env) == nil && len(env.Payload) > 0 && looksLikeJSONObject(env.Payload) {
		innerRaw = env.Payload
	}
	var inner consumeInnerHint
	if len(innerRaw) > 0 {
		_ = json.Unmarshal(innerRaw, &inner)
	}
	title = firstNonEmpty(
		strings.TrimSpace(inner.Title),
		strings.TrimSpace(inner.Summary),
		strings.TrimSpace(inner.Message),
		strings.TrimSpace(inner.Text),
		joinConsumeTitle(strings.TrimSpace(inner.EventType), repositoryName(inner.Repository)),
		strings.TrimSpace(env.Title),
		strings.TrimSpace(env.Summary),
		strings.TrimSpace(env.Message),
		strings.TrimSpace(env.Text),
	)
	detail = firstNonEmpty(
		strings.TrimSpace(inner.Summary),
		strings.TrimSpace(inner.Title),
		joinConsumeTitle(strings.TrimSpace(inner.EventType), repositoryName(inner.Repository)),
	)
	return title, detail
}

func repositoryName(raw json.RawMessage) string {
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

func joinConsumeTitle(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return strings.Join(out, " · ")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func shortPayloadText(p []byte) string {
	if len(p) == 0 {
		return ""
	}
	raw := decodeConsumePayload(p)
	if len(raw) == 0 {
		return ""
	}
	if !utf8.Valid(raw) {
		return ""
	}
	if title, _ := consumeTitleDetail(raw); title != "" {
		return truncate(title, 80)
	}
	var obj map[string]any
	if json.Unmarshal(raw, &obj) == nil {
		for _, k := range []string{"text", "message", "title", "summary"} {
			if v, ok := obj[k].(string); ok {
				if t := strings.TrimSpace(v); t != "" {
					return truncate(t, 80)
				}
			}
		}
	}
	s := strings.Join(strings.Fields(string(raw)), " ")
	if strings.HasPrefix(s, "eyJ") {
		return ""
	}
	return truncate(s, 80)
}
