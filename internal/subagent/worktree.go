package subagent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// safeID matches subagent ids we accept as worktree directory names.
var safeID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// GitWorktree isolates child agents using `git worktree add --detach`.
// Worktrees live under <repo>/.iomesh/worktrees/<id> by default.
type GitWorktree struct {
	// BaseDir is relative to the parent repo root (default ".iomesh/worktrees").
	BaseDir string
	// KeepOnSuccess leaves the worktree after a successful run (default false → cleanup).
	KeepOnSuccess bool
	// GitBinary overrides the git executable (default "git").
	GitBinary string
}

// NewGitWorktree returns a backend with defaults (keep worktrees after success).
func NewGitWorktree() *GitWorktree {
	return &GitWorktree{
		BaseDir:       filepath.Join(".iomesh", "worktrees"),
		GitBinary:     "git",
		KeepOnSuccess: true,
	}
}

// Available reports whether parentRoot is a git work tree and git is runnable.
func (g *GitWorktree) Available(ctx context.Context, parentRoot string) bool {
	if g == nil {
		return false
	}
	bin := g.gitBin()
	cmd := exec.CommandContext(ctx, bin, "-C", parentRoot, "rev-parse", "--is-inside-work-tree")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

// Create adds a detached worktree at <parentRoot>/<BaseDir>/<id>.
// cleanup removes the worktree (best-effort) when KeepOnSuccess is false,
// or always on failed create partial state.
func (g *GitWorktree) Create(ctx context.Context, parentRoot, id string) (path string, cleanup func() error, err error) {
	if g == nil {
		return "", nil, fmt.Errorf("worktree: nil backend")
	}
	if !safeID.MatchString(id) {
		return "", nil, fmt.Errorf("worktree: invalid id %q", id)
	}
	parentRoot, err = filepath.Abs(parentRoot)
	if err != nil {
		return "", nil, err
	}
	if !g.Available(ctx, parentRoot) {
		return "", nil, fmt.Errorf("worktree: %s is not a git work tree (isolation=worktree requires git)", parentRoot)
	}

	base := g.BaseDir
	if base == "" {
		base = filepath.Join(".iomesh", "worktrees")
	}
	// Prevent base escaping parent root.
	absBase := filepath.Join(parentRoot, base)
	absBase = filepath.Clean(absBase)
	if rel, err := filepath.Rel(parentRoot, absBase); err != nil || strings.HasPrefix(rel, "..") {
		return "", nil, fmt.Errorf("worktree: base_dir escapes repository root")
	}
	if err := os.MkdirAll(absBase, 0o755); err != nil {
		return "", nil, fmt.Errorf("worktree: mkdir base: %w", err)
	}

	path = filepath.Join(absBase, id)
	if _, err := os.Stat(path); err == nil {
		return "", nil, fmt.Errorf("worktree: path already exists: %s", path)
	}

	// Prefer HEAD; fall back to main/master if detached empty repo edge cases.
	ref := "HEAD"
	bin := g.gitBin()
	add := exec.CommandContext(ctx, bin, "-C", parentRoot, "worktree", "add", "--detach", path, ref)
	if out, err := add.CombinedOutput(); err != nil {
		// Retry with default branch name if HEAD unborn.
		for _, branch := range []string{"main", "master"} {
			add2 := exec.CommandContext(ctx, bin, "-C", parentRoot, "worktree", "add", "--detach", path, branch)
			if out2, err2 := add2.CombinedOutput(); err2 == nil {
				err = nil
				out = out2
				break
			}
			_ = out
		}
		if err != nil {
			_ = os.RemoveAll(path)
			return "", nil, fmt.Errorf("worktree add: %w: %s", err, strings.TrimSpace(string(out)))
		}
	}

	removed := false
	cleanup = func() error {
		if removed {
			return nil
		}
		removed = true
		cctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		rm := exec.CommandContext(cctx, bin, "-C", parentRoot, "worktree", "remove", "--force", path)
		if out, err := rm.CombinedOutput(); err != nil {
			// Fall back to directory delete + prune.
			_ = os.RemoveAll(path)
			prune := exec.CommandContext(cctx, bin, "-C", parentRoot, "worktree", "prune")
			_ = prune.Run()
			return fmt.Errorf("worktree remove: %w: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	}

	if g.KeepOnSuccess {
		// Still return cleanup for failure paths; callers may choose not to call it.
		return path, cleanup, nil
	}
	return path, cleanup, nil
}

func (g *GitWorktree) gitBin() string {
	if g != nil && g.GitBinary != "" {
		return g.GitBinary
	}
	return "git"
}

// LookupGit returns a GitWorktree if git is available in PATH, else NopWorktree.
func LookupGit() WorktreeBackend {
	if _, err := exec.LookPath("git"); err != nil {
		return NopWorktree{}
	}
	return NewGitWorktree()
}
