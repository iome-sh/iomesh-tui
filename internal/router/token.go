package router

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Vertex token helper — short-lived OAuth access tokens for Vertex OpenAI-compat.
//
// Resolution order (see ResolveVertexAccessToken):
//  1. VERTEX_API_KEY / GOOGLE_OAUTH_ACCESS_TOKEN env (static override)
//  2. In-process cache from a prior gcloud/ADC fetch
//  3. `gcloud auth print-access-token` when VERTEX_ADC is not "0"/"false"/"off"
//
// Never log token values. Cache TTL defaults to 50m (tokens are typically ~1h).

const defaultVertexTokenTTL = 50 * time.Minute

var (
	vertexTokMu   sync.Mutex
	vertexTok     string
	vertexTokExp  time.Time
	// gcloudTokenRunner is swappable for tests.
	gcloudTokenRunner = runGcloudPrintAccessToken
)

// InvalidateVertexAccessToken clears the cached gcloud/ADC token (e.g. after HTTP 401).
func InvalidateVertexAccessToken() {
	vertexTokMu.Lock()
	defer vertexTokMu.Unlock()
	vertexTok = ""
	vertexTokExp = time.Time{}
}

// ResolveVertexAccessToken returns a Bearer token for Vertex OpenAI-compat calls.
func ResolveVertexAccessToken(ctx context.Context) (string, error) {
	if v := strings.TrimSpace(getenv("VERTEX_API_KEY")); v != "" {
		return v, nil
	}
	if v := strings.TrimSpace(getenv("GOOGLE_OAUTH_ACCESS_TOKEN")); v != "" {
		return v, nil
	}
	if vertexADCDisabled() {
		return "", fmt.Errorf("vertex auth: set VERTEX_API_KEY or GOOGLE_OAUTH_ACCESS_TOKEN (or enable VERTEX_ADC for gcloud print-access-token)")
	}

	vertexTokMu.Lock()
	defer vertexTokMu.Unlock()
	if vertexTok != "" && time.Now().Before(vertexTokExp) {
		return vertexTok, nil
	}

	tok, err := gcloudTokenRunner(ctx)
	if err != nil {
		return "", err
	}
	vertexTok = tok
	vertexTokExp = time.Now().Add(defaultVertexTokenTTL)
	return vertexTok, nil
}

func vertexADCDisabled() bool {
	v := strings.ToLower(strings.TrimSpace(getenv("VERTEX_ADC")))
	switch v {
	case "0", "false", "off", "no":
		return true
	default:
		// Default: ADC/gcloud helper enabled when env token unset.
		return false
	}
}

func runGcloudPrintAccessToken(ctx context.Context) (string, error) {
	// Short timeout — founder laptop / CI with gcloud installed.
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, "gcloud", "auth", "print-access-token")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("vertex auth: gcloud auth print-access-token failed: %s (set VERTEX_API_KEY or run gcloud auth login)", msg)
	}
	tok := strings.TrimSpace(stdout.String())
	if tok == "" {
		return "", fmt.Errorf("vertex auth: gcloud returned empty access token")
	}
	// Access tokens are not JWT-only; still reject obvious multi-line garbage.
	if strings.ContainsAny(tok, "\n\r") {
		return "", fmt.Errorf("vertex auth: unexpected multi-line gcloud token output")
	}
	return tok, nil
}
