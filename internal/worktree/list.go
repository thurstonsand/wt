package worktree

import (
	"fmt"
	"path/filepath"

	"github.com/thurstonsand/wt/internal/config"
	"github.com/thurstonsand/wt/internal/git"
)

// ListResult holds the main worktree info alongside linked worktrees.
type ListResult struct {
	Main      git.WorktreeInfo
	Worktrees []*Worktree
}

// ListAll returns the main worktree and all linked worktrees for this repository.
// Uses git worktree list as the source of truth and enriches linked worktrees with branch metadata.
func (m *Manager) ListAll() (*ListResult, error) {
	main, err := m.git.MainWorktree()
	if err != nil {
		return nil, err
	}

	linked, err := m.git.LinkedWorktrees()
	if err != nil {
		return nil, err
	}

	var worktrees []*Worktree
	for _, gw := range linked {
		meta, err := m.git.GetBranchMeta(gw.Branch)
		if err != nil {
			return nil, fmt.Errorf("failed to get branch metadata for %s: %w", gw.Branch, err)
		}

		wt := &Worktree{
			RepoName:     m.repoName,
			Name:         filepath.Base(gw.Path),
			WorktreePath: gw.Path,
			Branch:       gw.Branch,
			BranchMeta:   meta,
			git:          git.New(gw.Path),
		}
		worktrees = append(worktrees, wt)
	}
	return &ListResult{Main: main, Worktrees: worktrees}, nil
}

// Get retrieves a specific worktree by name.
// Matches by:
// 1. Folder name exact match (filepath.Base(path) == name)
// 2. Branch name exact match
// 3. Unsanitized name match (name with "--" converted to "/")
func (m *Manager) Get(name string) (*Worktree, error) {
	linkedWorktrees, err := m.git.LinkedWorktrees()
	if err != nil {
		return nil, err
	}

	unsanitized := config.UnsanitizePathComponent(name)

	for _, gw := range linkedWorktrees {
		folderName := filepath.Base(gw.Path)

		if folderName == name || gw.Branch == name || gw.Branch == unsanitized {
			meta, err := m.git.GetBranchMeta(gw.Branch)
			if err != nil {
				return nil, fmt.Errorf("failed to get branch metadata: %w", err)
			}

			return &Worktree{
				RepoName:     m.repoName,
				Name:         folderName,
				WorktreePath: gw.Path,
				Branch:       gw.Branch,
				BranchMeta:   meta,
				git:          git.New(gw.Path),
			}, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrWorktreeNotFound, name)
}

// Exists checks if a worktree with the given name exists.
func (m *Manager) Exists(name string) bool {
	_, err := m.Get(name)
	return err == nil
}

// StaleWorktree describes a worktree directory that could not be read.
type StaleWorktree struct {
	RepoName string
	Name     string
	Path     string
	Err      error
}

// ListAllReposResult holds the output of a cross-repo worktree scan.
type ListAllReposResult struct {
	Worktrees []*Worktree
	Stale     []StaleWorktree
}

// ListAllRepos returns worktrees across all repositories managed under the global store.
// Directories where git operations fail are reported in Result.Stale.
func ListAllRepos(store *config.GlobalStore) (*ListAllReposResult, error) {
	dirs, err := store.ListWorktreeDirs()
	if err != nil {
		return nil, err
	}

	var res ListAllReposResult
	for _, d := range dirs {
		g := git.New(d.Path)
		branch, err := g.CurrentBranch()
		if err != nil {
			res.Stale = append(res.Stale, StaleWorktree{
				RepoName: d.RepoName,
				Name:     d.Name,
				Path:     d.Path,
				Err:      err,
			})
			continue
		}
		meta, _ := g.GetBranchMeta(branch)

		res.Worktrees = append(res.Worktrees, &Worktree{
			RepoName:     d.RepoName,
			Name:         d.Name,
			WorktreePath: d.Path,
			Branch:       branch,
			BranchMeta:   meta,
			git:          g,
		})
	}
	return &res, nil
}
