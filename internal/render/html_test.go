package render_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/IndependentImpact/ttl2d3/internal/graph"
	"github.com/IndependentImpact/ttl2d3/internal/render"
)

// ---------------------------------------------------------------------------
// RenderHTML – error cases
// ---------------------------------------------------------------------------

func TestRenderHTML_NilModel(t *testing.T) {
	var buf bytes.Buffer
	err := render.RenderHTML(nil, render.HTMLOptions{}, &buf)
	if err == nil {
		t.Fatal("RenderHTML(nil) expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// RenderHTML – structural invariants
// ---------------------------------------------------------------------------

func TestRenderHTML_ContainsStructuralElements(t *testing.T) {
	gm := graph.NewGraphModel(nil, nil, graph.Metadata{Title: "Test Ontology"})
	var buf bytes.Buffer
	if err := render.RenderHTML(&gm, render.HTMLOptions{}, &buf); err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		`<!DOCTYPE html>`,
		`<html lang="en">`,
		`<meta charset="UTF-8"`,
		`<title>Test Ontology</title>`,
		`<svg`,
		`id="graph"`,
		`id="legend"`,
		`id="search"`,
		`id="toolbar"`,
		`id="tooltip"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestRenderHTML_ContainsD3Script(t *testing.T) {
	gm := graph.NewGraphModel(nil, nil, graph.Metadata{})
	var buf bytes.Buffer
	if err := render.RenderHTML(&gm, render.HTMLOptions{}, &buf); err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, `https://cdn.jsdelivr.net/npm/d3@7`) {
		t.Errorf("output does not reference D3 v7 CDN")
	}
}

func TestRenderHTML_TitleFallbacks(t *testing.T) {
	tests := []struct {
		name     string
		metadata graph.Metadata
		optTitle string
		wantTag  string
	}{
		{
			name:     "explicit opts.Title wins over metadata",
			optTitle: "Explicit Title",
			metadata: graph.Metadata{Title: "Metadata Title"},
			wantTag:  "<title>Explicit Title</title>",
		},
		{
			name:     "metadata title used when opts empty",
			metadata: graph.Metadata{Title: "Metadata Title"},
			wantTag:  "<title>Metadata Title</title>",
		},
		{
			name:     "base IRI used as fallback",
			metadata: graph.Metadata{BaseIRI: "https://example.org/"},
			wantTag:  "<title>https://example.org/</title>",
		},
		{
			name:     "default title when all empty",
			metadata: graph.Metadata{},
			wantTag:  "<title>ttl2d3 Graph</title>",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gm := graph.NewGraphModel(nil, nil, tc.metadata)
			var buf bytes.Buffer
			opts := render.HTMLOptions{Title: tc.optTitle}
			if err := render.RenderHTML(&gm, opts, &buf); err != nil {
				t.Fatalf("RenderHTML: %v", err)
			}
			if !strings.Contains(buf.String(), tc.wantTag) {
				t.Errorf("output missing %q\ngot:\n%s", tc.wantTag, buf.String())
			}
		})
	}
}

func TestRenderHTML_ContainsGraphJSON(t *testing.T) {
	nodes := []graph.Node{
		graph.NewNode("https://example.org/A", "ClassA", graph.NodeTypeClass, "example"),
		graph.NewNode("https://example.org/B", "ClassB", graph.NodeTypeClass, "example"),
	}
	links := []graph.Link{
		graph.NewLink("https://example.org/A", "https://example.org/B", "subClassOf"),
	}
	gm := graph.NewGraphModel(nodes, links, graph.Metadata{})
	var buf bytes.Buffer
	if err := render.RenderHTML(&gm, render.HTMLOptions{}, &buf); err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	out := buf.String()

	// The node IRIs and labels must appear in the embedded JSON.
	for _, want := range []string{
		`"https://example.org/A"`,
		`"https://example.org/B"`,
		`"ClassA"`,
		`"ClassB"`,
		`"subClassOf"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("embedded JSON missing %q", want)
		}
	}
}

func TestRenderHTML_ConfigurableParams(t *testing.T) {
	gm := graph.NewGraphModel(nil, nil, graph.Metadata{})
	var buf bytes.Buffer
	opts := render.HTMLOptions{
		LinkDistance:   120,
		ChargeStrength: -500,
		CollideRadius:  30,
	}
	if err := render.RenderHTML(&gm, opts, &buf); err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	out := buf.String()

	for _, want := range []string{"120", "-500", "30"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected config value %q to appear in output", want)
		}
	}
}

func TestRenderHTML_DefaultParamsApplied(t *testing.T) {
	// Zero-value opts should fall back to DefaultHTMLOptions values.
	gm := graph.NewGraphModel(nil, nil, graph.Metadata{})
	var buf bytes.Buffer
	if err := render.RenderHTML(&gm, render.HTMLOptions{}, &buf); err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	out := buf.String()

	defaults := render.DefaultHTMLOptions()
	for _, want := range []string{
		fmt.Sprintf("%g", defaults.LinkDistance),
		fmt.Sprintf("%g", defaults.ChargeStrength),
		fmt.Sprintf("%g", defaults.CollideRadius),
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected default value %q in output", want)
		}
	}
}

func TestRenderHTML_LegendPresent(t *testing.T) {
	gm := graph.NewGraphModel(nil, nil, graph.Metadata{})
	var buf bytes.Buffer
	if err := render.RenderHTML(&gm, render.HTMLOptions{}, &buf); err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	out := buf.String()

	for _, want := range []string{"Class", "Union", "Property", "Instance", "Literal", "Origin", "Namespaces"} {
		if !strings.Contains(out, want) {
			t.Errorf("legend missing entry %q", want)
		}
	}
}

func TestRenderHTML_NodeTypesInOutput(t *testing.T) {
	nodes := []graph.Node{
		graph.NewNode("https://example.org/Cls", "MyClass", graph.NodeTypeClass, "ex"),
		graph.NewNode("https://example.org/Prop", "myProp", graph.NodeTypeProperty, "ex"),
		graph.NewNode("https://example.org/Union", "union", graph.NodeTypeUnion, "owl"),
		graph.NewNode("https://example.org/Inst", "MyInstance", graph.NodeTypeInstance, "ex"),
		graph.NewNode("https://example.org/Lit", "42", graph.NodeTypeLiteral, ""),
	}
	gm := graph.NewGraphModel(nodes, nil, graph.Metadata{})
	var buf bytes.Buffer
	if err := render.RenderHTML(&gm, render.HTMLOptions{}, &buf); err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	out := buf.String()

	for _, want := range []string{`"class"`, `"property"`, `"union"`, `"instance"`, `"literal"`} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing node type %q", want)
		}
	}
}

func TestRenderHTML_HTMLEscapingInTitle(t *testing.T) {
	// Titles with HTML special characters must be escaped in the <title> element.
	gm := graph.NewGraphModel(nil, nil, graph.Metadata{Title: `My <Ontology> & "More"`})
	var buf bytes.Buffer
	if err := render.RenderHTML(&gm, render.HTMLOptions{}, &buf); err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	out := buf.String()

	// html/template must escape the title correctly.
	if strings.Contains(out, `<My`) {
		t.Errorf("title was not HTML-escaped; raw < found in output")
	}
	if !strings.Contains(out, `My`) {
		t.Errorf("title content missing from output")
	}
}

// ---------------------------------------------------------------------------
// RenderHTML – golden-file comparison
// ---------------------------------------------------------------------------

func TestRenderHTML_Golden_Simple(t *testing.T) {
	nodes := []graph.Node{
		graph.NewNode("https://example.org/A", "A", graph.NodeTypeClass, "example"),
		graph.NewNode("https://example.org/B", "B", graph.NodeTypeClass, "example"),
	}
	links := []graph.Link{
		graph.NewLink("https://example.org/A", "https://example.org/B", "subClassOf"),
	}
	meta := graph.NewMetadata("Test Ontology", "", "1.0", "https://example.org/")
	gm := graph.NewGraphModel(nodes, links, meta)
	var buf bytes.Buffer
	opts := render.DefaultHTMLOptions()
	opts.Title = "Test Ontology"
	if err := render.RenderHTML(&gm, opts, &buf); err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	assertGolden(t, goldenPath(t, "simple.html"), &buf)
}

func TestRenderHTML_Golden_Empty(t *testing.T) {
	gm := graph.NewGraphModel(nil, nil, graph.Metadata{})
	var buf bytes.Buffer
	if err := render.RenderHTML(&gm, render.DefaultHTMLOptions(), &buf); err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	assertGolden(t, goldenPath(t, "empty.html"), &buf)
}

// ---------------------------------------------------------------------------
// RenderHTML – parallel-edge curvature
// ---------------------------------------------------------------------------

func TestRenderHTML_ParallelEdgesUsePath(t *testing.T) {
	// Links must be rendered as SVG <path> elements (not <line>) so that the
	// JavaScript can apply per-edge curvature offsets for parallel edges.
	nodes := []graph.Node{
		graph.NewNode("https://example.org/A", "A", graph.NodeTypeClass, "example"),
		graph.NewNode("https://example.org/B", "B", graph.NodeTypeClass, "example"),
	}
	links := []graph.Link{
		graph.NewLink("https://example.org/A", "https://example.org/B", "prop one"),
		graph.NewLink("https://example.org/A", "https://example.org/B", "prop two"),
	}
	gm := graph.NewGraphModel(nodes, links, graph.Metadata{})
	var buf bytes.Buffer
	if err := render.RenderHTML(&gm, render.HTMLOptions{}, &buf); err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	out := buf.String()

	// Both link labels must appear in the embedded JSON.
	for _, want := range []string{`"prop one"`, `"prop two"`} {
		if !strings.Contains(out, want) {
			t.Errorf("embedded JSON missing %q", want)
		}
	}

	// Links must be rendered as <path> with curvature pre-computation logic.
	if !strings.Contains(out, `join('path')`) {
		t.Error("links must use <path> elements, not <line>")
	}
	if strings.Contains(out, `join('line')`) {
		t.Error("links must not use <line> elements")
	}
	if !strings.Contains(out, `_curvature`) {
		t.Error("template must include curvature pre-computation for parallel edges")
	}
}

// ---------------------------------------------------------------------------
// RenderHTML – layered layout
// ---------------------------------------------------------------------------

func TestRenderHTML_Layered_ContainsStructuralElements(t *testing.T) {
	gm := graph.NewGraphModel(nil, nil, graph.Metadata{Title: "Layered Test"})
	var buf bytes.Buffer
	opts := render.HTMLOptions{Layout: render.LayoutLayered}
	if err := render.RenderHTML(&gm, opts, &buf); err != nil {
		t.Fatalf("RenderHTML layered: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		`<!DOCTYPE html>`,
		`<title>Layered Test</title>`,
		`<svg`,
		`id="graph"`,
		`id="legend"`,
		`id="search"`,
		`id="toolbar"`,
		`id="tooltip"`,
		`https://cdn.jsdelivr.net/npm/d3@7`,
		`layout-badge`,
		`layered`,
		// Back-edge detection must be present.
		`_isBackEdge`,
		// Rank assignment must be present.
		`rank`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("layered output missing %q", want)
		}
	}
}

