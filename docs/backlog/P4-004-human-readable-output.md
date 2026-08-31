# P4-004 Human Readable Output

## Story

As a CLI user, I want readable terminal output so that I can quickly understand scan results without
parsing JSON.

## Already Delivered

`P4-002` and `P4-006` covered part of the original scope. The following already ships and is not
part of this story:

- per-file grouping and `=== <path> ===` headers for multi-file runs (`renderScans`)
- category, severity, source, and evidence on every finding line (`renderResult`)
- a one-line clean verdict, and the HTML report pointer

## Scope

What is actually left:

- order findings within a file by severity, highest first, instead of merge order
- a count line per file and a single run-level tally
- colour when stdout is a terminal, and none when it is not
- evidence that stays readable when the underlying snippet is long

## Decisions To Implement

- **Ordering.** Sort by severity `error` → `warning` → `info`; within a severity, keep the existing
  merge order so rule findings stay ahead of analyzer findings. Sort at render time only —
  `result.Result` keeps its current ordering so the JSON contract is untouched.
- **Run tally.** A multi-file run ends with one line, e.g.
  `3 files scanned · 1 clean · 2 flagged · 6 findings (2 error, 4 warning)`. A single-file run
  does not print it; its existing per-file line already says this.
- **Colour.** Severity labels only. Off unless stdout is a TTY, and always off when `NO_COLOR` is
  set or `--no-color` is passed. `run` takes an `io.Writer`, so detection must happen in `main` and
  reach the renderer as a value — not by sniffing `os.Stdout` inside the render path, which would
  break the existing tests' ability to capture output.
- **Evidence truncation.** Truncate evidence summaries over 200 runes with a trailing `…`. The HTML
  report keeps the full text, so nothing is lost — say so in the README.
- **The clean line stays as it is.** `THIS SKILL APPEARS TO BE CLEAN, PLEASE MANUALLY VERIFY TO BE
  SURE` is deliberate; do not soften it.

## Boundary With `P5-004`

This story is per-run presentation in text only. Aggregate counts by category, machine-readable
summaries, and the JSON summary shape all belong to `P5-004-repository-summary-reporting`.

## Acceptance Criteria

- findings within a file print in severity order, highest first
- a multi-file run prints one closing tally whose counts match the findings above it
- a single-file run's output is byte-identical to today's apart from severity ordering
- colour is absent when the writer is not a terminal, when `NO_COLOR` is set, or with `--no-color`
- an evidence summary longer than the limit is truncated in the console and complete in the HTML report
- table-driven tests cover ordering, the tally, truncation, and both colour states

## Must Not Regress

- single-file output shape, including the absence of a path header
- `--json` prints the JSON and nothing else
- findings are never dropped from the console, only reordered and truncated

## Documentation

- refresh the console examples in `README.md`, including the tally line, and document `--no-color`

## Dependencies

- depends on `P4-002`
- coordinate with `P4-003`: both change `run`'s output path
