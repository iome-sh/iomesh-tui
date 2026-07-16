package session

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/router"
	"github.com/iome-sh/iomesh-tui/internal/subagent"
)

func TestSaveLoadList(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	snap := &Snapshot{
		ID:        "ses-test-1",
		Workspace: dir,
		Messages: []router.Message{
			{Role: "system", Content: "sys"},
			{Role: "user", Content: "hello world from session"},
			{Role: "assistant", Content: "hi"},
		},
		Subagents: []subagent.Record{{
			ID: "sa-1", Status: subagent.StatusCompleted, Summary: "done",
			StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC(),
			Spec: subagent.Spec{Prompt: "p", SubagentType: subagent.TypeExplore},
		}},
		SubagentSeq: 3,
	}
	if err := st.Save(snap); err != nil {
		t.Fatal(err)
	}
	got, err := st.Load("ses-test-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title == "" || len(got.Messages) != 3 || got.SubagentSeq != 3 {
		t.Fatalf("%+v", got)
	}
	list, err := st.List()
	if err != nil || len(list) != 1 || list[0].ID != "ses-test-1" {
		t.Fatalf("%+v %v", list, err)
	}
	if list[0].Subagents != 1 {
		t.Fatalf("subagents=%d", list[0].Subagents)
	}
}

func TestLatestAndCompact(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	_ = st.Save(&Snapshot{ID: "ses-a", Messages: []router.Message{{Role: "user", Content: "old"}}})
	time.Sleep(5 * time.Millisecond)
	_ = st.Save(&Snapshot{ID: "ses-b", Messages: []router.Message{{Role: "user", Content: "new"}}})
	latest, err := st.Latest()
	if err != nil || latest == nil || latest.ID != "ses-b" {
		t.Fatalf("%+v %v", latest, err)
	}

	snap := &Snapshot{
		Messages: []router.Message{
			{Role: "system", Content: "sys"},
			{Role: "user", Content: "u1"},
			{Role: "assistant", Content: "a1"},
			{Role: "tool", Content: string(make([]byte, MaxStoredToolContent+100))},
			{Role: "user", Content: "u2"},
			{Role: "assistant", Content: "a2"},
		},
	}
	Compact(snap, 1)
	if len(snap.Messages) < 3 {
		t.Fatalf("len=%d", len(snap.Messages))
	}
	// First is system; tool payload truncated if still present.
	for _, m := range snap.Messages {
		if m.Role == "tool" && len(m.Content) > MaxStoredToolContent+20 {
			t.Fatal("tool not compacted")
		}
	}
}

func TestInvalidID(t *testing.T) {
	st, _ := Open(t.TempDir())
	if _, err := st.PathFor("../x"); err == nil {
		t.Fatal()
	}
	if _, err := st.PathFor("a/b"); err == nil {
		t.Fatal()
	}
}

func TestPathUnderSessions(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	p, err := st.PathFor("ses-ok")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(p) != st.Dir() {
		t.Fatalf("%s vs %s", p, st.Dir())
	}
}
