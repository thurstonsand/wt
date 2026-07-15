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

func initPruneTest(t *testing.T) *config.GlobalStore {
	t.Helper()
	wtHome := t.TempDir()
	t.Setenv("WT_HOME", wtHome)
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	return config.NewGlobalStore(wtHome)
}

func mkRepo(t *testing.T, store *config.GlobalStore, name string) (*Manager, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	testutil.NewGitRepo(t, dir).Init()
	mgr, err := NewManager(dir, WithGit(git.New(dir)), WithGlobalStore(store))
	if err != nil {
		t.Fatal(err)
	}
	return mgr, dir
}

func TestFindStaleEmpty(t *testing.T) {
	store := initPruneTest(t)

	stale, err := FindStale(store)
	if err != nil {
		t.Fatalf("FindStale() error = %v", err)
	}
	if len(stale) != 0 {
		t.Errorf("expected 0 stale, got %d", len(stale))
	}
}

func TestFindStaleDeletedSourceRepo(t *testing.T) {
	store := initPruneTest(t)
	mgr, dir := mkRepo(t, store, "myrepo")

	if _, err := mgr.Fork(ForkOptions{Name: "wt-orphan"}); err != nil {
		t.Fatalf("fork: %v", err)
	}

	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}

	stale, err := FindStale(store)
	if err != nil {
		t.Fatalf("FindStale() error = %v", err)
	}
	if len(stale) != 1 {
		t.Fatalf("expected 1 stale, got %d", len(stale))
	}
	if stale[0].Name != "wt-orphan" {
		t.Errorf("Name = %q, want %q", stale[0].Name, "wt-orphan")
	}
}

func TestPruneStaleDeletesAndCleansParent(t *testing.T) {
	store := initPruneTest(t)
	mgr, dir := mkRepo(t, store, "myrepo")

	if _, err := mgr.Fork(ForkOptions{Name: "wt-stale"}); err != nil {
		t.Fatalf("fork: %v", err)
	}

	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}

	stale, err := FindStale(store)
	if err != nil {
		t.Fatal(err)
	}

	removed, err := PruneStale(store, stale)
	if err != nil {
		t.Fatalf("PruneStale() error = %v", err)
	}
	if len(removed) != 1 {
		t.Fatalf("expected 1 removed, got %d", len(removed))
	}

	if _, err := os.Stat(stale[0].Path); !os.IsNotExist(err) {
		t.Error("expected worktree dir to be removed")
	}

	parentDir := filepath.Dir(stale[0].Path)
	if _, err := os.Stat(parentDir); !os.IsNotExist(err) {
		t.Error("expected empty parent repo dir to be removed")
	}
}

func TestPruneStaleLeavesLiveWorktrees(t *testing.T) {
	store := initPruneTest(t)

	mgrA, dirA := mkRepo(t, store, "repo-a")
	if _, err := mgrA.Fork(ForkOptions{Name: "wt-a"}); err != nil {
		t.Fatal(err)
	}

	mgrB, _ := mkRepo(t, store, "repo-b")
	if _, err := mgrB.Fork(ForkOptions{Name: "wt-b"}); err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(dirA); err != nil {
		t.Fatal(err)
	}

	stale, err := FindStale(store)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 1 {
		t.Fatalf("expected 1 stale, got %d", len(stale))
	}

	removed, err := PruneStale(store, stale)
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 1 {
		t.Fatalf("expected 1 removed, got %d", len(removed))
	}

	res, err := ListAllRepos(store)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Worktrees) != 1 {
		t.Fatalf("expected 1 live worktree, got %d", len(res.Worktrees))
	}
	if res.Worktrees[0].Name != "wt-b" {
		t.Errorf("expected live worktree wt-b, got %q", res.Worktrees[0].Name)
	}
}

func TestFindStaleBareDir(t *testing.T) {
	store := initPruneTest(t)

	bareDir := store.WorktreePath("ghost-repo", "bare-wt")
	if err := os.MkdirAll(bareDir, 0o750); err != nil {
		t.Fatal(err)
	}

	stale, err := FindStale(store)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 1 {
		t.Fatalf("expected 1 stale, got %d", len(stale))
	}

	removed, err := PruneStale(store, stale)
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 1 {
		t.Fatalf("expected 1 removed, got %d", len(removed))
	}

	if _, err := os.Stat(bareDir); !os.IsNotExist(err) {
		t.Error("expected bare dir to be removed")
	}
}

