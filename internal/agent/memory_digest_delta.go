package agent

import (
	"regexp"
	"strings"

	"github.com/iome-sh/iomesh-tui/internal/iomesh"
)

// Delta briefs + external color (#370 / FR-24 · FR-30):
//   - Briefs are a change vs the prior window (new language, stall, support theme, paging shape).
//   - A no-delta / "what is true" recap is rejected → insufficient-signal when nothing remains.
//   - source=external never satisfies require-sources mesh,private.
//   - External color is a third labeled pane, never the heartbeat.
//   - First-party consume remains the only path that fills mesh citations.
//   - dual_write OFF · not Memory GA · do not invent an external demand-feed connector.

const (
	digestExternalColorPane = "external color"
	digestExternalCitePin   = "external/sponsored do not satisfy cite-both"
	digestNoDeltaRejectNote = "rejected %d no-delta recap(s) (not a change vs prior window)"
)

// digestRecapRE matches "what is true" / status-quo language that is not a delta brief.
var digestRecapRE = regexp.MustCompile(`(?i)(\bwhat\s+is\s+true\b|\bstatus\s*quo\b|\brecap\b|\bsnapshot\b|\binventory\b|\bunchanged\b|\bno[-\s]?change\b|\bno[-\s]?delta\b|\bcurrent\s+state\b|\bstill\s+true\b|\bremains\s+(?:true|healthy|green|unchanged)\b)`)

// digestRecapKindRE matches explicit non-delta kind tokens.
var digestRecapKindRE = regexp.MustCompile(`(?i)^(recap|status|status_quo|snapshot|inventory|baseline|what_is_true|current_state)$`)

// digestDeltaKindRE matches kinds that are already a change vs the prior window.
var digestDeltaKindRE = regexp.MustCompile(`(?i)^(delta|change|stall|theme|paging|emergent|shift|new(?:_.+)?)$`)

// digestDeltaLangRE matches summary language that claims a change vs prior.
var digestDeltaLangRE = regexp.MustCompile(`(?i)(\bdelta\b|\bvs\.?\s+prior\b|\bchanged\b|\bemerge[dn]?\b|\bshift(?:ed|ing)?\b|\bnew\s+(?:language|stall|theme|paging)\b|\bpaging\s+shape\b|\bsupport\s+theme\b|\bstall(?:ed|ing)?\b)`)

// patternHasDeltaSignal reports an explicit change vs the prior window.
func patternHasDeltaSignal(p iomesh.MemoryOpsDigestPattern) bool {
	if p.Delta != nil && *p.Delta {
		return true
	}
	if strings.TrimSpace(p.ComparisonWindow) != "" {
		return true
	}
	kind := strings.TrimSpace(p.Kind)
	if kind != "" && digestDeltaKindRE.MatchString(kind) {
		return true
	}
	if digestDeltaLangRE.MatchString(p.Summary) {
		return true
	}
	return false
}

// patternIsNoDelta is true for "what is true" recaps / explicit delta=false without a delta signal.
// Activity patterns (recurrence/volume with identity) are not auto-rejected — only recap-shaped briefs.
func patternIsNoDelta(p iomesh.MemoryOpsDigestPattern) bool {
	if p.Delta != nil && !*p.Delta && !patternHasDeltaSignal(p) {
		return true
	}
	if patternHasDeltaSignal(p) {
		return false
	}
	kind := strings.TrimSpace(p.Kind)
	if kind != "" && digestRecapKindRE.MatchString(kind) {
		return true
	}
	blob := strings.TrimSpace(p.Summary) + " " + kind + " " + strings.TrimSpace(p.Subject)
	return digestRecapRE.MatchString(blob)
}

// isExternalDigestReceipt reports source_hint classified as external/sponsored color.
func isExternalDigestReceipt(r iomesh.MemoryOpsDigestReceipt) bool {
	return ClassifyDigestSourceHint(r.SourceHint) == DigestSourceExternal
}
