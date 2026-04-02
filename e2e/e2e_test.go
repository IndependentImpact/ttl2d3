//go:build integration

// Package e2e contains end-to-end tests that build the ttl2d3 binary and
// exercise the full conversion pipeline via the CLI.
package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// binaryPath is populated in TestMain after the binary is compiled.
var binaryPath string

func TestMain(m *testing.M) {
	// Compile the binary into a temp directory so all tests share one build.
	tmp, err := os.MkdirTemp("", "ttl2d3-e2e-*")
	if err != nil {
		panic("MkdirTemp: " + err.Error())
	}
	defer os.RemoveAll(tmp)

	binaryPath = filepath.Join(tmp, "ttl2d3")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/ttl2d3")
	cmd.Dir = repoRoot()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("go build failed: " + err.Error())
	}

	os.Exit(m.Run())
}

// repoRoot returns the absolute path to the repository root by walking up from
// this file's directory until go.mod is found.
func repoRoot() string {
	dir, err := filepath.Abs(".")
	if err != nil {
		panic("filepath.Abs: " + err.Error())
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("go.mod not found")
		}
		dir = parent
	}
}

// testdata returns the absolute path to a file under the repo testdata directory.
func testdata(name string) string {
	return filepath.Join(repoRoot(), "testdata", name)
}

