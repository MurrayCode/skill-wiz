# P3-002 Prompt Hardening

## Story

As a maintainer, I want LLM prompt construction to be safer so that untrusted skill bodies have less influence over the scanner verdict.

## Scope

- improve prompt boundaries around untrusted input
- reduce instruction bleed from scanned content
- keep the prompt format inspectable and testable

## Proposed Changes

- separate system intent from scanned content more clearly
- quote or delimit skill content consistently
- reduce reliance on free-form concatenation

## Acceptance Criteria

- prompt construction uses clear boundaries between instructions and input data
- prompt content is covered by focused tests
- scanner behaviour degrades safely if the analyzer cannot produce a usable response

## Dependencies

- depends on `P3-001`
