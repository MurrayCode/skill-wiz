# P7-005 Required Metadata Rule

## Story

As a registry operator, I want to require metadata fields beyond `name` and `description`, so that
skills cannot be published without the information redistribution needs.

## Scope

- a `required-metadata` deterministic rule whose field list comes from policy
- the per-rule configuration in the policy schema that the field list needs
- parse the fields the rule can be asked about

## The Problem

`Skill.Validate` requires exactly two fields: `name` and `description`. `license` and
`compatibility` are parsed and never checked, and `version` is not parsed at all
(`skill/skill.go:18`). `docs/publishing-integration.md` §2 works through where the check belongs and
rules out both obvious homes:

- **not `skill.Validate`** — validation short-circuits the scan, so a skill missing a `version`
  would never get its body scanned. That is backwards: the shell-execution check matters *most* on
  the skill whose metadata is sloppy.
- **not `policy`** — policy enables, disables, and re-grades rules and generates no findings of its
  own. Teaching it to emit findings makes it a second rule engine.

The right shape is a rule configured by policy: `skill` decides whether a file is parseable, `rules`
decides whether it is acceptable, and `policy` decides what acceptable means here.

## Decisions To Implement

- **Rule ID `required-metadata`**, added to `rules.Default()` and therefore to the public ID
  contract. Like every other ID it may never be renamed.
- **Empty field list by default, so the rule is inert without a policy.** A policy-free run must
  stay byte-identical, and `examples/CLEANSKILL.md` must stay clean — it declares no `version`. The
  rule only ever asks about fields a policy named.
- **Per-rule configuration is a new capability of the schema.** `ruleConfig` grows a `fields` list
  alongside `enabled` and `severity`. Decoding stays strict, and `fields` on a rule that does not
  take one is a load failure, not a silent no-op — the realistic way to lose an enforcement is a
  typo.
- **Policy stays rule-agnostic.** It must not learn what `fields` means. It exposes the configured
  values for a rule ID as strings; `main` reads them when it builds its rule set, exactly as it
  already filters that set through `Enabled`. `policy` still imports neither `rules` nor `skill`.
- **Profiles overlay configuration the same way they overlay everything else.** A profile naming a
  rule replaces the base's entry whole — a profile that sets `fields` does not merge with the base's
  list. That is the existing overlay semantics and this story must not special-case around them.
- **Validate the field names at load, not at scan.** A policy naming a field the parser does not
  know is a misconfiguration and should fail the run with the policy path, alongside the existing
  rule ID validation. It must not become a finding on every skill.
- **Add `version` to `Skill` as a parsed passthrough only.** No format rules, no semver check — this
  story is about presence. Publisher and content digest from §2's table are new concepts rather than
  new fields and are deliberately not in scope.
- **One finding per missing field**, mirroring how `main` unpacks `ValidationErrors`, so a policy
  demanding three fields on a bare skill reports three addressable findings rather than one blob.
- **Severity comes from policy like any other rule.** Default `warning`; a registry that means it
  writes `severity: error` and the existing override machinery moves it across the exit-code gate.

## Proposed Changes

- add `Version` to `skill.Skill`
- add a `required-metadata` rule taking its field list from configuration, defaulting to none
- extend `ruleConfig` with `fields`, validated at load against the fields the rule understands
- expose the configured values from `policy` as strings and construct the rule in `main`

## Acceptance Criteria

- with no policy, the rule produces nothing and the `examples/` corpus results are unchanged
- a policy naming `[version, license, compatibility]` produces one finding per field absent from the
  skill, each carrying the `required-metadata` rule ID
- a skill declaring every named field produces no finding from this rule
- a policy naming an unknown field fails the load with the policy path, before anything is scanned
- a severity override and `enabled: false` both work on it, like any other rule
- a profile setting `fields` replaces the base's list rather than merging with it
- a skill missing required metadata still has its body scanned by every other rule

## Must Not Regress

- validation still short-circuits on `name` and `description` only
- policy still generates no findings of its own and imports neither `rules` nor `skill`
- strict policy decoding — an unknown key still fails the load
- the `examples/` regression corpus: `CLEANSKILL.md` clean, `MISMATCHSKILL.md` mismatch,
  `HIDDENBASHSKILL.md` shell
- `require` and rule ID validation behave as before

## Documentation

- document the rule, its configuration, and the field list in the policy section of `README.md`
- state plainly that metadata requirements are a rule, not validation, and why the body is still
  scanned
- add the new rule ID to any list of IDs in `README.md` and `CLAUDE.md`

## Dependencies

- depends on `P5-001` for the policy loader and `P5-003` for severity overrides
- `P7-001` makes its findings addressable in JSON, which is most of the point for a registry
