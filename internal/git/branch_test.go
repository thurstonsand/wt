//go:build integration

package git

import (
	"testing"

	"github.com/thurstonsand/wt/internal/testutil"
)

func TestCurrentBranch(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	branch, err := g.CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	if branch != "main" {
		t.Errorf("expected main, got %q", branch)
	}
}

func TestDefaultBranch(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	branch, err := g.DefaultBranch()
	if err != nil {
		t.Fatal(err)
	}
	if branch != "main" {
		t.Errorf("expected main, got %q", branch)
	}
}

func TestDefaultBranchPreferMain(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	if err := g.CreateBranch("master", ""); err != nil {
		t.Fatal(err)
	}

	branch, err := g.DefaultBranch()
	if err != nil {
		t.Fatal(err)
	}
	if branch != "main" {
		t.Errorf("expected main (preferred over master), got %q", branch)
	}
}

func TestIsDirty(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	dirty, err := g.IsDirty()
	if err != nil {
		t.Fatal(err)
	}
	if dirty {
		t.Error("expected clean repo")
	}

	r.WriteFile("new.txt", "content")

	dirty, err = g.IsDirty()
	if err != nil {
		t.Fatal(err)
	}
	if !dirty {
		t.Error("expected dirty repo")
	}
}

func TestLocalBranchExists(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	exists, err := g.LocalBranchExists("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("expected nonexistent branch to not exist")
	}

	if err := g.CreateBranch("feature", ""); err != nil {
		t.Fatal(err)
	}

	exists, err = g.LocalBranchExists("feature")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("expected feature branch to exist")
	}
}

func TestCheckout(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	if err := g.CreateBranch("feature", ""); err != nil {
		t.Fatal(err)
	}

	if err := g.Switch("feature"); err != nil {
		t.Fatal(err)
	}

	branch, err := g.CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	if branch != "feature" {
		t.Errorf("expected feature, got %q", branch)
	}
}

func TestCheckoutNonexistent(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	err := g.Switch("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent branch")
	}
}

func TestErrNotGitRepo(t *testing.T) {
	g := New(t.TempDir())

	_, err := g.CurrentBranch()
	if !IsErrNotGitRepo(err) {
		t.Errorf("expected ErrNotGitRepo, got %v", err)
	}
}

func TestUntrackedFiles(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	files, err := g.UntrackedFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("expected no untracked files, got %v", files)
	}

	r.WriteFile("untracked.txt", "data")
	r.WriteFile("subdir/another.txt", "more data")

	files, err = g.UntrackedFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 untracked files, got %v", files)
	}
}

func TestUntrackedFilesRespectsGitignore(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	r.WriteFile(".gitignore", "*.log\n")
	r.Run("add", ".gitignore")
	r.Run("commit", "-m", "add gitignore")

	r.WriteFile("keep.txt", "keep")
	r.WriteFile("ignore.log", "ignore")

	files, err := g.UntrackedFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 untracked file, got %v", files)
	}
	if files[0] != "keep.txt" {
		t.Errorf("expected keep.txt, got %s", files[0])
	}
}

func TestFilesMatchingPatterns(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	// tracked-and-committed, then locally modified
	r.WriteFile("settings.json", "committed\n")
	r.Run("add", "settings.json")
	r.Run("commit", "-m", "settings")
	r.WriteFile("settings.json", "local\n")
	// gitignored + untracked
	r.WriteFile(".gitignore", ".env\n*.log\n")
	r.WriteFile(".env", "SECRET=x")
	r.WriteFile("debug.log", "noise")
	// untracked, not ignored, unmatched
	r.WriteFile("notes.txt", "data")

	t.Run("empty patterns yield nothing", func(t *testing.T) {
		files, err := g.FilesMatchingPatterns(nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 0 {
			t.Errorf("expected no files, got %v", files)
		}
	})

	t.Run("matches tracked and gitignored, ignores unmatched", func(t *testing.T) {
		files, err := g.FilesMatchingPatterns([]string{".env", "settings.json"})
		if err != nil {
			t.Fatal(err)
		}
		got := map[string]bool{}
		for _, f := range files {
			got[f] = true
		}
		if !got[".env"] {
			t.Errorf("expected gitignored .env to match, got %v", files)
		}
		if !got["settings.json"] {
			t.Errorf("expected tracked settings.json to match, got %v", files)
		}
		if got["debug.log"] || got["notes.txt"] {
			t.Errorf("expected unmatched files excluded, got %v", files)
		}
	})

	t.Run("later negation overrides earlier include", func(t *testing.T) {
		r.WriteFile(".env.local", "LOCAL=y")
		files, err := g.FilesMatchingPatterns([]string{".env*", "!.env.local"})
		if err != nil {
			t.Fatal(err)
		}
		got := map[string]bool{}
		for _, f := range files {
			got[f] = true
		}
		if !got[".env"] {
			t.Errorf("expected .env to match, got %v", files)
		}
		if got[".env.local"] {
			t.Errorf("expected .env.local excluded by negation, got %v", files)
		}
	})
}

func TestListBranches(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	branches, err := g.ListBranches(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(branches) != 1 || branches[0] != "main" {
		t.Errorf("expected [main], got %v", branches)
	}

	if err := g.CreateBranch("alpha", ""); err != nil {
		t.Fatal(err)
	}
	if err := g.CreateBranch("beta", ""); err != nil {
		t.Fatal(err)
	}

	branches, err = g.ListBranches(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(branches) != 3 {
		t.Fatalf("expected 3 branches, got %v", branches)
	}
	want := map[string]bool{"alpha": true, "beta": true, "main": true}
	for _, b := range branches {
		if !want[b] {
			t.Errorf("unexpected branch %q", b)
		}
	}
}

func TestListBranchesIncludeRemote(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)
	setupRemote(t, r)
	r.Run("push", "origin", "main:remote-only")
	r.Run("fetch", "origin")

	local, err := g.ListBranches(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(local) != 1 || local[0] != "main" {
		t.Errorf("local-only expected [main], got %v", local)
	}

	all, err := g.ListBranches(true)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"main": true, "remote-only": true}
	if len(all) != len(want) {
		t.Fatalf("expected %d branches, got %v", len(want), all)
	}
	for _, b := range all {
		if !want[b] {
			t.Errorf("unexpected branch %q", b)
		}
	}
}

func TestDeleteBranch(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	if err := g.CreateBranch("feature", ""); err != nil {
		t.Fatal(err)
	}

	exists, _ := g.LocalBranchExists("feature")
	if !exists {
		t.Fatal("expected feature branch to exist before delete")
	}

	if err := g.DeleteBranch("feature", false); err != nil {
		t.Fatal(err)
	}

	exists, _ = g.LocalBranchExists("feature")
	if exists {
		t.Error("expected feature branch to be deleted")
	}
}

func TestDeleteBranchForce(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	if err := g.CreateBranch("feature", ""); err != nil {
		t.Fatal(err)
	}
	if err := g.Switch("feature"); err != nil {
		t.Fatal(err)
	}
	r.CommitFile(t, "feature-only.txt", "feature commit")
	if err := g.Switch("main"); err != nil {
		t.Fatal(err)
	}

	err := g.DeleteBranch("feature", false)
	if err == nil {
		t.Error("expected error deleting unmerged branch without force")
	}

	if err := g.DeleteBranch("feature", true); err != nil {
		t.Fatalf("expected force delete to succeed: %v", err)
	}

	exists, _ := g.LocalBranchExists("feature")
	if exists {
		t.Error("expected feature branch to be deleted")
	}
}
