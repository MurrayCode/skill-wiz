# P6-003 Collapse Equivalent Paths In Discovery

## Story

As a user passing overlapping paths on the command line, I want each skill file scanned exactly
once so that output, reports, and tallies are not silently duplicated.

## Scope

- normalise paths before the duplicate check in `discover.Files`
- keep argument ordering and directory sorting as they are

## The Defect

`discover.Files` deduplicates on the raw argument string (`discover/discover.go:32`), so two
spellings of the same file both survive:

```
$ skill-wiz examples/CLEANSKILL.md examples/./CLEANSKILL.md examples
[examples/CLEANSKILL.md  examples/./CLEANSKILL.md  examples/HIDDENBASHSKILL.md  examples/MISMATCHSKILL.md]
```

The package doc claims "duplicates are collapsed so that overlapping arguments scan a file once",
so this is a gap against stated behaviour rather than a design choice. Consequences today:

- the file is parsed, rule-scanned, and sent to the analyzer twice — a wasted LLM call per duplicate
- it appears twice in the console output and twice in the HTML report picker
- its findings are counted twice in the multi-file tally, and `N files scanned` overstates the run

Note the walk already collapses correctly against the *first* argument, which is why this reads as
an edge case rather than an obvious break.

## Decisions To Implement

- **Normalise with `filepath.Clean` for the key only.** Clean the path to derive the `seen` key, but
  keep appending the path as the user spelled it, so console output and report headers still echo
  the argument the user typed.
- **Do not resolve symlinks or absolute paths.** `filepath.Abs` would make the key depend on the
  working directory, and `EvalSymlinks` would touch the filesystem and can fail. `Clean` is a pure
  lexical fix and covers the realistic cases (`./`, `../`, trailing slashes, doubled separators).
  Two genuinely different paths reaching one file through a symlink stay out of scope.
- **First spelling wins.** When two spellings collide, keep the first in argument order and drop the
  later one, which is what the existing `seen` check already does.

## Proposed Changes

- clean the path when computing the `seen` key in `discover.Files`
- add table-driven cases for `./`-prefixed, `../`-traversing, trailing-slash, and doubled-separator
  spellings of the same file, and for a file named both explicitly and via its parent directory

## Acceptance Criteria

- `examples/CLEANSKILL.md` and `examples/./CLEANSKILL.md` collapse into one entry
- an explicitly named file inside a directory that is also named collapses into one entry
- the retained entry keeps the spelling of its first appearance
- explicit paths still keep argument order and directory matches are still sorted
- two genuinely different files are never collapsed
- `ErrNoSkillFiles` still fires for an empty expansion

## Must Not Regress

- explicit files still bypass the `.md` extension check
- hidden entries such as `.git` are still skipped during the walk
- single-file runs still produce a JSON object, not an array

## Documentation

- none user-facing; the package doc already describes the intended behaviour this story delivers

## Dependencies

- none