func TestRenderHTML_Layered_NoForceSimulation(t *testing.T) {
	gm := graph.NewGraphModel(nil, nil, graph.Metadata{})
	var buf bytes.Buffer
	opts := render.HTMLOptions{Layout: render.LayoutLayered}
	if err := render.RenderHTML(&gm, opts, &buf); err != nil {
		t.Fatalf("RenderHTML layered: %v", err)
	}
	out := buf.String()

	// Layered layout must not use D3 force simulation.
	if strings.Contains(out, `forceSimulation`) {
		t.Error("layered layout must not contain forceSimulation")
	}
}

func TestRenderHTML_Layered_BackEdgeDetection(t *testing.T) {
	// A graph with a cycle: A → B → C → A
	nodes := []graph.Node{
		graph.NewNode("https://example.org/A", "A", graph.NodeTypeClass, "ex"),
		graph.NewNode("https://example.org/B", "B", graph.NodeTypeClass, "ex"),
		graph.NewNode("https://example.org/C", "C", graph.NodeTypeClass, "ex"),
	}
	links := []graph.Link{
		graph.NewLink("https://example.org/A", "https://example.org/B", "next"),
		graph.NewLink("https://example.org/B", "https://example.org/C", "next"),
		graph.NewLink("https://example.org/C", "https://example.org/A", "loop"), // back-edge
	}
	gm := graph.NewGraphModel(nodes, links, graph.Metadata{})
	var buf bytes.Buffer
	opts := render.HTMLOptions{Layout: render.LayoutLayered}
	if err := render.RenderHTML(&gm, opts, &buf); err != nil {
		t.Fatalf("RenderHTML layered: %v", err)
	}
	out := buf.String()

	// Back-edge CSS class and separate arrowhead must appear.
	if !strings.Contains(out, `back-edge`) {
		t.Error("layered output must reference back-edge CSS class")
	}
	if !strings.Contains(out, `arrowhead-back`) {
		t.Error("layered output must reference arrowhead-back marker for back edges")
	}
}

