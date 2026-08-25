// Package cli implements demo's command tree.
package cli

import (
	"os"

	"github.com/spf13/cobra"
)

// BuildInfo carries the values main injects via -ldflags.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// Execute runs the root command.
func Execute(b BuildInfo) error {
	return NewRootCmd(b).Execute()
}

// NewRootCmd builds the command tree. It returns a fresh command on every call
// so a test can run it repeatedly against its own output buffer.
func NewRootCmd(b BuildInfo) *cobra.Command {
	root := &cobra.Command{
		Use:   "demo",
		Short: "a demo CLI",
		// A usage dump on a runtime error buries the message; cobra still
		// prints usage for genuine flag and argument mistakes.
		SilenceUsage: true,
	}
	root.SetOut(os.Stdout)
	root.AddCommand(newVersionCmd(b))
	return root
}
