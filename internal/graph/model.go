// Package graph defines the internal graph data model for ttl2d3.
package graph

import (
	"errors"
	"fmt"
)

// NodeType classifies a node in the graph.
type NodeType string

const (
	// NodeTypeClass represents an OWL class or SKOS concept.
	NodeTypeClass NodeType = "class"
	// NodeTypeProperty represents an OWL datatype property.
	NodeTypeProperty NodeType = "property"
	// NodeTypeInstance represents an OWL named individual or SKOS concept instance.
	NodeTypeInstance NodeType = "instance"
	// NodeTypeLiteral represents a literal value node.
	NodeTypeLiteral NodeType = "literal"
)

// Node is a vertex in the graph, representing a class, property, instance, or literal.
type Node struct {
	// ID is the IRI of the resource.
	ID string
	// Label is the human-readable name (rdfs:label, skos:prefLabel, or IRI local name).
	Label string
	// Type classifies the node as class, property, instance, or literal.
	Type NodeType
	// Group is the namespace prefix or domain group used for visual grouping.
	Group string
}

// Link is a directed edge in the graph, representing a relationship between two nodes.
type Link struct {
	// Source is the IRI of the source node.
	Source string
	// Target is the IRI of the target node.
	Target string
	// Label is the human-readable name of the relationship (property local name).
	Label string
}

// Metadata holds ontology-level information extracted from the input file.
type Metadata struct {
	// Title is the ontology or concept scheme title.
	Title string
	// Description is a human-readable description of the ontology.
	Description string
	// Version is the ontology version string (e.g. owl:versionInfo).
	Version string
	// BaseIRI is the base IRI of the ontology.
	BaseIRI string
}

// GraphModel is the central data structure passed between the transform and render stages.
type GraphModel struct {
	Nodes    []Node
	Links    []Link
	Metadata Metadata
}

// NewNode creates a Node with the given fields.
func NewNode(id, label string, nodeType NodeType, group string) Node {
	return Node{ID: id, Label: label, Type: nodeType, Group: group}
}

// NewLink creates a Link with the given source, target, and label.
func NewLink(source, target, label string) Link {
	return Link{Source: source, Target: target, Label: label}
}

// NewMetadata creates a Metadata value with the given fields.
func NewMetadata(title, description, version, baseIRI string) Metadata {
	return Metadata{Title: title, Description: description, Version: version, BaseIRI: baseIRI}
}

// NewGraphModel creates a GraphModel with the given nodes, links, and metadata.
func NewGraphModel(nodes []Node, links []Link, metadata Metadata) GraphModel {
	return GraphModel{Nodes: nodes, Links: links, Metadata: metadata}
}

// Validate checks the consistency of the GraphModel.
// It returns an error if any node has an empty ID, if node IDs are not unique,
// if any link has an empty Source or Target, or if a link references an unknown node ID.
func (g *GraphModel) Validate() error {
	if g == nil {
		return errors.New("GraphModel is nil")
	}

	nodeIDs := make(map[string]struct{}, len(g.Nodes))
	for i, n := range g.Nodes {
		if n.ID == "" {
			return fmt.Errorf("node[%d] has empty ID", i)
		}
		if _, dup := nodeIDs[n.ID]; dup {
			return fmt.Errorf("duplicate node ID %q", n.ID)
		}
		nodeIDs[n.ID] = struct{}{}
	}

	for i, l := range g.Links {
		if l.Source == "" {
			return fmt.Errorf("link[%d] has empty Source", i)
		}
		if l.Target == "" {
			return fmt.Errorf("link[%d] has empty Target", i)
		}
		if _, ok := nodeIDs[l.Source]; !ok {
			return fmt.Errorf("link[%d] source %q does not reference a known node", i, l.Source)
		}
		if _, ok := nodeIDs[l.Target]; !ok {
			return fmt.Errorf("link[%d] target %q does not reference a known node", i, l.Target)
		}
	}

	return nil
}

// NodeCount returns the number of nodes in the graph.
func (g *GraphModel) NodeCount() int { return len(g.Nodes) }

// LinkCount returns the number of links in the graph.
func (g *GraphModel) LinkCount() int { return len(g.Links) }
