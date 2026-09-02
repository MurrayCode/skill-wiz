# P7-001 Expose Rule IDs In JSON Output

## Story

As a consumer of `--json`, I want each rule finding to carry the identifier that produced it, so
that I can address a finding by the same name a policy uses instead of matching on its message.

## Scope

- serialise `result.Finding.RuleID` in the JSON output
- leave every existing field's name and meaning untouched

## The Problem

`result.Finding` already carries `RuleID` (`result/result.go:134`) and `rules.Scan` stamps it
centrally, but `jsonFinding` (`main.go:580`) does not serialise it. A gate reading the JSON can see
that a `shell` finding fired; it cannot tell `shell-script` from `shell-command` without matching on
`message`, which is prose and not a contract. Policy can switch a rule off wholesale, but a consumer
that wants to accept one known finding on one skill has nothing stable to key on.

`docs/publishing-integration.md` §1 names this the single cheapest gap to close, and §3 ranks it
first by value per line changed.

## Decisions To Implement

- **Additive, and named for the thing it identifies.** A new `rule` key on each finding object,
  omitted when empty. Validation and analyzer findings have no rule identity and must not gain a
  placeholder — absent means "not produced by a rule", which is exactly what the struct already says.
- **The IDs are already a public contract.** `CLAUDE.md` states rule IDs may be added but never
  renamed. Publishing them in the JSON does not create that promise, it exposes one that already
  exists for policy files; no ID changes under this story.
- **A schema version is not bundled with this.** `docs/publishing-integration.md` §1 lists one as
  gap 5, and adding a `rule` field is the obvious moment to reach for it. Resist that: the JSON
  contract has only ever grown additively, nothing outside this repository reads it, and a version
  number is a promise about *future* change that nobody has asked for. Raise it when a consumer is
  named — the backlog `README.md` records it under "Deliberately Not Stories Yet" so it is not lost.
- **The console and the report are out of scope.** A rule ID on every console line is noise for a
  human reader, and the report already groups by category. This story is about the machine surface
  only.

## Proposed Changes

- add the `rule` field to `jsonFinding` and populate it where findings are mapped
- cover a rule finding, an analyzer finding, and a validation finding in the JSON tests

## Acceptance Criteria

- a rule finding in `--json` carries `"rule"` set to the ID of the rule that produced it
- validation and analyzer findings omit the field entirely
- every existing JSON field keeps its name, type, and meaning
- both JSON shapes carry it — the single-file object and the multi-file array — and the
  `--summary` `{summary, results}` form too
- asserted against the `examples/` corpus, so the IDs in the output are the real ones

## Must Not Regress

- `Merge`'s content-based dedupe, which deliberately excludes `RuleID` from `findingKey`
- console and HTML report output, which do not change at all
- `--json` prints the JSON and nothing else

## Documentation

- document the `rule` field in the JSON section of `README.md`, next to `overridden_from`
- note that the value is the same identifier a policy file names

## Dependencies

- none; `RuleID` landed with `P5-003`
