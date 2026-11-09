package transform_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/IndependentImpact/ttl2d3/internal/graph"
	"github.com/IndependentImpact/ttl2d3/internal/parser"
	"github.com/IndependentImpact/ttl2d3/internal/transform"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// parseTurtle is a test helper that parses a Turtle string and fatals on error.
func parseTurtle(t *testing.T, src, base string) *parser.Graph {
	t.Helper()
	g, err := parser.ParseTurtle(strings.NewReader(src), base)
	if err != nil {
		t.Fatalf("ParseTurtle: %v", err)
	}
	return g
}

// parseTurtleFile is a test helper that reads a Turtle file from testdata and
// parses it.
func parseTurtleFile(t *testing.T, rel string) *parser.Graph {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", rel)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile %q: %v", path, err)
	}
	g, err := parser.ParseTurtle(strings.NewReader(string(b)), "http://test/")
	if err != nil {
		t.Fatalf("ParseTurtle %q: %v", rel, err)
	}
	return g
}

// findNode returns the Node with the given ID, or nil if not found.
func findNode(nodes []graph.Node, id string) *graph.Node {
	for i := range nodes {
		if nodes[i].ID == id {
			return &nodes[i]
		}
	}
	return nil
}

// hasLink reports whether links contains a link with the given source, target,
// and label.
func hasLink(links []graph.Link, src, tgt, lbl string) bool {
	for _, l := range links {
		if l.Source == src && l.Target == tgt && l.Label == lbl {
			return true
		}
	}
	return false
}

// nodeIDs returns a sorted slice of node IDs for easy comparison.
func nodeIDs(nodes []graph.Node) []string {
	ids := make([]string, len(nodes))
	for i, n := range nodes {
		ids[i] = n.ID
	}
	sort.Strings(ids)
	return ids
}

// ---------------------------------------------------------------------------
// BuildGraphModel – nil / empty graph
// ---------------------------------------------------------------------------

func TestBuildGraphModel_NilGraph(t *testing.T) {
	_, err := transform.BuildGraphModel(nil)
	if err == nil {
		t.Fatal("expected error for nil graph, got nil")
	}
}

func TestBuildGraphModel_EmptyGraph(t *testing.T) {
	g := &parser.Graph{}
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel(empty): unexpected error: %v", err)
	}
	if gm.NodeCount() != 0 {
		t.Errorf("NodeCount = %d, want 0", gm.NodeCount())
	}
	if gm.LinkCount() != 0 {
		t.Errorf("LinkCount = %d, want 0", gm.LinkCount())
	}
}

// ---------------------------------------------------------------------------
// BuildGraphModel – simple OWL ontology (testdata/simple.ttl)
// ---------------------------------------------------------------------------

func TestBuildGraphModel_SimpleOWL(t *testing.T) {
	g := parseTurtleFile(t, "simple.ttl")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	// ── Nodes ────────────────────────────────────────────────────────────────
	// Expected: 5 OWL classes.  hasParent is an ObjectProperty with domain AND
	// range so it becomes an edge, not a node.
	const (
		iriAnimal     = "http://example.org/ontology#Animal"
		iriVertebrate = "http://example.org/ontology#Vertebrate"
		iriMammal     = "http://example.org/ontology#Mammal"
		iriBird       = "http://example.org/ontology#Bird"
		iriFish       = "http://example.org/ontology#Fish"
		iriHasParent  = "http://example.org/ontology#hasParent"
	)

	wantNodeIDs := []string{iriAnimal, iriVertebrate, iriMammal, iriBird, iriFish}
	sort.Strings(wantNodeIDs)
	gotIDs := nodeIDs(gm.Nodes)

	if strings.Join(gotIDs, ",") != strings.Join(wantNodeIDs, ",") {
		t.Errorf("node IDs =\n  %v\nwant\n  %v", gotIDs, wantNodeIDs)
	}

	// All 5 nodes must be of type class.
	for _, id := range wantNodeIDs {
		n := findNode(gm.Nodes, id)
		if n == nil {
			t.Errorf("node %q not found", id)
			continue
		}
		if n.Type != graph.NodeTypeClass {
			t.Errorf("node %q Type = %q, want %q", id, n.Type, graph.NodeTypeClass)
		}
	}

	// Node labels must come from rdfs:label.
	wantLabels := map[string]string{
		iriAnimal:     "Animal",
		iriVertebrate: "Vertebrate",
		iriMammal:     "Mammal",
		iriBird:       "Bird",
		iriFish:       "Fish",
	}
	for id, want := range wantLabels {
		n := findNode(gm.Nodes, id)
		if n == nil {
			continue // already reported above
		}
		if n.Label != want {
			t.Errorf("node %q Label = %q, want %q", id, n.Label, want)
		}
	}

	// Group should be derived from the namespace.
	for _, id := range wantNodeIDs {
		n := findNode(gm.Nodes, id)
		if n == nil {
			continue
		}
		if n.Group != "ontology" {
			t.Errorf("node %q Group = %q, want %q", id, n.Group, "ontology")
		}
	}

	// hasParent must NOT be a node (it is an ObjectProperty with domain+range).
	if findNode(gm.Nodes, iriHasParent) != nil {
		t.Errorf("hasParent should be an edge, not a node")
	}

	// ── Links ────────────────────────────────────────────────────────────────
	// Hierarchy links from rdfs:subClassOf.
	wantSubClassLinks := [][2]string{
		{iriVertebrate, iriAnimal},
		{iriMammal, iriVertebrate},
		{iriBird, iriVertebrate},
		{iriFish, iriVertebrate},
	}
	for _, pair := range wantSubClassLinks {
		if !hasLink(gm.Links, pair[0], pair[1], "subClassOf") {
			t.Errorf("missing subClassOf link %s → %s", pair[0], pair[1])
		}
	}

	// ObjectProperty edge: Animal → Animal labelled "has parent".
	if !hasLink(gm.Links, iriAnimal, iriAnimal, "has parent") {
		t.Errorf("missing objectProperty edge Animal → Animal (has parent)")
	}

	// ── Metadata ─────────────────────────────────────────────────────────────
	if gm.Metadata.Title != "Simple Example Ontology" {
		t.Errorf("Metadata.Title = %q, want %q", gm.Metadata.Title, "Simple Example Ontology")
	}
	if gm.Metadata.Version != "0.1.0" {
		t.Errorf("Metadata.Version = %q, want %q", gm.Metadata.Version, "0.1.0")
	}
	if gm.Metadata.BaseIRI != "http://example.org/ontology" {
		t.Errorf("Metadata.BaseIRI = %q, want %q", gm.Metadata.BaseIRI, "http://example.org/ontology")
	}

	// ── Validate ─────────────────────────────────────────────────────────────
	if err := gm.Validate(); err != nil {
		t.Errorf("GraphModel.Validate() = %v", err)
	}
}

// ---------------------------------------------------------------------------
// BuildGraphModel – SKOS concept scheme (testdata/skos.ttl)
// ---------------------------------------------------------------------------

