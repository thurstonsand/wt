package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/thurstonsand/wt/internal/shell"
	"github.com/thurstonsand/wt/internal/worktree"
)

func newForkCmd() *cobra.Command {
	var opts struct {
		base    string
		clean   bool
		noClean bool
		with    []string
	}

	cmd := &cobra.Command{
		Use:   "fork [name]",
		Short: "Fork current work to a new worktree",
		Long: `Fork current work to a new worktree.

By default, copies staged/unstaged changes and untracked files to the new worktree.
Use --clean to create a clean worktree without copying changes.
Use --base to create from a different branch (implies --clean).
Include files (patterns in .worktreeinclude, e.g. .env*) are always copied regardless of --clean.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.clean && opts.noClean {
				return fmt.Errorf("%w: --clean and --no-clean are mutually exclusive", ErrInvalidFlagCombination)
			}
			if opts.base != "" && opts.noClean {
				return fmt.Errorf("%w: --no-clean cannot be used with --base", ErrInvalidFlagCombination)
			}

			mgr, err := defaultManager()
			if err != nil {
				return err
			}

			forkOpts := worktree.ForkOptions{
				Base: opts.base,
				With: opts.with,
			}
			if len(args) > 0 {
				forkOpts.Name = args[0]
			}

			if opts.clean {
				t := true
				forkOpts.Clean = &t
			} else if opts.noClean {
				f := false
				forkOpts.Clean = &f
			}

			wt, err := mgr.Fork(forkOpts)
			if err != nil {
				return err
			}

			cfg, _, err := mgr.Store().LoadConfig()
			if err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: failed to load config for hooks: %v\n", err)
			} else {
				mgr.RunPostCreateHooks(cfg, wt.WorktreePath, cmd.ErrOrStderr())
			}

			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "Forked to worktree %q\n", wt.Name)
			shell.PrintWithCD(wt.WorktreePath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.base, "base", "b", "", "base branch to create worktree from (implies --clean)")
	cmd.Flags().BoolVar(&opts.clean, "clean", false, "create clean worktree without copying changes")
	cmd.Flags().BoolVar(&opts.noClean, "no-clean", false, "copy changes even if config defaults to clean")
	cmd.Flags().StringSliceVar(&opts.with, "with", nil, "extra include patterns to copy (.gitignore syntax, in addition to .worktreeinclude)")

	return cmd
}
