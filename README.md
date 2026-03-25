# ttl2d3

> Convert semantic-web ontologies and concept schemes to interactive D3.js
> force-directed graph visualisations.

`ttl2d3` is a Go CLI tool that reads ontologies or concept schemes in common
RDF formats (`.ttl`, `.owl`, `.json-ld`, `.rdf`) and produces either:

* a **standalone D3 JSON object** ready to embed in any webpage, or
* a **self-contained HTML page** with an interactive D3 force-directed graph
  (zoom, pan, drag, tooltips, search) – similar to WebVOWL but output as a
  single static file.

---

## Status

🚧 **Pre-release – implementation in progress.**  
See [`plan.md`](plan.md) for the full implementation roadmap and
[`spec.md`](spec.md) for the detailed specification.

---

## Quick Start (once implemented)

```bash
# Install
go install github.com/IndependentImpact/ttl2d3/cmd/ttl2d3@latest

# Generate a self-contained HTML diagram
ttl2d3 convert --input my-ontology.ttl --out diagram.html

# Generate D3 graph JSON only
ttl2d3 convert --input my-ontology.ttl --output json --out graph.json

# Read from stdin
cat my-ontology.ttl | ttl2d3 convert --input - --out diagram.html
```

---

## Supported Input Formats

| Extension          | Format  |
|--------------------|---------|
| `.ttl`             | Turtle  |
| `.owl` / `.rdf`    | RDF/XML |
| `.jsonld` / `.json`| JSON-LD |

---

## Documentation

* [`agents.md`](agents.md) – Repo-wide instructions for contributors and AI agents
* [`spec.md`](spec.md) – Full requirements and architecture specification
* [`plan.md`](plan.md) – Phased implementation plan with progress tracking

---

## Contributing

Read [`agents.md`](agents.md) before making any changes.  Every PR must pass
lint, vet, build, and tests.

## License

TBD
