# ttl2d3 – Specification

**Version:** 0.1.0-draft  
**Status:** Draft  
**Last updated:** 2026-04-08

---

## 1. Purpose and Goals

`ttl2d3` is a command-line tool written in Go that converts semantic-web
ontologies and concept schemes into interactive D3.js visualisations, including
force-directed and workflow-oriented layouts.

### Primary Goals
1. Accept ontology / concept-scheme input in common semantic-web formats.
2. Produce either an embeddable D3 graph JSON object or a self-contained HTML
   page.
3. Be fast, dependency-light, and easy to integrate into build pipelines.

### Non-Goals
* Full SPARQL query engine (read-only parsing only in v1).
* OWL reasoning / inference.
* Real-time collaborative editing.
* Mobile app.

---

## 2. Background and Research

### 2.1 Problem Space
Ontologies and concept schemes encoded in RDF-family formats (Turtle, OWL/XML,
JSON-LD, RDF/XML) are inherently graph-structured.  Existing visualisation
tools (WebVOWL, Protégé) require a browser plugin or a running server.
`ttl2d3` provides a zero-dependency static-file approach: run the CLI and
share a single HTML file.

### 2.2 Comparable Tools
| Tool        | Language | Output          | Notes                          |
|-------------|----------|-----------------|--------------------------------|
| WebVOWL     | Java/JS  | Browser SPA     | Requires server or app bundle  |
| OWLGrEd     | Java     | Desktop app     | Not web-friendly               |
| Protégé     | Java     | Desktop app     | Feature-rich but heavy         |
| rdflib+pyvis| Python   | HTML            | Python-only pipeline           |
| **ttl2d3**  | **Go**   | **JSON / HTML** | **Single static binary + file**|

### 2.3 Input Format Research
| Format     | MIME type                  | Parser approach                         |
|------------|----------------------------|-----------------------------------------|
| Turtle     | `text/turtle`              | `github.com/deiu/rdf2go` or hand-rolled |
| OWL/XML    | `application/rdf+xml`      | XML → RDF triples                       |
| JSON-LD    | `application/ld+json`      | `github.com/piprate/json-gold`          |
| RDF/XML    | `application/rdf+xml`      | Same as OWL/XML                         |

### 2.4 D3.js Research
D3 v7 provides the `d3-force` module for physics-based graph layouts.  Key
primitives:
* `d3.forceSimulation` – runs the physics tick loop.
* `d3.forceLink` – edge spring forces.
* `d3.forceManyBody` – node charge (repulsion).
* `d3.forceCollide` – prevents node overlap.
* `d3.zoom` – pan and zoom the SVG viewport.

The output graph JSON follows the D3 force-directed graph convention:
`{ nodes: [...], links: [...] }`.

The `layered` layout uses deterministic longest-path ranking and fixed SVG
coordinates instead of physics simulation.  The `swimlane` layout extends
the layered approach with lane bands (SVG `rect` + text labels) grouped by
node `group` (namespace prefix).  Both non-force layouts use D3 utilities
(zoom, symbol, select) but not `d3-force`.

---

## 3. Functional Requirements

### 3.1 Input
| ID    | Requirement |
|-------|-------------|
| IN-01 | Accept a single input file path via `--input` / `-i` flag. |
| IN-02 | Auto-detect format from file extension (`.ttl`, `.owl`, `.jsonld`, `.json`, `.rdf`). |
| IN-03 | Allow explicit format override via `--format` flag (`turtle`, `rdfxml`, `jsonld`). |
| IN-04 | Accept input from stdin when `--input -` is specified. |
| IN-05 | Validate that the input is well-formed; emit a clear error message on parse failure. |

### 3.2 Graph Model Extraction
| ID    | Requirement |
|-------|-------------|
| GM-01 | Extract all OWL classes and SKOS concepts as **nodes** of type `class`. |
| GM-02 | Extract all object properties / SKOS semantic relations as **edges**. |
| GM-03 | Extract datatype properties as **nodes** of type `property` with edges to their domain. |
| GM-04 | Preserve `rdfs:label` / `skos:prefLabel` as the human-readable node label. |
| GM-05 | Use the IRI fragment (or local name) as fallback when no label exists. |
| GM-06 | Record ontology-level metadata (title, description, version, base IRI). |
| GM-07 | Support `owl:subClassOf`, `rdfs:subClassOf`, and `skos:broader` as hierarchy edges. |
| GM-08 | Support `owl:equivalentClass` and `owl:disjointWith` relationship edges. |
| GM-09 | Preserve multiple object properties with identical domain + range as distinct edges, even when labels match. |
| GM-10 | Treat IRIs referenced in property `rdfs:domain` / `rdfs:range` as class nodes when not explicitly declared. |
| GM-11 | Resolve `owl:unionOf` lists in domain/range definitions to their member IRIs. |
| GM-12 | Represent `owl:unionOf` class expressions as explicit union nodes linked to each member via `unionOf` edges. |