func TestBuildGraphModel_SKOS(t *testing.T) {
	g := parseTurtleFile(t, "skos.ttl")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	const (
		iriColourScheme    = "http://example.org/colours#ColourScheme"
		iriColour          = "http://example.org/colours#Colour"
		iriPrimaryColour   = "http://example.org/colours#PrimaryColour"
		iriSecondaryColour = "http://example.org/colours#SecondaryColour"
		iriRed             = "http://example.org/colours#Red"
		iriBlue            = "http://example.org/colours#Blue"
	)

	// ── Nodes ────────────────────────────────────────────────────────────────
	wantNodeIDs := []string{
		iriColourScheme, iriColour, iriPrimaryColour,
		iriSecondaryColour, iriRed, iriBlue,
	}
	sort.Strings(wantNodeIDs)
	gotIDs := nodeIDs(gm.Nodes)

	if strings.Join(gotIDs, ",") != strings.Join(wantNodeIDs, ",") {
		t.Errorf("node IDs =\n  %v\nwant\n  %v", gotIDs, wantNodeIDs)
	}

	// All nodes must be of type class (skos:Concept and skos:ConceptScheme).
	for _, id := range wantNodeIDs {
		n := findNode(gm.Nodes, id)
		if n == nil {
			t.Errorf("node %q not found", id)
			continue
		}
		if n.Type != graph.NodeTypeClass {
			t.Errorf("node %q Type = %q, want %q", id, n.Type, graph.NodeTypeClass)
		}
	}

	// Labels: ColourScheme has rdfs:label "Colour Concept Scheme" which wins
	// over skos:prefLabel per the resolution priority.  The individual concepts
	// only have skos:prefLabel so those values are used.
	wantLabels := map[string]string{
		iriColourScheme:    "Colour Concept Scheme", // rdfs:label wins over skos:prefLabel
		iriColour:          "Colour",                // skos:prefLabel "Colour"@en
		iriPrimaryColour:   "Primary Colour",
		iriSecondaryColour: "Secondary Colour",
		iriRed:             "Red",
		iriBlue:            "Blue",
	}
	for id, want := range wantLabels {
		n := findNode(gm.Nodes, id)
		if n == nil {
			continue
		}
		if n.Label != want {
			t.Errorf("node %q Label = %q, want %q", id, n.Label, want)
		}
	}

	// ── Links (skos:broader) ─────────────────────────────────────────────────
	wantBroaderLinks := [][2]string{
		{iriPrimaryColour, iriColour},
		{iriSecondaryColour, iriColour},
		{iriRed, iriPrimaryColour},
		{iriBlue, iriPrimaryColour},
	}
	for _, pair := range wantBroaderLinks {
		if !hasLink(gm.Links, pair[0], pair[1], "broader") {
			t.Errorf("missing broader link %s → %s", pair[0], pair[1])
		}
	}

	// ── Validate ─────────────────────────────────────────────────────────────
	if err := gm.Validate(); err != nil {
		t.Errorf("GraphModel.Validate() = %v", err)
	}
}

// ---------------------------------------------------------------------------
// BuildGraphModel – inline Turtle snippets for targeted unit tests
// ---------------------------------------------------------------------------

func TestBuildGraphModel_LabelFallback(t *testing.T) {
	// Test label resolution priority:
	//  1. rdfs:label preferred over skos:prefLabel.
	//  2. skos:prefLabel used when no rdfs:label.
	//  3. IRI local name used when no explicit label.
	//  4. Full IRI used when local name cannot be extracted.
	const src = `
@prefix rdf:  <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix skos: <http://www.w3.org/2004/02/skos/core#> .
@prefix owl:  <http://www.w3.org/2002/07/owl#> .
@prefix ex:   <http://example.org/test#> .

ex:WithRDFSLabel a owl:Class ;
    rdfs:label "RDFS Label" ;
    skos:prefLabel "SKOS Label" .

ex:WithSKOSOnly a owl:Class ;
    skos:prefLabel "SKOS Only" .

ex:WithNoLabel a owl:Class .
`
	g := parseTurtle(t, src, "http://example.org/test")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	tests := []struct {
		iri  string
		want string
	}{
		{"http://example.org/test#WithRDFSLabel", "RDFS Label"}, // rdfs:label wins
		{"http://example.org/test#WithSKOSOnly", "SKOS Only"},   // skos:prefLabel fallback
		{"http://example.org/test#WithNoLabel", "WithNoLabel"},  // IRI local name fallback
	}

	for _, tc := range tests {
		n := findNode(gm.Nodes, tc.iri)
		if n == nil {
			t.Errorf("node %q not found", tc.iri)
			continue
		}
		if n.Label != tc.want {
			t.Errorf("node %q Label = %q, want %q", tc.iri, n.Label, tc.want)
		}
	}
}

func TestBuildGraphModel_DatatypeProperty(t *testing.T) {
	// Datatype properties become nodes of type "property" linked to their domain.
	const src = `
@prefix rdf:  <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl:  <http://www.w3.org/2002/07/owl#> .
@prefix ex:   <http://example.org/dp#> .

ex:Person a owl:Class .

ex:name a owl:DatatypeProperty ;
    rdfs:label "name" ;
    rdfs:domain ex:Person .
`
	g := parseTurtle(t, src, "http://example.org/dp")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	// ex:name must be a node of type "property".
	n := findNode(gm.Nodes, "http://example.org/dp#name")
	if n == nil {
		t.Fatal("datatype property node not found")
	}
	if n.Type != graph.NodeTypeProperty {
		t.Errorf("datatype property node Type = %q, want %q", n.Type, graph.NodeTypeProperty)
	}

	// Domain must link to the datatype property node.
	if !hasLink(gm.Links, "http://example.org/dp#Person", "http://example.org/dp#name", "") {
		t.Error("missing edge Person → name for datatype property")
	}

	// Validate graph consistency.
	if err := gm.Validate(); err != nil {
		t.Errorf("GraphModel.Validate() = %v", err)
	}
}

func TestBuildGraphModel_EquivalentAndDisjoint(t *testing.T) {
	// owl:equivalentClass and owl:disjointWith should become links.
	const src = `
@prefix rdf:  <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix owl:  <http://www.w3.org/2002/07/owl#> .
@prefix ex:   <http://example.org/eq#> .

ex:A a owl:Class .
ex:B a owl:Class .
ex:C a owl:Class .

ex:A owl:equivalentClass ex:B .
ex:A owl:disjointWith ex:C .
`
	g := parseTurtle(t, src, "http://example.org/eq")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	if !hasLink(gm.Links, "http://example.org/eq#A", "http://example.org/eq#B", "equivalentClass") {
		t.Error("missing equivalentClass link A → B")
	}
	if !hasLink(gm.Links, "http://example.org/eq#A", "http://example.org/eq#C", "disjointWith") {
		t.Error("missing disjointWith link A → C")
	}

	if err := gm.Validate(); err != nil {
		t.Errorf("GraphModel.Validate() = %v", err)
	}
}

func TestBuildGraphModel_NamedIndividual(t *testing.T) {
	// owl:NamedIndividual becomes a node of type "instance".
	const src = `
@prefix rdf:  <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix owl:  <http://www.w3.org/2002/07/owl#> .
@prefix ex:   <http://example.org/ind#> .

ex:Animal a owl:Class .
ex:Fido   a owl:NamedIndividual .
`
	g := parseTurtle(t, src, "http://example.org/ind")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	n := findNode(gm.Nodes, "http://example.org/ind#Fido")
	if n == nil {
		t.Fatal("named individual node not found")
	}
	if n.Type != graph.NodeTypeInstance {
		t.Errorf("named individual Type = %q, want %q", n.Type, graph.NodeTypeInstance)
	}

	if err := gm.Validate(); err != nil {
		t.Errorf("GraphModel.Validate() = %v", err)
	}
}

