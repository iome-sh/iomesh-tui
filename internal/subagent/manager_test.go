package subagent

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type fakeRunner struct {
	summary string
	err     error
	delay   time.Duration
	calls   *atomic.Int32
}

func (f *fakeRunner) Run(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if f.calls != nil {
		f.calls.Add(1)
	}
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return f.summary, f.err
}

func TestSpawn_SyncExplore(t *testing.T) {
	var calls atomic.Int32
	m := NewManager(Config{Enabled: true, Workspace: t.TempDir(), MaxDepth: 2}, func(ctx context.Context, sp SpawnParams) (Runner, error) {
		if sp.AllowWrite {
			t.Fatal("explore should not allow write")
		}
		if !sp.AllowShell {
			t.Fatal("explore default execute allows shell")
		}
		if sp.Definition.Type != TypeExplore {
			t.Fatalf("type=%s", sp.Definition.Type)
		}
		return &fakeRunner{summary: "found main.go", calls: &calls}, nil
	}, nil)

	res, err := m.Spawn(context.Background(), Spec{
		Prompt:       "find entrypoints",
		Description:  "scan entry",
		SubagentType: TypeExplore,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusCompleted {
		t.Fatalf("status=%s err=%s", res.Status, res.Error)
	}
	if res.Summary != "found main.go" {
		t.Fatalf("summary=%q", res.Summary)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestSpawn_BackgroundAndGet(t *testing.T) {
	m := NewManager(Config{Enabled: true, Workspace: t.TempDir()}, func(ctx context.Context, sp SpawnParams) (Runner, error) {
		return &fakeRunner{summary: "bg-done", delay: 80 * time.Millisecond}, nil
	}, nil)

	res, err := m.Spawn(context.Background(), Spec{
		Prompt:       "slow work",
		SubagentType: TypeGeneralPurpose,
		Background:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status == StatusCompleted {
		// Might already be done on very fast machines; still ok.
	}
	id := res.ID
	waited, err := m.Wait(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if waited.Status != StatusCompleted || waited.Summary != "bg-done" {
		t.Fatalf("%+v", waited)
	}
}

func TestSpawn_MaxDepth(t *testing.T) {
	m := NewManager(Config{Enabled: true, MaxDepth: 1, Workspace: t.TempDir()}, func(ctx context.Context, sp SpawnParams) (Runner, error) {
		return &fakeRunner{summary: "ok"}, nil
	}, nil)
	_, err := m.Spawn(context.Background(), Spec{Prompt: "x", Depth: 1})
	if err == nil || !strings.Contains(err.Error(), "max subagent depth") {
		t.Fatalf("err=%v", err)
	}
}

func TestSpawn_ResumeFrom(t *testing.T) {
	var sawPrompt string
	m := NewManager(Config{Enabled: true, Workspace: t.TempDir()}, func(ctx context.Context, sp SpawnParams) (Runner, error) {
		return &fakeRunner{summary: "first"}, nil
	}, nil)
	first, err := m.Spawn(context.Background(), Spec{Prompt: "research", SubagentType: TypeExplore})
	if err != nil {
		t.Fatal(err)
	}
	m.factory = func(ctx context.Context, sp SpawnParams) (Runner, error) {
		return &runCapture{onRun: func(sys, user string) (string, error) {
			sawPrompt = user
			return "second", nil
		}}, nil
	}
	second, err := m.Spawn(context.Background(), Spec{
		Prompt:       "continue",
		SubagentType: TypePlan,
		ResumeFrom:   first.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.Status != StatusCompleted {
		t.Fatal(second.Error)
	}
	if !strings.Contains(sawPrompt, "first") || !strings.Contains(sawPrompt, "continue") {
		t.Fatalf("prompt=%q", sawPrompt)
	}
}

type runCapture struct {
	onRun func(sys, user string) (string, error)
}

func (r *runCapture) Run(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return r.onRun(systemPrompt, userPrompt)
}

func TestResolveCapabilities(t *testing.T) {
	def := Builtins()[TypeGeneralPurpose]
	w, s := resolveCapabilities(def, CapabilityReadOnly)
	if w || s {
		t.Fatal("read-only")
	}
	w, s = resolveCapabilities(def, CapabilityReadWrite)
	if !w || s {
		t.Fatal("read-write")
	}
	w, s = resolveCapabilities(def, CapabilityExecute)
	if w || !s {
		t.Fatal("execute")
	}
}

func TestEffectiveTools(t *testing.T) {
	tools := EffectiveTools(false, true, false)
	joined := strings.Join(tools, ",")
	if strings.Contains(joined, "write_file") {
		t.Fatal(joined)
	}
	if !strings.Contains(joined, "run_shell") {
		t.Fatal(joined)
	}
}

func TestDisabled(t *testing.T) {
	m := NewManager(Config{Enabled: false}, nil, nil)
	_, err := m.Spawn(context.Background(), Spec{Prompt: "x"})
	if err == nil {
		t.Fatal("expected disabled error")
	}
}
