# P6-002 Unicode-Safe Rule Tokenisation

## Story

As a user scanning a skill written in any language, I want the deterministic rules to tokenise the
text correctly so that non-ASCII skills are checked as reliably as English ones.

## Scope

- fix byte-versus-rune splitting in `rules.splitAlphaNumeric`
- confirm the rest of the tokenisation path is rune-safe
- prove the fix with non-ASCII fixtures

## The Defect

`splitAlphaNumeric` (`rules/rules.go:378`) walks the token by byte index but classifies with
`unicode.IsLetter(rune(token[i]))`. For any multi-byte character that converts a single UTF-8
continuation byte into a Latin-1 code point, so the letter/digit boundary test fires mid-rune and
the token is torn apart:

```
"café2"   -> ["caf\xc3", "\xa92"]
"naïve1"  -> ["na\xc3", "\xaf", "ve", "1"]
"日本語2"  -> ["\xe6", "\x97\xa5", "\xe6", "\x9c\xac", "\xe8\xaa", "\x9e2"]
"abc123"  -> ["abc", "123"]          # ASCII is unaffected
```

Those invalid-UTF-8 fragments flow straight into `tokenSet`, which feeds `unrelatedURLRule`'s
intent matching:

```
tokenSet("café résumé") = [résumé caf<?> sum<?> cafécaf<?> résumér<?> café r<?>]
```

So for a non-ASCII skill the intent token set is partly garbage, and the unrelated-URL rule's
`hasOverlap` check becomes unreliable in both directions — it can miss a genuinely related URL and
it can fail to flag an unrelated one.

## Decisions To Implement

- **Range over runes.** Replace the byte-index loop with a `range` over the token, which yields
  runes and their byte offsets, and slice on those offsets. This keeps the function returning
  substrings of the original token rather than rebuilt strings.
- **Fix the splitter, not the callers.** `splitTokens` and `tokenSet` already use
  `strings.FieldsFunc` with a rune predicate and are correct; only `splitAlphaNumeric` is at fault.
  Do not restructure the tokenisation pipeline in this story.
- **Non-ASCII letters are letters.** `unicode.IsLetter` already returns true for `é` and `語`, so
  once the loop is rune-correct the alphanumeric boundary logic needs no further change. A word in
  a non-Latin script should produce one token, not one per character.
- **Expect fixture movement.** `CLAUDE.md` warns that changing tokenisation moves fixture results.
  The `examples/` corpus is currently all-ASCII, so this fix should be inert there — but that is a
  claim to verify, not assume.

## Proposed Changes

- rewrite `splitAlphaNumeric` to iterate runes
- add table-driven cases covering accented Latin, non-Latin scripts, mixed alphanumeric, and the
  existing ASCII behaviour
- add a non-ASCII skill fixture exercising `unrelatedURLRule` end to end

## Acceptance Criteria

- `splitAlphaNumeric` returns only valid UTF-8 for every input, asserted with `utf8.ValidString`
- `"café2"` splits into `["café", "2"]`; `"日本語2"` splits into `["日本語", "2"]`
- existing ASCII behaviour is unchanged — `"abc123"` still splits into `["abc", "123"]`
- `tokenSet` on non-ASCII text contains the words themselves, not fragments
- a non-ASCII skill whose URL matches its stated purpose is *not* flagged by `unrelatedURLRule`
- the three `examples/` fixtures produce byte-identical findings to before the change

## Must Not Regress

- `CLEANSKILL.md` stays clean, `MISMATCHSKILL.md` still produces `mismatch`, `HIDDENBASHSKILL.md`
  still produces `shell`
- the bigram and singular-form entries `tokenSet` builds keep their current shape for ASCII input

## Documentation

- none user-facing; note the rune-safety requirement alongside the existing tokenisation warning in
  `CLAUDE.md`

## Dependencies

- none
- overlaps `P6-004` in `rules/rules.go`; land whichever first and rebase the other
