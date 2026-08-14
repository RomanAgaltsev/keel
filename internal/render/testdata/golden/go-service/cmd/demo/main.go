// Command demo is the entrypoint for github.com/RomanAgaltsev/demo.
package main

import (
	"fmt"
	"os"
)

// Injected via -ldflags by Task and GoReleaser.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := run(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "demo:", err)
		os.Exit(1)
	}
}

func run(out *os.File) error {
	_, err := fmt.Fprintf(out, "demo %s (%s, %s)\n", version, commit, date)
	return err
}