func TestBuildGraphModel_ObjectPropertyNoNodeCreated(t *testing.T) {
	// An ObjectProperty with complete domain+range must appear as a link,
	// not as a node.
	const src = `
@prefix rdf:  <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl:  <http://www.w3.org/2002/07/owl#> .
@prefix ex:   <http://example.org/op#> .

ex:A a owl:Class .
ex:B a owl:Class .

ex:linksTo a owl:ObjectProperty ;
    rdfs:domain ex:A ;
    rdfs:range  ex:B .
`
	g := parseTurtle(t, src, "http://example.org/op")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	if findNode(gm.Nodes, "http://example.org/op#linksTo") != nil {
		t.Error("objectProperty with domain+range must not be a node")
	}

	if !hasLink(gm.Links, "http://example.org/op#A", "http://example.org/op#B", "linksTo") {
		t.Error("missing objectProperty edge A → B (linksTo)")
	}

	if err := gm.Validate(); err != nil {
		t.Errorf("GraphModel.Validate() = %v", err)
	}
}

func TestBuildGraphModel_ObjectPropertyNoDomainBecomesNode(t *testing.T) {
	// An ObjectProperty without domain or range cannot become a directed edge;
	// it falls back to a node of type "property".
	const src = `
@prefix rdf:  <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix owl:  <http://www.w3.org/2002/07/owl#> .
@prefix ex:   <http://example.org/op2#> .

ex:A a owl:Class .
ex:unknownProp a owl:ObjectProperty .
`
	g := parseTurtle(t, src, "http://example.org/op2")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	n := findNode(gm.Nodes, "http://example.org/op2#unknownProp")
	if n == nil {
		t.Fatal("objectProperty without domain/range should be a property node")
	}
	if n.Type != graph.NodeTypeProperty {
		t.Errorf("node Type = %q, want %q", n.Type, graph.NodeTypeProperty)
	}
}

func TestBuildGraphModel_MetadataExtraction(t *testing.T) {
	// Verify metadata extraction from owl:Ontology + dc:title + owl:versionInfo.
	const src = `
@prefix rdf:  <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl:  <http://www.w3.org/2002/07/owl#> .
@prefix dc:   <http://purl.org/dc/elements/1.1/> .

<http://example.org/meta>
    a owl:Ontology ;
    dc:title "Meta Ontology" ;
    dc:description "A test ontology for metadata." ;
    owl:versionInfo "2.0.0" .
`
	g := parseTurtle(t, src, "http://example.org/meta")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	if gm.Metadata.Title != "Meta Ontology" {
		t.Errorf("Title = %q, want %q", gm.Metadata.Title, "Meta Ontology")
	}
	if gm.Metadata.Description != "A test ontology for metadata." {
		t.Errorf("Description = %q, want %q", gm.Metadata.Description, "A test ontology for metadata.")
	}
	if gm.Metadata.Version != "2.0.0" {
		t.Errorf("Version = %q, want %q", gm.Metadata.Version, "2.0.0")
	}
	if gm.Metadata.BaseIRI != "http://example.org/meta" {
		t.Errorf("BaseIRI = %q, want %q", gm.Metadata.BaseIRI, "http://example.org/meta")
	}
}

func TestBuildGraphModel_NoDuplicateLinks(t *testing.T) {
	// Repeated triples must not produce duplicate links.
	const src = `
@prefix rdf:  <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl:  <http://www.w3.org/2002/07/owl#> .
@prefix ex:   <http://example.org/dup#> .

ex:A a owl:Class .
ex:B a owl:Class .

ex:A rdfs:subClassOf ex:B .
ex:A rdfs:subClassOf ex:B .
`
	g := parseTurtle(t, src, "http://example.org/dup")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	count := 0
	for _, l := range gm.Links {
		if l.Source == "http://example.org/dup#A" &&
			l.Target == "http://example.org/dup#B" &&
			l.Label == "subClassOf" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("duplicate link count = %d, want 1", count)
	}
}

func TestBuildGraphModel_MultipleObjectPropertiesSameDomainRange(t *testing.T) {
	// Two object properties with identical domain and range must each produce
	// their own distinct edge (they must not overprint or be collapsed into one).
	const src = `
@prefix rdf:  <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl:  <http://www.w3.org/2002/07/owl#> .
@prefix ex:   <http://example.org/par#> .

ex:A a owl:Class .
ex:B a owl:Class .

ex:prop1 a owl:ObjectProperty ;
    rdfs:label "prop one" ;
    rdfs:domain ex:A ;
    rdfs:range  ex:B .

ex:prop2 a owl:ObjectProperty ;
    rdfs:label "prop two" ;
    rdfs:domain ex:A ;
    rdfs:range  ex:B .
`
	g := parseTurtle(t, src, "http://example.org/par")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	const (
		iriA = "http://example.org/par#A"
		iriB = "http://example.org/par#B"
	)

	// Both properties must produce edges, not nodes.
	if findNode(gm.Nodes, "http://example.org/par#prop1") != nil {
		t.Error("prop1 should be an edge, not a node")
	}
	if findNode(gm.Nodes, "http://example.org/par#prop2") != nil {
		t.Error("prop2 should be an edge, not a node")
	}

	// Both edges must be present, each with its own label.
	if !hasLink(gm.Links, iriA, iriB, "prop one") {
		t.Errorf("missing edge A → B (prop one)")
	}
	if !hasLink(gm.Links, iriA, iriB, "prop two") {
		t.Errorf("missing edge A → B (prop two)")
	}

	// Exactly two edges between A and B; no deduplication across distinct labels.
	count := 0
	for _, l := range gm.Links {
		if l.Source == iriA && l.Target == iriB {
			count++
		}
	}
	if count != 2 {
		t.Errorf("expected 2 edges A → B, got %d", count)
	}

	if err := gm.Validate(); err != nil {
		t.Errorf("GraphModel.Validate() = %v", err)
	}
}

func TestBuildGraphModel_ObjectPropertiesSameLabelSameDomainRange(t *testing.T) {
	// Object properties that share the same domain + range AND label must
	// still be emitted as distinct edges (dedup by predicate IRI, not label).
	const src = `
@prefix rdf:  <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl:  <http://www.w3.org/2002/07/owl#> .
@prefix ex:   <http://example.org/duplabel#> .

ex:A a owl:Class .
ex:B a owl:Class .

ex:prop1 a owl:ObjectProperty ;
    rdfs:label "related to" ;
    rdfs:domain ex:A ;
    rdfs:range  ex:B .

ex:prop2 a owl:ObjectProperty ;
    rdfs:label "related to" ;
    rdfs:domain ex:A ;
    rdfs:range  ex:B .
`
	g := parseTurtle(t, src, "http://example.org/duplabel")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	const (
		iriA = "http://example.org/duplabel#A"
		iriB = "http://example.org/duplabel#B"
	)

	count := 0
	for _, l := range gm.Links {
		if l.Source == iriA && l.Target == iriB && l.Label == "related to" {
			count++
		}
	}
	if count != 2 {
		t.Errorf("expected 2 edges A → B with identical labels, got %d", count)
	}

	if err := gm.Validate(); err != nil {
		t.Errorf("GraphModel.Validate() = %v", err)
	}
}

