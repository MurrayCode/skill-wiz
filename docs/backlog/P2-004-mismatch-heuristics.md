# P2-004 Mismatch Heuristics

## Story

As a user, I want the scanner to detect obvious contradictions between the skill description and instructions so that off-purpose behaviour is caught deterministically.

## Scope

- compare declared description against instructions and capabilities
- detect unrelated instruction sections
- keep heuristics simple and explainable

## Proposed Changes

- implement keyword or phrase overlap heuristics
- identify when instruction topics diverge sharply from the declared purpose
- generate mismatch findings with clear evidence

## Acceptance Criteria

- `examples/MISMATCHSKILL.md` produces at least one mismatch finding
- findings explain which declared purpose conflicts with which instruction content
- heuristics remain deterministic and testable

## Dependencies

- depends on `P2-001`