### 3.3 Output – JSON mode (`--output json`)
| ID    | Requirement |
|-------|-------------|
| OJ-01 | Write a UTF-8 JSON file (or stdout) containing `nodes`, `links`, and `metadata` keys. |
| OJ-02 | Every node has: `id`, `label`, `type` (`class`\|`property`\|`union`\|`instance`\|`literal`), `group`. |
| OJ-03 | Every link has: `source`, `target`, `label`. |
| OJ-04 | Output is valid against the schema defined in Appendix A. |

### 3.4 Output – HTML mode (`--output html`, default)
| ID    | Requirement |
|-------|-------------|
| OH-01 | Write a single self-contained HTML file with all CSS and JS inlined. |
| OH-02 | Load D3 v7 from `https://cdn.jsdelivr.net/npm/d3@7` (CDN). |
| OH-03 | Embed the graph JSON inline in a `<script>` block. |
| OH-04 | Render an interactive D3 visualisation in the selected layout mode. |
| OH-04a| Support `force` layout mode with zoom, pan, and drag (default). |
| OH-04b| Support `layered` layout mode for workflow/state-transition visualisation. |
| OH-04c| Support `swimlane` layout mode for process-style visualisation. |
| OH-05 | Node colour and shape encode entity type (class=circle, property=diamond, instance=square). |
| OH-06 | Hovering a node shows a tooltip with its IRI, label, and type. |
| OH-07 | Include a visible legend. |
| OH-08 | SVG is responsive (percentage-based width, viewport-relative height). |
| OH-09 | Provide a search/filter input box to highlight nodes by label substring. |
| OH-10 | Show origin (local vs imported) and namespace legend derived from node IRIs. |
| OH-11 | Non-force layout output must be deterministic for identical input. |
| OH-12 | Back-edges and loop edges must be visually distinct in non-force layouts. |

### 3.5 CLI Interface
```
ttl2d3 [flags]
ttl2d3 convert [flags]   (default sub-command)
ttl2d3 version
```

| Flag                    | Short | Default      | Description                                          |
|-------------------------|-------|--------------|------------------------------------------------------|
| `--input`               | `-i`  | *(required)* | Input file path or `-` for stdin                     |
| `--output`              | `-o`  | `html`       | Output format: `html` or `json`                      |
| `--out`                 | `-O`  | stdout       | Output file path                                     |
| `--format`              | `-f`  | auto-detect  | Input format override                                |
| `--title`               |       | ontology IRI | Title shown in HTML output                           |
| `--layout`              |       | `force`      | HTML layout mode: `force`, `layered`, `swimlane`     |
| `--layout-direction`    |       | `lr`         | Flow direction: `lr` (left-to-right) or `tb` (top-to-bottom) |
| `--rank-separation`     |       | `180`        | Pixel gap between ranks (layered/swimlane)           |
| `--node-separation`     |       | `80`         | Pixel gap between nodes within a rank (layered/swimlane) |
| `--link-distance`       |       | `80`         | D3 link distance (force only)                        |
| `--charge-strength`     |       | `-300`       | D3 charge strength (force only)                      |
| `--collide-radius`      |       | `20`         | D3 collide radius (force only)                       |
| `--gravity-strength`    |       | `0.1`        | D3 gravity toward centre (0–1, force only)           |
| `--verbose`             | `-v`  | false        | Enable debug logging                                 |
| `--help`                | `-h`  | —            | Show help                                            |

Validation:
* `--layout` applies only to HTML output; combining it with `--output json` is an error.
* Invalid `--layout`, `--layout-direction` values are rejected with a clear error message.

### 3.6 Versioning
* `ttl2d3 version` prints the version string, Git commit SHA, and build date.

---

## 4. Non-Functional Requirements

| ID    | Category        | Requirement |
|-------|-----------------|-------------|
| NF-01 | Performance     | Process a 10 000-triple ontology in under 5 seconds on commodity hardware. |
| NF-02 | Memory          | Peak heap usage under 256 MB for files up to 50 MB. |
| NF-03 | Portability     | Produce a single static binary for Linux, macOS, Windows (amd64 + arm64). |
| NF-04 | Correctness     | Parse results must be identical to those of a reference parser for the same input. |
| NF-05 | Testability     | Unit-test coverage ≥ 80 % of statements. |
| NF-06 | Usability       | `--help` output is self-explanatory; no manual required for basic usage. |
| NF-07 | Security        | No shell injection; validate and sanitise all IRI strings before HTML output. |
| NF-08 | Accessibility   | Generated HTML meets WCAG 2.1 AA colour-contrast requirements. |
| NF-09 | Determinism     | Non-force layouts must produce stable output coordinates for identical input. |
| NF-10 | Readability     | Workflow-oriented layouts should materially reduce overlap and visual ambiguity for sequential process graphs. |

