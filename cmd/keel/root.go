package main

import "github.com/spf13/cobra"

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "keel",
		Short:         "Scaffold a new git repository from composable template modules",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(
		newVersionCmd(),
		newListCmd(),
		newNewCmd(),
		newConfigCmd(),
		newOutdatedCmd(),
		newUpdateCmd(),
	)
	return root
}
