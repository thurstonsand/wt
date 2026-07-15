//go:build integration

package git

import (
	"testing"

	"github.com/thurstonsand/wt/internal/testutil"
)

func TestSetAndGetBranchMeta(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	if err := g.CreateBranch("test-branch", "HEAD"); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	meta := BranchMeta{Parent: "main"}
	if err := g.SetBranchMeta("test-branch", meta); err != nil {
		t.Fatalf("SetBranchMeta() error = %v", err)
	}

	got, err := g.GetBranchMeta("test-branch")
	if err != nil {
		t.Fatalf("GetBranchMeta() error = %v", err)
	}

	if got.Parent != "main" {
		t.Errorf("Parent = %q, want %q", got.Parent, "main")
	}
}

func TestGetBranchMetaNotFound(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	if err := g.CreateBranch("no-meta", "HEAD"); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	meta, err := g.GetBranchMeta("no-meta")
	if err != nil {
		t.Fatalf("GetBranchMeta() should not error for missing keys: %v", err)
	}

	if meta.Parent != "" {
		t.Errorf("Parent should be empty for branch without metadata, got %q", meta.Parent)
	}
}
