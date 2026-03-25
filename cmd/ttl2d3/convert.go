package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/IndependentImpact/ttl2d3/internal/config"
	"github.com/IndependentImpact/ttl2d3/internal/parser"
	"github.com/IndependentImpact/ttl2d3/internal/render"
	"github.com/IndependentImpact/ttl2d3/internal/transform"
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

			return runConvert(cfg)
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

// runConvert executes the parser → transform → render pipeline for the given
// configuration.  It is factored out of the cobra RunE closure so that it can
// be called from tests directly.
func runConvert(cfg config.Config) (retErr error) {
	// ------------------------------------------------------------------
	// 1. Open input reader.
	// ------------------------------------------------------------------
	var (
		r        io.Reader
		filename string
	)

	if cfg.Input == "-" {
		r = os.Stdin
		filename = "-"
		slog.Debug("reading from stdin")
	} else {
		f, err := os.Open(cfg.Input) //nolint:gosec // path is user-supplied CLI argument
		if err != nil {
			return fmt.Errorf("convert: opening input file: %w", err)
		}
		defer f.Close() //nolint:errcheck // read-only; close errors are not actionable
		r = f
		filename = cfg.Input
		slog.Debug("opened input file", "path", cfg.Input)
	}

	// ------------------------------------------------------------------
	// 2. Parse.
	// ------------------------------------------------------------------
	g, err := parser.Parse(r, filename, filename, cfg.Format)
	if err != nil {
		return fmt.Errorf("convert: parsing input: %w", err)
	}
	slog.Debug("parsed triples", "count", len(g.Triples))

	// ------------------------------------------------------------------
	// 3. Transform.
	// ------------------------------------------------------------------
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		return fmt.Errorf("convert: building graph model: %w", err)
	}
	slog.Debug("built graph model", "nodes", len(gm.Nodes), "links", len(gm.Links))

	// Apply title override from --title flag.  Setting it on the model
	// propagates to both JSON (via Metadata) and HTML (via HTMLOptions fallback).
	if cfg.Title != "" {
		gm.Metadata.Title = cfg.Title
	}

	// ------------------------------------------------------------------
	// 4. Open output writer.
	// ------------------------------------------------------------------
	var w io.Writer

	if cfg.Out == "" {
		w = os.Stdout
		slog.Debug("writing to stdout")
	} else {
		outFile, err := os.Create(cfg.Out) //nolint:gosec // path is user-supplied CLI argument
		if err != nil {
			return fmt.Errorf("convert: creating output file: %w", err)
		}
		// Capture close errors for the output file: a failed close may indicate
		// that the kernel could not flush the write buffer to disk.
		defer func() {
			if cerr := outFile.Close(); cerr != nil && retErr == nil {
				retErr = fmt.Errorf("convert: closing output file: %w", cerr)
			}
		}()
		w = outFile
		slog.Debug("opened output file", "path", cfg.Out)
	}

	// ------------------------------------------------------------------
	// 5. Render.
	// ------------------------------------------------------------------
	switch cfg.Output {
	case config.OutputJSON:
		if err := render.RenderJSON(gm, w); err != nil {
			return fmt.Errorf("convert: rendering JSON: %w", err)
		}
	default: // config.OutputHTML
		opts := render.HTMLOptions{
			LinkDistance:   cfg.LinkDistance,
			ChargeStrength: cfg.ChargeStrength,
			CollideRadius:  cfg.CollideRadius,
		}
		if err := render.RenderHTML(gm, opts, w); err != nil {
			return fmt.Errorf("convert: rendering HTML: %w", err)
		}
	}

	slog.Debug("conversion complete")
	return nil
}
