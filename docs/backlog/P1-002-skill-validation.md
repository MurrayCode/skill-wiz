# P1-002 Skill Validation

## Story

As a maintainer, I want parsed skills to be validated so that missing or incomplete metadata is reported clearly before scanning begins.

## Scope

- validate required fields such as `name` and `description`
- validate key structural expectations after parsing
- return field-level validation errors

## Proposed Changes

- add a validation step in `skill` or a closely related package
- define validation errors that callers can surface directly
- keep validation separate from LLM analysis

## Acceptance Criteria

- missing `name` is reported as a validation error
- missing `description` is reported as a validation error
- valid skills pass validation without findings
- validation behaviour is covered with unit tests

## Dependencies

- can be delivered before or after `P1-001`
- should align with the structured result model from `P1-003`
