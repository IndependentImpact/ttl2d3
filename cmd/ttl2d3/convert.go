package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/IndependentImpact/ttl2d3/internal/config"
)

// newConvertCmd returns the `convert` sub-command with all flags from spec §3.5.
func newConvertCmd() *cobra.Command {
	cfg := config.DefaultConfig()

	cmd := &cobra.Command{
		Use:   "convert",
		Short: "Convert an ontology file to a D3 graph (HTML or JSON)",
		Long: `Convert an RDF/OWL/Turtle/JSON-LD ontology or concept scheme into
an interactive D3.js force-directed graph visualisation.

The input format is auto-detected from the file extension unless overridden
with --format. The default output format is self-contained HTML; use
--output json for a standalone D3 JSON object.`,
		Example: `  # Produce a self-contained HTML visualisation
  ttl2d3 convert --input ontology.ttl --out diagram.html

  # Produce a D3 JSON file
  ttl2d3 convert --input ontology.ttl --output json --out graph.json

  # Read from stdin, write to stdout
  ttl2d3 convert --input - --format turtle`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cfg.Validate(); err != nil {
				// User error → exit code 1 (cobra prints the error automatically).
				return fmt.Errorf("%w", err)
			}

			slog.Debug("configuration validated",
				"input", cfg.Input,
				"output", cfg.Output,
				"out", cfg.Out,
				"format", cfg.Format,
				"linkDistance", cfg.LinkDistance,
				"chargeStrength", cfg.ChargeStrength,
				"collideRadius", cfg.CollideRadius,
			)

			// TODO(phase-10): wire up parser → transform → render pipeline.
			fmt.Fprintf(os.Stderr, "convert: pipeline not yet implemented (Phase 10)\n")
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&cfg.Input, "input", "i", "", "Input file path, or \"-\" for stdin (required)")
	f.StringVarP((*string)(&cfg.Output), "output", "o", string(config.OutputHTML), "Output format: html or json")
	f.StringVarP(&cfg.Out, "out", "O", "", "Output file path (default: stdout)")
	f.StringVarP((*string)(&cfg.Format), "format", "f", "", "Input format override: turtle, rdfxml, jsonld (default: auto-detect)")
	f.StringVar(&cfg.Title, "title", "", "Title shown in HTML output (default: ontology IRI)")
	f.Float64Var(&cfg.LinkDistance, "link-distance", cfg.LinkDistance, "D3 force link distance")
	f.Float64Var(&cfg.ChargeStrength, "charge-strength", cfg.ChargeStrength, "D3 many-body charge strength")
	f.Float64Var(&cfg.CollideRadius, "collide-radius", cfg.CollideRadius, "D3 collision-detection radius")

	return cmd
}
