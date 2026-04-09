package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/IndependentImpact/ttl2d3/internal/config"
	"github.com/IndependentImpact/ttl2d3/internal/fetcher"
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

The input may be a local file path, "-" for stdin, or an HTTP/HTTPS URL.
The input format is auto-detected from the file extension or HTTP Content-Type
header unless overridden with --format. The default output format is
self-contained HTML; use --output json for a standalone D3 JSON object.`,
		Example: `  # Produce a self-contained HTML visualisation from a local file
  ttl2d3 convert --input ontology.ttl --out diagram.html

  # Fetch an ontology from a URL and produce HTML
  ttl2d3 convert --input https://w3id.org/aiao --out aiao.html

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
				"workflowPlan", cfg.WorkflowPlan,
				"simplify", cfg.Simplify,
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
	f.BoolVar(&cfg.WorkflowPlan, "workflowplan", false, "Render indimp:WorkflowPlan resources as a directed process / swimlane diagram (HTML output only)")
	f.BoolVar(&cfg.Simplify, "simplify", false, "Render owl:unionOf as repeated direct edges instead of a triangle union node")

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

	if fetcher.IsURL(cfg.Input) {
		rc, detectedFormat, err := fetcher.Fetch(context.Background(), cfg.Input, cfg.Format)
		if err != nil {
			return fmt.Errorf("convert: fetching URL: %w", err)
		}
		defer rc.Close() //nolint:errcheck // read-only; close errors are not actionable
		r = rc
		filename = cfg.Input
		// Only update format when auto-detect is still active so that an
		// explicit --format flag is never overridden.
		if cfg.Format == config.InputAuto {
			cfg.Format = detectedFormat
		}
		slog.Debug("fetched URL", "url", cfg.Input, "detectedFormat", detectedFormat)
	} else if cfg.Input == "-" {
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
	gm, err := transform.BuildGraphModel(g, transform.Options{Simplify: cfg.Simplify})
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
		if cfg.WorkflowPlan {
			// Build the specialised WorkflowModel from the parsed RDF graph and
			// render it as a directed process / swimlane diagram.
			wm, err := transform.BuildWorkflowModel(g)
			if err != nil {
				return fmt.Errorf("convert: building workflow model: %w", err)
			}
			title := cfg.Title
			if title == "" {
				title = gm.Metadata.Title
			}
			if title == "" {
				title = gm.Metadata.BaseIRI
			}
			if err := render.RenderWorkflowPlan(wm, title, w); err != nil {
				return fmt.Errorf("convert: rendering workflow plan: %w", err)
			}
		} else {
			opts := render.HTMLOptions{
				LinkDistance:   cfg.LinkDistance,
				ChargeStrength: cfg.ChargeStrength,
				CollideRadius:  cfg.CollideRadius,
			}
			if err := render.RenderHTML(gm, opts, w); err != nil {
				return fmt.Errorf("convert: rendering HTML: %w", err)
			}
		}
	}

	slog.Debug("conversion complete")
	return nil
}
