package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/thurstonsand/wt/internal/shell"
	"github.com/thurstonsand/wt/internal/worktree"
)

func newCheckoutCmd() *cobra.Command {
	var opts struct {
		parent string
		with   []string
	}

	cmd := &cobra.Command{
		Use:               "checkout <branch>",
		Short:             "Check out an existing branch in a new worktree",
		Aliases:           []string{"co"},
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeBranchNames,
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := defaultManager()
			if err != nil {
				return err
			}

			wt, created, err := mgr.Checkout(worktree.CheckoutOptions{
				Branch: args[0],
				Parent: opts.parent,
				With:   opts.with,
			})
			if err != nil {
				return err
			}

			if created {
				cfg, _, err := mgr.Store().LoadConfig()
				if err != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: failed to load config for hooks: %v\n", err)
				} else {
					mgr.RunPostCreateHooks(cfg, wt.WorktreePath, cmd.ErrOrStderr())
				}
			}

			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "Checked out %q\n", wt.Name)
			shell.PrintWithCD(wt.WorktreePath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.parent, "parent", "p", "", "set parent branch for merge/rebase tracking")
	cmd.Flags().StringSliceVar(&opts.with, "with", nil, "extra include patterns to copy (.gitignore syntax, in addition to .worktreeinclude)")

	return cmd
}