func TestFindPrunableBranchesCleanRepo(t *testing.T) {
	store := initPruneTest(t)
	mgr, _ := mkRepo(t, store, "clean-repo")

	orphans, err := FindPrunableBranches(mgr.Git(), false)
	if err != nil {
		t.Fatalf("FindPrunableBranches() error = %v", err)
	}
	if len(orphans) != 0 {
		t.Errorf("expected 0 orphans, got %d", len(orphans))
	}
}

func TestFindPrunableBranchesActiveWorktree(t *testing.T) {
	store := initPruneTest(t)
	mgr, _ := mkRepo(t, store, "active-repo")

	if _, err := mgr.Fork(ForkOptions{Name: "active-wt"}); err != nil {
		t.Fatal(err)
	}

	orphans, err := FindPrunableBranches(mgr.Git(), false)
	if err != nil {
		t.Fatalf("FindPrunableBranches() error = %v", err)
	}
	if len(orphans) != 0 {
		t.Errorf("active worktree branch should not be orphaned, got %d orphans", len(orphans))
	}
}

func TestFindPrunableBranchesPreserved(t *testing.T) {
	store := initPruneTest(t)
	mgr, _ := mkRepo(t, store, "preserved-repo")

	if _, err := mgr.Fork(ForkOptions{Name: "preserved-wt"}); err != nil {
		t.Fatal(err)
	}

	if _, err := mgr.Remove(RemoveOptions{Name: "preserved-wt", Force: true, PreserveBranch: true}); err != nil {
		t.Fatal(err)
	}

	orphans, err := FindPrunableBranches(mgr.Git(), false)
	if err != nil {
		t.Fatalf("FindPrunableBranches() error = %v", err)
	}
	if len(orphans) != 1 {
		t.Fatalf("expected 1 orphan, got %d", len(orphans))
	}
	if orphans[0].Branch != "preserved-wt" {
		t.Errorf("Branch = %q, want %q", orphans[0].Branch, "preserved-wt")
	}
	if orphans[0].Parent != "main" {
		t.Errorf("Parent = %q, want %q", orphans[0].Parent, "main")
	}
	if orphans[0].AheadCount != 0 {
		t.Errorf("AheadCount = %d, want 0", orphans[0].AheadCount)
	}
}

func TestFindPrunableBranchesAheadCount(t *testing.T) {
	store := initPruneTest(t)
	mgr, _ := mkRepo(t, store, "ahead-repo")
	r := testutil.NewGitRepo(t, mgr.Git().Dir())

	wt, err := mgr.Fork(ForkOptions{Name: "ahead-wt"})
	if err != nil {
		t.Fatal(err)
	}

	wtRepo := testutil.NewGitRepo(t, wt.WorktreePath)
	wtRepo.CommitFile(t, "extra.txt", "add extra file")
	wtRepo.CommitFile(t, "extra2.txt", "add another file")
	_ = r

	if _, err := mgr.Remove(RemoveOptions{Name: "ahead-wt", Force: true, PreserveBranch: true}); err != nil {
		t.Fatal(err)
	}

	orphans, err := FindPrunableBranches(mgr.Git(), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(orphans) != 1 {
		t.Fatalf("expected 1 orphan, got %d", len(orphans))
	}
	if orphans[0].AheadCount != 2 {
		t.Errorf("AheadCount = %d, want 2", orphans[0].AheadCount)
	}
}

func TestFindPrunableBranchesParentGone(t *testing.T) {
	store := initPruneTest(t)
	mgr, _ := mkRepo(t, store, "parent-gone-repo")
	g := mgr.Git()

	if err := g.CreateBranch("feature", ""); err != nil {
		t.Fatal(err)
	}
	if err := g.Switch("feature"); err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{Name: "child-wt"})
	if err != nil {
		t.Fatal(err)
	}
	_ = wt

	if err := g.Switch("main"); err != nil {
		t.Fatal(err)
	}

	if _, err := mgr.Remove(RemoveOptions{Name: "child-wt", Force: true, PreserveBranch: true}); err != nil {
		t.Fatal(err)
	}

	if err := g.DeleteBranch("feature", true); err != nil {
		t.Fatal(err)
	}

	orphans, err := FindPrunableBranches(g, false)
	if err != nil {
		t.Fatalf("FindPrunableBranches() error = %v", err)
	}
	if len(orphans) != 1 {
		t.Fatalf("expected 1 orphan, got %d", len(orphans))
	}
	if orphans[0].AheadCount != -1 {
		t.Errorf("AheadCount = %d, want -1 (parent gone)", orphans[0].AheadCount)
	}
}

