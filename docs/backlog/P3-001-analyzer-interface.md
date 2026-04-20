# P3-001 Analyzer Interface

## Story

As a maintainer, I want an analyzer interface so that LLM-backed analysis can be optional, swappable, and testable.

## Scope

- define an abstraction for semantic analysis
- allow scanner execution with or without an analyzer
- make mocking straightforward in tests

## Proposed Changes

- add an analyzer interface with a small surface area
- update scanner orchestration to treat LLM analysis as optional
- provide a Gemini-backed implementation behind the interface

## Acceptance Criteria

- scanner can run in deterministic-only mode
- scanner can run with an analyzer implementation when configured
- tests can inject a fake analyzer without live model calls

## Dependencies

- builds on `P1-004` and `P2-001`
