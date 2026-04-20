# Skill-Wiz Implementation Proposal

## Purpose

`skill-wiz` is currently a proof of concept for scanning a single skill file by parsing frontmatter and body content, then asking an LLM to identify mismatches and suspicious behaviour. The next phase should turn it into a reliable hybrid scanner that combines deterministic checks with LLM-assisted reasoning.

This proposal outlines how to evolve the project from a narrow CLI prototype into a tool that is trustworthy, testable, and suitable for local usage and CI enforcement.

## Current State

The current implementation has a straightforward flow:

1. `main.go` reads a single file path from the command line.
2. `skill.Parse` extracts YAML frontmatter and markdown body content.
3. `main.go` builds a prompt from the description and body.
4. `analyse.Analyze` sends the prompt to Gemini and returns free-text output.

This provides a good starting point, but the current design has a few hard limits:

- Detection depends almost entirely on the LLM.
- Results are unstructured and hard to automate.
- Prompt construction is vulnerable to prompt injection from the skill body.
- Parsing is basic and not yet robust enough for wider input variation.
- The code is designed around a single-file CLI rather than a reusable scanner engine.

## Vision

The target product should be a hybrid scanner with three properties:

1. Deterministic by default for obvious and high-confidence findings.
2. LLM-assisted for semantic reasoning, explanation, and edge cases.
3. Structured in its outputs so it can be used in local workflows, CI pipelines, and future integrations.

In practice, this means the scanner should inspect skill files for:

- mismatches between description and instructions
- suspicious tool or shell execution
- hidden or unrelated behaviour
- risky external access patterns
- metadata and structural issues

## Proposed Architecture

The project should move towards a small set of focused packages.

### `skill`

Responsibilities:

- parse skill files
- validate required frontmatter
- normalise line endings and body formatting
- expose a stable in-memory model

Suggested additions:

- field validation
- explicit parse and validation errors
- support for CRLF and small formatting variations

### `scanner`

Responsibilities:

- orchestrate analysis of a parsed skill
- run deterministic rules
- optionally invoke LLM analysis
- return structured findings

This should become the core package of the application.

### `rules`

Responsibilities:

- detect suspicious patterns using deterministic checks
- inspect body text, metadata, URLs, and references
- return findings with evidence

Initial rule categories:

- shell or script execution references
- local file execution or hidden script calls
- unrelated links or domains
- contradiction between declared purpose and instructions
- suspicious keywords such as exfiltration, hidden execution, secrets, or persistence

### `analyse`

Responsibilities:

- provide LLM-backed semantic analysis behind an interface
- explain or enrich findings from rule-based checks
- optionally detect nuanced mismatches not captured by rules

This package should stop terminating the process directly and instead return errors to the caller.

### `cmd` or improved `main.go`

Responsibilities:

- parse CLI flags
- scan one or many files
- render text or JSON output
- return useful exit codes

If the CLI grows, it should eventually move into `cmd/skill-wiz`.

## Data Model

The scanner should produce structured findings instead of only free-text output.

Suggested finding model:

```go
type Finding struct {
    Category   string
    Severity   string
    Message    string
    Evidence   string
    Source     string
    Confidence string
}
```

Possible categories:

- `mismatch`
- `suspicious`
- `hidden`
- `validation`

Possible severities:

- `info`
- `warning`
- `high`
- `critical`

The scanner should return a result object similar to:

```go
type Result struct {
    FilePath  string
    Findings  []Finding
    Summary   string
    Clean     bool
}
```

This output format will make the tool easier to test and ready for `--json` output.

## Delivery Plan

### Phase 1: Stabilise the Core

Goal: turn the parser and scanner output into something reliable and testable.

Tasks:

1. Improve `skill.Parse` to support more input variants.
2. Add validation for required fields such as `name` and `description`.
3. Introduce structured result and finding types.
4. Refactor `analyse.Analyze` to return errors instead of calling `log.Fatal`.
5. Add tests for parser edge cases and result generation.

Expected outcome:

- safer parsing
- clearer error handling
- a reusable scanner contract

### Phase 2: Add Deterministic Rule Scanning

Goal: reduce dependence on the LLM for obvious or high-confidence detections.

