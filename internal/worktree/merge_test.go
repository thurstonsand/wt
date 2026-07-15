//go:build integration

package worktree

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thurstonsand/wt/internal/config"
	"github.com/thurstonsand/wt/internal/git"
	"github.com/thurstonsand/wt/internal/testutil"
)

func TestMergeSquash(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("checkout", "-b", "develop")

	mgr, err := NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{Name: "squash-test"})
	if err != nil {
		t.Fatal(err)
	}

	wtRepo := r.AtWorktree(wt.WorktreePath)
	content := "squash content"
	wtRepo.WriteFile("squash.txt", content)
	wtRepo.Run("add", "squash.txt")
	wtRepo.Run("commit", "-m", "commit 1")
	wtRepo.WriteFile("squash2.txt", "more content")
	wtRepo.Run("add", "squash2.txt")
	wtRepo.Run("commit", "-m", "commit 2")

	result, err := mgr.Merge(MergeOptions{Name: "squash-test", Mode: config.MergeModeSquash})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify result metadata
	if result.WorktreeName != "squash-test" {
		t.Errorf("WorktreeName = %q, want %q", result.WorktreeName, "squash-test")
	}
	if result.TargetBranch != "develop" {
		t.Errorf("TargetBranch = %q, want %q", result.TargetBranch, "develop")
	}
	if result.Mode != config.MergeModeSquash {
		t.Errorf("Mode = %v, want %v", result.Mode, config.MergeModeSquash)
	}
	if len(result.Commits) != 1 {
		t.Errorf("expected 1 commit in result, got %d", len(result.Commits))
	}
	if len(result.Commits) > 0 && !strings.Contains(result.Commits[0].Subject, "squash") {
		t.Errorf("squash commit should mention 'squash', got %q", result.Commits[0].Subject)
	}

	got, err := os.ReadFile(filepath.Join(r.Dir, "squash.txt"))
	if err != nil {
		t.Fatal("squash.txt should exist after merge")
	}
	if string(got) != content {
		t.Errorf("content = %q, want %q", got, content)
	}

	if mgr.Exists("squash-test") {
		t.Error("worktree should be deleted after successful merge")
	}

	log := r.Run("log", "--oneline", "develop")
	lines := strings.Split(strings.TrimSpace(log), "\n")
	if len(lines) != 2 {
		t.Errorf("squash should produce 1 commit (2 total with initial), got %d commits:\n%s", len(lines), log)
	}
	if !strings.Contains(log, "squash") {
		t.Errorf("squash commit message should mention 'squash', got:\n%s", log)
	}
}

func TestMergeStaged(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("checkout", "-b", "develop")

	mgr, err := NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{Name: "staged-test"})
	if err != nil {
		t.Fatal(err)
	}

	wtRepo := r.AtWorktree(wt.WorktreePath)
	content := "staged content"
	wtRepo.WriteFile("staged.txt", content)
	wtRepo.Run("add", "staged.txt")
	wtRepo.Run("commit", "-m", "staged commit")

	result, err := mgr.Merge(MergeOptions{Name: "staged-test", Mode: config.MergeModeStaged})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify result metadata
	if result.Mode != config.MergeModeStaged {
		t.Errorf("Mode = %v, want %v", result.Mode, config.MergeModeStaged)
	}
	if result.FileCount != 1 {
		t.Errorf("FileCount = %d, want 1", result.FileCount)
	}

	status := r.Run("status", "--porcelain")
	if status == "" {
		t.Error("parent should have staged changes")
	}

	if mgr.Exists("staged-test") {
		t.Error("worktree should be deleted after successful merge")
	}

	log := r.Run("log", "--oneline", "develop")
	lines := strings.Split(strings.TrimSpace(log), "\n")
	if len(lines) != 1 {
		t.Errorf("staged should add NO commits, got %d commits:\n%s", len(lines), log)
	}
}

func TestMergeRebase(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("checkout", "-b", "develop")

	mgr, err := NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{Name: "rebase-test"})
	if err != nil {
		t.Fatal(err)
	}

	wtRepo := r.AtWorktree(wt.WorktreePath)
	content := "rebase content"
	wtRepo.WriteFile("rebase.txt", content)
	wtRepo.Run("add", "rebase.txt")
	wtRepo.Run("commit", "-m", "rebase commit 1")
	wtRepo.WriteFile("rebase2.txt", "more content")
	wtRepo.Run("add", "rebase2.txt")
	wtRepo.Run("commit", "-m", "rebase commit 2")

	_, err = mgr.Merge(MergeOptions{Name: "rebase-test", Mode: config.MergeModeRebase})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(r.Dir, "rebase.txt"))
	if err != nil {
		t.Fatal("rebase.txt should exist after merge")
	}
	if string(got) != content {
		t.Errorf("content = %q, want %q", got, content)
	}

	if mgr.Exists("rebase-test") {
		t.Error("worktree should be deleted after successful merge")
	}

	log := r.Run("log", "--oneline", "develop")
	lines := strings.Split(strings.TrimSpace(log), "\n")
	if len(lines) != 3 {
		t.Errorf("rebase should preserve 2 commits (3 total with initial), got %d commits:\n%s", len(lines), log)
	}
	if !strings.Contains(log, "rebase commit 1") || !strings.Contains(log, "rebase commit 2") {
		t.Errorf("rebase should preserve original commit messages, got:\n%s", log)
	}
}

