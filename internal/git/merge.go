package git

import (
	"os"
	"path/filepath"
	"strings"
)

// MergeSquash performs a squash merge of the given branch into the current branch.
// Returns nil on success. On conflict, returns an error with exit code 1.
func (g *Git) MergeSquash(branch string) error {
	_, err := g.run(runOpts{}, "merge", "--squash", branch)
	return err
}

// MergeFastForward performs a fast-forward only merge of the given branch.
func (g *Git) MergeFastForward(branch string) error {
	_, err := g.run(runOpts{}, "merge", "--ff-only", branch)
	return err
}

// Rebase rebases the current branch onto the given branch.
// Returns nil on success. On conflict, returns an error with exit code 1.
func (g *Git) Rebase(onto string) error {
	_, err := g.run(runOpts{}, "rebase", onto)
	return err
}

// RebaseAutostash rebases the current branch onto the given branch,
// automatically stashing and restoring uncommitted changes.
func (g *Git) RebaseAutostash(onto string) error {
	_, err := g.run(runOpts{}, "rebase", "--autostash", onto)
	return err
}

// RebaseOnto transplants commits from oldBase..HEAD onto newBase.
// Uses --autostash to handle uncommitted changes.
func (g *Git) RebaseOnto(newBase, oldBase string) error {
	_, err := g.run(runOpts{}, "rebase", "--autostash", "--onto", newBase, oldBase)
	return err
}

// RebaseInProgress returns true if a rebase is currently in progress.
func (g *Git) RebaseInProgress() (bool, error) {
	gitDir, err := g.RevParse("--git-dir")
	if err != nil {
		return false, err
	}
	for _, dir := range []string{"rebase-merge", "rebase-apply"} {
		if _, err := os.Stat(filepath.Join(gitDir, dir)); err == nil {
			return true, nil
		}
	}
	return false, nil
}

// RebaseAbort aborts an in-progress rebase.
func (g *Git) RebaseAbort() error {
	_, err := g.run(runOpts{}, "rebase", "--abort")
	return err
}

// DiffBranchFiles returns files changed between two branches.
func (g *Git) DiffBranchFiles(from, to string) ([]string, error) {
	out, err := g.run(runOpts{}, "diff", "--name-only", from+"..."+to)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

// DiffBranchFileCount returns the number of files changed between two branches.
func (g *Git) DiffBranchFileCount(from, to string) (int, error) {
	files, err := g.DiffBranchFiles(from, to)
	return len(files), err
}

// Commit creates a commit with the given message.
func (g *Git) Commit(message string) error {
	_, err := g.run(runOpts{}, "commit", "-m", message)
	return err
}

// HasConflicts checks if the working tree has unresolved conflicts.
func (g *Git) HasConflicts() (bool, error) {
	out, err := g.run(runOpts{}, "diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return false, err
	}
	return out != "", nil
}

// CommitInfo holds basic commit information.
type CommitInfo struct {
	Hash    string
	Subject string
}

// parseCommitLine parses a "hash subject" line into CommitInfo.
func parseCommitLine(line string) CommitInfo {
	if hash, subject, found := strings.Cut(line, " "); found {
		return CommitInfo{Hash: hash, Subject: subject}
	}
	return CommitInfo{Hash: line}
}

// LastCommit returns info about the most recent commit.
func (g *Git) LastCommit() (CommitInfo, error) {
	out, err := g.run(runOpts{}, "log", "-1", "--format=%h %s")
	if err != nil {
		return CommitInfo{}, err
	}
	if out == "" {
		return CommitInfo{}, nil
	}
	return parseCommitLine(out), nil
}

// CommitsBetween returns commits from base (exclusive) to head (inclusive).
func (g *Git) CommitsBetween(base, head string) ([]CommitInfo, error) {
	out, err := g.run(runOpts{}, "log", "--format=%h %s", base+".."+head)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	var commits []CommitInfo
	for line := range strings.SplitSeq(out, "\n") {
		if line == "" {
			continue
		}
		commits = append(commits, parseCommitLine(line))
	}
	return commits, nil
}

// StagedFileCount returns the number of staged files.
func (g *Git) StagedFileCount() (int, error) {
	out, err := g.run(runOpts{}, "diff", "--cached", "--name-only")
	if err != nil {
		return 0, err
	}
	return countNonEmptyLines(out), nil
}

// countNonEmptyLines counts non-empty lines in output.
func countNonEmptyLines(out string) int {
	if out == "" {
		return 0
	}
	count := 0
	for line := range strings.SplitSeq(out, "\n") {
		if line != "" {
			count++
		}
	}
	return count
}