---

## 5. Architecture Overview

```
┌────────────────────────────────────────────────────────────────┐
│                         CLI (cobra)                            │
│                    cmd/ttl2d3/main.go                          │
└────────────────┬───────────────────────────────────────────────┘
                 │
         ┌───────▼────────┐
         │   config/      │  flags, defaults, validation
         └───────┬────────┘
                 │
         ┌───────▼────────┐
         │   parser/      │  format detection + parsing
         │  ┌────────────┐│  → internal triple store
         │  │  turtle    ││
         │  │  rdfxml    ││
         │  │  jsonld    ││
         │  └────────────┘│
         └───────┬────────┘
                 │
         ┌───────▼────────┐
         │  transform/    │  triples → GraphModel
         └───────┬────────┘
                 │
         ┌───────▼────────┐
         │   render/      │  GraphModel → JSON / HTML
         │  ┌────────────┐│
         │  │  json.go   ││
         │  │  html.go   ││
         │  │  templates/││  graph.html (force)
         │  │            ││  graph_layered.html
         │  │            ││  graph_swimlane.html
         │  └────────────┘│
         └────────────────┘
```

### 5.1 Internal Graph Model
```go
// GraphModel is the central data structure passed between transform and render.
type GraphModel struct {
    Nodes    []Node
    Links    []Link
    Metadata Metadata
}

type Node struct {
    ID    string // IRI
    Label string // rdfs:label or local name
    Type  NodeType
    Group string // namespace prefix or domain group
}

type Link struct {
    Source string // node IRI
    Target string // node IRI
    Label  string // property local name
}

type Metadata struct {
    Title       string
    Description string
    Version     string
    BaseIRI     string
}
```

---

## 6. Error Handling

* All errors are returned (never panicked) from library code.
* The CLI prints human-readable messages to `stderr` and exits with the
  appropriate exit code (see §3.5).
* Parse errors include the line/column number where available.

---

## 7. Testing Strategy

| Layer       | Type             | Location                    | Tags        |
|-------------|------------------|-----------------------------|-------------|
| parser      | unit             | `internal/parser/*_test.go` | (none)      |
| transform   | unit             | `internal/transform/*_test.go` | (none)  |
| render      | unit + golden    | `internal/render/*_test.go` | (none)      |
| CLI         | integration      | `cmd/ttl2d3/*_test.go`      | `integration`|
| End-to-end  | integration      | `e2e/*_test.go`             | `integration`|

Golden-file tests compare render output against checked-in expected files in
`testdata/golden/`.

---

## 8. Supported Input Formats – Detail

### 8.1 Turtle (`.ttl`)
* W3C Turtle 1.1
* Supports prefix declarations, blank nodes, collections, and multi-valued
  properties.
* Parser: `github.com/deiu/rdf2go` (or a hand-rolled parser for full control).

### 8.2 OWL/RDF XML (`.owl`, `.rdf`)
* RDF/XML syntax as defined by W3C.
* Parsed via Go `encoding/xml` into an RDF triple stream.

### 8.3 JSON-LD (`.jsonld`, `.json`)
* JSON-LD 1.1.
* Parser: `github.com/piprate/json-gold`.

---

## 9. Security Considerations

* All IRI strings written into HTML output are HTML-escaped to prevent XSS.
* Input file paths are validated; directory traversal is rejected.
* No network requests are made during parsing; CDN link is the only external
  reference in HTML output.

---

## 10. Future Work (Post-v1)

* SPARQL endpoint output (write a SPARQL query to extract subgraphs).
* Plugin / template system for custom HTML themes.
* Watch mode (`--watch`) to regenerate on file change.
* `--lane-by` strategy for swimlane (currently uses `group`/namespace prefix).
* WASM build for in-browser usage.

---

## Appendix A – JSON Output Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "ttl2d3 Graph JSON",
  "type": "object",
  "required": ["nodes", "links", "metadata"],
  "properties": {
    "nodes": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["id", "label", "type"],
        "properties": {
          "id":    { "type": "string" },
          "label": { "type": "string" },
          "type":  { "type": "string", "enum": ["class", "property", "union", "instance", "literal"] },
          "group": { "type": "string" }
        }
      }
    },
    "links": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["source", "target"],
        "properties": {
          "source": { "type": "string" },
          "target": { "type": "string" },
          "label":  { "type": "string" }
        }
      }
    },
    "metadata": {
      "type": "object",
      "properties": {
        "title":       { "type": "string" },
        "description": { "type": "string" },
        "version":     { "type": "string" },
        "baseIRI":     { "type": "string" }
      }
    }
  }
}
```
