package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/iomesh"
)

// Sticky cite-both (#419 / #460): classify receipts from provenance/tags as
// well as source_hint, and pin older mesh-stamped turns into the active set
// when they are in the fetched export. When --require-sources still misses a
// class, pin the newest stamped turn of that class from the explicit tenant
// palace. That palace pin is age-agnostic: a day or week ops window does not
// drop a stamp that is already on disk. A class with no stamped turns stays
// an honest miss. Unstamped palace_timeline and other-org palaces do not
// satisfy. Never invent mesh. dual_write OFF.

const (
	opsDigestLimitDefault  = 20
	opsDigestLimitCiteBoth = 50 // MCP ops_digest_export cap
	palaceCiteScanFileCap  = 8000
	palaceCiteScanMaxBytes = 1 << 20
)

// palaceCiteTiers are the on-disk palace tiers that hold turns (kernel layout).
// Indexes, wal, and other org directories are not walked.
var palaceCiteTiers = []string{
	"tier-1-working",
	"tier-2-contextual",
	"tier-3-archival",
	"tier-4-semantic",
}

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

func digestResultOrStub(res *iomesh.MemoryOpsDigestResult, fetchLimit int) *iomesh.MemoryOpsDigestResult {
	if res == nil {
		res = &iomesh.MemoryOpsDigestResult{}
	}
	if res.FetchLimit <= 0 && fetchLimit > 0 {
		res.FetchLimit = fetchLimit
	}
	return res
}

// mergeDigestReceipts unions MCP palace receipts into the HTTP/sidecar set.
// Same-id rows keep export source_hint (often palace_timeline) and take
// provenance/tags from whichever side has them — never invent mesh.
func mergeDigestReceipts(dst, src []iomesh.MemoryOpsDigestReceipt) []iomesh.MemoryOpsDigestReceipt {
	if len(src) == 0 {
		return dst
	}
	out := append([]iomesh.MemoryOpsDigestReceipt(nil), dst...)
	idx := map[string]int{}
	indexReceipt := func(i int, r iomesh.MemoryOpsDigestReceipt) {
		idx[receiptPinKey(r)] = i
		if id := strings.TrimSpace(r.ID); id != "" {
			idx["id:"+id] = i
		}
	}
	for i, r := range out {
		indexReceipt(i, r)
	}
	for _, r := range src {
		key := receiptPinKey(r)
		i, ok := idx[key]
		if !ok {
			if id := strings.TrimSpace(r.ID); id != "" {
				i, ok = idx["id:"+id]
			}
		}
		if ok {
			out[i] = enrichDigestReceipt(out[i], r)
			continue
		}
		out = append(out, r)
		indexReceipt(len(out)-1, r)
	}
	return out
}

func enrichDigestReceipt(base, extra iomesh.MemoryOpsDigestReceipt) iomesh.MemoryOpsDigestReceipt {
	if strings.TrimSpace(base.Provenance.SourceHint) == "" {
		base.Provenance.SourceHint = extra.Provenance.SourceHint
	}
	if strings.TrimSpace(base.Provenance.SourceStep) == "" {
		base.Provenance.SourceStep = extra.Provenance.SourceStep
	}
	base.Tags = unionDigestTags(base.Tags, extra.Tags)
	if strings.TrimSpace(base.Summary) == "" {
		base.Summary = extra.Summary
	}
	if strings.TrimSpace(base.EventTime) == "" {
		base.EventTime = extra.EventTime
	}
	if strings.TrimSpace(base.Pointer) == "" {
		base.Pointer = extra.Pointer
	}
	return base
}

