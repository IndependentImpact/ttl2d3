package parser_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IndependentImpact/ttl2d3/internal/parser"
)

// ---------------------------------------------------------------------------
// Term.String tests
// ---------------------------------------------------------------------------

func TestTermString(t *testing.T) {
	tests := []struct {
		name string
		term parser.Term
		want string
	}{
		{
			name: "IRI",
			term: parser.Term{Kind: parser.TermIRI, Value: "http://example.org/Thing"},
			want: "<http://example.org/Thing>",
		},
		{
			name: "blank node",
			term: parser.Term{Kind: parser.TermBlank, Value: "b0"},
			want: "_:b0",
		},
		{
			name: "plain literal",
			term: parser.Term{Kind: parser.TermLiteral, Value: "hello"},
			want: `"hello"`,
		},
		{
			name: "language-tagged literal",
			term: parser.Term{Kind: parser.TermLiteral, Value: "hello", Language: "en"},
			want: `"hello"@en`,
		},
		{
			name: "typed literal",
			term: parser.Term{
				Kind:     parser.TermLiteral,
				Value:    "42",
				Datatype: "http://www.w3.org/2001/XMLSchema#integer",
			},
			want: `"42"^^<http://www.w3.org/2001/XMLSchema#integer>`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.term.String()
			if got != tc.want {
				t.Errorf("Term.String() = %q, want %q", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Graph.Len tests
// ---------------------------------------------------------------------------

func TestGraphLen(t *testing.T) {
	g := &parser.Graph{}
	if g.Len() != 0 {
		t.Errorf("empty Graph.Len() = %d, want 0", g.Len())
	}

	g.Triples = []parser.Triple{
		{
			Subject:   parser.Term{Kind: parser.TermIRI, Value: "http://example.org/A"},
			Predicate: parser.Term{Kind: parser.TermIRI, Value: "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"},
			Object:    parser.Term{Kind: parser.TermIRI, Value: "http://example.org/Class"},
		},
	}
	if g.Len() != 1 {
		t.Errorf("Graph.Len() = %d, want 1", g.Len())
	}
}

// ---------------------------------------------------------------------------
// ParseTurtle tests
// ---------------------------------------------------------------------------

func TestParseTurtle(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		baseIRI     string
		wantTriples int
		wantErr     bool
		// wantContains lists subject+predicate+object IRI triples that must exist.
		wantContains []tripleSpec
	}{
		{
			name:        "empty document",
			input:       "",
			baseIRI:     "http://example.org/",
			wantTriples: 0,
			wantErr:     false,
		},
		{
			name:        "single triple",
			input:       `<http://example.org/A> a <http://example.org/B> .`,
			baseIRI:     "http://example.org/",
			wantTriples: 1,
			wantErr:     false,
			wantContains: []tripleSpec{
				{
					"http://example.org/A",
					"http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
					"http://example.org/B",
				},
			},
		},
		{
			name:        "literal object",
			input:       `<http://example.org/A> <http://www.w3.org/2000/01/rdf-schema#label> "Thing" .`,
			baseIRI:     "http://example.org/",
			wantTriples: 1,
			wantErr:     false,
		},
		{
			name:        "language-tagged literal",
			input:       `<http://example.org/A> <http://www.w3.org/2004/02/skos/core#prefLabel> "Thing"@en .`,
			baseIRI:     "http://example.org/",
			wantTriples: 1,
			wantErr:     false,
		},
		{
			name:    "invalid turtle syntax",
			input:   `this is not valid turtle !!!`,
			baseIRI: "http://example.org/",
			wantErr: true,
		},
		{
			// Regression test: prefix declarations without whitespace between
			// the prefix name colon and the IRI reference must parse correctly.
			// Example: @prefix ex:<http://example.org/> (no space before <).
			name: "prefix without space before IRI",
			input: `@prefix ex:<http://example.org/> .
ex:Thing a <http://www.w3.org/2002/07/owl#Class> .`,
			baseIRI:     "http://example.org/",
			wantTriples: 1,
			wantErr:     false,
			wantContains: []tripleSpec{
				{
					"http://example.org/Thing",
					"http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
					"http://www.w3.org/2002/07/owl#Class",
				},
			},
		},
		{
			// Regression test: the 'a' keyword (rdf:type shorthand) must be
			// recognised even when the object IRI immediately follows without
			// any intervening whitespace (e.g. a<owl:Class>).
			name: "a keyword without space before IRI object",
			input: `@prefix owl: <http://www.w3.org/2002/07/owl#> .
<http://example.org/Thing> a<http://www.w3.org/2002/07/owl#Class> .`,
			baseIRI:     "http://example.org/",
			wantTriples: 1,
			wantErr:     false,
			wantContains: []tripleSpec{
				{
					"http://example.org/Thing",
					"http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
					"http://www.w3.org/2002/07/owl#Class",
				},
			},
		},
		{
			// Regression test: the 'a' keyword must be recognised when the
			// object is a blank node property list that immediately follows
			// without whitespace (e.g. a[...]).
			name: "a keyword without space before blank node object",
			input: `@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
<http://example.org/Thing> a[a owl:Class ; rdfs:label "Thing"] .`,
			baseIRI:     "http://example.org/",
			wantTriples: 3,
			wantErr:     false,
		},
		{
			name:        "simple OWL ontology file",
			input:       mustReadFile(t, filepath.Join("..", "..", "testdata", "simple.ttl")),
			baseIRI:     "http://example.org/ontology",
			wantTriples: 21,
			wantErr:     false,
			wantContains: []tripleSpec{
				// Ontology declaration
				{
					"http://example.org/ontology",
					"http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
					"http://www.w3.org/2002/07/owl#Ontology",
				},
				// Animal is an OWL class
				{
					"http://example.org/ontology#Animal",
					"http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
					"http://www.w3.org/2002/07/owl#Class",
				},
				// Vertebrate subClassOf Animal
				{
					"http://example.org/ontology#Vertebrate",
					"http://www.w3.org/2000/01/rdf-schema#subClassOf",
					"http://example.org/ontology#Animal",
				},
				// Mammal subClassOf Vertebrate
				{
					"http://example.org/ontology#Mammal",
					"http://www.w3.org/2000/01/rdf-schema#subClassOf",
					"http://example.org/ontology#Vertebrate",
				},
				// Bird subClassOf Vertebrate
				{
					"http://example.org/ontology#Bird",
					"http://www.w3.org/2000/01/rdf-schema#subClassOf",
					"http://example.org/ontology#Vertebrate",
				},
				// Fish subClassOf Vertebrate
				{
					"http://example.org/ontology#Fish",
					"http://www.w3.org/2000/01/rdf-schema#subClassOf",
					"http://example.org/ontology#Vertebrate",
				},
				// hasParent domain Animal
				{
					"http://example.org/ontology#hasParent",
					"http://www.w3.org/2000/01/rdf-schema#domain",
					"http://example.org/ontology#Animal",
				},
				// hasParent range Animal
				{
					"http://example.org/ontology#hasParent",
					"http://www.w3.org/2000/01/rdf-schema#range",
					"http://example.org/ontology#Animal",
				},
			},
		},
		{
			name:        "SKOS concept scheme file",
			input:       mustReadFile(t, filepath.Join("..", "..", "testdata", "skos.ttl")),
			baseIRI:     "http://example.org/colours",
			wantTriples: 23,
			wantErr:     false,
			wantContains: []tripleSpec{
				// ConceptScheme declaration
				{
					"http://example.org/colours#ColourScheme",
					"http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
					"http://www.w3.org/2004/02/skos/core#ConceptScheme",
				},
				// Red broader PrimaryColour
				{
					"http://example.org/colours#Red",
					"http://www.w3.org/2004/02/skos/core#broader",
					"http://example.org/colours#PrimaryColour",
				},
				// Blue broader PrimaryColour
				{
					"http://example.org/colours#Blue",
					"http://www.w3.org/2004/02/skos/core#broader",
					"http://example.org/colours#PrimaryColour",
				},
				// PrimaryColour broader Colour
				{
					"http://example.org/colours#PrimaryColour",
					"http://www.w3.org/2004/02/skos/core#broader",
					"http://example.org/colours#Colour",
				},
				// SecondaryColour broader Colour
				{
					"http://example.org/colours#SecondaryColour",
					"http://www.w3.org/2004/02/skos/core#broader",
					"http://example.org/colours#Colour",
				},
				// Colour is topConceptOf ColourScheme
				{
					"http://example.org/colours#Colour",
					"http://www.w3.org/2004/02/skos/core#topConceptOf",
					"http://example.org/colours#ColourScheme",
				},
			},
		},
		{
			// Regression test: local names that contain a slash character
			// (e.g. rep:domain/GENERAL) must be parsed without error.  This is
			// a common real-world pattern (path-style IRI local parts) that the
			// Turtle 1.1 spec requires to be escaped as '\/' but many
			// ontologies use unescaped.  The parser accepts them leniently.
			name: "slash in local name",
			input: `@prefix rep: <https://independentimpact.org/ns/reputation#> .
@prefix skos: <http://www.w3.org/2004/02/skos/core#> .
rep:domain/GENERAL a skos:Concept .`,
			baseIRI:     "https://independentimpact.org/ns/reputation",
			wantTriples: 1,
			wantErr:     false,
			wantContains: []tripleSpec{
				{
					"https://independentimpact.org/ns/reputation#domain/GENERAL",
					"http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
					"http://www.w3.org/2004/02/skos/core#Concept",
				},
			},
		},
		{
			// Regression test: multiple slashes and deeper path segments in
			// local names must also be handled correctly.
			name: "multiple slashes in local name",
			input: `@prefix ex: <http://example.org/> .
@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
ex:a/b/c rdf:type ex:Thing .`,
			baseIRI:     "http://example.org/",
			wantTriples: 1,
			wantErr:     false,
			wantContains: []tripleSpec{
				{
					"http://example.org/a/b/c",
					"http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
					"http://example.org/Thing",
				},
			},
		},
		{
			name:        "reputation vocabulary file",
			input:       mustReadFile(t, filepath.Join("..", "..", "testdata", "reputation-vocabulary.ttl")),
			baseIRI:     "https://independentimpact.org/ns/reputation",
			wantTriples: 15,
			wantErr:     false,
			wantContains: []tripleSpec{
				// ConceptScheme declarations
				{
					"https://independentimpact.org/ns/reputation#ReputationDomainScheme",
					"http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
					"http://www.w3.org/2004/02/skos/core#ConceptScheme",
				},
				// Concept with slash local name
				{
					"https://independentimpact.org/ns/reputation#domain/GENERAL",
					"http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
					"http://www.w3.org/2004/02/skos/core#Concept",
				},
				// inScheme link
				{
					"https://independentimpact.org/ns/reputation#domain/GENERAL",
					"http://www.w3.org/2004/02/skos/core#inScheme",
					"https://independentimpact.org/ns/reputation#ReputationDomainScheme",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g, err := parser.ParseTurtle(strings.NewReader(tc.input), tc.baseIRI)

			if (err != nil) != tc.wantErr {
				t.Fatalf("ParseTurtle() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			if g.Len() != tc.wantTriples {
				t.Errorf("ParseTurtle() triple count = %d, want %d", g.Len(), tc.wantTriples)
			}
			if g.BaseIRI != tc.baseIRI {
				t.Errorf("ParseTurtle() BaseIRI = %q, want %q", g.BaseIRI, tc.baseIRI)
			}

			for _, spec := range tc.wantContains {
				if !containsIRITriple(g, spec.subject, spec.predicate, spec.object) {
					t.Errorf("ParseTurtle() missing triple <%s> <%s> <%s>",
						spec.subject, spec.predicate, spec.object)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// tripleSpec identifies an IRI-only subject–predicate–object triple by IRI strings.
type tripleSpec struct {
	subject, predicate, object string
}

// containsIRITriple reports whether g contains a triple whose subject,
// predicate and object are all IRIs with the given values.
func containsIRITriple(g *parser.Graph, subject, predicate, object string) bool {
	for _, t := range g.Triples {
		if t.Subject.Kind == parser.TermIRI && t.Subject.Value == subject &&
			t.Predicate.Kind == parser.TermIRI && t.Predicate.Value == predicate &&
			t.Object.Kind == parser.TermIRI && t.Object.Value == object {
			return true
		}
	}
	return false
}

// mustReadFile reads a file and returns its content as a string.
// It calls t.Fatal on error.
func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("mustReadFile(%q): %v", path, err)
	}
	return string(b)
}
