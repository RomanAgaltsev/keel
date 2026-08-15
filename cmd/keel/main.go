// Command keel scaffolds a new git repository from composable template modules,
// driven by an interactive wizard or an answers file, and optionally creates and
// pushes the remote. See https://github.com/RomanAgaltsev/keel for usage.
package main

import (
	"errors"
	"fmt"
	"os"
)

// exitCodeError carries a specific process exit code out of a command's RunE.
// Commands that must distinguish outcomes (settings --check: 0 in sync, 1 drift,
// 2 error) return it instead of a plain error. A nil err means "this outcome is
// not a failure, just a non-zero exit" — drift, for instance, is already fully
// described by the report the command printed.
type exitCodeError struct {
	code int
	err  error
}

func (e exitCodeError) Error() string {
	if e.err == nil {
		return fmt.Sprintf("exit %d", e.code)
	}
	return e.err.Error()
}

func (e exitCodeError) Unwrap() error { return e.err }

// Injected via -ldflags by Task/GoReleaser.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		var ec exitCodeError
		if errors.As(err, &ec) {
			if ec.err != nil {
				fmt.Fprintln(os.Stderr, "error:", ec.err)
			}
			os.Exit(ec.code)
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
