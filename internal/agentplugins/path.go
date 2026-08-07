// Package agentplugins implements the Agent Plugins v1.0.0 package client
// slice: closed plugin.json validation, fixed-location discovery of skills
// and mcp.json mapping, plus opt-in runtime wire helpers (s1331) that map
// packages into skills dirs and mcp.ServerConfig for the existing Skills + MCP
// runtimes. Discover/map success ≠ Connected / install APPLY green · not GA.
package agentplugins

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Confine resolves a plugin-relative path that MUST begin with "./" against
// root and ensures the result stays inside the filesystem-resolved root.
// Returns the absolute cleaned path on success.
func Confine(rel, root string) (string, error) {
	rel = strings.TrimSpace(rel)
	root = strings.TrimSpace(root)
	if root == "" {
		return "", fmt.Errorf("agentplugins: empty plugin root")
	}
	if !strings.HasPrefix(rel, "./") {
		return "", fmt.Errorf("agentplugins: path %q must start with ./", rel)
	}
	absRoot, err := resolveRoot(root)
	if err != nil {
		return "", err
	}
	// Strip leading "./" then clean; join under root.
	suffix := strings.TrimPrefix(rel, "./")
	joined := filepath.Join(absRoot, filepath.FromSlash(suffix))
	cleaned := filepath.Clean(joined)
	if !withinRoot(cleaned, absRoot) {
		return "", fmt.Errorf("agentplugins: path %q escapes plugin root", rel)
	}
	// EvalSymlinks when the target exists; still enforce containment.
	if resolved, err := filepath.EvalSymlinks(cleaned); err == nil {
		if !withinRoot(resolved, absRoot) {
			return "", fmt.Errorf("agentplugins: path %q resolves outside plugin root", rel)
		}
		return resolved, nil
	}
	return cleaned, nil
}

// resolveRoot returns the absolute, cleaned, optionally symlink-resolved root.
func resolveRoot(root string) (string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("agentplugins: abs root: %w", err)
	}
	abs = filepath.Clean(abs)
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved, nil
	}
	return abs, nil
}

// withinRoot reports whether candidate is absRoot or a descendant.
func withinRoot(candidate, absRoot string) bool {
	candidate = filepath.Clean(candidate)
	absRoot = filepath.Clean(absRoot)
	if candidate == absRoot {
		return true
	}
	sep := string(filepath.Separator)
	prefix := absRoot
	if !strings.HasSuffix(prefix, sep) {
		prefix += sep
	}
	return strings.HasPrefix(candidate, prefix)
}
