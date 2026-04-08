// Package config defines the CLI flag values and validation logic for ttl2d3.
package config

import (
	"errors"
	"fmt"
)

// OutputFormat represents the output format for the convert command.
type OutputFormat string

const (
	// OutputHTML produces a self-contained HTML file.
	OutputHTML OutputFormat = "html"
	// OutputJSON produces a standalone D3-compatible JSON file.
	OutputJSON OutputFormat = "json"
)

// InputFormat represents the input RDF serialisation format.
type InputFormat string

const (
	// InputAuto triggers automatic format detection from the file extension.
	InputAuto InputFormat = ""
	// InputTurtle selects the Turtle 1.1 parser.
	InputTurtle InputFormat = "turtle"
	// InputRDFXML selects the RDF/XML parser.
	InputRDFXML InputFormat = "rdfxml"
	// InputJSONLD selects the JSON-LD parser.
	InputJSONLD InputFormat = "jsonld"
)

// LayoutMode represents the HTML rendering layout mode.
type LayoutMode string

const (
	// LayoutForce is the existing D3 force-directed layout (default).
	LayoutForce LayoutMode = "force"
	// LayoutLayered is a deterministic layered layout for workflow/state-machine graphs.
	LayoutLayered LayoutMode = "layered"
	// LayoutSwimlane is a swimlane process layout with grouped lanes.
	LayoutSwimlane LayoutMode = "swimlane"
)

// LayoutDirection controls the primary flow direction for layered/swimlane layouts.
type LayoutDirection string

const (
	// LayoutDirectionLR flows left-to-right (default).
	LayoutDirectionLR LayoutDirection = "lr"
	// LayoutDirectionTB flows top-to-bottom.
	LayoutDirectionTB LayoutDirection = "tb"
)

// Config holds all resolved configuration values for a convert run.
type Config struct {
	// Input is the path to the input file, or "-" for stdin.
	Input string
	// Output is the desired output format (html or json).
	Output OutputFormat
	// Out is the destination file path, or "" for stdout.
	Out string
	// Format is the explicit input format override; empty means auto-detect.
	Format InputFormat
	// Title overrides the ontology IRI as the page title in HTML output.
	Title string
	// LinkDistance is the D3 force link distance parameter.
	LinkDistance float64
	// ChargeStrength is the D3 many-body charge strength parameter.
	ChargeStrength float64
	// CollideRadius is the D3 collision-detection radius parameter.
	CollideRadius float64
	// Verbose enables DEBUG-level structured logging.
	Verbose bool
	// Layout selects the HTML rendering layout mode (force, layered, swimlane).
	// Applies to HTML output only.
	Layout LayoutMode
	// LayoutDirection controls the primary flow axis for layered/swimlane layouts.
	LayoutDirection LayoutDirection
	// RankSeparation is the pixel gap between ranks in layered/swimlane layouts.
	RankSeparation float64
	// NodeSeparation is the pixel gap between nodes within the same rank.
	NodeSeparation float64
}

// DefaultConfig returns a Config populated with the default values from the spec.
func DefaultConfig() Config {
	return Config{
		Output:          OutputHTML,
		LinkDistance:    80,
		ChargeStrength:  -300,
		CollideRadius:   20,
		Layout:          LayoutForce,
		LayoutDirection: LayoutDirectionLR,
		RankSeparation:  180,
		NodeSeparation:  80,
	}
}

// Validate checks that all required fields are set and all values are legal.
// It returns a user-friendly error (exit code 1) on invalid input.
func (c *Config) Validate() error {
	if c.Input == "" {
		return errors.New("--input is required")
	}

	switch c.Output {
	case OutputHTML, OutputJSON:
		// valid
	default:
		return fmt.Errorf("invalid --output value %q: must be \"html\" or \"json\"", c.Output)
	}

	switch c.Format {
	case InputAuto, InputTurtle, InputRDFXML, InputJSONLD:
		// valid
	default:
		return fmt.Errorf("invalid --format value %q: must be \"turtle\", \"rdfxml\", or \"jsonld\"", c.Format)
	}

	if c.LinkDistance <= 0 {
		return fmt.Errorf("--link-distance must be positive, got %g", c.LinkDistance)
	}
	if c.CollideRadius <= 0 {
		return fmt.Errorf("--collide-radius must be positive, got %g", c.CollideRadius)
	}

	// Validate layout flags — layout applies only to HTML output.
	switch c.Layout {
	case LayoutForce, LayoutLayered, LayoutSwimlane:
		// valid
	default:
		return fmt.Errorf("invalid --layout value %q: must be \"force\", \"layered\", or \"swimlane\"", c.Layout)
	}

	if c.Layout != LayoutForce && c.Output == OutputJSON {
		return fmt.Errorf("--layout %q is only valid for HTML output; use --output html or omit --layout", c.Layout)
	}

	switch c.LayoutDirection {
	case LayoutDirectionLR, LayoutDirectionTB:
		// valid
	default:
		return fmt.Errorf("invalid --layout-direction value %q: must be \"lr\" or \"tb\"", c.LayoutDirection)
	}

	if c.RankSeparation <= 0 {
		return fmt.Errorf("--rank-separation must be positive, got %g", c.RankSeparation)
	}
	if c.NodeSeparation <= 0 {
		return fmt.Errorf("--node-separation must be positive, got %g", c.NodeSeparation)
	}

	return nil
}
