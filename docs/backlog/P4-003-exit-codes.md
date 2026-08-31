# P4-003 Exit Codes

## Story

As a CI user, I want meaningful exit codes so that pipelines can fail when skills are flagged.

## Scope

- a named exit code for a clean run
- a named exit code for a run with findings, distinct from the operational failure code
- a severity threshold that decides which findings gate the build
- precedence when a run both fails on one file and flags another

## Exit Code Contract

| Code | Meaning |
| --- | --- |
| `0` | every scanned file was clean at or above the active threshold |
| `1` | operational failure — usage, discovery, read, parse, or scan error |
| `2` | at least one finding at or above the active threshold |

`1` keeps its current meaning; flagged results take the new code `2`. Do not renumber `1` — the
existing `run` failure paths and their tests depend on it.

## Decisions To Implement

- **Threshold.** `error` is the default gate; `warning` findings alone exit `0`. A `--fail-on`
  flag accepts `error`, `warning`, or `info` and lowers the gate. `--fail-on info` makes any
  finding fail.
- **Validation findings count.** They are `error` severity, so a skill missing `name` or
  `description` exits `2` under the default threshold.
- **Precedence.** An operational failure outranks a finding: if any file failed to read, parse, or
  scan, the run exits `1` even when another file was flagged. Failures are already reported
  per file on stderr, so nothing is lost.
- **`--json` uses the same codes.** The JSON path exits exactly as the text path does.

## Proposed Changes

- add the exit code constants and a helper that maps scans plus the run's failure flag onto a code
- add `--fail-on` to `parseOptions`, rejecting unknown severities with a clear error the way
  `--model` and `--timeout` already do
- return the mapped code from both the text and JSON branches of `run`

## Acceptance Criteria

- a clean scan exits `0`
- a scan flagged at or above the threshold exits `2`
- usage, discovery, read, parse, and scan errors still exit `1`, and outrank findings in a mixed run
- `--fail-on warning` fails a run whose only findings are warnings; the same run exits `0` by default
- an invalid `--fail-on` value is reported as an error, once, and exits `1`
- table-driven tests cover clean, flagged, mixed, threshold, and invalid-flag cases

## Must Not Regress

- `--json` prints the JSON and nothing else — no summary line, no report pointer
- one bad file still never hides the rest; every file is still scanned and reported
- the HTML report is still written on every path, including runs that will exit non-zero

## Documentation

- replace the "Findings do not affect the exit code yet" caveat in `README.md` with the contract
  table above, and document `--fail-on` in the flag table
- update the matching invariant in `CLAUDE.md`

## Dependencies

- depends on `P4-001` and `P4-002`
