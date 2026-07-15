//go:build integration

package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thurstonsand/wt/internal/testutil"
	"github.com/thurstonsand/wt/internal/worktree"
)

func TestMergeSquash(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("checkout", "-b", "develop")

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(worktree.ForkOptions{Name: "feature"})
	if err != nil {
		t.Fatal(err)
	}

	content := "feature work"
	wtRepo := r.AtWorktree(wt.WorktreePath)
	wtRepo.WriteFile("feature.txt", content)
	wtRepo.Run("add", "feature.txt")
	wtRepo.Run("commit", "-m", "add feature")

	buf, err := runCmd("merge", "--squash", "feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Merged \"") {
		t.Errorf("expected 'Merged, got: %s", out)
	}

	got, err := os.ReadFile(filepath.Join(r.Dir, "feature.txt"))
	if err != nil {
		t.Fatal("feature.txt should exist after merge")
	}
	if string(got) != content {
		t.Errorf("content = %q, want %q", got, content)
	}

	if mgr.Exists("feature") {
		t.Error("worktree should be deleted after successful merge")
	}
}

func TestMergeStaged(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("checkout", "-b", "develop")

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(worktree.ForkOptions{Name: "staged-feature"})
	if err != nil {
		t.Fatal(err)
	}

	content := "staged content"
	wtRepo := r.AtWorktree(wt.WorktreePath)
	wtRepo.WriteFile("staged.txt", content)
	wtRepo.Run("add", "staged.txt")
	wtRepo.Run("commit", "-m", "add staged")

	buf, err := runCmd("merge", "--staged", "staged-feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Staged changes from") {
		t.Errorf("expected 'Staged changes from', got: %s", out)
	}

	status := r.Run("status", "--porcelain")
	if !strings.Contains(status, "staged.txt") {
		t.Error("staged.txt should be in staged changes")
	}
}

func TestMergeProtectedBranchDefaultsToStaged(t *testing.T) {
	r := testutil.InitGitRepo(t)

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(worktree.ForkOptions{Name: "protected-test"})
	if err != nil {
		t.Fatal(err)
	}

	wtRepo := r.AtWorktree(wt.WorktreePath)
	wtRepo.WriteFile("protected.txt", "protected content")
	wtRepo.Run("add", "protected.txt")
	wtRepo.Run("commit", "-m", "add protected")

	_, err = runCmd("merge", "protected-test")
	if err == nil {
		t.Fatal("expected error when merging to protected branch without --staged or -f")
	}
	if !errors.Is(err, worktree.ErrProtectedBranch) {
		t.Errorf("expected ErrProtectedBranch, got: %v", err)
	}
}

