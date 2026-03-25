package parser_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/IndependentImpact/ttl2d3/internal/parser"
)

// ---------------------------------------------------------------------------
// ParseRDFXML tests
// ---------------------------------------------------------------------------

func TestParseRDFXML(t *testing.T) {
	const (
		rdfType    = "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"
		rdfsLabel  = "http://www.w3.org/2000/01/rdf-schema#label"
		rdfsSCO    = "http://www.w3.org/2000/01/rdf-schema#subClassOf"
		rdfsDomain = "http://www.w3.org/2000/01/rdf-schema#domain"
		rdfsRange  = "http://www.w3.org/2000/01/rdf-schema#range"
		owlClass   = "http://www.w3.org/2002/07/owl#Class"
		owlOntol   = "http://www.w3.org/2002/07/owl#Ontology"
		owlObjProp = "http://www.w3.org/2002/07/owl#ObjectProperty"
		pizza      = "http://example.org/pizza"
	)

	tests := []struct {
		name         string
		input        string
		baseIRI      string
		wantTriples  int
		wantErr      bool
		wantContains []tripleSpec
		wantLiterals []literalSpec
	}{
		{
			name:        "empty rdf:RDF element",
			input:       `<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"/>`,
			baseIRI:     "http://example.org/",
			wantTriples: 0,
			wantErr:     false,
		},
		{
			name:    "invalid XML",
			input:   `<broken`,
			baseIRI: "http://example.org/",
			wantErr: true,
		},
		{
			name: "single typed node",
			input: `<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
         xmlns:owl="http://www.w3.org/2002/07/owl#">
  <owl:Class rdf:about="http://example.org/Thing"/>
</rdf:RDF>`,
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
			name: "rdf:resource property",
			input: `<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
         xmlns:owl="http://www.w3.org/2002/07/owl#"
         xmlns:rdfs="http://www.w3.org/2000/01/rdf-schema#">
  <owl:Class rdf:about="http://example.org/Child">
    <rdfs:subClassOf rdf:resource="http://example.org/Parent"/>
  </owl:Class>
</rdf:RDF>`,
			baseIRI:     "http://example.org/",
			wantTriples: 2,
			wantErr:     false,
			wantContains: []tripleSpec{
				{
					"http://example.org/Child",
					"http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
					"http://www.w3.org/2002/07/owl#Class",
				},
				{
					"http://example.org/Child",
					"http://www.w3.org/2000/01/rdf-schema#subClassOf",
					"http://example.org/Parent",
				},
			},
		},
		{
			name: "language-tagged literal",
			input: `<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
         xmlns:rdfs="http://www.w3.org/2000/01/rdf-schema#"
         xmlns:owl="http://www.w3.org/2002/07/owl#">
  <owl:Class rdf:about="http://example.org/Thing">
    <rdfs:label xml:lang="en">Thing</rdfs:label>
  </owl:Class>
</rdf:RDF>`,
			baseIRI:     "http://example.org/",
			wantTriples: 2,
			wantErr:     false,
			wantLiterals: []literalSpec{
				{
					subject:   "http://example.org/Thing",
					predicate: "http://www.w3.org/2000/01/rdf-schema#label",
					value:     "Thing",
					language:  "en",
				},
			},
		},
		{
			name: "typed literal (rdf:datatype)",
			input: `<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
         xmlns:owl="http://www.w3.org/2002/07/owl#"
         xmlns:ex="http://example.org/">
  <owl:Class rdf:about="http://example.org/Thing">
    <ex:count rdf:datatype="http://www.w3.org/2001/XMLSchema#integer">42</ex:count>
  </owl:Class>
</rdf:RDF>`,
			baseIRI:     "http://example.org/",
			wantTriples: 2,
			wantErr:     false,
			wantLiterals: []literalSpec{
				{
					subject:   "http://example.org/Thing",
					predicate: "http://example.org/count",
					value:     "42",
					datatype:  "http://www.w3.org/2001/XMLSchema#integer",
				},
			},
		},
		{
			name: "blank node via rdf:nodeID",
			input: `<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
         xmlns:owl="http://www.w3.org/2002/07/owl#"
         xmlns:rdfs="http://www.w3.org/2000/01/rdf-schema#">
  <owl:Class rdf:about="http://example.org/A">
    <rdfs:subClassOf rdf:nodeID="b0"/>
  </owl:Class>
  <owl:Class rdf:nodeID="b0">
    <rdfs:label xml:lang="en">Unnamed</rdfs:label>
  </owl:Class>
</rdf:RDF>`,
			baseIRI:     "http://example.org/",
			wantTriples: 4,
			wantErr:     false,
		},
		{
			name: "rdf:parseType Resource",
			input: `<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
         xmlns:ex="http://example.org/">
  <rdf:Description rdf:about="http://example.org/s">
    <ex:prop rdf:parseType="Resource">
      <ex:value>hello</ex:value>
    </ex:prop>
  </rdf:Description>
</rdf:RDF>`,
			// s ex:prop _:b (1) + _:b ex:value "hello" (2)
			baseIRI:     "http://example.org/",
			wantTriples: 2,
			wantErr:     false,
		},
		{
			name: "rdf:parseType Collection",
			input: `<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
         xmlns:owl="http://www.w3.org/2002/07/owl#"
         xmlns:ex="http://example.org/">
  <rdf:Description rdf:about="http://example.org/s">
    <ex:items rdf:parseType="Collection">
      <owl:Class rdf:about="http://example.org/A"/>
      <owl:Class rdf:about="http://example.org/B"/>
    </ex:items>
  </rdf:Description>
</rdf:RDF>`,
			baseIRI: "http://example.org/",
			// rdf:type triples for A and B (2)
			// s ex:items list-head (1)
			// list-head rdf:first A (1)
			// list-head rdf:rest  list-tail (1)
			// list-tail rdf:first B (1)
			// list-tail rdf:rest  rdf:nil (1)
			wantTriples: 7,
			wantErr:     false,
		},
		{
			name: "rdf:ID subject",
			input: `<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
         xmlns:owl="http://www.w3.org/2002/07/owl#">
  <owl:Class rdf:ID="MyClass"/>
</rdf:RDF>`,
			baseIRI:     "http://example.org/",
			wantTriples: 1,
			wantErr:     false,
			wantContains: []tripleSpec{
				{
					"http://example.org/#MyClass",
					"http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
					"http://www.w3.org/2002/07/owl#Class",
				},
			},
		},
		{
			name:        "pizza OWL ontology file",
			input:       mustReadFile(t, filepath.Join("..", "..", "testdata", "pizza.owl")),
			baseIRI:     pizza,
			wantTriples: 27,
			wantErr:     false,
			wantContains: []tripleSpec{
				// Ontology declaration
				{pizza, rdfType, owlOntol},
				// Food is an OWL class
				{pizza + "#Food", rdfType, owlClass},
				// Pizza subClassOf Food
				{pizza + "#Pizza", rdfsSCO, pizza + "#Food"},
				// MargheritaPizza subClassOf Pizza
				{pizza + "#MargheritaPizza", rdfsSCO, pizza + "#Pizza"},
				// CheeseTopping subClassOf PizzaTopping
				{pizza + "#CheeseTopping", rdfsSCO, pizza + "#PizzaTopping"},
				// hasTopping is an ObjectProperty
				{pizza + "#hasTopping", rdfType, owlObjProp},
				// hasTopping domain Pizza
				{pizza + "#hasTopping", rdfsDomain, pizza + "#Pizza"},
				// hasTopping range PizzaTopping
				{pizza + "#hasTopping", rdfsRange, pizza + "#PizzaTopping"},
				// hasBase range PizzaBase
				{pizza + "#hasBase", rdfsRange, pizza + "#PizzaBase"},
			},
			wantLiterals: []literalSpec{
				{pizza, rdfsLabel, "Pizza Ontology", "en", ""},
				{pizza + "#Pizza", rdfsLabel, "Pizza", "en", ""},
				{pizza + "#hasTopping", rdfsLabel, "has topping", "en", ""},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g, err := parser.ParseRDFXML(strings.NewReader(tc.input), tc.baseIRI)

			if (err != nil) != tc.wantErr {
				t.Fatalf("ParseRDFXML() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			if g.Len() != tc.wantTriples {
				t.Errorf("ParseRDFXML() triple count = %d, want %d", g.Len(), tc.wantTriples)
			}
			if g.BaseIRI != tc.baseIRI {
				t.Errorf("ParseRDFXML() BaseIRI = %q, want %q", g.BaseIRI, tc.baseIRI)
			}

			for _, spec := range tc.wantContains {
				if !containsIRITriple(g, spec.subject, spec.predicate, spec.object) {
					t.Errorf("ParseRDFXML() missing triple <%s> <%s> <%s>",
						spec.subject, spec.predicate, spec.object)
				}
			}

			for _, spec := range tc.wantLiterals {
				if !containsLiteralTriple(g, spec.subject, spec.predicate, spec.value, spec.language, spec.datatype) {
					t.Errorf("ParseRDFXML() missing literal triple <%s> <%s> %q lang=%q dt=%q",
						spec.subject, spec.predicate, spec.value, spec.language, spec.datatype)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// literalSpec identifies a triple whose object is a literal.
type literalSpec struct {
	subject, predicate, value, language, datatype string
}

// containsLiteralTriple reports whether g contains a triple with an IRI
// subject, an IRI predicate, and a literal object matching the given fields.
func containsLiteralTriple(g *parser.Graph, subject, predicate, value, language, datatype string) bool {
	for _, t := range g.Triples {
		if t.Subject.Kind == parser.TermIRI && t.Subject.Value == subject &&
			t.Predicate.Kind == parser.TermIRI && t.Predicate.Value == predicate &&
			t.Object.Kind == parser.TermLiteral && t.Object.Value == value &&
			t.Object.Language == language && t.Object.Datatype == datatype {
			return true
		}
	}
	return false
}
