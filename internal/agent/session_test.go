package agent

import (
	"testing"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/router"
	"github.com/iome-sh/iomesh-tui/internal/session"
	"github.com/iome-sh/iomesh-tui/internal/subagent"
)

func TestSaveLoadSession_WithSubagents(t *testing.T) {
	models := []router.ModelConfig{{
		Name: "m", BaseURL: "http://127.0.0.1:9", ModelID: "m", APIKey: "k",
		CostTier: 1, MaxContext: 1000, Capabilities: []string{"fast"}, Priority: 1,
	}}
	rtr, err := router.New(models, "m")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	rt, err := New(Config{Workspace: dir, SubagentsEnabled: true}, rtr, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Seed transcript + fake subagent record.
	rt.messages = append(rt.messages, router.Message{Role: "user", Content: "explore router"})
	rt.messages = append(rt.messages, router.Message{Role: "assistant", Content: "ok"})
	if rt.subagents != nil {
		rt.subagents.Registry().Put(&subagent.Record{
			ID: "sa-persist-1", Status: subagent.StatusCompleted, Summary: "found files",
			StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC(),
			Spec: subagent.Spec{Prompt: "p", SubagentType: subagent.TypeExplore},
		})
	}
	_ = rtr.SetOverride("m")

	st, err := session.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := rt.SaveSession(st, 0)
	if err != nil {
		t.Fatal(err)
	}
	if snap.ID == "" || len(snap.Subagents) != 1 {
		t.Fatalf("%+v", snap)
	}

	// New runtime, load session.
	rt2, err := New(Config{Workspace: dir, SubagentsEnabled: true}, rtr, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := st.Load(snap.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := rt2.LoadSession(loaded); err != nil {
		t.Fatal(err)
	}
	if rt2.SessionID() != snap.ID {
		t.Fatal(rt2.SessionID())
	}
	if len(rt2.Messages()) < 3 {
		t.Fatalf("msgs=%d", len(rt2.Messages()))
	}
	rec, ok := rt2.Subagents().Registry().Get("sa-persist-1")
	if !ok || rec.Summary != "found files" {
		t.Fatalf("%+v ok=%v", rec, ok)
	}
}

func TestExportRunningBecomesCancelled(t *testing.T) {
	reg := subagent.NewRegistry()
	reg.Put(&subagent.Record{ID: "sa-run", Status: subagent.StatusRunning, Spec: subagent.Spec{Prompt: "x"}})
	recs, _ := reg.Export()
	if recs[0].Status != subagent.StatusCancelled {
		t.Fatalf("%s", recs[0].Status)
	}
}
