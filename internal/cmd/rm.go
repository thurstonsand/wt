package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/thurstonsand/wt/internal/shell"
	"github.com/thurstonsand/wt/internal/worktree"
)

func newRmCmd() *cobra.Command {
	var force bool
	var preserveBranch bool
	var validateOnly bool

	cmd := &cobra.Command{
		Use:               "rm [name]",
		Short:             "Remove a worktree",
		ValidArgsFunction: completeWorktreeNames(false),
		Long: `Remove a worktree, its metadata, and branch.

If run from within a worktree, the name argument is optional.
Fails if the worktree has uncommitted changes unless -f is used.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := defaultManager()
			if err != nil {
				return err
			}

			name := ""
			if len(args) > 0 {
				name = args[0]
			}

			opts := worktree.RemoveOptions{
				Name:           name,
				Force:          force,
				PreserveBranch: preserveBranch,
				ValidateOnly:   validateOnly,
			}

			result, err := mgr.Remove(opts)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if validateOnly {
				_, _ = fmt.Fprintln(out, result.WorktreePath)
				if result.TargetPath != "" {
					shell.PrintWithCD(result.TargetPath)
				}
				return nil
			}
			_, _ = fmt.Fprintf(out, "Removed worktree %q\n", result.WorktreeName)
			if result.TargetPath != "" {
				shell.PrintWithCD(result.TargetPath)
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "force removal even if worktree is dirty")
	cmd.Flags().BoolVar(&preserveBranch, "preserve-branch", false, "keep the branch after removing worktree")
	cmd.Flags().BoolVar(&validateOnly, "validate-only", false, "validate removal without changing the worktree")
	_ = cmd.Flags().MarkHidden("validate-only")

	return cmd
}
