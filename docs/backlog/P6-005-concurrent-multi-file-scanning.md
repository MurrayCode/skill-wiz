# P6-005 Concurrent Multi-File Scanning

## Story

As a user scanning a directory of skills, I want files scanned concurrently so that a run costs
roughly one analyzer round trip rather than one per file.

## Scope

- scan discovered files through a bounded worker pool
- preserve output, report, and exit-code behaviour exactly
- keep stderr reporting deterministic

## The Cost

`run` loops files one at a time (`main.go:143`), and every `scanFile` blocks on a Gemini call inside
`scanner.Scan`. The deterministic rules take microseconds; the analyzer takes seconds. So wall clock
is approximately `N x latency` — scanning fifty skills is fifty sequential round trips. With a pool
of eight it becomes approximately `ceil(N/8) x latency`.

This is the largest available performance win in the project and nothing else is close.

## Why This Is Safe

The scan path has no shared mutable state:

- `skillRules` is a package-level slice of stateless `RuleFunc`s
- the compiled `regexp` values in `rules` are documented safe for concurrent use
- `GeminiAnalyzer` is a stateless value; `analyse.Config` is read-only
- `scanFile` reads its own file and returns a value
- `report` and `renderScans` both run after every scan has finished

## Decisions To Implement

- **Bounded pool, not one goroutine per file.** A directory scan can cover hundreds of files;
  unbounded goroutines would mean hundreds of simultaneous API requests and near-certain rate
  limiting. Use a fixed worker count.
- **Concurrency is configurable, with a sane default.** Add `--concurrency`, defaulting to a small
  fixed number rather than `GOMAXPROCS` — the work is network-bound, not CPU-bound, so the right
  default follows the API's tolerance, not the machine's core count. `--concurrency 1` must restore
  exactly the current sequential behaviour, which also gives the tests a deterministic mode.
  Reject zero and negative values the way `--timeout` already does.
- **Preserve order by index, not by completion.** Allocate a results slice sized to the file list
  and have each worker write to its own index. Never append from a goroutine. Console output, the
  report, the JSON array, and the tally all keep their current file ordering for free.
- **Buffer stderr per file and emit in file order after the run.** Writing to `stderr` from workers
  would interleave failure messages non-deterministically and break the existing multi-file error
  tests. Collect each failure alongside its index and print them in order.
- **`--timeout` stays per request.** It currently bounds one analyzer call and must keep doing so.
  Do not repurpose it as a whole-run deadline; that would be a silent behaviour change for anyone
  scanning a directory today.
- **Cancellation is out of scope.** Do not add a shared cancellable context that aborts siblings on
  the first failure. The run's contract is that one bad file never hides the rest, and a fail-fast
  pool would break exactly that.

## Proposed Changes

- add `--concurrency` to `parseOptions` with validation matching the existing flags
- replace the scan loop in `run` with a bounded worker pool writing into an indexed slice
- collect per-file failures with their index and report them in file order
- filter the indexed slice down to successful scans before rendering, preserving order
- add tests that assert ordering and the failure-reporting order under concurrency, using a stubbed
  analyzer with staggered delays so completion order differs from file order

## Acceptance Criteria

- console output, JSON, report contents, and the tally are byte-identical to a sequential run for
  the same inputs
- stderr failure messages appear in file order regardless of completion order
- `--concurrency 1` behaves exactly as the current implementation does
- an invalid `--concurrency` is reported once and exits `1`
- a run where some files fail and others are flagged still exits `1`, per the `P4-003` precedence
- `go test -race ./...` passes
- a stubbed-analyzer test proves files complete out of order yet render in order

## Must Not Regress

- one bad file never hides the rest; a run where every file failed still prints nothing and exits `1`
- exit code precedence — operational failure outranks findings
- one run, one report; every JSON entry carries the same `report_path`
- single-file output is unchanged: no path header, a JSON object rather than an array, no tally
- `--json` prints the JSON and nothing else

## Documentation

- document `--concurrency`, its default, and that `--timeout` remains per request in `README.md`
- add the ordering and stderr-buffering guarantees to the invariants in `CLAUDE.md`

## Dependencies

- depends on `P4-002` for multi-file scanning
- interacts with `P6-009`: preflighting the API key avoids N concurrent failures when the key is
  missing, and is worth landing first
