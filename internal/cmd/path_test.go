//go:build integration

package cmd

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thurstonsand/wt/internal/testutil"
	"github.com/thurstonsand/wt/internal/worktree"
)

func TestPathNoArgs(t *testing.T) {
	r := testutil.InitGitRepo(t)

	buf, err := runCmd("path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := strings.TrimSpace(buf.String())
	outReal, _ := filepath.EvalSymlinks(out)
	expectedReal, _ := filepath.EvalSymlinks(r.Dir)
	if outReal != expectedReal {
		t.Errorf("expected main path %s, got: %s", r.Dir, out)
	}
}

func TestPathExists(t *testing.T) {
	r := testutil.InitGitRepo(t)

	mgr, err := worktree.NewManager(r.Dir)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(worktree.ForkOptions{Name: "my-feature"})
	if err != nil {
		t.Fatal(err)
	}

	buf, err := runCmd("path", "my-feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := strings.TrimSpace(buf.String())
	outReal, _ := filepath.EvalSymlinks(out)
	expectedReal, _ := filepath.EvalSymlinks(wt.WorktreePath)
	if outReal != expectedReal {
		t.Errorf("expected path %s, got: %s", wt.WorktreePath, out)
	}
}

func TestPathNotFound(t *testing.T) {
	testutil.InitGitRepo(t)

	_, err := runCmd("path", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent worktree")
	}

	if !errors.Is(err, worktree.ErrWorktreeNotFound) {
		t.Errorf("expected ErrWorktreeNotFound, got: %v", err)
	}
}
