// Command keel scaffolds a new git repository from composable template modules,
// driven by an interactive wizard or an answers file, and optionally creates and
// pushes the remote. See https://github.com/RomanAgaltsev/keel for usage.
package main

import (
	"fmt"
	"os"
)

// Injected via -ldflags by Task/GoReleaser.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
