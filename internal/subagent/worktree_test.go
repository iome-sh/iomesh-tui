package subagent

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func initTestRepo(t *testing.T) string {
	t.Helper()
	requireGit(t)
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=t@example.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=t@example.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
	run("git", "init", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", "README.md")
	run("git", "commit", "-m", "init")
	return dir
}

func TestGitWorktree_CreateAndCleanup(t *testing.T) {
	repo := initTestRepo(t)
	gw := NewGitWorktree()
	gw.KeepOnSuccess = false

	ctx := context.Background()
	if !gw.Available(ctx, repo) {
		t.Fatal("expected git repo available")
	}

	path, cleanup, err := gw.Create(ctx, repo, "sa-test-1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(path, filepath.Join(".iomesh", "worktrees", "sa-test-1")) {
		t.Fatalf("path=%s", path)
	}
	// Worktree has the committed file.
	if _, err := os.Stat(filepath.Join(path, "README.md")); err != nil {
		t.Fatal(err)
	}
	// Write only in worktree.
	if err := os.WriteFile(filepath.Join(path, "child-only.txt"), []byte("iso\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repo, "child-only.txt")); err == nil {
		t.Fatal("parent should not see child-only.txt")
	}

	if err := cleanup(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("worktree dir should be gone: %v", err)
	}
}

func TestGitWorktree_InvalidID(t *testing.T) {
	repo := initTestRepo(t)
	gw := NewGitWorktree()
	_, _, err := gw.Create(context.Background(), repo, "../escape")
	if err == nil {
		t.Fatal("expected invalid id")
	}
}

func TestGitWorktree_NotARepo(t *testing.T) {
	requireGit(t)
	gw := NewGitWorktree()
	dir := t.TempDir()
	_, _, err := gw.Create(context.Background(), dir, "sa-x")
	if err == nil {
		t.Fatal("expected not a git work tree")
	}
}

func TestSpawn_IsolationWorktree(t *testing.T) {
	repo := initTestRepo(t)
	var sawWS string
	m := NewManager(Config{
		Enabled: true, Workspace: repo, WorktreeAutoRemove: false,
	}, func(ctx context.Context, sp SpawnParams) (Runner, error) {
		sawWS = sp.Workspace
		return &fakeRunner{summary: "worked in " + sp.Workspace}, nil
	}, nil)
	m.SetWorktreeBackend(NewGitWorktree())

	res, err := m.Spawn(context.Background(), Spec{
		Prompt:       "edit something",
		Description:  "iso-test",
		SubagentType: TypeGeneralPurpose,
		Isolation:    IsolationWorktree,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusCompleted {
		t.Fatalf("%+v", res)
	}
	if res.WorktreePath == "" {
		t.Fatal("expected worktree path on result")
	}
	if sawWS != res.WorktreePath {
		t.Fatalf("factory workspace=%q result=%q", sawWS, res.WorktreePath)
	}
	if !strings.Contains(res.Summary, "worked in") {
		t.Fatalf("summary=%q", res.Summary)
	}
	// Worktree kept (AutoRemove false)
	if _, err := os.Stat(res.WorktreePath); err != nil {
		t.Fatal("worktree should remain for inspection")
	}
	// Cleanup manually for test hygiene
	_ = exec.Command("git", "-C", repo, "worktree", "remove", "--force", res.WorktreePath).Run()
}

func TestSpawn_IsolationWorktree_AutoRemove(t *testing.T) {
	repo := initTestRepo(t)
	m := NewManager(Config{
		Enabled: true, Workspace: repo, WorktreeAutoRemove: true,
	}, func(ctx context.Context, sp SpawnParams) (Runner, error) {
		return &fakeRunner{summary: "ok"}, nil
	}, nil)
	m.SetWorktreeBackend(NewGitWorktree())

	res, err := m.Spawn(context.Background(), Spec{
		Prompt: "x", Isolation: IsolationWorktree, SubagentType: TypeExplore,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusCompleted {
		t.Fatalf("%+v", res)
	}
	// Path cleared after auto-remove
	time.Sleep(20 * time.Millisecond)
	if res.WorktreePath != "" {
		// resultOf after update may clear path
		if _, err := os.Stat(res.WorktreePath); err == nil {
			// may still briefly exist; re-get
		}
	}
	final, _ := m.Get(res.ID)
	if final.WorktreePath != "" {
		if _, err := os.Stat(final.WorktreePath); err == nil {
			t.Fatalf("expected auto-removed worktree, still at %s", final.WorktreePath)
		}
	}
}

func TestLookupGit(t *testing.T) {
	b := LookupGit()
	if b == nil {
		t.Fatal()
	}
}