func TestMergeProtectedBranch(t *testing.T) {
	r := testutil.InitGitRepo(t)

	mgr, err := NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{Name: "protected-test"})
	if err != nil {
		t.Fatal(err)
	}

	wtRepo := r.AtWorktree(wt.WorktreePath)
	wtRepo.WriteFile("protected.txt", "content")
	wtRepo.Run("add", "protected.txt")
	wtRepo.Run("commit", "-m", "protected commit")

	_, err = mgr.Merge(MergeOptions{Name: "protected-test", Mode: config.MergeModeSquash})
	if !errors.Is(err, ErrProtectedBranch) {
		t.Errorf("expected ErrProtectedBranch, got: %v", err)
	}

	_, err = mgr.Merge(MergeOptions{Name: "protected-test", Mode: config.MergeModeSquash, Force: true})
	if err != nil {
		t.Fatalf("unexpected error with Force: %v", err)
	}
}

func TestMergeNotFound(t *testing.T) {
	r := testutil.InitGitRepo(t)

	mgr, err := NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = mgr.Merge(MergeOptions{Name: "nonexistent", Mode: config.MergeModeSquash})
	if !errors.Is(err, ErrWorktreeNotFound) {
		t.Errorf("expected ErrWorktreeNotFound, got: %v", err)
	}
}

func TestMergeNoChanges(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("checkout", "-b", "develop")

	mgr, err := NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = mgr.Fork(ForkOptions{Name: "no-changes"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = mgr.Merge(MergeOptions{Name: "no-changes", Mode: config.MergeModeSquash})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mgr.Exists("no-changes") {
		t.Error("worktree should be deleted even with no changes")
	}
}

func TestMergeTargetNotCheckedOut(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("checkout", "-b", "develop")

	mgr, err := NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{Name: "orphan-target"})
	if err != nil {
		t.Fatal(err)
	}

	wtRepo := r.AtWorktree(wt.WorktreePath)
	wtRepo.WriteFile("test.txt", "test")
	wtRepo.Run("add", "test.txt")
	wtRepo.Run("commit", "-m", "test")

	r.Run("checkout", "main")

	_, err = mgr.Merge(MergeOptions{Name: "orphan-target", Mode: config.MergeModeSquash, Force: true})
	if !errors.Is(err, ErrTargetNotCheckedOut) {
		t.Errorf("expected ErrTargetNotCheckedOut, got: %v", err)
	}
}

func TestMergeParentDirty(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("checkout", "-b", "develop")

	mgr, err := NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{Name: "dirty-parent"})
	if err != nil {
		t.Fatal(err)
	}

	wtRepo := r.AtWorktree(wt.WorktreePath)
	wtRepo.WriteFile("test.txt", "test")
	wtRepo.Run("add", "test.txt")
	wtRepo.Run("commit", "-m", "test")

	r.WriteFile("dirty.txt", "dirty")

	_, err = mgr.Merge(MergeOptions{Name: "dirty-parent", Mode: config.MergeModeSquash})
	if !errors.Is(err, ErrParentDirty) {
		t.Errorf("expected ErrParentDirty, got: %v", err)
	}

	_, err = mgr.Merge(MergeOptions{Name: "dirty-parent", Mode: config.MergeModeStaged})
	if err != nil {
		t.Fatalf("staged should allow dirty parent: %v", err)
	}
}

func TestMergeDeletesBranch(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("checkout", "-b", "develop")

	mgr, err := NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{Name: "delete-branch"})
	if err != nil {
		t.Fatal(err)
	}

	wtRepo := r.AtWorktree(wt.WorktreePath)
	wtRepo.WriteFile("test.txt", "content")
	wtRepo.Run("add", "test.txt")
	wtRepo.Run("commit", "-m", "test commit")

	out := r.Run("branch", "--list", "delete-branch")
	if out == "" {
		t.Fatal("branch should exist before merge")
	}

	_, err = mgr.Merge(MergeOptions{Name: "delete-branch", Mode: config.MergeModeSquash})
	if err != nil {
		t.Fatal(err)
	}

	out = r.Run("branch", "--list", "delete-branch")
	if strings.TrimSpace(out) != "" {
		t.Errorf("branch should be deleted after merge, got: %q", out)
	}
}

