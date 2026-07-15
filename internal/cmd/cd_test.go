package cmd

import (
	"bytes"
	"errors"
	"testing"
)

func TestCdRequiresShellIntegration(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"cd", "test-wt"})

	err := cmd.Execute()
	if !errors.Is(err, ErrShellIntegrationRequired) {
		t.Errorf("expected ErrShellIntegrationRequired, got %v", err)
	}

	output := out.String()
	if !bytes.Contains([]byte(output), []byte("Shell integration required")) {
		t.Errorf("expected shell integration message, got %q", output)
	}
	if !bytes.Contains([]byte(output), []byte("wt shell zsh")) {
		t.Errorf("expected zsh setup hint, got %q", output)
	}
}

func TestCdNoArgsRequiresShellIntegration(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"cd"})

	err := cmd.Execute()
	if !errors.Is(err, ErrShellIntegrationRequired) {
		t.Errorf("expected ErrShellIntegrationRequired, got %v", err)
	}
}
