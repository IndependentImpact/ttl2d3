package transform

import (
	"sort"

	"github.com/IndependentImpact/ttl2d3/internal/parser"
)

// indimp namespace IRIs for the WorkflowPlan vocabulary
// (canonical: https://independentimpact.org/ns/indimp#).
const (
	iriIndImpWorkflowPlan       = "https://independentimpact.org/ns/indimp#WorkflowPlan"       //nolint:gosec // IRI constant, not a credential
	iriIndImpWorkflowTransition = "https://independentimpact.org/ns/indimp#WorkflowTransition" //nolint:gosec
	iriIndImpWorkflowGate       = "https://independentimpact.org/ns/indimp#WorkflowGate"       //nolint:gosec
	iriIndImpHasTransition      = "https://independentimpact.org/ns/indimp#hasTransition"      //nolint:gosec
	iriIndImpHasGate            = "https://independentimpact.org/ns/indimp#hasGate"            //nolint:gosec
	iriIndImpFromStep           = "https://independentimpact.org/ns/indimp#fromStep"           //nolint:gosec
	iriIndImpToStep             = "https://independentimpact.org/ns/indimp#toStep"             //nolint:gosec
	iriIndImpFromGate           = "https://independentimpact.org/ns/indimp#fromGate"           //nolint:gosec
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
//  2. Identify steps via objects of indimp:fromStep or indimp:toStep on
//     indimp:WorkflowTransition resources, and gate IRIs via indimp:fromGate.
//  3. Identify transitions: resources with indimp:fromStep or indimp:fromGate
//     combined with indimp:toStep.
//  4. For explicit plans, derive step membership by following indimp:hasTransition
//     to transition resources, then collecting their from/to endpoints; also
//     include gates linked via indimp:hasGate.
//  5. Attach rdfs:label literals to steps and transitions.
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
	// Identify all node IRIs (steps and gates).
	// Nodes may be:
	//   (a) objects of indimp:fromStep or indimp:toStep on a
	//       WorkflowTransition resource, or
	//   (b) objects of indimp:fromGate on a WorkflowTransition resource
	//       (WorkflowGate IRIs treated as step-like nodes), or
	//   (c) objects of indimp:hasGate on a WorkflowPlan resource.
	// -----------------------------------------------------------------------
	stepSet := make(map[string]struct{})

	// Objects of indimp:fromStep, indimp:toStep, and indimp:fromGate.
	for subj, preds := range subjPredToObjs {
		fromSteps := preds[iriIndImpFromStep]
		fromGates := preds[iriIndImpFromGate]
		toSteps := preds[iriIndImpToStep]
		if len(fromSteps) == 0 && len(fromGates) == 0 && len(toSteps) == 0 {
			continue
		}
		for _, s := range fromSteps {
			if s != "" {
				stepSet[s] = struct{}{}
			}
		}
		for _, g := range fromGates {
			if g != "" {
				stepSet[g] = struct{}{} // gates as step-like nodes
			}
		}
		for _, s := range toSteps {
			if s != "" {
				stepSet[s] = struct{}{}
			}
		}
		// Direct step→step: if the subject itself has toStep but no fromStep/fromGate,
		// it is also a node.
		if len(fromSteps) == 0 && len(fromGates) == 0 && len(toSteps) > 0 {
			if subj != "" {
				stepSet[subj] = struct{}{}
			}
		}
	}

	// Gate IRIs directly linked via indimp:hasGate on plans.
	for planIRI := range planSet {
		for _, g := range subjPredToObjs[planIRI][iriIndImpHasGate] {
			if g != "" {
				stepSet[g] = struct{}{}
			}
		}
	}

	// -----------------------------------------------------------------------
	// Collect all (from, to) transition pairs.
	// transLabels[from][to] = label string
	// -----------------------------------------------------------------------
	type transKey struct{ from, to string }
	transLabels := make(map[transKey]string)

	// Explicit Transition resources: have indimp:fromStep or indimp:fromGate.
	for transSubj, preds := range subjPredToObjs {
		fromSteps := preds[iriIndImpFromStep]
		fromGates := preds[iriIndImpFromGate]
		if len(fromSteps) == 0 && len(fromGates) == 0 {
			continue
		}
		toSteps := preds[iriIndImpToStep]
		allFroms := append(fromSteps, fromGates...) //nolint:gocritic // intentional append to new slice
		for _, fromNode := range allFroms {
			for _, toStep := range toSteps {
				key := transKey{from: fromNode, to: toStep}
				if _, exists := transLabels[key]; !exists {
					transLabels[key] = labelOf(transSubj)
				}
			}
		}
	}
	// Direct step→step: step has indimp:toStep but no indimp:fromStep or indimp:fromGate.
	for stepSubj, preds := range subjPredToObjs {
		if len(preds[iriIndImpFromStep]) > 0 || len(preds[iriIndImpFromGate]) > 0 {
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

		// Steps: prefer explicit membership derived from indimp:hasTransition.
		// Follow each transition linked to the plan and collect its from/to endpoints.
		var stepIRIs []string
		if planIRI != "" {
			transitionIRIs := sortedUniq(subjPredToObjs[planIRI][iriIndImpHasTransition])
			if len(transitionIRIs) > 0 {
				derived := make(map[string]struct{})
				for _, tIRI := range transitionIRIs {
					for _, s := range subjPredToObjs[tIRI][iriIndImpFromStep] {
						if s != "" {
							derived[s] = struct{}{}
						}
					}
					for _, s := range subjPredToObjs[tIRI][iriIndImpToStep] {
						if s != "" {
							derived[s] = struct{}{}
						}
					}
					for _, g := range subjPredToObjs[tIRI][iriIndImpFromGate] {
						if g != "" {
							derived[g] = struct{}{}
						}
					}
				}
				stepIRIs = sortedKeys(derived)
			}
		}
		if len(stepIRIs) == 0 {
			// Fall back to all known steps.
			stepIRIs = sortedKeys(stepSet)
		}

		for _, sid := range stepIRIs {
			plan.Steps = append(plan.Steps, WorkflowStep{
				ID:    sid,
				Label: labelOf(sid),
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
