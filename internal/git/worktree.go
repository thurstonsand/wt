package git

import (
	"bufio"
	"strings"
)

// WorktreeInfo holds information about a git worktree.
type WorktreeInfo struct {
	Path   string
	HEAD   string
	Branch string
	Bare   bool
}

// WorktreeAdd creates a new worktree at path for the given branch.
// If createBranch is true, creates a new branch at baseCommit (or HEAD if empty).
// If createBranch is false, checks out the existing branch.
func (g *Git) WorktreeAdd(path, branch string, createBranch bool, baseCommit string) error {
	args := []string{"worktree", "add"}
	if createBranch {
		args = append(args, "-b", branch)
	}
	args = append(args, path)
	if createBranch {
		if baseCommit != "" {
			args = append(args, baseCommit)
		}
	} else {
		args = append(args, branch)
	}
	_, err := g.run(runOpts{}, args...)
	return err
}

// WorktreeRemove removes a worktree at the given path.
// If force is true, removes even if dirty.
func (g *Git) WorktreeRemove(path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)
	_, err := g.run(runOpts{}, args...)
	return err
}

// worktreeList returns all worktrees for this repository, including the main worktree.
func (g *Git) worktreeList() ([]WorktreeInfo, error) {
	out, err := g.run(runOpts{}, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parseWorktreeList(out), nil
}

// LinkedWorktrees returns all linked worktrees (excludes the main worktree).
func (g *Git) LinkedWorktrees() ([]WorktreeInfo, error) {
	all, err := g.worktreeList()
	if err != nil {
		return nil, err
	}
	if len(all) <= 1 {
		return nil, nil
	}
	return all[1:], nil
}

// MainWorktree returns the main (first) worktree for this repository.
func (g *Git) MainWorktree() (WorktreeInfo, error) {
	list, err := g.worktreeList()
	if err != nil {
		return WorktreeInfo{}, err
	}
	if len(list) == 0 {
		return WorktreeInfo{}, ErrNoWorktrees
	}
	return list[0], nil
}

// FindWorktreeByBranch finds the worktree that has the given branch checked out.
// Returns the worktree info and true if found, empty and false otherwise.
func (g *Git) FindWorktreeByBranch(branch string) (WorktreeInfo, bool, error) {
	all, err := g.worktreeList()
	if err != nil {
		return WorktreeInfo{}, false, err
	}
	for _, wt := range all {
		if wt.Branch == branch {
			return wt, true, nil
		}
	}
	return WorktreeInfo{}, false, nil
}

func parseWorktreeList(output string) []WorktreeInfo {
	var result []WorktreeInfo
	var current WorktreeInfo

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "worktree "):
			if current.Path != "" {
				result = append(result, current)
			}
			current = WorktreeInfo{Path: strings.TrimPrefix(line, "worktree ")}
		case strings.HasPrefix(line, "HEAD "):
			current.HEAD = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		case line == "bare":
			current.Bare = true
		}
	}
	if current.Path != "" {
		result = append(result, current)
	}
	return result
}
