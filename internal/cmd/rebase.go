package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/thurstonsand/wt/internal/worktree"
)

func newRebaseCmd() *cobra.Command {
	var opts struct {
		onto string
	}

	cmd := &cobra.Command{
		Use:               "rebase [name]",
		Short:             "Update worktree by rebasing onto parent branch",
		ValidArgsFunction: completeWorktreeNames(false),
		Long: `Rebase the worktree branch onto the latest parent branch commits.

If run from within a worktree, the name argument is optional.
Uncommitted changes are automatically stashed and restored.

Examples:
  wt rebase feature-x              # Rebase feature-x onto its parent
  wt rebase                        # Rebase current worktree
  wt rebase feature-x --onto dev   # Rebase onto dev (changes parent)`,
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

			wt, err := mgr.Rebase(worktree.RebaseOptions{
				Name: name,
				Onto: opts.onto,
			})
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Rebased worktree %q\n", wt.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.onto, "onto", "", "rebase onto different branch (updates parent)")

	return cmd
}
