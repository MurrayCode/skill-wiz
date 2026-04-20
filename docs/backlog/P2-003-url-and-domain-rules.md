# P2-003 URL And Domain Rules

## Story

As a user, I want the scanner to flag suspicious or unrelated URLs so that skills cannot quietly redirect the agent to off-topic or risky external resources.

## Scope

- detect URLs in the body
- compare linked domains against stated purpose where possible
- flag obviously unrelated domains as suspicious

## Proposed Changes

- extract URLs from skill content
- classify domains or links against the skill description and body intent
- emit findings with evidence and rationale

## Acceptance Criteria

- `examples/MISMATCHSKILL.md` produces a suspicious or mismatch finding for the bird watching domain
- clean examples do not produce unrelated-domain findings
- tests cover multiple URLs and mixed-content cases

## Dependencies

- depends on `P2-001`
