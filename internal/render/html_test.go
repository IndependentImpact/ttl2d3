package render_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/IndependentImpact/ttl2d3/internal/graph"
	"github.com/IndependentImpact/ttl2d3/internal/render"
	"github.com/IndependentImpact/ttl2d3/internal/transform"
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
		`id="search-edges"`,
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

func TestRenderHTML_TypeFilterCheckboxes(t *testing.T) {
	// Each node type must have a corresponding type-filter checkbox in the legend,
	// defaulting to checked (all types visible on initial load).
	gm := graph.NewGraphModel(nil, nil, graph.Metadata{})
	var buf bytes.Buffer
	if err := render.RenderHTML(&gm, render.HTMLOptions{}, &buf); err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	out := buf.String()

	for _, nodeType := range []string{"class", "union", "property", "instance", "literal"} {
		want := `class="type-filter" data-type="` + nodeType + `" checked`
		if !strings.Contains(out, want) {
			t.Errorf("legend missing checked type-filter checkbox for %q", nodeType)
		}
	}
}

func TestRenderHTML_PropertySearchInput(t *testing.T) {
	// The property/edge search input must be present in the toolbar.
	gm := graph.NewGraphModel(nil, nil, graph.Metadata{})
	var buf bytes.Buffer
	if err := render.RenderHTML(&gm, render.HTMLOptions{}, &buf); err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		`id="search-edges"`,
		`placeholder="Search properties…"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing property search element %q", want)
		}
	}
}

func TestRenderHTML_ApplyFiltersInScript(t *testing.T) {
	// The combined applyFilters function and type-filter/edge-search event
	// listeners must be present in the rendered JavaScript.
	gm := graph.NewGraphModel(nil, nil, graph.Metadata{})
	var buf bytes.Buffer
	if err := render.RenderHTML(&gm, render.HTMLOptions{}, &buf); err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		`applyFilters`,
		`visibleTypes`,
		`search-edges`,
		`type-filter`,
		`type-hidden`,
		`link-highlighted`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("script missing expected token %q", want)
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
// RenderWorkflowPlan – structural invariants
// ---------------------------------------------------------------------------

func TestRenderWorkflowPlan_NilModel(t *testing.T) {
	var buf bytes.Buffer
	err := render.RenderWorkflowPlan(nil, "Test", render.WorkflowPlanOptions{}, &buf)
	if err == nil {
		t.Fatal("RenderWorkflowPlan(nil) expected error, got nil")
	}
}

func TestRenderWorkflowPlan_ContainsStructuralElements(t *testing.T) {
	wm := &transform.WorkflowModel{
		Plans: []transform.WorkflowPlan{
			{ID: "https://example.org/Plan1", Label: "Plan One"},
		},
	}
	var buf bytes.Buffer
	if err := render.RenderWorkflowPlan(wm, "My Workflow", render.WorkflowPlanOptions{}, &buf); err != nil {
		t.Fatalf("RenderWorkflowPlan: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		`<!DOCTYPE html>`,
		`<html lang="en">`,
		`<meta charset="UTF-8"`,
		`<title>My Workflow</title>`,
		`id="toolbar"`,
		`id="tooltip"`,
		`id="search"`,
		`workflow plan`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("workflowplan output missing %q", want)
		}
	}
}

func TestRenderWorkflowPlan_DefaultTitle(t *testing.T) {
	wm := &transform.WorkflowModel{Plans: nil}
	var buf bytes.Buffer
	if err := render.RenderWorkflowPlan(wm, "", render.WorkflowPlanOptions{}, &buf); err != nil {
		t.Fatalf("RenderWorkflowPlan: %v", err)
	}
	if !strings.Contains(buf.String(), "ttl2d3 Workflow Plan") {
		t.Error("default title missing from output")
	}
}

func TestRenderWorkflowPlan_EmptyPlans(t *testing.T) {
	wm := &transform.WorkflowModel{Plans: nil}
	var buf bytes.Buffer
	if err := render.RenderWorkflowPlan(wm, "Empty", render.WorkflowPlanOptions{}, &buf); err != nil {
		t.Fatalf("RenderWorkflowPlan: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "No indimp:WorkflowPlan") {
		t.Error("empty-plans message missing from output")
	}
}

func TestRenderWorkflowPlan_StepsAndTransitionsEmbedded(t *testing.T) {
	wm := &transform.WorkflowModel{
		Plans: []transform.WorkflowPlan{
			{
				ID:    "https://example.org/Plan1",
				Label: "Document Approval",
				Steps: []transform.WorkflowStep{
					{ID: "https://example.org/Submit", Label: "Submit", Actor: "Author"},
					{ID: "https://example.org/Review", Label: "Review", Actor: "Reviewer"},
				},
				Transitions: []transform.WorkflowTransition{
					{From: "https://example.org/Submit", To: "https://example.org/Review", Label: "submit"},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := render.RenderWorkflowPlan(wm, "Test Workflow", render.WorkflowPlanOptions{}, &buf); err != nil {
		t.Fatalf("RenderWorkflowPlan: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		`"Document Approval"`,
		`"Submit"`,
		`"Review"`,
		`"Author"`,
		`"Reviewer"`,
		`"submit"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("workflow plan output missing %q", want)
		}
	}
}

func TestRenderWorkflowPlan_HTMLEscaping(t *testing.T) {
	wm := &transform.WorkflowModel{
		Plans: []transform.WorkflowPlan{
			{ID: "https://example.org/P", Label: `Plan <One> & "Two"`},
		},
	}
	var buf bytes.Buffer
	if err := render.RenderWorkflowPlan(wm, `Title <&> "test"`, render.WorkflowPlanOptions{}, &buf); err != nil {
		t.Fatalf("RenderWorkflowPlan: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, `<Title`) {
		t.Error("title was not HTML-escaped in workflowplan output")
	}
}

func TestRenderWorkflowPlan_NodeSpacingApplied(t *testing.T) {
	wm := &transform.WorkflowModel{
		Plans: []transform.WorkflowPlan{
			{ID: "https://example.org/Plan1", Label: "Plan One"},
		},
	}

	tests := []struct {
		name        string
		opts        render.WorkflowPlanOptions
		wantPx      string
		wantDefault bool
	}{
		{
			name:        "custom spacing 320px appears in output",
			opts:        render.WorkflowPlanOptions{NodeSpacing: 320},
			wantPx:      "320px",
			wantDefault: false,
		},
		{
			name:        "zero opts falls back to default 180px",
			opts:        render.WorkflowPlanOptions{},
			wantPx:      "180px",
			wantDefault: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := render.RenderWorkflowPlan(wm, "Spacing Test", tc.opts, &buf); err != nil {
				t.Fatalf("RenderWorkflowPlan: %v", err)
			}
			out := buf.String()
			if !strings.Contains(out, tc.wantPx) {
				t.Errorf("expected %q in output; got (excerpt):\n%s",
					tc.wantPx, out[:min(500, len(out))])
			}
		})
	}
}

func TestRenderWorkflowPlan_DefaultNodeSpacing(t *testing.T) {
	defaults := render.DefaultWorkflowPlanOptions()
	if defaults.NodeSpacing != 180 {
		t.Errorf("DefaultWorkflowPlanOptions().NodeSpacing = %g, want 180", defaults.NodeSpacing)
	}
}
