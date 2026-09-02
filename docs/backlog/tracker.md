# Story Tracker

This tracker records implementation progress for backlog stories in `docs/backlog/`.

## Status Values

- `todo`: story has not started
- `in_progress`: story is currently being implemented
- `done`: story has been implemented and verified
- `blocked`: story cannot currently be completed

## Update Rules

1. Update this file when a backlog story is completed.
2. Mark a story as `done` only after implementation and verification are complete.
3. Add a short completion note with the date and a brief outcome.
4. If a story is blocked, capture the reason clearly.

## Phase 1

| Story | Status | Completed On | Notes |
| --- | --- | --- | --- |
| `P1-001-parser-input-variants.md` | `done` | 2026-04-20 | Added CRLF and trailing-newline parser coverage and verified `go test ./...`. |
| `P1-002-skill-validation.md` | `done` | 2026-05-24 | Added required metadata validation with field-level errors, surfaced validation findings before analysis, and verified `go test ./...`. |
| `P1-003-structured-results.md` | `done` | 2026-04-20 | Added reusable result and finding types, structured analyzer mapping, and tests for clean versus flagged behavior. |
| `P1-004-analysis-error-handling.md` | `done` | 2026-05-24 | Added explicit missing-key and upstream failure handling in `analyse.Analyze`, moved CLI failures into a testable `run` path, and verified `go test ./...`. |
| `P1-005-core-test-coverage.md` | `done` | 2026-05-24 | Expanded parser and validation coverage for error and whitespace cases, strengthened result model tests, and verified `go test ./...`. |

## Phase 2

| Story | Status | Completed On | Notes |
| --- | --- | --- | --- |
| `P2-001-rules-package.md` | `done` | 2026-05-24 | Added a `rules` package with a shared rule contract, default deterministic rule set, scanner aggregation, and verified `go test ./...`. |
| `P2-002-shell-execution-rules.md` | `done` | 2026-05-24 | Added default shell execution rules with evidence snippets, tuned local script execution to error severity, covered the hidden bash fixture, and verified `go test ./...`. |
| `P2-003-url-and-domain-rules.md` | `done` | 2026-05-24 | Added deterministic URL extraction and unrelated-domain rule heuristics, covered mismatch and mixed-URL cases, and verified `go test ./...`. |
| `P2-004-mismatch-heuristics.md` | `done` | 2026-05-24 | Added deterministic description-versus-instruction mismatch heuristics with section-level evidence and verified `go test ./...`. |
| `P2-005-fixture-driven-tests.md` | `done` | 2026-05-24 | Added fixture-driven scanner regression coverage for the example skills, tuned deterministic mismatch and URL heuristics to match the fixture corpus, and verified `go test ./...`. |

## Phase 3

| Story | Status | Completed On | Notes |
| --- | --- | --- | --- |
| `P3-001-analyzer-interface.md` | `done` | 2026-05-24 | Added a swappable analyzer interface with scanner orchestration, deterministic-only execution, Gemini adapter wiring, and verified `go test ./...`. |
| `P3-002-prompt-hardening.md` | `done` | 2026-05-24 | Hardened analyzer prompts with separate system instructions, JSON-delimited skill payloads, unusable-response fallback findings, and verified `go test ./...`. |
| `P3-003-structured-llm-output.md` | `done` | 2026-05-24 | Switched Gemini analysis to structured JSON output, validated and mapped analyzer findings safely, and verified `go test ./...`. |
| `P3-004-merge-rule-and-llm-findings.md` | `done` | 2026-05-24 | Merged rule and analyzer findings into one report, de-duplicated exact overlaps, preserved source provenance in output, and verified `go test ./...`. |
| `P3-005-llm-failure-tests.md` | `done` | 2026-08-30 | Added missing-key, client-creation, upstream-failure, timeout and malformed-output coverage, an integration-style `GeminiAnalyzer`-through-`scanner.Scan` failure table, and CLI cases asserting degraded scans and error surfacing; all stubbed, no live model access. Verified `go test ./...`. |

## Phase 4

