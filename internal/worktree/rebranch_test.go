//go:build integration

package worktree

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/thurstonsand/wt/internal/config"
	"github.com/thurstonsand/wt/internal/git"
	"github.com/thurstonsand/wt/internal/testutil"
)

// rebranchEnv wires a repo + bare origin + manager for rebranch tests.
type rebranchEnv struct {
	r     *testutil.GitRepo
	mgr   *Manager
	store *config.GlobalStore
}

func setupRebranchEnv(t *testing.T) *rebranchEnv {
	t.Helper()
	r := testutil.InitGitRepo(t)

	remoteDir := t.TempDir()
	bare := testutil.NewGitRepo(t, remoteDir)
	bare.Run("init", "--bare", "--initial-branch=main")
	r.Run("remote", "add", "origin", remoteDir)
	r.Run("push", "-u", "origin", "main")
	r.Run("remote", "set-head", "origin", "main")

	store := config.NewGlobalStore(t.TempDir())
	mgr, err := NewManager(r.Dir, WithGit(git.New(r.Dir)), WithGlobalStore(store))
	if err != nil {
		t.Fatal(err)
	}
	return &rebranchEnv{r: r, mgr: mgr, store: store}
}

// forkAndPush forks a worktree, makes a commit in it, and pushes the branch
// with upstream tracking. Returns the worktree.
func (e *rebranchEnv) forkAndPush(t *testing.T, name string) *Worktree {
	t.Helper()
	wt, err := e.mgr.Fork(ForkOptions{Name: name})
	if err != nil {
		t.Fatal(err)
	}
	wr := e.r.AtWorktree(wt.WorktreePath)
	wr.CommitFile(t, "feature.txt", "pr work")
	wr.Run("push", "-u", "origin", name)
	return wt
}

// land simulates a squash-merge: the branch's content lands on main (via a
// squash commit), the remote branch is deleted, and the local repo prunes.
func (e *rebranchEnv) land(t *testing.T, branch string) {
	t.Helper()
	e.r.Run("merge", "--squash", branch)
	e.r.Run("commit", "-m", "squash-merge of "+branch)
	e.r.Run("push", "origin", "main")
	e.r.Run("push", "origin", "--delete", branch)
	e.r.Run("fetch", "--prune", "origin")
}

func TestRebranchDirtyOnly(t *testing.T) {
	e := setupRebranchEnv(t)
	wt := e.forkAndPush(t, "wt1")
	e.land(t, "wt1")

	wr := e.r.AtWorktree(wt.WorktreePath)
	wr.WriteFile("newfeature.txt", "continued work")
	wr.WriteFile("feature.txt", "pr work\nmore\n")

	res, err := e.mgr.Rebranch(RebranchOptions{NewBranch: "wt1-cont", ForWorktree: "wt1"})
	if err != nil {
		t.Fatalf("rebranch failed: %v", err)
	}

	if res.Conflict {
		t.Error("unexpected conflict")
	}
	if res.NewBranch != "wt1-cont" {
		t.Errorf("expected new branch wt1-cont, got %s", res.NewBranch)
	}
	if res.WorktreeName != "wt1" {
		t.Errorf("folder name should be preserved as wt1, got %s", res.WorktreeName)
	}
	if res.Baseline != "main" {
		t.Errorf("expected baseline main, got %s", res.Baseline)
	}

	// Folder unchanged; branch swapped.
	wr = e.r.AtWorktree(wt.WorktreePath)
	branch := wr.Run("rev-parse", "--abbrev-ref", "HEAD")
	if got := filepath.Base(wt.WorktreePath); got != "wt1" {
		t.Errorf("folder should remain wt1, got %s", got)
	}
	if branch != "wt1-cont\n" {
		t.Errorf("expected branch wt1-cont, got %q", branch)
	}

	// Dirty work carried forward.
	if _, err := os.Stat(filepath.Join(wt.WorktreePath, "newfeature.txt")); err != nil {
		t.Errorf("untracked file not carried: %v", err)
	}

	// Old branch left behind.
	if exists, _ := e.mgr.Git().LocalBranchExists("wt1"); !exists {
		t.Error("old branch wt1 should be left behind")
	}

	// New branch records baseline as parent.
	meta, err := e.mgr.Git().GetBranchMeta("wt1-cont")
	if err != nil {
		t.Fatal(err)
	}
	if meta.Parent != "main" {
		t.Errorf("expected parent main, got %s", meta.Parent)
	}

	// The new branch tracks origin/main and must NOT be reported as landed.
	landed, err := e.mgr.Git().UpstreamGone("wt1-cont")
	if err != nil {
		t.Fatal(err)
	}
	if landed {
		t.Error("freshly rebranched wt1-cont should not be reported as landed")
	}
}

