// Package workspace abstracts host filesystem operations rooted at a project
// directory (Grok Build workspace crate equivalent).
//
// Security: all paths are jail-checked against the workspace root. Symlink
// targets that resolve outside the root are rejected. Read size is capped.
package workspace

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/iome-sh/iomesh-tui/internal/security"
)

// DefaultMaxReadBytes caps a single ReadFile call (2 MiB).
const DefaultMaxReadBytes = 2 << 20

// Workspace is a project-rooted filesystem view.
type Workspace struct {
	root         string
	maxReadBytes int64
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
	// Resolve symlinks on the root itself so jail comparisons are consistent.
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("workspace: not a directory: %s", abs)
	}
	return &Workspace{root: abs, maxReadBytes: DefaultMaxReadBytes}, nil
}

// Root returns the absolute workspace path.
func (w *Workspace) Root() string { return w.root }

// SetMaxReadBytes overrides the per-read size cap (tests / config).
func (w *Workspace) SetMaxReadBytes(n int64) {
	if n > 0 {
		w.maxReadBytes = n
	}
}

// Resolve cleans a path and ensures the final target stays under root.
// Symlink escape is blocked by EvalSymlinks + re-check when the path exists.
func (w *Workspace) Resolve(rel string) (string, error) {
	if err := security.ValidateRelPath(rel); err != nil {
		return "", err
	}
	if rel == "" {
		rel = "."
	}

	var abs string
	if filepath.IsAbs(rel) {
		// Absolute paths only allowed if they already lie under the root.
		// Normalize via EvalSymlinks when possible so /var vs /private/var matches Open().
		cleaned, err := filepath.Abs(rel)
		if err != nil {
			return "", err
		}
		if resolved, err := filepath.EvalSymlinks(cleaned); err == nil {
			cleaned = resolved
		}
		abs = cleaned
	} else {
		joined := filepath.Join(w.root, filepath.Clean(rel))
		var err error
		abs, err = filepath.Abs(joined)
		if err != nil {
			return "", err
		}
	}

	if !security.PathUnderRoot(w.root, abs) {
		// Retry with EvalSymlinks for paths that exist (macOS /var → /private/var).
		if resolved, err := filepath.EvalSymlinks(abs); err == nil {
			abs = resolved
		}
		if !security.PathUnderRoot(w.root, abs) {
			return "", fmt.Errorf("path escapes workspace: %s", rel)
		}
	}

	// If the path exists, resolve symlinks and re-check the jail.
	if fi, err := os.Lstat(abs); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(abs)
			if err == nil {
				if !security.PathUnderRoot(w.root, resolved) {
					return "", fmt.Errorf("path escapes workspace via symlink: %s", rel)
				}
				abs = resolved
			}
		} else if fi.IsDir() {
			if resolved, err := filepath.EvalSymlinks(abs); err == nil {
				if !security.PathUnderRoot(w.root, resolved) {
					return "", fmt.Errorf("path escapes workspace via symlink: %s", rel)
				}
				abs = resolved
			}
		}
	}

	// Parent directory: only re-check when the parent itself is under the root.
	// (The parent of the workspace root is outside by definition.)
	parent := filepath.Dir(abs)
	if parent != abs && security.PathUnderRoot(w.root, parent) {
		if resolved, err := filepath.EvalSymlinks(parent); err == nil {
			if !security.PathUnderRoot(w.root, resolved) {
				return "", fmt.Errorf("path escapes workspace via parent symlink: %s", rel)
			}
		}
	}

	return abs, nil
}

// ReadFile returns file contents; offset/limit are 1-based line numbers (0 = all).
func (w *Workspace) ReadFile(rel string, offset, limit int) (string, error) {
	path, err := w.Resolve(rel)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if fi.IsDir() {
		return "", fmt.Errorf("read_file: is a directory")
	}
	if fi.Size() > w.maxReadBytes {
		return "", fmt.Errorf("read_file: file exceeds max size %d bytes (got %d)", w.maxReadBytes, fi.Size())
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// LimitReader as defense in depth if size races.
	data, err := io.ReadAll(io.LimitReader(f, w.maxReadBytes+1))
	if err != nil {
		return "", err
	}
	if int64(len(data)) > w.maxReadBytes {
		return "", fmt.Errorf("read_file: file exceeds max size %d bytes", w.maxReadBytes)
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
	if int64(len(content)) > w.maxReadBytes*4 {
		return fmt.Errorf("write_file: content exceeds max size")
	}
	path, err := w.Resolve(rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	// O_NOFOLLOW when available would be ideal; Go's OpenFile doesn't expose it
	// portably. Resolve already blocks symlink targets outside root.
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
	if len(pattern) > 1024 {
		return "", fmt.Errorf("grep: pattern too long")
	}
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
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" || name == ".iomesh" {
				return filepath.SkipDir
			}
			// Skip if directory resolves outside root (symlink dir).
			if resolved, err := filepath.EvalSymlinks(path); err == nil {
				if !security.PathUnderRoot(w.root, resolved) {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if matches >= maxMatches {
			return filepath.SkipAll
		}
		// Skip obvious binaries by extension.
		switch strings.ToLower(filepath.Ext(path)) {
		case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".pdf", ".zip", ".gz", ".exe", ".dylib", ".so", ".bin", ".o", ".a":
			return nil
		}
		// Skip large files.
		if fi, err := d.Info(); err == nil && fi.Size() > w.maxReadBytes {
			return nil
		}
		// Ensure file is still under root after symlink resolution.
		if resolved, err := filepath.EvalSymlinks(path); err == nil {
			if !security.PathUnderRoot(w.root, resolved) {
				return nil
			}
			path = resolved
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
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
