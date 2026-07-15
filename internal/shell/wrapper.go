// Package shell provides shell integration scripts for wt.
package shell

import (
	"syscall"
)

// cdFD is the file descriptor the shell wrapper opens as a private channel for
// the cd target. Keeping it off stdout leaves stdout/stderr attached to the
// terminal, so commands can stream output and run interactive pickers.
const cdFD = 3

// PrintWithCD writes dir to the wrapper's cd channel (fd 3). When wt runs
// without the wrapper that fd is closed and the write is silently dropped.
func PrintWithCD(dir string) {
	_, _ = syscall.Write(cdFD, []byte(dir))
}

func wrapperScript() string {
	return `
wt() {
  case "$1" in
    cd)
      local dir
      if dir="$(command wt path "${@:2}")"; then
        cd "$dir"
      else
        return 1
      fi
      ;;
    __complete)
      command wt "$@"
      ;;
    *)
      local dir exit_code
      { dir="$(command wt "$@" 3>&1 1>&4 2>&4)"; exit_code=$?; } 4>&1
      if [[ -n "$dir" && -d "$dir" ]]; then
        cd "$dir"
      fi
      return $exit_code
      ;;
  esac
}

`
}

// zshCompletionMatcher adds substring matching as a fallback for wt
// completions: normal prefix matching runs first, and only when it finds
// nothing does the labeled "wt-substring" pass match a fragment from the
// middle of a name (e.g. "dry" -> "DATAPLAT-32089/dry-run-on-push"). The
// matcher and menu styles attach to that pass alone, so prefix and flag
// completion keep their defaults. menu cycles full names rather than inserting
// the matches' common substring, which can begin with "-" and stall zsh into
// flag parsing.
const zshCompletionMatcher = "zstyle ':completion:*:*:wt:*' tag-order '*' '*:-wt-substring'\n" +
	"zstyle ':completion:*wt-substring*' matcher 'l:|=* r:|=*'\n" +
	"zstyle ':completion:*wt-substring*' menu yes select\n"

// ZshWrapper returns the zsh wrapper function for wt cd support.
func ZshWrapper() string {
	return "# wt shell integration for zsh\n# Add to ~/.zshrc: eval \"$(wt shell zsh)\"\n" + zshCompletionMatcher + wrapperScript()
}

// BashWrapper returns the bash wrapper function for wt cd support.
func BashWrapper() string {
	return "# wt shell integration for bash\n# Add to ~/.bashrc: eval \"$(wt shell bash)\"\n" + wrapperScript()
}
