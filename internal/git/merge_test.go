//go:build integration

package git

import (
	"testing"

	"github.com/thurstonsand/wt/internal/testutil"
)

func TestLastCommit(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	commit, err := g.LastCommit()
	if err != nil {
		t.Fatal(err)
	}
	// InitGitRepo creates "initial" commit
	if commit.Hash == "" {
		t.Error("expected non-empty hash")
	}
	if commit.Subject != "initial" {
		t.Errorf("expected 'initial', got %q", commit.Subject)
	}

	// Create a new commit and verify it's returned
	commitMsg := "test commit message"
	r.CommitFile(t, "test.txt", commitMsg)

	commit, err = g.LastCommit()
	if err != nil {
		t.Fatal(err)
	}
	if commit.Subject != commitMsg {
		t.Errorf("expected %q, got %q", commitMsg, commit.Subject)
	}
}

func TestCommitsBetween(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	base, _ := g.RevParse("HEAD")

	msg1 := "first commit"
	msg2 := "second commit"
	r.CommitFile(t, "a.txt", msg1)
	r.CommitFile(t, "b.txt", msg2)

	commits, err := g.CommitsBetween(base, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(commits))
	}
	// Log output is newest first
	if commits[0].Subject != msg2 {
		t.Errorf("commits[0] expected %q, got %q", msg2, commits[0].Subject)
	}
	if commits[1].Subject != msg1 {
		t.Errorf("commits[1] expected %q, got %q", msg1, commits[1].Subject)
	}
}

func TestCommitsBetweenEmpty(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	head, _ := g.RevParse("HEAD")

	commits, err := g.CommitsBetween(head, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 0 {
		t.Errorf("expected 0 commits, got %d", len(commits))
	}
}

func TestStagedFileCount(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	count, err := g.StagedFileCount()
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("expected 0 staged files, got %d", count)
	}

	r.WriteFile("a.txt", "a")
	r.WriteFile("b.txt", "b")
	r.Run("add", "a.txt", "b.txt")

	count, err = g.StagedFileCount()
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("expected 2 staged files, got %d", count)
	}
}

func TestParseCommitLine(t *testing.T) {
	tests := []struct {
		line    string
		hash    string
		subject string
	}{
		{"abc123 test message", "abc123", "test message"},
		{"abc123 message with spaces", "abc123", "message with spaces"},
		{"abc123", "abc123", ""},
		{"", "", ""},
	}

	for _, tc := range tests {
		got := parseCommitLine(tc.line)
		if got.Hash != tc.hash {
			t.Errorf("parseCommitLine(%q).Hash = %q, want %q", tc.line, got.Hash, tc.hash)
		}
		if got.Subject != tc.subject {
			t.Errorf("parseCommitLine(%q).Subject = %q, want %q", tc.line, got.Subject, tc.subject)
		}
	}
}

func TestDiffBranchFileCount(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	r.Run("checkout", "-b", "feature")
	r.CommitFile(t, "a.txt", "add a")
	r.CommitFile(t, "b.txt", "add b")

	count, err := g.DiffBranchFileCount("main", "feature")
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("expected 2 files, got %d", count)
	}
}

func TestDiffBranchFileCountEmpty(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	r.Run("checkout", "-b", "feature")

	count, err := g.DiffBranchFileCount("main", "feature")
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("expected 0 files, got %d", count)
	}
}
