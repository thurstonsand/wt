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

func TestForkBasic(t *testing.T) {
	testutil.InitGitRepo(t)

	buf, err := runCmd("fork", "test-fork")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Forked to worktree") {
		t.Errorf("expected 'Forked to worktree', got: %s", out)
	}
	if !strings.Contains(out, "test-fork") {
		t.Errorf("expected worktree name in output, got: %s", out)
	}
}

func TestForkGeneratesName(t *testing.T) {
	testutil.InitGitRepo(t)

	buf, err := runCmd("fork")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "wt-") {
		t.Errorf("expected generated name with wt- prefix, got: %s", out)
	}
}

func TestForkDuplicate(t *testing.T) {
	testutil.InitGitRepo(t)

	_, err := runCmd("fork", "dup")
	if err != nil {
		t.Fatalf("first fork failed: %v", err)
	}

	_, err = runCmd("fork", "dup")
	if err == nil {
		t.Fatal("expected error for duplicate worktree")
	}
	if !errors.Is(err, worktree.ErrWorktreeExists) {
		t.Errorf("expected ErrWorktreeExists, got: %v", err)
	}
}

func TestForkWithBase(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("branch", "develop")

	buf, err := runCmd("fork", "feature", "--base", "develop")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "feature") {
		t.Errorf("expected worktree name in output, got: %s", out)
	}
}

func TestForkClean(t *testing.T) {
	r := testutil.InitGitRepo(t)

	r.WriteFile("dirty.txt", "dirty content")
	r.Run("add", "dirty.txt")

	buf, err := runCmd("fork", "clean-fork", "--clean")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "clean-fork") {
		t.Errorf("expected worktree name in output, got: %s", out)
	}
}

func TestForkCopiesUntrackedDirectory(t *testing.T) {
	r := testutil.InitGitRepo(t)

	r.WriteFile(".claude/workflow-ref/step1.yaml", "step1")
	r.WriteFile(".claude/workflow-ref/nested/step2.yaml", "step2")

	_, err := runCmd("fork", "dir-fork")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mgr, err := defaultManager()
	if err != nil {
		t.Fatalf("unexpected error creating manager: %v", err)
	}
	wtDir := mgr.WorktreePath("dir-fork")

	for _, rel := range []string{
		".claude/workflow-ref/step1.yaml",
		".claude/workflow-ref/nested/step2.yaml",
	} {
		if _, err := os.Stat(filepath.Join(wtDir, rel)); err != nil {
			t.Errorf("expected %s to exist in worktree: %v", rel, err)
		}
	}
}

func TestForkInvalidFlags(t *testing.T) {
	testutil.InitGitRepo(t)

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "clean with no-clean",
			args: []string{"fork", "wt", "--clean", "--no-clean"},
		},
		{
			name: "no-clean with base",
			args: []string{"fork", "wt2", "--no-clean", "--base", "main"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runCmd(tt.args...)
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, ErrInvalidFlagCombination) {
				t.Errorf("expected ErrInvalidFlagCombination, got: %v", err)
			}
		})
	}
}
