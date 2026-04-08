package render

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"io"

	"github.com/IndependentImpact/ttl2d3/internal/graph"
)

//go:embed templates/graph.html
var rawHTMLTemplate string

//go:embed templates/graph_layered.html
var rawHTMLLayeredTemplate string

//go:embed templates/graph_swimlane.html
var rawHTMLSwimlaneTemplate string

// htmlTmpl is the compiled HTML template for force layout, parsed once at package initialisation.
var htmlTmpl = template.Must(template.New("graph").Parse(rawHTMLTemplate))

// htmlLayeredTmpl is the compiled HTML template for layered layout.
var htmlLayeredTmpl = template.Must(template.New("graph_layered").Parse(rawHTMLLayeredTemplate))

// htmlSwimlaneTmpl is the compiled HTML template for swimlane layout.
var htmlSwimlaneTmpl = template.Must(template.New("graph_swimlane").Parse(rawHTMLSwimlaneTemplate))

// LayoutMode selects the HTML rendering layout.
type LayoutMode string

const (
	// LayoutForce is the existing D3 force-directed layout (default).
	LayoutForce LayoutMode = "force"
	// LayoutLayered is a deterministic layered layout for workflow graphs.
	LayoutLayered LayoutMode = "layered"
	// LayoutSwimlane is a swimlane process layout.
	LayoutSwimlane LayoutMode = "swimlane"
)

// LayoutDirection controls the primary flow axis for layered/swimlane layouts.
type LayoutDirection string

const (
	// LayoutDirectionLR flows left-to-right (default).
	LayoutDirectionLR LayoutDirection = "lr"
	// LayoutDirectionTB flows top-to-bottom.
	LayoutDirectionTB LayoutDirection = "tb"
)

// HTMLOptions configures the HTML renderer.
type HTMLOptions struct {
	// Title is the page title shown in the browser tab and page header.
	// Falls back to ontology metadata title, then base IRI, then "ttl2d3 Graph".
	Title string
	// LinkDistance is the D3 force link-distance parameter (default 80).
	LinkDistance float64
	// ChargeStrength is the D3 many-body charge-strength parameter (default -300).
	ChargeStrength float64
	// CollideRadius is the D3 collision-detection radius (default 20).
	CollideRadius float64
	// Layout selects the rendering mode (force, layered, swimlane). Default: force.
	Layout LayoutMode
	// LayoutDirection controls the primary flow axis for layered/swimlane layouts.
	LayoutDirection LayoutDirection
	// RankSeparation is the pixel gap between ranks in layered/swimlane layouts.
	RankSeparation float64
	// NodeSeparation is the pixel gap between nodes within a rank.
	NodeSeparation float64
}

// DefaultHTMLOptions returns HTMLOptions populated with the default values from
// the spec (§3.5).
func DefaultHTMLOptions() HTMLOptions {
	return HTMLOptions{
		LinkDistance:    80,
		ChargeStrength:  -300,
		CollideRadius:   20,
		Layout:          LayoutForce,
		LayoutDirection: LayoutDirectionLR,
		RankSeparation:  180,
		NodeSeparation:  80,
	}
}

// templateData is the value passed to graph.html during template execution.
type templateData struct {
	Title           string
	GraphJSON       template.JS
	LinkDistance    float64
	ChargeStrength  float64
	CollideRadius   float64
	LayoutDirection string
	RankSeparation  float64
	NodeSeparation  float64
}

// RenderHTML serialises gm as a self-contained HTML page and writes it to w.
//
// The output satisfies requirements OH-01–OH-12 from spec.md §3.4:
//   - Single file with all CSS and JS inlined (OH-01)
//   - D3 v7 loaded from cdn.jsdelivr.net (OH-02)
//   - Graph JSON embedded in a <script> block (OH-03)
//   - Interactive graph with zoom, pan, and drag (OH-04)
//   - Node colour + shape encode entity type (OH-05)
//   - Hover tooltip with IRI, label, and type (OH-06)
//   - Visible legend (OH-07)
//   - Responsive SVG (OH-08)
//   - Search/filter input box (OH-09)
//   - Deterministic output for non-force layouts (OH-11)
//   - Back-edges visually distinct in non-force layouts (OH-12)
//
// If opts.LinkDistance, opts.ChargeStrength, or opts.CollideRadius are zero the
// values from DefaultHTMLOptions are used.
func RenderHTML(gm *graph.GraphModel, opts HTMLOptions, w io.Writer) error {
	if gm == nil {
		return errors.New("render: GraphModel is nil")
	}

	// Apply defaults for zero-value fields.
	defaults := DefaultHTMLOptions()
	if opts.LinkDistance == 0 {
		opts.LinkDistance = defaults.LinkDistance
	}
	if opts.ChargeStrength == 0 {
		opts.ChargeStrength = defaults.ChargeStrength
	}
	if opts.CollideRadius == 0 {
		opts.CollideRadius = defaults.CollideRadius
	}
	if opts.Layout == "" {
		opts.Layout = defaults.Layout
	}
	if opts.LayoutDirection == "" {
		opts.LayoutDirection = defaults.LayoutDirection
	}
	if opts.RankSeparation == 0 {
		opts.RankSeparation = defaults.RankSeparation
	}
	if opts.NodeSeparation == 0 {
		opts.NodeSeparation = defaults.NodeSeparation
	}

	// Resolve page title.
	title := opts.Title
	if title == "" {
		title = gm.Metadata.Title
	}
	if title == "" {
		title = gm.Metadata.BaseIRI
	}
	if title == "" {
		title = "ttl2d3 Graph"
	}

	// Serialise the graph to JSON for inline embedding.
	// encoding/json HTML-escapes < > & in string values, making the embedded
	// JSON safe inside a <script> block without additional sanitisation.
	var jsonBuf bytes.Buffer
	if err := RenderJSON(gm, &jsonBuf); err != nil {
		return fmt.Errorf("render: serialising graph to JSON: %w", err)
	}

	data := templateData{
		Title: title,
		// template.JS marks the value as safe JavaScript; the content is the
		// JSON output from RenderJSON which always HTML-escapes string values.
		GraphJSON:       template.JS(jsonBuf.String()), //nolint:gosec // JSON encoder escapes < > &
		LinkDistance:    opts.LinkDistance,
		ChargeStrength:  opts.ChargeStrength,
		CollideRadius:   opts.CollideRadius,
		LayoutDirection: string(opts.LayoutDirection),
		RankSeparation:  opts.RankSeparation,
		NodeSeparation:  opts.NodeSeparation,
	}

	// Select template based on layout mode.
	var tmpl *template.Template
	switch opts.Layout {
	case LayoutLayered:
		tmpl = htmlLayeredTmpl
	case LayoutSwimlane:
		tmpl = htmlSwimlaneTmpl
	default:
		tmpl = htmlTmpl
	}

	if err := tmpl.Execute(w, data); err != nil {
		return fmt.Errorf("render: executing HTML template: %w", err)
	}
	return nil
}
