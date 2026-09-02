# P7-003 Require The Analysis Leg

## Story

As a publishing gate, I want to insist that the model analysis actually ran, so that a rules-only
run cannot be mistaken for a full audit.

## Scope

- a flag that turns a degraded, rules-only run into an operational failure
- leave the default degrade-rather-than-fail behaviour untouched

## The Problem

`P6-009` made a missing `GEMINI_API_KEY` degrade to a rules-only run by design: `run` preflights
`analyse.HasAPIKey`, prints one warning, and passes a nil analyzer. That is right for a developer at
a terminal and wrong for a gate — a CI job with an expired or unset secret keeps exiting `0` on
skills nobody has fully audited, and the only signal is a stderr line and
`"analysis_skipped": true` in a payload the gate may not parse.

`docs/publishing-integration.md` §1 lists this as gap 4 and judges the check "more likely to be
wanted than not" for a publishing gate. §3 ranks it third.

## Decisions To Implement

- **`--require-analysis`, off by default.** The current behaviour is the right default; this is an
  opt-in for callers who know they need the model leg. Changing the default would break every
  keyless run.
- **Fail before scanning, not after.** The preflight already happens ahead of the scan loop. With
  the flag set, a missing key exits immediately with a clear message and no file is scanned — the
  same shape as a policy failure, and it avoids spending a directory's worth of work on a run whose
  result is going to be discarded.
- **It is exit code `1`, not `2`.** A configuration problem is an operational failure, and
  `exitCode` already gives operational failures precedence over findings. It is not a finding and
  must not become one — nothing about the *skills* changed.
- **Scope it to the preflight in this story.** A per-file analyzer error that `scanner.Scan`
  currently swallows when rules found something (`scanner/scanner.go:38`) is a different question
  with a different blast radius. Note it as a follow-up; do not change `Scan` here.
- **The message names the cause.** "analysis required but GEMINI_API_KEY is not set" — a gate
  operator reading a CI log should not have to consult the README to learn which secret is missing.

## Proposed Changes

- add `--require-analysis` to `parseOptions`
- in `run`, when the flag is set and `analyse.HasAPIKey` is false, print the message and return the
  operational failure code before any file is scanned

## Acceptance Criteria

- with the flag set and no key, the run exits `1`, prints one clear message naming the variable, and
  scans nothing
- with the flag set and a key present, the run behaves exactly as it does without the flag
- without the flag, a keyless run still degrades to rules-only with one warning, as `P6-009` defined
- the failure is reported on stderr and produces no JSON document on stdout

## Must Not Regress

- the rules-only fallback and its console note, report line, and `analysis_skipped` JSON field
- the scanner's degrade-rather-than-fail behaviour for analyzer errors that are not a missing key
- the three-way exit code contract, and operational failure outranking findings
- the test suite still makes no live API calls; `TestMain` still unsets the key

## Documentation

- document `--require-analysis` in the flags table in `README.md` and next to the `GEMINI_API_KEY`
  note, stating plainly that it is for gates
- add it to the CI recipe in `P7-004` if that story has landed

## Dependencies

- depends on `P6-009` for the preflight it hangs off
- interacts with `P4-003`: it is an operational failure, not a finding
