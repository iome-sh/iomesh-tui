package subagent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestWaitAllParallel_FasterThanSerial(t *testing.T) {
	const n = 6
	const delay = 80 * time.Millisecond
	var active, peak atomic.Int32
	m := NewManager(Config{Enabled: true, MaxConcurrent: n, Workspace: t.TempDir()}, func(ctx context.Context, sp SpawnParams) (Runner, error) {
		return &slowRunner{delay: delay, active: &active, peak: &peak, summary: "ok"}, nil
	}, nil)

	specs := make([]Spec, n)
	for i := range specs {
		specs[i] = Spec{Prompt: fmt.Sprintf("t%d", i), Description: fmt.Sprintf("%d", i), SubagentType: TypeExplore}
	}
	start := time.Now()
	batch, err := m.SpawnMany(context.Background(), specs, SpawnManyOptions{Wait: true})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if batch.Completed != n {
		t.Fatalf("%+v", batch)
	}
	if batch.ElapsedMS <= 0 {
		t.Fatal("elapsed")
	}
	// Parallel wait should finish near 1× delay, not n×.
	if elapsed > delay*3 {
		t.Fatalf("elapsed=%s want ~%s (parallel join)", elapsed, delay)
	}
	if peak.Load() < int32(n) {
		t.Fatalf("peak=%d", peak.Load())
	}
}

func TestApplyMany_Parallel(t *testing.T) {
	repo := initTestRepo(t)
	gw := NewGitWorktree()
	m := NewManager(Config{Enabled: true, Workspace: repo, MaxConcurrent: 4}, func(ctx context.Context, sp SpawnParams) (Runner, error) {
		name := "out-" + filepath.Base(sp.Workspace) + ".txt"
		_ = os.WriteFile(filepath.Join(sp.Workspace, name), []byte("x\n"), 0o644)
		return &fakeRunner{summary: name}, nil
	}, nil)
	m.SetWorktreeBackend(gw)

	const n = 4
	specs := make([]Spec, n)
	for i := range specs {
		specs[i] = Spec{
			Prompt: "write", Description: fmt.Sprintf("w%d", i),
			SubagentType: TypeGeneralPurpose, Isolation: IsolationWorktree,
		}
	}
	batch, err := m.SpawnMany(context.Background(), specs, SpawnManyOptions{Wait: true})
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, n)
	for _, r := range batch.Results {
		if r.ID == "" || r.WorktreePath == "" {
			t.Fatalf("missing worktree: %+v", r)
		}
		ids = append(ids, r.ID)
	}
	start := time.Now()
	applies, err := m.ApplyMany(context.Background(), ids, true)
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 10*time.Second {
		t.Fatal("apply too slow")
	}
	ok := 0
	for _, a := range applies {
		if a.Error == "" && len(a.Applied) > 0 {
			ok++
		}
	}
	if ok < n {
		t.Fatalf("applies ok=%d want %d: %+v", ok, n, applies)
	}
}

func TestSpawnMany_ApplyAfter(t *testing.T) {
	repo := initTestRepo(t)
	m := NewManager(Config{Enabled: true, Workspace: repo, MaxConcurrent: 3}, func(ctx context.Context, sp SpawnParams) (Runner, error) {
		_ = os.WriteFile(filepath.Join(sp.Workspace, "f-"+filepath.Base(sp.Workspace)+".txt"), []byte("1\n"), 0o644)
		return &fakeRunner{summary: "ok"}, nil
	}, nil)
	m.SetWorktreeBackend(NewGitWorktree())

	batch, err := m.SpawnMany(context.Background(), []Spec{
		{Prompt: "a", Isolation: IsolationWorktree, SubagentType: TypeGeneralPurpose},
		{Prompt: "b", Isolation: IsolationWorktree, SubagentType: TypeGeneralPurpose},
	}, SpawnManyOptions{Wait: true, ApplyAfter: true, RemoveAfterApply: true})
	if err != nil {
		t.Fatal(err)
	}
	if batch.AppliesOK < 2 {
		t.Fatalf("applies: ok=%d failed=%d %+v", batch.AppliesOK, batch.AppliesFailed, batch.Applies)
	}
}

func TestSpawnMany_ApplyAfterRequiresWait(t *testing.T) {
	m := NewManager(Config{Enabled: true, Workspace: t.TempDir()}, func(ctx context.Context, sp SpawnParams) (Runner, error) {
		return &fakeRunner{summary: "ok"}, nil
	}, nil)
	_, err := m.SpawnMany(context.Background(), []Spec{{Prompt: "x"}}, SpawnManyOptions{ApplyAfter: true, Wait: false})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAbsoluteCaps(t *testing.T) {
	m := NewManager(Config{Enabled: true, MaxConcurrent: 9999, MaxBatch: 9999, Workspace: t.TempDir()}, nil, nil)
	if m.MaxConcurrent() != AbsoluteMaxConcurrent {
		t.Fatalf("conc=%d", m.MaxConcurrent())
	}
	if m.MaxBatch() != AbsoluteMaxBatch {
		t.Fatalf("batch=%d", m.MaxBatch())
	}
}

func TestDefaultMaxConcurrentIsThirtyTwo(t *testing.T) {
	m := NewManager(Config{Enabled: true, Workspace: t.TempDir()}, func(ctx context.Context, sp SpawnParams) (Runner, error) {
		return &fakeRunner{summary: "ok"}, nil
	}, nil)
	if m.MaxConcurrent() != DefaultMaxConcurrent || DefaultMaxConcurrent != 32 {
		t.Fatalf("max=%d", m.MaxConcurrent())
	}
}
