package worktree

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/thurstonsand/wt/internal/config"
	"github.com/thurstonsand/wt/internal/files"
	"github.com/thurstonsand/wt/internal/git"
)

type dirtyState struct {
	stagedFiles   []string
	stagedDels    []string
	unstagedFiles []string
	unstagedDels  []string
	untracked     []string
}

func captureDirtyState(source *git.Git) (dirtyState, error) {
	var state dirtyState
	var err error

	state.stagedFiles, err = source.DiffNameOnly(true, "d")
	if err != nil {
		return state, fmt.Errorf("failed to list staged files: %w", err)
	}
	state.stagedDels, err = source.DiffNameOnly(true, "D")
	if err != nil {
		return state, fmt.Errorf("failed to list staged deletions: %w", err)
	}
	state.unstagedFiles, err = source.DiffNameOnly(false, "d")
	if err != nil {
		return state, fmt.Errorf("failed to list unstaged files: %w", err)
	}
	state.unstagedDels, err = source.DiffNameOnly(false, "D")
	if err != nil {
		return state, fmt.Errorf("failed to list unstaged deletions: %w", err)
	}
	state.untracked, err = source.UntrackedFiles()
	if err != nil {
		return state, fmt.Errorf("failed to list untracked files: %w", err)
	}
	return state, nil
}

func (s dirtyState) apply(source, target *git.Git) error {
	if err := source.CheckoutIndexTo(target.Dir(), s.stagedFiles); err != nil {
		return fmt.Errorf("failed to copy staged files: %w", err)
	}
	if err := removeFiles(target.Dir(), s.stagedDels); err != nil {
		return fmt.Errorf("failed to apply staged deletions: %w", err)
	}
	if err := target.AddAll(slices.Concat(s.stagedFiles, s.stagedDels)); err != nil {
		return fmt.Errorf("failed to stage files: %w", err)
	}
	if err := files.CopyFiles(source.Dir(), target.Dir(), s.unstagedFiles); err != nil {
		return fmt.Errorf("failed to copy unstaged files: %w", err)
	}
	if err := removeFiles(target.Dir(), s.unstagedDels); err != nil {
		return fmt.Errorf("failed to apply unstaged deletions: %w", err)
	}
	if err := files.CopyFiles(source.Dir(), target.Dir(), s.untracked); err != nil {
		return fmt.Errorf("failed to copy untracked files: %w", err)
	}
	return nil
}

func (s dirtyState) countWith(paths []string) int {
	unique := make(map[string]struct{})
	for _, path := range slices.Concat(paths, s.stagedFiles, s.stagedDels, s.unstagedFiles, s.unstagedDels, s.untracked) {
		unique[path] = struct{}{}
	}
	return len(unique)
}

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
