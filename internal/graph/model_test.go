package graph_test

import (
	"testing"

	"github.com/IndependentImpact/ttl2d3/internal/graph"
)

// ---------------------------------------------------------------------------
// Constructor tests
// ---------------------------------------------------------------------------

func TestNewNode(t *testing.T) {
	n := graph.NewNode(
		"https://example.org/Animal",
		"Animal",
		graph.NodeTypeClass,
		"example",
	)

	if n.ID != "https://example.org/Animal" {
		t.Errorf("NewNode ID = %q, want %q", n.ID, "https://example.org/Animal")
	}
	if n.Label != "Animal" {
		t.Errorf("NewNode Label = %q, want %q", n.Label, "Animal")
	}
	if n.Type != graph.NodeTypeClass {
		t.Errorf("NewNode Type = %q, want %q", n.Type, graph.NodeTypeClass)
	}
	if n.Group != "example" {
		t.Errorf("NewNode Group = %q, want %q", n.Group, "example")
	}
}

func TestNewLink(t *testing.T) {
	l := graph.NewLink(
		"https://example.org/Animal",
		"https://example.org/Mammal",
		"subClassOf",
	)

	if l.Source != "https://example.org/Animal" {
		t.Errorf("NewLink Source = %q, want %q", l.Source, "https://example.org/Animal")
	}
	if l.Target != "https://example.org/Mammal" {
		t.Errorf("NewLink Target = %q, want %q", l.Target, "https://example.org/Mammal")
	}
	if l.Label != "subClassOf" {
		t.Errorf("NewLink Label = %q, want %q", l.Label, "subClassOf")
	}
}

func TestNewMetadata(t *testing.T) {
	m := graph.NewMetadata("My Ontology", "A test ontology.", "1.0", "https://example.org/")

	if m.Title != "My Ontology" {
		t.Errorf("NewMetadata Title = %q, want %q", m.Title, "My Ontology")
	}
	if m.Description != "A test ontology." {
		t.Errorf("NewMetadata Description = %q, want %q", m.Description, "A test ontology.")
	}
	if m.Version != "1.0" {
		t.Errorf("NewMetadata Version = %q, want %q", m.Version, "1.0")
	}
	if m.BaseIRI != "https://example.org/" {
		t.Errorf("NewMetadata BaseIRI = %q, want %q", m.BaseIRI, "https://example.org/")
	}
}

func TestNewGraphModel(t *testing.T) {
	nodes := []graph.Node{
		graph.NewNode("https://example.org/A", "A", graph.NodeTypeClass, "example"),
		graph.NewNode("https://example.org/B", "B", graph.NodeTypeClass, "example"),
	}
	links := []graph.Link{
		graph.NewLink("https://example.org/A", "https://example.org/B", "related"),
	}
	meta := graph.NewMetadata("Test", "", "", "https://example.org/")

	gm := graph.NewGraphModel(nodes, links, meta)

	if gm.NodeCount() != 2 {
		t.Errorf("NewGraphModel NodeCount = %d, want 2", gm.NodeCount())
	}
	if gm.LinkCount() != 1 {
		t.Errorf("NewGraphModel LinkCount = %d, want 1", gm.LinkCount())
	}
	if gm.Metadata.Title != "Test" {
		t.Errorf("NewGraphModel Metadata.Title = %q, want %q", gm.Metadata.Title, "Test")
	}
}

// ---------------------------------------------------------------------------
// NodeType constant tests
// ---------------------------------------------------------------------------

