# P7-004 CI And Pre-Commit Recipes

## Story

As someone adopting `skill-wiz`, I want the four lines that turn it into a pre-commit hook or a CI
gate, so that I do not have to derive them from the exit code contract myself.

## Scope

- documented, copy-pasteable recipes for a pre-commit hook and a CI gate
- documentation only

## The Problem

`docs/publishing-integration.md` §3 concludes that the two integrations anyone is actually likely to
want — a pre-commit hook and a CI gate — are *already buildable* on the current CLI: exit codes
(`P4-003`), `--fail-on`, and multi-path scanning do the whole job. §4 calls a documented recipe
"the cheapest thing that would be genuinely useful", at zero production code. What is missing is a
page telling someone which four lines to paste.

This story exists so that the recommendation to *not* build an integration surface is accompanied by
the thing that makes an integration unnecessary.

## Why This One First

If exactly one story from Phase 7 is ever done, it should be this one. The assessment's central
finding is not that `skill-wiz` needs integration work — it is that the two integrations anyone
actually wants are *already buildable*, and the only thing missing is the page that says so. Every
other story in the phase makes a future integration cheaper; this one makes the present one
discoverable, at zero production code and zero contract risk.

## Decisions To Implement

- **Recipes, not a shipped hook or action.** No `.pre-commit-hooks.yaml`, no `action.yml`, no
  wrapper script in the repo. Shipping either is a maintained interface with a versioning story
  attached; a documented snippet a reader owns and edits is not.
- **Lead with the exit code.** Both recipes rest on "non-zero means do not publish". State the
  three-way contract at the top of the section, because a reader who understands it can write the
  recipe we did not think of.
- **Show `--fail-on` in the CI recipe.** Strictness is the one decision a team has to make, and it
  is already expressible on the command line. Show `--fail-on warning` and say what it changes.
- **Say what a rules-only run means for a gate.** A CI job with no `GEMINI_API_KEY` still passes on
  rules alone. Point at `analysis_skipped` and, if `P7-003` has landed, at `--require-analysis`.
- **Name what is not covered.** Inline annotations on the offending line are not possible today
  (`P7-006`), and a registry API is not the CLI's job. A reader should learn the boundary from the
  page rather than from a failed afternoon.
- **The pre-commit recipe scans staged files only.** Passing the whole directory on every commit is
  slower and flags files the author did not touch; the recipe should show the staged-path form.

## Proposed Changes

- a section in `README.md` covering: the exit code contract, a pre-commit hook, a CI gate, and the
  limits
- link `docs/publishing-integration.md` from it as the reasoning behind the shape

## Acceptance Criteria

- `README.md` carries both recipes, each runnable as written against a checkout of this repository
- the exit code contract is stated where the recipes can rely on it
- the rules-only degradation and how a gate detects it are covered
- the section names what is out of scope and why, rather than implying full coverage
- every flag the recipes use exists at the time the story lands

## Must Not Regress

- no production code changes under this story
- no new files that constitute a maintained integration surface

## Documentation

- this story *is* the documentation; the deliverable is the `README.md` section

## Dependencies

- rests on `P4-003` (exit codes) and `P4-002` (multi-path scanning), both done
- best written after `P7-002` and `P7-003`, whose flags belong in the CI recipe — but it is
  deliverable without them and should not wait if they slip
