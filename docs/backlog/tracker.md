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
| `P4-002-multi-file-scanning.md` | `done` | 2026-08-30 | Added a `discover` package that expands file and directory paths into skill files, scanned every discovered file per run without letting one bad file stop the rest, headed multi-file output with per-file paths, wrote a per-file HTML report, and extended `--json` to an array for multi-file runs. Verified `go test ./...`. |
| `P4-003-exit-codes.md` | `todo` |  |  |
| `P4-004-human-readable-output.md` | `todo` |  |  |
| `P4-005-readme-usage.md` | `todo` |  |  |
| `P4-006-html-report.md` | `done` | 2026-08-30 | Added a `report` package that renders scans to a self-contained dark-themed HTML page, wired it into the CLI with a `file://` pointer, and verified `go test ./...`. |

## Phase 5

| Story | Status | Completed On | Notes |
| --- | --- | --- | --- |
| `P5-001-policy-support.md` | `todo` |  |  |
| `P5-002-environment-specific-rules.md` | `todo` |  |  |
| `P5-003-severity-overrides.md` | `todo` |  |  |
| `P5-004-repository-summary-reporting.md` | `todo` |  |  |
| `P5-005-publishing-integrations.md` | `todo` |  |  |
