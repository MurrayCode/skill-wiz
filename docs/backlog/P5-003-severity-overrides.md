# P5-003 Severity Overrides

## Story

As a team, I want severity overrides so that scanner findings match local risk tolerance and workflow expectations.

## Scope

- override severity by rule or category
- preserve sensible defaults
- avoid making results hard to understand

## Proposed Changes

- add severity override support to policy configuration
- apply overrides during result generation
- include source or explanation when severity is policy-adjusted

## Acceptance Criteria

- severity can be raised or lowered through policy
- default severities still apply when no override exists
- tests cover override behaviour

## Dependencies

- depends on `P5-001`
