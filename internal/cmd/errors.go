package cmd

import "errors"

// ErrInvalidFlagCombination indicates mutually exclusive flags were used together.
var ErrInvalidFlagCombination = errors.New("invalid flag combination")

// ErrShellIntegrationRequired indicates a command requires shell integration.
var ErrShellIntegrationRequired = errors.New("shell integration required")

var errAborted = errors.New("aborted")
