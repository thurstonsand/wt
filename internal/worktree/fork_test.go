//go:build integration

package worktree

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/thurstonsand/wt/internal/config"
	"github.com/thurstonsand/wt/internal/git"
	"github.com/thurstonsand/wt/internal/testutil"
)

func TestFork(t *testing.T) {
	r := testutil.InitGitRepo(t)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	mgr, err := NewManager(r.Dir,
		WithGit(git.New(r.Dir)),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{Name: "test-fork"})
	if err != nil {
		t.Fatal(err)
	}

	if wt.Name != "test-fork" {
		t.Errorf("expected name test-fork, got %s", wt.Name)
	}
	if wt.Branch != "test-fork" {
		t.Errorf("expected branch test-fork, got %s", wt.Branch)
	}
	if wt.Parent != "main" {
		t.Errorf("expected parent branch main, got %s", wt.Parent)
	}

	expectedPath := globalStore.WorktreePath(mgr.RepoName(), "test-fork")
	if wt.WorktreePath != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, wt.WorktreePath)
	}

	if _, err := os.Stat(wt.WorktreePath); os.IsNotExist(err) {
		t.Errorf("worktree directory does not exist: %s", wt.WorktreePath)
	} else if err != nil {
		t.Fatal(err)
	}
}

func TestForkFromLinkedWorktree(t *testing.T) {
	r := testutil.InitGitRepo(t)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	mgr, err := NewManager(r.Dir,
		WithGit(git.New(r.Dir)),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	first, err := mgr.Fork(ForkOptions{Name: "first"})
	if err != nil {
		t.Fatal(err)
	}

	// Fork again from inside the linked worktree. The new worktree must group
	// under the canonical repo name, not the linked worktree's basename.
	nested, err := NewManager(first.WorktreePath,
		WithGit(git.New(first.WorktreePath)),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	if nested.RepoName() != mgr.RepoName() {
		t.Fatalf("RepoName() = %q, want canonical %q", nested.RepoName(), mgr.RepoName())
	}

	second, err := nested.Fork(ForkOptions{Name: "second"})
	if err != nil {
		t.Fatal(err)
	}

	expectedPath := globalStore.WorktreePath(mgr.RepoName(), "second")
	if second.WorktreePath != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, second.WorktreePath)
	}
}

