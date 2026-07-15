package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "path [name]",
		Short:             "Print worktree path",
		ValidArgsFunction: completeWorktreeNames(true),
		Long: `Print the filesystem path for a worktree.

With no arguments, prints the main repository path.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := defaultManager()
			if err != nil {
				return err
			}

			var path string
			if len(args) == 0 || args[0] == rootWorktreeName {
				info, err := mgr.Git().MainWorktree()
				if err != nil {
					return err
				}
				path = info.Path
			} else {
				wt, err := mgr.Get(args[0])
				if err != nil {
					return err
				}
				path = wt.WorktreePath
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		},
	}
}