func TestBuildGraphModel_ObjectPropertyUnionDomain(t *testing.T) {
	// owl:unionOf domains should be represented as explicit union nodes.
	const src = `
@prefix rdf:  <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl:  <http://www.w3.org/2002/07/owl#> .
@prefix ex:   <http://example.org/union#> .

ex:A a owl:Class .

ex:relatedTo a owl:ObjectProperty ;
    rdfs:label "related to" ;
    rdfs:domain [ owl:unionOf ( ex:A ex:B ) ] ;
    rdfs:range ex:C .
`
	g := parseTurtle(t, src, "http://example.org/union")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	const (
		iriA = "http://example.org/union#A"
		iriB = "http://example.org/union#B"
		iriC = "http://example.org/union#C"
	)

	// Implied classes from union domain/range must exist as nodes.
	for _, iri := range []string{iriA, iriB, iriC} {
		n := findNode(gm.Nodes, iri)
		if n == nil {
			t.Fatalf("expected node %q from domain/range", iri)
		}
		if n.Type != graph.NodeTypeClass {
			t.Errorf("node %q Type = %q, want %q", iri, n.Type, graph.NodeTypeClass)
		}
	}

	unionNodes := make([]graph.Node, 0)
	for _, n := range gm.Nodes {
		if n.Type == graph.NodeTypeUnion {
			unionNodes = append(unionNodes, n)
		}
	}
	if len(unionNodes) != 1 {
		t.Fatalf("expected 1 union node, got %d", len(unionNodes))
	}
	unionID := unionNodes[0].ID

	// Union node should link to its members via unionOf.
	if !hasLink(gm.Links, unionID, iriA, "unionOf") {
		t.Error("missing unionOf edge union → A")
	}
	if !hasLink(gm.Links, unionID, iriB, "unionOf") {
		t.Error("missing unionOf edge union → B")
	}

	// Property edge should originate from the union node.
	if !hasLink(gm.Links, unionID, iriC, "related to") {
		t.Error("missing edge union → C (related to)")
	}
	if hasLink(gm.Links, iriA, iriC, "related to") || hasLink(gm.Links, iriB, iriC, "related to") {
		t.Error("unexpected direct edges from union members to range")
	}

	if err := gm.Validate(); err != nil {
		t.Errorf("GraphModel.Validate() = %v", err)
	}
}

func TestBuildGraphModel_DatatypePropertyDomainImpliedClass(t *testing.T) {
	// Datatype property domains should imply class nodes even without explicit
	// owl:Class declarations; datatype ranges should not become nodes.
	const src = `
@prefix rdf:  <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl:  <http://www.w3.org/2002/07/owl#> .
@prefix xsd:  <http://www.w3.org/2001/XMLSchema#> .
@prefix ex:   <http://example.org/dt#> .

ex:value a owl:DatatypeProperty ;
    rdfs:label "value" ;
    rdfs:domain ex:Thing ;
    rdfs:range xsd:string .
`
	g := parseTurtle(t, src, "http://example.org/dt")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	const (
		iriThing = "http://example.org/dt#Thing"
		iriXSD   = "http://www.w3.org/2001/XMLSchema#string"
	)

	n := findNode(gm.Nodes, iriThing)
	if n == nil {
		t.Fatalf("expected implied domain node %q", iriThing)
	}
	if n.Type != graph.NodeTypeClass {
		t.Errorf("node %q Type = %q, want %q", iriThing, n.Type, graph.NodeTypeClass)
	}

	if findNode(gm.Nodes, iriXSD) != nil {
		t.Errorf("datatype range %q should not become a node", iriXSD)
	}

	if err := gm.Validate(); err != nil {
		t.Errorf("GraphModel.Validate() = %v", err)
	}
}

func TestBuildGraphModel_NamespaceGroup(t *testing.T) {
	// Nodes from different namespaces get different group values.
	const src = `
@prefix rdf:  <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix owl:  <http://www.w3.org/2002/07/owl#> .
@prefix ex:   <http://example.org/ns1#> .
@prefix ex2:  <http://example.org/ns2#> .

ex:Alpha  a owl:Class .
ex2:Beta  a owl:Class .
`
	g := parseTurtle(t, src, "http://example.org/")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	alpha := findNode(gm.Nodes, "http://example.org/ns1#Alpha")
	beta := findNode(gm.Nodes, "http://example.org/ns2#Beta")

	if alpha == nil || beta == nil {
		t.Fatal("expected both nodes to be present")
	}
	if alpha.Group != "ns1" {
		t.Errorf("Alpha Group = %q, want %q", alpha.Group, "ns1")
	}
	if beta.Group != "ns2" {
		t.Errorf("Beta Group = %q, want %q", beta.Group, "ns2")
	}
}

// ---------------------------------------------------------------------------
// BuildGraphModel – SKOS inScheme / hasTopConcept / topConceptOf
// ---------------------------------------------------------------------------

func TestBuildGraphModel_SKOSInScheme(t *testing.T) {
	// skos:inScheme triples must produce "inScheme" links between a concept
	// and its concept scheme.
	const src = `
@prefix rdf:  <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix skos: <http://www.w3.org/2004/02/skos/core#> .
@prefix ex:   <http://example.org/cs#> .

ex:MyScheme a skos:ConceptScheme .

ex:ConceptA a skos:Concept ;
    skos:prefLabel "Concept A"@en ;
    skos:inScheme ex:MyScheme .

ex:ConceptB a skos:Concept ;
    skos:prefLabel "Concept B"@en ;
    skos:inScheme ex:MyScheme .
`
	g := parseTurtle(t, src, "http://example.org/cs")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	const (
		iriScheme   = "http://example.org/cs#MyScheme"
		iriConceptA = "http://example.org/cs#ConceptA"
		iriConceptB = "http://example.org/cs#ConceptB"
	)

	// Both concepts and the scheme must be nodes.
	for _, id := range []string{iriScheme, iriConceptA, iriConceptB} {
		if findNode(gm.Nodes, id) == nil {
			t.Errorf("node %q not found", id)
		}
	}

	// skos:inScheme must produce "inScheme" links from each concept to the scheme.
	if !hasLink(gm.Links, iriConceptA, iriScheme, "inScheme") {
		t.Errorf("missing inScheme link ConceptA → MyScheme")
	}
	if !hasLink(gm.Links, iriConceptB, iriScheme, "inScheme") {
		t.Errorf("missing inScheme link ConceptB → MyScheme")
	}

	if err := gm.Validate(); err != nil {
		t.Errorf("GraphModel.Validate() = %v", err)
	}
}