func TestFindPrunableBranchesSkipsProtected(t *testing.T) {
	store := initPruneTest(t)
	mgr, _ := mkRepo(t, store, "protected-repo")
	g := mgr.Git()

	// Manually set wt-parent on main (shouldn't happen normally, but guards against it)
	if err := g.SetBranchMeta("main", git.BranchMeta{Parent: "main"}); err != nil {
		t.Fatal(err)
	}

	orphans, err := FindPrunableBranches(g, false)
	if err != nil {
		t.Fatalf("FindPrunableBranches() error = %v", err)
	}
	if len(orphans) != 0 {
		t.Errorf("protected branches should be skipped, got %d orphans", len(orphans))
	}
}

// TestFindPrunableBranchesLandedWithoutParent covers a branch that is CreateBranch
// with raw git (no wt-parent), pushed and then merged+deleted on the remote, its
// prunable purely because its upstream is gone.
func TestFindPrunableBranchesLandedWithoutParent(t *testing.T) {
	e := setupRebranchEnv(t)

	e.r.Run("checkout", "-b", "REL-70869")
	e.r.CommitFile(t, "rel.txt", "release work")
	e.r.Run("push", "-u", "origin", "REL-70869")
	e.r.Run("checkout", "main")
	e.r.Run("push", "origin", "--delete", "REL-70869")
	e.r.Run("fetch", "--prune", "origin")

	prunable, err := FindPrunableBranches(e.mgr.Git(), false)
	if err != nil {
		t.Fatalf("FindPrunableBranches() error = %v", err)
	}
	if len(prunable) != 1 {
		t.Fatalf("expected 1 prunable branch, got %d", len(prunable))
	}
	b := prunable[0]
	if b.Branch != "REL-70869" {
		t.Errorf("Branch = %q, want %q", b.Branch, "REL-70869")
	}
	if !b.Landed {
		t.Error("expected Landed = true for branch with upstream gone")
	}
	if b.Parent != "" {
		t.Errorf("Parent = %q, want empty (no wt-parent)", b.Parent)
	}
}

