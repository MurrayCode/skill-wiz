# P9-002 OpenAI Provider

## Story

As a user with an OpenAI account and no Google one, I want `--provider openai`, so that I can run
the analysis leg on the credential I already have.

## Scope

- one OpenAI provider implementation behind the `P9-001` seam
- `OPENAI_API_KEY` and an OpenAI default model
- passing the conformance suite

## Decisions To Implement

- **`net/http` and `encoding/json`, not the SDK — unless the SDK earns it.** This is one
  non-streaming request with a system prompt, a user turn, a temperature, and a JSON response mode.
  `P6-001` deliberately dropped a dependency and eleven transitive ones; pulling a full SDK back in
  for a single POST inverts that. Record the measured `go.mod` and `go.sum` delta in the pull
  request. If auth, error shapes, or response parsing prove genuinely fiddly, adopting the official
  SDK is an acceptable escape hatch — but it is a decision to state, not a default.
- **Use the provider's structured-output mechanism, not a politely worded request.** OpenAI has a
  JSON response mode; use it, so malformed output is the API's problem before it is the parser's.
  The shared `resultFromText` remains the backstop, not the plan.
- **The system instruction goes in the system/developer message.** Never concatenated into the user
  turn. This is one of the five hardening properties and the conformance suite asserts it.
- **Temperature `0`**, matching every other provider. If the chosen model does not accept a
  temperature, say so in the story rather than silently dropping it — a scanner that is
  non-deterministic run to run is a different tool.
- **Pick the current small, fast model as the default and name it in one place.** Analysis is one
  short classification per skill; the flagship model is not the right default for a per-file scan.
  `--model` remains how a user overrides it. Do not scatter the ID across the code and the docs.
- **This backlog deliberately names no OpenAI model ID.** Model names turn over faster than backlog
  documents, and a stale ID written down here would be copied into the code months later by someone
  trusting it. Choosing the current one is part of implementing the story; the durable instruction
  is "small and fast, in one place", not a string.
- **Map HTTP failures onto the existing error path.** A non-2xx response is an analyzer error, which
  `scanner.Scan` already handles: swallowed when rules found something, propagated when they did
  not. Do not add a second failure vocabulary. Include the status code and the API's message in the
  wrapped error, because "generate analysis: 401" is what a user needs to see.
- **Respect `--timeout` through the request context**, so a hung upstream cannot stall a scan — the
  same bound `Config.Timeout` places on Gemini today.
- **No streaming, no tools, no retries in this story.** Retry policy is a cross-provider question
  and belongs in its own story if request failures turn out to be common.

## Proposed Changes

- an OpenAI provider file implementing the `P9-001` interface
- `OPENAI_API_KEY` wired into the provider-aware credential lookup
- the OpenAI default model
- provider registration so `--provider openai` resolves

## Acceptance Criteria

- `--provider openai` with a key set produces analyzer findings on a flagged skill and a clean
  result on a clean one, against a substituted transport
- the conformance suite from `P9-001` passes for this provider with no provider-specific exemption
- `--provider openai` with no `OPENAI_API_KEY` degrades to rules-only with one warning naming that
  variable
- a non-2xx response becomes an analyzer error carrying the status and the API's message
- `--model` overrides the default and reaches the request, asserted on the recorded payload
- `--timeout` bounds the request
- the JSON, console, and report provenance record `openai` and the model used
- no test makes a live API call

## Must Not Regress

- the five prompt-hardening properties, asserted here by the shared suite
- the fail-closed response mapping, which stays in `resultFromText`
- the Gemini path, which must be untouched by this story
- dependency weight: any addition to `go.mod` is justified in the pull request

## Documentation

- add OpenAI to the provider table in `README.md`: flag value, environment variable, default model
- note the default model and that `--model` overrides it

## Dependencies

- depends on `P9-001`
