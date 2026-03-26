package transform

import (
	"fmt"
	"sort"
	"strings"

	"github.com/IndependentImpact/ttl2d3/internal/graph"
	"github.com/IndependentImpact/ttl2d3/internal/parser"
)

// Well-known RDF / OWL / SKOS / Dublin Core predicate and type IRIs.
const (
	iriRDFType  = "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"
	iriRDFFirst = "http://www.w3.org/1999/02/22-rdf-syntax-ns#first"
	iriRDFRest  = "http://www.w3.org/1999/02/22-rdf-syntax-ns#rest"
	iriRDFNil   = "http://www.w3.org/1999/02/22-rdf-syntax-ns#nil"

	iriRDFSLabel      = "http://www.w3.org/2000/01/rdf-schema#label"
	iriRDFSComment    = "http://www.w3.org/2000/01/rdf-schema#comment"
	iriRDFSSubClassOf = "http://www.w3.org/2000/01/rdf-schema#subClassOf"
	iriRDFSDomain     = "http://www.w3.org/2000/01/rdf-schema#domain"
	iriRDFSRange      = "http://www.w3.org/2000/01/rdf-schema#range"

	iriOWLOntology         = "http://www.w3.org/2002/07/owl#Ontology"
	iriOWLClass            = "http://www.w3.org/2002/07/owl#Class"
	iriOWLObjectProperty   = "http://www.w3.org/2002/07/owl#ObjectProperty"
	iriOWLDatatypeProperty = "http://www.w3.org/2002/07/owl#DatatypeProperty"
	iriOWLAnnotationProp   = "http://www.w3.org/2002/07/owl#AnnotationProperty"
	iriOWLNamedIndividual  = "http://www.w3.org/2002/07/owl#NamedIndividual"
	iriOWLEquivalentClass  = "http://www.w3.org/2002/07/owl#equivalentClass"
	iriOWLDisjointWith     = "http://www.w3.org/2002/07/owl#disjointWith"
	iriOWLUnionOf          = "http://www.w3.org/2002/07/owl#unionOf"
	iriOWLVersionInfo      = "http://www.w3.org/2002/07/owl#versionInfo"

	iriSKOSConcept       = "http://www.w3.org/2004/02/skos/core#Concept"
	iriSKOSConceptScheme = "http://www.w3.org/2004/02/skos/core#ConceptScheme"
	iriSKOSPrefLabel     = "http://www.w3.org/2004/02/skos/core#prefLabel"
	iriSKOSBroader       = "http://www.w3.org/2004/02/skos/core#broader"
	iriSKOSNarrower      = "http://www.w3.org/2004/02/skos/core#narrower"
	iriSKOSRelated       = "http://www.w3.org/2004/02/skos/core#related"
	iriSKOSInScheme      = "http://www.w3.org/2004/02/skos/core#inScheme"
	iriSKOSHasTopConcept = "http://www.w3.org/2004/02/skos/core#hasTopConcept"
	iriSKOSTopConceptOf  = "http://www.w3.org/2004/02/skos/core#topConceptOf"

	iriDCTitle       = "http://purl.org/dc/elements/1.1/title"
	iriDCDescription = "http://purl.org/dc/elements/1.1/description"

	iriDCTermsTitle       = "http://purl.org/dc/terms/title"
	iriDCTermsDescription = "http://purl.org/dc/terms/description"
)

// labelEntry holds a candidate label value together with its source priority
// (lower is better) and language tag.  This lets us prefer rdfs:label over
// skos:prefLabel, and English-tagged literals over others.
type labelEntry struct {
	value    string
	priority int    // 0 = rdfs:label, 1 = skos:prefLabel
	lang     string // BCP-47 language tag, empty = plain literal
}

