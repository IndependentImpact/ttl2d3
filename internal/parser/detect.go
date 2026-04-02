package parser

import (
	"bytes"
	"errors"
	"fmt"
	"mime"
	"path/filepath"
	"strings"

	"github.com/IndependentImpact/ttl2d3/internal/config"
)

// FormatFromContentType returns the [config.InputFormat] best matching the
// supplied HTTP Content-Type header value.  MIME type parameters such as
// "charset=utf-8" are stripped before matching.  Returns [config.InputAuto]
// when the MIME type does not correspond to a known RDF serialisation.
func FormatFromContentType(contentType string) config.InputFormat {
	if contentType == "" {
		return config.InputAuto
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		// Fall back to simple trimming when the header is malformed.
		if idx := strings.IndexByte(contentType, ';'); idx >= 0 {
			mediaType = strings.TrimSpace(contentType[:idx])
		} else {
			mediaType = strings.TrimSpace(contentType)
		}
	}
	switch strings.ToLower(mediaType) {
	case "text/turtle", "application/x-turtle", "text/n3":
		return config.InputTurtle
	case "application/rdf+xml", "application/xml", "text/xml":
		return config.InputRDFXML
	case "application/ld+json", "application/json":
		return config.InputJSONLD
	default:
		return config.InputAuto
	}
}

// sniffSize is the number of bytes read from the stream for content sniffing.
const sniffSize = 512

// DetectFormat returns the [config.InputFormat] inferred from filename's
// file-extension.  It returns an error when the extension is not recognised.
func DetectFormat(filename string) (config.InputFormat, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".ttl":
		return config.InputTurtle, nil
	case ".owl", ".rdf":
		return config.InputRDFXML, nil
	case ".jsonld", ".json":
		return config.InputJSONLD, nil
	default:
		return config.InputAuto, fmt.Errorf("parser: cannot detect format: unrecognised extension %q", ext)
	}
}

// SniffFormat examines up to [sniffSize] bytes of data and returns the most
// likely [config.InputFormat].  It returns an error when no known signature is
// found in the provided data.
//
// Recognised signatures:
//
//   - JSON-LD  – first non-whitespace byte is '{' or '['
//   - RDF/XML  – content starts with "<?xml" or "<rdf:"
//   - Turtle   – content starts with '@', '#', '_:', or a bare IRI ("<http://…>")
//   - Turtle   – content starts with "PREFIX" or "BASE" (SPARQL-style keywords)
func SniffFormat(data []byte) (config.InputFormat, error) {
	// Strip UTF-8 BOM (EF BB BF) if present.
	s := data
	if bytes.HasPrefix(s, []byte{0xef, 0xbb, 0xbf}) {
		s = s[3:]
	}
	// Skip leading ASCII whitespace.
	s = bytes.TrimLeft(s, " \t\r\n")

	if len(s) == 0 {
		return config.InputAuto, errors.New("parser: cannot detect format: input is empty")
	}

	// JSON-LD is always a JSON object or array.
	if s[0] == '{' || s[0] == '[' {
		return config.InputJSONLD, nil
	}

	// RDF/XML: XML declaration or explicit rdf: root element.
	if bytes.HasPrefix(s, []byte("<?xml")) || bytes.HasPrefix(s, []byte("<rdf:")) {
		return config.InputRDFXML, nil
	}

	// Turtle: @prefix / @base directive.
	if s[0] == '@' {
		return config.InputTurtle, nil
	}

	// Turtle: comment line.
	if s[0] == '#' {
		return config.InputTurtle, nil
	}

	// Turtle: blank-node subject (_:...).
	if s[0] == '_' {
		return config.InputTurtle, nil
	}

	// '<' is ambiguous: it opens a bare IRI in Turtle ("<http://…>") or an
	// XML element in RDF/XML ("<owl:Ontology …>").  Distinguish by checking
	// whether the bytes after '<' look like an IRI scheme ("http://", etc.).
	if s[0] == '<' {
		if looksLikeTurtleIRI(s[1:]) {
			return config.InputTurtle, nil
		}
		return config.InputRDFXML, nil
	}

	// Turtle: SPARQL-style PREFIX or BASE keywords (case-insensitive).
	upper := bytes.ToUpper(s)
	if bytes.HasPrefix(upper, []byte("PREFIX")) || bytes.HasPrefix(upper, []byte("BASE")) {
		return config.InputTurtle, nil
	}

	return config.InputAuto, errors.New("parser: cannot detect format from content")
}

// iriSchemes lists the byte prefixes that indicate a Turtle bare-IRI subject
// (as opposed to an XML element name that would appear in RDF/XML).
var iriSchemes = [][]byte{
	[]byte("http://"),
	[]byte("https://"),
	[]byte("urn:"),
	[]byte("mailto:"),
	[]byte("file:"),
}

// looksLikeTurtleIRI reports whether the bytes immediately following the
// opening '<' of a term look like an absolute IRI rather than an XML element
// name.  It checks for the most common URI schemes used in ontologies.
func looksLikeTurtleIRI(after []byte) bool {
	for _, scheme := range iriSchemes {
		if bytes.HasPrefix(after, scheme) {
			return true
		}
	}
	return false
}
