package parser

import (
	"fmt"
	"io"

	gon3 "github.com/deiu/gon3"
)

// ParseTurtle reads Turtle 1.1 input from r, using baseIRI as the document
// base IRI, and returns a [Graph] of triples.  It returns a non-nil error if
// the input cannot be parsed.
func ParseTurtle(r io.Reader, baseIRI string) (*Graph, error) {
	g, err := gon3.NewParser(baseIRI).Parse(r)
	if err != nil {
		return nil, fmt.Errorf("turtle parse error: %w", err)
	}

	triples := make([]Triple, 0)
	for t := range g.IterTriples() {
		triples = append(triples, convertTriple(t))
	}

	return &Graph{BaseIRI: baseIRI, Triples: triples}, nil
}

// convertTriple converts a gon3 Triple to our internal Triple type.
func convertTriple(t *gon3.Triple) Triple {
	return Triple{
		Subject:   convertTerm(t.Subject),
		Predicate: convertTerm(t.Predicate),
		Object:    convertTerm(t.Object),
	}
}

// convertTerm converts a gon3 Term to our internal Term type.
func convertTerm(t gon3.Term) Term {
	switch v := t.(type) {
	case *gon3.IRI:
		return Term{Kind: TermIRI, Value: v.RawValue()}
	case *gon3.BlankNode:
		return Term{Kind: TermBlank, Value: v.Label}
	case *gon3.Literal:
		dt := ""
		if v.DatatypeIRI != nil {
			dt = v.DatatypeIRI.RawValue()
		}
		return Term{Kind: TermLiteral, Value: v.LexicalForm, Language: v.LanguageTag, Datatype: dt}
	default:
		return Term{Kind: TermIRI, Value: t.RawValue()}
	}
}
