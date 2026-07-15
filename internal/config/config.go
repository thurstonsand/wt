// Package config provides configuration management for wt.
package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const configFile = "config.yaml"

// ProjectIncludeFile is the repo-root file naming files to copy into new
// worktrees, using .gitignore pattern syntax.
const ProjectIncludeFile = ".worktreeinclude"

// userIncludeFile is the user-level equivalent of ProjectIncludeFile, stored in
// WT_HOME (no leading dot; it lives in wt's own directory).
const userIncludeFile = "worktreeinclude"

// SanitizePathComponent converts a name to a safe filesystem path component.
// Replaces "/" with "--" to avoid nested directories.
func SanitizePathComponent(name string) string {
	return strings.ReplaceAll(name, "/", "--")
}

// UnsanitizePathComponent converts a sanitized path component back to original format.
// Reverses SanitizePathComponent: "fix--foo" -> "fix/foo"
func UnsanitizePathComponent(name string) string {
	return strings.ReplaceAll(name, "--", "/")
}

// ErrUnknownKey is returned when an unknown config key is requested.
var ErrUnknownKey = errors.New("unknown config key")

// MergeMode specifies how to merge worktree changes back to the parent branch.
type MergeMode string

const (
	// MergeModeSquash squashes all commits into one on the parent branch.
	MergeModeSquash MergeMode = "squash"

	// MergeModeRebase rebases commits onto the parent branch (preserves individual commits).
	MergeModeRebase MergeMode = "rebase"

	// MergeModeStaged applies changes as staged (no commit).
	MergeModeStaged MergeMode = "staged"
)

// ParseMergeMode converts a string to MergeMode, returning error for invalid values.
func ParseMergeMode(s string) (MergeMode, error) {
	switch s {
	case "squash":
		return MergeModeSquash, nil
	case "rebase":
		return MergeModeRebase, nil
	case "staged":
		return MergeModeStaged, nil
	case "":
		return MergeModeRebase, nil
	default:
		return "", fmt.Errorf("invalid merge mode %q (valid: squash, rebase, staged)", s)
	}
}

// UnmarshalYAML validates MergeMode when reading from config.
func (m *MergeMode) UnmarshalYAML(unmarshal func(any) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	parsed, err := ParseMergeMode(s)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

// GlobalConfig holds the global wt configuration.
type GlobalConfig struct {
	Clean      bool      `yaml:"clean"`
	Merge      MergeMode `yaml:"merge,omitempty"`
	Direnv     bool      `yaml:"direnv"`
	PostCreate []string  `yaml:"post_create,omitempty"`
	Repos      []string  `yaml:"repos,omitempty"`
}

// FieldMeta describes a config field for display and file generation.
type FieldMeta struct {
	Key       string
	Desc      string
	ValidOpts string
	IsList    bool
	Parse     func(string) (any, error)
	Get       func(GlobalConfig) any
	Format    func(any) string
}

// ConfigFields defines metadata for all config fields (single source of truth).
var ConfigFields = []FieldMeta{
	{
		Key:       "clean",
		Desc:      "When true, 'wt fork' creates clean worktrees without copying changes.",
		ValidOpts: "true, false",
		Parse:     parseBool,
		Get:       func(c GlobalConfig) any { return c.Clean },
		Format:    func(v any) string { return fmt.Sprintf("%v", v) },
	},
	{
		Key:       "merge",
		Desc:      "Default merge mode for 'wt merge'.",
		ValidOpts: "squash, rebase, staged",
		Parse:     func(s string) (any, error) { return ParseMergeMode(s) },
		Get:       func(c GlobalConfig) any { return c.Merge },
		Format:    func(v any) string { m, _ := v.(MergeMode); return string(m) },
	},
	{
		Key:       "direnv",
		Desc:      "When true, automatically run 'direnv allow' in new worktrees.",
		ValidOpts: "true, false",
		Parse:     parseBool,
		Get:       func(c GlobalConfig) any { return c.Direnv },
		Format:    func(v any) string { return fmt.Sprintf("%v", v) },
	},
	{
		Key:       "post_create",
		Desc:      "Shell commands to run in new worktrees after creation.",
		ValidOpts: "shell commands",
		IsList:    true,
		Parse:     parseStringList,
		Get:       func(c GlobalConfig) any { return c.PostCreate },
		Format:    formatStringList,
	},
	{
		Key:       "repos",
		Desc:      "Repositories tracked by wt (auto-populated).",
		ValidOpts: "filesystem paths",
		IsList:    true,
		Parse:     parseStringList,
		Get:       func(c GlobalConfig) any { return c.Repos },
		Format:    formatStringList,
	},
}

func parseBool(s string) (any, error) {
	switch strings.ToLower(s) {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off":
		return false, nil
	default:
		return nil, fmt.Errorf("not a boolean: %q", s)
	}
}

func parseStringList(s string) (any, error) {
	if s == "" {
		return []string{}, nil
	}
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts, nil
}

func formatStringList(v any) string {
	list, _ := v.([]string)
	if len(list) == 0 {
		return "(empty)"
	}
	return strings.Join(list, ", ")
}

// GetFieldMeta returns metadata for a config key, or nil if not found.
func GetFieldMeta(key string) *FieldMeta {
	for i := range ConfigFields {
		if ConfigFields[i].Key == key {
			return &ConfigFields[i]
		}
	}
	return nil
}

// FprintValue writes the field's value with description to the given writer.
func (f *FieldMeta) FprintValue(out io.Writer, cfg GlobalConfig) {
	_, _ = fmt.Fprintf(out, "%s:\t%s\n", f.Key, f.Format(f.Get(cfg)))
	_, _ = fmt.Fprintf(out, "\t%s\n", f.Desc)
	if f.ValidOpts != "" {
		_, _ = fmt.Fprintf(out, "\tValid: %s\n", f.ValidOpts)
	}
}

// UnknownKeyError returns an error wrapping ErrUnknownKey with context.
func UnknownKeyError(key string) error {
	return fmt.Errorf("%w: %q (valid: %s)", ErrUnknownKey, key, strings.Join(ValidKeys(), ", "))
}

// ValidKeys returns a slice of all valid config keys.
func ValidKeys() []string {
	keys := make([]string, len(ConfigFields))
	for i, f := range ConfigFields {
		keys[i] = f.Key
	}
	return keys
}

// ListKeys returns keys for fields that support add/remove operations.
func ListKeys() []string {
	var keys []string
	for _, f := range ConfigFields {
		if f.IsList {
			keys = append(keys, f.Key)
		}
	}
	return keys
}

// GetListValue returns the []string value for a list field.
func (c GlobalConfig) GetListValue(key string) []string {
	field := GetFieldMeta(key)
	if field == nil || !field.IsList {
		return nil
	}
	val, _ := field.Get(c).([]string)
	return val
}

// DefaultConfig returns a GlobalConfig with default values.
func DefaultConfig() GlobalConfig {
	return GlobalConfig{
		Clean: false,
		Merge: MergeModeRebase,
	}
}

// LoadConfig reads the global config from disk. Creates default if missing.
func (s *GlobalStore) LoadConfig() (cfg GlobalConfig, isNew bool, err error) {
	err = s.loadYAML(&cfg, configFile)
	if err != nil {
		if os.IsNotExist(err) {
			cfg = DefaultConfig()
			if err = s.SaveDefaultConfig(cfg); err != nil {
				return GlobalConfig{}, false, err
			}
			return cfg, true, nil
		}
		return GlobalConfig{}, false, err
	}
	return cfg, false, nil
}
