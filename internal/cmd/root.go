// Package cmd implements CLI commands for the wt worktree helper.
package cmd

import (
	"runtime/debug"

	"github.com/spf13/cobra"
)

// Release builds set version with a linker flag; module metadata supports go install.
var version = "dev"

func init() {
	info, ok := debug.ReadBuildInfo()
	version = resolveVersion(version, info, ok)
}

func resolveVersion(linkerVersion string, info *debug.BuildInfo, hasBuildInfo bool) string {
	if linkerVersion != "dev" {
		return linkerVersion
	}
	if hasBuildInfo && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return linkerVersion
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
