// ttl2d3 converts semantic-web ontologies and concept schemes into interactive
// D3.js force-directed graph visualisations.
//
// Usage:
//
//	ttl2d3 [flags]
//	ttl2d3 convert [flags]
//	ttl2d3 version
package main

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

// Version information injected at build time via -ldflags.
var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

// newRootCmd builds and returns the root cobra command.
func newRootCmd() *cobra.Command {
	var verbose bool

	root := &cobra.Command{
		Use:   "ttl2d3",
		Short: "Convert RDF/OWL/Turtle ontologies to D3.js force-directed graphs",
		Long: `ttl2d3 converts semantic-web ontologies and concept schemes
(.ttl, .owl, .jsonld, .rdf) into interactive D3.js force-directed graph
visualisations.

Output can be either a standalone D3 JSON object or a fully self-contained
HTML page (similar to WebVOWL) that requires no server to view.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			level := slog.LevelInfo
			if verbose {
				level = slog.LevelDebug
			}
			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: level,
			}))
			slog.SetDefault(logger)
		},
		// Running the root command without a sub-command is an error.
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable debug logging")

	root.AddCommand(newVersionCmd())
	root.AddCommand(newConvertCmd())

	return root
}
