package agentplugins

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SkillRef is a discovered skill (immediate child of skills/ with SKILL.md).
type SkillRef struct {
	// Name is the skill directory name (not necessarily frontmatter name).
	Name string
	// Path is the absolute path to SKILL.md.
	Path string
	// Dir is the absolute skill directory.
	Dir string
}

// Plugin is a discovered Agent Plugin package (validation + map only).
type Plugin struct {
	Root       string
	Manifest   PluginManifest
	Skills     []SkillRef
	MCPServers []MCPServerRef
	Warnings   []string
}

// Discover loads plugin.json, discovers skills/ and mcp.json under pluginRoot.
// Missing skills/ or mcp.json is OK. Component failures are fail-open per type.
// Does not attach MCP processes or inject skills into the agent runtime.
func Discover(pluginRoot string) (*Plugin, error) {
	absRoot, err := resolveRoot(pluginRoot)
	if err != nil {
		return nil, err
	}
	// Ensure root is a directory.
	st, err := os.Stat(absRoot)
	if err != nil {
		return nil, fmt.Errorf("agentplugins: plugin root: %w", err)
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("agentplugins: plugin root is not a directory")
	}

	mr, err := LoadManifest(absRoot)
	if err != nil {
		return nil, err
	}

	p := &Plugin{
		Root:     absRoot,
		Manifest: mr.Manifest,
		Warnings: append([]string{}, mr.Warnings...),
	}

	skills, sw := discoverSkills(absRoot)
	p.Skills = skills
	p.Warnings = append(p.Warnings, sw...)

	mcp, err := LoadMCPJSON(absRoot)
	if err != nil {
		p.Warnings = append(p.Warnings, fmt.Sprintf("mcp.json: %v", err))
	} else if mcp != nil {
		p.MCPServers = mcp.Servers
		p.Warnings = append(p.Warnings, mcp.Warnings...)
	}

	return p, nil
}

// discoverSkills scans only immediate children of skills/ for SKILL.md.
// No deep recursion. Missing skills/ is OK.
func discoverSkills(pluginRoot string) ([]SkillRef, []string) {
	var warnings []string
	skillsDir := filepath.Join(pluginRoot, "skills")
	st, err := os.Stat(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []string{fmt.Sprintf("skills/: %v", err)}
	}
	if !st.IsDir() {
		return nil, []string{"skills is not a directory; skills component invalid"}
	}
	// Containment: skillsDir must stay under root.
	absSkills, err := filepath.Abs(skillsDir)
	if err != nil {
		return nil, []string{fmt.Sprintf("skills/: %v", err)}
	}
	absRoot, _ := resolveRoot(pluginRoot)
	if resolved, err := filepath.EvalSymlinks(absSkills); err == nil {
		absSkills = resolved
	}
	if !withinRoot(absSkills, absRoot) {
		return nil, []string{"skills/ resolves outside plugin root; skills component invalid"}
	}

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, []string{fmt.Sprintf("skills/: readdir: %v", err)}
	}

	var out []SkillRef
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// Skip hidden / extension-like noise? Spec: each immediate child dir with SKILL.md.
		name := e.Name()
		if name == "" || name == "." || name == ".." {
			continue
		}
		skillMD := filepath.Join(skillsDir, name, "SKILL.md")
		// Path confinement for discovered SKILL.md.
		if !withinRoot(filepath.Clean(skillMD), absRoot) {
			warnings = append(warnings, fmt.Sprintf("skill %q: path escapes root; skipped", name))
			continue
		}
		if resolved, err := filepath.EvalSymlinks(skillMD); err == nil {
			if !withinRoot(resolved, absRoot) {
				warnings = append(warnings, fmt.Sprintf("skill %q: SKILL.md resolves outside root; skipped", name))
				continue
			}
			skillMD = resolved
		}
		info, err := os.Stat(skillMD)
		if err != nil {
			if os.IsNotExist(err) {
				// No SKILL.md — not a skill; skip silently (not an error).
				continue
			}
			warnings = append(warnings, fmt.Sprintf("skill %q: %v; skipped", name, err))
			continue
		}
		if info.IsDir() {
			warnings = append(warnings, fmt.Sprintf("skill %q: SKILL.md is a directory; skipped", name))
			continue
		}
		dir := filepath.Join(skillsDir, name)
		if resolved, err := filepath.EvalSymlinks(dir); err == nil {
			dir = resolved
		}
		out = append(out, SkillRef{
			Name: name,
			Path: skillMD,
			Dir:  dir,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, warnings
}

// ExpandMCPServer returns a copy of ref with args/env/cwd placeholders expanded.
// Command is never expanded. pluginData is the client-managed PLUGIN_DATA path.
func ExpandMCPServer(ref MCPServerRef, pluginRoot, pluginData string) MCPServerRef {
	out := ref
	out.Args = ExpandStringSlice(ref.Args, pluginRoot, pluginData)
	out.Env = ExpandEnvMap(ref.Env, pluginRoot, pluginData)
	if ref.Cwd != "" {
		out.Cwd = ExpandPlaceholders(ref.Cwd, pluginRoot, pluginData)
	}
	return out
}

// Summary returns a one-line residual-honest summary of the discovered package.
func (p *Plugin) Summary() string {
	if p == nil {
		return "plugin: nil"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "plugin %q skills=%d mcp_servers=%d warnings=%d",
		p.Manifest.Name, len(p.Skills), len(p.MCPServers), len(p.Warnings))
	return b.String()
}
