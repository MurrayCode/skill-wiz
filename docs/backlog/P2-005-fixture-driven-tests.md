# P2-005 Fixture-Driven Tests

## Story

As a maintainer, I want fixture-driven scanner tests using the example skills so that core scanner behaviour is protected by realistic regression tests.

## Scope

- use `examples/` as a seed evaluation corpus
- define expected findings for each fixture
- test deterministic scanning without live model calls

## Proposed Changes

- load example skills in tests
- assert on expected categories or counts of findings
- keep assertions stable enough to support iterative rule tuning

## Acceptance Criteria

- `CLEANSKILL.md` is treated as clean
- `MISMATCHSKILL.md` produces mismatch findings
- `HIDDENBASHSKILL.md` produces suspicious or hidden findings
- tests run without external API credentials

## Dependencies

- depends on `P2-002`, `P2-003`, and `P2-004`
