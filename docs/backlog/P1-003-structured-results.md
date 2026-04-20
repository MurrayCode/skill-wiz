# P1-003 Structured Results

## Story

As a user of `skill-wiz`, I want scan output represented as structured findings so that results are testable, automatable, and reusable outside the CLI.

## Scope

- define result and finding types
- capture category, severity, message, and evidence
- support both validation and rule-based findings

## Proposed Changes

- introduce a result model such as `Result` and `Finding`
- keep the data model independent from CLI rendering
- support a clean versus flagged summary state

## Acceptance Criteria

- scan output can be represented without relying on free-form text
- findings include enough evidence for a human to verify them
- the model can represent a clean result with no findings
- unit tests cover result creation and expected zero-finding behaviour

## Dependencies

- enables later work in rules, CLI JSON output, and policy support
