// Package skills loads SKILL.md catalogs (YAML frontmatter + markdown body)
// from workspace and user directories for agent discovery.
package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Skill is one loaded skill document.
type Skill struct {
	Name        string
	Description string
	Body        string
	Path        string
	SourceDir   string
}

// Catalog is a name-indexed skill set (last load wins on name collision).
type Catalog struct {
	byName map[string]Skill
	order  []string
}

// LoadDirs walks dirs for SKILL.md (or *.md with frontmatter name).
// Missing dirs are ignored. Empty dirs yield an empty catalog.
func LoadDirs(dirs ...string) (*Catalog, error) {
	c := &Catalog{byName: map[string]Skill{}}
	for _, dir := range dirs {
		dir = expandHome(strings.TrimSpace(dir))
		if dir == "" {
			continue
		}
		info, err := os.Stat(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("skills: stat %s: %w", dir, err)
		}
		if !info.IsDir() {
			continue
		}
		if err := c.loadDir(dir); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func (c *Catalog) loadDir(dir string) error {
	// Prefer subdir/SKILL.md layout (Grok / Claude style).
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("skills: readdir %s: %w", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			p := filepath.Join(dir, e.Name(), "SKILL.md")
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				if err := c.loadFile(p, dir); err != nil {
					return err
				}
			}
			continue
		}
		name := e.Name()
		if strings.EqualFold(name, "SKILL.md") || (strings.HasSuffix(strings.ToLower(name), ".md") && strings.Contains(strings.ToLower(name), "skill")) {
			if err := c.loadFile(filepath.Join(dir, name), dir); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Catalog) loadFile(path, sourceDir string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("skills: read %s: %w", path, err)
	}
	// Cap skill size (DoS / prompt bloat).
	const maxBytes = 256 * 1024
	if len(data) > maxBytes {
		data = data[:maxBytes]
	}
	sk, err := Parse(string(data))
	if err != nil {
		return fmt.Errorf("skills: parse %s: %w", path, err)
	}
	if sk.Name == "" {
		// Fall back to parent dir name or file stem.
		base := filepath.Base(filepath.Dir(path))
		if strings.EqualFold(filepath.Base(path), "SKILL.md") && base != "." && base != "/" {
			sk.Name = base
		} else {
			sk.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		}
	}
	sk.Name = sanitizeName(sk.Name)
	if sk.Name == "" {
		return nil
	}
	sk.Path = path
	sk.SourceDir = sourceDir
	if _, exists := c.byName[sk.Name]; !exists {
		c.order = append(c.order, sk.Name)
	}
	c.byName[sk.Name] = sk
	return nil
}

// Len returns skill count.
func (c *Catalog) Len() int {
	if c == nil {
		return 0
	}
	return len(c.byName)
}

// List returns skills in load order.
func (c *Catalog) List() []Skill {
	if c == nil {
		return nil
	}
	out := make([]Skill, 0, len(c.order))
	for _, n := range c.order {
		if sk, ok := c.byName[n]; ok {
			out = append(out, sk)
		}
	}
	return out
}

// skillNameAliases maps deprecated public ids to the current builtin name.
// aion-agent-onboarding is a one-release loader alias for mesh-agent-onboarding (#394).
var skillNameAliases = map[string]string{
	"aion-agent-onboarding": "mesh-agent-onboarding",
}

// Get returns a skill by name. Deprecated aliases resolve to the public id.
func (c *Catalog) Get(name string) (Skill, bool) {
	if c == nil {
		return Skill{}, false
	}
	key := sanitizeName(name)
	if alias, ok := skillNameAliases[key]; ok {
		key = alias
	}
	sk, ok := c.byName[key]
	return sk, ok
}

// Names returns sorted skill names.
func (c *Catalog) Names() []string {
	if c == nil {
		return nil
	}
	names := make([]string, 0, len(c.byName))
	for n := range c.byName {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// PromptBlock is a compact catalog for the system prompt.
func (c *Catalog) PromptBlock() string {
	if c == nil || c.Len() == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Available skills (use list_skills / read_skill tools for full content):\n")
	for _, sk := range c.List() {
		desc := sk.Description
		if desc == "" {
			desc = "(no description)"
		}
		if len(desc) > 160 {
			desc = desc[:157] + "…"
		}
		fmt.Fprintf(&b, "- %s: %s\n", sk.Name, strings.ReplaceAll(desc, "\n", " "))
	}
	return b.String()
}

// Parse extracts YAML-ish frontmatter and body from a SKILL.md document.
func Parse(raw string) (Skill, error) {
	raw = strings.TrimPrefix(raw, "\uFEFF")
	var sk Skill
	if !strings.HasPrefix(strings.TrimSpace(raw), "---") {
		sk.Body = strings.TrimSpace(raw)
		return sk, nil
	}
	// Frontmatter between first two --- lines.
	rest := strings.TrimSpace(raw)
	rest = strings.TrimPrefix(rest, "---")
	rest = strings.TrimLeft(rest, "\r\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		sk.Body = strings.TrimSpace(raw)
		return sk, nil
	}
	fm := rest[:end]
	body := rest[end+4:] // after \n---
	body = strings.TrimLeft(body, "\r\n")
	sk.Body = strings.TrimSpace(body)

	// Minimal key: value parser (no full YAML dependency).
	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Skip nested blocks (metadata:) for simplicity; capture name/description only.
		if !strings.Contains(line, ":") {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"'`)
		// Multi-line `>` description: take rest of scalar on same line only;
		// folded blocks are uncommon in our tests — capture single-line.
		if strings.HasPrefix(val, ">") {
			val = strings.TrimSpace(strings.TrimPrefix(val, ">"))
		}
		switch key {
		case "name":
			sk.Name = val
		case "description":
			sk.Description = val
		}
	}
	// If description used YAML folded `>` with following indented lines:
	if sk.Description == "" {
		// collect indented lines after description: >
		lines := strings.Split(fm, "\n")
		for i, line := range lines {
			trim := strings.TrimSpace(line)
			if strings.HasPrefix(strings.ToLower(trim), "description:") {
				rest := strings.TrimSpace(strings.TrimPrefix(trim, "description:"))
				rest = strings.TrimSpace(strings.TrimPrefix(rest, "Description:"))
				if strings.HasPrefix(rest, ">") || rest == "" {
					var parts []string
					if strings.HasPrefix(rest, ">") {
						parts = append(parts, strings.TrimSpace(strings.TrimPrefix(rest, ">")))
					}
					for j := i + 1; j < len(lines); j++ {
						l := lines[j]
						if len(l) > 0 && (l[0] == ' ' || l[0] == '\t') {
							parts = append(parts, strings.TrimSpace(l))
							continue
						}
						break
					}
					sk.Description = strings.TrimSpace(strings.Join(parts, " "))
				} else {
					sk.Description = strings.Trim(rest, `"'`)
				}
				break
			}
		}
	}
	return sk, nil
}

func sanitizeName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteByte('-')
		}
	}
	return b.String()
}

func expandHome(p string) string {
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

// DefaultDirs returns workspace + user skill roots.
func DefaultDirs(workspace string) []string {
	dirs := []string{}
	if workspace != "" {
		dirs = append(dirs, filepath.Join(workspace, ".iomesh", "skills"))
	}
	if h, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(h, ".iomesh", "skills"))
	}
	return dirs
}
