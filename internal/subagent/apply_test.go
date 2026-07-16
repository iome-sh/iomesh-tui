package subagent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyWorktree_CopiesIntoParent(t *testing.T) {
	repo := initTestRepo(t)
	gw := NewGitWorktree()
	ctx := context.Background()

	path, cleanup, err := gw.Create(ctx, repo, "sa-apply-1")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cleanup() }()

	// Change file in worktree only.
	if err := os.WriteFile(filepath.Join(path, "README.md"), []byte("from-worktree\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "newfile.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	diff, err := gw.Diff(ctx, repo, "sa-apply-1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "newfile.go") && !strings.Contains(diff, "README") {
		t.Fatalf("diff missing changes: %s", diff)
	}

	res, err := gw.Apply(ctx, repo, "sa-apply-1", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Applied) < 1 {
		t.Fatalf("expected applied files: %+v", res)
	}

	got, err := os.ReadFile(filepath.Join(repo, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "from-worktree\n" {
		t.Fatalf("parent README=%q", got)
	}
	if _, err := os.Stat(filepath.Join(repo, "newfile.go")); err != nil {
		t.Fatal("newfile should exist in parent")
	}
}

func TestApplyWorktree_RemoveAfter(t *testing.T) {
	repo := initTestRepo(t)
	gw := NewGitWorktree()
	ctx := context.Background()
	path, _, err := gw.Create(ctx, repo, "sa-apply-rm")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "x.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := gw.Apply(ctx, repo, "sa-apply-rm", true)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Removed {
		t.Fatal("expected removed")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("worktree should be gone: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, "x.txt")); err != nil {
		t.Fatal("file should remain in parent")
	}
}

func TestApplyWorktree_PathJail(t *testing.T) {
	repo := initTestRepo(t)
	gw := NewGitWorktree()
	_, err := gw.ResolveWorktreePath(repo, "../escape")
	if err == nil {
		t.Fatal("expected escape refusal")
	}
}

func TestListWorktrees(t *testing.T) {
	repo := initTestRepo(t)
	gw := NewGitWorktree()
	ctx := context.Background()
	path, cleanup, err := gw.Create(ctx, repo, "sa-list-1")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cleanup() }()
	list, err := gw.List(repo)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range list {
		if w.ID == "sa-list-1" && w.Path == path {
			found = true
		}
	}
	if !found {
		t.Fatalf("list=%+v", list)
	}
}

func TestManager_ApplyViaRegistryID(t *testing.T) {
	repo := initTestRepo(t)
	m := NewManager(Config{Enabled: true, Workspace: repo}, func(ctx context.Context, sp SpawnParams) (Runner, error) {
		// Write in the worktree workspace.
		_ = os.WriteFile(filepath.Join(sp.Workspace, "from-child.txt"), []byte("hi\n"), 0o644)
		return &fakeRunner{summary: "wrote from-child.txt"}, nil
	}, nil)
	m.SetWorktreeBackend(NewGitWorktree())

	res, err := m.Spawn(context.Background(), Spec{
		Prompt: "write", Isolation: IsolationWorktree, SubagentType: TypeGeneralPurpose,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.WorktreePath == "" {
		t.Fatal("need worktree path")
	}
	// Status may not see untracked until we apply — write already done.
	// Force status by applying.
	ar, err := m.ApplyWorktree(context.Background(), res.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(ar.Applied) == 0 && len(ar.Skipped) > 0 && ar.Skipped[0] == "(no changes)" {
		// git status might not see file if not... it should see untracked
		t.Fatalf("apply result=%+v", ar)
	}
	if _, err := os.Stat(filepath.Join(repo, "from-child.txt")); err != nil {
		// if apply used porcelain and file exists untracked it should copy
		t.Fatalf("parent missing file: %v apply=%+v", err, ar)
	}
	_ = m.RemoveWorktree(context.Background(), res.ID)
}
