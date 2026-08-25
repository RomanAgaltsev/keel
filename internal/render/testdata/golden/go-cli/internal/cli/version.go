package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newVersionCmd reports the build provenance injected via -ldflags. A binary
// built without them reports "dev", which is the honest answer.
func newVersionCmd(b BuildInfo) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "demo %s (%s, %s)\n",
				b.Version, b.Commit, b.Date)
			return err
		},
	}
}
