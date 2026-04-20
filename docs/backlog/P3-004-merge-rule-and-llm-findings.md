# P3-004 Merge Rule And LLM Findings

## Story

As a user, I want deterministic and LLM-assisted findings combined into one result so that I receive a single coherent scan report.

## Scope

- merge findings from multiple sources
- preserve evidence and provenance
- avoid duplicate or contradictory output where possible

## Proposed Changes

- add source tracking to findings if needed
- define merge behaviour for overlapping findings
- ensure summary generation reflects both sources

## Acceptance Criteria

- scan results can include deterministic and analyzer findings together
- duplicate findings are reduced or clearly labelled
- tests cover merge behaviour for overlapping categories

## Dependencies

- depends on `P3-003`