// BuildGraphModel converts an RDF triple store produced by one of the parsers
// into a [graph.GraphModel] suitable for rendering.
//
// Links are deduplicated per (source, target, predicate IRI) so that distinct
// properties with identical labels remain separate edges. Domain/range IRIs
// (including owl:unionOf lists) imply class nodes when not explicitly declared.
//
// The algorithm performs three passes over the triples:
//  1. Collect entity type declarations, candidate labels, and metadata.
//  2. Build graph nodes from declared entities.
//  3. Build graph links from structural relationships.
//
// Blank-node subjects and objects are silently ignored – they carry no stable
// IRI identity and cannot be referenced from other resources.
func BuildGraphModel(g *parser.Graph) (*graph.GraphModel, error) {
	if g == nil {
		return nil, fmt.Errorf("transform: input graph is nil")
	}

	// -----------------------------------------------------------------------
	// PASS 1 – Collect declarations, labels, and property domain/range.
	// -----------------------------------------------------------------------

	// nodeTypes maps entity IRI → NodeType for all explicitly typed entities.
	nodeTypes := make(map[string]graph.NodeType)

	// labelCandidates holds the best label candidate seen so far per IRI.
	labelCandidates := make(map[string]labelEntry)

	// metaStrings holds ontology-level metadata keyed by predicate IRI.
	metaStrings := make(map[string]string)

	// ontologyIRI is the IRI of the owl:Ontology declaration (if present).
	var ontologyIRI string

	// domainOf maps property IRI → slice of domain class IRIs.
	domainOf := make(map[string][]string)

	// rangeOf maps property IRI → slice of range class IRIs.
	rangeOf := make(map[string][]string)

	// objectProps tracks IRIs explicitly declared as owl:ObjectProperty.
	objectProps := make(map[string]struct{})

	// datatypeProps tracks IRIs explicitly declared as owl:DatatypeProperty.
	datatypeProps := make(map[string]struct{})

	// unionOfList maps blank node IDs to the list head term for owl:unionOf.
	unionOfList := make(map[string]parser.Term)

	// listFirst and listRest capture RDF collection nodes (rdf:first/rest).
	listFirst := make(map[string]parser.Term)
	listRest := make(map[string]parser.Term)

	for _, t := range g.Triples {
		predIRI := termIRI(t.Predicate)
		if predIRI == "" {
			continue // ignore triples with blank/literal predicate
		}

		// Capture RDF list / unionOf triples with blank-node subjects before
		// skipping blank nodes in the main logic.
		if t.Subject.Kind == parser.TermBlank {
			switch predIRI {
			case iriOWLUnionOf:
				unionOfList[t.Subject.Value] = t.Object
			case iriRDFFirst:
				listFirst[t.Subject.Value] = t.Object
			case iriRDFRest:
				listRest[t.Subject.Value] = t.Object
			}
			continue
		}

		subjIRI := termIRI(t.Subject)
		if subjIRI == "" {
			continue // ignore triples with blank/literal subject
		}

		switch predIRI {
		case iriRDFType:
			objIRI := termIRI(t.Object)
			if objIRI == "" {
				continue
			}
			switch objIRI {
			case iriOWLOntology:
				ontologyIRI = subjIRI
			case iriOWLClass, iriSKOSConcept, iriSKOSConceptScheme:
				nodeTypes[subjIRI] = graph.NodeTypeClass
			case iriOWLObjectProperty:
				// Object properties become edges; record separately.
				objectProps[subjIRI] = struct{}{}
				if _, exists := nodeTypes[subjIRI]; !exists {
					// Mark as property so we can build edges from domain/range.
					nodeTypes[subjIRI] = graph.NodeTypeProperty
				}
			case iriOWLDatatypeProperty, iriOWLAnnotationProp:
				if objIRI == iriOWLDatatypeProperty {
					datatypeProps[subjIRI] = struct{}{}
				}
				nodeTypes[subjIRI] = graph.NodeTypeProperty
			case iriOWLNamedIndividual:
				nodeTypes[subjIRI] = graph.NodeTypeInstance
			}

		case iriRDFSLabel:
			if t.Object.Kind == parser.TermLiteral {
				recordLabel(labelCandidates, subjIRI, t.Object.Value, t.Object.Language, 0)
			}

		case iriSKOSPrefLabel:
			if t.Object.Kind == parser.TermLiteral {
				recordLabel(labelCandidates, subjIRI, t.Object.Value, t.Object.Language, 1)
			}

		case iriRDFSDomain:
			if objIRI := termIRI(t.Object); objIRI != "" {
				domainOf[subjIRI] = appendUnique(domainOf[subjIRI], objIRI)
				continue
			}
			if t.Object.Kind == parser.TermBlank {
				for _, iri := range unionMembers(t.Object.Value, unionOfList, listFirst, listRest) {
					domainOf[subjIRI] = appendUnique(domainOf[subjIRI], iri)
				}
			}

		case iriRDFSRange:
			if objIRI := termIRI(t.Object); objIRI != "" {
				rangeOf[subjIRI] = appendUnique(rangeOf[subjIRI], objIRI)
				continue
			}
			if t.Object.Kind == parser.TermBlank {
				for _, iri := range unionMembers(t.Object.Value, unionOfList, listFirst, listRest) {
					rangeOf[subjIRI] = appendUnique(rangeOf[subjIRI], iri)
				}
			}

		// Ontology-level metadata predicates.
		// For the title we accept dc:title and dcterms:title; both are stored
		// under iriDCTitle so that buildMetadata has a single lookup key.
		// Priority is first-seen: whichever predicate appears first in the
		// triple stream wins.
		case iriOWLVersionInfo:
			if t.Object.Kind == parser.TermLiteral && metaStrings[predIRI] == "" {
				metaStrings[predIRI] = t.Object.Value
			}
		case iriDCTitle, iriDCTermsTitle:
			if t.Object.Kind == parser.TermLiteral && metaStrings[iriDCTitle] == "" {
				metaStrings[iriDCTitle] = t.Object.Value
			}
		// For the description we accept dc:description, dcterms:description,
		// and rdfs:comment as equivalent alternatives; all are stored under
		// iriDCDescription so that buildMetadata has a single lookup key.
		// Priority is first-seen.
		case iriDCDescription, iriDCTermsDescription, iriRDFSComment:
			if t.Object.Kind == parser.TermLiteral && metaStrings[iriDCDescription] == "" {
				metaStrings[iriDCDescription] = t.Object.Value
			}
		}
	}

	// Build the resolved label map (best candidate per IRI).
	labels := resolvedLabels(labelCandidates)

	// -----------------------------------------------------------------------
	// Implied class nodes from object/datatype property domains and ranges.
	// -----------------------------------------------------------------------

	ensureClass := func(iri string) {
		if iri == "" || isXMLSchemaDatatype(iri) {
			return
		}
		if _, exists := nodeTypes[iri]; !exists {
			nodeTypes[iri] = graph.NodeTypeClass
		}
	}

	for prop := range objectProps {
		for _, dom := range domainOf[prop] {
			ensureClass(dom)
		}
		for _, rng := range rangeOf[prop] {
			ensureClass(rng)
		}
	}

	for prop := range datatypeProps {
		for _, dom := range domainOf[prop] {
			ensureClass(dom)
		}
	}

	// -----------------------------------------------------------------------
	// PASS 2 – Build nodes from declared entities.
	// -----------------------------------------------------------------------

	// Separate object properties from nodes; they become edges in pass 3.
	objectPropIRIs := make(map[string]struct{})
	nodes := make([]graph.Node, 0, len(nodeTypes))

	// Collect IRIs and sort for deterministic output.
	entityIRIs := make([]string, 0, len(nodeTypes))
	for iri := range nodeTypes {
		entityIRIs = append(entityIRIs, iri)
	}
	sort.Strings(entityIRIs)

	for _, iri := range entityIRIs {
		ntype := nodeTypes[iri]

		// Object properties with both domain and range become edges.
		// If they lack domain/range they fall back to property nodes.
		if ntype == graph.NodeTypeProperty {
			// Check whether this is an ObjectProperty with domain+range.
			hasDomain := len(domainOf[iri]) > 0
			hasRange := len(rangeOf[iri]) > 0
			if hasDomain && hasRange {
				// Will be emitted as links in pass 3.
				objectPropIRIs[iri] = struct{}{}
				continue
			}
		}

		nodes = append(nodes, graph.NewNode(
			iri,
			resolveLabel(iri, labels),
			ntype,
			namespaceGroup(iri),
		))
	}

	// -----------------------------------------------------------------------
	// PASS 3 – Build links from structural / semantic relationships.
	// -----------------------------------------------------------------------

	// nodeSet is the set of node IRIs produced in pass 2 (used to validate
	// that both endpoints of a link actually exist as nodes).
	nodeSet := make(map[string]struct{}, len(nodes))
	for _, n := range nodes {
		nodeSet[n.ID] = struct{}{}
	}

	type linkKey struct {
		src  string
		tgt  string
		pred string
	}
	links := make([]graph.Link, 0)
	linkSeen := make(map[linkKey]struct{}) // dedup links by predicate IRI

	addLink := func(src, tgt, lbl, predKey string) {
		key := linkKey{src: src, tgt: tgt, pred: predKey}
		if _, dup := linkSeen[key]; dup {
			return
		}
		linkSeen[key] = struct{}{}
		links = append(links, graph.NewLink(src, tgt, lbl))
	}

	for _, t := range g.Triples {
		subjIRI, predIRI := termIRI(t.Subject), termIRI(t.Predicate)
		if subjIRI == "" || predIRI == "" {
			continue
		}

		objIRI := termIRI(t.Object)

		switch predIRI {
		case iriRDFSSubClassOf:
			if objIRI != "" {
				_, srcOK := nodeSet[subjIRI]
				_, tgtOK := nodeSet[objIRI]
				if srcOK && tgtOK {
					addLink(subjIRI, objIRI, "subClassOf", iriRDFSSubClassOf)
				}
			}

		case iriOWLEquivalentClass:
			if objIRI != "" {
				_, srcOK := nodeSet[subjIRI]
				_, tgtOK := nodeSet[objIRI]
				if srcOK && tgtOK {
					addLink(subjIRI, objIRI, "equivalentClass", iriOWLEquivalentClass)
				}
			}

		case iriOWLDisjointWith:
			if objIRI != "" {
				_, srcOK := nodeSet[subjIRI]
				_, tgtOK := nodeSet[objIRI]
				if srcOK && tgtOK {
					addLink(subjIRI, objIRI, "disjointWith", iriOWLDisjointWith)
				}
			}

		case iriSKOSBroader:
			if objIRI != "" {
				_, srcOK := nodeSet[subjIRI]
				_, tgtOK := nodeSet[objIRI]
				if srcOK && tgtOK {
					addLink(subjIRI, objIRI, "broader", iriSKOSBroader)
				}
			}

		case iriSKOSNarrower:
			if objIRI != "" {
				_, srcOK := nodeSet[subjIRI]
				_, tgtOK := nodeSet[objIRI]
				if srcOK && tgtOK {
					addLink(subjIRI, objIRI, "narrower", iriSKOSNarrower)
				}
			}

		case iriSKOSRelated:
			if objIRI != "" {
				_, srcOK := nodeSet[subjIRI]
				_, tgtOK := nodeSet[objIRI]
				if srcOK && tgtOK {
					addLink(subjIRI, objIRI, "related", iriSKOSRelated)
				}
			}

		case iriSKOSInScheme:
			if objIRI != "" {
				_, srcOK := nodeSet[subjIRI]
				_, tgtOK := nodeSet[objIRI]
				if srcOK && tgtOK {
					addLink(subjIRI, objIRI, "inScheme", iriSKOSInScheme)
				}
			}

		case iriSKOSHasTopConcept:
			if objIRI != "" {
				_, srcOK := nodeSet[subjIRI]
				_, tgtOK := nodeSet[objIRI]
				if srcOK && tgtOK {
					addLink(subjIRI, objIRI, "hasTopConcept", iriSKOSHasTopConcept)
				}
			}

		case iriSKOSTopConceptOf:
			if objIRI != "" {
				_, srcOK := nodeSet[subjIRI]
				_, tgtOK := nodeSet[objIRI]
				if srcOK && tgtOK {
					addLink(subjIRI, objIRI, "topConceptOf", iriSKOSTopConceptOf)
				}
			}
		}
	}

	// Emit edges for object properties (domain → range).
	for propIRI := range objectPropIRIs {
		propLabel := resolveLabel(propIRI, labels)
		for _, dom := range domainOf[propIRI] {
			for _, rng := range rangeOf[propIRI] {
				_, srcOK := nodeSet[dom]
				_, tgtOK := nodeSet[rng]
				if srcOK && tgtOK {
					addLink(dom, rng, propLabel, propIRI)
				}
			}
		}
	}

	// -----------------------------------------------------------------------
	// Build Metadata.
	// -----------------------------------------------------------------------
	meta := buildMetadata(ontologyIRI, g.BaseIRI, labels, metaStrings)

	gm := graph.NewGraphModel(nodes, links, meta)
	return &gm, nil
}

