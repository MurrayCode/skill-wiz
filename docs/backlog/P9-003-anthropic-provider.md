# P9-003 Anthropic Provider

## Story

As a user with an Anthropic account, I want `--provider anthropic`, so that I can run the analysis
leg on Claude.

## Scope

- one Anthropic provider implementation behind the `P9-001` seam
- `ANTHROPIC_API_KEY` and an Anthropic default model
- passing the conformance suite

## The Interesting Difference

Anthropic's Messages API has no response-MIME-type equivalent to the `ResponseMIMEType:
"application/json"` the Gemini path sets. That single difference is most of this story: every other
provider gets structured output by asking for it, and this one has to be built to guarantee it.

## Decisions To Implement

- **Guarantee JSON structurally, not by instruction.** Two mechanisms are available and either is
  acceptable, but the story must pick one and say why:
  - **assistant prefill** — seed the assistant turn with `{` so the model cannot open with prose;
    the provider then re-attaches the prefix so the shared parser receives a complete document. It
    must hand `resultFromText` valid JSON, never a fragment.
  - **forced tool use** — declare a tool whose input schema is the finding shape and require it, so
    conformance is the API's job.
  Whichever is chosen, the shared `resultFromText` still validates the result: the fail-closed
  contract is not delegated to the provider.
- **The system prompt goes in the top-level `system` parameter.** Not as a first user message.
  Same hardening property, different field name.
- **`max_tokens` is required and must be set deliberately.** A finding list truncated mid-JSON is
  unparseable, and the shared parser will correctly call that unusable — which is right, but a
  budget too small to hold a realistic finding list would turn every busy skill into an
  "Analyzer returned unusable response". Pick a value that fits the response shape and say what it
  is.
- **Send the `anthropic-version` header.** It is required, and pinning it in one named constant is
  what stops a future API change from arriving unannounced.
- **`net/http` first, on the same reasoning as `P9-002`.** One non-streaming request. Record the
  dependency delta; adopting the official SDK is an acceptable, stated escape hatch.
- **Default to a mid-tier model, and name the cheap one in the docs.** `claude-sonnet-5` is the
  sensible default for hostile-content review, with `claude-haiku-4-5-20251001` documented as the
  cheaper option for large corpora. `--model` overrides either. Put the ID in one place.
  This is a deliberate asymmetry with `P9-002` and `P9-004`, which name no model ID at all: these
  two were confirmed current when the story was written, and a confirmed ID is worth recording
  where a guessed one is worth omitting. Re-check them at implementation time regardless — the rule
  the other stories state still applies, this ticket has simply already done that step once.
- **Temperature `0`**, as everywhere else.
- **Map HTTP and API errors onto the existing analyzer error path**, with the status and the API's
  message in the wrapped error. No second failure vocabulary.
- **Respect `--timeout` through the request context.**

## Proposed Changes

- an Anthropic provider file implementing the `P9-001` interface
- the chosen structured-output mechanism, with the reason recorded in the code
- `ANTHROPIC_API_KEY` wired into the provider-aware credential lookup
- the default model and the pinned API version constant

## Acceptance Criteria

- `--provider anthropic` with a key set produces analyzer findings on a flagged skill and a clean
  result on a clean one, against a substituted transport
- the conformance suite from `P9-001` passes with no provider-specific exemption, including the
  JSON-output property — the chosen mechanism satisfies it rather than being waived
- if prefill is used, the shared parser receives a complete JSON document, asserted directly
- a response truncated at `max_tokens` produces the "Analyzer returned unusable response" finding
  rather than a clean result
- `--provider anthropic` with no `ANTHROPIC_API_KEY` degrades to rules-only with one warning naming
  that variable
- `--model` and `--timeout` reach the request, asserted on the recorded payload
- the provenance records `anthropic` and the model used
- no test makes a live API call

## Must Not Regress

- the five prompt-hardening properties
- the fail-closed response mapping in `resultFromText`
- the Gemini and OpenAI paths, untouched by this story
- dependency weight

## Documentation

- add Anthropic to the provider table in `README.md`: flag value, environment variable, default model
- document the cheaper model option and when it is the better choice
- record the structured-output mechanism and its reason in `CLAUDE.md`, next to the hardening
  properties — it is the one place a provider satisfies them differently

## Dependencies

- depends on `P9-001`
- independent of `P9-002`; either may land first
