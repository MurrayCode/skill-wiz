# P5-001 Policy Support

## Story

As a team, I want configurable policy support so that scanner behaviour can enforce
organisation-specific rules.

## Prerequisite: rules need identities

`rules.Rule` is currently just `Check(*skill.Skill) []result.Finding` and `Default()` returns an
unnamed slice, so no configuration can refer to a rule. Before any policy work:

- add a stable `ID() string` to the `Rule` contract — `shell-script`, `shell-command`,
  `unrelated-url`, `description-mismatch`, `empty-body`
- extend `RuleFunc` so plain functions still satisfy it
- treat the IDs as a public contract from that point on, the same way the JSON field names are

Do this as the first commit of the story, with its own tests, so the policy work lands on top of a
stable naming surface.

## Scope

- a policy file that can disable rules and raise the floor on required behaviour
- loading and validating that file at runtime
- applying it alongside the deterministic rules without changing default behaviour

## Decisions To Implement

- **Format and name.** YAML, `.skill-wiz.yaml`, one document. YAML matches the skill frontmatter the
  tool already parses, so no new dependency class.
- **Discovery.** `--policy <path>` wins. Otherwise look for `.skill-wiz.yaml` in the working
  directory only — no upward search, no home directory, no environment variable. Explicit beats
  implicit, and a scanner whose verdict depends on where it was run from is hard to trust.
- **No policy is a valid run.** With no file and no flag, behaviour is byte-identical to today.
- **What policy addresses.** Rule IDs, not categories. Categories are a display grouping and several
  rules share one; IDs are unambiguous.
- **Initial vocabulary.** Keep it to two keys for this story:
  - `rules.<id>.enabled: false` — skip that rule entirely
  - `require: [<id>, ...]` — fail the load if a listed rule is not in the active rule set, so a
    policy cannot silently stop enforcing something after a refactor
  Severity overrides are `P5-003`; environment profiles are `P5-002`. Design the schema so both can
  be added without renaming these keys.
- **Findings from policy.** This story generates no new findings — it enables and disables existing
  rules. `result.Source` gains no new value. A malformed or unreadable policy is an operational
  failure that stops the run, not a finding.

## Proposed Changes

- add rule IDs as described above
- add a `policy` package: load, validate, and answer "is rule X enabled"
- filter `rules.Default()` through the loaded policy in `main`, leaving `scanner` unchanged
- report policy load and validation errors on stderr with the file path and the offending key

## Acceptance Criteria

- every default rule exposes a stable ID, asserted by a test that fails if one changes
- a run with no policy file produces exactly the findings it produces today, proven against the
  `examples/` corpus
- a policy disabling `shell-script` removes that finding from `HIDDENBASHSKILL.md` and leaves its
  other findings intact
- a policy listing an unknown rule under `require` fails the run with a message naming the rule
- a malformed policy file fails the run once, with the path in the message

## Must Not Regress

- the `examples/` corpus expectations under a default (policy-free) run
- `scanner.Scanner` stays unaware of policy; it still takes a rule slice and an analyzer

## Documentation

- document the file name, discovery order, and the two keys in `README.md`
- add the rule ID table to `README.md` alongside the existing checks table

## Dependencies

- depends on `P1-003` structured results and the `P2-*` rule set, both complete
