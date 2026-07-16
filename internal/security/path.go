package security

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// ValidateRelPath checks a workspace-relative path for control characters and
// obvious escape attempts before Join. Absolute paths are allowed only when
// they will later be re-checked against the workspace root.
func ValidateRelPath(rel string) error {
	if !utf8.ValidString(rel) {
		return fmt.Errorf("path: not valid UTF-8")
	}
	if strings.ContainsRune(rel, 0) {
		return fmt.Errorf("path: contains NUL byte")
	}
	// Reject Windows-style volume escapes when running cross-platform.
	if len(rel) >= 2 && rel[1] == ':' {
		// e.g. C:\ — treat as absolute foreign root
		return fmt.Errorf("path: drive-letter paths not allowed")
	}
	return nil
}

// PathUnderRoot reports whether abs is equal to or nested under root.
// Both paths should be cleaned absolute paths. Uses filepath.Rel semantics.
func PathUnderRoot(root, abs string) bool {
	root = filepath.Clean(root)
	abs = filepath.Clean(abs)
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}
