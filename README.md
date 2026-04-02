# ttl2d3

[![CI](https://github.com/IndependentImpact/ttl2d3/actions/workflows/ci.yml/badge.svg)](https://github.com/IndependentImpact/ttl2d3/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/IndependentImpact/ttl2d3)](go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/IndependentImpact/ttl2d3)](https://goreportcard.com/report/github.com/IndependentImpact/ttl2d3)
[![License](https://img.shields.io/github/license/IndependentImpact/ttl2d3)](LICENSE)

> Convert semantic-web ontologies and concept schemes to interactive D3.js
> force-directed graph visualisations.

`ttl2d3` is a Go CLI tool that reads ontologies or concept schemes in common
RDF formats (`.ttl`, `.owl`, `.jsonld`, `.rdf`) – from a **local file, stdin,
or an HTTP/HTTPS URL** – and produces either:

* a **standalone D3 JSON object** ready to embed in any webpage, or
* a **self-contained HTML page** with an interactive D3 force-directed graph
  (zoom, pan, drag, tooltips, search) – similar to WebVOWL but output as a
  single static file.

---

## Status

✅ **v0.1.0 release candidate** – all phases 1–11 complete.  
See [`plan.md`](plan.md) for the full implementation roadmap and
[`spec.md`](spec.md) for the detailed specification.

---

## Installation

```bash
# Requires Go 1.22+
go install github.com/IndependentImpact/ttl2d3/cmd/ttl2d3@latest
```

Or build from source:

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

| Flag                | Short | Default     | Description                                               |
|---------------------|-------|-------------|-----------------------------------------------------------|
| `--input`           | `-i`  | *(required)*| Input file path, `-` for stdin, or an `http(s)://` URL   |
| `--output`          | `-o`  | `html`      | Output format: `html` or `json`                          |
| `--out`             | `-O`  | stdout      | Output file path                                         |
| `--format`          | `-f`  | auto-detect | Input format: `turtle`, `rdfxml`, `jsonld`               |
| `--title`           |       | ontology IRI| Title shown in HTML output                               |
| `--link-distance`   |       | `80`        | D3 force link distance                                   |
| `--charge-strength` |       | `-300`      | D3 many-body charge strength                             |
| `--collide-radius`  |       | `20`        | D3 collision-detection radius                            |
| `--verbose`         | `-v`  | false       | Enable debug logging                                     |
| `--help`            | `-h`  | —           | Show help                                                |

### Examples

```bash
# Generate a self-contained HTML diagram
./ttl2d3 convert --input my-ontology.ttl --out diagram.html

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