func TestForkDuplicate(t *testing.T) {
	r := testutil.InitGitRepo(t)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	mgr, err := NewManager(r.Dir,
		WithGit(git.New(r.Dir)),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = mgr.Fork(ForkOptions{Name: "dup"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = mgr.Fork(ForkOptions{Name: "dup"})
	if !errors.Is(err, ErrWorktreeExists) {
		t.Errorf("expected ErrWorktreeExists, got %v", err)
	}
}

func TestForkPreservesStagedChanges(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	stagedContent := "staged content"

	r.WriteFile("staged.txt", stagedContent)
	r.Run("add", "staged.txt")

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{Name: "staged-fork"})
	if err != nil {
		t.Fatal(err)
	}

	wtGit := git.New(wt.WorktreePath)
	stagedDiff, err := wtGit.DiffCached()
	if err != nil {
		t.Fatal(err)
	}
	if stagedDiff == "" {
		t.Error("expected staged changes in forked worktree")
	}

	content, err := os.ReadFile(filepath.Join(wt.WorktreePath, "staged.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != stagedContent {
		t.Errorf("expected 'staged content', got %q", content)
	}
}

func TestForkPreservesUnstagedChanges(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	modifiedContent := "modified content"

	r.CommitFile(t, "tracked.txt", "initial commit")
	r.WriteFile("tracked.txt", modifiedContent)

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{Name: "unstaged-fork"})
	if err != nil {
		t.Fatal(err)
	}

	wtGit := git.New(wt.WorktreePath)
	unstagedDiff, err := wtGit.Diff()
	if err != nil {
		t.Fatal(err)
	}
	if unstagedDiff == "" {
		t.Error("expected unstaged changes in forked worktree")
	}

	content, err := os.ReadFile(filepath.Join(wt.WorktreePath, "tracked.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != modifiedContent {
		t.Errorf("expected '%s', got %q", modifiedContent, content)
	}
}

func TestForkPreservesStagedAndUnstagedSeparately(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	r.CommitFile(t, "file.txt", "initial")

	r.WriteFile("file.txt", "line1\nline2\nline3\n")
	r.Run("add", "file.txt")
	r.WriteFile("file.txt", "line1\nline2\nline3\nline4\n")

	originalStagedDiff, _ := g.DiffCached()
	originalUnstagedDiff, _ := g.Diff()

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{Name: "mixed-fork"})
	if err != nil {
		t.Fatal(err)
	}

	wtGit := git.New(wt.WorktreePath)

	stagedDiff, err := wtGit.DiffCached()
	if err != nil {
		t.Fatal(err)
	}
	if stagedDiff != originalStagedDiff {
		t.Errorf("staged diff mismatch\noriginal:\n%s\nforked:\n%s", originalStagedDiff, stagedDiff)
	}

	unstagedDiff, err := wtGit.Diff()
	if err != nil {
		t.Fatal(err)
	}
	if unstagedDiff != originalUnstagedDiff {
		t.Errorf("unstaged diff mismatch\noriginal:\n%s\nforked:\n%s", originalUnstagedDiff, unstagedDiff)
	}
}

func TestForkGeneratesName(t *testing.T) {
	r := testutil.InitGitRepo(t)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	mgr, err := NewManager(r.Dir,
		WithGit(git.New(r.Dir)),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(wt.Name) == 0 {
		t.Error("expected generated name")
	}
	if wt.Name[:3] != "wt-" {
		t.Errorf("expected name to start with wt-, got %s", wt.Name)
	}
}

func TestForkFromNonMainBranch(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	if err := g.CreateBranch("develop", "HEAD"); err != nil {
		t.Fatal(err)
	}
	if err := g.Switch("develop"); err != nil {
		t.Fatal(err)
	}

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{Name: "from-develop"})
	if err != nil {
		t.Fatal(err)
	}

	if wt.Parent != "develop" {
		t.Errorf("expected parent branch develop, got %s", wt.Parent)
	}
}

func TestForkSourceRepoUnchanged(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	r.WriteFile("staged.txt", "staged")
	r.Run("add", "staged.txt")
	r.CommitFile(t, "tracked.txt", "initial")
	r.WriteFile("tracked.txt", "modified")

	origStagedDiff, _ := g.DiffCached()
	origUnstagedDiff, _ := g.Diff()

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = mgr.Fork(ForkOptions{Name: "no-side-effects"})
	if err != nil {
		t.Fatal(err)
	}

	afterStagedDiff, _ := g.DiffCached()
	afterUnstagedDiff, _ := g.Diff()

	if origStagedDiff != afterStagedDiff {
		t.Error("fork modified source repo staged changes")
	}
	if origUnstagedDiff != afterUnstagedDiff {
		t.Error("fork modified source repo unstaged changes")
	}
}

func TestForkNoCleanWithBaseErrors(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)

	if err := g.CreateBranch("develop", "HEAD"); err != nil {
		t.Fatal(err)
	}

	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	noClean := false
	_, err = mgr.Fork(ForkOptions{
		Name:  "bad",
		Base:  "develop",
		Clean: &noClean,
	})
	if !errors.Is(err, ErrCleanWithBase) {
		t.Errorf("expected ErrCleanWithBase, got %v", err)
	}
}

func TestForkWithBase(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)

	if err := g.CreateBranch("develop", "HEAD"); err != nil {
		t.Fatal(err)
	}

	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{Name: "feature", Base: "develop"})
	if err != nil {
		t.Fatal(err)
	}

	if wt.Parent != "develop" {
		t.Errorf("expected parent branch develop, got %s", wt.Parent)
	}
}

