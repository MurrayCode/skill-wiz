# P4-006 HTML Report

## Story

As a CLI user, I want each scan to produce a static HTML report so that I can review findings in a browser instead of scrolling terminal output.

## Scope

- self-contained HTML file written on every successful scan
- console pointer to the generated file
- branded, readable presentation of findings

## Proposed Changes

- add a `report` package that renders a `result.Result` to a standalone HTML document
- write the report from the CLI after rendering console output
- print the report path and a `file://` link to stdout

## Acceptance Criteria

- clean and flagged scans both produce a report
- validation-only failures produce a report without running rules or the analyzer
- findings show severity, category, source, and evidence, ordered by severity
- untrusted skill content is escaped and cannot inject markup
- report write failures warn without failing the scan

## Dependencies

- depends on `P3-004`
