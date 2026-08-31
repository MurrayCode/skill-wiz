# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`skill-wiz` is a Go CLI that audits agent "skill" files — Markdown with YAML frontmatter, in the
Claude Code / opencode skill format — for mismatches between what a skill *says* it does and what its
instructions *actually* tell an agent to do, plus suspicious or hidden behaviour (e.g. a skill that
claims to link to racing news but tells the agent to run a local bash script).

It is a **hybrid scanner**: deterministic rules run first and always, and Gemini runs as an enrichment
layer on top. The deliberate design goal (see `docs/proposal.md`) is that obvious, high-confidence
detections never depend on the model.

## Commands

- Build: `go build ./...`
- Vet: `go vet ./...`
- Run: `go run . [flags] <path-to-skill-file-or-directory>...` — e.g.
  `go run . examples/HIDDENBASHSKILL.md` or `go run . examples`. Flags: `--json` (machine-readable
  output), `--model` (default `gemini-2.5-flash`), `--timeout` (default `1m`), `--fail-on`
  (default `error`).
- Test all: `go test ./...`
- Test one package: `go test ./rules/...`
- Test one case: `go test ./skill/... -run TestParse/valid_skill`

`GEMINI_API_KEY` must be exported for the LLM leg to run — there is no `.env` loading in code, despite
`.env` being gitignored. Without it `run` warns once and scans rules-only rather than failing per
file; see the preflight invariant below. **The test suite never calls the API**; every LLM path sits behind a swappable
seam. Keep it that way when adding tests.

## Architecture

Data flows one way, and every layer returns findings rather than printing:

`main.run` → `discover.Files` → `main.scanFiles` (bounded pool) → per file: `main.scanFile` (`skill.Parse` → `Skill.Validate` → `scanner.Scan` → (`rules.Scan` + `analyse.GeminiAnalyzer`) → `result.Merge`) → `render.Scans` + `report.Write`

- **`result`** is the leaf package and the common currency. `Finding` carries `Source`
  (`validation` | `rule` | `analyzer`), `Category`, `Severity`, `Message`, `Evidence`; `Result` wraps
  `[]Finding` with `Clean()`. It also owns the shared severity vocabulary — `Severities`, `Known`,
  `GateRank`, `DisplayRank` — plus `FormatSources` and `Pluralize`. Nothing here knows about skills,
  rules, or the LLM — add new producers, not new coupling.
- **`skill`** parses and validates only. `Parse` normalises CRLF and accepts both a `\n---\n` closing
  fence and a trailing `\n---` at EOF. `Validate` returns `ValidationErrors` (a slice of field-level
  `ValidationError`), which `main` unpacks into one finding per missing field.
- **`rules`** holds the deterministic checks — `Default()` is empty-body, shell-script,
  shell-command, unrelated-URL, and description-mismatch. A rule is anything with
  `ID() string` and `Check(*skill.Skill) []result.Finding`; `RuleFunc` pairs a plain function with
  its ID, and `IDs` lists a rule set's identifiers. The IDs are a **public contract** — policy files
  name rules by them, so add IDs but never rename one.
- **`analyse`** wraps Gemini behind the same contract. `GeminiAnalyzer.Analyze(*skill.Skill)` is the
  **only** exported way to the model: it builds the hardened payload and delegates to the unexported
  `analyzeWithConfig(prompt string, Config)`. `Config` carries `Model` and `Timeout`; its zero value
  means `DefaultModel` / `DefaultTimeout`, and the timeout bounds the request context. `HasAPIKey`
  is the preflight check, and is the only other exported function.
- **`scanner`** orchestrates. It owns the `Analyzer` interface
  (`Analyze(*skill.Skill) (result.Result, error)`) and `AnalyzerFunc`, so the LLM is optional and
  swappable.
- **`discover`** expands the CLI paths into files to scan. A named file is taken as given whatever
  its extension; a directory is walked for `.md` files, skipping hidden entries so `.git` never
  reaches the scanner. Explicit paths keep argument order, directory matches are sorted, duplicates
  collapse, and an empty expansion is `ErrNoSkillFiles`. It knows nothing about skill content.
- **`render`** owns the console output: `Scans` for a whole run, `Result` for one scan, plus the
  severity labels, colour constants, tally, and evidence truncation. Like `report` it imports
  `result` only — no `skill`, `rules`, `scanner`, or `analyse`, no `flag`, no filesystem, and never
  `os.Stdout`. `main` maps `fileScan` onto `render.Input` (path plus result) and decides colour,
  which arrives as a `render.Style` value. JSON stays in `main`: it is an output contract, not
  console presentation.
- **`report`** renders a run into one self-contained HTML page from the embedded
  `report/template.html`. `Render`/`Write` are variadic over `Input` (one per scanned skill): a single
  skill renders exactly as before, several render as stacked `.skill` sections plus a `<select>`
  picker. It imports `result` only — it knows nothing about skills or rules. Every field goes through
  `html/template`, which is what keeps hostile skill text from becoming markup; don't swap in
  `text/template` or hand-built string concatenation.
