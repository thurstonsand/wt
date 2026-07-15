package worktree

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/thurstonsand/wt/internal/config"
)

// RunPostCreateHooks runs built-in and custom hooks after worktree creation.
// All errors are reported as warnings; hooks never block worktree creation.
func (m *Manager) RunPostCreateHooks(cfg config.GlobalConfig, wtPath string, w io.Writer) {
	m.runDirenvHook(cfg, wtPath, w)
	m.runPostCreateCommands(cfg, wtPath, w)
}

func (m *Manager) runDirenvHook(cfg config.GlobalConfig, wtPath string, w io.Writer) {
	if !cfg.Direnv {
		return
	}

	envrc := filepath.Join(wtPath, ".envrc")
	if _, err := os.Stat(envrc); err != nil {
		return
	}

	if err := m.cmdRunner.Run(wtPath, "direnv", "allow"); err != nil {
		_, _ = fmt.Fprintf(w, "warning: direnv allow failed: %v\n", err)
	}
}

func (m *Manager) runPostCreateCommands(cfg config.GlobalConfig, wtPath string, w io.Writer) {
	for _, cmd := range cfg.PostCreate {
		if err := m.cmdRunner.Run(wtPath, "sh", "-c", cmd); err != nil {
			_, _ = fmt.Fprintf(w, "warning: post_create hook failed: %s: %v\n", cmd, err)
		}
	}
}
