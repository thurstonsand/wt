//go:build integration

package worktree

import (
	"os"
	"path/filepath"
	"testing"
)

// A gitignored file matching an include pattern must land even on a normal
// (non-clean) fork, where dirty-state transfer deliberately skips it. A
// gitignored file matching nothing must stay behind.
func TestForkIncludeCarriesGitignoredFile(t *testing.T) {
	mgr, r := newTestManager(t)
	r.WriteFile(".gitignore", ".env\n*.log\n")
	r.Run("add", ".gitignore")
	r.Run("commit", "-m", "gitignore")
	r.WriteFile(".env", "SECRET=x")
	r.WriteFile("debug.log", "noise")
	r.WriteFile(".worktreeinclude", ".env\n")

	wt, err := mgr.Fork(ForkOptions{Name: "inc"})
	if err != nil {
		t.Fatal(err)
	}

	assertFileExists(t, wt.WorktreePath, ".env")
	assertFileAbsent(t, wt.WorktreePath, "debug.log")
}

// Include files land on a clean fork; dirty-state (untracked, not matched) does
// not.
func TestForkCleanCopiesIncludeOnly(t *testing.T) {
	mgr, r := newTestManager(t)
	r.WriteFile(".env", "SECRET=x")
	r.WriteFile("scratch.txt", "data")
	r.WriteFile(".worktreeinclude", ".env\n")

	clean := true
	wt, err := mgr.Fork(ForkOptions{Name: "clean-inc", Clean: &clean})
	if err != nil {
		t.Fatal(err)
	}

	assertFileExists(t, wt.WorktreePath, ".env")
	assertFileAbsent(t, wt.WorktreePath, "scratch.txt")
}

// A locally-modified tracked file matched by an include must carry its
// working-tree content into a clean fork (the supported anti-pattern).
func TestForkIncludeCarriesModifiedTrackedFile(t *testing.T) {
	mgr, r := newTestManager(t)
	r.WriteFile("settings.json", "committed\n")
	r.Run("add", "settings.json")
	r.Run("commit", "-m", "add settings")
	r.WriteFile("settings.json", "LOCAL OVERRIDE\n")
	r.WriteFile(".worktreeinclude", "settings.json\n")

	clean := true
	wt, err := mgr.Fork(ForkOptions{Name: "tracked-mod", Clean: &clean})
	if err != nil {
		t.Fatal(err)
	}

	assertFileContent(t, wt.WorktreePath, "settings.json", "LOCAL OVERRIDE\n")
}

// User-level worktreeinclude (in WT_HOME) drives copying too.
func TestForkUserLevelInclude(t *testing.T) {
	mgr, r := newTestManager(t)
	r.WriteFile(".env", "SECRET=x")
	if err := os.WriteFile(mgr.Store().UserIncludePath(), []byte(".env\n"), 0600); err != nil {
		t.Fatal(err)
	}

	clean := true
	wt, err := mgr.Fork(ForkOptions{Name: "user-inc", Clean: &clean})
	if err != nil {
		t.Fatal(err)
	}

	assertFileExists(t, wt.WorktreePath, ".env")
}

// A --with negation overrides a user-level include (last-wins precedence).
func TestForkWithNegationOverridesInclude(t *testing.T) {
	mgr, r := newTestManager(t)
	r.WriteFile(".env", "SECRET=x")
	r.WriteFile(".env.local", "LOCAL=y")
	if err := os.WriteFile(mgr.Store().UserIncludePath(), []byte(".env*\n"), 0600); err != nil {
		t.Fatal(err)
	}

	clean := true
	wt, err := mgr.Fork(ForkOptions{Name: "negate", Clean: &clean, With: []string{"!.env.local"}})
	if err != nil {
		t.Fatal(err)
	}

	assertFileExists(t, wt.WorktreePath, ".env")
	assertFileAbsent(t, wt.WorktreePath, ".env.local")
}

func TestForkCopiesUntrackedByDefault(t *testing.T) {
	mgr, r := newTestManager(t)
	r.WriteFile("untracked.txt", "data")

	wt, err := mgr.Fork(ForkOptions{Name: "test-untracked"})
	if err != nil {
		t.Fatal(err)
	}

	assertFileExists(t, wt.WorktreePath, "untracked.txt")
}

func TestForkWithBaseDisablesUntracked(t *testing.T) {
	mgr, r := newTestManager(t)
	r.WriteFile("untracked.txt", "data")
	if err := mgr.Git().CreateBranch("develop", "HEAD"); err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{Name: "from-base", Base: "develop"})
	if err != nil {
		t.Fatal(err)
	}

	assertFileAbsent(t, wt.WorktreePath, "untracked.txt")
}

func assertFileExists(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist in %s", name, dir)
	}
}

func assertFileAbsent(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected file %s to NOT exist in %s", name, dir)
	}
}

func assertFileContent(t *testing.T, dir, name, want string) {
	t.Helper()
	path := filepath.Join(dir, name)
	data, err := os.ReadFile(path) //nolint:gosec // test file
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	if string(data) != want {
		t.Errorf("expected %s content %q, got %q", name, want, string(data))
	}
}
