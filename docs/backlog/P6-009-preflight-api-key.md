# P6-009 Preflight The API Key

## Story

As a user who has forgotten to export `GEMINI_API_KEY`, I want to be told once before the run starts
rather than once per file after it finishes.

## Scope

- check for the analyzer credential before scanning begins
- keep the scanner's degrade-rather-than-fail behaviour untouched

## The Problem

The key is read inside `AnalyzeWithConfig` (`analyse/analyse.go:110`), so it is checked once per
file. Scanning a directory of thirty clean skills with no key exported produces thirty identical
`missing GEMINI_API_KEY` failures on stderr, thirty wasted parse-and-rule passes, and an exit code
of `1` — with the real cause buried in repetition.

It interacts badly with two existing behaviours:

- the scanner only propagates an analyzer error when the rules found nothing
  (`scanner/scanner.go:38`), so a missing key is fatal for *clean* skills and invisible for flagged
  ones — the same run reports some files and not others for the same underlying cause
- under `P6-005` those N failures become N *concurrent* failures, which is noisier still

## Decisions To Implement

- **Check in `run`, before the scan loop.** That is where the environment already gets read and
  where a single clear message can be printed.
- **Missing key is not fatal.** The project's stated design goal is that obvious, high-confidence
  detections never depend on the model. Aborting the run would make the deterministic rules
  unreachable without a credential, which inverts that. Print one clear warning to stderr, run
  rules-only, and carry on.
- **Rules-only is an explicit state, not an accident.** When the key is absent, pass a nil analyzer
  into the scanner — `Scanner.Scan` already handles that by returning the rule findings alone. This
  is a behaviour change worth stating plainly: today a clean skill with no key produces a scan
  *failure*; afterwards it produces a clean rules-only *result*.
- **Say so in the output.** A rules-only run must not look like a full one. Note on the console that
  the analysis leg was skipped, and mark it in the JSON with a new additive field so an automated
  consumer can tell a rules-only result from a complete one. Do not rename or repurpose existing
  fields.
- **Do not add a flag in this story.** An explicit `--no-llm` for deliberate rules-only runs is a
  reasonable follow-up, but the present story is about the accidental case. Keep the surface small.
- **Leave the per-call check in place.** `AnalyzeWithConfig` should still refuse to run without a
  key. The preflight is an addition, not a replacement — the package must stay safe when called
  directly.

## Proposed Changes

- detect the missing credential once in `run`
- print one warning and fall through to a nil analyzer so the run is rules-only
- add an additive JSON field recording that the analysis leg was skipped
- note the skipped leg in the console output and the HTML report

## Acceptance Criteria

- a run with no key prints exactly one warning, whatever the file count
- that run still produces rule findings for every file and exits on findings, not on failure
- the console output and the report both show that the analysis leg was skipped
- the JSON carries the additive field, and every existing field keeps its name and meaning
- a run with a key present behaves exactly as it does today, with no extra output
- `analyse` still returns its missing-key error when called directly without a key

## Must Not Regress

- deterministic rules still run without any credential
- the scanner still degrades rather than fails when the analyzer errors for other reasons
- exit code contract — rules-only findings still gate on `--fail-on` as usual
- `--json` prints the JSON and nothing else; the warning goes to stderr
- the test suite still makes no live API calls

## Documentation

- document the rules-only fallback and the new JSON field in `README.md`, next to the existing
  `GEMINI_API_KEY` note
- update the environment-variable line in `CLAUDE.md`, which currently states the key must be
  exported for the LLM leg to run

## Dependencies

- none
- worth landing before `P6-005`, which would otherwise multiply the duplicate failures concurrently
