package iomesh

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/router"
)

// ModelUsage is per-model rollup for the local process meter.
type ModelUsage struct {
	Model            string  `json:"model"`
	Calls            int     `json:"calls"`
	Errors           int     `json:"errors"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	EstUSD           float64 `json:"est_usd"`
	DurationMS       int64   `json:"duration_ms"`
}

// UsageSnapshot is a point-in-time local metering view (not remote dashboard state).
type UsageSnapshot struct {
	Started time.Time    `json:"started"`
	AsOf    time.Time    `json:"as_of"`
	Calls   int          `json:"calls"`
	Errors  int          `json:"errors"`
	Tokens  int          `json:"tokens"`
	EstUSD  float64      `json:"est_usd"`
	ByModel []ModelUsage `json:"by_model"`
}

// UsageMeter aggregates LLM call metrics in-process for `iomesh mesh usage`.
type UsageMeter struct {
	mu      sync.Mutex
	started time.Time
	byModel map[string]*ModelUsage
}

func newUsageMeter() *UsageMeter {
	return &UsageMeter{
		started: time.Now().UTC(),
		byModel: map[string]*ModelUsage{},
	}
}

// Record adds one LLM call to the rollup.
func (m *UsageMeter) Record(meta router.CallMeta, usage router.Usage, err error) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	name := meta.ModelName
	if name == "" {
		name = meta.ModelID
	}
	if name == "" {
		name = "unknown"
	}
	row, ok := m.byModel[name]
	if !ok {
		row = &ModelUsage{Model: name}
		m.byModel[name] = row
	}
	row.Calls++
	if err != nil {
		row.Errors++
	}
	row.PromptTokens += usage.PromptTokens
	row.CompletionTokens += usage.CompletionTokens
	row.TotalTokens += usage.TotalTokens
	row.EstUSD += meta.EstimatedUSD
	row.DurationMS += meta.Duration.Milliseconds()
}

// Snapshot returns a copy of the current rollup.
func (m *UsageMeter) Snapshot() UsageSnapshot {
	if m == nil {
		return UsageSnapshot{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	snap := UsageSnapshot{
		Started: m.started,
		AsOf:    time.Now().UTC(),
	}
	for _, row := range m.byModel {
		cp := *row
		snap.ByModel = append(snap.ByModel, cp)
		snap.Calls += cp.Calls
		snap.Errors += cp.Errors
		snap.Tokens += cp.TotalTokens
		snap.EstUSD += cp.EstUSD
	}
	sort.Slice(snap.ByModel, func(i, j int) bool {
		return snap.ByModel[i].Model < snap.ByModel[j].Model
	})
	return snap
}

// FormatUsage renders a human-readable metering table for the CLI.
func FormatUsage(s UsageSnapshot) string {
	var b strings.Builder
	b.WriteString("iomesh local usage meter (process lifetime)\n")
	if !s.Started.IsZero() {
		b.WriteString(fmt.Sprintf("started=%s as_of=%s\n", s.Started.Format(time.RFC3339), s.AsOf.Format(time.RFC3339)))
	}
	b.WriteString(fmt.Sprintf("totals: calls=%d errors=%d tokens=%d est_usd=%.6f\n", s.Calls, s.Errors, s.Tokens, s.EstUSD))
	if len(s.ByModel) == 0 {
		b.WriteString("(no LLM calls recorded in this process yet)\n")
		return b.String()
	}
	b.WriteString(fmt.Sprintf("%-28s %6s %6s %10s %12s %10s\n", "model", "calls", "err", "tokens", "est_usd", "dur_ms"))
	for _, row := range s.ByModel {
		b.WriteString(fmt.Sprintf("%-28s %6d %6d %10d %12.6f %10d\n",
			truncateRunes(row.Model, 28), row.Calls, row.Errors, row.TotalTokens, row.EstUSD, row.DurationMS))
	}
	return b.String()
}

// ModelUsagePrint is a CLI-side print row for mesh usage --json by_model[].
// Always emits all fields (empty string / 0 honest) without omitempty gaps.
//
// s738: mold CatalogPrint s735 + KVEntryPrint time honesty. Peer aion s737.
// Beta · offline unit ≠ live APPLY · dual_write OFF · not full mesh RBAC GA ·
// local process meter ≠ remote dashboard · DTO ≠ invent usage success.
type ModelUsagePrint struct {
	Model            string  `json:"model"`
	Calls            int     `json:"calls"`
	Errors           int     `json:"errors"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	EstUSD           float64 `json:"est_usd"`
	DurationMS       int64   `json:"duration_ms"`
}

