package agentplugins

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/iome-sh/iomesh-tui/internal/mcp"
)

// DiscoverAll discovers plugins from configured dirs (fail-open per dir / child).
// Each dir may be:
//   - a plugin package root (contains plugin.json), or
//   - a parent whose immediate children are package roots.
//
// Paths support ~/ and ~ expansion. Empty entries are skipped.
// Residual honesty: discovery success ≠ Connected / install APPLY green.
func DiscoverAll(dirs []string) (plugins []*Plugin, warnings []string) {
	seenRoots := map[string]struct{}{}
	for _, raw := range dirs {
		dir := expandHomePath(strings.TrimSpace(raw))
		if dir == "" {
			continue
		}
		abs, err := filepath.Abs(dir)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("plugins dir %q: abs: %v; skipped", raw, err))
			continue
		}
		st, err := os.Stat(abs)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("plugins dir %q: %v; skipped", raw, err))
			continue
		}
		if !st.IsDir() {
			warnings = append(warnings, fmt.Sprintf("plugins dir %q: not a directory; skipped", raw))
			continue
		}

		// Package root?
		if hasPluginJSON(abs) {
			p, w := discoverOne(abs, seenRoots)
			if p != nil {
				plugins = append(plugins, p)
			}
			warnings = append(warnings, w...)
			continue
		}

		// Parent of package roots: scan immediate children only.
		entries, err := os.ReadDir(abs)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("plugins dir %q: readdir: %v; skipped", raw, err))
			continue
		}
		foundChild := false
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			child := filepath.Join(abs, e.Name())
			if !hasPluginJSON(child) {
				continue
			}
			foundChild = true
			p, w := discoverOne(child, seenRoots)
			if p != nil {
				plugins = append(plugins, p)
			}
			warnings = append(warnings, w...)
		}
		if !foundChild {
			warnings = append(warnings, fmt.Sprintf("plugins dir %q: no plugin.json at root or immediate children; skipped", raw))
		}
	}
	// Stable order by manifest name then root.
	sort.SliceStable(plugins, func(i, j int) bool {
		ni, nj := plugins[i].Manifest.Name, plugins[j].Manifest.Name
		if ni != nj {
			return ni < nj
		}
		return plugins[i].Root < plugins[j].Root
	})
	return plugins, warnings
}

func hasPluginJSON(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, "plugin.json"))
	return err == nil && !st.IsDir()
}

func discoverOne(root string, seen map[string]struct{}) (*Plugin, []string) {
	abs, err := resolveRoot(root)
	if err != nil {
		return nil, []string{fmt.Sprintf("plugin %q: %v; skipped", root, err)}
	}
	if _, ok := seen[abs]; ok {
		return nil, []string{fmt.Sprintf("plugin %q: duplicate root; skipped", abs)}
	}
	p, err := Discover(abs)
	if err != nil {
		return nil, []string{fmt.Sprintf("plugin %q: %v; skipped", abs, err)}
	}
	seen[abs] = struct{}{}
	return p, nil
}

// SkillDirs returns absolute skills/ parent dirs for skills.LoadDirs / LoadWithBuiltin.
// Plugin skills live at pluginRoot/skills/<name>/SKILL.md — pass pluginRoot/skills.
// Plugins with no discovered skills are omitted. Order is stable by path.
func SkillDirs(plugins []*Plugin) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, p := range plugins {
		if p == nil || len(p.Skills) == 0 {
			continue
		}
		d := filepath.Join(p.Root, "skills")
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}

// DefaultPluginDataRoot returns workspace/.iomesh/plugin-data when dataDirRoot is empty.
// Caller should pass cfg.Plugins.DataDir when set; otherwise this default under workspace.
func DefaultPluginDataRoot(workspace, dataDirRoot string) string {
	if strings.TrimSpace(dataDirRoot) != "" {
		return expandHomePath(strings.TrimSpace(dataDirRoot))
	}
	ws := strings.TrimSpace(workspace)
	if ws == "" {
		if wd, err := os.Getwd(); err == nil {
			ws = wd
		}
	}
	if ws == "" {
		return filepath.Join(".iomesh", "plugin-data")
	}
	return filepath.Join(ws, ".iomesh", "plugin-data")
}

