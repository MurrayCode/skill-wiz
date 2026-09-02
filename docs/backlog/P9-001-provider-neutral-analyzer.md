# P9-001 Provider-Neutral Analyzer Seam

## Story

As a maintainer about to add three more model providers, I want the analysis leg split into the part
that is the same for every provider and the part that is not, so that adding a provider is one file
rather than one fork of the package.

## Scope

- a provider abstraction inside `analyse`, with Gemini as its only implementation
- the CLI surface for choosing a provider and its credential
- a shared conformance test suite every future provider must pass
- no new provider, and no behaviour change to the Gemini path

## The Problem

`analyse` is Gemini-shaped in five specific places, and only five:

1. `contentGenerator` (`analyse/analyse.go:78`) is typed on `genai.Content` and
   `genai.GenerateContentConfig`, so the test seam itself is provider-specific.
2. `analyzeWithConfig` builds a `genai.GenerateContentConfig` — `SystemInstruction`, `Temperature`,
   `ResponseMIMEType` — and every provider expresses those three differently.
3. `apiKeyEnvVar` is the single constant `"GEMINI_API_KEY"`, and `HasAPIKey` reads only it.
4. `DefaultModel` is a Gemini model, and `--model`'s help text says "Gemini model".
5. `GeminiAnalyzer` is the exported type `main` constructs.

Everything else is already provider-neutral and must stay shared: `promptForSkill` builds the
hardened `<skill_input>` payload, `resultFromText` and `validateAnalyzerFinding` map a response onto
findings and fail closed on anything unusable, and `Config` carries model and timeout. Those are the
parts that took `P3-002`, `P3-003`, and `P6-008` to get right — three copies of them is the failure
mode this story exists to prevent.

## Decisions To Implement

- **One package, one file per provider. Not sub-packages.** `analyse/openai.go` and friends, not
  `analyse/openai/`. A sub-package cannot reach `promptForSkill` or `resultFromText` without those
  becoming exported, and an exported prompt builder is exactly what `P6-008` deleted — it would let a
  caller put arbitrary text where untrusted content is supposed to sit. The package boundary *is*
  the hardening.
- **The provider interface is narrow and text-in, text-out.** Something close to
  `complete(ctx, system, user string, config Config) (string, error)`. A provider translates that
  into its own wire format and returns the model's raw text. It does not see a `*skill.Skill`, does
  not build prompts, and does not map findings — `GeminiAnalyzer.Analyze` stays the one entry point,
  and every provider's output goes through the same `resultFromText`.
- **The fail-closed contract belongs to the shared parser, not to providers.** Empty output,
  non-JSON, `clean: true` alongside findings, a finding missing a field — all of it stays in
  `resultFromText`. A provider that hand-rolled its own mapping could quietly return a clean result
  for a broken response, which `P3-002` explicitly rules out.
- **All five prompt-hardening properties are per-provider obligations.** Each provider must put the job
  description in its own system slot, carry the skill content only as the JSON payload in the user
  turn, explicitly instruct the model to treat user content as data, set temperature `0`, and request
  JSON output through whatever mechanism it has. Where a provider has no JSON mode, the shared parser
  is the backstop but not the excuse — the provider story says how it gets structured output.
- **`--provider`, defaulting to `gemini`.** Explicit selection only. Do **not** auto-detect from
  whichever key happens to be exported: a scanner whose verdict depends on ambient environment is
  hard to trust, which is the same reasoning `policy.Discover` records for not searching upwards.
  An unknown provider name is a usage error listing the ones that exist, exactly as `--profile` does.
- **One credential per provider, each named by that provider.** `GEMINI_API_KEY` keeps working
  unchanged. `HasAPIKey` becomes provider-aware, and the missing-key message from `P6-009` names the
  variable the *selected* provider needs — telling someone to set `GEMINI_API_KEY` when they asked
  for OpenAI is worse than saying nothing.
- **The default model belongs to the provider, not to the package.** `DefaultModel` becomes
  per-provider, resolved after `--provider` is known. `--model` with no `--provider` must keep
  meaning what it means today.
- **Say which model judged.** An audit tool that will not tell you what audited is a gap, and it
  becomes one the moment there is a choice. Record the provider and model on the JSON, the console
  note, and the report, additively and alongside the existing `analysis_skipped` — do not rename or
  repurpose that field.
- **Delete `GeminiAnalyzer` rather than aliasing it.** Nothing outside this module consumes the
  package and `P8-001` has not released anything, so there is no compatibility to keep. Replace it
  with one constructor that takes the resolved provider.
- **Ship the conformance suite in this story.** A table-driven test set, run against every provider
  implementation, asserting: the system instruction never appears in the user turn; a hostile skill
  body that tries to close the `<skill_input>` wrapper is escaped; temperature is zero; JSON output
  is requested; a missing key errors without a request being made; the timeout bounds the request;
  and each of the five unusable-response shapes maps to the "Analyzer returned unusable response"
  finding. Landing this before any second provider is the point — a provider added first and tested
  afterwards is a provider tested to whatever it happens to do.
- **A provider is `net/http` until it earns an SDK, and that is a phase-wide stance.** Each provider
  makes one non-streaming request with a system prompt, a user turn, a temperature, and a JSON
  response mode. `P6-001` deliberately dropped a dependency and eleven transitive ones; three SDKs
  would undo that for a single POST each. Gemini keeps the SDK it already has — this is not a
  licence to rewrite a working path. Every provider story records its measured `go.mod` and `go.sum`
  delta, and adopting an official SDK stays available as a *stated* decision rather than a silent
  default. If this stance is wrong it is wrong once, here, rather than three times over.
- **No live calls, ever.** The suite substitutes each provider's transport, as `newGenerator` does
  today. Adding a provider must not add a test that needs a key.

## Proposed Changes

- extract a provider interface and move the Gemini request construction behind it
- make the credential, the default model, and the preflight provider-aware
- add `--provider` with `gemini` as the default and an error listing valid values
- add additive provider/model provenance to the JSON, console, and report
- add the shared conformance suite and run it against Gemini

## Acceptance Criteria

- a Gemini run is byte-identical to today on console, JSON, and HTML, with the provenance fields
  added — proven against the `examples/` corpus
- `--provider gemini` and no `--provider` behave the same
- an unknown `--provider` value is a usage error naming the valid values, and scans nothing
- with the selected provider's key absent, the run degrades to rules-only and the one warning names
  that provider's variable
- the conformance suite passes against the Gemini provider and fails if any hardening property is
  removed, asserted by a test that removes one
- no test makes a live API call

## Must Not Regress

- the five prompt-hardening properties in `CLAUDE.md`, all of which are now per-provider obligations
- `GeminiAnalyzer.Analyze`'s replacement is still the only exported way to the model; no exported
  entry point takes a caller-supplied prompt string
- the LLM layer still fails closed on every unusable response shape
- `P6-009`'s single preflight warning and rules-only fallback
- the scanner still degrades rather than fails when the analyzer errors
- `--model` and `--timeout` keep their meanings

## Documentation

- document `--provider`, the per-provider credential, and the provenance fields in `README.md`
- rewrite the `analyse` section of `CLAUDE.md` around the provider seam, and record the
  one-package-no-sub-packages decision with its reason

## Dependencies

- none, but it blocks `P9-002`, `P9-003`, and `P9-004`, none of which should start before it lands
