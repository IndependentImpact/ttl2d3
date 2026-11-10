# ttl2d3

[![CI](https://github.com/IndependentImpact/ttl2d3/actions/workflows/ci.yml/badge.svg)](https://github.com/IndependentImpact/ttl2d3/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/IndependentImpact/ttl2d3)](go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/IndependentImpact/ttl2d3)](https://goreportcard.com/report/github.com/IndependentImpact/ttl2d3)
[![License](https://img.shields.io/github/license/IndependentImpact/ttl2d3)](LICENSE)

> Convert semantic-web ontologies and concept schemes to interactive D3.js
> visualisations, including force-directed and workflow-oriented layouts.

`ttl2d3` is a Go CLI tool that reads ontologies or concept schemes in common
RDF formats (`.ttl`, `.owl`, `.jsonld`, `.rdf`) – from a **local file, stdin,
or an HTTP/HTTPS URL** – and produces either:

* a **standalone D3 JSON object** ready to embed in any webpage, or
* a **self-contained HTML page** with an interactive D3 visualisation
  (zoom, pan, drag, tooltips, search), either as a D3 force-directed graph
  (default) or, with `--workflowplan`, as a process / swimlane diagram of an
  `indimp:WorkflowPlan`.

---

## Status

✅ **v0.1.0 released** – see
[Releases](https://github.com/IndependentImpact/ttl2d3/releases).  
See [`plan.md`](plan.md) for the full implementation roadmap and
[`spec.md`](spec.md) for the detailed specification.

---

## Installation

Download the archive for your platform (Linux, macOS, Windows; amd64 and
arm64) from the
[latest release](https://github.com/IndependentImpact/ttl2d3/releases/latest),
verify it against `checksums.txt`, and put `ttl2d3` on your `PATH`.

Or build from source (requires the Go version in `go.mod`):

```bash
git clone https://github.com/IndependentImpact/ttl2d3.git
cd ttl2d3
go build -o ttl2d3 ./cmd/ttl2d3
```

To run the locally built binary, either use `./ttl2d3` from the repo root
or move it onto your `PATH` (the file is already executable after `go build`).

---

## Usage

```
./ttl2d3 [flags]
./ttl2d3 convert [flags]   (default sub-command)
./ttl2d3 version
```

### Flags

| Flag                   | Short | Default     | Description                                               |
|------------------------|-------|-------------|-----------------------------------------------------------|
| `--input`              | `-i`  | *(required)*| Input file path, `-` for stdin, or an `http(s)://` URL   |
| `--output`             | `-o`  | `html`      | Output format: `html` or `json`                          |
| `--out`                | `-O`  | stdout      | Output file path                                         |
| `--format`             | `-f`  | auto-detect | Input format: `turtle`, `rdfxml`, `jsonld`               |
| `--title`              |       | ontology IRI| Title shown in HTML output                               |
| `--simplify`           |       | false       | Render `owl:unionOf` as direct edges instead of a union node |
| `--workflowplan`       |       | false       | Render `indimp:WorkflowPlan` resources as a process / swimlane diagram (HTML only) |
| `--node-spacing`       |       | `180`       | Column width in pixels for `--workflowplan`              |
| `--link-distance`      |       | `80`        | D3 force link distance (force layout only)               |
| `--charge-strength`    |       | `-300`      | D3 many-body charge strength (force layout only)         |
| `--collide-radius`     |       | `20`        | D3 collision-detection radius (force layout only)        |
| `--gravity-strength`   |       | `0.1`       | D3 gravity pulling nodes toward the centre (0–1, force layout only) |
| `--verbose`            | `-v`  | false       | Enable debug logging                                     |
| `--help`               | `-h`  | —           | Show help                                                |

### Examples

```bash
# Generate a self-contained HTML diagram (force layout, default)
./ttl2d3 convert --input my-ontology.ttl --out diagram.html

# Workflow plan as a process / swimlane diagram
./ttl2d3 convert --input reference/workflow-plan.ttl --workflowplan --out plan.html

# Fetch an ontology directly from a URL
./ttl2d3 convert --input https://w3id.org/aiao --out aiao.html

# Generate D3 graph JSON only
./ttl2d3 convert --input my-ontology.ttl --output json --out graph.json

# Read from stdin, write HTML to stdout
cat my-ontology.ttl | ./ttl2d3 convert --input - --format turtle

# Print version
./ttl2d3 version
```

---

## Supported Input Formats

| Extension           | Format   |
|---------------------|----------|
| `.ttl`              | Turtle   |
| `.owl` / `.rdf`     | RDF/XML  |
| `.jsonld` / `.json` | JSON-LD  |

When `--input` is an HTTP/HTTPS URL the format is resolved from the `Content-Type`
response header (using HTTP content negotiation), then from the URL path
extension, and finally from byte-sniffing as a last resort.

Supported `Content-Type` values:

| MIME type                             | Format   |
|---------------------------------------|----------|
| `text/turtle`, `application/x-turtle` | Turtle   |
| `application/rdf+xml`                 | RDF/XML  |
| `application/ld+json`                 | JSON-LD  |

---

## Notes

- Multiple object properties with the same domain and range are preserved as distinct links, even if their labels match.
- Domain/range IRIs imply class nodes even without explicit class declarations.
- `owl:unionOf` domains/ranges are visualised as explicit union nodes linked to their member classes.
- HTML output distinguishes local vs imported classes and lists namespaces derived from node IRIs.
- Parallel edges and inverse property pairs (A→B and B→A) are drawn as separate curved arcs.
- In the HTML view, dragged nodes stay pinned; double-click a node or use **Unpin all** to release them.
- `--workflowplan` applies only to HTML output; combining it with `--output json` is an error.

---

## Development

```bash
# Format
goimports -w .

# Lint
golangci-lint run ./...

# Vet
go vet ./...

# Build
go build ./...

# Test (unit)
go test ./...

# Test (unit + integration)
go test -tags integration ./...

# Test (network – requires outbound HTTPS, not run in CI)
go test -tags network ./e2e/...

# Check for known vulnerabilities
govulncheck ./...

# Build CLI binary
go build -o ttl2d3 ./cmd/ttl2d3
```

---

## Documentation

* [`agents.md`](agents.md) – Repo-wide instructions for contributors and AI agents
* [`spec.md`](spec.md) – Full requirements and architecture specification
* [`plan.md`](plan.md) – Phased implementation plan with progress tracking
* [`CONTRIBUTING.md`](CONTRIBUTING.md) – Contribution guidelines

---

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for detailed contribution guidelines.

## License

Copyright 2026 Nova Institute NPC. Licensed under the
[Apache License, Version 2.0](LICENSE).
