# P5-003 Severity Overrides

## Story

As a team, I want severity overrides so that scanner findings match local risk tolerance and
workflow expectations.

## Scope

- raise or lower the severity of a rule's findings through policy
- suppress a rule's findings without disabling the rule
- make an adjusted severity visible rather than silent

## Decisions To Implement

- **Keyed by rule ID.** Same identifiers as `P5-001`, not categories. One category can come from
  several rules, and a policy that moves them all together is rarely what anyone means.
- **Vocabulary.** `severity: error | warning | info | off`. `off` keeps the rule running but drops
  its findings from output; `rules.<id>.enabled: false` from `P5-001` stops the rule running at all.
  The two are different, and the difference matters once `P5-004` starts counting.
- **Applied after `Merge`, before rendering.** `result.Merge` dedupes on a key that hashes severity
  (`findingKey`), so overriding first would change which findings collapse together — a policy that
  lowers a rule's severity must not silently start or stop deduping it against the analyzer's
  version of the same finding.
- **Analyzer findings are not overridable in this story.** They carry no rule ID. Leave them alone
  rather than inventing a second addressing scheme; revisit only if it is actually wanted.
- **Provenance.** `result.Finding` gains one optional field recording the original severity when
  policy changed it, e.g. `OverriddenFrom Severity`. This is additive to the JSON contract: a new
  `overridden_from` key, present only when set. Do not rename or repurpose `severity` — downstream
  consumers read it, and they should keep reading the effective value.
- **Exit codes see the effective severity.** An override to `info` takes a finding below the default
  `P4-003` gate; an override to `error` puts it above. That is the point of the feature — state it
  in the docs so nobody is surprised by a green build.

## Proposed Changes

- extend the policy schema with per-rule severity
- apply overrides in one place, after merge, tagging each adjusted finding with its original severity
- surface the original severity in the console line, the JSON, and the HTML report

## Acceptance Criteria

- a rule's severity can be raised and lowered through policy, asserted in both directions
- `off` removes the finding from console, JSON, and the report while the rule still runs
- findings with no override keep their default severity, proven against the `examples/` corpus
- an overridden finding carries its original severity in all three outputs
- overriding severity does not change which findings `Merge` collapses, asserted by a test with a
  rule and an analyzer finding that currently dedupe

## Must Not Regress

- `Merge`'s content-based dedupe and rule-first provenance
- the JSON contract stays additive — existing fields keep their names and meanings
- default severities when no policy is present

## Documentation

- document the severity vocabulary, the `off`-versus-`enabled: false` distinction, and the exit code
  consequence in `README.md`

## Dependencies

- depends on `P5-001`
- interacts with `P4-003`: overrides move findings across the exit code threshold
