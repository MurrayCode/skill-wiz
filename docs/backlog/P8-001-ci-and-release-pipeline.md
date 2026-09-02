# P8-001 CI And Release Pipeline

## Story

As a maintainer, I want every push and pull request verified automatically and every tag turned into
downloadable binaries, so that the checks `CLAUDE.md` already asks for are enforced by something
other than memory.

## Not To Be Confused With `P7-004`

`P7-004` documents how *someone else* runs `skill-wiz` in *their* CI. This story is this
repository's own pipeline. The two share nothing but the word — no shipped GitHub Action, no
published workflow for consumers; that stays parked with `P5-005`.

## Scope

- a CI workflow running build, vet, format, and tests on push and pull request
- a release workflow turning a tag into cross-platform binaries on a GitHub Release
- the minimum production change a released binary needs to identify itself

## The Problem

There is no `.github` directory. `go build ./...`, `go vet ./...`, and `go test ./...` are documented
in `CLAUDE.md` as the commands for this project, and every phase so far has been verified by running
them by hand. Nothing enforces that, nothing runs them on a pull request, and nothing catches the
failure modes this codebase has specifically built defences against:

- `P6-005` introduced a bounded worker pool. `go test -race ./...` passes today, and a data race
  reintroduced later would be invisible to a plain `go test`.
- `TestMain` unsets `GEMINI_API_KEY`, points `policyDirectory` at an empty temporary directory, and
  points `reportPath` at `t.TempDir()`. Those seams exist so the suite cannot depend on the
  developer's environment or write into the working tree — a machine with none of that set up is the
  only honest test of whether they still hold.
- There is no released artefact at all. The only way to run the tool is to have a Go toolchain and
  the source.

## Decisions To Implement

### CI

- **GitHub Actions, two workflows, not one.** `ci.yml` on push and pull request; `release.yml` on
  tags. Merging them means a release job that has to be skipped on every ordinary push, which is how
  a release pipeline stops being read.
- **No secrets, and that is a property to protect.** The suite makes no live API calls by design —
  every LLM path sits behind a swappable seam. CI must not be given a `GEMINI_API_KEY`, because a
  test that only passes with one is a regression this project has explicitly ruled out, and CI is
  where that would be discovered. Grant the workflow `permissions: contents: read`.
- **Pin the toolchain to `go.mod`.** `actions/setup-go` with `go-version-file: go.mod`, so the
  version CI uses cannot drift from the version the module declares. Do not hard-code `1.24` in the
  workflow.
- **Run the race detector, not just the tests.** `go test -race ./...` is the check that protects
  `P6-005`; a plain `go test` would let a race land green.
- **Check formatting as a step, not as a linter.** `gofmt -l .` failing the job when it prints
  anything. No `golangci-lint` in this story: adding a linter to a codebase that has never had one
  is its own change with its own backlog of findings, and bundling it means the first CI run is a
  wall of unrelated failures.
- **Assert the working tree is clean after the tests.** `git diff --exit-code` (and a check for
  untracked files) as the step after `go test`. This is the only automatic guard on the test seams:
  a test that writes `skill-wiz-report.html` or a `.skill-wiz.yaml` into the repository fails the
  job instead of quietly polluting a developer's checkout.
- **Dogfood the binary against `examples/`.** Build it, then run it over the corpus and assert the
  exit codes: `examples/CLEANSKILL.md` exits `0`, `examples/HIDDENBASHSKILL.md` exits `2`. This runs
  with no API key, which is exactly the rules-only path from `P6-009`, so it also proves the
  degraded run still gates correctly. The unit tests assert findings; this asserts the contract a
  consumer actually depends on.
- **Ubuntu only for CI.** Cross-platform behaviour is a build concern, and the release matrix covers
  it. Running the suite three times to catch a path separator bug the code does not have is cost
  without a finding behind it.
- **The baseline is already green, so a red first run means the workflow is wrong.** `gofmt -l .`
  and `go vet ./...` were both run against the tree on 2026-09-02 and produced no output. That is
  what makes "CI passes on the current `main` with no source changes" a criterion rather than an
  aspiration: if the first run fails, suspect the workflow before suspecting the code.
