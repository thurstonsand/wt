package cmd

import (
	"bytes"
	"runtime/debug"
	"strings"
	"testing"
)

func TestRootCommand(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--version"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "wt version") {
		t.Errorf("expected version output, got: %s", out)
	}
}

func TestResolveVersion(t *testing.T) {
	buildInfo := &debug.BuildInfo{Main: debug.Module{Version: "v1.2.3"}}

	tests := []struct {
		name          string
		linkerVersion string
		buildInfo     *debug.BuildInfo
		hasBuildInfo  bool
		want          string
	}{
		{name: "linker version wins", linkerVersion: "v2.0.0", buildInfo: buildInfo, hasBuildInfo: true, want: "v2.0.0"},
		{name: "module version fallback", linkerVersion: "dev", buildInfo: buildInfo, hasBuildInfo: true, want: "v1.2.3"},
		{name: "development build", linkerVersion: "dev", buildInfo: &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, hasBuildInfo: true, want: "dev"},
		{name: "missing build info", linkerVersion: "dev", want: "dev"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveVersion(tt.linkerVersion, tt.buildInfo, tt.hasBuildInfo); got != tt.want {
				t.Fatalf("resolveVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRootCommandHelp(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Git worktree helper") {
		t.Errorf("expected help output, got: %s", out)
	}
}
