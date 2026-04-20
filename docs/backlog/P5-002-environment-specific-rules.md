# P5-002 Environment Specific Rules

## Story

As a team, I want environment-specific rules so that different deployment contexts can allow or forbid different skill behaviours.

## Scope

- support rules such as no shell access
- support rules such as no external URLs
- allow policy to vary by environment or mode

## Proposed Changes

- extend policy configuration with environment-specific controls
- map policy settings onto rule enablement or severity
- keep the configuration model simple at first

## Acceptance Criteria

- one environment can allow behaviour that another forbids
- resulting findings reflect the active environment policy
- tests cover at least two distinct environment configurations

## Dependencies

- depends on `P5-001`
