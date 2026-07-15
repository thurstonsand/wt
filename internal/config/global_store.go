package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v4"
)

// GlobalStore manages global configuration persistence at ~/.wt.
type GlobalStore struct {
	rootDir string
}

// NewGlobalStore creates a GlobalStore with the given root directory.
func NewGlobalStore(rootDir string) *GlobalStore {
	return &GlobalStore{rootDir: resolveDir(rootDir)}
}

// DefaultGlobalStore returns a GlobalStore using WT_HOME or ~/.wt as the root directory.
func DefaultGlobalStore() *GlobalStore {
	if root := os.Getenv("WT_HOME"); root != "" {
		return &GlobalStore{rootDir: resolveDir(root)}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		panic("cannot determine home directory: " + err.Error())
	}
	return &GlobalStore{rootDir: resolveDir(filepath.Join(home, ".wt"))}
}

// resolveDir returns a canonical absolute path, resolving symlinks when possible.
func resolveDir(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return abs
}

// RootDir returns the store's root directory.
func (s *GlobalStore) RootDir() string {
	return s.rootDir
}

func (s *GlobalStore) pathOf(elem ...string) string {
	return filepath.Join(append([]string{s.rootDir}, elem...)...)
}

func (s *GlobalStore) loadYAML(v any, elem ...string) error {
	data, err := os.ReadFile(s.pathOf(elem...))
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, v)
}

func (s *GlobalStore) worktreeBasePath() string {
	return s.pathOf("worktrees")
}

// WorktreePath returns the filesystem path for a worktree directory.
func (s *GlobalStore) WorktreePath(repoName, wtName string) string {
	return filepath.Join(s.worktreeBasePath(), repoName, SanitizePathComponent(wtName))
}

// IsManagedPath reports whether the given path falls under the managed worktree directory.
func (s *GlobalStore) IsManagedPath(path string) bool {
	return strings.HasPrefix(path, s.worktreeBasePath()+string(filepath.Separator))
}

// ConfigPath returns the path to the config file.
func (s *GlobalStore) ConfigPath() string {
	return s.pathOf(configFile)
}

// UserIncludePath returns the path to the user-level worktreeinclude file.
func (s *GlobalStore) UserIncludePath() string {
	return s.pathOf(userIncludeFile)
}

