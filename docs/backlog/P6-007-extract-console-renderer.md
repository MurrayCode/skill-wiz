# P6-007 Extract The Console Renderer

## Story

As a maintainer, I want console rendering to live in its own package so that `main` is wiring and
the rendering rules can be tested without going through the CLI.

## Scope

- move console rendering out of `main.go` into a `render` package
- split the corresponding tests out of `main_test.go`
- leave behaviour untouched

## The Problem

`main.go` is 647 lines doing three jobs: flag parsing, orchestration, and console rendering. The
rendering half — `renderScans`, `renderTally`, `severityBreakdown`, `renderResult`,
`orderedFindings`, `renderRank`, `severityLabel`, `truncateEvidence`, `formatSources`, and the
colour constants, roughly 170 lines — depends on nothing from `flag`, `os`, or the filesystem. It
takes findings and a style and returns a string.

`main_test.go` is 1,589 lines and splits along the same seam, so most of the rendering assertions
currently reach their subject through `run` when they could call it directly.

## Decisions To Implement

- **`render` depends on `result` only.** Same rule the `report` package already follows. It must
  not import `skill`, `rules`, `scanner`, or `analyse`. If something needs a `*skill.Skill`, it does
  not belong in this package.
- **`renderStyle` moves too, and stays a value.** Colour is decided in `main` by `isTerminal` and
  `colorEnabled` and passed in. The render package must never look at `os.Stdout` — the tests write
  to buffers and colour must stay absent there. Keep `colorEnabled` and `isTerminal` in `main`,
  where the environment lives.
- **Pass what rendering needs, not the scan struct.** `fileScan` carries a `*skill.Skill` the
  renderer never reads. Define a small input type in `render` — path plus result — mirroring how
  `report.Input` already works, so `main` maps `fileScan` onto it. This is what keeps the dependency
  direction clean.
- **JSON stays in `main`.** `renderJSON` and its payload types are an output contract, not console
  presentation, and moving them would put a documented stable shape behind another package boundary
  for no gain. Out of scope.
- **Move tests with their subject.** Rendering assertions currently in `main_test.go` move to
  `render`; keep the end-to-end CLI cases in `main_test.go` so `run`'s behaviour stays covered at
  the boundary.
- **No behaviour change.** Pure refactor. If any output byte changes, something is wrong.

## Proposed Changes

- add a `render` package holding the console rendering functions, the style type, and the colour
  constants
- define a render input type and map `fileScan` onto it in `main`
- move the rendering tests out of `main_test.go` into the new package
- reduce `main.go` to flags, wiring, orchestration, JSON, and the report pointer

## Acceptance Criteria

- console output is byte-identical to before the change, for clean, flagged, single-file,
  multi-file, coloured, and uncoloured runs
- the `render` package imports `result` and the standard library only
- rendering behaviour is tested without invoking `run`
- `main.go` is meaningfully shorter and holds no rendering logic
- `go test ./...` and `go vet ./...` pass

## Must Not Regress

- single-file output — no path header, no tally
- multi-file output — `=== <path> ===` headers, blank line between sections, one closing tally
- colour applied to the severity label only, so the rest of a line stays greppable, and absent
  whenever the writer is not a terminal
- evidence truncation at 200 runes in the console while JSON and HTML keep the full text
- `--json` prints the JSON and nothing else

## Documentation

- update the architecture section and the data-flow line in `CLAUDE.md` to include `render`

## Dependencies

- depends on `P6-006` — the shared severity and source helpers should be in `result` before the
  renderer moves, so the new package does not carry a copy with it