| Story | Status | Completed On | Notes |
| --- | --- | --- | --- |
| `P4-001-cli-flags.md` | `done` | 2026-08-30 | Added `--json`, `--model`, and `--timeout` flag parsing with clear invalid-value errors, threaded an `analyse.Config` through the analyzer seam, added machine-readable JSON output, and verified `go test ./...`. |
| `P4-002-multi-file-scanning.md` | `done` | 2026-08-30 | Added a `discover` package that expands file and directory paths into skill files, scanned every discovered file per run without letting one bad file stop the rest, headed multi-file output with per-file paths, rendered every scanned skill into one HTML report with a dropdown picker, and extended `--json` to an array for multi-file runs. Verified `go test ./...`. |
| `P4-003-exit-codes.md` | `done` | 2026-08-31 | Added named exit codes (`0` clean, `1` operational failure, `2` findings at or above the threshold), a `--fail-on` flag accepting `error`/`warning`/`info` with a clear rejection of anything else, and an `exitCode` helper giving operational failures precedence over findings on both the text and JSON paths. Verified `go test ./...`. |
| `P4-004-human-readable-output.md` | `done` | 2026-08-31 | Ordered console findings by severity at render time on a copy so the JSON contract is untouched, closed multi-file runs with a single tally, coloured severity labels only when stdout is a terminal with `--no-color` and `NO_COLOR` opt-outs, truncated console evidence at 200 runes while the HTML report keeps the full text, and documented all four in the README. Verified `go test ./...`. |
| `P4-005-readme-usage.md` | `done` | 2026-08-31 | README already covered install, usage, environment variables and tests; corrected the stale single-file claim after `P4-002`, documented multi-path and directory scanning, the multi-file console and JSON array shapes, the one-run-one-report page, and the `discover` package in the layout. |
| `P4-006-html-report.md` | `done` | 2026-08-30 | Added a `report` package that renders scans to a self-contained dark-themed HTML page, wired it into the CLI with a `file://` pointer, and verified `go test ./...`. |

## Phase 5

| Story | Status | Completed On | Notes |
| --- | --- | --- | --- |
| `P5-001-policy-support.md` | `done` | 2026-08-31 | Added stable rule IDs as the prerequisite commit, then a `policy` package that loads one strict `.skill-wiz.yaml` document, discovers it in the working directory only, and validates the rule IDs it names against the active rule set. `--policy` wins over discovery; `rules.<id>.enabled: false` filters `rules.Default()` in `main` so `scanner` stays configuration-unaware, and `require` fails the load when a listed rule is missing or disabled. A policy-free run is byte-identical to before, proven against the `examples/` corpus. Verified `go test ./...`. |
| `P5-002-environment-specific-rules.md` | `done` | 2026-08-31 | Added a `profiles:` map to the policy schema and `policy.LoadProfile`, which overlays the selected profile onto the base key by key — a named rule takes the profile's entry whole, an unnamed one keeps the base's, and a profile `require` list replaces the base's outright. Selection is `--profile` only, with no environment auto-detection; an unknown profile fails the run listing the ones that exist, as does `--profile` with no policy file. Resolution stays in `policy`, so `rules` and `scanner` learn nothing about profiles. Verified `go test ./...`. |
| `P5-003-severity-overrides.md` | `done` | 2026-08-31 | Added `rules.<id>.severity` accepting `error`/`warning`/`info`/`off`, validated across the base and every profile at load. `result.Finding` gained `RuleID` (stamped centrally by `rules.Scan`) and `OverriddenFrom`; `policy.Apply` runs in `scanFile` after `scanner.Scan` has merged, so overrides cannot change what `findingKey` collapses — asserted by a test on a rule and analyzer finding that dedupe. `off` drops a finding while the rule still runs. The original severity is additive in all three surfaces: a console suffix, `overridden_from` in the JSON, and a `was <severity>` chip on the report. Verified `go test ./...`. |
| `P5-004-repository-summary-reporting.md` | `done` | 2026-09-02 | Added `result.Summarize`, which aggregates files scanned/clean/flagged/failed and findings by severity, category and source, ranking category and source by count descending then name ascending so repeated runs diff cleanly. `--summary` appends `render.WithSummary` to the console and switches `--json` to `{summary, results}` with `results` always an array; the default JSON shape is untouched. The HTML report renders the summary on every run and derives it from the inputs it shows, so the figures cannot disagree with the findings. Verified `go test ./...`. |
| `P5-005-publishing-integrations.md` | `done` | 2026-09-02 | Delivered `docs/publishing-integration.md`, answering all four questions with references to the code each claim rests on. Conclusion: exit codes, policy, JSON and the run summary already make a pre-commit hook and a CI gate buildable with no new code, while inline annotations and a registry API are blocked on the rule contract (per-finding positions) and a stable library entry point rather than on an integration layer. Six gaps named, three of them one-liners. Recommendation is **park the story** — no named consumer exists — with six follow-up stories listed for when one does. No production code changed. |

## Phase 6

