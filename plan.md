# ttl2d3 – Implementation Plan

**Version:** 0.1.0-draft  
**Status:** Active  
**Last updated:** 2026-03-26 (Phase 15 complete)

Progress legend: ✅ Done · 🔄 In progress · ⬜ Not started

---

## Phase 0 – Repository Bootstrap ✅

- [x] Create `agents.md` (repo-wide AI + dev instructions)
- [x] Create `spec.md` (requirements and architecture)
- [x] Create `plan.md` (this file)
- [x] Update `README.md` with project description, install, and usage section

---

## Phase 1 – Go Module + CLI Skeleton ✅

**Goal:** `go build ./...` succeeds and `ttl2d3 --help` works.

- [x] 1.1 `go mod init github.com/IndependentImpact/ttl2d3`
- [x] 1.2 Add `cobra` dependency (`github.com/spf13/cobra`)
- [x] 1.3 Create `cmd/ttl2d3/main.go` with root command and `version` sub-command
- [x] 1.4 Create `cmd/ttl2d3/convert.go` with `convert` sub-command and all flags
      (see spec §3.5)
- [x] 1.5 Create `internal/config/config.go` – flag values struct + validation
- [x] 1.6 Wire up structured logging (`log/slog`) with `--verbose` flag
- [x] 1.7 Add `.golangci.yml` linter configuration
- [x] 1.8 Add `.github/workflows/ci.yml` GitHub Actions pipeline
- [x] 1.9 Add `.gitignore`
- [x] 1.10 Run: lint + vet + build + test; commit

---

## Phase 2 – Internal Graph Model ✅

**Goal:** Core data structures with unit tests.

- [x] 2.1 Create `internal/graph/model.go` – `GraphModel`, `Node`, `Link`, `Metadata`, `NodeType`
- [x] 2.2 Write `internal/graph/model_test.go` – constructor and validation tests
- [x] 2.3 Run: lint + vet + build + test; commit

---

## Phase 3 – Turtle Parser ✅

**Goal:** Parse `.ttl` files into a triple store.

- [x] 3.1 Evaluate `github.com/deiu/rdf2go` vs writing a minimal Turtle 1.1 parser
      – Decision: use `github.com/deiu/rdf2go` (pure Go, MIT, Turtle 1.1 support)
- [x] 3.2 Add chosen dependency / implement parser in `internal/parser/turtle.go`
      – Also added `internal/parser/triple.go` with internal `Term`, `Triple`, `Graph` types
- [x] 3.3 Add `testdata/` directory with sample Turtle files:
      - `testdata/simple.ttl` – 5-class OWL ontology (21 triples)
      - `testdata/skos.ttl` – small SKOS concept scheme (23 triples)
- [x] 3.4 Write `internal/parser/turtle_test.go` (table-driven, specific triple assertions)
- [x] 3.5 Run: lint + vet + build + test; commit

---

## Phase 4 – RDF/XML Parser ✅

**Goal:** Parse `.owl` / `.rdf` files into the triple store.

- [x] 4.1 Implement `internal/parser/rdfxml.go` using `encoding/xml`
- [x] 4.2 Add `testdata/pizza.owl` (a canonical test ontology)
- [x] 4.3 Write `internal/parser/rdfxml_test.go`
- [x] 4.4 Run: lint + vet + build + test; commit

---

## Phase 5 – JSON-LD Parser ✅

**Goal:** Parse `.jsonld` / `.json` files into the triple store.

- [x] 5.1 Add `github.com/piprate/json-gold` dependency
- [x] 5.2 Implement `internal/parser/jsonld.go`
- [x] 5.3 Add `testdata/example.jsonld`
- [x] 5.4 Write `internal/parser/jsonld_test.go`
- [x] 5.5 Run: lint + vet + build + test; commit

---

## Phase 6 – Format Detection ✅

**Goal:** Auto-detect input format; support `--format` override.

- [x] 6.1 Implement `internal/parser/detect.go` – extension + MIME sniffing
- [x] 6.2 Implement `internal/parser/parse.go` – dispatcher that calls the correct parser
- [x] 6.3 Write `internal/parser/detect_test.go`
- [x] 6.4 Run: lint + vet + build + test; commit

---

## Phase 7 – Ontology → GraphModel Transform ✅

**Goal:** Convert a triple store into a `GraphModel`.

- [x] 7.1 Implement `internal/transform/ontology.go`
      - Extract OWL classes, object properties, datatype properties
      - Extract SKOS concepts and semantic relations
      - Populate `Node.Group` from namespace prefix
- [x] 7.2 Implement `internal/transform/label.go` – label resolution strategy
      (`rdfs:label` → `skos:prefLabel` → IRI fragment → full IRI)
- [x] 7.3 Write `internal/transform/ontology_test.go` (table-driven with testdata)
- [x] 7.4 Run: lint + vet + build + test; commit

---

## Phase 8 – JSON Renderer ✅

**Goal:** Emit the D3-compatible JSON output.

- [x] 8.1 Implement `internal/render/json.go` – serialize `GraphModel` to JSON
      conforming to Appendix A of spec
- [x] 8.2 Write `internal/render/json_test.go` with golden-file comparison
      (`testdata/golden/*.json`)
