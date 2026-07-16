package subagent

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// slowRunner blocks for delay, counting peak concurrency.
type slowRunner struct {
	delay   time.Duration
	active  *atomic.Int32
	peak    *atomic.Int32
	summary string
}

func (s *slowRunner) Run(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	n := s.active.Add(1)
	for {
		cur := s.peak.Load()
		if n <= cur || s.peak.CompareAndSwap(cur, n) {
			break
		}
	}
	defer s.active.Add(-1)
	select {
	case <-time.After(s.delay):
		return s.summary, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func TestSpawnMany_MaxParallel(t *testing.T) {
	var active, peak atomic.Int32
	const (
		workers = 8
		delay   = 120 * time.Millisecond
	)
	m := NewManager(Config{
		Enabled:       true,
		MaxConcurrent: workers,
		MaxBatch:      32,
		Workspace:     t.TempDir(),
	}, func(ctx context.Context, sp SpawnParams) (Runner, error) {
		return &slowRunner{delay: delay, active: &active, peak: &peak, summary: "ok-" + sp.Spec.Description}, nil
	}, nil)

	specs := make([]Spec, 0, workers)
	for i := 0; i < workers; i++ {
		specs = append(specs, Spec{
			Prompt:       fmt.Sprintf("task %d", i),
			Description:  fmt.Sprintf("t%d", i),
			SubagentType: TypeExplore,
		})
	}

	start := time.Now()
	batch, err := m.SpawnMany(context.Background(), specs, SpawnManyOptions{Wait: true})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if batch.Spawned != workers || batch.Completed != workers {
		t.Fatalf("batch=%+v", batch)
	}
	// Sequential would be workers*delay; parallel should be ~1× delay (+slack).
	if elapsed > delay*3 {
		t.Fatalf("expected parallel wall time, elapsed=%s peak=%d", elapsed, peak.Load())
	}
	if peak.Load() < int32(workers) {
		t.Fatalf("peak concurrency %d < %d", peak.Load(), workers)
	}
}

func TestSpawnMany_RespectsSemaphore(t *testing.T) {
	var active, peak atomic.Int32
	const (
		maxConc = 3
		tasks   = 9
		delay   = 40 * time.Millisecond
	)
	m := NewManager(Config{
		Enabled: true, MaxConcurrent: maxConc, MaxBatch: 32, Workspace: t.TempDir(),
	}, func(ctx context.Context, sp SpawnParams) (Runner, error) {
		return &slowRunner{delay: delay, active: &active, peak: &peak, summary: "x"}, nil
	}, nil)

	specs := make([]Spec, tasks)
	for i := range specs {
		specs[i] = Spec{Prompt: "p", Description: fmt.Sprintf("%d", i), SubagentType: TypeExplore}
	}
	batch, err := m.SpawnMany(context.Background(), specs, SpawnManyOptions{Wait: true})
	if err != nil {
		t.Fatal(err)
	}
	if batch.Completed != tasks {
		t.Fatalf("%+v", batch)
	}
	if peak.Load() > int32(maxConc) {
		t.Fatalf("peak %d exceeded max %d", peak.Load(), maxConc)
	}
	if peak.Load() < int32(maxConc) {
		t.Fatalf("peak %d never reached max %d", peak.Load(), maxConc)
	}
}

func TestSpawnMany_MaxBatch(t *testing.T) {
	m := NewManager(Config{Enabled: true, MaxBatch: 2, Workspace: t.TempDir()}, func(ctx context.Context, sp SpawnParams) (Runner, error) {
		return &fakeRunner{summary: "ok"}, nil
	}, nil)
	_, err := m.SpawnMany(context.Background(), []Spec{
		{Prompt: "a"}, {Prompt: "b"}, {Prompt: "c"},
	}, SpawnManyOptions{})
	if err == nil {
		t.Fatal("expected max_batch error")
	}
}

func TestSpawnMany_NoWaitReturnsRunning(t *testing.T) {
	m := NewManager(Config{Enabled: true, MaxConcurrent: 4, Workspace: t.TempDir()}, func(ctx context.Context, sp SpawnParams) (Runner, error) {
		return &slowRunner{delay: 80 * time.Millisecond, active: &atomic.Int32{}, peak: &atomic.Int32{}, summary: "later"}, nil
	}, nil)
	batch, err := m.SpawnMany(context.Background(), []Spec{
		{Prompt: "a", Description: "a"},
		{Prompt: "b", Description: "b"},
	}, SpawnManyOptions{Wait: false})
	if err != nil {
		t.Fatal(err)
	}
	if batch.Spawned != 2 {
		t.Fatalf("%+v", batch)
	}
	ids := []string{batch.Results[0].ID, batch.Results[1].ID}
	finals, err := m.WaitAll(context.Background(), ids)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range finals {
		if r.Status != StatusCompleted {
			t.Fatalf("%+v", r)
		}
	}
}

func TestGetMany(t *testing.T) {
	m := NewManager(Config{Enabled: true, Workspace: t.TempDir()}, func(ctx context.Context, sp SpawnParams) (Runner, error) {
		return &fakeRunner{summary: "done"}, nil
	}, nil)
	r, err := m.Spawn(context.Background(), Spec{Prompt: "x", Background: false})
	if err != nil {
		t.Fatal(err)
	}
	got := m.GetMany([]string{r.ID, "missing"})
	if len(got) != 2 || got[0].Status != StatusCompleted || got[1].Status != StatusFailed {
		t.Fatalf("%+v", got)
	}
}
