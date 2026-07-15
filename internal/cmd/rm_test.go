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

func TestRmClean(t *testing.T) {
	r := testutil.InitGitRepo(t)

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(worktree.ForkOptions{Name: "to-remove"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(wt.WorktreePath); os.IsNotExist(err) {
		t.Fatal("worktree directory should exist before removal")
	}

	buf, err := runCmd("rm", "to-remove")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Removed worktree") {
		t.Errorf("expected 'Removed worktree', got: %s", out)
	}

	if _, err := os.Stat(wt.WorktreePath); !os.IsNotExist(err) {
		t.Error("worktree directory should not exist after removal")
	}

	if mgr.Exists("to-remove") {
		t.Error("worktree metadata should not exist after removal")
	}
}

func TestRmDirtyFails(t *testing.T) {
	r := testutil.InitGitRepo(t)

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(worktree.ForkOptions{Name: "dirty-wt"})
	if err != nil {
		t.Fatal(err)
	}

	wtRepo := r.AtWorktree(wt.WorktreePath)
	wtRepo.WriteFile("dirty.txt", "dirty")

	_, err = runCmd("rm", "dirty-wt")
	if err == nil {
		t.Fatal("expected error when removing dirty worktree")
	}

	if !errors.Is(err, worktree.ErrWorktreeDirty) {
		t.Errorf("expected ErrWorktreeDirty, got: %v", err)
	}

	if !mgr.Exists("dirty-wt") {
		t.Error("dirty worktree should not be removed without force")
	}
}

func TestRmDirtyForce(t *testing.T) {
	r := testutil.InitGitRepo(t)

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(worktree.ForkOptions{Name: "dirty-force"})
	if err != nil {
		t.Fatal(err)
	}

	wtRepo := r.AtWorktree(wt.WorktreePath)
	wtRepo.WriteFile("dirty.txt", "dirty")

	buf, err := runCmd("rm", "-f", "dirty-force")
	if err != nil {
		t.Fatalf("unexpected error with -f: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Removed worktree") {
		t.Errorf("expected 'Removed worktree', got: %s", out)
	}

	if mgr.Exists("dirty-force") {
		t.Error("worktree should be removed with force flag")
	}
}

func TestRmNotFound(t *testing.T) {
	testutil.InitGitRepo(t)

	_, err := runCmd("rm", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent worktree")
	}

	if !errors.Is(err, worktree.ErrWorktreeNotFound) {
		t.Errorf("expected ErrWorktreeNotFound, got: %v", err)
	}
}

func TestRmRequiresNameOrWorktree(t *testing.T) {
	testutil.InitGitRepo(t)

	_, err := runCmd("rm")
	if err == nil {
		t.Fatal("expected error when no argument and not in worktree")
	}
	if !errors.Is(err, worktree.ErrNotInWorktree) {
		t.Errorf("expected ErrNotInWorktree, got: %v", err)
	}
}
