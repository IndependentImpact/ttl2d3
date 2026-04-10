# ttl2d3 – Repo-Wide Agent Instructions

This file provides authoritative, repo-wide guidance for every AI coding agent
(and human contributor) working in this repository.  Follow these instructions
for **every** development action, large or small.

---

## 1. Project Overview

`ttl2d3` is a Go CLI tool that converts semantic-web ontologies and concept
schemes (`.ttl`, `.owl`, `.json-ld`, `.rdf`) into interactive D3.js
force-directed graph visualisations.  The tool outputs either:

* **Standalone D3 JSON** – a graph object ready to embed in any existing page.
* **Self-contained HTML** – a complete webpage (similar to WebVOWL) with the
  D3 visualisation bundled inside.

---

## 2. Git Practices

### 2.1 Branching
* Use `main` as the stable branch; never commit directly to `main`.
* Branch names: `feature/<short-description>`, `fix/<issue-id>-<description>`,
  `docs/<what-changed>`, `chore/<what-changed>`.
* Keep branches short-lived; open a PR as soon as meaningful work exists.

### 2.2 Commits
* **Commit often** – every logical unit of change gets its own commit. Make sure lint, vet, build and test pass before committing.
* Write commit messages in the imperative mood using the Conventional Commits
  format:
  ```
  <type>(<scope>): <short summary>

  [optional body – why, not what]

  [optional footer: Closes #<issue>]
  ```
  Types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `perf`, `ci`.
* Keep the summary line ≤ 72 characters.
* Reference the relevant issue number in the footer when applicable.

### 2.3 Pull Requests
* Every PR must pass all CI checks (lint, vet, build, test) before merge.
* Include a short description of *what* changed and *why*.
* Keep PRs focused; one concern per PR.

### 2.4 .gitignore
Ensure the following are always ignored:
```
# Build artefacts
/ttl2d3
/ttl2d3.exe
dist/

# Go toolchain
vendor/
*.test
*.out
coverage.html

# OS / editor
.DS_Store
*.swp
.idea/
.vscode/
```

---

## 3. Go Development Practices

### 3.1 Code Style
* Target **Go 1.22+** (use the version specified in `go.mod`).
* Follow `gofmt` / `goimports` formatting – no exceptions.
* Apply `golangci-lint` with the project's `.golangci.yml` on every change.
* Exported identifiers must have Go doc comments.
* Avoid `init()` functions; prefer explicit initialisation.
* Use structured errors (`fmt.Errorf("...: %w", err)`) and avoid `panic`
  outside of `main` startup.

### 3.2 Package Layout
```
ttl2d3/
├── cmd/ttl2d3/        # CLI entry point (cobra)
├── internal/
│   ├── parser/        # RDF/OWL/JSON-LD/Turtle parsing
│   ├── graph/         # Graph data model (nodes, edges, metadata)
│   ├── transform/     # Ontology → graph model transformation
│   ├── render/        # Graph model → D3 JSON / HTML output
│   └── config/        # CLI flags and configuration
├── testdata/          # Sample ontology files for tests
├── agents.md
├── spec.md
├── plan.md
└── README.md
```

### 3.3 Testing
* **Write a test for every exported function.**
* Use `_test.go` files co-located with the package under test.
* Use table-driven tests (`[]struct{ name, input, want }`) wherever practical.
* Aim for ≥ 80 % statement coverage; enforce with `go test -cover`.
* Integration tests that need real ontology files live in `testdata/` and are
  gated with `//go:build integration`.
* Run `go test ./...` before every commit.

### 3.4 Pre-commit Checklist
Run **all** of the following before every commit:
```bash
goimports -w .
golangci-lint run ./...
go vet ./...
go build ./...
go test ./...
```

### 3.5 Dependencies
* Prefer the standard library; add external dependencies only when they provide
  substantial, well-maintained value.
* Pin dependencies in `go.sum`; never use `replace` directives in production
  code.
* Check for vulnerabilities with `govulncheck ./...` before adding or upgrading
  any dependency.