func TestRebranchMoveRenamesWorktree(t *testing.T) {
	e := setupRebranchEnv(t)
	wt := e.forkAndPush(t, "wt1")
	e.land(t, "wt1")

	e.r.AtWorktree(wt.WorktreePath).WriteFile("continued.txt", "more work")
	newPath := e.mgr.WorktreePath("feature/continued")

	res, err := e.mgr.Rebranch(RebranchOptions{
		NewBranch:   "feature/continued",
		ForWorktree: "wt1",
		Move:        true,
	})
	if err != nil {
		t.Fatalf("rebranch failed: %v", err)
	}

	if res.WorktreePath != newPath {
		t.Errorf("WorktreePath = %q, want %q", res.WorktreePath, newPath)
	}
	if res.WorktreeName != filepath.Base(newPath) {
		t.Errorf("WorktreeName = %q, want %q", res.WorktreeName, filepath.Base(newPath))
	}
	if _, err := os.Stat(wt.WorktreePath); !os.IsNotExist(err) {
		t.Errorf("old worktree path should be gone, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(newPath, "continued.txt")); err != nil {
		t.Errorf("dirty work not carried to moved worktree: %v", err)
	}
	if branch := e.r.AtWorktree(newPath).Run("rev-parse", "--abbrev-ref", "HEAD"); branch != "feature/continued\n" {
		t.Errorf("expected branch feature/continued, got %q", branch)
	}
}

func TestRebranchRefusesNotLanded(t *testing.T) {
	e := setupRebranchEnv(t)
	e.forkAndPush(t, "wt1")
	// Not landed: remote branch still exists.

	_, err := e.mgr.Rebranch(RebranchOptions{NewBranch: "wt1-cont", ForWorktree: "wt1"})
	if !errors.Is(err, ErrNotLanded) {
		t.Errorf("expected ErrNotLanded, got %v", err)
	}
}

func TestRebranchRefusesNeverPushed(t *testing.T) {
	e := setupRebranchEnv(t)
	_, err := e.mgr.Fork(ForkOptions{Name: "wt1"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = e.mgr.Rebranch(RebranchOptions{NewBranch: "wt1-cont", ForWorktree: "wt1"})
	if !errors.Is(err, ErrNeverPushed) {
		t.Errorf("expected ErrNeverPushed, got %v", err)
	}
}

func TestRebranchRefusesNameCollision(t *testing.T) {
	e := setupRebranchEnv(t)
	e.forkAndPush(t, "wt1")
	e.land(t, "wt1")

	_, err := e.mgr.Rebranch(RebranchOptions{NewBranch: "main", ForWorktree: "wt1"})
	if !errors.Is(err, ErrWorktreeExists) {
		t.Errorf("expected ErrWorktreeExists for existing branch, got %v", err)
	}
}

func TestRebranchConflictPreservesMovedWorktree(t *testing.T) {
	e := setupRebranchEnv(t)
	wt := e.forkAndPush(t, "wt1")

	// Land wt1, then have main diverge on the same file so restoring the
	// dirty edit conflicts.
	e.land(t, "wt1")
	e.r.WriteFile("feature.txt", "main rewrote this line\n")
	e.r.Run("add", "feature.txt")
	e.r.Run("commit", "-m", "main edits feature.txt")
	e.r.Run("push", "origin", "main")
	e.r.Run("fetch", "--prune", "origin")

	// Dirty edit to the same file in the worktree.
	wr := e.r.AtWorktree(wt.WorktreePath)
	wr.WriteFile("feature.txt", "my conflicting edit\n")

	res, err := e.mgr.Rebranch(RebranchOptions{NewBranch: "wt1-cont", ForWorktree: "wt1", Move: true})
	if !errors.Is(err, ErrRebranchConflict) {
		t.Fatalf("expected ErrRebranchConflict, got %v", err)
	}
	if res == nil || !res.Conflict {
		t.Fatal("result should report conflict")
	}
	if res.WorktreePath != e.mgr.WorktreePath("wt1-cont") {
		t.Errorf("conflict path = %q, want %q", res.WorktreePath, e.mgr.WorktreePath("wt1-cont"))
	}
	if _, err := os.Stat(wt.WorktreePath); !os.IsNotExist(err) {
		t.Errorf("old worktree path should be gone, got %v", err)
	}
	branch := e.r.AtWorktree(res.WorktreePath).Run("rev-parse", "--abbrev-ref", "HEAD")
	if branch != "wt1-cont\n" {
		t.Errorf("worktree should be on wt1-cont, got %q", branch)
	}
	if exists, _ := e.mgr.Git().LocalBranchExists("wt1"); !exists {
		t.Error("old branch wt1 should be preserved on conflict")
	}
}

func TestRebranchPreservesIndependentParentCommits(t *testing.T) {
	e := setupRebranchEnv(t)
	wt := e.forkAndPush(t, "wt1")

	// Another contributor lands an independent file on main via the remote.
	e.r.CommitFile(t, "other-contributor.txt", "independent work")
	e.r.Run("push", "origin", "main")

	e.land(t, "wt1")

	wr := e.r.AtWorktree(wt.WorktreePath)
	wr.WriteFile("mine.txt", "my continued work")

	res, err := e.mgr.Rebranch(RebranchOptions{NewBranch: "wt1-cont", ForWorktree: "wt1"})
	if err != nil {
		t.Fatalf("rebranch failed: %v", err)
	}
	if res.Conflict {
		t.Error("unexpected conflict")
	}

	// The other contributor's file must be present (came in via fresh baseline).
	if _, err := os.Stat(filepath.Join(wt.WorktreePath, "other-contributor.txt")); err != nil {
		t.Errorf("other-contributor.txt should be present from fresh baseline: %v", err)
	}
	// My dirty work carried.
	if _, err := os.Stat(filepath.Join(wt.WorktreePath, "mine.txt")); err != nil {
		t.Errorf("mine.txt should be carried: %v", err)
	}
}

func TestRebranchOntoCustomBaseline(t *testing.T) {
	e := setupRebranchEnv(t)

	// A develop branch exists on the remote as an alternate baseline.
	e.r.Run("checkout", "-b", "develop")
	e.r.CommitFile(t, "develop.txt", "develop baseline")
	e.r.Run("push", "-u", "origin", "develop")
	e.r.Run("checkout", "main")

	wt := e.forkAndPush(t, "wt1")
	e.land(t, "wt1")

	wr := e.r.AtWorktree(wt.WorktreePath)
	wr.WriteFile("mine.txt", "continued work")

	res, err := e.mgr.Rebranch(RebranchOptions{NewBranch: "wt1-cont", ForWorktree: "wt1", Onto: "develop"})
	if err != nil {
		t.Fatalf("rebranch failed: %v", err)
	}
	if res.Baseline != "develop" {
		t.Errorf("expected baseline develop, got %s", res.Baseline)
	}
	if _, err := os.Stat(filepath.Join(wt.WorktreePath, "develop.txt")); err != nil {
		t.Errorf("develop baseline content should be present: %v", err)
	}
	meta, err := e.mgr.Git().GetBranchMeta("wt1-cont")
	if err != nil {
		t.Fatal(err)
	}
	if meta.Parent != "develop" {
		t.Errorf("expected parent develop, got %s", meta.Parent)
	}
}
