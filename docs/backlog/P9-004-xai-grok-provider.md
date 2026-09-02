# P9-004 xAI (Grok) Provider

## Story

As a user with an xAI account, I want `--provider xai`, so that I can run the analysis leg on Grok.

## Scope

- one xAI provider implementation behind the `P9-001` seam
- `XAI_API_KEY` and a Grok default model
- passing the conformance suite

## Why This Is The Smallest Of The Three

xAI exposes an OpenAI-compatible chat completions API, so this is substantially `P9-002` against a
different base URL, credential, and model name. The story is small, and its real content is deciding
how much of the OpenAI provider to reuse and how to notice when that compatibility stops holding.

## Decisions To Implement

- **Reuse the OpenAI request path, parameterised by base URL, credential, and default model.** Two
  near-identical copies of the same request builder is the outcome to avoid; a shared
  OpenAI-compatible transport with per-provider settings is the shape.
- **Reuse is an assumption with an expiry date, and the tests are what enforce it.** Compatibility
  is a courtesy, not a contract: a provider that changes its JSON-mode field or its error envelope
  breaks a shared path silently. The conformance suite must run against **xAI's own recorded
  payloads**, not against OpenAI's with the name changed, so a divergence fails a test rather than a
  user's scan. If the divergence ever grows past parameterisation, splitting the file is the correct
  response and not a regression.
- **`--provider xai`, with `grok` accepted as an alias.** The vendor is xAI and the model family is
  Grok; users will reach for both. One canonical value in the docs and the provenance output, one
  alias that resolves to it.
- **Verify the JSON-output mechanism actually works rather than assuming it does.** The conformance
  suite's JSON property must be satisfied by what xAI honours, not by what OpenAI honours. If the
  compatible endpoint does not support the structured-output field, say so in the story and rely on
  the shared parser's fail-closed behaviour — but say it explicitly, because a silently ignored
  parameter looks identical to a working one until the model returns prose.
- **Name the current Grok model as the default in one place**, with `--model` as the override. As in
  `P9-002`, this backlog deliberately names no ID: pick the current one when implementing, because a
  model name written into a backlog document ages badly and gets trusted anyway.
- **Same error mapping, same `--timeout` handling** as every other provider.

## Proposed Changes

- parameterise the OpenAI-compatible request path by base URL, credential, and default model
- register `xai` (alias `grok`) with `XAI_API_KEY` and the Grok default model
- conformance tests driven by xAI's own recorded request and response payloads

## Acceptance Criteria

- `--provider xai` and `--provider grok` both resolve to the same provider, and the provenance
  reports one canonical name
- the conformance suite passes using xAI's recorded payloads, not OpenAI's
- `--provider xai` with no `XAI_API_KEY` degrades to rules-only with one warning naming that variable
- `--model` and `--timeout` reach the request
- refactoring the OpenAI provider to share the transport leaves `P9-002`'s tests passing unchanged
- no test makes a live API call

## Must Not Regress

- the five prompt-hardening properties
- the fail-closed response mapping
- `P9-002`, which is refactored but not behaviourally changed by this story
- dependency weight — this story should add no dependency at all

## Documentation

- add xAI to the provider table in `README.md`, noting the `grok` alias
- state that the implementation rides the OpenAI-compatible endpoint, so a reader knows why the two
  behave alike and where to look when they stop

## Dependencies

- depends on `P9-001` and `P9-002`