func TestRenderHTML_Layered_DeterministicOutput(t *testing.T) {
	nodes := []graph.Node{
		graph.NewNode("https://example.org/A", "A", graph.NodeTypeClass, "ex"),
		graph.NewNode("https://example.org/B", "B", graph.NodeTypeClass, "ex"),
	}
	links := []graph.Link{
		graph.NewLink("https://example.org/A", "https://example.org/B", "next"),
	}
	gm := graph.NewGraphModel(nodes, links, graph.Metadata{})
	opts := render.HTMLOptions{Layout: render.LayoutLayered}

	var buf1, buf2 bytes.Buffer
	if err := render.RenderHTML(&gm, opts, &buf1); err != nil {
		t.Fatalf("RenderHTML layered (run 1): %v", err)
	}
	if err := render.RenderHTML(&gm, opts, &buf2); err != nil {
		t.Fatalf("RenderHTML layered (run 2): %v", err)
	}
	if buf1.String() != buf2.String() {
		t.Error("layered layout output is not deterministic across runs")
	}
}

func TestRenderHTML_Layered_LayoutDirection_TB(t *testing.T) {
	gm := graph.NewGraphModel(nil, nil, graph.Metadata{})
	var buf bytes.Buffer
	opts := render.HTMLOptions{
		Layout:          render.LayoutLayered,
		LayoutDirection: render.LayoutDirectionTB,
	}
	if err := render.RenderHTML(&gm, opts, &buf); err != nil {
		t.Fatalf("RenderHTML layered tb: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `'tb'`) {
		t.Error("layered tb output must embed direction 'tb'")
	}
}

func TestRenderHTML_Layered_CustomSeparations(t *testing.T) {
	gm := graph.NewGraphModel(nil, nil, graph.Metadata{})
	var buf bytes.Buffer
	opts := render.HTMLOptions{
		Layout:         render.LayoutLayered,
		RankSeparation: 250,
		NodeSeparation: 100,
	}
	if err := render.RenderHTML(&gm, opts, &buf); err != nil {
		t.Fatalf("RenderHTML layered custom sep: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"250", "100"} {
		if !strings.Contains(out, want) {
			t.Errorf("layered output missing custom separation value %q", want)
		}
	}
}

func TestRenderHTML_Layered_Golden(t *testing.T) {
	nodes := []graph.Node{
		graph.NewNode("https://example.org/A", "A", graph.NodeTypeClass, "example"),
		graph.NewNode("https://example.org/B", "B", graph.NodeTypeClass, "example"),
		graph.NewNode("https://example.org/C", "C", graph.NodeTypeClass, "example"),
	}
	links := []graph.Link{
		graph.NewLink("https://example.org/A", "https://example.org/B", "next"),
		graph.NewLink("https://example.org/B", "https://example.org/C", "next"),
		graph.NewLink("https://example.org/C", "https://example.org/A", "loop"),
	}
	meta := graph.NewMetadata("Layered Test", "", "1.0", "https://example.org/")
	gm := graph.NewGraphModel(nodes, links, meta)
	var buf bytes.Buffer
	opts := render.DefaultHTMLOptions()
	opts.Layout = render.LayoutLayered
	opts.Title = "Layered Test"
	if err := render.RenderHTML(&gm, opts, &buf); err != nil {
		t.Fatalf("RenderHTML layered golden: %v", err)
	}
	assertGolden(t, goldenPath(t, "layered.html"), &buf)
}

// ---------------------------------------------------------------------------
// RenderHTML – swimlane layout
// ---------------------------------------------------------------------------

func TestRenderHTML_Swimlane_ContainsStructuralElements(t *testing.T) {
	gm := graph.NewGraphModel(nil, nil, graph.Metadata{Title: "Swimlane Test"})
	var buf bytes.Buffer
	opts := render.HTMLOptions{Layout: render.LayoutSwimlane}
	if err := render.RenderHTML(&gm, opts, &buf); err != nil {
		t.Fatalf("RenderHTML swimlane: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		`<!DOCTYPE html>`,
		`<title>Swimlane Test</title>`,
		`<svg`,
		`id="graph"`,
		`id="legend"`,
		`id="search"`,
		`id="toolbar"`,
		`id="tooltip"`,
		`https://cdn.jsdelivr.net/npm/d3@7`,
		`layout-badge`,
		`swimlane`,
		`lane-bg`,
		`lane-label`,
		`_isBackEdge`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("swimlane output missing %q", want)
		}
	}
}

func TestRenderHTML_Swimlane_NoForceSimulation(t *testing.T) {
	gm := graph.NewGraphModel(nil, nil, graph.Metadata{})
	var buf bytes.Buffer
	opts := render.HTMLOptions{Layout: render.LayoutSwimlane}
	if err := render.RenderHTML(&gm, opts, &buf); err != nil {
		t.Fatalf("RenderHTML swimlane: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, `forceSimulation`) {
		t.Error("swimlane layout must not contain forceSimulation")
	}
}

func TestRenderHTML_Swimlane_LaneBandsPresent(t *testing.T) {
	// Nodes with different groups should produce distinct lanes.
	nodes := []graph.Node{
		graph.NewNode("https://example.org/A", "A", graph.NodeTypeClass, "groupA"),
		graph.NewNode("https://example.org/B", "B", graph.NodeTypeClass, "groupB"),
	}
	gm := graph.NewGraphModel(nodes, nil, graph.Metadata{})
	var buf bytes.Buffer
	opts := render.HTMLOptions{Layout: render.LayoutSwimlane}
	if err := render.RenderHTML(&gm, opts, &buf); err != nil {
		t.Fatalf("RenderHTML swimlane: %v", err)
	}
	out := buf.String()

	// Lane background rectangles must be rendered.
	if !strings.Contains(out, `lane-bg`) {
		t.Error("swimlane output must contain lane-bg elements")
	}
}

func TestRenderHTML_Swimlane_Golden(t *testing.T) {
	nodes := []graph.Node{
		graph.NewNode("https://example.org/A", "A", graph.NodeTypeClass, "groupA"),
		graph.NewNode("https://example.org/B", "B", graph.NodeTypeClass, "groupA"),
		graph.NewNode("https://example.org/C", "C", graph.NodeTypeClass, "groupB"),
	}
	links := []graph.Link{
		graph.NewLink("https://example.org/A", "https://example.org/B", "next"),
		graph.NewLink("https://example.org/B", "https://example.org/C", "cross"),
	}
	meta := graph.NewMetadata("Swimlane Test", "", "1.0", "https://example.org/")
	gm := graph.NewGraphModel(nodes, links, meta)
	var buf bytes.Buffer
	opts := render.DefaultHTMLOptions()
	opts.Layout = render.LayoutSwimlane
	opts.Title = "Swimlane Test"
	if err := render.RenderHTML(&gm, opts, &buf); err != nil {
		t.Fatalf("RenderHTML swimlane golden: %v", err)
	}
	assertGolden(t, goldenPath(t, "swimlane.html"), &buf)
}

