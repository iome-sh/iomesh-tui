package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/iomesh"
)

func TestMemoryRecallCache_HitMissAndTTL(t *testing.T) {
	c := newMemoryRecallCache(50) // 50ms TTL for fast expiry
	if c == nil {
		t.Fatal("expected cache")
	}
	key := memoryRecallCacheKey{Tenant: "t1", Session: "s1", Query: "q", Limit: 8}
	if _, _, ok := c.get(key); ok {
		t.Fatal("expected miss on empty cache")
	}
	hits := []iomesh.MemoryHit{{Summary: "cached hit", Score: 0.9}}
	c.put(key, hits, 42)
	got, lat, ok := c.get(key)
	if !ok {
		t.Fatal("expected hit")
	}
	if lat != 42 {
		t.Fatalf("latency=%d", lat)
	}
	if len(got) != 1 || got[0].Summary != "cached hit" {
		t.Fatalf("hits=%+v", got)
	}
	// Mutation isolation: caller must not affect store.
	got[0].Summary = "mutated"
	got2, _, ok := c.get(key)
	if !ok || got2[0].Summary != "cached hit" {
		t.Fatalf("store mutated: %+v", got2)
	}
	time.Sleep(60 * time.Millisecond)
	if _, _, ok := c.get(key); ok {
		t.Fatal("expected miss after TTL")
	}
}

func TestMemoryRecallCache_KeyIsolation(t *testing.T) {
	c := newMemoryRecallCache(5000)
	base := memoryRecallCacheKey{Tenant: "t1", Session: "s1", Query: "q", Limit: 8}
	c.put(base, []iomesh.MemoryHit{{Summary: "base"}}, 1)

	variants := []memoryRecallCacheKey{
		{Tenant: "t2", Session: "s1", Query: "q", Limit: 8},
		{Tenant: "t1", Session: "s2", Query: "q", Limit: 8},
		{Tenant: "t1", Session: "s1", Query: "other", Limit: 8},
		{Tenant: "t1", Session: "s1", Query: "q", Limit: 4},
		{Tenant: "t1", Session: "s1", Query: "q", Limit: 8, Since: "2026-01-01"},
		{Tenant: "t1", Session: "s1", Query: "q", Limit: 8, Until: "2026-12-31"},
	}
	for _, k := range variants {
		if _, _, ok := c.get(k); ok {
			t.Fatalf("expected miss for key isolation variant %+v", k)
		}
	}
	// Same key hits.
	if _, _, ok := c.get(base); !ok {
		t.Fatal("expected hit for base key")
	}
}

func TestMemoryRecallCache_DisabledWhenTTLZero(t *testing.T) {
	if c := newMemoryRecallCache(0); c != nil {
		t.Fatal("ttl 0 must disable cache")
	}
	if c := newMemoryRecallCache(-1); c != nil {
		t.Fatal("negative ttl must disable cache")
	}
	// nil-safe
	var c *memoryRecallCache
	c.put(memoryRecallCacheKey{Query: "q"}, []iomesh.MemoryHit{{Summary: "x"}}, 1)
	if _, _, ok := c.get(memoryRecallCacheKey{Query: "q"}); ok {
		t.Fatal("nil cache get must miss")
	}
}

func TestMemoryRecall_CacheHitMissHTTP(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"memories": []map[string]any{
				{"summary": "from network", "score": 0.8},
			},
		})
	}))
	defer srv.Close()

	mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: srv.URL, Tenant: "dept.x"}, nil)
	rt := &Runtime{
		mesh: mesh,
		memory: MemoryConfig{
			Enabled: true, Tenant: "dept.x", Server: "memory",
			Limit: 5, SessionID: "sess-a",
			RecallCacheTTLMS: 3000,
			MaxSnippetBytes:  6000,
		},
	}
	rt.initMemoryRecallCache()

	out1, err := rt.MemoryRecall(context.Background(), "same query")
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if !strings.Contains(out1, "from network") {
		t.Fatalf("out1=%q", out1)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls after first=%d", calls.Load())
	}
	if rt.LastMemoryRetrieveMS() < 0 {
		t.Fatalf("last ms=%d", rt.LastMemoryRetrieveMS())
	}

	// Second identical recall must reuse cache (no extra HTTP).
	out2, err := rt.MemoryRecall(context.Background(), "same query")
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if out2 != out1 {
		t.Fatalf("out2=%q out1=%q", out2, out1)
	}
	if calls.Load() != 1 {
		t.Fatalf("cache miss: calls=%d want 1", calls.Load())
	}
	if !rt.LastMemoryRetrieveCacheHit() {
		t.Fatal("expected last path to be cache hit")
	}

	// Different tenant key → network.
	rt.memory.Tenant = "dept.other"
	_, err = rt.MemoryRecall(context.Background(), "same query")
	if err != nil {
		t.Fatalf("tenant isol: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("tenant isol calls=%d want 2", calls.Load())
	}
	if rt.LastMemoryRetrieveCacheHit() {
		t.Fatal("cross-tenant must not be cache hit")
	}
}

