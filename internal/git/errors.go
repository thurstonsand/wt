package git

import (
	"errors"
	"fmt"
	"strings"
)

// ExecError represents an error from executing a git command.
type ExecError struct {
	Cmd      string
	Stderr   string
	ExitCode int
}

func (e *ExecError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("git %s: %s", e.Cmd, e.Stderr)
	}
	return fmt.Sprintf("git %s: exit code %d", e.Cmd, e.ExitCode)
}

// Is reports whether this error matches target based on stderr content.
func (e *ExecError) Is(target error) bool {
	if t, ok := target.(*ExecError); ok && t.Stderr != "" {
		return strings.Contains(e.Stderr, t.Stderr)
	}
	return false
}

// ErrNotGitRepo indicates the directory is not a git repository.
var ErrNotGitRepo = &ExecError{Stderr: "not a git repository"}

// ErrNoDefaultBranch indicates no main/master branch was found.
var ErrNoDefaultBranch = errors.New("no default branch found (main or master)")

// ErrNoWorktrees indicates no worktrees were found.
var ErrNoWorktrees = errors.New("no worktrees found")

// IsErrNotGitRepo returns true if the error indicates the directory is not a git repository.
func IsErrNotGitRepo(err error) bool {
	return errors.Is(err, ErrNotGitRepo)
}

// IsErrNoDefaultBranch returns true if the error indicates no default branch was found.
func IsErrNoDefaultBranch(err error) bool {
	return errors.Is(err, ErrNoDefaultBranch)
}
