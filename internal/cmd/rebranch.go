package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/thurstonsand/wt/internal/shell"
	"github.com/thurstonsand/wt/internal/worktree"
)

func newRebranchCmd() *cobra.Command {
	var opts struct {
		forWorktree string
		onto        string
	}

	cmd := &cobra.Command{
		Use:   "rebranch <new-branch>",
		Short: "Re-seat a landed worktree onto a fresh baseline under a new branch",
		Long: `Re-seat a landed worktree onto a fresh baseline under a new branch.

A worktree is "landed" once its branch was pushed, merged via PR/MR, and
deleted on the remote. rebranch keeps the same directory, rebaselines onto
origin's default branch under <new-branch>, and carries uncommitted changes
forward. The spent branch is left behind for 'wt prune --all'; committed
work is never destroyed.

Run from inside the worktree, or target another with -w.

Examples:
  wt rebranch feature-2                      # rebranch the current worktree
  wt rebranch feature-2 -w wt1               # rebranch the wt1 worktree
  wt rebranch feature-2 --onto develop       # rebranch onto develop`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := defaultManager()
			if err != nil {
				return err
			}

			res, err := mgr.Rebranch(worktree.RebranchOptions{
				NewBranch:   args[0],
				ForWorktree: opts.forWorktree,
				Onto:        opts.onto,
			})
			if err != nil {
				if res != nil && res.Conflict {
					printRebranchConflict(cmd, res)
					shell.PrintWithCD(res.WorktreePath)
				}
				return err
			}

			printRebranchSuccess(cmd, res)
			shell.PrintWithCD(res.WorktreePath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.forWorktree, "for-worktree", "w", "", "worktree to rebranch (defaults to current)")
	cmd.Flags().StringVar(&opts.onto, "onto", "", "baseline to rebranch onto (defaults to origin's default branch)")
	_ = cmd.RegisterFlagCompletionFunc("for-worktree", completeWorktreeNames(false))

	return cmd
}

func printRebranchSuccess(cmd *cobra.Command, res *worktree.RebranchResult) {
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "Rebranched %q onto %q as %q\n", res.WorktreeName, res.Baseline, res.NewBranch)
	if !res.IndexRestored {
		_, _ = fmt.Fprintln(out, "  note: staged/unstaged split could not be preserved; changes are unstaged")
	}
	if res.StaleCommits > 0 {
		_, _ = fmt.Fprintf(out, "  the previous branch %q (%s) was left behind\n", res.OldBranch, commitCountText(res.StaleCommits))
		_, _ = fmt.Fprintf(out, "  recover its commits, or drop it with: wt prune --all\n")
	} else {
		_, _ = fmt.Fprintf(out, "  the previous branch %q was left behind; drop it with: wt prune --all\n", res.OldBranch)
	}
}

func printRebranchConflict(cmd *cobra.Command, res *worktree.RebranchResult) {
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "Rebranched %q onto %q as %q, but restoring changes hit a conflict.\n",
		res.WorktreeName, res.Baseline, res.NewBranch)
	_, _ = fmt.Fprintf(out, "  resolve the conflict in %s, then commit or stash\n", res.WorktreePath)
	_, _ = fmt.Fprintf(out, "  your previous branch %q is intact if you need to recover\n", res.OldBranch)
}

func commitCountText(n int) string {
	if n == 1 {
		return "1 commit"
	}
	return fmt.Sprintf("%d commits", n)
}
