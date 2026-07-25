package iomesh

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// MemoryPullOptions configures RunMemoryPull (mesh durable consumer → local ingest).
// Cost-max M1 (s652): pull egress fills customer-local Palace; dual_write remains optional audit.
type MemoryPullOptions struct {
	Stream   string
	Name     string // durable consumer name
	Filter   string // optional filter_subject
	Batch    int
	MaxWait  time.Duration
	MaxLoops int  // 0 = forever; 1 = single fetch cycle (dogfood / --once)
	Ack      bool // ack after successful ingest (or dry-run)
	DryRun   bool // map + log only; no LocalIngest
	// LocalIngest writes one mapped envelope to local palace (MCP memory_ingest_turn).
	// Required when DryRun is false.
	LocalIngest func(ctx context.Context, env MemoryEnvelope) error
	// OnMessage is optional observability after each message (dry-run or ingest).
	OnMessage func(msg StreamMessage, env MemoryEnvelope, skipped bool, err error)
}

// MemoryPullStats summarizes one RunMemoryPull invocation.
type MemoryPullStats struct {
	Loops      int
	Fetched    int
	Ingested   int
	Skipped    int
	Acked      int
	Errors     int
	LastError  string
	CreateOK   bool
	Consumer   string
	Stream     string
}

// MapStreamMessageToEnvelope converts a durable-fetch message into a MemoryEnvelope for local ingest.
// Supports:
//   - MEMORY_INGEST style JSON (type memory_ingest / fields content, role, session_id, session_seq, event_time)
//   - generic JSON with content/text/body/message
//   - raw text payload (connector events)
// Returns ok=false when there is no ingestible content.
func MapStreamMessageToEnvelope(msg StreamMessage) (MemoryEnvelope, string /*dedupeKey*/, bool) {
	payload := bytesTrimSpace(msg.Payload)
	if len(payload) == 0 {
		return MemoryEnvelope{}, "", false
	}
	dedupe := fmt.Sprintf("%s:%d", msg.Stream, msg.Seq)

	var env MemoryEnvelope
	if json.Unmarshal(payload, &env) == nil && strings.TrimSpace(env.Content) != "" {
		if strings.TrimSpace(env.Type) == "" {
			env.Type = memoryEnvelopeIngest
		}
		if env.SessionSeq > 0 && strings.TrimSpace(env.SessionID) != "" {
			dedupe = env.SessionID + ":" + fmt.Sprintf("%d", env.SessionSeq)
		}
		return env, dedupe, true
	}

	// Generic event JSON — prefer common text fields.
	var generic map[string]any
	if json.Unmarshal(payload, &generic) == nil {
		content := firstString(generic, "content", "text", "body", "message", "summary")
		if content == "" {
			// Compact JSON as content when structured but no text field.
			if b, err := json.Marshal(generic); err == nil && len(b) > 0 && len(b) < 32<<10 {
				content = string(b)
			}
		}
		if content != "" {
			role := firstString(generic, "role")
			if role == "" {
				role = "system"
			}
			sid := firstString(generic, "session_id", "session")
			if sid == "" {
				sid = msg.Subject
			}
			seq := intFromAny(generic["session_seq"])
			if seq == 0 {
				seq = int(msg.Seq)
			}
			if sid != "" && seq > 0 {
				dedupe = sid + ":" + fmt.Sprintf("%d", seq)
			}
			et := firstString(generic, "event_time", "timestamp", "time")
			if et == "" && !msg.Timestamp.IsZero() {
				et = msg.Timestamp.UTC().Format(time.RFC3339)
			}
			return MemoryEnvelope{
				Type:       memoryEnvelopeIngest,
				SessionID:  sid,
				Role:       role,
				Content:    content,
				EventTime:  et,
				SessionSeq: seq,
			}, dedupe, true
		}
	}

	// Raw text / non-JSON
	text := string(payload)
	if strings.TrimSpace(text) == "" {
		return MemoryEnvelope{}, "", false
	}
	et := ""
	if !msg.Timestamp.IsZero() {
		et = msg.Timestamp.UTC().Format(time.RFC3339)
	}
	return MemoryEnvelope{
		Type:       memoryEnvelopeIngest,
		SessionID:  msg.Subject,
		Role:       "system",
		Content:    text,
		EventTime:  et,
		SessionSeq: int(msg.Seq),
	}, dedupe, true
}

