package agent

import (
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/iomesh"
)

func boolPtr(v bool) *bool { return &v }

func TestFormatOpsDigest_RejectsNoDeltaRecapAsInsufficientSignal(t *testing.T) {
	out := formatOpsDigest(&iomesh.MemoryOpsDigestResult{
		Window:  "day",
		Horizon: "ops",
		Patterns: []iomesh.MemoryOpsDigestPattern{
			{ID: "r1", Kind: "recap", Subject: "dept.ops", Summary: "what is true about checkout today", Count: 3},
			{ID: "r2", Kind: "status", Summary: "status quo: systems remain healthy", Window: "24h"},
			{ID: "r3", Summary: "current state unchanged vs yesterday"},
		},
		Honesty: iomesh.MemoryOpsDigestHonesty{NeverInventGA: true, DualWriteDefault: "off"},
	}, 4000)
	if !strings.Contains(out, digestInsufficientSignal) {
		t.Fatalf("want insufficient-signal for no-delta window: %q", out)
	}
	if !strings.Contains(out, "rejected 3 no-delta recap") {
		t.Fatalf("want no-delta rejection note: %q", out)
	}
	if strings.Contains(out, "what is true") || strings.Contains(out, "status quo") || strings.Contains(out, "current state") {
		t.Fatalf("recap text must not render: %q", out)
	}
	if !strings.Contains(out, "delta≠recap") || !strings.Contains(out, "external≠heartbeat") {
		t.Fatalf("honesty pins missing: %q", out)
	}
}

func TestFormatOpsDigest_AcceptsDeltaVsPriorWindow(t *testing.T) {
	out := formatOpsDigest(&iomesh.MemoryOpsDigestResult{
		Window:  "day",
		Horizon: "ops",
		Patterns: []iomesh.MemoryOpsDigestPattern{
			{
				ID:               "d1",
				Kind:             "stall",
				Subject:          "dept.cs",
				Summary:          "new support theme: billing cancel",
				Count:            5,
				Window:           "24h",
				ComparisonWindow: "prior_24h",
			},
			{
				ID:      "d2",
				Kind:    "delta",
				Subject: "dept.ops",
				Summary: "paging shape shifted vs prior",
				Count:   2,
				Window:  "24h",
				Delta:   boolPtr(true),
			},
			{ID: "bad", Kind: "recap", Summary: "what is true inventory"},
		},
	}, 4000)
	if strings.Contains(out, digestInsufficientSignal) {
		t.Fatalf("delta patterns should not yield insufficient-signal: %q", out)
	}
	if !strings.Contains(out, "patterns (2)") {
		t.Fatalf("want 2 accepted deltas: %q", out)
	}
	if !strings.Contains(out, "dept.cs") || !strings.Contains(out, "vs=prior_24h") {
		t.Fatalf("delta with comparison window missing: %q", out)
	}
	if !strings.Contains(out, "dept.ops") {
		t.Fatalf("explicit delta pattern missing: %q", out)
	}
	if !strings.Contains(out, "rejected 1 no-delta recap") {
		t.Fatalf("want mixed rejection note: %q", out)
	}
	if strings.Contains(out, "what is true inventory") {
		t.Fatalf("recap must not render beside deltas: %q", out)
	}
}

func TestFormatOpsDigest_ExplicitDeltaFalseRejected(t *testing.T) {
	out := formatOpsDigest(&iomesh.MemoryOpsDigestResult{
		Window:  "week",
		Horizon: "ops",
		Patterns: []iomesh.MemoryOpsDigestPattern{
			{Subject: "dept.gtm", Summary: "weekly market pulse", Count: 4, Window: "7d", Delta: boolPtr(false)},
		},
	}, 4000)
	if !strings.Contains(out, digestInsufficientSignal) {
		t.Fatalf("delta=false without change signal → insufficient-signal: %q", out)
	}
	if !strings.Contains(out, "rejected 1 no-delta recap") {
		t.Fatalf("want no-delta note: %q", out)
	}
}

func TestFormatOpsDigest_ExternalColorThirdPaneNotHeartbeat(t *testing.T) {
	out := formatOpsDigest(&iomesh.MemoryOpsDigestResult{
		Window:  "day",
		Horizon: "ops",
		Patterns: []iomesh.MemoryOpsDigestPattern{
			{Kind: "recurrence", Subject: "dept.ops", Count: 3, Window: "24h", ComparisonWindow: "prior_24h"},
		},
		Receipts: []iomesh.MemoryOpsDigestReceipt{
			{ID: "m1", Summary: "mesh P2 checkout", SourceHint: "mesh_consume", Pointer: "https://tickets.example/M-1"},
			{ID: "p1", Summary: "private RCA", SourceHint: "palace_timeline", AccountHash: "acct_aaa"},
			{ID: "e1", Summary: "sponsored TAM color blurb", SourceHint: "external", Pointer: "https://sponsor.example/x"},
			{ID: "e2", Summary: "demand feed row", SourceHint: "sponsored_demand"},
		},
	}, 4000)
	if !strings.Contains(out, "receipts (2)") {
		t.Fatalf("first-party receipts should exclude external: %q", out)
	}
	if !strings.Contains(out, "external color (2) · not heartbeat · not cite-both") {
		t.Fatalf("want labeled external pane: %q", out)
	}
	if !strings.Contains(out, "source=external") || !strings.Contains(out, "source=sponsored_demand") {
		t.Fatalf("external source hints missing in pane: %q", out)
	}
	if strings.Contains(out, "sponsored TAM color blurb") || strings.Contains(out, "demand feed row") {
		t.Fatalf("raw external text must not leak: %q", out)
	}
	// Heartbeat receipts stay pointers+hashes; mesh cite path unchanged.
	if !strings.Contains(out, "source=mesh_consume") || !strings.Contains(out, "source=palace_timeline") {
		t.Fatalf("first-party sources missing: %q", out)
	}
}

