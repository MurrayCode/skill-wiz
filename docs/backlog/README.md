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

### Phase 6: Performance, Correctness and Maintainability

Unlike Phases 1-5, this phase does not come from `docs/proposal.md`. It was raised on 2026-08-31
from an audit of the implemented code, so its stories describe defects and costs in what already
exists rather than new capability. That makes the phase independent of Phase 5 — nothing here waits
on policy support, and the two can proceed in either order.

- `P6-001-drop-unused-adk-dependency.md`
- `P6-002-unicode-safe-tokenisation.md`
- `P6-003-collapse-equivalent-paths.md`
- `P6-004-linear-time-rule-scanning.md`
- `P6-005-concurrent-multi-file-scanning.md`
- `P6-006-shared-severity-vocabulary.md`
- `P6-007-extract-console-renderer.md`
- `P6-008-structural-prompt-hardening.md`
- `P6-009-preflight-api-key.md`

### Phase 7: Integration Readiness

Like Phase 6, this phase does not come from `docs/proposal.md`. It comes from
`docs/publishing-integration.md`, the `P5-005` assessment, which concluded that no publishing
integration should be built yet — there is no named consumer — but named six concrete gaps worth
closing regardless of whether anything is ever integrated. These stories are those gaps. None of
them is an integration surface: no GitHub Action, no exported library entry point, no server. Each
one makes a future integration cheaper without promising anything to a stranger.

- `P7-001-rule-ids-in-json-output.md`
- `P7-002-configurable-report-destination.md`
- `P7-003-require-analysis-flag.md`
- `P7-004-ci-recipe-documentation.md`
- `P7-005-required-metadata-rule.md`
- `P7-006-finding-source-positions.md`

#### Deliberately Not Stories Yet

Two gaps from the assessment have no story, on purpose:

- **A JSON schema version** (§1, gap 5). Worth a field before a third party depends on the payload,
  but the contract has only ever grown additively and nothing external reads it yet. Raise it the
  moment a consumer is named.
- **Splitting exit code `1`** (§1, gap 6). Usage errors, discovery errors, and one unparseable skill
  all return `exitFailure`. That is a documented deliberate simplification; splitting it changes a
  contract every existing caller depends on, to serve a caller that does not exist.

The integration surfaces themselves — a GitHub Action, a registry API, a shipped pre-commit hook —
stay parked with `P5-005` until a real consumer exists.

### Phase 8: Project Infrastructure

Raised on 2026-09-02. This phase is about the repository rather than the tool: the checks, the
pipeline, and the distribution that have so far been a matter of running the documented commands by
hand. Nothing here changes what the scanner finds.

- `P8-001-ci-and-release-pipeline.md`

It is independent of every other phase and can be picked up at any point. Landing it early is worth
more than landing it late — every phase after it is verified by it.

### Phase 9: Multi-Provider Analysis

Raised on 2026-09-02. The analysis leg has only ever spoken to Gemini, which makes the credential a
user needs an accident of who wrote the tool. This phase makes the provider a choice.

- `P9-001-provider-neutral-analyzer.md`
- `P9-002-openai-provider.md`
- `P9-003-anthropic-provider.md`
- `P9-004-xai-grok-provider.md`

The deterministic rules are unaffected — they never call a model, which is the project's stated
design goal and the reason this phase changes what the enrichment layer runs on without changing
what a rules-only run finds.

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

### Phase 6 Sequence

Ordered by value against risk, not by story number:

1. `P6-001-drop-unused-adk-dependency.md` — no code change, no risk, removes eleven dependencies
2. `P6-004-linear-time-rule-scanning.md` — contained to `rules`, measured, verifiable against `examples/`
3. `P6-002-unicode-safe-tokenisation.md` — same package as `P6-004`; land one, rebase the other
4. `P6-003-collapse-equivalent-paths.md` — small, self-contained in `discover`
5. `P6-009-preflight-api-key.md` — worth landing before `P6-005`, which would multiply the noise it removes
6. `P6-005-concurrent-multi-file-scanning.md` — the largest win and the largest change
7. `P6-006-shared-severity-vocabulary.md` — pure refactor, prerequisite for the next
8. `P6-007-extract-console-renderer.md` — depends on `P6-006`
9. `P6-008-structural-prompt-hardening.md` — independent; can land at any point

`P6-002` and `P6-003` fix defects rather than tune performance, so pull them forward if
non-ASCII skills or overlapping path arguments are in real use.

### Phase 7 Sequence

Ordered smallest first; the first four are independent of each other and of everything else:

1. `P7-001-rule-ids-in-json-output.md` — one field, already on the struct
2. `P7-002-configurable-report-destination.md` — a flag over the existing `reportPath` seam
3. `P7-003-require-analysis-flag.md` — a flag over the existing `P6-009` preflight
4. `P7-004-ci-recipe-documentation.md` — documentation only, and the highest value per line changed;
   best written once the two flags above exist, but do not let it wait on them
5. `P7-005-required-metadata-rule.md` — needs per-rule configuration in the policy schema
6. `P7-006-finding-source-positions.md` — largest, and touches every rule that carries evidence

If only one story from this phase is ever done, make it `P7-004`: the assessment's finding is that
the two integrations anyone actually wants are already buildable, and what is missing is the page
that says so.

### Phase 8 Sequence

One story, no ordering. Do it before the next phase of feature work rather than after, so the
verification exists while the code is being changed rather than being applied to it retrospectively.

### Phase 9 Sequence

`P9-001` is a hard prerequisite for the other three; nothing else should start before it lands.

1. `P9-001-provider-neutral-analyzer.md` — the seam, the CLI surface, and the conformance suite
2. `P9-002-openai-provider.md` — the first provider through the seam, and the one that proves it
3. `P9-003-anthropic-provider.md` — independent of `P9-002`; the only one with no JSON response mode
4. `P9-004-xai-grok-provider.md` — smallest, and depends on `P9-002`'s request path

`P9-002` and `P9-003` can proceed in parallel once the seam exists. The order above puts the OpenAI
provider first only because it is the most conventional of the three, so a seam that is wrong in
some way is likelier to reveal it there than on the provider with the awkward differences.

## Tracking

- story implementation status is recorded in `tracker.md`
- update `tracker.md` whenever a story in this directory changes to `in_progress`, `done`, or `blocked`
- the project skill `.opencode/skill/backlog-tracker/SKILL.md` defines the workflow for updating the tracker consistently
