// Package git provides wrappers for git commands.
package git

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Git executes git commands in a specific repository.
type Git struct {
	dir string
	env []string
}

// Option configures a Git instance.
type Option func(*Git)

// WithEnv sets custom environment variables for git commands.
func WithEnv(env []string) Option {
	return func(g *Git) { g.env = env }
}

// New creates a Git instance for the given directory.
func New(dir string, opts ...Option) *Git {
	g := &Git{dir: dir}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// Dir returns the repository directory.
func (g *Git) Dir() string {
	return g.dir
}

// Toplevel returns the absolute path to the root of the working tree the
// directory belongs to. For a linked worktree this is that worktree's root.
func (g *Git) Toplevel() (string, error) {
	return g.run(runOpts{}, "rev-parse", "--show-toplevel")
}

// CommonDir returns the absolute path to the shared git directory.
// This is the .git directory for the main repo, even when called from a worktree.
func (g *Git) CommonDir() (string, error) {
	commonDir, err := g.run(runOpts{}, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(commonDir) {
		return commonDir, nil
	}
	return filepath.Join(g.dir, commonDir), nil
}

// Env returns the environment variables used for git commands.
func (g *Git) Env() []string {
	return g.env
}

type runResult struct {
	stdout   string
	exitCode int
}

type runOpts struct {
	stdin string
	raw   bool
}

func (g *Git) run(opts runOpts, args ...string) (string, error) {
	r, err := g.exec(opts, args...)
	if opts.raw {
		return r.stdout, err
	}
	return strings.TrimSpace(r.stdout), err
}

func (g *Git) exec(opts runOpts, args ...string) (runResult, error) {
	cmd := exec.Command("git", args...) //nolint:gosec // args are constructed internally, not from untrusted input
	cmd.Dir = g.dir
	if g.env != nil {
		cmd.Env = g.env
	} else {
		cmd.Env = os.Environ()
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if opts.stdin != "" {
		cmd.Stdin = strings.NewReader(opts.stdin)
	}

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
		return runResult{exitCode: exitCode}, &ExecError{
			Cmd:      args[0],
			Stderr:   strings.TrimSpace(stderr.String()),
			ExitCode: exitCode,
		}
	}
	return runResult{stdout: stdout.String(), exitCode: 0}, nil
}
