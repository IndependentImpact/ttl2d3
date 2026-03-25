package parser_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/IndependentImpact/ttl2d3/internal/parser"
)

// ---------------------------------------------------------------------------
// ParseJSONLD tests
// ---------------------------------------------------------------------------

func TestParseJSONLD(t *testing.T) {
	const (
		rdfType    = "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"
		rdfLangStr = "http://www.w3.org/1999/02/22-rdf-syntax-ns#langString"
		rdfsLabel  = "http://www.w3.org/2000/01/rdf-schema#label"
		rdfsSCO    = "http://www.w3.org/2000/01/rdf-schema#subClassOf"
		rdfsDomain = "http://www.w3.org/2000/01/rdf-schema#domain"
		rdfsRange  = "http://www.w3.org/2000/01/rdf-schema#range"
		owlClass   = "http://www.w3.org/2002/07/owl#Class"
		owlOntol   = "http://www.w3.org/2002/07/owl#Ontology"
		owlObjProp = "http://www.w3.org/2002/07/owl#ObjectProperty"
		xsdString  = "http://www.w3.org/2001/XMLSchema#string"
		base       = "http://example.org/ontology"
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
			name:        "empty JSON-LD object",
			input:       `{}`,
			baseIRI:     "http://example.org/",
			wantTriples: 0,
			wantErr:     false,
		},
		{
			name:    "invalid JSON",
			input:   `{ not valid json`,
			baseIRI: "http://example.org/",
			wantErr: true,
		},
		{
			name: "single typed node",
			input: `{
  "@context": { "owl": "http://www.w3.org/2002/07/owl#" },
  "@id": "http://example.org/Thing",
  "@type": "owl:Class"
}`,
			baseIRI:     "http://example.org/",
			wantTriples: 1,
			wantErr:     false,
			wantContains: []tripleSpec{
				{
					"http://example.org/Thing",
					rdfType,
					owlClass,
				},
			},
		},
		{
			name: "plain string literal gets xsd:string datatype",
			input: `{
  "@context": { "rdfs": "http://www.w3.org/2000/01/rdf-schema#" },
  "@id": "http://example.org/Animal",
  "rdfs:label": "Animal"
}`,
			baseIRI:     "http://example.org/",
			wantTriples: 1,
			wantErr:     false,
			wantLiterals: []literalSpec{
				{
					subject:   "http://example.org/Animal",
					predicate: rdfsLabel,
					value:     "Animal",
					language:  "",
					datatype:  xsdString,
				},
			},
		},
		{
			name: "language-tagged literal",
			input: `{
  "@context": {
    "owl":  "http://www.w3.org/2002/07/owl#",
    "rdfs": "http://www.w3.org/2000/01/rdf-schema#"
  },
  "@id":   "http://example.org/Animal",
  "@type": "owl:Class",
  "rdfs:label": { "@language": "en", "@value": "Animal" }
}`,
			baseIRI:     "http://example.org/",
			wantTriples: 2,
			wantErr:     false,
			wantContains: []tripleSpec{
				{"http://example.org/Animal", rdfType, owlClass},
			},
			wantLiterals: []literalSpec{
				{
					subject:   "http://example.org/Animal",
					predicate: rdfsLabel,
					value:     "Animal",
					language:  "en",
					datatype:  rdfLangStr,
				},
			},
		},
		{
			name: "IRI-valued property",
			input: `{
  "@context": {
    "owl":  "http://www.w3.org/2002/07/owl#",
    "rdfs": "http://www.w3.org/2000/01/rdf-schema#"
  },
  "@id":   "http://example.org/Mammal",
  "@type": "owl:Class",
  "rdfs:subClassOf": { "@id": "http://example.org/Animal" }
}`,
			baseIRI:     "http://example.org/",
			wantTriples: 2,
			wantErr:     false,
			wantContains: []tripleSpec{
				{"http://example.org/Mammal", rdfType, owlClass},
				{"http://example.org/Mammal", rdfsSCO, "http://example.org/Animal"},
			},
		},
		{
			name:        "example JSON-LD ontology file",
			input:       mustReadFile(t, filepath.Join("..", "..", "testdata", "example.jsonld")),
			baseIRI:     base,
			wantTriples: 10,
			wantErr:     false,
			wantContains: []tripleSpec{
				// Ontology declaration
				{base, rdfType, owlOntol},
				// Animal is an OWL class
				{base + "#Animal", rdfType, owlClass},
				// Mammal is an OWL class
				{base + "#Mammal", rdfType, owlClass},
				// Mammal subClassOf Animal
				{base + "#Mammal", rdfsSCO, base + "#Animal"},
				// hasParent is an ObjectProperty
				{base + "#hasParent", rdfType, owlObjProp},
				// hasParent domain Animal
				{base + "#hasParent", rdfsDomain, base + "#Animal"},
				// hasParent range Animal
				{base + "#hasParent", rdfsRange, base + "#Animal"},
			},
			wantLiterals: []literalSpec{
				{base, rdfsLabel, "Example Ontology", "en", rdfLangStr},
				{base + "#Animal", rdfsLabel, "Animal", "en", rdfLangStr},
				{base + "#Mammal", rdfsLabel, "Mammal", "en", rdfLangStr},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g, err := parser.ParseJSONLD(strings.NewReader(tc.input), tc.baseIRI)

			if (err != nil) != tc.wantErr {
				t.Fatalf("ParseJSONLD() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			if g.Len() != tc.wantTriples {
				t.Errorf("ParseJSONLD() triple count = %d, want %d", g.Len(), tc.wantTriples)
			}
			if g.BaseIRI != tc.baseIRI {
				t.Errorf("ParseJSONLD() BaseIRI = %q, want %q", g.BaseIRI, tc.baseIRI)
			}

			for _, spec := range tc.wantContains {
				if !containsIRITriple(g, spec.subject, spec.predicate, spec.object) {
					t.Errorf("ParseJSONLD() missing triple <%s> <%s> <%s>",
						spec.subject, spec.predicate, spec.object)
				}
			}

			for _, spec := range tc.wantLiterals {
				if !containsLiteralTriple(g, spec.subject, spec.predicate, spec.value, spec.language, spec.datatype) {
					t.Errorf("ParseJSONLD() missing literal triple <%s> <%s> %q lang=%q dt=%q",
						spec.subject, spec.predicate, spec.value, spec.language, spec.datatype)
				}
			}
		})
	}
}
