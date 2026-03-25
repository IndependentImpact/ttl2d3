// Package parser provides RDF format parsers for ttl2d3.
// Each parser reads a supported RDF serialisation and returns a [Graph]
// of triples that the transform layer can consume.
package parser

import "fmt"

// TermKind classifies an RDF term.
type TermKind int

const (
	// TermIRI is an absolute IRI / URI reference.
	TermIRI TermKind = iota
	// TermLiteral is an RDF literal (plain, language-tagged, or datatyped).
	TermLiteral
	// TermBlank is a blank node.
	TermBlank
)

// Term is a single node value in an RDF triple (subject, predicate, or object).
type Term struct {
	// Kind classifies the term.
	Kind TermKind
	// Value is the IRI string, literal lexical form, or blank-node identifier.
	Value string
	// Language holds the BCP-47 language tag for language-tagged literals.
	// It is non-empty only when Kind == TermLiteral.
	Language string
	// Datatype holds the datatype IRI for typed literals.
	// It is non-empty only when Kind == TermLiteral.
	Datatype string
}

// String returns an N-Triples-style representation of the term.
func (t Term) String() string {
	switch t.Kind {
	case TermIRI:
		return "<" + t.Value + ">"
	case TermBlank:
		return "_:" + t.Value
	case TermLiteral:
		s := fmt.Sprintf("%q", t.Value)
		if t.Language != "" {
			return s + "@" + t.Language
		}
		if t.Datatype != "" {
			return s + "^^<" + t.Datatype + ">"
		}
		return s
	default:
		return t.Value
	}
}

// Triple is an RDF subject–predicate–object statement.
type Triple struct {
	Subject   Term
	Predicate Term
	Object    Term
}

// Graph is an in-memory collection of RDF triples produced by a parser.
type Graph struct {
	// Triples contains all parsed statements.
	Triples []Triple
	// BaseIRI is the document base IRI supplied to the parser.
	BaseIRI string
}

// Len returns the number of triples in the graph.
func (g *Graph) Len() int {
	if g == nil {
		return 0
	}
	return len(g.Triples)
}
