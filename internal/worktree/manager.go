// Package worktree manages git worktrees with metadata persistence.
package worktree

import (
	"os/exec"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/thurstonsand/wt/internal/config"
	"github.com/thurstonsand/wt/internal/git"
)

// CommandRunner executes external commands in a given directory.
type CommandRunner interface {
	Run(dir, name string, args ...string) error
}

type execRunner struct{}

func (execRunner) Run(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...) //nolint:gosec // args come from user config, not untrusted input
	cmd.Dir = dir
	return cmd.Run()
}

// Manager coordinates worktree operations between git and config storage.
type Manager struct {
	repoName    string
	git         *git.Git
	globalStore *config.GlobalStore
	cmdRunner   CommandRunner
}

// ManagerOption configures a Manager instance.
type ManagerOption func(*Manager)

// WithGit sets a custom Git instance for the source repository.
func WithGit(g *git.Git) ManagerOption {
	return func(m *Manager) {
		m.git = g
		m.repoName = filepath.Base(g.Dir())
	}
}

// WithCommandRunner sets a custom CommandRunner for executing hook commands.
func WithCommandRunner(r CommandRunner) ManagerOption {
	return func(m *Manager) { m.cmdRunner = r }
}

// WithGlobalStore sets a custom GlobalStore for global config.
func WithGlobalStore(s *config.GlobalStore) ManagerOption {
	return func(m *Manager) { m.globalStore = s }
}

// NewManager creates a Manager for the given repository directory.
func NewManager(repoDir string, opts ...ManagerOption) (*Manager, error) {
	m := &Manager{
		repoName: filepath.Base(repoDir),
	}
	for _, opt := range opts {
		opt(m)
	}
	if m.git == nil {
		m.git = git.New(toplevelOrDir(repoDir))
	}
	if m.globalStore == nil {
		m.globalStore = config.DefaultGlobalStore()
	}
	if m.cmdRunner == nil {
		m.cmdRunner = execRunner{}
	}

	// The repo name must key off the main worktree, not the current directory.
	// Forking from inside a linked worktree would otherwise group the new
	// worktree under the current worktree's basename instead of the repo's.
	if mainWt, err := m.git.MainWorktree(); err == nil {
		m.repoName = filepath.Base(mainWt.Path)
		_ = m.globalStore.RegisterRepo(mainWt.Path)
	}

	return m, nil
}

// Git returns the underlying Git instance.
func (m *Manager) Git() *git.Git {
	return m.git
}

// Store returns the underlying GlobalStore instance for global config.
func (m *Manager) Store() *config.GlobalStore {
	return m.globalStore
}

// RepoName returns the repository name used for worktree paths.
func (m *Manager) RepoName() string {
	return m.repoName
}

// toplevelOrDir resolves the working-tree root of dir, falling back to dir
// itself when it cannot be resolved (e.g. not inside a repository). Anchoring
// on the toplevel keeps source-repo operations independent of the invocation
// subdirectory.
func toplevelOrDir(dir string) string {
	if top, err := git.New(dir).Toplevel(); err == nil && top != "" {
		return top
	}
	return dir
}

// GenerateName creates a unique worktree name with "wt-" prefix.
func GenerateName() string {
	id := uuid.New().String()
	return "wt-" + id[:8]
}

// WorktreePath returns the filesystem path for a worktree directory.
func (m *Manager) WorktreePath(name string) string {
	return m.globalStore.WorktreePath(m.repoName, name)
}