func TestMemoryRecall_CacheDisabled(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"memories": []map[string]any{{"summary": "n"}},
		})
	}))
	defer srv.Close()

	mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: srv.URL, Tenant: "t"}, nil)
	rt := &Runtime{
		mesh: mesh,
		memory: MemoryConfig{
			Enabled: true, Tenant: "t", Server: "memory",
			Limit: 3, RecallCacheTTLMS: 0, // disabled
		},
	}
	rt.initMemoryRecallCache()

	for i := 0; i < 2; i++ {
		if _, err := rt.MemoryRecall(context.Background(), "q"); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("disabled cache must hit network twice, calls=%d", calls.Load())
	}
}

func TestFormatMemoryHits_EarlyStop(t *testing.T) {
	hits := []iomesh.MemoryHit{
		{Summary: strings.Repeat("a", 40)},
		{Summary: strings.Repeat("b", 40)},
		{Summary: strings.Repeat("c", 40)},
	}
	// Max bytes small enough that only first hit fits fully.
	got := formatMemoryHits(hits, 50)
	if !strings.Contains(got, "aaa") {
		t.Fatalf("expected first hit: %q", got)
	}
	if strings.Contains(got, "bbb") || strings.Contains(got, "ccc") {
		t.Fatalf("early-stop should omit later hits: %q", got)
	}
	// Unlimited formats all.
	all := formatMemoryHits(hits, 0)
	if !strings.Contains(all, "bbb") || !strings.Contains(all, "ccc") {
		t.Fatalf("unlimited: %q", all)
	}
}

func TestMaybeInjectMemoryRecall_EmitsLatency(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"memories": []map[string]any{{"summary": "lat test"}},
		})
	}))
	defer srv.Close()

	mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: srv.URL, Tenant: "acme"}, nil)
	rt := &Runtime{
		mesh: mesh,
		memory: MemoryConfig{
			Enabled: true, AutoRecall: true, Tenant: "acme",
			Server: "memory", MaxSnippetBytes: 6000,
			RecallCacheTTLMS: 3000,
		},
	}
	rt.initMemoryRecallCache()

	var events []Event
	rt.maybeInjectMemoryRecall(context.Background(), "hello latency", func(e Event) {
		events = append(events, e)
	})
	if len(events) != 1 || events[0].Type != EventMemoryRecall {
		t.Fatalf("events=%v", events)
	}
	if events[0].Duration < 0 {
		t.Fatalf("duration must be set: %v", events[0].Duration)
	}
	if !strings.Contains(events[0].Text, "ms") {
		t.Fatalf("text should include latency: %q", events[0].Text)
	}
	if rt.LastMemoryRetrieveMS() < 0 {
		t.Fatalf("LastMemoryRetrieveMS=%d", rt.LastMemoryRetrieveMS())
	}
	// Cache hit path also reports latency field (0 for cache).
	events = nil
	rt.maybeInjectMemoryRecall(context.Background(), "hello latency", func(e Event) {
		events = append(events, e)
	})
	if len(events) != 1 {
		t.Fatalf("cache inject events=%v", events)
	}
	if !strings.Contains(events[0].Text, "cache") {
		t.Fatalf("expected cache marker: %q", events[0].Text)
	}
}

func TestDefaultMemoryConfig_RecallCacheTTL(t *testing.T) {
	d := DefaultMemoryConfig()
	if d.RecallCacheTTLMS != DefaultRecallCacheTTLMS {
		t.Fatalf("default TTL=%d want %d", d.RecallCacheTTLMS, DefaultRecallCacheTTLMS)
	}
	if d.DualWrite {
		t.Fatal("dual_write must remain OFF")
	}
}
