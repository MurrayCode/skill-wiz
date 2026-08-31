# P6-004 Linear-Time Rule Scanning

## Story

As a user scanning a large skill, I want the deterministic rules to run in time proportional to the
input so that a long skill body does not cost disproportionately more than a short one.

## Scope

- remove the quadratic URL-stripping loop in `unrelatedURLRule`
- stop running the shell regex on lines that cannot match
- stop re-stripping URLs from text that has already had them removed

## Measured Baseline

`rules.Scan` with the default rule set, against a synthetic skill of repeated prose plus one URL per
paragraph (Apple M2 Pro, `-benchtime 500x`):

| Body size | Time | Allocations |
| --- | --- | --- |
| 1.3 KB | 178 µs | 654 |
| 13 KB | 2.31 ms | 6,174 |
| 53 KB | 15.88 ms | 24,524 |

Ten times the input costs eighty-nine times the time. The growth is superlinear, and the cause is
one loop.

## The Three Costs

**1. Quadratic URL stripping — `rules/rules.go:231`**

`intentTokens` removes URLs with one `strings.ReplaceAll` pass over the whole text *per URL*. With
200 URLs in 26 KB that is roughly 5 MB scanned to strip 200 substrings. `keywords()` already solves
the same problem in a single pass with `urlPattern.ReplaceAllString`.

```
current   2,587,517 ns/op   1,037,810 B/op   5,224 allocs/op
fixed       571,882 ns/op     979,516 B/op   5,235 allocs/op    4.5x
```

**2. Shell regex on every line — `rules/rules.go:272`**

`shellCommandPattern` is a case-insensitive alternation, so Go's regexp engine gets no literal
prefix to anchor on and pays roughly 22 ns per byte — including on lines with no shell reference at
all. Because `sh` is a substring of `bash`, an ASCII case-folded `sh` check is a sound necessary
condition and skips the engine entirely.

```
current   587,919 ns/op   0 allocs/op
fixed      26,072 ns/op   0 allocs/op    22x
```

**3. Redundant URL strip per segment — `rules/rules.go:112`**

`bodySegments` strips URLs from the body, then `descriptionMismatchRule` calls `keywords(segment)`
on each of the resulting segments, and `keywords` runs `urlPattern.ReplaceAllString` again on text
that provably contains no URLs. Across 400 segments:

```
current   203,384 ns/op   254,575 B/op   2,800 allocs/op
fixed     144,908 ns/op   203,204 B/op   1,600 allocs/op    1.4x
```

Together these take a 26 KB skill from roughly 4.5 ms to roughly 1.4 ms, and — the point of the
story — make the curve linear.

## Decisions To Implement

- **Strip URLs with the regex, not a per-URL loop.** In `intentTokens`, replace the loop over
  `urls` with a single `urlPattern.ReplaceAllString`. This is behaviour-preserving: the regex match
  is a superset of the punctuation-trimmed URL that `extractURLs` returns, and the extra trailing
  character is a tokenisation delimiter either way. `intentTokens` then no longer needs its `urls`
  argument.
- **Prefilter the shell rule, do not rewrite the pattern.** Keep `shellCommandPattern` exactly as it
  is — it is tuned against the fixtures. Add a cheap allocation-free ASCII case-folded substring
  check for `sh` in front of it. Do not use `strings.ToLower` for the prefilter; that allocates per
  line and gives back much of the win.
- **Be honest about when the prefilter helps.** The 22x above is on a corpus where `sh` is rare. On
  prose full of "should", "finish", and "publish" the prefilter rarely fires and the gain
  approaches zero. It never makes things slower, but do not record 22x as a general claim.
- **Split `keywords`, do not add a flag.** Give the tokenising half its own function and have
  `keywords` call it after stripping. `descriptionMismatchRule` then calls the inner function
  directly for segments. A boolean parameter would be cheaper to write and worse to read.
- **Do not touch `tokenSet`'s allocation volume.** The bigram construction is the largest remaining
  allocation source but it is already linear. Leave it; a separate story can take it if profiling
  says it matters.

## Proposed Changes

- rewrite `intentTokens` to strip URLs in one regex pass and drop its now-unused parameter
- add an allocation-free case-folded prefilter ahead of the per-line shell regex
- extract the tokenising half of `keywords` and call it directly from `descriptionMismatchRule`
- add benchmarks covering the three paths so the scaling claim stays checkable

## Acceptance Criteria

- `go test ./rules/...` passes with the existing assertions unchanged
- the three `examples/` fixtures produce byte-identical findings to before the change
- scan time grows roughly linearly with body size across the 1.3 KB / 13 KB / 53 KB benchmark
- benchmarks for the URL, shell, and mismatch paths are committed and runnable with `-bench`
- a line containing `bash`, `BASH`, or `Sh` still reaches the regex — the prefilter is
  case-insensitive and never filters out a real match

## Must Not Regress

- `CLEANSKILL.md` stays clean, `MISMATCHSKILL.md` still produces `mismatch`, `HIDDENBASHSKILL.md`
  still produces `shell` with `./scripts/racing.sh` as evidence
- severity assignment is unchanged — local script execution stays `error`, a bare shell mention
  stays `warning`
- `benignShellMention` still suppresses the documentation-style phrases it suppresses today
- the first-match-wins behaviour of both shell branches, including which evidence snippet is
  reported

## Documentation

- none user-facing; the results are identical, only faster

## Dependencies

- none
- overlaps `P6-002` in `rules/rules.go`; land whichever first and rebase the other