func unionDigestTags(a, b []string) []string {
	if len(b) == 0 {
		return a
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(a)+len(b))
	for _, s := range append(append([]string{}, a...), b...) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// salvageDigestReceiptsJSON pulls complete receipt objects from truncated or
// envelope-wrapped MCP text so a 20KB display cap cannot wipe private RCA.
func salvageDigestReceiptsJSON(raw string) []iomesh.MemoryOpsDigestReceipt {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	marker := `"receipts"`
	idx := strings.Index(raw, marker)
	if idx < 0 {
		return nil
	}
	rest := raw[idx+len(marker):]
	brack := strings.IndexByte(rest, '[')
	if brack < 0 {
		return nil
	}
	dec := json.NewDecoder(strings.NewReader(rest[brack:]))
	tok, err := dec.Token()
	if err != nil || tok != json.Delim('[') {
		return nil
	}
	var out []iomesh.MemoryOpsDigestReceipt
	for dec.More() {
		var item json.RawMessage
		if err := dec.Decode(&item); err != nil {
			break
		}
		var r iomesh.MemoryOpsDigestReceipt
		if json.Unmarshal(item, &r) != nil {
			continue
		}
		if strings.TrimSpace(r.ID) == "" && strings.TrimSpace(r.Summary) == "" &&
			strings.TrimSpace(r.SourceHint) == "" && len(r.Tags) == 0 {
			continue
		}
		out = append(out, r)
	}
	return out
}

func fillDigestWindowFromRaw(res *iomesh.MemoryOpsDigestResult, raw string) {
	if res == nil {
		return
	}
	if res.Since == "" {
		res.Since = jsonStringField(raw, "since")
	}
	if res.AsOf == "" {
		res.AsOf = jsonStringField(raw, "as_of")
	}
	if res.Window == "" {
		res.Window = jsonStringField(raw, "window")
	}
	if res.Horizon == "" {
		res.Horizon = jsonStringField(raw, "horizon")
	}
}

func jsonStringField(raw, key string) string {
	i := strings.Index(raw, `"`+key+`"`)
	if i < 0 {
		return ""
	}
	rest := raw[i+len(key)+2:]
	colon := strings.IndexByte(rest, ':')
	if colon < 0 {
		return ""
	}
	dec := json.NewDecoder(strings.NewReader(strings.TrimSpace(rest[colon+1:])))
	var s string
	if dec.Decode(&s) != nil {
		return ""
	}
	return strings.TrimSpace(s)
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
	if extra := formatPalaceOutsideWindow(res, missing); extra != "" {
		fmt.Fprintf(&b, " · %s", extra)
	}
	return b.String()
}

func formatPalaceOutsideWindow(res *iomesh.MemoryOpsDigestResult, missing []string) string {
	if res == nil || len(res.PalaceOutsideWindow) == 0 || len(missing) == 0 {
		return ""
	}
	want := map[string]bool{}
	for _, m := range missing {
		want[m] = true
	}
	var parts []string
	for _, o := range res.PalaceOutsideWindow {
		if o.Count <= 0 || !want[o.Class] {
			continue
		}
		newest := strings.TrimSpace(o.Newest)
		if newest == "" {
			newest = "(none)"
		}
		parts = append(parts, fmt.Sprintf("%s on palace outside window · %s_on_disk=%d · newest_%s=%s",
			o.Class, o.Class, o.Count, o.Class, newest))
	}
	return strings.Join(parts, " · ")
}

// supplementCiteBothReceipts fills required classes the day export omitted (#460).
// It merges a week ops-digest export, then reads the named local palace for this
// tenant. The newest stamped turn of each still-missing class is pinned with no
// day/week age gate. An unnamed default palace is not scanned. Other orgs are
// not read. Never invents mesh. dual_write OFF.
func (rt *Runtime) supplementCiteBothReceipts(ctx context.Context, res *iomesh.MemoryOpsDigestResult, required []string, window, horizon string, fetchLimit int, asOf string) {
	if rt == nil || res == nil || len(required) == 0 {
		return
	}
	if len(missingCiteClasses(res.Receipts, required)) == 0 {
		return
	}
	if !strings.EqualFold(strings.TrimSpace(window), "week") {
		bound := strings.TrimSpace(asOf)
		if bound == "" {
			bound = strings.TrimSpace(res.AsOf)
		}
		rt.mergeCiteWindowReceipts(ctx, res, horizon, fetchLimit, bound)
	}
	missing := missingCiteClasses(res.Receipts, required)
	if len(missing) == 0 {
		return
	}
	root, explicit := resolvePalaceRoot(rt.memory.PalaceRoot, rt.mcpPalaceArgs())
	if !explicit || !palaceDirExists(root) {
		return
	}
	pins := scanPalaceCiteClasses(root, rt.memoryTenant(), missing)
	if len(pins) > 0 {
		res.Receipts = mergeDigestReceipts(res.Receipts, pins)
	}
}

func (rt *Runtime) mergeCiteWindowReceipts(ctx context.Context, res *iomesh.MemoryOpsDigestResult, horizon string, fetchLimit int, asOf string) {
	if res == nil {
		return
	}
	if rt.syncMemoryReady() {
		wider, err := rt.mesh.ExportOpsDigest(ctx, rt.memoryTenant(), iomesh.MemoryOpsDigestOptions{
			Window:  "week",
			Horizon: horizon,
			Limit:   fetchLimit,
			AsOf:    asOf,
		})
		if err == nil && wider != nil {
			res.Receipts = mergeDigestReceipts(res.Receipts, wider.Receipts)
		}
	}
	if !rt.mcpMemoryReady() {
		return
	}
	mcpRes, mcpText, err := rt.fetchMCPOpsDigest(ctx, "week", horizon, fetchLimit, asOf, 0)
	if err == nil && mcpRes != nil {
		res.Receipts = mergeDigestReceipts(res.Receipts, mcpRes.Receipts)
		return
	}
	if err == nil {
		if salvaged := salvageDigestReceiptsJSON(mcpText); len(salvaged) > 0 {
			res.Receipts = mergeDigestReceipts(res.Receipts, salvaged)
		}
	}
}

func missingCiteClasses(receipts []iomesh.MemoryOpsDigestReceipt, required []string) []string {
	present := map[string]bool{}
	for _, r := range receipts {
		if c := ClassifyDigestReceipt(r); c != "" {
			present[c] = true
		}
	}
	var missing []string
	for _, req := range required {
		if !present[req] {
			missing = append(missing, req)
		}
	}
	return missing
}

func citeTimeUsable(t time.Time) bool {
	return !t.IsZero() && t.Year() >= 2000
}

// scanPalaceCiteClasses reads one explicit tenant palace for required classes
// the export missed. The newest stamped turn of each class is returned as a
// real receipt, including stamps older than the day/week ops window (#460).
// A stamp with no usable event time is used only when no timed stamp exists.
// Unstamped palace_timeline does not match a missing mesh class. Other org
// directories are not read. Never invents a turn.
func scanPalaceCiteClasses(root, tenant string, classes []string) []iomesh.MemoryOpsDigestReceipt {
	dir, ok := palaceTenantDir(root, tenant)
	if !ok || len(classes) == 0 {
		return nil
	}
	want := map[string]bool{}
	for _, c := range classes {
		want[c] = true
	}
	type palaceCiteHit struct {
		receipt iomesh.MemoryOpsDigestReceipt
		when    time.Time
	}
	type palaceCiteClass struct {
		newest     palaceCiteHit
		hasNewest  bool
		untimed    iomesh.MemoryOpsDigestReceipt
		hasUntimed bool
	}
	byClass := map[string]*palaceCiteClass{}
	scanned := 0
	for _, tier := range palaceCiteTiers {
		tierDir := filepath.Join(dir, tier)
		_ = filepath.WalkDir(tierDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d == nil || d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(d.Name()), ".json") {
				return nil
			}
			if scanned >= palaceCiteScanFileCap {
				return fs.SkipAll
			}
			scanned++
			r, ok := readPalaceTurn(path)
			if !ok {
				return nil
			}
			class := ClassifyDigestReceipt(r)
			if !want[class] {
				return nil
			}
			acc := byClass[class]
			if acc == nil {
				acc = &palaceCiteClass{}
				byClass[class] = acc
			}
			when := receiptEventTime(r)
			if citeTimeUsable(when) {
				if !acc.hasNewest || when.After(acc.newest.when) {
					acc.newest = palaceCiteHit{receipt: r, when: when}
					acc.hasNewest = true
				}
				return nil
			}
			if !acc.hasUntimed {
				acc.untimed = r
				acc.hasUntimed = true
			}
			return nil
		})
	}
	var pins []iomesh.MemoryOpsDigestReceipt
	for _, class := range classes {
		acc := byClass[class]
		if acc == nil {
			continue
		}
		if acc.hasNewest {
			pins = append(pins, acc.newest.receipt)
			continue
		}
		if acc.hasUntimed {
			pins = append(pins, acc.untimed)
		}
	}
	return pins
}

