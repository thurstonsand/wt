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

func TestCheckout(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	if err := g.CreateBranch("develop", "HEAD"); err != nil {
		t.Fatal(err)
	}

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	wt, _, err := mgr.Checkout(CheckoutOptions{Branch: "develop"})
	if err != nil {
		t.Fatal(err)
	}

	if wt.Name != "develop" {
		t.Errorf("expected name develop, got %s", wt.Name)
	}
	if wt.Branch != "develop" {
		t.Errorf("expected branch develop, got %s", wt.Branch)
	}
	if _, err := os.Stat(wt.WorktreePath); os.IsNotExist(err) {
		t.Errorf("worktree directory does not exist: %s", wt.WorktreePath)
	}
}

func TestCheckoutFromLinkedWorktree(t *testing.T) {
	r := testutil.InitGitRepo(t)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	if err := git.New(r.Dir).CreateBranch("develop", "HEAD"); err != nil {
		t.Fatal(err)
	}

	mgr, err := NewManager(r.Dir,
		WithGit(git.New(r.Dir)),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	first, err := mgr.Fork(ForkOptions{Name: "first"})
	if err != nil {
		t.Fatal(err)
	}

	// Checking out from inside a linked worktree must group the new worktree
	// under the canonical repo name, not the linked worktree's basename.
	nested, err := NewManager(first.WorktreePath,
		WithGit(git.New(first.WorktreePath)),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	wt, _, err := nested.Checkout(CheckoutOptions{Branch: "develop"})
	if err != nil {
		t.Fatal(err)
	}

	expectedPath := globalStore.WorktreePath(mgr.RepoName(), "develop")
	if wt.WorktreePath != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, wt.WorktreePath)
	}
}

func TestCheckoutDuplicate(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	if err := g.CreateBranch("develop", "HEAD"); err != nil {
		t.Fatal(err)
	}

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	first, created, err := mgr.Checkout(CheckoutOptions{Branch: "develop"})
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("first checkout should create a worktree")
	}

	second, created, err := mgr.Checkout(CheckoutOptions{Branch: "develop"})
	if err != nil {
		t.Fatalf("checkout of existing worktree failed: %v", err)
	}
	if created {
		t.Fatal("second checkout should reuse the worktree")
	}
	if second.WorktreePath != first.WorktreePath {
		t.Errorf("existing worktree path = %q, want %q", second.WorktreePath, first.WorktreePath)
	}
}

func TestCheckoutNonexistentBranch(t *testing.T) {
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

	_, _, err = mgr.Checkout(CheckoutOptions{Branch: "nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent branch, got nil")
	}
}

func TestCheckoutCopiesIncludeFiles(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.WriteFile(".env", "SECRET=x")
	r.WriteFile(".env.local", "LOCAL=y")
	r.WriteFile(".worktreeinclude", ".env*\n")

	g := git.New(r.Dir)
	if err := g.CreateBranch("feature", "HEAD"); err != nil {
		t.Fatal(err)
	}

	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)
	_, _, _ = globalStore.LoadConfig()

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	wt, _, err := mgr.Checkout(CheckoutOptions{Branch: "feature"})
	if err != nil {
		t.Fatal(err)
	}

	assertFileExists(t, wt.WorktreePath, ".env")
	assertFileExists(t, wt.WorktreePath, ".env.local")
}

func TestCheckoutWithNonexistentParent(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	if err := g.CreateBranch("feature", "HEAD"); err != nil {
		t.Fatal(err)
	}

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	wt, _, err := mgr.Checkout(CheckoutOptions{Branch: "feature", Parent: "no-such-branch"})
	if err != nil {
		t.Fatal(err)
	}

	if wt.Parent != "no-such-branch" {
		t.Errorf("expected parent no-such-branch, got %s", wt.Parent)
	}
	if _, err := os.Stat(wt.WorktreePath); os.IsNotExist(err) {
		t.Errorf("worktree directory does not exist: %s", wt.WorktreePath)
	}
}

