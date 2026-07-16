package subagent

import (
	"testing"
	"time"
)

func TestRegistry_ListAndCount(t *testing.T) {
	r := NewRegistry()
	id1 := r.AllocID()
	id2 := r.AllocID()
	r.Put(&Record{ID: id1, Status: StatusRunning, StartedAt: time.Now().Add(-time.Second)})
	r.Put(&Record{ID: id2, Status: StatusCompleted, StartedAt: time.Now()})
	list := r.List()
	if len(list) != 2 {
		t.Fatalf("len=%d", len(list))
	}
	// Newest first
	if list[0].ID != id2 {
		t.Fatalf("order=%v", list)
	}
	if r.CountRunning() != 1 {
		t.Fatalf("running=%d", r.CountRunning())
	}
	if err := r.Update(id1, func(rec *Record) { rec.Status = StatusCompleted }); err != nil {
		t.Fatal(err)
	}
	if r.CountRunning() != 0 {
		t.Fatal()
	}
	if _, ok := r.Get("missing"); ok {
		t.Fatal()
	}
	if err := r.Update("missing", func(*Record) {}); err == nil {
		t.Fatal()
	}
}
