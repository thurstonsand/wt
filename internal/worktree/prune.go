package worktree

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"github.com/thurstonsand/wt/internal/config"
	"github.com/thurstonsand/wt/internal/git"
)

// FindStale scans all managed worktree directories and returns those
// that are stale (broken git pointer, missing .git file, etc).
func FindStale(store *config.GlobalStore) ([]StaleWorktree, error) {
	res, err := ListAllRepos(store)
	if err != nil {
		return nil, err
	}
	return res.Stale, nil
}

// PruneStale removes the given stale worktree directories from disk.
// Returns the entries that were successfully removed.
func PruneStale(store *config.GlobalStore, stale []StaleWorktree) ([]StaleWorktree, error) {
	var removed []StaleWorktree
	for _, s := range stale {
		if err := removeDirAndCleanup(store, s.Path); err != nil {
			return removed, fmt.Errorf("removing %s/%s: %w", s.RepoName, s.Name, err)
		}
		removed = append(removed, s)
	}
	return removed, nil
}

// PrunableBranch describes a local branch that prune may remove: one that is
// not checked out in any worktree and is either landed (its upstream was
// deleted on the remote) or wt-managed (carries wt-parent metadata).
type PrunableBranch struct {
	Branch      string
	Parent      string // "" when the branch has no wt-parent metadata
	AheadCount  int    // commits ahead of Parent; -1 if parent missing/unknown
	Landed      bool   // upstream [gone]: pushed, merged, and deleted on remote
	HasUpstream bool   // an upstream was configured (branch was pushed)
}

// FindPrunableBranches returns local branches that are not checked out in any
// worktree and are not protected, limited to those that are either landed
// (upstream gone) or carry wt-parent metadata. A hand-made branch that was
// never pushed and was never managed by wt stays invisible. When landedOnly is
// true, only landed branches are returned, omitting wt-managed branches whose
// upstream still exists or that were never pushed.
func FindPrunableBranches(g *git.Git, landedOnly bool) ([]PrunableBranch, error) {
	parents, err := g.GetBranchesWithParent()
	if err != nil {
		return nil, err
	}

	locals, err := g.ListBranches(false)
	if err != nil {
		return nil, err
	}
	if len(locals) == 0 {
		return nil, nil
	}

	activeBranches := make(map[string]bool)
	mainWt, err := g.MainWorktree()
	if err != nil {
		return nil, err
	}
	activeBranches[mainWt.Branch] = true

	linked, err := g.LinkedWorktrees()
	if err != nil {
		return nil, err
	}
	for _, wt := range linked {
		activeBranches[wt.Branch] = true
	}

	var prunable []PrunableBranch
	for _, branch := range locals {
		if isProtectedBranch(branch) {
			continue
		}
		if activeBranches[branch] {
			continue
		}

		landed, err := g.UpstreamGone(branch)
		if err != nil {
			return nil, err
		}
		if landedOnly && !landed {
			continue
		}
		parent, hasParent := parents[branch]
		if !landed && !hasParent {
			continue
		}

		hasUpstream, err := g.HasUpstream(branch)
		if err != nil {
			return nil, err
		}

		ahead := -1
		if hasParent {
			parentExists, err := g.RefExists(parent)
			if err != nil {
				return nil, err
			}
			if parentExists {
				commits, err := g.CommitsBetween(parent, branch)
				if err == nil {
					ahead = len(commits)
				}
			}
		}

		prunable = append(prunable, PrunableBranch{
			Branch:      branch,
			Parent:      parent,
			AheadCount:  ahead,
			Landed:      landed,
			HasUpstream: hasUpstream,
		})
	}

	sort.Slice(prunable, func(i, j int) bool {
		return prunable[i].Branch < prunable[j].Branch
	})
	return prunable, nil
}

// DeletePrunableBranches force-deletes the given branches.
// Returns the branches that were successfully deleted.
func DeletePrunableBranches(g *git.Git, branches []PrunableBranch) ([]PrunableBranch, error) {
	var deleted []PrunableBranch
	for _, b := range branches {
		if err := g.DeleteBranch(b.Branch, true); err != nil {
			return deleted, fmt.Errorf("deleting branch %q: %w", b.Branch, err)
		}
		deleted = append(deleted, b)
	}
	return deleted, nil
}

// ResolveRepos returns the validated list of repos to scan for orphaned branches.
// It loads the persisted repos list, injects extraRepoDir (typically cwd's main
// worktree path), deduplicates preserving insertion order, validates each path as
// a git repo, removes dead entries, and persists any changes back to config.
func ResolveRepos(store *config.GlobalStore, extraRepoDir string) ([]string, error) {
	cfg, _, err := store.LoadConfig()
	if err != nil {
		return nil, err
	}

	candidates := cfg.Repos
	if extraRepoDir != "" {
		resolved, err := filepath.EvalSymlinks(extraRepoDir)
		if err != nil {
			resolved = extraRepoDir
		}
		if !slices.Contains(candidates, resolved) {
			candidates = append(candidates, resolved)
		}
	}

	seen := make(map[string]bool)
	var deduped []string
	for _, p := range candidates {
		if !seen[p] {
			seen[p] = true
			deduped = append(deduped, p)
		}
	}

	var live []string
	for _, p := range deduped {
		if isGitRepo(p) {
			live = append(live, p)
		}
	}

	if !slices.Equal(cfg.Repos, live) {
		_ = store.SetConfigValue("repos", live)
	}

	return live, nil
}

func isGitRepo(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false
	}
	g := git.New(dir)
	_, err = g.MainWorktree()
	return err == nil
}
