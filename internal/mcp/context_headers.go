package mcp

import "strings"

// IOMesh multi-tenant context header names (HTTP MCP only; stdio has no headers).
// s1267 residual-honest opt-in inject — not install APPLY green, not dual-auth ship.
const (
	HeaderIOMeshTenant    = "X-IOMesh-Tenant"
	HeaderIOMeshOrg       = "X-IOMesh-Org"
	HeaderIOMeshWorkspace = "X-IOMesh-Workspace"
)

// ApplyIOMeshContextHeaders merges multi-tenant context into headers for HTTP MCP.
//
// Residual honesty (s1267):
//   - only non-empty values are set (never invent tenant/org/workspace)
//   - never overwrites an existing explicit header for the same key (case-insensitive)
//   - inject ≠ install Connected / INSTALL_STORE green / dual-auth install list
//   - callers must opt in; default config leaves headers unchanged
//
// Returns a new map when headers is nil and at least one value is applied; otherwise
// mutates/returns the (possibly same) map for convenience at ServerConfig build time.
func ApplyIOMeshContextHeaders(headers map[string]string, tenant, org, workspace string) map[string]string {
	type kv struct{ k, v string }
	candidates := []kv{
		{HeaderIOMeshTenant, strings.TrimSpace(tenant)},
		{HeaderIOMeshOrg, strings.TrimSpace(org)},
		{HeaderIOMeshWorkspace, strings.TrimSpace(workspace)},
	}
	any := false
	for _, c := range candidates {
		if c.v != "" {
			any = true
			break
		}
	}
	if !any {
		return headers
	}
	if headers == nil {
		headers = map[string]string{}
	}
	for _, c := range candidates {
		if c.v == "" {
			continue
		}
		if headerKeyPresent(headers, c.k) {
			continue // never overwrite explicit server headers
		}
		headers[c.k] = c.v
	}
	return headers
}

// headerKeyPresent reports whether headers already has key (HTTP case-insensitive).
func headerKeyPresent(headers map[string]string, key string) bool {
	if _, ok := headers[key]; ok {
		return true
	}
	want := strings.ToLower(key)
	for k := range headers {
		if strings.ToLower(k) == want {
			return true
		}
	}
	return false
}
