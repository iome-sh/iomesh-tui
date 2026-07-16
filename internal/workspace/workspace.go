// Package workspace abstracts host filesystem operations rooted at a project
// directory (Grok Build workspace crate equivalent).
package workspace

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Workspace is a project-rooted filesystem view.
type Workspace struct {
	root string
}

// Open resolves root ("" → cwd) and verifies it is a directory.
func Open(root string) (*Workspace, error) {
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		root = wd
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("workspace: not a directory: %s", abs)
	}
	return &Workspace{root: abs}, nil
}

// Root returns the absolute workspace path.
func (w *Workspace) Root() string { return w.root }

// Resolve cleans a relative path and ensures it stays under root.
func (w *Workspace) Resolve(rel string) (string, error) {
	if rel == "" {
		rel = "."
	}
	joined := filepath.Join(w.root, rel)
	abs, err := filepath.Abs(joined)
	if err != nil {
		return "", err
	}
	relToRoot, err := filepath.Rel(w.root, abs)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(relToRoot, "..") {
		return "", fmt.Errorf("path escapes workspace: %s", rel)
	}
	return abs, nil
}

// ReadFile returns file contents; offset/limit are 1-based line numbers (0 = all).
func (w *Workspace) ReadFile(rel string, offset, limit int) (string, error) {
	path, err := w.Resolve(rel)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if offset <= 0 && limit <= 0 {
		return string(data), nil
	}
	lines := strings.Split(string(data), "\n")
	start := 0
	if offset > 0 {
		start = offset - 1
	}
	if start > len(lines) {
		return "", nil
	}
	end := len(lines)
	if limit > 0 && start+limit < end {
		end = start + limit
	}
	var b strings.Builder
	for i := start; i < end; i++ {
		fmt.Fprintf(&b, "%d|%s\n", i+1, lines[i])
	}
	return b.String(), nil
}

// WriteFile writes content to a path under the workspace (creates parents).
func (w *Workspace) WriteFile(rel, content string) error {
	path, err := w.Resolve(rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// ListDir lists directory entries (names only).
func (w *Workspace) ListDir(rel string) ([]string, error) {
	path, err := w.Resolve(rel)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		out = append(out, name)
	}
	return out, nil
}

// Grep searches files under rel (or root) for pattern. Caps output for agents.
func (w *Workspace) Grep(pattern, rel string) (string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}
	root := w.root
	if rel != "" {
		root, err = w.Resolve(rel)
		if err != nil {
			return "", err
		}
	}
	var b strings.Builder
	matches := 0
	const maxMatches = 50
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && (d.Name() == ".git" || d.Name() == "node_modules" || d.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if matches >= maxMatches {
			return filepath.SkipAll
		}
		// Skip obvious binaries by extension.
		switch strings.ToLower(filepath.Ext(path)) {
		case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".pdf", ".zip", ".gz", ".exe", ".dylib", ".so":
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		// Increase buffer for long lines.
		buf := make([]byte, 0, 64*1024)
		sc.Buffer(buf, 1024*1024)
		lineNo := 0
		relPath, _ := filepath.Rel(w.root, path)
		for sc.Scan() {
			lineNo++
			line := sc.Text()
			if re.FindStringIndex(line) != nil {
				fmt.Fprintf(&b, "%s:%d:%s\n", relPath, lineNo, line)
				matches++
				if matches >= maxMatches {
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		return b.String(), err
	}
	if matches == 0 {
		return "no matches", nil
	}
	return b.String(), nil
}
