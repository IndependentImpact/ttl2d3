package transform_test

import (
	"strings"
	"testing"

	"github.com/IndependentImpact/ttl2d3/internal/config"
	"github.com/IndependentImpact/ttl2d3/internal/parser"
	"github.com/IndependentImpact/ttl2d3/internal/transform"
)

// parseWorkflowTurtle is a helper that parses a Turtle string and returns the
// resulting graph.
func parseWorkflowTurtle(t *testing.T, src string) *parser.Graph {
	t.Helper()
	r := strings.NewReader(src)
	g, err := parser.Parse(r, "test.ttl", "test.ttl", config.InputTurtle)
	if err != nil {
		t.Fatalf("parseWorkflowTurtle: %v", err)
	}
	return g
}

// TestBuildWorkflowModel_NilGraph checks that a nil graph returns an empty model.
func TestBuildWorkflowModel_NilGraph(t *testing.T) {
	wm, err := transform.BuildWorkflowModel(nil)
	if err != nil {
		t.Fatalf("BuildWorkflowModel(nil) error = %v", err)
	}
	if wm == nil {
		t.Fatal("BuildWorkflowModel(nil) returned nil model")
	}
	if len(wm.Plans) != 0 {
		t.Errorf("expected 0 plans, got %d", len(wm.Plans))
	}
}

// TestBuildWorkflowModel_ExplicitPlan exercises a fully-annotated
// indimp:WorkflowPlan with steps and WorkflowTransition resources.
func TestBuildWorkflowModel_ExplicitPlan(t *testing.T) {
	const ttl = `
@prefix rdf:    <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs:   <http://www.w3.org/2000/01/rdf-schema#> .
@prefix skos:   <http://www.w3.org/2004/02/skos/core#> .
@prefix indimp: <https://independentimpact.org/ns/indimp#> .
@prefix ex:     <https://example.org/> .

ex:Plan a indimp:WorkflowPlan ;
    rdfs:label "My Plan" ;
    indimp:hasTransition ex:T1 .

ex:StepA a skos:Concept ;
    rdfs:label "Step A" .

ex:StepB a skos:Concept ;
    rdfs:label "Step B" .

ex:T1 a indimp:WorkflowTransition ;
    rdfs:label "next" ;
    indimp:fromStep ex:StepA ;
    indimp:toStep   ex:StepB .
`
	g := parseWorkflowTurtle(t, ttl)
	wm, err := transform.BuildWorkflowModel(g)
	if err != nil {
		t.Fatalf("BuildWorkflowModel error = %v", err)
	}

	if len(wm.Plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(wm.Plans))
	}

	plan := wm.Plans[0]
	if plan.Label != "My Plan" {
		t.Errorf("plan.Label = %q, want %q", plan.Label, "My Plan")
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(plan.Steps))
	}

	// Steps are sorted by IRI; ex:StepA < ex:StepB.
	if plan.Steps[0].Label != "Step A" {
		t.Errorf("steps[0].Label = %q, want %q", plan.Steps[0].Label, "Step A")
	}
	if plan.Steps[1].Label != "Step B" {
		t.Errorf("steps[1].Label = %q, want %q", plan.Steps[1].Label, "Step B")
	}

	if len(plan.Transitions) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(plan.Transitions))
	}
	tr := plan.Transitions[0]
	if tr.Label != "next" {
		t.Errorf("transition.Label = %q, want %q", tr.Label, "next")
	}
	if !strings.HasSuffix(tr.From, "StepA") {
		t.Errorf("transition.From = %q, want suffix StepA", tr.From)
	}
	if !strings.HasSuffix(tr.To, "StepB") {
		t.Errorf("transition.To = %q, want suffix StepB", tr.To)
	}
}

// TestBuildWorkflowModel_DirectToStep exercises steps with indimp:toStep
// directly (no separate WorkflowTransition resource).
func TestBuildWorkflowModel_DirectToStep(t *testing.T) {
	const ttl = `
@prefix rdf:    <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs:   <http://www.w3.org/2000/01/rdf-schema#> .
@prefix skos:   <http://www.w3.org/2004/02/skos/core#> .
@prefix indimp: <https://independentimpact.org/ns/indimp#> .
@prefix ex:     <https://example.org/> .

ex:StepA a skos:Concept ;
    rdfs:label "Step A" ;
    indimp:toStep ex:StepB .

ex:StepB a skos:Concept ;
    rdfs:label "Step B" .
`
	g := parseWorkflowTurtle(t, ttl)
	wm, err := transform.BuildWorkflowModel(g)
	if err != nil {
		t.Fatalf("BuildWorkflowModel error = %v", err)
	}

	if len(wm.Plans) != 1 {
		t.Fatalf("expected 1 synthetic plan, got %d", len(wm.Plans))
	}
	plan := wm.Plans[0]
	if len(plan.Transitions) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(plan.Transitions))
	}
}

// TestBuildWorkflowModel_NoWorkflowData returns empty model when no indimp
// triples are present.
func TestBuildWorkflowModel_NoWorkflowData(t *testing.T) {
	const ttl = `
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix ex:   <https://example.org/> .

ex:A rdfs:label "Just a class" .
`
	g := parseWorkflowTurtle(t, ttl)
	wm, err := transform.BuildWorkflowModel(g)
	if err != nil {
		t.Fatalf("BuildWorkflowModel error = %v", err)
	}
	if len(wm.Plans) != 0 {
		t.Errorf("expected 0 plans for non-workflow input, got %d", len(wm.Plans))
	}
}

