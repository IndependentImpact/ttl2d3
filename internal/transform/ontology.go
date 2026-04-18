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
	iriOWLInverseOf        = "http://www.w3.org/2002/07/owl#inverseOf"

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

// genericObjTriple records a non-builtin IRI→IRI triple (subject, predicate,
// object) collected during Pass 1.  Triples whose subject is a known node are
// used to expand the graph: the object is promoted to an implied node and a
// labelled link is emitted between the two endpoints.
type genericObjTriple struct {
	subj, pred, obj string
}

// Options controls optional behaviours for [BuildGraphModel].
type Options struct {
	// Simplify enables simplified union rendering.  When true, owl:unionOf
	// class expressions are not represented as explicit triangle union nodes;
	// instead the originating object-property edge is repeated once for each
	// member of the union, pointing directly from the domain (or range) class
	// to each union-member class.  This produces a simpler graph that is
	// easier to read as a map of possibilities.
	Simplify bool
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
//
// An optional [Options] value may be provided to control build behaviour.
func BuildGraphModel(g *parser.Graph, opts ...Options) (*graph.GraphModel, error) {
	var opt Options
	if len(opts) > 0 {
		opt = opts[0]
	}
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

	// conceptSchemeIRI is the IRI of the first skos:ConceptScheme declaration
	// (used as metadata base IRI / title fallback when no owl:Ontology exists).
	var conceptSchemeIRI string

	// skosImplied holds IRIs that are implied to be skos:Concept nodes because
	// they appear as subject or object of a SKOS semantic relation (broader,
	// narrower, related, inScheme, topConceptOf, hasTopConcept).  These are
	// processed after the main loop by ensureClass so that concepts that are
	// never explicitly typed still appear as nodes.  A map is used to avoid
	// redundant ensureClass calls when the same IRI appears in multiple triples.
	skosImplied := make(map[string]struct{})

	// genericObjTriples collects all non-builtin IRI→IRI triples encountered in
	// Pass 1.  After the main loop, triples whose subject is a known node are
	// used to expand the graph: see the expansion step below.
	genericObjTriples := make([]genericObjTriple, 0)

	// domainOf maps property IRI → slice of domain class IRIs.
	domainOf := make(map[string][]string)

	// rangeOf maps property IRI → slice of range class IRIs.
	rangeOf := make(map[string][]string)

	// domainUnion maps property IRI → slice of blank node IDs representing union domains.
	domainUnion := make(map[string][]string)

	// rangeUnion maps property IRI → slice of blank node IDs representing union ranges.
	rangeUnion := make(map[string][]string)

	// unionBlankNodes tracks blank nodes used as owl:unionOf class expressions.
	unionBlankNodes := make(map[string]struct{})

	// objectProps tracks IRIs explicitly declared as owl:ObjectProperty.
	objectProps := make(map[string]struct{})

	// inverseOf maps a property IRI to the IRI of its owl:inverseOf property.
	// Both directions are stored: if "A owl:inverseOf B" appears, we record
	// inverseOf[A]=B (and symmetrically derive B's domain/range from A's).
	inverseOf := make(map[string]string)

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
			case iriSKOSConceptScheme:
				nodeTypes[subjIRI] = graph.NodeTypeClass
				if conceptSchemeIRI == "" {
					conceptSchemeIRI = subjIRI
				}
			case iriOWLClass, iriSKOSConcept:
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
			default:
				// Resources typed with a class URI that is not owl:Class,
				// skos:Concept, owl:ObjectProperty, owl:DatatypeProperty,
				// owl:AnnotationProperty, or owl:NamedIndividual are treated as
				// named individuals so that they appear as nodes and their
				// outgoing custom properties can be visualised.
				if _, exists := nodeTypes[subjIRI]; !exists {
					nodeTypes[subjIRI] = graph.NodeTypeInstance
				}
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
				if _, ok := unionOfList[t.Object.Value]; ok {
					domainUnion[subjIRI] = appendUnique(domainUnion[subjIRI], t.Object.Value)
					unionBlankNodes[t.Object.Value] = struct{}{}
				}
			}

		case iriRDFSRange:
			if objIRI := termIRI(t.Object); objIRI != "" {
				rangeOf[subjIRI] = appendUnique(rangeOf[subjIRI], objIRI)
				continue
			}
			if t.Object.Kind == parser.TermBlank {
				if _, ok := unionOfList[t.Object.Value]; ok {
					rangeUnion[subjIRI] = appendUnique(rangeUnion[subjIRI], t.Object.Value)
					unionBlankNodes[t.Object.Value] = struct{}{}
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

		// SKOS semantic relations: record subject/object as implied skos:Concept
		// nodes so that concepts that are never explicitly typed as skos:Concept
		// still appear in the graph.
		case iriSKOSBroader, iriSKOSNarrower, iriSKOSRelated:
			if objIRI := termIRI(t.Object); objIRI != "" {
				skosImplied[subjIRI] = struct{}{}
				skosImplied[objIRI] = struct{}{}
			}
		case iriSKOSInScheme, iriSKOSTopConceptOf:
			if subjIRI != "" {
				skosImplied[subjIRI] = struct{}{}
			}
		case iriSKOSHasTopConcept:
			if objIRI := termIRI(t.Object); objIRI != "" {
				skosImplied[objIRI] = struct{}{}
			}
		case iriOWLInverseOf:
			// Record owl:inverseOf so that we can infer domain/range for the
			// implied inverse property after the main loop.  We do NOT fall
			// through to genericObjTriples so the pair is not rendered as a
			// raw "inverseOf" edge between two property nodes.
			if objIRI := termIRI(t.Object); objIRI != "" {
				inverseOf[subjIRI] = objIRI
			}
		default:
			// Collect non-builtin IRI→IRI triples for later graph expansion.
			// All qualifying triples are stored here regardless of whether the
			// subject is currently a known node; the filtering step after the
			// main loop determines which subjects are known and promotes their
			// object IRIs to implied nodes.
			if objIRI := termIRI(t.Object); objIRI != "" {
				genericObjTriples = append(genericObjTriples, genericObjTriple{
					subj: subjIRI,
					pred: predIRI,
					obj:  objIRI,
				})
			}
		}
	}

	// Build the resolved label map (best candidate per IRI).
	labels := resolvedLabels(labelCandidates)

	unionCache := make(map[string][]string)
	getUnionMembers := func(blankID string) []string {
		if members, ok := unionCache[blankID]; ok {
			return members
		}
		members := unionMembers(blankID, unionOfList, listFirst, listRest)
		unionCache[blankID] = members
		return members
	}

	// -----------------------------------------------------------------------
	// Infer domain/range for properties implied by owl:inverseOf.
	//
	// If "A owl:inverseOf B" is present and A is a known object property, B
	// is also an object property whose domain = A's range and range = A's
	// domain (and vice-versa).  We only fill in domain/range that are not
	// already explicitly stated.
	// -----------------------------------------------------------------------

	for propA, propB := range inverseOf {
		_, aIsObjectProp := objectProps[propA]
		_, bIsObjectProp := objectProps[propB]

		if aIsObjectProp {
			// Register propB as an object property if not yet known.
			objectProps[propB] = struct{}{}
			if _, exists := nodeTypes[propB]; !exists {
				nodeTypes[propB] = graph.NodeTypeProperty
			}
			// Infer propB's domain from propA's ranges.
			if len(domainOf[propB]) == 0 && len(domainUnion[propB]) == 0 {
				for _, rng := range rangeOf[propA] {
					domainOf[propB] = appendUnique(domainOf[propB], rng)
				}
				for _, rngUnion := range rangeUnion[propA] {
					domainUnion[propB] = appendUnique(domainUnion[propB], rngUnion)
					unionBlankNodes[rngUnion] = struct{}{}
				}
			}
			// Infer propB's range from propA's domains.
			if len(rangeOf[propB]) == 0 && len(rangeUnion[propB]) == 0 {
				for _, dom := range domainOf[propA] {
					rangeOf[propB] = appendUnique(rangeOf[propB], dom)
				}
				for _, domUnion := range domainUnion[propA] {
					rangeUnion[propB] = appendUnique(rangeUnion[propB], domUnion)
					unionBlankNodes[domUnion] = struct{}{}
				}
			}
		}

		if bIsObjectProp {
			// Register propA as an object property if not yet known.
			objectProps[propA] = struct{}{}
			if _, exists := nodeTypes[propA]; !exists {
				nodeTypes[propA] = graph.NodeTypeProperty
			}
			// Infer propA's domain from propB's ranges.
			if len(domainOf[propA]) == 0 && len(domainUnion[propA]) == 0 {
				for _, rng := range rangeOf[propB] {
					domainOf[propA] = appendUnique(domainOf[propA], rng)
				}
				for _, rngUnion := range rangeUnion[propB] {
					domainUnion[propA] = appendUnique(domainUnion[propA], rngUnion)
					unionBlankNodes[rngUnion] = struct{}{}
				}
			}
			// Infer propA's range from propB's domains.
			if len(rangeOf[propA]) == 0 && len(rangeUnion[propA]) == 0 {
				for _, dom := range domainOf[propB] {
					rangeOf[propA] = appendUnique(rangeOf[propA], dom)
				}
				for _, domUnion := range domainUnion[propB] {
					rangeUnion[propA] = appendUnique(rangeUnion[propA], domUnion)
					unionBlankNodes[domUnion] = struct{}{}
				}
			}
		}
	}

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

	for blankID := range unionBlankNodes {
		for _, iri := range getUnionMembers(blankID) {
			ensureClass(iri)
		}
	}

	// Implied concept nodes from SKOS semantic relations.  Concepts referenced
	// by broader/narrower/related/inScheme/topConceptOf/hasTopConcept that were
	// never explicitly typed as skos:Concept are added here so they still
	// appear as nodes in the graph.
	for iri := range skosImplied {
		ensureClass(iri)
	}

	// Expand the graph from known nodes via non-builtin object triples.
	// For every generic IRI→IRI triple whose subject is already a known node
	// (either explicitly typed or SKOS-implied), add the object IRI as an
	// implied node so that it will appear in the visualisation.
	for _, gt := range genericObjTriples {
		_, inNodeTypes := nodeTypes[gt.subj]
		_, inSkosImplied := skosImplied[gt.subj]
		if inNodeTypes || inSkosImplied {
			ensureClass(gt.obj)
		}
	}

	unionNodeIDs := make(map[string]string, len(unionBlankNodes))
	unionNodes := make([]graph.Node, 0, len(unionBlankNodes))
	if !opt.Simplify {
		for blankID := range unionBlankNodes {
			unionID := unionNodeID(blankID)
			unionNodeIDs[blankID] = unionID
			unionNodes = append(unionNodes, graph.NewNode(unionID, "union", graph.NodeTypeUnion, "owl"))
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
			if _, isObjectProp := objectProps[iri]; isObjectProp {
				// Check whether this is an ObjectProperty with domain+range.
				hasDomain := len(domainOf[iri]) > 0 || len(domainUnion[iri]) > 0
				hasRange := len(rangeOf[iri]) > 0 || len(rangeUnion[iri]) > 0
				if hasDomain && hasRange {
					// Will be emitted as links in pass 3.
					objectPropIRIs[iri] = struct{}{}
					continue
				}
			}
		}

		nodes = append(nodes, graph.NewNode(
			iri,
			resolveLabel(iri, labels),
			ntype,
			namespaceGroup(iri),
		))
	}

	sort.Slice(unionNodes, func(i, j int) bool {
		return unionNodes[i].ID < unionNodes[j].ID
	})
	nodes = append(nodes, unionNodes...)

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

	collectEndpoints := func(direct []string, unionRefs []string) []string {
		out := make([]string, 0, len(direct)+len(unionRefs))
		for _, iri := range direct {
			if _, ok := nodeSet[iri]; ok {
				out = appendUnique(out, iri)
			}
		}
		if opt.Simplify {
			// In simplified mode, expand union references directly to their
			// member IRIs rather than routing through an intermediate union node.
			for _, blankID := range unionRefs {
				for _, memberIRI := range getUnionMembers(blankID) {
					if _, ok := nodeSet[memberIRI]; ok {
						out = appendUnique(out, memberIRI)
					}
				}
			}
		} else {
			for _, blankID := range unionRefs {
				if unionID, ok := unionNodeIDs[blankID]; ok {
					if _, ok := nodeSet[unionID]; ok {
						out = appendUnique(out, unionID)
					}
				}
			}
		}
		return out
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

	// Emit union membership edges (only in standard mode; simplified mode
	// routes edges directly from domain/range to union members instead).
	if !opt.Simplify {
		for blankID, unionID := range unionNodeIDs {
			for _, member := range getUnionMembers(blankID) {
				if _, ok := nodeSet[member]; ok {
					addLink(unionID, member, "unionOf", iriOWLUnionOf)
				}
			}
		}
	}

	// Emit edges for datatype properties (domain → property node).
	for propIRI := range datatypeProps {
		propNodeID := propIRI
		if _, ok := nodeSet[propNodeID]; !ok {
			continue
		}
		domains := collectEndpoints(domainOf[propIRI], domainUnion[propIRI])
		for _, dom := range domains {
			addLink(dom, propNodeID, "", propIRI)
		}
	}

	// Emit edges for object properties (domain → range).
	for propIRI := range objectPropIRIs {
		propLabel := resolveLabel(propIRI, labels)
		domains := collectEndpoints(domainOf[propIRI], domainUnion[propIRI])
		ranges := collectEndpoints(rangeOf[propIRI], rangeUnion[propIRI])
		for _, dom := range domains {
			for _, rng := range ranges {
				addLink(dom, rng, propLabel, propIRI)
			}
		}
	}

	// Emit links for non-builtin IRI→IRI triples where both endpoints are
	// nodes.  This covers custom object properties that link named individuals
	// or concept-scheme members to other resources.  The addLink helper
	// deduplicates by (src, tgt, predIRI) so no duplicate edges are emitted
	// even if a triple was already handled by the switch above.
	for _, gt := range genericObjTriples {
		_, srcOK := nodeSet[gt.subj]
		_, tgtOK := nodeSet[gt.obj]
		if srcOK && tgtOK {
			lbl := resolveLabel(gt.pred, labels)
			addLink(gt.subj, gt.obj, lbl, gt.pred)
		}
	}

	// -----------------------------------------------------------------------
	// Build Metadata.
	// -----------------------------------------------------------------------
	meta := buildMetadata(ontologyIRI, conceptSchemeIRI, g.BaseIRI, labels, metaStrings)

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

const unionNodePrefix = "urn:ttl2d3:union:"

func unionNodeID(blankID string) string {
	return unionNodePrefix + blankID
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
// conceptSchemeIRI is the IRI of the first skos:ConceptScheme declaration, used
// as a fallback title source and base IRI when no owl:Ontology is present.
func buildMetadata(ontologyIRI, conceptSchemeIRI, baseIRI string, labels map[string]string, metaStrings map[string]string) graph.Metadata {
	// Title priority:
	//  1. dc:title / dcterms:title
	//  2. rdfs:label of the owl:Ontology subject
	//  3. rdfs:label / skos:prefLabel of the skos:ConceptScheme subject
	title := metaStrings[iriDCTitle]
	if title == "" && ontologyIRI != "" {
		title = labels[ontologyIRI]
	}
	if title == "" && conceptSchemeIRI != "" {
		title = labels[conceptSchemeIRI]
	}

	description := metaStrings[iriDCDescription]
	version := metaStrings[iriOWLVersionInfo]

	// BaseIRI priority: owl:Ontology IRI → skos:ConceptScheme IRI → parser base IRI.
	effectiveBaseIRI := ontologyIRI
	if effectiveBaseIRI == "" {
		effectiveBaseIRI = conceptSchemeIRI
	}
	if effectiveBaseIRI == "" {
		effectiveBaseIRI = baseIRI
	}

	return graph.NewMetadata(title, description, version, effectiveBaseIRI)
}
