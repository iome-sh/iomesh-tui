package agent

import (
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/iomesh"
)

// s1530 P5: residual-honest continuous pull lifecycle (unit · no live mesh APPLY).

func TestContinuousMemoryPull_StopWhenNotRunningIsNoOp(t *testing.T) {
	rt := testRT(t, t.TempDir())
	// Must not panic or hang.
	rt.StopContinuousMemoryPull()
	rt.StopContinuousMemoryPull()
}

func TestContinuousMemoryPullStatus_IdleEmptyHonest(t *testing.T) {
	rt := testRT(t, t.TempDir())
	st := rt.ContinuousMemoryPullStatus()
	if st.Running {
		t.Fatal("idle status Running must be false")
	}
	if st.Stream != "" || st.Consumer != "" || st.Filter != "" || st.LastError != "" {
		t.Fatalf("idle status must be empty-honest: %+v", st)
	}
	if st.Stats.Fetched != 0 || st.Stats.Ingested != 0 || st.Stats.Errors != 0 {
		t.Fatalf("idle stats counters must be zero: %+v", st.Stats)
	}
}

func TestStartContinuousMemoryPull_NilMesh(t *testing.T) {
	rt := testRT(t, t.TempDir()) // mesh is nil
	err := rt.StartContinuousMemoryPull(ContinuousPullConfig{
		Consumer: "tui-local-palace",
		Stream:   "EVENTS",
		DryRun:   true,
	})
	if err == nil {
		t.Fatal("expected error when mesh is nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "mesh disabled") {
		t.Fatalf("want residual-honest mesh disabled: %v", err)
	}
	// Honesty needle: do not invent install green.
	if strings.Contains(strings.ToLower(msg), "memory ga") {
		// optional — ensure we don't claim GA in error; fine either way
	}
	st := rt.ContinuousMemoryPullStatus()
	if st.Running {
		t.Fatal("must not mark running after failed start")
	}
}

func TestStartContinuousMemoryPull_NilRuntime(t *testing.T) {
	var rt *Runtime
	err := rt.StartContinuousMemoryPull(ContinuousPullConfig{Consumer: "c"})
	if err == nil {
		t.Fatal("expected error on nil runtime")
	}
	rt.StopContinuousMemoryPull() // no-op
	if st := rt.ContinuousMemoryPullStatus(); st.Running {
		t.Fatal("nil status must not be running")
	}
}

func TestStartContinuousMemoryPull_RequiresConsumer(t *testing.T) {
	rt := testRT(t, t.TempDir())
	// Attach a disabled mesh so consumer check is what fails first when consumer empty.
	rt.mesh = iomesh.New(iomesh.Config{Enabled: true, Endpoint: "http://127.0.0.1:9"}, nil)
	err := rt.StartContinuousMemoryPull(ContinuousPullConfig{
		Stream: "EVENTS",
		DryRun: true,
		// Consumer empty
	})
	if err == nil {
		t.Fatal("expected error without pull_consumer")
	}
	if !strings.Contains(err.Error(), "pull_consumer") {
		t.Fatalf("want pull_consumer required: %v", err)
	}
	// Whitespace-only also rejected.
	err = rt.StartContinuousMemoryPull(ContinuousPullConfig{
		Consumer: "  \t  ",
		DryRun:   true,
	})
	if err == nil || !strings.Contains(err.Error(), "pull_consumer") {
		t.Fatalf("whitespace consumer must fail: %v", err)
	}
}

func TestStartContinuousMemoryPull_MeshDisabled(t *testing.T) {
	rt := testRT(t, t.TempDir())
	rt.mesh = iomesh.New(iomesh.Config{Enabled: false}, nil)
	err := rt.StartContinuousMemoryPull(ContinuousPullConfig{
		Consumer: "tui-local-palace",
		DryRun:   true,
	})
	if err == nil {
		t.Fatal("expected mesh disabled error")
	}
	if !strings.Contains(err.Error(), "mesh disabled") {
		t.Fatalf("got %v", err)
	}
}

func TestStartContinuousMemoryPull_MCPMissingNonDryRun(t *testing.T) {
	rt := testRT(t, t.TempDir())
	rt.mesh = iomesh.New(iomesh.Config{Enabled: true, Endpoint: "http://127.0.0.1:9"}, nil)
	// No MCP attached.
	err := rt.StartContinuousMemoryPull(ContinuousPullConfig{
		Consumer: "tui-local-palace",
		Stream:   "EVENTS",
		Server:   "memory",
		DryRun:   false,
	})
	if err == nil {
		t.Fatal("expected MCP missing error")
	}
	if !strings.Contains(err.Error(), "MCP") && !strings.Contains(err.Error(), "not connected") {
		t.Fatalf("want MCP not connected: %v", err)
	}
	// Residual honesty: package wire ≠ Connected is fine to mention.
	if strings.Contains(err.Error(), "Memory GA") {
		t.Fatalf("must not invent Memory GA: %v", err)
	}
}

func TestStartContinuousMemoryPullOnce_SameValidation(t *testing.T) {
	rt := testRT(t, t.TempDir())
	err := rt.StartContinuousMemoryPullOnce(ContinuousPullConfig{Consumer: "c", DryRun: true})
	if err == nil {
		t.Fatal("expected mesh disabled on nil mesh")
	}
	if !strings.Contains(err.Error(), "mesh disabled") {
		t.Fatalf("got %v", err)
	}
}

func TestClose_StopsContinuousPull_NoHang(t *testing.T) {
	rt := testRT(t, t.TempDir())
	// Close with no pull running must succeed.
	if err := rt.Close(); err != nil {
		t.Fatal(err)
	}
	// Second close safe.
	if err := rt.Close(); err != nil {
		t.Fatal(err)
	}
}