func TestBuildGraphModel_SKOSHasTopConcept(t *testing.T) {
	// skos:hasTopConcept must produce a "hasTopConcept" link from a scheme to
	// its top concept, and skos:topConceptOf must produce a "topConceptOf"
	// link from a concept to its scheme.
	const src = `
@prefix rdf:  <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix skos: <http://www.w3.org/2004/02/skos/core#> .
@prefix ex:   <http://example.org/tc#> .

ex:TopScheme a skos:ConceptScheme ;
    skos:hasTopConcept ex:Root .

ex:Root a skos:Concept ;
    skos:prefLabel "Root"@en ;
    skos:topConceptOf ex:TopScheme .
`
	g := parseTurtle(t, src, "http://example.org/tc")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	const (
		iriScheme = "http://example.org/tc#TopScheme"
		iriRoot   = "http://example.org/tc#Root"
	)

	// Both nodes must be present.
	if findNode(gm.Nodes, iriScheme) == nil {
		t.Errorf("node %q not found", iriScheme)
	}
	if findNode(gm.Nodes, iriRoot) == nil {
		t.Errorf("node %q not found", iriRoot)
	}

	// skos:hasTopConcept must produce a link scheme → root.
	if !hasLink(gm.Links, iriScheme, iriRoot, "hasTopConcept") {
		t.Errorf("missing hasTopConcept link TopScheme → Root")
	}
	// skos:topConceptOf must produce a link root → scheme.
	if !hasLink(gm.Links, iriRoot, iriScheme, "topConceptOf") {
		t.Errorf("missing topConceptOf link Root → TopScheme")
	}

	if err := gm.Validate(); err != nil {
		t.Errorf("GraphModel.Validate() = %v", err)
	}
}

func TestBuildGraphModel_SlashLocalName(t *testing.T) {
	// Local names containing a slash (e.g. rep:domain/GENERAL) must be parsed
	// correctly and the resulting IRI used as the node ID.
	const src = `
@prefix rep:  <https://independentimpact.org/ns/reputation#> .
@prefix skos: <http://www.w3.org/2004/02/skos/core#> .

rep:MyScheme a skos:ConceptScheme .

rep:domain/GENERAL a skos:Concept ;
    skos:prefLabel "General"@en ;
    skos:inScheme rep:MyScheme .
`
	g := parseTurtle(t, src, "https://independentimpact.org/ns/reputation")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	const (
		iriScheme  = "https://independentimpact.org/ns/reputation#MyScheme"
		iriGeneral = "https://independentimpact.org/ns/reputation#domain/GENERAL"
	)

	n := findNode(gm.Nodes, iriGeneral)
	if n == nil {
		t.Fatalf("node %q not found", iriGeneral)
	}
	if n.Label != "General" {
		t.Errorf("node %q Label = %q, want %q", iriGeneral, n.Label, "General")
	}
	if !hasLink(gm.Links, iriGeneral, iriScheme, "inScheme") {
		t.Errorf("missing inScheme link domain/GENERAL → MyScheme")
	}

	if err := gm.Validate(); err != nil {
		t.Errorf("GraphModel.Validate() = %v", err)
	}
}

// ---------------------------------------------------------------------------
// BuildGraphModel – implied skos:Concept nodes from SKOS relations
// ---------------------------------------------------------------------------

// TestBuildGraphModel_SKOSImpliedConceptsFromBroader verifies that concepts
// referenced via skos:broader/skos:narrower/skos:related are implied as nodes
// even when they are never explicitly typed as skos:Concept.
func TestBuildGraphModel_SKOSImpliedConceptsFromBroader(t *testing.T) {
	const src = `
@prefix skos: <http://www.w3.org/2004/02/skos/core#> .
@prefix cs:   <http://example.org/implied#> .

cs:Scheme a skos:ConceptScheme ;
    skos:prefLabel "My Scheme"@en .

cs:Root skos:prefLabel "Root"@en ;
    skos:topConceptOf cs:Scheme .

cs:Child skos:prefLabel "Child"@en ;
    skos:broader cs:Root .

cs:Sibling skos:prefLabel "Sibling"@en ;
    skos:broader cs:Root ;
    skos:related cs:Child .
`
	g := parseTurtle(t, src, "http://example.org/implied")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	const (
		iriScheme  = "http://example.org/implied#Scheme"
		iriRoot    = "http://example.org/implied#Root"
		iriChild   = "http://example.org/implied#Child"
		iriSibling = "http://example.org/implied#Sibling"
	)

	// All four entities must be present as nodes even though only the scheme
	// is explicitly typed.
	for _, id := range []string{iriScheme, iriRoot, iriChild, iriSibling} {
		n := findNode(gm.Nodes, id)
		if n == nil {
			t.Errorf("node %q not found (should be implied by SKOS relations)", id)
			continue
		}
		if n.Type != graph.NodeTypeClass {
			t.Errorf("node %q Type = %q, want %q", id, n.Type, graph.NodeTypeClass)
		}
	}

	// Labels from skos:prefLabel must be resolved for the implied nodes.
	wantLabels := map[string]string{
		iriScheme:  "My Scheme",
		iriRoot:    "Root",
		iriChild:   "Child",
		iriSibling: "Sibling",
	}
	for id, want := range wantLabels {
		n := findNode(gm.Nodes, id)
		if n == nil {
			continue
		}
		if n.Label != want {
			t.Errorf("node %q Label = %q, want %q", id, n.Label, want)
		}
	}

	// SKOS structural links must be present.
	if !hasLink(gm.Links, iriRoot, iriScheme, "topConceptOf") {
		t.Error("missing topConceptOf link Root → Scheme")
	}
	if !hasLink(gm.Links, iriChild, iriRoot, "broader") {
		t.Error("missing broader link Child → Root")
	}
	if !hasLink(gm.Links, iriSibling, iriRoot, "broader") {
		t.Error("missing broader link Sibling → Root")
	}
	if !hasLink(gm.Links, iriSibling, iriChild, "related") {
		t.Error("missing related link Sibling → Child")
	}

	if err := gm.Validate(); err != nil {
		t.Errorf("GraphModel.Validate() = %v", err)
	}
}

// TestBuildGraphModel_SKOSImpliedConceptsFromHasTopConcept verifies that
// concepts referenced via skos:hasTopConcept appear as nodes even when never
// explicitly typed.
func TestBuildGraphModel_SKOSImpliedConceptsFromHasTopConcept(t *testing.T) {
	const src = `
@prefix skos: <http://www.w3.org/2004/02/skos/core#> .
@prefix cs:   <http://example.org/htc#> .

cs:Scheme a skos:ConceptScheme ;
    skos:prefLabel "Decision Scheme"@en ;
    skos:hasTopConcept cs:Top .

cs:Top skos:prefLabel "Top"@en .

cs:Leaf skos:broader cs:Top ;
    skos:prefLabel "Leaf"@en .
`
	g := parseTurtle(t, src, "http://example.org/htc")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	const (
		iriScheme = "http://example.org/htc#Scheme"
		iriTop    = "http://example.org/htc#Top"
		iriLeaf   = "http://example.org/htc#Leaf"
	)

	for _, id := range []string{iriScheme, iriTop, iriLeaf} {
		if findNode(gm.Nodes, id) == nil {
			t.Errorf("node %q not found", id)
		}
	}

	if !hasLink(gm.Links, iriScheme, iriTop, "hasTopConcept") {
		t.Error("missing hasTopConcept link Scheme → Top")
	}
	if !hasLink(gm.Links, iriLeaf, iriTop, "broader") {
		t.Error("missing broader link Leaf → Top")
	}

	if err := gm.Validate(); err != nil {
		t.Errorf("GraphModel.Validate() = %v", err)
	}
}

// ---------------------------------------------------------------------------
// BuildGraphModel – SKOS-only metadata (no owl:Ontology)
// ---------------------------------------------------------------------------

