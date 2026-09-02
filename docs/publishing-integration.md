# Publishing Integration — Assessment

**Story:** `P5-005-publishing-integrations.md` · **Status:** spike, no production code
· **Written:** 2026-09-02

This document answers whether `skill-wiz` could gate a skill registry or a publishing workflow
today, what it would be missing, and whether any of it is worth building yet. It is an assessment,
not a design: nothing here has been implemented, and anything it recommends becomes its own story.

Every claim about the tool as it stands names the code or contract it rests on.

---

## 1. What a gate actually needs

A gate is a small thing. It runs the scanner over some skills, decides pass or fail, and leaves
enough of a trail that a human can see why. Four surfaces exist today; here is what each one covers.

### Sufficient on its own

**Exit codes (`P4-003`) are a complete pass/fail contract.** `exitCode` in `main.go` maps a finished
run onto `0` clean, `1` operational failure, `2` findings at or above the threshold, and an
operational failure outranks findings, so a gate that treats anything non-zero as "do not publish"
is correct without parsing anything. `--fail-on` moves the threshold through `parseSeverity` and
`result.GateRank`, which means the *policy* decision of how strict to be is already expressible on
the command line. A shell one-liner is a working gate:

```bash
skill-wiz --fail-on warning ./skills || exit 1
```

**Policy (`P5-001`–`P5-003`) already covers per-organisation rules.** `.skill-wiz.yaml` names rules
by stable IDs, `require` fails the load when a rule a policy insists on is missing from the active
set, profiles let one file serve both a developer's machine and CI, and severity overrides move
findings across the exit-code threshold deliberately. A registry that wants "shell execution is
never acceptable here" writes it in a file it commits; it does not need an API.

**JSON output (`P4-001`) is enough to render a review.** Each entry carries `path`, `skill`, `clean`,
`findings[]` with `source`/`category`/`severity`/`message`/`evidence`, `report_path`, and the
additive `analysis_skipped` and `overridden_from` flags. That is everything a bot comment or a
registry review page needs to show a human what was flagged and how confident the tool is about it.

**The run summary (`P5-004`) is enough to report over a whole registry.** `--summary` emits
`{"summary": {...}, "results": [...]}` with counts by severity, category, and source, ordered
deterministically, and files that failed to scan counted separately from files that were clean.
A weekly "state of the registry" job needs nothing more.

### Genuinely missing

These are gaps, not complaints — each one is a thing a real consumer would hit on day one.

1. **Findings are not addressable in the output.** `result.Finding` carries `RuleID`, and
   `rules.Scan` stamps it, but `jsonFinding` in `main.go` does not serialise it. A gate can see that
   a `shell` finding fired; it cannot say "this one is `shell-command`, and we allow that here"
   without matching on the human-readable `message`, which is not a contract. Policy can disable the
   rule wholesale, but a gate that wants to accept one known finding on one skill has nothing to key
   on. This is the single cheapest gap to close.

2. **There is no per-finding location.** `Evidence.Summary` is a snippet, not a position — there is
   no line or byte offset anywhere in `result.Finding`. Any integration that wants inline
   annotations (a GitHub Action posting review comments on the offending line) cannot be built
   without adding positions to every rule that produces evidence.

3. **The HTML report path is fixed.** `reportFileName` is a constant and `defaultReportPath` joins
   it to `os.Getwd()`. There is no flag to redirect it, so a CI job that wants the report in an
   artifacts directory has to `cd` first or move the file afterwards. `reportPath` is already a
   package-level var for tests, so the flag is a few lines.

4. **A gate cannot insist on the analysis leg.** A missing `GEMINI_API_KEY` degrades to a rules-only
   run by design (`run` preflights `analyse.HasAPIKey`), and says so on stderr, on the report, and
   with `"analysis_skipped": true`. A consumer can check that field, but there is no
   `--require-analysis` that turns a silently degraded run into an operational failure. For a
   publishing gate — where a rules-only pass could be mistaken for a full audit — that check is more
   likely to be wanted than not.

5. **The JSON has no schema version.** The field names are treated as a contract and have only ever
   grown, but there is nothing in the payload a consumer can branch on. Not urgent while the
   contract is additive; worth a field before a third party depends on it.

6. **Exit code `1` conflates two different failures.** Usage errors, discovery errors, and a single
   unparseable skill all return `exitFailure`. A gate that wants to distinguish "your configuration
   is broken" from "one skill in this batch is malformed" has to parse stderr. This is a documented
   deliberate simplification, not a bug, but it *is* a limit on what a gate can report.

---

## 2. The minimum metadata a registry would require

`skill.Skill` parses `name`, `description`, `license`, `compatibility`, and a `metadata` block of
`audience` and `workflow`. `Skill.Validate` requires exactly two of them: `name` and `description`.
Everything else is parsed and passed through unchecked.

A registry publishing skills to other people would, at minimum, also want:

| Field | Why a registry needs it | Today |
| --- | --- | --- |
| `version` | Nothing can be updated, pinned, or rolled back without one | not parsed at all |
| `license` | Redistribution needs stated terms | parsed, never validated |
| `compatibility` | Which agent runtimes the skill claims to work with | parsed, never validated |
| publisher / author | Attribution and abuse response | not parsed at all |
| content digest | Detecting a skill edited after review | not computed |

