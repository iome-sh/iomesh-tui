package iomesh

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestWaitReady_SucceedsAfterNFailures(t *testing.T) {
	var readyHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ready", "/readyz":
			n := readyHits.Add(1)
			if n < 3 {
				w.WriteHeader(503)
				return
			}
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ready"))
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.WaitReady(ctx, WaitReadyOptions{Interval: 10 * time.Millisecond}); err != nil {
		t.Fatalf("WaitReady: %v", err)
	}
	if readyHits.Load() < 3 {
		t.Fatalf("expected ≥3 ready hits, got %d", readyHits.Load())
	}
}

func TestWaitReady_TimeoutOnAlwaysFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	err := c.WaitReady(ctx, WaitReadyOptions{Interval: 15 * time.Millisecond})
	if err == nil {
		t.Fatal("expected error on timeout")
	}
	if !strings.Contains(err.Error(), "wait ready") {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}

func TestWaitReady_Unreachable(t *testing.T) {
	// Closed listener: connection refused.
	c := New(Config{Enabled: true, Endpoint: "http://127.0.0.1:1"}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := c.WaitReady(ctx, WaitReadyOptions{Interval: 20 * time.Millisecond})
	if err == nil {
		t.Fatal("expected error for unreachable endpoint")
	}
}

func TestWaitReady_DisabledNil(t *testing.T) {
	ctx := context.Background()
	var nilClient *Client
	if err := nilClient.WaitReady(ctx, WaitReadyOptions{}); err != nil {
		t.Fatalf("nil client: %v", err)
	}
	c := New(Config{Enabled: false, Endpoint: ""}, nil)
	if err := c.WaitReady(ctx, WaitReadyOptions{}); err != nil {
		t.Fatalf("disabled: %v", err)
	}
}

func TestWaitReady_RequireHealth(t *testing.T) {
	var healthHits, readyHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			n := healthHits.Add(1)
			if n < 2 {
				w.WriteHeader(503)
				return
			}
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		case "/ready", "/readyz":
			readyHits.Add(1)
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ready"))
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(Config{Enabled: true, Endpoint: srv.URL}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.WaitReady(ctx, WaitReadyOptions{
		Interval:      10 * time.Millisecond,
		RequireHealth: true,
	}); err != nil {
		t.Fatalf("WaitReady: %v", err)
	}
	if healthHits.Load() < 2 {
		t.Fatalf("expected health retries, got %d", healthHits.Load())
	}
	if readyHits.Load() < 1 {
		t.Fatal("expected ready after health ok")
	}
}
