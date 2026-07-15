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

// landWorktree forks <name>, pushes it, squash-merges to main, deletes the
// remote branch, and prunes — leaving <name> landed. Returns the worktree path.
func landWorktree(t *testing.T, r *testutil.GitRepo, name string) string {
	t.Helper()

	remoteDir := t.TempDir()
	bare := testutil.NewGitRepo(t, remoteDir)
	bare.Run("init", "--bare", "--initial-branch=main")
	r.Run("remote", "add", "origin", remoteDir)
	r.Run("push", "-u", "origin", "main")
	r.Run("remote", "set-head", "origin", "main")

	if _, err := runCmd("fork", name); err != nil {
		t.Fatalf("fork failed: %v", err)
	}
	pathBuf, err := runCmd("path", name)
	if err != nil {
		t.Fatalf("path failed: %v", err)
	}
	wtPath := strings.TrimSpace(pathBuf.String())

	wr := r.AtWorktree(wtPath)
	wr.CommitFile(t, "feature.txt", "pr work")
	wr.Run("push", "-u", "origin", name)

	r.Run("merge", "--squash", name)
	r.Run("commit", "-m", "squash-merge of "+name)
	r.Run("push", "origin", "main")
	r.Run("push", "origin", "--delete", name)
	r.Run("fetch", "--prune", "origin")

	return wtPath
}

func TestRebranchSuccess(t *testing.T) {
	r := testutil.InitGitRepo(t)
	wtPath := landWorktree(t, r, "wt1")

	r.AtWorktree(wtPath).WriteFile("continued.txt", "more work")

	buf, err := runCmd("rebranch", "wt1-cont", "-w", "wt1")
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, buf.String())
	}
	out := buf.String()
	if !strings.Contains(out, "Rebranched") {
		t.Errorf("expected 'Rebranched', got: %s", out)
	}
	if !strings.Contains(out, "left behind") {
		t.Errorf("expected note about spent branch, got: %s", out)
	}
	if _, err := os.Stat(filepath.Join(wtPath, "continued.txt")); err != nil {
		t.Errorf("dirty work not carried: %v", err)
	}
}

func TestRebranchRefusesNotLanded(t *testing.T) {
	r := testutil.InitGitRepo(t)

	remoteDir := t.TempDir()
	bare := testutil.NewGitRepo(t, remoteDir)
	bare.Run("init", "--bare", "--initial-branch=main")
	r.Run("remote", "add", "origin", remoteDir)
	r.Run("push", "-u", "origin", "main")
	r.Run("remote", "set-head", "origin", "main")

	if _, err := runCmd("fork", "wt1"); err != nil {
		t.Fatal(err)
	}
	pathBuf, _ := runCmd("path", "wt1")
	wtPath := strings.TrimSpace(pathBuf.String())
	wr := r.AtWorktree(wtPath)
	wr.CommitFile(t, "feature.txt", "work")
	wr.Run("push", "-u", "origin", "wt1")

	_, err := runCmd("rebranch", "wt1-cont", "-w", "wt1")
	if !errors.Is(err, worktree.ErrNotLanded) {
		t.Errorf("expected ErrNotLanded, got %v", err)
	}
}
