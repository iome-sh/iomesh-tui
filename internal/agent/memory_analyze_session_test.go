package agent

import (
	"strings"
	"testing"
	"time"
)

// s1534 P6: residual-honest analyze tick lifecycle (unit · no live mesh APPLY).

func TestAnalyzeTick_StopWhenNotRunningIsNoOp(t *testing.T) {
	rt := testRT(t, t.TempDir())
	// Must not panic or hang.
	rt.StopAnalyzeTick()
	rt.StopAnalyzeTick()
}

func TestAnalyzeTickStatus_IdleEmptyHonest(t *testing.T) {
	rt := testRT(t, t.TempDir())
	st := rt.AnalyzeTickStatus()
	if st.Running {
		t.Fatal("idle status Running must be false")
	}
	if st.Mode != "" || st.IntervalSec != 0 || st.LastSummary != "" || st.LastError != "" || st.TickCount != 0 {
		t.Fatalf("idle status must be empty-honest: %+v", st)
	}
	if !st.LastAt.IsZero() {
		t.Fatalf("idle LastAt must be zero: %v", st.LastAt)
	}
}

func TestStartAnalyzeTick_NilRuntime(t *testing.T) {
	var rt *Runtime
	err := rt.StartAnalyzeTick(AnalyzeTickConfig{Mode: "status"})
	if err == nil {
		t.Fatal("expected error on nil runtime")
	}
	rt.StopAnalyzeTick() // no-op
	if st := rt.AnalyzeTickStatus(); st.Running {
		t.Fatal("nil status must not be running")
	}
	if snap := rt.DriftSnapshot(); snap.AnalyzeRunning || snap.MemoryEnabled {
		t.Fatalf("nil DriftSnapshot must be empty-honest: %+v", snap)
	}
}

func TestStartAnalyzeTick_InvalidMode(t *testing.T) {
	rt := testRT(t, t.TempDir())
	err := rt.StartAnalyzeTick(AnalyzeTickConfig{Mode: "bogus"})
	if err == nil {
		t.Fatal("expected invalid mode error")
	}
	if !strings.Contains(err.Error(), "status") || !strings.Contains(err.Error(), "digest") {
		t.Fatalf("want mode residual: %v", err)
	}
	if st := rt.AnalyzeTickStatus(); st.Running {
		t.Fatal("must not mark running after failed start")
	}
}

func TestStartAnalyzeTick_DigestRequiresMemoryEnabled(t *testing.T) {
	rt := testRT(t, t.TempDir())
	// Default memory disabled.
	err := rt.StartAnalyzeTick(AnalyzeTickConfig{Mode: "digest", IntervalSec: 60})
	if err == nil {
		t.Fatal("expected digest mode to require memory enabled")
	}
	if !strings.Contains(err.Error(), "enabled") {
		t.Fatalf("want residual-honest enabled: %v", err)
	}
	if strings.Contains(err.Error(), "Memory GA") && !strings.Contains(err.Error(), "≠ invent") && !strings.Contains(err.Error(), "not") {
		// Must not invent Memory GA as a positive claim.
		t.Fatalf("must not invent Memory GA: %v", err)
	}
	st := rt.AnalyzeTickStatus()
	if st.Running {
		t.Fatal("must not mark running after failed digest start")
	}
}

