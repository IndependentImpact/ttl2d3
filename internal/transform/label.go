// Package transform converts an RDF triple store into the internal GraphModel
// that is consumed by the render layer.
package transform

import "strings"

// resolveLabel returns a human-readable label for the given IRI.
// It consults the labels map (populated from rdfs:label and skos:prefLabel
// triples) first, then falls back to the IRI local name (fragment or last
// path segment), and finally to the full IRI.
func resolveLabel(iri string, labels map[string]string) string {
	if label, ok := labels[iri]; ok && label != "" {
		return label
	}
	if name := localName(iri); name != "" {
		return name
	}
	return iri
}

// localName extracts the local name (fragment or last path segment) from an
// absolute IRI.  It returns the empty string if none can be determined.
//
// Examples:
//
//	"http://example.org/ontology#Animal"  →  "Animal"
//	"http://example.org/ontology/Animal"  →  "Animal"
//	"http://example.org/ontology#"        →  ""
func localName(iri string) string {
	if i := strings.LastIndexByte(iri, '#'); i >= 0 {
		if name := iri[i+1:]; name != "" {
			return name
		}
		// Fragment present but empty – fall through to path check.
	}
	if i := strings.LastIndexByte(iri, '/'); i >= 0 {
		if name := iri[i+1:]; name != "" {
			return name
		}
	}
	return ""
}

// namespaceGroup returns a short group identifier derived from the IRI
// namespace (the portion up to and including the last '#' or '/').
//
// Well-known namespaces are mapped to conventional prefix abbreviations;
// for all other namespaces the last path segment of the base URL is used.
func namespaceGroup(iri string) string {
	idx := strings.LastIndexByte(iri, '#')
	if idx < 0 {
		idx = strings.LastIndexByte(iri, '/')
	}
	if idx < 0 {
		return ""
	}

	// ns is the IRI up to (but not including) the '#' or '/' delimiter at idx.
	ns := iri[:idx]

	switch ns {
	case "http://www.w3.org/2002/07/owl":
		return "owl"
	case "http://www.w3.org/1999/02/22-rdf-syntax-ns":
		return "rdf"
	case "http://www.w3.org/2000/01/rdf-schema":
		return "rdfs"
	case "http://www.w3.org/2004/02/skos/core":
		return "skos"
	case "http://www.w3.org/2001/XMLSchema":
		return "xsd"
	case "http://purl.org/dc/elements/1.1":
		return "dc"
	case "http://purl.org/dc/terms":
		return "dcterms"
	}

	// Fallback: the last path segment of the namespace URL.
	if i := strings.LastIndexByte(ns, '/'); i >= 0 {
		if seg := ns[i+1:]; seg != "" {
			return seg
		}
	}
	return ns
}
