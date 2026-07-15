package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/thurstonsand/wt/internal/config"
	"github.com/thurstonsand/wt/internal/worktree"
)

func newListCmd() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List worktrees",
		Long:    "List all worktrees with branch, parent, and dirty status.",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if all {
				return listAllRepos(cmd)
			}
			return listCurrentRepo(cmd)
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "List worktrees across all repositories")

	return cmd
}

func listAllRepos(cmd *cobra.Command) error {
	store := config.DefaultGlobalStore()

	res, err := worktree.ListAllRepos(store)
	if err != nil {
		return err
	}

	stderr := cmd.OutOrStderr()
	for _, s := range res.Stale {
		_, _ = fmt.Fprintf(stderr, "warning: skipping %s/%s: %v\n", s.RepoName, s.Name, s.Err)
	}

	out := cmd.OutOrStdout()
	if len(res.Worktrees) == 0 {
		_, _ = fmt.Fprintln(out, "No worktrees found.")
		return nil
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "REPO\tNAME\tBRANCH\tPARENT\tSTATE"); err != nil {
		return err
	}

	for _, wt := range res.Worktrees {
		state := wt.State()
		parent := wt.Parent
		if parent == "" {
			parent = "-"
		}
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", wt.RepoName, wt.Name, wt.Branch, parent, state); err != nil {
			return err
		}
	}

	return w.Flush()
}

func listCurrentRepo(cmd *cobra.Command) error {
	mgr, err := defaultManager()
	if err != nil {
		return err
	}

	result, err := mgr.ListAll()
	if err != nil {
		return err
	}

	cwd, err := currentWorktreePath()
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "NAME\tBRANCH\tPARENT\tSTATE"); err != nil {
		return err
	}

	rootName := "(" + rootWorktreeName + ")"
	if pathsEqual(cwd, result.Main.Path) {
		rootName += " *"
	}
	if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t\n", rootName, result.Main.Branch, "-"); err != nil {
		return err
	}

	for _, wt := range result.Worktrees {
		state := wt.State()
		parent := wt.Parent
		if parent == "" {
			parent = "-"
		}
		name := wt.Name
		if pathsEqual(cwd, wt.WorktreePath) {
			name += " *"
		}
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", name, wt.Branch, parent, state); err != nil {
			return err
		}
	}

	if err := w.Flush(); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(out, "\nrepo: %s\n", shortenHome(result.Main.Path))

	return nil
}

func currentWorktreePath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		return cwd, nil //nolint:nilerr // fall back to unresolved path if symlink eval fails
	}
	return resolved, nil
}

func shortenHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

func pathsEqual(a, b string) bool {
	cleanA := filepath.Clean(a)
	cleanB := filepath.Clean(b)
	return cleanA == cleanB
}
