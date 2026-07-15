package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupConfigTest(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("WT_HOME", tmp)
	return tmp
}

func runConfigCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func mustRunConfigCmd(t *testing.T, args ...string) string {
	t.Helper()
	output, err := runConfigCmd(t, args...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return output
}

func TestConfigShow(t *testing.T) {
	setupConfigTest(t)

	output := mustRunConfigCmd(t, "config", "show")

	required := []string{
		"clean: false",
		"merge: rebase",
		"post_create:",
		"Created default config",
	}
	for _, s := range required {
		if !strings.Contains(output, s) {
			t.Errorf("output missing %q", s)
		}
	}
}

func TestConfigShowExisting(t *testing.T) {
	tmp := setupConfigTest(t)

	if err := os.WriteFile(filepath.Join(tmp, "config.yaml"), []byte("clean: true\nmerge: squash\n"), 0600); err != nil {
		t.Fatal(err)
	}

	output := mustRunConfigCmd(t, "config", "show")

	if strings.Contains(output, "Created default config") {
		t.Error("should not show 'Created' message for existing config")
	}
	if !strings.Contains(output, "clean: true") {
		t.Error("should show clean: true")
	}
	if !strings.Contains(output, "merge: squash") {
		t.Error("should show merge: squash")
	}
}

func TestConfigGet(t *testing.T) {
	setupConfigTest(t)

	tests := []struct {
		key      string
		contains []string
	}{
		{"clean", []string{"clean: false", "true, false"}},
		{"merge", []string{"merge: rebase", "squash, rebase, staged"}},
		{"post_create", []string{"post_create:"}},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			output := mustRunConfigCmd(t, "config", "get", tt.key)
			for _, s := range tt.contains {
				if !strings.Contains(output, s) {
					t.Errorf("output missing %q", s)
				}
			}
		})
	}
}

