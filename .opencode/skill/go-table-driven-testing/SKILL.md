---
name: go-table-driven-testing
description: |-
  Write and refactor Go tests into table-driven form using idiomatic test case slices, subtests, and focused assertions. Use for adding coverage across multiple inputs, cleaning up repetitive Go tests, or extending parser and validation tests with new fixtures. Use proactively when a Go test repeats the same setup for multiple cases or when new edge cases should be added consistently.

  Examples:
  - user: "Add tests for more parse edge cases" → create or extend a table-driven Go test with explicit expected outcomes
  - user: "Refactor this repetitive Go test" → convert repeated assertions into a table-driven subtest loop
  - user: "Cover CRLF and empty input cases" → add named table entries for each parser variant
  - user: "Make these Go tests easier to extend" → reorganise test inputs into a single table with clear fields
---
# Go Table-Driven Testing

Use this skill when writing or refactoring Go tests that should cover multiple related cases with a consistent structure.

## Goal

Produce idiomatic Go tests that are easy to extend, easy to read, and explicit about inputs and expected outcomes.

## Preferred Pattern

1. Define a local `tests := []struct { ... }{ ... }` slice.
2. Include only the fields needed for the behaviour under test.
3. Add a `name` field for readable subtest output.
4. Loop over the test cases with `for _, tt := range tests`.
5. Use `t.Run(tt.name, func(t *testing.T) { ... })` for each case.
6. Assert on the behaviour that matters for that case and keep the checks direct.

## Guidelines

- prefer one table per behaviour or function under test
- keep the struct fields small and specific
- name cases by behaviour, not by implementation detail
- add edge cases alongside normal cases in the same table when they exercise the same code path shape
- avoid over-abstracting assertion helpers unless they remove clear duplication
- keep setup inline unless it is genuinely reused across many tests

## Assertion Style

- use simple `if got != want` checks when enough
- check errors explicitly with `wantErr` or equivalent fields
- avoid vague assertions that make failures hard to interpret
- include relevant values in failure messages

## Good Candidates

- parser input variants
- validation rules with multiple field combinations
- scanner rules that inspect several text patterns
- CLI argument parsing with valid and invalid inputs

## Avoid

- forcing unrelated behaviours into one table
- creating giant case structs with many unused fields
- hiding key expectations in complex helper code
- using table-driven style when a single-case test is clearer

## Output Expectations

When using this skill:

1. prefer updating existing Go tests before creating new helpers
2. keep the resulting test idiomatic for the surrounding package
3. add only the cases needed for the requested behaviour