Tasks:

1. Create a `rules` package.
2. Implement checks for script execution references such as `bash`, `sh`, local scripts, and shell command phrasing.
3. Implement checks for suspicious URLs and unrelated domains.
4. Implement checks for description-body mismatch heuristics.
5. Add fixture-driven tests using the files in `examples/`.

Expected outcome:

- repeatable findings for the most important suspicious patterns
- better scanner trustworthiness
- stronger regression coverage

### Phase 3: Make LLM Usage Safer and More Useful

Goal: use the LLM as an enhancement layer, not the sole detection engine.

Tasks:

1. Define an analyzer interface so the scanner can run with or without an LLM.
2. Tighten prompt construction to reduce prompt injection risk.
3. Ask the model for structured reasoning or constrained output.
4. Merge deterministic findings with LLM-assisted findings.
5. Add tests around failure modes and no-key behaviour.

Expected outcome:

- safer LLM integration
- better explainability
- an architecture that can support alternative providers later

### Phase 4: Upgrade the CLI

Goal: make the tool useful for local engineering workflows and CI.

Tasks:

1. Add CLI flags such as `--json`, `--model`, and `--timeout`.
2. Support scanning directories or multiple files.
3. Add exit codes for clean versus flagged results.
4. Improve stdout formatting for human-readable reports.
5. Add README usage examples.

Expected outcome:

- practical local usage
- CI-ready automation
- easier onboarding for contributors

### Phase 5: Policy and Product Expansion

Goal: move beyond a generic scanner into a configurable skill safety tool.

Tasks:

1. Add policy support for disallowed behaviours.
2. Allow environment-specific rules such as no shell access or no external URLs.
3. Add severity overrides and team-specific policy configuration.
4. Support repository-wide scanning and summary reporting.
5. Consider publishing integrations or registry validation flows.

Expected outcome:

- team-specific enforcement
- more compelling product direction
- clearer differentiation from a generic LLM wrapper

## Testing Strategy

Testing should grow with the architecture.

### Unit Tests

- parser behaviour
- validation logic
- individual deterministic rules
- result aggregation

### Fixture Tests

The `examples/` directory should become an evaluation corpus.

Add expected outcomes for each example, for example:

- `CLEANSKILL.md` should produce no findings
- `MISMATCHSKILL.md` should produce mismatch findings
- `HIDDENBASHSKILL.md` should produce suspicious or hidden findings

### Integration Tests

- CLI invocation on known fixtures
- JSON output validation
- error handling when `GEMINI_API_KEY` is missing

### Non-LLM First

The default test strategy should not require external model calls. LLM-dependent behaviour should be mocked or isolated behind interfaces.

## Immediate Recommendations

The best first milestone is:

**Build a deterministic scanner core with structured findings, then layer the LLM on top.**

That keeps the project small, improves reliability quickly, and creates a base that can support CI and policy work later.

The next concrete engineering tasks should be:

1. Create a `scanner` package with a `Result` and `Finding` model.
2. Refactor `analyse.Analyze` to return `(string, error)`.
3. Add validation to `skill.Parse` or a new `Validate` step.
4. Implement the first rule set for shell execution and unrelated URLs.
5. Add fixture-driven tests covering the current examples.

## Longer-Term Opportunities

If the project continues beyond the initial scanner, it could grow into:

- a CI gate for skill repositories
- a linter for skill authoring
- a publishing validator for skill registries
- a policy engine for enterprise usage
- a benchmarked safety scanner with a maintained corpus of good and malicious skills

## Success Criteria

The next version of `skill-wiz` should be considered successful if it can:

1. Scan skills without requiring the LLM for obvious detections.
2. Produce structured findings with evidence.
3. Reliably flag the current suspicious example files.
4. Run cleanly in automated workflows.
5. Remain simple enough for a single contributor to maintain.

## Conclusion

The project already has the right seed: a simple parser, examples, and a clear problem to solve. The most valuable next move is not adding more prompt logic, but building a proper scanner core around deterministic checks, structured results, and safer LLM usage.

That path turns `skill-wiz` from a promising POC into a credible foundation for a real skill safety tool.
