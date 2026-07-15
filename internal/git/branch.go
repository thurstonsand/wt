package git

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// CurrentBranch returns the name of the current branch.
func (g *Git) CurrentBranch() (string, error) {
	return g.run(runOpts{}, "rev-parse", "--abbrev-ref", "HEAD")
}

// DefaultBranch returns "main" or "master", whichever exists.
// Checks main first, then master.
func (g *Git) DefaultBranch() (string, error) {
	for _, branch := range []string{"main", "master"} {
		exists, err := g.LocalBranchExists(branch)
		if err != nil {
			return "", err
		}
		if exists {
			return branch, nil
		}
	}
	return "", ErrNoDefaultBranch
}

// IsDirty returns true if there are uncommitted changes (staged or unstaged).
func (g *Git) IsDirty() (bool, error) {
	out, err := g.run(runOpts{}, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return out != "", nil
}

// LocalBranchExists checks if a local branch exists under refs/heads/.
func (g *Git) LocalBranchExists(name string) (bool, error) {
	r, _ := g.exec(runOpts{}, "rev-parse", "--verify", "--quiet", "refs/heads/"+name)
	return r.exitCode == 0, nil
}

// RefExists checks if a ref exists (local branch, remote-tracking branch, or tag).
func (g *Git) RefExists(name string) (bool, error) {
	r, _ := g.exec(runOpts{}, "rev-parse", "--verify", "--quiet", name)
	return r.exitCode == 0, nil
}

// RemoteBranchExists checks whether any remote-tracking branch matches name
// (e.g. refs/remotes/origin/<name>). Returns the qualified short name of the
// first match ("origin/<name>") for use in messages.
func (g *Git) RemoteBranchExists(name string) (bool, string, error) {
	out, err := g.run(runOpts{}, "for-each-ref", "--format=%(refname)", "refs/remotes/")
	if err != nil {
		return false, "", err
	}
	for ref := range strings.SplitSeq(out, "\n") {
		if ref == "" {
			continue
		}
		parts := strings.TrimPrefix(ref, "refs/remotes/")
		remote, short, ok := strings.Cut(parts, "/")
		if !ok || short == "HEAD" {
			continue
		}
		if short == name {
			return true, remote + "/" + short, nil
		}
	}
	return false, "", nil
}

// HasUpstream reports whether the branch has an upstream configured
// (branch.<name>.merge set), i.e. it was pushed and given a tracking ref.
func (g *Git) HasUpstream(branch string) (bool, error) {
	_, err := g.run(runOpts{}, "config", "--get", "branch."+branch+".merge")
	if err != nil {
		if IsConfigKeyNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// RemoteTrackingExists reports whether the remote-tracking ref
// refs/remotes/<remote>/<branch> currently exists locally. After a
// fetch --prune, this goes false once the branch is deleted on the remote.
func (g *Git) RemoteTrackingExists(remote, branch string) (bool, error) {
	return g.RefExists("refs/remotes/" + remote + "/" + branch)
}

// UpstreamGone reports whether the branch's configured upstream has been
// deleted on the remote, using git's own tracking signal (%(upstream:track)
// is "[gone]"). This follows the actual configured upstream regardless of its
// name. A branch has "landed" when its upstream is gone: it was pushed, merged
// via PR/MR, and deleted on the remote. Detection is local; freshness depends
// on the last fetch --prune.
func (g *Git) UpstreamGone(branch string) (bool, error) {
	out, err := g.run(runOpts{}, "for-each-ref", "--format=%(upstream:track)", "refs/heads/"+branch)
	if err != nil {
		return false, err
	}
	return out == "[gone]", nil
}

// DefaultRemoteBranch resolves the default branch on the given remote,
// origin/HEAD symbolic ref, then main, then master.
// Returns the short branch name (e.g. "main").
func (g *Git) DefaultRemoteBranch(remote string) (string, error) {
	ref, err := g.run(runOpts{}, "symbolic-ref", "--quiet", "refs/remotes/"+remote+"/HEAD")
	if err == nil && ref != "" {
		return strings.TrimPrefix(ref, "refs/remotes/"+remote+"/"), nil
	}
	for _, branch := range []string{"main", "master"} {
		exists, err := g.RemoteTrackingExists(remote, branch)
		if err != nil {
			return "", err
		}
		if exists {
			return branch, nil
		}
	}
	return "", ErrNoDefaultBranch
}

// CreateBranch creates a new branch at the given commit (or HEAD if empty).
func (g *Git) CreateBranch(name, commit string) error {
	if commit == "" {
		commit = "HEAD"
	}
	_, err := g.run(runOpts{}, "branch", name, commit)
	return err
}

// Switch changes the working tree to the specified existing branch.
func (g *Git) Switch(branch string) error {
	_, err := g.run(runOpts{}, "switch", branch)
	return err
}

// SwitchCreate creates a new branch at start point and switches to it,
// equivalent to `git switch -c <name> <start>`. Fails if the branch exists.
func (g *Git) SwitchCreate(branch, start string) error {
	_, err := g.run(runOpts{}, "switch", "-c", branch, start)
	return err
}

// DeleteBranch deletes a branch. If force is true, uses -D (force delete),
// otherwise uses -d (fails if not fully merged).
func (g *Git) DeleteBranch(name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := g.run(runOpts{}, "branch", flag, name)
	return err
}

// UntrackedFiles returns a list of untracked files (respects .gitignore).
func (g *Git) UntrackedFiles() ([]string, error) {
	out, err := g.run(runOpts{}, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	var files []string
	for line := range strings.SplitSeq(out, "\n") {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// FilesMatchingPatterns returns working-tree files (tracked and untracked) that
// match the given .gitignore-style patterns. Patterns are the sole authority:
// the repository's own .gitignore is not consulted, so a file is returned iff
// it matches a pattern here, regardless of its ignore status. Later patterns
// win (a trailing "!pattern" negation excludes an earlier match). Paths are
// returned relative to the working-tree root. Empty patterns yield no files.
func (g *Git) FilesMatchingPatterns(patterns []string) ([]string, error) {
	if len(patterns) == 0 {
		return nil, nil
	}

	tmp, err := os.CreateTemp("", "wt-include-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err = tmp.WriteString(strings.Join(patterns, "\n") + "\n"); err != nil {
		_ = tmp.Close()
		return nil, err
	}
	if err = tmp.Close(); err != nil {
		return nil, err
	}

	out, err := g.run(runOpts{}, "ls-files", "--cached", "--others", "--ignored", "--exclude-from="+tmp.Name(), "--full-name")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	var files []string
	for line := range strings.SplitSeq(out, "\n") {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// ListBranches returns local branch names. When includeRemote is
// true it also appends remote branch names (with the remote prefix stripped),
// covering branches that git worktree add can resolve via automatic tracking
// branch creation.
func (g *Git) ListBranches(includeRemote bool) ([]string, error) {
	refspecs := []string{"refs/heads/"}
	if includeRemote {
		refspecs = append(refspecs, "refs/remotes/")
	}
	args := append([]string{"for-each-ref", "--format=%(refname)"}, refspecs...)
	out, err := g.run(runOpts{}, args...)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	seen := make(map[string]bool)
	var branches []string
	for ref := range strings.SplitSeq(out, "\n") {
		if ref == "" {
			continue
		}
		var name string
		switch {
		case strings.HasPrefix(ref, "refs/heads/"):
			name = strings.TrimPrefix(ref, "refs/heads/")
		case strings.HasPrefix(ref, "refs/remotes/"):
			// "refs/remotes/origin/feature" → "feature"
			// Skip HEAD pointers like "refs/remotes/origin/HEAD"
			parts := strings.TrimPrefix(ref, "refs/remotes/")
			_, short, ok := strings.Cut(parts, "/")
			if !ok || short == "HEAD" {
				continue
			}
			name = short
		default:
			continue
		}
		if !seen[name] {
			seen[name] = true
			branches = append(branches, name)
		}
	}
	return branches, nil
}

// RevParse runs git rev-parse with the given arguments and returns the output.
func (g *Git) RevParse(args ...string) (string, error) {
	cmdArgs := append([]string{"rev-parse"}, args...)
	return g.run(runOpts{}, cmdArgs...)
}

// HeadCommitTime returns the committer timestamp of HEAD.
func (g *Git) HeadCommitTime() (time.Time, error) {
	out, err := g.run(runOpts{}, "log", "-1", "--format=%ct")
	if err != nil {
		return time.Time{}, err
	}
	epoch, err := strconv.ParseInt(out, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(epoch, 0), nil
}
