package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/thurstonsand/wt/internal/config"
	"github.com/thurstonsand/wt/internal/shell"
	"github.com/thurstonsand/wt/internal/worktree"
)

func newMergeCmd() *cobra.Command {
	var opts struct {
		squash bool
		rebase bool
		staged bool
		force  bool
		base   string
	}

	cmd := &cobra.Command{
		Use:               "merge [name]",
		Short:             "Merge worktree changes back to parent branch",
		ValidArgsFunction: completeWorktreeNames(false),
		Long: `Merge worktree changes back to the parent branch.

If run from within a worktree, the name argument is optional.

Merge modes (mutually exclusive):
  --rebase  Fast-forward parent to rebased worktree commits (default)
  --squash  Squash all commits into one on parent
  --staged  Apply changes as staged (no commit)

Protected branches (main/master) default to --staged mode.
Use -f/--force to allow squash/rebase into protected branches.

For external worktrees (not created by wt fork), use --base to specify
the parent branch to merge into.

On success, the worktree is deleted.
On conflict, the worktree is preserved for manual resolution.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			modeCount := countTrue(opts.squash, opts.rebase, opts.staged)
			if modeCount > 1 {
				return fmt.Errorf("%w: --squash, --rebase, and --staged are mutually exclusive", ErrInvalidFlagCombination)
			}

			mgr, err := defaultManager()
			if err != nil {
				return err
			}

			mode := resolveMode(mgr, opts.squash, opts.rebase, opts.staged)

			name := ""
			if len(args) > 0 {
				name = args[0]
			}

			mergeOpts := worktree.MergeOptions{
				Name:  name,
				Mode:  mode,
				Force: opts.force,
				Base:  opts.base,
			}

			result, err := mgr.Merge(mergeOpts)
			if err != nil {
				if result != nil && result.ConflictPath != "" {
					shell.PrintWithCD(result.ConflictPath)
				}
				return err
			}

			out := cmd.OutOrStdout()
			switch result.Mode {
			case config.MergeModeSquash, config.MergeModeRebase:
				_, _ = fmt.Fprintf(out, "Merged %q into %q\n", result.WorktreeName, result.TargetBranch)
				for _, c := range result.Commits {
					_, _ = fmt.Fprintf(out, "  %s %s\n", c.Hash, c.Subject)
				}
			case config.MergeModeStaged:
				_, _ = fmt.Fprintf(out, "Staged changes from %q onto %q (%d files)\n",
					result.WorktreeName, result.TargetBranch, result.FileCount)
			}
			shell.PrintWithCD(result.TargetPath)
			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.squash, "squash", false, "squash all commits into one")
	cmd.Flags().BoolVar(&opts.rebase, "rebase", false, "fast-forward to rebased commits (default)")
	cmd.Flags().BoolVar(&opts.staged, "staged", false, "apply changes as staged (no commit)")
	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "allow merging into protected branches")
	cmd.Flags().StringVar(&opts.base, "base", "", "parent branch to merge into (required for external worktrees)")

	return cmd
}

func resolveMode(mgr *worktree.Manager, squash, rebase, staged bool) config.MergeMode {
	if squash {
		return config.MergeModeSquash
	}
	if rebase {
		return config.MergeModeRebase
	}
	if staged {
		return config.MergeModeStaged
	}

	cfg, _, _ := mgr.Store().LoadConfig()
	return cfg.Merge
}

func countTrue(vals ...bool) int {
	count := 0
	for _, v := range vals {
		if v {
			count++
		}
	}
	return count
}
