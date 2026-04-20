# P5-004 Repository Summary Reporting

## Story

As a repository maintainer, I want repository-wide summary reporting so that I can understand scanner results across many skills quickly.

## Scope

- aggregate counts by file, category, and severity
- preserve per-file details
- support local and CI review flows

## Proposed Changes

- add a summary layer on top of multi-file scanning
- provide counts and totals in both text and JSON output
- keep detailed findings accessible

## Acceptance Criteria

- multi-file scans include a top-level summary
- summary counts match detailed findings
- output remains understandable for both small and large scans

## Dependencies

- depends on `P4-002` and policy maturity