// TestFindPrunableBranchesLandedOnly verifies that the landedOnly filter drops
// wt-managed-but-not-landed branches, keeping only landed ones.
func TestFindPrunableBranchesLandedOnly(t *testing.T) {
	e := setupRebranchEnv(t)

	// A landed branch (wt-managed, pushed, merged, remote deleted).
	e.forkAndPush(t, "landed-wt")
	e.land(t, "landed-wt")
	if _, err := e.mgr.Remove(RemoveOptions{Name: "landed-wt", Force: true, PreserveBranch: true}); err != nil {
		t.Fatal(err)
	}

	// A wt-managed branch that was never landed (no worktree, parent present).
	if _, err := e.mgr.Fork(ForkOptions{Name: "open-wt"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.mgr.Remove(RemoveOptions{Name: "open-wt", Force: true, PreserveBranch: true}); err != nil {
		t.Fatal(err)
	}

	all, err := FindPrunableBranches(e.mgr.Git(), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("landedOnly=false expected 2 branches, got %d: %+v", len(all), all)
	}

	landed, err := FindPrunableBranches(e.mgr.Git(), true)
	if err != nil {
		t.Fatal(err)
	}
	if len(landed) != 1 {
		t.Fatalf("landedOnly=true expected 1 branch, got %d: %+v", len(landed), landed)
	}
	if landed[0].Branch != "landed-wt" || !landed[0].Landed {
		t.Errorf("expected only landed-wt, got %+v", landed[0])
	}
}

// TestFindPrunableBranchesPushedNotLanded covers a wt-managed branch that was
// pushed but whose upstream still exists (PR still open): it is listed because
// it carries wt-parent, but not flagged as landed.
func TestFindPrunableBranchesPushedNotLanded(t *testing.T) {
	e := setupRebranchEnv(t)
	e.forkAndPush(t, "open-pr")

	if _, err := e.mgr.Remove(RemoveOptions{Name: "open-pr", Force: true, PreserveBranch: true}); err != nil {
		t.Fatal(err)
	}

	prunable, err := FindPrunableBranches(e.mgr.Git(), false)
	if err != nil {
		t.Fatalf("FindPrunableBranches() error = %v", err)
	}
	if len(prunable) != 1 {
		t.Fatalf("expected 1 prunable branch, got %d", len(prunable))
	}
	b := prunable[0]
	if b.Landed {
		t.Error("expected Landed = false while upstream still exists")
	}
	if !b.HasUpstream {
		t.Error("expected HasUpstream = true for a pushed branch")
	}
}

func TestDeletePrunableBranches(t *testing.T) {
	store := initPruneTest(t)
	mgr, _ := mkRepo(t, store, "delete-repo")
	g := mgr.Git()

	if _, err := mgr.Fork(ForkOptions{Name: "del-wt"}); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Remove(RemoveOptions{Name: "del-wt", Force: true, PreserveBranch: true}); err != nil {
		t.Fatal(err)
	}

	orphans, err := FindPrunableBranches(g, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(orphans) != 1 {
		t.Fatalf("expected 1 orphan, got %d", len(orphans))
	}

	deleted, err := DeletePrunableBranches(g, orphans)
	if err != nil {
		t.Fatalf("DeletePrunableBranches() error = %v", err)
	}
	if len(deleted) != 1 {
		t.Fatalf("expected 1 deleted, got %d", len(deleted))
	}

	exists, _ := g.LocalBranchExists("del-wt")
	if exists {
		t.Error("branch should be deleted")
	}

	meta, err := g.GetBranchMeta("del-wt")
	if err != nil {
		t.Fatal(err)
	}
	if meta.Parent != "" {
		t.Errorf("config should be cleaned up, got Parent=%q", meta.Parent)
	}
}

func TestResolveReposInjectsCwd(t *testing.T) {
	store := initPruneTest(t)
	_, dir := mkRepo(t, store, "cwd-repo")

	repos, err := ResolveRepos(store, dir)
	if err != nil {
		t.Fatalf("ResolveRepos() error = %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}
	resolvedDir, _ := filepath.EvalSymlinks(dir)
	if repos[0] != resolvedDir {
		t.Errorf("expected %q, got %q", resolvedDir, repos[0])
	}

	cfg, _, _ := store.LoadConfig()
	if len(cfg.Repos) != 1 {
		t.Errorf("expected repos persisted, got %d", len(cfg.Repos))
	}
}

func TestResolveReposRemovesDeadPaths(t *testing.T) {
	store := initPruneTest(t)
	_, liveDir := mkRepo(t, store, "live-repo")

	_ = store.RegisterRepo(liveDir)
	_ = store.RegisterRepo("/nonexistent/dead-repo")

	repos, err := ResolveRepos(store, "")
	if err != nil {
		t.Fatalf("ResolveRepos() error = %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 live repo, got %d: %v", len(repos), repos)
	}
	resolvedLive, _ := filepath.EvalSymlinks(liveDir)
	if repos[0] != resolvedLive {
		t.Errorf("expected %q, got %q", resolvedLive, repos[0])
	}

	cfg, _, _ := store.LoadConfig()
	if len(cfg.Repos) != 1 {
		t.Errorf("dead path should be removed from config, got %d entries", len(cfg.Repos))
	}
}

func TestResolveReposDeduplicates(t *testing.T) {
	store := initPruneTest(t)
	_, dir := mkRepo(t, store, "dup-repo")

	_ = store.RegisterRepo(dir)
	_ = store.RegisterRepo(dir)

	repos, err := ResolveRepos(store, dir)
	if err != nil {
		t.Fatalf("ResolveRepos() error = %v", err)
	}
	if len(repos) != 1 {
		t.Errorf("expected 1 repo after dedup, got %d", len(repos))
	}
}
