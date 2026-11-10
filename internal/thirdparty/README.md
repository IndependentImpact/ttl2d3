# Vendored third-party code

Patched copies of upstream Go packages, kept as internal packages so that
`go.mod` needs no `replace` directives and `go install` works.

| Package   | Upstream                                                  | Upstream commit / date |
|-----------|-----------------------------------------------------------|------------------------|
| `gon3`    | [deiu/gon3](https://github.com/deiu/gon3) (fork of rychipman/gon3) | 93153c0, 2024-12-12 |
| `easylex` | [rychipman/easylex](https://github.com/rychipman/easylex) | 49ee776, 2016-01-29    |

Local changes are recorded in this repository's git history (see
`git log --follow` on the files, including their earlier `third_party/` path).

**Licence status:** neither upstream repository publishes a licence. Permission
to use and redistribute this code is being sought from the authors.
