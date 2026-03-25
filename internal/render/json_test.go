package render_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IndependentImpact/ttl2d3/internal/graph"
	"github.com/IndependentImpact/ttl2d3/internal/render"
)

// update controls whether golden files are regenerated during test runs.
// Run: go test -update ./internal/render/
var update = flag.Bool("update", false, "update golden files")

// goldenPath returns the absolute path to a golden file inside testdata/golden/.
func goldenPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("..", "..", "testdata", "golden", name)
}

// assertGolden checks buf against the golden file at path.
// When -update is set the golden file is overwritten instead.
func assertGolden(t *testing.T, path string, buf *bytes.Buffer) {
	t.Helper()
	if *update {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("assertGolden: mkdir %s: %v", dir, err)
		}
		if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
			t.Fatalf("assertGolden: write %s: %v", path, err)
		}
		t.Logf("updated golden file: %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("assertGolden: read %s: %v (hint: run with -update to create it)", path, err)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("output mismatch for %s\ngot:\n%s\nwant:\n%s", path, buf.String(), string(want))
	}
}

// ---------------------------------------------------------------------------
// RenderJSON – error cases
// ---------------------------------------------------------------------------

func TestRenderJSON_NilModel(t *testing.T) {
	var buf bytes.Buffer
	err := render.RenderJSON(nil, &buf)
	if err == nil {
		t.Fatal("RenderJSON(nil) expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// RenderJSON – structural invariants
// ---------------------------------------------------------------------------

func TestRenderJSON_EmptyModel(t *testing.T) {
	gm := graph.NewGraphModel(nil, nil, graph.Metadata{})
	var buf bytes.Buffer
	if err := render.RenderJSON(&gm, &buf); err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}

	out := buf.String()

	// Top-level required keys must always be present.
	for _, key := range []string{`"nodes"`, `"links"`, `"metadata"`} {
		if !strings.Contains(out, key) {
			t.Errorf("output missing required key %s", key)
		}
	}

	// nodes and links must be empty arrays, not null.
	if !strings.Contains(out, `"nodes": []`) {
		t.Errorf("empty nodes should be [] not null; got:\n%s", out)
	}
	if !strings.Contains(out, `"links": []`) {
		t.Errorf("empty links should be [] not null; got:\n%s", out)
	}
}

func TestRenderJSON_NodeFields(t *testing.T) {
	// Every node must carry id, label, type; group is omitted when empty.
	nodes := []graph.Node{
		graph.NewNode("https://example.org/A", "A Label", graph.NodeTypeClass, "example"),
		graph.NewNode("https://example.org/B", "B Label", graph.NodeTypeProperty, ""),
	}
	gm := graph.NewGraphModel(nodes, nil, graph.Metadata{})
	var buf bytes.Buffer
	if err := render.RenderJSON(&gm, &buf); err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}

	out := buf.String()

	for _, want := range []string{
		`"id": "https://example.org/A"`,
		`"label": "A Label"`,
		`"type": "class"`,
		`"group": "example"`,
		`"id": "https://example.org/B"`,
		`"type": "property"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\ngot:\n%s", want, out)
		}
	}

	// group must be absent for node B (empty group → omitempty).
	if strings.Count(out, `"group"`) != 1 {
		t.Errorf("expected exactly 1 'group' key (node B has empty group); got:\n%s", out)
	}
}

func TestRenderJSON_LinkFields(t *testing.T) {
	// Every link must carry source and target; label is omitted when empty.
	nodes := []graph.Node{
		graph.NewNode("https://example.org/A", "A", graph.NodeTypeClass, ""),
		graph.NewNode("https://example.org/B", "B", graph.NodeTypeClass, ""),
		graph.NewNode("https://example.org/C", "C", graph.NodeTypeClass, ""),
	}
	links := []graph.Link{
		graph.NewLink("https://example.org/A", "https://example.org/B", "subClassOf"),
		graph.NewLink("https://example.org/B", "https://example.org/C", ""), // no label
	}
	gm := graph.NewGraphModel(nodes, links, graph.Metadata{})
	var buf bytes.Buffer
	if err := render.RenderJSON(&gm, &buf); err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}

	out := buf.String()

	for _, want := range []string{
		`"source": "https://example.org/A"`,
		`"target": "https://example.org/B"`,
		`"label": "subClassOf"`,
		`"source": "https://example.org/B"`,
		`"target": "https://example.org/C"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\ngot:\n%s", want, out)
		}
	}

	// The link with an empty label must omit the "label" key entirely.
	// We verify this by checking that "label": "" never appears in the output.
	if strings.Contains(out, `"label": ""`) {
		t.Errorf("link with empty label must not emit 'label' key; got:\n%s", out)
	}
}

