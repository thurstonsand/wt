package cmd

import (
	"sort"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/thurstonsand/wt/internal/git"
)

const rootWorktreeName = "root"

type completionEntry struct {
	name string
	time time.Time
}

func completeWorktreeNames(includeSource bool) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		mgr, err := defaultManager()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		result, err := mgr.ListAll()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		entries := make([]completionEntry, len(result.Worktrees))
		var wg sync.WaitGroup
		for i, wt := range result.Worktrees {
			wg.Go(func() {
				entries[i].name = wt.Branch
				g := git.New(wt.WorktreePath)
				if t, err := g.HeadCommitTime(); err == nil {
					entries[i].time = t
				}
			})
		}
		wg.Wait()

		sort.Slice(entries, func(i, j int) bool {
			return entries[i].time.After(entries[j].time)
		})

		var names []string
		if includeSource {
			names = append(names, rootWorktreeName)
		}
		for _, e := range entries {
			names = append(names, e.name)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

func completeBranchNames(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	mgr, err := defaultManager()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	branches, err := mgr.Git().ListBranches(true)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return branches, cobra.ShellCompDirectiveNoFileComp
}
