package agentplugins

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Supported MCP transport types (Agent Plugins 1.0.0).
const (
	TransportStdio          = "stdio"
	TransportStreamableHTTP = "streamable-http"
	TransportSSE            = "sse"
)

// MCPServerRef is a mapped MCP server entry (structs only — no process attach).
type MCPServerRef struct {
	Name    string
	Type    string
	Command string            // stdio: bare name or ./ relative (no placeholder expand)
	Args    []string          // may contain placeholders (unexpanded in map)
	Env     map[string]string // values may contain placeholders; keys never PLUGIN_ROOT/DATA
	Cwd     string            // optional; may contain placeholders
	URL     string            // streamable-http | sse
	Headers map[string]string
}

// MCPConfigResult is the parse result for root mcp.json.
type MCPConfigResult struct {
	Servers  []MCPServerRef
	Warnings []string
}

var mcpAllowedTopKeys = map[string]struct{}{
	"$schema":    {},
	"mcpServers": {},
}

// LoadMCPJSON reads and parses pluginRoot/mcp.json.
// Missing file is OK (returns empty servers). Invalid file disables MCP with warnings.
func LoadMCPJSON(pluginRoot string) (*MCPConfigResult, error) {
	absRoot, err := resolveRoot(pluginRoot)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(absRoot, "mcp.json")
	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &MCPConfigResult{}, nil
		}
		return nil, fmt.Errorf("agentplugins: stat mcp.json: %w", err)
	}
	if st.IsDir() {
		return &MCPConfigResult{
			Warnings: []string{"mcp.json is a directory; MCP component disabled"},
		}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("agentplugins: read mcp.json: %w", err)
	}
	return ParseMCPJSON(data, absRoot)
}

// ParseMCPJSON validates mcp.json bytes and maps valid server entries.
// Top-level failures disable MCP (empty servers + warnings); per-entry failures skip that entry.
func ParseMCPJSON(data []byte, pluginRoot string) (*MCPConfigResult, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return &MCPConfigResult{
			Warnings: []string{fmt.Sprintf("mcp.json invalid JSON: %v; MCP component disabled", err)},
		}, nil
	}

	res := &MCPConfigResult{}
	for k := range raw {
		if _, ok := mcpAllowedTopKeys[k]; !ok {
			res.Warnings = append(res.Warnings, fmt.Sprintf("mcp.json unknown top-level key %q; MCP component disabled", k))
			return res, nil
		}
	}

	// $schema required for valid mcp.json when file is present and parseable.
	rawSchema, ok := raw["$schema"]
	if !ok {
		res.Warnings = append(res.Warnings, "mcp.json missing $schema; MCP component disabled")
		return res, nil
	}
	var schema string
	if err := json.Unmarshal(rawSchema, &schema); err != nil || schema != MCPSchemaID {
		res.Warnings = append(res.Warnings, fmt.Sprintf("mcp.json unsupported or invalid $schema; MCP component disabled (want %s)", MCPSchemaID))
		return res, nil
	}

	rawServers, ok := raw["mcpServers"]
	if !ok {
		res.Warnings = append(res.Warnings, "mcp.json missing mcpServers; MCP component disabled")
		return res, nil
	}
	var servers map[string]json.RawMessage
	if err := json.Unmarshal(rawServers, &servers); err != nil {
		res.Warnings = append(res.Warnings, "mcp.json mcpServers must be an object; MCP component disabled")
		return res, nil
	}

	// Stable iteration order not required; sort by name for determinism.
	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	// simple insertion sort to avoid extra imports
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && names[j] < names[j-1]; j-- {
			names[j], names[j-1] = names[j-1], names[j]
		}
	}

	for _, name := range names {
		entry, warn, err := parseMCPServerEntry(name, servers[name], pluginRoot)
		if err != nil {
			res.Warnings = append(res.Warnings, err.Error())
			continue
		}
		if warn != "" {
			res.Warnings = append(res.Warnings, warn)
		}
		if entry != nil {
			res.Servers = append(res.Servers, *entry)
		}
	}
	return res, nil
}

