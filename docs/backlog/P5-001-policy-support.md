# P5-001 Policy Support

## Story

As a team, I want configurable policy support so that scanner behaviour can enforce organisation-specific rules.

## Scope

- disallowed behaviour configuration
- policy-driven finding generation
- simple initial policy format

## Proposed Changes

- define a policy model
- load policy configuration at runtime
- apply policy checks alongside core deterministic rules

## Acceptance Criteria

- policies can disallow selected behaviour classes
- violations are returned as structured findings
- scanner behaviour without a policy remains sensible

## Dependencies

- depends on mature structured results and deterministic rules