// TestBuildWorkflowModel_Workflowplan_Testdata exercises parsing the
// testdata/workflowplan.ttl fixture (canonical indimp namespace).
func TestBuildWorkflowModel_Workflowplan_Testdata(t *testing.T) {
	g := parseWorkflowTurtle(t, workflowplanTTL)
	wm, err := transform.BuildWorkflowModel(g)
	if err != nil {
		t.Fatalf("BuildWorkflowModel error = %v", err)
	}
	if len(wm.Plans) == 0 {
		t.Fatal("expected at least one plan from workflowplan.ttl, got none")
	}
	plan := wm.Plans[0]
	if len(plan.Steps) == 0 {
		t.Error("expected steps in plan, got none")
	}
	if len(plan.Transitions) == 0 {
		t.Error("expected transitions in plan, got none")
	}
}

// TestBuildWorkflowModel_FromGate exercises a WorkflowTransition that uses
// indimp:fromGate (gate node as source) instead of indimp:fromStep.
func TestBuildWorkflowModel_FromGate(t *testing.T) {
	const ttl = `
@prefix rdf:    <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs:   <http://www.w3.org/2000/01/rdf-schema#> .
@prefix skos:   <http://www.w3.org/2004/02/skos/core#> .
@prefix indimp: <https://independentimpact.org/ns/indimp#> .
@prefix ex:     <https://example.org/> .

ex:Plan a indimp:WorkflowPlan ;
    rdfs:label "Gated Plan" ;
    indimp:hasTransition ex:T1, ex:T2 ;
    indimp:hasGate ex:Gate1 .

ex:StepA a skos:Concept ; rdfs:label "Step A" .
ex:StepB a skos:Concept ; rdfs:label "Step B" .
ex:Gate1 a indimp:WorkflowGate ; rdfs:label "Approval Gate" .

ex:T1 a indimp:WorkflowTransition ; rdfs:label "to gate" ;
    indimp:fromStep ex:StepA ;
    indimp:toStep   ex:Gate1 .

ex:T2 a indimp:WorkflowTransition ; rdfs:label "proceed" ;
    indimp:fromGate ex:Gate1 ;
    indimp:toStep   ex:StepB .
`
	g := parseWorkflowTurtle(t, ttl)
	wm, err := transform.BuildWorkflowModel(g)
	if err != nil {
		t.Fatalf("BuildWorkflowModel error = %v", err)
	}
	if len(wm.Plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(wm.Plans))
	}
	plan := wm.Plans[0]
	if plan.Label != "Gated Plan" {
		t.Errorf("plan.Label = %q, want %q", plan.Label, "Gated Plan")
	}
	// Expect 3 nodes: StepA, Gate1, StepB.
	if len(plan.Steps) != 3 {
		t.Fatalf("expected 3 steps/gates, got %d", len(plan.Steps))
	}
	if len(plan.Transitions) != 2 {
		t.Fatalf("expected 2 transitions, got %d", len(plan.Transitions))
	}
	// Check that the gate node is present.
	found := false
	for _, s := range plan.Steps {
		if strings.HasSuffix(s.ID, "Gate1") {
			found = true
			if s.Label != "Approval Gate" {
				t.Errorf("gate label = %q, want %q", s.Label, "Approval Gate")
			}
		}
	}
	if !found {
		t.Error("gate node Gate1 not found in plan steps")
	}
	// The fromGate transition should be present.
	foundGateTr := false
	for _, tr := range plan.Transitions {
		if strings.HasSuffix(tr.From, "Gate1") && strings.HasSuffix(tr.To, "StepB") {
			foundGateTr = true
			if tr.Label != "proceed" {
				t.Errorf("gate transition label = %q, want %q", tr.Label, "proceed")
			}
		}
	}
	if !foundGateTr {
		t.Error("gate→StepB transition not found")
	}
}

// workflowplanTTL is the content of testdata/workflowplan.ttl inlined for use
// without a file-system dependency in the unit test.
const workflowplanTTL = `
@prefix rdf:    <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs:   <http://www.w3.org/2000/01/rdf-schema#> .
@prefix skos:   <http://www.w3.org/2004/02/skos/core#> .
@prefix indimp: <https://independentimpact.org/ns/indimp#> .
@prefix wfp:    <https://example.org/workflowplan#> .

wfp:DocumentApprovalWorkflow
    a indimp:WorkflowPlan ;
    rdfs:label "Document Approval Workflow" ;
    indimp:hasTransition wfp:T1, wfp:T2, wfp:T3, wfp:T4, wfp:T5 .

wfp:Submit  a skos:Concept ; rdfs:label "Submit Document" .
wfp:Review  a skos:Concept ; rdfs:label "Review Document" .
wfp:Clarify a skos:Concept ; rdfs:label "Request Clarification" .
wfp:Approve a skos:Concept ; rdfs:label "Approve" .
wfp:Reject  a skos:Concept ; rdfs:label "Reject" .

wfp:T1 a indimp:WorkflowTransition ; rdfs:label "submit" ;
    indimp:fromStep wfp:Submit ; indimp:toStep wfp:Review .
wfp:T2 a indimp:WorkflowTransition ; rdfs:label "needs clarification" ;
    indimp:fromStep wfp:Review ; indimp:toStep wfp:Clarify .
wfp:T3 a indimp:WorkflowTransition ; rdfs:label "approved" ;
    indimp:fromStep wfp:Review ; indimp:toStep wfp:Approve .
wfp:T4 a indimp:WorkflowTransition ; rdfs:label "rejected" ;
    indimp:fromStep wfp:Review ; indimp:toStep wfp:Reject .
wfp:T5 a indimp:WorkflowTransition ; rdfs:label "resubmit" ;
    indimp:fromStep wfp:Clarify ; indimp:toStep wfp:Review .
`