func TestForkClean(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	r.WriteFile("staged.txt", "staged")
	r.Run("add", "staged.txt")
	r.CommitFile(t, "tracked.txt", "initial")
	r.WriteFile("tracked.txt", "modified")

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	clean := true
	wt, err := mgr.Fork(ForkOptions{Name: "clean-fork", Clean: &clean})
	if err != nil {
		t.Fatal(err)
	}

	wtGit := git.New(wt.WorktreePath)
	stagedDiff, err := wtGit.DiffCached()
	if err != nil {
		t.Fatal(err)
	}
	if stagedDiff != "" {
		t.Error("expected no staged changes in clean worktree")
	}

	unstagedDiff, err := wtGit.Diff()
	if err != nil {
		t.Fatal(err)
	}
	if unstagedDiff != "" {
		t.Error("expected no unstaged changes in clean worktree")
	}
}

func TestForkCleanupDeletesBranchOnFailure(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	// Create a committed file, then make it unreadable so copyFiles fails.
	r.CommitFile(t, "secret.bin", "initial")
	secretPath := filepath.Join(r.Dir, "secret.bin")
	if err := os.Chmod(secretPath, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(secretPath, 0o644) })

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	clean := true
	_, err = mgr.Fork(ForkOptions{
		Name:  "fail-fork",
		Clean: &clean,
		With:  []string{"secret.bin"},
	})
	if err == nil {
		t.Fatal("expected fork to fail due to unreadable file")
	}

	// Branch must not survive a failed fork.
	exists, err := g.LocalBranchExists("fail-fork")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("expected branch to be deleted after failed fork")
	}

	// Retry with the same name should succeed now.
	if err := os.Chmod(secretPath, 0o644); err != nil {
		t.Fatal(err)
	}
	wt, err := mgr.Fork(ForkOptions{
		Name:  "fail-fork",
		Clean: &clean,
		With:  []string{"secret.bin"},
	})
	if err != nil {
		t.Fatalf("expected retry to succeed, got: %v", err)
	}
	if wt.Name != "fail-fork" {
		t.Errorf("expected name fail-fork, got %s", wt.Name)
	}
}

func TestForkWithExtraFiles(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	r.WriteFile("extra.txt", "extra content")

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{
		Name: "with-extra",
		With: []string{"extra.txt"},
	})
	if err != nil {
		t.Fatal(err)
	}

	assertFileExists(t, wt.WorktreePath, "extra.txt")

	content, err := os.ReadFile(filepath.Join(wt.WorktreePath, "extra.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "extra content" {
		t.Errorf("expected 'extra content', got %q", content)
	}
}

func TestForkPreservesStagedBinaryChanges(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	original := string([]byte{0x89, 0x50, 0x4E, 0x47, 0x00, 0x01})
	modified := string([]byte{0x89, 0x50, 0x4E, 0x47, 0xFF, 0xFE})

	r.WriteFile("image.bin", original)
	r.Run("add", "image.bin")
	r.Run("commit", "-m", "add binary")

	r.WriteFile("image.bin", modified)
	r.Run("add", "image.bin")

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{Name: "staged-binary"})
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(wt.WorktreePath, "image.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != modified {
		t.Errorf("binary content mismatch in worktree: got %v, want %v", got, []byte(modified))
	}

	wtGit := git.New(wt.WorktreePath)
	stagedDiff, err := wtGit.DiffCached()
	if err != nil {
		t.Fatal(err)
	}
	if stagedDiff == "" {
		t.Error("expected binary to be staged in worktree")
	}
}

func TestForkPreservesUnstagedBinaryChanges(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	original := string([]byte{0x00, 0x01, 0x02, 0x03})
	modified := string([]byte{0xFF, 0xFE, 0xFD, 0xFC})

	r.WriteFile("data.bin", original)
	r.Run("add", "data.bin")
	r.Run("commit", "-m", "add binary")

	r.WriteFile("data.bin", modified)

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{Name: "unstaged-binary"})
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(wt.WorktreePath, "data.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != modified {
		t.Errorf("binary content mismatch: got %v, want %v", got, []byte(modified))
	}

	wtGit := git.New(wt.WorktreePath)
	unstagedDiff, err := wtGit.Diff()
	if err != nil {
		t.Fatal(err)
	}
	if unstagedDiff == "" {
		t.Error("expected binary to show as unstaged in worktree")
	}
}

