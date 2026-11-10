package main

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// newVersionCmd returns the `version` sub-command.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version, commit SHA, and build date",
		Long:  `Print the ttl2d3 version string, Git commit SHA, and build date.`,
		Run: func(cmd *cobra.Command, args []string) {
			info, _ := debug.ReadBuildInfo()
			v, c, d := resolveVersion(version, commit, buildDate, info)
			fmt.Fprintf(cmd.OutOrStdout(), "ttl2d3 %s (commit %s, built %s)\n", v, c, d)
		},
	}
}

// resolveVersion fills in values the release build did not inject via
// -ldflags, using the Go build info: the module version for
// `go install …@vX`, and VCS stamping for local builds.
func resolveVersion(ver, com, date string, info *debug.BuildInfo) (string, string, string) {
	if info == nil {
		return ver, com, date
	}
	if ver == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		ver = info.Main.Version
	}
	for _, s := range info.Settings {
		switch {
		case s.Key == "vcs.revision" && com == "none":
			com = s.Value
		case s.Key == "vcs.time" && date == "unknown":
			date = s.Value
		}
	}
	return ver, com, date
}