func TestMergeProtectedBranchWithForce(t *testing.T) {
	r := testutil.InitGitRepo(t)

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(worktree.ForkOptions{Name: "force-test"})
	if err != nil {
		t.Fatal(err)
	}

	wtRepo := r.AtWorktree(wt.WorktreePath)
	wtRepo.WriteFile("force.txt", "force content")
	wtRepo.Run("add", "force.txt")
	wtRepo.Run("commit", "-m", "add force")

	buf, err := runCmd("merge", "-f", "force-test")
	if err != nil {
		t.Fatalf("unexpected error with -f: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Merged \"") {
		t.Errorf("expected 'Merged, got: %s", out)
	}

	if mgr.Exists("force-test") {
		t.Error("worktree should be deleted after successful merge")
	}
}

func TestMergeMutuallyExclusiveFlags(t *testing.T) {
	testutil.InitGitRepo(t)

	tests := []struct {
		name string
		args []string
	}{
		{"squash_and_rebase", []string{"merge", "--squash", "--rebase", "test"}},
		{"squash_and_staged", []string{"merge", "--squash", "--staged", "test"}},
		{"rebase_and_staged", []string{"merge", "--rebase", "--staged", "test"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runCmd(tc.args...)
			if err == nil {
				t.Fatal("expected error for mutually exclusive flags")
			}
			if !errors.Is(err, ErrInvalidFlagCombination) {
				t.Errorf("expected ErrInvalidFlagCombination, got: %v", err)
			}
		})
	}
}

func TestMergeImplicitNotInWorktree(t *testing.T) {
	testutil.InitGitRepo(t)

	_, err := runCmd("merge")
	if err == nil {
		t.Fatal("expected error when no argument and not in a worktree")
	}
	if !errors.Is(err, worktree.ErrNotInWorktree) {
		t.Errorf("expected ErrNotInWorktree, got: %v", err)
	}
}

func TestMergeImplicitCurrentWorktree(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("checkout", "-b", "develop")

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(worktree.ForkOptions{Name: "implicit-merge"})
	if err != nil {
		t.Fatal(err)
	}

	wtRepo := r.AtWorktree(wt.WorktreePath)
	wtRepo.WriteFile("implicit.txt", "implicit content")
	wtRepo.Run("add", "implicit.txt")
	wtRepo.Run("commit", "-m", "add implicit")

	if err := os.Chdir(wt.WorktreePath); err != nil {
		t.Fatal(err)
	}

	buf, err := runCmd("merge", "--squash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Merged \"") {
		t.Errorf("expected 'Merged', got: %s", out)
	}

	got, err := os.ReadFile(filepath.Join(r.Dir, "implicit.txt"))
	if err != nil {
		t.Fatal("implicit.txt should exist after merge")
	}
	if string(got) != "implicit content" {
		t.Errorf("content = %q, want %q", got, "implicit content")
	}
}

func TestMergeNotFound(t *testing.T) {
	testutil.InitGitRepo(t)

	_, err := runCmd("merge", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent worktree")
	}
	if !errors.Is(err, worktree.ErrWorktreeNotFound) {
		t.Errorf("expected ErrWorktreeNotFound, got: %v", err)
	}
}

func TestMergeRebase(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("checkout", "-b", "develop")
	r.CommitFile(t, "develop.txt", "develop commit")

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(worktree.ForkOptions{Name: "rebase-test"})
	if err != nil {
		t.Fatal(err)
	}

	content := "rebase content"
	wtRepo := r.AtWorktree(wt.WorktreePath)
	wtRepo.WriteFile("rebase.txt", content)
	wtRepo.Run("add", "rebase.txt")
	wtRepo.Run("commit", "-m", "add rebase")

	buf, err := runCmd("merge", "--rebase", "rebase-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Merged \"") {
		t.Errorf("expected 'Merged, got: %s", out)
	}

	got, err := os.ReadFile(filepath.Join(r.Dir, "rebase.txt"))
	if err != nil {
		t.Fatal("rebase.txt should exist after merge")
	}
	if string(got) != content {
		t.Errorf("content = %q, want %q", got, content)
	}
}

func TestMergeTargetNotCheckedOut(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("checkout", "-b", "develop")

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(worktree.ForkOptions{Name: "orphan-target"})
	if err != nil {
		t.Fatal(err)
	}

	wtRepo := r.AtWorktree(wt.WorktreePath)
	wtRepo.WriteFile("test.txt", "test")
	wtRepo.Run("add", "test.txt")
	wtRepo.Run("commit", "-m", "test commit")

	r.Run("checkout", "main")

	_, err = runCmd("merge", "orphan-target")
	if err == nil {
		t.Fatal("expected error when target branch not checked out")
	}
	if !errors.Is(err, worktree.ErrTargetNotCheckedOut) {
		t.Errorf("expected ErrTargetNotCheckedOut, got: %v", err)
	}
}

func TestMergeParentDirty(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("checkout", "-b", "develop")

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(worktree.ForkOptions{Name: "dirty-parent"})
	if err != nil {
		t.Fatal(err)
	}

	wtRepo := r.AtWorktree(wt.WorktreePath)
	wtRepo.WriteFile("test.txt", "test")
	wtRepo.Run("add", "test.txt")
	wtRepo.Run("commit", "-m", "test commit")

	r.WriteFile("dirty.txt", "dirty")

	_, err = runCmd("merge", "--squash", "dirty-parent")
	if err == nil {
		t.Fatal("expected error when parent is dirty")
	}
	if !errors.Is(err, worktree.ErrParentDirty) {
		t.Errorf("expected ErrParentDirty, got: %v", err)
	}
}

func TestMergeStagedAllowsDirtyParent(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("checkout", "-b", "develop")

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(worktree.ForkOptions{Name: "staged-dirty"})
	if err != nil {
		t.Fatal(err)
	}

	wtRepo := r.AtWorktree(wt.WorktreePath)
	wtRepo.WriteFile("test.txt", "test")
	wtRepo.Run("add", "test.txt")
	wtRepo.Run("commit", "-m", "test commit")

	r.WriteFile("dirty.txt", "dirty")

	buf, err := runCmd("merge", "--staged", "staged-dirty")
	if err != nil {
		t.Fatalf("--staged should allow dirty parent: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Staged changes from") {
		t.Errorf("expected 'Staged changes from', got: %s", out)
	}
}
