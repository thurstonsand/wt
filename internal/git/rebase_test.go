//go:build integration

package git

import (
	"testing"

	"github.com/thurstonsand/wt/internal/testutil"
)

func TestRebaseAutostash(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("checkout", "-b", "feature")
	r.CommitFile(t, "feature.txt", "feature commit")
	r.Run("checkout", "main")
	r.CommitFile(t, "main.txt", "main commit")
	r.Run("checkout", "feature")

	g := New(r.Dir)

	if err := g.RebaseAutostash("main"); err != nil {
		t.Fatalf("rebase failed: %v", err)
	}

	// Verify feature commit is on top of main commit
	log := r.Run("log", "--oneline", "-2")
	if !contains(log, "feature commit") || !contains(log, "main commit") {
		t.Errorf("expected both commits in log, got: %s", log)
	}
}

func TestRebaseAutostashWithDirtyState(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("checkout", "-b", "feature")
	r.CommitFile(t, "feature.txt", "feature commit")
	r.Run("checkout", "main")
	r.CommitFile(t, "main.txt", "main commit")
	r.Run("checkout", "feature")

	// Create uncommitted changes
	r.WriteFile("uncommitted.txt", "dirty state")

	g := New(r.Dir)

	if err := g.RebaseAutostash("main"); err != nil {
		t.Fatalf("rebase with autostash failed: %v", err)
	}

	// Verify uncommitted file still exists
	dirty, _ := g.IsDirty()
	if !dirty {
		t.Error("expected dirty state to be preserved after autostash")
	}
}

func TestRebaseOnto(t *testing.T) {
	r := testutil.InitGitRepo(t)

	// Create divergent branches: main -> develop -> feature
	r.Run("checkout", "-b", "develop")
	r.CommitFile(t, "develop.txt", "develop commit")

	r.Run("checkout", "-b", "feature")
	r.CommitFile(t, "feature.txt", "feature commit")

	// Create a new target branch from main
	r.Run("checkout", "main")
	r.Run("checkout", "-b", "release")
	r.CommitFile(t, "release.txt", "release commit")

	// Go back to feature
	r.Run("checkout", "feature")

	g := New(r.Dir)

	// Rebase feature onto release, removing develop commits
	if err := g.RebaseOnto("release", "develop"); err != nil {
		t.Fatalf("rebase --onto failed: %v", err)
	}

	log := r.Run("log", "--oneline", "-3")
	if !contains(log, "feature commit") {
		t.Error("expected feature commit after rebase")
	}
	if !contains(log, "release commit") {
		t.Error("expected release commit after rebase")
	}
	if contains(log, "develop commit") {
		t.Error("develop commit should NOT be in history after --onto")
	}
}

func TestRebaseInProgress(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	inProgress, err := g.RebaseInProgress()
	if err != nil {
		t.Fatal(err)
	}
	if inProgress {
		t.Error("expected no rebase in progress")
	}
}

func TestRebaseInProgressDuringConflict(t *testing.T) {
	r := testutil.InitGitRepo(t)

	// Create conflicting branches
	r.WriteFile("conflict.txt", "main content")
	r.Run("add", "conflict.txt")
	r.Run("commit", "-m", "main version")

	r.Run("checkout", "-b", "feature", "HEAD~1")
	r.WriteFile("conflict.txt", "feature content")
	r.Run("add", "conflict.txt")
	r.Run("commit", "-m", "feature version")

	g := New(r.Dir)

	// This will fail due to conflict
	_ = g.Rebase("main")

	inProgress, err := g.RebaseInProgress()
	if err != nil {
		t.Fatal(err)
	}
	if !inProgress {
		t.Error("expected rebase in progress during conflict")
	}

	// Cleanup
	_ = g.RebaseAbort()
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
