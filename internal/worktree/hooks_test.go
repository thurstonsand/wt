package worktree

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thurstonsand/wt/internal/config"
)

type call struct {
	Dir  string
	Name string
	Args []string
}

type mockRunner struct {
	calls []call
	err   error
}

func (r *mockRunner) Run(dir, name string, args ...string) error {
	r.calls = append(r.calls, call{Dir: dir, Name: name, Args: args})
	return r.err
}

func TestDirenvHookRunsWhenEnabledAndEnvrcPresent(t *testing.T) {
	runner := &mockRunner{}
	m := &Manager{cmdRunner: runner}

	wtDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(wtDir, ".envrc"), []byte("use nix"), 0600); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	cfg := config.GlobalConfig{Direnv: true}
	m.RunPostCreateHooks(cfg, wtDir, &buf)

	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(runner.calls))
	}
	c := runner.calls[0]
	if c.Name != "direnv" || len(c.Args) != 1 || c.Args[0] != "allow" {
		t.Errorf("expected direnv allow, got %s %v", c.Name, c.Args)
	}
	if c.Dir != wtDir {
		t.Errorf("expected dir %s, got %s", wtDir, c.Dir)
	}
}

func TestDirenvHookSkipsWhenNoEnvrc(t *testing.T) {
	runner := &mockRunner{}
	m := &Manager{cmdRunner: runner}

	wtDir := t.TempDir()
	var buf bytes.Buffer
	cfg := config.GlobalConfig{Direnv: true}
	m.RunPostCreateHooks(cfg, wtDir, &buf)

	if len(runner.calls) != 0 {
		t.Errorf("expected 0 calls, got %d", len(runner.calls))
	}
}

func TestDirenvHookSkipsWhenDisabled(t *testing.T) {
	runner := &mockRunner{}
	m := &Manager{cmdRunner: runner}

	wtDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(wtDir, ".envrc"), []byte("use nix"), 0600); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	cfg := config.GlobalConfig{Direnv: false}
	m.RunPostCreateHooks(cfg, wtDir, &buf)

	if len(runner.calls) != 0 {
		t.Errorf("expected 0 calls, got %d", len(runner.calls))
	}
}

func TestDirenvHookWarnsOnError(t *testing.T) {
	runner := &mockRunner{err: errors.New("not found")}
	m := &Manager{cmdRunner: runner}

	wtDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(wtDir, ".envrc"), []byte("use nix"), 0600); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	cfg := config.GlobalConfig{Direnv: true}
	m.RunPostCreateHooks(cfg, wtDir, &buf)

	if !strings.Contains(buf.String(), "warning: direnv allow failed") {
		t.Errorf("expected warning, got %q", buf.String())
	}
}

func TestPostCreateCommandsRun(t *testing.T) {
	runner := &mockRunner{}
	m := &Manager{cmdRunner: runner}

	wtDir := t.TempDir()
	var buf bytes.Buffer
	cfg := config.GlobalConfig{
		PostCreate: []string{"echo hello", "echo world"},
	}
	m.RunPostCreateHooks(cfg, wtDir, &buf)

	if len(runner.calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(runner.calls))
	}
	for i, expected := range []string{"echo hello", "echo world"} {
		c := runner.calls[i]
		if c.Name != "sh" || len(c.Args) != 2 || c.Args[0] != "-c" || c.Args[1] != expected {
			t.Errorf("call %d: expected sh -c %q, got %s %v", i, expected, c.Name, c.Args)
		}
	}
}

func TestPostCreateCommandWarnsOnError(t *testing.T) {
	runner := &mockRunner{err: errors.New("exit 1")}
	m := &Manager{cmdRunner: runner}

	wtDir := t.TempDir()
	var buf bytes.Buffer
	cfg := config.GlobalConfig{PostCreate: []string{"false"}}
	m.RunPostCreateHooks(cfg, wtDir, &buf)

	if !strings.Contains(buf.String(), "warning: post_create hook failed") {
		t.Errorf("expected warning, got %q", buf.String())
	}
}

func TestMixedHooksRunInOrder(t *testing.T) {
	runner := &mockRunner{}
	m := &Manager{cmdRunner: runner}

	wtDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(wtDir, ".envrc"), []byte("use nix"), 0600); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	cfg := config.GlobalConfig{
		Direnv:     true,
		PostCreate: []string{"echo post"},
	}
	m.RunPostCreateHooks(cfg, wtDir, &buf)

	if len(runner.calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(runner.calls))
	}
	if runner.calls[0].Name != "direnv" {
		t.Error("direnv should run first")
	}
	if runner.calls[1].Name != "sh" {
		t.Error("post_create should run second")
	}
}

func TestNoHooksWhenEmpty(t *testing.T) {
	runner := &mockRunner{}
	m := &Manager{cmdRunner: runner}

	var buf bytes.Buffer
	cfg := config.GlobalConfig{}
	m.RunPostCreateHooks(cfg, t.TempDir(), &buf)

	if len(runner.calls) != 0 {
		t.Errorf("expected 0 calls, got %d", len(runner.calls))
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output, got %q", buf.String())
	}
}
