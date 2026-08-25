// Command demo runs the demo analyzer.
//
// singlechecker provides its own flag handling, including -V, so this
// entrypoint takes no ldflags-injected version variables.
package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/RomanAgaltsev/demo/analyzer/demo"
)

func main() {
	singlechecker.Main(demo.Analyzer)
}
