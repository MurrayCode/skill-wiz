# P6-001 Drop the Unused ADK Dependency

## Story

As a maintainer, I want the module graph to contain only what the code imports so that the
dependency surface stays small and auditable.

## Scope

- remove `google.golang.org/adk` from `go.mod`
- let the transitive dependencies it alone pulled in fall away with it

## Context

No `.go` file in the repository references `google.golang.org/adk`. It is listed as a *direct*
require, so `go mod tidy` has never dropped it.

Running `go mod tidy` removes it plus eleven transitive dependencies, for a net `-36` lines across
`go.mod` and `go.sum`:

| Dependency | Pulled in by |
| --- | --- |
| `github.com/a2aproject/a2a-go` | adk |
| `github.com/awalterschulze/gographviz` | adk |
| `github.com/google/safehtml` | adk |
| `github.com/google/uuid` | adk |
| `github.com/gorilla/mux` | adk |
| `github.com/mitchellh/mapstructure` | adk |
| `go.opentelemetry.io/otel/sdk` | adk |
| `golang.org/x/sync` | adk |
| `rsc.io/omap` | adk |
| `rsc.io/ordered` | adk |

`google.golang.org/genai` and `gopkg.in/yaml.v3` remain the only direct requires.

## Decisions To Implement

- **Tidy, do not hand-edit.** Run `go mod tidy` and commit the result, so the file stays consistent
  with what the toolchain would produce.
- **No code change.** This story touches `go.mod` and `go.sum` only. If a later story wants ADK, it
  can add the dependency back at that point with an import to justify it.

## Proposed Changes

- run `go mod tidy`
- commit the resulting `go.mod` and `go.sum`

## Acceptance Criteria

- `google.golang.org/adk` no longer appears in `go.mod` or `go.sum`
- `go build ./...` and `go vet ./...` succeed
- `go test ./...` passes with no test changes
- `go mod tidy` is a no-op afterwards — running it again leaves the tree clean

## Must Not Regress

- the Gemini analysis leg still builds and runs against `google.golang.org/genai`
- no import in any package changes

## Documentation

- none required; no user-facing behaviour changes

## Dependencies

- none — this story is independent of every other Phase 6 story and can land first
