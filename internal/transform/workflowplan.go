package transform

import (
	"sort"

	"github.com/IndependentImpact/ttl2d3/internal/parser"
)

// indimp namespace IRIs for the WorkflowPlan vocabulary.
const (
	iriIndImpWorkflowPlan = "https://w3id.org/indimp#WorkflowPlan" //nolint:gosec // false positive: IRI, not a credential
	iriIndImpStep         = "https://w3id.org/indimp#Step"
	iriIndImpHasStep      = "https://w3id.org/indimp#hasStep"
	iriIndImpFromStep     = "https://w3id.org/indimp#fromStep"
	iriIndImpToStep       = "https://w3id.org/indimp#toStep"
	iriIndImpActor        = "https://w3id.org/indimp#actor"
)

// WorkflowStep represents a single step in an indimp:WorkflowPlan.
type WorkflowStep struct {
	// ID is the IRI of the step resource.
	ID string `json:"id"`
	// Label is the human-readable name of the step.
	Label string `json:"label"`
	// Actor is the role or person responsible for this step (used as swimlane).
	// Empty when no indimp:actor triple is present.
	Actor string `json:"actor,omitempty"`
}

// WorkflowTransition is a directed connection between two workflow steps.
type WorkflowTransition struct {
	// From is the IRI of the source step.
	From string `json:"from"`
	// To is the IRI of the target step.
	To string `json:"to"`
	// Label is the human-readable label of the transition (rdfs:label).
	Label string `json:"label,omitempty"`
}

// WorkflowPlan captures the ordered structure of a single indimp:WorkflowPlan.
type WorkflowPlan struct {
	// ID is the IRI of the workflow plan resource.
	ID string `json:"id"`
	// Label is the human-readable title of the plan.
	Label string `json:"label"`
	// Steps contains all steps belonging to this plan (sorted by IRI).
	// Topological ordering is applied by the HTML template in the browser.
	Steps []WorkflowStep `json:"steps"`
	// Transitions contains all directed connections between steps.
	Transitions []WorkflowTransition `json:"transitions"`
}

// WorkflowModel is the structured model produced from indimp:WorkflowPlan
// resources, ready for rendering as a directed process / swimlane diagram.
type WorkflowModel struct {
	// Plans is the list of workflow plans found in the input.
	Plans []WorkflowPlan
}