func TestMergeExternalWorktreeWithBase(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)

	extPath := filepath.Join(t.TempDir(), "external-wt")
	if err := g.WorktreeAdd(extPath, "external-branch", true, ""); err != nil {
		t.Fatalf("failed to create external worktree: %v", err)
	}

	extRepo := r.AtWorktree(extPath)
	extRepo.WriteFile("external.txt", "external content")
	extRepo.Run("add", "external.txt")
	extRepo.Run("commit", "-m", "external commit")

	mgr, err := NewManager(r.Dir, WithGit(g))
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Get("external-branch")
	if err != nil {
		t.Fatalf("expected to find external worktree: %v", err)
	}
	if wt.Parent != "" {
		t.Errorf("external worktree should have empty parent, got %q", wt.Parent)
	}

	_, err = mgr.Merge(MergeOptions{Name: "external-branch", Mode: config.MergeModeSquash})
	if !errors.Is(err, ErrParentUnknown) {
		t.Errorf("expected ErrParentUnknown without --base, got: %v", err)
	}

	_, err = mgr.Merge(MergeOptions{Name: "external-branch", Mode: config.MergeModeSquash, Base: "main", Force: true})
	if err != nil {
		t.Fatalf("merge with --base should succeed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(r.Dir, "external.txt"))
	if err != nil {
		t.Fatal("external.txt should exist after merge")
	}
	if string(content) != "external content" {
		t.Errorf("content = %q, want %q", content, "external content")
	}
}

func TestMergeDeletesDirectoryAndEmptyParent(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)
	r.Run("checkout", "-b", "develop")

	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{Name: "merge-cleanup"})
	if err != nil {
		t.Fatal(err)
	}

	repoDir := filepath.Dir(wt.WorktreePath)

	wtRepo := r.AtWorktree(wt.WorktreePath)
	wtRepo.CommitFile(t, "test.txt", "test commit")

	_, err = mgr.Merge(MergeOptions{Name: "merge-cleanup", Mode: config.MergeModeSquash})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(wt.WorktreePath); !os.IsNotExist(err) {
		t.Error("expected worktree directory to be removed")
	}
	if _, err := os.Stat(repoDir); !os.IsNotExist(err) {
		t.Error("expected empty repo parent directory to be removed")
	}
}

func TestMergeExternalWorktreeSkipsParentCleanup(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	extParent := filepath.Join(t.TempDir(), "my-project")
	extPath := filepath.Join(extParent, "my-wt")

	if err := g.WorktreeAdd(extPath, "ext-merge", true, ""); err != nil {
		t.Fatalf("failed to create external worktree: %v", err)
	}

	extRepo := r.AtWorktree(extPath)
	extRepo.CommitFile(t, "ext.txt", "external commit")

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = mgr.Merge(MergeOptions{
		Name: "ext-merge", Mode: config.MergeModeSquash, Base: "main", Force: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(extPath); !os.IsNotExist(err) {
		t.Error("expected external worktree directory to be removed")
	}
	if _, err := os.Stat(extParent); os.IsNotExist(err) {
		t.Error("external parent directory should NOT be removed")
	}
}

func TestMergeWorktreeToWorktree(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)

	r.Run("checkout", "-b", "develop")
	r.CommitFile(t, "develop-base.txt", "develop base")

	mgr, err := NewManager(r.Dir, WithGit(g))
	if err != nil {
		t.Fatal(err)
	}

	wtParent, err := mgr.Fork(ForkOptions{Name: "parent-wt"})
	if err != nil {
		t.Fatal(err)
	}

	parentRepo := r.AtWorktree(wtParent.WorktreePath)
	parentRepo.WriteFile("parent.txt", "parent content")
	parentRepo.Run("add", "parent.txt")
	parentRepo.Run("commit", "-m", "parent commit")

	wtChild, err := mgr.Fork(ForkOptions{Name: "child-wt", Base: "parent-wt"})
	if err != nil {
		t.Fatal(err)
	}

	childRepo := r.AtWorktree(wtChild.WorktreePath)
	childRepo.WriteFile("child.txt", "child content")
	childRepo.Run("add", "child.txt")
	childRepo.Run("commit", "-m", "child commit")

	_, err = mgr.Merge(MergeOptions{Name: "child-wt", Mode: config.MergeModeSquash})
	if err != nil {
		t.Fatalf("merge child to parent worktree should succeed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(wtParent.WorktreePath, "child.txt"))
	if err != nil {
		t.Fatal("child.txt should exist in parent worktree after merge")
	}
	if string(content) != "child content" {
		t.Errorf("content = %q, want %q", content, "child content")
	}

	if mgr.Exists("child-wt") {
		t.Error("child worktree should be deleted after merge")
	}
}
