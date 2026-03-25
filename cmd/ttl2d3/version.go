package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newVersionCmd returns the `version` sub-command.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version, commit SHA, and build date",
		Long:  `Print the ttl2d3 version string, Git commit SHA, and build date.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "ttl2d3 %s (commit %s, built %s)\n", version, commit, buildDate)
		},
	}
}