// BuildWorkflowModel extracts all indimp:WorkflowPlan resources from g and
// constructs a WorkflowModel ready for the process-diagram renderer.
//
// The algorithm:
//  1. Scan triples for rdf:type == indimp:WorkflowPlan to find plan IRIs.
//  2. Identify steps via indimp:hasStep on each plan, or via rdf:type ==
//     indimp:Step, or via objects of indimp:fromStep / indimp:toStep triples.
//  3. Identify transitions: resources with both indimp:fromStep and
//     indimp:toStep triples, or steps carrying indimp:toStep directly.
//  4. Attach rdfs:label and indimp:actor literals to steps and transitions.
func BuildWorkflowModel(g *parser.Graph) (*WorkflowModel, error) {
	if g == nil {
		return &WorkflowModel{}, nil
	}

	// -----------------------------------------------------------------------
	// Build lookup indexes.
	//   subjPredToObjs[subj][pred] = []IRI-objects
	//   subjPredToLit[subj][pred]  = first literal value
	// -----------------------------------------------------------------------
	subjPredToObjs := make(map[string]map[string][]string)
	subjPredToLit := make(map[string]map[string]string)

	addObjTriple := func(subj, pred, obj string) {
		if subjPredToObjs[subj] == nil {
			subjPredToObjs[subj] = make(map[string][]string)
		}
		subjPredToObjs[subj][pred] = append(subjPredToObjs[subj][pred], obj)
	}

	addLitTriple := func(subj, pred, val string) {
		if subjPredToLit[subj] == nil {
			subjPredToLit[subj] = make(map[string]string)
		}
		if _, exists := subjPredToLit[subj][pred]; !exists {
			subjPredToLit[subj][pred] = val
		}
	}

	for _, t := range g.Triples {
		subj := termIRI(t.Subject)
		pred := termIRI(t.Predicate)
		if subj == "" || pred == "" {
			continue
		}
		if t.Object.Kind == parser.TermIRI {
			addObjTriple(subj, pred, t.Object.Value)
		} else if t.Object.Kind == parser.TermLiteral {
			addLitTriple(subj, pred, t.Object.Value)
		}
	}

	// -----------------------------------------------------------------------
	// labelOf returns the rdfs:label or local-name fallback for an IRI.
	// -----------------------------------------------------------------------
	labelOf := func(iri string) string {
		if m, ok := subjPredToLit[iri]; ok {
			if v, ok2 := m[iriRDFSLabel]; ok2 {
				return v
			}
		}
		return localName(iri)
	}

	// -----------------------------------------------------------------------
	// typeSet builds the set of subjects declared with the given rdf:type IRI.
	// -----------------------------------------------------------------------
	typeSet := func(typeIRI string) map[string]struct{} {
		out := make(map[string]struct{})
		for subj, preds := range subjPredToObjs {
			for _, obj := range preds[iriRDFType] {
				if obj == typeIRI {
					out[subj] = struct{}{}
				}
			}
		}
		return out
	}

	// -----------------------------------------------------------------------
	// Identify plan IRIs.
	// -----------------------------------------------------------------------
	planSet := typeSet(iriIndImpWorkflowPlan)
	planIRIs := sortedKeys(planSet)

	// -----------------------------------------------------------------------
	// Identify all step IRIs.
	// Steps may be:
	//   (a) declared with rdf:type indimp:Step, or
	//   (b) referenced as objects of indimp:hasStep triples, or
	//   (c) objects (values) of indimp:fromStep or indimp:toStep on a
	//       Transition resource.
	// -----------------------------------------------------------------------
	stepSet := typeSet(iriIndImpStep)

	// Objects of indimp:hasStep.
	for _, preds := range subjPredToObjs {
		for _, obj := range preds[iriIndImpHasStep] {
			if obj != "" {
				stepSet[obj] = struct{}{}
			}
		}
	}

	// Objects of indimp:fromStep and indimp:toStep on Transition resources.
	// Note: we add the *objects* (step IRIs), not the subjects (transition IRIs).
	for subj, preds := range subjPredToObjs {
		if froms, ok := preds[iriIndImpFromStep]; ok {
			for _, fromStep := range froms {
				if fromStep != "" {
					stepSet[fromStep] = struct{}{}
				}
			}
			for _, toStep := range preds[iriIndImpToStep] {
				if toStep != "" {
					stepSet[toStep] = struct{}{}
				}
			}
			_ = subj // subj is the Transition IRI; do not add to stepSet
		}
	}
	// Steps that carry indimp:toStep directly (no separate Transition resource).
	for subj, preds := range subjPredToObjs {
		if _, isTransition := preds[iriIndImpFromStep]; isTransition {
			continue
		}
		for _, toStep := range preds[iriIndImpToStep] {
			if toStep != "" {
				stepSet[toStep] = struct{}{}
			}
		}
		if _, hasToStep := preds[iriIndImpToStep]; hasToStep {
			if subj != "" {
				stepSet[subj] = struct{}{}
			}
		}
	}

	// -----------------------------------------------------------------------
	// Collect all (from, to) transition pairs.
	// transLabels[from][to] = label string
	// -----------------------------------------------------------------------
	type transKey struct{ from, to string }
	transLabels := make(map[transKey]string)

	// Explicit Transition resources: have indimp:fromStep.
	for transSubj, preds := range subjPredToObjs {
		fromSteps, hasFrm := preds[iriIndImpFromStep]
		if !hasFrm {
			continue
		}
		toSteps := preds[iriIndImpToStep]
		for _, fromStep := range fromSteps {
			for _, toStep := range toSteps {
				key := transKey{from: fromStep, to: toStep}
				if _, exists := transLabels[key]; !exists {
					transLabels[key] = labelOf(transSubj)
				}
			}
		}
	}
	// Direct step→step: step has indimp:toStep but not indimp:fromStep.
	for stepSubj, preds := range subjPredToObjs {
		if _, isTransition := preds[iriIndImpFromStep]; isTransition {
			continue
		}
		for _, toStep := range preds[iriIndImpToStep] {
			key := transKey{from: stepSubj, to: toStep}
			if _, exists := transLabels[key]; !exists {
				transLabels[key] = ""
			}
		}
	}

	// -----------------------------------------------------------------------
	// Synthesise a plan from loose steps when no explicit WorkflowPlan is typed.
	// -----------------------------------------------------------------------
	if len(planIRIs) == 0 && len(stepSet) > 0 {
		planIRIs = []string{""}
	}

	// -----------------------------------------------------------------------
	// Build WorkflowPlan entries.
	// -----------------------------------------------------------------------
	var plans []WorkflowPlan

	for _, planIRI := range planIRIs {
		plan := WorkflowPlan{
			ID:    planIRI,
			Label: labelOf(planIRI),
		}
		if planIRI == "" {
			plan.Label = "Workflow"
		}

		// Steps: prefer explicit indimp:hasStep membership.
		var stepIRIs []string
		if planIRI != "" {
			stepIRIs = sortedUniq(subjPredToObjs[planIRI][iriIndImpHasStep])
		}
		if len(stepIRIs) == 0 {
			// Fall back to all known steps.
			stepIRIs = sortedKeys(stepSet)
		}

		for _, sid := range stepIRIs {
			actor := ""
			if m, ok := subjPredToLit[sid]; ok {
				actor = m[iriIndImpActor]
			}
			plan.Steps = append(plan.Steps, WorkflowStep{
				ID:    sid,
				Label: labelOf(sid),
				Actor: actor,
			})
		}

		// Transitions: include all whose endpoints belong to this plan's steps.
		stepInPlan := make(map[string]bool, len(stepIRIs))
		for _, s := range stepIRIs {
			stepInPlan[s] = true
		}

		type transEntry struct {
			key   transKey
			label string
		}
		var entries []transEntry
		for k, lbl := range transLabels {
			if planIRI == "" || stepInPlan[k.from] || stepInPlan[k.to] {
				entries = append(entries, transEntry{k, lbl})
			}
		}
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].key.from != entries[j].key.from {
				return entries[i].key.from < entries[j].key.from
			}
			return entries[i].key.to < entries[j].key.to
		})

		for _, e := range entries {
			plan.Transitions = append(plan.Transitions, WorkflowTransition{
				From:  e.key.from,
				To:    e.key.to,
				Label: e.label,
			})
		}

		plans = append(plans, plan)
	}

	return &WorkflowModel{Plans: plans}, nil
}

// sortedKeys returns the keys of m as a sorted slice.
func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		if k != "" {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// sortedUniq returns a sorted, deduplicated copy of ss (empty strings excluded).
func sortedUniq(ss []string) []string {
	seen := make(map[string]struct{}, len(ss))
	var out []string
	for _, s := range ss {
		if s != "" {
			if _, ok := seen[s]; !ok {
				seen[s] = struct{}{}
				out = append(out, s)
			}
		}
	}
	sort.Strings(out)
	return out
}
