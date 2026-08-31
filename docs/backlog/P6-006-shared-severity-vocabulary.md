# P6-006 One Severity Vocabulary

## Story

As a maintainer, I want severity ordering and finding-source formatting defined once so that the
console, the report, and the exit code cannot drift apart.

## Scope

- move severity ordering into `result` as the single source of truth
- deduplicate `formatSources` and the pluralisation helpers
- keep the two distinct ranking *behaviours* intact

## The Duplication

Severity ordering exists three times:

| Location | Purpose | Unknown severity |
| --- | --- | --- |
| `main.severityRank` | gates the exit code | ranks lowest, so it fails nothing on its own |
| `main.renderRank` | orders console findings | ranks `-1`, so it prints last |
| `report.severityOrder` | orders report findings | ranks last |

`formatSources` is duplicated between `main.go:598` and `report/report.go` — byte-identical except
for an empty-source guard the report copy has and `main` does not. `main.pluralize` and
`report.pluralise` are the same idea under two spellings.

Three tables that must agree, in three packages, with nothing enforcing agreement.

## Decisions To Implement

- **`result` owns the ordering.** It is already the leaf package and the common currency, and it
  owns the `Severity` type. Put one ordered severity list there and derive everything from it.
- **Keep both ranking behaviours — they are a real distinction, not an accident.** Gating must treat
  an unknown severity as lowest so a malformed finding can never fail a build by itself; display
  must sort it last so it never outranks a real finding. Express both as named functions over the
  one table rather than collapsing them into a single rank and losing the difference. `CLAUDE.md`
  documents this distinction deliberately — preserve it and its comments.
- **`formatSources` moves with the ordering.** It is presentation, but it is presentation shared by
  two packages and depends only on `result.Source`. Put it in `result` with the report's
  empty-source guard, which is the safer of the two behaviours and changes nothing for `main`
  because `main` never produces an empty source.
- **Pick one spelling.** The repository is otherwise British-English in prose and American in
  identifiers; choose one for the shared pluralisation helper and delete the other. Note that the
  two helpers differ in signature — `main.pluralize(count, noun)` is general, `report.pluralise`
  is hardcoded to "finding" — so the general form is the one to keep.
- **No behaviour change.** This story is a pure refactor. If any output byte changes, something is
  wrong.

## Proposed Changes

- add the ordered severity list and the two ranking functions to `result`
- move `formatSources` to `result`, keeping the empty-source guard
- move the general pluralisation helper to a shared home and delete the duplicate
- update `main` and `report` to call the shared functions and delete their local copies

## Acceptance Criteria

- exactly one severity ordering table exists in the codebase
- both ranking behaviours are preserved, each with a test asserting the unknown-severity case
- `formatSources` and the pluralisation helper each exist once
- console output, JSON output, and rendered HTML are byte-identical to before the change
- `go test ./...` passes with no assertion changes
- `result` still imports nothing from `skill`, `rules`, or `analyse`

## Must Not Regress

- exit code gating — an unrecognised severity still fails nothing on its own
- console display order — errors first, unknown last, stable within a severity so rule findings stay
  ahead of analyzer findings
- report display order and the severity count chips
- the one-, two-, and three-or-more source phrasings ("a and b", "a, b, and c")

## Documentation

- update the presentation and exit-code invariants in `CLAUDE.md` to point at the shared definitions

## Dependencies

- none
- `P6-007` builds on this; land this first
