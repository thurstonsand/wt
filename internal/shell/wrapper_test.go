package shell

import (
	"strings"
	"testing"
)

func TestZshWrapper(t *testing.T) {
	script := ZshWrapper()

	required := []string{
		"wt()",
		"case \"$1\" in",
		"cd)",
		"command wt path",
		"__complete)",
		"command wt \"$@\"",
		"3>&1 1>&4 2>&4",
	}

	for _, s := range required {
		if !strings.Contains(script, s) {
			t.Errorf("script missing %q", s)
		}
	}
}

func TestBashWrapper(t *testing.T) {
	script := BashWrapper()

	required := []string{
		"wt()",
		"case \"$1\" in",
		"cd)",
		"__complete)",
		"command wt path",
		"command wt \"$@\"",
		"3>&1 1>&4 2>&4",
	}

	for _, s := range required {
		if !strings.Contains(script, s) {
			t.Errorf("script missing %q", s)
		}
	}
}

func TestZshWrapperEnablesSubstringMatching(t *testing.T) {
	zsh := ZshWrapper()
	if !strings.Contains(zsh, "matcher 'l:|=* r:|=*'") {
		t.Error("zsh wrapper should enable substring completion matching")
	}

	if strings.Contains(BashWrapper(), "matcher") {
		t.Error("bash wrapper should not contain zsh matcher style (unsupported in bash)")
	}
}

func TestWrapperBodyShared(t *testing.T) {
	zsh := ZshWrapper()
	bash := BashWrapper()
	body := wrapperScript()

	if !strings.Contains(zsh, body) {
		t.Error("zsh wrapper should contain shared body")
	}
	if !strings.Contains(bash, body) {
		t.Error("bash wrapper should contain shared body")
	}
}
