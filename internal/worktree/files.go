package worktree

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/thurstonsand/wt/internal/config"
	"github.com/thurstonsand/wt/internal/files"
)

// copyIncludes copies files matching the worktree include patterns into the new
// worktree. Patterns are layered in ascending precedence — project-level
// (<toplevel>/.worktreeinclude), user-level (WT_HOME/worktreeinclude), then any
// extra CLI patterns — so a later "!pattern" negation overrides an earlier
// include. Matches are copied from the source working tree, carrying local
// modifications to tracked files. Runs regardless of clean, so it carries the
// gitignored files that dirty-state transfer deliberately skips.
func (m *Manager) copyIncludes(wtPath string, extra []string) error {
	patterns := slices.Concat(
		readIncludeFile(m.projectIncludePath()),
		readIncludeFile(m.globalStore.UserIncludePath()),
		extra,
	)

	fileList, err := m.git.FilesMatchingPatterns(patterns)
	if err != nil {
		return err
	}
	if len(fileList) == 0 {
		return nil
	}

	return files.CopyFiles(m.git.Dir(), wtPath, fileList)
}

func (m *Manager) projectIncludePath() string {
	return filepath.Join(m.git.Dir(), config.ProjectIncludeFile)
}

// readIncludeFile returns the non-empty, non-comment lines of an include file.
// A missing file contributes nothing.
func readIncludeFile(path string) []string {
	data, err := os.ReadFile(path) //nolint:gosec // path from trusted config locations
	if err != nil {
		return nil
	}
	var patterns []string
	for line := range strings.SplitSeq(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		patterns = append(patterns, trimmed)
	}
	return patterns
}
