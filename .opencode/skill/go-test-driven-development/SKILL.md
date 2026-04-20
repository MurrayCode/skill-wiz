---
name: go-test-driven-development
description: |-
  Apply Go test-driven development with small red-green-refactor loops, starting from behaviour-focused tests before implementation. Use for adding new Go features safely, fixing regressions with a reproducing test first, or evolving parser and scanner behaviour through incremental test coverage. Use proactively when a Go change is easiest to drive from an expected failing test.

  Examples:
  - user: "Implement validation for required fields" → write the failing Go tests first, then add the smallest implementation to pass
  - user: "Fix this parser bug" → reproduce the bug in a Go test before changing the parser
  - user: "Add deterministic URL scanning" → drive the rule design from failing scanner tests
  - user: "Refactor this behaviour safely" → lock the current or intended behaviour with tests, then refactor in small steps
---
# Go Test-Driven Development

Use this skill when implementing or fixing Go behaviour through short, disciplined test-first cycles.

## Goal

Deliver Go changes by specifying behaviour in tests first, then implementing the smallest correct change, then refactoring while keeping tests green.

## Workflow

1. Identify the behaviour to add, fix, or protect.
2. Write a failing Go test that captures that behaviour clearly.
3. Run the smallest relevant test scope and confirm it fails for the expected reason.
4. Implement the smallest code change needed to make the test pass.
5. Run the same test again.
6. Refactor only after the behaviour is covered and passing.
7. Run the broader relevant test set when the change is complete.

## Guidelines

- start from observable behaviour, not internal implementation
- keep each loop small enough that failures stay obvious
- prefer reproducing bugs with a focused regression test before editing production code
- add only enough implementation to satisfy the current test
- refactor after green, not during red
- keep test names explicit about the behaviour being driven

## Good Fits

- parser and validation changes
- scanner rule additions
- regression fixes
- CLI behaviour with clear input and output expectations

## Avoid

- writing large batches of speculative tests before feedback
- changing production code before reproducing the failure when fixing a bug
- using TDD mechanically when the task is pure documentation or trivial formatting

## Output Expectations

When using this skill:

1. explain the red-green-refactor sequence briefly in progress updates when useful
2. keep tests and implementation changes closely paired
3. run the narrowest useful test first, then broader verification at the end
