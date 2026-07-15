package git

import "strings"

// DiffCached returns the diff between HEAD and the index (staged changes).
// Returns empty string if no staged changes.
func (g *Git) DiffCached() (string, error) {
	return g.run(runOpts{raw: true}, "diff", "--cached")
}

// Diff returns the diff between the index and the working tree (unstaged changes).
// Returns empty string if no unstaged changes.
func (g *Git) Diff() (string, error) {
	return g.run(runOpts{raw: true}, "diff")
}

// DiffNameOnly returns paths of changed files.
// If cached is true, compares index vs HEAD (staged).
// Otherwise compares working tree vs index (unstaged).
// Filter is a git diff-filter value (e.g. "d" to exclude deletions, "D" for deletions only).
// Pass empty string for no filter.
func (g *Git) DiffNameOnly(cached bool, filter string) ([]string, error) {
	args := []string{"diff", "--name-only"}
	if filter != "" {
		args = append(args, "--diff-filter="+filter)
	}
	if cached {
		args = append(args, "--cached")
	}
	out, err := g.run(runOpts{}, args...)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

// CheckoutIndexTo extracts staged files from the index to an arbitrary directory.
// Overwrites existing files. File paths within destDir mirror the repo layout.
func (g *Git) CheckoutIndexTo(destDir string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	prefix := destDir
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	args := make([]string, 0, 4+len(paths))
	args = append(args, "checkout-index", "-f", "--prefix="+prefix, "--")
	args = append(args, paths...)
	_, err := g.run(runOpts{}, args...)
	return err
}

// Add stages the given paths in the index.
func (g *Git) Add(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	args := make([]string, 0, 2+len(paths))
	args = append(args, "add", "--")
	args = append(args, paths...)
	_, err := g.run(runOpts{}, args...)
	return err
}

// Remove deletes paths from the working tree and stages the removal.
func (g *Git) Remove(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	args := make([]string, 0, 2+len(paths))
	args = append(args, "rm", "--")
	args = append(args, paths...)
	_, err := g.run(runOpts{}, args...)
	return err
}
