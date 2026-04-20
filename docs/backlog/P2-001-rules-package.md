# P2-001 Rules Package

## Story

As a maintainer, I want a dedicated rules package so that deterministic checks can be implemented and tested independently from parsing and LLM analysis.

## Scope

- create package boundaries for deterministic rules
- define how rules receive a parsed skill and return findings
- keep rule execution composable

## Proposed Changes

- add a `rules` package
- define a rule interface or simple rule function contract
- add a scanner orchestration path that runs rules and collects findings

## Acceptance Criteria

- at least one rule can be executed through a common interface
- rules return structured findings
- scanner orchestration can aggregate findings from multiple rules
- unit tests cover rule execution order or aggregation behaviour

## Dependencies

- depends on `P1-003`
