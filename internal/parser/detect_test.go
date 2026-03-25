package parser_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/IndependentImpact/ttl2d3/internal/config"
	"github.com/IndependentImpact/ttl2d3/internal/parser"
)

// ---------------------------------------------------------------------------
// DetectFormat tests
// ---------------------------------------------------------------------------

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     config.InputFormat
		wantErr  bool
	}{
		// Turtle extensions
		{name: "ttl lowercase", filename: "ontology.ttl", want: config.InputTurtle},
		{name: "ttl uppercase", filename: "ONTOLOGY.TTL", want: config.InputTurtle},
		{name: "ttl with path", filename: "/data/my-ontology.ttl", want: config.InputTurtle},
		// RDF/XML extensions
		{name: "owl", filename: "pizza.owl", want: config.InputRDFXML},
		{name: "rdf", filename: "schema.rdf", want: config.InputRDFXML},
		{name: "OWL uppercase", filename: "PIZZA.OWL", want: config.InputRDFXML},
		// JSON-LD extensions
		{name: "jsonld", filename: "graph.jsonld", want: config.InputJSONLD},
		{name: "json", filename: "data.json", want: config.InputJSONLD},
		{name: "JSON uppercase", filename: "DATA.JSON", want: config.InputJSONLD},
		// Unknown extensions
		{name: "no extension", filename: "myfile", wantErr: true},
		{name: "txt extension", filename: "readme.txt", wantErr: true},
		{name: "n3 extension", filename: "data.n3", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parser.DetectFormat(tc.filename)
			if (err != nil) != tc.wantErr {
				t.Fatalf("DetectFormat(%q) error = %v, wantErr %v", tc.filename, err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("DetectFormat(%q) = %q, want %q", tc.filename, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SniffFormat tests
// ---------------------------------------------------------------------------

func TestSniffFormat(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    config.InputFormat
		wantErr bool
	}{
		// JSON-LD
		{name: "JSON object", data: `{"@context": {}}`, want: config.InputJSONLD},
		{name: "JSON array", data: `[{"@id": "http://x.org/A"}]`, want: config.InputJSONLD},
		{name: "JSON with leading whitespace", data: "  \n{}", want: config.InputJSONLD},
		// RDF/XML – XML declaration
		{name: "RDF/XML with XML decl", data: `<?xml version="1.0"?><rdf:RDF/>`, want: config.InputRDFXML},
		{name: "RDF/XML no XML decl", data: `<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"/>`, want: config.InputRDFXML},
		{name: "RDF/XML leading whitespace", data: "\n<?xml version=\"1.0\"?>\n<rdf:RDF/>", want: config.InputRDFXML},
		// Turtle – @prefix / @base
		{name: "Turtle @prefix", data: `@prefix owl: <http://www.w3.org/2002/07/owl#> .`, want: config.InputTurtle},
		{name: "Turtle @base", data: `@base <http://example.org/> .`, want: config.InputTurtle},
		// Turtle – comment
		{name: "Turtle comment", data: "# An ontology\n@prefix owl: <http://www.w3.org/2002/07/owl#> .", want: config.InputTurtle},
		// Turtle – blank node subject
		{name: "Turtle blank node", data: "_:b0 a <http://example.org/Thing> .", want: config.InputTurtle},
		// Turtle – bare IRI subject
		{name: "Turtle IRI subject http", data: "<http://example.org/Ontology> a <http://www.w3.org/2002/07/owl#Ontology> .", want: config.InputTurtle},
		{name: "Turtle IRI subject https", data: "<https://example.org/Ontology> a <https://www.w3.org/2002/07/owl#Ontology> .", want: config.InputTurtle},
		{name: "Turtle IRI subject urn", data: "<urn:example:thing> a <http://www.w3.org/2002/07/owl#Class> .", want: config.InputTurtle},
		// Turtle – SPARQL-style keywords
		{name: "Turtle PREFIX keyword", data: "PREFIX owl: <http://www.w3.org/2002/07/owl#>", want: config.InputTurtle},
		{name: "Turtle BASE keyword", data: "BASE <http://example.org/>", want: config.InputTurtle},
		{name: "Turtle prefix lowercase", data: "prefix owl: <http://www.w3.org/2002/07/owl#>", want: config.InputTurtle},
		// UTF-8 BOM
		{name: "Turtle with BOM", data: "\xef\xbb\xbf@prefix owl: <http://www.w3.org/2002/07/owl#> .", want: config.InputTurtle},
		{name: "JSON with BOM", data: "\xef\xbb\xbf{\"@context\":{}}", want: config.InputJSONLD},
		// Error cases
		{name: "empty", data: "", wantErr: true},
		{name: "whitespace only", data: "   \n\t", wantErr: true},
		{name: "unrecognised content", data: "Hello, world!", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parser.SniffFormat([]byte(tc.data))
			if (err != nil) != tc.wantErr {
				t.Fatalf("SniffFormat() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("SniffFormat() = %q, want %q", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Parse dispatcher tests
// ---------------------------------------------------------------------------

func TestParse(t *testing.T) {
	const base = "http://example.org/ontology"

	tests := []struct {
		name        string
		filename    string
		format      config.InputFormat
		content     string
		wantErr     bool
		wantTriples int
	}{
		// Explicit format overrides
		{
			name:        "explicit turtle format",
			filename:    "anything.txt",
			format:      config.InputTurtle,
			content:     `@prefix owl: <http://www.w3.org/2002/07/owl#> . <http://example.org/A> a owl:Class .`,
			wantTriples: 1,
		},
		{
			name:     "explicit rdfxml format",
			filename: "anything.txt",
			format:   config.InputRDFXML,
			content: `<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
         xmlns:owl="http://www.w3.org/2002/07/owl#">
  <owl:Class rdf:about="http://example.org/Animal"/>
</rdf:RDF>`,
			wantTriples: 1,
		},
		{
			name:     "explicit jsonld format",
			filename: "anything.txt",
			format:   config.InputJSONLD,
			content: `{
  "@context": {"owl": "http://www.w3.org/2002/07/owl#"},
  "@id": "http://example.org/Thing",
  "@type": "owl:Class"
}`,
			wantTriples: 1,
		},
		// Auto-detection via file extension
		{
			name:        "auto detect .ttl",
			filename:    "ontology.ttl",
			format:      config.InputAuto,
			content:     `@prefix owl: <http://www.w3.org/2002/07/owl#> . <http://example.org/A> a owl:Class .`,
			wantTriples: 1,
		},
		{
			name:     "auto detect .owl",
			filename: "pizza.owl",
			format:   config.InputAuto,
			content: `<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
         xmlns:owl="http://www.w3.org/2002/07/owl#">
  <owl:Class rdf:about="http://example.org/Pizza"/>
</rdf:RDF>`,
			wantTriples: 1,
		},
		{
			name:     "auto detect .jsonld",
			filename: "graph.jsonld",
			format:   config.InputAuto,
			content: `{
  "@context": {"owl": "http://www.w3.org/2002/07/owl#"},
  "@id": "http://example.org/Thing",
  "@type": "owl:Class"
}`,
			wantTriples: 1,
		},
		{
			name:     "auto detect .rdf",
			filename: "schema.rdf",
			format:   config.InputAuto,
			content: `<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
         xmlns:owl="http://www.w3.org/2002/07/owl#">
  <owl:Class rdf:about="http://example.org/Schema"/>
</rdf:RDF>`,
			wantTriples: 1,
		},
		{
			name:     "auto detect .json",
			filename: "data.json",
			format:   config.InputAuto,
			content: `{
  "@context": {"owl": "http://www.w3.org/2002/07/owl#"},
  "@id": "http://example.org/Thing",
  "@type": "owl:Class"
}`,
			wantTriples: 1,
		},
		// Auto-detection via content sniffing (unknown extension / stdin)
		{
			name:        "sniff turtle from stdin",
			filename:    "-",
			format:      config.InputAuto,
			content:     `@prefix owl: <http://www.w3.org/2002/07/owl#> . <http://example.org/A> a owl:Class .`,
			wantTriples: 1,
		},
		{
			name:     "sniff rdfxml from stdin",
			filename: "-",
			format:   config.InputAuto,
			content: `<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
         xmlns:owl="http://www.w3.org/2002/07/owl#">
  <owl:Class rdf:about="http://example.org/Animal"/>
</rdf:RDF>`,
			wantTriples: 1,
		},
		{
			name:     "sniff jsonld from stdin",
			filename: "-",
			format:   config.InputAuto,
			content: `{
  "@context": {"owl": "http://www.w3.org/2002/07/owl#"},
  "@id": "http://example.org/Thing",
  "@type": "owl:Class"
}`,
			wantTriples: 1,
		},
		{
			name:        "sniff turtle from unknown extension",
			filename:    "myontology.n3",
			format:      config.InputAuto,
			content:     "# comment\n@prefix owl: <http://www.w3.org/2002/07/owl#> . <http://example.org/A> a owl:Class .",
			wantTriples: 1,
		},
		// Error cases
		{
			name:     "unknown format string",
			filename: "ontology.ttl",
			format:   config.InputFormat("ntriples"),
			wantErr:  true,
		},
		{
			name:     "stdin with unrecognisable content",
			filename: "-",
			format:   config.InputAuto,
			content:  "Hello, world!",
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g, err := parser.Parse(strings.NewReader(tc.content), tc.filename, base, tc.format)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if g.Len() != tc.wantTriples {
				t.Errorf("Parse() triple count = %d, want %d", g.Len(), tc.wantTriples)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Parse with real testdata files
// ---------------------------------------------------------------------------

func TestParseTestdata(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		wantTriples int
	}{
		{
			name:        "simple.ttl via extension",
			filename:    filepath.Join("..", "..", "testdata", "simple.ttl"),
			wantTriples: 21,
		},
		{
			name:        "skos.ttl via extension",
			filename:    filepath.Join("..", "..", "testdata", "skos.ttl"),
			wantTriples: 23,
		},
		{
			name:        "pizza.owl via extension",
			filename:    filepath.Join("..", "..", "testdata", "pizza.owl"),
			wantTriples: 27,
		},
		{
			name:        "example.jsonld via extension",
			filename:    filepath.Join("..", "..", "testdata", "example.jsonld"),
			wantTriples: 10,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			content := mustReadFile(t, tc.filename)
			g, err := parser.Parse(
				strings.NewReader(content),
				tc.filename,
				"http://example.org/test",
				config.InputAuto,
			)
			if err != nil {
				t.Fatalf("Parse() unexpected error: %v", err)
			}
			if g.Len() != tc.wantTriples {
				t.Errorf("Parse() triple count = %d, want %d", g.Len(), tc.wantTriples)
			}
		})
	}
}
