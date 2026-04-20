# P1-004 Analysis Error Handling

## Story

As a developer, I want `analyse.Analyze` to return errors rather than terminating the process so that the scanner can be tested and reused safely.

## Scope

- remove `log.Fatal` from package-level analysis code
- return explicit errors to the caller
- improve environment and API failure handling

## Proposed Changes

- change the function contract to return `(string, error)` or an equivalent structured response
- detect missing `GEMINI_API_KEY` explicitly
- allow callers to decide how failures are rendered or surfaced

## Acceptance Criteria

- missing API key returns a standard error
- upstream API failures do not terminate the process inside the package
- the CLI still exits with a useful message when analysis fails
- tests cover no-key and upstream failure cases through mocks or seams

## Dependencies

- should be completed before deeper LLM integration stories in Phase 3
