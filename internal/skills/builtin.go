package skills

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// Builtin skills shipped with the binary:
//   - s1251 connector-integrations-setup
//   - s1288 memory-advanced-agent
//
// Layout: builtin/<name>/SKILL.md — always merged when skills are enabled so
// residual-honest guidance is available even if user/workspace skill dirs are empty.
//
//go:embed builtin/*/SKILL.md
var builtinFS embed.FS

// LoadBuiltin returns the catalog of embedded skills.
// Always non-empty when at least one SKILL.md is embedded.
func LoadBuiltin() (*Catalog, error) {
	c := &Catalog{byName: map[string]Skill{}}
	err := fs.WalkDir(builtinFS, "builtin", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		base := path.Base(p)
		if !strings.EqualFold(base, "SKILL.md") {
			return nil
		}
		data, err := builtinFS.ReadFile(p)
		if err != nil {
			return fmt.Errorf("skills builtin read %s: %w", p, err)
		}
		// Cap skill size (same as loadFile).
		const maxBytes = 256 * 1024
		if len(data) > maxBytes {
			data = data[:maxBytes]
		}
		sk, err := Parse(string(data))
		if err != nil {
			return fmt.Errorf("skills builtin parse %s: %w", p, err)
		}
		if sk.Name == "" {
			// Parent dir under builtin/ is the skill name.
			parent := path.Base(path.Dir(p))
			if parent != "" && parent != "." && parent != "builtin" {
				sk.Name = parent
			}
		}
		sk.Name = sanitizeName(sk.Name)
		if sk.Name == "" {
			return nil
		}
		sk.Path = "builtin:" + p
		sk.SourceDir = "builtin"
		if _, exists := c.byName[sk.Name]; !exists {
			c.order = append(c.order, sk.Name)
		}
		c.byName[sk.Name] = sk
		return nil
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

// Merge overlays other onto c. Names already in c are **overwritten** by other
// (user/workspace skills override builtin). New names from other are appended
// in other's load order. Nil other is a no-op. If c is nil, returns a copy of other.
func (c *Catalog) Merge(other *Catalog) *Catalog {
	if other == nil || other.Len() == 0 {
		if c == nil {
			return &Catalog{byName: map[string]Skill{}}
		}
		return c
	}
	if c == nil {
		out := &Catalog{byName: map[string]Skill{}, order: make([]string, 0, other.Len())}
		for _, n := range other.order {
			if sk, ok := other.byName[n]; ok {
				out.byName[n] = sk
				out.order = append(out.order, n)
			}
		}
		return out
	}
	for _, n := range other.order {
		sk, ok := other.byName[n]
		if !ok {
			continue
		}
		if _, exists := c.byName[n]; !exists {
			c.order = append(c.order, n)
		}
		c.byName[n] = sk
	}
	return c
}

// LoadWithBuiltin loads builtin skills first, then overlays dirs (user/workspace
// win on name collision). Missing dirs are ignored. Always returns builtin skills
// even when all dirs are empty — so connector-integrations-setup (s1251) and
// memory-advanced-agent (s1288) always appear when skills are enabled.
func LoadWithBuiltin(dirs ...string) (*Catalog, error) {
	cat, err := LoadBuiltin()
	if err != nil {
		return nil, err
	}
	user, err := LoadDirs(dirs...)
	if err != nil {
		return nil, err
	}
	return cat.Merge(user), nil
}
