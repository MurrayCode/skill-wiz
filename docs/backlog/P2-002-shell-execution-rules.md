# P2-002 Shell Execution Rules

## Story

As a user, I want the scanner to flag shell execution references so that hidden or risky command execution can be detected without depending on an LLM.

## Scope

- references to `bash`, `sh`, and shell command phrasing
- local script execution such as `./scripts/*.sh`
- tool invocation phrasing that implies command execution

## Proposed Changes

- implement one or more rules for shell-related patterns
- include evidence snippets in findings
- tune severity for explicit local script execution versus generic wording

## Acceptance Criteria

- `examples/HIDDENBASHSKILL.md` produces at least one finding
- evidence includes the triggering text
- false positives are reduced for benign mentions where possible
- unit or fixture tests cover the core shell patterns

## Dependencies

- depends on `P2-001`
