//go:build integration

package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/thurstonsand/wt/internal/testutil"
	"github.com/thurstonsand/wt/internal/worktree"
)

func TestPruneNoStale(t *testing.T) {
	testutil.InitGitRepo(t)

	buf, err := runCmd("prune", "--force")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "Nothing to prune") {
		t.Errorf("expected 'Nothing to prune' message, got: %s", buf.String())
	}
}

func TestPruneDryRun(t *testing.T) {
	r := testutil.InitGitRepo(t)

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Fork(worktree.ForkOptions{Name: "orphan-wt"}); err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(r.Dir); err != nil {
		t.Fatal(err)
	}

	buf, err := runCmd("prune", "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "orphan-wt") {
		t.Errorf("expected stale entry in output, got: %s", out)
	}
	if strings.Contains(out, "Removed") {
		t.Error("dry-run should not remove anything")
	}

	// Verify nothing was actually deleted by running force after
	buf, err = runCmd("prune", "--force")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "Removed") {
		t.Error("expected entries to still exist after dry-run")
	}
}

func TestPruneForce(t *testing.T) {
	r := testutil.InitGitRepo(t)

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}
	wt, err := mgr.Fork(worktree.ForkOptions{Name: "force-wt"})
	if err != nil {
		t.Fatal(err)
	}
	wtPath := wt.WorktreePath

	if err := os.RemoveAll(r.Dir); err != nil {
		t.Fatal(err)
	}

	buf, err := runCmd("prune", "--force")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "Removed") {
		t.Errorf("expected 'Removed' message, got: %s", buf.String())
	}

	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Error("worktree directory should be removed")
	}
}

func TestPruneRejectsArgs(t *testing.T) {
	testutil.InitGitRepo(t)

	_, err := runCmd("prune", "extra-arg")
	if err == nil {
		t.Fatal("expected error for extra argument")
	}
}

func TestPruneBranchesDryRun(t *testing.T) {
	r := testutil.InitGitRepo(t)

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Fork(worktree.ForkOptions{Name: "orphan-br"}); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Remove(worktree.RemoveOptions{Name: "orphan-br", Force: true, PreserveBranch: true}); err != nil {
		t.Fatal(err)
	}

	buf, err := runCmd("prune", "--all", "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "orphan-br") {
		t.Errorf("expected orphaned branch in output, got: %s", out)
	}
	if strings.Contains(out, "Deleted") {
		t.Error("dry-run should not delete anything")
	}
}

func TestPruneForceWithoutAllKeepsBranches(t *testing.T) {
	r := testutil.InitGitRepo(t)

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Fork(worktree.ForkOptions{Name: "keep-br"}); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Remove(worktree.RemoveOptions{Name: "keep-br", Force: true, PreserveBranch: true}); err != nil {
		t.Fatal(err)
	}

	buf, err := runCmd("prune", "--force")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(buf.String(), "Deleted branches:") {
		t.Errorf("--force without --all should not delete branches, got: %s", buf.String())
	}

	// Branch should still exist and be deletable with --all.
	buf, err = runCmd("prune", "--all", "--force")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "keep-br") {
		t.Errorf("expected orphaned branch to survive --force, got: %s", buf.String())
	}
}

func TestPruneBranchesForce(t *testing.T) {
	r := testutil.InitGitRepo(t)

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Fork(worktree.ForkOptions{Name: "force-br"}); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Remove(worktree.RemoveOptions{Name: "force-br", Force: true, PreserveBranch: true}); err != nil {
		t.Fatal(err)
	}

	buf, err := runCmd("prune", "--all", "--force")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "force-br") {
		t.Errorf("expected branch in output, got: %s", out)
	}
	if !strings.Contains(out, "Deleted branches:") {
		t.Errorf("expected 'Deleted branches:' message, got: %s", out)
	}
}

func TestPruneBranchesMultiRepo(t *testing.T) {
	r := testutil.InitGitRepo(t)

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Fork(worktree.ForkOptions{Name: "multi-br"}); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Remove(worktree.RemoveOptions{Name: "multi-br", Force: true, PreserveBranch: true}); err != nil {
		t.Fatal(err)
	}

	buf, err := runCmd("prune", "--all", "--force")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "multi-br") {
		t.Errorf("expected orphaned branch in output, got: %s", out)
	}
	if !strings.Contains(out, "Deleted branches:") {
		t.Errorf("expected 'Deleted branches:' message, got: %s", out)
	}
}
