package subagent

import (
	"fmt"
	"sync"
	"time"
)

// Registry stores subagent records for a parent session.
type Registry struct {
	mu   sync.RWMutex
	byID map[string]*Record
	seq  uint64
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{byID: make(map[string]*Record)}
}

// AllocID returns a new unique subagent id.
func (r *Registry) AllocID() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	return fmt.Sprintf("sa-%d-%d", time.Now().UnixMilli(), r.seq)
}

// Put inserts or replaces a record.
func (r *Registry) Put(rec *Record) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[rec.ID] = rec
}

// Get returns a copy of the record if present.
func (r *Registry) Get(id string) (Record, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.byID[id]
	if !ok {
		return Record{}, false
	}
	return *rec, true
}

// Update applies fn under lock to the live record.
func (r *Registry) Update(id string, fn func(*Record)) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.byID[id]
	if !ok {
		return fmt.Errorf("subagent %q not found", id)
	}
	fn(rec)
	return nil
}

// List returns all records (newest first by StartedAt).
func (r *Registry) List() []Record {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Record, 0, len(r.byID))
	for _, rec := range r.byID {
		out = append(out, *rec)
	}
	// Simple insertion order is fine for scaffold; sort by StartedAt desc.
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].StartedAt.After(out[i].StartedAt) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// CountRunning returns how many subagents are currently running.
func (r *Registry) CountRunning() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n := 0
	for _, rec := range r.byID {
		if rec.Status == StatusRunning || rec.Status == StatusPending {
			n++
		}
	}
	return n
}
