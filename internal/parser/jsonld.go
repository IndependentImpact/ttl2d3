package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/piprate/json-gold/ld"
)

// ParseJSONLD reads JSON-LD input from r, using baseIRI as the document base
// IRI, and returns a [Graph] of triples.  It returns a non-nil error if the
// input cannot be parsed.
//
// All triples from all named graphs in the JSON-LD document are included in
// the returned [Graph]; graph names are discarded.
func ParseJSONLD(r io.Reader, baseIRI string) (*Graph, error) {
	var doc interface{}
	if err := json.NewDecoder(r).Decode(&doc); err != nil {
		return nil, fmt.Errorf("jsonld parse error: %w", err)
	}

	proc := ld.NewJsonLdProcessor()
	options := ld.NewJsonLdOptions(baseIRI)

	result, err := proc.ToRDF(doc, options)
	if err != nil {
		return nil, fmt.Errorf("jsonld parse error: %w", err)
	}

	dataset, ok := result.(*ld.RDFDataset)
	if !ok {
		return nil, fmt.Errorf("jsonld parse error: unexpected result type %T", result)
	}

	g := &Graph{BaseIRI: baseIRI}
	for _, quads := range dataset.Graphs {
		for _, quad := range quads {
			subj, err := convertJSONLDNode(quad.Subject)
			if err != nil {
				return nil, fmt.Errorf("jsonld parse error: subject: %w", err)
			}
			pred, err := convertJSONLDNode(quad.Predicate)
			if err != nil {
				return nil, fmt.Errorf("jsonld parse error: predicate: %w", err)
			}
			obj, err := convertJSONLDNode(quad.Object)
			if err != nil {
				return nil, fmt.Errorf("jsonld parse error: object: %w", err)
			}
			g.Triples = append(g.Triples, Triple{
				Subject:   subj,
				Predicate: pred,
				Object:    obj,
			})
		}
	}

	return g, nil
}

// convertJSONLDNode converts a json-gold [ld.Node] to our internal [Term] type.
// Blank node IDs from json-gold carry a "_:" prefix which is stripped here to
// match the convention used by the Turtle and RDF/XML parsers.
func convertJSONLDNode(n ld.Node) (Term, error) {
	switch v := n.(type) {
	case ld.IRI:
		return Term{Kind: TermIRI, Value: v.Value}, nil
	case ld.BlankNode:
		// v.Attribute holds the blank node identifier including the "_:" prefix
		// (e.g. "_:b0").  Strip the prefix to match the convention used by the
		// Turtle and RDF/XML parsers, which store only the local name.
		return Term{Kind: TermBlank, Value: strings.TrimPrefix(v.Attribute, "_:")}, nil
	case ld.Literal:
		return Term{Kind: TermLiteral, Value: v.Value, Language: v.Language, Datatype: v.Datatype}, nil
	default:
		return Term{}, fmt.Errorf("unknown node type: %T", n)
	}
}