func TestNodeTypeConstants(t *testing.T) {
	tests := []struct {
		constant graph.NodeType
		want     string
	}{
		{graph.NodeTypeClass, "class"},
		{graph.NodeTypeProperty, "property"},
		{graph.NodeTypeUnion, "union"},
		{graph.NodeTypeInstance, "instance"},
		{graph.NodeTypeLiteral, "literal"},
	}

	for _, tc := range tests {
		if string(tc.constant) != tc.want {
			t.Errorf("NodeType constant = %q, want %q", tc.constant, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// GraphModel.Validate tests
// ---------------------------------------------------------------------------

func TestGraphModelValidate(t *testing.T) {
	validNodes := []graph.Node{
		graph.NewNode("https://example.org/A", "A", graph.NodeTypeClass, "example"),
		graph.NewNode("https://example.org/B", "B", graph.NodeTypeProperty, "example"),
	}
	validLinks := []graph.Link{
		graph.NewLink("https://example.org/A", "https://example.org/B", "uses"),
	}
	validMeta := graph.NewMetadata("Ontology", "", "1.0", "https://example.org/")

	tests := []struct {
		name    string
		model   graph.GraphModel
		wantErr bool
	}{
		{
			name:    "valid model",
			model:   graph.NewGraphModel(validNodes, validLinks, validMeta),
			wantErr: false,
		},
		{
			name:    "empty model (no nodes or links)",
			model:   graph.NewGraphModel(nil, nil, validMeta),
			wantErr: false,
		},
		{
			name: "node with empty ID",
			model: graph.NewGraphModel(
				[]graph.Node{
					graph.NewNode("", "Missing ID", graph.NodeTypeClass, ""),
				},
				nil,
				validMeta,
			),
			wantErr: true,
		},
		{
			name: "duplicate node IDs",
			model: graph.NewGraphModel(
				[]graph.Node{
					graph.NewNode("https://example.org/A", "A", graph.NodeTypeClass, ""),
					graph.NewNode("https://example.org/A", "A duplicate", graph.NodeTypeClass, ""),
				},
				nil,
				validMeta,
			),
			wantErr: true,
		},
		{
			name: "link with empty Source",
			model: graph.NewGraphModel(
				validNodes,
				[]graph.Link{
					graph.NewLink("", "https://example.org/B", "uses"),
				},
				validMeta,
			),
			wantErr: true,
		},
		{
			name: "link with empty Target",
			model: graph.NewGraphModel(
				validNodes,
				[]graph.Link{
					graph.NewLink("https://example.org/A", "", "uses"),
				},
				validMeta,
			),
			wantErr: true,
		},
		{
			name: "link source references unknown node",
			model: graph.NewGraphModel(
				validNodes,
				[]graph.Link{
					graph.NewLink("https://example.org/Unknown", "https://example.org/B", "uses"),
				},
				validMeta,
			),
			wantErr: true,
		},
		{
			name: "link target references unknown node",
			model: graph.NewGraphModel(
				validNodes,
				[]graph.Link{
					graph.NewLink("https://example.org/A", "https://example.org/Unknown", "uses"),
				},
				validMeta,
			),
			wantErr: true,
		},
		{
			name: "self-referencing link is valid",
			model: graph.NewGraphModel(
				[]graph.Node{
					graph.NewNode("https://example.org/A", "A", graph.NodeTypeClass, ""),
				},
				[]graph.Link{
					graph.NewLink("https://example.org/A", "https://example.org/A", "sameAs"),
				},
				validMeta,
			),
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.model.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// NodeCount / LinkCount tests
// ---------------------------------------------------------------------------

func TestNodeCountLinkCount(t *testing.T) {
	nodes := []graph.Node{
		graph.NewNode("https://example.org/A", "A", graph.NodeTypeClass, ""),
		graph.NewNode("https://example.org/B", "B", graph.NodeTypeInstance, ""),
		graph.NewNode("https://example.org/C", "C", graph.NodeTypeLiteral, ""),
	}
	links := []graph.Link{
		graph.NewLink("https://example.org/A", "https://example.org/B", "rel1"),
		graph.NewLink("https://example.org/B", "https://example.org/C", "rel2"),
	}

	gm := graph.NewGraphModel(nodes, links, graph.Metadata{})

	if gm.NodeCount() != 3 {
		t.Errorf("NodeCount = %d, want 3", gm.NodeCount())
	}
	if gm.LinkCount() != 2 {
		t.Errorf("LinkCount = %d, want 2", gm.LinkCount())
	}
}
