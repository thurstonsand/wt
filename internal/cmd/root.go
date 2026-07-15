// Package cmd implements CLI commands for the wt worktree helper.
package cmd

import (
	"runtime/debug"

	"github.com/spf13/cobra"
)

var version = "dev"

func init() {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		version = info.Main.Version
	}
}

// NewRootCmd creates the root command for wt.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "wt",
		Short:         "Git worktree helper",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(newCheckoutCmd())
	cmd.AddCommand(newForkCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newMergeCmd())
	cmd.AddCommand(newPathCmd())
	cmd.AddCommand(newRebaseCmd())
	cmd.AddCommand(newRebranchCmd())
	cmd.AddCommand(newRmCmd())
	cmd.AddCommand(newShellCmd())
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newCdCmd())
	cmd.AddCommand(newPruneCmd())
	return cmd
}

// Execute runs the root command.
func Execute() error {
	return NewRootCmd().Execute()
}