| Story | Status | Completed On | Notes |
| --- | --- | --- | --- |
| `P6-001-drop-unused-adk-dependency.md` | `done` | 2026-08-31 | Ran `go mod tidy`, dropping `google.golang.org/adk` and eleven transitive dependencies (-36 lines across `go.mod`/`go.sum`). No import changed; `go build ./...`, `go vet ./...` and `go test ./...` pass and a second tidy is a no-op. |
| `P6-002-unicode-safe-tokenisation.md` | `done` | 2026-08-31 | Rewrote `splitAlphaNumeric` to range over runes so multi-byte characters stay whole and `tokenSet` holds only valid UTF-8. Added table-driven cases for accented Latin, non-Latin scripts and the unchanged ASCII behaviour, plus an end-to-end `unrelatedURLRule` case that fails against the old byte-indexed splitter. Fixture findings byte-identical. |
| `P6-003-collapse-equivalent-paths.md` | `done` | 2026-08-31 | Keyed the `seen` check on `filepath.Clean` while still appending the path as the user spelled it, so dot segments, doubled separators, parent traversal and trailing slashes collapse and the first spelling is retained. Added a table-driven test that joins by concatenation so `filepath.Join` cannot hide the spellings under test. |
| `P6-004-linear-time-rule-scanning.md` | `done` | 2026-08-31 | Replaced the per-URL `intentTokens` loop with a single `urlPattern` pass over the body, added an allocation-free ASCII case-folded `sh` prefilter ahead of the per-line shell regex, and split `keywords` so `descriptionMismatchRule` tokenises already-stripped segments. Committed benchmarks; a 400-paragraph scan went 24.7ms to 7.9ms and cost per paragraph is now flat (19.9us vs 61.8us). Fixture findings byte-identical. Review follow-up: the prefilter now covers the regexp's Unicode fold set (`ſ` folds to `s`) and the URL strip is applied to the body before joining, so name and description keep the URLs they declare. |
| `P6-005-concurrent-multi-file-scanning.md` | `done` | 2026-08-31 | Replaced the sequential scan loop with `scanFiles`, a bounded worker pool writing into a pre-sized indexed slice, and added `--concurrency` (default 8, rejects non-positive values). Failures are collected with their index and printed in file order, so console, JSON, report, tally and stderr are byte-identical to a sequential run — verified by diffing `--concurrency 1` against the default. Tests use a recording stub analyzer that asserts peak scans in flight and that completion order is the reverse of file order, so a regression to a sequential loop fails the test; `go test -race ./...` passes. |
| `P6-006-shared-severity-vocabulary.md` | `done` | 2026-08-31 | Moved the severity ordering into `result` as one table with `Severities`, `Known`, `GateRank` and `DisplayRank` derived from it, keeping the deliberate disagreement over unknown severities (gates below every known severity, displays last). Moved `FormatSources` (with the report's empty-source guard) and the general `Pluralize` there too and deleted the duplicates from `main` and `report`. Console, JSON and HTML output verified byte-identical against a binary built before the refactor. Review follow-up: `GateRank` now puts an unrecognised severity strictly below `info` rather than level with it, so a malformed finding no longer exits `2` under `--fail-on info` — the one place the pre-refactor code contradicted its own documented guarantee. |
| `P6-007-extract-console-renderer.md` | `done` | 2026-08-31 | Moved console rendering into a `render` package importing `result` and the standard library only, with `render.Input` (path plus result) and an exported `render.Style` so colour stays a caller decision. `main.go` dropped to 539 lines and holds no rendering logic; nine rendering test functions moved out of `main_test.go` into `render` and now run without invoking `run`. Console, JSON and HTML output verified byte-identical against a binary built before the move. |
| `P6-008-structural-prompt-hardening.md` | `done` | 2026-08-31 | Deleted `Analyze(prompt string)` and unexported `AnalyzeWithConfig`, so `GeminiAnalyzer.Analyze` is the only exported way to the model and the `<skill_input>` payload cannot be bypassed. Rewrote the missing-key, client-creation, upstream-failure, timeout and config tests to drive the skill path, using a hostile skill whose body tries to close the wrapper — the prompt assertions now prove `encoding/json` escapes it. |
| `P6-009-preflight-api-key.md` | `done` | 2026-08-31 | Added `analyse.HasAPIKey` and preflighted it once in `run`: a missing key now prints one warning, passes a nil analyzer, and produces a rules-only run instead of one failure per otherwise-clean file. Marked the skipped leg on the console, on the HTML report (covered by table-driven `report.Render` tests), and with an additive `analysis_skipped` JSON field; the per-call check in `analyse` is unchanged. Added a `TestMain` unsetting the key so the suite no longer depends on the developer's environment. |
