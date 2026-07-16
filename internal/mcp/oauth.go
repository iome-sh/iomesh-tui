package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/security"
)

// OAuthConfig configures bearer acquisition for streamable HTTP MCP.
// Prefer AccessTokenEnv for static tokens; use TokenURL + client credentials for M2M.
type OAuthConfig struct {
	// TokenURL is the OAuth2 token endpoint (client_credentials).
	TokenURL string `toml:"token_url" json:"token_url"`
	// ClientID for client_credentials grant.
	ClientID string `toml:"client_id" json:"client_id"`
	// ClientSecretEnv names the env var holding the client secret (never inline secrets).
	ClientSecretEnv string `toml:"client_secret_env" json:"client_secret_env"`
	// Scopes space-separated or multi.
	Scopes []string `toml:"scopes" json:"scopes"`
	// AccessTokenEnv: if set, use that env as Bearer (skips token endpoint).
	AccessTokenEnv string `toml:"access_token_env" json:"access_token_env"`
	// AllowLoopback permits http://127.0.0.1 token URLs.
	AllowLoopback *bool `toml:"allow_loopback" json:"allow_loopback"`
}

// ResolveBearer returns a bearer token string (without "Bearer " prefix).
// Order: AccessTokenEnv / ServerConfig.AccessTokenEnv → client_credentials → empty.
func (c *Client) ResolveBearer(ctx context.Context) (string, error) {
	if c == nil {
		return "", nil
	}
	// Static token from server-level env shortcut.
	if env := c.cfg.AccessTokenEnv; env != "" {
		if t := strings.TrimSpace(os.Getenv(env)); t != "" {
			return t, nil
		}
	}
	oa := c.cfg.OAuth
	if oa == nil {
		return "", nil
	}
	if env := oa.AccessTokenEnv; env != "" {
		if t := strings.TrimSpace(os.Getenv(env)); t != "" {
			return t, nil
		}
	}
	if oa.TokenURL == "" || oa.ClientID == "" {
		return "", nil
	}
	return c.fetchClientCredentials(ctx, oa)
}

func (c *Client) fetchClientCredentials(ctx context.Context, oa *OAuthConfig) (string, error) {
	c.oauthMu.Lock()
	defer c.oauthMu.Unlock()
	if c.oauthTok != "" && time.Now().Before(c.oauthExp.Add(-30*time.Second)) {
		return c.oauthTok, nil
	}
	allowLoop := true
	if oa.AllowLoopback != nil {
		allowLoop = *oa.AllowLoopback
	}
	if err := security.ValidateHTTPURL(oa.TokenURL, allowLoop); err != nil {
		return "", fmt.Errorf("mcp oauth token_url: %w", err)
	}
	secretEnv := oa.ClientSecretEnv
	if secretEnv == "" {
		secretEnv = "MCP_CLIENT_SECRET"
	}
	secret := os.Getenv(secretEnv)
	if secret == "" {
		return "", fmt.Errorf("mcp oauth: empty secret env %s", secretEnv)
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", oa.ClientID)
	form.Set("client_secret", secret)
	if len(oa.Scopes) > 0 {
		form.Set("scope", strings.Join(oa.Scopes, " "))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, oa.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	hc := c.httpClient
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("mcp oauth token: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("mcp oauth token: http %d: %s", resp.StatusCode, security.Redact(string(body)))
	}
	var tr struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", fmt.Errorf("mcp oauth decode: %w", err)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("mcp oauth: empty access_token")
	}
	c.oauthTok = tr.AccessToken
	if tr.ExpiresIn > 0 {
		c.oauthExp = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	} else {
		c.oauthExp = time.Now().Add(55 * time.Minute)
	}
	return c.oauthTok, nil
}

// ApplyAuthHeaders sets Authorization if a bearer can be resolved.
// Does not overwrite an existing Authorization header from cfg.Headers.
func (c *Client) ApplyAuthHeaders(ctx context.Context, h http.Header) {
	if h.Get("Authorization") != "" {
		return
	}
	tok, err := c.ResolveBearer(ctx)
	if err != nil {
		if c.logger != nil {
			c.logger.Debug("mcp oauth", "err", err)
		}
		return
	}
	if tok != "" {
		h.Set("Authorization", "Bearer "+tok)
	}
}

// PKCEChallenge is a minimal S256 helper for future auth-code flows (exported for tests/tools).
func PKCEChallenge(verifier string) (challenge string, method string) {
	// Use plain method only as documented fallback; real S256 needs crypto/sha256 base64url.
	// Prefer S256:
	return pkceS256(verifier), "S256"
}

func pkceS256(verifier string) string {
	// deferred to avoid import cycle noise — implemented in oauth_pkce.go style inline:
	return pkceS256Impl(verifier)
}
