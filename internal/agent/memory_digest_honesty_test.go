package agent

import (
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/iomesh"
)

func TestFormatOpsDigest_InsufficientSignalWhenEmpty(t *testing.T) {
	out := formatOpsDigest(&iomesh.MemoryOpsDigestResult{
		Window:   "day",
		Horizon:  "ops",
		Patterns: nil,
		Receipts: nil,
		Honesty:  iomesh.MemoryOpsDigestHonesty{NeverInventGA: true, DualWriteDefault: "off"},
	}, 4000)
	if !strings.Contains(out, digestInsufficientSignal) {
		t.Fatalf("want insufficient-signal, got %q", out)
	}
	if strings.Contains(out, "patterns: (none)") {
		t.Fatalf("legacy empty marker should not appear: %q", out)
	}
	if !strings.Contains(out, "catalog list ≠ consume") || !strings.Contains(out, "dual_write=off") {
		t.Fatalf("honesty pins missing: %q", out)
	}
}

func TestFormatOpsDigest_RejectsRateClaimMissingDenom(t *testing.T) {
	out := formatOpsDigest(&iomesh.MemoryOpsDigestResult{
		Window:  "day",
		Horizon: "ops",
		Patterns: []iomesh.MemoryOpsDigestPattern{
			{ID: "bad", Kind: "volume", Subject: "dept.cs", Summary: "renewal risk rate +12% this week", Count: 3},
			{ID: "ok", Kind: "recurrence", Subject: "dept.ops", Summary: "checkout p95 pages", Count: 8, Window: "24h"},
		},
		Honesty: iomesh.MemoryOpsDigestHonesty{NeverInventGA: true},
	}, 4000)
	if !strings.Contains(out, "rejected 1 rate claim") {
		t.Fatalf("expected rate rejection note: %q", out)
	}
	if strings.Contains(out, "+12%") || strings.Contains(out, "renewal risk") {
		t.Fatalf("rejected rate claim should not render: %q", out)
	}
	if !strings.Contains(out, "dept.ops") || !strings.Contains(out, "n=8") || !strings.Contains(out, "window=24h") {
		t.Fatalf("honest non-rate pattern missing: %q", out)
	}
}

func TestFormatOpsDigest_AcceptsRateWithNofNAndWindow(t *testing.T) {
	out := formatOpsDigest(&iomesh.MemoryOpsDigestResult{
		Window:  "week",
		Horizon: "ops",
		Patterns: []iomesh.MemoryOpsDigestPattern{
			{
				ID:               "p-rate",
				Kind:             "volume",
				Subject:          "dept.gtm",
				Summary:          "churn rate elevated",
				Count:            3,
				Total:            25,
				Window:           "7d",
				ComparisonWindow: "prior_7d",
				Links:            []string{"https://tickets.example/T-1", "https://tickets.example/T-2"},
				Score:            0.8,
			},
		},
		Honesty: iomesh.MemoryOpsDigestHonesty{DualWriteDefault: "off", NeverInventGA: true},
	}, 4000)
	if !strings.Contains(out, "3 of 25") || !strings.Contains(out, "window=7d") || !strings.Contains(out, "vs=prior_7d") {
		t.Fatalf("rate line missing n/N/window: %q", out)
	}
	if !strings.Contains(out, "links=2") || !strings.Contains(out, "<https://tickets.example/T-1>") {
		t.Fatalf("links missing: %q", out)
	}
	// Do not paste the free-form "churn rate elevated" quote on a rate line.
	if strings.Contains(out, "churn rate elevated") {
		t.Fatalf("pasted rate quote should be omitted: %q", out)
	}
}

func TestFormatOpsDigest_AllRateRejectedYieldsInsufficientSignal(t *testing.T) {
	out := formatOpsDigest(&iomesh.MemoryOpsDigestResult{
		Window:  "day",
		Horizon: "ops",
		Patterns: []iomesh.MemoryOpsDigestPattern{
			{Summary: "error rate 40%", Count: 2},
			{Summary: "12% checkout failures"},
		},
	}, 4000)
	if !strings.Contains(out, digestInsufficientSignal) {
		t.Fatalf("want insufficient-signal when all rates rejected: %q", out)
	}
	if !strings.Contains(out, "rejected 2 rate claim") {
		t.Fatalf("want rejection count: %q", out)
	}
}

func TestFormatOpsDigest_ReceiptsPointersAndHashesNotRawText(t *testing.T) {
	raw := "Customer Jane said billing failed on invoice #4411"
	out := formatOpsDigest(&iomesh.MemoryOpsDigestResult{
		Window:  "day",
		Horizon: "ops",
		Patterns: []iomesh.MemoryOpsDigestPattern{
			{Subject: "dept.billing", Kind: "recurrence", Count: 4, Window: "24h"},
		},
		Receipts: []iomesh.MemoryOpsDigestReceipt{
			{
				ID:         "evt-9",
				EventTime:  "2026-08-04T10:00:00Z",
				Summary:    raw,
				SourceHint: "palace_timeline",
				Pointer:    "https://tickets.example/BILL-9",
			},
			{
				ID:          "evt-10",
				Summary:     "another private note",
				AccountHash: "acct_deadbeef",
			},
		},
	}, 4000)
	if strings.Contains(out, raw) || strings.Contains(out, "Jane") || strings.Contains(out, "another private note") {
		t.Fatalf("raw customer text leaked: %q", out)
	}
	if !strings.Contains(out, "pointer=https://tickets.example/BILL-9") {
		t.Fatalf("explicit pointer missing: %q", out)
	}
	if !strings.Contains(out, "hash=") {
		t.Fatalf("hash missing: %q", out)
	}
	if !strings.Contains(out, "hash=acct_deadbeef") {
		t.Fatalf("account_hash not preferred: %q", out)
	}
	if !strings.Contains(out, "pointer=id:evt-10") {
		t.Fatalf("id pointer fallback missing: %q", out)
	}
}

func TestPatternClaimsRate(t *testing.T) {
	if !patternClaimsRate(iomesh.MemoryOpsDigestPattern{Summary: "error rate climbed"}) {
		t.Fatal("expected rate claim")
	}
	if !patternClaimsRate(iomesh.MemoryOpsDigestPattern{Summary: "failures at 12%"}) {
		t.Fatal("expected percent claim")
	}
	if !patternClaimsRate(iomesh.MemoryOpsDigestPattern{Total: 10, Count: 2, Window: "day"}) {
		t.Fatal("explicit total implies rate framing")
	}
	if patternClaimsRate(iomesh.MemoryOpsDigestPattern{Summary: "subject recurred 5 times", Count: 5, Window: "24h"}) {
		t.Fatal("plain recurrence is not a rate claim")
	}
}

func TestDigestHonesty_DualWriteRemainsOff(t *testing.T) {
	cfg := DefaultMemoryConfig()
	if cfg.DualWrite {
		t.Fatal("dual_write must default OFF for private overlay")
	}
}
