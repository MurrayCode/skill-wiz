# P1-005 Core Test Coverage

## Story

As a maintainer, I want stronger test coverage around parsing and result generation so that core scanner changes can be made safely.

## Scope

- parser edge cases
- validation behaviour
- result aggregation basics

## Proposed Changes

- extend `skill/skill_test.go`
- add tests for validation errors
- add tests for structured result generation

## Acceptance Criteria

- parser tests cover valid, invalid, CRLF, and empty-body variants
- validation tests cover required fields
- result model tests cover clean and flagged outcomes
- tests do not require live model calls

## Dependencies

- depends on work from `P1-002` and `P1-003`