The first three are close to free; the last two are new concepts rather than new fields.

### Where that validation belongs

**Not in `skill.Validate`.** That function answers one question — can this file be scanned at all? —
and the scan short-circuits on it: `scanFile` returns validation findings *without* running the rules
or the model. Adding registry requirements there would mean a skill missing a `version` never gets
its body scanned, which is exactly backwards for a registry: you most want the shell-execution check
on the skill whose metadata is sloppy.

**Not in `policy` as it stands either.** Policy today enables, disables, and re-grades rules, and
generates no findings of its own (`P5-001` states this explicitly, and `result.Source` gained no new
value). Teaching it to emit findings would make it a second rule engine.

**The right shape is a deterministic rule configured by policy.** A `required-metadata` rule with a
stable ID, whose required field list comes from the policy file:

```yaml
rules:
  required-metadata:
    severity: error
    fields: [version, license, compatibility]
```

That keeps each layer doing what it already does: `skill` decides whether a file is parseable,
`rules` decides whether it is acceptable, and `policy` decides what "acceptable" means here. It also
means different registries can demand different fields without a fork. The one new thing it needs is
per-rule configuration beyond `enabled`/`severity`, which the schema was designed to allow.

---

## 3. Where the boundary sits

| Candidate | Buildable on today's CLI? | What it would force |
| --- | --- | --- |
| **Pre-commit hook** | **Yes, entirely** | Nothing. `skill-wiz --fail-on error <staged files>` and the exit code. |
| **CI template** (workflow YAML calling the binary) | **Yes, essentially** | Nothing required; a `--report <path>` flag would remove an awkward `cd` or `mv`. |
| **GitHub Action, pass/fail + summary comment** | **Yes** | Nothing required. `--json --summary` gives it everything it renders. Adding `rule` to each finding would make its output far more useful. |
| **GitHub Action with inline annotations** | **No** | Per-finding line positions in `result.Finding`, and every rule updated to produce them. That is a change to the rule contract, not an integration. |
| **Registry API gating uploads** | **Not cleanly** | Either shelling out to the binary per upload — workable but ugly, since the tool writes an HTML report to the working directory on every run — or a library entry point. Every package is exported, but there is no single `Scan(content string) (result.Result, error)`: the read → parse → validate → scan → apply-policy pipeline lives in `main.scanFile`, which is unexported and takes a path. A server consumer would have to re-implement it and would be pinned to internals with no stability promise. |

### The cheapest thing that would be genuinely useful

**A documented CI recipe plus a pre-commit hook — both of which are pure documentation.** The exit
code contract, `--fail-on`, and multi-path scanning already do the work; what is missing is a page in
the README telling someone the four lines to paste. Zero production code, and it covers the two
consumers a small tool actually gets.

If something must be *built*, the honest ranking by value per line changed is:

1. `rule` in the JSON finding (the field already exists on the struct; it is one line in
   `jsonFinding` and one in `newJSONReport`)
2. `--report <path>` (a flag, threaded into the existing `reportPath` var)
3. `--require-analysis` (turn `analysisSkipped` into an operational failure on request)

None of the three is an integration surface. All three make every future integration easier.

---

## 4. Recommendation

**Do not build a publishing integration now.**

The reasons, plainly:

- **There is no named consumer.** The story itself says not to start until there is one, and there
  still is not. Building an integration surface for an imagined consumer is how a small tool grows
  an API nobody uses, and the API is the part you cannot take back.
- **The two integrations anyone is actually likely to want are already possible.** A pre-commit hook
  and a CI gate need the exit code and nothing else. What is missing for them is documentation, not
  code.
- **The integrations that are *not* possible are blocked on the rule contract, not on an
  integration layer.** Inline annotations need positions on findings; a server needs a stable
  library entry point. Neither is made cheaper by building a GitHub Action first — both would have to
  be built again underneath it.
- **The cost of waiting is nearly zero, and the cost of guessing is not.** A published JSON shape or
  exported entry point is a promise to strangers. Every month it stays unpublished is a month the
  scanner can keep changing shape freely.

**Park this story.** Revisit when a real consumer exists — a registry that wants to gate uploads, or
a team asking for CI annotations. At that point the consumer's actual requirements will settle
questions this document can only guess at.

### If it is picked up, these are the pieces

Proposed as separate stories, smallest first. Only the last two are integration work; the first three
are gaps worth closing regardless of whether anything is ever integrated.

| Proposed story | Scope | Depends on |
| --- | --- | --- |
| `expose rule ids in json output` | Serialise `Finding.RuleID` as an additive `rule` field, so a consumer can address a finding by the same identifier a policy uses. | nothing |
| `configurable report destination` | A `--report <path>` flag over the existing `reportPath` seam, so a CI job can place the page in its artifacts directory. | nothing |
| `require the analysis leg` | A `--require-analysis` flag turning a rules-only run into an operational failure, so a gate cannot mistake a degraded run for a full audit. | nothing |
| `required metadata rule` | A `required-metadata` rule with a policy-configured field list, per section 2 — including the per-rule configuration the policy schema would need. | `P5-001` |
| `finding source positions` | Line positions on findings, produced by every rule that carries evidence. Prerequisite for any annotating integration. | nothing, but it touches every rule |
| `ci recipe documentation` | README recipes for a pre-commit hook and a CI gate. Documentation only. | nothing |
