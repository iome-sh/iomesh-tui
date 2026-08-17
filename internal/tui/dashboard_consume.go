package tui

import (
	"context"
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
// Time from timestamp, Dept from subject dept.X token, Title from subject or
// short payload text, Kind=ops. Does not invent P2 checkout titles.
func heartbeatFromStreamMessage(m iomesh.StreamMessage) HeartbeatEvent {
	t := ""
	if !m.Timestamp.IsZero() {
		t = m.Timestamp.UTC().Format("15:04:05")
	}
	dept := deptFromSubject(m.Subject)
	title := strings.TrimSpace(m.Subject)
	if title == "" {
		title = shortPayloadText(m.Payload)
	}
	detail := shortPayloadText(m.Payload)
	if detail == "" || detail == title {
		detail = strings.TrimSpace(m.Stream)
	}
	return HeartbeatEvent{
		T:      t,
		Dept:   dept,
		Kind:   kindOps,
		Title:  title,
		Detail: detail,
	}
}

func deptFromSubject(subject string) string {
	parts := strings.Split(strings.TrimSpace(subject), ".")
	if len(parts) >= 2 && strings.EqualFold(parts[0], "dept") {
		return parts[1]
	}
	return ""
}

func shortPayloadText(p []byte) string {
	if len(p) == 0 {
		return ""
	}
	if !utf8.Valid(p) {
		return ""
	}
	var obj map[string]any
	if json.Unmarshal(p, &obj) == nil {
		for _, k := range []string{"text", "message", "title", "summary", "type"} {
			if v, ok := obj[k].(string); ok {
				if t := strings.TrimSpace(v); t != "" {
					return truncate(t, 80)
				}
			}
		}
	}
	s := strings.Join(strings.Fields(string(p)), " ")
	return truncate(s, 80)
}
