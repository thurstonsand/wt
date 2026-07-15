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

func newTestManager(t *testing.T) (*Manager, *testutil.GitRepo) {
	t.Helper()
	repo := testutil.InitGitRepo(t)
	s := config.NewGlobalStore(t.TempDir())
	g := git.New(repo.Dir)
	m, err := NewManager(repo.Dir, WithGit(g), WithGlobalStore(s))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	return m, repo
}

func TestListAllEmpty(t *testing.T) {
	m, repo := newTestManager(t)

	result, err := m.ListAll()
	if err != nil {
		t.Fatalf("ListAll() error = %v", err)
	}
	if len(result.Worktrees) != 0 {
		t.Errorf("expected empty worktrees list, got %d items", len(result.Worktrees))
	}
	repoDir, _ := filepath.EvalSymlinks(repo.Dir)
	if result.Main.Path != repoDir {
		t.Errorf("Main.Path = %q, want %q", result.Main.Path, repoDir)
	}
	if result.Main.Branch != "main" {
		t.Errorf("Main.Branch = %q, want %q", result.Main.Branch, "main")
	}
}

func TestExistsNotFound(t *testing.T) {
	m, _ := newTestManager(t)

	if m.Exists("nonexistent") {
		t.Error("expected Exists() = false for nonexistent worktree")
	}
}

func TestGetNotFound(t *testing.T) {
	m, _ := newTestManager(t)

	_, err := m.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent worktree")
	}
}

func TestListAllWithWorktree(t *testing.T) {
	m, repo := newTestManager(t)

	wt, err := m.Fork(ForkOptions{Name: "wt-test"})
	if err != nil {
		t.Fatalf("failed to fork: %v", err)
	}

	result, err := m.ListAll()
	if err != nil {
		t.Fatalf("ListAll() error = %v", err)
	}

	repoDir, _ := filepath.EvalSymlinks(repo.Dir)
	if result.Main.Path != repoDir {
		t.Errorf("Main.Path = %q, want %q", result.Main.Path, repoDir)
	}
	if result.Main.Branch != "main" {
		t.Errorf("Main.Branch = %q, want %q", result.Main.Branch, "main")
	}

	if len(result.Worktrees) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(result.Worktrees))
	}

	if result.Worktrees[0].Name != "wt-test" {
		t.Errorf("Name = %q, want %q", result.Worktrees[0].Name, "wt-test")
	}
	if result.Worktrees[0].Branch != "wt-test" {
		t.Errorf("Branch = %q, want %q", result.Worktrees[0].Branch, "wt-test")
	}
	if result.Worktrees[0].Parent != "main" {
		t.Errorf("Parent = %q, want %q", result.Worktrees[0].Parent, "main")
	}
	if result.Worktrees[0].WorktreePath != wt.WorktreePath {
		t.Errorf("WorktreePath = %q, want %q", result.Worktrees[0].WorktreePath, wt.WorktreePath)
	}
}

func TestExistsWithWorktree(t *testing.T) {
	m, _ := newTestManager(t)

	_, err := m.Fork(ForkOptions{Name: "wt-exists"})
	if err != nil {
		t.Fatalf("failed to fork: %v", err)
	}

	if !m.Exists("wt-exists") {
		t.Error("expected Exists() = true")
	}
}

func TestGetWithWorktree(t *testing.T) {
	m, _ := newTestManager(t)

	created, err := m.Fork(ForkOptions{Name: "wt-get"})
	if err != nil {
		t.Fatalf("failed to fork: %v", err)
	}

	wt, err := m.Get("wt-get")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if wt.Name != "wt-get" {
		t.Errorf("Name = %q, want %q", wt.Name, "wt-get")
	}
	if wt.WorktreePath != created.WorktreePath {
		t.Errorf("WorktreePath = %q, want %q", wt.WorktreePath, created.WorktreePath)
	}
	if wt.Parent != "main" {
		t.Errorf("Parent = %q, want %q", wt.Parent, "main")
	}
}

