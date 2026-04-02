package fetcher_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/IndependentImpact/ttl2d3/internal/config"
	"github.com/IndependentImpact/ttl2d3/internal/fetcher"
)

// ---------------------------------------------------------------------------
// IsURL
// ---------------------------------------------------------------------------

func TestIsURL(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"http://example.org/onto.ttl", true},
		{"https://w3id.org/aiao", true},
		{"https://example.org/onto#fragment", true},
		{"/local/path/file.ttl", false},
		{"relative/path.ttl", false},
		{"-", false},
		{"", false},
		{"ftp://example.org/onto.ttl", false},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			if got := fetcher.IsURL(tc.input); got != tc.want {
				t.Errorf("IsURL(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Fetch – successful responses
// ---------------------------------------------------------------------------

func TestFetch_OK_Turtle(t *testing.T) {
	const body = `@prefix owl: <http://www.w3.org/2002/07/owl#> . <http://example.org/A> a owl:Class .`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/turtle")
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	rc, format, err := fetcher.Fetch(context.Background(), srv.URL+"/onto.ttl", config.InputAuto)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	defer rc.Close()

	if format != config.InputTurtle {
		t.Errorf("Fetch() format = %q, want %q", format, config.InputTurtle)
	}

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if string(got) != body {
		t.Errorf("Fetch() body = %q, want %q", string(got), body)
	}
}

func TestFetch_OK_RDFXML(t *testing.T) {
	const body = `<?xml version="1.0"?><rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"/>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rdf+xml")
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	rc, format, err := fetcher.Fetch(context.Background(), srv.URL+"/onto.rdf", config.InputAuto)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	defer rc.Close()

	if format != config.InputRDFXML {
		t.Errorf("Fetch() format = %q, want %q", format, config.InputRDFXML)
	}
}

func TestFetch_OK_JSONLD(t *testing.T) {
	const body = `{"@context":{},"@id":"http://example.org/A","@type":"http://www.w3.org/2002/07/owl#Class"}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/ld+json")
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	rc, format, err := fetcher.Fetch(context.Background(), srv.URL+"/onto.jsonld", config.InputAuto)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	defer rc.Close()

	if format != config.InputJSONLD {
		t.Errorf("Fetch() format = %q, want %q", format, config.InputJSONLD)
	}
}

func TestFetch_OK_ContentTypeOverridesExtension(t *testing.T) {
	// URL has .ttl extension but server returns JSON-LD; Content-Type should win.
	const body = `{"@context":{}}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/ld+json")
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	rc, format, err := fetcher.Fetch(context.Background(), srv.URL+"/onto.ttl", config.InputAuto)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	defer rc.Close()

	if format != config.InputJSONLD {
		t.Errorf("Fetch() format = %q, want Content-Type to override extension", format)
	}
}

func TestFetch_OK_FallbackExtension(t *testing.T) {
	// Server returns text/plain; format should fall back to URL extension (.ttl).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, `@prefix owl: <http://www.w3.org/2002/07/owl#> .`)
	}))
	defer srv.Close()

	rc, format, err := fetcher.Fetch(context.Background(), srv.URL+"/onto.ttl", config.InputAuto)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	defer rc.Close()

	if format != config.InputTurtle {
		t.Errorf("Fetch() format = %q, want %q (fallback from extension)", format, config.InputTurtle)
	}
}

func TestFetch_OK_HintOverridesContentType(t *testing.T) {
	// Explicit hint should be returned regardless of Content-Type header.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/turtle")
		fmt.Fprint(w, `@prefix owl: <http://www.w3.org/2002/07/owl#> .`)
	}))
	defer srv.Close()

	rc, format, err := fetcher.Fetch(context.Background(), srv.URL+"/onto", config.InputRDFXML)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	defer rc.Close()

	if format != config.InputRDFXML {
		t.Errorf("Fetch() format = %q, want explicit hint %q to be returned", format, config.InputRDFXML)
	}
}

func TestFetch_AcceptHeader_Turtle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept")
		if !strings.Contains(accept, "text/turtle") {
			t.Errorf("Accept header %q does not contain text/turtle", accept)
		}
		w.Header().Set("Content-Type", "text/turtle")
		fmt.Fprint(w, `@prefix owl: <http://www.w3.org/2002/07/owl#> .`)
	}))
	defer srv.Close()

	rc, _, err := fetcher.Fetch(context.Background(), srv.URL+"/onto", config.InputTurtle)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	rc.Close()
}

// ---------------------------------------------------------------------------
// Fetch – error cases
// ---------------------------------------------------------------------------

func TestFetch_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	_, _, err := fetcher.Fetch(context.Background(), srv.URL+"/missing.ttl", config.InputAuto)
	if err == nil {
		t.Fatal("Fetch() expected error for 404, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error %q should mention 404", err.Error())
	}
}

func TestFetch_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, _, err := fetcher.Fetch(context.Background(), srv.URL+"/onto.ttl", config.InputAuto)
	if err == nil {
		t.Fatal("Fetch() expected error for 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error %q should mention 500", err.Error())
	}
}

func TestFetch_BodyTooLarge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/turtle")
		// Write more than MaxBodyBytes (50 MiB).
		chunk := strings.Repeat("x", 1024*1024) // 1 MiB chunk
		for i := 0; i < 52; i++ {
			if _, err := fmt.Fprint(w, chunk); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	rc, _, err := fetcher.Fetch(context.Background(), srv.URL+"/large.ttl", config.InputAuto)
	if err != nil {
		t.Fatalf("Fetch() unexpected error before read: %v", err)
	}
	defer rc.Close()

	_, readErr := io.ReadAll(rc)
	if readErr == nil {
		t.Fatal("expected error when reading oversized body, got nil")
	}
	if !strings.Contains(readErr.Error(), "limit") {
		t.Errorf("error %q should mention limit", readErr.Error())
	}
}

func TestFetch_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until the client disconnects.
		select {
		case <-r.Context().Done():
		case <-time.After(10 * time.Second):
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, _, err := fetcher.Fetch(ctx, srv.URL+"/slow.ttl", config.InputAuto)
	if err == nil {
		t.Fatal("Fetch() expected timeout error, got nil")
	}
	// Accept either "deadline" or "context" or "timeout" in error message.
	lower := strings.ToLower(err.Error())
	if !strings.Contains(lower, "deadline") && !strings.Contains(lower, "timeout") && !strings.Contains(lower, "context") {
		t.Errorf("error %q should indicate timeout/deadline", err.Error())
	}
}

func TestFetch_UnsupportedScheme(t *testing.T) {
	_, _, err := fetcher.Fetch(context.Background(), "ftp://example.org/onto.ttl", config.InputAuto)
	if err == nil {
		t.Fatal("Fetch() expected error for ftp://, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("error %q should mention unsupported scheme", err.Error())
	}
}