func TestRenderJSON_MetadataFields(t *testing.T) {
	meta := graph.NewMetadata("My Ontology", "A description.", "3.0", "https://example.org/")
	gm := graph.NewGraphModel(nil, nil, meta)
	var buf bytes.Buffer
	if err := render.RenderJSON(&gm, &buf); err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}

	out := buf.String()

	for _, want := range []string{
		`"title": "My Ontology"`,
		`"description": "A description."`,
		`"version": "3.0"`,
		`"baseIRI": "https://example.org/"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\ngot:\n%s", want, out)
		}
	}
}

func TestRenderJSON_NodeTypeValues(t *testing.T) {
	// NodeType strings must match the enum values in the JSON schema.
	tests := []struct {
		nodeType graph.NodeType
		want     string
	}{
		{graph.NodeTypeClass, `"type": "class"`},
		{graph.NodeTypeProperty, `"type": "property"`},
		{graph.NodeTypeInstance, `"type": "instance"`},
		{graph.NodeTypeLiteral, `"type": "literal"`},
	}

	for _, tc := range tests {
		t.Run(string(tc.nodeType), func(t *testing.T) {
			nodes := []graph.Node{
				graph.NewNode("https://example.org/X", "X", tc.nodeType, ""),
			}
			gm := graph.NewGraphModel(nodes, nil, graph.Metadata{})
			var buf bytes.Buffer
			if err := render.RenderJSON(&gm, &buf); err != nil {
				t.Fatalf("RenderJSON: %v", err)
			}
			if !strings.Contains(buf.String(), tc.want) {
				t.Errorf("output missing %q\ngot:\n%s", tc.want, buf.String())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// RenderJSON – golden-file comparison
// ---------------------------------------------------------------------------

func TestRenderJSON_Golden_Empty(t *testing.T) {
	gm := graph.NewGraphModel(nil, nil, graph.Metadata{})
	var buf bytes.Buffer
	if err := render.RenderJSON(&gm, &buf); err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}
	assertGolden(t, goldenPath(t, "empty.json"), &buf)
}

func TestRenderJSON_Golden_Simple(t *testing.T) {
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
	if err := render.RenderJSON(&gm, &buf); err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}
	assertGolden(t, goldenPath(t, "simple.json"), &buf)
}

func TestRenderJSON_Golden_AllTypes(t *testing.T) {
	nodes := []graph.Node{
		graph.NewNode("https://example.org/Cls", "MyClass", graph.NodeTypeClass, "example"),
		graph.NewNode("https://example.org/Prop", "myProp", graph.NodeTypeProperty, "example"),
		graph.NewNode("https://example.org/Inst", "MyInstance", graph.NodeTypeInstance, "example"),
		graph.NewNode("https://example.org/Lit", "42", graph.NodeTypeLiteral, ""),
	}
	links := []graph.Link{
		graph.NewLink("https://example.org/Cls", "https://example.org/Prop", "uses"),
		graph.NewLink("https://example.org/Inst", "https://example.org/Cls", "instanceOf"),
		graph.NewLink("https://example.org/Prop", "https://example.org/Lit", "hasValue"),
	}
	meta := graph.NewMetadata("All Types Ontology", "Covers all node types.", "2.0", "https://example.org/")
	gm := graph.NewGraphModel(nodes, links, meta)
	var buf bytes.Buffer
	if err := render.RenderJSON(&gm, &buf); err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}
	assertGolden(t, goldenPath(t, "all_types.json"), &buf)
}
