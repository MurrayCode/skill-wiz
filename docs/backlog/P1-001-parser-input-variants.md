# P1-001 Parser Input Variants

## Story

As a maintainer, I want `skill.Parse` to accept common skill file formatting variants so that valid skills do not fail because of minor newline or fence differences.

## Scope

- support CRLF line endings
- handle trailing newline variations
- preserve the existing frontmatter plus body structure
- avoid broad format changes that introduce ambiguity

## Proposed Changes

- normalise line endings before parsing
- make frontmatter delimiter handling more tolerant where safe
- keep parsing logic explicit and easy to reason about

## Acceptance Criteria

- CRLF files parse successfully
- files with or without trailing newline parse consistently
- malformed frontmatter still returns a clear error
- existing parser tests continue to pass

## Notes

This story should stay focused on input handling, not validation rules.
