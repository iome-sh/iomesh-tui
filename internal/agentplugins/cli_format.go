package agentplugins

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ResidualCLIHonesty is the residual-honest one-liner for plugins CLI footers (s1336).
// list/validate ≠ invent Agent Plugins GA · dual_write OFF · Discover ≠ Connected · not Memory GA · book-demo OFF.
const ResidualCLIHonesty = "honesty: list/validate ≠ invent Agent Plugins GA · dual_write OFF · Discover ≠ Connected · not Memory GA · book-demo OFF"

// PluginsOptInMessage is printed when no dirs are available and [plugins] is default-off.
const PluginsOptInMessage = "[plugins] is opt-in (default enabled=false). Set [plugins].dirs or pass -dir for one-shot list/validate. package wire ≠ invent Agent Plugins GA."

// PluginListRow is one row for `iomesh plugins list`.
type PluginListRow struct {
	Name    string
	Version string
	Skills  int
	MCP     int
	Warn    int
	Root    string
}

// PluginToListRow maps a discovered plugin to a list table row.
func PluginToListRow(p *Plugin) PluginListRow {
	if p == nil {
		return PluginListRow{Name: "-", Version: "-", Root: "-"}
	}
	ver := p.Manifest.Version
	if ver == "" {
		ver = "-"
	}
	return PluginListRow{
		Name:    p.Manifest.Name,
		Version: ver,
		Skills:  len(p.Skills),
		MCP:     len(p.MCPServers),
		Warn:    len(p.Warnings),
		Root:    p.Root,
	}
}

// FormatListHeader returns the column header for plugins list.
func FormatListHeader() string {
	return fmt.Sprintf("%-24s %-12s %6s %4s %4s  %s", "NAME", "VERSION", "SKILLS", "MCP", "WARN", "ROOT")
}

// FormatListRow formats one plugins list table row.
func FormatListRow(r PluginListRow) string {
	name := r.Name
	if name == "" {
		name = "-"
	}
	ver := r.Version
	if ver == "" {
		ver = "-"
	}
	root := r.Root
	if root == "" {
		root = "-"
	}
	return fmt.Sprintf("%-24s %-12s %6d %4d %4d  %s", name, ver, r.Skills, r.MCP, r.Warn, root)
}

// FormatListEmptyFooter returns residual-honest empty-list guidance.
// When dirs are empty and plugins not enabled, prefers the opt-in message.
func FormatListEmptyFooter(pluginsEnabled bool, dirsSpecified bool) string {
	if !dirsSpecified && !pluginsEnabled {
		return PluginsOptInMessage
	}
	if !dirsSpecified {
		return "no plugins found: [plugins].dirs empty (and no -dir). Discover ≠ Connected · package wire ≠ invent Agent Plugins GA."
	}
	return "no plugins discovered (fail-open). Discover ≠ Connected / install APPLY green · package wire ≠ invent Agent Plugins GA."
}

// ValidateOutcome is one package-root result for `iomesh plugins validate`.
type ValidateOutcome struct {
	Path     string
	OK       bool
	Name     string
	Version  string
	Skills   int
	MCP      int
	Warnings []string
	Error    string // non-empty when !OK
}

// FormatValidateOK formats a successful package validation line.
func FormatValidateOK(o ValidateOutcome) string {
	name := o.Name
	if name == "" {
		name = "-"
	}
	ver := o.Version
	if ver == "" {
		ver = "-"
	}
	return fmt.Sprintf("OK   %-24s version=%-12s skills=%d mcp=%d warnings=%d  %s",
		name, ver, o.Skills, o.MCP, len(o.Warnings), o.Path)
}

// FormatValidateFail formats a fatal package validation line.
func FormatValidateFail(path, errMsg string) string {
	if path == "" {
		path = "-"
	}
	if errMsg == "" {
		errMsg = "unknown error"
	}
	return fmt.Sprintf("FAIL %s: %s", path, errMsg)
}

// ParseDirFlagValue splits a -dir flag value on commas and trims empties.
// Supports repeatable flags by calling Set/ParseDirFlagValue per occurrence.
func ParseDirFlagValue(v string) []string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// MergePluginDirs appends CLI -dir values after config dirs (supplement).
// Empty entries are dropped. Order preserved; duplicates kept (DiscoverAll de-dupes roots).
func MergePluginDirs(configDirs, cliDirs []string) []string {
	var out []string
	for _, d := range configDirs {
		d = strings.TrimSpace(d)
		if d != "" {
			out = append(out, d)
		}
	}
	for _, d := range cliDirs {
		d = strings.TrimSpace(d)
		if d != "" {
			out = append(out, d)
		}
	}
	return out
}

