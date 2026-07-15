// Package main is the entry point for the wt CLI.
package main

import (
	"fmt"
	"os"

	"github.com/thurstonsand/wt/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "wt: %s\n", err)
		os.Exit(1)
	}
}