// TestBuildGraphModel_SKOSMetadata verifies that when a file contains only a
// skos:ConceptScheme (no owl:Ontology), the ConceptScheme IRI and label are
// used as the metadata base IRI and title respectively.
func TestBuildGraphModel_SKOSMetadata(t *testing.T) {
	const src = `
@prefix rdf:  <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix skos: <http://www.w3.org/2004/02/skos/core#> .

<http://example.org/decisions>
    a skos:ConceptScheme ;
    rdfs:label "Reviewer decision to outcome mapping" .
`
	g := parseTurtle(t, src, "http://example.org/decisions")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	if gm.Metadata.Title != "Reviewer decision to outcome mapping" {
		t.Errorf("Metadata.Title = %q, want %q", gm.Metadata.Title, "Reviewer decision to outcome mapping")
	}
	if gm.Metadata.BaseIRI != "http://example.org/decisions" {
		t.Errorf("Metadata.BaseIRI = %q, want %q", gm.Metadata.BaseIRI, "http://example.org/decisions")
	}
}

// ---------------------------------------------------------------------------
// BuildGraphModel – custom object properties from SKOS concept scheme members
// ---------------------------------------------------------------------------

// TestBuildGraphModel_CustomObjectPropertiesFromSKOSMember verifies the fix
// for the issue "Expand on display of skos Concept scheme": resources that are
// part of a concept scheme (via skos:inScheme) and that use custom/unrecognised
// object properties to link to other concepts must have both the target
// concepts and the property links included in the graph.
//
// Example pattern from the issue:
//
// map:DecisionOutcomeMappingApprove a indimp:DecisionOutcomeMapping ;
//
//	skos:inScheme map:DecisionOutcomeMappingScheme ;
//	indimp:decision concept:Approve ;
//	indimp:mapsOutcome concept:DocOutcomeApproved .
func TestBuildGraphModel_CustomObjectPropertiesFromSKOSMember(t *testing.T) {
	const src = `
@prefix rdf:     <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs:    <http://www.w3.org/2000/01/rdf-schema#> .
@prefix skos:    <http://www.w3.org/2004/02/skos/core#> .
@prefix map:     <http://example.org/map#> .
@prefix indimp:  <http://example.org/indimp#> .
@prefix concept: <http://example.org/concept#> .

map:DecisionOutcomeMappingScheme a skos:ConceptScheme ;
    rdfs:label "Decision Outcome Mapping Scheme" .

map:DecisionOutcomeMappingApprove a indimp:DecisionOutcomeMapping ;
    skos:inScheme map:DecisionOutcomeMappingScheme ;
    indimp:decision concept:Approve ;
    indimp:mapsOutcome concept:DocOutcomeApproved .

concept:Approve a skos:Concept ;
    skos:prefLabel "Approve"@en .

concept:DocOutcomeApproved a skos:Concept ;
    skos:prefLabel "Document Outcome Approved"@en .
`
	g := parseTurtle(t, src, "http://example.org/map")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	const (
		iriScheme     = "http://example.org/map#DecisionOutcomeMappingScheme"
		iriMapping    = "http://example.org/map#DecisionOutcomeMappingApprove"
		iriApprove    = "http://example.org/concept#Approve"
		iriDocOutcome = "http://example.org/concept#DocOutcomeApproved"
		iriDecision   = "http://example.org/indimp#decision"
		iriMaps       = "http://example.org/indimp#mapsOutcome"
	)

	// All four resources must appear as nodes.
	for _, id := range []string{iriScheme, iriMapping, iriApprove, iriDocOutcome} {
		if findNode(gm.Nodes, id) == nil {
			t.Errorf("node %q not found", id)
		}
	}

	// The skos:inScheme link must still be present.
	if !hasLink(gm.Links, iriMapping, iriScheme, "inScheme") {
		t.Errorf("missing inScheme link Mapping → Scheme")
	}

	// Custom property links must be present.
	if !hasLink(gm.Links, iriMapping, iriApprove, "decision") {
		t.Errorf("missing indimp:decision link Mapping → Approve")
	}
	if !hasLink(gm.Links, iriMapping, iriDocOutcome, "mapsOutcome") {
		t.Errorf("missing indimp:mapsOutcome link Mapping → DocOutcomeApproved")
	}

	if err := gm.Validate(); err != nil {
		t.Errorf("GraphModel.Validate() = %v", err)
	}
}

// TestBuildGraphModel_CustomObjectPropertiesImplyTargetNodes verifies that
// even when the target concepts of custom object properties are NOT explicitly
// typed anywhere in the input, they are still added as implied nodes.
func TestBuildGraphModel_CustomObjectPropertiesImplyTargetNodes(t *testing.T) {
	const src = `
@prefix skos:    <http://www.w3.org/2004/02/skos/core#> .
@prefix map:     <http://example.org/map#> .
@prefix indimp:  <http://example.org/indimp#> .
@prefix concept: <http://example.org/concept#> .

map:Scheme a skos:ConceptScheme .

map:MappingA a indimp:Mapping ;
    skos:inScheme map:Scheme ;
    indimp:decision concept:Accept .
`
	g := parseTurtle(t, src, "http://example.org/map")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	const (
		iriScheme   = "http://example.org/map#Scheme"
		iriMappingA = "http://example.org/map#MappingA"
		iriAccept   = "http://example.org/concept#Accept"
	)

	// concept:Accept is never explicitly typed but must be implied as a node.
	for _, id := range []string{iriScheme, iriMappingA, iriAccept} {
		if findNode(gm.Nodes, id) == nil {
			t.Errorf("node %q not found", id)
		}
	}

	if !hasLink(gm.Links, iriMappingA, iriScheme, "inScheme") {
		t.Errorf("missing inScheme link")
	}
	if !hasLink(gm.Links, iriMappingA, iriAccept, "decision") {
		t.Errorf("missing indimp:decision link MappingA → Accept")
	}

	if err := gm.Validate(); err != nil {
		t.Errorf("GraphModel.Validate() = %v", err)
	}
}

// TestBuildGraphModel_CustomTypedInstanceCustomProperties verifies that a
// resource typed with an unrecognised class (not owl:Class, skos:Concept, etc.)
// appears as a NodeTypeInstance and its custom IRI-valued properties become
// graph edges.
func TestBuildGraphModel_CustomTypedInstanceCustomProperties(t *testing.T) {
	const src = `
@prefix skos:    <http://www.w3.org/2004/02/skos/core#> .
@prefix ex:      <http://example.org/ex#> .

ex:MyClass a ex:CustomClass ;
    ex:relatesTo ex:OtherClass .

ex:OtherClass a skos:Concept ;
    skos:prefLabel "Other"@en .
`
	g := parseTurtle(t, src, "http://example.org/ex")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	const (
		iriMyClass    = "http://example.org/ex#MyClass"
		iriOtherClass = "http://example.org/ex#OtherClass"
	)

	n := findNode(gm.Nodes, iriMyClass)
	if n == nil {
		t.Fatalf("node %q not found", iriMyClass)
	}
	if n.Type != graph.NodeTypeInstance {
		t.Errorf("node %q Type = %q, want %q", iriMyClass, n.Type, graph.NodeTypeInstance)
	}

	if findNode(gm.Nodes, iriOtherClass) == nil {
		t.Errorf("node %q not found", iriOtherClass)
	}

	if !hasLink(gm.Links, iriMyClass, iriOtherClass, "relatesTo") {
		t.Errorf("missing ex:relatesTo link MyClass → OtherClass")
	}

	if err := gm.Validate(); err != nil {
		t.Errorf("GraphModel.Validate() = %v", err)
	}
}

