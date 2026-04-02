//go:build network

// Package e2e – network-gated tests.
// Run with: go test -tags network ./e2e/...
package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestE2E_ConvertFromAIAO fetches the Artificial Intelligence Assessment
// Ontology from https://w3id.org/aiao and verifies that ttl2d3 can parse and
// visualise it.
//
// This test requires an outbound HTTPS connection and is therefore excluded
// from the standard CI pipeline.  Run it manually or in a dedicated
// network-enabled job with:
//
//	go test -tags network ./e2e/...
func TestE2E_ConvertFromAIAO(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "aiao.html")

	stdout, stderr, err := runBinary(t,
		"convert",
		"--input", "https://w3id.org/aiao",
		"--out", outPath,
	)
	if err != nil {
		t.Fatalf("ttl2d3 convert --input https://w3id.org/aiao: %v\nstdout: %s\nstderr: %s",
			err, stdout, stderr)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "<!DOCTYPE html>") {
		t.Error("output missing <!DOCTYPE html>")
	}
	if !strings.Contains(content, "d3") {
		t.Error("output missing D3 reference")
	}
	// The AIAO ontology has nodes; the JSON embedded in the HTML must be non-trivial.
	if !strings.Contains(content, `"nodes"`) {
		t.Error("output missing \"nodes\" key in embedded graph data")
	}
}