// -----------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------

// termIRI returns the IRI string of t if t.Kind == TermIRI, else "".
func termIRI(t parser.Term) string {
	if t.Kind == parser.TermIRI {
		return t.Value
	}
	return ""
}

// recordLabel updates labelCandidates[iri] if the new candidate is preferred
// over the current one.  Priority: lower value wins; within same priority,
// English-tagged literals win over others; within same language, first-seen wins.
func recordLabel(candidates map[string]labelEntry, iri, value, lang string, priority int) {
	cur, exists := candidates[iri]
	if !exists {
		candidates[iri] = labelEntry{value: value, priority: priority, lang: lang}
		return
	}
	// Prefer lower-priority source.
	if priority < cur.priority {
		candidates[iri] = labelEntry{value: value, priority: priority, lang: lang}
		return
	}
	if priority > cur.priority {
		return
	}
	// Same priority: prefer English over no-language or other languages.
	if lang == "en" && cur.lang != "en" {
		candidates[iri] = labelEntry{value: value, priority: priority, lang: lang}
	}
}

// resolvedLabels converts labelCandidates to a plain map[string]string.
func resolvedLabels(candidates map[string]labelEntry) map[string]string {
	out := make(map[string]string, len(candidates))
	for iri, e := range candidates {
		out[iri] = e.value
	}
	return out
}

