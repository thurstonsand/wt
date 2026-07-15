//go:build integration

// Package testutil provides shared test utilities.
package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// GitRepo is a test git repository.
type GitRepo struct {
	Dir string
	t   *testing.T
}

// NewGitRepo creates a GitRepo for the given directory without initializing it.
func NewGitRepo(t *testing.T, dir string) *GitRepo {
	t.Helper()
	return &GitRepo{Dir: dir, t: t}
}

// Init creates a git repository at Dir with an initial commit on "main".
func (r *GitRepo) Init() {
	r.t.Helper()
	r.Run("init", "--initial-branch=main")
	r.Run("config", "user.email", "test@test.com")
	r.Run("config", "user.name", "Test")
	r.WriteFile("README.md", "# Test\n")
	r.Run("add", "README.md")
	r.Run("commit", "-m", "initial")
}

// InitGitRepo creates a new git repository in a temp directory with isolated config.
// The repo has one initial commit and uses "main" as the default branch.
// Sets WT_HOME to a temp directory for isolated worktree storage.
// Changes to the repo directory and restores the original on test cleanup.
func InitGitRepo(t *testing.T) *GitRepo {
	t.Helper()

	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	t.Setenv("WT_HOME", t.TempDir())

	r := NewGitRepo(t, t.TempDir())

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(r.Dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	r.Init()

	return r
}

// Run executes a git command in the test repo, failing the test on error.
func (r *GitRepo) Run(args ...string) string {
	r.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir

	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %v failed: %v\noutput: %s", args, err, out)
	}
	return string(out)
}

// WriteFile creates a file in the repo directory.
func (r *GitRepo) WriteFile(name, content string) {
	r.t.Helper()
	path := filepath.Join(r.Dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		r.t.Fatal(err)
	}
}

// CommitFile creates a file and commits it.
func (r *GitRepo) CommitFile(t *testing.T, name, message string) {
	t.Helper()
	r.WriteFile(name, message+"\n")
	r.Run("add", name)
	r.Run("commit", "-m", message)
}

// AtWorktree returns a GitRepo pointing at a worktree directory.
func (r *GitRepo) AtWorktree(path string) *GitRepo {
	return &GitRepo{
		Dir: path,
		t:   r.t,
	}
}
