//go:build integration

package git

import (
	"path/filepath"
	"testing"

	"github.com/thurstonsand/wt/internal/testutil"
)

func TestCommonDir(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	commonDir, err := g.CommonDir()
	if err != nil {
		t.Fatal(err)
	}

	expected := filepath.Join(r.Dir, ".git")
	commonDirReal, _ := filepath.EvalSymlinks(commonDir)
	expectedReal, _ := filepath.EvalSymlinks(expected)
	if commonDirReal != expectedReal {
		t.Errorf("CommonDir() = %q, want %q", commonDir, expected)
	}
}

func TestCommonDirFromWorktree(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	wtPath := filepath.Join(t.TempDir(), "wt-test")
	if err := g.WorktreeAdd(wtPath, "feature", true, ""); err != nil {
		t.Fatal(err)
	}

	wtGit := New(wtPath)
	commonDir, err := wtGit.CommonDir()
	if err != nil {
		t.Fatal(err)
	}

	expected := filepath.Join(r.Dir, ".git")
	commonDirReal, _ := filepath.EvalSymlinks(commonDir)
	expectedReal, _ := filepath.EvalSymlinks(expected)
	if commonDirReal != expectedReal {
		t.Errorf("CommonDir() from worktree = %q, want %q", commonDir, expected)
	}
}

func TestWorktreeAddFromBase(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	if err := g.CreateBranch("develop", "HEAD"); err != nil {
		t.Fatal(err)
	}
	r.CommitFile(t, "second.txt", "second commit")

	wtPath := filepath.Join(t.TempDir(), "wt-from-develop")
	if err := g.WorktreeAdd(wtPath, "feature-from-develop", true, "develop"); err != nil {
		t.Fatal(err)
	}

	wtGit := New(wtPath)
	wtHead, err := wtGit.run(runOpts{}, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}

	developCommit, err := g.run(runOpts{}, "rev-parse", "develop")
	if err != nil {
		t.Fatal(err)
	}

	if wtHead != developCommit {
		t.Errorf("worktree HEAD %s != develop %s", wtHead, developCommit)
	}
}

func TestWorktreeAddRemoveList(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	list, err := g.worktreeList()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 worktree, got %d", len(list))
	}

	wtPath := filepath.Join(t.TempDir(), "wt-test")
	if err := g.WorktreeAdd(wtPath, "feature", true, ""); err != nil {
		t.Fatal(err)
	}

	list, err = g.worktreeList()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 worktrees, got %d", len(list))
	}

	var found bool
	for _, wt := range list {
		wtPathReal, _ := filepath.EvalSymlinks(wtPath)
		wtRealPath, _ := filepath.EvalSymlinks(wt.Path)
		if wtRealPath == wtPathReal && wt.Branch == "feature" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("worktree not found in list: %+v", list)
	}

	if err := g.WorktreeRemove(wtPath, false); err != nil {
		t.Fatal(err)
	}

	list, err = g.worktreeList()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 worktree after remove, got %d", len(list))
	}
}

func TestParseWorktreeList(t *testing.T) {
	input := `worktree /home/user/repo
HEAD abc123
branch refs/heads/main

worktree /home/user/repo-wt
HEAD def456
branch refs/heads/feature

`
	result := parseWorktreeList(input)

	if len(result) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(result))
	}

	if result[0].Path != "/home/user/repo" {
		t.Errorf("expected /home/user/repo, got %s", result[0].Path)
	}
	if result[0].Branch != "main" {
		t.Errorf("expected main, got %s", result[0].Branch)
	}
	if result[1].Path != "/home/user/repo-wt" {
		t.Errorf("expected /home/user/repo-wt, got %s", result[1].Path)
	}
	if result[1].Branch != "feature" {
		t.Errorf("expected feature, got %s", result[1].Branch)
	}
}

func TestMainWorktree(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	main, err := g.MainWorktree()
	if err != nil {
		t.Fatal(err)
	}

	mainPathReal, _ := filepath.EvalSymlinks(main.Path)
	expectedReal, _ := filepath.EvalSymlinks(r.Dir)
	if mainPathReal != expectedReal {
		t.Errorf("MainWorktree().Path = %q, want %q", main.Path, r.Dir)
	}
}

func TestMainWorktreeFromWorktree(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	wtPath := filepath.Join(t.TempDir(), "wt-test")
	if err := g.WorktreeAdd(wtPath, "feature", true, ""); err != nil {
		t.Fatal(err)
	}

	wtGit := New(wtPath)
	main, err := wtGit.MainWorktree()
	if err != nil {
		t.Fatal(err)
	}

	mainPathReal, _ := filepath.EvalSymlinks(main.Path)
	expectedReal, _ := filepath.EvalSymlinks(r.Dir)
	if mainPathReal != expectedReal {
		t.Errorf("MainWorktree() from worktree = %q, want %q", main.Path, r.Dir)
	}
}

func TestFindWorktreeByBranch(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	wtPath := filepath.Join(t.TempDir(), "wt-feature")
	if err := g.WorktreeAdd(wtPath, "feature", true, ""); err != nil {
		t.Fatal(err)
	}

	wt, found, err := g.FindWorktreeByBranch("feature")
	if err != nil {
		t.Fatalf("FindWorktreeByBranch() error = %v", err)
	}
	if !found {
		t.Fatal("expected to find feature branch")
	}
	wtPathReal, _ := filepath.EvalSymlinks(wtPath)
	wtFoundReal, _ := filepath.EvalSymlinks(wt.Path)
	if wtFoundReal != wtPathReal {
		t.Errorf("Path = %q, want %q", wt.Path, wtPath)
	}

	wt, found, err = g.FindWorktreeByBranch("main")
	if err != nil {
		t.Fatalf("FindWorktreeByBranch(main) error = %v", err)
	}
	if !found {
		t.Fatal("expected to find main branch")
	}
	mainReal, _ := filepath.EvalSymlinks(r.Dir)
	mainFoundReal, _ := filepath.EvalSymlinks(wt.Path)
	if mainFoundReal != mainReal {
		t.Errorf("main Path = %q, want %q", wt.Path, r.Dir)
	}

	_, found, err = g.FindWorktreeByBranch("nonexistent")
	if err != nil {
		t.Fatalf("FindWorktreeByBranch(nonexistent) error = %v", err)
	}
	if found {
		t.Error("expected not to find nonexistent branch")
	}
}
