package config

import "github.com/iome-sh/iomesh-tui/internal/mcp"

// BuildMCPServerConfig maps one [[mcp.servers]] entry to mcp.ServerConfig.
// When inject_iomesh_context is enabled (global [mcp] or per-server), merges
// non-empty X-IOMesh-Tenant/Org/Workspace/Department from IOMeshMCPContext without
// overwriting explicit headers (s1267 residual-honest opt-in).
//
// Residual honesty:
//   - inject ≠ install APPLY / Connected / INSTALL_STORE green
//   - inject ≠ dual-auth install list shipped
//   - empty values not sent (no invent tenant)
//   - headers apply on HTTP URL dial only; stdio ignores Headers
func (c *Config) BuildMCPServerConfig(s MCPServerTOML) mcp.ServerConfig {
	var headers map[string]string
	if len(s.Headers) > 0 {
		headers = make(map[string]string, len(s.Headers))
		for k, v := range s.Headers {
			headers[k] = v
		}
	}
	sc := mcp.ServerConfig{
		Name: s.Name, Command: s.Command, Args: s.Args, Env: s.Env,
		URL: s.URL, Headers: headers, AllowLoopback: s.AllowLoopback,
		Enabled: s.Enabled, Mutating: s.Mutating,
		StartupTimeoutSec: s.StartupTimeoutSec, ToolTimeoutSec: s.ToolTimeoutSec,
		AccessTokenEnv: s.OAuthTokenEnv,
	}
	if s.OAuth != nil {
		sc.OAuth = &mcp.OAuthConfig{
			TokenURL:        s.OAuth.TokenURL,
			ClientID:        s.OAuth.ClientID,
			ClientSecretEnv: s.OAuth.ClientSecretEnv,
			Scopes:          s.OAuth.Scopes,
			AccessTokenEnv:  s.OAuth.AccessTokenEnv,
			AllowLoopback:   s.OAuth.AllowLoopback,
		}
	}
	if c != nil && c.MCP.WantsInjectIOMeshContext(s) {
		tenant, org, workspace, department := c.IOMeshMCPContext()
		sc.Headers = mcp.ApplyIOMeshContextHeaders(sc.Headers, tenant, org, workspace, department)
	}
	return sc
}

// BuildMCPServerConfigs builds ServerConfig for all configured MCP servers (s1267 inject applied).
func (c *Config) BuildMCPServerConfigs() []mcp.ServerConfig {
	if c == nil || len(c.MCP.Servers) == 0 {
		return nil
	}
	out := make([]mcp.ServerConfig, 0, len(c.MCP.Servers))
	for _, s := range c.MCP.Servers {
		out = append(out, c.BuildMCPServerConfig(s))
	}
	return out
}
