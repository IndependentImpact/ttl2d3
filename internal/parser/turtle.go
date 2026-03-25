package parser

import (
	"fmt"
	"io"

	rdf2go "github.com/deiu/rdf2go"
)

// ParseTurtle reads Turtle 1.1 input from r, using baseIRI as the document
// base IRI, and returns a [Graph] of triples.  It returns a non-nil error if
// the input cannot be parsed.
func ParseTurtle(r io.Reader, baseIRI string) (*Graph, error) {
	g := rdf2go.NewGraph(baseIRI)
	if err := g.Parse(r, "text/turtle"); err != nil {
		return nil, fmt.Errorf("turtle parse error: %w", err)
	}

	out := &Graph{BaseIRI: baseIRI, Triples: make([]Triple, 0, g.Len())}
	for t := range g.IterTriples() {
		out.Triples = append(out.Triples, convertTriple(t))
	}

	return out, nil
}

// convertTriple converts an rdf2go Triple to our internal Triple type.
func convertTriple(t *rdf2go.Triple) Triple {
	return Triple{
		Subject:   convertTerm(t.Subject),
		Predicate: convertTerm(t.Predicate),
		Object:    convertTerm(t.Object),
	}
}

// convertTerm converts an rdf2go Term to our internal Term type.
func convertTerm(t rdf2go.Term) Term {
	switch v := t.(type) {
	case *rdf2go.Resource:
		return Term{Kind: TermIRI, Value: v.URI}
	case *rdf2go.BlankNode:
		return Term{Kind: TermBlank, Value: v.ID}
	case *rdf2go.Literal:
		dt := ""
		if res, ok := v.Datatype.(*rdf2go.Resource); ok {
			dt = res.URI
		}
		return Term{Kind: TermLiteral, Value: v.Value, Language: v.Language, Datatype: dt}
	default:
		return Term{Kind: TermIRI, Value: t.RawValue()}
	}
}
