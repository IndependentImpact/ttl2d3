// Package render provides serializers that convert a GraphModel into output
// formats suitable for D3.js visualisations.
package render

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/IndependentImpact/ttl2d3/internal/graph"
)

// jsonNode is the JSON representation of a graph node conforming to Appendix A
// of spec.md.
type jsonNode struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Type  string `json:"type"`
	Group string `json:"group,omitempty"`
}

// jsonLink is the JSON representation of a graph link conforming to Appendix A
// of spec.md.
type jsonLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label,omitempty"`
}

// jsonMetadata is the JSON representation of graph metadata conforming to
// Appendix A of spec.md.
type jsonMetadata struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
	BaseIRI     string `json:"baseIRI,omitempty"`
}

// jsonGraph is the top-level JSON document structure conforming to the ttl2d3
// JSON schema defined in Appendix A of spec.md.
type jsonGraph struct {
	Nodes    []jsonNode   `json:"nodes"`
	Links    []jsonLink   `json:"links"`
	Metadata jsonMetadata `json:"metadata"`
}

// RenderJSON serialises gm as a UTF-8 JSON document and writes it to w.
// The output conforms to the ttl2d3 JSON schema defined in Appendix A of
// spec.md: a top-level object with "nodes", "links", and "metadata" keys.
// nodes and links are always written as JSON arrays (never null) even when
// empty, which satisfies requirement OJ-01.
func RenderJSON(gm *graph.GraphModel, w io.Writer) error {
	if gm == nil {
		return errors.New("render: GraphModel is nil")
	}

	nodes := make([]jsonNode, len(gm.Nodes))
	for i, n := range gm.Nodes {
		nodes[i] = jsonNode{
			ID:    n.ID,
			Label: n.Label,
			Type:  string(n.Type),
			Group: n.Group,
		}
	}

	links := make([]jsonLink, len(gm.Links))
	for i, l := range gm.Links {
		links[i] = jsonLink{
			Source: l.Source,
			Target: l.Target,
			Label:  l.Label,
		}
	}

	out := jsonGraph{
		Nodes: nodes,
		Links: links,
		Metadata: jsonMetadata{
			Title:       gm.Metadata.Title,
			Description: gm.Metadata.Description,
			Version:     gm.Metadata.Version,
			BaseIRI:     gm.Metadata.BaseIRI,
		},
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