func parseMCPServerEntry(name string, raw json.RawMessage, pluginRoot string) (*MCPServerRef, string, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, "", fmt.Errorf("mcp server %q: not an object; skipped", name)
	}
	rawType, ok := obj["type"]
	if !ok {
		return nil, "", fmt.Errorf("mcp server %q: missing type; skipped", name)
	}
	var typ string
	if err := json.Unmarshal(rawType, &typ); err != nil {
		return nil, "", fmt.Errorf("mcp server %q: type must be a string; skipped", name)
	}

	switch typ {
	case TransportStdio:
		return parseStdioServer(name, obj, pluginRoot)
	case TransportStreamableHTTP, TransportSSE:
		return parseHTTPServer(name, typ, obj)
	default:
		return nil, "", fmt.Errorf("mcp server %q: unsupported type %q; skipped", name, typ)
	}
}

func parseStdioServer(name string, obj map[string]json.RawMessage, pluginRoot string) (*MCPServerRef, string, error) {
	allowed := map[string]struct{}{
		"type": {}, "command": {}, "args": {}, "env": {}, "cwd": {},
	}
	for k := range obj {
		if _, ok := allowed[k]; !ok {
			return nil, "", fmt.Errorf("mcp server %q: unknown field %q; skipped", name, k)
		}
	}
	rawCmd, ok := obj["command"]
	if !ok {
		return nil, "", fmt.Errorf("mcp server %q: stdio requires command; skipped", name)
	}
	var command string
	if err := json.Unmarshal(rawCmd, &command); err != nil || strings.TrimSpace(command) == "" {
		return nil, "", fmt.Errorf("mcp server %q: command must be a non-empty string; skipped", name)
	}
	// command: bare name or ./ relative; no placeholder expansion.
	if strings.Contains(command, "${") {
		return nil, "", fmt.Errorf("mcp server %q: command must not contain placeholders; skipped", name)
	}
	if strings.HasPrefix(command, "./") {
		if _, err := Confine(command, pluginRoot); err != nil {
			return nil, "", fmt.Errorf("mcp server %q: command path invalid: %v; skipped", name, err)
		}
	} else if strings.Contains(command, "/") || strings.Contains(command, `\`) {
		// Not bare and not ./ — reject paths that do not start with ./
		return nil, "", fmt.Errorf("mcp server %q: command path must be bare or start with ./; skipped", name)
	}

	ref := &MCPServerRef{Name: name, Type: TransportStdio, Command: command}

	if v, ok := obj["args"]; ok {
		var args []string
		if err := json.Unmarshal(v, &args); err != nil {
			return nil, "", fmt.Errorf("mcp server %q: args must be string array; skipped", name)
		}
		ref.Args = args
	}
	if v, ok := obj["env"]; ok {
		var env map[string]string
		if err := json.Unmarshal(v, &env); err != nil {
			return nil, "", fmt.Errorf("mcp server %q: env must be object of strings; skipped", name)
		}
		for k := range env {
			if k == "PLUGIN_ROOT" || k == "PLUGIN_DATA" {
				return nil, "", fmt.Errorf("mcp server %q: env must not set %s; skipped", name, k)
			}
		}
		ref.Env = env
	}
	if v, ok := obj["cwd"]; ok {
		var cwd string
		if err := json.Unmarshal(v, &cwd); err != nil {
			return nil, "", fmt.Errorf("mcp server %q: cwd must be a string; skipped", name)
		}
		if err := validateCwdForm(cwd); err != nil {
			return nil, "", fmt.Errorf("mcp server %q: %v; skipped", name, err)
		}
		// Containment for ./ and ${PLUGIN_ROOT}/ forms (expand with synthetic PLUGIN_DATA for check).
		if err := checkCwdContainment(cwd, pluginRoot); err != nil {
			return nil, "", fmt.Errorf("mcp server %q: %v; skipped", name, err)
		}
		ref.Cwd = cwd
	}
	return ref, "", nil
}

func validateCwdForm(cwd string) error {
	if strings.HasPrefix(cwd, "./") {
		return nil
	}
	if cwd == "${PLUGIN_ROOT}" || strings.HasPrefix(cwd, "${PLUGIN_ROOT}/") {
		return nil
	}
	if cwd == "${PLUGIN_DATA}" || strings.HasPrefix(cwd, "${PLUGIN_DATA}/") {
		return nil
	}
	return fmt.Errorf("cwd %q must start with ./, ${PLUGIN_ROOT}, or ${PLUGIN_DATA}", cwd)
}

func checkCwdContainment(cwd, pluginRoot string) error {
	// PLUGIN_DATA-rooted: client-managed; form already validated — no package-root check.
	if cwd == "${PLUGIN_DATA}" || strings.HasPrefix(cwd, "${PLUGIN_DATA}/") {
		return nil
	}
	if strings.HasPrefix(cwd, "./") {
		_, err := Confine(cwd, pluginRoot)
		return err
	}
	// ${PLUGIN_ROOT} or ${PLUGIN_ROOT}/... — expand against resolved root so
	// macOS /var → /private/var (EvalSymlinks) still contains.
	absRoot, err := resolveRoot(pluginRoot)
	if err != nil {
		return err
	}
	expanded := ExpandPlaceholders(cwd, absRoot, "")
	cleaned := filepath.Clean(expanded)
	if !withinRoot(cleaned, absRoot) {
		return fmt.Errorf("cwd escapes plugin root")
	}
	return nil
}

func parseHTTPServer(name, typ string, obj map[string]json.RawMessage) (*MCPServerRef, string, error) {
	allowed := map[string]struct{}{
		"type": {}, "url": {}, "headers": {},
	}
	for k := range obj {
		if _, ok := allowed[k]; !ok {
			return nil, "", fmt.Errorf("mcp server %q: unknown field %q; skipped", name, k)
		}
	}
	rawURL, ok := obj["url"]
	if !ok {
		return nil, "", fmt.Errorf("mcp server %q: %s requires url; skipped", name, typ)
	}
	var u string
	if err := json.Unmarshal(rawURL, &u); err != nil || strings.TrimSpace(u) == "" {
		return nil, "", fmt.Errorf("mcp server %q: url must be a non-empty string; skipped", name)
	}
	if err := validateMCPURL(u); err != nil {
		return nil, "", fmt.Errorf("mcp server %q: %v; skipped", name, err)
	}
	ref := &MCPServerRef{Name: name, Type: typ, URL: u}
	if v, ok := obj["headers"]; ok {
		var headers map[string]string
		if err := json.Unmarshal(v, &headers); err != nil {
			return nil, "", fmt.Errorf("mcp server %q: headers must be object of strings; skipped", name)
		}
		// Case-insensitive duplicate header names are invalid.
		seen := map[string]string{}
		for k, val := range headers {
			lk := strings.ToLower(k)
			if prev, ok := seen[lk]; ok && prev != k {
				return nil, "", fmt.Errorf("mcp server %q: duplicate header name %q/%q; skipped", name, prev, k)
			}
			seen[lk] = k
			_ = val
		}
		// Headers are visible package data — not a portable secret mechanism.
		// We map as-is; secrets never come from portable fields (client-owned).
		ref.Headers = headers
	}
	return ref, "", nil
}

func validateMCPURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("url scheme must be http or https")
	}
	if u.User != nil {
		return fmt.Errorf("url must not contain user information")
	}
	if u.Fragment != "" {
		return fmt.Errorf("url must not contain a fragment")
	}
	if u.Host == "" {
		return fmt.Errorf("url host is empty")
	}
	if u.Scheme == "http" && !isLoopbackHost(u.Hostname()) {
		return fmt.Errorf("non-loopback endpoints must use https")
	}
	return nil
}

func isLoopbackHost(host string) bool {
	h := strings.ToLower(host)
	if h == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}
