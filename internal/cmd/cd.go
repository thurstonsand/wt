package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newCdCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "cd [name]",
		Short:             "Change directory to a worktree",
		ValidArgsFunction: completeWorktreeNames(true),
		Long: `Change directory to a worktree.

With no arguments, changes to the main repository directory.

This command requires shell integration to work. The shell wrapper
intercepts 'wt cd' and uses the shell's builtin cd command.

To enable, add to your shell's rc file:

  # Bash (~/.bashrc)
  eval "$(wt shell bash)"

  # Zsh (~/.zshrc)
  eval "$(wt shell zsh)"

Then restart your shell or run: source ~/.zshrc`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, _ = fmt.Fprintln(cmd.OutOrStderr(), `Shell integration required for 'wt cd'.

Add to your shell config:

  eval "$(wt shell zsh)"   # for zsh
  eval "$(wt shell bash)"  # for bash

Then restart your shell.`)
			return ErrShellIntegrationRequired
		},
	}
}
