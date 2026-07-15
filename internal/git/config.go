package git

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// BranchMeta holds wt-specific metadata stored in git config.
// Stored as branch.<name>.wt-* keys.
type BranchMeta struct {
	Parent string `gitconfig:"wt-parent"`
}

// GetBranchMeta retrieves wt metadata for a branch from git config.
func (g *Git) GetBranchMeta(branch string) (BranchMeta, error) {
	var meta BranchMeta
	v := reflect.ValueOf(&meta).Elem()
	t := v.Type()

	for i := range t.NumField() {
		field := t.Field(i)
		tag := field.Tag.Get("gitconfig")
		if tag == "" {
			continue
		}

		key := fmt.Sprintf("branch.%s.%s", branch, tag)
		val, err := g.run(runOpts{}, "config", "--get", key)
		if err != nil {
			if IsConfigKeyNotFound(err) {
				continue
			}
			return meta, err
		}
		v.Field(i).SetString(val)
	}

	return meta, nil
}

// SetBranchMeta stores wt metadata for a branch in git config.
func (g *Git) SetBranchMeta(branch string, meta BranchMeta) error {
	v := reflect.ValueOf(meta)
	t := v.Type()

	for i := range t.NumField() {
		field := t.Field(i)
		tag := field.Tag.Get("gitconfig")
		if tag == "" {
			continue
		}

		val := v.Field(i).String()
		if val == "" {
			continue
		}

		key := fmt.Sprintf("branch.%s.%s", branch, tag)
		if _, err := g.run(runOpts{}, "config", key, val); err != nil {
			return err
		}
	}
	return nil
}

// GetBranchesWithParent returns all branches that have wt-parent metadata set.
// The returned map is branch name → parent branch name.
func (g *Git) GetBranchesWithParent() (map[string]string, error) {
	out, err := g.run(runOpts{}, "config", "--get-regexp", `branch\..*\.wt-parent`)
	if err != nil {
		if IsConfigKeyNotFound(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	if out == "" {
		return map[string]string{}, nil
	}

	result := make(map[string]string)
	for line := range strings.SplitSeq(out, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		parent := parts[1]
		branch := strings.TrimPrefix(key, "branch.")
		branch = strings.TrimSuffix(branch, ".wt-parent")
		result[branch] = parent
	}
	return result, nil
}

// IsConfigKeyNotFound returns true if the error indicates a config key was not found.
func IsConfigKeyNotFound(err error) bool {
	var execErr *ExecError
	if errors.As(err, &execErr) {
		return execErr.ExitCode == 1 && execErr.Stderr == ""
	}
	return false
}