func TestGetByBranchName(t *testing.T) {
	m, _ := newTestManager(t)

	_, err := m.Fork(ForkOptions{Name: "feature-branch"})
	if err != nil {
		t.Fatalf("failed to fork: %v", err)
	}

	wt, err := m.Get("feature-branch")
	if err != nil {
		t.Fatalf("Get() by branch name error = %v", err)
	}

	if wt.Branch != "feature-branch" {
		t.Errorf("Branch = %q, want %q", wt.Branch, "feature-branch")
	}
}

func TestGetUnsanitizedBranchName(t *testing.T) {
	m, _ := newTestManager(t)

	_, err := m.Fork(ForkOptions{Name: "fix/bug-123"})
	if err != nil {
		t.Fatalf("failed to fork: %v", err)
	}

	wt, err := m.Get("fix--bug-123")
	if err != nil {
		t.Fatalf("Get() by sanitized name error = %v", err)
	}

	if wt.Branch != "fix/bug-123" {
		t.Errorf("Branch = %q, want %q", wt.Branch, "fix/bug-123")
	}
}

func TestListAllRepoName(t *testing.T) {
	m, _ := newTestManager(t)

	_, err := m.Fork(ForkOptions{Name: "wt-repo"})
	if err != nil {
		t.Fatalf("failed to fork: %v", err)
	}

	result, err := m.ListAll()
	if err != nil {
		t.Fatalf("ListAll() error = %v", err)
	}
	if len(result.Worktrees) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Worktrees))
	}
	if result.Worktrees[0].RepoName != m.RepoName() {
		t.Errorf("RepoName = %q, want %q", result.Worktrees[0].RepoName, m.RepoName())
	}

	wt, err := m.Get("wt-repo")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if wt.RepoName != m.RepoName() {
		t.Errorf("Get RepoName = %q, want %q", wt.RepoName, m.RepoName())
	}
}

func TestListAllRepos(t *testing.T) {
	wtHome := t.TempDir()
	t.Setenv("WT_HOME", wtHome)
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")

	store := config.NewGlobalStore(wtHome)

	mkRepo := func(name string) string {
		t.Helper()
		dir := filepath.Join(t.TempDir(), name)
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatal(err)
		}
		testutil.NewGitRepo(t, dir).Init()
		return dir
	}

	dirA := mkRepo("repo-a")
	mgrA, err := NewManager(dirA, WithGit(git.New(dirA)), WithGlobalStore(store))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mgrA.Fork(ForkOptions{Name: "wt-a1"}); err != nil {
		t.Fatalf("fork repo-a: %v", err)
	}

	dirB := mkRepo("repo-b")
	mgrB, err := NewManager(dirB, WithGit(git.New(dirB)), WithGlobalStore(store))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mgrB.Fork(ForkOptions{Name: "wt-b1"}); err != nil {
		t.Fatalf("fork repo-b: %v", err)
	}

	res, err := ListAllRepos(store)
	if err != nil {
		t.Fatalf("ListAllRepos() error = %v", err)
	}
	if len(res.Worktrees) != 2 {
		t.Fatalf("expected 2 items, got %d", len(res.Worktrees))
	}

	found := map[string]bool{}
	for _, wt := range res.Worktrees {
		found[wt.RepoName+"/"+wt.Name] = true
	}
	for _, key := range []string{"repo-a/wt-a1", "repo-b/wt-b1"} {
		if !found[key] {
			t.Errorf("expected %q in results", key)
		}
	}
}

func TestListAllReposSkipsStale(t *testing.T) {
	wtHome := t.TempDir()
	store := config.NewGlobalStore(wtHome)

	base := filepath.Join(wtHome, "worktrees", "stale-repo", "broken-wt")
	if err := os.MkdirAll(base, 0750); err != nil {
		t.Fatal(err)
	}

	res, err := ListAllRepos(store)
	if err != nil {
		t.Fatalf("ListAllRepos() error = %v", err)
	}
	if len(res.Worktrees) != 0 {
		t.Errorf("expected 0 worktrees, got %d", len(res.Worktrees))
	}
	if len(res.Stale) != 1 {
		t.Fatalf("expected 1 stale entry, got %d", len(res.Stale))
	}
	if res.Stale[0].RepoName != "stale-repo" {
		t.Errorf("Stale RepoName = %q, want %q", res.Stale[0].RepoName, "stale-repo")
	}
}
