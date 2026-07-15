package git

import "strings"

// StashPushAll stashes all changes including untracked files, with a message.
// Returns hasStash=false when there was nothing to stash.
func (g *Git) StashPushAll(message string) (hasStash bool, err error) {
	before, err := g.stashCount()
	if err != nil {
		return false, err
	}
	if _, err = g.run(runOpts{}, "stash", "push", "--include-untracked", "-m", message); err != nil {
		return false, err
	}
	after, err := g.stashCount()
	if err != nil {
		return false, err
	}
	return after > before, nil
}

// StashPopIndex restores the most recent stash, preserving the staged/unstaged
// split (--index). Returns an error on conflict or when --index cannot reapply.
func (g *Git) StashPopIndex() error {
	_, err := g.run(runOpts{}, "stash", "pop", "--index")
	return err
}

// StashPop restores the most recent stash without preserving the staged split.
// Used as a fallback when StashPopIndex cannot reapply the index cleanly.
func (g *Git) StashPop() error {
	_, err := g.run(runOpts{}, "stash", "pop")
	return err
}

func (g *Git) stashCount() (int, error) {
	out, err := g.run(runOpts{}, "stash", "list")
	if err != nil {
		return 0, err
	}
	if out == "" {
		return 0, nil
	}
	return strings.Count(out, "\n") + 1, nil
}