func palaceTenantDir(root, tenant string) (string, bool) {
	root = filepath.Clean(strings.TrimSpace(root))
	tenant = strings.TrimSpace(tenant)
	if root == "" || root == "." || tenant == "" {
		return "", false
	}
	if tenant != filepath.Base(tenant) || tenant == "." || tenant == ".." {
		return "", false
	}
	dir := filepath.Join(root, tenant)
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return "", false
	}
	return dir, true
}

func readPalaceTurn(path string) (iomesh.MemoryOpsDigestReceipt, bool) {
	f, err := os.Open(path)
	if err != nil {
		return iomesh.MemoryOpsDigestReceipt{}, false
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Size() <= 0 || st.Size() > palaceCiteScanMaxBytes {
		return iomesh.MemoryOpsDigestReceipt{}, false
	}
	raw, err := io.ReadAll(io.LimitReader(f, palaceCiteScanMaxBytes+1))
	if err != nil || len(raw) == 0 || len(raw) > palaceCiteScanMaxBytes {
		return iomesh.MemoryOpsDigestReceipt{}, false
	}
	var r iomesh.MemoryOpsDigestReceipt
	if json.Unmarshal(raw, &r) != nil {
		return iomesh.MemoryOpsDigestReceipt{}, false
	}
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return iomesh.MemoryOpsDigestReceipt{}, false
	}
	if strings.TrimSpace(r.Summary) == "" {
		if c, ok := m["content"].(map[string]any); ok {
			if s, ok := c["summary"].(string); ok && strings.TrimSpace(s) != "" {
				r.Summary = strings.TrimSpace(s)
			} else if s, ok := c["full"].(string); ok {
				r.Summary = strings.TrimSpace(s)
			}
		}
	}
	if extra := stringListFromAny(m["temporal_tags"]); len(extra) > 0 {
		r.Tags = unionDigestTags(r.Tags, extra)
	}
	if strings.TrimSpace(r.ID) == "" {
		return iomesh.MemoryOpsDigestReceipt{}, false
	}
	if strings.TrimSpace(r.EventTime) == "" {
		if s, ok := m["timestamp"].(string); ok {
			r.EventTime = strings.TrimSpace(s)
		}
	}
	return r, true
}

func stringListFromAny(v any) []string {
	switch t := v.(type) {
	case nil:
		return nil
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			s, ok := item.(string)
			if !ok {
				continue
			}
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return t
	default:
		return nil
	}
}
