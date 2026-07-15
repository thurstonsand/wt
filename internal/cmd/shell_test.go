package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestShellCmd_Zsh(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"shell", "zsh"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := out.String()

	required := []string{
		"wt()",
		"cd",
		"command wt",
		"compdef",
		"_wt",
	}
	for _, s := range required {
		if !strings.Contains(output, s) {
			t.Errorf("output missing %q", s)
		}
	}
}

func TestShellCmd_Bash(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"shell", "bash"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := out.String()

	required := []string{
		"wt()",
		"cd",
		"command wt",
		"_wt",
		"complete",
	}
	for _, s := range required {
		if !strings.Contains(output, s) {
			t.Errorf("output missing %q", s)
		}
	}
}

func TestShellCmd_NoArg(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"shell"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing shell type")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Errorf("expected arg count error, got: %v", err)
	}
}

func TestShellCmd_UnsupportedShell(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"shell", "fish"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for unsupported shell")
	}
	if !strings.Contains(err.Error(), "unsupported shell") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestShellCmd_ExtraArg(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"shell", "zsh", "extra"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for extra argument")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Errorf("expected arg count error, got: %v", err)
	}
}
