//go:build integration

package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/thurstonsand/wt/internal/testutil"
	"github.com/thurstonsand/wt/internal/worktree"
)

func TestCheckoutBasic(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("branch", "develop")

	buf, err := runCmd("checkout", "develop")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Checked out") {
		t.Errorf("expected 'Checked out', got: %s", out)
	}
	if !strings.Contains(out, "develop") {
		t.Errorf("expected branch name in output, got: %s", out)
	}
}

func TestCheckoutWithParent(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("branch", "feature")

	buf, err := runCmd("checkout", "feature", "--parent", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "feature") {
		t.Errorf("expected branch name in output, got: %s", out)
	}

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}
	wt, err := mgr.Get("feature")
	if err != nil {
		t.Fatal(err)
	}
	if wt.Parent != "main" {
		t.Errorf("expected parent main, got %s", wt.Parent)
	}
}

func TestCheckoutAlias(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("branch", "develop")

	buf, err := runCmd("co", "develop")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Checked out") {
		t.Errorf("expected 'Checked out', got: %s", out)
	}
}

func TestCheckoutDuplicate(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("branch", "develop")

	_, err := runCmd("checkout", "develop")
	if err != nil {
		t.Fatalf("first checkout failed: %v", err)
	}

	_, err = runCmd("checkout", "develop")
	if err == nil {
		t.Fatal("expected error for duplicate worktree")
	}
	if !errors.Is(err, worktree.ErrWorktreeExists) {
		t.Errorf("expected ErrWorktreeExists, got: %v", err)
	}
}

func TestCheckoutNonexistentBranch(t *testing.T) {
	testutil.InitGitRepo(t)

	_, err := runCmd("checkout", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent branch")
	}
}

func TestCheckoutRequiresArg(t *testing.T) {
	testutil.InitGitRepo(t)

	_, err := runCmd("checkout")
	if err == nil {
		t.Fatal("expected error when no argument provided")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Errorf("expected arg count error, got: %v", err)
	}
}

func TestCheckoutWithNonexistentParent(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("branch", "feature")

	buf, err := runCmd("checkout", "feature", "--parent", "no-such-branch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "feature") {
		t.Errorf("expected branch name in output, got: %s", out)
	}

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}
	wt, err := mgr.Get("feature")
	if err != nil {
		t.Fatal(err)
	}
	if wt.Parent != "no-such-branch" {
		t.Errorf("expected parent no-such-branch, got %s", wt.Parent)
	}
}
