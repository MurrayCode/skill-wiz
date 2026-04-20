# P4-004 Human Readable Output

## Story

As a CLI user, I want readable terminal output so that I can quickly understand scan results without parsing JSON.

## Scope

- concise summary output
- clear listing of findings by file
- evidence visibility for review

## Proposed Changes

- add text rendering for clean and flagged results
- group findings by file and severity
- keep output compact enough for terminal and CI logs

## Acceptance Criteria

- clean scans produce a concise success summary
- flagged scans show categories, severity, and evidence
- multi-file output remains readable

## Dependencies

- depends on `P4-002`
