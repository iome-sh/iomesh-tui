// Package security provides secret redaction, environment scrubbing, URL
// validation, and shell command policy for the agent harness.
//
// Design goals for an open-source coding agent:
//   - Never log or return raw API keys
//   - Do not inherit provider secrets into shell tool children
//   - Restrict outbound HTTP schemes (SSRF posture for mesh/LLM clients)
//   - Fail closed on path / shell policy violations
package security

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
)

// SecretEnvNames are environment variable names scrubbed from shell tool env
// and never safe to log.
var SecretEnvNames = []string{
	"DEEPSEEK_API_KEY",
	"XAI_API_KEY",
	"OPENAI_API_KEY",
	"ANTHROPIC_API_KEY",
	"ANTHROPIC_AUTH_TOKEN",
	"IOMESH_API_KEY",
	"GROK_CODE_XAI_API_KEY",
	"AWS_SECRET_ACCESS_KEY",
	"AWS_SESSION_TOKEN",
	"GITHUB_TOKEN",
	"GH_TOKEN",
	"NPM_TOKEN",
	"HF_TOKEN",
	"HUGGING_FACE_HUB_TOKEN",
}

// secretEnvSet is an uppercase lookup for scrubbing.
var secretEnvSet = func() map[string]struct{} {
	m := make(map[string]struct{}, len(SecretEnvNames))
	for _, n := range SecretEnvNames {
		m[strings.ToUpper(n)] = struct{}{}
	}
	return m
}()

var (
	// bearerRE redacts Authorization bearer tokens in free text.
	bearerRE = regexp.MustCompile(`(?i)(Bearer\s+)[A-Za-z0-9\-._~+/]+=*`)
	// apiKeyAssignRE redacts key=value style secrets.
	apiKeyAssignRE = regexp.MustCompile(`(?i)((?:api[_-]?key|token|secret|password|authorization)\s*[=:]\s*)([^\s"',}]+)`)
	// skLikeRE redacts common sk- / key- prefixes.
	skLikeRE = regexp.MustCompile(`\b(sk-[A-Za-z0-9_\-]{8,}|xai-[A-Za-z0-9_\-]{8,})\b`)
)

// Redact replaces credential-like substrings with a placeholder.
func Redact(s string) string {
	if s == "" {
		return s
	}
	s = bearerRE.ReplaceAllString(s, "${1}***")
	s = apiKeyAssignRE.ReplaceAllString(s, "${1}***")
	s = skLikeRE.ReplaceAllString(s, "***")
	return s
}

// IsSecretEnv reports whether name should be scrubbed from child processes / logs.
func IsSecretEnv(name string) bool {
	u := strings.ToUpper(name)
	if _, ok := secretEnvSet[u]; ok {
		return true
	}
	for _, suf := range []string{"_API_KEY", "_AUTH_TOKEN", "_SECRET", "_SECRET_KEY", "_ACCESS_TOKEN", "_PASSWORD"} {
		if strings.HasSuffix(u, suf) {
			return true
		}
	}
	return false
}

// ScrubEnv returns a copy of env suitable for shell tools: secrets removed.
// If env is nil, uses os.Environ().
func ScrubEnv(env []string) []string {
	if env == nil {
		env = os.Environ()
	}
	out := make([]string, 0, len(env))
	for _, e := range env {
		key, _, ok := strings.Cut(e, "=")
		if !ok {
			continue
		}
		if IsSecretEnv(key) {
			continue
		}
		out = append(out, e)
	}
	return out
}

// ValidateHTTPURL ensures raw is an absolute http(s) URL suitable for LLM/mesh
// clients. Rejects file:, gopher:, and other schemes. When allowLoopback is
// false, also rejects loopback/link-local hosts.
// Tests and local OpenAI-compatible servers pass allowLoopback=true.
func ValidateHTTPURL(raw string, allowLoopback bool) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("empty url")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported url scheme %q (want http or https)", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("url missing host")
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("url missing hostname")
	}
	if !allowLoopback && isLoopbackOrLinkLocal(host) {
		return fmt.Errorf("loopback/link-local host not allowed: %s", host)
	}
	return nil
}

func isLoopbackOrLinkLocal(host string) bool {
	h := strings.ToLower(host)
	if h == "localhost" || strings.HasSuffix(h, ".localhost") {
		return true
	}
	ip := net.ParseIP(h)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}
