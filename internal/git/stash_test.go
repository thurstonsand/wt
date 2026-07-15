//go:build integration

package git

import (
	"testing"

	"github.com/thurstonsand/wt/internal/testutil"
)

func TestStashPushPopRoundTrip(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	r.WriteFile("untracked.txt", "new")
	r.WriteFile("README.md", "# changed\n")

	has, err := g.StashPushAll("test")
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal("expected a stash to be created")
	}

	if dirty, _ := g.IsDirty(); dirty {
		t.Error("working tree should be clean after stash")
	}

	if err := g.StashPopIndex(); err != nil {
		t.Fatal(err)
	}
	if dirty, _ := g.IsDirty(); !dirty {
		t.Error("changes should be restored after pop")
	}
}

func TestStashPushAllNothingToStash(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	has, err := g.StashPushAll("test")
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Error("expected no stash on clean tree")
	}
}

func TestSwitchCreate(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	if err := g.SwitchCreate("feature", "main"); err != nil {
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