- **`main.go`** is flag parsing, wiring, and rendering, kept testable: `main` only calls
  `run(args, stdout, stderr, terminal) int`. `parseOptions` returns `options` (`paths`, `json`,
  `noColor`, `model`, `timeout`, `failOn`, `concurrency`) or an error; it prints usage itself and the flag set is silenced with `io.Discard` so
  every failure is reported exactly once. `run` scans each discovered file through `scanFile` into a
  `fileScan` (`path`, `skill`, `result`), writes the one report, then renders them together through
  `render`. `--json` prints
  the JSON and nothing else — no clean message, no HTML report pointer — so keep that path free of
  stray stdout writes, and treat the JSON field names as a contract (add fields, don't rename them).

### Invariants worth knowing before changing things

- **Validation short-circuits.** If `Validate` fails, `scanFile` returns those findings *without*
  running rules or the LLM.
- **Files are scanned concurrently, but consumed in file order.** `scanFiles` runs a bounded pool
  (`--concurrency`, default `defaultConcurrency`) whose workers write into their own index of a
  pre-sized `[]scanOutcome` and never append. `run` then walks that slice in order, so results *and*
  the stderr failure messages are deterministic regardless of completion order — never write to
  stderr from a worker. There is deliberately no shared cancellation: a fail-fast pool would break
  "one bad file never hides the rest". `--timeout` still bounds a single request, not the run.
- **One bad file never hides the rest.** A read, parse, or analysis failure is reported on stderr and
  the run continues with the remaining files; only a run where *every* file failed prints nothing.
- **Single-file output is unchanged by multi-file support.** One file still renders with no path
  header and emits a single JSON *object*. More than one file heads each result with `=== <path> ===`
  and emits a JSON *array* of the same object.
- **One run, one report.** However many files a run covers, `writeReport` writes a single
  `skill-wiz-report.html` and the console prints one pointer to it; every JSON entry carries that same
  `report_path`. The page holds every scanned skill and the reader picks between them.
- **The report picker is progressive enhancement.** Every `.skill` section renders visible and the
  `<select>` is `hidden`; the inline script unhides the picker and hides the other sections. Without
  JavaScript a reader still sees every skill rather than one panel and a dead control — keep that
  order if you touch the script.
- **One severity vocabulary.** `result` owns the ordering (`result.Severities`) and both rankings
  derived from it: `result.GateRank` for the exit-code threshold (unknown ranks below every known
  severity, so a malformed finding fails nothing on its own at any threshold) and `result.DisplayRank` for console and report order
  (unknown sorts last, so it never outranks a real finding). The two disagree about the unknown case
  deliberately — do not collapse them. `result.FormatSources` and `result.Pluralize` live there for
  the same reason: `main` and `report` both need them and `result` is the leaf package. No package
  should grow a severity table of its own.
- **Presentation lives at render time only.** `render.Result` sorts a *copy* of the findings by
  `result.DisplayRank` (error → warning → info, unknown last) and truncates evidence past
  `maxEvidenceRunes`; `result.Result` keeps its merge order and the JSON and HTML paths keep the
  full text. Sorting is stable, so rule findings stay ahead of analyzer findings within a severity.
- **Colour is a value, not a lookup.** `main` decides with `isTerminal(os.Stdout)` and passes it
  into `run`, which folds in `--no-color` and `NO_COLOR` via `colorEnabled` and hands the renderer a
  `render.Style`. Never sniff `os.Stdout` inside the `render` package — the tests write to buffers, and
  colour must stay absent there. Only the severity label is coloured, so the rest of a line stays
  greppable.
- **The tally is multi-file only.** `render.Scans` closes a run over more than one file with one
  `N files scanned · … · N findings (…)` line counting the files it actually scanned. A single-file
  run prints nothing extra, so its output is unchanged.
- **Exit codes are a three-way contract.** `0` clean, `1` operational failure (usage, discovery,
  read, parse, scan), `2` at least one finding at or above the active threshold. `exitCode` maps a
  finished run onto that, and an operational failure outranks findings — a run with both exits `1`.
  `--fail-on` (`error` by default, also `warning` or `info`) sets the threshold via `parseSeverity`
  and `result.GateRank`; an unrecognised severity ranks at `result.UnknownGateRank`, *strictly below*
  every known severity, so it can never fail a build on its own — level with `info` would not be low
  enough, because the comparison is `>=` and `info` is a selectable threshold.
  Both the text and JSON branches return the same code. Don't renumber `1`.
- **The scanner degrades rather than fails.** If the analyzer errors but rules already found
  something, `Scan` returns the rule findings and swallows the error; it propagates the error only
  when rules were clean (`scanner/scanner.go:38`). So an analyzer error is fatal only for
  otherwise-clean skills.
- **The API key is preflighted once per run, not once per file.** `run` checks `analyse.HasAPIKey`
  before the scan loop; with no key it prints one warning, passes a **nil** analyzer, and the run is
  rules-only — a console note, an `analysis` line on the report, and an additive
  `"analysis_skipped": true` in the JSON say so. The per-call check inside `analyse` stays, so the
  package is still safe when called directly. `main_test.go` has a `TestMain` that unsets the key, so
  a test wanting the analysis leg sets a placeholder itself and the suite never depends on the
  developer's environment.
- **The HTML report never fails the scan.** `writeReport` warns on stderr and returns; the console
  output already carries every finding. It runs on the validation path too, so a skill missing
  required metadata still gets a report.
- **`Merge` dedupes on content, not source.** `findingKey` hashes category + severity + normalised
  message + evidence, deliberately excluding `Source`, so a rule and the LLM reporting the same issue
  collapse into one finding — and since rules merge first, the rule's provenance wins.
- **The LLM layer fails closed.** Empty output, non-JSON, `clean: true` alongside findings, or a
  finding missing any field all become a `warning` "Analyzer returned unusable response" finding
  rather than a clean result (`analyse/analyse.go:120`). Preserve this: a broken model response must
  never read as "clean".
- **The shell prefilter must fold exactly as the regexp does.** `mentionsShellToken` gates
  `shellCommandPattern` per line and is only sound because it accepts everything `(?i)` would.
  That is *Unicode* simple folding, not ASCII: `s` folds to `{s, S, ſ}`, so `baſh script.txt` is a
  real match. `TestFoldSets` pins both fold sets against `unicode.SimpleFold`.
- **`intentTokens` strips URLs from the body only.** The rule asks whether a body link shares
  vocabulary with the skill's stated intent, and a URL the name or description declares *is* part of
  that intent — only the body's own links are removed, so a link cannot vouch for itself. Stripping
  the joined string instead would newly flag body URLs that a metadata URL used to vouch for.
- **`shell-command` stands down on a body `shell-script` already claimed.** The line naming a local
  script almost always mentions `bash` or `sh` too, so `shellCommandRule` returns nothing when
  `localShellScriptPattern` matches anywhere in the body — one problem, one finding. It defers on the
  *pattern*, not on whether the other rule ran, so disabling `shell-script` through policy removes
  that finding without a warning appearing in its place.
- The rule heuristics are keyword/token based and were tuned against the fixtures in `examples/`.
  Changing tokenisation (`tokenSet`, `keywords`, `ignoredToken`, `weakMismatchOverlap`) will move
  fixture results — re-run `go test ./rules/...`. Tokenisation must also stay **rune-safe**: iterate
  runes, never byte indices, or multi-byte characters tear apart and invalid UTF-8 poisons the token
  set (`splitAlphaNumeric` is the one that got this wrong).

### Prompt hardening — do not regress this

The analyzer audits hostile input, so untrusted text must never be able to read as instructions:

- The job description lives in `SystemInstruction`, never in the user turn.
- Skill content goes in the user turn as a JSON object inside `<skill_input>` tags, so quoting is
  escaped by `encoding/json` rather than by string concatenation.
- The system instruction explicitly tells the model to treat user content as data.
- `Temperature: 0` and `ResponseMIMEType: "application/json"`.
- **A skill goes in, a result comes out.** There is no exported entry point taking a caller-supplied
  prompt string, so the payload construction cannot be bypassed. Do not add one back — that is what
  makes the hardening structural rather than a convention.

Keep all five properties if you touch prompt construction.

### Test seams

Two package-level `var`s exist purely so tests can substitute the model; tests save and restore them:

- `analyse.newGenerator` — returns the `contentGenerator` interface instead of a real `*genai.Client`.
- `main.newSkillAnalyzer` — a `func(analyse.Config) scanner.Analyzer` factory, so tests can both swap
  the analyzer and assert which `--model` / `--timeout` reached it.
- `main.reportPath` — where the HTML report is written; tests point it at `t.TempDir()` so `run`
  never writes `skill-wiz-report.html` into the repo.

## Working conventions

This repo is backlog-driven, and `.opencode/skill/` holds project skills that define how work is done:

- `docs/proposal.md` is the phased roadmap; `docs/backlog/P*-*.md` are the individual stories.
- **Update `docs/backlog/tracker.md` whenever a story changes status.** The `backlog-tracker` skill
  defines the format: status vocabulary (`todo`/`in_progress`/`done`/`blocked`), `YYYY-MM-DD`
  completion date, short outcome note. Mark `done` only after implementation *and* verification.
- `go-test-driven-development` and `go-table-driven-testing` describe the expected approach: write the
  failing test first, and structure tests as a `tests := []struct{...}` table driven with
  `t.Run(tt.name, ...)`. Every package's tests already follow this — match it.

## Examples as a regression corpus

`examples/` is not just sample data; `rules/rules_test.go` and `main_test.go` assert against it:

- `CLEANSKILL.md` — must stay clean (no findings).
- `MISMATCHSKILL.md` — must produce `mismatch` findings.
- `HIDDENBASHSKILL.md` — must produce `shell` findings (its body tells the agent to execute
  `./scripts/racing.sh`).

Tuning any heuristic means re-checking all three.

Note: the Go module path is `github.com/murraycode/skill-wiz` while the repo directory is `skill-wizard`.