// ---------------------------------------------------------------------------
// BuildGraphModel – simplified union rendering (Options.Simplify)
// ---------------------------------------------------------------------------

// TestBuildGraphModel_SimplifyUnionDomain verifies that when Options.Simplify
// is true, owl:unionOf domain class expressions produce direct edges from each
// union member to the range rather than routing through a triangle union node.
func TestBuildGraphModel_SimplifyUnionDomain(t *testing.T) {
	const src = `
@prefix rdf:  <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl:  <http://www.w3.org/2002/07/owl#> .
@prefix ex:   <http://example.org/union#> .

ex:A a owl:Class .

ex:relatedTo a owl:ObjectProperty ;
    rdfs:label "related to" ;
    rdfs:domain [ owl:unionOf ( ex:A ex:B ) ] ;
    rdfs:range ex:C .
`
	g := parseTurtle(t, src, "http://example.org/union")
	gm, err := transform.BuildGraphModel(g, transform.Options{Simplify: true})
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	const (
		iriA = "http://example.org/union#A"
		iriB = "http://example.org/union#B"
		iriC = "http://example.org/union#C"
	)

	// All member classes and the range must still exist as nodes.
	for _, iri := range []string{iriA, iriB, iriC} {
		if findNode(gm.Nodes, iri) == nil {
			t.Fatalf("expected node %q", iri)
		}
	}

	// No union node should be created in simplified mode.
	for _, n := range gm.Nodes {
		if n.Type == graph.NodeTypeUnion {
			t.Errorf("unexpected union node %q in simplified mode", n.ID)
		}
	}

	// Direct edges from each union member to the range must exist.
	if !hasLink(gm.Links, iriA, iriC, "related to") {
		t.Error("missing direct edge A → C (related to)")
	}
	if !hasLink(gm.Links, iriB, iriC, "related to") {
		t.Error("missing direct edge B → C (related to)")
	}

	// No unionOf edges should be present.
	for _, l := range gm.Links {
		if l.Label == "unionOf" {
			t.Errorf("unexpected unionOf edge %q → %q in simplified mode", l.Source, l.Target)
		}
	}

	if err := gm.Validate(); err != nil {
		t.Errorf("GraphModel.Validate() = %v", err)
	}
}

// TestBuildGraphModel_SimplifyUnionRange verifies that when Options.Simplify is
// true, owl:unionOf range class expressions produce direct edges from the
// domain to each union member rather than routing through a triangle union node.
func TestBuildGraphModel_SimplifyUnionRange(t *testing.T) {
	const src = `
@prefix rdf:  <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl:  <http://www.w3.org/2002/07/owl#> .
@prefix ex:   <http://example.org/union#> .

ex:A a owl:Class .
ex:B a owl:Class .

ex:relatedTo a owl:ObjectProperty ;
    rdfs:label "related to" ;
    rdfs:domain ex:A ;
    rdfs:range [ owl:unionOf ( ex:B ex:C ) ] .
`
	g := parseTurtle(t, src, "http://example.org/union")
	gm, err := transform.BuildGraphModel(g, transform.Options{Simplify: true})
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	const (
		iriA = "http://example.org/union#A"
		iriB = "http://example.org/union#B"
		iriC = "http://example.org/union#C"
	)

	// All nodes must exist.
	for _, iri := range []string{iriA, iriB, iriC} {
		if findNode(gm.Nodes, iri) == nil {
			t.Fatalf("expected node %q", iri)
		}
	}

	// No union node in simplified mode.
	for _, n := range gm.Nodes {
		if n.Type == graph.NodeTypeUnion {
			t.Errorf("unexpected union node %q in simplified mode", n.ID)
		}
	}

	// Direct edges from the domain to each union member.
	if !hasLink(gm.Links, iriA, iriB, "related to") {
		t.Error("missing direct edge A → B (related to)")
	}
	if !hasLink(gm.Links, iriA, iriC, "related to") {
		t.Error("missing direct edge A → C (related to)")
	}

	// No unionOf edges.
	for _, l := range gm.Links {
		if l.Label == "unionOf" {
			t.Errorf("unexpected unionOf edge %q → %q in simplified mode", l.Source, l.Target)
		}
	}

	if err := gm.Validate(); err != nil {
		t.Errorf("GraphModel.Validate() = %v", err)
	}
}

// TestBuildGraphModel_SimplifyFalsePreservesUnionNode confirms that the default
// (non-simplified) behaviour is unchanged when Options.Simplify is false.
func TestBuildGraphModel_SimplifyFalsePreservesUnionNode(t *testing.T) {
	const src = `
@prefix rdf:  <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl:  <http://www.w3.org/2002/07/owl#> .
@prefix ex:   <http://example.org/union#> .

ex:A a owl:Class .

ex:relatedTo a owl:ObjectProperty ;
    rdfs:label "related to" ;
    rdfs:domain [ owl:unionOf ( ex:A ex:B ) ] ;
    rdfs:range ex:C .
`
	g := parseTurtle(t, src, "http://example.org/union")
	gm, err := transform.BuildGraphModel(g, transform.Options{Simplify: false})
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	// A union node must exist.
	var unionID string
	for _, n := range gm.Nodes {
		if n.Type == graph.NodeTypeUnion {
			unionID = n.ID
			break
		}
	}
	if unionID == "" {
		t.Fatal("expected a union node when Simplify is false")
	}

	// Edge must go from union node to range, not directly from members.
	const iriC = "http://example.org/union#C"
	if !hasLink(gm.Links, unionID, iriC, "related to") {
		t.Error("expected edge union → C (related to)")
	}
}

// ---------------------------------------------------------------------------
// BuildGraphModel – owl:inverseOf object property inference
// ---------------------------------------------------------------------------

// TestBuildGraphModel_InverseOfPropertyBasic verifies the core issue scenario:
// an object property implied by owl:inverseOf produces an edge with swapped
// domain and range.
func TestBuildGraphModel_InverseOfPropertyBasic(t *testing.T) {
	// hasState has explicit domain :Thing and range :State.
	// isStateOf is declared only via "hasState owl:inverseOf isStateOf";
	// its domain should be :State and its range should be :Thing.
	const src = `
@prefix rdf:  <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl:  <http://www.w3.org/2002/07/owl#> .
@prefix ex:   <http://example.org/inv#> .

ex:Thing a owl:Class .
ex:State a owl:Class .

ex:hasState a owl:ObjectProperty ;
    rdfs:label "hasState"@en ;
    rdfs:domain ex:Thing ;
    rdfs:range  ex:State ;
    owl:inverseOf ex:isStateOf .
`
	g := parseTurtle(t, src, "http://example.org/inv")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	const (
		iriThing     = "http://example.org/inv#Thing"
		iriState     = "http://example.org/inv#State"
		iriHasState  = "http://example.org/inv#hasState"
		iriIsStateOf = "http://example.org/inv#isStateOf"
	)

	// The forward property must still produce an edge Thing → State.
	if !hasLink(gm.Links, iriThing, iriState, "hasState") {
		t.Error("missing forward edge Thing → State (hasState)")
	}

	// The inverse property must produce an edge State → Thing.
	if !hasLink(gm.Links, iriState, iriThing, "isStateOf") {
		t.Error("missing inverse edge State → Thing (isStateOf)")
	}

	// Neither property IRI should appear as a standalone node.
	if findNode(gm.Nodes, iriHasState) != nil {
		t.Error("hasState should be an edge, not a node")
	}
	if findNode(gm.Nodes, iriIsStateOf) != nil {
		t.Error("isStateOf should be an edge, not a node")
	}

	if err := gm.Validate(); err != nil {
		t.Errorf("GraphModel.Validate() = %v", err)
	}
}