func TestCheckoutWithExtraFiles(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.WriteFile("extra.txt", "extra content")

	g := git.New(r.Dir)
	if err := g.CreateBranch("feature-with", "HEAD"); err != nil {
		t.Fatal(err)
	}

	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)
	_, _, _ = globalStore.LoadConfig()

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	wt, _, err := mgr.Checkout(CheckoutOptions{
		Branch: "feature-with",
		With:   []string{"extra.txt"},
	})
	if err != nil {
		t.Fatal(err)
	}

	assertFileExists(t, wt.WorktreePath, "extra.txt")

	content, err := os.ReadFile(filepath.Join(wt.WorktreePath, "extra.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "extra content" {
		t.Errorf("expected 'extra content', got %q", content)
	}
}

// TestCheckoutStampsDefaultParent verifies that a checkout with no explicit
// parent records origin/<default> so the branch is visible to prune later.
func TestCheckoutStampsDefaultParent(t *testing.T) {
	r := testutil.InitGitRepo(t)
	remoteDir := t.TempDir()
	testutil.NewGitRepo(t, remoteDir).Run("init", "--bare", "--initial-branch=main")
	r.Run("remote", "add", "origin", remoteDir)
	r.Run("push", "origin", "main")

	g := git.New(r.Dir)
	if err := g.CreateBranch("feature", "HEAD"); err != nil {
		t.Fatal(err)
	}

	mgr, err := NewManager(r.Dir, WithGit(g), WithGlobalStore(config.NewGlobalStore(t.TempDir())))
	if err != nil {
		t.Fatal(err)
	}

	if _, _, err := mgr.Checkout(CheckoutOptions{Branch: "feature"}); err != nil {
		t.Fatal(err)
	}

	meta, err := g.GetBranchMeta("feature")
	if err != nil {
		t.Fatal(err)
	}
	if meta.Parent != "origin/main" {
		t.Errorf("Parent = %q, want %q", meta.Parent, "origin/main")
	}
}

// TestCheckoutDefaultParentLocalFallback verifies that without a remote the
// parent falls back to the local default branch.
func TestCheckoutDefaultParentLocalFallback(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)
	if err := g.CreateBranch("feature", "HEAD"); err != nil {
		t.Fatal(err)
	}

	mgr, err := NewManager(r.Dir, WithGit(g), WithGlobalStore(config.NewGlobalStore(t.TempDir())))
	if err != nil {
		t.Fatal(err)
	}

	if _, _, err := mgr.Checkout(CheckoutOptions{Branch: "feature"}); err != nil {
		t.Fatal(err)
	}

	meta, err := g.GetBranchMeta("feature")
	if err != nil {
		t.Fatal(err)
	}
	if meta.Parent != "main" {
		t.Errorf("Parent = %q, want %q", meta.Parent, "main")
	}
}

func TestCheckoutTracksRemoteBranchAfterFetch(t *testing.T) {
	r := testutil.InitGitRepo(t)

	remoteDir := t.TempDir()
	testutil.NewGitRepo(t, remoteDir).Run("init", "--bare", "--initial-branch=main")
	r.Run("remote", "add", "origin", remoteDir)
	r.Run("push", "origin", "main")
	r.Run("branch", "feature-remote")
	r.Run("push", "origin", "feature-remote")
	r.Run("branch", "-D", "feature-remote")
	// Intentionally do NOT fetch: checkout should fetch and resolve it.

	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	mgr, err := NewManager(r.Dir,
		WithGit(git.New(r.Dir)),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	wt, _, err := mgr.Checkout(CheckoutOptions{Branch: "feature-remote"})
	if err != nil {
		t.Fatalf("checkout of remote-only branch failed: %v", err)
	}

	wtGit := git.New(wt.WorktreePath)
	upstream, err := wtGit.RevParse("--abbrev-ref", "feature-remote@{upstream}")
	if err != nil {
		t.Fatalf("expected tracking branch, got error: %v", err)
	}
	if upstream != "origin/feature-remote" {
		t.Errorf("expected upstream origin/feature-remote, got %q", upstream)
	}
}
