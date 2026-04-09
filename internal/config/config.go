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
	// WorkflowPlan enables the custom WorkflowPlan visualiser.  When set, the
	// input is expected to contain resources of type indimp:WorkflowPlan and
	// the output is a directed process / swimlane diagram rather than the
	// default force-directed network graph.  Applies to HTML output only.
	WorkflowPlan bool
	// NodeSpacing is the column width in pixels used in the --workflowplan
	// swimlane table.  Increase this value to prevent step labels from
	// overprinting one another in dense workflow diagrams.  Default: 180.
	NodeSpacing float64
	// Simplify enables simplified union rendering.  When true, owl:unionOf
	// class expressions are not represented as explicit triangle union nodes;
	// instead the originating object-property edge is repeated once for each
	// member of the union, pointing directly from the domain (or range) class
	// to each union-member class.  This produces a simpler graph that is
	// easier to read as a map of possibilities.
	Simplify bool
}

// DefaultConfig returns a Config populated with the default values from the spec.
func DefaultConfig() Config {
	return Config{
		Output:         OutputHTML,
		LinkDistance:   80,
		ChargeStrength: -300,
		CollideRadius:  20,
		NodeSpacing:    180,
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

	if c.NodeSpacing <= 0 {
		return fmt.Errorf("--node-spacing must be positive, got %g", c.NodeSpacing)
	}

	if c.WorkflowPlan && c.Output == OutputJSON {
		return fmt.Errorf("--workflowplan is only valid for HTML output; use --output html or omit --workflowplan")
	}

	return nil
}