func TestConfigGetUnknownKey(t *testing.T) {
	setupConfigTest(t)

	_, err := runConfigCmd(t, "config", "get", "unknown")
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
	if !strings.Contains(err.Error(), "unknown config key") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfigSetClean(t *testing.T) {
	setupConfigTest(t)

	mustRunConfigCmd(t, "config", "set", "clean", "true")
	output := mustRunConfigCmd(t, "config", "get", "clean")

	if !strings.Contains(output, "clean: true") {
		t.Error("expected clean to be true after set")
	}
}

func TestConfigSetMerge(t *testing.T) {
	setupConfigTest(t)

	for _, mode := range []string{"squash", "rebase", "staged"} {
		t.Run(mode, func(t *testing.T) {
			mustRunConfigCmd(t, "config", "set", "merge", mode)
			output := mustRunConfigCmd(t, "config", "get", "merge")

			if !strings.Contains(output, "merge: "+mode) {
				t.Errorf("expected merge to be %s", mode)
			}
		})
	}
}

func TestConfigSetMergeInvalid(t *testing.T) {
	setupConfigTest(t)

	_, err := runConfigCmd(t, "config", "set", "merge", "invalid")
	if err == nil {
		t.Fatal("expected error for invalid merge mode")
	}
	if !strings.Contains(err.Error(), "invalid merge mode") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfigSetPostCreate(t *testing.T) {
	setupConfigTest(t)

	mustRunConfigCmd(t, "config", "set", "post_create", "echo one, echo two")
	output := mustRunConfigCmd(t, "config", "get", "post_create")

	if !strings.Contains(output, "echo one") {
		t.Error("expected post_create to contain 'echo one'")
	}
	if !strings.Contains(output, "echo two") {
		t.Error("expected post_create to contain 'echo two'")
	}
}

func TestConfigSetEmptyPostCreate(t *testing.T) {
	setupConfigTest(t)

	mustRunConfigCmd(t, "config", "set", "post_create", "")
	output := mustRunConfigCmd(t, "config", "get", "post_create")

	if !strings.Contains(output, "(empty)") {
		t.Error("expected empty post_create")
	}
}

func TestConfigSetUnknownKey(t *testing.T) {
	setupConfigTest(t)

	_, err := runConfigCmd(t, "config", "set", "unknown", "value")
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
	if !strings.Contains(err.Error(), "unknown config key") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfigSetPreservesComments(t *testing.T) {
	tmp := setupConfigTest(t)

	mustRunConfigCmd(t, "config", "show")

	data, _ := os.ReadFile(filepath.Join(tmp, "config.yaml")) //nolint:gosec // test file
	if !strings.Contains(string(data), "# wt configuration") {
		t.Error("default config should have comments")
	}

	mustRunConfigCmd(t, "config", "set", "clean", "true")

	data2, _ := os.ReadFile(filepath.Join(tmp, "config.yaml")) //nolint:gosec // test file
	if !strings.Contains(string(data2), "# wt configuration") {
		t.Error("comments should be preserved after set")
	}
	if !strings.Contains(string(data2), "clean: true") {
		t.Error("value should be updated")
	}
}

func TestConfigSetCleanInvalid(t *testing.T) {
	setupConfigTest(t)

	_, err := runConfigCmd(t, "config", "set", "clean", "maybe")
	if err == nil {
		t.Fatal("expected error for invalid boolean")
	}
	if !strings.Contains(err.Error(), "not a boolean") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfigNoArgs(t *testing.T) {
	output := mustRunConfigCmd(t, "config")

	if !strings.Contains(output, "show") {
		t.Error("expected help output with subcommands")
	}
}

func TestConfigAdd(t *testing.T) {
	setupConfigTest(t)

	output := mustRunConfigCmd(t, "config", "add", "post_create", "echo hi")

	if !strings.Contains(output, `Added "echo hi" to post_create`) {
		t.Errorf("unexpected output: %s", output)
	}

	getOutput := mustRunConfigCmd(t, "config", "get", "post_create")
	if !strings.Contains(getOutput, "echo hi") {
		t.Error("expected post_create to contain 'echo hi'")
	}
}

func TestConfigAddDuplicate(t *testing.T) {
	setupConfigTest(t)

	mustRunConfigCmd(t, "config", "add", "post_create", "echo hi")
	mustRunConfigCmd(t, "config", "add", "post_create", "echo hi")
}

func TestConfigAddNonList(t *testing.T) {
	setupConfigTest(t)

	_, err := runConfigCmd(t, "config", "add", "clean", "true")
	if err == nil {
		t.Fatal("expected error for non-list key")
	}
	if !strings.Contains(err.Error(), "not a list") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfigRemove(t *testing.T) {
	setupConfigTest(t)

	mustRunConfigCmd(t, "config", "add", "post_create", "echo one")
	mustRunConfigCmd(t, "config", "add", "post_create", "echo two")

	output := mustRunConfigCmd(t, "config", "remove", "post_create", "echo one")

	if !strings.Contains(output, `Removed "echo one" from post_create`) {
		t.Errorf("unexpected output: %s", output)
	}

	getOutput := mustRunConfigCmd(t, "config", "get", "post_create")
	if strings.Contains(getOutput, "echo one") {
		t.Error("expected 'echo one' to be removed")
	}
	if !strings.Contains(getOutput, "echo two") {
		t.Error("expected other items to remain")
	}
}

func TestConfigRemoveNonList(t *testing.T) {
	setupConfigTest(t)

	_, err := runConfigCmd(t, "config", "remove", "merge", "squash")
	if err == nil {
		t.Fatal("expected error for non-list key")
	}
	if !strings.Contains(err.Error(), "not a list") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfigRemoveByIndex(t *testing.T) {
	setupConfigTest(t)

	mustRunConfigCmd(t, "config", "add", "post_create", "echo one")
	mustRunConfigCmd(t, "config", "add", "post_create", "echo two")

	output := mustRunConfigCmd(t, "config", "remove", "post_create", "0")
	if !strings.Contains(output, "Removed [0]") {
		t.Errorf("unexpected output: %s", output)
	}

	getOutput := mustRunConfigCmd(t, "config", "get", "post_create")
	if strings.Contains(getOutput, "echo one") {
		t.Error("expected 'echo one' to be removed by index")
	}
	if !strings.Contains(getOutput, "echo two") {
		t.Error("expected other items to remain")
	}
}

func TestConfigRemoveShowEntries(t *testing.T) {
	setupConfigTest(t)

	mustRunConfigCmd(t, "config", "add", "post_create", "echo one")

	output := mustRunConfigCmd(t, "config", "remove", "post_create")
	if !strings.Contains(output, "[0]") {
		t.Error("expected indexed listing")
	}
	if !strings.Contains(output, "echo one") {
		t.Error("expected entries to be shown")
	}
	if !strings.Contains(output, "wt config remove post_create") {
		t.Error("expected usage hint")
	}
}

func TestConfigRemoveByIndexOutOfRange(t *testing.T) {
	setupConfigTest(t)

	mustRunConfigCmd(t, "config", "add", "post_create", "echo one")

	_, err := runConfigCmd(t, "config", "remove", "post_create", "99")
	if err == nil {
		t.Fatal("expected error for out-of-range index")
	}
	if !strings.Contains(err.Error(), "out of range") {
		t.Errorf("unexpected error: %v", err)
	}
}
