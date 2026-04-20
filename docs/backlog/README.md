# Skill-Wiz Backlog

This backlog breaks `docs/proposal.md` into individual implementation stories.

## Principles

- keep stories small enough to complete independently
- preserve the phase ordering from the proposal
- make each story testable with clear acceptance criteria
- prioritise deterministic scanning before deeper LLM work

## Story Order

### Phase 1: Stabilise the Core

- `P1-001-parser-input-variants.md`
- `P1-002-skill-validation.md`
- `P1-003-structured-results.md`
- `P1-004-analysis-error-handling.md`
- `P1-005-core-test-coverage.md`

### Phase 2: Deterministic Rule Scanning

- `P2-001-rules-package.md`
- `P2-002-shell-execution-rules.md`
- `P2-003-url-and-domain-rules.md`
- `P2-004-mismatch-heuristics.md`
- `P2-005-fixture-driven-tests.md`

### Phase 3: Safer LLM Integration

- `P3-001-analyzer-interface.md`
- `P3-002-prompt-hardening.md`
- `P3-003-structured-llm-output.md`
- `P3-004-merge-rule-and-llm-findings.md`
- `P3-005-llm-failure-tests.md`

### Phase 4: CLI Upgrade

- `P4-001-cli-flags.md`
- `P4-002-multi-file-scanning.md`
- `P4-003-exit-codes.md`
- `P4-004-human-readable-output.md`
- `P4-005-readme-usage.md`

### Phase 5: Policy and Product Expansion

- `P5-001-policy-support.md`
- `P5-002-environment-specific-rules.md`
- `P5-003-severity-overrides.md`
- `P5-004-repository-summary-reporting.md`
- `P5-005-publishing-integrations.md`

## Suggested Delivery Sequence

The first recommended build slice is:

1. `P1-003-structured-results.md`
2. `P1-004-analysis-error-handling.md`
3. `P1-002-skill-validation.md`
4. `P2-001-rules-package.md`
5. `P2-002-shell-execution-rules.md`
6. `P2-003-url-and-domain-rules.md`
7. `P2-005-fixture-driven-tests.md`

That sequence creates the scanner core quickly while keeping the implementation incremental.

## Tracking

- story implementation status is recorded in `tracker.md`
- update `tracker.md` whenever a story in this directory changes to `in_progress`, `done`, or `blocked`
- the project skill `.opencode/skill/backlog-tracker/SKILL.md` defines the workflow for updating the tracker consistently
