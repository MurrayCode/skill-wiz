# P3-003 Structured LLM Output

## Story

As a user, I want LLM-assisted analysis returned in a structured form so that semantic findings can be merged cleanly with deterministic findings.

## Scope

- request constrained or structured model output
- map model output onto the scanner finding model
- reject unusable responses safely

## Proposed Changes

- define an intermediate response shape for analyzer output
- parse and validate model responses before use
- convert validated analyzer output into findings

## Acceptance Criteria

- analyzer output can be transformed into structured findings
- malformed analyzer output does not crash the scan
- tests cover valid and invalid structured analyzer responses

## Dependencies

- depends on `P3-001` and `P3-002`
