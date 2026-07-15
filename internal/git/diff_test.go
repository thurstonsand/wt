//go:build integration

package git

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/thurstonsand/wt/internal/testutil"
)

func TestDiffCachedEmpty(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	diff, err := g.DiffCached()
	if err != nil {
		t.Fatal(err)
	}
	if diff != "" {
		t.Errorf("expected empty diff, got %q", diff)
	}
}

func TestDiffCachedWithStagedChanges(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	const content = "hello\n"
	r.WriteFile("foo.txt", content)
	r.Run("add", "foo.txt")

	diff, err := g.DiffCached()
	if err != nil {
		t.Fatal(err)
	}
	if diff == "" {
		t.Error("expected non-empty diff for staged changes")
	}
	if !strings.Contains(diff, "foo.txt") || !strings.Contains(diff, "+hello") {
		t.Errorf("diff should contain foo.txt and +hello, got:\n%s", diff)
	}
}

func TestDiffEmpty(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	diff, err := g.Diff()
	if err != nil {
		t.Fatal(err)
	}
	if diff != "" {
		t.Errorf("expected empty diff, got %q", diff)
	}
}

func TestDiffWithUnstagedChanges(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	const modified = "# Modified\n"
	r.WriteFile("README.md", modified)

	diff, err := g.Diff()
	if err != nil {
		t.Fatal(err)
	}
	if diff == "" {
		t.Error("expected non-empty diff for unstaged changes")
	}
	if !strings.Contains(diff, "README.md") || !strings.Contains(diff, "+# Modified") {
		t.Errorf("diff should contain README.md and +# Modified, got:\n%s", diff)
	}
}

func TestStagedVsUnstagedSeparation(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	r.WriteFile("staged.txt", "staged content\n")
	r.Run("add", "staged.txt")
	r.WriteFile("unstaged.txt", "unstaged content\n")

	stagedDiff, err := g.DiffCached()
	if err != nil {
		t.Fatal(err)
	}
	unstagedDiff, err := g.Diff()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(stagedDiff, "staged.txt") {
		t.Error("staged diff should contain staged.txt")
	}
	if strings.Contains(stagedDiff, "unstaged.txt") {
		t.Error("staged diff should NOT contain unstaged.txt")
	}
	if strings.Contains(unstagedDiff, "staged.txt") {
		t.Error("unstaged diff should NOT contain staged.txt")
	}
}

func TestDiffNameOnlyCached(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	r.WriteFile("new.txt", "content\n")
	r.Run("add", "new.txt")

	files, err := g.DiffNameOnly(true, "")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(files, "new.txt") {
		t.Errorf("expected new.txt in staged files, got %v", files)
	}
}

func TestDiffNameOnlyUnstaged(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	r.WriteFile("README.md", "modified\n")

	files, err := g.DiffNameOnly(false, "")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(files, "README.md") {
		t.Errorf("expected README.md in unstaged files, got %v", files)
	}
}

func TestDiffNameOnlyFilterDeletions(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	r.CommitFile(t, "keep.txt", "add keep")
	r.CommitFile(t, "remove.txt", "add remove")

	r.Run("rm", "remove.txt")
	r.WriteFile("keep.txt", "modified\n")
	r.Run("add", "keep.txt")

	// "d" excludes deletions
	nonDel, err := g.DiffNameOnly(true, "d")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(nonDel, "keep.txt") {
		t.Errorf("expected keep.txt, got %v", nonDel)
	}
	if slices.Contains(nonDel, "remove.txt") {
		t.Error("should not contain remove.txt with filter=d")
	}

	// "D" only deletions
	del, err := g.DiffNameOnly(true, "D")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(del, "remove.txt") {
		t.Errorf("expected remove.txt, got %v", del)
	}
	if slices.Contains(del, "keep.txt") {
		t.Error("should not contain keep.txt with filter=D")
	}
}

func TestDiffNameOnlyEmpty(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	files, err := g.DiffNameOnly(true, "")
	if err != nil {
		t.Fatal(err)
	}
	if files != nil {
		t.Errorf("expected nil for clean repo, got %v", files)
	}
}

func TestCheckoutIndexTo(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	binContent := string([]byte{0x89, 0x50, 0x4E, 0x47, 0x00, 0x01})
	r.WriteFile("image.bin", binContent)
	r.Run("add", "image.bin")

	destDir := t.TempDir()
	if err := g.CheckoutIndexTo(destDir, []string{"image.bin"}); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(destDir, "image.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != binContent {
		t.Errorf("extracted content mismatch: got %v, want %v", got, []byte(binContent))
	}
}

func TestAdd(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	r.WriteFile("new.txt", "content\n")

	if err := g.Add([]string{"new.txt"}); err != nil {
		t.Fatal(err)
	}

	diff, err := g.DiffCached()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "new.txt") {
		t.Error("new.txt should be staged after Add")
	}
}

func TestRemove(t *testing.T) {
	r := testutil.InitGitRepo(t)
	g := New(r.Dir)

	r.CommitFile(t, "doomed.txt", "add doomed")

	if err := g.Remove([]string{"doomed.txt"}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(r.Dir, "doomed.txt")); !os.IsNotExist(err) {
		t.Error("file should be deleted from working tree")
	}

	del, err := g.DiffNameOnly(true, "D")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(del, "doomed.txt") {
		t.Errorf("expected doomed.txt in staged deletions, got %v", del)
	}
}

func TestRemoveEmpty(t *testing.T) {
	_ = testutil.InitGitRepo(t)
	g := New(t.TempDir())

	if err := g.Remove(nil); err != nil {
		t.Errorf("empty Remove should not error: %v", err)
	}
}