// DirFlag is a flag.Value for repeatable -dir (comma-separated each occurrence).
type DirFlag []string

// String implements flag.Value.
func (d *DirFlag) String() string {
	if d == nil {
		return ""
	}
	return strings.Join(*d, ",")
}

// Set implements flag.Value; appends ParseDirFlagValue(v).
func (d *DirFlag) Set(v string) error {
	*d = append(*d, ParseDirFlagValue(v)...)
	return nil
}

// ValidateDirs inspects each configured dir (package root or parent of roots)
// and reports OK/FAIL per package root for operator DX (s1336).
// Unlike DiscoverAll (fail-open list), ValidateDirs surfaces fatals explicitly.
// Residual honesty: OK ≠ Connected / install APPLY green · not Agent Plugins GA.
func ValidateDirs(dirs []string) (outcomes []ValidateOutcome, scanWarnings []string) {
	seenRoots := map[string]struct{}{}
	for _, raw := range dirs {
		dir := expandHomePath(strings.TrimSpace(raw))
		if dir == "" {
			continue
		}
		abs, err := filepath.Abs(dir)
		if err != nil {
			outcomes = append(outcomes, ValidateOutcome{
				Path:  raw,
				OK:    false,
				Error: fmt.Sprintf("abs: %v", err),
			})
			continue
		}
		st, err := os.Stat(abs)
		if err != nil {
			outcomes = append(outcomes, ValidateOutcome{
				Path:  abs,
				OK:    false,
				Error: err.Error(),
			})
			continue
		}
		if !st.IsDir() {
			outcomes = append(outcomes, ValidateOutcome{
				Path:  abs,
				OK:    false,
				Error: "not a directory",
			})
			continue
		}

		if hasPluginJSON(abs) {
			outcomes = append(outcomes, validateOne(abs, seenRoots))
			continue
		}

		// Parent of package roots: scan immediate children only.
		entries, err := os.ReadDir(abs)
		if err != nil {
			outcomes = append(outcomes, ValidateOutcome{
				Path:  abs,
				OK:    false,
				Error: fmt.Sprintf("readdir: %v", err),
			})
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
			outcomes = append(outcomes, validateOne(child, seenRoots))
		}
		if !foundChild {
			outcomes = append(outcomes, ValidateOutcome{
				Path:  abs,
				OK:    false,
				Error: "no plugin.json at root or immediate children",
			})
		}
	}
	// Stable order: FAIL first then OK, then by path.
	sort.SliceStable(outcomes, func(i, j int) bool {
		if outcomes[i].OK != outcomes[j].OK {
			return !outcomes[i].OK && outcomes[j].OK
		}
		return outcomes[i].Path < outcomes[j].Path
	})
	return outcomes, scanWarnings
}

func validateOne(root string, seen map[string]struct{}) ValidateOutcome {
	abs, err := resolveRoot(root)
	if err != nil {
		return ValidateOutcome{Path: root, OK: false, Error: err.Error()}
	}
	if _, ok := seen[abs]; ok {
		return ValidateOutcome{Path: abs, OK: false, Error: "duplicate root"}
	}
	p, err := Discover(abs)
	if err != nil {
		return ValidateOutcome{Path: abs, OK: false, Error: err.Error()}
	}
	seen[abs] = struct{}{}
	ver := p.Manifest.Version
	if ver == "" {
		ver = "-"
	}
	return ValidateOutcome{
		Path:     abs,
		OK:       true,
		Name:     p.Manifest.Name,
		Version:  ver,
		Skills:   len(p.Skills),
		MCP:      len(p.MCPServers),
		Warnings: append([]string{}, p.Warnings...),
	}
}

// ValidateHasFatal reports whether any outcome is a fatal FAIL.
func ValidateHasFatal(outcomes []ValidateOutcome) bool {
	for _, o := range outcomes {
		if !o.OK {
			return true
		}
	}
	return false
}

// ValidateOKCount returns the number of OK outcomes (successful package validates).
func ValidateOKCount(outcomes []ValidateOutcome) int {
	n := 0
	for _, o := range outcomes {
		if o.OK {
			n++
		}
	}
	return n
}