// MCPServersFromPlugins maps discovered plugin MCP servers to mcp.ServerConfig.
// TOML [[mcp.servers]] remains primary — callers should append these after TOML servers.
//
// Mapping rules (fail-open per server):
//   - name: <pluginManifest.name>-<serverName> (stable unique across plugins)
//   - streamable-http/sse: URL + headers (PLUGIN_* expanded in header values only)
//   - stdio: Command as-is (no placeholder expand); ./ paths resolved under plugin root;
//     Args/Env/Cwd expanded via ExpandPlaceholders; inject PLUGIN_ROOT + PLUGIN_DATA env
//   - Mutating left nil → default true (fail-closed approvals)
//   - PLUGIN_DATA dir created (MkdirAll) per plugin before return
//   - secrets never invented from portable fields (map package headers/env only after expand)
//
// Residual honesty: map success ≠ Connected / install APPLY green · dual_write OFF.
func MCPServersFromPlugins(plugins []*Plugin, dataDirRoot string) ([]mcp.ServerConfig, []string) {
	var (
		out      []mcp.ServerConfig
		warnings []string
		// Track names for uniqueness within this batch (TOML collisions handled by manager last-wins).
		usedNames = map[string]struct{}{}
	)
	dataRoot := expandHomePath(strings.TrimSpace(dataDirRoot))
	if dataRoot == "" {
		dataRoot = filepath.Join(".iomesh", "plugin-data")
	}

	for _, p := range plugins {
		if p == nil || len(p.MCPServers) == 0 {
			continue
		}
		pluginData := filepath.Join(dataRoot, p.Manifest.Name)
		if err := os.MkdirAll(pluginData, 0o755); err != nil {
			warnings = append(warnings, fmt.Sprintf("plugin %q: PLUGIN_DATA mkdir %s: %v; MCP servers skipped",
				p.Manifest.Name, pluginData, err))
			continue
		}
		// Prefer absolute, symlink-resolved data dir for env injection.
		if abs, err := filepath.Abs(pluginData); err == nil {
			if resolved, err := filepath.EvalSymlinks(abs); err == nil {
				pluginData = resolved
			} else {
				pluginData = abs
			}
		}

		for _, ref := range p.MCPServers {
			sc, warn, err := mapMCPServer(p, ref, pluginData, usedNames)
			if err != nil {
				warnings = append(warnings, err.Error())
				continue
			}
			if warn != "" {
				warnings = append(warnings, warn)
			}
			if sc != nil {
				out = append(out, *sc)
			}
		}
	}
	return out, warnings
}

func mapMCPServer(p *Plugin, ref MCPServerRef, pluginData string, usedNames map[string]struct{}) (*mcp.ServerConfig, string, error) {
	name := pluginServerName(p.Manifest.Name, ref.Name)
	if _, ok := usedNames[name]; ok {
		return nil, "", fmt.Errorf("plugin %q mcp %q: name %q already used; skipped", p.Manifest.Name, ref.Name, name)
	}

	expanded := ExpandMCPServer(ref, p.Root, pluginData)

	switch expanded.Type {
	case TransportStdio:
		cmd := expanded.Command
		// Resolve ./ relative commands to absolute under plugin root (no placeholder expand).
		if strings.HasPrefix(cmd, "./") {
			absCmd, err := Confine(cmd, p.Root)
			if err != nil {
				return nil, "", fmt.Errorf("plugin %q mcp %q: command path: %v; skipped", p.Manifest.Name, ref.Name, err)
			}
			cmd = absCmd
		}
		env := map[string]string{}
		for k, v := range expanded.Env {
			// Never let package set client-owned keys (already rejected at parse; belt+suspenders).
			if k == "PLUGIN_ROOT" || k == "PLUGIN_DATA" {
				continue
			}
			env[k] = v
		}
		// Client-owned injection: absolute paths only.
		env["PLUGIN_ROOT"] = p.Root
		env["PLUGIN_DATA"] = pluginData

		sc := &mcp.ServerConfig{
			Name:    name,
			Command: cmd,
			Args:    expanded.Args,
			Env:     env,
			Cwd:     expanded.Cwd,
			// Mutating nil → default true (fail-closed approvals).
		}
		usedNames[name] = struct{}{}
		return sc, "", nil

	case TransportStreamableHTTP, TransportSSE:
		headers := map[string]string{}
		for k, v := range expanded.Headers {
			// Expand PLUGIN_* only; secrets never invented from portable fields.
			headers[k] = ExpandPlaceholders(v, p.Root, pluginData)
		}
		sc := &mcp.ServerConfig{
			Name:    name,
			URL:     expanded.URL,
			Headers: headers,
			// Mutating nil → default true.
		}
		usedNames[name] = struct{}{}
		return sc, "", nil

	default:
		return nil, "", fmt.Errorf("plugin %q mcp %q: unsupported type %q; skipped", p.Manifest.Name, ref.Name, expanded.Type)
	}
}

// pluginServerName builds a stable runtime server name: <plugin>-<server>.
func pluginServerName(pluginName, serverName string) string {
	pluginName = strings.TrimSpace(pluginName)
	serverName = strings.TrimSpace(serverName)
	if pluginName == "" {
		return serverName
	}
	if serverName == "" {
		return pluginName
	}
	return pluginName + "-" + serverName
}

func expandHomePath(p string) string {
	if p == "" {
		return p
	}
	if p == "~" {
		if h, err := os.UserHomeDir(); err == nil {
			return h
		}
		return p
	}
	if strings.HasPrefix(p, "~/") {
		if h, err := os.UserHomeDir(); err == nil {
			return filepath.Join(h, p[2:])
		}
	}
	return p
}
