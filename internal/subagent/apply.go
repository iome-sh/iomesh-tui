package subagent

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/security"
)

// ApplyResult summarizes merging a child worktree into the parent workspace.
type ApplyResult struct {
	WorktreePath string   `json:"worktree_path"`
	Applied      []string `json:"applied,omitempty"`   // created/updated relative paths
	Deleted      []string `json:"deleted,omitempty"`   // removed relative paths
	Skipped      []string `json:"skipped,omitempty"`   // refused or empty
	Removed      bool     `json:"removed"`             // worktree removed after apply
	DiffStat     string   `json:"diff_stat,omitempty"` // pre-apply shortstat when available
	Error        string   `json:"error,omitempty"`     // set by ApplyMany on per-item failure
}

// WorktreeInfo is a listed isolation worktree directory.
type WorktreeInfo struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

// ResolveWorktreePath maps a subagent id or absolute/relative path to a worktree dir
// under parentRoot/.iomesh/worktrees (or configured base).
func (g *GitWorktree) ResolveWorktreePath(parentRoot, idOrPath string) (string, error) {
	if g == nil {
		return "", fmt.Errorf("worktree: nil backend")
	}
	parentRoot, err := filepath.Abs(parentRoot)
	if err != nil {
		return "", err
	}
	base := g.BaseDir
	if base == "" {
		base = filepath.Join(".iomesh", "worktrees")
	}
	absBase := filepath.Clean(filepath.Join(parentRoot, base))
	if !security.PathUnderRoot(parentRoot, absBase) {
		return "", fmt.Errorf("worktree: base escapes repository root")
	}

	candidate := strings.TrimSpace(idOrPath)
	if candidate == "" {
		return "", fmt.Errorf("worktree: empty id or path")
	}
	// Bare id → under base.
	if safeID.MatchString(candidate) && !strings.Contains(candidate, string(filepath.Separator)) {
		candidate = filepath.Join(absBase, candidate)
	} else if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(parentRoot, candidate)
	}
	abs, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	if !security.PathUnderRoot(absBase, abs) {
		return "", fmt.Errorf("worktree: path escapes worktree base: %s", idOrPath)
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("worktree: %w", err)
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("worktree: not a directory: %s", abs)
	}
	return abs, nil
}

// Diff returns porcelain status + shortstat for a worktree vs its HEAD.
func (g *GitWorktree) Diff(ctx context.Context, parentRoot, idOrPath string) (string, error) {
	path, err := g.ResolveWorktreePath(parentRoot, idOrPath)
	if err != nil {
		return "", err
	}
	bin := g.gitBin()
	var b strings.Builder
	status := exec.CommandContext(ctx, bin, "-C", path, "status", "--porcelain", "-uall")
	out, err := status.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("worktree status: %w: %s", err, strings.TrimSpace(string(out)))
	}
	b.WriteString("## status\n")
	if len(out) == 0 {
		b.WriteString("(clean)\n")
	} else {
		b.Write(out)
		if !strings.HasSuffix(string(out), "\n") {
			b.WriteByte('\n')
		}
	}
	diff := exec.CommandContext(ctx, bin, "-C", path, "diff", "--stat", "HEAD")
	// Untracked won't show in diff HEAD; also diff staged if any.
	if dout, err := diff.CombinedOutput(); err == nil && len(dout) > 0 {
		b.WriteString("\n## diff --stat HEAD\n")
		b.Write(dout)
	}
	// Include untracked file list already in status.
	return b.String(), nil
}

