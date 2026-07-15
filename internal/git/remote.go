package git

import "strings"

// Fetch fetches updates from a remote for a specific branch.
func (g *Git) Fetch(remote, branch string) error {
	_, err := g.run(runOpts{}, "fetch", remote, branch)
	return err
}

// FetchAll updates all remote-tracking refs from every configured remote.
// When prune is true, remote-tracking refs whose upstream branch was deleted
// are removed.
func (g *Git) FetchAll(prune bool) error {
	args := []string{"fetch", "--all", "--quiet"}
	if prune {
		args = append(args, "--prune")
	}
	_, err := g.run(runOpts{}, args...)
	return err
}

// HasRemotes reports whether the repository has any configured remotes.
func (g *Git) HasRemotes() (bool, error) {
	out, err := g.run(runOpts{}, "remote")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}