// appendUnique appends s to slice only if it is not already present.
func appendUnique(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}

// unionMembers resolves owl:unionOf lists for the given blank-node ID,
// returning any IRI members found in order of appearance.
func unionMembers(blankID string, unionOfList map[string]parser.Term, listFirst map[string]parser.Term, listRest map[string]parser.Term) []string {
	head, ok := unionOfList[blankID]
	if !ok {
		return nil
	}

	out := make([]string, 0)
	seen := make(map[string]struct{})
	cur := head

	for {
		if cur.Kind == parser.TermIRI {
			if cur.Value == iriRDFNil {
				break
			}
			out = appendUnique(out, cur.Value)
			break
		}
		if cur.Kind != parser.TermBlank {
			break
		}
		if _, dup := seen[cur.Value]; dup {
			break
		}
		seen[cur.Value] = struct{}{}

		if first, ok := listFirst[cur.Value]; ok {
			if iri := termIRI(first); iri != "" {
				out = appendUnique(out, iri)
			}
		}

		next, ok := listRest[cur.Value]
		if !ok {
			break
		}
		cur = next
	}

	return out
}

// isXMLSchemaDatatype reports whether iri is an XML Schema datatype IRI.
func isXMLSchemaDatatype(iri string) bool {
	return strings.HasPrefix(iri, "http://www.w3.org/2001/XMLSchema#")
}

// buildMetadata assembles the Metadata struct from collected information.
func buildMetadata(ontologyIRI, baseIRI string, labels map[string]string, metaStrings map[string]string) graph.Metadata {
	// Title: prefer dc:title, then rdfs:label of the ontology IRI, then empty.
	title := metaStrings[iriDCTitle]
	if title == "" && ontologyIRI != "" {
		title = labels[ontologyIRI]
	}

	description := metaStrings[iriDCDescription]
	version := metaStrings[iriOWLVersionInfo]

	// BaseIRI: prefer the explicit owl:Ontology subject, then the parser's
	// base IRI.
	effectiveBaseIRI := ontologyIRI
	if effectiveBaseIRI == "" {
		effectiveBaseIRI = baseIRI
	}

	return graph.NewMetadata(title, description, version, effectiveBaseIRI)
}
