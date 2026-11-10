# Vendored third-party code

Patched copies of upstream Go packages, kept as internal packages so that
`go.mod` needs no `replace` directives and `go install` works.

| Package   | Upstream | Upstream commit / date | Licence |
|-----------|----------|------------------------|---------|
| `gon3`    | [deiu/gon3](https://github.com/deiu/gon3) (fork of rychipman/gon3) | 93153c0, 2024-12-12 | Apache 2.0 (added upstream 2026-11-10), see [gon3/LICENSE](gon3/LICENSE) |
| `easylex` | [rychipman/easylex](https://github.com/rychipman/easylex) | 49ee776, 2016-01-29 | None published; requested in [rychipman/easylex#2](https://github.com/rychipman/easylex/issues/2) |

Both packages are modified from upstream (import paths, `go vet` fixes,
and in `gon3` several lexer and IRI-resolution changes). Each modified `gon3`
file carries a header saying so. The full history is in this repository's git log
(`git log --follow`, including the earlier `third_party/` path).

The spec-compliant `gon3` fixes have been offered upstream in
[deiu/gon3#6](https://github.com/deiu/gon3/pull/6).
