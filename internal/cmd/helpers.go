package cmd

import (
	"fmt"
	"os"

	"github.com/thurstonsand/wt/internal/worktree"
)

func defaultManager() (*worktree.Manager, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	return worktree.NewManager(wd)
}
