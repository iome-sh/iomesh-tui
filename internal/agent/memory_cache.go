package agent

import (
	"sync"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/iomesh"
)

// DefaultRecallCacheTTLMS is the default short-TTL for sync retrieve reuse (s1069).
// Client-side fail-open only — not product Memory GA; 0 disables.
const DefaultRecallCacheTTLMS = 3000

// memoryRecallCacheKey uniquely identifies a sync RetrieveMemory call.
// Tenant + session + query + limit are always part of the key (no silent
// wrong-tenant reuse). Since/Until reserved for temporal options (s1068 peer).
type memoryRecallCacheKey struct {
	Tenant  string
	Session string
	Query   string
	Limit   int
	Since   string
	Until   string
}

// memoryRecallCacheEntry holds hits from a successful sync retrieve.
type memoryRecallCacheEntry struct {
	Hits      []iomesh.MemoryHit
	StoredAt  time.Time
	LatencyMS int // original network retrieve latency
}

// memoryRecallCache is a short-TTL, process-local cache for sync RetrieveMemory.
// Fail-open: never required for correctness; disabled when ttl <= 0.
type memoryRecallCache struct {
	mu  sync.Mutex
	ttl time.Duration
	// Single-entry map is enough for rapid auto-recall + slash recall of the
	// same query; multi-entry keeps isolation under tenant/session churn.
	entries map[memoryRecallCacheKey]memoryRecallCacheEntry
}

func newMemoryRecallCache(ttlMS int) *memoryRecallCache {
	if ttlMS <= 0 {
		return nil
	}
	return &memoryRecallCache{
		ttl:     time.Duration(ttlMS) * time.Millisecond,
		entries: make(map[memoryRecallCacheKey]memoryRecallCacheEntry),
	}
}

// get returns a copy of cached hits when still within TTL.
func (c *memoryRecallCache) get(key memoryRecallCacheKey) (hits []iomesh.MemoryHit, latencyMS int, ok bool) {
	if c == nil {
		return nil, 0, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	ent, found := c.entries[key]
	if !found {
		return nil, 0, false
	}
	if time.Since(ent.StoredAt) > c.ttl {
		delete(c.entries, key)
		return nil, 0, false
	}
	return cloneMemoryHits(ent.Hits), ent.LatencyMS, true
}

// put stores hits (copied) under key. Fail-open: nil cache is a no-op.
func (c *memoryRecallCache) put(key memoryRecallCacheKey, hits []iomesh.MemoryHit, latencyMS int) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// Bound growth: drop expired entries opportunistically.
	now := time.Now()
	for k, ent := range c.entries {
		if now.Sub(ent.StoredAt) > c.ttl {
			delete(c.entries, k)
		}
	}
	c.entries[key] = memoryRecallCacheEntry{
		Hits:      cloneMemoryHits(hits),
		StoredAt:  now,
		LatencyMS: latencyMS,
	}
}

func cloneMemoryHits(in []iomesh.MemoryHit) []iomesh.MemoryHit {
	if len(in) == 0 {
		if in == nil {
			return nil
		}
		return []iomesh.MemoryHit{}
	}
	out := make([]iomesh.MemoryHit, len(in))
	copy(out, in)
	return out
}
