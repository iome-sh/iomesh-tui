package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/iome-sh/iomesh-tui/internal/iomesh"
)

// Digest honesty (#369 / FR-25 · FR-26 · FR-31) + delta briefs (#370 / FR-24 · FR-30):
//   - Empty or rejected patterns → insufficient-signal / nothing reliable today (do not invent).
//   - Rate-claiming pattern lines require n of N and a window, else rejected.
//   - No-delta / "what is true" recaps are rejected (briefs must be a change vs prior window).
//   - Receipts default to pointers + hashes, not raw customer text.
//   - source=external is a third labeled pane; never satisfies cite-both mesh,private.
//   - dual_write stays off · catalog list ≠ consume · not Memory GA.

const (
	digestInsufficientSignal = "insufficient-signal · nothing reliable today"
	digestHonestyExtraPin    = "catalog list ≠ consume · receipts=pointers+hashes · insufficient-signal allowed · delta≠recap · external≠heartbeat"
)

// digestRateClaimRE matches summaries/kinds that claim a rate (need n/N + window).
var digestRateClaimRE = regexp.MustCompile(`(?i)(%|\bpercent(?:age)?\b|\bpct\b|\brate\b|\bratio\b|\bper\s+(?:day|week|hour|month|minute)\b)`)

// patternClaimsRate reports whether the pattern line claims a rate / percentage.
func patternClaimsRate(p iomesh.MemoryOpsDigestPattern) bool {
	blob := strings.TrimSpace(p.Summary) + " " + strings.TrimSpace(p.Kind) + " " + strings.TrimSpace(p.Subject)
	if digestRateClaimRE.MatchString(blob) {
		return true
	}
	// Explicit denominator on the wire implies a rate framing even without "%".
	if p.Total > 0 {
		return true
	}
	return false
}

// patternRateHonest is true when a rate claim carries n of N and a window.
func patternRateHonest(p iomesh.MemoryOpsDigestPattern) bool {
	return p.Count > 0 && p.Total > 0 && strings.TrimSpace(p.Window) != ""
}

// acceptDigestPattern keeps non-rate patterns, and rate patterns only when n/N+window are present.
// Rejected rate claims and no-delta recaps are omitted (never invented into the operator brief).
func acceptDigestPattern(p iomesh.MemoryOpsDigestPattern) bool {
	// #370: no-delta / "what is true" recap is not a brief.
	if patternIsNoDelta(p) {
		return false
	}
	if patternClaimsRate(p) {
		return patternRateHonest(p)
	}
	// Non-rate: require some identity — do not invent a blank "pattern".
	return strings.TrimSpace(p.Summary) != "" ||
		strings.TrimSpace(p.Subject) != "" ||
		strings.TrimSpace(p.Kind) != "" ||
		strings.TrimSpace(p.ID) != ""
}

// formatDigestPatternLine renders one accepted pattern. Rate lines always show n of N + window.
func formatDigestPatternLine(i int, p iomesh.MemoryOpsDigestPattern) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  %d. ", i)
	if p.Score > 0 {
		fmt.Fprintf(&b, "[%.2f] ", p.Score)
	}
	if k := strings.TrimSpace(p.Kind); k != "" {
		b.WriteString(k)
		b.WriteByte(' ')
	}
	label := strings.TrimSpace(p.Subject)
	if label == "" {
		label = strings.TrimSpace(p.Summary)
	}
	if label == "" {
		label = strings.TrimSpace(p.ID)
	}
	// Rate claims: never paste a free-form quote; show n of N + window (+ optional compare/links).
	if patternClaimsRate(p) {
		if label != "" && !digestRateClaimRE.MatchString(label) {
			b.WriteString(label)
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%d of %d", p.Count, p.Total)
		if p.Total > 0 {
			fmt.Fprintf(&b, " (%.0f%%)", 100*float64(p.Count)/float64(p.Total))
		}
		fmt.Fprintf(&b, " window=%s", strings.TrimSpace(p.Window))
		if cw := strings.TrimSpace(p.ComparisonWindow); cw != "" {
			fmt.Fprintf(&b, " vs=%s", cw)
		}
		if n := len(p.Links); n > 0 {
			fmt.Fprintf(&b, " links=%d", n)
			// Print at most two pointer links (URLs / ticket ids) — not pasted quotes.
			for j, link := range p.Links {
				if j >= 2 {
					break
				}
				link = strings.TrimSpace(link)
				if link == "" {
					continue
				}
				fmt.Fprintf(&b, " <%s>", link)
			}
		}
		return b.String()
	}
	if label != "" {
		b.WriteString(label)
	}
	if p.Count > 0 {
		fmt.Fprintf(&b, " n=%d", p.Count)
	}
	if w := strings.TrimSpace(p.Window); w != "" {
		fmt.Fprintf(&b, " window=%s", w)
	}
	// #370: show prior-window baseline on delta briefs (change vs prior).
	if cw := strings.TrimSpace(p.ComparisonWindow); cw != "" {
		fmt.Fprintf(&b, " vs=%s", cw)
	}
	return b.String()
}

// shortDigestHash returns a stable 12-hex fingerprint (sha256) for receipt redaction.
func shortDigestHash(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:12]
}

func receiptPointer(r iomesh.MemoryOpsDigestReceipt) string {
	if p := strings.TrimSpace(r.Pointer); p != "" {
		return p
	}
	if h := strings.TrimSpace(r.SourceHint); h != "" && looksLikeReceiptPointer(h) {
		return h
	}
	if id := strings.TrimSpace(r.ID); id != "" {
		return "id:" + id
	}
	if h := strings.TrimSpace(r.SourceHint); h != "" {
		return "src:" + h
	}
	return "id:-"
}

func looksLikeReceiptPointer(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		if u, err := url.Parse(s); err == nil && u.Host != "" {
			return true
		}
	}
	// Ticket-ish pointers (JIRA/Linear/GH) — not free prose.
	if strings.Contains(s, "/") && !strings.Contains(s, " ") {
		return true
	}
	return false
}

func receiptHash(r iomesh.MemoryOpsDigestReceipt) string {
	if h := strings.TrimSpace(r.AccountHash); h != "" {
		return h
	}
	// Fingerprint wire summary so operators can correlate without pasting customer text.
	if h := shortDigestHash(r.Summary); h != "" {
		return h
	}
	return shortDigestHash(r.ID + "|" + r.SourceHint)
}

// formatDigestReceiptLine prints pointer + hash only — never raw customer summary text.
// source_hint is a classification token (mesh|private|catalog|grant), not customer language.
func formatDigestReceiptLine(i int, r iomesh.MemoryOpsDigestReceipt) string {
	ptr := receiptPointer(r)
	hash := receiptHash(r)
	when := strings.TrimSpace(r.EventTime)
	hint := strings.TrimSpace(r.SourceHint)
	var b strings.Builder
	if when != "" {
		fmt.Fprintf(&b, "  %d. [%s] pointer=%s hash=%s", i, when, ptr, hash)
	} else {
		fmt.Fprintf(&b, "  %d. pointer=%s hash=%s", i, ptr, hash)
	}
	if hint != "" {
		fmt.Fprintf(&b, " source=%s", hint)
	}
	return b.String()
}
