// Command demo is the entrypoint for github.com/RomanAgaltsev/demo.
package main

import (
	"os"

	"github.com/RomanAgaltsev/demo/internal/cli"
)

// Injected via -ldflags by Task and GoReleaser.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// cobra has already printed the error; main only decides the exit code.
	if err := cli.Execute(cli.BuildInfo{Version: version, Commit: commit, Date: date}); err != nil {
		os.Exit(1)
	}
}
