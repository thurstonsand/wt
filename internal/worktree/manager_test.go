//go:build integration

package worktree

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thurstonsand/wt/internal/config"
	"github.com/thurstonsand/wt/internal/git"
	"github.com/thurstonsand/wt/internal/testutil"
)

func TestNewManager(t *testing.T) {
	repo := testutil.InitGitRepo(t)
	s := config.NewGlobalStore(t.TempDir())
	g := git.New(repo.Dir)
	m, err := NewManager(repo.Dir, WithGit(g), WithGlobalStore(s))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	if m.Git() == nil {
		t.Error("expected non-nil Git instance")
	}
	if m.RepoName() != filepath.Base(repo.Dir) {
		t.Errorf("RepoName() = %q, want %q", m.RepoName(), filepath.Base(repo.Dir))
	}
}

func TestNewManagerReseatsOnToplevel(t *testing.T) {
	repo := testutil.InitGitRepo(t)
	s := config.NewGlobalStore(t.TempDir())

	sub := filepath.Join(repo.Dir, "nested", "deep")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatal(err)
	}

	// No WithGit: construction must resolve the toplevel from the subdir.
	m, err := NewManager(sub, WithGlobalStore(s))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	resolvedRepo, _ := filepath.EvalSymlinks(repo.Dir)
	resolvedGit, _ := filepath.EvalSymlinks(m.Git().Dir())
	if resolvedGit != resolvedRepo {
		t.Errorf("expected source git at toplevel %q, got %q", resolvedRepo, resolvedGit)
	}
}

func TestManagerWithGlobalStore(t *testing.T) {
	repo := testutil.InitGitRepo(t)
	s := config.NewGlobalStore(t.TempDir())
	g := git.New(repo.Dir)

	m, err := NewManager(repo.Dir, WithGit(g), WithGlobalStore(s))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	if m.Store() != s {
		t.Error("expected same Store instance")
	}
}

func TestManagerWithGit(t *testing.T) {
	repo := testutil.InitGitRepo(t)
	s := config.NewGlobalStore(t.TempDir())
	g := git.New(repo.Dir)

	m, err := NewManager(repo.Dir, WithGit(g), WithGlobalStore(s))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	if m.Git() != g {
		t.Error("expected same Git instance")
	}
	if m.RepoName() != filepath.Base(repo.Dir) {
		t.Errorf("RepoName() = %q, want %q", m.RepoName(), filepath.Base(repo.Dir))
	}
}

func TestGenerateName(t *testing.T) {
	name1 := GenerateName()
	name2 := GenerateName()

	if !strings.HasPrefix(name1, "wt-") {
		t.Errorf("expected prefix 'wt-', got %q", name1)
	}
	if len(name1) != 11 { // "wt-" + 8 chars
		t.Errorf("expected length 11, got %d for %q", len(name1), name1)
	}
	if name1 == name2 {
		t.Error("expected unique names")
	}
}

func TestWorktreePath(t *testing.T) {
	repo := testutil.InitGitRepo(t)
	s := config.NewGlobalStore(t.TempDir())
	g := git.New(repo.Dir)
	m, err := NewManager(repo.Dir, WithGit(g), WithGlobalStore(s))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	path := m.WorktreePath("wt-test")
	if !strings.Contains(path, m.RepoName()) {
		t.Errorf("expected path to contain repo name, got %q", path)
	}
	if !strings.Contains(path, "wt-test") {
		t.Errorf("expected path to contain worktree name, got %q", path)
	}
}
