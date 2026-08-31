# P5-002 Environment Specific Rules

## Story

As a team, I want environment-specific rules so that different deployment contexts can allow or
forbid different skill behaviours.

## Relationship To `P5-001`

This story adds one thing to the policy file: named profiles. Everything else — the file name,
discovery, rule IDs, validation, load failures — is inherited from `P5-001` and must not be
redesigned here. If profiles turn out to be the only variation anyone wants, fold this into
`P5-001` rather than shipping a second configuration concept.

## Scope

- named profiles inside one policy file
- explicit profile selection, defaulting to a base configuration
- profiles that differ in which rules are enabled, e.g. shell execution allowed locally and
  forbidden in CI

## Decisions To Implement

- **Shape.** A top-level `profiles:` map whose entries hold the same keys as the base policy. The
  base applies first, the selected profile overlays it key by key. No inheritance between profiles,
  no merge of lists — the profile's value replaces the base's.
- **Selection.** `--profile <name>` only. No auto-detection from `CI`, `ENV`, or any other
  environment variable: a scanner that quietly changes verdict depending on where it runs is worse
  than one that makes you say which rules you want.
- **Unknown profile is a failure.** `--profile ci` with no `ci` entry stops the run and names the
  profiles that do exist. Silently falling back to the base would hide a broken CI configuration.
- **No profile named is the base policy**, which is `P5-001`'s behaviour unchanged.

## Proposed Changes

- extend the policy schema and loader with `profiles`
- resolve base plus selected profile into the single effective policy the rest of the run already
  consumes — everything downstream of resolution stays profile-unaware
- add `--profile` to `parseOptions` alongside `--policy`

## Acceptance Criteria

- one policy file defines two profiles where a rule is enabled in one and disabled in the other, and
  scanning the same fixture under each produces different findings
- selecting no profile reproduces the base policy's findings exactly
- an unknown profile name fails the run with a message listing the available profiles
- a profile key overlays the base rather than merging with it, asserted directly
- tests cover at least two distinct profile configurations

## Must Not Regress

- a run with no policy file and no `--profile` still behaves exactly as a default run does
- policy resolution stays in the `policy` package; `rules` and `scanner` learn nothing about profiles

## Documentation

- document `profiles` and `--profile` in the policy section of `README.md`, including a
  local-versus-CI example

## Dependencies

- depends on `P5-001`