// RunMemoryPull creates (idempotent) a durable consumer and loops fetch → map → local ingest → ack.
// Mesh client must be enabled. Fail-open per message: ingest errors increment Errors and may skip ack.
func (c *Client) RunMemoryPull(ctx context.Context, opt MemoryPullOptions) (MemoryPullStats, error) {
	var st MemoryPullStats
	if c == nil || !c.Enabled() {
		return st, fmt.Errorf("mesh disabled")
	}
	opt.Stream = strings.TrimSpace(opt.Stream)
	opt.Name = strings.TrimSpace(opt.Name)
	if opt.Stream == "" || opt.Name == "" {
		return st, fmt.Errorf("memory pull: stream and consumer name required")
	}
	if opt.Batch <= 0 {
		opt.Batch = 8
	}
	if opt.MaxWait <= 0 {
		opt.MaxWait = 2 * time.Second
	}
	if !opt.DryRun && opt.LocalIngest == nil {
		return st, fmt.Errorf("memory pull: LocalIngest required unless DryRun")
	}
	st.Stream = opt.Stream
	st.Consumer = opt.Name

	if _, err := c.CreateConsumer(ctx, opt.Stream, opt.Name, opt.Filter); err != nil {
		return st, fmt.Errorf("memory pull create consumer: %w", err)
	}
	st.CreateOK = true

	seen := map[string]struct{}{}
	for {
		if ctx.Err() != nil {
			return st, ctx.Err()
		}
		if opt.MaxLoops > 0 && st.Loops >= opt.MaxLoops {
			return st, nil
		}
		st.Loops++

		msgs, err := c.ConsumerFetch(ctx, opt.Stream, opt.Name, opt.Batch, opt.MaxWait)
		if err != nil {
			st.Errors++
			st.LastError = err.Error()
			if opt.MaxLoops == 1 {
				return st, err
			}
			// Back off slightly on fetch errors in long-run mode.
			select {
			case <-ctx.Done():
				return st, ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}
			continue
		}
		if len(msgs) == 0 {
			if opt.MaxLoops > 0 && st.Loops >= opt.MaxLoops {
				return st, nil
			}
			continue
		}

		var ackSeqs []uint64
		for _, msg := range msgs {
			st.Fetched++
			env, key, ok := MapStreamMessageToEnvelope(msg)
			if !ok {
				st.Skipped++
				if opt.OnMessage != nil {
					opt.OnMessage(msg, env, true, nil)
				}
				// Ack empty/unusable to avoid poison redelivery.
				if opt.Ack {
					ackSeqs = append(ackSeqs, msg.Seq)
				}
				continue
			}
			if key != "" {
				if _, dup := seen[key]; dup {
					st.Skipped++
					if opt.OnMessage != nil {
						opt.OnMessage(msg, env, true, nil)
					}
					if opt.Ack {
						ackSeqs = append(ackSeqs, msg.Seq)
					}
					continue
				}
				seen[key] = struct{}{}
			}

			var ingErr error
			if opt.DryRun {
				// no-op
			} else {
				ingErr = opt.LocalIngest(ctx, env)
			}
			if opt.OnMessage != nil {
				opt.OnMessage(msg, env, false, ingErr)
			}
			if ingErr != nil {
				st.Errors++
				st.LastError = ingErr.Error()
				// Do not ack on ingest failure so message can redeliver.
				continue
			}
			st.Ingested++
			if opt.Ack {
				ackSeqs = append(ackSeqs, msg.Seq)
			}
		}
		if opt.Ack && len(ackSeqs) > 0 {
			if _, err := c.ConsumerAck(ctx, opt.Stream, opt.Name, ackSeqs...); err != nil {
				st.Errors++
				st.LastError = err.Error()
			} else {
				st.Acked += len(ackSeqs)
			}
		}
	}
}

func bytesTrimSpace(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case string:
				if s := strings.TrimSpace(t); s != "" {
					return s
				}
			case fmt.Stringer:
				if s := strings.TrimSpace(t.String()); s != "" {
					return s
				}
			}
		}
	}
	return ""
}

func intFromAny(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case json.Number:
		i, _ := t.Int64()
		return int(i)
	default:
		return 0
	}
}
