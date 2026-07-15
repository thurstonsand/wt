package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/thurstonsand/wt/internal/shell"
)

func newShellCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "shell <shell>",
		Short:     "Output shell integration",
		ValidArgs: []string{"bash", "zsh"},
		Long: `Output shell integration script for the specified shell.

Supported shells:
  bash   Bash wrapper function and completions
  zsh    Zsh wrapper function and completions

Usage:
  eval "$(wt shell zsh)"

Add to your shell's rc file for persistent integration.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			switch args[0] {
			case "bash":
				_, _ = fmt.Fprint(out, shell.BashWrapper())
				return cmd.Root().GenBashCompletion(out)
			case "zsh":
				_, _ = fmt.Fprint(out, shell.ZshWrapper())
				return cmd.Root().GenZshCompletion(out)
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}
	return cmd
}
