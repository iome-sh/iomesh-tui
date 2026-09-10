package agent

import (
	"fmt"
	"strings"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/iomesh"
)

// Sticky cite-both (#419): classify receipts from provenance/tags as well as
// source_hint, pin older mesh-stamped turns into the active set when they are
// in the fetched export, and print an explicit newest-first window reason when
// a required class is still missing. Never invent mesh. dual_write OFF.

const (
	opsDigestLimitDefault  = 20
	opsDigestLimitCiteBoth = 50 // MCP ops_digest_export cap
)

// digestCiteLimits returns fetch vs display caps for one digest call.
// --require-sources with no explicit --limit fetches up to the export cap so
// older mesh-stamped turns are not dropped behind newer private RCA, then
// pins required classes into the default display window.
func digestCiteLimits(call MemoryOpsDigestOpts) (fetchLimit, displayLimit int) {
	fetchLimit = call.Limit
	displayLimit = call.Limit
	if fetchLimit <= 0 {
		if len(call.RequireSources) > 0 {
			fetchLimit = opsDigestLimitCiteBoth
		} else {
			fetchLimit = opsDigestLimitDefault
		}
	}
	if displayLimit <= 0 {
		displayLimit = opsDigestLimitDefault
	}
	return fetchLimit, displayLimit
}

// ClassifyDigestReceipt maps a receipt to mesh|private|catalog|grant|external|"".
// Provenance.source_hint and tags (source_hint:mesh) win over an export-origin
// source_hint such as palace_timeline. Catalog/grant/external still never
// satisfy mesh or private. Empty / unknown does not invent a class.
func ClassifyDigestReceipt(r iomesh.MemoryOpsDigestReceipt) string {
	var fallback string
	for _, cand := range digestReceiptHintCandidates(r) {
		switch ClassifyDigestSourceHint(cand) {
		case DigestSourceMesh:
			return DigestSourceMesh
		case DigestSourceCatalog:
			if fallback == "" || fallback == DigestSourcePrivate {
				fallback = DigestSourceCatalog
			}
		case DigestSourceGrant:
			if fallback == "" || fallback == DigestSourcePrivate {
				fallback = DigestSourceGrant
			}
		case DigestSourceExternal:
			if fallback == "" || fallback == DigestSourcePrivate {
				fallback = DigestSourceExternal
			}
		case DigestSourcePrivate:
			if fallback == "" {
				fallback = DigestSourcePrivate
			}
		}
	}
	return fallback
}

