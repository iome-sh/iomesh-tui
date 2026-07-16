package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/mcp"
	"github.com/iome-sh/iomesh-tui/internal/router"
)

func TestTruncateBytes(t *testing.T) {
	if got := truncateBytes("hello", 100); got != "hello" {
		t.Fatalf("got %q", got)
	}
	long := strings.Repeat("a", 100)
	got := truncateBytes(long, 20)
	if len(got) > 20 {
		t.Fatalf("len=%d got=%q", len(got), got)
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("expected ellipsis: %q", got)
	}
}

func TestMemoryStatusDisabled(t *testing.T) {
	rt := &Runtime{memory: DefaultMemoryConfig()}
	s := rt.MemoryStatusLine()
	if !strings.Contains(s, "disabled") {
		t.Fatalf("status=%q", s)
	}
}

func TestAttachMemorySystemNote(t *testing.T) {
	rtr, err := router.New(router.DefaultModels(), router.DefaultModelName)
	if err != nil {
		t.Fatal(err)
	}
	rt, err := New(Config{Workspace: t.TempDir(), SubagentsEnabled: false}, rtr, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	rt.AttachMemory(MemoryConfig{
		Enabled:    true,
		Server:     "memory",
		Tenant:     "acme",
		AutoRecall: true,
	})
	msgs := rt.Messages()
	if len(msgs) == 0 || !strings.Contains(msgs[0].Content, "<memory>") {
		t.Fatalf("expected memory system note: %+v", msgs)
	}
}

func TestMemoryRecallRequiresMCP(t *testing.T) {
	rt := &Runtime{memory: MemoryConfig{Enabled: true, Server: "memory"}}
	_, err := rt.MemoryRecall(context.Background(), "q")
	if err == nil {
		t.Fatal("expected error without mcp")
	}
}

func TestMaybeInjectMemoryRecall_NoClient(t *testing.T) {
	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, AutoRecall: true, Server: "memory"},
		mcp:    mcp.NewManagerEmpty(nil),
	}
	var events []Event
	rt.maybeInjectMemoryRecall(context.Background(), "hello", func(e Event) {
		events = append(events, e)
	})
	if len(events) != 0 {
		t.Fatalf("expected fail-open no events, got %v", events)
	}
}

func TestDefaultMemoryConfig(t *testing.T) {
	d := DefaultMemoryConfig()
	if d.Enabled || d.Server != "memory" || !d.AutoRecall || d.AutoIngest {
		t.Fatalf("%+v", d)
	}
}