func TestStartAnalyzeTick_StatusModeWorksWithMemoryDisabled(t *testing.T) {
	rt := testRT(t, t.TempDir())
	// status mode uses MemoryStatusLine even when hooks disabled.
	err := rt.StartAnalyzeTickOnce(AnalyzeTickConfig{
		Mode:        "status",
		IntervalSec: 60,
	})
	if err != nil {
		t.Fatalf("status once: %v", err)
	}
	// Wait for once goroutine to finish.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		st := rt.AnalyzeTickStatus()
		if !st.Running && st.TickCount >= 1 {
			if st.LastSummary == "" {
				t.Fatalf("expected LastSummary from MemoryStatusLine: %+v", st)
			}
			if !strings.Contains(st.LastSummary, "memory:") {
				t.Fatalf("summary should be status line: %q", st.LastSummary)
			}
			if st.Mode != "status" {
				t.Fatalf("mode=%q", st.Mode)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	st := rt.AnalyzeTickStatus()
	t.Fatalf("timeout waiting for once tick: %+v", st)
}

func TestStartAnalyzeTick_IntervalDefaultsAndFloor(t *testing.T) {
	rt := testRT(t, t.TempDir())
	// 0 → 300 default.
	if err := rt.StartAnalyzeTick(AnalyzeTickConfig{Mode: "status", IntervalSec: 0}); err != nil {
		t.Fatal(err)
	}
	st := rt.AnalyzeTickStatus()
	if st.IntervalSec != defaultAnalyzeIntervalSec {
		t.Fatalf("default interval want %d got %d", defaultAnalyzeIntervalSec, st.IntervalSec)
	}
	rt.StopAnalyzeTick()

	// Below floor → 30.
	if err := rt.StartAnalyzeTick(AnalyzeTickConfig{Mode: "status", IntervalSec: 5}); err != nil {
		t.Fatal(err)
	}
	st = rt.AnalyzeTickStatus()
	if st.IntervalSec != minAnalyzeIntervalSec {
		t.Fatalf("floor want %d got %d", minAnalyzeIntervalSec, st.IntervalSec)
	}
	rt.StopAnalyzeTick()
}

func TestStartAnalyzeTick_RestartWhileRunning(t *testing.T) {
	rt := testRT(t, t.TempDir())
	if err := rt.StartAnalyzeTick(AnalyzeTickConfig{Mode: "status", IntervalSec: 60}); err != nil {
		t.Fatal(err)
	}
	// Immediate first tick should land quickly.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if rt.AnalyzeTickStatus().TickCount >= 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err := rt.StartAnalyzeTick(AnalyzeTickConfig{Mode: "status", IntervalSec: 90}); err != nil {
		t.Fatal(err)
	}
	st := rt.AnalyzeTickStatus()
	if !st.Running {
		t.Fatal("restarted tick should be running")
	}
	if st.IntervalSec != 90 {
		t.Fatalf("restarted interval=%d", st.IntervalSec)
	}
	rt.StopAnalyzeTick()
	if st := rt.AnalyzeTickStatus(); st.Running {
		t.Fatal("after stop Running must be false")
	}
}

func TestClose_StopsAnalyzeTick_NoHang(t *testing.T) {
	rt := testRT(t, t.TempDir())
	if err := rt.StartAnalyzeTick(AnalyzeTickConfig{Mode: "status", IntervalSec: 60}); err != nil {
		t.Fatal(err)
	}
	// Close must stop analyze and return promptly.
	done := make(chan error, 1)
	go func() { done <- rt.Close() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Close hung with analyze tick running")
	}
	if st := rt.AnalyzeTickStatus(); st.Running {
		t.Fatal("after Close Running must be false")
	}
	// Second close safe.
	if err := rt.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestDriftSnapshot_ResidualHonest(t *testing.T) {
	rt := testRT(t, t.TempDir())
	snap := rt.DriftSnapshot()
	if snap.MCPAttached || snap.MCPServerCount != 0 || snap.MemoryServerOK {
		t.Fatalf("no MCP: %+v", snap)
	}
	if snap.MeshEnabled || snap.PullRunning || snap.AnalyzeRunning {
		t.Fatalf("no mesh/pull/analyze: %+v", snap)
	}
	if snap.DualWrite || snap.MemoryEnabled {
		t.Fatalf("defaults dual_write OFF / memory off: %+v", snap)
	}
	// Memory server name defaults empty-honest to "memory" for lookup identity.
	if snap.MemoryServer != "memory" {
		t.Fatalf("MemoryServer default=%q", snap.MemoryServer)
	}

	rt.AttachMemory(MemoryConfig{Enabled: true, Server: "palace", DualWrite: false})
	snap = rt.DriftSnapshot()
	if !snap.MemoryEnabled {
		t.Fatal("MemoryEnabled after AttachMemory")
	}
	if snap.DualWrite {
		t.Fatal("dual_write must remain OFF")
	}
	if snap.MemoryServer != "palace" {
		t.Fatalf("MemoryServer=%q", snap.MemoryServer)
	}
	if snap.MemoryServerOK {
		t.Fatal("MemoryServerOK must be false without MCP client (package wire ≠ Connected)")
	}
}

func TestAnalyzeTick_SummaryTruncated(t *testing.T) {
	// Reuse package truncateRunes (integrations.go) — cap ~2k for LastSummary.
	long := strings.Repeat("x", maxAnalyzeSummaryChars+50)
	got := truncateRunes(long, maxAnalyzeSummaryChars)
	if n := len([]rune(got)); n != maxAnalyzeSummaryChars {
		t.Fatalf("want %d runes got %d", maxAnalyzeSummaryChars, n)
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("want ellipsis: suffix %q", got[len(got)-3:])
	}
}
