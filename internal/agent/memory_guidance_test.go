package agent

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/mcp"
)

// TestMemoryNextStepLines_HonestyNeedles pins s1831 residual-honest next-step
// after /memory status|help|digest (peer of OnboardNextStepLines s1825 · IntegrationsNextStepLines s1727).
func TestMemoryNextStepLines_HonestyNeedles(t *testing.T) {
	lines := MemoryNextStepLines()
	if len(lines) == 0 {
		t.Fatal("empty memory next-step lines")
	}
	out := strings.Join(lines, "\n")
	for _, want := range []string{
		"dual path residual-honest after memory surfaces",
		"TUI/session running",
		"/setup preflight",
		"/setup reload",
		"/memory digest",
		"/onboard next memory",
		"memory-pull",
		"cold start",
		"restart iomesh",
		"iomesh setup preflight",
		"iomesh memory pull",
		"dual_write OFF",
		"not Memory GA",
		"local-primary",
		"package wire ≠ Connected",
		"soft ≠ invent live dogfood",
		"s1831",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("memory next-step missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Memory GA shipped") {
		t.Fatalf("must not invent dual_write ON / Memory GA shipped:\n%s", out)
	}
	if strings.Contains(out, "Connected: yes") {
		t.Fatalf("must not invent Connected green:\n%s", out)
	}
}

// TestMemoryAdvancedStatus_S1831NextStep pins s1831 next-step on /memory status inventory.
func TestMemoryAdvancedStatus_S1831NextStep(t *testing.T) {
	rt := testRT(t, t.TempDir())
	out, err := rt.MemoryAdvancedStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"s1831",
		"/setup preflight",
		"/setup reload",
		"/memory digest",
		"dual_write OFF",
		"not Memory GA",
		"local-primary",
		"package wire ≠ Connected",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("MemoryAdvancedStatus missing s1831 next-step %q in:\n%s", want, out)
		}
	}
}

// s1291: MemoryAdvancedAgentGuidanceNote residual-honest needles.
func TestMemoryAdvancedAgentGuidanceNote_HonestyNeedles(t *testing.T) {
	out := MemoryAdvancedAgentGuidanceNote()
	if out == "" {
		t.Fatal("empty guidance note")
	}
	for _, want := range []string{
		"prefer_shorter_hops",
		"HITL",
		"--i-confirm",
		"not medical",
		"dual_write OFF",
		"multi-hop lite",
		"not Memory GA",
		"patterns/anomalies",
		"not OTel",
		"not invent GA window engine",
		"facts-as-of",
		"supersede",
		"/memory related",
		"|patterns|anomalies|timeline|compact-status|semantic|ingest-event|trigger-compact|status", // slash mirrors (s1296+s1301+s1311)
		"timeline",
		"compact-status",
		"semantic",
		"ingest-event",
		"trigger-compact",
		"tier-4 semantic",
		"s138 T1",
		"not conversation turn",
		"memory_trigger_compact", // s1311 HITL lock
		"not invent compaction green",
		"memory-advanced-agent",
		"memory_retrieve",
		"K4 lite",
		"A3 lite",
		"opt-in",
		"memory_write",
		"/memory write",
		"entity_key",
		"seed_query",
		"not a new live tools=N",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("guidance missing %q in:\n%s", want, out)
		}
	}
	// Must not invent Memory GA product success language.
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "Memory GA green") {
		t.Fatalf("must not invent Memory GA claim: %s", out)
	}
}

// s1291: AttachMCP injects <memory-advanced> system note when MCP manager is present.
func TestAttachMCP_InjectsMemoryAdvancedGuidance(t *testing.T) {
	cInR, cInW := io.Pipe()
	cOutR, cOutW := io.Pipe()
	go mockIntegrationsMCP(cOutW, cInR)

	mut := false
	cl := mcp.NewClientForTest(mcp.ServerConfig{Name: "aion-scenario", Command: "x", Mutating: &mut}, cInW, cOutR, nil)
	defer cl.Close()
	if err := cl.InitForTest(context.Background()); err != nil {
		t.Fatal(err)
	}

	mgr := mcp.NewManagerEmpty(nil)
	mgr.Attach(cl)
	rt := testRT(t, t.TempDir())
	rt.AttachMCP(mgr)

	sys := rt.Messages()[0].Content
	if !strings.Contains(sys, "<memory-advanced>") {
		t.Fatalf("want <memory-advanced> system note: %s", sys)
	}
	if !strings.Contains(sys, "</memory-advanced>") {
		t.Fatalf("want closed memory-advanced tag: %s", sys)
	}
	for _, want := range []string{
		"prefer_shorter_hops",
		"HITL",
		"not medical",
		"dual_write OFF",
		"multi-hop lite",
		"not Memory GA",
		"patterns/anomalies",
	} {
		if !strings.Contains(sys, want) {
			t.Fatalf("memory-advanced note missing %q: %s", want, sys)
		}
	}
	// Integrations note still present (s1251 + s1291 co-inject).
	if !strings.Contains(sys, "<integrations>") {
		t.Fatalf("want <integrations> note too: %s", sys)
	}
	if !strings.Contains(sys, "<mcp>") {
		t.Fatalf("want <mcp> note too: %s", sys)
	}
}