- **Cancel superseded runs.** A `concurrency` group keyed on the ref, so a force-push does not leave
  two runs racing to report on the same branch.

### Release

- **Tag-triggered, `v*`, and nothing else.** No release on merge to `main`. A release is a
  deliberate act.
- **A version the binary can report.** There is no version anywhere in the code today, so a
  downloaded binary cannot say what it is. Add one package-level variable stamped through
  `-ldflags -X`, defaulting to `dev`, printed by a `--version` flag that prints and exits `0`. This
  is the one piece of production code in the story and it should stay that small — no build date, no
  commit hash, no `version` subcommand. It is kept here because a release without it ships a binary
  that cannot say what it is; if the CI half is wanted sooner than the release half, splitting
  `--version` and `release.yml` into their own story is a clean cut, and `ci.yml` stands alone
  without them.
- **Build matrix: `linux`, `darwin`, `windows` × `amd64`, `arm64`.** `CGO_ENABLED=0` and
  `-trimpath`, so the binaries are static and the paths in any panic are not the runner's.
- **Attach checksums.** One `SHA256SUMS` file alongside the archives. Anyone gating a publishing
  workflow on this tool will want to verify what they downloaded, and adding it later means the
  early releases are the ones without it.
- **`permissions: contents: write` on the release job only.** The CI workflow stays read-only.
- **Pin actions to major version tags.** `actions/checkout@v4` and friends. Full SHA pinning is the
  stricter choice and is worth revisiting if this repository ever publishes something others
  install; it is not worth the update burden yet. State the choice in the workflow so the next
  person knows it was a decision.
- **No package managers in this story.** Homebrew, `go install` documentation, Docker images — each
  is a distribution channel with its own maintenance. Ship binaries first and see if anyone wants
  more.

## Proposed Changes

- `.github/workflows/ci.yml`: checkout, setup-go from `go.mod`, `gofmt -l .`, `go build ./...`,
  `go vet ./...`, `go test -race ./...`, clean-tree assertion, dogfood run against `examples/`
- `.github/workflows/release.yml`: tag-triggered build matrix, archives, checksums, GitHub Release
- a version variable in `main` plus a `--version` flag, defaulting to `dev`

## Acceptance Criteria

- a pull request runs the CI workflow and reports a status
- CI fails when: the code is unformatted, `go vet` complains, any test fails, the race detector
  fires, the working tree is dirty after the tests, or the `examples/` exit codes change
- CI passes on the current `main` with no source changes other than the workflow files
- CI runs with no repository secrets configured
- pushing a `v0.1.0` tag produces a GitHub Release with six binaries and a `SHA256SUMS` file
- a released binary prints its tag for `--version`; a locally built one prints `dev`
- `--version` prints and exits `0` without requiring a path argument

## Must Not Regress

- the test suite still makes no live API calls, and CI must never be handed an API key to make one pass
- the three-way exit code contract, which the dogfood step now depends on
- `--json` prints the JSON and nothing else — `--version` must not leak into it
- the existing flags: `--version` is additive and takes no argument

## Documentation

- a status badge and a short "Contributing" note in `README.md` giving the commands CI runs, so a
  contributor can reproduce a failure locally
- an install section covering the released binaries and checksum verification
- document `--version` in the flags table
- note in `CLAUDE.md` that CI enforces `gofmt`, `go vet`, `go test -race`, and a clean working tree

## Follow-Ups Deliberately Not In This Story

- `golangci-lint`, which needs its own story and its own first cleanup pass
- coverage reporting, which needs a target before a number means anything
- Dependabot or Renovate for the twenty-odd transitive dependencies `P6-001` left
- branch protection requiring the CI check, which is a repository setting rather than a file and
  must be enabled by hand once the workflow has run green at least once

## Dependencies

- none in the code
- the dogfood step rests on `P4-003` (exit codes) and the `examples/` corpus, both done
- `P7-002` would let the dogfood step put its report somewhere other than the workspace, which the
  clean-tree assertion would otherwise have to tolerate; not a blocker, since the report is gitignored
