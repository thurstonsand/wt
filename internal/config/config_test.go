package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Clean {
		t.Error("Clean should default to false")
	}
	if cfg.Direnv {
		t.Error("Direnv should default to false")
	}
	if len(cfg.PostCreate) != 0 {
		t.Errorf("PostCreate should default to empty, got %d", len(cfg.PostCreate))
	}
}

func TestConfigLoadMissingCreatesDefault(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewGlobalStore(tmpDir)

	cfg, isNew, err := s.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if !isNew {
		t.Error("isNew should be true for missing file")
	}

	expected := DefaultConfig()
	if cfg.Clean != expected.Clean {
		t.Error("missing file should return default config")
	}

	path := s.pathOf(configFile)
	if _, err := os.Stat(path); err != nil {
		t.Error("config file should be created")
	}
}

func TestParseMergeMode(t *testing.T) {
	tests := []struct {
		input   string
		want    MergeMode
		wantErr bool
	}{
		{"squash", MergeModeSquash, false},
		{"rebase", MergeModeRebase, false},
		{"staged", MergeModeStaged, false},
		{"", MergeModeRebase, false},
		{"rebade", "", true},
		{"SQUASH", "", true},
		{"invalid", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := ParseMergeMode(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseMergeMode(%q) should error", tc.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseMergeMode(%q) unexpected error: %v", tc.input, err)
				return
			}
			if got != tc.want {
				t.Errorf("ParseMergeMode(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestMergeModeUnmarshalYAML(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewGlobalStore(tmpDir)

	validYAML := []byte("merge: squash\n")
	if err := os.WriteFile(s.pathOf(configFile), validYAML, 0600); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := s.LoadConfig()
	if err != nil {
		t.Fatalf("valid config should load: %v", err)
	}
	if cfg.Merge != MergeModeSquash {
		t.Errorf("Merge = %q, want %q", cfg.Merge, MergeModeSquash)
	}
}

func TestMergeModeUnmarshalYAMLInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewGlobalStore(tmpDir)

	invalidYAML := []byte("merge: rebade\n")
	if err := os.WriteFile(s.pathOf(configFile), invalidYAML, 0600); err != nil {
		t.Fatal(err)
	}

	_, _, err := s.LoadConfig()
	if err == nil {
		t.Fatal("invalid merge mode should cause load error")
	}
}

func TestUserIncludePath(t *testing.T) {
	s := NewGlobalStore(t.TempDir())
	expected := s.RootDir() + "/worktreeinclude"
	if got := s.UserIncludePath(); got != expected {
		t.Errorf("UserIncludePath = %s, want %s", got, expected)
	}
}

func TestWorktreePath(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewGlobalStore(tmpDir)

	path := s.WorktreePath("myrepo", "wt-test")
	expected := s.RootDir() + "/worktrees/myrepo/wt-test"
	if path != expected {
		t.Errorf("WorktreePath = %s, want %s", path, expected)
	}
}

func TestWorktreePathWithSlash(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewGlobalStore(tmpDir)

	path := s.WorktreePath("myrepo", "fix/buildkit-shell")
	expected := s.RootDir() + "/worktrees/myrepo/fix--buildkit-shell"
	if path != expected {
		t.Errorf("WorktreePath = %s, want %s", path, expected)
	}
}

func TestSanitizePathComponent(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"fix/foo", "fix--foo"},
		{"feature/auth/login", "feature--auth--login"},
		{"no-slash", "no-slash"},
	}
	for _, tc := range tests {
		got := SanitizePathComponent(tc.input)
		if got != tc.want {
			t.Errorf("SanitizePathComponent(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestUnsanitizePathComponent(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"fix--foo", "fix/foo"},
		{"feature--bar--baz", "feature/bar/baz"},
		{"simple-branch", "simple-branch"},
		{"no-slashes", "no-slashes"},
	}

	for _, tt := range tests {
		got := UnsanitizePathComponent(tt.input)
		if got != tt.want {
			t.Errorf("UnsanitizePathComponent(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestListWorktreeDirs(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewGlobalStore(tmpDir)

	base := filepath.Join(tmpDir, "worktrees")
	for _, p := range []string{
		filepath.Join("repo-a", "wt-1"),
		filepath.Join("repo-a", "wt-2"),
		filepath.Join("repo-b", "feature"),
	} {
		if err := os.MkdirAll(filepath.Join(base, p), 0750); err != nil {
			t.Fatal(err)
		}
	}
	// file in repo dir should be skipped
	if err := os.WriteFile(filepath.Join(base, "repo-a", "stale-file"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}

	dirs, err := s.ListWorktreeDirs()
	if err != nil {
		t.Fatalf("ListWorktreeDirs() error = %v", err)
	}
	if len(dirs) != 3 {
		t.Fatalf("expected 3 dirs, got %d", len(dirs))
	}

	want := map[string]string{
		"repo-a/wt-1":    "repo-a",
		"repo-a/wt-2":    "repo-a",
		"repo-b/feature": "repo-b",
	}
	for _, d := range dirs {
		key := d.RepoName + "/" + d.Name
		if wantRepo, ok := want[key]; !ok {
			t.Errorf("unexpected dir: %s", key)
		} else if d.RepoName != wantRepo {
			t.Errorf("RepoName = %q, want %q", d.RepoName, wantRepo)
		}
	}
}

func TestListWorktreeDirsMissing(t *testing.T) {
	s := NewGlobalStore(t.TempDir())

	dirs, err := s.ListWorktreeDirs()
	if err != nil {
		t.Fatalf("ListWorktreeDirs() error = %v", err)
	}
	if dirs != nil {
		t.Errorf("expected nil, got %v", dirs)
	}
}

func TestDirenvLoadsFromYAML(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewGlobalStore(tmpDir)

	yamlData := []byte("direnv: true\n")
	if err := os.MkdirAll(tmpDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.ConfigPath(), yamlData, 0600); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := s.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if !cfg.Direnv {
		t.Error("Direnv should be true")
	}
}

func TestPostCreateLoadsFromYAML(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewGlobalStore(tmpDir)

	yamlData := []byte("post_create:\n  - \"echo hello\"\n  - \"echo world\"\n")
	if err := os.MkdirAll(tmpDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.ConfigPath(), yamlData, 0600); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := s.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if len(cfg.PostCreate) != 2 {
		t.Fatalf("expected 2 post_create entries, got %d", len(cfg.PostCreate))
	}
	if cfg.PostCreate[0] != "echo hello" {
		t.Errorf("expected 'echo hello', got %q", cfg.PostCreate[0])
	}
}

func TestRemoveFromConfigListByIndex(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewGlobalStore(tmpDir)

	yamlData := []byte("post_create:\n  - \"echo hello\"\n  - \"echo world\"\n  - \"echo third\"\n")
	if err := os.WriteFile(s.ConfigPath(), yamlData, 0600); err != nil {
		t.Fatal(err)
	}

	removed, err := s.RemoveFromConfigListByIndex("post_create", 1)
	if err != nil {
		t.Fatalf("RemoveFromConfigListByIndex failed: %v", err)
	}
	if removed != "echo world" {
		t.Errorf("removed = %q, want %q", removed, "echo world")
	}

	cfg, _, err := s.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if len(cfg.PostCreate) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(cfg.PostCreate))
	}
	if cfg.PostCreate[0] != "echo hello" || cfg.PostCreate[1] != "echo third" {
		t.Errorf("unexpected entries: %v", cfg.PostCreate)
	}
}

func TestRemoveFromConfigListByIndexOutOfRange(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewGlobalStore(tmpDir)

	yamlData := []byte("post_create:\n  - \"echo hello\"\n")
	if err := os.WriteFile(s.ConfigPath(), yamlData, 0600); err != nil {
		t.Fatal(err)
	}

	_, err := s.RemoveFromConfigListByIndex("post_create", 5)
	if err == nil {
		t.Fatal("expected error for out-of-range index")
	}

	_, err = s.RemoveFromConfigListByIndex("post_create", -1)
	if err == nil {
		t.Fatal("expected error for negative index")
	}
}

func TestRegisterRepoDedup(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewGlobalStore(tmpDir)
	if err := s.SaveDefaultConfig(DefaultConfig()); err != nil {
		t.Fatal(err)
	}

	repoDir := filepath.Join(t.TempDir(), "myrepo")
	if err := os.MkdirAll(repoDir, 0750); err != nil {
		t.Fatal(err)
	}

	if err := s.RegisterRepo(repoDir); err != nil {
		t.Fatalf("first RegisterRepo: %v", err)
	}
	if err := s.RegisterRepo(repoDir); err != nil {
		t.Fatalf("second RegisterRepo: %v", err)
	}

	cfg, _, err := s.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Repos) != 1 {
		t.Errorf("expected 1 repo entry after dedup, got %d", len(cfg.Repos))
	}
}

func TestRegisterRepoResolvesSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewGlobalStore(tmpDir)
	if err := s.SaveDefaultConfig(DefaultConfig()); err != nil {
		t.Fatal(err)
	}

	realDir := filepath.Join(t.TempDir(), "real-repo")
	if err := os.MkdirAll(realDir, 0750); err != nil {
		t.Fatal(err)
	}
	linkDir := filepath.Join(t.TempDir(), "link-repo")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatal(err)
	}

	if err := s.RegisterRepo(linkDir); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := s.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}
	// Resolve expected path too (macOS /tmp → /private/tmp)
	resolvedReal, _ := filepath.EvalSymlinks(realDir)
	if cfg.Repos[0] != resolvedReal {
		t.Errorf("expected resolved path %q, got %q", resolvedReal, cfg.Repos[0])
	}
}
