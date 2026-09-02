# P7-006 Finding Source Positions

## Story

As an integration that annotates code, I want each finding to say where in the file it came from, so
that a reviewer sees it on the offending line rather than in a list beside the file.

## Scope

- a position on findings that have one
- every rule that produces evidence updated to carry it
- position surfaced in the JSON

## The Problem

`result.Evidence` is a single `Summary` string (`result/result.go:121`) — a snippet, not a position.
Nothing in `result.Finding` records a line or an offset. `docs/publishing-integration.md` §1 lists
this as gap 2 and §3 marks it the one thing that makes a GitHub Action with inline annotations
unbuildable: it is a change to the rule contract, not an integration.

It is the largest story in the phase and the only one that touches every rule, which is why it sits
last despite being the prerequisite for the most visible integration.

## Decisions To Implement

- **Line numbers, 1-based, relative to the file.** Not byte offsets: every consumer that would use
  this — a review comment, an editor, a CI annotation — speaks lines, and a byte offset would have to
  be converted by each of them.
- **The position is on the evidence, not the finding.** Evidence is the thing that has a location; a
  finding without evidence has nothing to point at. Extending `Evidence` keeps the "no evidence, no
  position" case representable without a sentinel on `Finding`.
- **Optional, and zero means unknown.** Validation findings concern absent frontmatter fields and
  analyzer findings come from a model that is not given line numbers — neither can honestly claim
  one. A consumer must be able to tell "line 12" from "no position", so the JSON omits the field
  rather than emitting `0`.
- **Lines are counted in the file, not in the body.** A rule sees `Skill.Body`, which starts after
  the frontmatter, so a body-relative line number would point at the wrong place in the file the
  reviewer opens. The offset from the frontmatter must be accounted for; that is a parser
  responsibility, since `skill.Parse` is the only thing that knows where the body began.
- **It must not change `findingKey`.** `Merge` dedupes on category, severity, message, and evidence
  summary. Adding position to the key would stop a rule and the analyzer collapsing on the same
  issue, because the analyzer has no position — the exact regression `P5-003` guarded against for
  severity. Position is additive to `Evidence` and excluded from the hash.
- **Do the rules that already carry evidence, and no more.** The line-oriented ones —
  `shell-script`, `shell-command`, `unrelated-url` — know their line already. `description-mismatch`
  reports a section and `empty-body` has nowhere to point; leaving those without a position is
  correct, not incomplete.
- **JSON only in this story.** The console has a path header and the report has a card; deciding how
  either renders a position is a separate design question. Ship the machine surface, which is what
  the annotating consumer reads.

## Proposed Changes

- add an optional line to `result.Evidence`, excluded from `findingKey`
- make `skill.Parse` record where the body starts so body lines can be reported as file lines
- carry the line through the rules that match per line
- serialise it as an additive, omitted-when-absent JSON field

## Acceptance Criteria

- a `shell-script` finding on a skill reports the file line its evidence came from, verified by
  opening the fixture at that line
- the reported line accounts for the frontmatter, proven by a fixture with a long frontmatter block
- CRLF input and a trailing `\n---` fence both produce the same line numbers as their LF equivalents
- findings with no position omit the JSON field entirely
- a rule and an analyzer finding that dedupe today still dedupe when only one has a position
- the `examples/` corpus produces the same findings as before, with positions added

## Must Not Regress

- `Merge`'s content-based dedupe and rule-first provenance
- console and HTML report output, which do not change under this story
- rune-safe tokenisation and the linear-time scanning from `P6-002` and `P6-004` — counting lines
  must not reintroduce a per-line rescan of the body
- the `examples/` regression corpus

## Documentation

- document the JSON field in `README.md`
- state which rules produce positions and which cannot, so a consumer does not treat absence as a bug
- record the `findingKey` exclusion in `CLAUDE.md` alongside the existing dedupe invariant

## Dependencies

- none, but it touches every rule that produces evidence, so land it when `rules` is otherwise quiet
- `P7-001` pairs with it: an annotation wants the rule ID and the line together
