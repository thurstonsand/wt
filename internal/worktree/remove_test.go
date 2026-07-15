//go:build integration

package worktree

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thurstonsand/wt/internal/config"
	"github.com/thurstonsand/wt/internal/git"
	"github.com/thurstonsand/wt/internal/testutil"
)

func TestRemoveDeletesBranch(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{Name: "to-remove"})
	if err != nil {
		t.Fatal(err)
	}

	exists, _ := g.LocalBranchExists("to-remove")
	if !exists {
		t.Fatal("expected branch to exist after fork")
	}

	result, err := mgr.Remove(RemoveOptions{Name: wt.Name})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(wt.WorktreePath); !os.IsNotExist(err) {
		t.Error("expected worktree directory to be removed")
	}

	exists, _ = g.LocalBranchExists("to-remove")
	if exists {
		t.Error("expected branch to be deleted after remove")
	}

	if result.TargetPath != "" {
		t.Errorf("expected empty target path when not inside worktree, got %s", result.TargetPath)
	}
}

func TestRemovePreservesBranch(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{Name: "preserve-me"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = mgr.Remove(RemoveOptions{Name: wt.Name, PreserveBranch: true})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(wt.WorktreePath); !os.IsNotExist(err) {
		t.Error("expected worktree directory to be removed")
	}

	exists, _ := g.LocalBranchExists("preserve-me")
	if !exists {
		t.Error("expected branch to be preserved with PreserveBranch option")
	}
}

func TestRemoveDeletesDirectoryAndEmptyParent(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{Name: "cleanup-test"})
	if err != nil {
		t.Fatal(err)
	}

	repoDir := filepath.Dir(wt.WorktreePath)

	_, err = mgr.Remove(RemoveOptions{Name: wt.Name})
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

func TestRemoveKeepsParentWhenSiblingExists(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	wt1, err := mgr.Fork(ForkOptions{Name: "sibling-a"})
	if err != nil {
		t.Fatal(err)
	}
	wt2, err := mgr.Fork(ForkOptions{Name: "sibling-b"})
	if err != nil {
		t.Fatal(err)
	}

	repoDir := filepath.Dir(wt1.WorktreePath)

	_, err = mgr.Remove(RemoveOptions{Name: wt1.Name})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(wt1.WorktreePath); !os.IsNotExist(err) {
		t.Error("expected removed worktree directory to be gone")
	}
	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		t.Error("repo parent should still exist when sibling worktree present")
	}
	if _, err := os.Stat(wt2.WorktreePath); os.IsNotExist(err) {
		t.Error("sibling worktree should still exist")
	}
}

func TestRemoveExternalWorktreeSkipsParentCleanup(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	extParent := filepath.Join(t.TempDir(), "my-project")
	extPath := filepath.Join(extParent, "my-wt")

	if err := g.WorktreeAdd(extPath, "ext-branch", true, ""); err != nil {
		t.Fatalf("failed to create external worktree: %v", err)
	}

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = mgr.Remove(RemoveOptions{Name: "ext-branch"})
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

func TestRemoveForceDeletesUnmergedBranch(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{Name: "unmerged"})
	if err != nil {
		t.Fatal(err)
	}

	wtRepo := r.AtWorktree(wt.WorktreePath)
	wtRepo.CommitFile(t, "new-file.txt", "unmerged commit")

	_, err = mgr.Remove(RemoveOptions{Name: wt.Name, Force: true})
	if err != nil {
		t.Fatal(err)
	}

	exists, _ := g.LocalBranchExists("unmerged")
	if exists {
		t.Error("expected unmerged branch to be force-deleted")
	}
}
