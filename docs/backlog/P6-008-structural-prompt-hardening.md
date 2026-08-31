# P6-008 Make Prompt Hardening Structural

## Story

As a maintainer, I want the only way to reach the model to be through the hardened skill payload so
that prompt safety is enforced by the API rather than by convention.

## Scope

- close the raw-prompt entry points into the `analyse` package
- keep the config seam and the test seam intact

## The Problem

`analyse` exports two functions that take a caller-supplied prompt string:

```go
func Analyze(prompt string) (result.Result, error)
func AnalyzeWithConfig(prompt string, config Config) (result.Result, error)
```

Both send that string as the user turn. The hardening `P3-002` delivered — the JSON payload wrapped
in `<skill_input>` tags, escaped by `encoding/json` rather than string concatenation — lives in
`promptForSkill`, which only `GeminiAnalyzer.Analyze` calls. Any caller reaching for the exported
prompt-string functions bypasses it entirely and can put arbitrary text where untrusted skill
content is supposed to sit, already labelled as data.

Nothing in production does this today — `GeminiAnalyzer.Analyze` is the only real caller — so this
is about removing the option, not fixing a live bug. `CLAUDE.md` also notes the two different
`Analyze` functions as a known confusion; this removes that too.

## Decisions To Implement

- **`GeminiAnalyzer.Analyze` becomes the only exported way in.** Unexport the prompt-string path.
  A skill goes in, a result comes out, and the payload construction is not the caller's business.
- **Keep `Config` and its defaults exported.** `main` builds an `analyse.Config` from `--model` and
  `--timeout` and passes it through the `newSkillAnalyzer` seam. That plumbing stays exactly as it
  is.
- **Delete `Analyze(prompt string)` rather than unexporting it.** It is a shorthand for
  `AnalyzeWithConfig(prompt, Config{})` with no remaining caller, and `Config`'s zero value already
  means "defaults". Keeping an unexported alias would preserve the naming confusion for no benefit.
- **Update the tests to go through the skill path.** The existing coverage in `analyse_test.go` —
  missing key, client creation failure, upstream failure, timeout, malformed output — is valuable
  and must all survive. Rewrite it to construct a `*skill.Skill` and call
  `GeminiAnalyzer.Analyze`, not to call the internal function directly, so the tests exercise the
  same path production does.
- **`newGenerator` stays as it is.** It is the documented test seam for substituting the model and
  is unaffected.

## Proposed Changes

- delete `Analyze(prompt string)` and unexport `AnalyzeWithConfig`
- rewrite the affected tests to drive `GeminiAnalyzer.Analyze` with a skill
- confirm no caller outside the package is left

## Acceptance Criteria

- `analyse` exports no function that accepts a caller-supplied prompt string
- `GeminiAnalyzer.Analyze` is the only exported entry point to the model
- every existing failure-mode test still passes, driven through the skill path
- the four hardening properties still hold: job description in `SystemInstruction`, skill content as
  JSON inside `<skill_input>` in the user turn, the explicit treat-as-data instruction,
  `Temperature: 0` with `ResponseMIMEType: "application/json"`
- `go build ./...` and `go test ./...` pass

## Must Not Regress

- the analyzer fails closed — empty output, non-JSON, `clean: true` alongside findings, or a finding
  missing any field all still become a `warning` "Analyzer returned unusable response" finding
- `Config`'s zero value still means `DefaultModel` and `DefaultTimeout`
- the timeout still bounds the request context
- `--model` and `--timeout` still reach the analyzer, as `main_test.go` asserts
- the test suite still makes no live API calls

## Documentation

- update the `analyse` description in `CLAUDE.md` — the "two different `Analyze`s, don't confuse
  them" note becomes obsolete
- add the single-entry-point property to the prompt hardening section

## Dependencies

- depends on `P3-002` and `P3-003`
