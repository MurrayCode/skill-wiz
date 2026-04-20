# P3-005 LLM Failure Tests

## Story

As a maintainer, I want LLM failure-mode coverage so that scanner behaviour remains predictable when the analyzer is unavailable or misconfigured.

## Scope

- missing key behaviour
- upstream API failure behaviour
- malformed analyzer output behaviour

## Proposed Changes

- add integration-style tests around analyzer failure paths
- assert deterministic-only execution still works where intended
- verify errors are surfaced clearly to CLI callers

## Acceptance Criteria

- missing key scenarios are covered
- upstream failures are covered
- malformed analyzer output is covered
- no test requires live remote model access by default

## Dependencies

- depends on `P3-001` through `P3-003`
