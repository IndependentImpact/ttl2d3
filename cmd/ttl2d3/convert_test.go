//go:build integration

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IndependentImpact/ttl2d3/internal/config"
)

// captureStdout temporarily replaces os.Stdout with a pipe so that
// runConvert output can be captured when cfg.Out is "".
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}

	orig := os.Stdout
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = orig

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("reading captured stdout: %v", err)
	}
	return buf.String()
}

// testdata returns the absolute path to a file under the repo testdata directory.
func testdataPath(name string) string {
	return filepath.Join("..", "..", "testdata", name)
}

func TestRunConvert_HTMLToFile(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "out.html")
	cfg := config.Config{
		Input:          testdataPath("simple.ttl"),
		Output:         config.OutputHTML,
		Out:            outPath,
		LinkDistance:   80,
		ChargeStrength: -300,
		CollideRadius:  20,
	}

	if err := runConvert(cfg); err != nil {
		t.Fatalf("runConvert() error = %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	content := string(data)

	// Basic sanity checks on the HTML output.
	if !strings.Contains(content, "<!DOCTYPE html>") {
		t.Error("output missing <!DOCTYPE html>")
	}
	if !strings.Contains(content, "d3") {
		t.Error("output missing D3 reference")
	}
}

func TestRunConvert_JSONToFile(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "out.json")
	cfg := config.Config{
		Input:          testdataPath("simple.ttl"),
		Output:         config.OutputJSON,
		Out:            outPath,
		LinkDistance:   80,
		ChargeStrength: -300,
		CollideRadius:  20,
	}

	if err := runConvert(cfg); err != nil {
		t.Fatalf("runConvert() error = %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	content := string(data)

	for _, want := range []string{`"nodes"`, `"links"`, `"metadata"`} {
		if !strings.Contains(content, want) {
			t.Errorf("JSON output missing key %q", want)
		}
	}
}

func TestRunConvert_HTMLToStdout(t *testing.T) {
	cfg := config.Config{
		Input:          testdataPath("simple.ttl"),
		Output:         config.OutputHTML,
		Out:            "", // stdout
		LinkDistance:   80,
		ChargeStrength: -300,
		CollideRadius:  20,
	}

	output := captureStdout(t, func() {
		if err := runConvert(cfg); err != nil {
			t.Errorf("runConvert() error = %v", err)
		}
	})

	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Error("stdout output missing <!DOCTYPE html>")
	}
}

func TestRunConvert_JSONToStdout(t *testing.T) {
	cfg := config.Config{
		Input:          testdataPath("simple.ttl"),
		Output:         config.OutputJSON,
		Out:            "", // stdout
		LinkDistance:   80,
		ChargeStrength: -300,
		CollideRadius:  20,
	}

	output := captureStdout(t, func() {
		if err := runConvert(cfg); err != nil {
			t.Errorf("runConvert() error = %v", err)
		}
	})

	if !strings.Contains(output, `"nodes"`) {
		t.Error("stdout JSON output missing \"nodes\" key")
	}
}

func TestRunConvert_StdinInput(t *testing.T) {
	// Open a test Turtle file and pipe it through stdin.
	f, err := os.Open(testdataPath("simple.ttl"))
	if err != nil {
		t.Fatalf("os.Open: %v", err)
	}
	defer f.Close()

	origStdin := os.Stdin
	os.Stdin = f
	defer func() { os.Stdin = origStdin }()

	outPath := filepath.Join(t.TempDir(), "out.json")
	cfg := config.Config{
		Input:          "-",
		Output:         config.OutputJSON,
		Out:            outPath,
		Format:         config.InputTurtle,
		LinkDistance:   80,
		ChargeStrength: -300,
		CollideRadius:  20,
	}

	if err := runConvert(cfg); err != nil {
		t.Fatalf("runConvert() error = %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	if !strings.Contains(string(data), `"nodes"`) {
		t.Error("JSON output missing \"nodes\" key")
	}
}

func TestRunConvert_TitleOverride(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "out.html")
	cfg := config.Config{
		Input:          testdataPath("simple.ttl"),
		Output:         config.OutputHTML,
		Out:            outPath,
		Title:          "My Custom Title",
		LinkDistance:   80,
		ChargeStrength: -300,
		CollideRadius:  20,
	}

	if err := runConvert(cfg); err != nil {
		t.Fatalf("runConvert() error = %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	if !strings.Contains(string(data), "My Custom Title") {
		t.Error("HTML output does not contain custom title")
	}
}

func TestRunConvert_AllFormats(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		format   config.InputFormat
		wantJSON bool
	}{
		{name: "turtle", input: "simple.ttl", wantJSON: true},
		{name: "skos", input: "skos.ttl", wantJSON: true},
		{name: "rdfxml", input: "pizza.owl", wantJSON: true},
		{name: "jsonld", input: "example.jsonld", wantJSON: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			outPath := filepath.Join(t.TempDir(), "out.json")
			cfg := config.Config{
				Input:          testdataPath(tc.input),
				Output:         config.OutputJSON,
				Out:            outPath,
				Format:         tc.format,
				LinkDistance:   80,
				ChargeStrength: -300,
				CollideRadius:  20,
			}

			if err := runConvert(cfg); err != nil {
				t.Fatalf("runConvert() error = %v", err)
			}

			data, err := os.ReadFile(outPath)
			if err != nil {
				t.Fatalf("reading output: %v", err)
			}
			if !strings.Contains(string(data), `"nodes"`) {
				t.Errorf("output missing \"nodes\" key")
			}
		})
	}
}

func TestRunConvert_InvalidInput(t *testing.T) {
	cfg := config.Config{
		Input:          "/nonexistent/path/file.ttl",
		Output:         config.OutputHTML,
		LinkDistance:   80,
		ChargeStrength: -300,
		CollideRadius:  20,
	}

	if err := runConvert(cfg); err == nil {
		t.Error("expected error for nonexistent input file, got nil")
	}
}