// SetConfigValue updates a single key in the config file, preserving comments.
func (s *GlobalStore) SetConfigValue(key string, value any) error {
	path := s.ConfigPath()

	// Read existing file as node tree to preserve comments
	data, err := os.ReadFile(path) //nolint:gosec // trusted path
	if err != nil {
		if os.IsNotExist(err) {
			// Create default config first
			if _, _, err = s.LoadConfig(); err != nil {
				return err
			}
			data, err = os.ReadFile(path) //nolint:gosec // trusted path
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	var doc yaml.Node
	if err = yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	// Document should have a mapping node as content
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return errors.New("invalid config structure")
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return errors.New("config root must be a mapping")
	}

	// Find and update the key
	var valueNode yaml.Node
	if err = valueNode.Encode(value); err != nil {
		return fmt.Errorf("encoding value: %w", err)
	}

	found := false
	for i := 0; i < len(root.Content)-1; i += 2 {
		if root.Content[i].Value == key {
			root.Content[i+1] = &valueNode
			found = true
			break
		}
	}

	if !found {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
		root.Content = append(root.Content, keyNode, &valueNode)
	}

	// Write back
	out, err := yaml.Marshal(&doc)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(path, out, 0600)
}

// SetConfigValueFromString parses a string input and updates the config file.
func (s *GlobalStore) SetConfigValueFromString(key, input string) error {
	field := GetFieldMeta(key)
	if field == nil {
		return UnknownKeyError(key)
	}
	val, err := field.Parse(input)
	if err != nil {
		return fmt.Errorf("invalid value for %s: %w", key, err)
	}
	return s.SetConfigValue(key, val)
}

// ErrNotList is returned when a list operation is attempted on a non-list field.
var ErrNotList = errors.New("not a list field")

func (s *GlobalStore) validateListField(key string) error {
	field := GetFieldMeta(key)
	if field == nil {
		return UnknownKeyError(key)
	}
	if !field.IsList {
		return fmt.Errorf("%w: %q does not support add/remove", ErrNotList, key)
	}
	return nil
}

// AddToConfigList appends an item to a list config field.
func (s *GlobalStore) AddToConfigList(key, item string) error {
	if err := s.validateListField(key); err != nil {
		return err
	}
	cfg, _, err := s.LoadConfig()
	if err != nil {
		return err
	}
	current := cfg.GetListValue(key)
	if slices.Contains(current, item) {
		return nil
	}
	return s.SetConfigValue(key, append(current, item))
}

// RemoveFromConfigList removes an item from a list config field by exact match.
func (s *GlobalStore) RemoveFromConfigList(key, item string) (string, error) {
	return s.removeFromConfigList(key, func(entries []string) (int, error) {
		i := slices.Index(entries, item)
		if i < 0 {
			return -1, nil
		}
		return i, nil
	})
}

// RemoveFromConfigListByIndex removes an item from a list config field by 0-based index.
func (s *GlobalStore) RemoveFromConfigListByIndex(key string, index int) (string, error) {
	return s.removeFromConfigList(key, func(entries []string) (int, error) {
		if index < 0 || index >= len(entries) {
			return -1, fmt.Errorf("index %d out of range (list has %d entries)", index, len(entries))
		}
		return index, nil
	})
}

// removeFromConfigList validates the field, loads the list, calls find to locate the
// target entry, removes it, and persists. find returns the index to remove or -1 for no-op.
func (s *GlobalStore) removeFromConfigList(key string, find func([]string) (int, error)) (string, error) {
	if err := s.validateListField(key); err != nil {
		return "", err
	}
	cfg, _, err := s.LoadConfig()
	if err != nil {
		return "", err
	}
	current := cfg.GetListValue(key)
	idx, err := find(current)
	if err != nil {
		return "", err
	}
	if idx < 0 {
		return "", nil
	}
	removed := current[idx]
	updated := slices.Delete(slices.Clone(current), idx, idx+1)
	return removed, s.SetConfigValue(key, updated)
}

// RegisterRepo adds a repository path to the tracked repos list.
// Resolves symlinks before storing to avoid duplicates.
func (s *GlobalStore) RegisterRepo(repoDir string) error {
	resolved, err := filepath.EvalSymlinks(repoDir)
	if err != nil {
		resolved = repoDir
	}
	return s.AddToConfigList("repos", resolved)
}

// WorktreeDir describes a worktree directory on disk.
type WorktreeDir struct {
	RepoName string
	Name     string
	Path     string
}

// ListWorktreeDirs walks the worktree base directory two levels deep,
// returning one entry per <repo>/<wt> directory found.
func (s *GlobalStore) ListWorktreeDirs() ([]WorktreeDir, error) {
	base := s.worktreeBasePath()

	repos, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var dirs []WorktreeDir
	for _, repo := range repos {
		if !repo.IsDir() {
			continue
		}
		wts, err := os.ReadDir(filepath.Join(base, repo.Name()))
		if err != nil {
			continue
		}
		for _, wt := range wts {
			if !wt.IsDir() {
				continue
			}
			dirs = append(dirs, WorktreeDir{
				RepoName: repo.Name(),
				Name:     wt.Name(),
				Path:     filepath.Join(base, repo.Name(), wt.Name()),
			})
		}
	}
	return dirs, nil
}

// SaveDefaultConfig writes config with comments derived from ConfigFields metadata.
func (s *GlobalStore) SaveDefaultConfig(cfg GlobalConfig) error {
	path := s.ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("# wt configuration\n")
	sb.WriteString("# See 'wt config show' for descriptions of each setting.\n\n")

	for i, field := range ConfigFields {
		sb.WriteString("# " + field.Desc + "\n")
		if field.ValidOpts != "" {
			sb.WriteString("# Valid: " + field.ValidOpts + "\n")
		}

		value := field.Get(cfg)
		if field.IsList {
			sb.WriteString(field.Key + ":\n")
			list, _ := value.([]string)
			for _, item := range list {
				sb.WriteString("    - " + item + "\n")
			}
		} else {
			_, _ = fmt.Fprintf(&sb, "%s: %s\n", field.Key, field.Format(value))
		}

		if i < len(ConfigFields)-1 {
			sb.WriteString("\n")
		}
	}

	return os.WriteFile(path, []byte(sb.String()), 0600)
}
