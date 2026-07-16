package agent

import (
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/router"
)

func TestFormatMemoryRecallSnippet_JSON(t *testing.T) {
	raw := `{
  "memories": [
    {"id": "m1", "content": "Prefer DeepSeek for routine tasks", "score": 0.91, "tier": 4},
    {"id": "m2", "summary": "Session used worktree isolation", "score": 0.7, "tier": 2},
    {"id": "m3", "text": "", "score": 0.1}
  ]
}`
	snip, n := FormatMemoryRecallSnippet(raw, 2)
	if n != 2 {
		t.Fatalf("hits=%d snip=%q", n, snip)
	}
	if !strings.Contains(snip, "m1") || !strings.Contains(snip, "DeepSeek") {
		t.Fatal(snip)
	}
	if !strings.Contains(snip, "+1 more") {
		t.Fatal(snip)
	}
}

func TestFormatMemoryRecallSnippet_Raw(t *testing.T) {
	snip, n := FormatMemoryRecallSnippet("plain recall text", 8)
	if n != 1 || !strings.Contains(snip, "plain recall") {
		t.Fatalf("%d %q", n, snip)
	}
}

func TestFormatMemoryRecallSnippet_Empty(t *testing.T) {
	snip, n := FormatMemoryRecallSnippet("", 8)
	if snip != "" || n != 0 {
		t.Fatalf("%q %d", snip, n)
	}
}

func TestMemoryStatus_Disabled(t *testing.T) {
	rt := &Runtime{}
	s := rt.MemoryStatus()
	if !strings.Contains(s, "disabled") {
		t.Fatal(s)
	}
}

func TestConfigureMemory_SystemNote(t *testing.T) {
	rt := &Runtime{
		messages: []router.Message{{Role: "system", Content: "base"}},
	}
	rt.ConfigureMemory(MemoryConfig{
		Enabled: true, Server: "aion-memory", Tenant: "dept.eng",
		AutoRecall: true, RecallLimit: 5,
	})
	if !rt.memCfg.Enabled || rt.memCfg.RecallLimit != 5 {
		t.Fatalf("%+v", rt.memCfg)
	}
	if !strings.Contains(rt.messages[0].Content, "Palace memory") {
		t.Fatal(rt.messages[0].Content)
	}
	s := rt.MemoryStatus()
	if !strings.Contains(s, "enabled") || !strings.Contains(s, "dept.eng") {
		t.Fatal(s)
	}
}
