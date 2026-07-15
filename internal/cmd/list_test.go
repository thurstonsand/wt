//go:build integration

package cmd

import (
	"strings"
	"testing"

	"github.com/thurstonsand/wt/internal/testutil"
	"github.com/thurstonsand/wt/internal/worktree"
)

func TestListNoWorktrees(t *testing.T) {
	testutil.InitGitRepo(t)

	buf, err := runCmd("list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "(root)") {
		t.Errorf("expected (root) row, got: %s", out)
	}
	if !strings.Contains(out, "NAME") {
		t.Errorf("expected table header, got: %s", out)
	}
	if !strings.Contains(out, "main") {
		t.Errorf("expected main branch in root row, got: %s", out)
	}
	if !strings.Contains(out, "repo:") {
		t.Errorf("expected repo subtitle, got: %s", out)
	}
}

func TestListWithWorktrees(t *testing.T) {
	r := testutil.InitGitRepo(t)

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = mgr.Fork(worktree.ForkOptions{Name: "feature-one"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = mgr.Fork(worktree.ForkOptions{Name: "feature-two"})
	if err != nil {
		t.Fatal(err)
	}

	buf, err := runCmd("list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "(root)") {
		t.Errorf("expected (root) row, got: %s", out)
	}
	if !strings.Contains(out, "NAME") {
		t.Errorf("expected header row, got: %s", out)
	}
	if !strings.Contains(out, "feature-one") {
		t.Errorf("expected feature-one in output, got: %s", out)
	}
	if !strings.Contains(out, "feature-two") {
		t.Errorf("expected feature-two in output, got: %s", out)
	}
	if !strings.Contains(out, "main") {
		t.Errorf("expected parent branch main in output, got: %s", out)
	}
}

func TestListStateColumn(t *testing.T) {
	r := testutil.InitGitRepo(t)

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}
	wt, err := mgr.Fork(worktree.ForkOptions{Name: "dirty-wt"})
	if err != nil {
		t.Fatal(err)
	}
	r.AtWorktree(wt.WorktreePath).WriteFile("change.txt", "uncommitted")

	buf, err := runCmd("list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "STATE") {
		t.Errorf("expected STATE header, got: %s", out)
	}
	if !strings.Contains(out, "dirty") {
		t.Errorf("expected dirty state for worktree with changes, got: %s", out)
	}
}