// Apply copies changed files from the worktree into the parent workspace root.
// Only paths under both trees are allowed (path jail). Optional removeAfter deletes the worktree.
func (g *GitWorktree) Apply(ctx context.Context, parentRoot, idOrPath string, removeAfter bool) (ApplyResult, error) {
	res := ApplyResult{}
	parentRoot, err := filepath.Abs(parentRoot)
	if err != nil {
		return res, err
	}
	wt, err := g.ResolveWorktreePath(parentRoot, idOrPath)
	if err != nil {
		return res, err
	}
	res.WorktreePath = wt

	if diff, err := g.Diff(ctx, parentRoot, wt); err == nil {
		// Keep short: first 4k
		if len(diff) > 4096 {
			res.DiffStat = diff[:4096] + "\n…[truncated]"
		} else {
			res.DiffStat = diff
		}
	}

	bin := g.gitBin()
	status := exec.CommandContext(ctx, bin, "-C", wt, "status", "--porcelain", "-uall")
	out, err := status.CombinedOutput()
	if err != nil {
		return res, fmt.Errorf("worktree status: %w: %s", err, strings.TrimSpace(string(out)))
	}
	if len(strings.TrimSpace(string(out))) == 0 {
		res.Skipped = append(res.Skipped, "(no changes)")
		if removeAfter {
			// Best-effort cleanup: parallel ApplyMany can race on git worktree
			// metadata; apply itself already succeeded with no file changes.
			if err := g.Remove(ctx, parentRoot, wt); err == nil {
				res.Removed = true
			}
		}
		return res, nil
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// porcelain: XY PATH or XY ORIG -> PATH for renames
		if len(line) < 4 {
			res.Skipped = append(res.Skipped, line)
			continue
		}
		xy := line[:2]
		rest := strings.TrimSpace(line[3:])
		var rel string
		if strings.Contains(rest, " -> ") {
			parts := strings.SplitN(rest, " -> ", 2)
			rel = strings.Trim(parts[len(parts)-1], `"`)
		} else {
			rel = strings.Trim(rest, `"`)
		}
		rel = filepath.Clean(rel)
		if rel == "." || strings.HasPrefix(rel, "..") {
			res.Skipped = append(res.Skipped, rel+": refused path")
			continue
		}

		src := filepath.Join(wt, rel)
		dst := filepath.Join(parentRoot, rel)
		if !security.PathUnderRoot(wt, src) || !security.PathUnderRoot(parentRoot, dst) {
			res.Skipped = append(res.Skipped, rel+": path jail")
			continue
		}

		// Deletion: D in either index or worktree column.
		if xy[0] == 'D' || xy[1] == 'D' {
			if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
				res.Skipped = append(res.Skipped, rel+": delete failed: "+err.Error())
				continue
			}
			res.Deleted = append(res.Deleted, rel)
			continue
		}

		// Copy file (or skip dirs — recurse simple files only).
		info, err := os.Lstat(src)
		if err != nil {
			res.Skipped = append(res.Skipped, rel+": "+err.Error())
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			res.Skipped = append(res.Skipped, rel+": symlink skipped")
			continue
		}
		if info.IsDir() {
			res.Skipped = append(res.Skipped, rel+": directory skipped (file changes only)")
			continue
		}
		if err := copyFile(src, dst, info.Mode()); err != nil {
			res.Skipped = append(res.Skipped, rel+": "+err.Error())
			continue
		}
		res.Applied = append(res.Applied, rel)
	}

	if removeAfter {
		// Best-effort cleanup after a successful apply. Parallel ApplyMany can
		// race git worktree remove (Invalid path / already gone); do not fail
		// the apply or mark ApplyResult.Error when files were already copied.
		if err := g.Remove(ctx, parentRoot, wt); err == nil {
			res.Removed = true
		}
	}
	return res, nil
}

// Remove deletes a worktree via git worktree remove --force.
func (g *GitWorktree) Remove(ctx context.Context, parentRoot, idOrPath string) error {
	parentRoot, err := filepath.Abs(parentRoot)
	if err != nil {
		return err
	}
	path, err := g.ResolveWorktreePath(parentRoot, idOrPath)
	if err != nil {
		return err
	}
	bin := g.gitBin()
	cctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	rm := exec.CommandContext(cctx, bin, "-C", parentRoot, "worktree", "remove", "--force", path)
	if out, err := rm.CombinedOutput(); err != nil {
		_ = os.RemoveAll(path)
		_ = exec.CommandContext(cctx, bin, "-C", parentRoot, "worktree", "prune").Run()
		return fmt.Errorf("worktree remove: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// List returns isolation worktrees under the configured base directory.
func (g *GitWorktree) List(parentRoot string) ([]WorktreeInfo, error) {
	parentRoot, err := filepath.Abs(parentRoot)
	if err != nil {
		return nil, err
	}
	base := g.BaseDir
	if base == "" {
		base = filepath.Join(".iomesh", "worktrees")
	}
	absBase := filepath.Clean(filepath.Join(parentRoot, base))
	if !security.PathUnderRoot(parentRoot, absBase) {
		return nil, fmt.Errorf("worktree: base escapes root")
	}
	entries, err := os.ReadDir(absBase)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []WorktreeInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if !safeID.MatchString(e.Name()) {
			continue
		}
		out = append(out, WorktreeInfo{
			ID:   e.Name(),
			Path: filepath.Join(absBase, e.Name()),
		})
	}
	return out, nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	// Avoid following symlinks at destination.
	tmp := dst + ".iomesh-tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, dst)
}
