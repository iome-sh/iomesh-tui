package config

import (
	"net/url"
	"strings"
)

// InferredBroker is a residual-honest mapping from portal MCP → stream broker.
// apiv1.iome.sh/v7/mcp is catalog only; streams live on hooks.iome.sh.
// Infer ≠ Connected / HITL / INSTALL_STORE.
type InferredBroker struct {
	Endpoint string
	Tenant   string
	Org      string
	FromURL  string
}

// InferHooksEndpoint maps a portal MCP URL host to the public broker host.
// Unknown hosts return empty (no invent).
func InferHooksEndpoint(portalMCPURL string) string {
	u, err := url.Parse(strings.TrimSpace(portalMCPURL))
	if err != nil || u.Hostname() == "" {
		return ""
	}
	switch strings.ToLower(u.Hostname()) {
	case "apiv1.iome.sh":
		return "https://hooks.iome.sh"
	case "apiv1.staging.iome.sh":
		return "https://hooks.staging.iome.sh"
	default:
		return ""
	}
}

// InferBrokerFromPortalMCP scans [[mcp.servers]] for a portal MCP URL and
// optional X-IOMesh-Tenant / X-IOMesh-Org headers.
func (c *Config) InferBrokerFromPortalMCP() InferredBroker {
	var out InferredBroker
	if c == nil {
		return out
	}
	for _, s := range c.MCP.Servers {
		ep := InferHooksEndpoint(s.URL)
		if ep == "" {
			continue
		}
		out.Endpoint = ep
		out.FromURL = strings.TrimSpace(s.URL)
		if s.Headers != nil {
			out.Tenant = firstHeader(s.Headers, "X-IOMesh-Tenant", "x-iomesh-tenant")
			out.Org = firstHeader(s.Headers, "X-IOMesh-Org", "x-iomesh-org")
		}
		return out
	}
	return out
}

func firstHeader(h map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(h[k]); v != "" {
			return v
		}
	}
	// TOML keys may be stored as written
	for k, v := range h {
		lk := strings.ToLower(strings.TrimSpace(k))
		for _, want := range keys {
			if lk == strings.ToLower(want) {
				return strings.TrimSpace(v)
			}
		}
	}
	return ""
}
