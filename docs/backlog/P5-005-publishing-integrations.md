# P5-005 Publishing Integrations — Spike

## Story

As a platform owner, I want to know what `skill-wiz` would need in order to gate a skill registry or
publishing workflow, so that we can decide whether to build it.

## This Is A Spike, Not An Implementation

There is no concrete integration target yet, and building an integration surface for an imagined
consumer is how a small tool grows an API nobody uses. The deliverable is a written assessment.
Do not add packages, flags, or exported types under this story; anything it recommends becomes its
own story with its own acceptance criteria.

## Deliverable

One document, `docs/publishing-integration.md`, answering:

1. **What a gate actually needs.** Which of the existing surfaces — exit codes (`P4-003`), JSON
   output (`P4-001`), the run summary (`P5-004`), policy (`P5-001`) — is sufficient on its own, and
   what is genuinely missing.
2. **The minimum skill metadata a registry would require** beyond `name` and `description`, and
   whether validating it belongs in `skill` or in a policy.
3. **Where the boundary sits.** Which candidate integrations (GitHub Action, pre-commit hook,
   registry API, CI template) can be built entirely on the current CLI contract, and which would
   force a library or server surface. Name the cheapest one that would be genuinely useful.
4. **The recommendation.** Build now, build later, or do not build — with the reason stated plainly,
   including the case for not building.

## Acceptance Criteria

- `docs/publishing-integration.md` exists and answers all four questions
- each claim about a current surface being sufficient or insufficient names the code or contract it
  is based on
- the document ends with a recommendation and, if it is "build", a proposed follow-up story per
  piece of work
- no production code changes land under this story

## Dependencies

- best done after `P4-003` and `P5-004`, since both shape what a gate would consume
- do not start until there is a real candidate consumer to write about; parking this is a valid
  outcome