func TestForkPreservesMixedTextAndBinary(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	r.WriteFile("image.bin", string([]byte{0x00, 0x01}))
	r.Run("add", "image.bin")
	r.Run("commit", "-m", "add binary")
	r.CommitFile(t, "code.go", "initial")

	binModified := string([]byte{0xFF, 0xFE})
	r.WriteFile("image.bin", binModified)
	r.WriteFile("code.go", "modified text\n")

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{Name: "mixed-fork"})
	if err != nil {
		t.Fatal(err)
	}

	gotBin, err := os.ReadFile(filepath.Join(wt.WorktreePath, "image.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if string(gotBin) != binModified {
		t.Errorf("binary content mismatch: got %v, want %v", gotBin, []byte(binModified))
	}

	gotText, err := os.ReadFile(filepath.Join(wt.WorktreePath, "code.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(gotText) != "modified text\n" {
		t.Errorf("text content mismatch: got %q", gotText)
	}
}

func TestForkPreservesStagedDeletion(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	r.CommitFile(t, "doomed.txt", "add doomed")

	r.Run("rm", "doomed.txt")

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{Name: "staged-del"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(wt.WorktreePath, "doomed.txt")); !os.IsNotExist(err) {
		t.Error("deleted file should not exist in worktree")
	}

	wtGit := git.New(wt.WorktreePath)
	del, err := wtGit.DiffNameOnly(true, "D")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range del {
		if f == "doomed.txt" {
			found = true
			break
		}
	}
	if !found {
		t.Error("deletion should be staged in worktree")
	}
}

func TestForkPreservesUnstagedDeletion(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := git.New(r.Dir)
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	r.CommitFile(t, "doomed.txt", "add doomed")

	os.Remove(filepath.Join(r.Dir, "doomed.txt"))

	mgr, err := NewManager(r.Dir,
		WithGit(g),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := mgr.Fork(ForkOptions{Name: "unstaged-del"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(wt.WorktreePath, "doomed.txt")); !os.IsNotExist(err) {
		t.Error("deleted file should not exist in worktree working tree")
	}

	wtGit := git.New(wt.WorktreePath)
	stagedDel, err := wtGit.DiffNameOnly(true, "D")
	if err != nil {
		t.Fatal(err)
	}
	if len(stagedDel) > 0 {
		t.Error("unstaged deletion should NOT be staged in worktree")
	}
}

func TestForkRejectsExistingLocalBranch(t *testing.T) {
	r := testutil.InitGitRepo(t)
	r.Run("branch", "feature-x")
	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	mgr, err := NewManager(r.Dir,
		WithGit(git.New(r.Dir)),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = mgr.Fork(ForkOptions{Name: "feature-x"})
	if !errors.Is(err, ErrBranchExists) {
		t.Errorf("expected ErrBranchExists, got %v", err)
	}
}

func TestForkRejectsExistingRemoteBranch(t *testing.T) {
	r := testutil.InitGitRepo(t)

	// Stand up a bare remote and publish a branch that only lives there.
	remoteDir := t.TempDir()
	testutil.NewGitRepo(t, remoteDir).Run("init", "--bare", "--initial-branch=main")
	r.Run("remote", "add", "origin", remoteDir)
	r.Run("push", "origin", "main")
	r.Run("branch", "feature-y")
	r.Run("push", "origin", "feature-y")
	r.Run("branch", "-D", "feature-y")
	// Intentionally do NOT fetch: the guard should fetch and detect it.

	wtRoot := t.TempDir()
	globalStore := config.NewGlobalStore(wtRoot)

	mgr, err := NewManager(r.Dir,
		WithGit(git.New(r.Dir)),
		WithGlobalStore(globalStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = mgr.Fork(ForkOptions{Name: "feature-y"})
	if !errors.Is(err, ErrBranchExists) {
		t.Errorf("expected ErrBranchExists for remote-only branch, got %v", err)
	}
}
