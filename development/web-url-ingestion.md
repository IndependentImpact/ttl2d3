# Feature Specification: Web URL Ingestion

**Status:** Implemented  
**Issue:** [Plan ingestion of ttl, json-ld or rdf/owl from web address](https://github.com/IndependentImpact/ttl2d3/issues)  
**Last updated:** 2026-04-02

---

## 1. Overview

Allow `ttl2d3 convert` to accept an HTTP or HTTPS URL as the `--input`
argument in addition to a local file path or `-` for stdin.  When a URL is
supplied the tool fetches the remote ontology, auto-detects its serialisation
format from the HTTP `Content-Type` response header (or falls back to URL path
extension / byte sniffing), and passes the content through the existing
parse → transform → render pipeline without any other behavioural changes.

### Example usage

```bash
# Self-contained HTML from a public ontology
ttl2d3 convert --input https://w3id.org/aiao --out aiao.html

# D3 JSON from a Turtle file served over HTTP
ttl2d3 convert --input https://example.org/ontology.ttl --output json

# Force Turtle parsing (overrides Content-Type)
ttl2d3 convert --input https://example.org/onto --format turtle --out out.html
```

---

## 2. Background

Many public ontologies (AIAO, schema.org, Dublin Core, FOAF, SKOS, …) are
published at `http://` or `https://` IRIs and served with proper semantic-web
content negotiation.  Without URL support, users must download the file
manually before running `ttl2d3`.  Enabling URL ingestion removes that
friction and enables the tool to participate in publish/subscribe pipelines
such as CI linting or automated documentation generation.

### Content negotiation

Semantic-web servers typically implement **HTTP content negotiation**: the
server inspects the client's `Accept` header and returns the format it prefers
from those available.  The relevant MIME types are:

| Format   | Primary MIME type         | Secondary / legacy MIME types              |
|----------|---------------------------|--------------------------------------------|
| Turtle   | `text/turtle`             | `application/x-turtle`                     |
| RDF/XML  | `application/rdf+xml`     | `application/xml`, `text/xml`              |
| JSON-LD  | `application/ld+json`     | `application/json`                         |

The tool must send an `Accept` header so that servers that support multiple
formats return a machine-readable one rather than an HTML page.

---

## 3. Scope

### In scope
* Fetch any `http://` or `https://` URL supplied to `--input`.
* Set a sensible `Accept` header to trigger content negotiation.
* Detect the returned format from the `Content-Type` response header.
* Fall back to URL path-extension detection and byte sniffing when the
  `Content-Type` is absent, `application/octet-stream`, or `text/plain`.
* Respect an explicit `--format` flag (skip content-type detection).
* Follow HTTP redirects (the standard Go HTTP client does this by default,
  up to 10 hops).
* Enforce a configurable request timeout (default 30 s).
* Enforce a response-body size limit (default 50 MiB) to prevent OOM on
  unexpectedly large responses.
* Produce user-friendly error messages for common HTTP failures (4xx, 5xx,
  network error, timeout).

### Out of scope
* HTTP authentication (Bearer, Basic, API keys).
* HTTPS certificate pinning.
* Caching of fetched resources.
* Fetching from non-HTTP schemes (`ftp://`, `file://`, `s3://`, …).
* Parsing HTML pages to discover embedded RDF (RDFa, Microdata).

---

## 4. Architecture

### 4.1 New package: `internal/fetcher`

A new package handles all HTTP concerns, keeping network logic out of the
converter and parser.

```
internal/
  fetcher/
    fetch.go        – exported Fetch() function and helpers
    fetch_test.go   – unit tests with httptest.NewServer mock
```

#### `fetch.go` – public API

```go
package fetcher

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"

    "github.com/IndependentImpact/ttl2d3/internal/config"
)

// DefaultTimeout is the HTTP request timeout applied when no deadline is set
// in the supplied context.
const DefaultTimeout = 30 * time.Second

// MaxBodyBytes is the maximum number of bytes read from an HTTP response body.
// Responses larger than this are rejected with an error.
const MaxBodyBytes = 50 * 1024 * 1024 // 50 MiB

// IsURL reports whether s looks like an HTTP or HTTPS URL.
func IsURL(s string) bool {
    return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// Fetch retrieves the resource at url and returns its body as a ReadCloser,
// the resolved InputFormat (from Content-Type or URL extension), and any
// error.  The caller must close the returned ReadCloser.
//
// hint may be config.InputAuto to trigger automatic detection, or an explicit
// format to override Content-Type detection (the Accept header is still sent
// for the hinted format so the server returns an appropriate representation).
//
// A 30-second timeout is applied if ctx has no deadline set.
func Fetch(ctx context.Context, url string, hint config.InputFormat) (io.ReadCloser, config.InputFormat, error)
```

#### `fetch.go` – Accept header construction

| hint            | Accept header value                                                                         |
|-----------------|---------------------------------------------------------------------------------------------|
| `InputAuto`     | `text/turtle, application/ld+json;q=0.9, application/rdf+xml;q=0.8, application/xml;q=0.7, */*;q=0.1` |
| `InputTurtle`   | `text/turtle, application/x-turtle;q=0.9, */*;q=0.1`                                       |
| `InputRDFXML`   | `application/rdf+xml, application/xml;q=0.9, text/xml;q=0.8, */*;q=0.1`                    |
| `InputJSONLD`   | `application/ld+json, application/json;q=0.9, */*;q=0.1`                                   |

#### `fetch.go` – Content-Type → InputFormat mapping

```go
// FormatFromContentType returns the InputFormat inferred from a Content-Type
// header value.  It strips any parameters (e.g. "charset=utf-8") before
// matching.  Returns config.InputAuto when the MIME type is unknown.
func FormatFromContentType(ct string) config.InputFormat
```

| Content-Type (after parameter stripping)                   | InputFormat     |
|------------------------------------------------------------|-----------------|
| `text/turtle`                                              | `InputTurtle`   |
| `application/x-turtle`                                     | `InputTurtle`   |
| `text/n3`                                                  | `InputTurtle`   |
| `application/n-triples`                                    | *(unsupported)* |
| `application/rdf+xml`                                      | `InputRDFXML`   |
| `application/xml`                                          | `InputRDFXML`   |
| `text/xml`                                                 | `InputRDFXML`   |
| `application/ld+json`                                      | `InputJSONLD`   |
| `application/json`                                         | `InputJSONLD`   |
| anything else (including `text/html`, `text/plain`, …)     | `InputAuto`     |

When the resolved format is still `InputAuto` after Content-Type inspection,
`Fetch` calls the existing `parser.DetectFormat(url)` (extension-based) and
returns `InputAuto` if that also fails, letting the byte-sniffing path in
`parser.Parse` make the final determination.

### 4.2 Changes to `internal/parser/detect.go`

Export the Content-Type → InputFormat mapping as a standalone function
`FormatFromContentType` so it can be tested independently and reused by
future callers:

```go
// FormatFromContentType returns the InputFormat best matching the supplied
// Content-Type header value.  Parameters such as "charset=utf-8" are ignored.
// Returns config.InputAuto when no known RDF MIME type is found.
func FormatFromContentType(contentType string) config.InputFormat
```

> **Note:** This function lives in `internal/parser` (not `internal/fetcher`)
> because format detection is a parser concern.  `internal/fetcher` imports
> `internal/parser` to use it.

### 4.3 Changes to `cmd/ttl2d3/convert.go`

In `runConvert`, add a new branch before the existing file-open logic:

```go
// Step 1 – Open input reader.
if fetcher.IsURL(cfg.Input) {
    rc, detectedFormat, err := fetcher.Fetch(context.Background(), cfg.Input, cfg.Format)
    if err != nil {
        return fmt.Errorf("convert: fetching URL: %w", err)
    }
    defer rc.Close()
    r = rc
    filename = cfg.Input
    // Only override format when auto-detect is still active.
    if cfg.Format == config.InputAuto {
        cfg.Format = detectedFormat
    }
    slog.Debug("fetched URL", "url", cfg.Input, "format", cfg.Format)
} else if cfg.Input == "-" {
    // … existing stdin path …
} else {
    // … existing file path …
}
```

### 4.4 Changes to `internal/config/config.go`

No structural changes are needed.  The `Validate` method already accepts any
non-empty string for `Input`; URL strings satisfy that condition.

---

## 5. Security Considerations

| Risk | Mitigation |
|------|-----------|
| SSRF (Server-Side Request Forgery) | The tool is a CLI, not a server; the user who runs it controls the URL. No SSRF risk. |
| Redirect to internal address | CLI users are expected to control their network; no block list is required in v1. |
| Excessively large response (OOM) | Response body wrapped with `io.LimitReader(body, MaxBodyBytes)`. Returns an error if the limit is exceeded. |
| Slow server / hanging connection | `context.WithTimeout(ctx, DefaultTimeout)` applied to the HTTP request. |
| Insecure scheme (`http://`) | Allowed; the user is responsible for choosing `https://` when needed. |
| TLS certificate errors | Default Go `http.Client` behaviour: validate system CAs; return error on invalid cert. |
| `#nosec` / gosec suppressions | Annotate any necessary gosec-flagged lines with inline explanations. |

---

## 6. Error Messages

| Condition | Error text |
|-----------|-----------|
| Non-200 HTTP status | `fetcher: HTTP GET <url>: unexpected status 404 Not Found` |
| Network/DNS error | `fetcher: HTTP GET <url>: <net/http error>` |
| Response too large | `fetcher: response body exceeds 50 MiB limit` |
| Request timeout | `fetcher: HTTP GET <url>: context deadline exceeded` |
| Unsupported scheme | `fetcher: unsupported URL scheme "ftp"` |

---

## 7. Implementation Checklist

Track each item with `[x]` when done.

### Phase 1 – Package `internal/fetcher`
- [x] Create `internal/fetcher/fetch.go`:
  - [x] `IsURL(s string) bool`
  - [x] `acceptHeader(hint config.InputFormat) string`
  - [x] `Fetch(ctx, url, hint) (io.ReadCloser, config.InputFormat, error)`
    - [x] Validate that the scheme is `http` or `https`
    - [x] Apply `context.WithTimeout(ctx, DefaultTimeout)` when ctx has no
          deadline
    - [x] Build and send `Accept` header from hint
    - [x] Check HTTP response status; return error for non-200
    - [x] Call `parser.FormatFromContentType(resp.Header.Get("Content-Type"))`
    - [x] If still `InputAuto`, call `parser.DetectFormat(url)` (ignoring error)
    - [x] Wrap body with `io.LimitReader(body, MaxBodyBytes)`
    - [x] Return `io.NopCloser` wrapping the limit-reader plus resolved format

### Phase 2 – `internal/parser/detect.go`
- [x] Add `FormatFromContentType(contentType string) config.InputFormat`
  - [x] Strip MIME type parameters (split on `;`, take first part, trim space)
  - [x] Map MIME types per §4.2 table
  - [x] Return `config.InputAuto` for unknown types
- [x] Add unit tests for `FormatFromContentType` in `internal/parser/detect_test.go`

### Phase 3 – `cmd/ttl2d3/convert.go`
- [x] Import `internal/fetcher` and `context`
- [x] Add URL branch in `runConvert` before the stdin/file branches (§4.3)
- [x] Pass resolved format into `parser.Parse` via updated `cfg.Format`

### Phase 4 – Tests

#### Unit tests (`internal/fetcher/fetch_test.go`)
- [x] `TestIsURL` – verify URL detection for `http://`, `https://`, file paths,
      `-`, empty string
- [x] `TestFetch_OK_Turtle` – mock server returns 200 with `text/turtle` body;
      assert correct format and body content
- [x] `TestFetch_OK_RDFXML` – mock server returns `application/rdf+xml`
- [x] `TestFetch_OK_JSONLD` – mock server returns `application/ld+json`
- [x] `TestFetch_OK_ContentTypeOverridesExtension` – URL has `.ttl` extension
      but server returns `application/ld+json`; assert JSON-LD wins
- [x] `TestFetch_OK_FallbackExtension` – server returns `text/plain`; assert
      format falls back to extension detection from URL
- [x] `TestFetch_NotFound` – mock server returns 404; assert error contains
      "404"
- [x] `TestFetch_BodyTooLarge` – mock server streams > 50 MiB; assert error
      contains "limit"
- [x] `TestFetch_Timeout` – mock server hangs; assert error contains "deadline"
      or "timeout"
- [x] `TestFetch_UnsupportedScheme` – URL with `ftp://`; assert error contains
      "unsupported"

#### Integration tests (`cmd/ttl2d3/convert_test.go`)
- [x] `TestRunConvert_URLInput_HTML` – use `httptest.NewServer` to serve
      `testdata/simple.ttl` with `text/turtle`; call `runConvert`; assert no
      error and HTML output contains `<!DOCTYPE html>`
- [x] `TestRunConvert_URLInput_JSON` – as above but `--output json`; assert
      valid JSON with `nodes` and `links` keys

#### E2E tests (`e2e/e2e_test.go`)
- [x] `TestE2E_ConvertFromURL` – use `httptest.NewServer` within the test to
      serve `testdata/simple.ttl`; invoke binary with `--input <server_url>`;
      assert exit code 0 and non-empty HTML output

#### Network integration test (build tag `network`)
- [x] `TestE2E_ConvertFromAIAO` – fetch `https://w3id.org/aiao` (real network);
      assert exit code 0 and output contains expected class IRIs; guarded by
      `//go:build network` so it is **not** run in CI by default

### Phase 5 – Documentation
- [x] Update `README.md` – add URL input to the usage section and examples
- [x] Update `cmd/ttl2d3/convert.go` Long description and Example fields

### Phase 6 – Validation
- [ ] Run `go build ./...` – zero errors
- [ ] Run `go vet ./...` – zero warnings
- [ ] Run `golangci-lint run ./...` – zero findings
- [ ] Run `go test ./...` – all unit tests pass
- [ ] Run `go test -tags integration ./...` – all integration + e2e tests pass
- [ ] Manually run: `ttl2d3 convert --input https://w3id.org/aiao --out /tmp/aiao.html`
      and open the HTML in a browser

---

## 8. Testing the w3id.org/aiao Ontology

The [Artificial Intelligence Assessment Ontology (AIAO)](https://w3id.org/aiao)
is published at `https://w3id.org/aiao` and resolves via a `303 See Other`
redirect to the authoritative Turtle or RDF/XML document depending on the
client's `Accept` header.

### Verification steps

```bash
# 1. Verify content negotiation via curl
curl -L -H "Accept: text/turtle" -I https://w3id.org/aiao

# 2. Build the binary
go build -o /tmp/ttl2d3 ./cmd/ttl2d3

# 3. Fetch and visualise
/tmp/ttl2d3 convert --input https://w3id.org/aiao --out /tmp/aiao.html

# 4. Open in browser
open /tmp/aiao.html      # macOS
xdg-open /tmp/aiao.html  # Linux
```

### Expected outcome
* Exit code 0.
* `/tmp/aiao.html` is a valid HTML file containing a D3 force-directed graph.
* The graph should show AIAO classes (`AssessmentMethod`, `EthicalRisk`, …)
  and their properties.
* Run with `--verbose` to see the resolved format and triple count in the log.

---

## 9. File Inventory

| File | Change type | Notes |
|------|-------------|-------|
| `development/web-url-ingestion.md` | **new** | This document |
| `internal/fetcher/fetch.go` | **new** | HTTP fetch + format detection |
| `internal/fetcher/fetch_test.go` | **new** | Unit tests with mock server |
| `internal/parser/detect.go` | **modified** | Add `FormatFromContentType` |
| `internal/parser/detect_test.go` | **modified** | Tests for new function |
| `cmd/ttl2d3/convert.go` | **modified** | URL branch in `runConvert` |
| `cmd/ttl2d3/convert_test.go` | **modified** | Integration tests for URL input |
| `e2e/e2e_test.go` | **modified** | E2E test with local mock server + network-gated AIAO test |
| `README.md` | **modified** | Document URL input |