func TestFormatRequireSourcesCheck_ExternalNeverSatisfiesCiteBoth(t *testing.T) {
	res := &iomesh.MemoryOpsDigestResult{
		Receipts: []iomesh.MemoryOpsDigestReceipt{
			{ID: "e1", Summary: "TAM color", SourceHint: "external"},
			{ID: "e2", Summary: "sponsored row", SourceHint: "sponsored"},
			{ID: "p1", Summary: "private RCA", SourceHint: "palace_timeline"},
		},
	}
	out := FormatRequireSourcesCheck(res, []string{"mesh", "private"})
	if !strings.Contains(out, "require-sources: miss") || !strings.Contains(out, "missing=mesh") {
		t.Fatalf("want mesh miss when only external+private: %q", out)
	}
	if !strings.Contains(out, "cited=private") {
		t.Fatalf("want private cited: %q", out)
	}
	if !strings.Contains(out, digestExternalCitePin) {
		t.Fatalf("want external cite pin: %q", out)
	}
	if strings.Contains(out, "require-sources: ok") {
		t.Fatalf("external must not satisfy cite-both: %q", out)
	}
}

func TestFormatRequireSourcesCheck_ExternalOnlyMissBoth(t *testing.T) {
	res := &iomesh.MemoryOpsDigestResult{
		Receipts: []iomesh.MemoryOpsDigestReceipt{
			{ID: "e1", Summary: "demand feed", SourceHint: "demand_feed"},
			{ID: "e2", Summary: "tam", SourceHint: "tam_color"},
		},
	}
	out := FormatRequireSourcesCheck(res, []string{"mesh", "private"})
	if !strings.Contains(out, "missing=mesh,private") || !strings.Contains(out, "cited=(none)") {
		t.Fatalf("want both missing: %q", out)
	}
	if !strings.Contains(out, digestExternalCitePin) {
		t.Fatalf("want external pin: %q", out)
	}
}

func TestClassifyDigestSourceHint_External(t *testing.T) {
	cases := []struct {
		hint string
		want string
	}{
		{"external", DigestSourceExternal},
		{"sponsored", DigestSourceExternal},
		{"tam_color", DigestSourceExternal},
		{"demand_feed", DigestSourceExternal},
		{"EXTERNAL-SPONSOR", DigestSourceExternal},
		{"sponsored_demand", DigestSourceExternal},
		{"mesh_consume", DigestSourceMesh}, // first-party consume still mesh
		{"palace_timeline", DigestSourcePrivate},
	}
	for _, tc := range cases {
		if got := ClassifyDigestSourceHint(tc.hint); got != tc.want {
			t.Fatalf("hint=%q got=%q want=%q", tc.hint, got, tc.want)
		}
	}
}

func TestPatternIsNoDelta(t *testing.T) {
	if !patternIsNoDelta(iomesh.MemoryOpsDigestPattern{Kind: "recap", Summary: "daily truth"}) {
		t.Fatal("recap kind should be no-delta")
	}
	if !patternIsNoDelta(iomesh.MemoryOpsDigestPattern{Summary: "what is true for this account"}) {
		t.Fatal("what-is-true summary should be no-delta")
	}
	if patternIsNoDelta(iomesh.MemoryOpsDigestPattern{
		Kind: "recap", Summary: "what is true", ComparisonWindow: "prior_day",
	}) {
		t.Fatal("comparison window makes a change vs prior")
	}
	if patternIsNoDelta(iomesh.MemoryOpsDigestPattern{
		Kind: "recurrence", Subject: "dept.ops", Count: 4, Window: "24h",
	}) {
		t.Fatal("plain activity pattern is not a recap")
	}
	if !patternIsNoDelta(iomesh.MemoryOpsDigestPattern{
		Summary: "weekly pulse", Delta: boolPtr(false),
	}) {
		t.Fatal("explicit delta=false should be no-delta")
	}
}

func TestDeltaBriefs_DualWriteRemainsOff(t *testing.T) {
	if DefaultMemoryConfig().DualWrite {
		t.Fatal("dual_write must default OFF")
	}
}
