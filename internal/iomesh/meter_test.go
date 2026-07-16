package iomesh

import (
	"strings"
	"testing"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/router"
)

func TestUsageMeter_RecordAndFormat(t *testing.T) {
	c := New(Config{}, nil)
	c.RecordLLMCall(router.CallMeta{
		ModelName: "deepseek-v4-flash", Duration: 10 * time.Millisecond, EstimatedUSD: 0.001,
	}, router.Usage{PromptTokens: 5, CompletionTokens: 7, TotalTokens: 12}, nil)
	c.RecordLLMCall(router.CallMeta{
		ModelName: "deepseek-v4-flash", Duration: 5 * time.Millisecond, EstimatedUSD: 0.002,
	}, router.Usage{TotalTokens: 3}, nil)
	c.RecordLLMCall(router.CallMeta{
		ModelName: "grok-4.5", Duration: time.Millisecond, EstimatedUSD: 0.01,
	}, router.Usage{TotalTokens: 100}, assertErr{})

	snap := c.Usage()
	if snap.Calls != 3 || snap.Errors != 1 || snap.Tokens != 115 {
		t.Fatalf("%+v", snap)
	}
	if len(snap.ByModel) != 2 {
		t.Fatalf("by_model=%+v", snap.ByModel)
	}
	out := FormatUsage(snap)
	if !strings.Contains(out, "deepseek-v4-flash") || !strings.Contains(out, "grok-4.5") {
		t.Fatal(out)
	}
	if !strings.Contains(out, "totals:") {
		t.Fatal(out)
	}
}

type assertErr struct{}

func (assertErr) Error() string { return "boom" }
