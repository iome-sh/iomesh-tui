package setup

import (
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/config"
)

func TestBuildDriftReport_HonestyNeedles(t *testing.T) {
	cfg := config.Default()
	cfg.Memory.Enabled = true
	cfg.Memory.DualWrite = false
	cfg.Memory.PullContinuous = false
	cfg.Memory.AnalyzeContinuous = false

	snap := DriftSnapshot{
		MCPAttached:    true,
		MCPServerCount: 1,
		MemoryServerOK: true,
		MemoryServer:   "iomesh-memory-mcp",
		MeshEnabled:    false,
		PullRunning:    false,
		AnalyzeRunning: false,
		DualWrite:      false,
		MemoryEnabled:  true,
	}
	rep := BuildDriftReport(cfg, snap)
	if !rep.OK {
		t.Fatalf("expected OK residual-honest: findings=%v", rep.Findings)
	}
	if !rep.DualWriteHonest {
		t.Fatal("dual_write must be honest")
	}
	text := FormatDriftText(rep)
	for _, needle := range []string{
		"dual_write OFF",
		"not Memory GA",
		"≠ invent install green",
		"package wire ≠ Connected",
		"report-only",
		// s1542: guided repair tip (not pure “no auto-repair” alone)
		"/setup repair plan",
		"/setup repair apply --yes",
		"safe steps only",
		"guided repair",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("format missing honesty needle %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Connected: yes") || strings.Contains(text, "Memory GA: yes") {
		t.Fatalf("must not invent green:\n%s", text)
	}
	if strings.TrimSpace(text) == "" {
		t.Fatal("format non-empty")
	}
}

func TestBuildDriftReport_DualWriteDishonest(t *testing.T) {
	cfg := config.Default()
	cfg.Memory.Enabled = true
	cfg.Memory.DualWrite = true
	snap := DriftSnapshot{
		MCPAttached:    true,
		MCPServerCount: 1,
		MemoryServerOK: true,
		MemoryEnabled:  true,
	}
	rep := BuildDriftReport(cfg, snap)
	if rep.OK {
		t.Fatal("OK must be false when dual_write true")
	}
	if rep.DualWriteHonest {
		t.Fatal("DualWriteHonest must be false")
	}
	found := false
	for _, f := range rep.Findings {
		if strings.Contains(f, "dual_write=true") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected dual_write finding: %v", rep.Findings)
	}
	text := FormatDriftText(rep)
	if !strings.Contains(text, "dual_write OFF") {
		t.Fatalf("honesty footer required even when dishonest config:\n%s", text)
	}
}

func TestBuildDriftReport_PullMismatch(t *testing.T) {
	cfg := config.Default()
	cfg.Memory.Enabled = true
	cfg.Memory.DualWrite = false
	cfg.Memory.PullContinuous = true
	cfg.IOMesh.Enabled = true
	snap := DriftSnapshot{
		MCPAttached:    true,
		MCPServerCount: 1,
		MemoryServerOK: true,
		MeshEnabled:    true,
		PullRunning:    false, // mismatch
		MemoryEnabled:  true,
	}
	rep := BuildDriftReport(cfg, snap)
	if rep.OK {
		t.Fatal("OK must be false on pull_continuous vs not running")
	}
	found := false
	for _, f := range rep.Findings {
		if strings.Contains(f, "pull_continuous=true") && strings.Contains(f, "pull not running") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected pull mismatch finding: %v", rep.Findings)
	}
	text := FormatDriftText(rep)
	if !strings.Contains(text, "/setup pull start") {
		t.Fatalf("expected next-step note for pull:\n%s", text)
	}
	if strings.TrimSpace(text) == "" {
		t.Fatal("format non-empty")
	}
}

func TestBuildDriftReport_AnalyzeMismatch(t *testing.T) {
	cfg := config.Default()
	cfg.Memory.Enabled = true
	cfg.Memory.DualWrite = false
	cfg.Memory.AnalyzeContinuous = true
	snap := DriftSnapshot{
		MCPAttached:    true,
		MCPServerCount: 1,
		MemoryServerOK: true,
		AnalyzeRunning: false,
		MemoryEnabled:  true,
	}
	rep := BuildDriftReport(cfg, snap)
	if rep.OK {
		t.Fatal("OK must be false on analyze_continuous vs not running")
	}
	found := false
	for _, f := range rep.Findings {
		if strings.Contains(f, "analyze_continuous=true") && strings.Contains(f, "analyze not running") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected analyze mismatch finding: %v", rep.Findings)
	}
	text := FormatDriftText(rep)
	if !strings.Contains(text, "/setup analyze start") {
		t.Fatalf("expected analyze next step:\n%s", text)
	}
}

func TestBuildDriftReport_NoConfig(t *testing.T) {
	rep := BuildDriftReport(nil, DriftSnapshot{})
	if rep.ConfigPresent {
		t.Fatal("config not present")
	}
	if rep.OK {
		t.Fatal("OK false without config")
	}
	text := FormatDriftText(rep)
	if !strings.Contains(text, DriftHonestyFooter) && !strings.Contains(text, "dual_write OFF") {
		t.Fatalf("honesty required:\n%s", text)
	}
}

func TestBuildDriftReport_MeshPullWantedDisabled(t *testing.T) {
	cfg := config.Default()
	cfg.Memory.DualWrite = false
	cfg.Memory.PullContinuous = true
	cfg.IOMesh.Enabled = false
	snap := DriftSnapshot{
		MemoryServerOK: true,
		MeshEnabled:    false,
		PullRunning:    false,
	}
	// Give memory a free pass so mesh finding is visible
	cfg.Memory.Enabled = false
	rep := BuildDriftReport(cfg, snap)
	found := false
	for _, f := range rep.Findings {
		if strings.Contains(f, "mesh disabled") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected mesh disabled finding: %v", rep.Findings)
	}
	if rep.OK {
		t.Fatal("OK false when mesh pull wanted but disabled")
	}
}

func TestCheckDrift_Alias(t *testing.T) {
	cfg := config.Default()
	cfg.Memory.DualWrite = false
	a := BuildDriftReport(cfg, DriftSnapshot{MemoryServerOK: true})
	b := CheckDrift(cfg, DriftSnapshot{MemoryServerOK: true})
	if a.OK != b.OK || a.DualWriteHonest != b.DualWriteHonest {
		t.Fatalf("alias mismatch: %+v vs %+v", a, b)
	}
}

func TestFormatDriftText_NonEmpty(t *testing.T) {
	text := FormatDriftText(DriftReport{Findings: nil, Notes: nil})
	if strings.TrimSpace(text) == "" {
		t.Fatal("non-empty")
	}
	if !strings.Contains(text, "honesty:") {
		t.Fatalf("missing honesty line:\n%s", text)
	}
}