- [x] 8.3 Run: lint + vet + build + test; commit

---

## Phase 9 – HTML Renderer ✅

**Goal:** Emit a self-contained interactive HTML page.

- [x] 9.1 Create `internal/render/templates/graph.html` – Go `html/template` file
      with embedded D3 v7 script
- [x] 9.2 Implement force simulation in the template:
      - `d3.forceSimulation` with configurable parameters
      - `d3.zoom` for pan/zoom
      - Drag-and-drop on nodes
      - Tooltip on hover
      - Node shape/colour by type
      - Legend
      - Search/filter input
- [x] 9.3 Implement `internal/render/html.go` – execute template with `GraphModel`;
      add `HTMLOptions` and `DefaultHTMLOptions`
- [x] 9.4 Write `internal/render/html_test.go` – check rendered output contains
      expected HTML fragments; golden-file diff for full output
- [x] 9.5 Manually test with sample ontologies; take screenshots for PR review
- [x] 9.6 Run: lint + vet + build + test; commit

---

## Phase 10 – CLI Wiring ✅

**Goal:** End-to-end pipeline works from the CLI.

- [x] 10.1 Connect `convert` command to parser → transform → render pipeline
- [x] 10.2 Handle stdin input (`--input -`)
- [x] 10.3 Handle stdout output (default when `--out` not specified)
- [x] 10.4 Write `cmd/ttl2d3/convert_test.go` (integration, `//go:build integration`)
- [x] 10.5 Write end-to-end test in `e2e/e2e_test.go` (integration)
- [x] 10.6 Run: lint + vet + build + test (unit + integration); commit

---

## Phase 11 – Polish and Documentation ✅

**Goal:** Production-quality release candidate.

- [x] 11.1 Update `README.md` with:
      - Badges (CI, Go Report Card, Go version, License)
      - Installation instructions (go install, build from source)
      - Usage examples with sample commands
      - Contributing guide pointer to `CONTRIBUTING.md`
      - Updated status (v0.1.0 release candidate)
- [x] 11.2 Add `CONTRIBUTING.md` (references `agents.md`)
- [x] 11.3 Ensure all exported symbols have doc comments (verified via `revive`)
- [x] 11.4 Run `govulncheck ./...` – no findings
- [x] 11.5 Verify and fix WCAG AA contrast in generated HTML:
      - Link labels: `#6c757d` → `#636c72` (4.45 → 5.08 : 1)
      - Property node fill: `#ff7f0e` → `#9e5200` (2.40 → 5.44 : 1)
- [ ] 11.6 Tag `v0.1.0` release
- [x] 11.7 Run: lint + vet + build + test; commit

---

## Phase 12 – CI/CD Hardening ✅

- [x] 12.1 Add release workflow (`.github/workflows/release.yml`) using `goreleaser`
      to cross-compile for Linux/macOS/Windows × amd64/arm64
- [x] 12.2 Add dependabot config (`.github/dependabot.yml`) for Go modules and
      GitHub Actions
- [x] 12.3 Add code coverage reporting (Codecov) – `codecov/codecov-action@v5`
      wired into CI; set `CODECOV_TOKEN` repository secret to activate
- [x] 12.4 Run: lint + vet + build + test; commit

---

## Phase 13 – Bug Fixes ✅

- [x] 13.1 Preserve distinct object properties that share domain + range even when labels match

---

## Phase 14 – Inferred Classes ✅

- [x] 14.1 Promote domain/range IRIs to class nodes when not explicitly typed
- [x] 14.2 Resolve owl:unionOf members for implied classes

---

## Phase 15 – Union Nodes ✅

- [x] 15.1 Represent owl:unionOf class expressions as explicit union nodes

---

## Decisions Log

| Date       | Decision | Rationale |
|------------|----------|-----------|
| 2026-03-25 | Use `cobra` for CLI | Industry standard for Go CLIs; excellent flag + help handling |
| 2026-03-25 | Use `github.com/deiu/rdf2go` for Turtle | Pure Go, MIT license, covers Turtle + partial JSON-LD |
| 2026-03-25 | Use `encoding/xml` (stdlib) for RDF/XML | Zero additional dependencies; Go stdlib XML decoder handles namespaces and encoding |
| 2026-03-25 | Use `github.com/piprate/json-gold` for JSON-LD | Most complete W3C JSON-LD 1.1 implementation in Go |
| 2026-03-25 | Use `html/template` (stdlib) for HTML generation | Automatic HTML escaping prevents XSS; zero extra dependency |
| 2026-03-25 | D3 v7 via CDN | Keeps generated HTML small; v7 is current stable |
| 2026-03-25 | `log/slog` for logging | Go stdlib since 1.21; structured logging without extra deps |

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| RDF/XML parsing edge cases | Medium | High | Use W3C test suite cases in `testdata/` |
| D3 force layout instability for large graphs | Medium | Medium | Expose tunable parameters; document recommended settings |
| CDN unavailability in offline environments | Low | Medium | Document `--cdn-url` override flag (Phase 11+) |
| Breaking changes in dependencies | Low | Low | Pin exact versions in `go.sum`; dependabot alerts |
