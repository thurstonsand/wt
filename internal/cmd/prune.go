package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/thurstonsand/wt/internal/config"
	"github.com/thurstonsand/wt/internal/git"
	"github.com/thurstonsand/wt/internal/worktree"
)

func newPruneCmd() *cobra.Command {
	var dryRun bool
	var force bool
	var all bool

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove stale worktree directories and orphaned branches",
		Long: `Scan for stale worktree directories (broken git pointer, deleted source repo)
and prunable branches (landed, or wt-managed and no longer checked out anywhere),
then remove them via an interactive picker.

Stale directories and landed branches (upstream gone) are pre-selected. Other
wt-managed branches are listed but unselected by default, since deleting them is
destructive; pass --all to pre-select them too.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return pruneRun(cmd, dryRun, force, all)
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "report without removing")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "remove without prompting")
	cmd.Flags().BoolVar(&all, "all", false, "include orphaned branches (pre-selected)")

	return cmd
}

type repoBranches struct {
	repo     string
	branches []worktree.PrunableBranch
}

func pruneRun(cmd *cobra.Command, dryRun, force, all bool) error {
	store := config.DefaultGlobalStore()
	out := cmd.OutOrStdout()

	// Collect stale directories
	stale, err := worktree.FindStale(store)
	if err != nil {
		return err
	}

	// Collect prunable branches
	var cwdRepo string
	if mgr, merr := defaultManager(); merr == nil {
		if mainWt, werr := mgr.Git().MainWorktree(); werr == nil {
			cwdRepo = mainWt.Path
		}
	}

	repos, err := worktree.ResolveRepos(store, cwdRepo)
	if err != nil {
		return err
	}

	interactive := isInteractive() && !dryRun && !force

	// Outside the interactive picker there is no chance to toggle items, so
	// branch deletion stays conservative: only landed branches (upstream gone, a
	// strong "done" signal) act by default, while wt-managed-but-not-landed
	// branches require --all. Dry-run is read-only, so it reports everything.
	landedOnly := !interactive && !dryRun && !all

	var branchResults []repoBranches
	var totalBranches int
	for _, repoPath := range repos {
		g := git.New(repoPath)
		branches, err := worktree.FindPrunableBranches(g, landedOnly)
		if err != nil {
			_, _ = fmt.Fprintf(out, "warning: skipping %s: %v\n", filepath.Base(repoPath), err)
			continue
		}
		if len(branches) > 0 {
			branchResults = append(branchResults, repoBranches{repo: repoPath, branches: branches})
			totalBranches += len(branches)
		}
	}

	// Display findings
	hasWork := len(stale) > 0 || totalBranches > 0

	if !hasWork {
		_, _ = fmt.Fprintln(out, "Nothing to prune.")
		return nil
	}

	if !interactive {
		if len(stale) > 0 {
			_, _ = fmt.Fprintln(out, "Stale worktree directories:")
			for _, s := range stale {
				_, _ = fmt.Fprintf(out, "  %s/%s\n", s.RepoName, s.Name)
			}
		}
		if totalBranches > 0 {
			for _, r := range branchResults {
				_, _ = fmt.Fprintf(out, "Prunable branches (%s):\n", filepath.Base(r.repo))
				for _, b := range r.branches {
					_, _ = fmt.Fprintf(out, "  %s  (%s)\n", b.Branch, formatBranchDetail(b))
				}
			}
		}
	}

	if dryRun {
		return nil
	}

	if !force {
		if interactive {
			sel, err := pickPruneItems(stale, branchResults, all)
			if err != nil {
				if errors.Is(err, errAborted) {
					return nil
				}
				return err
			}
			stale = sel.stale
			branchResults = sel.branches
		} else {
			total := len(stale) + totalBranches
			_, _ = fmt.Fprintf(out, "\nRemove %d items? [y/N] ", total)
			reader := bufio.NewReader(cmd.InOrStdin())
			answer, _ := reader.ReadString('\n')
			if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(answer)), "y") {
				return nil
			}
		}
	}

	if len(stale) > 0 {
		removed, err := worktree.PruneStale(store, stale)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(out, "Removed worktree directories:")
		for _, s := range removed {
			_, _ = fmt.Fprintf(out, "  %s/%s\n", s.RepoName, s.Name)
		}
	}

	if len(branchResults) > 0 {
		var deletedNames []string
		var firstErr error
		for _, r := range branchResults {
			g := git.New(r.repo)
			deleted, err := worktree.DeletePrunableBranches(g, r.branches)
			for _, b := range deleted {
				deletedNames = append(deletedNames, b.Branch)
			}
			if err != nil && firstErr == nil {
				firstErr = err
			}
		}
		if len(deletedNames) > 0 {
			_, _ = fmt.Fprintln(out, "Deleted branches:")
			for _, name := range deletedNames {
				_, _ = fmt.Fprintf(out, "  %s\n", name)
			}
		}
		if firstErr != nil {
			return firstErr
		}
	}

	return nil
}

func formatBranchDetail(b worktree.PrunableBranch) string {
	switch {
	case b.Landed:
		return "landed, upstream gone"
	case b.Parent != "" && b.AheadCount < 0:
		return fmt.Sprintf("parent: %s, parent branch gone", b.Parent)
	case b.HasUpstream:
		return fmt.Sprintf("parent: %s, pushed, upstream still exists", b.Parent)
	case b.AheadCount <= 0:
		return fmt.Sprintf("parent: %s, no new commits", b.Parent)
	case b.AheadCount == 1:
		return fmt.Sprintf("parent: %s, 1 unpushed commit", b.Parent)
	default:
		return fmt.Sprintf("parent: %s, %d unpushed commits", b.Parent, b.AheadCount)
	}
}
