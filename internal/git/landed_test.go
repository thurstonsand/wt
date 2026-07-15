//go:build integration

package git

import (
	"path/filepath"
	"testing"

	"github.com/thurstonsand/wt/internal/testutil"
)

// setupRemote creates a bare remote, wires it as origin, and pushes main.
func setupRemote(t *testing.T, r *testutil.GitRepo) string {
	t.Helper()
	remoteDir := t.TempDir()
	bare := testutil.NewGitRepo(t, remoteDir)
	bare.Run("init", "--bare", "--initial-branch=main")
	r.Run("remote", "add", "origin", remoteDir)
	r.Run("push", "-u", "origin", "main")
	return remoteDir
}

func TestHasUpstream(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)
	setupRemote(t, r)

	r.Run("checkout", "-b", "feature")
	has, err := g.HasUpstream("feature")
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Error("feature has no upstream before push")
	}

	r.Run("push", "-u", "origin", "feature")
	has, err = g.HasUpstream("feature")
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Error("feature should have upstream after push -u")
	}
}

func TestIsLandedPushedThenDeleted(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)
	setupRemote(t, r)

	r.Run("checkout", "-b", "feature")
	r.CommitFile(t, "f.txt", "work")
	r.Run("push", "-u", "origin", "feature")

	landed, err := g.UpstreamGone("feature")
	if err != nil {
		t.Fatal(err)
	}
	if landed {
		t.Error("feature is not landed while remote branch exists")
	}

	r.Run("push", "origin", "--delete", "feature")
	r.Run("fetch", "--prune", "origin")

	landed, err = g.UpstreamGone("feature")
	if err != nil {
		t.Fatal(err)
	}
	if !landed {
		t.Error("feature should be landed after remote delete + prune")
	}
}

func TestIsLandedNeverPushed(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)
	setupRemote(t, r)

	r.Run("checkout", "-b", "local-only")
	r.CommitFile(t, "f.txt", "work")

	landed, err := g.UpstreamGone("local-only")
	if err != nil {
		t.Fatal(err)
	}
	if landed {
		t.Error("never-pushed branch is not landed")
	}
}

func TestIsLandedTracksDifferentUpstream(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)
	setupRemote(t, r)

	// A branch created from origin/main tracks origin/main, not origin/<itself>.
	// It must not be mistaken for landed just because origin/<its-name> is absent.
	r.Run("switch", "-C", "fresh", "origin/main")

	landed, err := g.UpstreamGone("fresh")
	if err != nil {
		t.Fatal(err)
	}
	if landed {
		t.Error("branch tracking a live origin/main must not be reported as landed")
	}
}

func TestRemoteTrackingExists(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)
	setupRemote(t, r)

	exists, err := g.RemoteTrackingExists("origin", "main")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("origin/main tracking ref should exist")
	}

	exists, err = g.RemoteTrackingExists("origin", "nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("origin/nonexistent should not exist")
	}
}

func TestDefaultRemoteBranch(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)
	setupRemote(t, r)

	branch, err := g.DefaultRemoteBranch("origin")
	if err != nil {
		t.Fatal(err)
	}
	if branch != "main" {
		t.Errorf("expected main, got %q", branch)
	}
}

func TestDefaultRemoteBranchSymbolicHead(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)
	setupRemote(t, r)

	// Point origin/HEAD explicitly and confirm it is honored.
	r.Run("remote", "set-head", "origin", "main")
	headPath := filepath.Join(r.Dir, ".git", "refs", "remotes", "origin", "HEAD")
	if _, err := g.run(runOpts{}, "symbolic-ref", "refs/remotes/origin/HEAD"); err != nil {
		t.Fatalf("expected origin/HEAD symbolic ref at %s: %v", headPath, err)
	}

	branch, err := g.DefaultRemoteBranch("origin")
	if err != nil {
		t.Fatal(err)
	}
	if branch != "main" {
		t.Errorf("expected main, got %q", branch)
	}
}
