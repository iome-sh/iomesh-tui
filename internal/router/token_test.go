package router

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestResolveVertexAccessToken_EnvOverride(t *testing.T) {
	t.Setenv("VERTEX_API_KEY", "env-tok-1")
	t.Setenv("GOOGLE_OAUTH_ACCESS_TOKEN", "")
	InvalidateVertexAccessToken()
	tok, err := ResolveVertexAccessToken(context.Background())
	if err != nil || tok != "env-tok-1" {
		t.Fatalf("got %q err=%v", tok, err)
	}
}

func TestResolveVertexAccessToken_Cache(t *testing.T) {
	t.Setenv("VERTEX_API_KEY", "")
	t.Setenv("GOOGLE_OAUTH_ACCESS_TOKEN", "")
	t.Setenv("VERTEX_ADC", "") // enable helper
	InvalidateVertexAccessToken()

	calls := 0
	prev := gcloudTokenRunner
	gcloudTokenRunner = func(ctx context.Context) (string, error) {
		calls++
		return "cached-tok", nil
	}
	t.Cleanup(func() { gcloudTokenRunner = prev })

	tok1, err := ResolveVertexAccessToken(context.Background())
	if err != nil || tok1 != "cached-tok" {
		t.Fatalf("first: %q %v", tok1, err)
	}
	tok2, err := ResolveVertexAccessToken(context.Background())
	if err != nil || tok2 != "cached-tok" {
		t.Fatalf("second: %q %v", tok2, err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 gcloud call, got %d", calls)
	}
}

func TestResolveVertexAccessToken_Invalidate(t *testing.T) {
	t.Setenv("VERTEX_API_KEY", "")
	t.Setenv("GOOGLE_OAUTH_ACCESS_TOKEN", "")
	InvalidateVertexAccessToken()

	n := 0
	prev := gcloudTokenRunner
	gcloudTokenRunner = func(ctx context.Context) (string, error) {
		n++
		return "tok-" + string(rune('A'+n-1)), nil
	}
	t.Cleanup(func() { gcloudTokenRunner = prev })

	_, _ = ResolveVertexAccessToken(context.Background())
	InvalidateVertexAccessToken()
	tok, err := ResolveVertexAccessToken(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if tok != "tok-B" {
		t.Fatalf("got %q", tok)
	}
	if n != 2 {
		t.Fatalf("calls=%d", n)
	}
}

func TestResolveVertexAccessToken_ADCDisabled(t *testing.T) {
	t.Setenv("VERTEX_API_KEY", "")
	t.Setenv("GOOGLE_OAUTH_ACCESS_TOKEN", "")
	t.Setenv("VERTEX_ADC", "0")
	InvalidateVertexAccessToken()
	prev := gcloudTokenRunner
	gcloudTokenRunner = func(ctx context.Context) (string, error) {
		return "", errors.New("should not run")
	}
	t.Cleanup(func() { gcloudTokenRunner = prev })

	_, err := ResolveVertexAccessToken(context.Background())
	if err == nil {
		t.Fatal("expected error when ADC disabled and no env token")
	}
}

func TestResolveVertexAccessToken_TTL(t *testing.T) {
	// Smoke that expiry field is set (unit-level: cache hit within TTL).
	t.Setenv("VERTEX_API_KEY", "")
	t.Setenv("GOOGLE_OAUTH_ACCESS_TOKEN", "")
	InvalidateVertexAccessToken()
	prev := gcloudTokenRunner
	gcloudTokenRunner = func(ctx context.Context) (string, error) { return "ttl-tok", nil }
	t.Cleanup(func() { gcloudTokenRunner = prev })

	_, _ = ResolveVertexAccessToken(context.Background())
	vertexTokMu.Lock()
	exp := vertexTokExp
	vertexTokMu.Unlock()
	if time.Until(exp) < 40*time.Minute {
		t.Fatalf("expected ~50m TTL, exp in %v", time.Until(exp))
	}
}
