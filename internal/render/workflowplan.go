package render

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/IndependentImpact/ttl2d3/internal/transform"
)

// renderWorkflowModelJSON serialises wm to JSON and writes it to w.
func renderWorkflowModelJSON(wm *transform.WorkflowModel, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(wm.Plans); err != nil {
		return fmt.Errorf("render: encoding workflow model JSON: %w", err)
	}
	return nil
}