// runBinary executes the ttl2d3 binary with args and returns stdout, stderr,
// and the exit error (nil on success).
func runBinary(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// TestE2E_Version checks that `ttl2d3 version` exits 0 and prints a version string.
func TestE2E_Version(t *testing.T) {
	stdout, _, err := runBinary(t, "version")
	if err != nil {
		t.Fatalf("ttl2d3 version: %v", err)
	}
	if !strings.Contains(stdout, "ttl2d3") {
		t.Errorf("version output = %q, want it to contain \"ttl2d3\"", stdout)
	}
}

// TestE2E_Help checks that `ttl2d3 --help` exits 0.
func TestE2E_Help(t *testing.T) {
	_, _, err := runBinary(t, "--help")
	if err != nil {
		t.Fatalf("ttl2d3 --help: %v", err)
	}
}

// TestE2E_ConvertHTMLToFile exercises the full pipeline for a Turtle file
// and verifies the output file is a valid HTML page.
func TestE2E_ConvertHTMLToFile(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "out.html")

	_, _, err := runBinary(t,
		"convert",
		"--input", testdata("simple.ttl"),
		"--output", "html",
		"--out", outPath,
	)
	if err != nil {
		t.Fatalf("ttl2d3 convert: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "<!DOCTYPE html>") {
		t.Error("HTML output missing <!DOCTYPE html>")
	}
	if !strings.Contains(content, "d3") {
		t.Error("HTML output missing D3 reference")
	}
}

// TestE2E_ConvertJSONToFile exercises the full pipeline for a Turtle file and
// verifies the output is well-formed JSON with the expected top-level keys.
func TestE2E_ConvertJSONToFile(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "out.json")

	_, _, err := runBinary(t,
		"convert",
		"--input", testdata("simple.ttl"),
		"--output", "json",
		"--out", outPath,
	)
	if err != nil {
		t.Fatalf("ttl2d3 convert: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}
	for _, key := range []string{"nodes", "links", "metadata"} {
		if _, ok := doc[key]; !ok {
			t.Errorf("JSON output missing key %q", key)
		}
	}
}

// TestE2E_ConvertJSONToStdout verifies that omitting --out writes to stdout.
func TestE2E_ConvertJSONToStdout(t *testing.T) {
	stdout, _, err := runBinary(t,
		"convert",
		"--input", testdata("simple.ttl"),
		"--output", "json",
	)
	if err != nil {
		t.Fatalf("ttl2d3 convert: %v", err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}
	if _, ok := doc["nodes"]; !ok {
		t.Error("stdout JSON missing \"nodes\" key")
	}
}

// TestE2E_ConvertHTMLToStdout verifies that HTML output goes to stdout.
func TestE2E_ConvertHTMLToStdout(t *testing.T) {
	stdout, _, err := runBinary(t,
		"convert",
		"--input", testdata("simple.ttl"),
		"--output", "html",
	)
	if err != nil {
		t.Fatalf("ttl2d3 convert: %v", err)
	}
	if !strings.Contains(stdout, "<!DOCTYPE html>") {
		t.Error("stdout HTML output missing <!DOCTYPE html>")
	}
}

// TestE2E_ConvertStdinInput verifies that `--input -` reads from stdin.
func TestE2E_ConvertStdinInput(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "out.json")

	turtleData, err := os.ReadFile(testdata("simple.ttl"))
	if err != nil {
		t.Fatalf("reading testdata: %v", err)
	}

	cmd := exec.Command(binaryPath,
		"convert",
		"--input", "-",
		"--format", "turtle",
		"--output", "json",
		"--out", outPath,
	)
	cmd.Stdin = strings.NewReader(string(turtleData))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ttl2d3 convert (stdin): %v\n%s", err, out)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}
	if _, ok := doc["nodes"]; !ok {
		t.Error("JSON output missing \"nodes\" key")
	}
}

// TestE2E_ConvertRDFXML exercises parsing a .owl file.
func TestE2E_ConvertRDFXML(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "out.json")

	_, _, err := runBinary(t,
		"convert",
		"--input", testdata("pizza.owl"),
		"--output", "json",
		"--out", outPath,
	)
	if err != nil {
		t.Fatalf("ttl2d3 convert pizza.owl: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}

	var nodes []json.RawMessage
	if err := json.Unmarshal(doc["nodes"], &nodes); err != nil {
		t.Fatalf("unmarshal nodes: %v", err)
	}
	if len(nodes) == 0 {
		t.Error("expected at least one node from pizza.owl, got none")
	}
}

// TestE2E_ConvertJSONLD exercises parsing a .jsonld file.
func TestE2E_ConvertJSONLD(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "out.json")

	_, _, err := runBinary(t,
		"convert",
		"--input", testdata("example.jsonld"),
		"--output", "json",
		"--out", outPath,
	)
	if err != nil {
		t.Fatalf("ttl2d3 convert example.jsonld: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}
	if _, ok := doc["nodes"]; !ok {
		t.Error("JSON output missing \"nodes\" key")
	}
}

// TestE2E_ConvertTitleFlag verifies that --title overrides the page title.
func TestE2E_ConvertTitleFlag(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "out.html")

	_, _, err := runBinary(t,
		"convert",
		"--input", testdata("simple.ttl"),
		"--output", "html",
		"--out", outPath,
		"--title", "E2E Test Title",
	)
	if err != nil {
		t.Fatalf("ttl2d3 convert: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	if !strings.Contains(string(data), "E2E Test Title") {
		t.Error("HTML output does not contain custom title \"E2E Test Title\"")
	}
}

// TestE2E_MissingInputFlag verifies that omitting --input exits non-zero.
func TestE2E_MissingInputFlag(t *testing.T) {
	_, _, err := runBinary(t, "convert")
	if err == nil {
		t.Error("expected non-zero exit when --input is missing, got nil error")
	}
}

// TestE2E_InvalidOutputFormat verifies that an unknown --output value exits non-zero.
func TestE2E_InvalidOutputFormat(t *testing.T) {
	_, _, err := runBinary(t,
		"convert",
		"--input", testdata("simple.ttl"),
		"--output", "svg",
	)
	if err == nil {
		t.Error("expected non-zero exit for invalid --output value, got nil error")
	}
}

// TestE2E_ForceParams verifies that D3 force parameters are accepted without error.
func TestE2E_ForceParams(t *testing.T) {
	_, _, err := runBinary(t,
		"convert",
		"--input", testdata("simple.ttl"),
		"--output", "json",
		"--link-distance", "120",
		"--charge-strength", "-500",
		"--collide-radius", "30",
		"--out", filepath.Join(t.TempDir(), "out.json"),
	)
	if err != nil {
		t.Fatalf("ttl2d3 convert with force params: %v", err)
	}
}

// TestE2E_ConvertFromURL verifies that a URL can be used as --input.
// A local httptest server serves a Turtle file so no real network is needed.
func TestE2E_ConvertFromURL(t *testing.T) {
ttlData, err := os.ReadFile(testdata("simple.ttl"))
if err != nil {
t.Fatalf("reading testdata: %v", err)
}

srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "text/turtle")
fmt.Fprint(w, string(ttlData))
}))
defer srv.Close()

outPath := filepath.Join(t.TempDir(), "out.html")
stdout, stderr, err := runBinary(t,
"convert",
"--input", srv.URL+"/onto.ttl",
"--out", outPath,
)
if err != nil {
t.Fatalf("ttl2d3 convert from URL: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
}

data, err := os.ReadFile(outPath)
if err != nil {
t.Fatalf("reading output: %v", err)
}
if !strings.Contains(string(data), "<!DOCTYPE html>") {
t.Error("HTML output missing <!DOCTYPE html>")
}
}

// TestE2E_ConvertFromURL_404 verifies that a 404 URL exits with a non-zero
// exit code and a useful error message.
func TestE2E_ConvertFromURL_404(t *testing.T) {
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
http.NotFound(w, r)
}))
defer srv.Close()

_, stderr, err := runBinary(t,
"convert",
"--input", srv.URL+"/missing.ttl",
)
if err == nil {
t.Error("expected non-zero exit for 404 URL, got nil")
}
if !strings.Contains(stderr, "404") {
t.Errorf("stderr %q should mention 404", stderr)
}
}