### 3.6 CLI (cobra)
* All flags and sub-commands are defined in `cmd/ttl2d3/`.
* Provide `--help` text for every flag.
* Exit codes: `0` = success, `1` = user error (bad input), `2` = internal
  error.

---

## 4. D3.js Practices

### 4.1 D3 Version
Use **D3 v7** (latest stable).  Import via CDN in the generated HTML:
```html
<script src="https://cdn.jsdelivr.net/npm/d3@7"></script>
```
For the embeddable JSON mode, target the D3 v7 API.

### 4.2 Output Formats

#### Standalone JSON (`--output json`)
Produce a JSON object with this shape:
```json
{
  "nodes": [
    { "id": "string", "label": "string", "type": "class|property|instance|literal", "group": "string" }
  ],
  "links": [
    { "source": "string", "target": "string", "label": "string" }
  ],
  "metadata": {
    "title": "string",
    "description": "string",
    "version": "string",
    "baseIRI": "string"
  }
}
```

#### Self-contained HTML (`--output html`, default)
* Embed the D3 JSON directly in the `<script>` block; no external data file.
* Include zoom + pan (`d3.zoom`).
* Include drag-and-drop node repositioning.
* Tooltips on hover (label, type, IRI).
* Node colour and shape encode the entity type
  (class = circle, property = diamond, instance = square).
* Provide a simple legend.
* Responsive SVG that adapts to viewport width.

### 4.3 Force Simulation Parameters
Default parameter starting points (tunable via CLI flags):
| Parameter        | Default |
|------------------|---------|
| `linkDistance`   | 80      |
| `chargeStrength` | -300    |
| `collideRadius`  | 20      |
| `gravityStrength`| 0.1     |

### 4.4 Accessibility
* Use `aria-label` on the SVG element.
* Maintain WCAG AA colour contrast for node labels.

---

## 5. AI-Assisted Development Best Practices

### 5.1 Scope of Changes
* Make **surgical, minimal changes** – touch only what is necessary.
* Never refactor unrelated code in the same commit.
* When unsure, ask the human for clarification before proceeding.

### 5.2 Test-Driven Flow
1. Write or update tests *first* to describe the desired behaviour.
2. Implement the code to make the tests pass.
3. Refactor while keeping tests green.

### 5.3 Documentation
* **Always** update `README.md`, `spec.md`, `plan.md`, and relevant package
  doc comments after each development action that changes public behaviour.
* Keep `plan.md` up to date – mark completed items, add newly discovered tasks.

### 5.4 Security
* Never commit credentials, tokens, or any secret.
* Validate all user-supplied input (file paths, IRIs, format strings).
* Run `govulncheck` before every dependency change.
* Run `gosec` as part of lint.

### 5.5 Performance
* Benchmark any function in a hot path with `go test -bench`.
* Profile before optimising; optimise only what the profiler proves is slow.

### 5.6 Logging and Observability
* Use structured logging (`log/slog`) – never `fmt.Println` in library code.
* Emit progress logs at `INFO` level; verbose/debug details at `DEBUG`.
* Errors must always propagate up with context (`%w`).

---

## 6. CI/CD

All PRs run the following GitHub Actions pipeline (`.github/workflows/ci.yml`):

| Step              | Command                                   |
|-------------------|-------------------------------------------|
| Format check      | `goimports -l . \| grep . && exit 1`      |
| Lint              | `golangci-lint run ./...`                 |
| Vet               | `go vet ./...`                            |
| Build             | `go build ./...`                          |
| Test (unit)       | `go test -cover ./...`                    |
| Vulnerability scan| `govulncheck ./...`                       |

A PR cannot be merged unless all steps are green.

---

## 7. Quick-Reference Cheat Sheet

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

# Test with coverage
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out

# Run integration tests
go test -tags integration ./...

# Vulnerability check
govulncheck ./...

# Build CLI binary
go build -o ttl2d3 ./cmd/ttl2d3

# Run CLI
./ttl2d3 --help
./ttl2d3 convert --input example.ttl --output html --out diagram.html
./ttl2d3 convert --input example.ttl --output json --out graph.json
```