// UsagePrint is a CLI-side print DTO for mesh usage --json.
// Always emits started/as_of as strings (RFC3339 when set; "" when zero — never
// "0001-01-01T00:00:00Z"), counters (0 honest), and by_model as [] not null.
// Wire UsageSnapshot keeps time.Time for in-process rollup; scrapers use this
// print surface.
//
// s738: mold CatalogPrint s735 + PubPrint s732 + KVPutPrint s729; peer aion s737.
// s756: completeness pin — docs + unit tests lock UsagePrint (s738) with
// PubPrint (s732) + KVPutPrint/KVDeletePrint (s729) always-emit keys; does not
// invent new DTO fields or re-claim s729/s732/s738 product bodies. Peer aion
// s755 residual. DTO ≠ invent usage success · local process meter ≠ remote
// dashboard · dual_write OFF · offline unit ≠ live APPLY · not full mesh RBAC GA.
// Beta · offline unit ≠ live APPLY · empty/0/[] honest · dual_write OFF ·
// not full mesh RBAC GA · local process meter ≠ remote dashboard ·
// DTO ≠ invent usage/meter success.
type UsagePrint struct {
	Started string            `json:"started"` // RFC3339 or "" when zero
	AsOf    string            `json:"as_of"`   // RFC3339 or "" when zero
	Calls   int               `json:"calls"`
	Errors  int               `json:"errors"`
	Tokens  int               `json:"tokens"`
	EstUSD  float64           `json:"est_usd"`
	ByModel []ModelUsagePrint `json:"by_model"` // empty [] not null
}

// NewUsagePrint maps UsageSnapshot → UsagePrint. Zero times become ""; nil
// ByModel becomes []ModelUsagePrint{} so JSON emits [] not null. Empty strings
// and 0 counters are honest when unset.
func NewUsagePrint(s UsageSnapshot) UsagePrint {
	p := UsagePrint{
		Calls:   s.Calls,
		Errors:  s.Errors,
		Tokens:  s.Tokens,
		EstUSD:  s.EstUSD,
		ByModel: make([]ModelUsagePrint, 0, len(s.ByModel)),
	}
	if !s.Started.IsZero() {
		p.Started = s.Started.UTC().Format(time.RFC3339)
	}
	if !s.AsOf.IsZero() {
		p.AsOf = s.AsOf.UTC().Format(time.RFC3339)
	}
	for _, row := range s.ByModel {
		p.ByModel = append(p.ByModel, ModelUsagePrint{
			Model:            row.Model,
			Calls:            row.Calls,
			Errors:           row.Errors,
			PromptTokens:     row.PromptTokens,
			CompletionTokens: row.CompletionTokens,
			TotalTokens:      row.TotalTokens,
			EstUSD:           row.EstUSD,
			DurationMS:       row.DurationMS,
		})
	}
	return p
}

// FormatUsageJSON returns indented JSON for stage CI / operator scrapers.
// Marshals UsagePrint (via NewUsagePrint) so zero times emit "" rather than
// "0001-01-01T00:00:00Z", and by_model is always [] not null. Call sites stay
// FormatUsageJSON(UsageSnapshot). Local process meter — not a remote dashboard.
//
// s738: UsagePrint always-emit. Mold CatalogPrint s735. Peer aion s737 residual.
func FormatUsageJSON(s UsageSnapshot) string {
	b, err := json.MarshalIndent(NewUsagePrint(s), "", "  ")
	if err != nil {
		return `{"error":"usage json marshal failed"}` + "\n"
	}
	return string(b) + "\n"
}

func truncateRunes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}
