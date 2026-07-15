//go:build integration

package cmd

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/thurstonsand/wt/internal/testutil"
	"github.com/thurstonsand/wt/internal/worktree"
)

func TestRebaseBasic(t *testing.T) {
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

	wtRepo := r.AtWorktree(wt.WorktreePath)
	wtRepo.WriteFile("feature.txt", "feature work")
	wtRepo.Run("add", "feature.txt")
	wtRepo.Run("commit", "-m", "add feature")

	r.CommitFile(t, "develop-update.txt", "develop updated after fork")

	buf, err := runCmd("rebase", "feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Rebased worktree") {
		t.Errorf("expected 'Rebased worktree', got: %s", out)
	}

	log := wtRepo.Run("log", "--oneline", "-3")
	if !strings.Contains(log, "add feature") {
		t.Error("worktree should still have feature commit")
	}
	if !strings.Contains(log, "develop updated after fork") {
		t.Error("worktree should have parent's new commit after rebase")
	}
}

func TestRebaseNoChanges(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("checkout", "-b", "develop")

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(worktree.ForkOptions{Name: "no-changes"})
	if err != nil {
		t.Fatal(err)
	}

	wtRepo := r.AtWorktree(wt.WorktreePath)
	wtRepo.WriteFile("feature.txt", "feature work")
	wtRepo.Run("add", "feature.txt")
	wtRepo.Run("commit", "-m", "add feature")

	buf, err := runCmd("rebase", "no-changes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Rebased worktree") {
		t.Errorf("expected 'Rebased worktree', got: %s", out)
	}
}

func TestRebaseOnto(t *testing.T) {
	r := testutil.InitGitRepo(t)

	r.Run("checkout", "-b", "develop")
	r.CommitFile(t, "develop.txt", "develop commit")

	r.Run("checkout", "main")
	r.Run("checkout", "-b", "release")
	r.CommitFile(t, "release.txt", "release commit")

	r.Run("checkout", "develop")

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(worktree.ForkOptions{Name: "onto-test"})
	if err != nil {
		t.Fatal(err)
	}

	wtRepo := r.AtWorktree(wt.WorktreePath)
	wtRepo.WriteFile("feature.txt", "feature work")
	wtRepo.Run("add", "feature.txt")
	wtRepo.Run("commit", "-m", "add feature")

	buf, err := runCmd("rebase", "--onto", "release", "onto-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Rebased worktree") {
		t.Errorf("expected 'Rebased worktree', got: %s", out)
	}

	log := wtRepo.Run("log", "--oneline", "-3")
	if !strings.Contains(log, "add feature") {
		t.Error("worktree should still have feature commit")
	}
	if !strings.Contains(log, "release commit") {
		t.Error("worktree should be on release branch after --onto")
	}
	if strings.Contains(log, "develop commit") {
		t.Error("worktree should NOT have develop commit after --onto")
	}

	updatedWt, err := mgr.Get("onto-test")
	if err != nil {
		t.Fatal(err)
	}
	if updatedWt.Parent != "release" {
		t.Errorf("Parent = %q, want %q", updatedWt.Parent, "release")
	}
}

func TestRebaseAutostash(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("checkout", "-b", "develop")

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(worktree.ForkOptions{Name: "autostash"})
	if err != nil {
		t.Fatal(err)
	}

	wtRepo := r.AtWorktree(wt.WorktreePath)
	wtRepo.WriteFile("committed.txt", "committed")
	wtRepo.Run("add", "committed.txt")
	wtRepo.Run("commit", "-m", "committed file")

	wtRepo.WriteFile("uncommitted.txt", "uncommitted work")

	r.CommitFile(t, "develop-update.txt", "develop updated")

	buf, err := runCmd("rebase", "autostash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Rebased worktree") {
		t.Errorf("expected 'Rebased worktree', got: %s", out)
	}

	content, err := os.ReadFile(wt.WorktreePath + "/uncommitted.txt")
	if err != nil {
		t.Fatal("uncommitted.txt should exist after rebase with autostash")
	}
	if string(content) != "uncommitted work" {
		t.Errorf("uncommitted content = %q, want %q", content, "uncommitted work")
	}
}

func TestRebaseCurrentWorktree(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("checkout", "-b", "develop")

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(worktree.ForkOptions{Name: "current-wt"})
	if err != nil {
		t.Fatal(err)
	}

	wtRepo := r.AtWorktree(wt.WorktreePath)
	wtRepo.WriteFile("feature.txt", "feature work")
	wtRepo.Run("add", "feature.txt")
	wtRepo.Run("commit", "-m", "add feature")

	r.CommitFile(t, "develop-update.txt", "develop updated")

	if err := os.Chdir(wt.WorktreePath); err != nil {
		t.Fatal(err)
	}

	// Create manager from worktree directory
	buf, err := runCmd("rebase")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Rebased worktree") {
		t.Errorf("expected 'Rebased worktree', got: %s", out)
	}

	log := wtRepo.Run("log", "--oneline", "-3")
	if !strings.Contains(log, "develop updated") {
		t.Error("worktree should have parent's new commit after rebase")
	}
}

func TestRebaseNotInWorktree(t *testing.T) {
	testutil.InitGitRepo(t)

	_, err := runCmd("rebase")
	if err == nil {
		t.Fatal("expected error when not in worktree and no name given")
	}
	if !errors.Is(err, worktree.ErrNotInWorktree) {
		t.Errorf("expected ErrNotInWorktree, got: %v", err)
	}
}

func TestRebaseNotFound(t *testing.T) {
	testutil.InitGitRepo(t)

	_, err := runCmd("rebase", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent worktree")
	}
	if !errors.Is(err, worktree.ErrWorktreeNotFound) {
		t.Errorf("expected ErrWorktreeNotFound, got: %v", err)
	}
}

func TestRebaseParentNotFound(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("checkout", "-b", "develop")

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(worktree.ForkOptions{Name: "orphan"})
	if err != nil {
		t.Fatal(err)
	}

	// Checkout to a different branch first so we can delete develop
	wtRepo := r.AtWorktree(wt.WorktreePath)
	wtRepo.Run("checkout", "-b", "orphan-detached")

	r.Run("checkout", "main")
	r.Run("branch", "-D", "develop")

	_, err = runCmd("rebase", "orphan")
	if err == nil {
		t.Fatal("expected error when parent branch deleted")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}
