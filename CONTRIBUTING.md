# Contributing to ttl2d3

Thank you for your interest in contributing!  Please read this guide before
opening a pull request.

> **AI agents and automated contributors:** Read [`agents.md`](agents.md)
> first – it contains the authoritative repo-wide instructions for every
> coding agent working in this repository.

---

## Prerequisites

| Tool | Minimum version | Purpose |
|------|-----------------|---------|
| Go | 1.22 | Build and test |
| `golangci-lint` | 1.57 | Lint checks |
| `goimports` | latest | Import formatting |
| `govulncheck` | latest | Vulnerability scanning |

Install linting tools:

```bash
go install golang.org/x/tools/cmd/goimports@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
# Install golangci-lint: https://golangci-lint.run/usage/install/
```

---

## Development Workflow

1. **Fork and clone** the repository.
2. **Create a branch** following the naming convention in `agents.md`:
   - `feature/<short-description>`
   - `fix/<issue-id>-<description>`
   - `docs/<what-changed>`
   - `chore/<what-changed>`
3. **Make your changes**, committing often with descriptive messages using the
   Conventional Commits format (see `agents.md §2.2`).
4. **Run the full quality gate** before pushing:

   ```bash
   goimports -w .
   go vet ./...
   golangci-lint run ./...
   go test ./...
   go test -tags integration ./...
   govulncheck ./...
   go build ./...
   ```

5. **Open a pull request** against `main`.  The CI pipeline must be green
   before a PR can be merged.

---

## Code Style

* Follow standard Go conventions (`gofmt`, `goimports`).
* All exported identifiers must have a doc comment.
* Keep functions small and focused; prefer table-driven tests.
* Do not introduce new dependencies without discussing them in the PR
  description and updating [`plan.md`](plan.md) Decisions Log.

---

## Testing

| Layer | Location | Build tags |
|-------|----------|------------|
| Unit tests | `internal/**/*_test.go` | *(none)* |
| CLI integration | `cmd/ttl2d3/*_test.go` | `integration` |
| End-to-end | `e2e/*_test.go` | `integration` |

Golden-file tests compare rendered output against files in `testdata/golden/`.
If your change intentionally alters rendered output, regenerate the golden
files and commit them with your PR.

---

## Security

* Never commit secrets, credentials, or sensitive data.
* All IRI strings written to HTML output must be HTML-escaped (the
  `html/template` package handles this automatically).
* Run `govulncheck ./...` and resolve any findings before opening a PR.

---

## Reporting Issues

Please open a GitHub issue and include:
* The command you ran and the input file (or a minimal reproducer).
* The actual output / error message.
* The expected output.
* Go version (`go version`) and OS/architecture.