// TestBuildGraphModel_InverseOfPropertyWithLabel verifies that when an explicit
// rdfs:label is provided for the inverse property it is used in the edge label.
func TestBuildGraphModel_InverseOfPropertyWithLabel(t *testing.T) {
	const src = `
@prefix rdf:  <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl:  <http://www.w3.org/2002/07/owl#> .
@prefix ex:   <http://example.org/invlabel#> .

ex:A a owl:Class .
ex:B a owl:Class .

ex:parentOf a owl:ObjectProperty ;
    rdfs:label "parent of"@en ;
    rdfs:domain ex:A ;
    rdfs:range  ex:B ;
    owl:inverseOf ex:childOf .

ex:childOf rdfs:label "child of"@en .
`
	g := parseTurtle(t, src, "http://example.org/invlabel")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	const (
		iriA       = "http://example.org/invlabel#A"
		iriB       = "http://example.org/invlabel#B"
		iriChildOf = "http://example.org/invlabel#childOf"
	)

	// Inverse edge must use its own rdfs:label.
	if !hasLink(gm.Links, iriB, iriA, "child of") {
		t.Error("missing inverse edge B → A (child of)")
	}
	if findNode(gm.Nodes, iriChildOf) != nil {
		t.Error("childOf should be an edge, not a node")
	}

	if err := gm.Validate(); err != nil {
		t.Errorf("GraphModel.Validate() = %v", err)
	}
}

// TestBuildGraphModel_InverseOfBothDirections verifies that when both properties
// explicitly declare each other as inverses, domain/range are still inferred
// correctly and no duplicate edges are produced.
func TestBuildGraphModel_InverseOfBothDirections(t *testing.T) {
	const src = `
@prefix rdf:  <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl:  <http://www.w3.org/2002/07/owl#> .
@prefix ex:   <http://example.org/invboth#> .

ex:A a owl:Class .
ex:B a owl:Class .

ex:knows a owl:ObjectProperty ;
    rdfs:domain ex:A ;
    rdfs:range  ex:B ;
    owl:inverseOf ex:isKnownBy .

ex:isKnownBy a owl:ObjectProperty ;
    owl:inverseOf ex:knows .
`
	g := parseTurtle(t, src, "http://example.org/invboth")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	const (
		iriA         = "http://example.org/invboth#A"
		iriB         = "http://example.org/invboth#B"
		iriKnows     = "http://example.org/invboth#knows"
		iriIsKnownBy = "http://example.org/invboth#isKnownBy"
	)

	if !hasLink(gm.Links, iriA, iriB, "knows") {
		t.Error("missing forward edge A → B (knows)")
	}
	if !hasLink(gm.Links, iriB, iriA, "isKnownBy") {
		t.Error("missing inverse edge B → A (isKnownBy)")
	}

	// No property IRI should remain as a node.
	if findNode(gm.Nodes, iriKnows) != nil {
		t.Error("knows should be an edge, not a node")
	}
	if findNode(gm.Nodes, iriIsKnownBy) != nil {
		t.Error("isKnownBy should be an edge, not a node")
	}

	if err := gm.Validate(); err != nil {
		t.Errorf("GraphModel.Validate() = %v", err)
	}
}

// TestBuildGraphModel_InverseOfDoesNotOverrideExplicitDomainRange verifies that
// when the inverse property already has its own explicit domain and range, those
// values are not overwritten by the inference step.
func TestBuildGraphModel_InverseOfDoesNotOverrideExplicitDomainRange(t *testing.T) {
	const src = `
@prefix rdf:  <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl:  <http://www.w3.org/2002/07/owl#> .
@prefix ex:   <http://example.org/invexplicit#> .

ex:A a owl:Class .
ex:B a owl:Class .
ex:C a owl:Class .
ex:D a owl:Class .

ex:forward a owl:ObjectProperty ;
    rdfs:domain ex:A ;
    rdfs:range  ex:B ;
    owl:inverseOf ex:backward .

ex:backward a owl:ObjectProperty ;
    rdfs:domain ex:C ;
    rdfs:range  ex:D .
`
	g := parseTurtle(t, src, "http://example.org/invexplicit")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	const (
		iriA = "http://example.org/invexplicit#A"
		iriB = "http://example.org/invexplicit#B"
		iriC = "http://example.org/invexplicit#C"
		iriD = "http://example.org/invexplicit#D"
	)

	// forward: A → B (unchanged).
	if !hasLink(gm.Links, iriA, iriB, "forward") {
		t.Error("missing forward edge A → B (forward)")
	}
	// backward: C → D (explicit domain/range must not be overridden).
	if !hasLink(gm.Links, iriC, iriD, "backward") {
		t.Error("missing backward edge C → D (backward)")
	}
	// The inferred inverse (B → A) must NOT appear because backward already
	// has explicit domain/range.
	if hasLink(gm.Links, iriB, iriA, "backward") {
		t.Error("backward edge B → A must not be created when explicit domain/range exist")
	}

	if err := gm.Validate(); err != nil {
		t.Errorf("GraphModel.Validate() = %v", err)
	}
}

// TestBuildGraphModel_InverseOfNoEdgeWhenNoForwardDomainRange verifies that
// when the forward property itself has no domain or range, no inverse edge is
// produced (neither property can become an edge).
func TestBuildGraphModel_InverseOfNoEdgeWhenNoForwardDomainRange(t *testing.T) {
	const src = `
@prefix rdf:  <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix owl:  <http://www.w3.org/2002/07/owl#> .
@prefix ex:   <http://example.org/invnone#> .

ex:forward a owl:ObjectProperty ;
    owl:inverseOf ex:backward .
`
	g := parseTurtle(t, src, "http://example.org/invnone")
	gm, err := transform.BuildGraphModel(g)
	if err != nil {
		t.Fatalf("BuildGraphModel: %v", err)
	}

	// forward has no domain/range, so it falls back to a property node.
	n := findNode(gm.Nodes, "http://example.org/invnone#forward")
	if n == nil {
		t.Fatal("forward without domain/range should be a property node")
	}
	if n.Type != graph.NodeTypeProperty {
		t.Errorf("forward node Type = %q, want %q", n.Type, graph.NodeTypeProperty)
	}

	// backward was inferred as an object property but also has no domain/range,
	// so it too falls back to a property node.
	nb := findNode(gm.Nodes, "http://example.org/invnone#backward")
	if nb == nil {
		t.Fatal("backward inferred without domain/range should be a property node")
	}
	if nb.Type != graph.NodeTypeProperty {
		t.Errorf("backward node Type = %q, want %q", nb.Type, graph.NodeTypeProperty)
	}

	// No links should exist.
	if len(gm.Links) != 0 {
		t.Errorf("expected 0 links, got %d", len(gm.Links))
	}
}
