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
- Run: `go run . [flags] <path-to-skill-file>` — e.g. `go run . examples/HIDDENBASHSKILL.md`.
  Flags: `--json` (machine-readable output), `--model` (default `gemini-2.5-flash`),
  `--timeout` (default `1m`).
- Test all: `go test ./...`
- Test one package: `go test ./rules/...`
- Test one case: `go test ./skill/... -run TestParse/valid_skill`

`GEMINI_API_KEY` must be exported for the LLM leg to run — there is no `.env` loading in code, despite
`.env` being gitignored. **The test suite never calls the API**; every LLM path sits behind a swappable
seam. Keep it that way when adding tests.

## Architecture

Data flows one way, and every layer returns findings rather than printing:

`main.run` → `skill.Parse` → `Skill.Validate` → `scanner.Scan` → (`rules.Scan` + `analyse.GeminiAnalyzer`) → `result.Merge` → `main.renderResult` + `report.Write`

- **`result`** is the leaf package and the common currency. `Finding` carries `Source`
  (`validation` | `rule` | `analyzer`), `Category`, `Severity`, `Message`, `Evidence`; `Result` wraps
  `[]Finding` with `Clean()`. Nothing here knows about skills, rules, or the LLM — add new producers,
  not new coupling.
- **`skill`** parses and validates only. `Parse` normalises CRLF and accepts both a `\n---\n` closing
  fence and a trailing `\n---` at EOF. `Validate` returns `ValidationErrors` (a slice of field-level
  `ValidationError`), which `main` unpacks into one finding per missing field.
- **`rules`** holds the deterministic checks — `Default()` is empty-body, shell-execution,
  unrelated-URL, and description-mismatch. A rule is anything with
  `Check(*skill.Skill) []result.Finding`; `RuleFunc` adapts plain functions.
- **`analyse`** wraps Gemini behind the same contract. `GeminiAnalyzer.Analyze(*skill.Skill)` builds
  the payload and delegates to the package-level `AnalyzeWithConfig(prompt string, Config)` — two
  different `Analyze`s, don't confuse them. `Config` carries `Model` and `Timeout`; its zero value
  means `DefaultModel` / `DefaultTimeout`, and the timeout bounds the request context. The older
  `Analyze(prompt string)` is now just the default-config shorthand.
- **`scanner`** orchestrates. It owns the `Analyzer` interface
  (`Analyze(*skill.Skill) (result.Result, error)`) and `AnalyzerFunc`, so the LLM is optional and
  swappable.
- **`report`** renders a `result.Result` into a self-contained HTML page from the embedded
  `report/template.html`. It imports `result` only — it knows nothing about skills or rules. Every
  field goes through `html/template`, which is what keeps hostile skill text from becoming markup;
  don't swap in `text/template` or hand-built string concatenation.
- **`main.go`** is flag parsing, wiring, and rendering, kept testable: `main` only calls
  `run(args, stdout, stderr) int`. `parseOptions` returns `options` (`path`, `json`, `model`,
  `timeout`) or an error; it prints usage itself and the flag set is silenced with `io.Discard` so
  every failure is reported exactly once. `--json` prints `jsonReport` and nothing else — no clean
  message, no HTML report pointer — so keep that path free of stray stdout writes, and treat the
  JSON field names as a contract (add fields, don't rename them).

### Invariants worth knowing before changing things

- **Validation short-circuits.** If `Validate` fails, `run` renders those findings and returns
  *without* running rules or the LLM (`main.go:40`).
- **Findings do not yet affect the exit code.** `run` returns 1 only for usage/read/parse/scan
  *errors*; a flagged skill still exits 0. That is story `P4-003-exit-codes`, still `todo` — don't
  "fix" it incidentally.
- **The scanner degrades rather than fails.** If the analyzer errors but rules already found
  something, `Scan` returns the rule findings and swallows the error; it propagates the error only
  when rules were clean (`scanner/scanner.go:38`). So a missing `GEMINI_API_KEY` is fatal only for
  otherwise-clean skills.
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
- The rule heuristics are keyword/token based and were tuned against the fixtures in `examples/`.
  Changing tokenisation (`tokenSet`, `keywords`, `ignoredToken`, `weakMismatchOverlap`) will move
  fixture results — re-run `go test ./rules/...`.

### Prompt hardening — do not regress this

The analyzer audits hostile input, so untrusted text must never be able to read as instructions:

- The job description lives in `SystemInstruction`, never in the user turn.
- Skill content goes in the user turn as a JSON object inside `<skill_input>` tags, so quoting is
  escaped by `encoding/json` rather than by string concatenation.
- The system instruction explicitly tells the model to treat user content as data.
- `Temperature: 0` and `ResponseMIMEType: "application/json"`.

Keep all four properties if you touch prompt construction.

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
