# P4-003 Exit Codes

## Story

As a CI user, I want meaningful exit codes so that pipelines can fail when skills are flagged.

## Scope

- clean exit status
- flagged exit status
- operational error exit status

## Proposed Changes

- define a simple exit code contract
- map scanner result states onto exit behaviour
- document the contract in CLI help or README

## Acceptance Criteria

- clean scans return success
- flagged scans return a non-zero status intended for CI gating
- runtime or configuration errors return a distinct non-zero status

## Dependencies

- depends on `P4-001` and `P4-002`
