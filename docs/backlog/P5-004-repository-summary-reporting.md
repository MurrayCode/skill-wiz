# P5-004 Repository Summary Reporting

## Story

As a repository maintainer, I want repository-wide summary reporting so that I can understand
scanner results across many skills quickly.

## Scope

- aggregate counts by file, category, and severity for a whole run
- a summary in all three surfaces: console, JSON, and the HTML report
- per-file detail stays reachable, not replaced

## The JSON Shape Decision

This is the decision that dominates the story, so make it before writing code.

Today a single-file run emits a JSON *object* and a multi-file run emits an *array* of that object.
A top-level summary cannot be added to either without changing what a consumer parses. Take the
**opt-in** route:

- default output is unchanged, object or array exactly as it is now
- `--summary` switches the payload to `{"summary": {...}, "results": [...]}`, where `results` is
  always the array form regardless of file count

That keeps `P4-001`'s contract intact for anything already parsing it, and gives new consumers one
stable shape rather than two. Do not add `summary` to the bare array — an array has nowhere to put
it, and switching the default to an object would break existing callers silently.

## Decisions To Implement

- **Counts.** Files scanned, files clean, files flagged, files failed to scan; findings by severity;
  findings by category; findings by source. Suppressed findings (`off` in `P5-003`) are excluded
  from every count — they are not findings any more.
- **Console.** The run tally from `P4-004` stays as the default one-liner. `--summary` adds the
  category and source breakdown beneath it. A single-file run with `--summary` prints the breakdown
  too; there is no reason to special-case it.
- **HTML report.** The summary always renders, no flag — it is the one surface with room for it, and
  the page already covers the whole run. Put it above the skill picker.
- **Sorting.** Category and source rows sort by count descending, then name ascending, so repeated
  runs produce stable, diffable output.

## Proposed Changes

- add summary computation to the `result` package, over a slice of results — it needs no knowledge
  of files, so pass counts in rather than teaching `result` about paths
- add `--summary` to `parseOptions` and thread it into both renderers
- extend `report.Render` with the run-level summary block

## Acceptance Criteria

- summary counts equal the detailed findings above them, asserted by a test that derives both from
  one fixture set
- default JSON output is byte-identical to today for both single and multi-file runs
- `--summary` JSON produces the documented `{"summary", "results"}` object, including for one file
- files that failed to scan are counted as failed and are absent from `results`
- the HTML report shows the summary for every run, including a single-file run
- category and source ordering is deterministic across runs

## Must Not Regress

- the default `--json` contract, and `--json` printing nothing else on stdout
- one run, one report
- per-file findings remain fully present alongside any summary

## Documentation

- document `--summary` and the summary JSON shape in `README.md`

## Dependencies

- depends on `P4-002`
- coordinate with `P4-004` for the console tally it extends, and `P5-003` for suppressed findings
- does **not** depend on policy support; it can ship before any of `P5-001`–`P5-003`
