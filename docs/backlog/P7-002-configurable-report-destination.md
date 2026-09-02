# P7-002 Configurable Report Destination

## Story

As a CI job, I want to choose where the HTML report is written, so that I can place it in an
artifacts directory without changing directory first or moving the file afterwards.

## Scope

- a flag that sets the report's destination path
- keep the current default exactly as it is

## The Problem

`reportFileName` is a constant (`main.go:27`) and `defaultReportPath` joins it to `os.Getwd()`
(`main.go:518`). There is no way to redirect it, so a CI job that wants the page under
`./artifacts/` has to `cd` before the run or `mv` after it — both of which also move or rename the
`report_path` value the JSON reports. `reportPath` is already a package-level `var` used as a test
seam, so the flag threads into an existing hole rather than opening a new one.

`docs/publishing-integration.md` §1 lists this as gap 3 and §3 ranks it second by value per line
changed.

## Decisions To Implement

- **`--report <path>`, naming a file, not a directory.** A directory argument would need a rule for
  what to call the file inside it and a second rule for what happens when it does not exist. One
  path, one file, no inference.
- **Create the parent directory if it is missing.** A CI job writing to `artifacts/report.html`
  should not have to `mkdir -p` first. Failing to create it is a report failure, not a run failure —
  see below.
- **The report still never fails the scan.** `writeReport` warns on stderr and returns; an
  unwritable `--report` path behaves the same way. The console output already carries every finding,
  and a gate must not go green or red on where a file landed. A bad path is a warning, not exit `1`.
- **`report_path` in the JSON follows the flag.** It is the pointer to the page that was actually
  written, so it must be the resolved destination — one run, one report, one path, as today.
- **Do not add `--no-report`.** Suppressing the page is a different request with a different
  argument about what the JSON should then say. Keep this story to the destination.

## Proposed Changes

- add `--report` to `parseOptions` and thread it through to `writeReport`
- resolve the destination once, ahead of the scan, so a bad flag value is reported before work starts
- keep `reportPath` as the default and as the test seam

## Acceptance Criteria

- `--report <path>` writes the page at that path and the console pointer names it
- every JSON entry's `report_path` is the resolved destination
- a missing parent directory is created
- an unwritable destination warns on stderr and leaves the exit code decided by the findings alone
- with no flag, the destination and the output are byte-identical to today

## Must Not Regress

- one run, one report — however many files a run covers
- the report is still written on the validation path
- `--json` prints the JSON and nothing else; the pointer and any warning go elsewhere
- the tests still never write `skill-wiz-report.html` into the repository

## Documentation

- document `--report` in the flags table in `README.md`
- update the report section to say the destination is configurable and defaults to the working
  directory

## Dependencies

- none
