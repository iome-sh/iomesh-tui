package honesty

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// SessionPalaceInput is the operator view used to choose a session palace.
// Only WorkspaceMemoryURL can select a host. The other fields are refusals:
// they are not a bind, and they are not Connected.
type SessionPalaceInput struct {
	// WorkspaceMemoryURL is the workspace's own MemoryURL.
	WorkspaceMemoryURL string
	// SharedMemoryURL is cfg.MemoryURL. It is never the workspace palace.
	SharedMemoryURL string
	// HostedPalaceEnabled is not a reason to bind.
	HostedPalaceEnabled bool
	// CatalogRow is a catalog listing. Not Connected and not a bind.
	CatalogRow bool
	// WorkspacePatched is a workspace PATCH. Not Connected and not a bind.
	WorkspacePatched bool
	// OpenGates is the OpenGates list. Empty is not Connected and not a bind.
	// A non-empty list is also not a host URL.
	OpenGates []string
}

// SelectSessionPalace returns the workspace MemoryURL when it is a real
// customer host. Otherwise it returns "" and DisplaySessionPalace prints
// palace=-. leftover_is_bind stays OPEN: this does not claim a QA sitting
// or that a laptop path was retrieved from a dedicated disk.
//
// A URL is not selected when it is empty, the laptop default ~/.iomesh/palace,
// the same string as cfg.MemoryURL, or the synthetic one-label placeholder
// aion-mem-<slug>.internal. HostedPalaceEnabled, a catalog row, a workspace
// PATCH, and OpenGates are ignored.
func SelectSessionPalace(in SessionPalaceInput) string {
	// Named so callers can pass them. None of these choose a host.
	_ = in.HostedPalaceEnabled
	_ = in.CatalogRow
	_ = in.WorkspacePatched
	_ = in.OpenGates

	raw := strings.TrimSpace(in.WorkspaceMemoryURL)
	if raw == "" || isLaptopPalaceDefault(raw) {
		return ""
	}
	if shared := strings.TrimSpace(in.SharedMemoryURL); shared != "" && sameMemoryURL(raw, shared) {
		return ""
	}
	if !realCustomerMemoryHost(raw) {
		return ""
	}
	return raw
}

// DisplaySessionPalace is palace=<url> when SelectSessionPalace accepts a
// workspace MemoryURL. Otherwise it is the exact token palace=-.
func DisplaySessionPalace(in SessionPalaceInput) string {
	if u := SelectSessionPalace(in); u != "" {
		return "palace=" + u
	}
	return UnboundPalaceToken
}

func realCustomerMemoryHost(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "https" && scheme != "http" {
		return false
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if host == "" || syntheticAionMemHost(host) {
		return false
	}
	return true
}

// syntheticAionMemHost is the one-label placeholder aion-mem-<slug>.internal.
// A second label before .internal is not that placeholder.
func syntheticAionMemHost(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	parts := strings.Split(host, ".")
	if len(parts) != 2 || parts[1] != "internal" {
		return false
	}
	return strings.HasPrefix(parts[0], "aion-mem-")
}

func sameMemoryURL(a, b string) bool {
	ka, oka := memoryURLKey(a)
	kb, okb := memoryURLKey(b)
	if oka && okb {
		return ka == kb
	}
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func memoryURLKey(raw string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Hostname() == "" {
		return "", false
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "https" && scheme != "http" {
		return "", false
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	port := u.Port()
	if (scheme == "https" && port == "443") || (scheme == "http" && port == "80") {
		port = ""
	}
	hostport := host
	if port != "" {
		hostport = netHostPort(host, port)
	}
	path := strings.TrimRight(u.EscapedPath(), "/")
	key := scheme + "://" + hostport + path
	if u.RawQuery != "" {
		key += "?" + u.RawQuery
	}
	return key, true
}

func netHostPort(host, port string) string {
	if strings.Contains(host, ":") {
		return "[" + host + "]:" + port
	}
	return host + ":" + port
}

func isLaptopPalaceDefault(raw string) bool {
	s := strings.TrimSpace(raw)
	if s == "" {
		return false
	}
	if strings.Contains(s, "://") {
		u, err := url.Parse(s)
		if err != nil || !strings.EqualFold(u.Scheme, "file") {
			return false
		}
		s = u.Path
		if s == "" {
			s = u.Opaque
		}
	}
	s = strings.TrimRight(s, "/")
	if s == "~/.iomesh/palace" || s == ".iomesh/palace" || strings.HasSuffix(s, "/.iomesh/palace") {
		return true
	}
	if strings.HasPrefix(s, "~/") {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			expanded := strings.TrimRight(filepath.Join(home, s[2:]), "/")
			if expanded == filepath.Join(home, ".iomesh", "palace") || strings.HasSuffix(expanded, "/.iomesh/palace") {
				return true
			}
		}
	}
	cleaned := filepath.Clean(s)
	sep := string(filepath.Separator)
	return cleaned == ".iomesh"+sep+"palace" || strings.HasSuffix(cleaned, sep+".iomesh"+sep+"palace")
}