func digestReceiptHintCandidates(r iomesh.MemoryOpsDigestReceipt) []string {
	out := make([]string, 0, 4+len(r.Tags))
	if s := strings.TrimSpace(r.SourceHint); s != "" {
		out = append(out, s)
	}
	if s := strings.TrimSpace(r.Provenance.SourceHint); s != "" {
		out = append(out, s)
	}
	if s := strings.TrimSpace(r.Provenance.SourceStep); s != "" {
		out = append(out, s)
	}
	for _, tag := range r.Tags {
		if s := normalizeDigestTag(tag); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func normalizeDigestTag(raw string) string {
	h := strings.ToLower(strings.TrimSpace(raw))
	h = strings.ReplaceAll(h, "-", "_")
	if i := strings.IndexByte(h, ':'); i >= 0 {
		head, tail := h[:i], h[i+1:]
		if head == "source" || head == "origin" || head == "source_hint" {
			return strings.TrimSpace(tail)
		}
	}
	return h
}

func receiptEventTime(r iomesh.MemoryOpsDigestReceipt) time.Time {
	s := strings.TrimSpace(r.EventTime)
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t
	}
	return time.Time{}
}

func receiptPinKey(r iomesh.MemoryOpsDigestReceipt) string {
	if id := strings.TrimSpace(r.ID); id != "" {
		return "id:" + id
	}
	return "ptr:" + strings.TrimSpace(r.Pointer) + "|" + r.EventTime + "|" + r.SourceHint + "|" + r.Summary
}

func annotateDigestFetchWindow(res *iomesh.MemoryOpsDigestResult, fetchLimit int) {
	if res == nil {
		return
	}
	res.FetchLimit = fetchLimit
	res.FetchedN = len(res.Receipts)
	var newest, oldest time.Time
	var newestRaw, oldestRaw string
	for _, r := range res.Receipts {
		t := receiptEventTime(r)
		if t.IsZero() {
			continue
		}
		if newest.IsZero() || t.After(newest) {
			newest = t
			newestRaw = strings.TrimSpace(r.EventTime)
		}
		if oldest.IsZero() || t.Before(oldest) {
			oldest = t
			oldestRaw = strings.TrimSpace(r.EventTime)
		}
	}
	res.FetchedNewest = newestRaw
	res.FetchedOldest = oldestRaw
}

func finalizeDigestForRequireSources(res *iomesh.MemoryOpsDigestResult, required []string, fetchLimit, displayLimit int) {
	if res == nil {
		return
	}
	annotateDigestFetchWindow(res, fetchLimit)
	if len(required) == 0 {
		return
	}
	res.Receipts = pinRequiredDigestReceipts(res.Receipts, required, displayLimit)
}

// pinRequiredDigestReceipts keeps newest-first receipts up to displayLimit and
// guarantees one receipt per required class when that class exists in the
// fetched set (so older mesh is not dropped behind newer private RCA).
func pinRequiredDigestReceipts(receipts []iomesh.MemoryOpsDigestReceipt, required []string, displayLimit int) []iomesh.MemoryOpsDigestReceipt {
	if len(receipts) == 0 {
		return receipts
	}
	if displayLimit <= 0 {
		displayLimit = opsDigestLimitDefault
	}
	if len(required) == 0 {
		if len(receipts) > displayLimit {
			return append([]iomesh.MemoryOpsDigestReceipt(nil), receipts[:displayLimit]...)
		}
		return receipts
	}

	pinKeys := map[string]bool{}
	var pins []iomesh.MemoryOpsDigestReceipt
	for _, req := range required {
		for _, r := range receipts {
			if ClassifyDigestReceipt(r) != req {
				continue
			}
			key := receiptPinKey(r)
			if pinKeys[key] {
				continue
			}
			pins = append(pins, r)
			pinKeys[key] = true
			break
		}
	}
	if len(receipts) <= displayLimit {
		return receipts
	}

	out := make([]iomesh.MemoryOpsDigestReceipt, 0, displayLimit)
	used := map[string]bool{}
	reserve := len(pins)
	for _, r := range receipts {
		key := receiptPinKey(r)
		isPin := pinKeys[key]
		if !isPin && len(out)+reserve >= displayLimit {
			continue
		}
		if used[key] {
			continue
		}
		out = append(out, r)
		used[key] = true
		if isPin {
			reserve--
		}
		if len(out) >= displayLimit {
			break
		}
	}
	for _, p := range pins {
		if len(out) >= displayLimit {
			break
		}
		key := receiptPinKey(p)
		if used[key] {
			continue
		}
		out = append(out, p)
		used[key] = true
	}
	sortReceiptsNewestFirst(out)
	return out
}

func sortReceiptsNewestFirst(receipts []iomesh.MemoryOpsDigestReceipt) {
	// Insertion sort — receipt windows are small (≤50).
	for i := 1; i < len(receipts); i++ {
		cur := receipts[i]
		curT := receiptEventTime(cur)
		j := i - 1
		for j >= 0 {
			prevT := receiptEventTime(receipts[j])
			// Missing times keep original (newest-first export) order.
			if curT.IsZero() || prevT.IsZero() {
				break
			}
			if !curT.After(prevT) {
				break
			}
			receipts[j+1] = receipts[j]
			j--
		}
		receipts[j+1] = cur
	}
}

func formatDigestReceiptWindowReason(res *iomesh.MemoryOpsDigestResult, missing []string) string {
	if res == nil || len(missing) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("receipt window newest-first")
	if res.FetchLimit > 0 {
		fmt.Fprintf(&b, " · limit=%d", res.FetchLimit)
	}
	n := res.FetchedN
	if n == 0 {
		n = len(res.Receipts)
	}
	fmt.Fprintf(&b, " · n=%d", n)
	if s := strings.TrimSpace(res.Since); s != "" {
		fmt.Fprintf(&b, " · since=%s", s)
	}
	if s := strings.TrimSpace(res.AsOf); s != "" {
		fmt.Fprintf(&b, " · as_of=%s", s)
	}
	if s := strings.TrimSpace(res.FetchedNewest); s != "" {
		fmt.Fprintf(&b, " · newest=%s", s)
	}
	if s := strings.TrimSpace(res.FetchedOldest); s != "" {
		fmt.Fprintf(&b, " · oldest=%s", s)
	}
	fmt.Fprintf(&b, " · %s not in this receipt set", strings.Join(missing, ","))
	return b.String()
}
