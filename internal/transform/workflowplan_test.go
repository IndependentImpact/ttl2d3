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
// indimp:WorkflowPlan with steps, actors, and Transition resources.
func TestBuildWorkflowModel_ExplicitPlan(t *testing.T) {
	const ttl = `
@prefix rdf:    <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs:   <http://www.w3.org/2000/01/rdf-schema#> .
@prefix indimp: <https://w3id.org/indimp#> .
@prefix ex:     <https://example.org/> .

ex:Plan a indimp:WorkflowPlan ;
    rdfs:label "My Plan" ;
    indimp:hasStep ex:StepA, ex:StepB .

ex:StepA a indimp:Step ;
    rdfs:label "Step A" ;
    indimp:actor "Author" .

ex:StepB a indimp:Step ;
    rdfs:label "Step B" ;
    indimp:actor "Reviewer" .

ex:T1 a indimp:Transition ;
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
	if plan.Steps[0].Actor != "Author" {
		t.Errorf("steps[0].Actor = %q, want %q", plan.Steps[0].Actor, "Author")
	}
	if plan.Steps[1].Label != "Step B" {
		t.Errorf("steps[1].Label = %q, want %q", plan.Steps[1].Label, "Step B")
	}
	if plan.Steps[1].Actor != "Reviewer" {
		t.Errorf("steps[1].Actor = %q, want %q", plan.Steps[1].Actor, "Reviewer")
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
// directly (no separate Transition resource).
func TestBuildWorkflowModel_DirectToStep(t *testing.T) {
	const ttl = `
@prefix rdf:    <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs:   <http://www.w3.org/2000/01/rdf-schema#> .
@prefix indimp: <https://w3id.org/indimp#> .
@prefix ex:     <https://example.org/> .

ex:StepA a indimp:Step ;
    rdfs:label "Step A" ;
    indimp:toStep ex:StepB .

ex:StepB a indimp:Step ;
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
// testdata/workflowplan.ttl fixture.
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

// workflowplanTTL is the content of testdata/workflowplan.ttl inlined for use
// without a file-system dependency in the unit test.
const workflowplanTTL = `
@prefix rdf:    <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs:   <http://www.w3.org/2000/01/rdf-schema#> .
@prefix indimp: <https://w3id.org/indimp#> .
@prefix wfp:    <https://example.org/workflowplan#> .

wfp:DocumentApprovalWorkflow
    a indimp:WorkflowPlan ;
    rdfs:label "Document Approval Workflow" ;
    indimp:hasStep wfp:Submit, wfp:Review, wfp:Clarify, wfp:Approve, wfp:Reject .

wfp:Submit a indimp:Step ; rdfs:label "Submit Document" ; indimp:actor "Author" .
wfp:Review a indimp:Step ; rdfs:label "Review Document" ; indimp:actor "Reviewer" .
wfp:Clarify a indimp:Step ; rdfs:label "Request Clarification" ; indimp:actor "Reviewer" .
wfp:Approve a indimp:Step ; rdfs:label "Approve" ; indimp:actor "Manager" .
wfp:Reject  a indimp:Step ; rdfs:label "Reject"  ; indimp:actor "Manager" .

wfp:T1 a indimp:Transition ; rdfs:label "submit" ;
    indimp:fromStep wfp:Submit ; indimp:toStep wfp:Review .
wfp:T2 a indimp:Transition ; rdfs:label "needs clarification" ;
    indimp:fromStep wfp:Review ; indimp:toStep wfp:Clarify .
wfp:T3 a indimp:Transition ; rdfs:label "approved" ;
    indimp:fromStep wfp:Review ; indimp:toStep wfp:Approve .
wfp:T4 a indimp:Transition ; rdfs:label "rejected" ;
    indimp:fromStep wfp:Review ; indimp:toStep wfp:Reject .
wfp:T5 a indimp:Transition ; rdfs:label "resubmit" ;
    indimp:fromStep wfp:Clarify ; indimp:toStep wfp:Review .
`
